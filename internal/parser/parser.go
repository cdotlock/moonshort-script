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
	"xp": true, "san": true, "affection": true, "signal": true,
	"butterfly": true, "if": true, "else": true, "label": true,
	"goto": true, "gates": true, "episode": true, "on": true,
	"option": true, "gate": true, "default": true,
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
	episode.Gates = gates

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return episode, nil
}

// parseEpisodeBody parses body nodes until RBRACE, handling @gates specially.
func (p *Parser) parseEpisodeBody() ([]ast.Node, *ast.GatesBlock, error) {
	var body []ast.Node
	var gates *ast.GatesBlock

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		// Drain any pending node queued by parseDialogueWithExpr before
		// checking for the @gates short-circuit path.
		if p.pending != nil {
			body = append(body, p.pending)
			p.pending = nil
			continue
		}
		if p.cur.Type == token.AT && p.peek.Literal == "gates" {
			g, err := p.parseGatesBlock()
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

	// Return CharExprNode first.
	return &ast.CharExprNode{
		Char: charID,
		Pose: pose,
	}, nil
}

// parseDirective parses an @-prefixed directive.
func (p *Parser) parseDirective() (ast.Node, error) {
	p.advance() // consume AT
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
	case "xp":
		return p.parseXp()
	case "san":
		return p.parseSan()
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

// parseXp parses: @xp +/-N
func (p *Parser) parseXp() (ast.Node, error) {
	p.advance() // consume "xp"
	delta, err := p.expect(token.SIGNED_NUMBER)
	if err != nil {
		return nil, err
	}
	return &ast.XpNode{Delta: delta.Literal}, nil
}

// parseSan parses: @san +/-N
func (p *Parser) parseSan() (ast.Node, error) {
	p.advance() // consume "san"
	delta, err := p.expect(token.SIGNED_NUMBER)
	if err != nil {
		return nil, err
	}
	return &ast.SanNode{Delta: delta.Literal}, nil
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

// parseIf parses: @if <condition> { body } [@else { body }]
func (p *Parser) parseIf() (ast.Node, error) {
	p.advance() // consume "if"

	// Collect the condition as a raw string until we hit LBRACE.
	condition := p.readUntilLBrace()

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
		Condition: condition,
		Then:      then,
	}

	// Check for @else.
	if p.cur.Type == token.AT && p.peek.Literal == "else" {
		p.advance() // consume AT
		p.advance() // consume "else"
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

	return node, nil
}

// readUntilLBrace collects token literals until LBRACE, building a raw condition string.
func (p *Parser) readUntilLBrace() string {
	var parts []string
	for p.cur.Type != token.LBRACE && p.cur.Type != token.EOF {
		parts = append(parts, p.cur.Literal)
		p.advance()
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
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
	case "expr":
		return p.parseCharExpr(char)
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
		Pose:     pose.Literal,
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

// parseCharExpr parses: expr <pose> [transition]
func (p *Parser) parseCharExpr(char string) (ast.Node, error) {
	pose, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	node := &ast.CharExprNode{
		Char: char,
		Pose: pose.Literal,
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

// parseGatesBlock parses: @gates { @gate ... @default ... }
func (p *Parser) parseGatesBlock() (*ast.GatesBlock, error) {
	p.advance() // consume AT
	p.advance() // consume "gates"
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	block := &ast.GatesBlock{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT {
			if p.peek.Literal == "gate" {
				gate, err := p.parseGate()
				if err != nil {
					return nil, err
				}
				block.Gates = append(block.Gates, gate)
			} else if p.peek.Literal == "default" {
				gate, err := p.parseDefault()
				if err != nil {
					return nil, err
				}
				block.Gates = append(block.Gates, gate)
			} else {
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

// parseGate parses: @gate <key> { type: ... trigger: ... condition: ... }
func (p *Parser) parseGate() (*ast.Gate, error) {
	p.advance() // consume AT
	p.advance() // consume "gate"
	target, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}

	gate := &ast.Gate{Target: target.Literal}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.IDENT {
			key := p.cur.Literal
			p.advance()
			if _, err := p.expect(token.COLON); err != nil {
				return nil, err
			}
			switch key {
			case "type":
				val, err := p.expect(token.IDENT)
				if err != nil {
					return nil, err
				}
				gate.GateType = val.Literal
			case "trigger":
				gate.Trigger = &ast.GateTrigger{}
				// Read trigger values — could be "option_A success" etc.
				for p.cur.Type == token.IDENT {
					lit := p.cur.Literal
					// Check if it looks like an option reference (option_X).
					if len(lit) > 7 && lit[:7] == "option_" {
						gate.Trigger.OptionID = lit[7:]
					} else if lit == "success" || lit == "fail" {
						gate.Trigger.CheckResult = lit
					} else {
						gate.Trigger.OptionID = lit
					}
					p.advance()
				}
			case "condition":
				// Read condition as raw text until next key or RBRACE.
				cond := p.readConditionValue()
				gate.Condition = cond
			}
		} else {
			p.advance()
		}
	}

	if _, err := p.expect(token.RBRACE); err != nil {
		return nil, err
	}
	return gate, nil
}

// readConditionValue reads a condition expression inside a gate block.
// Stops at the next key: pattern (IDENT COLON at block level) or RBRACE.
func (p *Parser) readConditionValue() string {
	var parts []string
	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		// Lookahead: if current is IDENT and peek is COLON, this is a new key.
		if p.cur.Type == token.IDENT && p.peek.Type == token.COLON {
			break
		}
		parts = append(parts, p.cur.Literal)
		p.advance()
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}

// parseDefault parses: @default <key>
func (p *Parser) parseDefault() (*ast.Gate, error) {
	p.advance() // consume AT
	p.advance() // consume "default"
	target, err := p.expect(token.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.Gate{
		Target:   target.Literal,
		GateType: "default",
	}, nil
}

// isDirectiveStart returns true if cur looks like the start of a new directive
// (i.e. the next thing is @something or NAME:dialogue). Used to stop consuming
// optional trailing IDENTs.
func (p *Parser) isDirectiveStart() bool {
	// We only need this for optional-ident lookahead. If the current IDENT
	// is followed by AT or followed by COLON (dialogue), it's a new statement.
	if p.peek.Type == token.AT {
		return false // The current ident is before an AT — it's likely a param.
	}
	if p.peek.Type == token.COLON {
		// Current IDENT + COLON = dialogue start. Don't consume.
		return true
	}
	return false
}
