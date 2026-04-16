package validator

import (
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// =============================================================================
// D. Validator whitelist edge cases
// =============================================================================

// D1. Position "" (empty string) — should be flagged as invalid.
func TestEmptyPosition(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharShowNode{Char: "c", Look: "l", Position: ""},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidPosition {
			found = true
			if !strings.Contains(e.Message, `""`) {
				t.Errorf("error message should include the invalid empty value in quotes, got: %s", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected INVALID_POSITION error for empty position")
	}
}

// D2. Transition "" (empty string) — should be VALID (means default/no transition).
// The code checks `if v.Transition != "" && !validTransitions[v.Transition]`.
// So empty string is allowed. Let's verify.
func TestEmptyTransition(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharShowNode{Char: "c", Look: "l", Position: "center", Transition: ""},
			&ast.BgSetNode{Name: "bg", Transition: ""},
			&ast.CgShowNode{Name: "cg", Transition: ""},
			&ast.CharHideNode{Char: "c", Transition: ""},
			&ast.CharLookNode{Char: "c", Look: "l", Transition: ""},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	for _, e := range errs {
		if e.Code == InvalidTransition {
			t.Errorf("empty transition should be valid, got error: %v", e)
		}
	}
}

// D3. Bubble type not in the list — error message should include the invalid value.
func TestInvalidBubbleTypeMessage(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharBubbleNode{Char: "test", BubbleType: "sparkle_invalid"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidBubbleType {
			found = true
			if !strings.Contains(e.Message, "sparkle_invalid") {
				t.Errorf("error message should include the invalid value 'sparkle_invalid', got: %s", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected INVALID_BUBBLE_TYPE error")
	}
}

// D4. Option mode "" (empty) — should be caught as invalid.
// The validator checks "brave", "safe", and else → InvalidOptionMode.
// Empty string falls into else.
func TestEmptyOptionMode(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{ID: "A", Mode: "", Text: "test"},
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
			if !strings.Contains(e.Message, `""`) {
				t.Errorf("error message should include the empty mode value in quotes, got: %s", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected INVALID_OPTION_MODE error for empty mode")
	}
}

// D5. Empty bubble type — should be flagged.
func TestEmptyBubbleType(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharBubbleNode{Char: "c", BubbleType: ""},
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
		t.Error("expected INVALID_BUBBLE_TYPE error for empty bubble type")
	}
}

// D6. Empty position on CharMoveNode.
func TestEmptyPositionOnMove(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharMoveNode{Char: "c", Position: ""},
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
		t.Error("expected INVALID_POSITION error for empty move position")
	}
}

// =============================================================================
// E. Validator recursion completeness
// =============================================================================

// The 4 recursive walks: collectLabels, checkGotos, checkBraveOptions, checkValues.
// All four must recurse into the same container node types.
// Container types: CgShowNode, ChoiceNode, IfNode, MinigameNode, PhoneShowNode.
//
// Let's verify each walk handles all container types by placing a relevant node
// deep inside each container type.

// E1. Test that all walks handle GateBlock correctly.
// GateBlock is NOT part of Episode.Body — it's a separate field.
// None of the walks recurse into GateBlock, which is correct since
// Gate routes don't contain body nodes.

// E2. Test that labels and gotos inside all option sub-fields are found.
// OptionNode has: OnSuccess, OnFail, Body — all three should be walked.
func TestRecursion_LabelsInAllOptionFields(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID: "A", Mode: "brave", Text: "Fight",
						Check:     &ast.CheckBlock{Attr: "STR", DC: 14},
						OnSuccess: []ast.Node{&ast.LabelNode{Name: "L1"}},
						OnFail:    []ast.Node{&ast.LabelNode{Name: "L2"}},
						Body:      []ast.Node{&ast.LabelNode{Name: "L3"}},
					},
				},
			},
			// Gotos referencing all three labels
			&ast.GotoNode{Name: "L1"},
			&ast.GotoNode{Name: "L2"},
			&ast.GotoNode{Name: "L3"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	for _, e := range errs {
		if e.Code == GotoNoLabel {
			t.Errorf("all labels in option fields should be found: %v", e)
		}
	}
}

// E3. Test that checkValues recurses into MinigameNode.OnResult.
func TestRecursion_ValuesInsideMinigame(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:   "mg1",
				Attr: "STR",
				OnResult: map[string][]ast.Node{
					"S": {&ast.CharShowNode{Char: "c", Look: "l", Position: "invalid_pos"}},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidPosition && strings.Contains(e.Message, "invalid_pos") {
			found = true
		}
	}
	if !found {
		t.Error("checkValues should recurse into MinigameNode.OnResult")
	}
}

// E4. Test that checkBraveOptions recurses into MinigameNode.OnResult.
func TestRecursion_BraveOptionsInsideMinigame(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:   "mg1",
				Attr: "STR",
				OnResult: map[string][]ast.Node{
					"S": {
						&ast.ChoiceNode{
							Options: []*ast.OptionNode{
								{ID: "A", Mode: "brave", Text: "a"},
							},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == BraveNoCheck {
			found = true
		}
	}
	if !found {
		t.Error("checkBraveOptions should recurse into MinigameNode.OnResult")
	}
}

// E5. Test deeply nested: CgShow > IfNode > ChoiceNode > Option with goto to missing label.
func TestRecursion_DeeplyNested(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name: "cg1",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.Condition{Type: "flag", Name: "A"},
						Then: []ast.Node{
							&ast.ChoiceNode{
								Options: []*ast.OptionNode{
									{
										ID: "A", Mode: "brave", Text: "Fight",
										Check:     &ast.CheckBlock{Attr: "STR", DC: 14},
										OnSuccess: []ast.Node{&ast.GotoNode{Name: "DEEP_MISSING"}},
										OnFail:    []ast.Node{&ast.NarratorNode{Text: "Fail."}},
									},
								},
							},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	foundGoto := false
	for _, e := range errs {
		if e.Code == GotoNoLabel && strings.Contains(e.Message, "DEEP_MISSING") {
			foundGoto = true
		}
	}
	if !foundGoto {
		t.Error("checkGotos should find DEEP_MISSING in CgShow > IfNode > ChoiceNode > Option.OnSuccess")
	}
}

// E6. Verify that checkValues handles CharShowNode inside PhoneShowNode.
func TestRecursion_CharShowInsidePhone(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.PhoneShowNode{
				Body: []ast.Node{
					&ast.CharShowNode{Char: "c", Look: "l", Position: "bogus"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidPosition && strings.Contains(e.Message, "bogus") {
			found = true
		}
	}
	if !found {
		t.Error("checkValues should recurse into PhoneShowNode body")
	}
}

// =============================================================================
// Additional validator edge cases
// =============================================================================

// EDGE: Empty Episode body — should only report missing gate if gate is nil.
func TestValidatorEmptyBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body:      []ast.Node{},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("empty body with gate should have no errors, got: %v", errs)
	}
}

// EDGE: Nil body (not just empty).
func TestValidatorNilBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body:      nil,
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("nil body with gate should have no errors, got: %v", errs)
	}
}

// EDGE: ChoiceNode with zero options.
func TestValidatorEmptyChoice(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{Options: []*ast.OptionNode{}},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	// Should not panic — just no option-related errors.
	for _, e := range errs {
		if e.Code == DuplicateOptionID || e.Code == BraveNoCheck || e.Code == InvalidOptionMode {
			t.Errorf("empty choice should not produce option errors: %v", e)
		}
	}
}

// EDGE: MinigameNode with empty OnResult map.
func TestValidatorEmptyMinigameResults(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:       "mg1",
				Attr:     "STR",
				OnResult: map[string][]ast.Node{},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	// Should not panic.
	if len(errs) != 0 {
		t.Errorf("empty minigame results should have no errors, got: %v", errs)
	}
}

// EDGE: MinigameNode with nil OnResult map.
func TestValidatorNilMinigameResults(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:       "mg1",
				Attr:     "STR",
				OnResult: nil,
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	// Should not panic when iterating nil map.
	if len(errs) != 0 {
		t.Errorf("nil minigame results should have no errors, got: %v", errs)
	}
}

// EDGE: Brave option with OnSuccess populated but OnFail empty.
func TestBraveOptionPartialOutcomes(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID:        "A",
						Mode:      "brave",
						Text:      "Fight",
						Check:     &ast.CheckBlock{Attr: "STR", DC: 14},
						OnSuccess: []ast.Node{&ast.NarratorNode{Text: "Win."}},
						OnFail:    []ast.Node{}, // empty, not nil
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == BraveMissingOutcome {
			found = true
		}
	}
	if !found {
		t.Error("brave option with empty OnFail should trigger BRAVE_MISSING_OUTCOME")
	}
}

// EDGE: Safe option mode — verify case sensitivity. "Safe" != "safe".
func TestOptionModeCaseSensitive(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{ID: "A", Mode: "Safe", Text: "a"},
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
		t.Error("'Safe' (capitalized) should be invalid mode — must be lowercase 'safe'")
	}
}

// EDGE: Transition "Fade" (capitalized) — should be invalid.
func TestTransitionCaseSensitive(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "bg", Transition: "Fade"},
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
		t.Error("'Fade' (capitalized) should be invalid transition — must be lowercase 'fade'")
	}
}

// EDGE: Position "Center" (capitalized) — should be invalid.
func TestPositionCaseSensitive(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharShowNode{Char: "c", Look: "l", Position: "Center"},
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
		t.Error("'Center' (capitalized) should be invalid position — must be lowercase 'center'")
	}
}
