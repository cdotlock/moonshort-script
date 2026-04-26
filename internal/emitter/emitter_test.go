package emitter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/parser"
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
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
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
	if len(steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(steps))
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

	// Check gate.
	gate, ok := result["gate"].(map[string]interface{})
	if !ok {
		t.Fatalf("gate should be an object, got %T", result["gate"])
	}
	if gate["next"] != "main:02" {
		t.Errorf("gate.next = %v, want main:02", gate["next"])
	}
	if gate["if"] != nil {
		t.Error("gate.if should be nil for unconditional route")
	}

	// No warnings expected.
	if len(em.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", em.Warnings)
	}
}

func TestEmitChoice(t *testing.T) {
	// Brave option body now uses @if (check.success) { ... } @else { ... }.
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
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then: []ast.Node{
									&ast.DialogueNode{Character: "MAURICIO", Text: "You got guts."},
								},
								Else: []ast.Node{
									&ast.DialogueNode{Character: "MAURICIO", Text: "Nice try."},
								},
							},
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
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Target: "main:02"},
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

	// on_success/on_fail are gone — the body is emitted as "steps".
	if _, has := optA["on_success"]; has {
		t.Error("brave option should not have 'on_success' key (body is in 'steps' now)")
	}
	if _, has := optA["on_fail"]; has {
		t.Error("brave option should not have 'on_fail' key (body is in 'steps' now)")
	}

	stepsA := optA["steps"].([]interface{})
	if len(stepsA) != 1 {
		t.Fatalf("len(optA.steps) = %d, want 1 (the @if tree)", len(stepsA))
	}
	ifStep := stepsA[0].(map[string]interface{})
	if ifStep["type"] != "if" {
		t.Errorf("optA.steps[0].type = %v, want 'if'", ifStep["type"])
	}
	cond := ifStep["condition"].(map[string]interface{})
	if cond["type"] != "check" {
		t.Errorf("optA.steps[0].condition.type = %v, want 'check'", cond["type"])
	}
	if cond["result"] != "success" {
		t.Errorf("optA.steps[0].condition.result = %v, want 'success'", cond["result"])
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

func TestEmitConcurrentGrouping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Concurrent",
		Body: []ast.Node{
			// Group: bg (leader) + music (concurrent) + char_show (concurrent)
			&ast.BgSetNode{Name: "school_classroom", Transition: "fade"},
			func() ast.Node {
				n := &ast.MusicPlayNode{Track: "calm_morning"}
				n.SetConcurrent(true)
				return n
			}(),
			func() ast.Node {
				n := &ast.CharShowNode{Char: "mauricio", Look: "neutral_smirk", Position: "right"}
				n.SetConcurrent(true)
				return n
			}(),
			// Standalone dialogue
			&ast.NarratorNode{Text: "Hello."},
			// Pause
			&ast.PauseNode{Clicks: 1},
		},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{{Target: "main:02"}},
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
	if len(steps) != 3 {
		t.Fatalf("len(steps) = %d, want 3 (1 group + 1 narrator + 1 pause)", len(steps))
	}

	// Step 0: concurrent group (array of 3 items)
	group, ok := steps[0].([]interface{})
	if !ok {
		t.Fatalf("steps[0] should be an array (concurrent group), got %T", steps[0])
	}
	if len(group) != 3 {
		t.Fatalf("concurrent group length: got %d, want 3", len(group))
	}
	bgStep := group[0].(map[string]interface{})
	if bgStep["type"] != "bg" {
		t.Errorf("group[0].type = %v, want bg", bgStep["type"])
	}
	musicStep := group[1].(map[string]interface{})
	if musicStep["type"] != "music_play" {
		t.Errorf("group[1].type = %v, want music_play", musicStep["type"])
	}
	charStep := group[2].(map[string]interface{})
	if charStep["type"] != "char_show" {
		t.Errorf("group[2].type = %v, want char_show", charStep["type"])
	}

	// Step 1: narrator (standalone object)
	narr := steps[1].(map[string]interface{})
	if narr["type"] != "narrator" {
		t.Errorf("steps[1].type = %v, want narrator", narr["type"])
	}

	// Step 2: pause
	pauseStep := steps[2].(map[string]interface{})
	if pauseStep["type"] != "pause" {
		t.Errorf("steps[2].type = %v, want pause", pauseStep["type"])
	}
	if pauseStep["clicks"] != float64(1) {
		t.Errorf("steps[2].clicks = %v, want 1", pauseStep["clicks"])
	}
}

func TestEmitDialogueLowercase(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.DialogueNode{Character: "JOSIE", Text: "Hi."},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	if !strings.Contains(string(data), `"character": "josie"`) {
		t.Error("expected lowercase character name in dialogue")
	}
}

func TestEmitGateNull(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{&ast.NarratorNode{Text: "Hi."}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	if !strings.Contains(string(data), `"gate": null`) {
		t.Error("expected gate: null when no gate")
	}
}

func TestEmitGateConditional(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{Condition: &ast.ChoiceCondition{Option: "A", Result: "fail"}, Target: "bad:01"},
				{Condition: &ast.FlagCondition{Name: "EP01_DONE"}, Target: "mid:01"},
				{Target: "main:02"},
			},
		},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	s := string(data)
	// Should have nested if/else
	if !strings.Contains(s, `"if"`) {
		t.Error("expected nested if in gate")
	}
	if !strings.Contains(s, `"else"`) {
		t.Error("expected else in gate chain")
	}
}

func TestEmitConditionTypes(t *testing.T) {
	tests := []struct {
		name string
		cond ast.Condition
	}{
		{"choice", &ast.ChoiceCondition{Option: "A", Result: "fail"}},
		{"flag", &ast.FlagCondition{Name: "EP01"}},
		{"comparison", &ast.ComparisonCondition{
			Left: ast.ComparisonOperand{Kind: ast.OperandValue, Name: "x"},
			Op:   ">=",
			Right: 5,
		}},
		{"influence", &ast.InfluenceCondition{Description: "desc"}},
		{"compound", &ast.CompoundCondition{
			Op:    "&&",
			Left:  &ast.FlagCondition{Name: "a"},
			Right: &ast.FlagCondition{Name: "b"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &ast.Episode{
				BranchKey: "main:01", Title: "T",
				Body: []ast.Node{
					&ast.IfNode{Condition: tt.cond, Then: []ast.Node{&ast.NarratorNode{Text: "Hi."}}},
				},
				Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
			}
			em := New(newMockResolver())
			data, _ := em.Emit(ep)
			s := string(data)
			if !strings.Contains(s, fmt.Sprintf(`"type": "%s"`, tt.name)) {
				t.Errorf("expected condition type %q in output", tt.name)
			}
		})
	}
}

func TestEmitElseIfUnwrap(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "A"},
				Then:      []ast.Node{&ast.NarratorNode{Text: "a"}},
				Else: []ast.Node{
					&ast.IfNode{
						Condition: &ast.FlagCondition{Name: "B"},
						Then:      []ast.Node{&ast.NarratorNode{Text: "b"}},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	s := string(data)
	// The else should be a bare object (not wrapped in array)
	// Check that "else": { appears, not "else": [
	if strings.Contains(s, `"else": [`) {
		t.Error("@else @if should produce bare object, not array")
	}
}

func TestEmitAllNodeTypes(t *testing.T) {
	// Ensure every node type emits without panic
	nodes := []ast.Node{
		&ast.BgSetNode{Name: "bg1", Transition: "fade"},
		&ast.CharShowNode{Char: "c", Look: "l", Position: "left"},
		&ast.CharHideNode{Char: "c", Transition: "fade"},
		&ast.CharLookNode{Char: "c", Look: "l"},
		&ast.CharMoveNode{Char: "c", Position: "right"},
		&ast.CharBubbleNode{Char: "c", BubbleType: "heart"},
		&ast.CgShowNode{Name: "cg1"},
		&ast.DialogueNode{Character: "CHAR", Text: "hi"},
		&ast.NarratorNode{Text: "n"},
		&ast.YouNode{Text: "y"},
		&ast.PhoneShowNode{Body: []ast.Node{
			&ast.TextMessageNode{Direction: "from", Char: "C", Content: "hi"},
		}},
		&ast.PhoneHideNode{},
		&ast.MusicPlayNode{Track: "t"},
		&ast.MusicCrossfadeNode{Track: "t"},
		&ast.MusicFadeoutNode{},
		&ast.SfxPlayNode{Sound: "s"},
		&ast.AffectionNode{Char: "c", Delta: "+1"},
		&ast.SignalNode{Kind: "mark", Event: "E"},
		&ast.ButterflyNode{Description: "d"},
		&ast.LabelNode{Name: "L"},
		&ast.GotoNode{Name: "L"},
		&ast.PauseNode{Clicks: 1},
	}
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body:      nodes,
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty output")
	}
}

func TestEmitConcurrentGroups(t *testing.T) {
	bg := &ast.BgSetNode{Name: "bg1"}
	music := &ast.MusicPlayNode{Track: "t"}
	music.SetConcurrent(true)
	char := &ast.CharShowNode{Char: "c", Look: "l", Position: "left"}
	char.SetConcurrent(true)
	narrator := &ast.NarratorNode{Text: "hi"}

	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body:      []ast.Node{bg, music, char, narrator},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	s := string(data)
	// The first 3 should be grouped as array, narrator separate
	if !strings.Contains(s, `"steps": [`) {
		t.Error("expected steps array")
	}
}

func TestEmitAssetWarning(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01", Title: "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "nonexistent"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	em.Emit(ep)
	if len(em.Warnings) == 0 {
		t.Error("expected warning for unknown asset")
	}
}

// TestEmitEndingKey verifies that @ending produces a structured "ending" key
// and that its absence produces a null "ending" key (always present for consumers).
func TestEmitEndingKey(t *testing.T) {
	r := &mockResolver{}
	e := New(r)

	ep := &ast.Episode{
		BranchKey: "main:15",
		Title:     "Finale",
		Body:      []ast.Node{&ast.NarratorNode{Text: "The end."}},
		Ending:    &ast.EndingNode{Type: ast.EndingComplete},
	}
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("Emit err: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	ending, ok := parsed["ending"].(map[string]interface{})
	if !ok {
		t.Fatalf("ending: got %v, want map[string]interface{}", parsed["ending"])
	}
	if ending["type"] != "complete" {
		t.Errorf("ending.type: got %v, want complete", ending["type"])
	}
	if parsed["gate"] != nil {
		t.Errorf("gate: got %v, want nil (ending is set)", parsed["gate"])
	}
}

func TestEmitEndingAbsent(t *testing.T) {
	r := &mockResolver{}
	e := New(r)

	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "Normal",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{{Target: "main:02"}},
		},
	}
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("Emit err: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	if _, present := parsed["ending"]; !present {
		t.Error("ending key should always be present (null when absent)")
	}
	if parsed["ending"] != nil {
		t.Errorf("ending: got %v, want nil", parsed["ending"])
	}
}

// TestEmitAchievementStep verifies that @achievement emits an inline step
// carrying id, name, rarity, and description.
func TestEmitAchievementStep(t *testing.T) {
	r := &mockResolver{}
	e := New(r)

	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.AchievementNode{
				ID:          "HEEL_WARRIOR",
				Name:        "Heel Warrior",
				Rarity:      ast.RarityRare,
				Description: "desc",
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("Emit err: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, has := parsed["achievements"]; has {
		t.Error("top-level 'achievements' key should not be present")
	}
	steps := parsed["steps"].([]interface{})
	if len(steps) != 1 {
		t.Fatalf("steps len: got %d, want 1", len(steps))
	}
	step := steps[0].(map[string]interface{})
	if step["type"] != "achievement" {
		t.Errorf("step type: got %v, want 'achievement'", step["type"])
	}
	for _, key := range []string{"id", "name", "rarity", "description"} {
		if _, ok := step[key]; !ok {
			t.Errorf("achievement step missing %q key", key)
		}
	}
	if step["rarity"] != "rare" {
		t.Errorf("rarity: got %v", step["rarity"])
	}
}

func TestEmitNoTopLevelAchievementsKey(t *testing.T) {
	r := &mockResolver{}
	e := New(r)

	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body:      []ast.Node{&ast.NarratorNode{Text: "hi"}},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("Emit err: %v", err)
	}
	if strings.Contains(string(out), `"achievements"`) {
		t.Errorf("top-level 'achievements' key should not appear when body has no achievement step:\n%s", string(out))
	}
}

// TestEmitSignalKind verifies the emitter includes the signal kind in the
// output step. The kind slot is retained so future kinds can be added
// without breaking consumers.
func TestEmitSignalKind(t *testing.T) {
	r := &mockResolver{}
	e := New(r)

	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.SignalNode{Kind: ast.SignalKindMark, Event: "A"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	out, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("Emit err: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `"kind": "mark"`) {
		t.Errorf("missing mark kind:\n%s", s)
	}
	if !strings.Contains(s, `"event": "A"`) {
		t.Errorf("missing event:\n%s", s)
	}
}

// firstBodyStep parses + emits an episode source and returns the first
// step of the resulting JSON, as a generic map.
func firstBodyStep(t *testing.T, src string) map[string]interface{} {
	t.Helper()
	l := lexer.New(src)
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	e := New(newMockResolver())
	data, err := e.Emit(ep)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	steps := m["steps"].([]interface{})
	first := steps[0]
	if arr, ok := first.([]interface{}); ok && len(arr) == 1 {
		return arr[0].(map[string]interface{})
	}
	return first.(map[string]interface{})
}

func assertStepEquals(t *testing.T, got, want map[string]interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("step mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestEmitSignalIntAssign(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections = 0
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"id":    "0001_sig",
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "=",
		"value": float64(0),
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalIntAdd(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections +1
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"id":    "0001_sig",
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "+",
		"value": float64(1),
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalIntSub(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal int rejections -2
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"id":    "0001_sig",
		"type":  "signal",
		"kind":  "int",
		"name":  "rejections",
		"op":    "-",
		"value": float64(2),
	}
	assertStepEquals(t, step, want)
}

func TestEmitSignalMarkUnchanged(t *testing.T) {
	src := `@episode main:01 "t" {
  @signal mark HIGH_HEEL_EP05
  @ending complete
}`
	step := firstBodyStep(t, src)
	want := map[string]interface{}{
		"id":    "0001_sig",
		"type":  "signal",
		"kind":  "mark",
		"event": "HIGH_HEEL_EP05",
	}
	assertStepEquals(t, step, want)
}

func TestEmitIfReadsIntVariableAsComparison(t *testing.T) {
	src := `@episode main:01 "t" {
  @if (rejections >= 3) {
    NARRATOR: too many
  }
  @ending complete
}`
	step := firstBodyStep(t, src)
	if step["type"] != "if" {
		t.Fatalf("expected if, got %v", step["type"])
	}
	cond := step["condition"].(map[string]interface{})
	if cond["type"] != "comparison" {
		t.Fatalf("expected comparison, got %v", cond["type"])
	}
	left := cond["left"].(map[string]interface{})
	if left["kind"] != "value" || left["name"] != "rejections" {
		t.Fatalf("unexpected left: %#v", left)
	}
	if cond["op"] != ">=" {
		t.Fatalf("unexpected op: %v", cond["op"])
	}
	if cond["right"].(float64) != 3 {
		t.Fatalf("unexpected right: %v", cond["right"])
	}
}

// ---------- Step ID tests (Task 1: Stable Step ID & Content-Addressed Cursor) ----------

// TestStepTypeTag verifies the type-tag mapping is exactly the documented
// contract. Backends key persisted player cursors on these tags — changing
// any value here is a breaking schema change requiring a data migration.
func TestStepTypeTag(t *testing.T) {
	tests := []struct {
		stepType string
		want     string
	}{
		{"dialogue", "dlg"},
		{"narrator", "nar"},
		{"you", "you"},
		{"pause", "pau"},
		{"choice", "ch"},
		{"minigame", "mg"},
		{"cg_show", "cg"},
		{"bg", "bg"},
		{"char_show", "char"},
		{"char_hide", "char"},
		{"char_look", "char"},
		{"char_move", "char"},
		{"bubble", "char"},
		{"music_play", "mus"},
		{"music_crossfade", "mus"},
		{"music_fadeout", "mus"},
		{"sfx_play", "sfx"},
		{"phone_show", "phn"},
		{"phone_hide", "phn"},
		{"text_message", "phn"},
		{"signal", "sig"},
		{"affection", "aff"},
		{"achievement", "ach"},
		{"butterfly", "btf"},
		{"if", "ctrl"},
		{"goto", "ctrl"},
		{"label", "ctrl"},
		{"unknown_future_type", "unk"},
	}
	for _, tt := range tests {
		t.Run(tt.stepType, func(t *testing.T) {
			if got := stepTypeTag(tt.stepType); got != tt.want {
				t.Errorf("stepTypeTag(%q) = %q, want %q", tt.stepType, got, tt.want)
			}
		})
	}
}

// TestStepIDFormatTopLevel verifies that top-level steps get sequential
// container-scoped 4-digit ids, in declaration order.
func TestStepIDFormatTopLevel(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "1"},
			&ast.DialogueNode{Character: "JOSIE", Text: "2"},
			&ast.YouNode{Text: "3"},
			&ast.PauseNode{Clicks: 1},
			&ast.BgSetNode{Name: "x"},
			&ast.MusicPlayNode{Track: "m"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	steps := result["steps"].([]interface{})

	wantIDs := []string{"0001_nar", "0002_dlg", "0003_you", "0004_pau", "0005_bg", "0006_mus"}
	if len(steps) != len(wantIDs) {
		t.Fatalf("len(steps)=%d, want %d", len(steps), len(wantIDs))
	}
	for i, want := range wantIDs {
		step := steps[i].(map[string]interface{})
		if got := step["id"]; got != want {
			t.Errorf("steps[%d].id = %v, want %q", i, got, want)
		}
	}
}

// TestStepIDConcurrentGroupSharesParentCounter verifies that a concurrent
// group consumes sequential seqs from the parent container — it is NOT a
// nested 0001 restart. The group as a JSON array does not itself carry an
// id; only the steps inside it do.
func TestStepIDConcurrentGroupSharesParentCounter(t *testing.T) {
	bg := &ast.BgSetNode{Name: "bg1"}
	music := &ast.MusicPlayNode{Track: "calm_morning"}
	music.SetConcurrent(true)
	char := &ast.CharShowNode{Char: "mauricio", Look: "neutral_smirk", Position: "right"}
	char.SetConcurrent(true)
	narr := &ast.NarratorNode{Text: "after"}

	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body:      []ast.Node{bg, music, char, narr},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})

	// steps[0] should be a concurrent group (array of 3) consuming seqs 1,2,3.
	group, ok := steps[0].([]interface{})
	if !ok {
		t.Fatalf("steps[0] should be array, got %T", steps[0])
	}
	if len(group) != 3 {
		t.Fatalf("group length = %d, want 3", len(group))
	}
	wantGroupIDs := []string{"0001_bg", "0002_mus", "0003_char"}
	for i, want := range wantGroupIDs {
		g := group[i].(map[string]interface{})
		if g["id"] != want {
			t.Errorf("group[%d].id = %v, want %q", i, g["id"], want)
		}
	}

	// steps[1] is the standalone narrator; it should consume seq 4
	// (NOT restart at 0001) because the concurrent group did not open a
	// new container.
	narrStep := steps[1].(map[string]interface{})
	if narrStep["id"] != "0004_nar" {
		t.Errorf("narrator after group should be 0004_nar, got %v", narrStep["id"])
	}
}

// TestStepIDChoiceContainerScoping verifies that each option's body
// restarts the seq counter at 0001 — option A's first step is 0001_*
// even though the choice itself is somewhere later in the parent counter,
// and option B's first step is also 0001_*.
func TestStepIDChoiceContainerScoping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "intro"}, // 0001_nar
			&ast.NarratorNode{Text: "intro2"}, // 0002_nar
			&ast.ChoiceNode{ // 0003_ch
				Options: []*ast.OptionNode{
					{
						ID:   "A",
						Mode: "safe",
						Text: "A",
						Body: []ast.Node{
							&ast.DialogueNode{Character: "X", Text: "a1"},
							&ast.NarratorNode{Text: "a2"},
							&ast.DialogueNode{Character: "X", Text: "a3"},
						},
					},
					{
						ID:   "B",
						Mode: "safe",
						Text: "B",
						Body: []ast.Node{
							&ast.DialogueNode{Character: "X", Text: "b1"},
							&ast.DialogueNode{Character: "X", Text: "b2"},
							&ast.NarratorNode{Text: "b3"},
						},
					},
				},
			},
			&ast.NarratorNode{Text: "outro"}, // 0004_nar (continues parent counter past choice)
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	// Top-level: 0001_nar, 0002_nar, 0003_ch, 0004_nar
	for i, want := range []string{"0001_nar", "0002_nar", "0003_ch", "0004_nar"} {
		got := steps[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("top-level steps[%d].id = %v, want %q", i, got, want)
		}
	}

	choice := steps[2].(map[string]interface{})
	options := choice["options"].([]interface{})

	// Option A body: 0001_dlg, 0002_nar, 0003_dlg
	stepsA := options[0].(map[string]interface{})["steps"].([]interface{})
	for i, want := range []string{"0001_dlg", "0002_nar", "0003_dlg"} {
		got := stepsA[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("optA.steps[%d].id = %v, want %q (option container should restart at 0001)", i, got, want)
		}
	}

	// Option B body: 0001_dlg, 0002_dlg, 0003_nar
	stepsB := options[1].(map[string]interface{})["steps"].([]interface{})
	for i, want := range []string{"0001_dlg", "0002_dlg", "0003_nar"} {
		got := stepsB[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("optB.steps[%d].id = %v, want %q (option container should restart at 0001)", i, got, want)
		}
	}
}

// TestStepIDIfContainerScoping verifies that if.then and if.else each
// restart at 0001 independently.
func TestStepIDIfContainerScoping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "X"},
				Then: []ast.Node{
					&ast.NarratorNode{Text: "t1"},
					&ast.DialogueNode{Character: "X", Text: "t2"},
				},
				Else: []ast.Node{
					&ast.DialogueNode{Character: "X", Text: "e1"},
					&ast.NarratorNode{Text: "e2"},
					&ast.PauseNode{Clicks: 1},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	ifStep := steps[0].(map[string]interface{})
	if ifStep["id"] != "0001_ctrl" {
		t.Errorf("if step id = %v, want 0001_ctrl", ifStep["id"])
	}

	thenBranch := ifStep["then"].([]interface{})
	for i, want := range []string{"0001_nar", "0002_dlg"} {
		got := thenBranch[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("then[%d].id = %v, want %q", i, got, want)
		}
	}

	elseBranch := ifStep["else"].([]interface{})
	for i, want := range []string{"0001_dlg", "0002_nar", "0003_pau"} {
		got := elseBranch[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("else[%d].id = %v, want %q", i, got, want)
		}
	}
}

// TestStepIDPhoneShowContainerScoping verifies that phone_show.messages is
// its own container, restarting at 0001.
func TestStepIDPhoneShowContainerScoping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "before"}, // 0001_nar
			&ast.PhoneShowNode{ // 0002_phn
				Body: []ast.Node{
					&ast.TextMessageNode{Direction: "from", Char: "easton", Content: "hi"},
					&ast.TextMessageNode{Direction: "to", Char: "malia", Content: "yo"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})

	if steps[0].(map[string]interface{})["id"] != "0001_nar" {
		t.Errorf("steps[0].id = %v, want 0001_nar", steps[0].(map[string]interface{})["id"])
	}
	phone := steps[1].(map[string]interface{})
	if phone["id"] != "0002_phn" {
		t.Errorf("phone_show id = %v, want 0002_phn", phone["id"])
	}

	messages := phone["messages"].([]interface{})
	for i, want := range []string{"0001_phn", "0002_phn"} {
		got := messages[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("messages[%d].id = %v, want %q (phone_show messages should restart at 0001)", i, got, want)
		}
	}
}

// TestStepIDMinigameContainerScoping verifies minigame.steps restarts at 0001.
func TestStepIDMinigameContainerScoping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "intro"}, // 0001_nar
			&ast.MinigameNode{ // 0002_mg
				ID:          "qte_challenge",
				Attr:        "ATK",
				Description: "d",
				Body: []ast.Node{
					&ast.NarratorNode{Text: "mg1"},
					&ast.DialogueNode{Character: "X", Text: "mg2"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})
	mg := steps[1].(map[string]interface{})
	if mg["id"] != "0002_mg" {
		t.Errorf("minigame id = %v, want 0002_mg", mg["id"])
	}
	mgSteps := mg["steps"].([]interface{})
	for i, want := range []string{"0001_nar", "0002_dlg"} {
		got := mgSteps[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("minigame.steps[%d].id = %v, want %q", i, got, want)
		}
	}
}

// TestStepIDCgShowContainerScoping verifies cg_show.steps restarts at 0001.
func TestStepIDCgShowContainerScoping(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.CgShowNode{
				Name:     "window_stare",
				Duration: "medium",
				Content:  "x",
				Body: []ast.Node{
					&ast.YouNode{Text: "y1"},
					&ast.YouNode{Text: "y2"},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})
	cg := steps[0].(map[string]interface{})
	if cg["id"] != "0001_cg" {
		t.Errorf("cg_show id = %v, want 0001_cg", cg["id"])
	}
	cgSteps := cg["steps"].([]interface{})
	for i, want := range []string{"0001_you", "0002_you"} {
		got := cgSteps[i].(map[string]interface{})["id"]
		if got != want {
			t.Errorf("cg_show.steps[%d].id = %v, want %q", i, got, want)
		}
	}
}

// TestStepIDDeterminism verifies that compiling the same source twice
// produces byte-identical output — the id assignment must be a pure
// function of declaration order.
func TestStepIDDeterminism(t *testing.T) {
	src := `@episode main:01 "T" {
  @bg set school_classroom fade
  &music play calm_morning
  NARRATOR: line one.
  YOU: thinking.
  @pause for 1
  @choice {
    @option A safe "go" {
      NARRATOR: a1
      X: hi
    }
    @option B safe "stay" {
      X: b1
      NARRATOR: b2
    }
  }
  @if (FLAG_X) {
    NARRATOR: t1
    X: t2
  } @else {
    X: e1
  }
  @signal mark DONE
  @ending complete
}`

	emit := func() string {
		l := lexer.New(src)
		p := parser.New(l)
		ep, err := p.Parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		em := New(newMockResolver())
		data, err := em.Emit(ep)
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
		return string(data)
	}

	first := emit()
	second := emit()
	if first != second {
		t.Errorf("compile is non-deterministic; identical source produced different output")
	}

	// And every step in the output must carry an id field.
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(first), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var walkAndCheck func(v interface{}, path string)
	walkAndCheck = func(v interface{}, path string) {
		switch x := v.(type) {
		case map[string]interface{}:
			// A "step" is a map that has a "type" field. (Conditions and
			// option records also have "type" but live under known parent
			// keys — we identify steps by checking for "type" + presence
			// at a known step location. Easier rule: if it's a map with
			// a "type" string AND the type is in stepTypeTag's table,
			// it's a step and must have id.)
			if typeVal, ok := x["type"].(string); ok && stepTypeTag(typeVal) != "unk" {
				// Skip conditions: they have their own type vocabulary
				// (choice/flag/comparison/influence/compound/check/rating)
				// that doesn't overlap with step types — except "choice"
				// is both a step type and a condition type. Disambiguate
				// by checking parent path.
				isCondition := strings.Contains(path, ".condition") ||
					strings.HasPrefix(path, "root.gate") &&
						(strings.HasSuffix(path, ".if") || strings.HasSuffix(path, ".left") || strings.HasSuffix(path, ".right"))
				if !isCondition {
					if _, hasID := x["id"]; !hasID {
						t.Errorf("step at %s (type=%s) missing id field", path, typeVal)
					}
				}
			}
			for k, val := range x {
				walkAndCheck(val, path+"."+k)
			}
		case []interface{}:
			for i, val := range x {
				walkAndCheck(val, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walkAndCheck(result, "root")
}

// TestStepIDUniqueWithinContainer verifies that within any single
// container, no two steps share the same id. (Across containers, ids
// repeat — that's the whole point of container-scoped seqs — but the
// container-escape segments in a cursor make the full path unique.)
func TestStepIDUniqueWithinContainer(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.NarratorNode{Text: "1"},
			&ast.NarratorNode{Text: "2"},
			&ast.NarratorNode{Text: "3"},
			&ast.NarratorNode{Text: "4"},
			&ast.NarratorNode{Text: "5"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:02"}}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})
	seen := map[string]bool{}
	for i, s := range steps {
		id := s.(map[string]interface{})["id"].(string)
		if seen[id] {
			t.Errorf("duplicate id %q at index %d", id, i)
		}
		seen[id] = true
	}
}
