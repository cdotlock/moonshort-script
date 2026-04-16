# MSS Directive Quick Reference

## Structure Control

| Directive | Example |
|-----------|---------|
| `@episode <key> <title> { }` | `@episode main:01 "Butterfly" { }` |
| `@gate { }` | Must be last in episode |
| `@if (<cond>): @next <key>` | `@if (A fail): @next main/bad/001:01` |
| `@else: @next <key>` | `@else: @next main:02` |
| `@label <name>` | `@label AFTER_FIGHT` |
| `@goto <name>` | `@goto AFTER_FIGHT` |

## Visual (object-action)

| Directive | Example |
|-----------|---------|
| `@<char> show <pose> at <pos>` | `@mauricio show neutral_smirk at right` |
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
| `@xp <+/-N>` | `@xp +3` |
| `@san <+/-N>` | `@san -20` |
| `@affection <char> <+/-N>` | `@affection easton +2` |
| `@signal <event>` | `@signal EP01_COMPLETE` |
| `@butterfly <desc>` | `@butterfly "Accepted Easton's approach"` |

## Flow Control

| Directive | Example |
|-----------|---------|
| `@if <cond> { }` | `@if affection.easton >= 5 && CHA >= 14 { }` |
| `@else { }` | `} @else { }` |

Conditions: flags (`FLAG_NAME`), comparisons (`affection.char >= N`, `san <= N`, `ATK > 14`), logic (`&&`, `||`)
Operators: `>=` `<=` `>` `<` `==` `!=`
