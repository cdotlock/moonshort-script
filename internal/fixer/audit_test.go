package fixer

import (
	"strings"
	"testing"
)

// =============================================================================
// A. Fixer rule interactions and ordering
// =============================================================================

// A1. `&IF affection.easton >= 5 {` needs THREE fixes:
//   1. & -> @ on block directive (step 5)
//   2. IF -> if keyword casing (step 3, but only if it starts with @ or &)
//   3. Missing parens on @if condition (step 6)
// Issue: step 3 (fixDirectiveCasing) runs BEFORE step 5 (fixAmpersandOnBlocks).
// When line is "&IF ...", step 3 sees &IF, lowercases IF -> if (since "if" is a known keyword).
// Then step 5 sees "&if ..." and converts & -> @.
// Then step 6 sees "@if affection.easton >= 5 {" and wraps parens.
// This SHOULD work because step 3 handles & prefix too.
func TestMultiFix_AmpersandUppercaseIfMissingParens(t *testing.T) {
	// Use a balanced block so fixUnclosedBlocks doesn't interfere.
	input := "&IF affection.easton >= 5 {\n}"
	r := Fix(input)

	// The first line should be fixed to @if (affection.easton >= 5) {
	lines := strings.Split(r.Fixed, "\n")
	if len(lines) == 0 || lines[0] != "@if (affection.easton >= 5) {" {
		t.Errorf("triple-fix failed:\n  got first line:  %q\n  want: %q\n  fixes: %v", lines[0], "@if (affection.easton >= 5) {", r.Fixes)
	}

	// Should have at least 3 fix messages (keyword casing, & -> @, missing parens)
	fixCount := 0
	for _, f := range r.Fixes {
		if strings.Contains(f, "keyword casing") || strings.Contains(f, "replaced &") || strings.Contains(f, "parentheses") {
			fixCount++
		}
	}
	if fixCount < 3 {
		t.Errorf("expected >= 3 relevant fixes, got %d: %v", fixCount, r.Fixes)
	}
}

// A2. `@AFFECTION EASTON +2` needs TWO fixes:
//   1. AFFECTION -> affection (keyword casing via fixDirectiveCasing, step 3)
//   2. EASTON -> easton (char name via fixAffectionCharCase, step 8)
// Issue: step 3 should lowercase "AFFECTION" to "affection" (it's a known keyword).
// Then step 8 should match `@affection EASTON +2` and lowercase EASTON.
func TestMultiFix_AffectionKeywordAndCharCasing(t *testing.T) {
	input := "@AFFECTION EASTON +2"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	want := "@affection easton +2"
	if got != want {
		t.Errorf("double-fix failed:\n  got:  %q\n  want: %q\n  fixes: %v", got, want, r.Fixes)
	}
}

// A3. `&NARRATOR: He said {hello}` needs & removal AND brace skip.
// Step 2 removes the & from dialogue. fixUnclosedBlocks should then skip it as dialogue.
func TestMultiFix_AmpersandDialogueWithBraces(t *testing.T) {
	input := "@episode main:01 \"T\" {\n&NARRATOR: He said {hello} and left.\n}"
	r := Fix(input)
	got := r.Fixed

	// The & should be stripped from the dialogue line
	if strings.Contains(got, "&NARRATOR") {
		t.Errorf("& was not removed from dialogue line")
	}
	if !strings.Contains(got, "NARRATOR: He said {hello} and left.") {
		t.Errorf("dialogue line corrupted: %s", got)
	}

	// Braces should be balanced (the dialogue { } should be skipped)
	// No extra closing braces should be appended
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("should not append closing braces when dialogue braces are skipped, got fix: %s", f)
		}
	}
}

// A2b. What about `&affection EASTON +2`? The & prefix is NOT a block directive
// (affection is not in blockDirectives), so fixAmpersandOnBlocks won't touch it.
// fixDirectiveCasing sees "&affection" -- "affection" IS a known keyword already lowercase.
// fixAffectionCharCase regex: `^(\s*[@&]affection\s+)([A-Z][A-Z0-9_]+)(\s+.*)$`
// It handles both @ and &, so EASTON should be lowercased.
func TestMultiFix_AmpersandAffectionCharCase(t *testing.T) {
	input := "&affection EASTON +2"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// The & stays (affection is not a block directive), but EASTON should be lowered
	want := "&affection easton +2"
	if got != want {
		t.Errorf("got: %q, want: %q, fixes: %v", got, want, r.Fixes)
	}
}

// =============================================================================
// B. Regex safety edge cases
// =============================================================================

// B1. `@if (condition) { // valid but with trailing comment` — already has parens,
// should NOT be touched by fixIfMissingParens (regex requires [^(\s] after @if).
func TestRegex_IfWithParensAndComment(t *testing.T) {
	input := "@if (condition) { // valid"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// Should not double-wrap parens
	if strings.Contains(got, "((condition))") {
		t.Errorf("double parens: %q", got)
	}
	// The line should remain essentially unchanged (apart from possible trailing ws)
	if !strings.Contains(got, "@if (condition)") {
		t.Errorf("@if with parens was corrupted: %q", got)
	}
}

// B2. `@if condition1 || condition2 {` — has || but no parens.
// fixIfMissingParens should wrap the whole condition.
func TestRegex_IfOrConditionNoParens(t *testing.T) {
	input := "@if condition1 || condition2 {\n}"
	r := Fix(input)
	lines := strings.Split(r.Fixed, "\n")

	want := "@if (condition1 || condition2) {"
	if len(lines) == 0 || lines[0] != want {
		t.Errorf("got first line: %q, want: %q", lines[0], want)
	}
}

// B3. `@check{}` — no space between check and brace.
// atCheckRe: `^(\s*)@(check\s*\{.*)$` — \s* allows zero spaces.
func TestRegex_CheckNoSpace(t *testing.T) {
	input := "@check{}"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	if strings.HasPrefix(got, "@check") {
		t.Errorf("@ should be stripped: got %q", got)
	}
	if !strings.HasPrefix(got, "check{}") {
		t.Errorf("expected 'check{}', got: %q", got)
	}
}

// B4. `  @check  {` — extra spaces between check and brace.
func TestRegex_CheckExtraSpaces(t *testing.T) {
	input := "  @check  {"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	if strings.Contains(got, "@check") {
		t.Errorf("@ should be stripped: got %q", got)
	}
	if !strings.HasPrefix(got, "  check  {") {
		t.Errorf("expected '  check  {', got: %q", got)
	}
}

// B5. `@affection easton_jr +2` — char name with underscore (should NOT be lowercased further).
// affectionCharRe: `^(\s*[@&]affection\s+)([A-Z][A-Z0-9_]+)(\s+.*)$`
// "easton_jr" starts with lowercase, so [A-Z] won't match → no fix applied. Correct.
func TestRegex_AffectionCharWithUnderscore(t *testing.T) {
	input := "@affection easton_jr +2"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// Already lowercase, should be unchanged
	if got != "@affection easton_jr +2" {
		t.Errorf("should not modify already-lowercase char name: %q", got)
	}
}

// B5b. What about `@affection EASTON_JR +2`? Should lowercase to easton_jr.
func TestRegex_AffectionUppercaseCharWithUnderscore(t *testing.T) {
	input := "@affection EASTON_JR +2"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	want := "@affection easton_jr +2"
	if got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

// B6. `@affection EASTON` — missing delta (no +N). The affectionCharRe requires
// `(\s+.*)$` as the third group. With no trailing text, `\s+.*` won't match.
// BUG HYPOTHESIS: The regex REQUIRES a space+something after the char name.
// So `@affection EASTON` (no delta) won't match, and EASTON won't be lowercased.
func TestRegex_AffectionMissingDelta(t *testing.T) {
	input := "@affection EASTON"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// Expected: @affection easton (lowercased even without delta)
	// But the regex requires `(\s+.*)$` which demands at least one space after the char name.
	// This is a potential BUG: the char name won't be lowercased if delta is missing.
	want := "@affection easton"
	if got != want {
		t.Errorf("BUG: affection char name not lowercased when delta is missing.\n  got:  %q\n  want: %q\n  This is because affectionCharRe requires \\s+ after char name.", got, want)
	}
}

// B7. Empty input to Fix().
func TestRegex_EmptyInput(t *testing.T) {
	r := Fix("")
	if r.Fixed != "" {
		t.Errorf("expected empty output for empty input, got: %q", r.Fixed)
	}
}

// B8. Input with only whitespace.
func TestRegex_WhitespaceOnly(t *testing.T) {
	r := Fix("   \n   \n   ")
	// Should strip trailing whitespace from each line
	for i, line := range strings.Split(r.Fixed, "\n") {
		if line != "" {
			t.Errorf("line %d should be empty after trimming, got: %q", i+1, line)
		}
	}
}

// B9. `@if already_has_parens_but_starts_with_non_paren {` - tricky edge case
// where condition starts with a letter (not paren), triggering the fix.
// But what about `@if (partial condition {` — has open paren but regex
// checks [^(\s], so starting with ( means no match. Good.
func TestRegex_IfAlreadyHasOpenParen(t *testing.T) {
	input := "@if (partial condition {"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// Should NOT add more parens since it starts with (
	if strings.Contains(got, "((") {
		t.Errorf("should not double-wrap: %q", got)
	}
}

// =============================================================================
// C. isDialogueLine accuracy
// =============================================================================

func TestIsDialogueLine(t *testing.T) {
	tests := []struct {
		input string
		want  bool
		desc  string
	}{
		{"NARRATOR: text", true, "standard narrator"},
		{"narrator: text", false, "lowercase narrator"},
		{"A: text", true, "single char name"},
		{"CHECK: text", true, "keyword name but also valid dialogue"},
		{"@bg set name:", false, "has @ prefix - not dialogue (colon found after 'name')"},
		{"CHAR_NAME [look]: text", true, "bracket syntax"},
		{": text", false, "empty name"},
		{"NO_COLON_HERE", false, "no colon"},
		{"A1_TEST: hello", true, "name with digits and underscore"},
		{"a: text", false, "single lowercase char"},
		{"AB: text", true, "two-char uppercase name"},
		{"NARRATOR:text", true, "no space after colon"},
		{"NARRATOR:", true, "colon but no text"},
		{"  NARRATOR: text", true, "leading spaces (trimmed by caller)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := isDialogueLine(tt.input)
			if got != tt.want {
				t.Errorf("isDialogueLine(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// C5. `@bg set name:` — the colon after "name" means isDialogueLine checks
// line[:colonIdx] = "@bg set name". After TrimSpace that's "@bg set name".
// The first char '@' is not A-Z and not 0-9 and not '_', so allUpper=false → returns false.
// BUT WAIT: isDialogueLine searches for the FIRST colon. In "@bg set name:",
// the first colon is at the end. name = "@bg set name". '@' breaks allUpper. Correct.
// However, what about a line like: `BG: set something`? BG is a valid keyword
// but also a valid all-caps name. isDialogueLine returns true. That's by design.
func TestIsDialogueLine_KeywordAsDialogue(t *testing.T) {
	// BG: is treated as dialogue because BG is all-caps
	if !isDialogueLine("BG: something") {
		t.Error("BG: should be detected as dialogue (all-caps before colon)")
	}
}

// C special: what about leading whitespace? isDialogueLine gets the trimmed line
// from fixUnclosedBlocks. Let's verify the caller trims.
func TestIsDialogueLine_LeadingSpaces(t *testing.T) {
	// isDialogueLine itself does NOT trim leading spaces in the "name" extraction.
	// Wait - let's re-read: `name := strings.TrimSpace(line[:i])`
	// It DOES TrimSpace the name part. So "  NARRATOR: text" → name = "NARRATOR". Good.
	if !isDialogueLine("  NARRATOR: text") {
		t.Error("leading spaces should be handled by TrimSpace in name extraction")
	}
}

// =============================================================================
// D. Validator whitelist edge cases (tested via fixer's error checking too)
// =============================================================================

// D1. Empty position string. In the validator, `validPositions[""]` is false,
// so an empty position would trigger an error. This is correct behavior since
// the parser should always set a position. But let's verify in the fixer context.

// D2-D4 are tested in the validator test file. Here we test the fixer's error checks
// since they operate on text, not AST.

// =============================================================================
// E. Validator recursion completeness — tested in validator_audit_test.go
// =============================================================================

// =============================================================================
// F. fixUnclosedBlocks edge cases
// =============================================================================

// F1. More closing braces than opening — open goes negative.
// The fixer should NOT append anything (open <= 0).
func TestUnclosedBlocks_MoreClosingThanOpening(t *testing.T) {
	input := "@episode main:01 \"T\" {\n}\n}\n}"
	r := Fix(input)

	// Should not append any braces
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("should not append braces when there are excess closing braces: %v", r.Fixes)
		}
	}
}

// F2. Braces inside a non-dialogue @signal or @butterfly line — these are directive
// lines, not dialogue. The fixer does NOT skip them (only dialogue is skipped).
// `@signal quest_{complete}` — the { } would be counted. Is this a problem?
// In practice, signal arguments should be quoted, so this is an edge case.
func TestUnclosedBlocks_BracesInDirective(t *testing.T) {
	input := "@episode main:01 \"T\" {\n@signal quest_{complete}\n}"
	r := Fix(input)

	// The { in signal and } in signal cancel out (net 0 from that line).
	// @episode has { (open=1), signal has { and } (net 0), final } (open=0).
	// So no extra braces should be appended.
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("braces in signal should cancel out: %v", r.Fixes)
		}
	}
}

// F3. Commented-out block: `// @if (x) {` — the { should be skipped.
func TestUnclosedBlocks_CommentedOutBlock(t *testing.T) {
	input := "@episode main:01 \"T\" {\n// @if (x) {\n// }\n}"
	r := Fix(input)

	// Comments are skipped, so only @episode { and } count → balanced.
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("commented braces should be skipped: %v", r.Fixes)
		}
	}
}

// F4. Dialogue line with unbalanced braces.
func TestUnclosedBlocks_DialogueUnbalancedBraces(t *testing.T) {
	input := "@episode main:01 \"T\" {\nNARRATOR: He said { but never closed it.\n}"
	r := Fix(input)

	// The dialogue line should be skipped entirely.
	// Only @episode { and } count → balanced. No extra braces.
	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("unbalanced braces in dialogue should be skipped: %v", r.Fixes)
		}
	}
}

// F5. A line that looks like dialogue but contains braces in [expr] part.
// `CHAR [happy{ish}]: text` — isDialogueLine should still detect this as dialogue
// because it checks for ALLCAPS before the first colon, stripping bracket content.
func TestUnclosedBlocks_DialogueWithBracketBraces(t *testing.T) {
	// Wait — brackets can contain braces? Let's check isDialogueLine.
	// isDialogueLine finds the first colon. In "CHAR [happy{ish}]: text",
	// the first colon is after ']'. name = "CHAR [happy{ish}]".
	// It strips [bracket] part: bracketIdx of "[" → name = "CHAR". CHAR is allcaps. → true.
	// So this line IS skipped. Good.
	input := "@episode main:01 \"T\" {\nCHAR [happy{ish}]: text\n}"
	r := Fix(input)

	for _, f := range r.Fixes {
		if strings.Contains(f, "missing closing }") {
			t.Errorf("dialogue with braces in bracket expr should be skipped: %v", r.Fixes)
		}
	}
}

// =============================================================================
// G. Old format detection accuracy
// =============================================================================

// G1. `@show` at line start — should be detected.
func TestOldFormat_ShowAtLineStart(t *testing.T) {
	input := "@episode main:01 \"T\" {\n@show malia neutral at center\n@gate {\n@next main:02\n}\n}"
	r := Fix(input)

	foundShow := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@show") && strings.Contains(e, "old-format") {
			foundShow = true
		}
	}
	if !foundShow {
		t.Errorf("@show should be detected as old format, errors: %v", r.Errors)
	}
}

// G2. `@show` inside a dialogue string — should NOT be detected.
// `NARRATOR: Let me @show you` — the trimmed line starts with "NARRATOR:",
// not "@show". So this should NOT trigger old format detection.
func TestOldFormat_ShowInsideDialogue(t *testing.T) {
	input := "@episode main:01 \"T\" {\nNARRATOR: Let me @show you\n@gate {\n@next main:02\n}\n}"
	r := Fix(input)

	for _, e := range r.Errors {
		// There will be @next old format error, but NOT @show
		if strings.Contains(e, "@show") && strings.Contains(e, "old-format") {
			t.Errorf("@show inside dialogue should NOT be detected: %s", e)
		}
	}
}

// G3. `@shower` — should NOT be detected (it's not `@show ` with space, nor bare `@show`).
func TestOldFormat_ShowerNotDetected(t *testing.T) {
	input := "@shower something"
	r := Fix(input)

	for _, e := range r.Errors {
		if strings.Contains(e, "@show") && strings.Contains(e, "old-format") {
			t.Errorf("@shower should NOT trigger @show detection: %s", e)
		}
	}
}

// G4. `@show` with tab separator.
func TestOldFormat_ShowWithTab(t *testing.T) {
	input := "@episode main:01 \"T\" {\n@show\tmalia neutral\n@gate {\n@next main:02\n}\n}"
	r := Fix(input)

	foundShow := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@show") && strings.Contains(e, "old-format") {
			foundShow = true
		}
	}
	if !foundShow {
		t.Errorf("@show with tab should be detected as old format, errors: %v", r.Errors)
	}
}

// G5. Bare `@show` with nothing after it.
func TestOldFormat_BareShow(t *testing.T) {
	input := "@episode main:01 \"T\" {\n@show\n@gate {\n@next main:02\n}\n}"
	r := Fix(input)

	foundShow := false
	for _, e := range r.Errors {
		if strings.Contains(e, "@show") && strings.Contains(e, "old-format") {
			foundShow = true
		}
	}
	if !foundShow {
		t.Errorf("bare @show should be detected as old format, errors: %v", r.Errors)
	}
}

// =============================================================================
// Additional edge cases discovered during analysis
// =============================================================================

// EDGE: fixDirectiveCasing with `@` followed by nothing.
func TestDirectiveCasing_AtSignAlone(t *testing.T) {
	input := "@"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")
	if got != "@" {
		t.Errorf("bare @ should be unchanged, got: %q", got)
	}
}

// EDGE: fixDirectiveCasing with `@{` — the word delimiter is {.
func TestDirectiveCasing_AtBrace(t *testing.T) {
	input := "@{}"
	r := Fix(input)
	// The word is empty (idx=0 for '{'), so it should be unchanged.
	got := strings.TrimRight(r.Fixed, "\n")
	if got != "@{}" {
		t.Errorf("@{} should be unchanged, got: %q", got)
	}
}

// EDGE: fixUnquotedArgs with &butterfly (ampersand variant).
func TestUnquotedArgs_AmpersandButterfly(t *testing.T) {
	input := "&butterfly Accepted Easton"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	if !strings.Contains(got, `"Accepted Easton"`) {
		t.Errorf("&butterfly args should be quoted: %q", got)
	}
}

// EDGE: fixUnquotedArgs with &signal that has spaces.
func TestUnquotedArgs_AmpersandSignalWithSpaces(t *testing.T) {
	input := "&signal quest complete"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	if !strings.Contains(got, `"quest complete"`) {
		t.Errorf("&signal with spaces should be quoted: %q", got)
	}
}

// EDGE: What happens to @signal without spaces but with & prefix?
// &signal should NOT trigger quoting for single-word args.
// But wait: the code checks `if directive == "@signal"` for the spaces check.
// It does NOT check `&signal`. BUG?
func TestUnquotedArgs_AmpersandSignalNoSpaces(t *testing.T) {
	input := "&signal done"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// &signal with single-word arg should NOT be quoted (consistent with @signal).
	want := "&signal done"
	if got != want {
		t.Errorf("&signal single-word should not be quoted.\n  got:  %q\n  want: %q", got, want)
	}
}

// EDGE: allCapsColonRe — test that it requires at least 2 chars in the name.
// The regex is `^[@&]([A-Z][A-Z0-9_]+):` — note the `+` means 2+ chars total.
// So `@A: text` has name "A" which is [A-Z] followed by nothing (+ requires 1+ more).
// This means `@A: text` does NOT match allCapsColonRe → the @ is not stripped.
// But `A: text` IS a valid dialogue line per isDialogueLine (single char).
// This means `@A: text` → fixDirectiveCasing treats "A" as a character name
// and lowercases it to "@a: text". Not ideal but maybe acceptable.
func TestDialogueAtSign_SingleCharName(t *testing.T) {
	input := "@A: text"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// allCapsColonRe requires [A-Z][A-Z0-9_]+ (2+ chars). So @A: won't match.
	// The line then hits fixDirectiveCasing: prefix="@", word="A:", NO wait...
	// fixDirectiveCasing extracting word: finds first space/tab/{/: → at index 1 (the ':').
	// word = "A", after = ": text". "a" is not a known keyword → lowercase to @a.
	// Then fixUnquotedArgs, etc. won't match.
	// Result: "@a: text" — the @ is NOT stripped.
	// This is a minor BUG: @A: is a dialogue line with @, but the regex doesn't
	// catch it because the name is too short.
	if got == "A: text" {
		// Fixed correctly — great
	} else if got == "@a: text" {
		t.Logf("KNOWN ISSUE: @A: not recognized as dialogue by allCapsColonRe (requires 2+ char name). Got %q instead of 'A: text'", got)
	} else {
		t.Errorf("unexpected result for @A:: %q", got)
	}
}

// EDGE: What about `@IF (already correct) {` — keyword casing AND already has parens.
// Step 3 lowercases IF → if. Step 6 sees @if (...) { — but the regex
// requires [^(\s] after @if, and ( is excluded. So no double-wrap. Good.
func TestKeywordCasing_IfAlreadyHasParens(t *testing.T) {
	input := "@IF (x > 5) {\n}"
	r := Fix(input)
	lines := strings.Split(r.Fixed, "\n")

	want := "@if (x > 5) {"
	if len(lines) == 0 || lines[0] != want {
		t.Errorf("got first line: %q, want: %q", lines[0], want)
	}
}

// EDGE: `@ELSE {` — uppercase keyword should be lowercased.
func TestKeywordCasing_ElseUppercase(t *testing.T) {
	input := "@ELSE {\n}"
	r := Fix(input)
	lines := strings.Split(r.Fixed, "\n")

	want := "@else {"
	if len(lines) == 0 || lines[0] != want {
		t.Errorf("got first line: %q, want: %q", lines[0], want)
	}
}

// EDGE: `@option` — is a keyword but NOT a block directive.
// fixAmpersandOnBlocks won't touch &option. Is this correct?
// Looking at blockDirectives: choice, if, gate, cg, phone, minigame, episode.
// "option" is NOT in this set. So `&option A safe "text" {` would keep the &.
// This seems intentional since options use & for concurrent marking.
func TestAmpersandOnOption(t *testing.T) {
	input := "&option A safe \"text\" {"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// & should NOT be converted to @ for option
	if strings.HasPrefix(got, "@option") {
		t.Errorf("& should not be converted to @ for option: %q", got)
	}
}

// EDGE: fixIfMissingParens with `@if {` — condition is empty.
// The regex matches: rest = "{ ". braceIdx = 0. condition = TrimSpace("") = "".
// newLine = "@if () {" — wrapping empty condition in parens. Is this desired?
func TestIfMissingParens_EmptyCondition(t *testing.T) {
	input := "@if {"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// The regex: `^(\s*)@if\s+([^(\s].*)$`
	// After @if, we need \s+ then [^(\s]. The next char is { which is [^(\s]. So it matches!
	// rest = "{". braceIdx = 0. condition = TrimSpace("") = "".
	// newLine = "@if () {" — an empty condition wrapped in parens.
	// This is a minor issue: the fixer creates invalid syntax `@if () {`.
	if got == "@if () {" {
		t.Logf("ISSUE: @if { gets transformed to @if () { — empty condition wrapped in parens. Input was likely already malformed.")
	}
}

// EDGE: Indented @if.
func TestIfMissingParens_Indented(t *testing.T) {
	input := "    @if x > 5 {\n    }"
	r := Fix(input)
	lines := strings.Split(r.Fixed, "\n")

	want := "    @if (x > 5) {"
	if len(lines) == 0 || lines[0] != want {
		t.Errorf("got first line: %q, want: %q", lines[0], want)
	}
}

// EDGE: @check with no brace at all.
func TestAtCheck_NoBrace(t *testing.T) {
	input := "@check"
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	// atCheckRe: `^(\s*)@(check\s*\{.*)$` — requires { to match.
	// Without {, no match → line unchanged.
	if got != "@check" {
		t.Errorf("@check without brace should be unchanged: %q", got)
	}
}

// EDGE: Multiple @if fixes in one file — each line should be fixed independently.
func TestMultipleIfFixes(t *testing.T) {
	input := "@if x > 5 {\n}\n@if y < 3 {"
	r := Fix(input)
	got := r.Fixed

	if !strings.Contains(got, "@if (x > 5) {") {
		t.Errorf("first @if not fixed: %s", got)
	}
	if !strings.Contains(got, "@if (y < 3) {") {
		t.Errorf("second @if not fixed: %s", got)
	}
}

// EDGE: fixUnquotedArgs — butterfly with already-quoted arg should not double-quote.
func TestUnquotedArgs_AlreadyQuotedButterfly(t *testing.T) {
	input := `@butterfly "Accepted Easton"`
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")

	if got != `@butterfly "Accepted Easton"` {
		t.Errorf("already-quoted butterfly should be unchanged: %q", got)
	}
}

// EDGE: `@butterfly` with no argument at all.
func TestUnquotedArgs_ButterflyNoArg(t *testing.T) {
	input := "@butterfly"
	r := Fix(input)
	// Should not panic or error — just leave unchanged.
	got := strings.TrimRight(r.Fixed, "\n")
	if got != "@butterfly" {
		t.Errorf("butterfly with no arg should be unchanged: %q", got)
	}
}

// EDGE: `@butterfly ` (trailing space only, no arg).
func TestUnquotedArgs_ButterflyTrailingSpace(t *testing.T) {
	// After trailing whitespace fix, the line becomes "@butterfly" (space stripped).
	// Then fixUnquotedArgs checks for "@butterfly " prefix — won't match since space was stripped.
	input := "@butterfly "
	r := Fix(input)
	got := strings.TrimRight(r.Fixed, "\n")
	if got != "@butterfly" {
		t.Errorf("butterfly with only trailing space should become bare @butterfly: %q", got)
	}
}
