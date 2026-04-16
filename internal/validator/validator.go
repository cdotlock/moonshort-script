// Package validator checks semantic correctness of a parsed NRS AST.
package validator

import (
	"fmt"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// Error codes for validation failures.
const (
	MissingGate         = "MISSING_GATE"
	GotoNoLabel         = "GOTO_NO_LABEL"
	BraveNoCheck        = "BRAVE_NO_CHECK"
	BraveMissingOutcome = "BRAVE_MISSING_OUTCOME"
)

// Error represents a single validation failure.
type Error struct {
	Code    string
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Validate checks semantic correctness of an Episode AST and returns all errors found.
func Validate(ep *ast.Episode) []Error {
	var errs []Error

	if ep.Gate == nil {
		errs = append(errs, Error{
			Code:    MissingGate,
			Message: "episode is missing a @gate block",
		})
	}

	// Collect all labels defined in the episode.
	labels := make(map[string]bool)
	collectLabels(ep.Body, labels)

	// 3. All @goto targets must have matching @label.
	checkGotos(ep.Body, labels, &errs)

	// 4 & 5. Brave options must have check block and both outcomes.
	checkBraveOptions(ep.Body, &errs)

	return errs
}

// collectLabels recursively finds all LabelNodes and records their names.
func collectLabels(nodes []ast.Node, labels map[string]bool) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.LabelNode:
			labels[v.Name] = true
		case *ast.CgShowNode:
			collectLabels(v.Body, labels)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				collectLabels(opt.OnSuccess, labels)
				collectLabels(opt.OnFail, labels)
				collectLabels(opt.Body, labels)
			}
		case *ast.IfNode:
			collectLabels(v.Then, labels)
			collectLabels(v.Else, labels)
		case *ast.MinigameNode:
			for _, nodes := range v.OnResult {
				collectLabels(nodes, labels)
			}
		case *ast.PhoneShowNode:
			collectLabels(v.Body, labels)
		}
	}
}

// checkGotos recursively validates that every GotoNode target has a matching label.
func checkGotos(nodes []ast.Node, labels map[string]bool, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.GotoNode:
			if !labels[v.Name] {
				*errs = append(*errs, Error{
					Code:    GotoNoLabel,
					Message: fmt.Sprintf("@goto %q has no matching @label", v.Name),
				})
			}
		case *ast.CgShowNode:
			checkGotos(v.Body, labels, errs)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				checkGotos(opt.OnSuccess, labels, errs)
				checkGotos(opt.OnFail, labels, errs)
				checkGotos(opt.Body, labels, errs)
			}
		case *ast.IfNode:
			checkGotos(v.Then, labels, errs)
			checkGotos(v.Else, labels, errs)
		case *ast.MinigameNode:
			for _, nodes := range v.OnResult {
				checkGotos(nodes, labels, errs)
			}
		case *ast.PhoneShowNode:
			checkGotos(v.Body, labels, errs)
		}
	}
}

// checkBraveOptions recursively validates brave option constraints.
func checkBraveOptions(nodes []ast.Node, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				if opt.Mode == "brave" {
					// 4. Brave options must have a check block.
					if opt.Check == nil {
						*errs = append(*errs, Error{
							Code:    BraveNoCheck,
							Message: fmt.Sprintf("brave option %q is missing a @check block", opt.ID),
						})
					}
					// 5. Brave options must have both on_success and on_fail.
					if len(opt.OnSuccess) == 0 || len(opt.OnFail) == 0 {
						*errs = append(*errs, Error{
							Code:    BraveMissingOutcome,
							Message: fmt.Sprintf("brave option %q must have both @on_success and @on_fail", opt.ID),
						})
					}
				}
				// Recurse into option bodies.
				checkBraveOptions(opt.OnSuccess, errs)
				checkBraveOptions(opt.OnFail, errs)
				checkBraveOptions(opt.Body, errs)
			}
		case *ast.CgShowNode:
			checkBraveOptions(v.Body, errs)
		case *ast.IfNode:
			checkBraveOptions(v.Then, errs)
			checkBraveOptions(v.Else, errs)
		case *ast.MinigameNode:
			for _, nodes := range v.OnResult {
				checkBraveOptions(nodes, errs)
			}
		case *ast.PhoneShowNode:
			checkBraveOptions(v.Body, errs)
		}
	}
}
