package validator

import (
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

func TestValidGateBlock(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "Hello."},
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
			},
		},
	}

	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestMissingGate(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "Hello."},
		},
		Gate: nil,
	}

	errs := Validate(ep)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != MissingTerminal {
		t.Errorf("error code = %q, want %q", errs[0].Code, MissingTerminal)
	}
}

func TestGotoWithoutLabel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.GotoNode{Name: "MISSING_LABEL"},
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
			},
		},
	}

	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected GOTO_NO_LABEL error, got %v", errs)
	}
}

func TestGotoWithLabel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.LabelNode{Name: "AFTER_FIGHT"},
			&ast.NarratorNode{Text: "Middle."},
			&ast.GotoNode{Name: "AFTER_FIGHT"},
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
			},
		},
	}

	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestBraveOptionWithoutCheck(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID:   "A",
						Mode: "brave",
						Text: "Confront him",
						// Check is nil — should trigger BRAVE_NO_CHECK.
					},
				},
			},
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
			},
		},
	}

	errs := Validate(ep)
	hasNoCheck := false
	for _, e := range errs {
		if e.Code == BraveNoCheck {
			hasNoCheck = true
		}
	}
	if !hasNoCheck {
		t.Errorf("expected BRAVE_NO_CHECK error, got %v", errs)
	}
}

func TestValidBraveOptionPass(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "Fight",
						Check: &ast.CheckBlock{Attr: "STR", DC: 14},
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then:      []ast.Node{&ast.NarratorNode{Text: "Win."}},
								Else:      []ast.Node{&ast.NarratorNode{Text: "Lose."}},
							},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestDuplicateOptionID(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{ID: "A", Mode: "safe", Text: "a"},
					{ID: "A", Mode: "safe", Text: "b"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == DuplicateOptionID {
			found = true
		}
	}
	if !found {
		t.Error("expected DUPLICATE_OPTION_ID error")
	}
}

func TestSafeOptionWithCheck(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "safe", Text: "a",
						Check: &ast.CheckBlock{Attr: "STR", DC: 10},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == SafeOptionHasCheck {
			found = true
		}
	}
	if !found {
		t.Error("expected SAFE_OPTION_HAS_CHECK error")
	}
}

func TestInvalidPosition(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharShowNode{Char: "c", Look: "l", Position: "centre"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidPosition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_POSITION error for 'centre'")
	}
}

func TestInvalidBubbleType(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharBubbleNode{Char: "c", BubbleType: "sparkle"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidBubbleType {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_BUBBLE_TYPE error")
	}
}

func TestInvalidOptionMode(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{ID: "A", Mode: "risky", Text: "a"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidOptionMode {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_OPTION_MODE error")
	}
}

func TestGotoInsideIfNode(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.LabelNode{Name: "L1"},
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "A"},
				Then:      []ast.Node{&ast.GotoNode{Name: "L1"}},
				Else:      []ast.Node{&ast.GotoNode{Name: "MISSING"}},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == GotoNoLabel && strings.Contains(e.Message, "MISSING") {
			found = true
		}
	}
	if !found {
		t.Error("expected GOTO_NO_LABEL for goto inside else branch")
	}
}

func TestGotoInsideCgShow(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "cg1",
				Duration: ast.CgDurationMedium,
				Content:  "cg content placeholder",
				Body:     []ast.Node{&ast.GotoNode{Name: "MISSING"}},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			found = true
		}
	}
	if !found {
		t.Error("expected GOTO_NO_LABEL for goto inside CgShowNode")
	}
}

func TestGotoInsideMinigame(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.RatingCondition{Grade: "S"},
						Then:      []ast.Node{&ast.GotoNode{Name: "MISSING"}},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			found = true
		}
	}
	if !found {
		t.Error("expected GOTO_NO_LABEL for goto inside MinigameNode")
	}
}

func TestGotoInsidePhoneShow(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.PhoneShowNode{
				Body: []ast.Node{&ast.GotoNode{Name: "MISSING"}},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			found = true
		}
	}
	if !found {
		t.Error("expected GOTO_NO_LABEL for goto inside PhoneShowNode")
	}
}

func TestLabelInsideCgShow(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "cg1",
				Duration: ast.CgDurationMedium,
				Content:  "cg content placeholder",
				Body:     []ast.Node{&ast.LabelNode{Name: "INNER"}},
			},
			&ast.GotoNode{Name: "INNER"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors (label inside cg should be collected), got %v", errs)
	}
}

func TestLabelInsideMinigame(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.RatingCondition{Grade: "S"},
						Then:      []ast.Node{&ast.LabelNode{Name: "MG_LABEL"}},
					},
				},
			},
			&ast.GotoNode{Name: "MG_LABEL"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestLabelInsidePhoneShow(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.PhoneShowNode{
				Body: []ast.Node{&ast.LabelNode{Name: "PHONE_LABEL"}},
			},
			&ast.GotoNode{Name: "PHONE_LABEL"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestLabelInsideIfNode(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "A"},
				Then:      []ast.Node{&ast.LabelNode{Name: "IF_LABEL"}},
				Else:      []ast.Node{&ast.LabelNode{Name: "ELSE_LABEL"}},
			},
			&ast.GotoNode{Name: "IF_LABEL"},
			&ast.GotoNode{Name: "ELSE_LABEL"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestLabelInsideChoiceOption(t *testing.T) {
	// Labels inside a brave option body (including inside the @if
	// check.success tree) must be collected so external gotos resolve.
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "Fight",
						Check: &ast.CheckBlock{Attr: "STR", DC: 14},
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then:      []ast.Node{&ast.LabelNode{Name: "SUCC_LABEL"}},
								Else:      []ast.Node{&ast.LabelNode{Name: "FAIL_LABEL"}},
							},
							&ast.LabelNode{Name: "BODY_LABEL"},
						},
					},
				},
			},
			&ast.GotoNode{Name: "SUCC_LABEL"},
			&ast.GotoNode{Name: "FAIL_LABEL"},
			&ast.GotoNode{Name: "BODY_LABEL"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestGotoInsideChoiceOption(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "Fight",
						Check: &ast.CheckBlock{Attr: "STR", DC: 14},
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then:      []ast.Node{&ast.GotoNode{Name: "MISSING1"}},
								Else:      []ast.Node{&ast.GotoNode{Name: "MISSING2"}},
							},
							&ast.GotoNode{Name: "MISSING3"},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	gotoCount := 0
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			gotoCount++
		}
	}
	if gotoCount != 3 {
		t.Errorf("expected 3 GOTO_NO_LABEL errors, got %d", gotoCount)
	}
}

func TestInvalidTransition(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "bg1", Transition: "wipe"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidTransition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_TRANSITION error for 'wipe'")
	}
}

func TestValidTransitions(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "bg1", Transition: "fade"},
			&ast.CharShowNode{Char: "c", Look: "l", Position: "center", Transition: "cut"},
			&ast.CharHideNode{Char: "c", Transition: "slow"},
			&ast.CharLookNode{Char: "c", Look: "l", Transition: "dissolve"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid transitions, got %v", errs)
	}
}

func TestInvalidTransitionOnCharHide(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharHideNode{Char: "c", Transition: "wipe"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidTransition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_TRANSITION for char hide")
	}
}

func TestInvalidTransitionOnCharLook(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharLookNode{Char: "c", Look: "l", Transition: "wipe"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidTransition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_TRANSITION for char look")
	}
}

func TestInvalidTransitionOnCgShow(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:       "cg1",
				Transition: "wipe",
				Duration:   ast.CgDurationMedium,
				Content:    "cg content placeholder",
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidTransition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_TRANSITION for cg show")
	}
}

func TestInvalidPositionOnCharMove(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharMoveNode{Char: "c", Position: "middle"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidPosition {
			found = true
		}
	}
	if !found {
		t.Error("expected INVALID_POSITION for char move")
	}
}

func TestValuesInsideNestedNodes(t *testing.T) {
	// Test checkValues recurses into CgShow, Choice, IfNode, Minigame, PhoneShow.
	// Brave-option success/fail branches are now inside the Body as an
	// @if (check.success) tree; we nest the bad bubble types in both branches.
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "cg1",
				Duration: ast.CgDurationMedium,
				Content:  "cg content placeholder",
				Body:     []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "oops"}},
			},
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "a",
						Check: &ast.CheckBlock{Attr: "STR", DC: 10},
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then:      []ast.Node{&ast.CharBubbleNode{Char: "c", BubbleType: "invalid1"}},
								Else:      []ast.Node{&ast.CharBubbleNode{Char: "c", BubbleType: "invalid2"}},
							},
							&ast.CharShowNode{Char: "c", Look: "l", Position: "oops2"},
						},
					},
				},
			},
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "A"},
				Then:      []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "oops3"}},
				Else:      []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "oops4"}},
			},
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.RatingCondition{Grade: "S"},
						Then:      []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "oops5"}},
					},
				},
			},
			&ast.PhoneShowNode{
				Body: []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "oops6"}},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	positionErrors := 0
	bubbleErrors := 0
	for _, e := range errs {
		if e.Code == InvalidPosition {
			positionErrors++
		}
		if e.Code == InvalidBubbleType {
			bubbleErrors++
		}
	}
	if positionErrors < 6 {
		t.Errorf("expected at least 6 INVALID_POSITION errors from nested nodes, got %d", positionErrors)
	}
	if bubbleErrors < 2 {
		t.Errorf("expected at least 2 INVALID_BUBBLE_TYPE errors from nested nodes, got %d", bubbleErrors)
	}
}

func TestBraveOptionsInsideNestedNodes(t *testing.T) {
	// Test checkBraveOptions recurses into CgShow, IfNode, Minigame, PhoneShow.
	// Each of the 5 brave options below is missing its check block, so
	// we expect BRAVE_NO_CHECK to fire 5 times.
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "cg1",
				Duration: ast.CgDurationMedium,
				Content:  "cg content placeholder",
				Body: []ast.Node{
					&ast.ChoiceNode{
						Options: []*ast.OptionNode{
							{ID: "A", Mode: "brave", Text: "a"},
						},
					},
				},
			},
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "A"},
				Then: []ast.Node{
					&ast.ChoiceNode{
						Options: []*ast.OptionNode{
							{ID: "B", Mode: "brave", Text: "b"},
						},
					},
				},
				Else: []ast.Node{
					&ast.ChoiceNode{
						Options: []*ast.OptionNode{
							{ID: "C", Mode: "brave", Text: "c"},
						},
					},
				},
			},
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.RatingCondition{Grade: "S"},
						Then: []ast.Node{
							&ast.ChoiceNode{
								Options: []*ast.OptionNode{
									{ID: "D", Mode: "brave", Text: "d"},
								},
							},
						},
					},
				},
			},
			&ast.PhoneShowNode{
				Body: []ast.Node{
					&ast.ChoiceNode{
						Options: []*ast.OptionNode{
							{ID: "E", Mode: "brave", Text: "e"},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	braveErrors := 0
	for _, e := range errs {
		if e.Code == BraveNoCheck {
			braveErrors++
		}
	}
	// 5 brave options without check → 5 BRAVE_NO_CHECK errors.
	if braveErrors < 5 {
		t.Errorf("expected at least 5 BRAVE_NO_CHECK errors from nested nodes, got %d", braveErrors)
	}
}

func TestValidatorErrorMethod(t *testing.T) {
	e := Error{Code: "TEST_CODE", Message: "test message"}
	s := e.Error()
	if !strings.Contains(s, "TEST_CODE") || !strings.Contains(s, "test message") {
		t.Errorf("Error() = %q, expected code and message", s)
	}
}

// TestSafeOptionWithCheckBlock verifies that a safe option with a check
// block triggers SAFE_OPTION_HAS_CHECK — check is only meaningful for
// brave options.
func TestSafeOptionWithCheckBlock(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "safe", Text: "a",
						Check: &ast.CheckBlock{Attr: "CHA", DC: 10},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == SafeOptionHasCheck {
			found = true
		}
	}
	if !found {
		t.Error("expected SAFE_OPTION_HAS_CHECK error for safe option with a check block")
	}
}

// TestBraveOptionWithCheckPasses verifies that a brave option with a
// check block and no explicit outcome body passes validation — outcome
// branching is a narrative choice the author makes via @if (check.success),
// not something the validator enforces.
func TestBraveOptionWithCheckPasses(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "a",
						Check: &ast.CheckBlock{Attr: "STR", DC: 14},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// TestValidEndingSatisfiesTerminal — an @ending replaces the need for @gate.
func TestValidEndingSatisfiesTerminal(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:15",
		Title:     "Finale",
		Body:      []ast.Node{&ast.NarratorNode{Text: "End."}},
		Ending:    &ast.EndingNode{Type: ast.EndingComplete},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestMissingTerminalErrors(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
	}
	errs := Validate(ep)
	if len(errs) == 0 {
		t.Fatal("expected MISSING_TERMINAL error, got none")
	}
	if errs[0].Code != MissingTerminal {
		t.Errorf("Code: got %q, want %q", errs[0].Code, MissingTerminal)
	}
}

func TestInvalidEndingType(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Ending:    &ast.EndingNode{Type: "nope"},
	}
	errs := Validate(ep)
	var found bool
	for _, e := range errs {
		if e.Code == InvalidEndingType {
			found = true
		}
	}
	if !found {
		t.Errorf("expected INVALID_ENDING_TYPE in %v", errs)
	}
}

func TestInvalidCompoundConditionOp(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.CompoundCondition{
					Op:    "xor", // invalid
					Left:  &ast.FlagCondition{Name: "A"},
					Right: &ast.FlagCondition{Name: "B"},
				},
				Then: []ast.Node{&ast.NarratorNode{Text: "x"}},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	var found bool
	for _, e := range errs {
		if e.Code == InvalidCondition && strings.Contains(e.Message, `"xor"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected INVALID_CONDITION for invalid compound op, got %v", errs)
	}
}

// TestAchievementIdDuplicationAllowed pins the decision that two inline
// @achievement steps sharing the same id are valid — engines dedup at
// unlock time, and authors may deliberately echo a single unlock point
// from several narrative branches.
func TestAchievementIdDuplicationAllowed(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.AchievementNode{ID: "A", Name: "x", Rarity: ast.RarityRare, Description: "d"},
			&ast.AchievementNode{ID: "A", Name: "x", Rarity: ast.RarityRare, Description: "d"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("duplicate ids should validate cleanly, got %v", errs)
	}
}

func TestInvalidRarityValidation(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.AchievementNode{ID: "A", Name: "n", Rarity: "common", Description: "d"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidRarity {
			found = true
		}
	}
	if !found {
		t.Errorf("expected INVALID_RARITY, got %v", errs)
	}
}

func TestInvalidSignalKindValidation(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{&ast.SignalNode{Kind: "garbage", Event: "X"}},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidSignalKind {
			found = true
		}
	}
	if !found {
		t.Errorf("expected INVALID_SIGNAL_KIND, got %v", errs)
	}
}

func TestValidateSignalIntReservedName(t *testing.T) {
	// @signal int san should fail — "san" is an engine-reserved name.
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "t",
		Body: []ast.Node{
			&ast.SignalNode{Kind: ast.SignalKindInt, Name: "san", Op: ast.SignalOpAssign, Value: 0},
		},
		Ending: &ast.EndingNode{Type: ast.EndingComplete},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == ReservedIntName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ReservedIntName error, got: %v", errs)
	}
}

func TestValidateSignalIntOK(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "t",
		Body: []ast.Node{
			&ast.SignalNode{Kind: ast.SignalKindInt, Name: "rejections", Op: ast.SignalOpAssign, Value: 0},
			&ast.SignalNode{Kind: ast.SignalKindInt, Name: "rejections", Op: ast.SignalOpAdd, Value: 1},
		},
		Ending: &ast.EndingNode{Type: ast.EndingComplete},
	}
	errs := Validate(ep)
	for _, e := range errs {
		if e.Code == ReservedIntName || e.Code == InvalidSignalKind {
			t.Fatalf("unexpected error: %v", e)
		}
	}
}
