---
name: mss-scriptwriting
description: >
  How to write correct MoonShort Script (MSS) for MobAI interactive visual novels.
  Use this skill whenever generating, editing, or reviewing .md script files for
  the MobAI game engine — including Dramatizer output, Remix Executor output,
  or manual script authoring. Triggers on: MSS scripts, episode scripts,
  visual novel scripts, game scripts, Dramatizer output formatting,
  Remix script generation, "@episode", "@choice", "@gate", or any mention
  of the MobAI script format.
---

# Writing MoonShort Scripts

You are generating scripts for a mobile interactive visual novel player. Each script file is one episode — a self-contained unit of narrative that includes dialogue, visual staging, game mechanics (D20 checks, mini-games), and routing to the next episode.

The player experiences this as: tap to read dialogue → see characters enter/leave → occasionally make choices → sometimes play a mini-game → episode ends and routes to the next one.

Your scripts will be parsed by a Go interpreter that outputs JSON for the frontend. The interpreter is strict — syntax errors break the build. This guide teaches you to write scripts the interpreter accepts and the player renders well.

## File Basics

- Extension: `.md`
- Encoding: UTF-8
- Comments: `//` at line start
- Strings: double quotes `"..."`
- Blocks: `{ }` for nesting, indent for readability
- Every file has exactly one `@episode` root block

## Episode Structure

Every script follows this skeleton:

```
@episode <branch_key> "<title>" {

  // Narrative content (scenes, dialogue, choices, mini-games)
  // ...

  // Terminal — MUST be last. Exactly one of @gate or @ending.

  // Option A: routing to another episode
  @gate {
    @if (<condition>): @next <target>
    @else @if (<condition>): @next <target>
    @else: @next <fallback>
  }

  // Option B: terminal ending (no next episode)
  // @ending complete        // full story end
  // @ending to_be_continued  // cliffhanger, next chapter not yet written
  // @ending bad_ending       // bad-path terminus
}
```

**Every episode must end with either `@gate { ... }` or `@ending <type>`** — they are mutually exclusive. An episode with neither is a validator error.

The `branch_key` follows a strict addressing format. Read `references/addressing.md` for the full scheme.

## The Three Syntaxes in One File

MSS has three syntaxes that alternate freely:

**1. Directive lines** start with `@` — these control visuals, audio, game mechanics, and flow. Each `@` starts a new sequential step:
```
@bg set school_hallway fade
@mauricio show neutral_smirk at right
@music play tense_strings
```

**2. Concurrent directives** start with `&` — these execute simultaneously with the preceding `@` directive. Use `&` to group actions that should happen at the same time (e.g. scene setup):
```
@bg set school_hallway fade          // @ starts a new step group
&music crossfade tense_strings       // & concurrent with bg
&mauricio show neutral_smirk at right // & concurrent with bg
```

**3. Dialogue lines** start with a CHARACTER NAME in caps followed by colon — these are what the player reads:
```
MAURICIO: Hey, Butterfly.
NARRATOR: Senior year. Day one.
YOU: He hasn't called me that in eight years.
```

Three special names: `NARRATOR` (third-person scene narration), `YOU` (MC's inner monologue), and any other name (character dialogue). Character names in dialogue are case-insensitive with their `@` directive counterparts — `MAURICIO:` in dialogue = `@mauricio` in directives.

**Syntax sugar — look change + dialogue in one line:**
```
CHARACTER [look]: text
```
This is shorthand for `@character look look` followed by `CHARACTER: text`. Use it to keep the script tight when a character's look changes right before they speak:
```
MAURICIO [arms_crossed_angry]: Your call, Butterfly.
```

## Visual Directives

All visual directives use **object-action** order: `@<object> <action> [params]`.

**Scene setup** — use `&` to group the background, music, and character entrances that happen together at the start of a scene:

```
@bg set school_front fade
&music crossfade upbeat_school
&malia show neutral_flat at left
&josie show cheerful_wave at right
```

### Characters

```
@mauricio show neutral_smirk at right     // Enter stage
@mauricio look arms_crossed_angry         // Change look (instant)
@mauricio look arms_crossed_angry dissolve // Change look (crossfade)
@mauricio move to left                    // Slide to new position
@mauricio bubble heart                    // Emotion bubble (auto-disappears)
@mauricio hide                            // Exit (instant)
@mauricio hide fade                       // Exit (fade out)
```

**Positions:** `left` | `center` | `right` | `left_far` | `right_far`

**Look names** are semantic — they map to asset files via the interpreter. Use `snake_case`: `neutral_smirk`, `arms_crossed_angry`, `vulnerable_hopeful`. You can use any descriptive snake_case name.

**Bubble types:** `anger` `sweat` `heart` `question` `exclaim` `idea` `music` `doom` `ellipsis`

### Backgrounds

```
@bg set malias_bedroom_morning            // Default dissolve (0.5s)
@bg set school_front fade                 // Black screen transition (1s)
@bg set school_hallway cut                // Instant cut
@bg set dream_sequence slow               // Slow fade (2s, for mood)
```

### CG (Full-screen illustrations)

CG replaces the entire screen. Nest dialogue inside the block — it plays over the CG. When the block ends, the previous background and characters restore automatically.

```
@cg show window_stare fade {
  YOU: The city lights blurred through my tears.
  NARRATOR: She stood there for what felt like hours.
}
```

## Concurrency (@ vs &)

**`@`** = sequential. Each `@` directive starts a new step that executes after the previous one completes.

**`&`** = concurrent. A `&` directive joins the previous `@` directive's step group — they execute simultaneously.

**Dialogue** = always standalone. Each dialogue line is its own step that waits for the player to tap.

### Rules

- `&` CANNOT be used on block structures: `choice`, `cg`, `minigame`, `phone`, `if`, `gate`, `episode`. These always use `@`.
- A `&` line must follow an `@` line or another `&` line — it cannot appear at the start of a sequence with no preceding `@`.

### When to use `&`

- **Scene setup**: bg + music + characters entering at the start of a scene should fire together:
  ```
  @bg set school_cafeteria fade
  &music crossfade casual_lunch
  &mark show grin_confident at right
  &malia show neutral_flat at left
  ```
- **State changes that happen together**: multiple stat changes after a single event.

### When to use `@`

- **Standalone actions**: anything that needs to visually complete before the next step (e.g. a character entrance the player should notice individually).
- **Anything that needs to complete before the next step**: a background change that sets the mood before characters appear.

### `@pause for N`

Wait for N player clicks before advancing to the next step. Use this to give the player a moment to absorb a new scene before dialogue starts.

```
@bg set malias_bedroom_morning
&music play calm_morning
&malia show neutral_phone at left
@pause for 1
NARRATOR: Senior year. Day one.
```

The player sees the scene load (background, music, and character all at once), then must tap once before the narrator line appears. This creates a natural beat — the player takes in the visuals before reading.

## Audio

```
@music play calm_morning          // Start BGM (loops)
@music crossfade tense_strings    // Smooth transition to new BGM
@music fadeout                    // Fade current BGM to silence
@sfx play door_slam               // One-shot sound effect
```

## Phone / Messages

The phone overlay sits on top of everything. Keep messages short — they render in chat bubbles.

```
@phone show {
  @text from EASTON: Can we talk? I miss you.
  @text from EASTON: I know I messed up.
  @text to MAURICIO: How do you know where I live?
}
@phone hide
```

## Game Mechanics

### Mini-games

Mini-games interrupt the reading flow. The player plays an H5 game, gets a rating (S/A/B/C/D), and the rating modifies their next D20 check. You can also add bonus narrative per rating.

```
@minigame qte_challenge ATK {
  @on S {
    NARRATOR: Your reflexes are razor-sharp today.
  }
  @on A B {
    NARRATOR: Not bad. You kept up.
  }
  @on C D {
    NARRATOR: Sloppy. You're distracted.
  }
}
```

- `@on` blocks are **bonus narrative only** — they don't change the story route
- Multiple ratings on one `@on` line: `@on A B { }` means "A or B"
- The rating's mechanical effect (check modifier) is handled by the game engine, not the script

### Choices (D20 Checks)

Every episode has one choice point. Each option has an ID (A, B, C...) and a mode (`brave` = triggers D20 check, `safe` = no check).

```
@choice {
  @option A brave "Stand your ground." {
    check {
      attr: CHA
      dc: 12
    }
    @on success {
      @easton look relieved
      EASTON: Can I sit?
      MALIA: You have two minutes.
      @affection easton +2
      @butterfly "Accepted Easton's approach at the cafeteria"
    }
    @on fail {
      @easton look hurt
      MALIA: I... I can't do this.
      @butterfly "Tried to face Easton but lost courage"
    }
  }
  @option B safe "Have Mark make a scene." {
    @mark show grin_mischief at right
    @easton hide fade
    MARK: HEY EASTON! You want some of my mystery casserole?
    YOU: Thank god for Mark.
    @butterfly "Had Mark create a diversion to avoid Easton"
  }
}
```

**Rules for `brave` options:**
- Must contain a `check { attr: ... dc: ... }` block
- Must contain `@on success { }` and `@on fail { }` blocks
- Attribute names are freeform strings (the engine knows how to interpret them)

**Rules for `safe` options:**
- No check block, no @on blocks
- Content goes directly inside the option block

**Every outcome path should include:**
- `@butterfly` description (what happened as a result of this choice)
- Optionally: `@affection` if the choice changes a relationship

**Do NOT drop a `@signal mark` on every choice outcome.** Marks are only for key story points that a later `@if` or `@achievement.when` will actually query. See "State Changes" below.

## State Changes

These are declarations — the game engine handles the actual math.

```
@affection easton +2                                   // Character relationship
@butterfly "Accepted Easton's approach"                // Flavor memory for LLM route evaluation
@signal mark HIGH_HEEL_EP05                            // Key story point — queried later
@signal achievement HIGH_HEEL_WARRIOR                  // Unlock a declared achievement
```

> **Engine-managed values** (XP, SAN/HP, etc.) are set by the game engine, not scripts. You can reference them in `@if` conditions (e.g., `@if (san <= 20) { }`) but cannot modify them from scripts. The value names (san, xp, hp, etc.) are defined by the engine, not the script.

**`@signal <kind> <event>` — kind is mandatory.** Two kinds with distinct roles:

| kind | Role | Checkable in `@if`? |
|------|------|---------------------|
| `mark` | Persistent boolean flag. Engine stores it forever. `@if (NAME)` queries this store. **Use only for key story points** — hidden-route triggers and achievement `when` conditions. | **Yes** — becomes a `FlagCondition` in conditions |
| `achievement` | Directly unlock a declared `@achievement` by id. Engine fires its achievement pipeline (UI popup, analytics, unlock). | **No** — achievements are outbound notifications, not story state |

`event` can be a bare identifier or a double-quoted string.

### Mark discipline — marks are NOT wayposts

**Every `@signal mark X` must have a reader.** Either some later `@if (X)` branch depends on it, or some `@achievement`'s `when` condition references it. If nothing reads it, delete it. Cluttering the flag store dilutes the signal (pun intended) and confuses downstream tooling.

**Write the mark second.** Start from the reader:

1. Identify the payoff — a hidden line of dialogue, a secret scene, an arc achievement
2. Write the `@if` or `@achievement.when` that would unlock it
3. *Then* trace back to where the mark should be emitted

**Do NOT emit marks for things the engine already tracks:**

- ❌ `@signal mark EP01_COMPLETE` at the end of every episode — the engine knows which episodes the player cleared from episode_id itself
- ❌ `@signal mark CHOSE_OPTION_A` after a choice — the choice history is in the engine's choice log; gate `@if (A.success)` queries it directly
- ❌ `@signal mark AFFECTION_RAISED` — `@affection` already updated the number; `@if (affection.easton >= 5)` queries the value directly
- ❌ `@signal mark EASTON_ACKNOWLEDGED` as a character-moment marker with no follow-up — that's what `@butterfly` is for (LLM-evaluated, not boolean-queried)

**DO emit marks for these cases:**

- ✅ **Hidden-route trigger**: A key moment in EP05 that a later episode's dialogue references:
  ```
  // EP05
  MALIA: One quick step. My heel went straight through his shoe.
  @signal mark HIGH_HEEL_EP05

  // EP10 — only players who did EP05 see this line
  @if (HIGH_HEEL_EP05) {
    NARRATOR: She glanced down at her shoes. She remembered.
  }
  ```
- ✅ **Arc achievement trigger**: Two or more marks feed into an achievement's `when`:
  ```
  // EP05
  @signal mark HIGH_HEEL_EP05

  // EP24
  @signal mark HIGH_HEEL_EP24

  @achievement HIGH_HEEL_DOUBLE_KILL {
    name: "Heel Twice Over"
    rarity: epic
    description: "Once is improvisation. Twice is a signature move."
    when: (HIGH_HEEL_EP05 && HIGH_HEEL_EP24)
  }
  ```
- ✅ **Imperative achievement fire**: A singular narrative beat unlocks an achievement directly:
  ```
  YOU: I leaned in. He didn't pull back.
  @signal achievement FIRST_KISS
  ```

**Name all signal events and achievement ids in `SCREAMING_SNAKE_CASE` English.** Keeps the flag store grep-able and unambiguous across the pipeline.

**`@butterfly` is critical.** The game engine accumulates all butterfly records across episodes and uses LLM evaluation to determine which story branches unlock. Write clear, specific descriptions of what happened and what it reveals about the player's tendencies. Bad: "Made a choice." Good: "Showed vulnerability by accepting help from a former rival."

### Achievements — `@achievement` declarations

Achievements are declared with `@achievement` blocks at episode top-level (alongside narrative content). Field names and rarity vocabulary mirror the [`cdotlock/story-achievement-generator`](https://github.com/cdotlock/story-achievement-generator) skill output — so the generator's table can be mechanically dropped into MSS.

```
@achievement HIGH_HEEL_DOUBLE_KILL {
  name: "Heel Twice Over"
  rarity: epic
  description: "Once is improvisation. Twice is a signature move."
  when: (HIGH_HEEL_EP05 && HIGH_HEEL_EP24)
}
```

**Required fields (all four). Use English for all user-facing text:**

| Field | Type | Rule |
|-------|------|------|
| `name` | quoted string | English display name. Short, concrete, evocative (≤30 chars recommended) |
| `rarity` | identifier | One of: `uncommon` / `rare` / `epic` / `legendary`. **`common` is banned** — if every player gets it, it's not an achievement |
| `description` | quoted string | English DM-voice flavor text (1-2 sentences) |
| `when` | condition | Trigger condition using the same grammar as `@if`. Usually references `mark` flags |

**Two complementary trigger patterns:**

1. **Declarative (preferred for arcs)** — Drop `@signal mark EP05_HEEL` in Episode 5 and `@signal mark EP24_HEEL` in Episode 24. Declare the achievement with `when: (EP05_HEEL && EP24_HEEL)`. The engine watches marks and unlocks the achievement when the condition becomes true. Cross-episode arcs work naturally this way.

2. **Imperative (for single-moment achievements)** — Declare the achievement with a placeholder flag, then use `@signal achievement FIRST_KISS` at the exact narrative beat. The engine unlocks the achievement by id, bypassing `when` evaluation.

**Placement:** Put the `@achievement` declaration in whatever episode most naturally "owns" it — usually the episode where the achievement unlocks, or the one that introduces the arc. Order within the episode doesn't matter (achievements are hoisted to the episode level).

**Generating achievements:** The [`story-achievement-generator`](https://github.com/cdotlock/story-achievement-generator) skill reads a finished script set and produces a curated ~15-achievement table. Dramatizer should run it after scripts are complete, then each achievement is translated into an `@achievement` block (and any required `@signal mark` instrumentation) in the appropriate episode.

## Flow Control

### Conditional content

Use `@if` to show different content based on game state. **Parentheses `()` are mandatory** around the condition. The engine evaluates conditions at runtime. Use `@else @if` to chain multiple conditions.

```
@if (affection.easton >= 5 && CHA >= 14) {
  EASTON: You remembered.
} @else @if (affection.easton >= 3) {
  EASTON: ...I wasn't sure you'd come.
} @else {
  EASTON: ...Hey.
}

@if (san <= 20 || FAILED_TWICE) {
  YOU: I can barely keep it together.
}
```

**Condition types (5 kinds, all compiled to structured AST — the backend receives typed condition objects, not expression strings):**

| Type | Syntax | Example |
|------|--------|---------|
| flag | `SIGNAL_NAME` | `@if (EP01_COMPLETE) { }` |
| comparison | `<operand> <op> <integer>` | `@if (affection.easton >= 5) { }` or `@if (san <= 20) { }` |
| compound | `<expr> && <expr>` / `<expr> \|\| <expr>` | `@if (san <= 20 \|\| FAILED_TWICE) { }` |
| choice | `OPTION.result` | `@if (A.fail) { }` — result: `success` / `fail` / `any` |
| influence | `influence "desc"` or `"desc"` | `@if (influence "player showed empathy") { }` or `@if ("player showed empathy") { }` — both forms accepted |

**Comparison operand** on the left of `>=` / `<=` / etc. must be either `affection.<char>` (character-specific affection) or a bare `<name>` (engine-managed value like `san`, `CHA`). The right side **must be an integer literal** — expressions like `a >= b` or `affection.easton >= (5 + 3)` are rejected.

**Operators:** `>=` `<=` `>` `<` `==` `!=`
**Logic:** `&&` (and, binds tighter), `||` (or, binds looser). Use `( )` for explicit grouping.

### Labels and goto (advanced, avoid if possible)

```
@if (SKIP_FIGHT) {
  @goto AFTER_FIGHT
}
// ... fight content ...
@label AFTER_FIGHT
NARRATOR: The dust settled.
```

Only use `@goto`/`@label` when structured nesting can't express the flow. Prefer `@if`/`@else` for conditional content.

## Routing (Gate)

The `@gate` block at the end of every episode declares where the player goes next. Use `@if` → `@else @if` → `@else` chains. First match wins. `@else` is the fallback.

```
@gate {
  // Choice-based: route based on the player's choice (dot notation)
  @if (A.fail): @next main/bad/001:01

  // Influence-based: route based on accumulated butterfly effects (LLM evaluates)
  // Both forms accepted: influence "desc" or bare "desc"
  @else @if (influence "Player has repeatedly shown empathy toward Easton"): @next main/route/001:01

  // Fallback
  @else: @next main:02
}
```

**Choice condition format:** `<option_id>.<result>` (dot notation, no space)
- option_id: A, B, C... (matches the option's ID)
- result: `success` | `fail` | `any` (`any` = regardless of check outcome)
- Example: `A.fail`, `B.success`, `C.any`

**All 5 condition types work in gate:**
- Choice: `@if (A.fail): @next ...`
- Flag: `@if (EP01_COMPLETE): @next ...`
- Comparison: `@if (affection.easton >= 5): @next ...`
- Influence: `@if (influence "player showed empathy"): @next ...` or `@if ("player showed empathy"): @next ...`
- Compound: `@if (san <= 20 || FAILED_TWICE): @next ...`

**Influence condition:** A natural language description. Two syntax forms: `@if (influence "description")` or `@if ("description")` (bare string). At runtime, the engine feeds all accumulated `@butterfly` records to an LLM and asks whether the condition is satisfied.

**`@else` is mandatory.** Every episode with `@gate` must have a fallback route.

## Ending (Terminal Episodes)

Not every episode routes onward. Some episodes **end the story** — reach the credits, land on a cliffhanger, or terminate on a bad path. Use `@ending <type>` at episode-body level instead of `@gate`:

```
@episode main:15 "The End" {
  NARRATOR: The sun rose over a changed city.
  @ending complete
}
```

Three ending types:

| Type | Meaning | Typical placement |
|------|---------|-------------------|
| `complete` | Story fully ends — credits roll | Main ending episodes; true-end / happy-end |
| `to_be_continued` | Cliffhanger — next chapter not written yet | Last episode of the current release |
| `bad_ending` | Bad path terminus — player failed / died / broke the relationship | Episodes under `main/bad/...` |

**Rules:**

- **`@ending` and `@gate` are mutually exclusive.** An episode that ends with `@ending` cannot also have a `@gate` — the story has terminated.
- **Every episode still needs exactly one terminal.** Missing both is a validator error.
- Only the listed three types are accepted; anything else is a parse error.

Example bad-path terminal:

```
@episode main/bad/001:02 "She Never Came Home" {
  @bg set malias_bedroom_night fade
  NARRATOR: The door never opened again.
  @ending bad_ending
}
```

## Common Mistakes

1. **Missing terminal** — Every episode must end with exactly one of `@gate { ... }` or `@ending <type>`. Neither = `MISSING_TERMINAL` validator error.
2. **Both `@gate` and `@ending`** — The two are mutually exclusive: a routing block cannot coexist with a terminal ending. Pick one.
3. **Missing `@else`** — Gate block without fallback = interpreter error.
4. **Brave option without `check { }`** — Brave means D20 check. No check block = error.
5. **Brave option without both `@on success` and `@on fail`** — Both outcomes must be defined.
6. **`@goto` without matching `@label`** — The target label must exist in the same episode.
7. **Putting `@gate` anywhere except the end** — Gate must be the last thing in `@episode`.
8. **Using character names inconsistently** — `@mauricio show ...` and `MAURICIO:` must use the same name (case-insensitive).
9. **Forgetting `@butterfly` in choice outcomes** — Every choice outcome should record what happened for the influence system.
10. **Writing asset paths instead of semantic names** — Scripts use names like `classroom_morning`, not URLs or file paths. The interpreter handles mapping.
11. **Nested `@choice` blocks** — Only one `@choice` per episode. Multiple choices in one episode is not supported.
12. **Using `&` on block structures** — `&choice`, `&cg show`, `&minigame`, `&phone show`, `&if`, `&gate` are all errors. Block structures always use `@`.
13. **Forgetting to use `&` for scene setup** — When a scene starts with bg + music + character entrances, the first directive uses `@` and the rest should use `&` so they execute together. Writing all of them with `@` makes them sequential, which looks choppy.
14. **Trying to set engine values from scripts** — `@xp`, `@san` are not valid directives. The engine manages these values internally. Scripts can only check them in `@if` conditions (e.g., `@if (san <= 20) { }`), not modify them.
15. **`@if` without parentheses** — `@if condition { }` is a syntax error. Must be `@if (condition) { }`. Parentheses are mandatory for both body `@if` and gate `@if`.
16. **Using `choice.A.fail` in gate conditions** — The `choice.` prefix is dropped. Use `A.fail`, not `choice.A.fail`. Use dot notation: `A.fail`, not `A fail`.
17. **Invalid `@ending` type** — Only `complete`, `to_be_continued`, `bad_ending` are accepted. Any other identifier is a parse error.
18. **`@signal` without a kind** — `@signal <kind> <event>` requires a kind token. Write `@signal mark EVENT` (persistent flag) or `@signal achievement ID` (unlock a declared achievement). Nothing else parses.
19. **Orphan marks** — `@signal mark X` with no reader. If no `@if (X)` and no `@achievement.when` references `X`, delete the mark. Marks that no one reads are noise: they dilute the flag store and mislead downstream tools. Never emit marks for "episode completed", "chose option A", "affection raised", or anything else the engine already tracks.
20. **Using non-English names** — All `event` identifiers, `@achievement` ids, achievement `name`, and `description` strings must be English. Mixed-language ids cause grep/search failures and ambiguous meaning across the pipeline.
21. **Using `common` rarity on `@achievement`** — banned. Rarity must be one of `uncommon` / `rare` / `epic` / `legendary`. If every player gets it, it's not an achievement.
22. **Duplicate `@achievement` ids in one episode** — validator error. If the same achievement spans multiple episodes via `when` condition, declare it once in the most appropriate episode (usually the terminal one).
23. **Checking an `achievement` signal in `@if`** — `@if (FIRST_KISS)` after `@signal achievement FIRST_KISS` will always be false. Achievements are outbound notifications, not flags. For conditional logic, use `@signal mark`.

**Auto-repair:** The interpreter includes a fixer (`mss fix <file>`) that auto-repairs many of these mistakes: missing `@if` parentheses, `&` on block structures, `@check` → `check`, uppercase character names in `@affection`, trailing whitespace, unclosed blocks, and BOM/CRLF encoding issues. Always run the fixer before compiling if the script was generated by an LLM.

## Remix Scripts

Remix scripts use the exact same format. The only differences:

- **Branch key** uses `remix/<session_id>:01` addressing
- **Content starts after the choice point** — the user's input replaced the original `@choice`
- **May or may not return to main line** — `@else: @next main:02` to rejoin, or `@else: @next remix/<session_id>:02` to continue the remix branch

```
@episode remix/abc123:01 "Mark's Casserole Incident" {
  @mark show grin_mischief at right
  MARK: FOOD FIGHT!
  @sfx play crowd_noise
  NARRATOR: The cafeteria descended into chaos.
  @butterfly "Mark started a food fight in the cafeteria"

  @gate {
    @else: @next main:02
  }
}
```

## Pacing Guidelines

These aren't enforced by the interpreter, but they make for a good player experience:

- **Scene changes**: Use `@bg set ... fade` between locations. Don't change backgrounds mid-dialogue without a beat.
- **Character entrances**: Show one character at a time. Let the player tap once to see who appeared before they speak.
- **Inner monologue**: Use `YOU:` liberally — it keeps the player in the protagonist's head and builds emotional investment.
- **Phone messages**: Keep them short (under 20 words per message). They're in chat bubbles.
- **Bubbles**: Use sparingly for emotional punctuation, not on every line.
- **Music transitions**: `crossfade` between scenes, `fadeout` before silence or big moments.
- **One choice per episode**: This is a design constraint. The whole episode builds toward one decision point.
- **Use `@pause for 1` after scene setup** to let the player absorb the new scene before dialogue starts. A scene that jumps straight from setup to narration feels rushed.

## References

- `references/directive-table.md` — directive cheat sheet, quick lookup when writing scripts
- `references/addressing.md` — episode ID addressing rules for branch_key and file naming
- `references/MSS-SPEC.md` — **complete format specification** with JSON output structure, asset mapping schema, interpreter behavior, and Remix compatibility details. Read this when you need to understand how the interpreter processes scripts, what JSON the frontend receives, or how the asset mapping JSON works.
