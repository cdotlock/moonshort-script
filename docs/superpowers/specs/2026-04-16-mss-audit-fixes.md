# MSS Language Audit Fix Spec

**Date:** 2026-04-16
**Scope:** Full audit fix — all CRITICAL, HIGH, and key MEDIUM issues
**Goal:** Production-ready MSS for 100k+ users

---

## Module 1: Parser Robustness (11 fixes)

### C3: Nested parentheses in readCondition
- Add depth counter: `depth := 1` after consuming LPAREN
- Increment on LPAREN, decrement on RPAREN, break at depth==0

### C4: Empty condition rejection
- After collecting tokens, `len(toks)==0` → return error "empty condition"

### C1/C5: Condition classifier fixes
- Single STRING token → influence (no keyword needed)
- Accept `any` alongside `success`/`fail` in choice conditions

### H6: Stray `}` detection
- Not applicable at episode body level (RBRACE legitimately ends body)
- Fix: validate after parse that Gate is present (validator handles this)

### H7: Multi-error recovery
- Defer to future version — single-error mode is acceptable for v1
- Add TODO comment for future error recovery

### H8: Integer overflow
- Check `strconv.Atoi` error in parsePause and parseBraveOptionBody
- Return parse error on invalid integers

### H9: Empty block validation
- @choice with 0 options → parser error
- @gate with 0 routes → parser error
- @minigame with 0 @on blocks → parser error (or validator warning)

### H10: &NARRATOR error improvement
- In parseConcurrentDirective, after parseDirective fails:
  check if original token was UPPERCASE+COLON pattern → suggest removing &

### B6: Multiple @gate detection
- Track `gateFound bool` in parseEpisodeBody
- Second @gate → error "duplicate @gate block"

### Recursion depth
- Add `depth int` to Parser struct
- Increment in parseBlock, decrement on exit
- Limit: 50

### influence without quotes
- classifyCondition: `influence` + non-STRING → error hint

---

## Module 2: JSON Output Normalization (5 fixes)

### C2: Character name lowercase
- `emitDialogue`: `strings.ToLower(n.Character)`
- `emitTextMessage`: `strings.ToLower(n.Char)`

### H3: @else @if unwrap
- In `emitIf`: if `len(n.Else)==1` and it's an IfNode → emit as bare object, not array

### H4: Gate always present
- In `Emit`: always output `"gate"` key, use `nil` when `ep.Gate == nil`

### H5: on_results separator
- Keep space-separated (matches current output)
- Update docs to match

### Delta warning
- `parseDelta`: if `strconv.Atoi` fails, call `e.warn()`

---

## Module 3: Fixer Enhancement (7 fixes)

### C6: Brace counting skip dialogue
- Lines matching `^[A-Z][A-Z0-9_]+:` or `^//` → skip brace counting

### C10: & on block structures
- Detect `&choice`, `&if`, `&gate`, `&cg`, `&phone`, `&minigame`, `&episode`
- Replace & with @ and record fix

### H14: Missing @if parentheses
- Regex: `^\s*@if\s+(?!\()` → wrap condition in parens

### H15: @check → check
- Detect `^\s*@check\s*\{` → strip @

### H19: BOM/CRLF
- Strip UTF-8 BOM at start of Fix()
- Replace \r\n with \n before splitting

### Affection char name
- After `@affection`/`&affection`, lowercase the next word if uppercase

### Old format detection
- Detect @show, @hide, @expr, @move, @endep, @endbranch, @endchoice, @gain, @wait
- Add to errors: "old-format syntax detected, use MSS v2"

---

## Module 4: Validator Enhancement (7 fixes)

### H16: Duplicate option IDs
- In ChoiceNode validation: collect IDs, error on duplicate

### H17: Safe option no check/on
- If mode=="safe" and (Check!=nil or OnSuccess/OnFail non-empty) → error

### H18: Error message typo
- `@on_success` → `@on success`, `@on_fail` → `@on fail`

### Position whitelist
- Valid: left, center, right, left_far, right_far

### Transition whitelist
- Valid: fade, cut, slow, dissolve, "" (empty = default)

### Bubble type whitelist
- Valid: anger, sweat, heart, question, exclaim, idea, music, doom, ellipsis

### Option mode whitelist
- Valid: brave, safe

---

## Module 5: Test Coverage (target 80%+)

### Golden file test
- TestGoldenEp01: compile ep01.md → compare with ep01_output.json

### Full pipeline tests
- TestPipelineRoundtrip: compile all testdata/*.md files, verify no errors

### Emitter tests (per node type)
- Every AST node type gets at least one emitter test
- All 5 condition types tested
- Gate if/else chain tested
- Concurrent groups: single, multi, all-concurrent, no-concurrent
- Asset warning path tested

### Parser error tests
- 10+ TestParseError_* cases for malformed input
- Verify error messages are helpful

### Parser coverage
- @cg show, @label/@goto, flag/influence/compound conditions
- @else @if chaining, standalone @if (no else)

### Fixer tests
- & prefix dialogue, &butterfly/&signal
- Keyword casing, & on block structures
- @if missing parens, @check fix, BOM/CRLF

### Validator tests
- Valid brave option (positive case)
- Safe option with spurious check
- Nested validation
- Position/transition/bubble whitelist validation

### Fuzz tests
- FuzzParse: must not panic on arbitrary input
- FuzzLex: must not panic on arbitrary input

---

## Module 6: Documentation Sync

### MSS-SPEC.md
- Influence syntax: add `influence` keyword or accept bare string
- `any` check_result
- Condition examples update

### JSON-OUTPUT.md
- character always lowercase
- gate: null when absent
- else: can be array or bare if-object
- on_results keys use spaces

### SKILL.md
- Influence syntax fix
- "YAML" → "JSON" for asset mapping
- Condition table consistency

### directive-table.md
- "pose" → "look"

### Code comments
- "NRS" → "MSS" where applicable

---

## Module 7: Minor Fixes

- PauseNode: add ConcurrentFlag OR reject &pause in parser
- Emitter default case: add warn for unknown node types
- @signal: accept STRING token (strip quotes)
- Keyword/character name collision: validator warning
- emitCondition: warn on unknown condition type
