package parser

import (
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
	@gates {
		@default main:02
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

	// Gates
	if ep.Gates == nil {
		t.Fatal("Gates: expected non-nil")
	}
	if len(ep.Gates.Gates) != 1 {
		t.Fatalf("Gates count: got %d, want 1", len(ep.Gates.Gates))
	}
	def := ep.Gates.Gates[0]
	if def.GateType != "default" {
		t.Errorf("Gate.GateType: got %q, want %q", def.GateType, "default")
	}
	if def.Target != "main:02" {
		t.Errorf("Gate.Target: got %q, want %q", def.Target, "main:02")
	}
}

// TestParseCharDirectives tests show, expr, bubble, move, hide, and dialogue.
func TestParseCharDirectives(t *testing.T) {
	src := `@episode main:01 "Chars" {
	@mauricio show neutral at center fade
	@mauricio expr angry
	@mauricio bubble heart
	@mauricio move to left
	MAURICIO: Hey there.
	@mauricio hide fade
	@gates { @default main:02 }
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
	if show.Pose != "neutral" {
		t.Errorf("CharShowNode.Pose: got %q, want %q", show.Pose, "neutral")
	}
	if show.Position != "center" {
		t.Errorf("CharShowNode.Position: got %q, want %q", show.Position, "center")
	}
	if show.Transition != "fade" {
		t.Errorf("CharShowNode.Transition: got %q, want %q", show.Transition, "fade")
	}

	// expr
	expr, ok := ep.Body[1].(*ast.CharExprNode)
	if !ok {
		t.Fatalf("Body[1]: expected *CharExprNode, got %T", ep.Body[1])
	}
	if expr.Char != "mauricio" || expr.Pose != "angry" {
		t.Errorf("CharExprNode: got char=%q pose=%q", expr.Char, expr.Pose)
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
	@gates { @default main:02 }
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

// TestParseStateChanges tests xp, san, affection, signal, and butterfly.
func TestParseStateChanges(t *testing.T) {
	src := `@episode main:01 "State" {
	@xp +3
	@san -5
	@affection mauricio +2
	@signal EP01_COMPLETE
	@butterfly "Player chose kindness"
	@gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 5 {
		t.Fatalf("Body length: got %d, want 5", len(ep.Body))
	}

	xp, ok := ep.Body[0].(*ast.XpNode)
	if !ok {
		t.Fatalf("Body[0]: expected *XpNode, got %T", ep.Body[0])
	}
	if xp.Delta != "+3" {
		t.Errorf("XpNode.Delta: got %q, want %q", xp.Delta, "+3")
	}

	san, ok := ep.Body[1].(*ast.SanNode)
	if !ok {
		t.Fatalf("Body[1]: expected *SanNode, got %T", ep.Body[1])
	}
	if san.Delta != "-5" {
		t.Errorf("SanNode.Delta: got %q, want %q", san.Delta, "-5")
	}

	aff, ok := ep.Body[2].(*ast.AffectionNode)
	if !ok {
		t.Fatalf("Body[2]: expected *AffectionNode, got %T", ep.Body[2])
	}
	if aff.Char != "mauricio" || aff.Delta != "+2" {
		t.Errorf("AffectionNode: got char=%q delta=%q", aff.Char, aff.Delta)
	}

	sig, ok := ep.Body[3].(*ast.SignalNode)
	if !ok {
		t.Fatalf("Body[3]: expected *SignalNode, got %T", ep.Body[3])
	}
	if sig.Event != "EP01_COMPLETE" {
		t.Errorf("SignalNode.Event: got %q, want %q", sig.Event, "EP01_COMPLETE")
	}

	btf, ok := ep.Body[4].(*ast.ButterflyNode)
	if !ok {
		t.Fatalf("Body[4]: expected *ButterflyNode, got %T", ep.Body[4])
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
				@xp +5
			}
			@on fail {
				NARRATOR: The guard throws you back.
				@san -3
			}
		}
		@option B safe "Talk it out" {
			NARRATOR: You reason with the guard.
			@affection guard +1
		}
	}
	@gates { @default main:02 }
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
	if len(optA.OnSuccess) != 2 {
		t.Fatalf("OnSuccess length: got %d, want 2", len(optA.OnSuccess))
	}
	if len(optA.OnFail) != 2 {
		t.Fatalf("OnFail length: got %d, want 2", len(optA.OnFail))
	}

	// Verify success body content.
	succNarr, ok := optA.OnSuccess[0].(*ast.NarratorNode)
	if !ok {
		t.Fatalf("OnSuccess[0]: expected *NarratorNode, got %T", optA.OnSuccess[0])
	}
	if succNarr.Text != "You overpower the guard." {
		t.Errorf("OnSuccess narr: got %q", succNarr.Text)
	}
	succXp, ok := optA.OnSuccess[1].(*ast.XpNode)
	if !ok {
		t.Fatalf("OnSuccess[1]: expected *XpNode, got %T", optA.OnSuccess[1])
	}
	if succXp.Delta != "+5" {
		t.Errorf("OnSuccess xp: got %q", succXp.Delta)
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
			@xp +10
		}
		@on A B {
			NARRATOR: Good job.
			@xp +5
		}
		@on C D {
			NARRATOR: Could be better.
			@xp +1
		}
	}
	@gates { @default main:02 }
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
	@gates { @default main:02 }
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
	@if affection.easton >= 5 {
		NARRATOR: Easton smiles at you warmly.
		@affection easton +1
	}
	@else {
		NARRATOR: Easton barely notices you.
	}
	@gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 1 {
		t.Fatalf("Body length: got %d, want 1", len(ep.Body))
	}

	ifNode, ok := ep.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Body[0]: expected *IfNode, got %T", ep.Body[0])
	}
	if ifNode.Condition != "affection . easton >= 5" {
		t.Errorf("IfNode.Condition: got %q, want %q", ifNode.Condition, "affection . easton >= 5")
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

// TestParseDialogueWithExpr tests that CHARACTER [pose_expr]: text expands to
// CharExprNode + DialogueNode in the correct order.
func TestParseDialogueWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  MAURICIO [arms_crossed_angry]: Your call, Butterfly.
  @gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharExprNode + DialogueNode)", len(ep.Body))
	}

	// Body[0] must be CharExprNode.
	expr, ok := ep.Body[0].(*ast.CharExprNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharExprNode, got %T", ep.Body[0])
	}
	if expr.Char != "mauricio" {
		t.Errorf("CharExprNode.Char: got %q, want %q", expr.Char, "mauricio")
	}
	if expr.Pose != "arms_crossed_angry" {
		t.Errorf("CharExprNode.Pose: got %q, want %q", expr.Pose, "arms_crossed_angry")
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

// TestParseYouWithExpr tests that YOU [pose_expr]: text expands to CharExprNode + YouNode.
func TestParseYouWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  YOU [thinking]: Why is he looking at me like that?
  @gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharExprNode + YouNode)", len(ep.Body))
	}

	// Body[0] must be CharExprNode for "you".
	expr, ok := ep.Body[0].(*ast.CharExprNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharExprNode, got %T", ep.Body[0])
	}
	if expr.Char != "you" {
		t.Errorf("CharExprNode.Char: got %q, want %q", expr.Char, "you")
	}
	if expr.Pose != "thinking" {
		t.Errorf("CharExprNode.Pose: got %q, want %q", expr.Pose, "thinking")
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
// CharExprNode + NarratorNode.
func TestParseNarratorWithExpr(t *testing.T) {
	src := `@episode main:01 "Test" {
  NARRATOR [somber]: Three days passed without a word.
  @gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 2 {
		t.Fatalf("Body length: got %d, want 2 (CharExprNode + NarratorNode)", len(ep.Body))
	}

	expr, ok := ep.Body[0].(*ast.CharExprNode)
	if !ok {
		t.Fatalf("Body[0]: expected *CharExprNode, got %T", ep.Body[0])
	}
	if expr.Char != "narrator" {
		t.Errorf("CharExprNode.Char: got %q, want %q", expr.Char, "narrator")
	}
	if expr.Pose != "somber" {
		t.Errorf("CharExprNode.Pose: got %q, want %q", expr.Pose, "somber")
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
  @gates { @default main:02 }
}`
	ep := parseOrFail(t, src)

	if len(ep.Body) != 3 {
		t.Fatalf("Body length: got %d, want 3", len(ep.Body))
	}

	if _, ok := ep.Body[0].(*ast.CharExprNode); !ok {
		t.Fatalf("Body[0]: expected *CharExprNode, got %T", ep.Body[0])
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

// TestParseGates tests gates with choice gate, influence gate, and default.
func TestParseGates(t *testing.T) {
	src := `@episode main:01 "Gates" {
	NARRATOR: The story branches.
	@gates {
		@gate main/good/001:01 {
			type: choice
			trigger: option_A success
		}
		@gate main/bad/001:01 {
			type: influence
			condition: affection.easton >= 10
		}
		@default main:02
	}
}`
	ep := parseOrFail(t, src)

	if ep.Gates == nil {
		t.Fatal("Gates: expected non-nil")
	}
	if len(ep.Gates.Gates) != 3 {
		t.Fatalf("Gates count: got %d, want 3", len(ep.Gates.Gates))
	}

	// Choice gate.
	g0 := ep.Gates.Gates[0]
	if g0.Target != "main/good/001:01" {
		t.Errorf("Gate[0].Target: got %q, want %q", g0.Target, "main/good/001:01")
	}
	if g0.GateType != "choice" {
		t.Errorf("Gate[0].GateType: got %q, want %q", g0.GateType, "choice")
	}
	if g0.Trigger == nil {
		t.Fatal("Gate[0].Trigger: expected non-nil")
	}
	if g0.Trigger.OptionID != "A" {
		t.Errorf("Gate[0].Trigger.OptionID: got %q, want %q", g0.Trigger.OptionID, "A")
	}
	if g0.Trigger.CheckResult != "success" {
		t.Errorf("Gate[0].Trigger.CheckResult: got %q, want %q", g0.Trigger.CheckResult, "success")
	}

	// Influence gate.
	g1 := ep.Gates.Gates[1]
	if g1.Target != "main/bad/001:01" {
		t.Errorf("Gate[1].Target: got %q, want %q", g1.Target, "main/bad/001:01")
	}
	if g1.GateType != "influence" {
		t.Errorf("Gate[1].GateType: got %q, want %q", g1.GateType, "influence")
	}
	if g1.Condition != "affection . easton >= 10" {
		t.Errorf("Gate[1].Condition: got %q, want %q", g1.Condition, "affection . easton >= 10")
	}

	// Default gate.
	g2 := ep.Gates.Gates[2]
	if g2.GateType != "default" {
		t.Errorf("Gate[2].GateType: got %q, want %q", g2.GateType, "default")
	}
	if g2.Target != "main:02" {
		t.Errorf("Gate[2].Target: got %q, want %q", g2.Target, "main:02")
	}
}
