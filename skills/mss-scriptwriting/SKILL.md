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

  // Routing — MUST be last
  @gate {
    @if (<condition>): @next <target>
    @else: @next <fallback>
  }
}
```

The `branch_key` follows a strict addressing format. Read `references/addressing.md` for the full scheme.

## The Two Languages in One File

MSS has two syntaxes that alternate freely:

**1. Directive lines** start with `@` — these control visuals, audio, game mechanics, and flow:
```
@bg set school_hallway fade
@mauricio show neutral_smirk at right
@music play tense_strings
```

**2. Dialogue lines** start with a CHARACTER NAME in caps followed by colon — these are what the player reads:
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
    @signal MINIGAME_PERFECT_EP01
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
      @xp +3
      @san +5
      @butterfly "Accepted Easton's approach at the cafeteria"
    }
    @on fail {
      @easton look hurt
      MALIA: I... I can't do this.
      @san -20
      @xp +1
      @butterfly "Tried to face Easton but lost courage"
    }
  }
  @option B safe "Have Mark make a scene." {
    @mark show grin_mischief at right
    @easton hide fade
    MARK: HEY EASTON! You want some of my mystery casserole?
    YOU: Thank god for Mark.
    @xp +1
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

**Every outcome path must include:**
- `@xp` change (brave success: +3, brave fail: +1, safe: +1)
- `@butterfly` description (what happened as a result of this choice)
- Optionally: `@san`, `@affection`, `@signal`

## State Changes

These are declarations — the game engine handles the actual math.

```
@xp +3                                    // Experience points
@san -20                                   // Sanity (health)
@affection easton +2                       // Character relationship
@signal EP01_COMPLETE                      // Event for engine (achievements, flags, etc.)
@butterfly "Accepted Easton's approach"    // Recorded for LLM-based route evaluation
```

**`@butterfly` is critical.** The game engine accumulates all butterfly records across episodes and uses LLM evaluation to determine which story branches unlock. Write clear, specific descriptions of what happened and what it reveals about the player's tendencies. Bad: "Made a choice." Good: "Showed vulnerability by accepting help from a former rival."

## Flow Control

### Conditional content

Use `@if` to show different content based on game state. The engine evaluates conditions at runtime.

```
@if affection.easton >= 5 && CHA >= 14 {
  EASTON: You remembered.
} @else {
  EASTON: ...Hey.
}

@if san <= 20 || FAILED_TWICE {
  YOU: I can barely keep it together.
}
```

**Condition syntax:**
- Flags (boolean): `FLAG_NAME` (true if the signal was ever emitted)
- Numeric comparisons: `affection.<char>`, `xp`, `san`, or any attribute name + operator + number
- Operators: `>=` `<=` `>` `<` `==` `!=`
- Logic: `&&` (and), `||` (or)

### Labels and goto (advanced, avoid if possible)

```
@if SKIP_FIGHT {
  @goto AFTER_FIGHT
}
// ... fight content ...
@label AFTER_FIGHT
NARRATOR: The dust settled.
```

Only use `@goto`/`@label` when structured nesting can't express the flow. Prefer `@if`/`@else` for conditional content.

## Routing (Gate)

The `@gate` block at the end of every episode declares where the player goes next. The engine evaluates conditions top-to-bottom — first match wins. If nothing matches, `@else` is the fallback.

```
@gate {
  // Choice-based: route based on the player's choice
  @if (A fail): @next main/bad/001:01

  // Influence-based: route based on accumulated butterfly effects (LLM evaluates)
  @if ("Player has repeatedly shown empathy toward Easton"): @next main/route/001:01

  // Fallback
  @else: @next main:02
}
```

**Choice condition format:** `<option_id> <result>`
- option_id: A, B, C... (matches the option's ID)
- result: `success` | `fail` | `any`

**Influence condition:** A quoted natural language description. At runtime, the engine feeds all accumulated `@butterfly` records to an LLM and asks whether the condition is satisfied.

**`@else` is mandatory.** Every episode must have a fallback route.

## Common Mistakes

1. **Forgetting `@gate` block** — Every episode needs routing. No gate = interpreter error.
2. **Missing `@else`** — Gate block without fallback = interpreter error.
3. **Brave option without `check { }`** — Brave means D20 check. No check block = error.
4. **Brave option without both `@on success` and `@on fail`** — Both outcomes must be defined.
5. **`@goto` without matching `@label`** — The target label must exist in the same episode.
6. **Putting `@gate` anywhere except the end** — Gate must be the last thing in `@episode`.
7. **Using character names inconsistently** — `@mauricio show ...` and `MAURICIO:` must use the same name (case-insensitive).
8. **Forgetting `@butterfly` in choice outcomes** — Every choice outcome should record what happened for the influence system.
9. **Writing asset paths instead of semantic names** — Scripts use names like `classroom_morning`, not URLs or file paths. The interpreter handles mapping.
10. **Nested `@choice` blocks** — Only one `@choice` per episode. Multiple choices in one episode is not supported.

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

## References

- `references/directive-table.md` — directive cheat sheet, quick lookup when writing scripts
- `references/addressing.md` — episode ID addressing rules for branch_key and file naming
- `references/MSS-SPEC.md` — **complete format specification** with JSON output structure, asset mapping schema, interpreter behavior, and Remix compatibility details. Read this when you need to understand how the interpreter processes scripts, what JSON the frontend receives, or how the asset mapping YAML works.
