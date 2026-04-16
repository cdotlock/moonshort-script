package token

// Type identifies the kind of token.
type Type string

const (
	// Special tokens
	EOF     Type = "EOF"
	ILLEGAL Type = "ILLEGAL"
	NEWLINE Type = "NEWLINE"

	// Literals
	STRING        Type = "STRING"
	IDENT         Type = "IDENT"
	NUMBER        Type = "NUMBER"
	SIGNED_NUMBER Type = "SIGNED_NUMBER"

	// Delimiters
	LBRACE   Type = "LBRACE"
	RBRACE   Type = "RBRACE"
	LBRACKET Type = "LBRACKET" // [
	RBRACKET Type = "RBRACKET" // ]
	LPAREN   Type = "LPAREN"   // (
	RPAREN   Type = "RPAREN"   // )

	// Operators / comparisons
	GTE Type = "GTE" // >=
	LTE Type = "LTE" // <=
	GT  Type = "GT"  // >
	LT  Type = "LT"  // <
	EQ  Type = "EQ"  // ==
	NEQ Type = "NEQ" // !=
	AND Type = "AND" // &&
	OR  Type = "OR"  // ||

	// Punctuation
	AT      Type = "AT"      // @
	COLON   Type = "COLON"   // :
	DOT     Type = "DOT"     // .
	COMMENT Type = "COMMENT" // // ...
)

// String returns a human-readable name for the token type.
func (t Type) String() string {
	switch t {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case NEWLINE:
		return "NEWLINE"
	case STRING:
		return "STRING"
	case IDENT:
		return "IDENT"
	case NUMBER:
		return "NUMBER"
	case SIGNED_NUMBER:
		return "SIGNED_NUMBER"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case GTE:
		return "GTE"
	case LTE:
		return "LTE"
	case GT:
		return "GT"
	case LT:
		return "LT"
	case EQ:
		return "EQ"
	case NEQ:
		return "NEQ"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case AT:
		return "AT"
	case COLON:
		return "COLON"
	case DOT:
		return "DOT"
	case COMMENT:
		return "COMMENT"
	default:
		return string(t)
	}
}

// Token is a single lexical unit produced by the lexer.
type Token struct {
	Type    Type
	Literal string
	Line    int
	Col     int
}
