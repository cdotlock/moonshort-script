package lexer

import (
	"testing"

	"github.com/cdotlock/moonshort-script/internal/token"
)

// helper: tokenize src and return all non-NEWLINE, non-EOF tokens.
func toks(src string) []token.Token {
	l := New(src)
	var out []token.Token
	for {
		t := l.NextToken()
		if t.Type == token.EOF {
			break
		}
		if t.Type == token.NEWLINE {
			continue
		}
		out = append(out, t)
	}
	return out
}

// assertTypes verifies the sequence of token types matches expected.
func assertTypes(t *testing.T, got []token.Token, want ...token.Type) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("token count: got %d, want %d\ngot:  %v\nwant: %v",
			len(got), len(want), typeSlice(got), want)
	}
	for i, tok := range got {
		if tok.Type != want[i] {
			t.Errorf("token[%d]: got %s (%q), want %s", i, tok.Type, tok.Literal, want[i])
		}
	}
}

func typeSlice(ts []token.Token) []token.Type {
	out := make([]token.Type, len(ts))
	for i, t := range ts {
		out[i] = t.Type
	}
	return out
}

// TestLexDirective: @bg set malias_bedroom_morning fade
func TestLexDirective(t *testing.T) {
	src := `@bg set malias_bedroom_morning fade`
	got := toks(src)
	assertTypes(t, got,
		token.AT,
		token.IDENT, // bg
		token.IDENT, // set
		token.IDENT, // malias_bedroom_morning
		token.IDENT, // fade
	)
	if got[1].Literal != "bg" {
		t.Errorf("ident literal: got %q, want %q", got[1].Literal, "bg")
	}
	if got[3].Literal != "malias_bedroom_morning" {
		t.Errorf("ident literal: got %q, want %q", got[3].Literal, "malias_bedroom_morning")
	}
}

// TestLexDialogue: MAURICIO: That's not a library.
// The lexer itself just produces IDENT + COLON; the parser calls ReadDialogueText.
func TestLexDialogue(t *testing.T) {
	src := `MAURICIO: That's not a library. That's a crime scene.`
	l := New(src)

	tok1 := l.NextToken()
	if tok1.Type != token.IDENT || tok1.Literal != "MAURICIO" {
		t.Fatalf("expected IDENT 'MAURICIO', got %s %q", tok1.Type, tok1.Literal)
	}

	tok2 := l.NextToken()
	if tok2.Type != token.COLON {
		t.Fatalf("expected COLON, got %s %q", tok2.Type, tok2.Literal)
	}

	// Simulate what the parser does after IDENT COLON.
	dialogueTok := l.ReadDialogueText()
	if dialogueTok.Type != token.STRING {
		t.Fatalf("ReadDialogueText: expected STRING, got %s", dialogueTok.Type)
	}
	want := "That's not a library. That's a crime scene."
	if dialogueTok.Literal != want {
		t.Errorf("ReadDialogueText literal: got %q, want %q", dialogueTok.Literal, want)
	}
}

// TestLexQuotedString: @episode main:01 "Butterfly" {
func TestLexQuotedString(t *testing.T) {
	src := `@episode main:01 "Butterfly" {`
	got := toks(src)
	assertTypes(t, got,
		token.AT,
		token.IDENT,  // episode
		token.IDENT,  // main:01
		token.STRING, // Butterfly
		token.LBRACE,
	)
	if got[2].Literal != "main:01" {
		t.Errorf("ident literal: got %q, want %q", got[2].Literal, "main:01")
	}
	if got[3].Literal != "Butterfly" {
		t.Errorf("string literal: got %q, want %q", got[3].Literal, "Butterfly")
	}
}

// TestLexComment: comment line followed by directive
func TestLexComment(t *testing.T) {
	src := "// this is a comment\n@bg set morning"
	l := New(src)

	c := l.NextToken()
	if c.Type != token.COMMENT {
		t.Fatalf("expected COMMENT, got %s", c.Type)
	}
	if c.Literal != "this is a comment" {
		t.Errorf("comment literal: got %q, want %q", c.Literal, "this is a comment")
	}

	// consume NEWLINE
	nl := l.NextToken()
	if nl.Type != token.NEWLINE {
		t.Fatalf("expected NEWLINE after comment, got %s", nl.Type)
	}

	// @bg set morning
	rest := toks("@bg set morning")
	if len(rest) != 4 {
		t.Fatalf("expected 4 tokens after comment, got %d", len(rest))
	}
}

// TestLexCondition: @if affection.easton >= 5 && CHA >= 14 {
func TestLexCondition(t *testing.T) {
	src := `@if affection.easton >= 5 && CHA >= 14 {`
	got := toks(src)
	assertTypes(t, got,
		token.AT,
		token.IDENT,  // if
		token.IDENT,  // affection
		token.DOT,    // .
		token.IDENT,  // easton
		token.GTE,    // >=
		token.NUMBER, // 5
		token.AND,    // &&
		token.IDENT,  // CHA
		token.GTE,    // >=
		token.NUMBER, // 14
		token.LBRACE,
	)
}

// TestLexSignedNumber: @xp +3
func TestLexSignedNumber(t *testing.T) {
	src := `@xp +3`
	got := toks(src)
	assertTypes(t, got,
		token.AT,
		token.IDENT,         // xp
		token.SIGNED_NUMBER, // +3
	)
	if got[2].Literal != "+3" {
		t.Errorf("signed number literal: got %q, want %q", got[2].Literal, "+3")
	}

	src2 := `@san -20`
	got2 := toks(src2)
	assertTypes(t, got2,
		token.AT,
		token.IDENT,
		token.SIGNED_NUMBER,
	)
	if got2[2].Literal != "-20" {
		t.Errorf("signed number literal: got %q, want %q", got2[2].Literal, "-20")
	}
}

// TestLexLineTracking: verify line numbers increment on newlines.
func TestLexLineTracking(t *testing.T) {
	src := "@bg set morning\n@char show mauricio\n@music play theme"
	l := New(src)

	var all []token.Token
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		all = append(all, tok)
	}

	// First AT should be line 1
	if all[0].Line != 1 {
		t.Errorf("first @ line: got %d, want 1", all[0].Line)
	}

	// Find the second AT token (after first NEWLINE)
	foundSecond := false
	for _, tok := range all {
		if tok.Type == token.AT && tok.Line == 2 {
			foundSecond = true
			break
		}
	}
	if !foundSecond {
		t.Error("expected AT token on line 2")
	}

	// Find the third AT token on line 3
	foundThird := false
	for _, tok := range all {
		if tok.Type == token.AT && tok.Line == 3 {
			foundThird = true
			break
		}
	}
	if !foundThird {
		t.Error("expected AT token on line 3")
	}
}

// TestLexOperators: verify all comparison and logical operators.
func TestLexOperators(t *testing.T) {
	cases := []struct {
		src  string
		want token.Type
		lit  string
	}{
		{">=", token.GTE, ">="},
		{"<=", token.LTE, "<="},
		{"==", token.EQ, "=="},
		{"!=", token.NEQ, "!="},
		{"&&", token.AND, "&&"},
		{"||", token.OR, "||"},
		{">", token.GT, ">"},
		{"<", token.LT, "<"},
	}
	for _, c := range cases {
		got := toks(c.src)
		if len(got) != 1 {
			t.Errorf("%q: expected 1 token, got %d", c.src, len(got))
			continue
		}
		if got[0].Type != c.want {
			t.Errorf("%q: type got %s, want %s", c.src, got[0].Type, c.want)
		}
		if got[0].Literal != c.lit {
			t.Errorf("%q: literal got %q, want %q", c.src, got[0].Literal, c.lit)
		}
	}
}
