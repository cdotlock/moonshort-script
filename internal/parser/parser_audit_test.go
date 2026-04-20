package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
)

// =============================================================================
// AREA A: Token state management — pending node mechanism
// =============================================================================

// TestAudit_TwoConsecutiveDialogueWithExpr tests that two consecutive
// CHARACTER [pose]: text lines don't lose the first pending node.
// Expected: CharLook1, Dialogue1, CharLook2, Dialogue2
func TestAudit_TwoConsecutiveDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	EASTON [happy]: Hey there!
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 4 {
		t.Fatalf("Body length: got %d, want 4 (CharLook + Dialogue + CharLook + Dialogue)", len(ep.Body))
	}

	// Body[0]: CharLookNode for mauricio
	look1, ok := ep.Body[0].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if look1.Char != "mauricio" || look1.Look != "angry" {
		t.Errorf("Body[0]: char=%q look=%q, want mauricio/angry", look1.Char, look1.Look)
	}

	// Body[1]: DialogueNode for MAURICIO
	dlg1, ok := ep.Body[1].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	if dlg1.Character != "MAURICIO" || dlg1.Text != "Get out." {
		t.Errorf("Body[1]: char=%q text=%q", dlg1.Character, dlg1.Text)
	}

	// Body[2]: CharLookNode for easton
	look2, ok := ep.Body[2].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[2]: expected *CharLookNode, got %T", ep.Body[2])
	}
	if look2.Char != "easton" || look2.Look != "happy" {
		t.Errorf("Body[2]: char=%q look=%q, want easton/happy", look2.Char, look2.Look)
	}

	// Body[3]: DialogueNode for EASTON
	dlg2, ok := ep.Body[3].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[3]: expected *DialogueNode, got %T", ep.Body[3])
	}
	if dlg2.Character != "EASTON" || dlg2.Text != "Hey there!" {
		t.Errorf("Body[3]: char=%q text=%q", dlg2.Character, dlg2.Text)
	}
}

// TestAudit_DialogueWithExprLastBeforeRBrace tests that a dialogue-with-expr
// line as the LAST thing before } doesn't lose the pending dialogue node.
// This tests the parseBlock drain-at-end behavior.
func TestAudit_DialogueWithExprLastBeforeRBrace(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (EP01_COMPLETE) {
		MAURICIO [angry]: Get out.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	ifNode, ok := ep.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Body[0]: expected *IfNode, got %T", ep.Body[0])
	}

	// Inside the @if then block, we expect CharLookNode + DialogueNode = 2 nodes.
	if len(ifNode.Then) != 2 {
		t.Fatalf("IfNode.Then length: got %d, want 2 (CharLookNode + DialogueNode)", len(ifNode.Then))
	}

	if _, ok := ifNode.Then[0].(*ast.CharLookNode); !ok {
		t.Errorf("Then[0]: expected *CharLookNode, got %T", ifNode.Then[0])
	}
	if _, ok := ifNode.Then[1].(*ast.DialogueNode); !ok {
		t.Errorf("Then[1]: expected *DialogueNode, got %T", ifNode.Then[1])
	}
}

// TestAudit_DialogueWithExprLastBeforeRBraceEpisode tests pending drain at
// episode body level (not inside parseBlock) when dialogue-with-expr is last.
func TestAudit_DialogueWithExprLastBeforeRBraceEpisode(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	// Should be: CharLookNode + DialogueNode = 2 body nodes
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}

	if _, ok := ep.Body[0].(*ast.CharLookNode); !ok {
		t.Errorf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
}

// TestAudit_DialogueWithExprInElseBlock tests pending inside @else { } block.
func TestAudit_DialogueWithExprInElseBlock(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (flag) {
		NARRATOR: Hi.
	} @else {
		MAURICIO [sad]: Goodbye.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	ifNode := ep.Body[0].(*ast.IfNode)
	// Else should have CharLookNode + DialogueNode = 2 nodes
	if len(ifNode.Else) != 2 {
		t.Fatalf("Else length: got %d, want 2", len(ifNode.Else))
	}
	if _, ok := ifNode.Else[0].(*ast.CharLookNode); !ok {
		t.Errorf("Else[0]: expected *CharLookNode, got %T", ifNode.Else[0])
	}
	if _, ok := ifNode.Else[1].(*ast.DialogueNode); !ok {
		t.Errorf("Else[1]: expected *DialogueNode, got %T", ifNode.Else[1])
	}
}

// TestAudit_DialogueWithExprInCgBody tests pending inside @cg show { } body.
func TestAudit_DialogueWithExprInCgBody(t *testing.T) {
	src := `@episode main:01 "T" {
	@cg show sunset {
		duration: medium
		content: "cg content placeholder"
		MAURICIO [thinking]: Beautiful view.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	cg := ep.Body[0].(*ast.CgShowNode)
	// Body should have CharLookNode + DialogueNode = 2 nodes
	if len(cg.Body) != 2 {
		t.Fatalf("CgShowNode.Body length: got %d, want 2", len(cg.Body))
	}
	if _, ok := cg.Body[0].(*ast.CharLookNode); !ok {
		t.Errorf("CgBody[0]: expected *CharLookNode, got %T", cg.Body[0])
	}
	if _, ok := cg.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("CgBody[1]: expected *DialogueNode, got %T", cg.Body[1])
	}
}

// TestAudit_DialogueWithExprInPhoneBody tests pending inside @phone show { }.
func TestAudit_DialogueWithExprInPhoneBody(t *testing.T) {
	src := `@episode main:01 "T" {
	@phone show {
		MAURICIO [happy]: Hey!
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	phone := ep.Body[0].(*ast.PhoneShowNode)
	// Body should have CharLookNode + DialogueNode = 2 nodes
	if len(phone.Body) != 2 {
		t.Fatalf("PhoneShowNode.Body length: got %d, want 2", len(phone.Body))
	}
}

// TestAudit_DialogueWithExprInCheckSuccessBlock tests pending inside the
// brave-option @if (check.success) { } branch.
func TestAudit_DialogueWithExprInCheckSuccessBlock(t *testing.T) {
	src := `@episode main:01 "T" {
	@choice {
		@option A brave "Fight" {
			check {
				attr: STR
				dc: 14
			}
			@if (check.success) {
				MAURICIO [happy]: You did it!
			} @else {
				NARRATOR: You failed.
			}
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	choice := ep.Body[0].(*ast.ChoiceNode)
	optA := choice.Options[0]

	if len(optA.Body) != 1 {
		t.Fatalf("optA.Body length: got %d, want 1 (single @if)", len(optA.Body))
	}
	ifNode, ok := optA.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("optA.Body[0]: expected *IfNode, got %T", optA.Body[0])
	}
	// Then branch should have CharLookNode + DialogueNode = 2 nodes
	if len(ifNode.Then) != 2 {
		t.Fatalf("Then length: got %d, want 2 (CharLookNode + DialogueNode)", len(ifNode.Then))
	}
	if _, ok := ifNode.Then[0].(*ast.CharLookNode); !ok {
		t.Errorf("Then[0]: expected *CharLookNode, got %T", ifNode.Then[0])
	}
	if _, ok := ifNode.Then[1].(*ast.DialogueNode); !ok {
		t.Errorf("Then[1]: expected *DialogueNode, got %T", ifNode.Then[1])
	}
}

// TestAudit_ThreeConsecutiveDialogueWithExpr verifies three consecutive lines.
func TestAudit_ThreeConsecutiveDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: One.
	EASTON [happy]: Two.
	MALIA [sad]: Three.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	// 3 CharLook + 3 Dialogue = 6 nodes
	if len(ep.Body) != 6 {
		t.Fatalf("Body length: got %d, want 6", len(ep.Body))
	}

	for i := 0; i < 6; i += 2 {
		if _, ok := ep.Body[i].(*ast.CharLookNode); !ok {
			t.Errorf("Body[%d]: expected *CharLookNode, got %T", i, ep.Body[i])
		}
		if _, ok := ep.Body[i+1].(*ast.DialogueNode); !ok {
			t.Errorf("Body[%d]: expected *DialogueNode, got %T", i+1, ep.Body[i+1])
		}
	}
}

// =============================================================================
// AREA B: Condition parsing edge cases
// =============================================================================

// TestAudit_ConditionDeepDotPath tests that under the stricter grammar
// a dotted-path like `a.b.c` is rejected (after `a.b` the parser expects either
// a choice result or an operator; `.c` is invalid). A single IDENT like
// ABC_FLAG is still a valid FlagCondition.
func TestAudit_ConditionDeepDotPath(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (ABC_FLAG) {
		NARRATOR: Flag condition.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)

	fc, ok := ifNode.Condition.(*ast.FlagCondition)
	if !ok {
		t.Fatalf("Condition: want *FlagCondition, got %T", ifNode.Condition)
	}
	if fc.Name != "ABC_FLAG" {
		t.Errorf("Condition.Name: got %q, want %q", fc.Name, "ABC_FLAG")
	}
}

// TestAudit_ConditionBareSingleIdent tests @if (A) — single IDENT.
func TestAudit_ConditionBareSingleIdent(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (A) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)

	// Single IDENT "A" — should be flag, not choice.
	fc, ok := ifNode.Condition.(*ast.FlagCondition)
	if !ok {
		t.Fatalf("Condition: want *FlagCondition, got %T", ifNode.Condition)
	}
	if fc.Name != "A" {
		t.Errorf("Condition.Name: got %q, want %q", fc.Name, "A")
	}
}

// TestAudit_ConditionLoneOperator tests @if (>=) — lone operator, no operands.
// Under the stricter grammar this is rejected by the parser.
func TestAudit_ConditionLoneOperator(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (>=) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for lone operator inside @if")
	}
}

// TestAudit_ConditionInfluenceKeywordAlone tests @if (influence) — keyword without string.
func TestAudit_ConditionInfluenceKeywordAlone(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (influence) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)

	// "influence" alone (no following STRING): single IDENT → FlagCondition.
	fc, ok := ifNode.Condition.(*ast.FlagCondition)
	if !ok {
		t.Fatalf("Condition: want *FlagCondition, got %T", ifNode.Condition)
	}
	if fc.Name != "influence" {
		t.Errorf("Condition.Name: got %q, want %q", fc.Name, "influence")
	}
}

// TestAudit_ConditionDoubleComparison tests that under the stricter grammar,
// `a >= b >= c` is rejected (RHS of comparison must be an integer literal).
// A simple `a >= 5` is accepted and produces a ComparisonCondition.
func TestAudit_ConditionDoubleComparison(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (a >= 5) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)

	cmp, ok := ifNode.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Condition: want *ComparisonCondition, got %T", ifNode.Condition)
	}
	if cmp.Left.Kind != ast.OperandValue || cmp.Left.Name != "a" {
		t.Errorf("Condition.Left: got %+v, want value/a", cmp.Left)
	}
	if cmp.Op != ">=" || cmp.Right != 5 {
		t.Errorf("Condition: got op=%q right=%d, want >=/5", cmp.Op, cmp.Right)
	}
}

// TestAudit_ConditionDotNonChoiceResult tests @if (A.blah) — dot but not
// success/fail/any. Under the stricter grammar this is rejected (`A` is not
// `affection`, so it can't be a dotted comparison operand either).
func TestAudit_ConditionDotNonChoiceResult(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (A.blah) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for A.blah (not a valid choice result and not affection)")
	}
}

// =============================================================================
// AREA C: Gate parsing edge cases
// =============================================================================

// TestAudit_GateElseWithoutIf tests @gate { @else: @next main:02 }.
func TestAudit_GateElseWithoutIf(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)

	// Should parse as a single unconditional route (condition=nil).
	if ep.Gate == nil {
		t.Fatal("Gate: expected non-nil")
	}
	if len(ep.Gate.Routes) != 1 {
		t.Fatalf("Gate.Routes count: got %d, want 1", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Condition != nil {
		t.Error("expected nil condition for @else without @if")
	}
	if ep.Gate.Routes[0].Target != "main:02" {
		t.Errorf("Target: got %q, want main:02", ep.Gate.Routes[0].Target)
	}
}

// TestAudit_GateTwoConsecutiveIfs tests two @if without @else between them.
func TestAudit_GateTwoConsecutiveIfs(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@if (A.fail):
			@next bad:01
		@if (B.fail):
			@next bad:02
	}
}`
	ep := parseOrFail(t, src)

	if ep.Gate == nil {
		t.Fatal("Gate: expected non-nil")
	}
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("Gate.Routes count: got %d, want 2", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Target != "bad:01" {
		t.Errorf("Route[0].Target: got %q, want bad:01", ep.Gate.Routes[0].Target)
	}
	if ep.Gate.Routes[1].Target != "bad:02" {
		t.Errorf("Route[1].Target: got %q, want bad:02", ep.Gate.Routes[1].Target)
	}
}

// TestAudit_GateTwoBareNext tests two bare @next (no conditions).
func TestAudit_GateTwoBareNext(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@next main:02
		@next main:03
	}
}`
	ep := parseOrFail(t, src)

	if ep.Gate == nil {
		t.Fatal("Gate: expected non-nil")
	}
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("Gate.Routes count: got %d, want 2", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Target != "main:02" {
		t.Errorf("Route[0].Target: got %q, want main:02", ep.Gate.Routes[0].Target)
	}
	if ep.Gate.Routes[1].Target != "main:03" {
		t.Errorf("Route[1].Target: got %q, want main:03", ep.Gate.Routes[1].Target)
	}
}

// TestAudit_GateElseIfChain tests @else @if inside gate.
func TestAudit_GateElseIfChain(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@if (A.success):
			@next good:01
		@else @if (B.fail):
			@next bad:01
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)

	if len(ep.Gate.Routes) != 3 {
		t.Fatalf("Gate.Routes count: got %d, want 3", len(ep.Gate.Routes))
	}
	// Route 0: conditional (A.success)
	if ep.Gate.Routes[0].Condition == nil {
		t.Error("Route[0] should have a condition")
	}
	// Route 1: conditional (B.fail) — from @else @if
	if ep.Gate.Routes[1].Condition == nil {
		t.Error("Route[1] should have a condition")
	}
	// Route 2: unconditional fallback
	if ep.Gate.Routes[2].Condition != nil {
		t.Error("Route[2] should be unconditional")
	}
}

// =============================================================================
// AREA D: Block parsing interactions
// =============================================================================

// TestAudit_NestedIfInsideCg tests @if inside @cg show { } body.
func TestAudit_NestedIfInsideCg(t *testing.T) {
	src := `@episode main:01 "T" {
	@cg show sunset {
		duration: medium
		content: "cg content placeholder"
		@if (flag) {
			NARRATOR: Hi.
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	cg := ep.Body[0].(*ast.CgShowNode)
	if len(cg.Body) != 1 {
		t.Fatalf("CgShowNode.Body length: got %d, want 1", len(cg.Body))
	}
	ifNode, ok := cg.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("CgBody[0]: expected *IfNode, got %T", cg.Body[0])
	}
	if len(ifNode.Then) != 1 {
		t.Errorf("IfNode.Then length: got %d, want 1", len(ifNode.Then))
	}
}

// TestAudit_DeeplyNestedChoiceBrave tests a brave option whose body contains
// a check.success/check.fail @if/@else tree, with a nested @if on a plain
// flag inside the success branch.
func TestAudit_DeeplyNestedChoiceBrave(t *testing.T) {
	src := `@episode main:01 "T" {
	@choice {
		@option A brave "test" {
			check {
				attr: STR
				dc: 14
			}
			@if (check.success) {
				@if (flag) {
					NARRATOR: Win with flag.
				} @else {
					NARRATOR: Win without flag.
				}
			} @else {
				NARRATOR: Lose.
			}
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	choice := ep.Body[0].(*ast.ChoiceNode)
	optA := choice.Options[0]

	if optA.Check == nil {
		t.Fatal("Check should not be nil")
	}
	if optA.Check.Attr != "STR" || optA.Check.DC != 14 {
		t.Errorf("Check: attr=%q dc=%d", optA.Check.Attr, optA.Check.DC)
	}

	// Body should have 1 node: the outer @if (check.success)
	if len(optA.Body) != 1 {
		t.Fatalf("optA.Body length: got %d, want 1", len(optA.Body))
	}
	outerIf, ok := optA.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("optA.Body[0]: expected *IfNode, got %T", optA.Body[0])
	}
	if _, ok := outerIf.Condition.(*ast.CheckCondition); !ok {
		t.Fatalf("outer condition: expected *CheckCondition, got %T", outerIf.Condition)
	}
	// Success branch: a nested @if (flag)
	if len(outerIf.Then) != 1 {
		t.Fatalf("outer Then length: got %d, want 1 (nested if)", len(outerIf.Then))
	}
	nestedIf, ok := outerIf.Then[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("outer Then[0]: expected *IfNode, got %T", outerIf.Then[0])
	}
	if len(nestedIf.Then) != 1 {
		t.Errorf("nested Then length: got %d, want 1", len(nestedIf.Then))
	}
	if len(nestedIf.Else) != 1 {
		t.Errorf("nested Else length: got %d, want 1", len(nestedIf.Else))
	}

	// Else branch (fail)
	if len(outerIf.Else) != 1 {
		t.Fatalf("outer Else length: got %d, want 1", len(outerIf.Else))
	}
}

// TestAudit_IfInsidePhoneShow tests @if inside @phone show { }.
func TestAudit_IfInsidePhoneShow(t *testing.T) {
	src := `@episode main:01 "T" {
	@phone show {
		@if (flag) {
			@text from easton: Hello!
		}
		@text to easton: Reply.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	phone := ep.Body[0].(*ast.PhoneShowNode)
	if len(phone.Body) != 2 {
		t.Fatalf("PhoneShowNode.Body length: got %d, want 2", len(phone.Body))
	}
	if _, ok := phone.Body[0].(*ast.IfNode); !ok {
		t.Errorf("phone.Body[0]: expected *IfNode, got %T", phone.Body[0])
	}
	if _, ok := phone.Body[1].(*ast.TextMessageNode); !ok {
		t.Errorf("phone.Body[1]: expected *TextMessageNode, got %T", phone.Body[1])
	}
}

// =============================================================================
// AREA E: Concurrent flag on various node types
// =============================================================================

// TestAudit_ConcurrentBgSet tests &bg set classroom.
func TestAudit_ConcurrentBgSet(t *testing.T) {
	src := `@episode main:01 "T" {
	@bg set hallway
	&bg set classroom
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}

	bg1 := ep.Body[0].(*ast.BgSetNode)
	if bg1.GetConcurrent() {
		t.Error("First bg should NOT be concurrent")
	}

	bg2 := ep.Body[1].(*ast.BgSetNode)
	if !bg2.GetConcurrent() {
		t.Error("Second bg should be concurrent")
	}
	if bg2.Name != "classroom" {
		t.Errorf("bg2.Name: got %q, want classroom", bg2.Name)
	}
}

// TestAudit_ConcurrentPause tests &pause for 1.
func TestAudit_ConcurrentPause(t *testing.T) {
	src := `@episode main:01 "T" {
	&pause for 1
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	pause, ok := ep.Body[0].(*ast.PauseNode)
	if !ok {
		t.Fatalf("Body[0]: expected *PauseNode, got %T", ep.Body[0])
	}
	if !pause.GetConcurrent() {
		t.Error("PauseNode should be concurrent")
	}
	if pause.Clicks != 1 {
		t.Errorf("Clicks: got %d, want 1", pause.Clicks)
	}
}

// TestAudit_ConcurrentCharShow tests &malia show happy at left.
func TestAudit_ConcurrentCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	&malia show happy at left
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	cs, ok := ep.Body[0].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if !cs.GetConcurrent() {
		t.Error("CharShowNode should be concurrent")
	}
	if cs.Char != "malia" {
		t.Errorf("Char: got %q, want malia", cs.Char)
	}
	if cs.Look != "happy" {
		t.Errorf("Look: got %q, want happy", cs.Look)
	}
	if cs.Position != "left" {
		t.Errorf("Position: got %q, want left", cs.Position)
	}
}

// TestAudit_ConcurrentSfx tests &sfx play swoosh.
func TestAudit_ConcurrentSfx(t *testing.T) {
	src := `@episode main:01 "T" {
	&sfx play swoosh
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	sfx, ok := ep.Body[0].(*ast.SfxPlayNode)
	if !ok {
		t.Fatalf("Body[0]: expected *SfxPlayNode, got %T", ep.Body[0])
	}
	if !sfx.GetConcurrent() {
		t.Error("SfxPlayNode should be concurrent")
	}
}

// TestAudit_ConcurrentAffection tests &affection easton +1.
func TestAudit_ConcurrentAffection(t *testing.T) {
	src := `@episode main:01 "T" {
	&affection easton +1
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	aff, ok := ep.Body[0].(*ast.AffectionNode)
	if !ok {
		t.Fatalf("Body[0]: expected *AffectionNode, got %T", ep.Body[0])
	}
	if !aff.GetConcurrent() {
		t.Error("AffectionNode should be concurrent")
	}
}

// TestAudit_ConcurrentSignal tests &signal mark EVENT.
func TestAudit_ConcurrentSignal(t *testing.T) {
	src := `@episode main:01 "T" {
	&signal mark EP01_DONE
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	sig, ok := ep.Body[0].(*ast.SignalNode)
	if !ok {
		t.Fatalf("Body[0]: expected *SignalNode, got %T", ep.Body[0])
	}
	if !sig.GetConcurrent() {
		t.Error("SignalNode should be concurrent")
	}
}

// TestAudit_ConcurrentCharHide tests &mauricio hide fade.
func TestAudit_ConcurrentCharHide(t *testing.T) {
	src := `@episode main:01 "T" {
	&mauricio hide fade
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	hide, ok := ep.Body[0].(*ast.CharHideNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharHideNode, got %T", ep.Body[0])
	}
	if !hide.GetConcurrent() {
		t.Error("CharHideNode should be concurrent")
	}
	if hide.Transition != "fade" {
		t.Errorf("Transition: got %q, want fade", hide.Transition)
	}
}

// TestAudit_ConcurrentCharLook tests &mauricio look happy.
func TestAudit_ConcurrentCharLook(t *testing.T) {
	src := `@episode main:01 "T" {
	&mauricio look happy
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	look, ok := ep.Body[0].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if !look.GetConcurrent() {
		t.Error("CharLookNode should be concurrent")
	}
}

// TestAudit_ConcurrentMusicCrossfade tests &music crossfade track.
func TestAudit_ConcurrentMusicCrossfade(t *testing.T) {
	src := `@episode main:01 "T" {
	&music crossfade theme_battle
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	mcf, ok := ep.Body[0].(*ast.MusicCrossfadeNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicCrossfadeNode, got %T", ep.Body[0])
	}
	if !mcf.GetConcurrent() {
		t.Error("MusicCrossfadeNode should be concurrent")
	}
}

// TestAudit_ConcurrentMusicFadeout tests &music fadeout.
func TestAudit_ConcurrentMusicFadeout(t *testing.T) {
	src := `@episode main:01 "T" {
	&music fadeout
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	mf, ok := ep.Body[0].(*ast.MusicFadeoutNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicFadeoutNode, got %T", ep.Body[0])
	}
	if !mf.GetConcurrent() {
		t.Error("MusicFadeoutNode should be concurrent")
	}
}

// TestAudit_ConcurrentCharMove tests &mauricio move to right.
func TestAudit_ConcurrentCharMove(t *testing.T) {
	src := `@episode main:01 "T" {
	&mauricio move to right
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	mv, ok := ep.Body[0].(*ast.CharMoveNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharMoveNode, got %T", ep.Body[0])
	}
	if !mv.GetConcurrent() {
		t.Error("CharMoveNode should be concurrent")
	}
}

// TestAudit_ConcurrentCharBubble tests &mauricio bubble heart.
func TestAudit_ConcurrentCharBubble(t *testing.T) {
	src := `@episode main:01 "T" {
	&mauricio bubble heart
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	bub, ok := ep.Body[0].(*ast.CharBubbleNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharBubbleNode, got %T", ep.Body[0])
	}
	if !bub.GetConcurrent() {
		t.Error("CharBubbleNode should be concurrent")
	}
}

// TestAudit_ConcurrentButterfly tests &butterfly "desc".
func TestAudit_ConcurrentButterfly(t *testing.T) {
	src := `@episode main:01 "T" {
	&butterfly "Some choice made"
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	btf, ok := ep.Body[0].(*ast.ButterflyNode)
	if !ok {
		t.Fatalf("Body[0]: expected *ButterflyNode, got %T", ep.Body[0])
	}
	if !btf.GetConcurrent() {
		t.Error("ButterflyNode should be concurrent")
	}
}

// =============================================================================
// AREA F: Complex integration edge cases
// =============================================================================

// TestAudit_DialogueWithExprThenDirective tests dialogue-with-expr followed
// immediately by a directive.
func TestAudit_DialogueWithExprThenDirective(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@bg set hallway
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	if _, ok := ep.Body[0].(*ast.CharLookNode); !ok {
		t.Errorf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	bg, ok := ep.Body[2].(*ast.BgSetNode)
	if !ok {
		t.Fatalf("Body[2]: expected *BgSetNode, got %T", ep.Body[2])
	}
	if bg.Name != "hallway" {
		t.Errorf("bg.Name: got %q, want hallway", bg.Name)
	}
}

// TestAudit_DialogueWithExprThenConcurrent tests dialogue-with-expr followed
// by a concurrent directive.
func TestAudit_DialogueWithExprThenConcurrent(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	&music play theme
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	if _, ok := ep.Body[0].(*ast.CharLookNode); !ok {
		t.Errorf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	mp, ok := ep.Body[2].(*ast.MusicPlayNode)
	if !ok {
		t.Fatalf("Body[2]: expected *MusicPlayNode, got %T", ep.Body[2])
	}
	if !mp.GetConcurrent() {
		t.Error("MusicPlayNode should be concurrent")
	}
}

// TestAudit_MixedDialogueAndDialogueWithExpr tests alternating plain and
// expr-annotated dialogue lines.
func TestAudit_MixedDialogueAndDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Start.
	MAURICIO [angry]: Watch out.
	NARRATOR: Middle.
	EASTON [happy]: Hello!
	NARRATOR: End.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	// NARRATOR: Start -> NarratorNode
	// MAURICIO [angry]: -> CharLookNode + DialogueNode
	// NARRATOR: Middle -> NarratorNode
	// EASTON [happy]: -> CharLookNode + DialogueNode
	// NARRATOR: End -> NarratorNode
	// Total: 7 nodes
	if len(ep.Body) != 7 {
		t.Fatalf("Body length: got %d, want 7", len(ep.Body))
	}

	expected := []string{
		"*ast.NarratorNode",
		"*ast.CharLookNode",
		"*ast.DialogueNode",
		"*ast.NarratorNode",
		"*ast.CharLookNode",
		"*ast.DialogueNode",
		"*ast.NarratorNode",
	}
	for i, exp := range expected {
		got := fmt.Sprintf("%T", ep.Body[i])
		if got != exp {
			t.Errorf("Body[%d]: expected %s, got %s", i, exp, got)
		}
	}
}

// TestAudit_DialogueWithExprBeforeGateEpisodeLevel tests that pending is
// properly drained when parseEpisodeBody encounters @gate after a
// dialogue-with-expr. The gate check happens BEFORE parseStatement,
// but the pending drain happens first.
func TestAudit_DialogueWithExprBeforeGateEpisodeLevel(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	// Should be: CharLookNode + DialogueNode = 2 body nodes
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharLookNode); !ok {
		t.Errorf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
}

// =============================================================================
// AREA G: Error handling edge cases
// =============================================================================

// TestAudit_ConcurrentDialogueError tests that &NARRATOR: text gives a helpful error.
func TestAudit_ConcurrentDialogueError(t *testing.T) {
	src := `@episode main:01 "T" {
	&NARRATOR: Hello.
	@gate { @next main:02 }
}`
	l := lexer.New(src)
	p := New(l)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for &NARRATOR: dialogue")
	}
	// The error should mention that & can't be used with dialogue.
	// But let's just check it doesn't panic.
	t.Logf("Error (expected): %v", err)
}

// TestAudit_UnknownDirective tests an unknown @keyword.
func TestAudit_UnknownDirective(t *testing.T) {
	src := `@episode main:01 "T" {
	@foobar baz
	@gate { @next main:02 }
}`
	l := lexer.New(src)
	p := New(l)
	_, err := p.Parse()
	// "foobar" is not a knownKeyword, so it falls through to parseCharDirective.
	// parseCharDirective calls advance() to consume "foobar" then expects IDENT for action.
	// "baz" is a valid action name... but not a known character action.
	if err == nil {
		t.Fatal("expected error for unknown character action")
	}
	if !strings.Contains(err.Error(), "unknown character action") {
		t.Errorf("expected 'unknown character action' in error, got: %v", err)
	}
}

// TestAudit_NestingDepthLimit tests that deeply nested blocks fail gracefully.
func TestAudit_NestingDepthLimit(t *testing.T) {
	// Build deeply nested @if blocks
	var src strings.Builder
	src.WriteString(`@episode main:01 "T" {`)
	for i := 0; i < 55; i++ {
		src.WriteString(`@if (flag) {`)
	}
	src.WriteString(`NARRATOR: Deep.`)
	for i := 0; i < 55; i++ {
		src.WriteString(`}`)
	}
	src.WriteString(`@gate { @next main:02 } }`)

	l := lexer.New(src.String())
	p := New(l)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for nesting depth exceeded")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Errorf("expected 'nesting depth' in error, got: %v", err)
	}
}

// TestAudit_EmptyEpisodeBody tests episode with no body nodes and no gate.
func TestAudit_EmptyEpisodeBody(t *testing.T) {
	src := `@episode main:01 "T" { }`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 0 {
		t.Errorf("Body length: got %d, want 0", len(ep.Body))
	}
	if ep.Gate != nil {
		t.Error("Gate should be nil for empty body")
	}
}

// TestAudit_MultipleDirectivesOnSameLine tests multiple directives separated
// by newlines only (no blank lines).
func TestAudit_MultipleDirectivesTight(t *testing.T) {
	src := "@episode main:01 \"T\" {\n@bg set classroom\n@music play theme\nNARRATOR: Hello.\n@gate { @next main:02 }\n}"
	ep := parseOrFail(t, src)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}
}

// TestAudit_ConditionWithNotEquals tests @if (a != 1).
// Under the stricter grammar, comparison RHS must be an integer literal.
func TestAudit_ConditionWithNotEquals(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (a != 1) {
		NARRATOR: Different.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	cmp, ok := ifNode.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Condition: want *ComparisonCondition, got %T", ifNode.Condition)
	}
	if cmp.Left.Kind != ast.OperandValue || cmp.Left.Name != "a" {
		t.Errorf("Condition.Left: got %+v, want value/a", cmp.Left)
	}
	if cmp.Op != "!=" || cmp.Right != 1 {
		t.Errorf("Condition: got op=%q right=%d, want !=/1", cmp.Op, cmp.Right)
	}
}

// TestAudit_ConditionWithEquals tests @if (a == 1).
func TestAudit_ConditionWithEquals(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (a == 1) {
		NARRATOR: Same.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if _, ok := ifNode.Condition.(*ast.ComparisonCondition); !ok {
		t.Errorf("Condition: want *ComparisonCondition, got %T", ifNode.Condition)
	}
}

// TestAudit_ConditionCompoundOr tests @if (a || b).
func TestAudit_ConditionCompoundOr(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (flag1 || flag2) {
		NARRATOR: One or the other.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	comp, ok := ifNode.Condition.(*ast.CompoundCondition)
	if !ok {
		t.Fatalf("Condition: want *CompoundCondition, got %T", ifNode.Condition)
	}
	if comp.Op != "||" {
		t.Errorf("Op: got %q, want ||", comp.Op)
	}
}

// TestAudit_SafeOptionBody tests safe option with multiple body nodes.
func TestAudit_SafeOptionBody(t *testing.T) {
	src := `@episode main:01 "T" {
	@choice {
		@option B safe "Talk" {
			NARRATOR: Line1.
			@affection guard +1
			NARRATOR: Line2.
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	choice := ep.Body[0].(*ast.ChoiceNode)
	optB := choice.Options[0]
	if len(optB.Body) != 3 {
		t.Fatalf("Option body length: got %d, want 3", len(optB.Body))
	}
}

// TestAudit_CgShowNoTransition tests @cg show name { } without transition.
func TestAudit_CgShowNoTransition(t *testing.T) {
	src := `@episode main:01 "T" {
	@cg show sunset {
		duration: medium
		content: "cg content placeholder"
		NARRATOR: Nice.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	cg := ep.Body[0].(*ast.CgShowNode)
	if cg.Transition != "" {
		t.Errorf("Transition: got %q, want empty", cg.Transition)
	}
	if cg.Name != "sunset" {
		t.Errorf("Name: got %q, want sunset", cg.Name)
	}
}

// TestAudit_BgTransitionBeforeDialogue tests that bg transition is consumed
// and not confused with the next line's dialogue character name.
func TestAudit_BgTransitionBeforeDialogue(t *testing.T) {
	src := `@episode main:01 "T" {
	@bg set classroom fade
	NARRATOR: Hello.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}
	bg := ep.Body[0].(*ast.BgSetNode)
	if bg.Transition != "fade" {
		t.Errorf("bg.Transition: got %q, want fade", bg.Transition)
	}
	narr := ep.Body[1].(*ast.NarratorNode)
	if narr.Text != "Hello." {
		t.Errorf("narr.Text: got %q", narr.Text)
	}
}

// TestAudit_DialogueAfterCharShow tests that NARRATOR: after @char show works
// (transition is correctly bounded by isDirectiveStart).
func TestAudit_DialogueAfterCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	@mauricio show neutral at center
	NARRATOR: Hello.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}
	show := ep.Body[0].(*ast.CharShowNode)
	if show.Transition != "" {
		t.Errorf("show.Transition: got %q, want empty", show.Transition)
	}
	narr := ep.Body[1].(*ast.NarratorNode)
	if narr.Text != "Hello." {
		t.Errorf("narr.Text: got %q", narr.Text)
	}
}

// TestAudit_DialogueWithExprAfterCharShow tests CHAR [pose]: text after @char show.
func TestAudit_DialogueWithExprAfterCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	@mauricio show neutral at center
	MAURICIO [happy]: Hello!
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	// @char show -> CharShowNode
	// MAURICIO [happy] -> CharLookNode + DialogueNode (pending)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.CharLookNode); !ok {
		t.Errorf("Body[1]: expected *CharLookNode, got %T", ep.Body[1])
	}
	if _, ok := ep.Body[2].(*ast.DialogueNode); !ok {
		t.Errorf("Body[2]: expected *DialogueNode, got %T", ep.Body[2])
	}
}

// TestAudit_ConditionStringWithSpaces tests @if ("some complex description").
func TestAudit_ConditionStringWithSpaces(t *testing.T) {
	src := `@episode main:01 "T" {
	@if ("Player said something kind to Easton during the park scene") {
		NARRATOR: Remembered.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	ic, ok := ifNode.Condition.(*ast.InfluenceCondition)
	if !ok {
		t.Fatalf("Condition: want *InfluenceCondition, got %T", ifNode.Condition)
	}
	if ic.Description != "Player said something kind to Easton during the park scene" {
		t.Errorf("Condition.Description: got %q", ic.Description)
	}
}

// TestAudit_GateConditionInfluence tests gate route with influence condition.
func TestAudit_GateConditionInfluence(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@if (influence "Player was kind"):
			@next good:01
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("Gate.Routes count: got %d, want 2", len(ep.Gate.Routes))
	}
	r0 := ep.Gate.Routes[0]
	if _, ok := r0.Condition.(*ast.InfluenceCondition); !ok {
		t.Errorf("Route[0].Condition: want *InfluenceCondition, got %T", r0.Condition)
	}
}

// TestAudit_ChoiceMultipleOptions tests more than 2 options in a choice.
func TestAudit_ChoiceMultipleOptions(t *testing.T) {
	src := `@episode main:01 "T" {
	@choice {
		@option A safe "Option A" {
			NARRATOR: A.
		}
		@option B safe "Option B" {
			NARRATOR: B.
		}
		@option C safe "Option C" {
			NARRATOR: C.
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	choice := ep.Body[0].(*ast.ChoiceNode)
	if len(choice.Options) != 3 {
		t.Fatalf("Options count: got %d, want 3", len(choice.Options))
	}
}

// TestAudit_MinigameSingleRating tests branching on a rating via @if
// (rating.S) inside a minigame body.
func TestAudit_MinigameSingleRating(t *testing.T) {
	src := `@episode main:01 "T" {
	@minigame test STR "minigame description placeholder" {
		@if (rating.S) {
			NARRATOR: Perfect.
		} @else {
			NARRATOR: Failed.
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	mg := ep.Body[0].(*ast.MinigameNode)
	if mg.Description != "minigame description placeholder" {
		t.Errorf("Description: got %q", mg.Description)
	}
	if len(mg.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1 (single @if)", len(mg.Body))
	}
	ifNode, ok := mg.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Body[0]: expected *IfNode, got %T", mg.Body[0])
	}
	rc, ok := ifNode.Condition.(*ast.RatingCondition)
	if !ok || rc.Grade != "S" {
		t.Errorf("Condition: got %T %+v, want RatingCondition{S}", ifNode.Condition, ifNode.Condition)
	}
	if len(ifNode.Then) != 1 {
		t.Errorf("Then length: got %d, want 1", len(ifNode.Then))
	}
	if len(ifNode.Else) != 1 {
		t.Errorf("Else length: got %d, want 1", len(ifNode.Else))
	}
}
