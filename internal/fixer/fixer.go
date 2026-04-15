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
var knownKeywords = map[string]bool{
	"bg":        true,
	"cg":        true,
	"phone":     true,
	"text":      true,
	"music":     true,
	"sfx":       true,
	"minigame":  true,
	"choice":    true,
	"option":    true,
	"xp":        true,
	"san":       true,
	"affection": true,
	"signal":    true,
	"butterfly": true,
	"if":        true,
	"else":      true,
	"label":     true,
	"goto":      true,
	"gates":     true,
	"gate":      true,
	"default":   true,
	"episode":   true,
	"on":        true,
	"check":     true,
}

// allCapsColonRe matches lines like "@NARRATOR: text" or "@MAURICIO: text".
var allCapsColonRe = regexp.MustCompile(`^@([A-Z][A-Z0-9_]+):`)

// Fix applies all auto-repairs to input and then runs error checks on the result.
func Fix(input string) *FixResult {
	r := &FixResult{}
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

		// 2. Extra @ on dialogue lines: @ALLCAPS: text → ALLCAPS: text
		if m := allCapsColonRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			if !knownKeywords[strings.ToLower(name)] {
				old := line
				line = line[1:] // strip leading @
				r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: removed @ from dialogue line @%s:", lineNum, name))
				_ = old
			}
		}

		// 3. Character name casing in directives: @Mauricio show → @mauricio show
		if strings.HasPrefix(line, "@") && !allCapsColonRe.MatchString("@"+line[1:]) {
			line = fixDirectiveCasing(line, lineNum, r)
		}

		// 4. Unquoted butterfly/signal arguments
		line = fixUnquotedArgs(line, lineNum, r)

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

// fixDirectiveCasing lowercases the word after @ if it's not a known keyword.
func fixDirectiveCasing(line string, lineNum int, r *FixResult) string {
	// line starts with "@"
	rest := line[1:]
	if len(rest) == 0 {
		return line
	}

	// Extract the first word after @
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
			r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: keyword casing @%s → @%s", lineNum, word, lower))
			return "@" + lower + after
		}
		return line
	}

	// Not a known keyword — this is a character name. Lowercase it.
	if word != lower {
		r.Fixes = append(r.Fixes, fmt.Sprintf("line %d: character name casing @%s → @%s", lineNum, word, lower))
		return "@" + lower + after
	}

	return line
}

// fixUnquotedArgs wraps unquoted arguments for @butterfly and @signal in double quotes.
func fixUnquotedArgs(line string, lineNum int, r *FixResult) string {
	trimmed := strings.TrimSpace(line)

	for _, directive := range []string{"@butterfly", "@signal"} {
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

		// For @signal, only quote if the argument contains spaces
		if directive == "@signal" && !strings.Contains(arg, " ") {
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
		// Ensure there's a blank line before the closing braces if the file doesn't end with one
		for i := 0; i < open; i++ {
			lines = append(lines, "}")
		}
	}

	return lines
}

// --- Error checks (post-fix validation on text) ---

// checkErrors scans the fixed text for structural problems that cannot be auto-fixed.
func checkErrors(r *FixResult) {
	lines := strings.Split(r.Fixed, "\n")

	checkMissingGates(lines, r)
	checkMissingDefault(lines, r)
	checkBraveOptions(lines, r)
	checkGotoLabels(lines, r)
	checkDuplicateOptionIDs(lines, r)
}

func checkMissingGates(lines []string, r *FixResult) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "@gates" || trimmed == "@gates {" || strings.HasPrefix(trimmed, "@gates ") {
			return
		}
	}
	r.Errors = append(r.Errors, "missing @gates block \u2014 every episode must declare routing")
}

func checkMissingDefault(lines []string, r *FixResult) {
	inGates := false
	depth := 0
	hasGates := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "@gates" || trimmed == "@gates {" || strings.HasPrefix(trimmed, "@gates ") {
			inGates = true
			hasGates = true
			if strings.Contains(trimmed, "{") {
				depth = 1
			}
			continue
		}

		if inGates {
			for _, ch := range trimmed {
				switch ch {
				case '{':
					depth++
				case '}':
					depth--
					if depth <= 0 {
						inGates = false
					}
				}
			}
			if inGates && (trimmed == "@default" || strings.HasPrefix(trimmed, "@default ") || strings.HasPrefix(trimmed, "@default\t")) {
				return
			}
		}
	}

	if hasGates {
		r.Errors = append(r.Errors, "missing @default in @gates \u2014 must specify fallback route")
	}
}

// braveOptionInfo tracks context while scanning for brave option problems.
type braveOptionInfo struct {
	id       string
	lineNum  int
	hasCheck bool
	hasOnSuccess bool
	hasOnFail    bool
}

func checkBraveOptions(lines []string, r *FixResult) {
	// Simple text scan: find @option X brave, then look for check {, @on success, @on fail
	// within the same block scope.
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

		// Scan forward within this option's block
		hasCheck := false
		hasOnSuccess := false
		hasOnFail := false
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
			if t == "@on success" || strings.HasPrefix(t, "@on success ") || strings.HasPrefix(t, "@on success\t") {
				hasOnSuccess = true
			}
			if t == "@on fail" || strings.HasPrefix(t, "@on fail ") || strings.HasPrefix(t, "@on fail\t") {
				hasOnFail = true
			}

			// End of block
			if started && depth <= 0 {
				break
			}
		}

		if !hasCheck {
			r.Errors = append(r.Errors, fmt.Sprintf("option %s is brave but has no check block \u2014 D20 check parameters required (line %d)", optionID, lineNum))
		}
		if !hasOnSuccess || !hasOnFail {
			r.Errors = append(r.Errors, fmt.Sprintf("option %s is brave but missing @on success/@on fail \u2014 both outcomes required (line %d)", optionID, lineNum))
		}
	}
}

func checkGotoLabels(lines []string, r *FixResult) {
	// Collect all labels
	labels := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@label ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				labels[parts[1]] = true
			}
		}
	}

	// Check all gotos
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@goto ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				target := parts[1]
				if !labels[target] {
					r.Errors = append(r.Errors, fmt.Sprintf("goto %s has no matching label (line %d)", target, i+1))
				}
			}
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
