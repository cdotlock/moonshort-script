package validator

import (
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

func TestValidGatesHasDefault(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "Hello."},
		},
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
			},
		},
	}

	errs := Validate(ep)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestMissingGates(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "Hello."},
		},
		Gates: nil,
	}

	errs := Validate(ep)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != MissingGates {
		t.Errorf("error code = %q, want %q", errs[0].Code, MissingGates)
	}
}

func TestGotoWithoutLabel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Test",
		Body: []ast.Node{
			&ast.GotoNode{Name: "MISSING_LABEL"},
		},
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
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
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
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
						// OnSuccess/OnFail both empty — should trigger BRAVE_MISSING_OUTCOME.
					},
				},
			},
		},
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
			},
		},
	}

	errs := Validate(ep)
	hasNoCheck := false
	hasMissingOutcome := false
	for _, e := range errs {
		if e.Code == BraveNoCheck {
			hasNoCheck = true
		}
		if e.Code == BraveMissingOutcome {
			hasMissingOutcome = true
		}
	}
	if !hasNoCheck {
		t.Errorf("expected BRAVE_NO_CHECK error, got %v", errs)
	}
	if !hasMissingOutcome {
		t.Errorf("expected BRAVE_MISSING_OUTCOME error, got %v", errs)
	}
}
