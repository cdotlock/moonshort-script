// Package parser converts a token stream from the lexer into an AST.
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/token"
)

// knownKeywords are @-directive keywords that are NOT character names.
var knownKeywords = map[string]bool{
	"bg": true, "cg": true, "phone": true, "text": true,
	"music": true, "sfx": true, "minigame": true, "choice": true,
	"affection": true, "signal": true,
	"butterfly": true, "if": true, "else": true, "label": true,
	"goto": true, "episode": true,
	"option": true, "gate": true, "next": true, "pause": true,
	"ending": true, "achievement": true,
}

// validEndingTypes enumerates the accepted values for @ending <type>.
var validEndingTypes = map[string]bool{
	"complete":        true,
	"to_be_continued": true,
	"bad_ending":      true,
}

// validSignalKinds enumerates accepted values for the kind token in
// @signal <kind> <event>. Only "mark" is currently implemented; future
// kinds will expand this whitelist.
var validSignalKinds = map[string]bool{
	"mark": true,
}

// validRarities enumerates accepted achievement rarities. Common is
// intentionally banned.
var validRarities = map[string]bool{
	"uncommon":  true,
	"rare":      true,
	"epic":      true,
	"legendary": true,
}

// Parser consumes tokens from a Lexer and produces an AST.
type Parser struct {
	l       *lexer.Lexer
	cur     token.Token
	peek    token.Token
	pending ast.Node // set by parseDialogueWithExpr; drained on next parseStatement call
	depth   int      // block nesting depth for recursion limit
}

// New creates a Parser that reads tokens from the given Lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	// Prime cur and peek.
	p.advance()
	p.advance()
	return p
}

// advance moves to the next non-COMMENT, non-NEWLINE token.
func (p *Parser) advance() {
	p.cur = p.peek
	for {
		p.peek = p.l.NextToken()
		if p.peek.Type != token.COMMENT && p.peek.Type != token.NEWLINE {
			break
		}
	}
}

// advanceRaw moves to the next token without skipping anything.
// Used when the caller needs precise control (e.g. reading conditions).
func (p *Parser) advanceRaw() {
	p.cur = p.peek
	p.peek = p.l.NextToken()
}

// expect checks that the current token has the given type, then advances.
// Returns an error if the type doesn't match.
func (p *Parser) expect(t token.Type) (token.Token, error) {
	if p.cur.Type != t {
		return p.cur, fmt.Errorf("line %d col %d: expected %s, got %s (%q)",
			p.cur.Line, p.cur.Col, t, p.cur.Type, p.cur.Literal)
	}
	tok := p.cur
	p.advance()
	return tok, nil
}

// Parse parses an entire @episode block and returns the root Episode node.
func (p *Parser) Parse() (*ast.Episode, error) {
	// @episode <branch_key> "<title>" { body @gates { ... } }
	if _, err := p.expect(token.AT); err != nil {
		return nil, err
	}
	ep, err := p.expect(token.IDENT)
	if err != nil || ep.Literal != "episode" {
		return nil, fmt.Errorf("line %d col %d: expected 'episode', got %q",
			ep.Line, ep.Col, ep.Literal)
	}

	key, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	title, err := p.expect(token.STRING)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	episode := &ast.Episode{
		BranchKey: key.Literal,
		Title:     title.Literal,
	}

	body, gates, ending, achievements, err := p.parseEpisodeBody()
	if err != nil {
		return nil, err
	}
	episode.Body = body
	episode.Gate = gates
	episode.Ending = ending
	episode.Achievements = achievements

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return episode, nil
}

// parseEpisodeBody parses body nodes until RBRACE. @gate, @ending, and
// @achievement are hoisted to episode-level fields instead of remaining in
// the body.  @gate and @ending are mutually exclusive terminals. Multiple
// @achievement declarations are accumulated.
func (p *Parser) parseEpisodeBody() ([]ast.Node, *ast.GateBlock, *ast.EndingNode, []*ast.AchievementNode, error) {
	var body []ast.Node
	var gates *ast.GateBlock
	var ending *ast.EndingNode
	var achievements []*ast.AchievementNode

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		// Drain any pending node queued by parseDialogueWithExpr before
		// checking the @gate / @ending / @achievement short-circuit paths.
		if p.pending != nil {
			body = append(body, p.pending)
			p.pending = nil
			continue
		}
		if p.cur.Type == token.AT && p.peek.Literal == "gate" {
			if gates != nil {
				return nil, nil, nil, nil, fmt.Errorf("line %d col %d: duplicate @gate block", p.cur.Line, p.cur.Col)
			}
			if ending != nil {
				return nil, nil, nil, nil, fmt.Errorf("line %d col %d: @gate cannot coexist with @ending (an ending is terminal)", p.cur.Line, p.cur.Col)
			}
			g, err := p.parseGateBlock()
			if err != nil {
				return nil, nil, nil, nil, err
			}
			gates = g
			continue
		}
		if p.cur.Type == token.AT && p.peek.Literal == "ending" {
			if ending != nil {
				return nil, nil, nil, nil, fmt.Errorf("line %d col %d: duplicate @ending directive", p.cur.Line, p.cur.Col)
			}
			if gates != nil {
				return nil, nil, nil, nil, fmt.Errorf("line %d col %d: @ending cannot coexist with @gate (an ending is terminal)", p.cur.Line, p.cur.Col)
			}
			e, err := p.parseEnding()
			if err != nil {
				return nil, nil, nil, nil, err
			}
			ending = e
			continue
		}
		node, err := p.parseStatement()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if node == nil {
			continue
		}
		// Declarative achievement blocks (with `{` ... `}`) are hoisted out
		// of the body into Episode.Achievements. Bare @achievement <id>
		// triggers (without `{`) stay in the body like any other step.
		if decl, ok := node.(*ast.AchievementNode); ok {
			achievements = append(achievements, decl)
			continue
		}
		body = append(body, node)
	}
	// Drain final pending node if any.
	if p.pending != nil {
		body = append(body, p.pending)
		p.pending = nil
	}
	return body, gates, ending, achievements, nil
}

// parseBlock parses body nodes until RBRACE (but does NOT consume the RBRACE).
func (p *Parser) parseBlock() ([]ast.Node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 50 {
		return nil, fmt.Errorf("line %d col %d: maximum nesting depth exceeded (50)", p.cur.Line, p.cur.Col)
	}
	var nodes []ast.Node
	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		// Drain any pending node queued by parseDialogueWithExpr.
		if p.pending != nil {
			nodes = append(nodes, p.pending)
			p.pending = nil
			continue
		}
		node, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	// Drain final pending node if any (e.g. dialogue-with-expr was the last
	// statement before RBRACE).
	if p.pending != nil {
		nodes = append(nodes, p.pending)
		p.pending = nil
	}
	return nodes, nil
}

// parseStatement dispatches to the appropriate parser based on current token.
// If a pending node was queued by parseDialogueWithExpr, it is returned first.
func (p *Parser) parseStatement() (ast.Node, error) {
	if p.pending != nil {
		node := p.pending
		p.pending = nil
		return node, nil
	}
	if p.cur.Type == token.AT {
		return p.parseDirective()
	}
	if p.cur.Type == token.AMPERSAND {
		return p.parseConcurrentDirective()
	}
	if p.cur.Type == token.IDENT && p.peek.Type == token.COLON {
		return p.parseDialogue()
	}
	if p.cur.Type == token.IDENT && p.peek.Type == token.LBRACKET {
		return p.parseDialogueWithExpr()
	}
	// Skip unexpected tokens to avoid infinite loops.
	p.advance()
	return nil, nil
}

// parseConcurrentDirective handles &-prefixed directives. It parses the
// directive the same way as @-prefixed ones, then marks the resulting node
// as concurrent.
func (p *Parser) parseConcurrentDirective() (ast.Node, error) {
	// Check if this looks like dialogue with & prefix (e.g., &NARRATOR: text)
	// We peek ahead: if AMPERSAND is followed by an IDENT+COLON pattern, it's a dialogue mistake.
	if p.peek.Type == token.IDENT {
		// Check if the ident is all-caps and followed by colon (dialogue pattern)
		name := p.peek.Literal
		isUpper := true
		for _, r := range name {
			if r < 'A' || r > 'Z' {
				if r != '_' && (r < '0' || r > '9') {
					isUpper = false
					break
				}
			}
		}
		if isUpper && len(name) > 0 {
			// Save state to check for colon - we need to peek 2 ahead
			// The lexer has already loaded peek, so we'd need the token after peek
			// Instead, just try parsing and catch the error with a better message
		}
	}

	node, err := p.parseDirective() // parseDirective consumes AT/AMPERSAND via advance
	if err != nil {
		// If the error is about expecting IDENT but got COLON, this is likely &DIALOGUE: text
		if strings.Contains(err.Error(), "got COLON") || strings.Contains(err.Error(), "got \":\"") {
			return nil, fmt.Errorf("%v (hint: & prefix cannot be used with dialogue lines, remove the &)", err)
		}
		return nil, err
	}
	if hc, ok := node.(ast.HasConcurrent); ok {
		hc.SetConcurrent(true)
	}
	return node, nil
}

// parseDialogue handles IDENT COLON text lines (NARRATOR, YOU, or character).
//
// When this is called, cur=IDENT and peek=COLON. The lexer's internal position
// is right after having read the COLON token, which is exactly where
// ReadDialogueText expects to start. We call it directly, then re-prime
// the parser's cur/peek lookahead.
func (p *Parser) parseDialogue() (ast.Node, error) {
	name := p.cur.Literal

	// The lexer is positioned right after COLON. Call ReadDialogueText.
	textTok := p.l.ReadDialogueText()

	// Re-prime cur and peek, skipping comments and newlines.
	p.cur = p.l.NextToken()
	for p.cur.Type == token.COMMENT || p.cur.Type == token.NEWLINE {
		p.cur = p.l.NextToken()
	}
	p.peek = p.l.NextToken()
	for p.peek.Type == token.COMMENT || p.peek.Type == token.NEWLINE {
		p.peek = p.l.NextToken()
	}

	switch name {
	case "NARRATOR":
		return &ast.NarratorNode{Text: textTok.Literal}, nil
	case "YOU":
		return &ast.YouNode{Text: textTok.Literal}, nil
	default:
		return &ast.DialogueNode{Character: name, Text: textTok.Literal}, nil
	}
}

// parseDialogueWithExpr handles the syntax sugar:
//
//	CHARACTER [pose_expr]: text
//
// Token stream when called: cur=IDENT("MAURICIO"), peek=LBRACKET
// Full stream:  IDENT LBRACKET IDENT RBRACKET COLON <dialogue text>
//
// This is equivalent to:
//
//	@character expr pose_expr
//	CHARACTER: text
//
// It returns CharExprNode immediately, and queues the dialogue node as p.pending
// so that the next parseStatement call drains it — preserving correct order.
//
// Lexer position note: each p.advance() reads one token from the lexer into
// p.peek, advancing the lexer's internal position. By the time cur==COLON,
// p.peek already holds whatever follows the COLON (first word of dialogue),
// meaning the lexer has consumed past the COLON. We must NOT call p.advance()
// again before ReadDialogueText — instead we call it directly on the position
// the lexer is already at (after cur was loaded into peek).
//
// The trick: after RBRACKET is confirmed in cur, p.peek == COLON.
// We do NOT advance again; instead we call ReadDialogueText which starts
// reading from the lexer's current position — which is right after COLON
// (since COLON was the token that was pulled into p.peek).
func (p *Parser) parseDialogueWithExpr() (ast.Node, error) {
	name := p.cur.Literal
	charID := strings.ToLower(name)

	// cur=IDENT(name), peek=LBRACKET
	p.advance() // cur=LBRACKET, peek=IDENT(pose)
	p.advance() // cur=IDENT(pose), peek=RBRACKET

	// Read pose_expr IDENT.
	if p.cur.Type != token.IDENT {
		return nil, fmt.Errorf("line %d col %d: expected pose expression after '[', got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}
	pose := p.cur.Literal

	p.advance() // cur=RBRACKET, peek=COLON

	// Confirm RBRACKET.
	if p.cur.Type != token.RBRACKET {
		return nil, fmt.Errorf("line %d col %d: expected ']' after pose expression, got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}

	// At this point: cur=RBRACKET, peek=COLON.
	// p.peek was loaded by calling p.l.NextToken() inside the last p.advance(),
	// so the lexer's internal pos is now right after the COLON character — exactly
	// where ReadDialogueText expects to start. Do NOT call p.advance() here.
	if p.peek.Type != token.COLON {
		return nil, fmt.Errorf("line %d col %d: expected ':' after ']', got %s (%q)",
			p.peek.Line, p.peek.Col, p.peek.Type, p.peek.Literal)
	}

	textTok := p.l.ReadDialogueText()

	// Re-prime cur and peek, skipping comments and newlines.
	p.cur = p.l.NextToken()
	for p.cur.Type == token.COMMENT || p.cur.Type == token.NEWLINE {
		p.cur = p.l.NextToken()
	}
	p.peek = p.l.NextToken()
	for p.peek.Type == token.COMMENT || p.peek.Type == token.NEWLINE {
		p.peek = p.l.NextToken()
	}

	// Queue the dialogue node as pending — returned on the next parseStatement call.
	switch name {
	case "NARRATOR":
		p.pending = &ast.NarratorNode{Text: textTok.Literal}
	case "YOU":
		p.pending = &ast.YouNode{Text: textTok.Literal}
	default:
		p.pending = &ast.DialogueNode{Character: name, Text: textTok.Literal}
	}

	// Return CharLookNode first.
	return &ast.CharLookNode{
		Char: charID,
		Look: pose,
	}, nil
}

// parseDirective parses an @-prefixed or &-prefixed directive.
func (p *Parser) parseDirective() (ast.Node, error) {
	p.advance() // consume AT or AMPERSAND
	keyword := p.cur.Literal

	switch keyword {
	case "bg":
		return p.parseBg()
	case "cg":
		return p.parseCg()
	case "phone":
		return p.parsePhone()
	case "text":
		return p.parseText()
	case "music":
		return p.parseMusic()
	case "sfx":
		return p.parseSfx()
	case "minigame":
		return p.parseMinigame()
	case "choice":
		return p.parseChoice()
	case "affection":
		return p.parseAffection()
	case "signal":
		return p.parseSignal()
	case "butterfly":
		return p.parseButterfly()
	case "if":
		return p.parseIf()
	case "label":
		return p.parseLabel()
	case "goto":
		return p.parseGoto()
	case "pause":
		return p.parsePause()
	case "achievement":
		return p.parseAchievement()
	case "on":
		return nil, fmt.Errorf("line %d col %d: @on is no longer supported — use @if (check.success) / @if (rating.S) inside brave options and minigames", p.cur.Line, p.cur.Col)
	default:
		// Not a known keyword — treat as character directive.
		return p.parseCharDirective(keyword)
	}
}

// parseBg parses: @bg set <name> [transition]
func (p *Parser) parseBg() (ast.Node, error) {
	p.advance() // consume "bg"
	if _, err := p.expect(token.IDENT); err != nil { // consume "set"
		return nil, err
	}
	name, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.BgSetNode{Name: name.Literal}
	// Optional transition.
	if p.cur.Type == token.IDENT && !p.isDirectiveStart() {
		node.Transition = p.cur.Literal
		p.advance()
	}
	return node, nil
}

// validCgDurations enumerates accepted @cg duration tiers.
var validCgDurations = map[string]bool{
	ast.CgDurationLow:    true,
	ast.CgDurationMedium: true,
	ast.CgDurationHigh:   true,
}

// parseCg parses:
//
//	@cg show <name> [transition] {
//	  duration: <low|medium|high>
//	  content: "<narrative>"
//	  <body nodes>
//	}
//
// duration and content are required fields that must appear before any
// body statement. Field order between them is free; they may not repeat.
func (p *Parser) parseCg() (ast.Node, error) {
	p.advance() // consume "cg"
	if _, err := p.expect(token.IDENT); err != nil { // consume "show"
		return nil, err
	}
	name, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.CgShowNode{Name: name.Literal}
	// Optional transition before LBRACE.
	if p.cur.Type == token.IDENT && p.cur.Literal != "{" {
		node.Transition = p.cur.Literal
		p.advance()
	}
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	// Read field-value pairs (duration, content) until we see a non-field
	// statement (@, &, dialogue pattern, or RBRACE).
	seen := map[string]bool{}
	for p.cur.Type == token.IDENT && p.peek.Type == token.COLON {
		key := p.cur.Literal
		if key != "duration" && key != "content" {
			// Not a CG field; fall through to body parsing (this path
			// is unusual — would mean an IDENT:COLON dialogue line whose
			// IDENT happens to look like lowercase. Dialogue uses caps so
			// this is rare but kept for safety).
			break
		}
		if seen[key] {
			return nil, fmt.Errorf("line %d col %d: duplicate @cg field %q", p.cur.Line, p.cur.Col, key)
		}
		seen[key] = true
		p.advance() // consume key
		if _, err := p.expect(token.COLON); err != nil {
			return nil, err
		}
		switch key {
		case "duration":
			val, err := p.expect(token.IDENT)
			if err != nil {
				return nil, fmt.Errorf("line %d col %d: cg duration must be an identifier (low|medium|high)", val.Line, val.Col)
			}
			if !validCgDurations[val.Literal] {
				return nil, fmt.Errorf("line %d col %d: invalid cg duration %q (must be low, medium, or high)", val.Line, val.Col, val.Literal)
			}
			node.Duration = val.Literal
		case "content":
			val, err := p.expect(token.STRING)
			if err != nil {
				return nil, fmt.Errorf("line %d col %d: cg content must be a quoted string", val.Line, val.Col)
			}
			node.Content = val.Literal
		}
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	node.Body = body
	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}

	if node.Duration == "" {
		return nil, fmt.Errorf("@cg show %q: missing required field 'duration'", node.Name)
	}
	if node.Content == "" {
		return nil, fmt.Errorf("@cg show %q: missing required field 'content'", node.Name)
	}
	return node, nil
}

// parsePhone parses: @phone show { body } or @phone hide
func (p *Parser) parsePhone() (ast.Node, error) {
	p.advance() // consume "phone"
	action, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	switch action.Literal {
	case "show":
		if _, err := p.expect(token.LBRACE); err != nil {
			return nil, err
		}
		node := &ast.PhoneShowNode{}
		// Parse body — can contain @text directives and other nodes.
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		node.Body = body
		if _, err := p.expect(token.RBRACE); err != nil {
			return nil, err
		}
		return node, nil
	case "hide":
		return &ast.PhoneHideNode{}, nil
	default:
		return nil, fmt.Errorf("line %d: unknown phone action %q", action.Line, action.Literal)
	}
}

// parseText parses: @text from <char>: content  or  @text to <char>: content
//
// After consuming "text", we see direction then char. The char IDENT is
// followed by COLON, which mirrors the dialogue IDENT COLON pattern.
// We consume "text" and "direction" via advance, then find char in cur
// with COLON in peek. At that point the lexer is right after COLON,
// so we call ReadDialogueText directly.
func (p *Parser) parseText() (ast.Node, error) {
	p.advance() // consume "text"
	direction := p.cur
	if direction.Type != token.IDENT {
		return nil, fmt.Errorf("line %d: expected direction (from/to), got %s",
			direction.Line, direction.Type)
	}
	p.advance() // consume direction

	// Now cur = char IDENT, peek = COLON.
	char := p.cur
	if char.Type != token.IDENT {
		return nil, fmt.Errorf("line %d: expected character name, got %s",
			char.Line, char.Type)
	}
	if p.peek.Type != token.COLON {
		return nil, fmt.Errorf("line %d: expected COLON after character in @text, got %s",
			p.peek.Line, p.peek.Type)
	}

	// Lexer is right after COLON. Call ReadDialogueText.
	textTok := p.l.ReadDialogueText()

	// Re-prime parser state.
	p.cur = p.l.NextToken()
	for p.cur.Type == token.COMMENT || p.cur.Type == token.NEWLINE {
		p.cur = p.l.NextToken()
	}
	p.peek = p.l.NextToken()
	for p.peek.Type == token.COMMENT || p.peek.Type == token.NEWLINE {
		p.peek = p.l.NextToken()
	}

	return &ast.TextMessageNode{
		Direction: direction.Literal,
		Char:      char.Literal,
		Content:   textTok.Literal,
	}, nil
}

// parseMusic parses: @music play <name>, @music crossfade <name>, @music fadeout
func (p *Parser) parseMusic() (ast.Node, error) {
	p.advance() // consume "music"
	action, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	switch action.Literal {
	case "play":
		track, err := p.expect(token.IDENT)
		if err != nil {
			return nil, err
		}
		return &ast.MusicPlayNode{Track: track.Literal}, nil
	case "crossfade":
		track, err := p.expect(token.IDENT)
		if err != nil {
			return nil, err
		}
		return &ast.MusicCrossfadeNode{Track: track.Literal}, nil
	case "fadeout":
		return &ast.MusicFadeoutNode{}, nil
	default:
		return nil, fmt.Errorf("line %d: unknown music action %q", action.Line, action.Literal)
	}
}

// parseSfx parses: @sfx play <name>
func (p *Parser) parseSfx() (ast.Node, error) {
	p.advance() // consume "sfx"
	if _, err := p.expect(token.IDENT); err != nil { // consume "play"
		return nil, err
	}
	sound, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.SfxPlayNode{Sound: sound.Literal}, nil
}

// parseMinigame parses:
//
//	@minigame <id> <ATTR> "<description>" {
//	  <body>  // typically @if (rating.S) { ... } @else @if (rating.A) { ... } ...
//	}
//
// Description is a short English narrative tying the minigame to the
// scene. Body is standard MSS; branching on the result uses
// RatingCondition (rating.<grade>) in @if.
func (p *Parser) parseMinigame() (ast.Node, error) {
	p.advance() // consume "minigame"
	id, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	attr, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	desc, err := p.expect(token.STRING)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: @minigame requires a quoted description after the attribute", desc.Line, desc.Col)
	}
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}

	return &ast.MinigameNode{
		ID:          id.Literal,
		Attr:        attr.Literal,
		Description: desc.Literal,
		Body:        body,
	}, nil
}

// parseChoice parses: @choice { @option ... }
func (p *Parser) parseChoice() (ast.Node, error) {
	p.advance() // consume "choice"
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	node := &ast.ChoiceNode{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT && p.peek.Literal == "option" {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			node.Options = append(node.Options, opt)
		} else {
			p.advance()
		}
	}

	if len(node.Options) == 0 {
		return nil, fmt.Errorf("line %d col %d: @choice block has no @option entries", p.cur.Line, p.cur.Col)
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return node, nil
}

// parseOption parses: @option <ID> <brave|safe> "<text>" { body }
func (p *Parser) parseOption() (*ast.OptionNode, error) {
	p.advance() // consume AT
	p.advance() // consume "option"
	id, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	mode, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	text, err := p.expect(token.STRING)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	opt := &ast.OptionNode{
		ID:   id.Literal,
		Mode: mode.Literal,
		Text: text.Literal,
	}

	if mode.Literal == "brave" {
		err := p.parseBraveOptionBody(opt)
		if err != nil {
			return nil, err
		}
	} else {
		// Safe option: body is direct narrative content.
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		opt.Body = body
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return opt, nil
}

// parseBraveOptionBody parses a brave option body:
//
//	check { attr: X  dc: N }
//	<any MSS body statements — typically @if (check.success) { ... } @else { ... }>
//
// The check block must appear somewhere in the body (validator enforces
// its presence). Beyond that, body statements are plain MSS — authors
// use @if with the CheckCondition (check.success / check.fail) to route
// on the resolved check outcome.
func (p *Parser) parseBraveOptionBody(opt *ast.OptionNode) error {
	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.IDENT && p.cur.Literal == "check" {
			p.advance() // consume "check"
			if _, err := p.expect(token.LBRACE); err != nil {
				return err
			}
			check := &ast.CheckBlock{}
			// Parse key-value pairs inside check block.
			for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
				if p.cur.Type == token.IDENT {
					key := p.cur.Literal
					p.advance() // consume key
					if _, err := p.expect(token.COLON); err != nil {
						return err
					}
					switch key {
					case "attr":
						val, err := p.expect(token.IDENT)
						if err != nil {
							return err
						}
						check.Attr = val.Literal
					case "dc":
						val, err := p.expect(token.NUMBER)
						if err != nil {
							return err
						}
						dc, dcErr := strconv.Atoi(val.Literal)
						if dcErr != nil {
							return fmt.Errorf("line %d col %d: invalid dc value %q", val.Line, val.Col, val.Literal)
						}
						check.DC = dc
					default:
						p.advance() // skip unknown key values
					}
				} else {
					p.advance()
				}
			}
			if _, err := p.expect(token.RBRACE); err != nil {
				return err
			}
			opt.Check = check
		} else {
			node, err := p.parseStatement()
			if err != nil {
				return err
			}
			if node != nil {
				opt.Body = append(opt.Body, node)
			}
		}
	}
	return nil
}

// parseAffection parses: @affection <char> +/-N
func (p *Parser) parseAffection() (ast.Node, error) {
	p.advance() // consume "affection"
	char, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	delta, err := p.expect(token.SIGNED_NUMBER)
	if err != nil {
		return nil, err
	}
	return &ast.AffectionNode{Char: char.Literal, Delta: delta.Literal}, nil
}

// parseSignal parses: @signal <kind> <event>
//
// Kind is mandatory. The whitelist is deliberately open-ended — currently
// only "mark" is implemented, but the slot is kept in source syntax so
// future signal kinds can be added without a grammar break.
//
// "mark" emits a persistent boolean flag testable via @if (NAME).
// Triggering achievements is a separate directive (@achievement <id>),
// not a signal kind.
//
// Event may be a bare IDENT or a double-quoted STRING.
func (p *Parser) parseSignal() (ast.Node, error) {
	p.advance() // consume "signal"
	kindTok, err := p.expect(token.IDENT)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected signal kind after '@signal' (currently only 'mark' is supported), got %s (%q)",
			kindTok.Line, kindTok.Col, kindTok.Type, kindTok.Literal)
	}
	if !validSignalKinds[kindTok.Literal] {
		return nil, fmt.Errorf("line %d col %d: invalid signal kind %q (currently only 'mark' is supported)",
			kindTok.Line, kindTok.Col, kindTok.Literal)
	}
	var event string
	if p.cur.Type == token.STRING {
		event = p.cur.Literal
		p.advance()
	} else {
		tok, err := p.expect(token.IDENT)
		if err != nil {
			return nil, fmt.Errorf("line %d col %d: expected event name after '@signal %s', got %s (%q)",
				tok.Line, tok.Col, kindTok.Literal, tok.Type, tok.Literal)
		}
		event = tok.Literal
	}
	return &ast.SignalNode{Kind: kindTok.Literal, Event: event}, nil
}

// parseAchievement parses both achievement forms, selected by the presence
// of a following `{`:
//
//   - Declaration (hoisted to Episode.Achievements):
//     @achievement <id> {
//       name: "..."
//       rarity: <uncommon|rare|epic|legendary>
//       description: "..."
//     }
//
//   - Trigger (stays in body as an AchievementTriggerNode):
//     @achievement <id>
//
// The caller (parseEpisodeBody) inspects the returned node's type and
// hoists AchievementNode; AchievementTriggerNode stays inline as a step.
func (p *Parser) parseAchievement() (ast.Node, error) {
	// AT and "achievement" have already been consumed by parseDirective
	// (parseDirective calls advance() once before dispatching, then the
	// keyword `achievement` is still in p.cur). Consume the keyword now.
	p.advance() // consume "achievement"

	idTok, err := p.expect(token.IDENT)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected achievement id after '@achievement', got %s (%q)",
			idTok.Line, idTok.Col, idTok.Type, idTok.Literal)
	}

	// Bare form (trigger): no LBRACE follows → emit AchievementTriggerNode.
	if p.cur.Type != token.LBRACE {
		return &ast.AchievementTriggerNode{ID: idTok.Literal}, nil
	}

	// Block form (declaration): parse fields.
	p.advance() // consume LBRACE

	node := &ast.AchievementNode{ID: idTok.Literal}
	seen := map[string]bool{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type != token.IDENT {
			return nil, fmt.Errorf("line %d col %d: expected achievement field key, got %s (%q)",
				p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
		}
		key := p.cur.Literal
		if seen[key] {
			return nil, fmt.Errorf("line %d col %d: duplicate achievement field %q", p.cur.Line, p.cur.Col, key)
		}
		seen[key] = true
		p.advance() // consume key
		if _, err := p.expect(token.COLON); err != nil {
			return nil, err
		}
		switch key {
		case "name":
			val, err := p.expect(token.STRING)
			if err != nil {
				return nil, fmt.Errorf("line %d col %d: achievement name must be a quoted string", val.Line, val.Col)
			}
			node.Name = val.Literal
		case "description":
			val, err := p.expect(token.STRING)
			if err != nil {
				return nil, fmt.Errorf("line %d col %d: achievement description must be a quoted string", val.Line, val.Col)
			}
			node.Description = val.Literal
		case "rarity":
			val, err := p.expect(token.IDENT)
			if err != nil {
				return nil, fmt.Errorf("line %d col %d: achievement rarity must be an identifier", val.Line, val.Col)
			}
			if !validRarities[val.Literal] {
				return nil, fmt.Errorf("line %d col %d: invalid rarity %q (must be uncommon, rare, epic, or legendary)",
					val.Line, val.Col, val.Literal)
			}
			node.Rarity = val.Literal
		default:
			return nil, fmt.Errorf("line %d col %d: unknown achievement field %q (expected name, rarity, or description)",
				p.cur.Line, p.cur.Col, key)
		}
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}

	// All three fields required.
	if node.Name == "" {
		return nil, fmt.Errorf("achievement %q: missing required field 'name'", node.ID)
	}
	if node.Rarity == "" {
		return nil, fmt.Errorf("achievement %q: missing required field 'rarity'", node.ID)
	}
	if node.Description == "" {
		return nil, fmt.Errorf("achievement %q: missing required field 'description'", node.ID)
	}
	return node, nil
}

// parseButterfly parses: @butterfly "<desc>"
func (p *Parser) parseButterfly() (ast.Node, error) {
	p.advance() // consume "butterfly"
	desc, err := p.expect(token.STRING)
	if err != nil {
		return nil, err
	}
	return &ast.ButterflyNode{Description: desc.Literal}, nil
}

// parseIf parses: @if (condition) { body } [@else @if (...) { } | @else { }]
func (p *Parser) parseIf() (ast.Node, error) {
	p.advance() // consume "if"

	cond, err := p.readCondition()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}
	then, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}

	node := &ast.IfNode{
		Condition: cond,
		Then:      then,
	}

	// Check for @else @if (chain) or @else { } (plain).
	if p.cur.Type == token.AT && p.peek.Literal == "else" {
		p.advance() // consume AT
		p.advance() // consume "else"

		if p.cur.Type == token.AT && p.peek.Literal == "if" {
			// @else @if — recursive chain
			p.advance() // consume AT so parseIf sees "if" as cur
			elseIf, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			node.Else = []ast.Node{elseIf}
		} else {
			// plain @else { }
			if _, err := p.expect(token.LBRACE); err != nil {
				return nil, err
			}
			els, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(token.RBRACE); err != nil {
				return nil, err
			}
			node.Else = els
		}
	}

	return node, nil
}

// readCondition reads a parenthesized condition and parses it into a typed
// Condition AST. Shared by body @if and gate @if.
//
// Grammar (recursive descent, left-associative):
//
//	expr       = or_expr
//	or_expr    = and_expr ( "||" and_expr )*
//	and_expr   = primary ( "&&" primary )*
//	primary    = "(" expr ")"
//	           | choice       ( IDENT "." ( "success" | "fail" | "any" ) )
//	           | comparison   ( operand OP ( NUMBER | SIGNED_NUMBER ) )
//	           | influence    ( "influence" STRING | STRING )
//	           | flag         ( IDENT )
//	operand    = IDENT "." IDENT  (affection.<char>)
//	           | IDENT             (bare engine-managed value, e.g. "san")
func (p *Parser) readCondition() (ast.Condition, error) {
	if _, err := p.expect(token.LPAREN); err != nil {
		return nil, err
	}

	cond, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.RPAREN); err != nil {
		return nil, err
	}
	return cond, nil
}

// parseOrExpr parses an 'or_expr' — one or more primaries joined by "||".
// "||" has lower precedence than "&&".
func (p *Parser) parseOrExpr() (ast.Condition, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == token.OR {
		p.advance() // consume ||
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = &ast.CompoundCondition{Op: "||", Left: left, Right: right}
	}
	return left, nil
}

// parseAndExpr parses an 'and_expr' — one or more primaries joined by "&&".
func (p *Parser) parseAndExpr() (ast.Condition, error) {
	left, err := p.parseConditionPrimary()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == token.AND {
		p.advance() // consume &&
		right, err := p.parseConditionPrimary()
		if err != nil {
			return nil, err
		}
		left = &ast.CompoundCondition{Op: "&&", Left: left, Right: right}
	}
	return left, nil
}

// parseConditionPrimary parses a single leaf condition or a parenthesized
// sub-expression.
func (p *Parser) parseConditionPrimary() (ast.Condition, error) {
	// Parenthesized sub-expression.
	if p.cur.Type == token.LPAREN {
		p.advance() // consume (
		inner, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(token.RPAREN); err != nil {
			return nil, err
		}
		return inner, nil
	}

	// Influence: bare STRING — the whole primary is a single string literal.
	if p.cur.Type == token.STRING {
		tok := p.cur
		p.advance()
		return &ast.InfluenceCondition{Description: tok.Literal}, nil
	}

	if p.cur.Type != token.IDENT {
		return nil, fmt.Errorf("line %d col %d: expected condition, got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}

	// Influence: "influence" STRING
	if p.cur.Literal == "influence" && p.peek.Type == token.STRING {
		p.advance() // consume "influence"
		tok := p.cur
		p.advance() // consume STRING
		return &ast.InfluenceCondition{Description: tok.Literal}, nil
	}

	// Dotted forms: <first>.<second>. Five shapes recognised:
	//   - check.success / check.fail   → CheckCondition   (context-local to brave option body)
	//   - rating.<grade>               → RatingCondition  (context-local to minigame body)
	//   - affection.<char> <op> <N>    → ComparisonCondition with affection operand
	//   - <option_id>.success|fail|any → ChoiceCondition  (retrospective query from outside option)
	//   - anything else dotted         → error
	if p.peek.Type == token.DOT {
		firstTok := p.cur
		p.advance() // consume first IDENT
		p.advance() // consume DOT
		if p.cur.Type != token.IDENT {
			return nil, fmt.Errorf("line %d col %d: expected identifier after '.', got %s (%q)",
				p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
		}
		secondTok := p.cur
		p.advance() // consume second IDENT

		// Context-local: check.success / check.fail.
		if firstTok.Literal == "check" {
			if secondTok.Literal != "success" && secondTok.Literal != "fail" {
				return nil, fmt.Errorf("line %d col %d: check.<result> must be 'success' or 'fail', got %q",
					secondTok.Line, secondTok.Col, secondTok.Literal)
			}
			return &ast.CheckCondition{Result: secondTok.Literal}, nil
		}

		// Context-local: rating.<grade>.
		if firstTok.Literal == "rating" {
			return &ast.RatingCondition{Grade: secondTok.Literal}, nil
		}

		// affection.<char> <op> <N> — comparison with affection operand.
		if firstTok.Literal == "affection" {
			if !isComparisonOp(p.cur.Type) {
				return nil, fmt.Errorf("line %d col %d: affection.%s must be followed by a comparison operator",
					secondTok.Line, secondTok.Col, secondTok.Literal)
			}
			op := operatorLiteral(p.cur.Type)
			p.advance() // consume operator
			right, err := p.readNumberLiteral()
			if err != nil {
				return nil, err
			}
			return &ast.ComparisonCondition{
				Left: ast.ComparisonOperand{
					Kind: ast.OperandAffection,
					Char: secondTok.Literal,
				},
				Op:    op,
				Right: right,
			}, nil
		}

		// Otherwise: choice condition — <option_id>.success|fail|any.
		result := secondTok.Literal
		if result != "success" && result != "fail" && result != "any" {
			return nil, fmt.Errorf("line %d col %d: choice condition result must be 'success', 'fail', or 'any', got %q",
				secondTok.Line, secondTok.Col, result)
		}
		return &ast.ChoiceCondition{Option: firstTok.Literal, Result: result}, nil
	}

	// Bare IDENT: either comparison (IDENT op N) or flag (IDENT alone).
	nameTok := p.cur
	p.advance() // consume IDENT

	if isComparisonOp(p.cur.Type) {
		op := operatorLiteral(p.cur.Type)
		p.advance() // consume operator
		right, err := p.readNumberLiteral()
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonCondition{
			Left: ast.ComparisonOperand{
				Kind: ast.OperandValue,
				Name: nameTok.Literal,
			},
			Op:    op,
			Right: right,
		}, nil
	}

	return &ast.FlagCondition{Name: nameTok.Literal}, nil
}

// isComparisonOp reports whether a token type is a comparison operator.
func isComparisonOp(t token.Type) bool {
	switch t {
	case token.GTE, token.LTE, token.GT, token.LT, token.EQ, token.NEQ:
		return true
	}
	return false
}

// operatorLiteral returns the canonical source-form of a comparison operator.
func operatorLiteral(t token.Type) string {
	switch t {
	case token.GTE:
		return ">="
	case token.LTE:
		return "<="
	case token.GT:
		return ">"
	case token.LT:
		return "<"
	case token.EQ:
		return "=="
	case token.NEQ:
		return "!="
	}
	return ""
}

// readNumberLiteral consumes a NUMBER or SIGNED_NUMBER token and returns
// its integer value.
func (p *Parser) readNumberLiteral() (int, error) {
	if p.cur.Type != token.NUMBER && p.cur.Type != token.SIGNED_NUMBER {
		return 0, fmt.Errorf("line %d col %d: expected integer on right side of comparison, got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}
	tok := p.cur
	p.advance()
	n, err := strconv.Atoi(tok.Literal)
	if err != nil {
		return 0, fmt.Errorf("line %d col %d: invalid integer literal %q", tok.Line, tok.Col, tok.Literal)
	}
	return n, nil
}

// parseEnding parses: @ending <type>
// Valid types: complete | to_be_continued | bad_ending.
func (p *Parser) parseEnding() (*ast.EndingNode, error) {
	p.advance() // consume AT
	p.advance() // consume "ending"
	kind, err := p.expect(token.IDENT)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected ending type after '@ending', got %s (%q)",
			kind.Line, kind.Col, kind.Type, kind.Literal)
	}
	if !validEndingTypes[kind.Literal] {
		return nil, fmt.Errorf("line %d col %d: invalid @ending type %q (must be one of: complete, to_be_continued, bad_ending)",
			kind.Line, kind.Col, kind.Literal)
	}
	return &ast.EndingNode{Type: kind.Literal}, nil
}

// parseLabel parses: @label <name>
func (p *Parser) parseLabel() (ast.Node, error) {
	p.advance() // consume "label"
	name, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.LabelNode{Name: name.Literal}, nil
}

// parseGoto parses: @goto <name>
func (p *Parser) parseGoto() (ast.Node, error) {
	p.advance() // consume "goto"
	name, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.GotoNode{Name: name.Literal}, nil
}

// parsePause parses: @pause for <N>
func (p *Parser) parsePause() (ast.Node, error) {
	p.advance() // consume "pause"
	// Expect "for"
	if p.cur.Type != token.IDENT || p.cur.Literal != "for" {
		return nil, fmt.Errorf("line %d col %d: expected 'for' after 'pause', got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}
	p.advance() // consume "for"
	n, err := p.expect(token.NUMBER)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected number after 'pause for', got %s (%q)",
			p.cur.Line, p.cur.Col, p.cur.Type, p.cur.Literal)
	}
	clicks, err2 := strconv.Atoi(n.Literal)
	if err2 != nil {
		return nil, fmt.Errorf("line %d col %d: invalid pause count %q", n.Line, n.Col, n.Literal)
	}
	if clicks < 1 {
		clicks = 1
	}
	return &ast.PauseNode{Clicks: clicks}, nil
}

// parseCharDirective parses character directives like:
//
//	@mauricio show neutral at center
//	@mauricio hide fade
//	@mauricio expr angry
//	@mauricio move to left
//	@mauricio bubble heart
func (p *Parser) parseCharDirective(char string) (ast.Node, error) {
	p.advance() // consume character name
	action, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	switch action.Literal {
	case "show":
		return p.parseCharShow(char)
	case "hide":
		return p.parseCharHide(char)
	case "look":
		return p.parseCharLook(char)
	case "move":
		return p.parseCharMove(char)
	case "bubble":
		return p.parseCharBubble(char)
	default:
		return nil, fmt.Errorf("line %d: unknown character action %q for %q",
			action.Line, action.Literal, char)
	}
}

// parseCharShow parses: show <pose> at <position> [transition]
func (p *Parser) parseCharShow(char string) (ast.Node, error) {
	pose, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	// "at" keyword
	if _, err := p.expect(token.IDENT); err != nil { // consume "at"
		return nil, err
	}
	pos, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.CharShowNode{
		Char:     char,
		Look:     pose.Literal,
		Position: pos.Literal,
	}
	// Optional transition.
	if p.cur.Type == token.IDENT && !p.isDirectiveStart() {
		node.Transition = p.cur.Literal
		p.advance()
	}
	return node, nil
}

// parseCharHide parses: hide [transition]
func (p *Parser) parseCharHide(char string) (ast.Node, error) {
	node := &ast.CharHideNode{Char: char}
	if p.cur.Type == token.IDENT && !p.isDirectiveStart() {
		node.Transition = p.cur.Literal
		p.advance()
	}
	return node, nil
}

// parseCharLook parses: look <pose> [transition]
func (p *Parser) parseCharLook(char string) (ast.Node, error) {
	pose, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.CharLookNode{
		Char: char,
		Look: pose.Literal,
	}
	if p.cur.Type == token.IDENT && !p.isDirectiveStart() {
		node.Transition = p.cur.Literal
		p.advance()
	}
	return node, nil
}

// parseCharMove parses: move to <position>
func (p *Parser) parseCharMove(char string) (ast.Node, error) {
	// consume "to"
	if _, err := p.expect(token.IDENT); err != nil {
		return nil, err
	}
	pos, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.CharMoveNode{
		Char:     char,
		Position: pos.Literal,
	}, nil
}

// parseCharBubble parses: bubble <type>
func (p *Parser) parseCharBubble(char string) (ast.Node, error) {
	bubbleType, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.CharBubbleNode{
		Char:       char,
		BubbleType: bubbleType.Literal,
	}, nil
}

// parseGateBlock parses: @gate { @if (cond): @next target ... @else: @next target }
func (p *Parser) parseGateBlock() (*ast.GateBlock, error) {
	p.advance() // consume AT
	p.advance() // consume "gate"
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	block := &ast.GateBlock{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT {
			switch p.peek.Literal {
			case "if":
				route, err := p.parseGateIf()
				if err != nil {
					return nil, err
				}
				block.Routes = append(block.Routes, route)
			case "else":
				route, err := p.parseGateElse()
				if err != nil {
					return nil, err
				}
				block.Routes = append(block.Routes, route)
			case "next":
				route, err := p.parseGateNext()
				if err != nil {
					return nil, err
				}
				block.Routes = append(block.Routes, route)
			default:
				p.advance()
			}
		} else {
			p.advance()
		}
	}

	if len(block.Routes) == 0 {
		return nil, fmt.Errorf("line %d col %d: @gate block has no routes", p.cur.Line, p.cur.Col)
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return block, nil
}

// parseGateIf parses: @if (<condition>): @next <target>
func (p *Parser) parseGateIf() (*ast.GateRoute, error) {
	p.advance() // consume AT
	p.advance() // consume "if"

	cond, err := p.readCondition()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.COLON); err != nil {
		return nil, err
	}

	target, err := p.parseGateNextTarget()
	if err != nil {
		return nil, err
	}

	return &ast.GateRoute{Condition: cond, Target: target}, nil
}

// parseGateElse parses: @else @if (...): @next ... OR @else: @next ...
func (p *Parser) parseGateElse() (*ast.GateRoute, error) {
	p.advance() // consume AT
	p.advance() // consume "else"

	// Check for @else @if (chained condition).
	if p.cur.Type == token.AT && p.peek.Literal == "if" {
		return p.parseGateIf()
	}

	// Plain @else: @next <target>
	if _, err := p.expect(token.COLON); err != nil {
		return nil, err
	}

	target, err := p.parseGateNextTarget()
	if err != nil {
		return nil, err
	}

	return &ast.GateRoute{Condition: nil, Target: target}, nil
}

// parseGateNext parses a bare: @next <target> (unconditional jump)
func (p *Parser) parseGateNext() (*ast.GateRoute, error) {
	target, err := p.parseGateNextTarget()
	if err != nil {
		return nil, err
	}
	return &ast.GateRoute{Condition: nil, Target: target}, nil
}

// parseGateNextTarget consumes @next <target> and returns the target string.
func (p *Parser) parseGateNextTarget() (string, error) {
	if _, err := p.expect(token.AT); err != nil {
		return "", err
	}
	kw, err := p.expect(token.IDENT)
	if err != nil {
		return "", err
	}
	if kw.Literal != "next" {
		return "", fmt.Errorf("line %d col %d: expected 'next', got %q", kw.Line, kw.Col, kw.Literal)
	}
	target, err := p.expect(token.IDENT)
	if err != nil {
		return "", err
	}
	return target.Literal, nil
}

// isDirectiveStart returns true if cur looks like the start of a new directive
// (i.e. the next thing is @something or NAME:dialogue). Used to stop consuming
// optional trailing IDENTs.
func (p *Parser) isDirectiveStart() bool {
	// We only need this for optional-ident lookahead. If the current IDENT
	// is followed by AT/AMPERSAND or followed by COLON (dialogue), it's a new statement.
	if p.peek.Type == token.AT || p.peek.Type == token.AMPERSAND {
		return false // The current ident is before an @/& — it's likely a param.
	}
	if p.peek.Type == token.COLON {
		// Current IDENT + COLON = dialogue start. Don't consume.
		return true
	}
	if p.peek.Type == token.LBRACKET {
		// Current IDENT + LBRACKET = dialogue sugar (CHARACTER [look]: text). Don't consume.
		return true
	}
	return false
}
