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

CG replaces the entire screen. It's not a still image — downstream the `cg_show` step is generated as a short video by agent-forge. MSS is flow-control plus content, so the script has to carry the narrative material the video pipeline needs.

Two fields are **required** at the top of a CG block, before any body nodes:

- **`duration:`** — one of `low` / `medium` / `high`. A length tier, not seconds. agent-forge decides the actual timing.
- **`content:`** — a quoted English string describing the CG's camera and story beats. Plain, concrete, no purple prose — what happens, what the camera does, what the frame emphasises.

Body nodes after the two fields play during the CG, same as before. When the block ends, the previous background and characters restore automatically.

```
@cg show window_stare fade {
  duration: medium
  content: "The camera opens on Malia's silhouette against the rain-streaked window. Slow push-in on her eyes — one tear tracks down, catching the cold blue of the skyline. Her reflection doubles her, ghost-like, in the glass."
  YOU: The city lights blurred through my tears.
  NARRATOR: She stood there for what felt like hours.
}
```

A `@cg show` block without both fields is a parse error. Validator checks a duration whitelist and rejects empty `content`.

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

Mini-games interrupt the reading flow. The player plays an H5 game, gets a rating (typically S / A / B / C / D), and the rating modifies their next D20 check. You can also branch bonus narrative per rating.

```
@minigame qte_challenge ATK "a quick hallway partner-assignment check where Malia matches Mauricio's beat or drops the rhythm" {
  @if (rating.S) {
    NARRATOR: Your reflexes are razor-sharp today.
  } @else @if (rating.A || rating.B) {
    NARRATOR: Not bad. You kept up.
  } @else {
    NARRATOR: Sloppy. You're distracted.
  }
}
```

- The third positional argument (a quoted English string) is a **required** short description tying the minigame to the scene. Mini-games are picked from a pre-built library (not generated), so the description is just for narrative glue — one sentence is plenty, don't over-write.
- Rating branching uses a standard `@if (rating.<grade>) { }` tree. `rating.S`, `rating.A`, etc. are pseudo-identifiers valid only inside a `@minigame` body. Use `||` to group grades (`@if (rating.A || rating.B)`).
- Bonus narrative under a rating branch doesn't change the story route. The rating's mechanical effect (check modifier) is handled by the game engine, not the script.
- Minigame body is optional — an empty `{ }` is valid if you don't need per-rating narrative.

### Choices (D20 Checks)

Every episode has one choice point. Each option has an ID (A, B, C...) and a mode (`brave` = triggers D20 check, `safe` = no check).

```
@choice {
  @option A brave "Stand your ground." {
    check {
      attr: CHA
      dc: 12
    }
    @if (check.success) {
      @easton look relieved
      EASTON: Can I sit?
      MALIA: You have two minutes.
      @affection easton +2
      @butterfly "Accepted Easton's approach at the cafeteria"
    } @else {
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
- Must contain a `check { attr: ... dc: ... }` block.
- Outcome branching uses a standard `@if (check.success) { } @else { }` tree. `check.success` / `check.fail` are context-local pseudo-identifiers valid only inside a brave option body.
- A missing `@else` is a narrative choice, not a syntax error — the validator does not require both branches to be populated. If you leave fail implicit, make sure the story still reads.
- Attribute names are freeform strings (the engine knows how to interpret them).

**Rules for `safe` options:**
- No check block, no outcome branching — safe options bypass the D20 entirely.
- Content goes directly inside the option block as plain body nodes.

**Every outcome path should include:**
- `@butterfly` description (what happened as a result of this choice)
- Optionally: `@affection` if the choice changes a relationship

**Do NOT drop a `@signal mark` on every choice outcome.** Marks are only for key story points that a later `@if` branch or an `@achievement` unlock will actually query. See "State Changes" below.

## State Changes

These are declarations — the game engine handles the actual math.

```
@affection easton +2                                   // Character relationship
@butterfly "Accepted Easton's approach"                // Flavor memory for LLM route evaluation
@signal mark HIGH_HEEL_EP05                            // Key story point — queried later
@signal int rejections +1                              // Persistent integer counter — free to mutate
@signal int rejections = 0                             // Explicit reset / initialization
@achievement HIGH_HEEL_WARRIOR {                       // Achievement unlock
  name: "Heel as Weapon"
  rarity: rare
  description: "You turned an accessory into a warning."
}
```

> **Engine-managed values** (XP, SAN/HP, etc.) are set by the game engine, not scripts. You can reference them in `@if` conditions (e.g., `@if (san <= 20) { }`) but cannot modify them from scripts. The value names (san, xp, hp, etc.) are defined by the engine, not the script.

**`@signal <kind> <...>` — kind is mandatory.** Two kinds are implemented:

- `@signal mark <event>` — persistent boolean flag. Use sparingly; every mark must have a reader (see below).
- `@signal int <name> <op> <value>` — persistent integer counter. Free to mutate (`= N`, `+N`, `-N`). Read via `@if (name >= N)` comparison.

Achievements are **not** a signal kind — use `@achievement <id> { ... }` for those.

| kind | Role | Checkable in `@if`? |
|------|------|---------------------|
| `mark` | Persistent boolean flag. Engine stores it forever. `@if (NAME)` queries this store. **Use only for key story points** — hidden-route triggers and achievement-unlock guards. | **Yes** — becomes a `FlagCondition` in conditions |
| `int` | Persistent integer variable. Engine stores across episodes. `@if (NAME <cmp> N)` queries the value via comparison. Free to mutate as often as needed — counters are the whole point. | **Yes** — becomes a `ComparisonCondition` with `left.kind="value"` |

For `mark`, `event` can be a bare identifier or a double-quoted string. For `int`, `name` is a bare `snake_case` identifier.

### Mark discipline — marks are NOT wayposts

**Every `@signal mark X` must have a reader.** Either some later `@if (X)` branch depends on it, or some `@if (X && ...) { @achievement ID { ... } }` unlock guard references it. If nothing reads it, delete it. Cluttering the flag store dilutes the signal (pun intended) and confuses downstream tooling.

**`@signal int` is not mark.** Counters are expected to be written often — `rejections +1` every time the player rejects Easton is exactly the point. The "marks are precious" discipline does NOT apply. Use `@signal int` whenever you need "if player did X at least N times" or "N-of-M threshold" branching. Prefer `@signal int counter +1` + `@if (counter >= N)` over stacking multiple boolean marks.

Guidelines for ints:
- Name in `snake_case` (e.g. `rejections`, `brave_count`, `times_met_easton`).
- Avoid names that look like engine values (`san`, `cha`, `hp`, `xp`) — the validator will reject these.
- `@signal int x = 0` is an unconditional assignment — if placed somewhere the player can revisit, it will reset the counter. That is the author's responsibility; the engine does not protect.
- For first-time reads, the engine treats undeclared variables as 0, so `@signal int x +1` and `@if (x >= 1)` work without any prior `= 0`.

**Write the mark second.** Start from the reader:

1. Identify the payoff — a hidden line of dialogue, a secret scene, an arc achievement
2. Write the `@if` guard that would unlock it
3. *Then* trace back to where the mark should be emitted

**Do NOT emit marks for things the engine already tracks:**

- ❌ `@signal mark EP01_COMPLETE` at the end of every episode — the engine knows which episodes the player cleared from episode_id itself
- ❌ `@signal mark CHOSE_OPTION_A` after a choice — the choice history is in the engine's choice log; gate `@if (A.success)` queries it directly
- ❌ `@signal mark AFFECTION_RAISED` — `@affection` already updated the number; `@if (affection.easton >= 5)` queries the value directly
- ❌ `@signal mark EASTON_ACKNOWLEDGED` as a character-moment marker with no follow-up — that's what `@butterfly` is for (LLM-evaluated, not boolean-queried)
- ❌ Don't use `@signal mark COUNTER_HIT_3` to track thresholds — that's what `@signal int` is for. Use `@signal int counter +1` then `@if (counter >= 3)`.

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
- ✅ **Arc achievement trigger**: Two or more marks feed into an achievement unlock guard:
  ```
  // EP05
  MALIA: One quick step. My heel went straight through his shoe.
  @signal mark HIGH_HEEL_EP05

  // EP24 — the second beat, and the place where the achievement fires
  MALIA: And again. Same shoe, same precision.
  @signal mark HIGH_HEEL_EP24
  @if (HIGH_HEEL_EP05 && HIGH_HEEL_EP24) {
    @achievement HIGH_HEEL_DOUBLE_KILL {
      name: "Heel Twice Over"
      rarity: epic
      description: "Once is improvisation. Twice is a signature move."
    }
  }
  ```
- ✅ **Single-beat achievement**: A singular narrative moment unlocks an achievement directly:
  ```
  YOU: I leaned in. He didn't pull back.
  @achievement FIRST_KISS {
    name: "First Kiss"
    rarity: uncommon
    description: "You stopped calculating and let it happen."
  }
  ```

**Name all signal events and achievement ids in `SCREAMING_SNAKE_CASE` English.** Keeps the flag store grep-able and unambiguous across the pipeline.

**Where to put the `@if ... { @achievement ID }` guard** for arc achievements: put it after the *last* mark in the chain is emitted — the guard only fires if every required mark is already set. Putting it at the start of an episode works too but is redundant (you'd re-check every time). One clean guard beats multiple scattered checks.

**`@butterfly` is critical.** The game engine accumulates all butterfly records across episodes and uses LLM evaluation to determine which story branches unlock. Write clear, specific descriptions of what happened and what it reveals about the player's tendencies. Bad: "Made a choice." Good: "Showed vulnerability by accepting help from a former rival."

### Achievements — `@achievement`

One directive, one form: `@achievement <id> { name / rarity / description }`. The block carries the full metadata and reaching this node in execution fires the achievement. Conditional triggering is just standard `@if` wrapping — there is no separate declaration step.

```
@if (HIGH_HEEL_EP05 && HIGH_HEEL_EP24) {
  @achievement HIGH_HEEL_DOUBLE_KILL {
    name: "Heel Twice Over"
    rarity: epic
    description: "Once is improvisation. Twice is a signature move."
  }
}
```

**Required fields (all three). Use English for all user-facing text:**

| Field | Type | Rule |
|-------|------|------|
| `name` | quoted string | English display name. Short, concrete, evocative (≤30 chars recommended) |
| `rarity` | identifier | One of: `uncommon` / `rare` / `epic` / `legendary`. **`common` is banned** — if every player gets it, it's not an achievement |
| `description` | quoted string | English DM-voice flavor text (1-2 sentences) |

**Trigger placement patterns:**

1. **Single-beat achievement** — fire inline at the moment the achievement makes narrative sense:
   ```
   YOU: I leaned in. He didn't pull back.
   @achievement FIRST_KISS {
     name: "First Kiss"
     rarity: uncommon
     description: "You stopped calculating and let it happen."
   }
   ```
2. **Arc achievement** — wrap in an `@if` that checks every required mark, placed right after the last mark in the chain is emitted:
   ```
   // Episode 24
   @signal mark HIGH_HEEL_EP24
   @if (HIGH_HEEL_EP05 && HIGH_HEEL_EP24) {
     @achievement HIGH_HEEL_DOUBLE_KILL {
       name: "Heel Twice Over"
       rarity: epic
       description: "Once is improvisation. Twice is a signature move."
     }
   }
   ```

**Same id in multiple places is fine** — the engine dedupes by id at unlock time. If you deliberately echo the same achievement from several narrative branches, write it out each time.

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

**Condition types (7 kinds, all compiled to structured AST — the backend receives typed condition objects, not expression strings):**

| Type | Syntax | Example |
|------|--------|---------|
| flag | `SIGNAL_NAME` | `@if (EP01_COMPLETE) { }` |
| comparison | `<operand> <op> <integer>` | `@if (affection.easton >= 5) { }` or `@if (san <= 20) { }` |
| compound | `<expr> && <expr>` / `<expr> \|\| <expr>` | `@if (san <= 20 \|\| FAILED_TWICE) { }` |
| choice | `OPTION.result` | `@if (A.fail) { }` — result: `success` / `fail` / `any`. Use from outside the option |
| influence | `influence "desc"` or `"desc"` | `@if (influence "player showed empathy") { }` or `@if ("player showed empathy") { }` — both forms accepted |
| check | `check.success` / `check.fail` | `@if (check.success) { }` — context-local, only valid inside a brave option body |
| rating | `rating.<grade>` | `@if (rating.S) { }` / `@if (rating.A \|\| rating.B) { }` — context-local, only valid inside a minigame body |

**Comparison operand** on the left of `>=` / `<=` / etc. must be either `affection.<char>` (character-specific affection) or a bare `<name>` (engine-managed value like `san`, `CHA`). The right side **must be an integer literal** — expressions like `a >= b` or `affection.easton >= (5 + 3)` are rejected.

**`check` vs `choice`** — both touch D20 outcomes but answer different questions. `check.success` inside a brave option body asks "did *this* option's roll just succeed?" It's the idiom for writing `@if (check.success) { } @else { }` immediately after `check { }`. `A.success` from anywhere asks "at some earlier point, did the player pick option A and pass its check?" — retrospective, not context-local. Use `check.*` inside the option body it describes; use `A.*` / `B.*` everywhere else.

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
5. **Brave option with `@on` blocks** — `@on` is not part of MSS. Use `@if (check.success) { } @else { }`. A missing `@else` is allowed (the validator does not force both branches), but it's usually still what you want.
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
18. **`@signal` without a kind** — `@signal <kind> <event>` requires a kind token. Currently only `mark` is implemented. `@signal achievement ...` is no longer valid — use the dedicated `@achievement <id>` trigger instead.
19. **`@minigame` without a description** — `@minigame <id> <ATTR> "<description>" { }`. The quoted description is mandatory; an empty body is fine, but the description is not.
20. **`@cg show` without `duration:` / `content:` fields** — Both are required at the top of the block, before any body nodes. `duration` must be `low` / `medium` / `high`; `content` is a quoted English narrative for the video-generation pipeline.
21. **Orphan marks** — `@signal mark X` with no reader. If no `@if (X)` and no `@if (X && ...) { @achievement ... }` references `X`, delete the mark. Marks that no one reads are noise: they dilute the flag store and mislead downstream tools. Never emit marks for "episode completed", "chose option A", "affection raised", or anything else the engine already tracks.
22. **Using non-English names** — All `event` identifiers, `@achievement` ids, achievement `name`, and `description` strings must be English. Mixed-language ids cause grep/search failures and ambiguous meaning across the pipeline.
23. **Using `common` rarity on `@achievement`** — banned. Rarity must be one of `uncommon` / `rare` / `epic` / `legendary`. If every player gets it, it's not an achievement.
24. **Writing a `when:` field on `@achievement`** — `when` is not part of the grammar. Achievements carry only `name` / `rarity` / `description`; trigger logic lives in the outer `@if (...)` that wraps the `@achievement` directive.
25. **Duplicate `@achievement` ids in one episode** — validator error. If the same achievement spans multiple episodes, declare it once in whichever one most naturally owns it.
26. **Triggering an achievement that was never declared** — `@achievement <id>` inline without a matching `@achievement <id> { ... }` declaration is a runtime dead letter. The runtime registry must have seen the declaration first. (Cross-episode is fine; the engine loads all declarations.)
27. **Using `check.success` outside a brave option body, or `rating.S` outside a minigame body** — both are context-local pseudo-identifiers. Outside their scope the runtime always returns false — that's a narrative bug, not a syntax error.

**Auto-repair:** The interpreter includes a fixer (`mss fix <file>`) that auto-repairs many of these mistakes: missing `@if` parentheses, `&` on block structures, `@check` → `check`, uppercase character names in `@affection`, trailing whitespace, unclosed blocks, and BOM/CRLF encoding issues. The fixer also flags `@on` usage with a migration hint pointing at `@if (check.success)` / `@if (rating.X)`. Always run the fixer before compiling if the script was generated by an LLM.

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
