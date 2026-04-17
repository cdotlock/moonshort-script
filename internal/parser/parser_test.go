package parser

import (
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
)

// helper parses src and returns the Episode or fails the test.
func parseOrFail(t *testing.T, src string) *ast.Episode {
	t.Helper()
	l := lexer.New(src)
	p := New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	return ep
}

// TestParseMinimal verifies a minimal episode with bg, narrator, you, and gates.
func TestParseMinimal(t *testing.T) {
	src := `@episode main:01 "Test" {
	@bg set classroom fade
	NARRATOR: Hello.
	YOU: Thinking.
	@gate {
		@next main:02
	}
}`
	ep := parseOrFail(t, src)

	if ep.BranchKey != "main:01" {
		t.Errorf("BranchKey: got %q, want %q", ep.BranchKey, "main:01")
	}
	if ep.Title != "Test" {
		t.Errorf("Title: got %q, want %q", ep.Title, "Test")
	}
	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	// Node 0: BgSetNode
	bg, ok := ep.Body[0].(*ast.BgSetNode)
	if !ok {
		t.Fatalf("Body[0]: expected *BgSetNode, got %T", ep.Body[0])
	}
	if bg.Name != "classroom" {
		t.Errorf("BgSetNode.Name: got %q, want %q", bg.Name, "classroom")
	}
	if bg.Transition != "fade" {
		t.Errorf("BgSetNode.Transition: got %q, want %q", bg.Transition, "fade")
	}

	// Node 1: NarratorNode
	narr, ok := ep.Body[1].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Body[1]: expected *NarratorNode, got %T", ep.Body[1])
	}
	if narr.Text != "Hello." {
		t.Errorf("NarratorNode.Text: got %q, want %q", narr.Text, "Hello.")
	}

	// Node 2: YouNode
	you, ok := ep.Body[2].(*ast.YouNode)
	if !ok {
		t.Fatalf("Body[2]: expected *YouNode, got %T", ep.Body[2])
	}
	if you.Text != "Thinking." {
		t.Errorf("YouNode.Text: got %q, want %q", you.Text, "Thinking.")
	}

	// Gate
	if ep.Gate == nil {
		t.Fatal("Gate: expected non-nil")
	}
	if len(ep.Gate.Routes) != 1 {
		t.Fatalf("Gate.Routes count: got %d, want 1", len(ep.Gate.Routes))
	}
	def := ep.Gate.Routes[0]
	if def.Condition != nil {
		t.Errorf("GateRoute.Condition: got %v, want nil", def.Condition)
	}
	if def.Target != "main:02" {
		t.Errorf("GateRoute.Target: got %q, want %q", def.Target, "main:02")
	}
}

// TestParseCharDirectives tests show, expr, bubble, move, hide, and dialogue.
func TestParseCharDirectives(t *testing.T) {
	src := `@episode main:01 "Chars" {
	@mauricio show neutral at center fade
	@mauricio look angry
	@mauricio bubble heart
	@mauricio move to left
	MAURICIO: Hey there.
	@mauricio hide fade
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 6 {
		t.Fatalf("Body length: got %d, want 6", len(ep.Body))
	}

	// show
	show, ok := ep.Body[0].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharShowNode, got %T", ep.Body[0])
	}
	if show.Char != "mauricio" {
		t.Errorf("CharShowNode.Char: got %q, want %q", show.Char, "mauricio")
	}
	if show.Look != "neutral" {
		t.Errorf("CharShowNode.Look: got %q, want %q", show.Look, "neutral")
	}
	if show.Position != "center" {
		t.Errorf("CharShowNode.Position: got %q, want %q", show.Position, "center")
	}
	if show.Transition != "fade" {
		t.Errorf("CharShowNode.Transition: got %q, want %q", show.Transition, "fade")
	}

	// expr (now CharLookNode)
	expr, ok := ep.Body[1].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[1]: expected *CharLookNode, got %T", ep.Body[1])
	}
	if expr.Char != "mauricio" || expr.Look != "angry" {
		t.Errorf("CharLookNode: got char=%q look=%q", expr.Char, expr.Look)
	}

	// bubble
	bubble, ok := ep.Body[2].(*ast.CharBubbleNode)
	if !ok {
		t.Fatalf("Body[2]: expected *CharBubbleNode, got %T", ep.Body[2])
	}
	if bubble.Char != "mauricio" || bubble.BubbleType != "heart" {
		t.Errorf("CharBubbleNode: got char=%q type=%q", bubble.Char, bubble.BubbleType)
	}

	// move
	mv, ok := ep.Body[3].(*ast.CharMoveNode)
	if !ok {
		t.Fatalf("Body[3]: expected *CharMoveNode, got %T", ep.Body[3])
	}
	if mv.Char != "mauricio" || mv.Position != "left" {
		t.Errorf("CharMoveNode: got char=%q pos=%q", mv.Char, mv.Position)
	}

	// dialogue
	dlg, ok := ep.Body[4].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[4]: expected *DialogueNode, got %T", ep.Body[4])
	}
	if dlg.Character != "MAURICIO" || dlg.Text != "Hey there." {
		t.Errorf("DialogueNode: got char=%q text=%q", dlg.Character, dlg.Text)
	}

	// hide
	hide, ok := ep.Body[5].(*ast.CharHideNode)
	if !ok {
		t.Fatalf("Body[5]: expected *CharHideNode, got %T", ep.Body[5])
	}
	if hide.Char != "mauricio" || hide.Transition != "fade" {
		t.Errorf("CharHideNode: got char=%q trans=%q", hide.Char, hide.Transition)
	}
}

// TestParseAudio tests music play, crossfade, fadeout, and sfx play.
func TestParseAudio(t *testing.T) {
	src := `@episode main:01 "Audio" {
	@music play theme_main
	@sfx play door_open
	@music crossfade theme_battle
	@music fadeout
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 4 {
		t.Fatalf("Body length: got %d, want 4", len(ep.Body))
	}

	// music play
	mp, ok := ep.Body[0].(*ast.MusicPlayNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicPlayNode, got %T", ep.Body[0])
	}
	if mp.Track != "theme_main" {
		t.Errorf("MusicPlayNode.Track: got %q, want %q", mp.Track, "theme_main")
	}

	// sfx play
	sfx, ok := ep.Body[1].(*ast.SfxPlayNode)
	if !ok {
		t.Fatalf("Body[1]: expected *SfxPlayNode, got %T", ep.Body[1])
	}
	if sfx.Sound != "door_open" {
		t.Errorf("SfxPlayNode.Sound: got %q, want %q", sfx.Sound, "door_open")
	}

	// music crossfade
	mcf, ok := ep.Body[2].(*ast.MusicCrossfadeNode)
	if !ok {
		t.Fatalf("Body[2]: expected *MusicCrossfadeNode, got %T", ep.Body[2])
	}
	if mcf.Track != "theme_battle" {
		t.Errorf("MusicCrossfadeNode.Track: got %q, want %q", mcf.Track, "theme_battle")
	}

	// music fadeout
	_, ok = ep.Body[3].(*ast.MusicFadeoutNode)
	if !ok {
		t.Fatalf("Body[3]: expected *MusicFadeoutNode, got %T", ep.Body[3])
	}
}

// TestParseStateChanges tests affection, signal, and butterfly.
func TestParseStateChanges(t *testing.T) {
	src := `@episode main:01 "State" {
	@affection mauricio +2
	@signal mark EP01_COMPLETE
	@butterfly "Player chose kindness"
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	aff, ok := ep.Body[0].(*ast.AffectionNode)
	if !ok {
		t.Fatalf("Body[0]: expected *AffectionNode, got %T", ep.Body[0])
	}
	if aff.Char != "mauricio" || aff.Delta != "+2" {
		t.Errorf("AffectionNode: got char=%q delta=%q", aff.Char, aff.Delta)
	}

	sig, ok := ep.Body[1].(*ast.SignalNode)
	if !ok {
		t.Fatalf("Body[1]: expected *SignalNode, got %T", ep.Body[1])
	}
	if sig.Kind != "mark" {
		t.Errorf("SignalNode.Kind: got %q, want %q", sig.Kind, "mark")
	}
	if sig.Event != "EP01_COMPLETE" {
		t.Errorf("SignalNode.Event: got %q, want %q", sig.Event, "EP01_COMPLETE")
	}

	btf, ok := ep.Body[2].(*ast.ButterflyNode)
	if !ok {
		t.Fatalf("Body[2]: expected *ButterflyNode, got %T", ep.Body[2])
	}
	if btf.Description != "Player chose kindness" {
		t.Errorf("ButterflyNode.Description: got %q, want %q", btf.Description, "Player chose kindness")
	}
}

// TestParseChoice tests a choice with brave (check + on success/fail) and safe options.
func TestParseChoice(t *testing.T) {
	src := `@episode main:01 "Choice" {
	@choice {
		@option A brave "Fight the guard" {
			check {
				attr: STR
				dc: 14
			}
			@on success {
				NARRATOR: You overpower the guard.
			}
			@on fail {
				NARRATOR: The guard throws you back.
			}
		}
		@option B safe "Talk it out" {
			NARRATOR: You reason with the guard.
			@affection guard +1
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	choice, ok := ep.Body[0].(*ast.ChoiceNode)
	if !ok {
		t.Fatalf("Body[0]: expected *ChoiceNode, got %T", ep.Body[0])
	}
	if len(choice.Options) != 2 {
		t.Fatalf("Options count: got %d, want 2", len(choice.Options))
	}

	// Brave option
	optA := choice.Options[0]
	if optA.ID != "A" || optA.Mode != "brave" || optA.Text != "Fight the guard" {
		t.Errorf("Option A: id=%q mode=%q text=%q", optA.ID, optA.Mode, optA.Text)
	}
	if optA.Check == nil {
		t.Fatal("Option A: expected Check to be non-nil")
	}
	if optA.Check.Attr != "STR" || optA.Check.DC != 14 {
		t.Errorf("Check: attr=%q dc=%d, want STR/14", optA.Check.Attr, optA.Check.DC)
	}
	if len(optA.OnSuccess) != 1 {
		t.Fatalf("OnSuccess length: got %d, want 1", len(optA.OnSuccess))
	}
	if len(optA.OnFail) != 1 {
		t.Fatalf("OnFail length: got %d, want 1", len(optA.OnFail))
	}

	// Verify success body content.
	succNarr, ok := optA.OnSuccess[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("OnSuccess[0]: expected *NarratorNode, got %T", optA.OnSuccess[0])
	}
	if succNarr.Text != "You overpower the guard." {
		t.Errorf("OnSuccess narr: got %q", succNarr.Text)
	}

	// Verify fail body content.
	failNarr, ok := optA.OnFail[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("OnFail[0]: expected *NarratorNode, got %T", optA.OnFail[0])
	}
	if failNarr.Text != "The guard throws you back." {
		t.Errorf("OnFail narr: got %q", failNarr.Text)
	}

	// Safe option
	optB := choice.Options[1]
	if optB.ID != "B" || optB.Mode != "safe" || optB.Text != "Talk it out" {
		t.Errorf("Option B: id=%q mode=%q text=%q", optB.ID, optB.Mode, optB.Text)
	}
	if optB.Check != nil {
		t.Error("Option B: expected Check to be nil for safe option")
	}
	if len(optB.Body) != 2 {
		t.Fatalf("Option B body length: got %d, want 2", len(optB.Body))
	}
}

// TestParseMinigame tests minigame with @on blocks for different rating groups.
func TestParseMinigame(t *testing.T) {
	src := `@episode main:01 "Mini" {
	@minigame arm_wrestle STR {
		@on S {
			NARRATOR: Perfect victory!
			@affection mauricio +10
		}
		@on A B {
			NARRATOR: Good job.
			@affection mauricio +5
		}
		@on C D {
			NARRATOR: Could be better.
			@affection mauricio +1
		}
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	mg, ok := ep.Body[0].(*ast.MinigameNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MinigameNode, got %T", ep.Body[0])
	}
	if mg.ID != "arm_wrestle" {
		t.Errorf("MinigameNode.ID: got %q, want %q", mg.ID, "arm_wrestle")
	}
	if mg.Attr != "STR" {
		t.Errorf("MinigameNode.Attr: got %q, want %q", mg.Attr, "STR")
	}
	if len(mg.OnResult) != 3 {
		t.Fatalf("OnResult count: got %d, want 3", len(mg.OnResult))
	}

	// Check each rating group exists.
	for _, key := range []string{"S", "A B", "C D"} {
		body, exists := mg.OnResult[key]
		if !exists {
			t.Errorf("OnResult[%q]: missing", key)
			continue
		}
		if len(body) != 2 {
			t.Errorf("OnResult[%q]: got %d nodes, want 2", key, len(body))
		}
	}

	// Verify S block content.
	sBody := mg.OnResult["S"]
	narr, ok := sBody[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("OnResult[S][0]: expected *NarratorNode, got %T", sBody[0])
	}
	if narr.Text != "Perfect victory!" {
		t.Errorf("S narr: got %q", narr.Text)
	}
}

// TestParsePhone tests phone show with text messages and phone hide.
func TestParsePhone(t *testing.T) {
	src := `@episode main:01 "Phone" {
	@phone show {
		@text from easton: Hey are you free?
		@text to easton: Yeah what's up?
		@text from easton: Meet me at the park.
	}
	@phone hide
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2", len(ep.Body))
	}

	phone, ok := ep.Body[0].(*ast.PhoneShowNode)
	if !ok {
		t.Fatalf("Body[0]: expected *PhoneShowNode, got %T", ep.Body[0])
	}
	if len(phone.Body) != 3 {
		t.Fatalf("PhoneShowNode.Body length: got %d, want 3", len(phone.Body))
	}

	// First text message.
	msg0, ok := phone.Body[0].(*ast.TextMessageNode)
	if !ok {
		t.Fatalf("phone.Body[0]: expected *TextMessageNode, got %T", phone.Body[0])
	}
	if msg0.Direction != "from" || msg0.Char != "easton" || msg0.Content != "Hey are you free?" {
		t.Errorf("msg0: dir=%q char=%q content=%q", msg0.Direction, msg0.Char, msg0.Content)
	}

	// Second text message.
	msg1, ok := phone.Body[1].(*ast.TextMessageNode)
	if !ok {
		t.Fatalf("phone.Body[1]: expected *TextMessageNode, got %T", phone.Body[1])
	}
	if msg1.Direction != "to" || msg1.Char != "easton" || msg1.Content != "Yeah what's up?" {
		t.Errorf("msg1: dir=%q char=%q content=%q", msg1.Direction, msg1.Char, msg1.Content)
	}

	// Third text message.
	msg2, ok := phone.Body[2].(*ast.TextMessageNode)
	if !ok {
		t.Fatalf("phone.Body[2]: expected *TextMessageNode, got %T", phone.Body[2])
	}
	if msg2.Direction != "from" || msg2.Char != "easton" || msg2.Content != "Meet me at the park." {
		t.Errorf("msg2: dir=%q char=%q content=%q", msg2.Direction, msg2.Char, msg2.Content)
	}

	// Phone hide.
	_, ok = ep.Body[1].(*ast.PhoneHideNode)
	if !ok {
		t.Fatalf("Body[1]: expected *PhoneHideNode, got %T", ep.Body[1])
	}
}

// TestParseIfElse tests @if with condition and @else block.
func TestParseIfElse(t *testing.T) {
	src := `@episode main:01 "Branch" {
	@if (affection.easton >= 5) {
		NARRATOR: Easton smiles at you warmly.
		@affection easton +1
	}
	@else {
		NARRATOR: Easton barely notices you.
	}
	@gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	ifNode, ok := ep.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Body[0]: expected *IfNode, got %T", ep.Body[0])
	}
	cmp, ok := ifNode.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Condition: expected *ComparisonCondition, got %T", ifNode.Condition)
	}
	if cmp.Left.Kind != ast.OperandAffection || cmp.Left.Char != "easton" {
		t.Errorf("Condition.Left: got %+v, want affection/easton", cmp.Left)
	}
	if cmp.Op != ">=" || cmp.Right != 5 {
		t.Errorf("Condition: got op=%q right=%d, want >=/5", cmp.Op, cmp.Right)
	}
	if len(ifNode.Then) != 2 {
		t.Fatalf("Then length: got %d, want 2", len(ifNode.Then))
	}
	if ifNode.Else == nil {
		t.Fatal("Else: expected non-nil")
	}
	if len(ifNode.Else) != 1 {
		t.Fatalf("Else length: got %d, want 1", len(ifNode.Else))
	}

	// Verify then branch content.
	thenNarr, ok := ifNode.Then[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Then[0]: expected *NarratorNode, got %T", ifNode.Then[0])
	}
	if thenNarr.Text != "Easton smiles at you warmly." {
		t.Errorf("Then narr: got %q", thenNarr.Text)
	}

	// Verify else branch content.
	elseNarr, ok := ifNode.Else[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Else[0]: expected *NarratorNode, got %T", ifNode.Else[0])
	}
	if elseNarr.Text != "Easton barely notices you." {
		t.Errorf("Else narr: got %q", elseNarr.Text)
	}
}

// TestParseElseIf tests @if / @else @if / @else chaining.
func TestParseElseIf(t *testing.T) {
	src := `@episode main:01 "ElseIf" {
    @if (affection.easton >= 5) {
        NARRATOR: High affection.
    } @else @if (CHA >= 14) {
        NARRATOR: High charisma.
    } @else {
        NARRATOR: Default.
    }
    @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	ifNode, ok := ep.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Body[0]: expected *IfNode, got %T", ep.Body[0])
	}
	if _, ok := ifNode.Condition.(*ast.ComparisonCondition); !ok {
		t.Errorf("Condition: want *ComparisonCondition, got %T", ifNode.Condition)
	}
	if len(ifNode.Then) != 1 {
		t.Fatalf("Then length: got %d, want 1", len(ifNode.Then))
	}

	// Else should contain exactly one IfNode (the else-if)
	if len(ifNode.Else) != 1 {
		t.Fatalf("Else length: got %d, want 1", len(ifNode.Else))
	}
	elseIf, ok := ifNode.Else[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Else[0]: expected *IfNode, got %T", ifNode.Else[0])
	}
	if _, ok := elseIf.Condition.(*ast.ComparisonCondition); !ok {
		t.Errorf("ElseIf.Condition: want *ComparisonCondition, got %T", elseIf.Condition)
	}
	if len(elseIf.Then) != 1 {
		t.Fatalf("ElseIf.Then length: got %d, want 1", len(elseIf.Then))
	}

	// Final else
	if len(elseIf.Else) != 1 {
		t.Fatalf("ElseIf.Else length: got %d, want 1", len(elseIf.Else))
	}
}

// TestParseDialogueWithExpr tests that CHARACTER [pose_expr]: text expands to
// CharLookNode + DialogueNode in the correct order.
func TestParseDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  MAURICIO [arms_crossed_angry]: Your call, Butterfly.
  @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharLookNode + DialogueNode)", len(ep.Body))
	}

	// Body[0] must be CharLookNode.
	expr, ok := ep.Body[0].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if expr.Char != "mauricio" {
		t.Errorf("CharLookNode.Char: got %q, want %q", expr.Char, "mauricio")
	}
	if expr.Look != "arms_crossed_angry" {
		t.Errorf("CharLookNode.Look: got %q, want %q", expr.Look, "arms_crossed_angry")
	}

	// Body[1] must be DialogueNode.
	dlg, ok := ep.Body[1].(*ast.DialogueNode)
	if !ok {
		t.Fatalf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	if dlg.Character != "MAURICIO" {
		t.Errorf("DialogueNode.Character: got %q, want %q", dlg.Character, "MAURICIO")
	}
	if dlg.Text != "Your call, Butterfly." {
		t.Errorf("DialogueNode.Text: got %q, want %q", dlg.Text, "Your call, Butterfly.")
	}
}

// TestParseYouWithExpr tests that YOU [pose_expr]: text expands to CharLookNode + YouNode.
func TestParseYouWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  YOU [thinking]: Why is he looking at me like that?
  @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharLookNode + YouNode)", len(ep.Body))
	}

	// Body[0] must be CharLookNode for "you".
	expr, ok := ep.Body[0].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if expr.Char != "you" {
		t.Errorf("CharLookNode.Char: got %q, want %q", expr.Char, "you")
	}
	if expr.Look != "thinking" {
		t.Errorf("CharLookNode.Look: got %q, want %q", expr.Look, "thinking")
	}

	// Body[1] must be YouNode.
	you, ok := ep.Body[1].(*ast.YouNode)
	if !ok {
		t.Fatalf("Body[1]: expected *YouNode, got %T", ep.Body[1])
	}
	if you.Text != "Why is he looking at me like that?" {
		t.Errorf("YouNode.Text: got %q, want %q", you.Text, "Why is he looking at me like that?")
	}
}

// TestParseNarratorWithExpr tests that NARRATOR [pose_expr]: text expands to
// CharLookNode + NarratorNode.
func TestParseNarratorWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  NARRATOR [somber]: Three days passed without a word.
  @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharLookNode + NarratorNode)", len(ep.Body))
	}

	expr, ok := ep.Body[0].(*ast.CharLookNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if expr.Char != "narrator" {
		t.Errorf("CharLookNode.Char: got %q, want %q", expr.Char, "narrator")
	}
	if expr.Look != "somber" {
		t.Errorf("CharLookNode.Look: got %q, want %q", expr.Look, "somber")
	}

	narr, ok := ep.Body[1].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Body[1]: expected *NarratorNode, got %T", ep.Body[1])
	}
	if narr.Text != "Three days passed without a word." {
		t.Errorf("NarratorNode.Text: got %q, want %q", narr.Text, "Three days passed without a word.")
	}
}

// TestParseDialogueWithExprFollowedByMore verifies that nodes after the syntax sugar
// are parsed correctly (pending is drained, then parsing continues normally).
func TestParseDialogueWithExprFollowedByMore(t *testing.T) {
	src := `@episode main:01 "Test" {
  MAURICIO [arms_crossed_angry]: Your call, Butterfly.
  NARRATOR: She turned away.
  @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	if _, ok := ep.Body[0].(*ast.CharLookNode); !ok {
		t.Fatalf("Body[0]: expected *CharLookNode, got %T", ep.Body[0])
	}
	if _, ok := ep.Body[1].(*ast.DialogueNode); !ok {
		t.Fatalf("Body[1]: expected *DialogueNode, got %T", ep.Body[1])
	}
	narr, ok := ep.Body[2].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Body[2]: expected *NarratorNode, got %T", ep.Body[2])
	}
	if narr.Text != "She turned away." {
		t.Errorf("NarratorNode.Text: got %q, want %q", narr.Text, "She turned away.")
	}
}

// TestParseGates tests gate block with conditional routes and a fallback.
func TestParseGates(t *testing.T) {
	src := `@episode main:01 "Gates" {
	NARRATOR: The story branches.
	@gate {
		@if (A.success):
			@next main/good/001:01
		@else @if (affection.easton >= 10):
			@next main/bad/001:01
		@else:
			@next main:02
	}
}`
	ep := parseOrFail(t, src)

	if ep.Gate == nil {
		t.Fatal("Gate: expected non-nil")
	}
	if len(ep.Gate.Routes) != 3 {
		t.Fatalf("Gate.Routes count: got %d, want 3", len(ep.Gate.Routes))
	}

	// Conditional route 0: choice A.success
	r0 := ep.Gate.Routes[0]
	ch0, ok := r0.Condition.(*ast.ChoiceCondition)
	if !ok {
		t.Fatalf("Route[0].Condition: want *ChoiceCondition, got %T", r0.Condition)
	}
	if ch0.Option != "A" {
		t.Errorf("Route[0].Condition.Option: got %q, want %q", ch0.Option, "A")
	}
	if ch0.Result != "success" {
		t.Errorf("Route[0].Condition.Result: got %q, want %q", ch0.Result, "success")
	}
	if r0.Target != "main/good/001:01" {
		t.Errorf("Route[0].Target: got %q, want %q", r0.Target, "main/good/001:01")
	}

	// Conditional route 1: comparison affection.easton >= 10
	r1 := ep.Gate.Routes[1]
	cmp1, ok := r1.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Route[1].Condition: want *ComparisonCondition, got %T", r1.Condition)
	}
	if cmp1.Left.Kind != ast.OperandAffection || cmp1.Left.Char != "easton" {
		t.Errorf("Route[1].Condition.Left: got %+v, want affection/easton", cmp1.Left)
	}
	if cmp1.Op != ">=" || cmp1.Right != 10 {
		t.Errorf("Route[1].Condition: got op=%q right=%d, want >=/10", cmp1.Op, cmp1.Right)
	}
	if r1.Target != "main/bad/001:01" {
		t.Errorf("Route[1].Target: got %q, want %q", r1.Target, "main/bad/001:01")
	}

	// Fallback route 2: unconditional
	r2 := ep.Gate.Routes[2]
	if r2.Condition != nil {
		t.Errorf("Route[2].Condition: got %v, want nil", r2.Condition)
	}
	if r2.Target != "main:02" {
		t.Errorf("Route[2].Target: got %q, want %q", r2.Target, "main:02")
	}
}

// TestParseConcurrent tests that & prefix produces nodes with Concurrent=true.
func TestParseConcurrent(t *testing.T) {
	src := `@episode main:01 "Concurrent" {
    @bg set classroom fade
    &music play theme_main
    &malia show neutral at left
    NARRATOR: Hello.
    @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 4 {
		t.Fatalf("Body length: got %d, want 4", len(ep.Body))
	}

	// Body[0]: bg (non-concurrent, leader)
	bg, ok := ep.Body[0].(*ast.BgSetNode)
	if !ok {
		t.Fatalf("Body[0]: expected *BgSetNode, got %T", ep.Body[0])
	}
	if bg.GetConcurrent() {
		t.Error("BgSetNode should not be concurrent (leader)")
	}

	// Body[1]: music (concurrent)
	mp, ok := ep.Body[1].(*ast.MusicPlayNode)
	if !ok {
		t.Fatalf("Body[1]: expected *MusicPlayNode, got %T", ep.Body[1])
	}
	if !mp.GetConcurrent() {
		t.Error("MusicPlayNode should be concurrent")
	}

	// Body[2]: char show (concurrent)
	cs, ok := ep.Body[2].(*ast.CharShowNode)
	if !ok {
		t.Fatalf("Body[2]: expected *CharShowNode, got %T", ep.Body[2])
	}
	if !cs.GetConcurrent() {
		t.Error("CharShowNode should be concurrent")
	}

	// Body[3]: narrator (not concurrent)
	narr, ok := ep.Body[3].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Body[3]: expected *NarratorNode, got %T", ep.Body[3])
	}
	_ = narr
}

// TestParsePause tests @pause for N.
func TestParsePause(t *testing.T) {
	src := `@episode main:01 "Pause" {
    @bg set classroom
    @pause for 2
    NARRATOR: Hello.
    @gate { @next main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	pause, ok := ep.Body[1].(*ast.PauseNode)
	if !ok {
		t.Fatalf("Body[1]: expected *PauseNode, got %T", ep.Body[1])
	}
	if pause.Clicks != 2 {
		t.Errorf("PauseNode.Clicks: got %d, want 2", pause.Clicks)
	}
}

// --- Error path tests ---

func TestParseError_EmptyCondition(t *testing.T) {
	src := `@episode main:01 "T" { @if () { NARRATOR: Hi. } @gate { @next main:02 } }`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected error for empty condition")
	}
	// The new parser reports "expected condition, got RPAREN" when () is empty.
	if !strings.Contains(err.Error(), "expected condition") {
		t.Errorf("expected 'expected condition' in error, got: %v", err)
	}
}

func TestParseError_EmptyChoice(t *testing.T) {
	src := `@episode main:01 "T" { @choice { } @gate { @next main:02 } }`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected error for empty choice")
	}
}

func TestParseError_EmptyGate(t *testing.T) {
	src := `@episode main:01 "T" { @gate { } }`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected error for empty gate")
	}
}

func TestParseError_DuplicateGate(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: Hi.
		@gate { @next main:02 }
		@gate { @next main:03 }
	}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected error for duplicate gate")
	}
}

func TestParseError_InvalidPauseCount(t *testing.T) {
	src := `@episode main:01 "T" { @pause for abc @gate { @next main:02 } }`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected error for invalid pause count")
	}
}

// --- Condition type tests ---

func TestParseConditionFlag(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (EP01_COMPLETE) {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	fc, ok := ifNode.Condition.(*ast.FlagCondition)
	if !ok {
		t.Fatalf("want *FlagCondition, got %T", ifNode.Condition)
	}
	if fc.Name != "EP01_COMPLETE" {
		t.Errorf("want name=EP01_COMPLETE, got %s", fc.Name)
	}
}

func TestParseConditionInfluence(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (influence "Player shows empathy") {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	ic, ok := ifNode.Condition.(*ast.InfluenceCondition)
	if !ok {
		t.Fatalf("want *InfluenceCondition, got %T", ifNode.Condition)
	}
	if ic.Description != "Player shows empathy" {
		t.Errorf("want description, got %s", ic.Description)
	}
}

func TestParseConditionInfluenceBareString(t *testing.T) {
	src := `@episode main:01 "T" {
		@if ("Player shows empathy") {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if _, ok := ifNode.Condition.(*ast.InfluenceCondition); !ok {
		t.Errorf("want *InfluenceCondition, got %T", ifNode.Condition)
	}
}

func TestParseConditionCompound(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (affection.easton >= 5 && CHA >= 14) {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if _, ok := ifNode.Condition.(*ast.CompoundCondition); !ok {
		t.Errorf("want *CompoundCondition, got %T", ifNode.Condition)
	}
}

func TestParseConditionChoiceAny(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (A.any) {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	ch, ok := ifNode.Condition.(*ast.ChoiceCondition)
	if !ok {
		t.Fatalf("want *ChoiceCondition, got %T", ifNode.Condition)
	}
	if ch.Result != "any" {
		t.Errorf("want result=any, got %s", ch.Result)
	}
}

// --- Nested parentheses ---

func TestParseNestedParenCondition(t *testing.T) {
	// New grammar: comparison RHS must be a single integer literal,
	// so `(5 + 3)` is not valid. Use a simple integer.
	src := `@episode main:01 "T" {
		@if (affection.easton >= 8) {
			NARRATOR: Done.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if _, ok := ifNode.Condition.(*ast.ComparisonCondition); !ok {
		t.Errorf("want *ComparisonCondition, got %T", ifNode.Condition)
	}
}

// --- CG show ---

func TestParseCgShow(t *testing.T) {
	src := `@episode main:01 "T" {
		@cg show sunset fade {
			NARRATOR: Beautiful.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	cg := ep.Body[0].(*ast.CgShowNode)
	if cg.Name != "sunset" {
		t.Errorf("want name=sunset, got %s", cg.Name)
	}
	if cg.Transition != "fade" {
		t.Errorf("want transition=fade, got %s", cg.Transition)
	}
	if len(cg.Body) != 1 {
		t.Errorf("want 1 body node, got %d", len(cg.Body))
	}
}

// --- Label and Goto ---

func TestParseLabelGoto(t *testing.T) {
	src := `@episode main:01 "T" {
		@label AFTER_FIGHT
		NARRATOR: Middle.
		@goto AFTER_FIGHT
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	label := ep.Body[0].(*ast.LabelNode)
	if label.Name != "AFTER_FIGHT" {
		t.Errorf("want AFTER_FIGHT, got %s", label.Name)
	}
	gotoNode := ep.Body[2].(*ast.GotoNode)
	if gotoNode.Name != "AFTER_FIGHT" {
		t.Errorf("want AFTER_FIGHT, got %s", gotoNode.Name)
	}
}

// --- Signal accepts STRING ---

func TestParseSignalString(t *testing.T) {
	src := `@episode main:01 "T" {
		@signal mark "EP01_COMPLETE"
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	sig := ep.Body[0].(*ast.SignalNode)
	if sig.Kind != "mark" {
		t.Errorf("want Kind=mark, got %s", sig.Kind)
	}
	if sig.Event != "EP01_COMPLETE" {
		t.Errorf("want EP01_COMPLETE, got %s", sig.Event)
	}
}

// TestParseSignalMissingKind verifies the parser rejects @signal without a kind.
func TestParseSignalMissingKind(t *testing.T) {
	src := `@episode main:01 "T" {
		@signal EP01_COMPLETE
		@gate { @next main:02 }
	}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for @signal without kind")
	}
}

// TestParseSignalInvalidKind verifies the parser rejects @signal with an
// unknown kind token.
func TestParseSignalInvalidKind(t *testing.T) {
	src := `@episode main:01 "T" {
		@signal foo EP01_COMPLETE
		@gate { @next main:02 }
	}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for @signal with invalid kind 'foo'")
	}
}

// --- Bg without transition ---

func TestParseBgNoTransition(t *testing.T) {
	src := `@episode main:01 "T" {
		@bg set classroom
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	bg := ep.Body[0].(*ast.BgSetNode)
	if bg.Name != "classroom" {
		t.Errorf("want name=classroom, got %s", bg.Name)
	}
	if bg.Transition != "" {
		t.Errorf("want empty transition, got %s", bg.Transition)
	}
}

// --- CG without body ---

func TestParseCgNoBody(t *testing.T) {
	src := `@episode main:01 "T" {
		@cg show sunset
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	cg := ep.Body[0].(*ast.CgShowNode)
	if cg.Name != "sunset" {
		t.Errorf("want name=sunset, got %s", cg.Name)
	}
	if len(cg.Body) != 0 {
		t.Errorf("want 0 body nodes, got %d", len(cg.Body))
	}
}

// --- CharShow without transition ---

func TestParseCharShowNoTransition(t *testing.T) {
	src := `@episode main:01 "T" {
		@mauricio show neutral at center
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	show := ep.Body[0].(*ast.CharShowNode)
	if show.Transition != "" {
		t.Errorf("want empty transition, got %s", show.Transition)
	}
}

// --- CharLook with transition ---

func TestParseCharLookWithTransition(t *testing.T) {
	src := `@episode main:01 "T" {
		@mauricio look angry fade
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	look := ep.Body[0].(*ast.CharLookNode)
	if look.Look != "angry" {
		t.Errorf("want look=angry, got %s", look.Look)
	}
	if look.Transition != "fade" {
		t.Errorf("want transition=fade, got %s", look.Transition)
	}
}

// --- CharHide without transition ---

func TestParseCharHideNoTransition(t *testing.T) {
	src := `@episode main:01 "T" {
		@mauricio hide
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	hide := ep.Body[0].(*ast.CharHideNode)
	if hide.Transition != "" {
		t.Errorf("want empty transition, got %s", hide.Transition)
	}
}

// --- Gate with @next and target containing slashes ---

func TestParseGateNextSlashTarget(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: Hi.
		@gate {
			@next main/good/001:01
		}
	}`
	ep := parseOrFail(t, src)
	if ep.Gate.Routes[0].Target != "main/good/001:01" {
		t.Errorf("want target=main/good/001:01, got %s", ep.Gate.Routes[0].Target)
	}
}

// --- Brave option with body after check/on blocks ---

func TestParseBraveOptionWithBody(t *testing.T) {
	src := `@episode main:01 "T" {
		@choice {
			@option A brave "Fight" {
				check {
					attr: STR
					dc: 14
				}
				@on success {
					NARRATOR: You win.
				}
				@on fail {
					NARRATOR: You lose.
				}
				NARRATOR: The dust settles.
			}
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	choice := ep.Body[0].(*ast.ChoiceNode)
	optA := choice.Options[0]
	if len(optA.Body) != 1 {
		t.Fatalf("expected 1 body node after on blocks, got %d", len(optA.Body))
	}
	narr, ok := optA.Body[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("Body[0]: expected *NarratorNode, got %T", optA.Body[0])
	}
	if narr.Text != "The dust settles." {
		t.Errorf("narr text: got %q", narr.Text)
	}
}

// --- Gate with plain @else (no @if chain) ---

func TestParseGatePlainElse(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: Hi.
		@gate {
			@if (EP01_DONE):
				@next main:03
			@else:
				@next main:02
		}
	}`
	ep := parseOrFail(t, src)
	if len(ep.Gate.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(ep.Gate.Routes))
	}
	// First route should be conditional
	if ep.Gate.Routes[0].Condition == nil {
		t.Error("expected conditional first route")
	}
	// Second route should be unconditional fallback
	if ep.Gate.Routes[1].Condition != nil {
		t.Error("expected unconditional fallback route")
	}
	if ep.Gate.Routes[1].Target != "main:02" {
		t.Errorf("fallback target: got %q, want main:02", ep.Gate.Routes[1].Target)
	}
}

// --- @pause for 1 (single click) ---

func TestParsePauseSingle(t *testing.T) {
	src := `@episode main:01 "T" {
		@pause for 1
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	pause := ep.Body[0].(*ast.PauseNode)
	if pause.Clicks != 1 {
		t.Errorf("want clicks=1, got %d", pause.Clicks)
	}
}

// --- Multiple episodes should error (Parse only returns first) ---

func TestParseComments(t *testing.T) {
	src := `@episode main:01 "T" {
		// This is a comment
		NARRATOR: Hello.
		// Another comment
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1 (comments should be skipped)", len(ep.Body))
	}
}

// --- @pause for 0 (clamped to 1) ---

func TestParsePauseZeroClamped(t *testing.T) {
	src := `@episode main:01 "T" {
		@pause for 0
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	pause := ep.Body[0].(*ast.PauseNode)
	if pause.Clicks != 1 {
		t.Errorf("want clicks=1 (clamped from 0), got %d", pause.Clicks)
	}
}

// --- Affection with negative delta ---

func TestParseAffectionNegative(t *testing.T) {
	src := `@episode main:01 "T" {
		@affection mauricio -5
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	aff := ep.Body[0].(*ast.AffectionNode)
	if aff.Delta != "-5" {
		t.Errorf("want delta=-5, got %s", aff.Delta)
	}
}

// --- Gate with direct @next (no condition) ---

func TestParseGateDirectNext(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: Hi.
		@gate {
			@next main:02
		}
	}`
	ep := parseOrFail(t, src)
	if len(ep.Gate.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(ep.Gate.Routes))
	}
	if ep.Gate.Routes[0].Condition != nil {
		t.Error("expected nil condition for direct @next")
	}
}

// --- Multiple nodes in a single @if then block ---

func TestParseIfMultipleBodyNodes(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (EP01_COMPLETE) {
			@bg set classroom
			NARRATOR: Hello.
			@affection mauricio +1
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if len(ifNode.Then) != 3 {
		t.Errorf("want 3 then nodes, got %d", len(ifNode.Then))
	}
}

// --- Phone hide standalone ---

func TestParsePhoneHide(t *testing.T) {
	src := `@episode main:01 "T" {
		@phone hide
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	_, ok := ep.Body[0].(*ast.PhoneHideNode)
	if !ok {
		t.Fatalf("Body[0]: expected *PhoneHideNode, got %T", ep.Body[0])
	}
}

// --- Music fadeout standalone ---

func TestParseMusicFadeout(t *testing.T) {
	src := `@episode main:01 "T" {
		@music fadeout
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	_, ok := ep.Body[0].(*ast.MusicFadeoutNode)
	if !ok {
		t.Fatalf("Body[0]: expected *MusicFadeoutNode, got %T", ep.Body[0])
	}
}

// --- Standalone @if (no else) ---

func TestParseIfNoElse(t *testing.T) {
	src := `@episode main:01 "T" {
		@if (EP01_COMPLETE) {
			NARRATOR: Hi.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	if ifNode.Else != nil {
		t.Errorf("expected nil Else, got %v", ifNode.Else)
	}
}

// ---- @ending directive tests ----

func TestParseEndingComplete(t *testing.T) {
	src := `@episode main:15 "Finale" {
		NARRATOR: The end.
		@ending complete
	}`
	ep := parseOrFail(t, src)
	if ep.Ending == nil {
		t.Fatal("Ending: got nil, want *EndingNode")
	}
	if ep.Ending.Type != ast.EndingComplete {
		t.Errorf("Ending.Type: got %q, want %q", ep.Ending.Type, ast.EndingComplete)
	}
	if ep.Gate != nil {
		t.Errorf("Gate: got non-nil, want nil (episode with @ending)")
	}
}

func TestParseEndingToBeContinued(t *testing.T) {
	src := `@episode main:05 "Cliffhanger" {
		NARRATOR: To be continued.
		@ending to_be_continued
	}`
	ep := parseOrFail(t, src)
	if ep.Ending == nil || ep.Ending.Type != ast.EndingToBeContinued {
		t.Fatalf("Ending: got %+v, want to_be_continued", ep.Ending)
	}
}

func TestParseEndingBadEnding(t *testing.T) {
	src := `@episode main/bad/001:02 "Bad End" {
		NARRATOR: Game over.
		@ending bad_ending
	}`
	ep := parseOrFail(t, src)
	if ep.Ending == nil || ep.Ending.Type != ast.EndingBad {
		t.Fatalf("Ending: got %+v, want bad_ending", ep.Ending)
	}
}

func TestParseEndingInvalidType(t *testing.T) {
	src := `@episode main:01 "Bad" {
		@ending nope
	}`
	l := lexer.New(src)
	p := New(l)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error for invalid @ending type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid @ending type") {
		t.Errorf("error = %v, want one mentioning 'invalid @ending type'", err)
	}
}

func TestParseEndingRejectsGateCoexistence(t *testing.T) {
	src := `@episode main:01 "Mixed" {
		NARRATOR: Hi.
		@gate { @next main:02 }
		@ending complete
	}`
	l := lexer.New(src)
	p := New(l)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error when both @gate and @ending present, got nil")
	}
	if !strings.Contains(err.Error(), "cannot coexist") {
		t.Errorf("error = %v, want one mentioning 'cannot coexist'", err)
	}
}

func TestParseEndingRejectsDuplicate(t *testing.T) {
	src := `@episode main:01 "Dup" {
		NARRATOR: Hi.
		@ending complete
		@ending bad_ending
	}`
	l := lexer.New(src)
	p := New(l)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected parse error for duplicate @ending, got nil")
	}
}

// ---- Structured-condition AST tests ----

func TestParseConditionChoiceStructured(t *testing.T) {
	src := `@episode main:01 "X" {
		@if (A.fail) {
			NARRATOR: fail.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	c, ok := ifNode.Condition.(*ast.ChoiceCondition)
	if !ok {
		t.Fatalf("Condition: got %T, want *ChoiceCondition", ifNode.Condition)
	}
	if c.Option != "A" || c.Result != "fail" {
		t.Errorf("Choice: got %+v, want Option=A Result=fail", c)
	}
}

func TestParseConditionAffectionComparisonStructured(t *testing.T) {
	src := `@episode main:01 "X" {
		@if (affection.easton >= 5) {
			NARRATOR: yes.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	c, ok := ifNode.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Condition: got %T, want *ComparisonCondition", ifNode.Condition)
	}
	if c.Left.Kind != ast.OperandAffection || c.Left.Char != "easton" {
		t.Errorf("Left: got %+v, want affection/easton", c.Left)
	}
	if c.Op != ">=" || c.Right != 5 {
		t.Errorf("Op/Right: got %q/%d, want '>=' /5", c.Op, c.Right)
	}
}

func TestParseConditionValueComparisonStructured(t *testing.T) {
	src := `@episode main:01 "X" {
		@if (san <= 20) {
			NARRATOR: ouch.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	c, ok := ifNode.Condition.(*ast.ComparisonCondition)
	if !ok {
		t.Fatalf("Condition: got %T, want *ComparisonCondition", ifNode.Condition)
	}
	if c.Left.Kind != ast.OperandValue || c.Left.Name != "san" {
		t.Errorf("Left: got %+v, want value/san", c.Left)
	}
	if c.Op != "<=" || c.Right != 20 {
		t.Errorf("Op/Right: got %q/%d, want '<='/20", c.Op, c.Right)
	}
}

func TestParseConditionCompoundStructured(t *testing.T) {
	src := `@episode main:01 "X" {
		@if (affection.easton >= 5 && CHA >= 14) {
			NARRATOR: ok.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	c, ok := ifNode.Condition.(*ast.CompoundCondition)
	if !ok {
		t.Fatalf("Condition: got %T, want *CompoundCondition", ifNode.Condition)
	}
	if c.Op != "&&" {
		t.Errorf("Op: got %q, want &&", c.Op)
	}
	if _, ok := c.Left.(*ast.ComparisonCondition); !ok {
		t.Errorf("Left: got %T, want *ComparisonCondition", c.Left)
	}
	if _, ok := c.Right.(*ast.ComparisonCondition); !ok {
		t.Errorf("Right: got %T, want *ComparisonCondition", c.Right)
	}
}

func TestParseConditionPrecedence(t *testing.T) {
	// `a || b && c` should parse as `a || (b && c)` — && binds tighter than ||.
	src := `@episode main:01 "X" {
		@if (A_FLAG || B_FLAG && C_FLAG) {
			NARRATOR: ok.
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	ifNode := ep.Body[0].(*ast.IfNode)
	top, ok := ifNode.Condition.(*ast.CompoundCondition)
	if !ok || top.Op != "||" {
		t.Fatalf("top: got %T op=%s, want *CompoundCondition op=||",
			ifNode.Condition, conditionOp(ifNode.Condition))
	}
	if _, ok := top.Left.(*ast.FlagCondition); !ok {
		t.Errorf("Left: got %T, want *FlagCondition", top.Left)
	}
	right, ok := top.Right.(*ast.CompoundCondition)
	if !ok || right.Op != "&&" {
		t.Fatalf("Right: got %T op=%s, want *CompoundCondition op=&&",
			top.Right, conditionOp(top.Right))
	}
}

// conditionOp extracts a compound op for assertion diagnostics.
func conditionOp(c ast.Condition) string {
	if cc, ok := c.(*ast.CompoundCondition); ok {
		return cc.Op
	}
	return ""
}

// ---- @achievement directive + @signal kind tests ----

func TestParseAchievementBasic(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: Hi.
		@achievement HIGH_HEEL_WARRIOR {
			name: "【高跟鞋战士】"
			rarity: rare
			description: "用高跟鞋当武器，一次是即兴，签名招式从此诞生。"
			when: (HIGH_HEEL_EP05)
		}
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	if len(ep.Achievements) != 1 {
		t.Fatalf("Achievements: got %d, want 1", len(ep.Achievements))
	}
	a := ep.Achievements[0]
	if a.ID != "HIGH_HEEL_WARRIOR" {
		t.Errorf("ID: got %q", a.ID)
	}
	if a.Name != "【高跟鞋战士】" {
		t.Errorf("Name: got %q", a.Name)
	}
	if a.Rarity != ast.RarityRare {
		t.Errorf("Rarity: got %q, want %q", a.Rarity, ast.RarityRare)
	}
	if a.Description == "" {
		t.Error("Description: empty")
	}
	fc, ok := a.Trigger.(*ast.FlagCondition)
	if !ok || fc.Name != "HIGH_HEEL_EP05" {
		t.Errorf("Trigger: got %T %+v, want FlagCondition{Name:HIGH_HEEL_EP05}", a.Trigger, a.Trigger)
	}
}

func TestParseAchievementArcTrigger(t *testing.T) {
	src := `@episode main:24 "Arc" {
		NARRATOR: hi.
		@achievement HIGH_HEEL_DOUBLE_KILL {
			name: "【高跟鞋双杀】"
			rarity: epic
			description: "用高跟鞋当武器，一次是即兴，两次是签名招式。"
			when: (HIGH_HEEL_EP05 && HIGH_HEEL_EP24)
		}
		@ending complete
	}`
	ep := parseOrFail(t, src)
	a := ep.Achievements[0]
	cc, ok := a.Trigger.(*ast.CompoundCondition)
	if !ok || cc.Op != "&&" {
		t.Fatalf("Trigger: got %T, want CompoundCondition(&&)", a.Trigger)
	}
}

func TestParseAchievementRejectsInvalidRarity(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: hi.
		@achievement A1 {
			name: "x"
			rarity: common
			description: "y"
			when: (X)
		}
		@gate { @next main:02 }
	}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for rarity 'common'")
	}
	if !strings.Contains(err.Error(), "invalid rarity") {
		t.Errorf("err = %v, want one mentioning 'invalid rarity'", err)
	}
}

func TestParseAchievementRejectsMissingField(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: hi.
		@achievement A1 {
			name: "x"
			rarity: rare
			when: (X)
		}
		@gate { @next main:02 }
	}`
	_, err := New(lexer.New(src)).Parse()
	if err == nil {
		t.Fatal("expected parse error for missing description")
	}
}

func TestParseSignalAchievementKind(t *testing.T) {
	src := `@episode main:01 "T" {
		NARRATOR: hi.
		@signal achievement FIRST_KISS
		@gate { @next main:02 }
	}`
	ep := parseOrFail(t, src)
	sig := ep.Body[1].(*ast.SignalNode)
	if sig.Kind != ast.SignalKindAchievement {
		t.Errorf("Kind: got %q, want %q", sig.Kind, ast.SignalKindAchievement)
	}
	if sig.Event != "FIRST_KISS" {
		t.Errorf("Event: got %q", sig.Event)
	}
}
