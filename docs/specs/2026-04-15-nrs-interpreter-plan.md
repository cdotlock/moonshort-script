# NRS Interpreter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go single-binary CLI tool (`nrs`) that parses NoRules Script `.md` files into structured JSON for the frontend visual novel player, resolving asset semantic names to OSS URLs via a separate mapping file.

**Architecture:** Lexer → Parser → AST → Validator → Resolver → JSON Emitter. The lexer tokenizes NRS syntax (@ directives, dialogue lines, blocks). The parser builds a typed AST. The validator checks semantic correctness. The resolver maps asset names to URLs. The emitter outputs player-ready JSON.

**Tech Stack:** Go 1.23+, stdlib only (no external dependencies). Single `go build` produces the binary.

---

## File Structure

```
moonshort-script/
├── cmd/nrs/
│   └── main.go                    # CLI entry: compile/validate subcommands
├── internal/
│   ├── lexer/
│   │   ├── lexer.go               # Tokenizer: input text → token stream
│   │   └── lexer_test.go
│   ├── token/
│   │   └── token.go               # Token types and Token struct
│   ├── parser/
│   │   ├── parser.go              # Token stream → AST
│   │   └── parser_test.go
│   ├── ast/
│   │   └── ast.go                 # All AST node types
│   ├── validator/
│   │   ├── validator.go           # Semantic checks on AST
│   │   └── validator_test.go
│   ├── resolver/
│   │   ├── resolver.go            # Asset mapping: semantic name → URL
│   │   └── resolver_test.go
│   └── emitter/
│       ├── emitter.go             # AST → JSON output
│       └── emitter_test.go
├── testdata/
│   ├── ep01.md                    # Full Episode 1 test script (from spec Appendix A)
│   ├── ep01_expected.json         # Expected JSON output for ep01.md
│   ├── mapping.yaml               # Test asset mapping
│   ├── minimal.md                 # Minimal valid script
│   ├── errors/                    # Scripts with intentional errors
│   │   ├── unclosed_block.md
│   │   ├── unknown_directive.md
│   │   └── bad_dialogue.md
│   └── fragments/                 # Focused test scripts for specific features
│       ├── choice.md
│       ├── minigame.md
│       ├── phone.md
│       ├── if_else.md
│       └── gates.md
├── docs/
│   └── specs/
│       ├── 2026-04-15-nrs-script-format-design.md
│       └── 2026-04-15-nrs-interpreter-plan.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

### Task 1: Project Scaffold + Token Types

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `internal/token/token.go`
- Create: `cmd/nrs/main.go` (stub)

- [ ] **Step 1: Initialize Go module**

```bash
cd "/Users/Clock/moonshort backend/moonshort-script"
go mod init github.com/cdotlock/moonshort-script
```

- [ ] **Step 2: Create Makefile**

```makefile
.PHONY: build test clean

build:
	go build -o bin/nrs ./cmd/nrs

test:
	go test ./... -v

clean:
	rm -rf bin/
```

- [ ] **Step 3: Create token types**

Create `internal/token/token.go`:

```go
package token

type Type int

const (
	// Structural
	EOF Type = iota
	ILLEGAL
	NEWLINE

	// Literals
	STRING    // "quoted text"
	IDENT     // bare word (character names, directive args)
	NUMBER    // integer like 12, +3, -20
	SIGNED_NUMBER // +3, -20

	// Delimiters
	LBRACE // {
	RBRACE // }

	// Operators (for @if conditions)
	GTE    // >=
	LTE    // <=
	GT     // >
	LT     // <
	EQ     // ==
	NEQ    // !=
	AND    // &&
	OR     // ||

	// Keywords
	AT        // @
	COLON     // :
	DOT       // . (for affection.char)
	COMMENT   // // ...

	// Dialogue markers (identified by parser context, not lexer)
	// NARRATOR, YOU, CHARACTER are just IDENTs at lexer level
)

type Token struct {
	Type    Type
	Literal string
	Line    int
	Col     int
}

func (t Type) String() string {
	names := map[Type]string{
		EOF: "EOF", ILLEGAL: "ILLEGAL", NEWLINE: "NEWLINE",
		STRING: "STRING", IDENT: "IDENT", NUMBER: "NUMBER",
		SIGNED_NUMBER: "SIGNED_NUMBER",
		LBRACE: "LBRACE", RBRACE: "RBRACE",
		GTE: "GTE", LTE: "LTE", GT: "GT", LT: "LT",
		EQ: "EQ", NEQ: "NEQ", AND: "AND", OR: "OR",
		AT: "AT", COLON: "COLON", DOT: "DOT", COMMENT: "COMMENT",
	}
	if s, ok := names[t]; ok {
		return s
	}
	return "UNKNOWN"
}
```

- [ ] **Step 4: Create CLI stub**

Create `cmd/nrs/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: nrs <compile|validate> <file.md> [--assets mapping.yaml] [-o output.json]")
		os.Exit(1)
	}
	fmt.Println("nrs: not yet implemented")
}
```

- [ ] **Step 5: Verify build**

```bash
make build
./bin/nrs
```

Expected: prints usage message, exits 1.

- [ ] **Step 6: Commit**

```bash
git init
git add go.mod Makefile cmd/ internal/ docs/
git commit -m "feat: project scaffold with token types and CLI stub"
```

---

### Task 2: Lexer

**Files:**
- Create: `internal/lexer/lexer.go`
- Create: `internal/lexer/lexer_test.go`

- [ ] **Step 1: Write lexer tests**

Create `internal/lexer/lexer_test.go`:

```go
package lexer

import (
	"testing"

	"github.com/cdotlock/moonshort-script/internal/token"
)

func TestLexDirective(t *testing.T) {
	input := `@bg set malias_bedroom_morning fade`
	l := New(input)
	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.AT, "@"},
		{token.IDENT, "bg"},
		{token.IDENT, "set"},
		{token.IDENT, "malias_bedroom_morning"},
		{token.IDENT, "fade"},
		{token.EOF, ""},
	}
	for i, e := range expected {
		tok := l.NextToken()
		if tok.Type != e.typ {
			t.Fatalf("token[%d]: expected type %s, got %s (literal=%q)", i, e.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != e.lit {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, e.lit, tok.Literal)
		}
	}
}

func TestLexDialogue(t *testing.T) {
	input := `MAURICIO: That's not a library.`
	l := New(input)
	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.IDENT, "MAURICIO"},
		{token.COLON, ":"},
		{token.STRING, "That's not a library."},
		{token.EOF, ""},
	}
	for i, e := range expected {
		tok := l.NextToken()
		if tok.Type != e.typ {
			t.Fatalf("token[%d]: expected type %s, got %s (literal=%q)", i, e.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != e.lit {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, e.lit, tok.Literal)
		}
	}
}

func TestLexQuotedString(t *testing.T) {
	input := `@episode main:01 "Butterfly" {`
	l := New(input)
	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.AT, "@"},
		{token.IDENT, "episode"},
		{token.IDENT, "main:01"},
		{token.STRING, "Butterfly"},
		{token.LBRACE, "{"},
		{token.EOF, ""},
	}
	for i, e := range expected {
		tok := l.NextToken()
		if tok.Type != e.typ {
			t.Fatalf("token[%d]: expected type %s, got %s (literal=%q)", i, e.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != e.lit {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, e.lit, tok.Literal)
		}
	}
}

func TestLexComment(t *testing.T) {
	input := "// this is a comment\n@bg set foo"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != token.COMMENT {
		t.Fatalf("expected COMMENT, got %s", tok.Type)
	}
	tok = l.NextToken() // NEWLINE
	if tok.Type != token.NEWLINE {
		t.Fatalf("expected NEWLINE, got %s", tok.Type)
	}
	tok = l.NextToken() // @
	if tok.Type != token.AT {
		t.Fatalf("expected AT, got %s", tok.Type)
	}
}

func TestLexCondition(t *testing.T) {
	input := `@if affection.easton >= 5 && CHA >= 14 {`
	l := New(input)
	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.AT, "@"},
		{token.IDENT, "if"},
		{token.IDENT, "affection"},
		{token.DOT, "."},
		{token.IDENT, "easton"},
		{token.GTE, ">="},
		{token.NUMBER, "5"},
		{token.AND, "&&"},
		{token.IDENT, "CHA"},
		{token.GTE, ">="},
		{token.NUMBER, "14"},
		{token.LBRACE, "{"},
		{token.EOF, ""},
	}
	for i, e := range expected {
		tok := l.NextToken()
		if tok.Type != e.typ {
			t.Fatalf("token[%d]: expected type %s, got %s (literal=%q)", i, e.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != e.lit {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, e.lit, tok.Literal)
		}
	}
}

func TestLexSignedNumber(t *testing.T) {
	input := `@xp +3`
	l := New(input)
	l.NextToken() // @
	l.NextToken() // xp
	tok := l.NextToken()
	if tok.Type != token.SIGNED_NUMBER {
		t.Fatalf("expected SIGNED_NUMBER, got %s (literal=%q)", tok.Type, tok.Literal)
	}
	if tok.Literal != "+3" {
		t.Fatalf("expected +3, got %q", tok.Literal)
	}
}

func TestLexLineTracking(t *testing.T) {
	input := "line1\nline2\nline3"
	l := New(input)
	l.NextToken() // line1
	l.NextToken() // NEWLINE
	tok := l.NextToken() // line2
	if tok.Line != 2 {
		t.Fatalf("expected line 2, got %d", tok.Line)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
make test
```

Expected: compilation error — `lexer.New` not defined.

- [ ] **Step 3: Implement lexer**

Create `internal/lexer/lexer.go`:

```go
package lexer

import (
	"github.com/cdotlock/moonshort-script/internal/token"
)

type Lexer struct {
	input   []rune
	pos     int
	readPos int
	ch      rune
	line    int
	col     int
}

func New(input string) *Lexer {
	l := &Lexer{input: []rune(input), line: 1, col: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) NextToken() token.Token {
	l.skipSpaces()

	tok := token.Token{Line: l.line, Col: l.col}

	switch {
	case l.ch == 0:
		tok.Type = token.EOF
		tok.Literal = ""
		return tok

	case l.ch == '\n':
		tok.Type = token.NEWLINE
		tok.Literal = "\n"
		l.line++
		l.col = 0
		l.readChar()
		return tok

	case l.ch == '/' && l.peekChar() == '/':
		tok.Type = token.COMMENT
		tok.Literal = l.readLineComment()
		return tok

	case l.ch == '@':
		tok.Type = token.AT
		tok.Literal = "@"
		l.readChar()
		return tok

	case l.ch == '{':
		tok.Type = token.LBRACE
		tok.Literal = "{"
		l.readChar()
		return tok

	case l.ch == '}':
		tok.Type = token.RBRACE
		tok.Literal = "}"
		l.readChar()
		return tok

	case l.ch == '.':
		tok.Type = token.DOT
		tok.Literal = "."
		l.readChar()
		return tok

	case l.ch == '"':
		tok.Type = token.STRING
		tok.Literal = l.readQuotedString()
		return tok

	case l.ch == '>' && l.peekChar() == '=':
		tok.Type = token.GTE
		tok.Literal = ">="
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '<' && l.peekChar() == '=':
		tok.Type = token.LTE
		tok.Literal = "<="
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '=' && l.peekChar() == '=':
		tok.Type = token.EQ
		tok.Literal = "=="
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '!' && l.peekChar() == '=':
		tok.Type = token.NEQ
		tok.Literal = "!="
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '&' && l.peekChar() == '&':
		tok.Type = token.AND
		tok.Literal = "&&"
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '|' && l.peekChar() == '|':
		tok.Type = token.OR
		tok.Literal = "||"
		l.readChar()
		l.readChar()
		return tok

	case l.ch == '>':
		tok.Type = token.GT
		tok.Literal = ">"
		l.readChar()
		return tok

	case l.ch == '<':
		tok.Type = token.LT
		tok.Literal = "<"
		l.readChar()
		return tok

	case l.ch == ':':
		tok.Type = token.COLON
		tok.Literal = ":"
		l.readChar()
		// After colon, rest of line is dialogue text (unquoted string)
		l.skipSpaces()
		if l.ch != 0 && l.ch != '\n' {
			// Return the dialogue text as the next token
			// But first return the colon
			return tok
		}
		return tok

	case (l.ch == '+' || l.ch == '-') && isDigit(l.peekChar()):
		tok.Type = token.SIGNED_NUMBER
		tok.Literal = l.readSignedNumber()
		return tok

	case isDigit(l.ch):
		tok.Type = token.NUMBER
		tok.Literal = l.readNumber()
		return tok

	case isIdentStart(l.ch):
		tok.Literal = l.readIdent()
		tok.Type = token.IDENT
		return tok

	default:
		tok.Type = token.ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
		return tok
	}
}

// ReadDialogueText reads the rest of the line as unquoted dialogue text.
// Called by the parser after seeing IDENT COLON pattern.
func (l *Lexer) ReadDialogueText() token.Token {
	l.skipSpaces()
	tok := token.Token{Type: token.STRING, Line: l.line, Col: l.col}
	start := l.pos
	for l.ch != 0 && l.ch != '\n' {
		l.readChar()
	}
	tok.Literal = string(l.input[start:l.pos])
	return tok
}

func (l *Lexer) skipSpaces() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readLineComment() string {
	start := l.pos
	for l.ch != 0 && l.ch != '\n' {
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readQuotedString() string {
	l.readChar() // skip opening "
	start := l.pos
	for l.ch != 0 && l.ch != '"' {
		if l.ch == '\\' {
			l.readChar() // skip escaped char
		}
		l.readChar()
	}
	s := string(l.input[start:l.pos])
	if l.ch == '"' {
		l.readChar() // skip closing "
	}
	return s
}

func (l *Lexer) readIdent() string {
	start := l.pos
	for isIdentChar(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readNumber() string {
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readSignedNumber() string {
	start := l.pos
	l.readChar() // skip + or -
	for isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentChar(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == ':' || ch == '/' || ch == '-'
}
```

- [ ] **Step 4: Run tests**

```bash
make test
```

Expected: all lexer tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/lexer/
git commit -m "feat: lexer tokenizes NRS script syntax"
```

---

### Task 3: AST Node Types

**Files:**
- Create: `internal/ast/ast.go`

- [ ] **Step 1: Define all AST node types**

Create `internal/ast/ast.go`:

```go
package ast

// Episode is the root node of a script file.
type Episode struct {
	BranchKey string
	Title     string
	Body      []Node
	Gates     *GatesBlock
}

// Node is the interface all AST nodes implement.
type Node interface {
	nodeType() string
}

// --- Structure ---

type GatesBlock struct {
	Gates   []Gate
	Default string // branch_key
}

type Gate struct {
	Target    string // branch_key
	GateType  string // "choice" or "influence"
	Trigger   *GateTrigger
	Condition string // for influence type
}

type GateTrigger struct {
	OptionID    string // A, B, C...
	CheckResult string // success, fail, any
}

type LabelNode struct {
	Name string
}

type GotoNode struct {
	Target string
}

// --- Visual ---

type BgSetNode struct {
	Name       string
	Transition string // "", "fade", "cut", "slow"
}

type CharShowNode struct {
	Character  string
	PoseExpr   string
	Position   string // left, center, right, left_far, right_far
}

type CharHideNode struct {
	Character  string
	Transition string
}

type CharExprNode struct {
	Character  string
	PoseExpr   string
	Transition string
}

type CharMoveNode struct {
	Character string
	Position  string
}

type CharBubbleNode struct {
	Character  string
	BubbleType string
}

type CgShowNode struct {
	Name       string
	Transition string
	Body       []Node
}

// --- Dialogue ---

type DialogueNode struct {
	Character string
	Text      string
}

type NarratorNode struct {
	Text string
}

type YouNode struct {
	Text string
}

// --- Phone ---

type PhoneShowNode struct {
	Body []Node // contains TextMessageNodes
}

type PhoneHideNode struct{}

type TextMessageNode struct {
	Direction string // "from" or "to"
	Character string
	Content   string
}

// --- Audio ---

type MusicPlayNode struct {
	Name string
}

type MusicCrossfadeNode struct {
	Name string
}

type MusicFadeoutNode struct{}

type SfxPlayNode struct {
	Name string
}

// --- Game Mechanics ---

type MinigameNode struct {
	GameID   string
	Attr     string
	OnResult map[string][]Node // "S" -> steps, "A,B" -> steps
}

type ChoiceNode struct {
	Options []OptionNode
}

type OptionNode struct {
	ID    string // A, B, C...
	Mode  string // "brave" or "safe"
	Text  string
	Check *CheckBlock // nil for safe
	OnSuccess []Node  // brave only
	OnFail    []Node  // brave only
	Body      []Node  // safe only (direct content)
}

type CheckBlock struct {
	Attr string
	DC   int
}

// --- State Changes ---

type XpNode struct {
	Delta int
}

type SanNode struct {
	Delta int
}

type AffectionNode struct {
	Character string
	Delta     int
}

type SignalNode struct {
	Event string
}

type ButterflyNode struct {
	Description string
}

// --- Flow Control ---

type IfNode struct {
	Condition string // raw condition string, engine evaluates
	Then      []Node
	Else      []Node // nil if no else
}

// --- Node interface implementations ---

func (n *BgSetNode) nodeType() string         { return "bg" }
func (n *CharShowNode) nodeType() string      { return "char_show" }
func (n *CharHideNode) nodeType() string      { return "char_hide" }
func (n *CharExprNode) nodeType() string      { return "char_expr" }
func (n *CharMoveNode) nodeType() string      { return "char_move" }
func (n *CharBubbleNode) nodeType() string    { return "bubble" }
func (n *CgShowNode) nodeType() string        { return "cg_show" }
func (n *DialogueNode) nodeType() string      { return "dialogue" }
func (n *NarratorNode) nodeType() string      { return "narrator" }
func (n *YouNode) nodeType() string           { return "you" }
func (n *PhoneShowNode) nodeType() string     { return "phone_show" }
func (n *PhoneHideNode) nodeType() string     { return "phone_hide" }
func (n *TextMessageNode) nodeType() string   { return "text_message" }
func (n *MusicPlayNode) nodeType() string     { return "music_play" }
func (n *MusicCrossfadeNode) nodeType() string { return "music_crossfade" }
func (n *MusicFadeoutNode) nodeType() string  { return "music_fadeout" }
func (n *SfxPlayNode) nodeType() string       { return "sfx_play" }
func (n *MinigameNode) nodeType() string      { return "minigame" }
func (n *ChoiceNode) nodeType() string        { return "choice" }
func (n *XpNode) nodeType() string            { return "xp" }
func (n *SanNode) nodeType() string           { return "san" }
func (n *AffectionNode) nodeType() string     { return "affection" }
func (n *SignalNode) nodeType() string        { return "signal" }
func (n *ButterflyNode) nodeType() string     { return "butterfly" }
func (n *IfNode) nodeType() string            { return "if" }
func (n *LabelNode) nodeType() string         { return "label" }
func (n *GotoNode) nodeType() string          { return "goto" }
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/ast/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ast/
git commit -m "feat: AST node types for all NRS directives"
```

---

### Task 4: Parser — Core Framework + Simple Directives

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

- [ ] **Step 1: Write parser test for minimal script**

Create `internal/parser/parser_test.go`:

```go
package parser

import (
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
)

func TestParseMinimal(t *testing.T) {
	input := `@episode main:01 "Test" {
  @bg set classroom_morning fade
  NARRATOR: Hello world.
  YOU: Thinking deeply.
}`
	l := lexer.New(input)
	p := New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if ep.BranchKey != "main:01" {
		t.Fatalf("expected branch_key main:01, got %s", ep.BranchKey)
	}
	if ep.Title != "Test" {
		t.Fatalf("expected title Test, got %s", ep.Title)
	}
	if len(ep.Body) != 3 {
		t.Fatalf("expected 3 body nodes, got %d", len(ep.Body))
	}

	bg, ok := ep.Body[0].(*ast.BgSetNode)
	if !ok {
		t.Fatalf("body[0]: expected BgSetNode, got %T", ep.Body[0])
	}
	if bg.Name != "classroom_morning" || bg.Transition != "fade" {
		t.Fatalf("bg: got name=%s transition=%s", bg.Name, bg.Transition)
	}

	nar, ok := ep.Body[1].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("body[1]: expected NarratorNode, got %T", ep.Body[1])
	}
	if nar.Text != "Hello world." {
		t.Fatalf("narrator text: got %q", nar.Text)
	}

	you, ok := ep.Body[2].(*ast.YouNode)
	if !ok {
		t.Fatalf("body[2]: expected YouNode, got %T", ep.Body[2])
	}
	if you.Text != "Thinking deeply." {
		t.Fatalf("you text: got %q", you.Text)
	}
}

func TestParseCharDirectives(t *testing.T) {
	input := `@episode main:01 "Test" {
  @mauricio show neutral_smirk at right
  @mauricio expr arms_crossed_angry dissolve
  @mauricio bubble heart
  @mauricio move to left
  @mauricio hide fade
  MAURICIO: Hey there.
}`
	l := lexer.New(input)
	p := New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ep.Body) != 6 {
		t.Fatalf("expected 6 body nodes, got %d", len(ep.Body))
	}

	show, ok := ep.Body[0].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("body[0]: expected CharShowNode, got %T", ep.Body[0])
	}
	if show.Character != "mauricio" || show.PoseExpr != "neutral_smirk" || show.Position != "right" {
		t.Fatalf("show: char=%s pose=%s pos=%s", show.Character, show.PoseExpr, show.Position)
	}

	expr, ok := ep.Body[1].(*ast.CharExprNode)
	if !ok {
		t.Fatalf("body[1]: expected CharExprNode, got %T", ep.Body[1])
	}
	if expr.Character != "mauricio" || expr.PoseExpr != "arms_crossed_angry" || expr.Transition != "dissolve" {
		t.Fatalf("expr: char=%s pose=%s trans=%s", expr.Character, expr.PoseExpr, expr.Transition)
	}
}

func TestParseAudio(t *testing.T) {
	input := `@episode main:01 "Test" {
  @music play calm_morning
  @music crossfade tense_strings
  @music fadeout
  @sfx play door_slam
}`
	l := lexer.New(input)
	p := New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ep.Body) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.MusicPlayNode); !ok {
		t.Fatalf("body[0]: expected MusicPlayNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.MusicCrossfadeNode); !ok {
		t.Fatalf("body[1]: expected MusicCrossfadeNode, got %T", ep.Body[1])
	}
	if _, ok := ep.Body[2].(*ast.MusicFadeoutNode); !ok {
		t.Fatalf("body[2]: expected MusicFadeoutNode, got %T", ep.Body[2])
	}
	if _, ok := ep.Body[3].(*ast.SfxPlayNode); !ok {
		t.Fatalf("body[3]: expected SfxPlayNode, got %T", ep.Body[3])
	}
}

func TestParseStateChanges(t *testing.T) {
	input := `@episode main:01 "Test" {
  @xp +3
  @san -20
  @affection easton +2
  @signal EP01_COMPLETE
  @butterfly "Accepted Easton"
}`
	l := lexer.New(input)
	p := New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ep.Body) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(ep.Body))
	}

	xp, ok := ep.Body[0].(*ast.XpNode)
	if !ok {
		t.Fatalf("body[0]: expected XpNode, got %T", ep.Body[0])
	}
	if xp.Delta != 3 {
		t.Fatalf("xp delta: expected 3, got %d", xp.Delta)
	}

	san, ok := ep.Body[1].(*ast.SanNode)
	if !ok {
		t.Fatalf("body[1]: expected SanNode, got %T", ep.Body[1])
	}
	if san.Delta != -20 {
		t.Fatalf("san delta: expected -20, got %d", san.Delta)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/parser/ -v
```

Expected: compilation error — `parser.New` not defined.

- [ ] **Step 3: Implement parser core + simple directives**

Create `internal/parser/parser.go`:

```go
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/token"
)

type Parser struct {
	l    *lexer.Lexer
	cur  token.Token
	peek token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	p.advance()
	p.advance()
	return p
}

func (p *Parser) advance() {
	p.cur = p.peek
	p.peek = p.l.NextToken()
	// Skip newlines and comments at the top level
	for p.peek.Type == token.NEWLINE || p.peek.Type == token.COMMENT {
		p.peek = p.l.NextToken()
	}
}

func (p *Parser) expect(t token.Type) error {
	if p.cur.Type != t {
		return fmt.Errorf("line %d: expected %s, got %s (%q)", p.cur.Line, t, p.cur.Type, p.cur.Literal)
	}
	return nil
}

func (p *Parser) Parse() (*ast.Episode, error) {
	// Skip leading whitespace/comments
	for p.cur.Type == token.NEWLINE || p.cur.Type == token.COMMENT {
		p.advance()
	}

	// Expect @episode
	if err := p.expect(token.AT); err != nil {
		return nil, err
	}
	p.advance()
	if p.cur.Literal != "episode" {
		return nil, fmt.Errorf("line %d: expected 'episode', got %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()

	ep := &ast.Episode{}
	ep.BranchKey = p.cur.Literal
	p.advance()

	if p.cur.Type != token.STRING {
		return nil, fmt.Errorf("line %d: expected episode title string, got %s", p.cur.Line, p.cur.Type)
	}
	ep.Title = p.cur.Literal
	p.advance()

	if err := p.expect(token.LBRACE); err != nil {
		return nil, err
	}
	p.advance()

	body, gates, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	ep.Body = body
	ep.Gates = gates

	return ep, nil
}

func (p *Parser) parseBlock() ([]ast.Node, *ast.GatesBlock, error) {
	var nodes []ast.Node
	var gates *ast.GatesBlock

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		switch p.cur.Type {
		case token.AT:
			node, g, err := p.parseDirective()
			if err != nil {
				return nil, nil, err
			}
			if g != nil {
				gates = g
			} else if node != nil {
				nodes = append(nodes, node)
			}
		case token.IDENT:
			node, err := p.parseDialogue()
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, node)
		default:
			p.advance()
		}
	}

	if p.cur.Type == token.RBRACE {
		p.advance()
	}

	return nodes, gates, nil
}

func (p *Parser) parseDirective() (ast.Node, *ast.GatesBlock, error) {
	p.advance() // skip @
	directive := p.cur.Literal
	p.advance()

	switch directive {
	// Background
	case "bg":
		return p.parseBg()
	// CG
	case "cg":
		return p.parseCg()
	// Phone
	case "phone":
		return p.parsePhone()
	// Text message (inside phone block)
	case "text":
		return p.parseTextMessage()
	// Audio
	case "music":
		return p.parseMusic()
	case "sfx":
		return p.parseSfx()
	// Game mechanics
	case "minigame":
		return p.parseMinigame()
	case "choice":
		return p.parseChoice()
	// State changes
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
	// Flow control
	case "if":
		return p.parseIf()
	case "label":
		return p.parseLabel()
	case "goto":
		return p.parseGoto()
	// Gates
	case "gates":
		g, err := p.parseGates()
		return nil, g, err
	default:
		// Dynamic character directive: @<char> <action> ...
		return p.parseCharDirective(directive)
	}
}

func (p *Parser) parseBg() (ast.Node, *ast.GatesBlock, error) {
	// @bg set <name> [transition]
	// cur = "set"
	if p.cur.Literal != "set" {
		return nil, nil, fmt.Errorf("line %d: expected 'set' after @bg, got %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()

	node := &ast.BgSetNode{Name: p.cur.Literal}
	p.advance()

	// Optional transition
	if p.cur.Type == token.IDENT && p.cur.Type != token.AT && p.cur.Literal != "" {
		if isTransition(p.cur.Literal) {
			node.Transition = p.cur.Literal
			p.advance()
		}
	}
	if node.Transition == "" {
		node.Transition = "dissolve"
	}

	return node, nil, nil
}

func (p *Parser) parseCg() (ast.Node, *ast.GatesBlock, error) {
	// @cg show <name> [transition] { body }
	if p.cur.Literal != "show" {
		return nil, nil, fmt.Errorf("line %d: expected 'show' after @cg, got %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()

	node := &ast.CgShowNode{Name: p.cur.Literal}
	p.advance()

	if p.cur.Type == token.IDENT && isTransition(p.cur.Literal) {
		node.Transition = p.cur.Literal
		p.advance()
	}

	if p.cur.Type == token.LBRACE {
		p.advance()
		body, _, err := p.parseBlock()
		if err != nil {
			return nil, nil, err
		}
		node.Body = body
	}

	return node, nil, nil
}

func (p *Parser) parseCharDirective(char string) (ast.Node, *ast.GatesBlock, error) {
	action := p.cur.Literal
	p.advance()

	switch action {
	case "show":
		node := &ast.CharShowNode{Character: char}
		node.PoseExpr = p.cur.Literal
		p.advance()
		// expect "at"
		if p.cur.Literal == "at" {
			p.advance()
			node.Position = p.cur.Literal
			p.advance()
		}
		return node, nil, nil

	case "hide":
		node := &ast.CharHideNode{Character: char}
		if p.cur.Type == token.IDENT && isTransition(p.cur.Literal) {
			node.Transition = p.cur.Literal
			p.advance()
		}
		return node, nil, nil

	case "expr":
		node := &ast.CharExprNode{Character: char}
		node.PoseExpr = p.cur.Literal
		p.advance()
		if p.cur.Type == token.IDENT && isTransition(p.cur.Literal) {
			node.Transition = p.cur.Literal
			p.advance()
		}
		return node, nil, nil

	case "move":
		node := &ast.CharMoveNode{Character: char}
		if p.cur.Literal == "to" {
			p.advance()
			node.Position = p.cur.Literal
			p.advance()
		}
		return node, nil, nil

	case "bubble":
		node := &ast.CharBubbleNode{Character: char, BubbleType: p.cur.Literal}
		p.advance()
		return node, nil, nil

	default:
		return nil, nil, fmt.Errorf("line %d: unknown character action %q for @%s", p.cur.Line, action, char)
	}
}

func (p *Parser) parseDialogue() (ast.Node, error) {
	name := p.cur.Literal
	p.advance() // skip name

	if p.cur.Type != token.COLON {
		return nil, fmt.Errorf("line %d: expected ':' after character name %q", p.cur.Line, name)
	}
	p.advance() // skip colon

	text := p.l.ReadDialogueText()

	switch strings.ToUpper(name) {
	case "NARRATOR":
		return &ast.NarratorNode{Text: text.Literal}, nil
	case "YOU":
		return &ast.YouNode{Text: text.Literal}, nil
	default:
		return &ast.DialogueNode{Character: strings.ToLower(name), Text: text.Literal}, nil
	}
}

func (p *Parser) parsePhone() (ast.Node, *ast.GatesBlock, error) {
	action := p.cur.Literal
	p.advance()

	if action == "hide" {
		return &ast.PhoneHideNode{}, nil, nil
	}

	// phone show { ... }
	if p.cur.Type != token.LBRACE {
		return nil, nil, fmt.Errorf("line %d: expected '{' after @phone show", p.cur.Line)
	}
	p.advance()

	body, _, err := p.parseBlock()
	if err != nil {
		return nil, nil, err
	}
	return &ast.PhoneShowNode{Body: body}, nil, nil
}

func (p *Parser) parseTextMessage() (ast.Node, *ast.GatesBlock, error) {
	direction := p.cur.Literal // "from" or "to"
	p.advance()

	character := strings.ToLower(p.cur.Literal)
	p.advance()

	// skip colon
	if p.cur.Type == token.COLON {
		p.advance()
	}

	text := p.l.ReadDialogueText()

	return &ast.TextMessageNode{
		Direction: direction,
		Character: character,
		Content:   text.Literal,
	}, nil, nil
}

func (p *Parser) parseMusic() (ast.Node, *ast.GatesBlock, error) {
	action := p.cur.Literal
	p.advance()

	switch action {
	case "play":
		name := p.cur.Literal
		p.advance()
		return &ast.MusicPlayNode{Name: name}, nil, nil
	case "crossfade":
		name := p.cur.Literal
		p.advance()
		return &ast.MusicCrossfadeNode{Name: name}, nil, nil
	case "fadeout":
		return &ast.MusicFadeoutNode{}, nil, nil
	default:
		return nil, nil, fmt.Errorf("line %d: unknown music action %q", p.cur.Line, action)
	}
}

func (p *Parser) parseSfx() (ast.Node, *ast.GatesBlock, error) {
	if p.cur.Literal != "play" {
		return nil, nil, fmt.Errorf("line %d: expected 'play' after @sfx, got %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()
	name := p.cur.Literal
	p.advance()
	return &ast.SfxPlayNode{Name: name}, nil, nil
}

func (p *Parser) parseXp() (ast.Node, *ast.GatesBlock, error) {
	delta, err := strconv.Atoi(p.cur.Literal)
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: invalid xp value %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()
	return &ast.XpNode{Delta: delta}, nil, nil
}

func (p *Parser) parseSan() (ast.Node, *ast.GatesBlock, error) {
	delta, err := strconv.Atoi(p.cur.Literal)
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: invalid san value %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()
	return &ast.SanNode{Delta: delta}, nil, nil
}

func (p *Parser) parseAffection() (ast.Node, *ast.GatesBlock, error) {
	char := strings.ToLower(p.cur.Literal)
	p.advance()
	delta, err := strconv.Atoi(p.cur.Literal)
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: invalid affection value %q", p.cur.Line, p.cur.Literal)
	}
	p.advance()
	return &ast.AffectionNode{Character: char, Delta: delta}, nil, nil
}

func (p *Parser) parseSignal() (ast.Node, *ast.GatesBlock, error) {
	event := p.cur.Literal
	p.advance()
	return &ast.SignalNode{Event: event}, nil, nil
}

func (p *Parser) parseButterfly() (ast.Node, *ast.GatesBlock, error) {
	if p.cur.Type != token.STRING {
		return nil, nil, fmt.Errorf("line %d: expected quoted string after @butterfly", p.cur.Line)
	}
	desc := p.cur.Literal
	p.advance()
	return &ast.ButterflyNode{Description: desc}, nil, nil
}

func (p *Parser) parseLabel() (ast.Node, *ast.GatesBlock, error) {
	name := p.cur.Literal
	p.advance()
	return &ast.LabelNode{Name: name}, nil, nil
}

func (p *Parser) parseGoto() (ast.Node, *ast.GatesBlock, error) {
	target := p.cur.Literal
	p.advance()
	return &ast.GotoNode{Target: target}, nil, nil
}

func (p *Parser) parseIf() (ast.Node, *ast.GatesBlock, error) {
	// Collect condition tokens until {
	var condParts []string
	for p.cur.Type != token.LBRACE && p.cur.Type != token.EOF {
		condParts = append(condParts, p.cur.Literal)
		p.advance()
	}
	condition := strings.Join(condParts, " ")

	p.advance() // skip {
	thenBody, _, err := p.parseBlock()
	if err != nil {
		return nil, nil, err
	}

	node := &ast.IfNode{Condition: condition, Then: thenBody}

	// Check for @else
	if p.cur.Type == token.AT && p.peek.Literal == "else" {
		p.advance() // skip @
		p.advance() // skip else
		if p.cur.Type != token.LBRACE {
			return nil, nil, fmt.Errorf("line %d: expected '{' after @else", p.cur.Line)
		}
		p.advance()
		elseBody, _, err := p.parseBlock()
		if err != nil {
			return nil, nil, err
		}
		node.Else = elseBody
	}

	return node, nil, nil
}

func (p *Parser) parseMinigame() (ast.Node, *ast.GatesBlock, error) {
	node := &ast.MinigameNode{OnResult: make(map[string][]ast.Node)}
	node.GameID = p.cur.Literal
	p.advance()
	node.Attr = p.cur.Literal
	p.advance()

	if p.cur.Type != token.LBRACE {
		return nil, nil, fmt.Errorf("line %d: expected '{' after @minigame", p.cur.Line)
	}
	p.advance()

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT {
			p.advance()
			if p.cur.Literal == "on" {
				p.advance()
				// Collect rating keys (S, A, B, etc.)
				var keys []string
				for p.cur.Type == token.IDENT {
					keys = append(keys, p.cur.Literal)
					p.advance()
				}
				key := strings.Join(keys, ",")

				if p.cur.Type != token.LBRACE {
					return nil, nil, fmt.Errorf("line %d: expected '{' after @on ratings", p.cur.Line)
				}
				p.advance()
				body, _, err := p.parseBlock()
				if err != nil {
					return nil, nil, err
				}
				node.OnResult[key] = body
			}
		} else {
			p.advance()
		}
	}
	if p.cur.Type == token.RBRACE {
		p.advance()
	}

	return node, nil, nil
}

func (p *Parser) parseChoice() (ast.Node, *ast.GatesBlock, error) {
	if p.cur.Type != token.LBRACE {
		return nil, nil, fmt.Errorf("line %d: expected '{' after @choice", p.cur.Line)
	}
	p.advance()

	node := &ast.ChoiceNode{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT {
			p.advance()
			if p.cur.Literal == "option" {
				opt, err := p.parseOption()
				if err != nil {
					return nil, nil, err
				}
				node.Options = append(node.Options, *opt)
			}
		} else {
			p.advance()
		}
	}
	if p.cur.Type == token.RBRACE {
		p.advance()
	}

	return node, nil, nil
}

func (p *Parser) parseOption() (*ast.OptionNode, error) {
	p.advance() // skip "option"

	opt := &ast.OptionNode{}
	opt.ID = p.cur.Literal
	p.advance()

	opt.Mode = p.cur.Literal // brave or safe
	p.advance()

	if p.cur.Type != token.STRING {
		return nil, fmt.Errorf("line %d: expected option text string", p.cur.Line)
	}
	opt.Text = p.cur.Literal
	p.advance()

	if p.cur.Type != token.LBRACE {
		return nil, fmt.Errorf("line %d: expected '{' after option text", p.cur.Line)
	}
	p.advance()

	if opt.Mode == "brave" {
		// Parse check block and @on success/@on fail
		for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
			if p.cur.Type == token.IDENT && p.cur.Literal == "check" {
				check, err := p.parseCheckBlock()
				if err != nil {
					return nil, err
				}
				opt.Check = check
			} else if p.cur.Type == token.AT {
				p.advance()
				if p.cur.Literal == "on" {
					p.advance()
					result := p.cur.Literal // success or fail
					p.advance()
					if p.cur.Type != token.LBRACE {
						return nil, fmt.Errorf("line %d: expected '{' after @on %s", p.cur.Line, result)
					}
					p.advance()
					body, _, err := p.parseBlock()
					if err != nil {
						return nil, err
					}
					if result == "success" {
						opt.OnSuccess = body
					} else {
						opt.OnFail = body
					}
				} else {
					// Other directives inside option
					node, _, err := p.parseDirective()
					if err != nil {
						return nil, err
					}
					if node != nil {
						opt.Body = append(opt.Body, node)
					}
				}
			} else {
				p.advance()
			}
		}
	} else {
		// Safe: parse body directly
		body, _, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		opt.Body = body
		return opt, nil
	}

	if p.cur.Type == token.RBRACE {
		p.advance()
	}
	return opt, nil
}

func (p *Parser) parseCheckBlock() (*ast.CheckBlock, error) {
	p.advance() // skip "check"
	if p.cur.Type != token.LBRACE {
		return nil, fmt.Errorf("line %d: expected '{' after check", p.cur.Line)
	}
	p.advance()

	check := &ast.CheckBlock{}
	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		key := p.cur.Literal
		p.advance()
		if p.cur.Type == token.COLON {
			p.advance()
		}
		value := p.cur.Literal
		p.advance()

		switch key {
		case "attr":
			check.Attr = value
		case "dc":
			dc, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid dc value %q", p.cur.Line, value)
			}
			check.DC = dc
		}
	}
	if p.cur.Type == token.RBRACE {
		p.advance()
	}
	return check, nil
}

func (p *Parser) parseGates() (*ast.GatesBlock, error) {
	if p.cur.Type != token.LBRACE {
		return nil, fmt.Errorf("line %d: expected '{' after @gates", p.cur.Line)
	}
	p.advance()

	gates := &ast.GatesBlock{}

	for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
		if p.cur.Type == token.AT {
			p.advance()
			switch p.cur.Literal {
			case "gate":
				p.advance()
				gate := ast.Gate{Target: p.cur.Literal}
				p.advance()
				if p.cur.Type != token.LBRACE {
					return nil, fmt.Errorf("line %d: expected '{' after gate target", p.cur.Line)
				}
				p.advance()
				// Parse gate body (key: value pairs)
				for p.cur.Type != token.RBRACE && p.cur.Type != token.EOF {
					key := p.cur.Literal
					p.advance()
					if p.cur.Type == token.COLON {
						p.advance()
					}
					switch key {
					case "type":
						gate.GateType = p.cur.Literal
						p.advance()
					case "trigger":
						gate.Trigger = &ast.GateTrigger{
							OptionID: p.cur.Literal,
						}
						p.advance()
						gate.Trigger.CheckResult = p.cur.Literal
						p.advance()
					case "condition":
						if p.cur.Type == token.STRING {
							gate.Condition = p.cur.Literal
							p.advance()
						}
					}
				}
				if p.cur.Type == token.RBRACE {
					p.advance()
				}
				gates.Gates = append(gates.Gates, gate)

			case "default":
				p.advance()
				gates.Default = p.cur.Literal
				p.advance()
			}
		} else {
			p.advance()
		}
	}
	if p.cur.Type == token.RBRACE {
		p.advance()
	}

	return gates, nil
}

func isTransition(s string) bool {
	switch s {
	case "fade", "cut", "slow", "dissolve":
		return true
	}
	return false
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/parser/ -v
```

Expected: all parser tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat: parser converts token stream to AST for all NRS directives"
```

---

### Task 5: Asset Resolver

**Files:**
- Create: `internal/resolver/resolver.go`
- Create: `internal/resolver/resolver_test.go`
- Create: `testdata/mapping.yaml`

- [ ] **Step 1: Create test mapping file**

Create `testdata/mapping.yaml`:

```yaml
base_url: "https://oss.mobai.com/novel_001"

assets:
  bg:
    malias_bedroom_morning: "bg/malias_bedroom_morning.png"
    school_front: "bg/school_front.png"
  characters:
    mauricio:
      neutral_smirk: "characters/mauricio_neutral_smirk.png"
    malia:
      neutral_phone: "characters/malia_neutral_phone.png"
  music:
    calm_morning: "music/calm_morning.mp3"
  sfx:
    door_slam: "sfx/door_slam.mp3"
  cg:
    window_stare: "cg/window_stare.png"
  minigames:
    qte_challenge: "minigames/qte_challenge/index.html"
```

- [ ] **Step 2: Write resolver tests**

Create `internal/resolver/resolver_test.go`:

```go
package resolver

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "mapping.yaml")
}

func TestLoadMapping(t *testing.T) {
	r, err := NewFromFile(testdataPath())
	if err != nil {
		t.Fatalf("failed to load mapping: %v", err)
	}
	if r.BaseURL != "https://oss.mobai.com/novel_001" {
		t.Fatalf("unexpected base_url: %s", r.BaseURL)
	}
}

func TestResolveBg(t *testing.T) {
	r, _ := NewFromFile(testdataPath())
	url, err := r.ResolveBg("malias_bedroom_morning")
	if err != nil {
		t.Fatalf("resolve bg: %v", err)
	}
	if url != "https://oss.mobai.com/novel_001/bg/malias_bedroom_morning.png" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestResolveCharacter(t *testing.T) {
	r, _ := NewFromFile(testdataPath())
	url, err := r.ResolveCharacter("mauricio", "neutral_smirk")
	if err != nil {
		t.Fatalf("resolve character: %v", err)
	}
	if url != "https://oss.mobai.com/novel_001/characters/mauricio_neutral_smirk.png" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestResolveMusic(t *testing.T) {
	r, _ := NewFromFile(testdataPath())
	url, err := r.ResolveMusic("calm_morning")
	if err != nil {
		t.Fatalf("resolve music: %v", err)
	}
	if url != "https://oss.mobai.com/novel_001/music/calm_morning.mp3" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestResolveMissing(t *testing.T) {
	r, _ := NewFromFile(testdataPath())
	_, err := r.ResolveBg("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestResolveMinigame(t *testing.T) {
	r, _ := NewFromFile(testdataPath())
	url, err := r.ResolveMinigame("qte_challenge")
	if err != nil {
		t.Fatalf("resolve minigame: %v", err)
	}
	if url != "https://oss.mobai.com/novel_001/minigames/qte_challenge/index.html" {
		t.Fatalf("unexpected url: %s", url)
	}
}
```

- [ ] **Step 3: Run tests to verify failure**

```bash
go test ./internal/resolver/ -v
```

Expected: compilation error.

- [ ] **Step 4: Implement resolver**

Create `internal/resolver/resolver.go`:

```go
package resolver

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// We avoid external deps — use a minimal YAML parser or encoding/json.
// Actually, Go stdlib has no YAML. Use a simple custom parser for this flat structure.
// For simplicity and correctness, we'll use gopkg.in/yaml.v3.

type Mapping struct {
	BaseURL string                       `yaml:"base_url"`
	Assets  map[string]interface{}       `yaml:"assets"`
}

type Resolver struct {
	BaseURL string
	data    map[string]interface{}
}

func NewFromFile(path string) (*Resolver, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mapping file: %w", err)
	}

	var m Mapping
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse mapping yaml: %w", err)
	}

	return &Resolver{BaseURL: m.BaseURL, data: m.Assets}, nil
}

func (r *Resolver) resolve(keys ...string) (string, error) {
	var current interface{} = r.data
	for _, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("asset not found: %s", strings.Join(keys, "."))
		}
		current, ok = m[key]
		if !ok {
			return "", fmt.Errorf("asset not found: %s", strings.Join(keys, "."))
		}
	}

	s, ok := current.(string)
	if !ok {
		return "", fmt.Errorf("asset path is not a string: %s", strings.Join(keys, "."))
	}

	return r.BaseURL + "/" + s, nil
}

func (r *Resolver) ResolveBg(name string) (string, error) {
	return r.resolve("bg", name)
}

func (r *Resolver) ResolveCharacter(char, poseExpr string) (string, error) {
	return r.resolve("characters", char, poseExpr)
}

func (r *Resolver) ResolveMusic(name string) (string, error) {
	return r.resolve("music", name)
}

func (r *Resolver) ResolveSfx(name string) (string, error) {
	return r.resolve("sfx", name)
}

func (r *Resolver) ResolveCg(name string) (string, error) {
	return r.resolve("cg", name)
}

func (r *Resolver) ResolveMinigame(gameID string) (string, error) {
	return r.resolve("minigames", gameID)
}
```

- [ ] **Step 5: Add yaml dependency**

```bash
cd "/Users/Clock/moonshort backend/moonshort-script"
go get gopkg.in/yaml.v3
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/resolver/ -v
```

Expected: all resolver tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/resolver/ testdata/mapping.yaml go.mod go.sum
git commit -m "feat: asset resolver maps semantic names to OSS URLs via YAML mapping"
```

---

### Task 6: JSON Emitter

**Files:**
- Create: `internal/emitter/emitter.go`
- Create: `internal/emitter/emitter_test.go`

- [ ] **Step 1: Write emitter test**

Create `internal/emitter/emitter_test.go`:

```go
package emitter

import (
	"encoding/json"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

type mockResolver struct{}

func (m *mockResolver) ResolveBg(name string) (string, error) {
	return "https://oss.test/" + name + ".png", nil
}
func (m *mockResolver) ResolveCharacter(char, pose string) (string, error) {
	return "https://oss.test/" + char + "_" + pose + ".png", nil
}
func (m *mockResolver) ResolveMusic(name string) (string, error) {
	return "https://oss.test/" + name + ".mp3", nil
}
func (m *mockResolver) ResolveSfx(name string) (string, error) {
	return "https://oss.test/" + name + ".mp3", nil
}
func (m *mockResolver) ResolveCg(name string) (string, error) {
	return "https://oss.test/" + name + ".png", nil
}
func (m *mockResolver) ResolveMinigame(id string) (string, error) {
	return "https://oss.test/" + id + "/index.html", nil
}

func TestEmitMinimal(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "classroom", Transition: "fade"},
			&ast.NarratorNode{Text: "Hello."},
			&ast.XpNode{Delta: 3},
		},
		Gates: &ast.GatesBlock{Default: "main:02"},
	}

	e := New(&mockResolver{})
	result, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("emit error: %v", err)
	}

	// Parse JSON to verify structure
	var out map[string]interface{}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if out["episode_id"] != "main:01" {
		t.Fatalf("episode_id: got %v", out["episode_id"])
	}
	if out["title"] != "Test" {
		t.Fatalf("title: got %v", out["title"])
	}

	steps, ok := out["steps"].([]interface{})
	if !ok {
		t.Fatalf("steps is not array: %T", out["steps"])
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	step0 := steps[0].(map[string]interface{})
	if step0["type"] != "bg" {
		t.Fatalf("step[0] type: got %v", step0["type"])
	}
	if step0["url"] != "https://oss.test/classroom.png" {
		t.Fatalf("step[0] url: got %v", step0["url"])
	}
}

func TestEmitChoice(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []ast.OptionNode{
					{
						ID:   "A",
						Mode: "brave",
						Text: "Fight",
						Check: &ast.CheckBlock{Attr: "CHA", DC: 12},
						OnSuccess: []ast.Node{&ast.XpNode{Delta: 3}},
						OnFail:    []ast.Node{&ast.SanNode{Delta: -20}},
					},
					{
						ID:   "B",
						Mode: "safe",
						Text: "Run",
						Body: []ast.Node{&ast.XpNode{Delta: 1}},
					},
				},
			},
		},
		Gates: &ast.GatesBlock{Default: "main:02"},
	}

	e := New(&mockResolver{})
	result, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("emit error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	steps := out["steps"].([]interface{})
	choice := steps[0].(map[string]interface{})
	if choice["type"] != "choice" {
		t.Fatalf("expected choice, got %v", choice["type"])
	}

	options := choice["options"].([]interface{})
	optA := options[0].(map[string]interface{})
	if optA["id"] != "A" {
		t.Fatalf("option A id: got %v", optA["id"])
	}
	if optA["mode"] != "brave" {
		t.Fatalf("option A mode: got %v", optA["mode"])
	}

	check := optA["check"].(map[string]interface{})
	if check["attr"] != "CHA" {
		t.Fatalf("check attr: got %v", check["attr"])
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./internal/emitter/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement emitter**

Create `internal/emitter/emitter.go`:

```go
package emitter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

type AssetResolver interface {
	ResolveBg(name string) (string, error)
	ResolveCharacter(char, poseExpr string) (string, error)
	ResolveMusic(name string) (string, error)
	ResolveSfx(name string) (string, error)
	ResolveCg(name string) (string, error)
	ResolveMinigame(gameID string) (string, error)
}

type Emitter struct {
	resolver AssetResolver
	warnings []string
}

func New(resolver AssetResolver) *Emitter {
	return &Emitter{resolver: resolver}
}

func (e *Emitter) Emit(ep *ast.Episode) ([]byte, error) {
	out := map[string]interface{}{
		"episode_id": ep.BranchKey,
		"branch_key": extractBranchKey(ep.BranchKey),
		"seq":        extractSeq(ep.BranchKey),
		"title":      ep.Title,
	}

	steps, err := e.emitNodes(ep.Body)
	if err != nil {
		return nil, err
	}
	out["steps"] = steps

	if ep.Gates != nil {
		out["gates"] = e.emitGates(ep.Gates)
	}

	return json.MarshalIndent(out, "", "  ")
}

func (e *Emitter) Warnings() []string {
	return e.warnings
}

func (e *Emitter) emitNodes(nodes []ast.Node) ([]interface{}, error) {
	var out []interface{}
	for _, node := range nodes {
		step, err := e.emitNode(node)
		if err != nil {
			return nil, err
		}
		if step != nil {
			out = append(out, step)
		}
	}
	return out, nil
}

func (e *Emitter) emitNode(node ast.Node) (map[string]interface{}, error) {
	switch n := node.(type) {
	case *ast.BgSetNode:
		url, err := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveBg(n.Name) })
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"type": "bg", "name": n.Name, "url": url, "transition": n.Transition}, nil

	case *ast.CharShowNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveCharacter(n.Character, n.PoseExpr) })
		return map[string]interface{}{"type": "char_show", "character": n.Character, "pose_expr": n.PoseExpr, "url": url, "position": n.Position}, nil

	case *ast.CharHideNode:
		m := map[string]interface{}{"type": "char_hide", "character": n.Character}
		if n.Transition != "" {
			m["transition"] = n.Transition
		}
		return m, nil

	case *ast.CharExprNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveCharacter(n.Character, n.PoseExpr) })
		m := map[string]interface{}{"type": "char_expr", "character": n.Character, "pose_expr": n.PoseExpr, "url": url}
		if n.Transition != "" {
			m["transition"] = n.Transition
		}
		return m, nil

	case *ast.CharMoveNode:
		return map[string]interface{}{"type": "char_move", "character": n.Character, "position": n.Position}, nil

	case *ast.CharBubbleNode:
		return map[string]interface{}{"type": "bubble", "character": n.Character, "bubble_type": n.BubbleType}, nil

	case *ast.CgShowNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveCg(n.Name) })
		m := map[string]interface{}{"type": "cg_show", "name": n.Name, "url": url}
		if n.Transition != "" {
			m["transition"] = n.Transition
		}
		if n.Body != nil {
			steps, err := e.emitNodes(n.Body)
			if err != nil {
				return nil, err
			}
			m["steps"] = steps
		}
		return m, nil

	case *ast.DialogueNode:
		return map[string]interface{}{"type": "dialogue", "character": n.Character, "text": n.Text}, nil

	case *ast.NarratorNode:
		return map[string]interface{}{"type": "narrator", "text": n.Text}, nil

	case *ast.YouNode:
		return map[string]interface{}{"type": "you", "text": n.Text}, nil

	case *ast.PhoneShowNode:
		msgs, err := e.emitNodes(n.Body)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"type": "phone_show", "messages": msgs}, nil

	case *ast.PhoneHideNode:
		return map[string]interface{}{"type": "phone_hide"}, nil

	case *ast.TextMessageNode:
		return map[string]interface{}{"type": "text_message", "direction": n.Direction, "character": n.Character, "text": n.Content}, nil

	case *ast.MusicPlayNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveMusic(n.Name) })
		return map[string]interface{}{"type": "music_play", "name": n.Name, "url": url}, nil

	case *ast.MusicCrossfadeNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveMusic(n.Name) })
		return map[string]interface{}{"type": "music_crossfade", "name": n.Name, "url": url}, nil

	case *ast.MusicFadeoutNode:
		return map[string]interface{}{"type": "music_fadeout"}, nil

	case *ast.SfxPlayNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveSfx(n.Name) })
		return map[string]interface{}{"type": "sfx_play", "name": n.Name, "url": url}, nil

	case *ast.MinigameNode:
		url, _ := e.resolveOrWarn(func() (string, error) { return e.resolver.ResolveMinigame(n.GameID) })
		onResults := make(map[string]interface{})
		for key, nodes := range n.OnResult {
			steps, err := e.emitNodes(nodes)
			if err != nil {
				return nil, err
			}
			onResults[key] = steps
		}
		return map[string]interface{}{"type": "minigame", "game_id": n.GameID, "game_url": url, "attr": n.Attr, "on_results": onResults}, nil

	case *ast.ChoiceNode:
		var options []interface{}
		for _, opt := range n.Options {
			o := map[string]interface{}{
				"id":   opt.ID,
				"mode": opt.Mode,
				"text": opt.Text,
			}
			if opt.Check != nil {
				o["check"] = map[string]interface{}{"attr": opt.Check.Attr, "dc": opt.Check.DC}
			}
			if opt.OnSuccess != nil {
				steps, err := e.emitNodes(opt.OnSuccess)
				if err != nil {
					return nil, err
				}
				o["on_success"] = steps
			}
			if opt.OnFail != nil {
				steps, err := e.emitNodes(opt.OnFail)
				if err != nil {
					return nil, err
				}
				o["on_fail"] = steps
			}
			if opt.Body != nil {
				steps, err := e.emitNodes(opt.Body)
				if err != nil {
					return nil, err
				}
				o["steps"] = steps
			}
			options = append(options, o)
		}
		return map[string]interface{}{"type": "choice", "options": options}, nil

	case *ast.XpNode:
		return map[string]interface{}{"type": "xp", "delta": n.Delta}, nil

	case *ast.SanNode:
		return map[string]interface{}{"type": "san", "delta": n.Delta}, nil

	case *ast.AffectionNode:
		return map[string]interface{}{"type": "affection", "character": n.Character, "delta": n.Delta}, nil

	case *ast.SignalNode:
		return map[string]interface{}{"type": "signal", "event": n.Event}, nil

	case *ast.ButterflyNode:
		return map[string]interface{}{"type": "butterfly", "description": n.Description}, nil

	case *ast.IfNode:
		m := map[string]interface{}{"type": "if", "condition": n.Condition}
		thenSteps, err := e.emitNodes(n.Then)
		if err != nil {
			return nil, err
		}
		m["then"] = thenSteps
		if n.Else != nil {
			elseSteps, err := e.emitNodes(n.Else)
			if err != nil {
				return nil, err
			}
			m["else"] = elseSteps
		}
		return m, nil

	case *ast.LabelNode:
		return map[string]interface{}{"type": "label", "name": n.Name}, nil

	case *ast.GotoNode:
		return map[string]interface{}{"type": "goto", "target": n.Target}, nil

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

func (e *Emitter) emitGates(gates *ast.GatesBlock) map[string]interface{} {
	var rules []interface{}
	for _, g := range gates.Gates {
		rule := map[string]interface{}{
			"target": g.Target,
			"type":   g.GateType,
		}
		if g.Trigger != nil {
			rule["trigger"] = map[string]interface{}{
				"option_id": g.Trigger.OptionID,
				"result":    g.Trigger.CheckResult,
			}
		}
		if g.Condition != "" {
			rule["condition"] = g.Condition
		}
		rules = append(rules, rule)
	}
	return map[string]interface{}{
		"rules":   rules,
		"default": gates.Default,
	}
}

func (e *Emitter) resolveOrWarn(fn func() (string, error)) (string, error) {
	url, err := fn()
	if err != nil {
		e.warnings = append(e.warnings, err.Error())
		return "", nil
	}
	return url, nil
}

func extractBranchKey(episodeID string) string {
	parts := strings.Split(episodeID, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return episodeID
}

func extractSeq(episodeID string) int {
	parts := strings.Split(episodeID, ":")
	if len(parts) > 1 {
		n, _ := strconv.Atoi(parts[1])
		return n
	}
	return 0
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/emitter/ -v
```

Expected: all emitter tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/emitter/
git commit -m "feat: JSON emitter converts AST to player-ready JSON with resolved URLs"
```

---

### Task 7: Validator

**Files:**
- Create: `internal/validator/validator.go`
- Create: `internal/validator/validator_test.go`

- [ ] **Step 1: Write validator tests**

Create `internal/validator/validator_test.go`:

```go
package validator

import (
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

func TestValidGatesHasDefault(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Gates:     &ast.GatesBlock{Default: "main:02"},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestMissingGates(t *testing.T) {
	ep := &ast.Episode{BranchKey: "main:01", Title: "Test"}
	errs := Validate(ep)
	hasGateErr := false
	for _, e := range errs {
		if e.Code == "MISSING_GATES" {
			hasGateErr = true
		}
	}
	if !hasGateErr {
		t.Fatal("expected MISSING_GATES error")
	}
}

func TestGotoWithoutLabel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body:      []ast.Node{&ast.GotoNode{Target: "MISSING"}},
		Gates:     &ast.GatesBlock{Default: "main:02"},
	}
	errs := Validate(ep)
	hasErr := false
	for _, e := range errs {
		if e.Code == "GOTO_NO_LABEL" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Fatal("expected GOTO_NO_LABEL error")
	}
}

func TestGotoWithLabel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.GotoNode{Target: "POINT"},
			&ast.LabelNode{Name: "POINT"},
		},
		Gates: &ast.GatesBlock{Default: "main:02"},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestBraveOptionWithoutCheck(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []ast.OptionNode{
					{ID: "A", Mode: "brave", Text: "Fight", Check: nil},
				},
			},
		},
		Gates: &ast.GatesBlock{Default: "main:02"},
	}
	errs := Validate(ep)
	hasErr := false
	for _, e := range errs {
		if e.Code == "BRAVE_NO_CHECK" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Fatal("expected BRAVE_NO_CHECK error")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
go test ./internal/validator/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement validator**

Create `internal/validator/validator.go`:

```go
package validator

import (
	"fmt"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

type Error struct {
	Code    string
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func Validate(ep *ast.Episode) []Error {
	var errs []Error

	// Collect labels
	labels := make(map[string]bool)
	gotos := make(map[string]bool)
	collectLabelsAndGotos(ep.Body, labels, gotos)

	// Check gotos reference existing labels
	for target := range gotos {
		if !labels[target] {
			errs = append(errs, Error{Code: "GOTO_NO_LABEL", Message: fmt.Sprintf("@goto %s has no matching @label", target)})
		}
	}

	// Check gates exist
	if ep.Gates == nil {
		errs = append(errs, Error{Code: "MISSING_GATES", Message: "episode missing @gates block"})
	} else if ep.Gates.Default == "" {
		errs = append(errs, Error{Code: "MISSING_DEFAULT", Message: "@gates missing @default"})
	}

	// Validate choices
	validateNodes(ep.Body, &errs)

	return errs
}

func collectLabelsAndGotos(nodes []ast.Node, labels, gotos map[string]bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.LabelNode:
			labels[n.Name] = true
		case *ast.GotoNode:
			gotos[n.Target] = true
		case *ast.IfNode:
			collectLabelsAndGotos(n.Then, labels, gotos)
			collectLabelsAndGotos(n.Else, labels, gotos)
		case *ast.ChoiceNode:
			for _, opt := range n.Options {
				collectLabelsAndGotos(opt.Body, labels, gotos)
				collectLabelsAndGotos(opt.OnSuccess, labels, gotos)
				collectLabelsAndGotos(opt.OnFail, labels, gotos)
			}
		case *ast.CgShowNode:
			collectLabelsAndGotos(n.Body, labels, gotos)
		case *ast.MinigameNode:
			for _, nodes := range n.OnResult {
				collectLabelsAndGotos(nodes, labels, gotos)
			}
		}
	}
}

func validateNodes(nodes []ast.Node, errs *[]Error) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ChoiceNode:
			for _, opt := range n.Options {
				if opt.Mode == "brave" && opt.Check == nil {
					*errs = append(*errs, Error{Code: "BRAVE_NO_CHECK", Message: fmt.Sprintf("option %s is brave but has no check block", opt.ID)})
				}
				if opt.Mode == "brave" && (opt.OnSuccess == nil || opt.OnFail == nil) {
					*errs = append(*errs, Error{Code: "BRAVE_MISSING_OUTCOME", Message: fmt.Sprintf("option %s is brave but missing @on success or @on fail", opt.ID)})
				}
				validateNodes(opt.Body, errs)
				validateNodes(opt.OnSuccess, errs)
				validateNodes(opt.OnFail, errs)
			}
		case *ast.IfNode:
			validateNodes(n.Then, errs)
			validateNodes(n.Else, errs)
		case *ast.CgShowNode:
			validateNodes(n.Body, errs)
		case *ast.MinigameNode:
			for _, nodes := range n.OnResult {
				validateNodes(nodes, errs)
			}
		}
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/validator/ -v
```

Expected: all validator tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/validator/
git commit -m "feat: validator checks semantic correctness of parsed AST"
```

---

### Task 8: CLI Wiring + End-to-End Test

**Files:**
- Modify: `cmd/nrs/main.go`
- Create: `testdata/ep01.md` (Episode 1 from spec Appendix A)
- Create: `testdata/minimal.md`

- [ ] **Step 1: Create test scripts**

Create `testdata/minimal.md`:

```
@episode main:01 "Test" {
  @bg set classroom_morning fade
  NARRATOR: Hello world.
  YOU: Thinking deeply.

  @gates {
    @default main:02
  }
}
```

Create `testdata/ep01.md`: Copy the Episode 1 script from the spec Appendix A (lines 751-918 of the design doc).

- [ ] **Step 2: Implement full CLI**

Replace `cmd/nrs/main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/emitter"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/parser"
	"github.com/cdotlock/moonshort-script/internal/resolver"
	"github.com/cdotlock/moonshort-script/internal/validator"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}

	cmd := os.Args[1]
	target := os.Args[2]
	assetsPath := ""
	outputPath := ""

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--assets":
			if i+1 < len(os.Args) {
				assetsPath = os.Args[i+1]
				i++
			}
		case "-o":
			if i+1 < len(os.Args) {
				outputPath = os.Args[i+1]
				i++
			}
		}
	}

	switch cmd {
	case "compile":
		if err := compile(target, assetsPath, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := validate(target, assetsPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK")
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: nrs <compile|validate> <file.md|dir/> [--assets mapping.yaml] [-o output.json]")
	os.Exit(1)
}

func compile(target, assetsPath, outputPath string) error {
	// Check if target is a directory
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return compileDir(target, assetsPath, outputPath)
	}
	return compileFile(target, assetsPath, outputPath)
}

func compileFile(path, assetsPath, outputPath string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	l := lexer.New(string(raw))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// Validate
	if errs := validator.Validate(ep); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, e)
		}
	}

	// Resolve assets
	var r emitter.AssetResolver
	if assetsPath != "" {
		rr, err := resolver.NewFromFile(assetsPath)
		if err != nil {
			return err
		}
		r = rr
	} else {
		r = &noopResolver{}
	}

	em := emitter.New(r)
	result, err := em.Emit(ep)
	if err != nil {
		return err
	}

	for _, w := range em.Warnings() {
		fmt.Fprintf(os.Stderr, "warn: %s\n", w)
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, result, 0644)
	}
	fmt.Println(string(result))
	return nil
}

func compileDir(dir, assetsPath, outputPath string) error {
	var episodes []json.RawMessage

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		l := lexer.New(string(raw))
		p := parser.New(l)
		ep, err := p.Parse()
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		if errs := validator.Validate(ep); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, e)
			}
		}

		var r emitter.AssetResolver
		if assetsPath != "" {
			rr, err := resolver.NewFromFile(assetsPath)
			if err != nil {
				return err
			}
			r = rr
		} else {
			r = &noopResolver{}
		}

		em := emitter.New(r)
		result, err := em.Emit(ep)
		if err != nil {
			return fmt.Errorf("emit %s: %w", path, err)
		}

		for _, w := range em.Warnings() {
			fmt.Fprintf(os.Stderr, "warn: %s\n", w)
		}

		episodes = append(episodes, result)
		return nil
	})
	if err != nil {
		return err
	}

	combined, err := json.MarshalIndent(episodes, "", "  ")
	if err != nil {
		return err
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, combined, 0644)
	}
	fmt.Println(string(combined))
	return nil
}

func validate(target, assetsPath string) error {
	raw, err := os.ReadFile(target)
	if err != nil {
		return err
	}

	l := lexer.New(string(raw))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		return err
	}

	errs := validator.Validate(ep)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%v\n", e)
		}
		return fmt.Errorf("%d validation error(s)", len(errs))
	}

	return nil
}

type noopResolver struct{}

func (r *noopResolver) ResolveBg(name string) (string, error)                  { return "", nil }
func (r *noopResolver) ResolveCharacter(char, pose string) (string, error)     { return "", nil }
func (r *noopResolver) ResolveMusic(name string) (string, error)               { return "", nil }
func (r *noopResolver) ResolveSfx(name string) (string, error)                 { return "", nil }
func (r *noopResolver) ResolveCg(name string) (string, error)                  { return "", nil }
func (r *noopResolver) ResolveMinigame(gameID string) (string, error)          { return "", nil }
```

- [ ] **Step 3: Build and test minimal script**

```bash
make build
./bin/nrs compile testdata/minimal.md
```

Expected: JSON output to stdout with episode_id, title, steps, gates.

- [ ] **Step 4: Test with mapping file**

```bash
./bin/nrs compile testdata/ep01.md --assets testdata/mapping.yaml -o testdata/ep01_output.json
```

Expected: JSON file with resolved OSS URLs.

- [ ] **Step 5: Test validate command**

```bash
./bin/nrs validate testdata/minimal.md
```

Expected: prints "OK".

- [ ] **Step 6: Commit**

```bash
git add cmd/nrs/ testdata/
git commit -m "feat: complete CLI with compile and validate commands"
```

---

### Task 9: README + Push to GitHub

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Create `README.md`:

````markdown
# moonshort-script

NoRules Script (NRS) interpreter for MobAI interactive visual novels.

Parses `.md` script files into structured JSON for the frontend player, resolving asset semantic names to OSS URLs.

## Install

```bash
go build -o bin/nrs ./cmd/nrs
```

## Usage

```bash
# Compile a single episode
nrs compile episode.md --assets mapping.yaml -o output.json

# Compile an entire novel directory
nrs compile novel_001/main/ --assets mapping.yaml -o novel.json

# Validate syntax only
nrs validate episode.md
```

## Script Format

See [NRS Script Format Design v2.1](docs/specs/2026-04-15-nrs-script-format-design.md) for the complete specification.

## Development

```bash
make test    # Run all tests
make build   # Build binary
```
````

- [ ] **Step 2: Init git remote and push**

```bash
cd "/Users/Clock/moonshort backend/moonshort-script"
git remote add origin https://github.com/cdotlock/moonshort-script.git
git branch -M main
git push -u origin main
```

- [ ] **Step 3: Commit README**

```bash
git add README.md
git commit -m "docs: add README with usage instructions"
git push
```

---

## Summary

| Task | What it builds | Test coverage |
|------|---------------|---------------|
| 1 | Project scaffold, token types, CLI stub | Build verification |
| 2 | Lexer (tokenizer) | 7 unit tests: directives, dialogue, strings, comments, conditions, signed numbers, line tracking |
| 3 | AST node types | Compile verification |
| 4 | Parser (all directive types) | 4 unit tests: minimal script, char directives, audio, state changes |
| 5 | Asset resolver (YAML mapping → URLs) | 5 unit tests: load, bg, character, music, missing, minigame |
| 6 | JSON emitter (AST → JSON) | 2 unit tests: minimal emit, choice emit |
| 7 | Validator (semantic checks) | 5 unit tests: valid gates, missing gates, goto/label, brave without check |
| 8 | CLI wiring + end-to-end | Manual e2e: compile minimal, compile ep01, validate |
| 9 | README + push to GitHub | N/A |
