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
	"goto": true, "episode": true, "on": true,
	"option": true, "gate": true, "next": true, "pause": true,
}

// Parser consumes tokens from a Lexer and produces an AST.
type Parser struct {
	l       *lexer.Lexer
	cur     token.Token
	peek    token.Token
	pending ast.Node // set by parseDialogueWithExpr; drained on next parseStatement call
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

	body, gates, err := p.parseEpisodeBody()
	if err != nil {
		return nil, err
	}
	episode.Body = body
	episode.Gate = gates

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return episode, nil
}

// parseEpisodeBody parses body nodes until RBRACE, handling @gate specially.
func (p *Parser) parseEpisodeBody() ([]ast.Node, *ast.GateBlock, error) {
	var body []ast.Node
	var gates *ast.GateBlock

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		// Drain any pending node queued by parseDialogueWithExpr before
		// checking for the @gate short-circuit path.
		if p.pending != nil {
			body = append(body, p.pending)
			p.pending = nil
			continue
		}
		if p.cur.Type == token.AT && p.peek.Literal == "gate" {
			g, err := p.parseGateBlock()
			if err != nil {
				return nil, nil, err
			}
			gates = g
			continue
		}
		node, err := p.parseStatement()
		if err != nil {
			return nil, nil, err
		}
		if node != nil {
			body = append(body, node)
		}
	}
	// Drain final pending node if any.
	if p.pending != nil {
		body = append(body, p.pending)
		p.pending = nil
	}
	return body, gates, nil
}

// parseBlock parses body nodes until RBRACE (but does NOT consume the RBRACE).
func (p *Parser) parseBlock() ([]ast.Node, error) {
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
	node, err := p.parseDirective() // parseDirective consumes AT/AMPERSAND via advance
	if err != nil {
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

// parseCg parses: @cg show <name> [transition] { body }
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
	if p.cur.Type == token.LBRACE {
		p.advance() // consume LBRACE
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		node.Body = body
		if _, err := p.expect(token.RBRACE); err != nil {
			return nil, err
		}
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

// parseMinigame parses: @minigame <id> <ATTR> { @on <ratings> { body } ... }
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
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	node := &ast.MinigameNode{
		ID:       id.Literal,
		Attr:     attr.Literal,
		OnResult: make(map[string][]ast.Node),
	}

	// Parse @on blocks.
	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT && p.peek.Literal == "on" {
			p.advance() // consume AT
			p.advance() // consume "on"
			// Collect rating identifiers until LBRACE.
			var ratings []string
			for p.cur.Type == token.IDENT {
				ratings = append(ratings, p.cur.Literal)
				p.advance()
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
			// Map each rating to the same body.
			key := ""
			for i, r := range ratings {
				if i > 0 {
					key += " "
				}
				key += r
			}
			node.OnResult[key] = body
		} else {
			p.advance() // skip unexpected tokens
		}
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return node, nil
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

// parseBraveOptionBody parses: check { attr: X  dc: N } @on success { } @on fail { }
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
						dc, _ := strconv.Atoi(val.Literal)
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
		} else if p.cur.Type == token.AT && p.peek.Literal == "on" {
			p.advance() // consume AT
			p.advance() // consume "on"
			result, err := p.expect(token.IDENT) // "success" or "fail"
			if err != nil {
				return err
			}
			if _, err := p.expect(token.LBRACE); err != nil {
				return err
			}
			body, err := p.parseBlock()
			if err != nil {
				return err
			}
			if _, err := p.expect(token.RBRACE); err != nil {
				return err
			}
			switch result.Literal {
			case "success":
				opt.OnSuccess = body
			case "fail":
				opt.OnFail = body
			}
		} else {
			// Could be other body nodes.
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

// parseSignal parses: @signal <event>
func (p *Parser) parseSignal() (ast.Node, error) {
	p.advance() // consume "signal"
	event, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.SignalNode{Event: event.Literal}, nil
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

// readCondition reads a parenthesized condition and classifies it into a Condition.
// Shared by body @if and gate @if.
func (p *Parser) readCondition() (*ast.Condition, error) {
	if _, err := p.expect(token.LPAREN); err != nil {
		return nil, err
	}

	var toks []token.Token
	for p.cur.Type != token.RPAREN && p.cur.Type != token.EOF {
		toks = append(toks, p.cur)
		p.advance()
	}

	if _, err := p.expect(token.RPAREN); err != nil {
		return nil, err
	}

	return classifyCondition(toks), nil
}

// classifyCondition determines the condition type from a token list.
func classifyCondition(toks []token.Token) *ast.Condition {
	expr := buildExpr(toks)

	// 1. Compound: has && or ||
	for _, t := range toks {
		if t.Type == token.AND || t.Type == token.OR {
			return &ast.Condition{Type: "compound", Expr: expr}
		}
	}

	// 2. Influence: influence "description"
	if len(toks) >= 2 && toks[0].Type == token.IDENT && toks[0].Literal == "influence" && toks[1].Type == token.STRING {
		return &ast.Condition{Type: "influence", Description: toks[1].Literal}
	}

	// 3. Choice: IDENT.result (e.g., A.fail, B.success)
	if len(toks) == 3 && toks[0].Type == token.IDENT && toks[1].Type == token.DOT && toks[2].Type == token.IDENT {
		result := toks[2].Literal
		if result == "success" || result == "fail" {
			return &ast.Condition{Type: "choice", Option: toks[0].Literal, Result: result}
		}
	}

	// 4. Comparison: has operator
	for _, t := range toks {
		switch t.Type {
		case token.GTE, token.LTE, token.GT, token.LT, token.EQ, token.NEQ:
			return &ast.Condition{Type: "comparison", Expr: expr}
		}
	}

	// 5. Flag: single identifier
	return &ast.Condition{Type: "flag", Name: expr}
}

// buildExpr joins condition tokens into a string, preserving dot notation
// (no spaces around dots).
func buildExpr(toks []token.Token) string {
	var sb strings.Builder
	for i, t := range toks {
		if i > 0 {
			prev := toks[i-1]
			if t.Type != token.DOT && prev.Type != token.DOT {
				sb.WriteString(" ")
			}
		}
		if t.Type == token.STRING {
			sb.WriteString(`"`)
			sb.WriteString(t.Literal)
			sb.WriteString(`"`)
		} else {
			sb.WriteString(t.Literal)
		}
	}
	return sb.String()
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
	clicks, _ := strconv.Atoi(n.Literal)
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
	return false
}
