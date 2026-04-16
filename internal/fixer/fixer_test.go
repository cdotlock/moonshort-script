package fixer

import (
	"strings"
	"testing"
)

func TestFixUnclosedBlocks(t *testing.T) {
	input := `@episode "Test" {
@choice {
@option A {
NARRATOR: Hello
}
@option B {
NARRATOR: World
}
`
	// Missing 2 closing braces: one for @choice, one for @episode
	r := Fix(input)

	if !strings.HasSuffix(r.Fixed, "\n}\n}") {
		t.Errorf("expected 2 closing braces appended, got:\n%s", r.Fixed)
	}

	foundBraceFix := false
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			foundBraceFix = true
			break
		}
	}
	if !foundBraceFix {
		t.Errorf("expected fix message about missing closing }, got: %v", r.Fixes)
	}
}

func TestFixCharacterCasing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		fixed    bool
	}{
		{
			name:     "character name lowercased",
			input:    "@Mauricio show happy",
			expected: "@mauricio show happy",
			fixed:    true,
		},
		{
			name:     "keyword stays unchanged",
			input:    "@bg set beach",
			expected: "@bg set beach",
			fixed:    false,
		},
		{
			name:     "keyword cg stays unchanged",
			input:    "@cg show sunset",
			expected: "@cg show sunset",
			fixed:    false,
		},
		{
			name:     "mixed case character name",
			input:    "@Elena hide",
			expected: "@elena hide",
			fixed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Fix(tt.input)
			// Trim trailing newlines for comparison
			got := strings.TrimRight(r.Fixed, "\n")
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
			if tt.fixed && len(r.Fixes) == 0 {
				t.Error("expected fix to be reported, got none")
			}
			if !tt.fixed && len(r.Fixes) > 0 {
				// Filter out non-casing fixes (e.g. gates errors)
				casingFixes := 0
				for _, f := range r.Fixes {
					if strings.Contains(f, "casing") {
						casingFixes++
					}
				}
				if casingFixes > 0 {
					t.Errorf("expected no casing fixes, got: %v", r.Fixes)
				}
			}
		})
	}
}

func TestFixDialogueAtSign(t *testing.T) {
	input := "@NARRATOR: Hello world\n@MAURICIO: How are you?"
	r := Fix(input)

	lines := strings.Split(r.Fixed, "\n")
	if lines[0] != "NARRATOR: Hello world" {
		t.Errorf("line 1: got %q, want %q", lines[0], "NARRATOR: Hello world")
	}
	if lines[1] != "MAURICIO: How are you?" {
		t.Errorf("line 2: got %q, want %q", lines[1], "MAURICIO: How are you?")
	}

	if len(r.Fixes) < 2 {
		t.Errorf("expected at least 2 fixes for dialogue @, got %d: %v", len(r.Fixes), r.Fixes)
	}
}

func TestFixUnquotedButterfly(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		fixed    bool
	}{
		{
			name:     "unquoted butterfly argument",
			input:    `@butterfly Accepted Easton`,
			expected: `@butterfly "Accepted Easton"`,
			fixed:    true,
		},
		{
			name:     "already quoted butterfly",
			input:    `@butterfly "Accepted Easton"`,
			expected: `@butterfly "Accepted Easton"`,
			fixed:    false,
		},
		{
			name:     "signal with spaces gets quoted",
			input:    `@signal quest complete`,
			expected: `@signal "quest complete"`,
			fixed:    true,
		},
		{
			name:     "signal without spaces stays unquoted",
			input:    `@signal done`,
			expected: `@signal done`,
			fixed:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Fix(tt.input)
			got := strings.TrimRight(r.Fixed, "\n")
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFixTrailingWhitespace(t *testing.T) {
	input := "NARRATOR: Hello   \n@bg set beach  \n"
	r := Fix(input)

	for i, line := range strings.Split(r.Fixed, "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d still has trailing whitespace: %q", i+1, line)
		}
	}

	foundWhitespaceFix := false
	for _, f := range r.Fixes {
		if strings.Contains(f, "trailing whitespace") {
			foundWhitespaceFix = true
			break
		}
	}
	if !foundWhitespaceFix {
		t.Errorf("expected trailing whitespace fix message, got: %v", r.Fixes)
	}
}

func TestFixNormalizeBlankLines(t *testing.T) {
	input := "line1\n\n\n\n\nline2\n\nline3"
	r := Fix(input)

	// Count max consecutive blank lines
	maxBlanks := 0
	blanks := 0
	for _, line := range strings.Split(r.Fixed, "\n") {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks > maxBlanks {
				maxBlanks = blanks
			}
		} else {
			blanks = 0
		}
	}

	if maxBlanks > 2 {
		t.Errorf("expected max 2 consecutive blank lines, got %d", maxBlanks)
	}

	foundNormFix := false
	for _, f := range r.Fixes {
		if strings.Contains(f, "blank lines") {
			foundNormFix = true
			break
		}
	}
	if !foundNormFix {
		t.Errorf("expected blank line normalization fix, got: %v", r.Fixes)
	}
}

func TestFixMultipleIssues(t *testing.T) {
	input := `@episode "Test" {

@Mauricio show happy
@NARRATOR: Once upon a time
@butterfly Accepted Easton
@bg set beach

@gate {
@default "end"
}
`
	// Missing closing } for @episode

	r := Fix(input)

	// Check that character name was lowercased
	if !strings.Contains(r.Fixed, "@mauricio show happy") {
		t.Error("expected @mauricio to be lowercased")
	}

	// Check that dialogue @ was removed
	if !strings.Contains(r.Fixed, "NARRATOR: Once upon a time") {
		t.Error("expected @ removed from NARRATOR dialogue")
	}
	if strings.Contains(r.Fixed, "@NARRATOR:") {
		t.Error("@NARRATOR: should have been fixed to NARRATOR:")
	}

	// Check butterfly was quoted
	if !strings.Contains(r.Fixed, `@butterfly "Accepted Easton"`) {
		t.Error("expected butterfly argument to be quoted")
	}

	// Check trailing whitespace removed
	for i, line := range strings.Split(r.Fixed, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("line %d still has trailing whitespace: %q", i+1, line)
		}
	}

	// Should have multiple fixes
	if len(r.Fixes) < 3 {
		t.Errorf("expected at least 3 fixes, got %d: %v", len(r.Fixes), r.Fixes)
	}
}

func TestErrorMissingGate(t *testing.T) {
	input := `@episode "Test" {
NARRATOR: Hello
}`
	r := Fix(input)

	foundGateError := false
	for _, e := range r.Errors {
		if strings.Contains(e, "missing @gate block") {
			foundGateError = true
			break
		}
	}
	if !foundGateError {
		t.Errorf("expected missing @gate error, got errors: %v", r.Errors)
	}
}

func TestErrorDuplicateOptionID(t *testing.T) {
	input := `@episode "Test" {
@choice {
@option A {
NARRATOR: First
}
@option A {
NARRATOR: Duplicate
}
}
@gate {
@default "end"
}
}`
	r := Fix(input)

	foundDupError := false
	for _, e := range r.Errors {
		if strings.Contains(e, "duplicate option ID: A") {
			foundDupError = true
			break
		}
	}
	if !foundDupError {
		t.Errorf("expected duplicate option ID error, got errors: %v", r.Errors)
	}
}

func TestErrorBraveNoCheck(t *testing.T) {
	input := `@episode "Test" {
@choice {
@option A brave {
NARRATOR: Brave but no check
@on success {
NARRATOR: Win
}
@on fail {
NARRATOR: Lose
}
}
}
@gate {
@default "end"
}
}`
	r := Fix(input)

	foundCheckError := false
	for _, e := range r.Errors {
		if strings.Contains(e, "brave but has no check block") {
			foundCheckError = true
			break
		}
	}
	if !foundCheckError {
		t.Errorf("expected brave no check error, got errors: %v", r.Errors)
	}
}

func TestErrorBraveMissingOutcomes(t *testing.T) {
	input := `@episode "Test" {
@choice {
@option A brave {
check {
skill "charisma"
dc 12
}
}
}
@gate {
@default "end"
}
}`
	r := Fix(input)

	foundOutcomeError := false
	for _, e := range r.Errors {
		if strings.Contains(e, "missing @on success/@on fail") {
			foundOutcomeError = true
			break
		}
	}
	if !foundOutcomeError {
		t.Errorf("expected brave missing outcomes error, got errors: %v", r.Errors)
	}
}

func TestErrorGotoWithoutLabel(t *testing.T) {
	input := `@episode "Test" {
@goto ending
@label start
NARRATOR: Hello
@gate {
@default "end"
}
}`
	r := Fix(input)

	foundGotoError := false
	for _, e := range r.Errors {
		if strings.Contains(e, "goto ending has no matching label") {
			foundGotoError = true
			break
		}
	}
	if !foundGotoError {
		t.Errorf("expected goto-without-label error, got errors: %v", r.Errors)
	}
}

func TestCleanFileNoChanges(t *testing.T) {
	input := `@episode "Clean Test" {

@mauricio show happy

NARRATOR: Everything is fine here.

@butterfly "Accepted Easton"

@choice {
@option A {
NARRATOR: Option A selected
}
@option B {
NARRATOR: Option B selected
}
}

@gate {
@default "end"
}
}`
	r := Fix(input)

	if len(r.Fixes) != 0 {
		t.Errorf("expected no fixes for clean file, got: %v", r.Fixes)
	}
	if len(r.Errors) != 0 {
		t.Errorf("expected no errors for clean file, got: %v", r.Errors)
	}
}
