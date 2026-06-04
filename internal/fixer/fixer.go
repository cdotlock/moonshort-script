// Package fixer applies text-level auto-repairs to MoonShort Script files.
// It works on raw lines (not the AST) so it can fix scripts that won't parse.
package fixer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// FixResult holds the output of a Fix operation.
type FixResult struct {
	Fixed  string   // the repaired text
	Fixes  []string // human-readable descriptions of what was fixed
	Errors []string // unfixable errors that require human intervention
}

// knownKeywords are directive keywords that follow @ and must NOT be lowercased.
//
// `on` is intentionally absent — outcome branching in brave options uses
// @if (check.success) / @else; a line starting with `@on` is caught by
// checkOldFormatSyntax and surfaces a hint pointing at the correct syntax.
var knownKeywords = map[string]bool{
	"bg":          true,
	"cg":          true,
	"phone":       true,
	"text":        true,
	"music":       true,
	"sfx":         true,
	"minigame":    true,
	"trick":       true,
	"choice":      true,
	"option":      true,
	"affection":   true,
	"signal":      true,
	"butterfly":   true,
	"if":          true,
	"else":        true,
	"gate":        true,
	"next":        true,
	"end":         true,
	"episode":     true,
	"check":       true,
	"pause":       true,
	"achievement": true,
}

// allCapsColonRe matches lines like "@NARRATOR: text" or "&NARRATOR: text".
var allCapsColonRe = regexp.MustCompile(`^[@&]([A-Z][A-Z0-9_]+):`)

// Fix applies all auto-repairs to input and then runs error checks on the result.
func Fix(input string) *FixResult {
	r := &FixResult{}

	// Normalize encoding: strip BOM and convert CRLF to LF.
	if strings.HasPrefix(input, "\xEF\xBB\xBF") {
		input = strings.TrimPrefix(input, "\xEF\xBB\xBF")
		r.Fixes = append(r.Fixes, "stripped UTF-8 BOM")
	}
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	lines := strings.Split(input, "\n")

	// Pass 1: per-line fixes
	for i, line := range lines {
		lineNum := i + 1

		// 1. Trailing whitespace
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
		if trimmed != line {
			r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: stripped trailing whitespace", lineNum))
			line = trimmed
		}

		// 2. Extra @/& on dialogue lines: @ALLCAPS: text → ALLCAPS: text
		if m := allCapsColonRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			if !knownKeywords[strings.ToLower(name)] {
				prefix := string(line[0])
				old := line
				line = line[1:] // strip leading @/&
				r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: removed %s from dialogue line %s%s:", lineNum, prefix, prefix, name))
				_ = old
			}
		}

		// 3. Character name casing in directives: @Mauricio/@mauricio, &Mauricio/&mauricio
		if (strings.HasPrefix(line, "@") || strings.HasPrefix(line, "&")) && !allCapsColonRe.MatchString(line) {
			line = fixDirectiveCasing(line, lineNum, r)
		}

		// 4. Unquoted butterfly/signal arguments
		line = fixUnquotedArgs(line, lineNum, r)

		// 5. Convert & to @ on block structure directives
		line = fixAmpersandOnBlocks(line, lineNum, r)

		// 6. Add missing parentheses to @if conditions
		line = fixIfMissingParens(line, lineNum, r)

		// 7. Strip @ from @check (should be bare check)
		line = fixAtCheck(line, lineNum, r)

		// 8. Lowercase character name in @affection/@affection directives
		line = fixAffectionCharCase(line, lineNum, r)

		lines[i] = line
	}

	// Pass 2: normalize blank lines (collapse 3+ consecutive blank lines to 2)
	lines = normalizeBlankLines(lines, r)

	// Pass 3: unclosed blocks — count braces
	lines = fixUnclosedBlocks(lines, r)

	r.Fixed = strings.Join(lines, "\n")

	// Pass 4: error checks on the fixed text
	checkErrors(r)

	return r
}

// fixDirectiveCasing lowercases the word after @/& if it's not a known keyword.
func fixDirectiveCasing(line string, lineNum int, r *FixResult) string {
	// line starts with "@" or "&"
	prefix := string(line[0])
	rest := line[1:]
	if len(rest) == 0 {
		return line
	}

	// Extract the first word after @/&
	idx := strings.IndexFunc(rest, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '{' || r == ':'
	})
	var word, after string
	if idx < 0 {
		word = rest
		after = ""
	} else {
		word = rest[:idx]
		after = rest[idx:]
	}

	if len(word) == 0 {
		return line
	}

	lower := strings.ToLower(word)

	// If it's a known keyword, leave as-is (keywords should already be lowercase
	// in the keyword set, so if the word already matches lowercase, nothing to do).
	if knownKeywords[lower] {
		// Still fix casing if the keyword itself was wrong-cased (e.g., @BG → @bg)
		if word != lower {
			r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: keyword casing %s%s → %s%s", lineNum, prefix, word, prefix, lower))
			return prefix + lower + after
		}
		return line
	}

	// Not a known keyword — this is a character name. Lowercase it.
	if word != lower {
		r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: character name casing %s%s → %s%s", lineNum, prefix, word, prefix, lower))
		return prefix + lower + after
	}

	return line
}

// fixUnquotedArgs wraps unquoted @butterfly descriptions in double quotes.
//
// @signal intentionally does not participate: its syntax is
// `@signal <kind> <event>` with a structured kind arg, so wrapping the
// remainder is not meaningful.
func fixUnquotedArgs(line string, lineNum int, r *FixResult) string {
	trimmed := strings.TrimSpace(line)

	for _, directive := range []string{"@butterfly", "&butterfly"} {
		if !strings.HasPrefix(trimmed, directive+" ") && !strings.HasPrefix(trimmed, directive+"\t") {
			continue
		}

		prefix := directive + " "
		// Find where the argument starts in the original line
		prefixIdx := strings.Index(line, directive)
		afterDirective := line[prefixIdx+len(directive):]

		// Skip whitespace to find the argument
		argStart := 0
		for argStart < len(afterDirective) && (afterDirective[argStart] == ' ' || afterDirective[argStart] == '\t') {
			argStart++
		}

		if argStart >= len(afterDirective) {
			break
		}

		arg := afterDirective[argStart:]

		// Already quoted — nothing to do
		if strings.HasPrefix(arg, "\"") {
			break
		}

		// Wrap in quotes
		leading := line[:prefixIdx+len(directive)] + afterDirective[:argStart]
		newLine := leading + "\"" + arg + "\""
		r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: quoted %s argument: %s → %s\"%s\"", lineNum, directive, prefix+arg, prefix, arg))
		return newLine
	}

	return line
}

// normalizeBlankLines collapses runs of 3+ consecutive blank lines into exactly 2.
func normalizeBlankLines(lines []string, r *FixResult) []string {
	var result []string
	blankCount := 0
	collapsed := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			} else {
				collapsed = true
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	if collapsed {
		r.Fixes = append(r.Fixes, "normalized consecutive blank lines (3+ → 2)")
	}

	return result
}

// fixUnclosedBlocks counts { and } and appends missing closing braces.
func fixUnclosedBlocks(lines []string, r *FixResult) []string {
	open := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comment lines
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		// Skip dialogue lines (ALLCAPS: text) — they may contain { } in text
		if isDialogueLine(trimmed) {
			continue
		}
		for _, ch := range line {
			switch ch {
			case '{':
				open++
			case '}':
				open--
			}
		}
	}

	if open > 0 {
		r.Fixes = append(r.Fixes, fmt.Sprintf("appended %d missing closing }", open))
		for i := 0; i < open; i++ {
			lines = append(lines, "}")
		}
	}

	return lines
}

// isDialogueLine returns true if the line looks like ALLCAPS: text or ALLCAPS [expr]: text.
func isDialogueLine(line string) bool {
	// Match pattern: optional leading spaces, then ALLCAPS IDENT, then COLON
	for i, ch := range line {
		if ch == ':' && i > 0 {
			name := strings.TrimSpace(line[:i])
			// Strip [expr] part if present
			if bracketIdx := strings.Index(name, "["); bracketIdx >= 0 {
				name = strings.TrimSpace(name[:bracketIdx])
			}
			if len(name) == 0 {
				return false
			}
			allUpper := true
			for _, r := range name {
				if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
					allUpper = false
					break
				}
			}
			return allUpper
		}
	}
	return false
}

// --- Error checks (post-fix validation on text) ---

// checkErrors scans the fixed text for structural problems that cannot be auto-fixed.
func checkErrors(r *FixResult) {
	lines := strings.Split(r.Fixed, "\n")

	checkMissingGate(lines, r)
	checkBraveOptions(lines, r)
	checkDuplicateOptionIDs(lines, r)
	checkOldFormatSyntax(lines, r)
}

func checkMissingGate(lines []string, r *FixResult) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "@gate" || trimmed == "@gate {" || strings.HasPrefix(trimmed, "@gate ") {
			return
		}
	}
	r.Errors = append(r.Errors, "missing @gate block — every episode must declare routing")
}

func checkBraveOptions(lines []string, r *FixResult) {
	// Simple text scan: find @option X brave, then look for a check block
	// within the same option body. Outcome completeness is not enforced —
	// authors express success/fail branching via @if (check.success) /
	// @else, and a missing @else is a story choice, not a syntax error.
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "@option ") {
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 3 || parts[2] != "brave" {
			continue
		}

		optionID := parts[1]
		lineNum := i + 1

		hasCheck := false
		depth := 0
		started := false

		for j := i; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			for _, ch := range t {
				switch ch {
				case '{':
					depth++
					started = true
				case '}':
					depth--
				}
			}

			if strings.HasPrefix(t, "check") && strings.Contains(t, "{") {
				hasCheck = true
			}

			if started && depth <= 0 {
				break
			}
		}

		if !hasCheck {
			r.Errors = append(r.Errors, fmt.Sprintf("option %s is brave but has no check block — D20 check parameters required (line %d)", optionID, lineNum))
		}
	}
}

func checkDuplicateOptionIDs(lines []string, r *FixResult) {
	// Track which @choice block we're in and the option IDs within it.
	type choiceCtx struct {
		depth   int
		options map[string]bool
	}

	var stack []choiceCtx

	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect @choice
		if trimmed == "@choice" || trimmed == "@choice {" || strings.HasPrefix(trimmed, "@choice ") {
			// Push a new choice context; we'll set its depth when we see the opening {
			stack = append(stack, choiceCtx{depth: depth + 1, options: make(map[string]bool)})
		}

		for _, ch := range trimmed {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				// Pop any choice contexts that have ended
				for len(stack) > 0 && depth < stack[len(stack)-1].depth {
					stack = stack[:len(stack)-1]
				}
			}
		}

		// Check @option within the current choice
		if strings.HasPrefix(trimmed, "@option ") && len(stack) > 0 {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				id := parts[1]
				ctx := &stack[len(stack)-1]
				if ctx.options[id] {
					r.Errors = append(r.Errors, fmt.Sprintf("duplicate option ID: %s", id))
				}
				ctx.options[id] = true
			}
		}
	}
}

// blockDirectives are directives that start blocks and must use @ not &.
// CG is a leaf directive (no body), so it is not included here.
var blockDirectives = map[string]bool{
	"choice": true, "if": true, "gate": true,
	"phone": true, "minigame": true, "episode": true,
}

// fixAmpersandOnBlocks converts & to @ on block structure directives.
func fixAmpersandOnBlocks(line string, lineNum int, r *FixResult) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "&") {
		return line
	}
	rest := trimmed[1:]
	// Extract the first word
	idx := strings.IndexFunc(rest, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '{'
	})
	var word string
	if idx < 0 {
		word = rest
	} else {
		word = rest[:idx]
	}
	if blockDirectives[strings.ToLower(word)] {
		// Find the & in the original line and replace with @
		ampIdx := strings.Index(line, "&")
		newLine := line[:ampIdx] + "@" + line[ampIdx+1:]
		r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: replaced & with @ on block directive &%s", lineNum, word))
		return newLine
	}
	return line
}

// ifNoParensRe matches @if followed by a non-paren character (missing parentheses).
var ifNoParensRe = regexp.MustCompile(`^(\s*)@if\s+([^(\s].*)$`)

// fixIfMissingParens wraps @if conditions in parentheses when missing.
func fixIfMissingParens(line string, lineNum int, r *FixResult) string {
	m := ifNoParensRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	indent := m[1]
	rest := m[2]
	// rest should end with { — find it
	braceIdx := strings.LastIndex(rest, "{")
	if braceIdx < 0 {
		// No brace on this line — could be multi-line, skip
		return line
	}
	condition := strings.TrimSpace(rest[:braceIdx])
	trailing := rest[braceIdx:]
	newLine := indent + "@if (" + condition + ") " + trailing
	r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: added missing parentheses to @if condition", lineNum))
	return newLine
}

// atCheckRe matches @check optionally followed by { or whitespace.
var atCheckRe = regexp.MustCompile(`^(\s*)@(check\s*\{.*)$`)

// fixAtCheck strips the @ prefix from @check inside brave options.
func fixAtCheck(line string, lineNum int, r *FixResult) string {
	m := atCheckRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	newLine := m[1] + m[2]
	r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: removed @ from @check (should be bare 'check')", lineNum))
	return newLine
}

// affectionCharRe matches @affection or &affection followed by an uppercase character name.
// The trailing delta (e.g. +2) is optional so we also match `@affection EASTON` without a delta.
var affectionCharRe = regexp.MustCompile(`^(\s*[@&]affection\s+)([A-Z][A-Z0-9_]+)(\s+.*)?$`)

// fixAffectionCharCase lowercases the character name in @affection directives.
func fixAffectionCharCase(line string, lineNum int, r *FixResult) string {
	m := affectionCharRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	lower := strings.ToLower(m[2])
	if lower == m[2] {
		return line
	}
	newLine := m[1] + lower + m[3]
	r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: lowercased character name in affection: %s → %s", lineNum, m[2], lower))
	return newLine
}

// oldFormatKeywords are keywords the fixer detects and rejects with a hint
// pointing at the correct MSS syntax. These catch common LLM mistakes
// where the generator produces a directive name drawn from an adjacent
// dialect rather than MSS itself.
//
// Char visual directives (@show / @hide / @expr / @look / @move / @at) are
// rejected here because the new AST collapses character visuals into a
// single `@<char> <pose>` form and hides are implicit. Block-form @cg with
// duration/content is rejected because CG is now a leaf. @label / @goto
// are gone because in-episode branching uses @if / @else and routing uses
// @gate { @next … }. Standalone @ending is gone because endings are gate
// leaves (@gate { @end <type> }).
var oldFormatKeywords = map[string]string{
	"@show":      "use @<char> <pose> (e.g. @malia worried)",
	"@hide":      "remove — character auto-hides when another speaks or NARRATOR/YOU appears (§3.3)",
	"@expr":      "use @<char> <pose>",
	"@look":      "use @<char> <pose>",
	"@move":      "removed — positions are fixed (MC left, others right)",
	"@at":        "removed — positions are fixed",
	"@label":     "removed — use @if/@else for in-episode branching",
	"@goto":      "removed — use @if/@else for in-episode branching",
	"@ending":    "use @gate { @end <type> }",
	"@endep":     "use closing } for @episode block",
	"@endbranch": "use closing } for option blocks",
	"@endchoice": "use closing } for @choice block",
	"@endgroup":  "use & prefix for concurrent directives",
	"@branch":    "use @option inside @choice block",
	"@gain":      "use @affection",
	"@wait":      "use @pause for N",
	"@timeskip":  "removed — use @bg set with transition",
	"@group":     "use & prefix for concurrent directives",
	"@on":        "not part of MSS syntax — use @if (check.success) / @else inside brave options",
}

// charLegacySuffixRe matches `@<char> show|hide|look|move ...` and `@<char> at ...`.
// The new AST uses `@<char> <pose>` directly; legacy verbs as the pose word
// are a strong signal the author / LLM is reaching for the old dialect.
var charLegacySuffixRe = regexp.MustCompile(`^\s*@[a-z][a-z0-9_]*\s+(show|hide|look|move|at)(\s|$)`)

// phoneLegacyRe matches `@phone show` / `@phone hide` — the new AST uses
// a single `@phone { ... }` block with implicit lifecycle.
var phoneLegacyRe = regexp.MustCompile(`^\s*@phone\s+(show|hide)(\s|$|\{)`)

// musicLegacyPlayRe matches `@music play <name>` and `@music crossfade <name>`.
var musicLegacyPlayRe = regexp.MustCompile(`^\s*@music\s+(play|crossfade)(\s|$)`)

// musicLegacyFadeoutRe matches bare `@music fadeout`.
var musicLegacyFadeoutRe = regexp.MustCompile(`^\s*@music\s+fadeout(\s|$)`)

// sfxLegacyPlayRe matches `@sfx play <name>`.
var sfxLegacyPlayRe = regexp.MustCompile(`^\s*@sfx\s+play(\s|$)`)

// pauseLegacyForRe matches `@pause for <N>` — pause is now a leaf with
// no duration argument; longer waits are expressed by repeating @pause.
var pauseLegacyForRe = regexp.MustCompile(`^\s*@pause\s+for(\s|$)`)

// ifStringLiteralRe matches `@if ("...")` where the condition is a bare
// double-quoted string literal — the new condition grammar does not
// accept string-literal conditions.
var ifStringLiteralRe = regexp.MustCompile(`^\s*@if\s*\(\s*"`)

// ifInfluenceRe matches `@if (influence ...)`. The influence condition
// was removed when the comparison grammar was generalized.
var ifInfluenceRe = regexp.MustCompile(`^\s*@if\s*\(\s*influence(\s|\)|>|<|=|!)`)

// cgBlockFormRe matches the legacy CG block form, where @cg show opens a
// block with `duration` / `content` children. CG is now a leaf:
// `@cg <name> "<content>"`.
var cgBlockFormRe = regexp.MustCompile(`^\s*@cg\s+show\b`)

func checkOldFormatSyntax(lines []string, r *FixResult) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		// Keyword-prefix matches (whole-directive replacements).
		matched := false
		for keyword, hint := range oldFormatKeywords {
			if strings.HasPrefix(trimmed, keyword+" ") || strings.HasPrefix(trimmed, keyword+"\t") || trimmed == keyword {
				r.Errors = append(r.Errors, fmt.Sprintf("line %d: old-format syntax %q detected — %s", lineNum, keyword, hint))
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Legacy `@<char> show|hide|look|move|at ...` form.
		if charLegacySuffixRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: legacy char directive — use @<char> <pose>; hides are implicit (§3.3)", lineNum))
			continue
		}

		// Legacy `@phone show` / `@phone hide`.
		if phoneLegacyRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: phone now uses @phone { ... } block — open/close lifecycle is implicit", lineNum))
			continue
		}

		// Legacy `@music play <name>` / `@music crossfade <name>`.
		if musicLegacyPlayRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: use @music <name> — the engine decides whether to fade in or cross-fade", lineNum))
			continue
		}

		// Legacy `@music fadeout`.
		if musicLegacyFadeoutRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: use @music stop", lineNum))
			continue
		}

		// Legacy `@sfx play <name>`.
		if sfxLegacyPlayRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: use @sfx <name>", lineNum))
			continue
		}

		// Legacy `@pause for <N>`.
		if pauseLegacyForRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: @pause is single-click only — repeat the directive for longer pauses", lineNum))
			continue
		}

		// String-literal `@if (\"…\")`.
		if ifStringLiteralRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: string literal not allowed as @if condition", lineNum))
			continue
		}

		// Removed `influence` condition.
		if ifInfluenceRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: influence condition removed — use affection comparisons or flag/value conditions", lineNum))
			continue
		}

		// Legacy block-form `@cg show ...`.
		if cgBlockFormRe.MatchString(line) {
			r.Errors = append(r.Errors, fmt.Sprintf("line %d: @cg is a leaf directive — use @cg <name> \"<content>\"", lineNum))
			continue
		}
	}
}
