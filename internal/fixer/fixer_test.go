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
			input:    "@Mauricio worried",
			expected: "@mauricio worried",
			fixed:    true,
		},
		{
			name:     "keyword stays unchanged",
			input:    "@bg set beach",
			expected: "@bg set beach",
			fixed:    false,
		},
		{
			name:     "keyword cg stays unchanged (leaf form)",
			input:    `@cg sunset "Golden hour fades across the bluffs."`,
			expected: `@cg sunset "Golden hour fades across the bluffs."`,
			fixed:    false,
		},
		{
			name:     "mixed case character name with pose",
			input:    "@Elena neutral",
			expected: "@elena neutral",
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
			// The fixer does not touch @signal lines — the directive
			// takes a structured `<kind> <event>` argument and quoting
			// the tail would corrupt it.
			name:     "signal with spaces is left alone",
			input:    `@signal mark quest_complete`,
			expected: `@signal mark quest_complete`,
			fixed:    false,
		},
		{
			name:     "signal without spaces stays unquoted",
			input:    `@signal mark done`,
			expected: `@signal mark done`,
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

@Mauricio worried
@NARRATOR: Once upon a time
@butterfly Accepted Easton
@bg set beach

@gate {
@end complete
}
`
	// Missing closing } for @episode

	r := Fix(input)

	// Check that character name was lowercased
	if !strings.Contains(r.Fixed, "@mauricio worried") {
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
@end complete
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
	// Brave option without a check block — the fixer surfaces this as an
	// error. The body uses the canonical @if (check.success) form; the
	// fixer only cares that `check { }` is missing.
	input := `@episode "Test" {
@choice {
@option A brave {
NARRATOR: Brave but no check
@if (check.success) {
NARRATOR: Win
} @else {
NARRATOR: Lose
}
}
}
@gate {
@end complete
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

// TestErrorOnSyntaxIsOldFormat verifies that `@on success` / `@on fail`
// inside a brave option is surfaced as an old-format hint pointing at
// the correct @if (check.success) / @else syntax.
func TestErrorOnSyntaxIsOldFormat(t *testing.T) {
	input := `@episode "Test" {
@choice {
@option A brave {
check {
attr: CHA
dc: 12
}
@on success {
NARRATOR: Win
}
@on fail {
NARRATOR: Lose
}
}
}
@gate {
@end complete
}
}`
	r := Fix(input)

	foundOnHint := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@on") && strings.Contains(e, "not part of MSS syntax") {
			foundOnHint = true
		}
	}
	if !foundOnHint {
		t.Errorf("expected @on old-format hint, got errors: %v", r.Errors)
	}
}

// TestErrorGotoAndLabelOldFormat verifies that the legacy `@goto` and
// `@label` directives are surfaced as old-format errors with hints
// pointing at the @if/@else replacement (the new AST has no
// label/goto nodes — in-episode branching is conditional only).
func TestErrorGotoAndLabelOldFormat(t *testing.T) {
	input := `@episode "Test" {
@goto ending
@label start
NARRATOR: Hello
@gate {
@end complete
}
}`
	r := Fix(input)

	foundGoto := false
	foundLabel := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@goto") && strings.Contains(e, "old-format") {
			foundGoto = true
		}
		if strings.Contains(e, "@label") && strings.Contains(e, "old-format") {
			foundLabel = true
		}
	}
	if !foundGoto {
		t.Errorf("expected @goto old-format error, got errors: %v", r.Errors)
	}
	if !foundLabel {
		t.Errorf("expected @label old-format error, got errors: %v", r.Errors)
	}
}

func TestFixAmpersandOnBlock(t *testing.T) {
	input := "&choice {\n  @option A safe \"test\" {\n  }\n}"
	r := Fix(input)
	if !strings.Contains(r.Fixed, "@choice") {
		t.Error("expected & converted to @ on choice")
	}
	if len(r.Fixes) == 0 {
		t.Error("expected fix recorded")
	}
}

func TestFixIfMissingParens(t *testing.T) {
	input := "@if affection.easton >= 5 {"
	r := Fix(input)
	if !strings.Contains(r.Fixed, "@if (affection.easton >= 5)") {
		t.Errorf("expected parens added, got: %s", r.Fixed)
	}
}

func TestFixAtCheck(t *testing.T) {
	input := "    @check {"
	r := Fix(input)
	if !strings.Contains(r.Fixed, "    check {") {
		t.Errorf("expected @check -> check, got: %s", r.Fixed)
	}
}

func TestFixBOM(t *testing.T) {
	input := "\xEF\xBB\xBF@episode main:01 \"T\" {"
	r := Fix(input)
	if strings.Contains(r.Fixed, "\xEF\xBB\xBF") {
		t.Error("BOM should be stripped")
	}
}

func TestFixCRLF(t *testing.T) {
	input := "line1\r\nline2\r\nline3"
	r := Fix(input)
	if strings.Contains(r.Fixed, "\r") {
		t.Error("CRLF should be normalized to LF")
	}
}

func TestFixAffectionCharCase(t *testing.T) {
	input := "@affection EASTON +2"
	r := Fix(input)
	if !strings.Contains(r.Fixed, "@affection easton +2") {
		t.Errorf("expected lowercase, got: %s", r.Fixed)
	}
}

func TestFixBraceCountSkipsDialogue(t *testing.T) {
	input := "@episode main:01 \"T\" {\n  NARRATOR: He said {goodbye} and left.\n"
	r := Fix(input)
	// The dialogue line contains "{goodbye}" (1 open + 1 close) but the fixer
	// should skip dialogue lines. It should append exactly 1 closing brace for @episode.
	// Total "}" in output = 1 inside dialogue text + 1 appended = 2.
	foundBraceFix := false
	for _, f := range r.Fixes {
		if strings.Contains(f, "1 missing closing }") {
			foundBraceFix = true
			break
		}
	}
	if !foundBraceFix {
		t.Errorf("expected fix for 1 missing closing brace, got: %v", r.Fixes)
	}
	// Verify it did NOT count dialogue braces (which would mean 0 appended)
	if !strings.HasSuffix(strings.TrimRight(r.Fixed, "\n"), "}") {
		t.Error("expected closing brace appended at end of fixed output")
	}
}

func TestFixOldFormatDetection(t *testing.T) {
	input := "@episode main:01 \"T\" {\n  @show malia neutral at center\n  @gate {\n    @next main:02\n  }\n}"
	r := Fix(input)
	foundOldFormat := false
	for _, e := range r.Errors {
		if strings.Contains(e, "old-format syntax") {
			foundOldFormat = true
			break
		}
	}
	if !foundOldFormat {
		t.Error("expected old-format syntax error for @show")
	}
}

func TestFixAmpersandDialogue(t *testing.T) {
	input := "&NARRATOR: Hello"
	r := Fix(input)
	if !strings.Contains(r.Fixed, "NARRATOR: Hello") {
		t.Errorf("expected & removed, got: %s", r.Fixed)
	}
}

// TestFixTrickKeywordPreserved verifies @trick survives the fixer's
// character-casing pass (it's a known keyword, not a character name).
func TestFixTrickKeywordPreserved(t *testing.T) {
	input := `@trick tap "Tap to keep up."`
	r := Fix(input)
	if !strings.Contains(r.Fixed, "@trick tap") {
		t.Errorf("expected @trick to be preserved, got: %s", r.Fixed)
	}
	for _, f := range r.Fixes {
		if strings.Contains(f, "character name casing") {
			t.Errorf("@trick should not be miscategorised as a character name: %v", r.Fixes)
		}
	}
}

// ---------------------------------------------------------------------------
// Legacy-pattern detection — one test per new check in checkOldFormatSyntax.
// ---------------------------------------------------------------------------

// TestLegacyCharShowSuffix verifies the new char-suffix legacy detector
// catches `@<char> show ...` (and friends) and surfaces the new
// `@<char> <pose>` hint.
func TestLegacyCharShowSuffix(t *testing.T) {
	cases := []string{
		"@malia show neutral",
		"@malia hide",
		"@malia look angry",
		"@malia move right",
		"@malia at center",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			r := Fix(line)
			found := false
			for _, e := range r.Errors {
				if strings.Contains(e, "legacy char directive") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected legacy-char-directive error for %q, got: %v", line, r.Errors)
			}
		})
	}
}

// TestLegacyPhoneShowHide verifies that the legacy `@phone show` /
// `@phone hide` form is rejected. The form `@phone show{` (no space
// before brace) bypasses the broad char-suffix regex and is caught
// specifically by phoneLegacyRe with the @phone-block hint.
func TestLegacyPhoneShowHide(t *testing.T) {
	// Brace-attached form exercises phoneLegacyRe directly.
	for _, line := range []string{"@phone show{", "@phone hide{"} {
		t.Run("specific/"+line, func(t *testing.T) {
			r := Fix(line)
			found := false
			for _, e := range r.Errors {
				if strings.Contains(e, "@phone { ... } block") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected phone-lifecycle hint for %q, got: %v", line, r.Errors)
			}
		})
	}

	// Space-separated form is also rejected, though through the broader
	// char-legacy-suffix check. The user-visible contract is just "the
	// legacy phone form yields a legacy-format error".
	for _, line := range []string{"@phone show", "@phone hide"} {
		t.Run("legacy/"+line, func(t *testing.T) {
			r := Fix(line)
			rejected := false
			for _, e := range r.Errors {
				if strings.Contains(e, "legacy") || strings.Contains(e, "@phone { ... } block") {
					rejected = true
					break
				}
			}
			if !rejected {
				t.Errorf("expected legacy-format error for %q, got: %v", line, r.Errors)
			}
		})
	}
}

// TestLegacyMusicPlay verifies `@music play <name>` / `@music crossfade <name>`
// is flagged with a hint pointing at the new `@music <name>` leaf.
func TestLegacyMusicPlay(t *testing.T) {
	for _, line := range []string{"@music play tense", "@music crossfade tense"} {
		t.Run(line, func(t *testing.T) {
			r := Fix(line)
			found := false
			for _, e := range r.Errors {
				if strings.Contains(e, "use @music <name>") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected music-play/crossfade hint for %q, got: %v", line, r.Errors)
			}
		})
	}
}

// TestLegacyMusicFadeout verifies bare `@music fadeout` is flagged with
// a hint pointing at the new `@music stop` leaf.
func TestLegacyMusicFadeout(t *testing.T) {
	r := Fix("@music fadeout")
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "use @music stop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected music-fadeout hint, got: %v", r.Errors)
	}
}

// TestLegacySfxPlay verifies `@sfx play <name>` is flagged with the
// `@sfx <name>` hint.
func TestLegacySfxPlay(t *testing.T) {
	r := Fix("@sfx play door_slam")
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "use @sfx <name>") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected sfx-play hint, got: %v", r.Errors)
	}
}

// TestLegacyPauseForN verifies `@pause for N` is flagged with the
// "repeat the directive" hint (the new AST has no duration parameter).
func TestLegacyPauseForN(t *testing.T) {
	r := Fix("@pause for 3")
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@pause is single-click only") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pause-for-N hint, got: %v", r.Errors)
	}
}

// TestLegacyIfStringLiteral verifies a bare double-quoted-string
// condition is rejected.
func TestLegacyIfStringLiteral(t *testing.T) {
	r := Fix(`@if ("EP01_COMPLETE") {`)
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "string literal not allowed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected string-literal-condition error, got: %v", r.Errors)
	}
}

// TestLegacyIfInfluence verifies the removed `influence` condition is
// rejected with a hint pointing at affection / value comparisons.
func TestLegacyIfInfluence(t *testing.T) {
	for _, line := range []string{
		"@if (influence >= 5) {",
		"@if (influence) {",
		"@if (influence_ok) {", // sanity: NOT influence (has trailing chars) — must NOT trigger
	} {
		t.Run(line, func(t *testing.T) {
			r := Fix(line)
			triggered := false
			for _, e := range r.Errors {
				if strings.Contains(e, "influence condition removed") {
					triggered = true
					break
				}
			}
			isInfluence := strings.Contains(line, "influence)") || strings.Contains(line, "influence >") || strings.Contains(line, "influence <") || strings.Contains(line, "influence =") || strings.Contains(line, "influence !")
			if isInfluence && !triggered {
				t.Errorf("expected influence-condition error for %q, got: %v", line, r.Errors)
			}
			if !isInfluence && triggered {
				t.Errorf("did NOT expect influence error for %q, got: %v", line, r.Errors)
			}
		})
	}
}

// TestLegacyCgBlockForm verifies the old block-form `@cg show ...`
// is rejected. The brace-attached form `@cg show{` bypasses the
// broad char-suffix regex and is caught specifically by
// cgBlockFormRe with the leaf-directive hint.
func TestLegacyCgBlockForm(t *testing.T) {
	// Brace-attached form exercises cgBlockFormRe directly.
	r := Fix("@cg show{")
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@cg is a leaf directive") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected cg-leaf-directive hint for @cg show{, got: %v", r.Errors)
	}

	// Space-separated `@cg show sunset {` is also rejected, though
	// through the broader char-legacy-suffix detector. The
	// user-visible contract is just "old @cg block form yields a
	// legacy-format error".
	r2 := Fix("@cg show sunset {")
	rejected := false
	for _, e := range r2.Errors {
		if strings.Contains(e, "legacy") || strings.Contains(e, "leaf directive") {
			rejected = true
			break
		}
	}
	if !rejected {
		t.Errorf("expected legacy-format error for `@cg show sunset {`, got: %v", r2.Errors)
	}
}

// TestLegacyEndingKeyword verifies standalone `@ending` is flagged
// (endings are now gate leaves via `@gate { @end <type> }`).
func TestLegacyEndingKeyword(t *testing.T) {
	r := Fix("@ending complete")
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@ending") && strings.Contains(e, "@gate { @end") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected @ending old-format error with @gate/@end hint, got: %v", r.Errors)
	}
}

func TestCleanFileNoChanges(t *testing.T) {
	input := `@episode "Clean Test" {

@mauricio neutral

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
@end complete
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
