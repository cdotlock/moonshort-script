package lexer

import (
	"strings"
	"unicode"

	"github.com/cdotlock/moonshort-script/internal/token"
)

// Lexer tokenizes an NRS script source string.
type Lexer struct {
	src  []rune
	pos  int // current read position
	line int
	col  int
}

// New creates a Lexer for the given source text.
func New(src string) *Lexer {
	return &Lexer{
		src:  []rune(src),
		pos:  0,
		line: 1,
		col:  1,
	}
}

// peek returns the rune at the current position without advancing.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

// peekAt returns the rune at current+offset without advancing.
func (l *Lexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

// advance returns the current rune and moves the position forward.
func (l *Lexer) advance() rune {
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

// makeToken creates a token at the current lexer position.
func (l *Lexer) makeToken(t token.Type, literal string, line, col int) token.Token {
	return token.Token{Type: t, Literal: literal, Line: line, Col: col}
}

// skipSpaces skips spaces and tabs (not newlines).
func (l *Lexer) skipSpaces() {
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

// readString reads a double-quoted string, returning content without quotes.
func (l *Lexer) readString(line, col int) token.Token {
	l.advance() // consume opening "
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == '"' {
			l.advance() // consume closing "
			break
		}
		if ch == '\n' {
			break // unterminated string
		}
		sb.WriteRune(l.advance())
	}
	return l.makeToken(token.STRING, sb.String(), line, col)
}

// readComment reads from // to end of line, returning content after //.
func (l *Lexer) readComment(line, col int) token.Token {
	l.advance() // first /
	l.advance() // second /
	var sb strings.Builder
	for l.pos < len(l.src) && l.peek() != '\n' {
		sb.WriteRune(l.advance())
	}
	return l.makeToken(token.COMMENT, strings.TrimSpace(sb.String()), line, col)
}

// isIdentStart returns true for runes that can start an NRS identifier.
func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isIdentContinue returns true for runes that can continue an NRS identifier
// (letters, digits, underscore, hyphen, forward-slash). Colon is handled
// separately: it is only consumed as part of an identifier when immediately
// followed by another word character (e.g. "main:01"), distinguishing it from
// the standalone colon in "MAURICIO: dialogue text".
func isIdentContinue(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) ||
		ch == '_' || ch == '-' || ch == '/'
}

// readIdent reads an identifier token. Colons embedded between word chars
// (like "main:01") are included in the identifier; a trailing colon is not.
func (l *Lexer) readIdent(line, col int) token.Token {
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.peek()
		if isIdentContinue(ch) {
			sb.WriteRune(l.advance())
			continue
		}
		// Consume an embedded colon only when the next char is a word char.
		if ch == ':' && l.pos+1 < len(l.src) && isIdentContinue(l.src[l.pos+1]) {
			sb.WriteRune(l.advance()) // consume ':'
			continue
		}
		break
	}
	return l.makeToken(token.IDENT, sb.String(), line, col)
}

// readNumber reads a plain non-negative integer.
func (l *Lexer) readNumber(line, col int) token.Token {
	var sb strings.Builder
	for l.pos < len(l.src) && unicode.IsDigit(l.peek()) {
		sb.WriteRune(l.advance())
	}
	return l.makeToken(token.NUMBER, sb.String(), line, col)
}

// readSignedNumber reads a +N or -N token (sign already consumed as ch).
func (l *Lexer) readSignedNumber(sign rune, line, col int) token.Token {
	var sb strings.Builder
	sb.WriteRune(sign)
	for l.pos < len(l.src) && unicode.IsDigit(l.peek()) {
		sb.WriteRune(l.advance())
	}
	return l.makeToken(token.SIGNED_NUMBER, sb.String(), line, col)
}

// NextToken returns the next token from the source.
func (l *Lexer) NextToken() token.Token {
	l.skipSpaces()

	if l.pos >= len(l.src) {
		return l.makeToken(token.EOF, "", l.line, l.col)
	}

	line, col := l.line, l.col
	ch := l.peek()

	switch {
	case ch == '\n':
		l.advance()
		return l.makeToken(token.NEWLINE, "\n", line, col)

	case ch == '@':
		l.advance()
		return l.makeToken(token.AT, "@", line, col)

	case ch == '{':
		l.advance()
		return l.makeToken(token.LBRACE, "{", line, col)

	case ch == '}':
		l.advance()
		return l.makeToken(token.RBRACE, "}", line, col)

	case ch == '[':
		l.advance()
		return l.makeToken(token.LBRACKET, "[", line, col)

	case ch == ']':
		l.advance()
		return l.makeToken(token.RBRACKET, "]", line, col)

	case ch == '(':
		l.advance()
		return l.makeToken(token.LPAREN, "(", line, col)

	case ch == ')':
		l.advance()
		return l.makeToken(token.RPAREN, ")", line, col)

	case ch == '.':
		l.advance()
		return l.makeToken(token.DOT, ".", line, col)

	case ch == ':':
		l.advance()
		return l.makeToken(token.COLON, ":", line, col)

	case ch == '"':
		return l.readString(line, col)

	case ch == '/' && l.peekAt(1) == '/':
		return l.readComment(line, col)

	case ch == '>' && l.peekAt(1) == '=':
		l.advance(); l.advance()
		return l.makeToken(token.GTE, ">=", line, col)

	case ch == '<' && l.peekAt(1) == '=':
		l.advance(); l.advance()
		return l.makeToken(token.LTE, "<=", line, col)

	case ch == '=' && l.peekAt(1) == '=':
		l.advance(); l.advance()
		return l.makeToken(token.EQ, "==", line, col)

	case ch == '!' && l.peekAt(1) == '=':
		l.advance(); l.advance()
		return l.makeToken(token.NEQ, "!=", line, col)

	case ch == '&' && l.peekAt(1) == '&':
		l.advance(); l.advance()
		return l.makeToken(token.AND, "&&", line, col)

	case ch == '&':
		l.advance()
		return l.makeToken(token.AMPERSAND, "&", line, col)

	case ch == '|' && l.peekAt(1) == '|':
		l.advance(); l.advance()
		return l.makeToken(token.OR, "||", line, col)

	case ch == '>':
		l.advance()
		return l.makeToken(token.GT, ">", line, col)

	case ch == '<':
		l.advance()
		return l.makeToken(token.LT, "<", line, col)

	// Signed numbers: +N or -N when followed immediately by a digit
	case (ch == '+' || ch == '-') && unicode.IsDigit(l.peekAt(1)):
		sign := l.advance()
		return l.readSignedNumber(sign, line, col)

	case unicode.IsDigit(ch):
		return l.readNumber(line, col)

	case isIdentStart(ch):
		return l.readIdent(line, col)

	default:
		l.advance()
		return l.makeToken(token.ILLEGAL, string(ch), line, col)
	}
}

// ReadDialogueText is called by the parser immediately after it has consumed
// IDENT COLON. It reads the remainder of the current line as a single STRING
// token (unquoted dialogue text), trimming leading/trailing whitespace.
func (l *Lexer) ReadDialogueText() token.Token {
	line, col := l.line, l.col
	// skip leading spaces
	l.skipSpaces()
	var sb strings.Builder
	for l.pos < len(l.src) && l.peek() != '\n' {
		sb.WriteRune(l.advance())
	}
	return l.makeToken(token.STRING, strings.TrimRight(sb.String(), " \t\r"), line, col)
}

// Tokenize is a convenience helper that returns all tokens (excluding EOF).
func Tokenize(src string) []token.Token {
	l := New(src)
	var tokens []token.Token
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		tokens = append(tokens, tok)
	}
	return tokens
}
