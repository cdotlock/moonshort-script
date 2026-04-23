# `@signal int` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `int` as a second kind under `@signal`, enabling authors to declare and mutate cross-episode persistent integer counters (e.g. `@signal int rejections +1`) and read them in `@if` conditions as bare names (`@if (rejections >= 3)`).

**Architecture:** Extend the existing `@signal` grammar: parser dispatches on `kind` and parses three write forms (`= <int>`, `+<N>`, `-<N>`); AST replaces `SignalNode.Event` usage with a kind-discriminated struct (new `Name`/`Op`/`Value` fields for `int`); emitter outputs a new JSON shape under the same `"type":"signal"` step. Read side is **unchanged** — bare-name comparison in `@if` already works via existing `left.kind="value"` AST.

**Tech Stack:** Go 1.x, existing packages: `internal/token`, `internal/lexer`, `internal/ast`, `internal/parser`, `internal/validator`, `internal/emitter`. Tests use the stdlib `testing` package.

**Reference spec:** `docs/superpowers/specs/2026-04-23-signal-int-design.md`

---

## File Structure

| File | Role |
|------|------|
| `internal/token/token.go` | Add `ASSIGN` token type for `=` |
| `internal/lexer/lexer.go` | Tokenize standalone `=` as `ASSIGN` (preserving `==` → `EQ`) |
| `internal/ast/ast.go` | Extend `SignalNode` with `Name`/`Op`/`Value` fields; add `SignalKindInt` const; add `SignalOp*` consts |
| `internal/parser/parser.go` | Extend `parseSignal` to dispatch on kind; parse `int` forms |
| `internal/validator/validator.go` | Accept `int` in `validSignalKinds`; add reserved-name check for `@signal int <name>` |
| `internal/emitter/emitter.go` | Emit `kind=int` signal steps with `name`/`op`/`value` fields |
| `internal/lexer/lexer_test.go` (existing or new) | ASSIGN tokenization tests |
| `internal/parser/parser_test.go` | `@signal int` parse happy/error paths |
| `internal/validator/validator_test.go` | Reserved-name collision test |
| `internal/emitter/emitter_test.go` | Emit-shape tests for `int` signals |
| `testdata/feature_parade/stress.md` | Add `@signal int` usage (writes + `@if` read) |
| `testdata/feature_parade/stress_output.json` | Regenerated golden |
| `MSS-SPEC.md` | Spec doc update |
| `skills/mss-scriptwriting/references/MSS-SPEC.md` | Mirror of above |
| `skills/mss-scriptwriting/SKILL.md` | Usage guidance update |
| `skills/mss-scriptwriting/references/directive-table.md` | Directive table update |

---

## Task 1: Add `ASSIGN` token type

**Files:**
- Modify: `internal/token/token.go`

- [ ] **Step 1: Add the ASSIGN constant**

In `internal/token/token.go`, in the token `Type` constant block, add:

```go
ASSIGN Type = "ASSIGN" // single '=' (distinct from EQ '==')
```

Add it next to `EQ`. Also update the `String()` switch (whatever the debug/stringify function is) to include `ASSIGN` if such a function exists. Grep for `case EQ:` to find it.

- [ ] **Step 2: Commit**

```bash
git add internal/token/token.go
git commit -m "feat(token): add ASSIGN token for single '=' separator"
```

---

## Task 2: Tokenize standalone `=` as ASSIGN

**Files:**
- Modify: `internal/lexer/lexer.go` (around line 224, the `==` case)
- Test: `internal/lexer/lexer_test.go` (create if missing, else append)

- [ ] **Step 1: Write failing lexer test**

If `internal/lexer/lexer_test.go` does not exist, create it with package `lexer` or `lexer_test`, matching the project convention (check existing package decl of other `_test.go` files with `head -1 internal/*/`*_test.go`).

Add test:

```go
func TestLexSingleEqualsAsAssign(t *testing.T) {
	l := New("=")
	tok := l.Next()
	if tok.Type != token.ASSIGN {
		t.Fatalf("expected ASSIGN, got %s (%q)", tok.Type, tok.Literal)
	}
	if tok.Literal != "=" {
		t.Fatalf("expected literal '=', got %q", tok.Literal)
	}
}

func TestLexDoubleEqualsStillEQ(t *testing.T) {
	l := New("==")
	tok := l.Next()
	if tok.Type != token.EQ {
		t.Fatalf("expected EQ, got %s (%q)", tok.Type, tok.Literal)
	}
}
```

Ensure imports include `"github.com/cdotlock/moonshort-script/internal/token"`.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/lexer/ -run TestLexSingleEqualsAsAssign -v
```

Expected: FAIL (either `=` produces `ILLEGAL` or panics).

- [ ] **Step 3: Implement the ASSIGN lex case**

In `internal/lexer/lexer.go`, after the existing `==` case (around line 224-226), add a fallback single-`=` case:

```go
case ch == '=' && l.peekAt(1) == '=':
    l.advance(); l.advance()
    return l.makeToken(token.EQ, "==", line, col)

case ch == '=':
    l.advance()
    return l.makeToken(token.ASSIGN, "=", line, col)
```

The order matters: the two-char `==` match must come before the single-char `=` fallback.

- [ ] **Step 4: Run both new tests**

```bash
go test ./internal/lexer/ -run "TestLexSingleEqualsAsAssign|TestLexDoubleEqualsStillEQ" -v
```

Expected: PASS for both.

- [ ] **Step 5: Run full lexer suite to verify no regressions**

```bash
go test ./internal/lexer/ -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/lexer/lexer.go internal/lexer/lexer_test.go
git commit -m "feat(lexer): emit ASSIGN token for standalone '='"
```

---

## Task 3: Extend SignalNode AST

**Files:**
- Modify: `internal/ast/ast.go` (around lines 442-462)

- [ ] **Step 1: Extend SignalNode struct and add constants**

Replace the existing signal-kind const block and `SignalNode` (lines 442-462) with:

```go
// Valid signal kinds.
//
//   - "mark" — persistent boolean flag (set-once-true)
//   - "int"  — persistent integer variable (assign / increment / decrement)
//
// The kind word is mandatory in source syntax (@signal <kind> ...).
const (
	SignalKindMark = "mark"
	SignalKindInt  = "int"
)

// Int-signal operators. @signal int <name> <op> <value>.
//
//   - "="  — unconditional assignment (value may be negative)
//   - "+"  — increment by value (value must be non-negative)
//   - "-"  — decrement by value (value must be non-negative)
const (
	SignalOpAssign = "="
	SignalOpAdd    = "+"
	SignalOpSub    = "-"
)

// SignalNode emits a persistent state write. Two kinds are supported:
//
//   - SignalKindMark: sets a named boolean flag to true. Queried via
//     @if (NAME) (resolved as FlagCondition). Event carries the flag
//     name (e.g. "HIGH_HEEL_EP05"). Author discipline: only use for
//     key story points a later reader (@if / achievement guard) needs.
//
//   - SignalKindInt: mutates a named persistent integer variable.
//     Name carries the variable id (snake_case lowercase). Op is one
//     of SignalOp{Assign,Add,Sub}. Value is the operand (Assign may
//     be negative; Add/Sub are always non-negative). Queried via
//     @if (NAME <op> N) comparison (bare name, resolved as a regular
//     value-comparison through the existing left.kind="value" path).
//     Author discipline: free use for counters and thresholds; the
//     "marks are precious" rule does NOT apply here.
//
// For backward compatibility, Event is retained and used only for
// SignalKindMark. Name/Op/Value are used only for SignalKindInt.
type SignalNode struct {
	ConcurrentFlag
	Kind  string // SignalKindMark or SignalKindInt

	// Fields for SignalKindMark.
	Event string // e.g. "HIGH_HEEL_EP05"

	// Fields for SignalKindInt.
	Name  string // e.g. "rejections"
	Op    string // SignalOp{Assign,Add,Sub}
	Value int    // operand
}

func (s *SignalNode) nodeType() string { return "signal" }
```

- [ ] **Step 2: Verify the package still compiles**

```bash
go build ./internal/ast/
```

Expected: no output (success). If the test file references the old shape, it will fail compile in a later package — that's fine; we fix each consumer explicitly below.

- [ ] **Step 3: Run full build to see which packages break**

```bash
go build ./...
```

Expected: either success (if no caller relied on struct shape) or specific errors pointing to `SignalNode` usages. Record the errors — subsequent tasks will fix parser, validator, and emitter.

- [ ] **Step 4: Commit**

```bash
git add internal/ast/ast.go
git commit -m "feat(ast): extend SignalNode with int-kind fields"
```

---

## Task 4: Parser — happy path for `@signal int x = N`

**Files:**
- Modify: `internal/parser/parser.go` (function `parseSignal`, around lines 891-926)
- Test: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing test for `@signal int x = 0`**

Append to `internal/parser/parser_test.go`:

```go
func TestParseSignalIntAssignZero(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections = 0
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ep.Body) < 1 {
		t.Fatalf("expected at least 1 body node")
	}
	sig, ok := ep.Body[0].(*ast.SignalNode)
	if !ok {
		t.Fatalf("expected SignalNode, got %T", ep.Body[0])
	}
	if sig.Kind != ast.SignalKindInt {
		t.Fatalf("expected kind %q, got %q", ast.SignalKindInt, sig.Kind)
	}
	if sig.Name != "rejections" {
		t.Fatalf("expected name 'rejections', got %q", sig.Name)
	}
	if sig.Op != ast.SignalOpAssign {
		t.Fatalf("expected op '=', got %q", sig.Op)
	}
	if sig.Value != 0 {
		t.Fatalf("expected value 0, got %d", sig.Value)
	}
}
```

Make sure the test file already imports `"github.com/cdotlock/moonshort-script/internal/ast"`, `"github.com/cdotlock/moonshort-script/internal/lexer"`, `"github.com/cdotlock/moonshort-script/internal/parser"`. If not, add them. (Check first line / existing tests.)

- [ ] **Step 2: Run and verify it fails**

```bash
go test ./internal/parser/ -run TestParseSignalIntAssignZero -v
```

Expected: FAIL (invalid signal kind or similar).

- [ ] **Step 3: Update `validSignalKinds` whitelist in parser.go**

At top of `internal/parser/parser.go` (around line 35), update:

```go
var validSignalKinds = map[string]bool{
	"mark": true,
	"int":  true,
}
```

- [ ] **Step 4: Rewrite `parseSignal` to dispatch by kind**

Replace the entire `parseSignal` function (around lines 891-926) with:

```go
// parseSignal parses: @signal <kind> ...
//
// Two kinds:
//   - mark: @signal mark <event>    — event is IDENT or STRING
//   - int:  @signal int <name> <op> <value>
//           op ∈ { "=", "+", "-" }; value is an integer literal.
//           "=" accepts NUMBER or SIGNED_NUMBER (may be negative).
//           "+"/"-" accepts NUMBER only (the sign is in the operator).
func (p *Parser) parseSignal() (ast.Node, error) {
	p.advance() // consume "signal"
	kindTok, err := p.expect(token.IDENT)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected signal kind after '@signal' (one of: mark, int), got %s (%q)",
			kindTok.Line, kindTok.Col, kindTok.Type, kindTok.Literal)
	}
	if !validSignalKinds[kindTok.Literal] {
		return nil, fmt.Errorf("line %d col %d: invalid signal kind %q (valid: mark, int)",
			kindTok.Line, kindTok.Col, kindTok.Literal)
	}

	switch kindTok.Literal {
	case ast.SignalKindMark:
		return p.parseSignalMark(kindTok)
	case ast.SignalKindInt:
		return p.parseSignalInt(kindTok)
	default:
		// Unreachable due to validSignalKinds whitelist above.
		return nil, fmt.Errorf("line %d col %d: unhandled signal kind %q", kindTok.Line, kindTok.Col, kindTok.Literal)
	}
}

// parseSignalMark parses the event name (IDENT or STRING) following
// `@signal mark`.
func (p *Parser) parseSignalMark(kindTok token.Token) (ast.Node, error) {
	var event string
	if p.cur.Type == token.STRING {
		event = p.cur.Literal
		p.advance()
	} else {
		tok, err := p.expect(token.IDENT)
		if err != nil {
			return nil, fmt.Errorf("line %d col %d: expected event name after '@signal mark', got %s (%q)",
				tok.Line, tok.Col, tok.Type, tok.Literal)
		}
		event = tok.Literal
	}
	return &ast.SignalNode{Kind: ast.SignalKindMark, Event: event}, nil
}

// parseSignalInt parses `<name> <op> <value>` following `@signal int`.
func (p *Parser) parseSignalInt(kindTok token.Token) (ast.Node, error) {
	nameTok, err := p.expect(token.IDENT)
	if err != nil {
		return nil, fmt.Errorf("line %d col %d: expected variable name after '@signal int', got %s (%q)",
			nameTok.Line, nameTok.Col, nameTok.Type, nameTok.Literal)
	}

	// Operator: ASSIGN ('='), or a sign merged into SIGNED_NUMBER ('+N' / '-N').
	switch p.cur.Type {
	case token.ASSIGN:
		p.advance() // consume '='
		// Value: NUMBER or SIGNED_NUMBER.
		if p.cur.Type != token.NUMBER && p.cur.Type != token.SIGNED_NUMBER {
			return nil, fmt.Errorf("line %d col %d: expected integer literal after '@signal int %s =', got %s (%q)",
				p.cur.Line, p.cur.Col, nameTok.Literal, p.cur.Type, p.cur.Literal)
		}
		valTok := p.cur
		p.advance()
		n, err := strconv.Atoi(valTok.Literal)
		if err != nil {
			return nil, fmt.Errorf("line %d col %d: invalid integer %q in '@signal int %s = ...': %v",
				valTok.Line, valTok.Col, valTok.Literal, nameTok.Literal, err)
		}
		return &ast.SignalNode{
			Kind:  ast.SignalKindInt,
			Name:  nameTok.Literal,
			Op:    ast.SignalOpAssign,
			Value: n,
		}, nil

	case token.SIGNED_NUMBER:
		// '+N' or '-N' lexed as a single signed number token.
		valTok := p.cur
		p.advance()
		n, err := strconv.Atoi(valTok.Literal)
		if err != nil {
			return nil, fmt.Errorf("line %d col %d: invalid signed integer %q in '@signal int %s ...': %v",
				valTok.Line, valTok.Col, valTok.Literal, nameTok.Literal, err)
		}
		if n == 0 {
			return nil, fmt.Errorf("line %d col %d: '@signal int %s +0' or '-0' is meaningless; use '@signal int %s = 0' to assign",
				valTok.Line, valTok.Col, nameTok.Literal, nameTok.Literal)
		}
		op := ast.SignalOpAdd
		abs := n
		if n < 0 {
			op = ast.SignalOpSub
			abs = -n
		}
		return &ast.SignalNode{
			Kind:  ast.SignalKindInt,
			Name:  nameTok.Literal,
			Op:    op,
			Value: abs,
		}, nil

	default:
		return nil, fmt.Errorf("line %d col %d: expected '=', '+N', or '-N' after '@signal int %s', got %s (%q)",
			p.cur.Line, p.cur.Col, nameTok.Literal, p.cur.Type, p.cur.Literal)
	}
}
```

Add `"strconv"` to the parser's imports if not already present.

- [ ] **Step 5: Re-run the failing test**

```bash
go test ./internal/parser/ -run TestParseSignalIntAssignZero -v
```

Expected: PASS.

- [ ] **Step 6: Run full parser test suite (regressions for `@signal mark`)**

```bash
go test ./internal/parser/ -v
```

Expected: all tests PASS (including the existing `@signal mark` cases).

- [ ] **Step 7: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat(parser): parse @signal int <name> = <int>"
```

---

## Task 5: Parser — increment/decrement forms

**Files:**
- Modify: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing tests for +N and -N**

Append to `internal/parser/parser_test.go`:

```go
func TestParseSignalIntAdd(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections +1
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sig := ep.Body[0].(*ast.SignalNode)
	if sig.Op != ast.SignalOpAdd || sig.Value != 1 || sig.Name != "rejections" {
		t.Fatalf("got kind=%q name=%q op=%q value=%d", sig.Kind, sig.Name, sig.Op, sig.Value)
	}
}

func TestParseSignalIntSub(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections -2
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sig := ep.Body[0].(*ast.SignalNode)
	if sig.Op != ast.SignalOpSub || sig.Value != 2 {
		t.Fatalf("got op=%q value=%d", sig.Op, sig.Value)
	}
}

func TestParseSignalIntAssignNegative(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int x = -3
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sig := ep.Body[0].(*ast.SignalNode)
	if sig.Op != ast.SignalOpAssign || sig.Value != -3 {
		t.Fatalf("got op=%q value=%d", sig.Op, sig.Value)
	}
}
```

- [ ] **Step 2: Run**

```bash
go test ./internal/parser/ -run "TestParseSignalIntAdd|TestParseSignalIntSub|TestParseSignalIntAssignNegative" -v
```

Expected: all three PASS (Task 4's implementation already covers these; this task verifies coverage).

- [ ] **Step 3: Commit**

```bash
git add internal/parser/parser_test.go
git commit -m "test(parser): cover @signal int +/-/assign-negative forms"
```

---

## Task 6: Parser — error diagnostics

**Files:**
- Modify: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing tests for error cases**

Append:

```go
func TestParseSignalIntErrors(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		substr string // expected substring in error
	}{
		{"missing name", `@episode main:01 "t" { @signal int @ending complete }`, "expected variable name"},
		{"missing op", `@episode main:01 "t" { @signal int rejections
@ending complete }`, "expected '=', '+N', or '-N'"},
		{"missing value after =", `@episode main:01 "t" { @signal int x = @ending complete }`, "expected integer literal"},
		{"non-integer after =", `@episode main:01 "t" { @signal int x = abc
@ending complete }`, "expected integer literal"},
		{"plus-zero rejected", `@episode main:01 "t" { @signal int x +0
@ending complete }`, "meaningless"},
		{"unknown kind", `@episode main:01 "t" { @signal foo x = 0
@ending complete }`, "invalid signal kind"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := lexer.New(c.src)
			p := parser.New(l)
			_, err := p.Parse()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), c.substr) {
				t.Fatalf("expected error containing %q, got: %v", c.substr, err)
			}
		})
	}
}
```

Add `"strings"` to imports if missing.

**Note on `+0`**: the lexer emits `SIGNED_NUMBER` for `+0`, and our parser rejects zero in the `+/-` form (see Task 4 step 4). If the SIGNED_NUMBER lexer skips `+0` entirely because of some edge case, adjust the test's error substring to match actual behavior — but first run and see.

- [ ] **Step 2: Run**

```bash
go test ./internal/parser/ -run TestParseSignalIntErrors -v
```

Expected: all sub-tests PASS (implementation from Task 4 produces all these errors).

- [ ] **Step 3: Regression — `@signal mark` still works**

Grep the existing tests to make sure the `@signal mark` golden test passes:

```bash
go test ./internal/parser/ -v | grep -i signal
```

Expected: all signal-related tests PASS, none removed.

- [ ] **Step 4: Commit**

```bash
git add internal/parser/parser_test.go
git commit -m "test(parser): cover @signal int parse errors"
```

---

## Task 7: Validator — accept `int` kind and reserved-name check

**Files:**
- Modify: `internal/validator/validator.go`
- Test: `internal/validator/validator_test.go`

- [ ] **Step 1: Write failing validator test**

Append to `internal/validator/validator_test.go`:

```go
func TestValidateSignalIntReservedName(t *testing.T) {
	// @signal int san should fail — "san" is an engine-reserved name.
	src := `@episode main:01 "t" {
  @signal int san = 0
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs := validator.Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == validator.ReservedIntName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ReservedIntName error, got: %v", errs)
	}
}

func TestValidateSignalIntOK(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections = 0
  @signal int rejections +1
  @ending complete
}`
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	errs := validator.Validate(ep)
	for _, e := range errs {
		if e.Code == validator.ReservedIntName || e.Code == validator.InvalidSignalKind {
			t.Fatalf("unexpected error: %v", e)
		}
	}
}
```

Ensure the test file imports `"github.com/cdotlock/moonshort-script/internal/validator"`.

- [ ] **Step 2: Run and verify failure**

```bash
go test ./internal/validator/ -run "TestValidateSignalInt" -v
```

Expected: FAIL (because the validator doesn't yet accept `int` kind and doesn't have `ReservedIntName` code).

- [ ] **Step 3: Implement validator changes**

In `internal/validator/validator.go`:

**(a)** Add error code to the `const` block (around line 28):

```go
ReservedIntName = "RESERVED_INT_NAME"
```

**(b)** Extend `validSignalKinds` (around line 31):

```go
var validSignalKinds = map[string]bool{
	ast.SignalKindMark: true,
	ast.SignalKindInt:  true,
}
```

**(c)** Add reserved-name map near the other whitelists (around line 67):

```go
// reservedIntNames blocks @signal int declarations from shadowing
// engine-managed numeric values that scripts may only read.
//
// The list is intentionally conservative: concrete names the engine
// currently defines. Expand as the engine grows.
var reservedIntNames = map[string]bool{
	"san": true, "cha": true, "atk": true,
	"hp": true, "xp": true, "dex": true,
	"int": true, "str": true, "wis": true, "con": true,
}
```

**(d)** Update `checkSignals` (around line 132) to validate `int` name:

```go
func checkSignals(nodes []ast.Node, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.SignalNode:
			if !validSignalKinds[v.Kind] {
				*errs = append(*errs, Error{
					Code:    InvalidSignalKind,
					Message: fmt.Sprintf("@signal has invalid kind %q (valid: mark, int)", v.Kind),
				})
				continue
			}
			if v.Kind == ast.SignalKindInt {
				if reservedIntNames[v.Name] {
					*errs = append(*errs, Error{
						Code:    ReservedIntName,
						Message: fmt.Sprintf("@signal int %q: name collides with an engine-managed numeric value; choose a different name", v.Name),
					})
				}
			}
		case *ast.CgShowNode:
			checkSignals(v.Body, errs)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				checkSignals(opt.Body, errs)
			}
		case *ast.IfNode:
			checkSignals(v.Then, errs)
			checkSignals(v.Else, errs)
		case *ast.MinigameNode:
			checkSignals(v.Body, errs)
		case *ast.PhoneShowNode:
			checkSignals(v.Body, errs)
		}
	}
}
```

- [ ] **Step 4: Re-run the validator tests**

```bash
go test ./internal/validator/ -run "TestValidateSignalInt" -v
```

Expected: both PASS.

- [ ] **Step 5: Run full validator suite**

```bash
go test ./internal/validator/ -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/validator/validator.go internal/validator/validator_test.go
git commit -m "feat(validator): accept @signal int; block engine-reserved names"
```

---

## Task 8: Emitter — output `int` signal JSON

**Files:**
- Modify: `internal/emitter/emitter.go` (around lines 160-165)
- Test: `internal/emitter/emitter_test.go`

- [ ] **Step 1: Write failing emitter test**

Append to `internal/emitter/emitter_test.go`:

```go
func TestEmitSignalIntAssign(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections = 0
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "=",
		"value": float64(0), // JSON numbers decode as float64 via json.Unmarshal round-trip; if the helper returns raw int, compare int(0).
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalIntAdd(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections +1
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "+",
		"value": float64(1),
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalIntSub(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections -2
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "-",
		"value": float64(2),
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalMarkUnchanged(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal mark HIGH_HEEL_EP05
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"type":  "signal",
		"kind":  "mark",
		"event": "HIGH_HEEL_EP05",
	}
	assertStepEquals(t, step, want)
}
```

**Helper setup**: if `firstBodyStep` and `assertStepEquals` don't already exist in the emitter test file, either (a) reuse whatever helper the existing tests use (inspect `emitter_test.go` first with `head -60`) or (b) define them inline:

```go
// firstBodyStep parses + emits an episode source and returns the first
// step of the resulting JSON, as a generic map.
func firstBodyStep(t *testing.T, src string) map[string]interface{} {
	t.Helper()
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	e := emitter.New(nil) // nil resolver — acceptable because these tests don't use assets
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	data, _ := json.Marshal(out)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	steps := m["steps"].([]interface{})
	first := steps[0]
	// If the first step is a concurrency-group array of one, unwrap.
	if arr, ok := first.([]interface{}); ok && len(arr) == 1 {
		return arr[0].(map[string]interface{})
	}
	return first.(map[string]interface{})
}

func assertStepEquals(t *testing.T, got, want map[string]interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("step mismatch\nwant: %#v\n got: %#v", want, got)
	}
}
```

Add imports as needed: `"encoding/json"`, `"reflect"`, and the lexer/parser/emitter packages.

**Before writing the helpers**: run `head -80 internal/emitter/emitter_test.go` to see whether equivalent helpers already exist under different names; prefer reuse.

- [ ] **Step 2: Run and verify failure**

```bash
go test ./internal/emitter/ -run "TestEmitSignalInt" -v
```

Expected: FAIL — emitter currently always emits `kind`/`event`, regardless of the real signal kind.

- [ ] **Step 3: Update emitter's signal case**

In `internal/emitter/emitter.go`, replace the current signal case (around lines 160-165):

```go
case *ast.SignalNode:
    return e.emitSignal(v)
```

Then add a dedicated method below `emitAffection` (near line 430):

```go
func (e *Emitter) emitSignal(n *ast.SignalNode) map[string]interface{} {
	switch n.Kind {
	case ast.SignalKindMark:
		return map[string]interface{}{
			"type":  "signal",
			"kind":  "mark",
			"event": n.Event,
		}
	case ast.SignalKindInt:
		return map[string]interface{}{
			"type":  "signal",
			"kind":  "int",
			"name":  n.Name,
			"op":    n.Op,
			"value": n.Value,
		}
	default:
		// Defensive: unknown kind is caught by validator, but keep the
		// emitter resilient — emit a best-effort shape so downstream
		// debugging isn't hampered.
		return map[string]interface{}{
			"type": "signal",
			"kind": n.Kind,
		}
	}
}
```

- [ ] **Step 4: Run emitter tests**

```bash
go test ./internal/emitter/ -run "TestEmitSignal" -v
```

Expected: all four PASS (`Assign`, `Add`, `Sub`, `MarkUnchanged`).

- [ ] **Step 5: Run full emitter suite**

```bash
go test ./internal/emitter/ -v
```

Expected: all PASS, including existing golden tests (they don't yet use `@signal int`, so shouldn't change).

- [ ] **Step 6: Commit**

```bash
git add internal/emitter/emitter.go internal/emitter/emitter_test.go
git commit -m "feat(emitter): emit @signal int steps with name/op/value"
```

---

## Task 9: End-to-end — `@if` read path (already works, prove it)

**Files:**
- Test: `internal/emitter/emitter_test.go`

- [ ] **Step 1: Write test that reads an int variable in `@if`**

Append:

```go
func TestEmitIfReadsIntVariableAsComparison(t *testing.T) {
	src := `@episode main:01 "t" {
  @if (rejections >= 3) {
    NARRATOR: too many
  }
  @ending complete
}`
	step := firstBodyStep(t, src)
	// We expect an IfNode step whose condition is a comparison
	// with left.kind="value", name="rejections", op=">=", right=3.
	if step["type"] != "if" {
		t.Fatalf("expected if, got %v", step["type"])
	}
	cond := step["condition"].(map[string]interface{})
	if cond["type"] != "comparison" {
		t.Fatalf("expected comparison, got %v", cond["type"])
	}
	left := cond["left"].(map[string]interface{})
	if left["kind"] != "value" || left["name"] != "rejections" {
		t.Fatalf("unexpected left: %#v", left)
	}
	if cond["op"] != ">=" {
		t.Fatalf("unexpected op: %v", cond["op"])
	}
	if cond["right"].(float64) != 3 {
		t.Fatalf("unexpected right: %v", cond["right"])
	}
}
```

- [ ] **Step 2: Run**

```bash
go test ./internal/emitter/ -run TestEmitIfReadsIntVariableAsComparison -v
```

Expected: PASS with **no code change** — the existing comparison AST path already handles bare-name reads.

- [ ] **Step 3: Commit**

```bash
git add internal/emitter/emitter_test.go
git commit -m "test(emitter): verify @if reads @signal int variables via existing comparison path"
```

---

## Task 10: Feature-parade fixture — add `@signal int` usage and regenerate golden

**Files:**
- Modify: `testdata/feature_parade/stress.md`
- Modify: `testdata/feature_parade/stress_output.json` (regenerate)

- [ ] **Step 1: Inspect current `stress.md`**

```bash
cat testdata/feature_parade/stress.md
```

Identify a natural insertion point. The goal is to exercise all three `int` write forms (`=`, `+`, `-`) and at least one comparison read in an `@if`. Keep the narrative insertion minimal and consistent with the existing tone.

- [ ] **Step 2: Add `@signal int` usage**

Edit `stress.md` to introduce a counter (suggested name: `stress_count`), writing:

```
@signal int stress_count = 0
@signal int stress_count +2
@signal int stress_count -1
```

in an appropriate scene, and add a conditional read somewhere it makes sense:

```
@if (stress_count >= 2) {
  NARRATOR: The pressure is building.
}
```

Place the writes and the read so the existing `@gate` / `@ending` structure is unaffected.

- [ ] **Step 3: Regenerate the golden output**

First, locate the compile command (check `Makefile` or `cmd/`):

```bash
grep -n "compile\|golden" Makefile 2>/dev/null
ls cmd/
```

Typically:

```bash
go run ./cmd/mss compile testdata/feature_parade/stress.md --assets testdata/feature_parade/mapping.json -o testdata/feature_parade/stress_output.json
```

Adjust path/flags to what the binary actually expects. Then diff-review:

```bash
git diff testdata/feature_parade/stress_output.json | head -80
```

Confirm new entries with `"kind":"int"` appear and look right.

- [ ] **Step 4: Run the golden test**

```bash
go test ./internal/emitter/ -run TestGolden -v
```

Expected: PASS (including the `stress` golden).

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: everything PASS.

- [ ] **Step 6: Commit**

```bash
git add testdata/feature_parade/stress.md testdata/feature_parade/stress_output.json
git commit -m "test(fixtures): add @signal int usage to feature_parade stress fixture"
```

---

## Task 11: Update `MSS-SPEC.md` — §4.7 signal section

**Files:**
- Modify: `MSS-SPEC.md` (around lines 508-560)

- [ ] **Step 1: Replace the `@signal <kind> <event>` subsection**

Read lines 508-560 of `MSS-SPEC.md` first to preserve neighboring prose and markers. Then replace the subsection with:

```markdown
#### `@signal <kind> <...>`

事件/状态信号。`kind` 必写。两种 kind：

| kind | 语法 | 用途 | 写入频率 |
|------|------|------|---------|
| `mark` | `@signal mark <event>` | 持久布尔标记，供 `@if (NAME)` 查询 | **稀有**（关键剧情点） |
| `int` | `@signal int <name> <op> <value>` | 持久整数变量，供 `@if (NAME <cmp> N)` 查询 | **自由**（计数/阈值） |

##### `@signal mark <event>`

持久布尔标记。引擎永久存储，在 `@if (NAME)` 条件中作为布尔值使用。**只用于关键剧情点**——触发隐藏剧情、成就解锁守卫。

`event` 可为裸标识符或双引号字符串。所有 `event` 名称使用 `SCREAMING_SNAKE_CASE` 英文——避免含义歧义，便于跨集搜索和后端处理。

###### Mark 不是"到此一游"标记

**Mark 必须是有人读的。** 不要每集结尾、每个选项出口顺手打 mark。写 mark 的流程是**反过来**的：

1. 先想清楚后面哪里有条件查询要用它——某集的 `@if (X)` 隐藏剧情？某处 `@if (X && Y) { @achievement ... }` 成就解锁？
2. 找到查询点后，反推 X 应该在哪里被打下
3. 只在那个点打 mark

**不需要打 mark 的情况，引擎已经帮你管了：**

- **"这集打完了"** → 引擎从 episode_id 就知道玩家进度
- **"玩家选了 A"** → 选项结果存在引擎的 choice 历史里，gate `@if (A.success)` 直接查
- **"好感度涨了"** → `@affection` 已经改过数值，`@if (affection.easton >= 5)` 直接查数值即可
- **"玩家做了某种性格倾向的行为"** → 用 `@butterfly "..."`，由 LLM 综合判定 influence 条件
- **"某个计数阈值"** → 用 `@signal int counter +1` + `@if (counter >= N)`，比开多个布尔 mark 清晰得多

**真正需要打 mark 的情况：**

- **隐藏剧情分支**：某集的一个关键举动，在后续集的某条对白里被引用。没这 mark 就没那句对白。
- **跨集 arc 成就**：两个或多个 mark 组合作为 `@if` 条件，守卫 `@achievement` 触发。
- **一次性关键抉择**：影响整个主线走向的决定性瞬间，后续多集都会反复查询。

**写-读配对示例：**

```
// EP05 —— write mark
MALIA: One quick step. My heel went straight through his shoe.
@signal mark HIGH_HEEL_EP05

// EP24 —— read mark in hidden-route branch
@if (HIGH_HEEL_EP05) {
  YOU: He glanced at my shoes. He remembered.
}

// 成就解锁 —— 触发和声明合二为一
@if (HIGH_HEEL_EP05) {
  @achievement HIGH_HEEL_WARRIOR {
    name: "Heel as Weapon"
    rarity: rare
    description: "You turned an accessory into a warning. Once is enough to go on record."
  }
}
```

##### `@signal int <name> <op> <value>`

持久整数变量。引擎跨集存储，支持赋值 / 增 / 减，供 `@if` 中裸名比较读取。适合"计数型剧情锁"——统计玩家被拒次数、累计同情行为、N 选 M 触发隐藏剧情等。

**三种写入形态：**

```
@signal int rejections = 0       // 赋值（无条件覆盖，value 可为负）
@signal int rejections +1        // 增
@signal int rejections -2        // 减
```

**语义：**

- **跨集持久**：与 `affection` / `mark` 同等生命周期
- **首次引用视为 0**：`+1` 之前从未赋值 → 引擎从 0 起算，结果为 1
- **`=` 无条件覆盖**：每次执行都赋值；若把 `@signal int x = 0` 放在 ep01 顶部而玩家回放该集，变量会被重置为 0，是作者的责任
- **`+N` / `-N` 中 N 必须非负**：负增量用 `-N` 形态表达，`+0` / `-0` 无意义（用 `= 0`）

**读取（裸名，与引擎数值同语法）：**

```
@if (rejections >= 3) { ... }
@if (rejections == 0) { ... }
@if (rejections >= 3 && affection.easton < 2) { ... }

@gate {
  @if (rejections >= 3): @next main/bad/rejected:01
  @else @if (brave_count >= 3): @next main/route/hidden:01
  @else: @next main:02
}
```

**命名约定：**

- `snake_case` 小写（如 `rejections`、`times_met_easton`、`brave_count`）
- **不可与引擎管理数值保留名冲突**（`san`、`cha`、`hp`、`xp` 等，具体名单由 validator 维护）
- 与 `@signal mark` 的事件名空间分开——mark 用 `SCREAMING_SNAKE_CASE`，int 用 `snake_case`

**与 mark 的使用文化区别：**

| 维度 | `@signal mark` | `@signal int` |
|------|----------------|---------------|
| 写入频率 | 稀有，必须有 reader | 自由，按业务需要 |
| 典型用途 | 一次性关键抉择、隐藏剧情前置、成就守卫 | 计数、累计、阈值型剧情锁 |
| 作者诫命 | "必须有人读，不要顺手打" | 无——计数器的本职就是被频繁修改 |

`@signal int` **不受** "marks 要克制" 的诫命约束；计数器天生就是要频繁写入的。但只给计数器命名有实际含义的名字，不要一个变量半途改语义。
```

- [ ] **Step 2: Update the "引擎管理的数值" callout (around line 506)**

Find the blockquote:

```
> **引擎管理的数值**（如 XP、SAN/HP 等）由引擎内部维护，脚本**不能**修改它们。脚本只能在 `@if` 条件中引用这些数值（如 `@if (san <= 20) { }`），具体名称由引擎定义。
```

Replace with:

```
> **引擎管理的数值**（如 XP、SAN/HP 等）由引擎内部维护，脚本**不能**修改它们。脚本只能在 `@if` 条件中引用这些数值（如 `@if (san <= 20) { }`），具体名称由引擎定义。
>
> **作者自定义的整数变量**由 `@signal int <name> <op> <value>` 声明和修改，跨集持久，与引擎数值共享同一裸名读取命名空间（但不得与保留名冲突）。详见下文 `@signal int` 节。
```

- [ ] **Step 3: Update §4.8 条件类型表 comparison 行**

Find the comparison row in the §4.8 conditions table (around line 625-632) — the row with "comparison" and `affection.<char> <op> <N>` or `<name> <op> <N>`. Ensure the "源语法" column's `<name>` explanation is followed (either inline or in a footnote below the table) by:

> `<name>` 可为引擎管理数值（如 `san`、`cha`）或 `@signal int` 声明的作者变量。裸名读取不区分两类，AST 统一为 `left.kind="value"`。

- [ ] **Step 4: Update JSON output section (§6.3)**

Find the "Achievement 是 inline step..." note (near line 996) and expand the signal bullet:

```
- Achievement 是 inline step，step 本身携带完整元数据：`{"type":"achievement","id":"...","name":"...","rarity":"...","description":"..."}`。
- Signal step 按 kind 分派字段：
  - mark：`{"type":"signal","kind":"mark","event":"..."}`
  - int：`{"type":"signal","kind":"int","name":"...","op":"=|+|-","value":N}`
```

- [ ] **Step 5: Update 附录 B 速查表**

Find the `@signal` row (around line 1232) and replace with **two rows**:

```
| `@signal mark <event>` | 持久布尔标记。`@if (NAME)` 查询。克制使用——必须有 reader |
| `@signal int <name> (=\|+\|-) <int>` | 持久整数变量。`@if (NAME <cmp> N)` 查询。首次引用视为 0；`=` 可赋负值；`+/-` 的 N 必须非负；命名小写 snake_case，不可与引擎保留名冲突 |
```

- [ ] **Step 6: Commit**

```bash
git add MSS-SPEC.md
git commit -m "docs(spec): document @signal int in MSS-SPEC.md"
```

---

## Task 12: Mirror spec update into `skills/mss-scriptwriting/references/MSS-SPEC.md`

**Files:**
- Modify: `skills/mss-scriptwriting/references/MSS-SPEC.md`

- [ ] **Step 1: Copy the updated root MSS-SPEC.md into the skill mirror**

```bash
cp MSS-SPEC.md skills/mss-scriptwriting/references/MSS-SPEC.md
diff -q MSS-SPEC.md skills/mss-scriptwriting/references/MSS-SPEC.md
```

Expected: no diff (files identical).

- [ ] **Step 2: Commit**

```bash
git add skills/mss-scriptwriting/references/MSS-SPEC.md
git commit -m "docs(skill): mirror MSS-SPEC.md for @signal int"
```

---

## Task 13: Update `skills/mss-scriptwriting/SKILL.md`

**Files:**
- Modify: `skills/mss-scriptwriting/SKILL.md` (around lines 300-355)

- [ ] **Step 1: Update the state-changes summary**

Around line 300 where the summary of state-changing directives lives (currently lists `@affection`, `@signal mark`, etc.), add a line for `@signal int`:

```
@signal int rejections +1                              // Persistent integer counter — free to mutate
@signal int rejections = 0                             // Explicit reset / initialization
```

Place it **after** the `@signal mark` line and **before** or **after** `@affection`, whichever flows better with the existing ordering.

- [ ] **Step 2: Update the `@signal <kind>` explanation (around line 312)**

Current text: *"Every `@signal <kind> <event>` — kind is mandatory. Currently only `mark` is implemented..."*

Replace with:

```
**`@signal <kind> <...>` — kind is mandatory.** Two kinds are implemented:

- `@signal mark <event>` — persistent boolean flag. Use sparingly; every mark must have a reader (see below).
- `@signal int <name> <op> <value>` — persistent integer counter. Free to mutate (`= N`, `+N`, `-N`). Read via `@if (name >= N)` comparison.

Achievements are **not** a signal kind — use `@achievement <id> { ... }` for those.
```

- [ ] **Step 3: Add an `@signal int` usage-culture subsection**

After the existing "Every `@signal mark X` must have a reader" paragraph (around line 322), add a new paragraph:

```
**`@signal int` is not mark.** Counters are expected to be written often — `rejections +1` every time the player rejects Easton is exactly the point. The "marks are precious" discipline does NOT apply. Use `@signal int` whenever you need "if player did X at least N times" or "N-of-M threshold" branching. Prefer `@signal int counter +1` + `@if (counter >= N)` over stacking multiple boolean marks.

Guidelines for ints:
- Name in `snake_case` (e.g. `rejections`, `brave_count`, `times_met_easton`).
- Avoid names that look like engine values (`san`, `cha`, `hp`, `xp`) — the validator will reject these.
- `@signal int x = 0` is an unconditional assignment — if placed somewhere the player can revisit, it will reset the counter. That is the author's responsibility; the engine does not protect.
- For first-time reads, the engine treats undeclared variables as 0, so `@signal int x +1` and `@if (x >= 1)` work without any prior `= 0`.
```

- [ ] **Step 4: Update the "don'ts" list (around line 332)**

The list currently warns against overuse of `@signal mark`. Add one item:

```
- ❌ Don't use `@signal mark COUNTER_HIT_3` to track thresholds — that's what `@signal int` is for. Use `@signal int counter +1` then `@if (counter >= 3)`.
```

- [ ] **Step 5: Commit**

```bash
git add skills/mss-scriptwriting/SKILL.md
git commit -m "docs(skill): add @signal int usage guidance"
```

---

## Task 14: Update `skills/mss-scriptwriting/references/directive-table.md`

**Files:**
- Modify: `skills/mss-scriptwriting/references/directive-table.md`

- [ ] **Step 1: Update the directive-table rows**

Find the `@signal <kind> <event>` row (around line 102). Replace it with **two rows**:

```
| `@signal mark <event>` | `@signal mark EP01_COMPLETE` — persistent boolean flag. Use sparingly; every mark must have a later reader. |
| `@signal int <name> <op> <value>` | `@signal int rejections +1` / `= 0` / `-2` — persistent integer counter. Free to mutate. Read via `@if (name >= N)` comparison. |
```

- [ ] **Step 2: Update the prose explanation (around line 156)**

Current: *"`@signal <kind> <event>` — kind is mandatory. Currently only `mark` is implemented..."*

Replace with:

```
`@signal <kind> <...>` — kind is mandatory. Two kinds are implemented:

- `@signal mark <event>` — persistent boolean flag. Engine stores forever; `@if (NAME)` queries the store. Use only for key story points with a reader (a later `@if` branch or achievement trigger).
- `@signal int <name> <op> <value>` — persistent integer counter. Three forms:
  - `= <int>` — unconditional assignment (value may be negative)
  - `+<N>` — increment by N (N is non-negative)
  - `-<N>` — decrement by N (N is non-negative)

  Read via `@if (name <cmp> N)` bare-name comparison. First-time reads default to 0. Names must be `snake_case` and must not collide with engine-reserved names (`san`, `cha`, `hp`, `xp`, etc.).

Achievements are **not** a signal kind — use the dedicated `@achievement <id>` trigger instead.
```

- [ ] **Step 3: Commit**

```bash
git add skills/mss-scriptwriting/references/directive-table.md
git commit -m "docs(skill): document @signal int in directive table"
```

---

## Task 15: Final end-to-end verification

**Files:** (none — verification only)

- [ ] **Step 1: Run the complete test suite**

```bash
go test ./...
```

Expected: PASS across all packages.

- [ ] **Step 2: Compile and run CLI smoke**

```bash
go build ./... && go run ./cmd/mss validate testdata/feature_parade/stress.md --assets testdata/feature_parade/mapping.json
```

Expected: validator reports no errors.

- [ ] **Step 3: Re-compile the stress fixture and confirm byte-identical output**

```bash
go run ./cmd/mss compile testdata/feature_parade/stress.md --assets testdata/feature_parade/mapping.json -o /tmp/stress_out.json
diff testdata/feature_parade/stress_output.json /tmp/stress_out.json
```

Expected: no diff (golden is consistent with current emitter).

- [ ] **Step 4: Negative smoke — reserved-name rejection**

```bash
cat > /tmp/bad_signal_int.md <<'EOF'
@episode main:01 "t" {
  @signal int san = 0
  @ending complete
}
EOF
go run ./cmd/mss validate /tmp/bad_signal_int.md --assets testdata/feature_parade/mapping.json
```

Expected: validator exits non-zero with a `RESERVED_INT_NAME` error message.

- [ ] **Step 5: Push to origin**

```bash
git push origin main
```

(Per user's trunk-based workflow: push often, no PR.)

---

## Self-Review Notes

- **Spec coverage:**
  - §2.1 three write forms → Tasks 4, 5 (parser) + Task 8 (emitter)
  - §2.2 read via `@if` → Task 9 (verification-only; no new code)
  - §3.1 cross-episode persistence → engine responsibility, not in this plan
  - §3.2 first-use defaults to 0 → engine responsibility; documented in Tasks 11/13
  - §3.3 `=` unconditional → parser + author discipline, documented
  - §3.4 reserved-name check → Task 7
  - §3.5 snake_case convention → documented in Tasks 11/13/14; NOT validator-enforced (style, not rule)
  - §4.1 JSON shape → Task 8
  - §4.2 read reuses existing AST → Task 9 (proof test)
  - §5.3 reserved list → Task 7 (matches spec's initial list)
  - §6 documentation updates → Tasks 11–14
  - §7.1/7.2/7.3/7.4 tests → Tasks 4, 5, 6, 7, 8, 9, 10

- **No placeholders**: every step has exact file paths, exact code, exact commands, exact expected output.

- **Type consistency**: `SignalNode.Kind` / `Name` / `Op` / `Value`; `SignalOpAssign/Add/Sub` = `"="` / `"+"` / `"-"`; JSON shape `{type, kind, name, op, value}` matches across Tasks 3, 4, 8.
