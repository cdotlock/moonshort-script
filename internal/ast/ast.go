// Package ast defines all AST node types for the MSS (MoonShort Script) format.
// This file contains only type definitions — no parsing logic.
package ast

// ----------------------------------------------------------------------------
// Node interface
// ----------------------------------------------------------------------------

// Node is the interface implemented by every AST node.
// nodeType returns the JSON "type" string used by the serialiser.
type Node interface {
	nodeType() string
}

// HasConcurrent is implemented by nodes that can carry the & (concurrent) flag.
type HasConcurrent interface {
	GetConcurrent() bool
	SetConcurrent(bool)
}

// ConcurrentFlag is embedded in body-level node types to track
// whether the node was prefixed with & (concurrent) instead of @ (sequential).
type ConcurrentFlag struct {
	Concurrent bool
}

func (c *ConcurrentFlag) GetConcurrent() bool  { return c.Concurrent }
func (c *ConcurrentFlag) SetConcurrent(v bool) { c.Concurrent = v }

// ----------------------------------------------------------------------------
// Root
// ----------------------------------------------------------------------------

// Episode is the root node of every MSS script file.
//
// Every episode MUST terminate with exactly one @gate block in the source.
// The gate is then lowered by the emitter into one of two top-level shapes:
//
//   - Gate: the structured routing AST (conditional or unconditional @next /
//     mixed @next / @end leaves).
//   - Ending: a simple terminal marker, populated by the emitter when the
//     source gate is the degenerate form `@gate { @end TYPE }` (Scheme B).
//     This pure-unconditional-ending shape lowers to Episode.Ending so the
//     overlay system can keep its current simple consumer.
//
// Ending is therefore a product of the emitter's lowering pass, NOT a
// separate source-level construct. At source level there is always exactly
// one @gate block per episode.
type Episode struct {
	BranchKey string      // e.g. "main:01"
	Title     string      // e.g. "Butterfly"
	Body      []Node      // ordered list of top-level nodes
	Gate      *GateBlock  // structured routing AST (set when gate has conditions or any @next leaf)
	Ending    *EndingNode // simple terminal marker (set only for pure `@gate { @end TYPE }` lowering)
}

// ----------------------------------------------------------------------------
// Structure nodes
// ----------------------------------------------------------------------------

// GateBlock holds the routing rules at the end of an episode.
// Every episode has exactly one gate. A gate's routes are evaluated
// top-to-bottom; the first matching condition wins. Routes may mix
// @next and @end leaves freely — a single gate can both route to other
// episodes and terminate the story on different conditions.
type GateBlock struct {
	Routes []*GateRoute
}

func (g *GateBlock) nodeType() string { return "gate" }

// Condition is the interface implemented by every condition node.
// ConditionKind returns the "type" string used by the emitter. Five concrete
// implementations exist:
//
//   - ChoiceCondition       : option check result (e.g. A.fail) — retrospective query from outside an option
//   - FlagCondition         : signal-flag truthiness (e.g. HIGH_HEEL_EP05)
//   - ComparisonCondition   : structured numeric comparison
//   - CompoundCondition     : && / || tree of sub-conditions
//   - CheckCondition        : this-option's check result (check.success / check.fail) — context-local to brave option body
type Condition interface {
	ConditionKind() string
}

// ChoiceCondition matches when an option's check resolved a given way.
// Result is one of: "success", "fail", "any".
type ChoiceCondition struct {
	Option string // option ID, e.g. "A", "B"
	Result string // "success" | "fail" | "any"
}

func (c *ChoiceCondition) ConditionKind() string { return "choice" }

// FlagCondition tests whether a named signal flag has been emitted.
type FlagCondition struct {
	Name string // e.g. "EP01_COMPLETE"
}

func (c *FlagCondition) ConditionKind() string { return "flag" }

// ComparisonOperandKind values for ComparisonOperand.Kind. Five kinds:
//
//   - OperandLiteral   : integer literal (uses Value)
//   - OperandAffection : per-character affection (uses Char)
//   - OperandValue     : engine-managed scalar or author @signal int variable, bare-name lookup (uses Name)
//   - OperandMax       : MAX aggregate over args (uses Args, len >= 2)
//   - OperandMin       : MIN aggregate over args (uses Args, len >= 2)
const (
	OperandLiteral   = "literal"
	OperandAffection = "affection"
	OperandValue     = "value"
	OperandMax       = "max"
	OperandMin       = "min"
)

// ComparisonOperand is one side of a comparison. Field usage by Kind:
//
//   - OperandLiteral   : Value holds the integer literal (may be negative).
//   - OperandAffection : Char holds the character id (e.g. "easton").
//   - OperandValue     : Name holds the bare scalar name (e.g. "san", "CHA",
//                        or an author @signal int variable name). The engine
//                        looks the name up in a shared namespace.
//   - OperandMax       : Args holds the aggregate operands (len >= 2). Each
//                        arg is itself a *ComparisonOperand and may recurse.
//   - OperandMin       : same shape as OperandMax.
//
// Fields not relevant to the active Kind are left at their zero value.
type ComparisonOperand struct {
	Kind  string               // OperandLiteral | OperandAffection | OperandValue | OperandMax | OperandMin
	Value int                  // when Kind == OperandLiteral
	Char  string               // when Kind == OperandAffection: character id
	Name  string               // when Kind == OperandValue: scalar / signal-int name
	Args  []*ComparisonOperand // when Kind == OperandMax | OperandMin: aggregate args, len >= 2
}

// ComparisonCondition is a structured numeric comparison.
// Both sides are full operands — any operand kind can appear on either side
// (literal-to-variable, variable-to-variable, aggregate-to-anything, etc.).
// Op is one of: ">=", "<=", ">", "<", "==", "!=".
type ComparisonCondition struct {
	Left  *ComparisonOperand
	Op    string
	Right *ComparisonOperand
}

func (c *ComparisonCondition) ConditionKind() string { return "comparison" }

// CompoundCondition combines two sub-conditions with && or ||.
// Op is "&&" (and) or "||" (or).
type CompoundCondition struct {
	Op    string
	Left  Condition
	Right Condition
}

func (c *CompoundCondition) ConditionKind() string { return "compound" }

// CheckCondition is the context-local condition for a brave option's
// check result. Source syntax: check.success / check.fail, valid only
// inside the body of an @option <ID> brave.
type CheckCondition struct {
	Result string // "success" | "fail"
}

func (c *CheckCondition) ConditionKind() string { return "check" }

// GateLeaf is the marker interface for the two terminal node shapes that
// can appear at the end of a gate route: @next (route to another episode)
// and @end (terminate the story with a specific ending type).
type GateLeaf interface {
	gateLeafKind() string
}

// NextLeaf routes to a destination episode.
// Source syntax: `@next <branch_key>`.
type NextLeaf struct {
	Target string // destination episode key, e.g. "main/bad/001:01"
}

func (n *NextLeaf) gateLeafKind() string { return "next" }

// EndLeaf terminates the story with the specified ending type.
// Source syntax: `@end <type>`.
type EndLeaf struct {
	Type string // EndingComplete | EndingToBeContinued | EndingBad
}

func (e *EndLeaf) gateLeafKind() string { return "end" }

// GateRoute is a single condition→leaf pair inside a @gate block. The leaf
// is either a *NextLeaf (route to another episode) or an *EndLeaf
// (terminate the story); a gate may mix both freely across its routes.
type GateRoute struct {
	Condition Condition // nil = unconditional/fallback
	Leaf      GateLeaf  // *NextLeaf or *EndLeaf
}

// Valid ending type values for EndingNode.Type and EndLeaf.Type.
const (
	EndingComplete      = "complete"
	EndingToBeContinued = "to_be_continued"
	EndingBad           = "bad_ending"
)

// EndingNode is the emitter-produced simple terminal marker, populated on
// Episode.Ending when the source gate is the degenerate form
// `@gate { @end TYPE }`. It is not a source-level construct — at source
// level every episode has a @gate; the emitter lowers the pure-uncondional-
// ending shape into this field for the overlay system's benefit.
type EndingNode struct {
	Type string // EndingComplete | EndingToBeContinued | EndingBad
}

func (e *EndingNode) nodeType() string { return "ending" }

// ----------------------------------------------------------------------------
// Visual nodes
// ----------------------------------------------------------------------------

// PauseNode inserts a single-click wait point. Source syntax: `@pause`.
// No parameters — @pause always waits for exactly one player click.
type PauseNode struct {
	ConcurrentFlag
}

func (p *PauseNode) nodeType() string { return "pause" }

// BgSetNode sets the background image.
type BgSetNode struct {
	ConcurrentFlag
	Name       string // semantic asset name, e.g. "classroom"
	Transition string // "" | "fade" | "cut" | "slow"
}

func (b *BgSetNode) nodeType() string { return "bg" }

// CharShowNode is the single node that covers both bringing a character
// onto screen for the first time AND swapping the pose of a character
// already on screen — the engine decides which case applies at runtime
// based on whether Char is currently visible. There is no separate
// hide / look / move node: the "one character on screen" rule means the
// engine implicitly hides whoever was there, and position is derived from
// gamestate (MC left, everyone else right) — never declared in source.
type CharShowNode struct {
	ConcurrentFlag
	Char       string // character id, e.g. "mauricio"
	Look       string // sprite variant, e.g. "neutral_smirk"
	Transition string // optional, e.g. "dissolve", "fade"
}

func (c *CharShowNode) nodeType() string { return "char_show" }

// CharBubbleNode shows a one-shot emotion bubble above a character.
// The bubble plays once and disappears; it follows the currently visible
// character and is removed if the character is swapped out.
//
// BubbleType must be one of nine values:
//
//	"anger"    — 💢 anger
//	"sweat"    — 💧 cold sweat
//	"heart"    — ❤️ infatuation
//	"question" — ❓ confusion
//	"exclaim"  — ❗ surprise
//	"idea"     — 💡 sudden insight
//	"music"    — 🎵 cheerful / humming
//	"doom"     — 💀 despair
//	"ellipsis" — ... silence / speechless
//
// `bubble` is a reserved word: no pose may be named "bubble".
type CharBubbleNode struct {
	ConcurrentFlag
	Char       string
	BubbleType string
}

func (c *CharBubbleNode) nodeType() string { return "bubble" }

// CgShowNode overlays a CG (computer-generated) illustration, which is
// produced downstream by agent-forge as a short video. CG is a leaf
// directive: no body, no per-script duration, no transition. Pacing,
// camera motion, and emphasis are all derived by the downstream pipeline
// from Content (a continuous English prose narrative describing the
// camera, the beats, and the emphasis).
type CgShowNode struct {
	ConcurrentFlag
	Name    string // CG asset handle
	Content string // continuous English narrative: camera + story beats
}

func (c *CgShowNode) nodeType() string { return "cg_show" }

// ----------------------------------------------------------------------------
// Dialogue nodes
// ----------------------------------------------------------------------------

// DialogueNode is a line of character dialogue: "CHARACTER: text".
type DialogueNode struct {
	Character string // all-caps character name
	Text      string
}

func (d *DialogueNode) nodeType() string { return "dialogue" }

// NarratorNode is a NARRATOR: line.
type NarratorNode struct {
	Text string
}

func (n *NarratorNode) nodeType() string { return "narrator" }

// YouNode is a YOU: line (player's internal voice / thought).
type YouNode struct {
	Text string
}

func (y *YouNode) nodeType() string { return "you" }

// ----------------------------------------------------------------------------
// Phone / text-message nodes
// ----------------------------------------------------------------------------

// PhoneShowNode opens the in-game phone overlay and manages its full
// lifecycle: when the body finishes playing the engine automatically
// dismisses the overlay. There is no separate phone_hide node.
//
// Body holds the sequence of messages shown in the overlay and accepts
// ONLY TextMessageNode children — the validator enforces a strict
// whitelist. Narration, sfx, state changes, etc. must live outside the
// @phone block.
type PhoneShowNode struct {
	ConcurrentFlag
	Body []Node // TextMessageNode only (validator-enforced)
}

func (p *PhoneShowNode) nodeType() string { return "phone_show" }

// TextMessageNode represents a single SMS message bubble.
// Direction: "from" | "to"
type TextMessageNode struct {
	Direction string // "from" | "to"
	Char      string // character id
	Content   string // message text
}

func (t *TextMessageNode) nodeType() string { return "text_message" }

// ----------------------------------------------------------------------------
// Audio nodes
// ----------------------------------------------------------------------------

// MusicSetNode plays a BGM track. Source syntax: `@music <name>`.
// The engine inspects current playback state and either fades in from
// silence or cross-fades from the currently playing track — the script
// does not distinguish the two cases.
type MusicSetNode struct {
	ConcurrentFlag
	Name string
}

func (m *MusicSetNode) nodeType() string { return "music" }

// MusicStopNode fades out the currently playing BGM.
// Source syntax: `@music stop`.
type MusicStopNode struct {
	ConcurrentFlag
}

func (m *MusicStopNode) nodeType() string { return "music_stop" }

// SfxNode plays a one-shot sound effect. Source syntax: `@sfx <name>`.
type SfxNode struct {
	ConcurrentFlag
	Name string
}

func (s *SfxNode) nodeType() string { return "sfx" }

// ----------------------------------------------------------------------------
// Game-mechanic nodes
// ----------------------------------------------------------------------------

// Valid trick types. The set is intentionally closed: each entry has
// engine-native detection on the device, no per-script tuning, no body.
// The trick is a mandatory body-interaction beat — the player must
// complete it to advance.
//
// Modality:
//   - touch  (no permission): TrickTap, TrickHold, TrickSwipe
//   - motion (no runtime prompt): TrickShake, TrickSwing, TrickTilt
//
// The engine owns the detection threshold and the prompt overlay; the
// script only declares what trick fires and what one-line prompt the
// player sees.
const (
	TrickTap   = "tap"
	TrickHold  = "hold"
	TrickSwipe = "swipe"
	TrickShake = "shake"
	TrickSwing = "swing"
	TrickTilt  = "tilt"
)

// TrickNode triggers a mandatory body-interaction beat. The player must
// complete <Type> for the engine to advance. There is no rating, no
// branching, no reward — see MinigameNode for that. Source syntax:
//
//	@trick <type> "<prompt>"
//
// Type is one of the six locked constants above; the validator rejects
// any other value. Prompt is a one-line imperative shown to the player
// as narrative glue (e.g. "Tap the screen until you hear the door give"
// for TrickTap, "Hold your breath" for TrickHold).
type TrickNode struct {
	ConcurrentFlag
	Type   string // one of the Trick* constants
	Prompt string // one-line player-facing imperative
}

func (t *TrickNode) nodeType() string { return "trick" }

// MinigameNode triggers an optional embedded mini-game. The game itself
// is generated downstream by a vibe-coding agent from Description (which
// describes both the scene and the simple gameplay) — there is no
// pre-built library. The player may play or skip; if they play, the
// engine scales the reward by the H5 result. There is no body, no
// rating-driven branching, no attribute coupling, and no script-side
// reward declaration: rewards are owned by the engine for anti-cheat.
//
// Source syntax: @minigame <name> "<description>"
type MinigameNode struct {
	ConcurrentFlag
	Name        string // asset handle (also used as @cg show name does)
	Description string // continuous English prose: scene + simple gameplay
}

func (m *MinigameNode) nodeType() string { return "minigame" }

// ChoiceNode presents a player choice menu.
type ChoiceNode struct {
	ConcurrentFlag
	Options []*OptionNode
}

func (c *ChoiceNode) nodeType() string { return "choice" }

// OptionNode is a single answer inside a @choice block.
// Mode: "safe" | "brave"
//
// For brave options the body typically contains an @if (check.success)
// branch with @else — the CheckCondition AST node queries the runtime
// check result. Authors write standard @if/@else trees; there is no
// dedicated outcome directive.
type OptionNode struct {
	ID    string      // letter / identifier, e.g. "A"
	Mode  string      // "safe" | "brave"
	Text  string      // display label
	Check *CheckBlock // nil for safe options
	Body  []Node      // body nodes (for brave: typically contains @if (check.success) … @else …)
}

func (o *OptionNode) nodeType() string { return "option" }

// CheckBlock is the D20-style skill check descriptor inside a brave option.
type CheckBlock struct {
	Attr string // attribute being checked, e.g. "CHA"
	DC   int    // difficulty class
}

func (c *CheckBlock) nodeType() string { return "check" }

// ----------------------------------------------------------------------------
// State-change nodes
// ----------------------------------------------------------------------------

// AffectionNode adjusts affection toward a specific character.
type AffectionNode struct {
	ConcurrentFlag
	Char  string
	Delta string
}

func (a *AffectionNode) nodeType() string { return "affection" }

// Valid signal kinds.
//
//   - "mark" — persistent boolean flag (set-once-true)
//   - "int"  — persistent integer variable (assign / increment / decrement)
//
// The kind word is mandatory in source syntax (@signal <kind> ...).
const (
	SignalKindMark = "mark"
	SignalKindInt  = "int"
)

// Int-signal operators. @signal int <name> <op> <value>.
//
//   - "="  — unconditional assignment (value may be negative)
//   - "+"  — increment by value (value must be non-negative)
//   - "-"  — decrement by value (value must be non-negative)
const (
	SignalOpAssign = "="
	SignalOpAdd    = "+"
	SignalOpSub    = "-"
)

// SignalNode emits a persistent state write. Two kinds are supported:
//
//   - SignalKindMark: sets a named boolean flag to true. Queried via
//     @if (NAME) (resolved as FlagCondition). Event carries the flag
//     name (e.g. "HIGH_HEEL_EP05"). Author discipline: only use for
//     key story points a later reader (@if / achievement guard) needs.
//
//   - SignalKindInt: mutates a named persistent integer variable.
//     Name carries the variable id (snake_case lowercase). Op is one
//     of SignalOp{Assign,Add,Sub}. Value is the operand (Assign may
//     be negative; Add/Sub are always non-negative). Queried via
//     @if (NAME <op> N) comparison (bare name, resolved as a regular
//     value-comparison through the existing left.kind="value" path).
//     Author discipline: free use for counters and thresholds; the
//     "marks are precious" rule does NOT apply here.
//
// For backward compatibility, Event is retained and used only for
// SignalKindMark. Name/Op/Value are used only for SignalKindInt.
type SignalNode struct {
	ConcurrentFlag
	Kind string // SignalKindMark or SignalKindInt

	// Fields for SignalKindMark.
	Event string // e.g. "HIGH_HEEL_EP05"

	// Fields for SignalKindInt.
	Name  string // e.g. "rejections"
	Op    string // SignalOp{Assign,Add,Sub}
	Value int    // operand
}

func (s *SignalNode) nodeType() string { return "signal" }

// Valid achievement rarities. "common" is intentionally excluded —
// achievements must require deliberate player action.
const (
	RarityUncommon  = "uncommon"
	RarityRare      = "rare"
	RarityEpic      = "epic"
	RarityLegendary = "legendary"
)

// AchievementNode is an inline achievement trigger carrying its full
// metadata. Source syntax: @achievement <id> { name / rarity / description }.
// The block is both the declaration and the firing point — reaching this
// node in execution unlocks the achievement.
//
// Conditional triggering is expressed by wrapping in @if, e.g.
// @if (MARK_A && MARK_B) { @achievement X { ... } }.
//
// Fields:
//   - ID: stable identifier
//   - Name: short English display name
//   - Rarity: uncommon | rare | epic | legendary (no "common")
//   - Description: DM-voice 1–2 sentence English flavor text
type AchievementNode struct {
	ConcurrentFlag
	ID          string
	Name        string
	Rarity      string
	Description string
}

func (a *AchievementNode) nodeType() string { return "achievement" }

// ButterflyNode records a butterfly-effect narrative decision. Description
// captures the player action plus its character implication in English
// prose, which downstream content-generation agents — the Remix Executor
// and Dream — consume to keep generated remix episodes and derivative
// stories aligned with the player's behavior profile.
//
// Butterfly records are NOT consulted by gate routing: every runtime
// decision (signal mark, signal int, affection, choice history) is
// deterministic, while butterfly is fuzzy player-profile fuel for the
// generation pipeline.
type ButterflyNode struct {
	ConcurrentFlag
	Description string // English prose: action + character implication
}

func (b *ButterflyNode) nodeType() string { return "butterfly" }

// ----------------------------------------------------------------------------
// Flow-control nodes
// ----------------------------------------------------------------------------

// IfNode is a conditional block. Else may be nil.
// For @else @if chains, Else contains a single IfNode.
type IfNode struct {
	ConcurrentFlag
	Condition Condition
	Then      []Node
	Else      []Node // nil when there is no @else branch
}

func (i *IfNode) nodeType() string { return "if" }
