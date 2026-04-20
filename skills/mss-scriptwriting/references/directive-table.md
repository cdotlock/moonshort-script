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
| `@cg show <name> [trans] { duration: ... content: "..." ... }` | See CG section below — `duration` and `content` fields are required |

Positions: `left` `center` `right` `left_far` `right_far`
Transitions: (none)=dissolve, `fade`, `cut`, `slow`
Bubbles: `anger` `sweat` `heart` `question` `exclaim` `idea` `music` `doom` `ellipsis`

### CG block fields (required)

```
@cg show window_stare fade {
  duration: medium
  content: "The camera opens on Malia's silhouette against the rain-streaked window, slow push-in as one tear tracks down."
  YOU: The city lights blurred through my tears.
}
```

- `duration` — one of `low` / `medium` / `high`. Length tier consumed by the video-generation pipeline (agent-forge) — scripts don't emit seconds.
- `content` — continuous English narrative describing camera + story beats during the CG. Plain, no purple prose.
- Body nodes after the two fields play during the CG, same as before.

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
| `@minigame <id> <ATTR> "<description>" { }` | `@minigame qte_challenge ATK "a quick reflex duel" { ... }` — description required |
| `@choice { }` | Contains @option blocks |
| `@option <ID> brave <text> { }` | Body must contain `check { }`; outcome branching via `@if (check.success) { } @else { }` |
| `@option <ID> safe <text> { }` | `@option B safe "Run" { ... }` |
| `check { attr: X  dc: N }` | Inside brave option |

Rating branching inside a minigame uses `@if (rating.<grade>) { }` trees, where `<grade>` is typically `S` / `A` / `B` / `C` / `D` (the language does not enforce a specific vocabulary).

Check-outcome branching inside a brave option uses `@if (check.success) { } @else { }`. The `check.success` / `check.fail` pseudo-identifier is valid only inside a brave option body.

## State Changes

| Directive | Example |
|-----------|---------|
| `@affection <char> <+/-N>` | `@affection easton +2` |
| `@signal <kind> <event>` | `@signal mark EP01_COMPLETE` — `<kind>` is mandatory; only `mark` is currently implemented. The kind slot is kept in the source grammar for future expansion |
| `@butterfly <desc>` | `@butterfly "Accepted Easton's approach"` |
| `@achievement <id> { name / rarity / description }` | Achievement unlock (block carries metadata; reaching this node fires it). Wrap in `@if (...) { ... }` for conditional triggers |

## Flow Control

| Directive | Example |
|-----------|---------|
| `@if (<cond>) { }` | `@if (affection.easton >= 5 && CHA >= 14) { }` — parens required |
| `@else @if (<cond>) { }` | `} @else @if (affection.easton >= 3) { }` — chained condition |
| `@else { }` | `} @else { }` |

Conditions (7 types, all fully parsed into structured AST — no raw expression strings in JSON output):

- choice: `A.fail` / `B.success` / `C.any`
- flag: `SIGNAL_NAME`
- comparison: `affection.<char> <op> <N>` or `<name> <op> <N>` (right side must be integer)
- influence: `"description"` or `influence "description"`
- compound: `<cond> && <cond>` / `<cond> || <cond>` (parens for grouping; `&&` binds tighter than `||`)
- check: `check.success` / `check.fail` — context-local, only valid inside a brave option body
- rating: `rating.<grade>` — context-local, only valid inside a minigame body

Operators: `>=` `<=` `>` `<` `==` `!=`
Logic: `&&` (and), `||` (or)

## Episode Termination

Every episode must end with exactly one of:

- `@gate { ... }` — route to another episode based on conditions
- `@ending <type>` — terminal state (no next episode)

The two are mutually exclusive. Missing both is a validator error.

## Achievements

One form: `@achievement <id> { name / rarity / description }`. The block carries the full metadata and reaching this node in execution fires the achievement. Conditional triggering is just standard `@if` wrapping.

```
@if (HIGH_HEEL_EP05 && HIGH_HEEL_EP24) {
  @achievement HIGH_HEEL_DOUBLE_KILL {
    name: "Heel Twice Over"
    rarity: epic
    description: "Once is improvisation. Twice is a signature move."
  }
}
```

- `rarity` must be one of `uncommon` / `rare` / `epic` / `legendary` (no `common`)
- All three fields are required; bare `@achievement <id>` without a block is a parse error
- Same id triggered from multiple places is fine — the engine dedupes by id at unlock time

## Signals

`@signal <kind> <event>` — kind is mandatory. Currently only `mark` is implemented:

- `@signal mark <event>` — persistent boolean flag. Engine stores it forever; `@if (NAME)` queries this store. Use only for key story points with a reader (a later `@if` branch or achievement trigger).

The kind slot is kept in the source grammar and AST so future kinds can be added without breaking scripts. Achievements are **not** a signal kind — use the dedicated `@achievement <id>` trigger instead.
