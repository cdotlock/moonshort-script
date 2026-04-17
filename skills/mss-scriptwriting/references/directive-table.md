# MSS Directive Quick Reference

## Structure Control

| Directive | Example |
|-----------|---------|
| `@episode <key> <title> { }` | `@episode main:01 "Butterfly" { }` |
| `@gate { }` | Routing block — mutually exclusive with `@ending` |
| `@ending <type>` | Terminal marker — `complete` / `to_be_continued` / `bad_ending` (mutually exclusive with `@gate`) |
| `@if (<cond>): @next <key>` | `@if (A.fail): @next main/bad/001:01` — parens required, dot notation |
| `@else @if (<cond>): @next <key>` | `@else @if ("player showed empathy"): @next main/route/001:01` |
| `@else: @next <key>` | `@else: @next main:02` |
| `@label <name>` | `@label AFTER_FIGHT` |
| `@goto <name>` | `@goto AFTER_FIGHT` |
| `@pause for <N>` | `@pause for 1` — Wait for N player clicks before advancing |

## Concurrency

| Prefix | Meaning | Example |
|--------|---------|---------|
| `@` | Sequential (new group) | `@bg set classroom fade` |
| `&` | Concurrent (join group) | `&music play calm_morning` |

`&` cannot be used on: `choice`, `cg show`, `minigame`, `phone show`, `if`, `gate`, `episode`

## Visual (object-action)

| Directive | Example |
|-----------|---------|
| `@<char> show <look> at <pos>` | `@mauricio show neutral_smirk at right` |
| `@<char> hide [trans]` | `@mauricio hide fade` |
| `@<char> look <look> [trans]` | `@mauricio look angry dissolve` |
| `@<char> move to <pos>` | `@mauricio move to left` |
| `@<char> bubble <type>` | `@josie bubble heart` |
| `@bg set <name> [trans]` | `@bg set classroom fade` |
| `@cg show <name> [trans] { }` | `@cg show kiss_scene fade { ... }` |

Positions: `left` `center` `right` `left_far` `right_far`
Transitions: (none)=dissolve, `fade`, `cut`, `slow`
Bubbles: `anger` `sweat` `heart` `question` `exclaim` `idea` `music` `doom` `ellipsis`

## Dialogue

| Format | Example |
|--------|---------|
| `CHARACTER: text` | `MAURICIO: Hey, Butterfly.` |
| `NARRATOR: text` | `NARRATOR: Senior year. Day one.` |
| `YOU: text` | `YOU: He hasn't called me that in eight years.` |
| `CHARACTER [look]: text` | `MAURICIO [arms_crossed_angry]: Your call, Butterfly.` — dialogue + look change (syntax sugar) |

## Phone

| Directive | Example |
|-----------|---------|
| `@phone show { }` | Overlay appears |
| `@phone hide` | Overlay removed |
| `@text from <char>: content` | `@text from EASTON: I miss you.` |
| `@text to <char>: content` | `@text to MAURICIO: Leave me alone.` |

## Audio

| Directive | Example |
|-----------|---------|
| `@music play <name>` | `@music play calm_morning` |
| `@music crossfade <name>` | `@music crossfade tense_strings` |
| `@music fadeout` | Stop with fade |
| `@sfx play <name>` | `@sfx play door_slam` |

## Game Mechanics

| Directive | Example |
|-----------|---------|
| `@minigame <id> <ATTR> { }` | `@minigame qte_challenge ATK { ... }` |
| `@choice { }` | Contains @option blocks |
| `@option <ID> brave <text> { }` | `@option A brave "Fight" { check { ... } @on success { } @on fail { } }` |
| `@option <ID> safe <text> { }` | `@option B safe "Run" { ... }` |
| `check { attr: X  dc: N }` | Inside brave option |
| `@on <cond> { }` | `@on success { }` `@on fail { }` `@on S { }` `@on A B { }` |

## State Changes

| Directive | Example |
|-----------|---------|
| `@affection <char> <+/-N>` | `@affection easton +2` |
| `@signal <event>` | `@signal EP01_COMPLETE` — also stores as persistent boolean flag; check with `@if (EP01_COMPLETE) { }` |
| `@butterfly <desc>` | `@butterfly "Accepted Easton's approach"` |

## Flow Control

| Directive | Example |
|-----------|---------|
| `@if (<cond>) { }` | `@if (affection.easton >= 5 && CHA >= 14) { }` — parens required |
| `@else @if (<cond>) { }` | `} @else @if (affection.easton >= 3) { }` — chained condition |
| `@else { }` | `} @else { }` |

Conditions (5 types, all fully parsed into structured AST — no raw expression strings in JSON output):
- choice: `A.fail` / `B.success` / `C.any`
- flag: `SIGNAL_NAME`
- comparison: `affection.<char> <op> <N>` or `<name> <op> <N>` (right side must be integer)
- influence: `"description"` or `influence "description"`
- compound: `<cond> && <cond>` / `<cond> || <cond>` (parens for grouping; `&&` binds tighter than `||`)

Operators: `>=` `<=` `>` `<` `==` `!=`
Logic: `&&` (and), `||` (or)

## Episode Termination

Every episode must end with exactly one of:

- `@gate { ... }` — route to another episode based on conditions
- `@ending <type>` — terminal state (no next episode)

The two are mutually exclusive. Missing both is a validator error.
