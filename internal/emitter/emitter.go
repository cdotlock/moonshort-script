// Package emitter converts an NRS AST into player-ready JSON with resolved asset URLs.
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

	if ep.Gates != nil {
		out["gates"] = e.emitGates(ep.Gates)
	}

	return json.MarshalIndent(out, "", "  ")
}

// emitNodes converts a slice of AST nodes into a slice of step maps.
func (e *Emitter) emitNodes(nodes []ast.Node) []interface{} {
	steps := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		step := e.emitNode(n)
		if step != nil {
			steps = append(steps, step)
		}
	}
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
	case *ast.CharExprNode:
		return e.emitCharExpr(v)
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
	case *ast.XpNode:
		return e.emitXp(v)
	case *ast.SanNode:
		return e.emitSan(v)
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
	case *ast.GotoNode:
		return map[string]interface{}{"type": "goto", "target": v.Name}
	default:
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
		"pose_expr": n.Pose,
		"position":  n.Position,
	}
	url, err := e.resolver.ResolveCharacter(n.Char, n.Pose)
	if err != nil {
		e.warn("char_show %q/%q: %v", n.Char, n.Pose, err)
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

func (e *Emitter) emitCharExpr(n *ast.CharExprNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "char_expr",
		"character": n.Char,
		"pose_expr": n.Pose,
	}
	url, err := e.resolver.ResolveCharacter(n.Char, n.Pose)
	if err != nil {
		e.warn("char_expr %q/%q: %v", n.Char, n.Pose, err)
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
		"character": n.Character,
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
		"character": n.Char,
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

func (e *Emitter) emitXp(n *ast.XpNode) map[string]interface{} {
	return map[string]interface{}{
		"type":  "xp",
		"delta": parseDelta(n.Delta),
	}
}

func (e *Emitter) emitSan(n *ast.SanNode) map[string]interface{} {
	return map[string]interface{}{
		"type":  "san",
		"delta": parseDelta(n.Delta),
	}
}

func (e *Emitter) emitAffection(n *ast.AffectionNode) map[string]interface{} {
	return map[string]interface{}{
		"type":      "affection",
		"character": n.Char,
		"delta":     parseDelta(n.Delta),
	}
}

func (e *Emitter) emitIf(n *ast.IfNode) map[string]interface{} {
	m := map[string]interface{}{
		"type":      "if",
		"condition": n.Condition,
		"then":      e.emitNodes(n.Then),
	}
	if len(n.Else) > 0 {
		m["else"] = e.emitNodes(n.Else)
	}
	return m
}

func (e *Emitter) emitGates(g *ast.GatesBlock) map[string]interface{} {
	result := map[string]interface{}{}
	rules := make([]interface{}, 0)
	for _, gate := range g.Gates {
		if gate.GateType == "default" {
			result["default"] = gate.Target
			continue
		}
		rule := map[string]interface{}{
			"target": gate.Target,
			"type":   gate.GateType,
		}
		if gate.Condition != "" {
			rule["condition"] = gate.Condition
		}
		if gate.Trigger != nil {
			trigger := map[string]interface{}{}
			if gate.Trigger.OptionID != "" {
				trigger["option_id"] = gate.Trigger.OptionID
			}
			if gate.Trigger.CheckResult != "" {
				trigger["check_result"] = gate.Trigger.CheckResult
			}
			rule["trigger"] = trigger
		}
		rules = append(rules, rule)
	}
	result["rules"] = rules
	return result
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
