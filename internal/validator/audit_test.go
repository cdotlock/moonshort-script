package validator

import (
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// =============================================================================
// D. Validator whitelist edge cases
// =============================================================================

// D2. Transition "" (empty string) — should be VALID (means default/no transition).
// The code checks `if v.Transition != "" && !validTransitions[v.Transition]`.
// So empty string is allowed.
func TestEmptyTransition(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.CharShowNode{Char: "c", Look: "l", Transition: ""},
			&ast.BgSetNode{Name: "bg", Transition: ""},
		},
		Gate: unconditionalGate("main:02"),
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
		Gate: unconditionalGate("main:02"),
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
		Gate: unconditionalGate("main:02"),
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
		Gate: unconditionalGate("main:02"),
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

// =============================================================================
// E. Validator recursion completeness
// =============================================================================

// Container types still walked by the validator after the AST cleanup:
// ChoiceNode, IfNode, PhoneShowNode. CgShow no longer has a Body; the
// legacy "label/goto" tests are gone with the LabelNode/GotoNode removal.

// E5. Deeply nested: IfNode > ChoiceNode > brave Option > @if (check.success)
// branch containing a bad bubble type — exercises the full recursion chain.
func TestRecursion_DeeplyNested(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
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
										Then: []ast.Node{
											&ast.CharBubbleNode{Char: "c", BubbleType: "deep_invalid"},
										},
										Else: []ast.Node{&ast.NarratorNode{Text: "Fail."}},
									},
								},
							},
						},
					},
				},
			},
		},
		Gate: unconditionalGate("main:02"),
	}
	errs := Validate(ep)
	found := false
	for _, e := range errs {
		if e.Code == InvalidBubbleType && strings.Contains(e.Message, "deep_invalid") {
			found = true
		}
	}
	if !found {
		t.Error("checkValues should reach IfNode > ChoiceNode > brave Option body > @if (check.success).Then")
	}
}

// E6. Verify that checkValues handles TextMessageNode inside PhoneShowNode
// (the only legal child kind) without complaint.
func TestRecursion_PhoneShowWithTextMessages(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.PhoneShowNode{
				Body: []ast.Node{
					&ast.TextMessageNode{Direction: "from", Char: "c", Content: "hi"},
				},
			},
		},
		Gate: unconditionalGate("main:02"),
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("legal phone body should validate cleanly, got: %v", errs)
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
		Gate:      unconditionalGate("main:02"),
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
		Gate:      unconditionalGate("main:02"),
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
		Gate: unconditionalGate("main:02"),
	}
	errs := Validate(ep)
	// Should not panic — just no option-related errors.
	for _, e := range errs {
		if e.Code == DuplicateOptionID || e.Code == BraveNoCheck || e.Code == InvalidOptionMode {
			t.Errorf("empty choice should not produce option errors: %v", e)
		}
	}
}

// EDGE: MinigameNode in the new leaf shape — name + description, no body.
func TestValidatorMinigameLeafShape(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				Name:        "mg1",
				Description: "minigame description placeholder",
			},
		},
		Gate: unconditionalGate("main:02"),
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("leaf minigame should have no errors, got: %v", errs)
	}
}

// EDGE: A brave option whose body covers only the check.success branch
// (no @else for check.fail) is not a validation error — authors own the
// completeness of their @if tree.
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
		Gate: unconditionalGate("main:02"),
	}
	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("success-only body should validate cleanly, got %v", errs)
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
		Gate: unconditionalGate("main:02"),
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
		Gate: unconditionalGate("main:02"),
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
