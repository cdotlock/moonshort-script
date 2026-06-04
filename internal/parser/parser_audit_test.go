package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// =============================================================================
// AREA A: Token state management — pending node mechanism
// =============================================================================

// TestAudit_TwoConsecutiveDialogueWithExpr tests that two consecutive
// CHARACTER [pose]: text lines don't lose the first pending node.
// Expected: CharShow1, Dialogue1, CharShow2, Dialogue2.
func TestAudit_TwoConsecutiveDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	EASTON [happy]: Hey there!
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 4 {
		t.Fatalf("Body length: got %d, want 4 (CharShow+Dialogue twice)", len(ep.Body))
	}

	show1, ok := ep.Body[0].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if show1.Char != "mauricio" || show1.Look != "angry" {
		t.Errorf("Body[0]: char=%q look=%q, want mauricio/angry", show1.Char, show1.Look)
	}

	dlg1, ok := ep.Body[1].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	if dlg1.Character != "MAURICIO" || dlg1.Text != "Get out." {
		t.Errorf("Body[1]: char=%q text=%q", dlg1.Character, dlg1.Text)
	}

	show2, ok := ep.Body[2].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[2]: expected *CharShowNode, got %T", ep.Body[2])
	}
	if show2.Char != "easton" || show2.Look != "happy" {
		t.Errorf("Body[2]: char=%q look=%q, want easton/happy", show2.Char, show2.Look)
	}

	dlg2, ok := ep.Body[3].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[3]: expected *DialogueNode, got %T", ep.Body[3])
	}
	if dlg2.Character != "EASTON" || dlg2.Text != "Hey there!" {
		t.Errorf("Body[3]: char=%q text=%q", dlg2.Character, dlg2.Text)
	}
}

// TestAudit_DialogueWithExprLastBeforeRBrace tests that a dialogue-with-expr
// line as the last thing before `}` doesn't lose the pending dialogue node.
func TestAudit_DialogueWithExprLastBeforeRBrace(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (EP01_COMPLETE) {
		MAURICIO [angry]: Get out.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if len(ifNode.Then) != 2 {
		t.Fatalf("Then length: got %d, want 2 (CharShow + Dialogue)", len(ifNode.Then))
	}
	if _, ok := ifNode.Then[0].(*ast.CharShowNode); !ok {
		t.Errorf("Then[0]: expected *CharShowNode, got %T", ifNode.Then[0])
	}
	if _, ok := ifNode.Then[1].(*ast.DialogueNode); !ok {
		t.Errorf("Then[1]: expected *DialogueNode, got %T", ifNode.Then[1])
	}
}

// TestAudit_DialogueWithExprLastBeforeRBraceEpisode tests pending drain at
// the episode-body level when dialogue-with-expr is the last thing before
// the closing brace before @gate.
func TestAudit_DialogueWithExprLastBeforeRBraceEpisode(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
}

// TestAudit_DialogueWithExprInElseBlock tests pending drain inside an
// @else { } block.
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
	if len(ifNode.Else) != 2 {
		t.Fatalf("Else length: got %d, want 2", len(ifNode.Else))
	}
	if _, ok := ifNode.Else[0].(*ast.CharShowNode); !ok {
		t.Errorf("Else[0]: expected *CharShowNode, got %T", ifNode.Else[0])
	}
	if _, ok := ifNode.Else[1].(*ast.DialogueNode); !ok {
		t.Errorf("Else[1]: expected *DialogueNode, got %T", ifNode.Else[1])
	}
}

// TestAudit_DialogueWithExprInCheckSuccessBlock tests pending inside the
// brave-option @if (check.success) branch.
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
		t.Fatalf("optA.Body length: got %d, want 1", len(optA.Body))
	}
	ifNode := optA.Body[0].(*ast.IfNode)
	if len(ifNode.Then) != 2 {
		t.Fatalf("Then length: got %d, want 2", len(ifNode.Then))
	}
	if _, ok := ifNode.Then[0].(*ast.CharShowNode); !ok {
		t.Errorf("Then[0]: expected *CharShowNode, got %T", ifNode.Then[0])
	}
	if _, ok := ifNode.Then[1].(*ast.DialogueNode); !ok {
		t.Errorf("Then[1]: expected *DialogueNode, got %T", ifNode.Then[1])
	}
}

// TestAudit_ThreeConsecutiveDialogueWithExpr verifies three consecutive
// dialogue-with-expr lines.
func TestAudit_ThreeConsecutiveDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: One.
	EASTON [happy]: Two.
	MALIA [sad]: Three.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	// 3 CharShow + 3 Dialogue = 6 nodes
	if len(ep.Body) != 6 {
		t.Fatalf("Body length: got %d, want 6", len(ep.Body))
	}
	for i := 0; i < 6; i += 2 {
		if _, ok := ep.Body[i].(*ast.CharShowNode); !ok {
			t.Errorf("Body[%d]: expected *CharShowNode, got %T", i, ep.Body[i])
		}
		if _, ok := ep.Body[i+1].(*ast.DialogueNode); !ok {
			t.Errorf("Body[%d]: expected *DialogueNode, got %T", i+1, ep.Body[i+1])
		}
	}
}

// =============================================================================
// AREA B: Condition parsing edge cases
// =============================================================================

// TestAudit_ConditionBareSingleIdent tests `@if (A)` — single IDENT → flag.
func TestAudit_ConditionBareSingleIdent(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (A) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	fc, ok := ifNode.Condition.(*ast.FlagCondition)
	if !ok {
		t.Fatalf("Condition: want *FlagCondition, got %T", ifNode.Condition)
	}
	if fc.Name != "A" {
		t.Errorf("Name: got %q", fc.Name)
	}
}

// TestAudit_ConditionLoneOperator tests that an operator-only condition is
// a parse error.
func TestAudit_ConditionLoneOperator(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (>=) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	_, err := parseSource(src)
	if err == nil {
		t.Fatal("expected parse error for lone operator")
	}
}

// TestAudit_ConditionDotNonChoiceResult tests that `A.blah` (not a valid
// choice result and not `affection`) is a parse error.
func TestAudit_ConditionDotNonChoiceResult(t *testing.T) {
	src := `@episode main:01 "T" {
	@if (A.blah) {
		NARRATOR: Hi.
	}
	@gate { @next main:02 }
}`
	_, err := parseSource(src)
	if err == nil {
		t.Fatal("expected parse error for A.blah")
	}
}

// =============================================================================
// AREA C: Gate parsing edge cases
// =============================================================================

// TestAudit_GateElseWithoutIf tests `@gate { @else: @next ... }`. Standalone
// @else parses as an unconditional fallback route.
func TestAudit_GateElseWithoutIf(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: Hi.
	@gate {
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)
	if len(ep.Gate.Routes) != 1 {
		t.Fatalf("Routes: got %d", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Condition != nil {
		t.Error("expected nil condition")
	}
	next, ok := ep.Gate.Routes[0].Leaf.(*ast.NextLeaf)
	if !ok || next.Target != "main:02" {
		t.Errorf("Leaf: got %+v", ep.Gate.Routes[0].Leaf)
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
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("Routes: got %d", len(ep.Gate.Routes))
	}
	next0 := ep.Gate.Routes[0].Leaf.(*ast.NextLeaf)
	if next0.Target != "bad:01" {
		t.Errorf("Route[0]: got %q", next0.Target)
	}
	next1 := ep.Gate.Routes[1].Leaf.(*ast.NextLeaf)
	if next1.Target != "bad:02" {
		t.Errorf("Route[1]: got %q", next1.Target)
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
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("Routes: got %d", len(ep.Gate.Routes))
	}
	if next := ep.Gate.Routes[0].Leaf.(*ast.NextLeaf); next.Target != "main:02" {
		t.Errorf("Route[0]: got %q", next.Target)
	}
	if next := ep.Gate.Routes[1].Leaf.(*ast.NextLeaf); next.Target != "main:03" {
		t.Errorf("Route[1]: got %q", next.Target)
	}
}

// TestAudit_GateElseIfChain tests @else @if chain inside a gate.
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
		t.Fatalf("Routes: got %d", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Condition == nil {
		t.Error("Route[0] should have condition")
	}
	if ep.Gate.Routes[1].Condition == nil {
		t.Error("Route[1] should have condition")
	}
	if ep.Gate.Routes[2].Condition != nil {
		t.Error("Route[2] should be unconditional")
	}
}

// TestAudit_GateMixedNextEnd verifies that a gate can mix @next and @end
// leaves in the same block.
func TestAudit_GateMixedNextEnd(t *testing.T) {
	src := `@episode main:01 "T" {
	NARRATOR: hi.
	@gate {
		@if (A.success):
			@next good:01
		@else @if (TOTALLY_DEAD):
			@end bad_ending
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)
	if len(ep.Gate.Routes) != 3 {
		t.Fatalf("Routes: got %d", len(ep.Gate.Routes))
	}
	if _, ok := ep.Gate.Routes[0].Leaf.(*ast.NextLeaf); !ok {
		t.Errorf("Route[0].Leaf: got %T", ep.Gate.Routes[0].Leaf)
	}
	if end, ok := ep.Gate.Routes[1].Leaf.(*ast.EndLeaf); !ok {
		t.Errorf("Route[1].Leaf: got %T", ep.Gate.Routes[1].Leaf)
	} else if end.Type != ast.EndingBad {
		t.Errorf("Route[1].End.Type: got %q", end.Type)
	}
	if _, ok := ep.Gate.Routes[2].Leaf.(*ast.NextLeaf); !ok {
		t.Errorf("Route[2].Leaf: got %T", ep.Gate.Routes[2].Leaf)
	}
}

// =============================================================================
// AREA D: Block parsing interactions
// =============================================================================

// TestAudit_DeeplyNestedChoiceBrave tests a brave option whose body contains
// check.success/check.fail @if/@else with a further nested @if.
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
	if optA.Check == nil || optA.Check.Attr != "STR" || optA.Check.DC != 14 {
		t.Fatalf("Check: %+v", optA.Check)
	}
	if len(optA.Body) != 1 {
		t.Fatalf("Body length: got %d", len(optA.Body))
	}
	outer := optA.Body[0].(*ast.IfNode)
	if _, ok := outer.Condition.(*ast.CheckCondition); !ok {
		t.Fatalf("outer condition: got %T", outer.Condition)
	}
	if len(outer.Then) != 1 {
		t.Fatalf("outer Then length: got %d", len(outer.Then))
	}
	nested, ok := outer.Then[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("outer Then[0]: got %T", outer.Then[0])
	}
	if len(nested.Then) != 1 || len(nested.Else) != 1 {
		t.Errorf("nested branches: then=%d else=%d", len(nested.Then), len(nested.Else))
	}
}

// TestAudit_IfInsidePhone is now expected to FAIL (the @phone whitelist is
// strict and only allows @text). Pin that behavior with an error test.
func TestAudit_IfInsidePhoneRejected(t *testing.T) {
	src := `@episode main:01 "T" {
	@phone {
		@if (flag) {
			@text from easton: Hello!
		}
	}
	@gate { @next main:02 }
}`
	_, err := parseSource(src)
	if err == nil {
		t.Fatal("expected parse error: @if not allowed inside @phone")
	}
	if !strings.Contains(err.Error(), "only @text") {
		t.Errorf("error should mention @text whitelist, got: %v", err)
	}
}

// =============================================================================
// AREA E: Concurrent flag on various node types
// =============================================================================

// TestAudit_ConcurrentBgSet verifies & on @bg set.
func TestAudit_ConcurrentBgSet(t *testing.T) {
	src := `@episode main:01 "T" {
	@bg set hallway
	&bg set classroom
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	if ep.Body[0].(*ast.BgSetNode).GetConcurrent() {
		t.Error("First bg should not be concurrent")
	}
	bg2 := ep.Body[1].(*ast.BgSetNode)
	if !bg2.GetConcurrent() {
		t.Error("Second bg should be concurrent")
	}
	if bg2.Name != "classroom" {
		t.Errorf("bg2.Name: got %q", bg2.Name)
	}
}

// TestAudit_ConcurrentPause verifies & on @pause.
func TestAudit_ConcurrentPause(t *testing.T) {
	src := `@episode main:01 "T" {
	&pause
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	pause, ok := ep.Body[0].(*ast.PauseNode)
	if !ok {
		t.Fatalf("Body[0]: expected *PauseNode, got %T", ep.Body[0])
	}
	if !pause.GetConcurrent() {
		t.Error("PauseNode should be concurrent")
	}
}

// TestAudit_ConcurrentCharShow verifies & on `@<char> <pose>`.
func TestAudit_ConcurrentCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	&malia happy
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	cs, ok := ep.Body[0].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if !cs.GetConcurrent() {
		t.Error("CharShowNode should be concurrent")
	}
	if cs.Char != "malia" || cs.Look != "happy" {
		t.Errorf("got char=%q look=%q", cs.Char, cs.Look)
	}
}

// TestAudit_ConcurrentSfx verifies & on @sfx.
func TestAudit_ConcurrentSfx(t *testing.T) {
	src := `@episode main:01 "T" {
	&sfx swoosh
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	sfx, ok := ep.Body[0].(*ast.SfxNode)
	if !ok {
		t.Fatalf("Body[0]: expected *SfxNode, got %T", ep.Body[0])
	}
	if !sfx.GetConcurrent() {
		t.Error("SfxNode should be concurrent")
	}
	if sfx.Name != "swoosh" {
		t.Errorf("Name: got %q", sfx.Name)
	}
}

// TestAudit_ConcurrentMusic verifies & on @music <name>.
func TestAudit_ConcurrentMusic(t *testing.T) {
	src := `@episode main:01 "T" {
	&music theme_main
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	mp, ok := ep.Body[0].(*ast.MusicSetNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicSetNode, got %T", ep.Body[0])
	}
	if !mp.GetConcurrent() {
		t.Error("MusicSetNode should be concurrent")
	}
}

// TestAudit_ConcurrentMusicStop verifies & on @music stop.
func TestAudit_ConcurrentMusicStop(t *testing.T) {
	src := `@episode main:01 "T" {
	&music stop
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	ms, ok := ep.Body[0].(*ast.MusicStopNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicStopNode, got %T", ep.Body[0])
	}
	if !ms.GetConcurrent() {
		t.Error("MusicStopNode should be concurrent")
	}
}

// TestAudit_ConcurrentAffection verifies & on @affection.
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

// TestAudit_ConcurrentSignal verifies & on @signal mark.
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

// TestAudit_ConcurrentCharBubble verifies & on `@<char> bubble <type>`.
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

// TestAudit_ConcurrentButterfly verifies & on @butterfly.
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

// TestAudit_DialogueWithExprThenDirective covers dialogue-with-expr followed
// by a directive on the next line.
func TestAudit_DialogueWithExprThenDirective(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@bg set hallway
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	bg, ok := ep.Body[2].(*ast.BgSetNode)
	if !ok || bg.Name != "hallway" {
		t.Errorf("Body[2]: got %+v", ep.Body[2])
	}
}

// TestAudit_DialogueWithExprThenConcurrent covers dialogue-with-expr followed
// by a concurrent directive.
func TestAudit_DialogueWithExprThenConcurrent(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	&music theme
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: got %T", ep.Body[1])
	}
	mp, ok := ep.Body[2].(*ast.MusicSetNode)
	if !ok {
		t.Fatalf("Body[2]: got %T", ep.Body[2])
	}
	if !mp.GetConcurrent() {
		t.Error("MusicSetNode should be concurrent")
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
	if len(ep.Body) != 7 {
		t.Fatalf("Body length: got %d, want 7", len(ep.Body))
	}
	expected := []string{
		"*ast.NarratorNode",
		"*ast.CharShowNode",
		"*ast.DialogueNode",
		"*ast.NarratorNode",
		"*ast.CharShowNode",
		"*ast.DialogueNode",
		"*ast.NarratorNode",
	}
	for i, exp := range expected {
		got := fmt.Sprintf("%T", ep.Body[i])
		if got != exp {
			t.Errorf("Body[%d]: got %s, want %s", i, got, exp)
		}
	}
}

// TestAudit_DialogueWithExprBeforeGateEpisodeLevel verifies pending is drained
// when @gate immediately follows a dialogue-with-expr line at episode level.
func TestAudit_DialogueWithExprBeforeGateEpisodeLevel(t *testing.T) {
	src := `@episode main:01 "T" {
	MAURICIO [angry]: Get out.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Errorf("Body[1]: got %T", ep.Body[1])
	}
}

// =============================================================================
// AREA G: Error handling edge cases
// =============================================================================

// TestAudit_ConcurrentDialogueError tests that &NARRATOR: ... gives a helpful
// error.
func TestAudit_ConcurrentDialogueError(t *testing.T) {
	src := `@episode main:01 "T" {
	&NARRATOR: Hello.
	@gate { @next main:02 }
}`
	_, err := parseSource(src)
	if err == nil {
		t.Fatal("expected error for &NARRATOR: dialogue")
	}
}

// TestAudit_NestingDepthLimit tests that deeply nested blocks fail with a
// nesting-depth error.
func TestAudit_NestingDepthLimit(t *testing.T) {
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

	_, err := parseSource(src.String())
	if err == nil {
		t.Fatal("expected nesting-depth error")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Errorf("expected 'nesting depth' in error, got: %v", err)
	}
}

// TestAudit_EmptyEpisodeBody verifies an empty episode body parses cleanly.
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

// TestAudit_MultipleDirectivesTight tests directives separated only by
// newlines.
func TestAudit_MultipleDirectivesTight(t *testing.T) {
	src := "@episode main:01 \"T\" {\n@bg set classroom\n@music theme\nNARRATOR: Hello.\n@gate { @next main:02 }\n}"
	ep := parseOrFail(t, src)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}
}

// =============================================================================
// AREA H: Bg / char show + dialogue interleaving
// =============================================================================

// TestAudit_BgTransitionBeforeDialogue verifies the optional bg transition is
// consumed and not confused with the next line's dialogue character name.
func TestAudit_BgTransitionBeforeDialogue(t *testing.T) {
	src := `@episode main:01 "T" {
	@bg set classroom fade
	NARRATOR: Hello.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	bg := ep.Body[0].(*ast.BgSetNode)
	if bg.Transition != "fade" {
		t.Errorf("Transition: got %q", bg.Transition)
	}
	narr := ep.Body[1].(*ast.NarratorNode)
	if narr.Text != "Hello." {
		t.Errorf("narr text: got %q", narr.Text)
	}
}

// TestAudit_DialogueAfterCharShow verifies that NARRATOR: after `@<char>
// <pose>` works (the optional transition lookahead is properly bounded by
// isDirectiveStart).
func TestAudit_DialogueAfterCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	@mauricio neutral
	NARRATOR: Hello.
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	show := ep.Body[0].(*ast.CharShowNode)
	if show.Transition != "" {
		t.Errorf("Transition: got %q, want empty", show.Transition)
	}
	if show.Look != "neutral" {
		t.Errorf("Look: got %q", show.Look)
	}
}

// TestAudit_DialogueWithExprAfterCharShow verifies CHARACTER [pose]: text
// after `@<char> <pose>` parses cleanly as 3 nodes (CharShow, CharShow,
// Dialogue).
func TestAudit_DialogueWithExprAfterCharShow(t *testing.T) {
	src := `@episode main:01 "T" {
	@mauricio neutral
	MAURICIO [happy]: Hello!
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d", len(ep.Body))
	}
	if _, ok := ep.Body[0].(*ast.CharShowNode); !ok {
		t.Errorf("Body[0]: got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.CharShowNode); !ok {
		t.Errorf("Body[1]: got %T", ep.Body[1])
	}
	if _, ok := ep.Body[2].(*ast.DialogueNode); !ok {
		t.Errorf("Body[2]: got %T", ep.Body[2])
	}
}

// TestAudit_SafeOptionBody tests a safe option with multiple body nodes.
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
	if len(choice.Options[0].Body) != 3 {
		t.Fatalf("Option body length: got %d", len(choice.Options[0].Body))
	}
}

// TestAudit_ChoiceMultipleOptions tests a choice with more than 2 options.
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
		t.Fatalf("Options count: got %d", len(choice.Options))
	}
}

// TestAudit_MinigameLeafShape pins the @minigame leaf form (name + description).
func TestAudit_MinigameLeafShape(t *testing.T) {
	src := `@episode main:01 "T" {
	@minigame test "minigame description placeholder"
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	mg := ep.Body[0].(*ast.MinigameNode)
	if mg.Name != "test" {
		t.Errorf("Name: got %q", mg.Name)
	}
	if mg.Description != "minigame description placeholder" {
		t.Errorf("Description: got %q", mg.Description)
	}
}

// TestAudit_CgLeafShape pins the @cg leaf form (name + content string only).
func TestAudit_CgLeafShape(t *testing.T) {
	src := `@episode main:01 "T" {
	@cg sunset "Camera pans across the horizon and holds on a silhouette of the cliff."
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	cg := ep.Body[0].(*ast.CgShowNode)
	if cg.Name != "sunset" {
		t.Errorf("Name: got %q", cg.Name)
	}
	if !strings.Contains(cg.Content, "horizon") {
		t.Errorf("Content: got %q", cg.Content)
	}
}
