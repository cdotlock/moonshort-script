// Package emitter converts an MSS AST into player-ready JSON with resolved asset URLs.
package emitter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// AssetResolver maps semantic asset names to full URLs.
type AssetResolver interface {
	ResolveBg(name string) (string, error)
	ResolveCharacter(char, poseExpr string) (string, error)
	ResolveMusic(name string) (string, error)
	ResolveSfx(name string) (string, error)
	ResolveCg(name string) (string, error)
	ResolveMinigame(gameID string) (string, error)
}

// Warning records a non-fatal issue encountered during emission.
type Warning struct {
	Message string
}

// Emitter walks an AST and produces player-ready JSON.
type Emitter struct {
	resolver AssetResolver
	Warnings []Warning
	seq      int // episode-scoped monotonic step counter (reset per Emit call)
}

// New creates an Emitter with the given asset resolver.
func New(resolver AssetResolver) *Emitter {
	return &Emitter{resolver: resolver}
}

// Emit converts an Episode AST into JSON bytes.
func (e *Emitter) Emit(ep *ast.Episode) ([]byte, error) {
	e.seq = 0
	out := map[string]interface{}{
		"episode_id": ep.BranchKey,
		"branch_key": extractBranchKey(ep.BranchKey),
		"seq":        extractSeq(ep.BranchKey),
		"title":      ep.Title,
		"steps":      e.emitNodes(ep.Body),
	}

	if ep.Gate != nil {
		out["gate"] = e.emitGate(ep.Gate)
	} else {
		out["gate"] = nil
	}

	if ep.Ending != nil {
		out["ending"] = map[string]interface{}{"type": ep.Ending.Type}
	} else {
		out["ending"] = nil
	}

	return json.MarshalIndent(out, "", "  ")
}

// isConcurrent checks if a node carries the concurrent (&) flag.
func isConcurrent(n ast.Node) bool {
	if hc, ok := n.(ast.HasConcurrent); ok {
		return hc.GetConcurrent()
	}
	return false
}

// stepTypeTag maps a step's "type" discriminator to the 2-4 letter tag
// used in the stable step id (see assignStepID).
//
// CONTRACT: This function — together with assignStepID — is the SOLE
// authority on the original-step id format emitted by the compiler.
// Once an episode JSON has been shipped to a backend that persists
// player-session cursors keyed by these ids, NEVER change either
// function's algorithm without coordinating a one-shot data migration.
// Renaming a tag, changing the seq width, or reordering grouping
// semantics will silently invalidate every persisted cursor.
//
// Returns "unk" for any unrecognized type. The validator should reject
// scripts that produce unknown types before they reach the emitter; the
// fallback exists so we never panic if a new step type is added without
// updating this table.
func stepTypeTag(stepType string) string {
	switch stepType {
	case "dialogue":
		return "dlg"
	case "narrator":
		return "nar"
	case "you":
		return "you"
	case "pause":
		return "pau"
	case "choice":
		return "ch"
	case "minigame":
		return "mg"
	case "cg_show":
		return "cg"
	case "bg":
		return "bg"
	case "char_show", "char_hide", "char_look", "char_move", "bubble":
		return "char"
	case "music_play", "music_crossfade", "music_fadeout":
		return "mus"
	case "sfx_play":
		return "sfx"
	case "phone_show", "phone_hide", "text_message":
		return "phn"
	case "signal":
		return "sig"
	case "affection":
		return "aff"
	case "achievement":
		return "ach"
	case "butterfly":
		return "btf"
	case "if", "goto", "label":
		return "ctrl"
	default:
		return "unk"
	}
}

// assignStepID stamps a stable id onto a single emitted step map.
//
// Format: <seq>_<tag>, where seq is a 4-digit zero-padded 1-based counter
// shared across the entire episode tree (monotonically increasing in DFS
// pre-order), and tag is the short type code returned by stepTypeTag.
// No two steps in the same episode can share an id.
//
// CONTRACT: This function is the SOLE authority on original-step ids.
// See stepTypeTag for the migration warning — both functions move
// together or not at all.
func assignStepID(step map[string]interface{}, seq int) {
	stepType, _ := step["type"].(string)
	step["id"] = fmt.Sprintf("%04d_%s", seq, stepTypeTag(stepType))
}

// emitNodes converts a slice of AST nodes into a slice of steps, grouping
// consecutive concurrent (&-prefixed) nodes into sub-arrays, and stamping
// every emitted step with an episode-scoped stable id.
//
// Grouping rule:
//   - A non-concurrent node (@-prefixed or dialogue) starts a potential group.
//   - Following &-concurrent nodes join the group.
//   - When the next non-concurrent node arrives, the group is flushed.
//   - Single-item groups are emitted as plain objects.
//   - Multi-item groups are emitted as arrays (concurrent execution).
//
// ID assignment: each emitted step (whether solo or inside a concurrent
// group) consumes one seq from e.seq — a single monotonic counter shared
// across the entire episode tree. The parent step is stamped first, then
// child containers (option.steps, minigame.steps, cg_show.steps,
// if.then/else, phone_show.messages) are emitted via emitChildren,
// giving children higher seqs than their parent (DFS pre-order).
func (e *Emitter) emitNodes(nodes []ast.Node) []interface{} {
	steps := make([]interface{}, 0)
	var group []interface{}

	flush := func() {
		if len(group) == 0 {
			return
		}
		if len(group) == 1 {
			steps = append(steps, group[0])
		} else {
			steps = append(steps, group)
		}
		group = nil
	}

	for _, n := range nodes {
		step := e.emitNode(n)
		if step == nil {
			continue
		}

		e.seq++
		assignStepID(step, e.seq)
		e.emitChildren(n, step)

		if isConcurrent(n) {
			group = append(group, step)
		} else {
			flush()
			group = append(group, step)
		}
	}
	flush()

	return steps
}

// emitNode converts a single AST node into a step map.
func (e *Emitter) emitNode(n ast.Node) map[string]interface{} {
	switch v := n.(type) {
	case *ast.BgSetNode:
		return e.emitBg(v)
	case *ast.CharShowNode:
		return e.emitCharShow(v)
	case *ast.CharHideNode:
		return e.emitCharHide(v)
	case *ast.CharLookNode:
		return e.emitCharLook(v)
	case *ast.CharMoveNode:
		return e.emitCharMove(v)
	case *ast.CharBubbleNode:
		return e.emitCharBubble(v)
	case *ast.CgShowNode:
		return e.emitCgShow(v)
	case *ast.DialogueNode:
		return e.emitDialogue(v)
	case *ast.NarratorNode:
		return e.emitNarrator(v)
	case *ast.YouNode:
		return e.emitYou(v)
	case *ast.PhoneShowNode:
		return e.emitPhoneShow(v)
	case *ast.PhoneHideNode:
		return map[string]interface{}{"type": "phone_hide"}
	case *ast.TextMessageNode:
		return e.emitTextMessage(v)
	case *ast.MusicPlayNode:
		return e.emitMusicPlay(v)
	case *ast.MusicCrossfadeNode:
		return e.emitMusicCrossfade(v)
	case *ast.MusicFadeoutNode:
		return map[string]interface{}{"type": "music_fadeout"}
	case *ast.SfxPlayNode:
		return e.emitSfxPlay(v)
	case *ast.MinigameNode:
		return e.emitMinigame(v)
	case *ast.ChoiceNode:
		return e.emitChoice(v)
	case *ast.AffectionNode:
		return e.emitAffection(v)
	case *ast.SignalNode:
		return e.emitSignal(v)
	case *ast.ButterflyNode:
		return map[string]interface{}{"type": "butterfly", "description": v.Description}
	case *ast.AchievementNode:
		// The JSON `achievement_id` field carries the semantic id from MSS
		// source `@achievement <id> { ... }` (e.g. "RARE_COURAGE"), distinct
		// from the new compiler-assigned `id` field (the cursor stable-step
		// id, format `<seq>_<tag>`). This mirrors MinigameStep.game_id —
		// keeping the semantic id under a domain-specific key avoids
		// collision with the universal stable step id stamped by
		// assignStepID after this map is returned.
		return map[string]interface{}{
			"type":           "achievement",
			"achievement_id": v.ID,
			"name":           v.Name,
			"rarity":         v.Rarity,
			"description":    v.Description,
		}
	case *ast.IfNode:
		return e.emitIf(v)
	case *ast.LabelNode:
		return map[string]interface{}{"type": "label", "name": v.Name}
	case *ast.PauseNode:
		return map[string]interface{}{"type": "pause", "clicks": v.Clicks}
	case *ast.GotoNode:
		return map[string]interface{}{"type": "goto", "target": v.Name}
	default:
		e.warn("unknown node type: %T", n)
		return nil
	}
}

func (e *Emitter) emitBg(n *ast.BgSetNode) map[string]interface{} {
	m := map[string]interface{}{
		"type": "bg",
		"name": n.Name,
	}
	url, err := e.resolver.ResolveBg(n.Name)
	if err != nil {
		e.warn("bg %q: %v", n.Name, err)
	} else {
		m["url"] = url
	}
	if n.Transition != "" {
		m["transition"] = n.Transition
	}
	return m
}

func (e *Emitter) emitCharShow(n *ast.CharShowNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "char_show",
		"character": n.Char,
		"look":      n.Look,
		"position":  n.Position,
	}
	url, err := e.resolver.ResolveCharacter(n.Char, n.Look)
	if err != nil {
		e.warn("char_show %q/%q: %v", n.Char, n.Look, err)
	} else {
		m["url"] = url
	}
	if n.Transition != "" {
		m["transition"] = n.Transition
	}
	return m
}

func (e *Emitter) emitCharHide(n *ast.CharHideNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "char_hide",
		"character": n.Char,
	}
	if n.Transition != "" {
		m["transition"] = n.Transition
	}
	return m
}

func (e *Emitter) emitCharLook(n *ast.CharLookNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "char_look",
		"character": n.Char,
		"look":      n.Look,
	}
	url, err := e.resolver.ResolveCharacter(n.Char, n.Look)
	if err != nil {
		e.warn("char_look %q/%q: %v", n.Char, n.Look, err)
	} else {
		m["url"] = url
	}
	if n.Transition != "" {
		m["transition"] = n.Transition
	}
	return m
}

func (e *Emitter) emitCharMove(n *ast.CharMoveNode) map[string]interface{} {
	return map[string]interface{}{
		"type":      "char_move",
		"character": n.Char,
		"position":  n.Position,
	}
}

func (e *Emitter) emitCharBubble(n *ast.CharBubbleNode) map[string]interface{} {
	return map[string]interface{}{
		"type":        "bubble",
		"character":   n.Char,
		"bubble_type": n.BubbleType,
	}
}

func (e *Emitter) emitCgShow(n *ast.CgShowNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":     "cg_show",
		"name":     n.Name,
		"duration": n.Duration,
		"content":  n.Content,
	}
	url, err := e.resolver.ResolveCg(n.Name)
	if err != nil {
		e.warn("cg_show %q: %v", n.Name, err)
	} else {
		m["url"] = url
	}
	if n.Transition != "" {
		m["transition"] = n.Transition
	}
	return m
}

func (e *Emitter) emitDialogue(n *ast.DialogueNode) map[string]interface{} {
	return map[string]interface{}{
		"type":      "dialogue",
		"character": strings.ToLower(n.Character),
		"text":      n.Text,
	}
}

func (e *Emitter) emitNarrator(n *ast.NarratorNode) map[string]interface{} {
	return map[string]interface{}{
		"type": "narrator",
		"text": n.Text,
	}
}

func (e *Emitter) emitYou(n *ast.YouNode) map[string]interface{} {
	return map[string]interface{}{
		"type": "you",
		"text": n.Text,
	}
}

func (e *Emitter) emitPhoneShow(n *ast.PhoneShowNode) map[string]interface{} {
	return map[string]interface{}{
		"type": "phone_show",
	}
}

func (e *Emitter) emitTextMessage(n *ast.TextMessageNode) map[string]interface{} {
	return map[string]interface{}{
		"type":      "text_message",
		"direction": n.Direction,
		"character": strings.ToLower(n.Char),
		"text":      n.Content,
	}
}

func (e *Emitter) emitMusicPlay(n *ast.MusicPlayNode) map[string]interface{} {
	m := map[string]interface{}{
		"type": "music_play",
		"name": n.Track,
	}
	url, err := e.resolver.ResolveMusic(n.Track)
	if err != nil {
		e.warn("music_play %q: %v", n.Track, err)
	} else {
		m["url"] = url
	}
	return m
}

func (e *Emitter) emitMusicCrossfade(n *ast.MusicCrossfadeNode) map[string]interface{} {
	m := map[string]interface{}{
		"type": "music_crossfade",
		"name": n.Track,
	}
	url, err := e.resolver.ResolveMusic(n.Track)
	if err != nil {
		e.warn("music_crossfade %q: %v", n.Track, err)
	} else {
		m["url"] = url
	}
	return m
}

func (e *Emitter) emitSfxPlay(n *ast.SfxPlayNode) map[string]interface{} {
	m := map[string]interface{}{
		"type": "sfx_play",
		"name": n.Sound,
	}
	url, err := e.resolver.ResolveSfx(n.Sound)
	if err != nil {
		e.warn("sfx_play %q: %v", n.Sound, err)
	} else {
		m["url"] = url
	}
	return m
}

func (e *Emitter) emitMinigame(n *ast.MinigameNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":        "minigame",
		"game_id":     n.ID,
		"attr":        n.Attr,
		"description": n.Description,
	}
	url, err := e.resolver.ResolveMinigame(n.ID)
	if err != nil {
		e.warn("minigame %q: %v", n.ID, err)
	} else {
		m["game_url"] = url
	}
	return m
}

func (e *Emitter) emitChoice(n *ast.ChoiceNode) map[string]interface{} {
	options := make([]interface{}, 0, len(n.Options))
	for _, opt := range n.Options {
		o := map[string]interface{}{
			"id":   opt.ID,
			"mode": opt.Mode,
			"text": opt.Text,
		}
		if opt.Check != nil {
			o["check"] = map[string]interface{}{
				"attr": opt.Check.Attr,
				"dc":   opt.Check.DC,
			}
		}
		options = append(options, o)
	}
	return map[string]interface{}{
		"type":    "choice",
		"options": options,
	}
}

func (e *Emitter) emitAffection(n *ast.AffectionNode) map[string]interface{} {
	delta := parseDelta(n.Delta)
	if delta == 0 && n.Delta != "0" && n.Delta != "+0" && n.Delta != "-0" {
		e.warn("affection %q: invalid delta %q, defaulting to 0", n.Char, n.Delta)
	}
	return map[string]interface{}{
		"type":      "affection",
		"character": n.Char,
		"delta":     delta,
	}
}

func (e *Emitter) emitSignal(n *ast.SignalNode) map[string]interface{} {
	switch n.Kind {
	case ast.SignalKindMark:
		return map[string]interface{}{
			"type":  "signal",
			"kind":  "mark",
			"event": n.Event,
		}
	case ast.SignalKindInt:
		return map[string]interface{}{
			"type":  "signal",
			"kind":  "int",
			"name":  n.Name,
			"op":    n.Op,
			"value": n.Value,
		}
	default:
		// Defensive: unknown kind is caught by validator, but keep the
		// emitter resilient — emit a best-effort shape so downstream
		// debugging isn't hampered.
		return map[string]interface{}{
			"type": "signal",
			"kind": n.Kind,
		}
	}
}

func (e *Emitter) emitIf(n *ast.IfNode) map[string]interface{} {
	return map[string]interface{}{
		"type":      "if",
		"condition": e.emitCondition(n.Condition),
	}
}

// emitChildren attaches child containers to a step AFTER the parent has
// been stamped with its id, ensuring children get higher seqs (DFS pre-order).
func (e *Emitter) emitChildren(n ast.Node, step map[string]interface{}) {
	switch v := n.(type) {
	case *ast.CgShowNode:
		if len(v.Body) > 0 {
			step["steps"] = e.emitNodes(v.Body)
		}
	case *ast.PhoneShowNode:
		if len(v.Body) > 0 {
			step["messages"] = e.emitNodes(v.Body)
		}
	case *ast.MinigameNode:
		if len(v.Body) > 0 {
			step["steps"] = e.emitNodes(v.Body)
		}
	case *ast.ChoiceNode:
		options := step["options"].([]interface{})
		for i, opt := range v.Options {
			if len(opt.Body) > 0 {
				options[i].(map[string]interface{})["steps"] = e.emitNodes(opt.Body)
			}
		}
	case *ast.IfNode:
		step["then"] = e.emitNodes(v.Then)
		if len(v.Else) > 0 {
			if len(v.Else) == 1 {
				if elseIf, ok := v.Else[0].(*ast.IfNode); ok {
					step["else"] = e.emitNestedIf(elseIf)
					return
				}
			}
			step["else"] = e.emitNodes(v.Else)
		}
	}
}

// emitNestedIf builds a nested @else @if node. The nested IfStep does NOT
// get an id (only its then/else children are stamped via emitNodes).
func (e *Emitter) emitNestedIf(n *ast.IfNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "if",
		"condition": e.emitCondition(n.Condition),
		"then":      e.emitNodes(n.Then),
	}
	if len(n.Else) > 0 {
		if len(n.Else) == 1 {
			if elseIf, ok := n.Else[0].(*ast.IfNode); ok {
				m["else"] = e.emitNestedIf(elseIf)
				return m
			}
		}
		m["else"] = e.emitNodes(n.Else)
	}
	return m
}

func (e *Emitter) emitGate(g *ast.GateBlock) interface{} {
	return e.emitGateRoute(g.Routes, 0)
}

// emitGateRoute recursively builds a nested if/else chain from gate routes.
func (e *Emitter) emitGateRoute(routes []*ast.GateRoute, idx int) map[string]interface{} {
	if idx >= len(routes) {
		return nil
	}

	r := routes[idx]
	m := map[string]interface{}{"next": r.Target}

	if r.Condition != nil {
		m["if"] = e.emitCondition(r.Condition)
		if idx+1 < len(routes) {
			m["else"] = e.emitGateRoute(routes, idx+1)
		}
	}

	return m
}

// emitCondition converts a Condition AST node to a JSON map.
// All condition types are fully structured — no raw expression strings —
// so the backend can consume them directly without re-parsing.
func (e *Emitter) emitCondition(c ast.Condition) map[string]interface{} {
	switch v := c.(type) {
	case *ast.ChoiceCondition:
		return map[string]interface{}{
			"type":   "choice",
			"option": v.Option,
			"result": v.Result,
		}
	case *ast.FlagCondition:
		return map[string]interface{}{
			"type": "flag",
			"name": v.Name,
		}
	case *ast.InfluenceCondition:
		return map[string]interface{}{
			"type":        "influence",
			"description": v.Description,
		}
	case *ast.ComparisonCondition:
		left := map[string]interface{}{"kind": v.Left.Kind}
		switch v.Left.Kind {
		case ast.OperandAffection:
			left["char"] = v.Left.Char
		case ast.OperandValue:
			left["name"] = v.Left.Name
		}
		return map[string]interface{}{
			"type":  "comparison",
			"left":  left,
			"op":    v.Op,
			"right": v.Right,
		}
	case *ast.CompoundCondition:
		return map[string]interface{}{
			"type":  "compound",
			"op":    v.Op,
			"left":  e.emitCondition(v.Left),
			"right": e.emitCondition(v.Right),
		}
	case *ast.CheckCondition:
		return map[string]interface{}{
			"type":   "check",
			"result": v.Result,
		}
	case *ast.RatingCondition:
		return map[string]interface{}{
			"type":  "rating",
			"grade": v.Grade,
		}
	default:
		e.warn("unknown condition type: %T", c)
		return map[string]interface{}{"type": "unknown"}
	}
}

func (e *Emitter) warn(format string, args ...interface{}) {
	e.Warnings = append(e.Warnings, Warning{Message: fmt.Sprintf(format, args...)})
}

// extractBranchKey extracts the branch key from an episode id like "main:01" -> "main".
func extractBranchKey(episodeID string) string {
	idx := strings.LastIndex(episodeID, ":")
	if idx < 0 {
		return episodeID
	}
	return episodeID[:idx]
}

// extractSeq extracts the sequence number from an episode id like "main:01" -> 1.
func extractSeq(episodeID string) int {
	idx := strings.LastIndex(episodeID, ":")
	if idx < 0 || idx == len(episodeID)-1 {
		return 0
	}
	n, err := strconv.Atoi(episodeID[idx+1:])
	if err != nil {
		return 0
	}
	return n
}

// parseDelta converts a signed delta string like "+3" or "-5" to an int.
func parseDelta(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
