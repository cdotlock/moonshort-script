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
}

// New creates an Emitter with the given asset resolver.
func New(resolver AssetResolver) *Emitter {
	return &Emitter{resolver: resolver}
}

// Emit converts an Episode AST into JSON bytes.
func (e *Emitter) Emit(ep *ast.Episode) ([]byte, error) {
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

	return json.MarshalIndent(out, "", "  ")
}

// isConcurrent checks if a node carries the concurrent (&) flag.
func isConcurrent(n ast.Node) bool {
	if hc, ok := n.(ast.HasConcurrent); ok {
		return hc.GetConcurrent()
	}
	return false
}

// emitNodes converts a slice of AST nodes into a slice of steps, grouping
// consecutive concurrent (&-prefixed) nodes into sub-arrays.
//
// Grouping rule:
//   - A non-concurrent node (@-prefixed or dialogue) starts a potential group.
//   - Following &-concurrent nodes join the group.
//   - When the next non-concurrent node arrives, the group is flushed.
//   - Single-item groups are emitted as plain objects.
//   - Multi-item groups are emitted as arrays (concurrent execution).
func (e *Emitter) emitNodes(nodes []ast.Node) []interface{} {
	steps := make([]interface{}, 0)
	var group []interface{} // current group being accumulated

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

		if isConcurrent(n) {
			// & node: join the current group
			group = append(group, step)
		} else {
			// @ or dialogue node: flush previous group, start new
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
		return map[string]interface{}{"type": "signal", "event": v.Event}
	case *ast.ButterflyNode:
		return map[string]interface{}{"type": "butterfly", "description": v.Description}
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
		"type": "cg_show",
		"name": n.Name,
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
	if len(n.Body) > 0 {
		m["steps"] = e.emitNodes(n.Body)
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
	m := map[string]interface{}{
		"type": "phone_show",
	}
	if len(n.Body) > 0 {
		m["messages"] = e.emitNodes(n.Body)
	}
	return m
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
		"type":    "minigame",
		"game_id": n.ID,
		"attr":    n.Attr,
	}
	url, err := e.resolver.ResolveMinigame(n.ID)
	if err != nil {
		e.warn("minigame %q: %v", n.ID, err)
	} else {
		m["game_url"] = url
	}
	if len(n.OnResult) > 0 {
		results := make(map[string]interface{})
		for grade, nodes := range n.OnResult {
			results[grade] = e.emitNodes(nodes)
		}
		m["on_results"] = results
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
		if len(opt.OnSuccess) > 0 {
			o["on_success"] = e.emitNodes(opt.OnSuccess)
		}
		if len(opt.OnFail) > 0 {
			o["on_fail"] = e.emitNodes(opt.OnFail)
		}
		if len(opt.Body) > 0 {
			o["steps"] = e.emitNodes(opt.Body)
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

func (e *Emitter) emitIf(n *ast.IfNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "if",
		"condition": e.emitCondition(n.Condition),
		"then":      e.emitNodes(n.Then),
	}
	if len(n.Else) > 0 {
		// If else branch is a single IfNode (@else @if chain), emit as bare object
		if len(n.Else) == 1 {
			if elseIf, ok := n.Else[0].(*ast.IfNode); ok {
				m["else"] = e.emitIf(elseIf)
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

// emitCondition converts a structured Condition to a JSON map.
func (e *Emitter) emitCondition(c *ast.Condition) map[string]interface{} {
	m := map[string]interface{}{"type": c.Type}
	switch c.Type {
	case "choice":
		m["option"] = c.Option
		m["result"] = c.Result
	case "flag":
		m["name"] = c.Name
	case "comparison", "compound":
		m["expr"] = c.Expr
	case "influence":
		m["description"] = c.Description
	}
	return m
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
