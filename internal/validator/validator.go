// Package validator checks semantic correctness of a parsed MSS AST.
package validator

import (
	"fmt"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// Error codes for validation failures.
const (
	MissingTerminal     = "MISSING_TERMINAL"
	GotoNoLabel         = "GOTO_NO_LABEL"
	BraveNoCheck        = "BRAVE_NO_CHECK"
	// BraveMissingOutcome is retained as a constant for backward compatibility
	// with downstream code; the validator no longer emits this code because
	// the new grammar expresses success/fail branching via @if (check.success)
	// / @else trees and the completeness cannot be statically verified without
	// control-flow analysis.
	BraveMissingOutcome = "BRAVE_MISSING_OUTCOME"
	DuplicateOptionID   = "DUPLICATE_OPTION_ID"
	SafeOptionHasCheck  = "SAFE_OPTION_HAS_CHECK"
	InvalidPosition     = "INVALID_POSITION"
	InvalidTransition   = "INVALID_TRANSITION"
	InvalidBubbleType   = "INVALID_BUBBLE_TYPE"
	InvalidOptionMode   = "INVALID_OPTION_MODE"
	InvalidEndingType   = "INVALID_ENDING_TYPE"
	InvalidCondition    = "INVALID_CONDITION"
	InvalidSignalKind   = "INVALID_SIGNAL_KIND"
	InvalidRarity       = "INVALID_RARITY"
	DuplicateAchievement = "DUPLICATE_ACHIEVEMENT_ID"
	AchievementMissingField = "ACHIEVEMENT_MISSING_FIELD"
	InvalidCgDuration = "INVALID_CG_DURATION"
	CgMissingContent  = "CG_MISSING_CONTENT"
	MinigameMissingDescription = "MINIGAME_MISSING_DESCRIPTION"
)

var validSignalKinds = map[string]bool{
	ast.SignalKindMark: true,
}

var validCgDurations = map[string]bool{
	ast.CgDurationLow:    true,
	ast.CgDurationMedium: true,
	ast.CgDurationHigh:   true,
}

var validRarities = map[string]bool{
	ast.RarityUncommon:  true,
	ast.RarityRare:      true,
	ast.RarityEpic:      true,
	ast.RarityLegendary: true,
}

// MissingGate is kept as an alias for backward compatibility of external
// tooling that references the old code. Prefer MissingTerminal.
const MissingGate = MissingTerminal

var validEndingTypes = map[string]bool{
	ast.EndingComplete:      true,
	ast.EndingToBeContinued: true,
	ast.EndingBad:           true,
}

var validPositions = map[string]bool{
	"left": true, "center": true, "right": true,
	"left_far": true, "right_far": true,
}

var validTransitions = map[string]bool{
	"fade": true, "cut": true, "slow": true, "dissolve": true,
}

var validBubbleTypes = map[string]bool{
	"anger": true, "sweat": true, "heart": true, "question": true,
	"exclaim": true, "idea": true, "music": true, "doom": true,
	"ellipsis": true,
}

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

	// Episode must terminate with exactly one of @gate or @ending.
	if ep.Gate == nil && ep.Ending == nil {
		errs = append(errs, Error{
			Code:    MissingTerminal,
			Message: "episode is missing a terminal: add @gate { ... } for routing or @ending <type> for a terminal state",
		})
	}

	// Validate @ending type if present.
	if ep.Ending != nil && !validEndingTypes[ep.Ending.Type] {
		errs = append(errs, Error{
			Code:    InvalidEndingType,
			Message: fmt.Sprintf("invalid @ending type %q (must be one of: complete, to_be_continued, bad_ending)", ep.Ending.Type),
		})
	}

	// Collect all labels defined in the episode.
	labels := make(map[string]bool)
	collectLabels(ep.Body, labels)

	// 3. All @goto targets must have matching @label.
	checkGotos(ep.Body, labels, &errs)

	// 4 & 5. Brave options must have check block and both outcomes.
	checkBraveOptions(ep.Body, &errs)

	// Check value whitelists.
	checkValues(ep.Body, &errs)

	// Validate condition trees (body @if and @gate routes).
	checkConditions(ep.Body, &errs)
	if ep.Gate != nil {
		for _, route := range ep.Gate.Routes {
			if route.Condition != nil {
				checkCondition(route.Condition, &errs)
			}
		}
	}

	// Validate signal kinds (recursive over whole body).
	checkSignals(ep.Body, &errs)

	// Validate achievements.
	checkAchievements(ep.Achievements, &errs)

	return errs
}

// checkSignals walks the body and ensures every SignalNode has a valid kind.
func checkSignals(nodes []ast.Node, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.SignalNode:
			if !validSignalKinds[v.Kind] {
				*errs = append(*errs, Error{
					Code:    InvalidSignalKind,
					Message: fmt.Sprintf("@signal %q has invalid kind %q (only 'mark' is currently supported)", v.Event, v.Kind),
				})
			}
		case *ast.CgShowNode:
			checkSignals(v.Body, errs)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				checkSignals(opt.Body, errs)
			}
		case *ast.IfNode:
			checkSignals(v.Then, errs)
			checkSignals(v.Else, errs)
		case *ast.MinigameNode:
			checkSignals(v.Body, errs)
		case *ast.PhoneShowNode:
			checkSignals(v.Body, errs)
		}
	}
}

// checkAchievements validates rarity whitelist, required fields, duplicate ids.
// Triggering is a separate step (AchievementTriggerNode); declarations just
// carry metadata.
func checkAchievements(list []*ast.AchievementNode, errs *[]Error) {
	seen := map[string]bool{}
	for _, a := range list {
		if a.ID == "" {
			*errs = append(*errs, Error{
				Code:    AchievementMissingField,
				Message: "achievement has empty id",
			})
			continue
		}
		if seen[a.ID] {
			*errs = append(*errs, Error{
				Code:    DuplicateAchievement,
				Message: fmt.Sprintf("duplicate @achievement id %q", a.ID),
			})
		}
		seen[a.ID] = true
		if a.Name == "" {
			*errs = append(*errs, Error{
				Code:    AchievementMissingField,
				Message: fmt.Sprintf("achievement %q missing 'name'", a.ID),
			})
		}
		if a.Description == "" {
			*errs = append(*errs, Error{
				Code:    AchievementMissingField,
				Message: fmt.Sprintf("achievement %q missing 'description'", a.ID),
			})
		}
		if !validRarities[a.Rarity] {
			*errs = append(*errs, Error{
				Code:    InvalidRarity,
				Message: fmt.Sprintf("achievement %q has invalid rarity %q (must be uncommon, rare, epic, or legendary)", a.ID, a.Rarity),
			})
		}
	}
}

// validCompoundOps and validComparisonOps enumerate accepted operators in
// structured condition nodes.
var validCompoundOps = map[string]bool{"&&": true, "||": true}
var validComparisonOps = map[string]bool{
	">=": true, "<=": true, ">": true, "<": true, "==": true, "!=": true,
}
var validChoiceResults = map[string]bool{"success": true, "fail": true, "any": true}

// checkConditions walks nodes and validates every Condition tree it finds.
func checkConditions(nodes []ast.Node, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.IfNode:
			if v.Condition != nil {
				checkCondition(v.Condition, errs)
			}
			checkConditions(v.Then, errs)
			checkConditions(v.Else, errs)
		case *ast.CgShowNode:
			checkConditions(v.Body, errs)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				checkConditions(opt.Body, errs)
			}
		case *ast.MinigameNode:
			checkConditions(v.Body, errs)
		case *ast.PhoneShowNode:
			checkConditions(v.Body, errs)
		}
	}
}

// checkCondition validates a single Condition AST node recursively.
func checkCondition(c ast.Condition, errs *[]Error) {
	switch v := c.(type) {
	case *ast.ChoiceCondition:
		if !validChoiceResults[v.Result] {
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: fmt.Sprintf("choice condition %s.%s has invalid result (must be success, fail, or any)", v.Option, v.Result),
			})
		}
	case *ast.ComparisonCondition:
		if !validComparisonOps[v.Op] {
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: fmt.Sprintf("comparison has invalid operator %q", v.Op),
			})
		}
		switch v.Left.Kind {
		case ast.OperandAffection:
			if v.Left.Char == "" {
				*errs = append(*errs, Error{
					Code:    InvalidCondition,
					Message: "affection operand has empty character name",
				})
			}
		case ast.OperandValue:
			if v.Left.Name == "" {
				*errs = append(*errs, Error{
					Code:    InvalidCondition,
					Message: "value operand has empty name",
				})
			}
		default:
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: fmt.Sprintf("comparison has unknown operand kind %q", v.Left.Kind),
			})
		}
	case *ast.CompoundCondition:
		if !validCompoundOps[v.Op] {
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: fmt.Sprintf("compound condition has invalid operator %q", v.Op),
			})
		}
		if v.Left != nil {
			checkCondition(v.Left, errs)
		}
		if v.Right != nil {
			checkCondition(v.Right, errs)
		}
	case *ast.FlagCondition, *ast.InfluenceCondition:
		// No further structural checks — any non-empty string is fine.
	case *ast.CheckCondition:
		if v.Result != "success" && v.Result != "fail" {
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: fmt.Sprintf("check condition result %q is invalid (must be success or fail)", v.Result),
			})
		}
	case *ast.RatingCondition:
		if v.Grade == "" {
			*errs = append(*errs, Error{
				Code:    InvalidCondition,
				Message: "rating condition has empty grade",
			})
		}
	case nil:
		// nil condition can appear in unconditional gate routes; caller handles.
	default:
		*errs = append(*errs, Error{
			Code:    InvalidCondition,
			Message: fmt.Sprintf("unknown condition type %T", c),
		})
	}
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
				collectLabels(opt.Body, labels)
			}
		case *ast.IfNode:
			collectLabels(v.Then, labels)
			collectLabels(v.Else, labels)
		case *ast.MinigameNode:
			collectLabels(v.Body, labels)
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
				checkGotos(opt.Body, labels, errs)
			}
		case *ast.IfNode:
			checkGotos(v.Then, labels, errs)
			checkGotos(v.Else, labels, errs)
		case *ast.MinigameNode:
			checkGotos(v.Body, labels, errs)
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
			// Check for duplicate option IDs within this choice.
			seen := make(map[string]bool)
			for _, opt := range v.Options {
				if seen[opt.ID] {
					*errs = append(*errs, Error{
						Code:    DuplicateOptionID,
						Message: fmt.Sprintf("duplicate option ID %q in @choice block", opt.ID),
					})
				}
				seen[opt.ID] = true

				if opt.Mode == "brave" {
					// 4. Brave options must have a check block.
					if opt.Check == nil {
						*errs = append(*errs, Error{
							Code:    BraveNoCheck,
							Message: fmt.Sprintf("brave option %q is missing a @check block", opt.ID),
						})
					}
				} else if opt.Mode == "safe" {
					if opt.Check != nil {
						*errs = append(*errs, Error{
							Code:    SafeOptionHasCheck,
							Message: fmt.Sprintf("safe option %q should not have a check block", opt.ID),
						})
					}
				} else {
					*errs = append(*errs, Error{
						Code:    InvalidOptionMode,
						Message: fmt.Sprintf("option %q has invalid mode %q (must be 'brave' or 'safe')", opt.ID, opt.Mode),
					})
				}
				// Recurse into option body.
				checkBraveOptions(opt.Body, errs)
			}
		case *ast.CgShowNode:
			checkBraveOptions(v.Body, errs)
		case *ast.IfNode:
			checkBraveOptions(v.Then, errs)
			checkBraveOptions(v.Else, errs)
		case *ast.MinigameNode:
			checkBraveOptions(v.Body, errs)
		case *ast.PhoneShowNode:
			checkBraveOptions(v.Body, errs)
		}
	}
}

// checkValues validates enum-like fields (positions, transitions, bubble types).
func checkValues(nodes []ast.Node, errs *[]Error) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.CharShowNode:
			if !validPositions[v.Position] {
				*errs = append(*errs, Error{
					Code:    InvalidPosition,
					Message: fmt.Sprintf("character %q has invalid position %q", v.Char, v.Position),
				})
			}
			if v.Transition != "" && !validTransitions[v.Transition] {
				*errs = append(*errs, Error{
					Code:    InvalidTransition,
					Message: fmt.Sprintf("character %q show has invalid transition %q", v.Char, v.Transition),
				})
			}
		case *ast.CharHideNode:
			if v.Transition != "" && !validTransitions[v.Transition] {
				*errs = append(*errs, Error{
					Code:    InvalidTransition,
					Message: fmt.Sprintf("character %q hide has invalid transition %q", v.Char, v.Transition),
				})
			}
		case *ast.CharLookNode:
			if v.Transition != "" && !validTransitions[v.Transition] {
				*errs = append(*errs, Error{
					Code:    InvalidTransition,
					Message: fmt.Sprintf("character %q look has invalid transition %q", v.Char, v.Transition),
				})
			}
		case *ast.CharMoveNode:
			if !validPositions[v.Position] {
				*errs = append(*errs, Error{
					Code:    InvalidPosition,
					Message: fmt.Sprintf("character %q move has invalid position %q", v.Char, v.Position),
				})
			}
		case *ast.CharBubbleNode:
			if !validBubbleTypes[v.BubbleType] {
				*errs = append(*errs, Error{
					Code:    InvalidBubbleType,
					Message: fmt.Sprintf("character %q has invalid bubble type %q", v.Char, v.BubbleType),
				})
			}
		case *ast.BgSetNode:
			if v.Transition != "" && !validTransitions[v.Transition] {
				*errs = append(*errs, Error{
					Code:    InvalidTransition,
					Message: fmt.Sprintf("bg %q has invalid transition %q", v.Name, v.Transition),
				})
			}
		case *ast.CgShowNode:
			if v.Transition != "" && !validTransitions[v.Transition] {
				*errs = append(*errs, Error{
					Code:    InvalidTransition,
					Message: fmt.Sprintf("cg %q has invalid transition %q", v.Name, v.Transition),
				})
			}
			if !validCgDurations[v.Duration] {
				*errs = append(*errs, Error{
					Code:    InvalidCgDuration,
					Message: fmt.Sprintf("cg %q has invalid duration %q (must be low, medium, or high)", v.Name, v.Duration),
				})
			}
			if v.Content == "" {
				*errs = append(*errs, Error{
					Code:    CgMissingContent,
					Message: fmt.Sprintf("cg %q missing required 'content' field", v.Name),
				})
			}
			checkValues(v.Body, errs)
		case *ast.ChoiceNode:
			for _, opt := range v.Options {
				checkValues(opt.Body, errs)
			}
		case *ast.IfNode:
			checkValues(v.Then, errs)
			checkValues(v.Else, errs)
		case *ast.MinigameNode:
			if v.Description == "" {
				*errs = append(*errs, Error{
					Code:    MinigameMissingDescription,
					Message: fmt.Sprintf("@minigame %q missing required description", v.ID),
				})
			}
			checkValues(v.Body, errs)
		case *ast.PhoneShowNode:
			checkValues(v.Body, errs)
		}
	}
}
