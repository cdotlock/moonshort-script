package emitter

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// mockResolver implements AssetResolver for testing.
type mockResolver struct {
	bg         map[string]string
	characters map[string]map[string]string
	music      map[string]string
	sfx        map[string]string
	cg         map[string]string
	minigames  map[string]string
}

func newMockResolver() *mockResolver {
	return &mockResolver{
		bg: map[string]string{
			"school_classroom": "https://cdn.test/bg/school_classroom.png",
		},
		characters: map[string]map[string]string{
			"mauricio": {
				"neutral_smirk":    "https://cdn.test/characters/mauricio_neutral_smirk.png",
				"arms_crossed_angry": "https://cdn.test/characters/mauricio_arms_crossed_angry.png",
			},
		},
		music: map[string]string{
			"calm_morning": "https://cdn.test/music/calm_morning.mp3",
		},
		sfx: map[string]string{
			"phone_buzz": "https://cdn.test/sfx/phone_buzz.mp3",
		},
		cg: map[string]string{
			"window_stare": "https://cdn.test/cg/window_stare.png",
		},
		minigames: map[string]string{
			"qte_challenge": "https://cdn.test/minigames/qte_challenge/index.html",
		},
	}
}

func (m *mockResolver) ResolveBg(name string) (string, error) {
	if url, ok := m.bg[name]; ok {
		return url, nil
	}
	return "", fmt.Errorf("unknown bg %q", name)
}

func (m *mockResolver) ResolveCharacter(char, poseExpr string) (string, error) {
	if poses, ok := m.characters[char]; ok {
		if url, ok := poses[poseExpr]; ok {
			return url, nil
		}
		return "", fmt.Errorf("unknown pose %q for %q", poseExpr, char)
	}
	return "", fmt.Errorf("unknown character %q", char)
}

func (m *mockResolver) ResolveMusic(name string) (string, error) {
	if url, ok := m.music[name]; ok {
		return url, nil
	}
	return "", fmt.Errorf("unknown music %q", name)
}

func (m *mockResolver) ResolveSfx(name string) (string, error) {
	if url, ok := m.sfx[name]; ok {
		return url, nil
	}
	return "", fmt.Errorf("unknown sfx %q", name)
}

func (m *mockResolver) ResolveCg(name string) (string, error) {
	if url, ok := m.cg[name]; ok {
		return url, nil
	}
	return "", fmt.Errorf("unknown cg %q", name)
}

func (m *mockResolver) ResolveMinigame(gameID string) (string, error) {
	if url, ok := m.minigames[gameID]; ok {
		return url, nil
	}
	return "", fmt.Errorf("unknown minigame %q", gameID)
}

func TestEmitMinimal(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Butterfly",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "school_classroom", Transition: "fade"},
			&ast.NarratorNode{Text: "The hallway is empty."},
			&ast.XpNode{Delta: "+3"},
		},
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
			},
		},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Check top-level fields.
	if result["episode_id"] != "main:01" {
		t.Errorf("episode_id = %v, want %q", result["episode_id"], "main:01")
	}
	if result["branch_key"] != "main" {
		t.Errorf("branch_key = %v, want %q", result["branch_key"], "main")
	}
	if result["seq"] != float64(1) {
		t.Errorf("seq = %v, want 1", result["seq"])
	}
	if result["title"] != "Butterfly" {
		t.Errorf("title = %v, want %q", result["title"], "Butterfly")
	}

	// Check steps.
	steps, ok := result["steps"].([]interface{})
	if !ok {
		t.Fatalf("steps is not an array: %T", result["steps"])
	}
	if len(steps) != 3 {
		t.Fatalf("len(steps) = %d, want 3", len(steps))
	}

	// Step 0: bg.
	bg := steps[0].(map[string]interface{})
	if bg["type"] != "bg" {
		t.Errorf("step[0].type = %v, want bg", bg["type"])
	}
	if bg["url"] != "https://cdn.test/bg/school_classroom.png" {
		t.Errorf("step[0].url = %v", bg["url"])
	}
	if bg["transition"] != "fade" {
		t.Errorf("step[0].transition = %v", bg["transition"])
	}

	// Step 1: narrator.
	narr := steps[1].(map[string]interface{})
	if narr["type"] != "narrator" {
		t.Errorf("step[1].type = %v, want narrator", narr["type"])
	}
	if narr["text"] != "The hallway is empty." {
		t.Errorf("step[1].text = %v", narr["text"])
	}

	// Step 2: xp.
	xp := steps[2].(map[string]interface{})
	if xp["type"] != "xp" {
		t.Errorf("step[2].type = %v, want xp", xp["type"])
	}
	if xp["delta"] != float64(3) {
		t.Errorf("step[2].delta = %v, want 3", xp["delta"])
	}

	// Check gates.
	gates, ok := result["gates"].(map[string]interface{})
	if !ok {
		t.Fatalf("gates is not a map: %T", result["gates"])
	}
	if gates["default"] != "main:02" {
		t.Errorf("gates.default = %v, want main:02", gates["default"])
	}

	// No warnings expected.
	if len(em.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", em.Warnings)
	}
}

func TestEmitChoice(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "The Choice",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID:   "A",
						Mode: "brave",
						Text: "Confront him",
						Check: &ast.CheckBlock{
							Attr: "CHA",
							DC:   14,
						},
						OnSuccess: []ast.Node{
							&ast.DialogueNode{Character: "MAURICIO", Text: "You got guts."},
						},
						OnFail: []ast.Node{
							&ast.DialogueNode{Character: "MAURICIO", Text: "Nice try."},
						},
						Body: []ast.Node{
							&ast.XpNode{Delta: "+2"},
						},
					},
					{
						ID:   "B",
						Mode: "safe",
						Text: "Walk away",
						Body: []ast.Node{
							&ast.NarratorNode{Text: "You turn around quietly."},
						},
					},
				},
			},
		},
		Gates: &ast.GatesBlock{
			Gates: []*ast.Gate{
				{Target: "main:02", GateType: "default"},
			},
		},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	steps := result["steps"].([]interface{})
	if len(steps) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(steps))
	}

	choice := steps[0].(map[string]interface{})
	if choice["type"] != "choice" {
		t.Errorf("type = %v, want choice", choice["type"])
	}

	options := choice["options"].([]interface{})
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}

	// Option A: brave.
	optA := options[0].(map[string]interface{})
	if optA["id"] != "A" {
		t.Errorf("optA.id = %v", optA["id"])
	}
	if optA["mode"] != "brave" {
		t.Errorf("optA.mode = %v", optA["mode"])
	}
	if optA["text"] != "Confront him" {
		t.Errorf("optA.text = %v", optA["text"])
	}

	check := optA["check"].(map[string]interface{})
	if check["attr"] != "CHA" {
		t.Errorf("check.attr = %v", check["attr"])
	}
	if check["dc"] != float64(14) {
		t.Errorf("check.dc = %v", check["dc"])
	}

	onSuccess := optA["on_success"].([]interface{})
	if len(onSuccess) != 1 {
		t.Fatalf("len(on_success) = %d, want 1", len(onSuccess))
	}
	successStep := onSuccess[0].(map[string]interface{})
	if successStep["type"] != "dialogue" {
		t.Errorf("on_success[0].type = %v", successStep["type"])
	}

	onFail := optA["on_fail"].([]interface{})
	if len(onFail) != 1 {
		t.Fatalf("len(on_fail) = %d, want 1", len(onFail))
	}

	stepsA := optA["steps"].([]interface{})
	if len(stepsA) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(stepsA))
	}

	// Option B: safe.
	optB := options[1].(map[string]interface{})
	if optB["id"] != "B" {
		t.Errorf("optB.id = %v", optB["id"])
	}
	if optB["mode"] != "safe" {
		t.Errorf("optB.mode = %v", optB["mode"])
	}
	if optB["check"] != nil {
		t.Errorf("optB.check should be nil for safe option")
	}

	stepsB := optB["steps"].([]interface{})
	if len(stepsB) != 1 {
		t.Fatalf("len(stepsB) = %d, want 1", len(stepsB))
	}
	if stepsB[0].(map[string]interface{})["type"] != "narrator" {
		t.Errorf("optB.steps[0].type = %v", stepsB[0].(map[string]interface{})["type"])
	}
}
