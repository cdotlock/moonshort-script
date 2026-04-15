// Package ast defines all AST node types for the NRS (NoRules Script) format.
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

// ----------------------------------------------------------------------------
// Root
// ----------------------------------------------------------------------------

// Episode is the root node of every NRS script file.
type Episode struct {
	BranchKey string     // e.g. "main:01"
	Title     string     // e.g. "Butterfly"
	Body      []Node     // ordered list of top-level nodes
	Gates     *GatesBlock // optional routing table at end of episode
}

// ----------------------------------------------------------------------------
// Structure nodes
// ----------------------------------------------------------------------------

// GatesBlock holds all @gate definitions at the end of an episode.
type GatesBlock struct {
	Gates []*Gate
}

func (g *GatesBlock) nodeType() string { return "gates" }

// Gate is a single routing rule inside a @gates block.
type Gate struct {
	Target    string       // destination episode key, e.g. "main/bad/001:01"
	GateType  string       // "choice" | "check" | "signal" | "default"
	Trigger   *GateTrigger // optional trigger descriptor
	Condition string       // optional raw condition expression
}

func (g *Gate) nodeType() string { return "gate" }

// GateTrigger describes what event activates a gate.
type GateTrigger struct {
	OptionID    string // option letter / id (for choice gates)
	CheckResult string // "success" | "fail"
}

// LabelNode marks a jump target inside the episode body.
type LabelNode struct {
	Name string // e.g. "AFTER_FIGHT"
}

func (l *LabelNode) nodeType() string { return "label" }

// GotoNode unconditionally jumps to a label within the episode.
type GotoNode struct {
	Name string // must match a LabelNode.Name
}

func (g *GotoNode) nodeType() string { return "goto" }

// DefaultNode specifies the fallthrough episode key.
type DefaultNode struct {
	Target string // episode key, e.g. "main:02"
}

func (d *DefaultNode) nodeType() string { return "default" }

// ----------------------------------------------------------------------------
// Visual nodes
// ----------------------------------------------------------------------------

// BgSetNode sets the background image.
type BgSetNode struct {
	Name       string // semantic asset name, e.g. "classroom"
	Transition string // "" | "fade" | "cut" | "slow"
}

func (b *BgSetNode) nodeType() string { return "bg" }

// CharShowNode brings a character sprite onto screen.
type CharShowNode struct {
	Char       string // character id, e.g. "mauricio"
	Pose       string // sprite variant, e.g. "neutral_smirk"
	Position   string // "left" | "center" | "right" | "left_far" | "right_far"
	Transition string
}

func (c *CharShowNode) nodeType() string { return "char_show" }

// CharHideNode removes a character sprite.
type CharHideNode struct {
	Char       string
	Transition string
}

func (c *CharHideNode) nodeType() string { return "char_hide" }

// CharExprNode changes a character's facial expression / pose in-place.
type CharExprNode struct {
	Char       string
	Pose       string
	Transition string
}

func (c *CharExprNode) nodeType() string { return "char_expr" }

// CharMoveNode slides a character to a new screen position.
type CharMoveNode struct {
	Char     string
	Position string
}

func (c *CharMoveNode) nodeType() string { return "char_move" }

// CharBubbleNode shows an emotion bubble above a character.
// BubbleType: "anger" | "sweat" | "heart" | "question" | "exclaim" |
//
//	"idea" | "music" | "doom" | "ellipsis"
type CharBubbleNode struct {
	Char       string
	BubbleType string
}

func (c *CharBubbleNode) nodeType() string { return "char_bubble" }

// CgShowNode overlays a CG (computer-generated) illustration.
// Body holds nodes that play while the CG is visible.
type CgShowNode struct {
	Name       string
	Transition string
	Body       []Node
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

// PhoneShowNode opens the in-game phone overlay.
// Body holds the sequence of text-message nodes shown in the overlay.
type PhoneShowNode struct {
	Body []Node
}

func (p *PhoneShowNode) nodeType() string { return "phone_show" }

// PhoneHideNode closes the phone overlay.
type PhoneHideNode struct{}

func (p *PhoneHideNode) nodeType() string { return "phone_hide" }

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

// MusicPlayNode starts background music.
type MusicPlayNode struct {
	Track string
}

func (m *MusicPlayNode) nodeType() string { return "music_play" }

// MusicCrossfadeNode cross-fades to a new music track.
type MusicCrossfadeNode struct {
	Track string
}

func (m *MusicCrossfadeNode) nodeType() string { return "music_crossfade" }

// MusicFadeoutNode fades out the current music track.
type MusicFadeoutNode struct{}

func (m *MusicFadeoutNode) nodeType() string { return "music_fadeout" }

// SfxPlayNode plays a one-shot sound effect.
type SfxPlayNode struct {
	Sound string
}

func (s *SfxPlayNode) nodeType() string { return "sfx_play" }

// ----------------------------------------------------------------------------
// Game-mechanic nodes
// ----------------------------------------------------------------------------

// MinigameNode triggers a mini-game.
// OnResult maps result grade strings (e.g. "S", "A", "B", "fail") to the
// sequence of nodes that execute for that outcome.
type MinigameNode struct {
	ID       string
	Attr     string // governing attribute, e.g. "ATK"
	OnResult map[string][]Node
}

func (m *MinigameNode) nodeType() string { return "minigame" }

// ChoiceNode presents a player choice menu.
type ChoiceNode struct {
	Options []*OptionNode
}

func (c *ChoiceNode) nodeType() string { return "choice" }

// OptionNode is a single answer inside a @choice block.
// Mode: "safe" | "brave"
type OptionNode struct {
	ID        string     // letter / identifier, e.g. "A"
	Mode      string     // "safe" | "brave"
	Text      string     // display label
	Check     *CheckBlock // nil for safe options
	OnSuccess []Node     // nodes that run on success (brave only)
	OnFail    []Node     // nodes that run on failure (brave only)
	Body      []Node     // nodes that always run after check resolution
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

// XpNode grants or removes experience points.
type XpNode struct {
	Delta string // e.g. "+3" or "-5" (signed)
}

func (x *XpNode) nodeType() string { return "xp" }

// SanNode adjusts the player's sanity score.
type SanNode struct {
	Delta string
}

func (s *SanNode) nodeType() string { return "san" }

// AffectionNode adjusts affection toward a specific character.
type AffectionNode struct {
	Char  string
	Delta string
}

func (a *AffectionNode) nodeType() string { return "affection" }

// SignalNode emits a named story signal / flag.
type SignalNode struct {
	Event string // e.g. "EP01_COMPLETE"
}

func (s *SignalNode) nodeType() string { return "signal" }

// ButterflyNode records a butterfly-effect narrative branch decision.
type ButterflyNode struct {
	Description string // human-readable description
}

func (b *ButterflyNode) nodeType() string { return "butterfly" }

// ----------------------------------------------------------------------------
// Flow-control nodes
// ----------------------------------------------------------------------------

// IfNode is a conditional block. Else may be nil.
type IfNode struct {
	Condition string // raw condition expression, e.g. "affection.easton >= 5 && CHA >= 14"
	Then      []Node
	Else      []Node // nil when there is no @else branch
}

func (i *IfNode) nodeType() string { return "if" }
