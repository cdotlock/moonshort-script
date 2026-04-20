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
			&ast.CgShowNode{
				Name:       "cg",
				Transition: "",
				Duration:   ast.CgDurationMedium,
				Content:    "cg content placeholder",
			},
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

// E2. Labels and gotos throughout the option body (including inside the
// @if (check.success) / @else tree) must all be discoverable. The
// OptionNode exposes a single Body slice, so the walk has to descend
// through IfNode.Then/Else to find everything.
func TestRecursion_LabelsInAllOptionFields(t *testing.T) {
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
								Then:      []ast.Node{&ast.LabelNode{Name: "L1"}},
								Else:      []ast.Node{&ast.LabelNode{Name: "L2"}},
							},
							&ast.LabelNode{Name: "L3"},
						},
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

// E3. checkValues must recurse into MinigameNode.Body — rating branching
// lives inside an @if (rating.X) tree in the plain body.
func TestRecursion_ValuesInsideMinigame(t *testing.T) {
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
						Then:      []ast.Node{&ast.CharShowNode{Char: "c", Look: "l", Position: "invalid_pos"}},
					},
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
		t.Error("checkValues should recurse into MinigameNode.Body (through IfNode.Then)")
	}
}

// E4. checkBraveOptions must recurse into MinigameNode.Body (through
// nested IfNode.Then/Else for rating-branch trees).
func TestRecursion_BraveOptionsInsideMinigame(t *testing.T) {
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
						Then: []ast.Node{
							&ast.ChoiceNode{
								Options: []*ast.OptionNode{
									{ID: "A", Mode: "brave", Text: "a"},
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
	found := false
	for _, e := range errs {
		if e.Code == BraveNoCheck {
			found = true
		}
	}
	if !found {
		t.Error("checkBraveOptions should recurse into MinigameNode.Body (through IfNode.Then)")
	}
}

// E5. Deeply nested: CgShow > IfNode > ChoiceNode > brave Option with a
// goto inside the check.success branch of its inner @if tree.
func TestRecursion_DeeplyNested(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "cg1",
				Duration: ast.CgDurationMedium,
				Content:  "cg content placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.FlagCondition{Name: "A"},
						Then: []ast.Node{
							&ast.ChoiceNode{
								Options: []*ast.OptionNode{
									{
										ID: "A", Mode: "brave", Text: "Fight",
										Check: &ast.CheckBlock{Attr: "STR", DC: 14},
										Body: []ast.Node{
											&ast.IfNode{
												Condition: &ast.CheckCondition{Result: "success"},
												Then:      []ast.Node{&ast.GotoNode{Name: "DEEP_MISSING"}},
												Else:      []ast.Node{&ast.NarratorNode{Text: "Fail."}},
											},
										},
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
		t.Error("checkGotos should find DEEP_MISSING in CgShow > IfNode > ChoiceNode > brave Option body > @if (check.success).Then")
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

// EDGE: MinigameNode with empty body — rating branching is optional, so
// an empty body is valid and must not raise a validator error.
func TestValidatorEmptyMinigameBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body:        []ast.Node{},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("empty minigame body should have no errors, got: %v", errs)
	}
}

// EDGE: MinigameNode with nil body — the walk must not panic on a nil slice.
func TestValidatorNilMinigameBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:          "mg1",
				Attr:        "STR",
				Description: "minigame description placeholder",
				Body:        nil,
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("nil minigame body should have no errors, got: %v", errs)
	}
}

// EDGE: A brave option whose body covers only the check.success branch
// (no @else for check.fail) is not a validation error — authors own the
// completeness of their @if tree. This test pins that the validator
// never emits BRAVE_MISSING_OUTCOME.
func TestBraveOptionSuccessOnlyBodyIsValid(t *testing.T) {
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
								// No Else — author deliberately left fail implicit.
							},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	errs := Validate(ep)
	for _, e := range errs {
		if e.Code == BraveMissingOutcome {
			t.Errorf("BRAVE_MISSING_OUTCOME should no longer be emitted: %v", e)
		}
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
