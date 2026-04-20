package emitter

// Deep audit tests for MSS emitter JSON output correctness.
// Each test constructs an AST directly, emits JSON, and verifies exact structure.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/ast"
)

// ---------- Audit A: Concurrent grouping correctness ----------

func TestAuditA_ConcurrentGrouping(t *testing.T) {
	// MSS equivalent:
	//   @bg set school_classroom
	//   &music play calm_morning
	//   &malia show neutral_phone at left
	//   NARRATOR: Hello.
	//   @malia look worried
	//   &josie show cheerful_wave at right
	//   YOU: Thinking.

	musicNode := &ast.MusicPlayNode{Track: "calm_morning"}
	musicNode.SetConcurrent(true)
	charShowNode := &ast.CharShowNode{Char: "malia", Look: "neutral_phone", Position: "left"}
	charShowNode.SetConcurrent(true)
	josieNode := &ast.CharShowNode{Char: "josie", Look: "cheerful_wave", Position: "right"}
	josieNode.SetConcurrent(true)

	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "school_classroom"},
			musicNode,
			charShowNode,
			&ast.NarratorNode{Text: "Hello."},
			&ast.CharLookNode{Char: "malia", Look: "worried"},
			josieNode,
			&ast.YouNode{Text: "Thinking."},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
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

	// Expected: 4 top-level items
	//   [0] = array of 3 (bg + music + char_show) — concurrent group
	//   [1] = object narrator
	//   [2] = array of 2 (char_look + char_show) — concurrent group
	//   [3] = object you
	if len(steps) != 4 {
		t.Fatalf("len(steps) = %d, want 4", len(steps))
	}

	// Step 0: concurrent group (array)
	group0, ok := steps[0].([]interface{})
	if !ok {
		t.Fatalf("steps[0] should be array (concurrent group), got %T", steps[0])
	}
	if len(group0) != 3 {
		t.Fatalf("steps[0] length = %d, want 3", len(group0))
	}
	assertType(t, group0[0], "bg", "steps[0][0]")
	assertType(t, group0[1], "music_play", "steps[0][1]")
	assertType(t, group0[2], "char_show", "steps[0][2]")

	// Step 1: narrator (object, not array)
	narr, ok := steps[1].(map[string]interface{})
	if !ok {
		t.Fatalf("steps[1] should be object, got %T", steps[1])
	}
	assertType(t, narr, "narrator", "steps[1]")
	if narr["text"] != "Hello." {
		t.Errorf("steps[1].text = %v, want 'Hello.'", narr["text"])
	}

	// Step 2: concurrent group of 2
	group2, ok := steps[2].([]interface{})
	if !ok {
		t.Fatalf("steps[2] should be array (concurrent group), got %T", steps[2])
	}
	if len(group2) != 2 {
		t.Fatalf("steps[2] length = %d, want 2", len(group2))
	}
	assertType(t, group2[0], "char_look", "steps[2][0]")
	assertType(t, group2[1], "char_show", "steps[2][1]")

	// Step 3: you (object, not array)
	you, ok := steps[3].(map[string]interface{})
	if !ok {
		t.Fatalf("steps[3] should be object, got %T", steps[3])
	}
	assertType(t, you, "you", "steps[3]")
}

// ---------- Audit B: Gate if/else chain ----------

func TestAuditB_GateIfElseChain(t *testing.T) {
	// @gate {
	//   @if (A.fail): @next bad:01
	//   @else @if (influence "Player shows empathy"): @next route:01
	//   @else: @next main:02
	// }
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{
					Condition: &ast.ChoiceCondition{Option: "A", Result: "fail"},
					Target:    "bad:01",
				},
				{
					Condition: &ast.InfluenceCondition{Description: "Player shows empathy"},
					Target:    "route:01",
				},
				{
					Target: "main:02",
				},
			},
		},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	gate := result["gate"].(map[string]interface{})

	// Level 1: gate.if.type == "choice"
	gateIf := gate["if"].(map[string]interface{})
	if gateIf["type"] != "choice" {
		t.Errorf("gate.if.type = %v, want 'choice'", gateIf["type"])
	}
	if gateIf["option"] != "A" {
		t.Errorf("gate.if.option = %v, want 'A'", gateIf["option"])
	}
	if gateIf["result"] != "fail" {
		t.Errorf("gate.if.result = %v, want 'fail'", gateIf["result"])
	}
	if gate["next"] != "bad:01" {
		t.Errorf("gate.next = %v, want 'bad:01'", gate["next"])
	}

	// Level 2: gate.else.if.type == "influence"
	gateElse := gate["else"].(map[string]interface{})
	elseIf := gateElse["if"].(map[string]interface{})
	if elseIf["type"] != "influence" {
		t.Errorf("gate.else.if.type = %v, want 'influence'", elseIf["type"])
	}
	if elseIf["description"] != "Player shows empathy" {
		t.Errorf("gate.else.if.description = %v", elseIf["description"])
	}
	if gateElse["next"] != "route:01" {
		t.Errorf("gate.else.next = %v, want 'route:01'", gateElse["next"])
	}

	// Level 3: gate.else.else.next == "main:02" (fallback, no "if")
	fallback := gateElse["else"].(map[string]interface{})
	if fallback["next"] != "main:02" {
		t.Errorf("gate.else.else.next = %v, want 'main:02'", fallback["next"])
	}
	if _, hasIf := fallback["if"]; hasIf {
		t.Error("gate.else.else should NOT have 'if' key (it's the fallback)")
	}
}

// ---------- Audit C: @else @if body output ----------

func TestAuditC_ElseIfBodyOutput(t *testing.T) {
	// @if (flag_A) { NARRATOR: A. }
	// @else @if (flag_B) { NARRATOR: B. }
	// @else { NARRATOR: C. }
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "flag_A"},
				Then:      []ast.Node{&ast.NarratorNode{Text: "A."}},
				Else: []ast.Node{
					&ast.IfNode{
						Condition: &ast.FlagCondition{Name: "flag_B"},
						Then:      []ast.Node{&ast.NarratorNode{Text: "B."}},
						Else:      []ast.Node{&ast.NarratorNode{Text: "C."}},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	ifNode := steps[0].(map[string]interface{})

	// Level 1: else should be a BARE OBJECT (not wrapped in array)
	elseVal := ifNode["else"]
	elseObj, isObj := elseVal.(map[string]interface{})
	if !isObj {
		t.Fatalf("level 1 else should be bare object, got %T", elseVal)
	}
	if elseObj["type"] != "if" {
		t.Errorf("level 1 else.type = %v, want 'if'", elseObj["type"])
	}

	// Level 2: else should be an ARRAY (terminal else)
	elseVal2 := elseObj["else"]
	elseArr, isArr := elseVal2.([]interface{})
	if !isArr {
		t.Fatalf("level 2 else should be array, got %T", elseVal2)
	}
	if len(elseArr) != 1 {
		t.Fatalf("level 2 else length = %d, want 1", len(elseArr))
	}
	cNode := elseArr[0].(map[string]interface{})
	if cNode["type"] != "narrator" {
		t.Errorf("level 2 else[0].type = %v, want 'narrator'", cNode["type"])
	}
	if cNode["text"] != "C." {
		t.Errorf("level 2 else[0].text = %v, want 'C.'", cNode["text"])
	}

	// Verify raw JSON: "else": { (bare object), not "else": [
	s := string(data)
	// Find the first "else" — should be followed by { not [
	idx := strings.Index(s, `"else"`)
	if idx < 0 {
		t.Fatal("no else in output")
	}
	// Find what comes after "else": in the JSON
	afterElse := s[idx+len(`"else"`):]
	afterElse = strings.TrimSpace(afterElse)
	if !strings.HasPrefix(afterElse, ": {") && !strings.HasPrefix(afterElse, ":{") {
		// JSON may have `: {\n` or `: {` form
		colonIdx := strings.Index(afterElse, ":")
		if colonIdx >= 0 {
			afterColon := strings.TrimSpace(afterElse[colonIdx+1:])
			if strings.HasPrefix(afterColon, "[") {
				t.Error("first else should be bare object { not array [")
			}
		}
	}
}

// ---------- Audit D: Character name consistency (always lowercase) ----------

func TestAuditD_CharacterNamesAlwaysLowercase(t *testing.T) {
	// Test ALL node types that emit "character" field
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.DialogueNode{Character: "JOSIE", Text: "Hi."},
			&ast.CharShowNode{Char: "mauricio", Look: "neutral_smirk", Position: "right"},
			&ast.CharHideNode{Char: "malia"},
			&ast.CharLookNode{Char: "easton", Look: "hurt"},
			&ast.CharMoveNode{Char: "mark", Position: "left"},
			&ast.CharBubbleNode{Char: "josie", BubbleType: "heart"},
			&ast.AffectionNode{Char: "easton", Delta: "+2"},
			&ast.PhoneShowNode{Body: []ast.Node{
				&ast.TextMessageNode{Direction: "from", Char: "MARK", Content: "Hey"},
			}},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// Check every "character" field in the output
	var checkChars func(v interface{}, path string)
	checkChars = func(v interface{}, path string) {
		switch x := v.(type) {
		case map[string]interface{}:
			if charVal, ok := x["character"]; ok {
				charStr := charVal.(string)
				for _, r := range charStr {
					if r >= 'A' && r <= 'Z' {
						t.Errorf("%s.character = %q contains uppercase", path, charStr)
						break
					}
				}
			}
			for k, val := range x {
				checkChars(val, path+"."+k)
			}
		case []interface{}:
			for i, val := range x {
				checkChars(val, path+"["+string(rune('0'+i))+"]")
			}
		}
	}
	checkChars(result, "root")
}

// Audit D sub-check: DialogueNode, TextMessageNode lowercase, but others
// like CharShowNode, CharHideNode, CharLookNode, CharMoveNode, CharBubbleNode, AffectionNode
// pass through as-is (parser lowercases, but emitter doesn't for these types).
func TestAuditD_EmitterLowercaseScope(t *testing.T) {
	// The emitter only lowercases DialogueNode and TextMessageNode.
	// Other character types (char_show, char_hide, char_look, char_move, bubble, affection)
	// pass the Char field through unchanged.
	// This is correct IF the parser always stores lowercase — let's verify emitter behavior.

	tests := []struct {
		name     string
		node     ast.Node
		wantChar string
	}{
		{"dialogue_upper", &ast.DialogueNode{Character: "JOSIE", Text: "x"}, "josie"},
		{"dialogue_lower", &ast.DialogueNode{Character: "josie", Text: "x"}, "josie"},
		{"char_show_lower", &ast.CharShowNode{Char: "josie", Look: "a", Position: "left"}, "josie"},
		{"char_show_upper", &ast.CharShowNode{Char: "JOSIE", Look: "a", Position: "left"}, "JOSIE"},
		{"char_hide_upper", &ast.CharHideNode{Char: "MALIA"}, "MALIA"},
		{"char_look_upper", &ast.CharLookNode{Char: "EASTON", Look: "a"}, "EASTON"},
		{"char_move_upper", &ast.CharMoveNode{Char: "MARK", Position: "left"}, "MARK"},
		{"bubble_upper", &ast.CharBubbleNode{Char: "JOSIE", BubbleType: "heart"}, "JOSIE"},
		{"affection_upper", &ast.AffectionNode{Char: "EASTON", Delta: "+1"}, "EASTON"},
		{"text_msg_upper", &ast.TextMessageNode{Direction: "from", Char: "MARK", Content: "x"}, "mark"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := New(newMockResolver())
			step := em.emitNode(tt.node)
			if step == nil {
				t.Fatal("emitNode returned nil")
			}
			got := step["character"].(string)
			if got != tt.wantChar {
				t.Errorf("character = %q, want %q", got, tt.wantChar)
			}
		})
	}
}

// ---------- Audit E: Choice/option structure ----------

func TestAuditE_ChoiceOptionStructure(t *testing.T) {
	// Brave option with check and an @if (check.success) / @else body.
	// Safe option with steps.
	// Concurrent groups inside the success branch and inside safe steps.
	musicNode := &ast.MusicPlayNode{Track: "calm_morning"}
	musicNode.SetConcurrent(true)
	josieNode := &ast.CharShowNode{Char: "josie", Look: "cheerful_wave", Position: "right"}
	josieNode.SetConcurrent(true)

	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.ChoiceNode{
				Options: []*ast.OptionNode{
					{
						ID:    "A",
						Mode:  "brave",
						Text:  "Stand your ground.",
						Check: &ast.CheckBlock{Attr: "CHA", DC: 12},
						Body: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CheckCondition{Result: "success"},
								Then: []ast.Node{
									&ast.CharLookNode{Char: "malia", Look: "worried"},
									josieNode,
									&ast.DialogueNode{Character: "MALIA", Text: "You did it."},
								},
								Else: []ast.Node{
									&ast.NarratorNode{Text: "You faltered."},
								},
							},
						},
					},
					{
						ID:   "B",
						Mode: "safe",
						Text: "Walk away.",
						Body: []ast.Node{
							&ast.BgSetNode{Name: "school_classroom"},
							musicNode,
							&ast.NarratorNode{Text: "You left."},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	choice := steps[0].(map[string]interface{})
	if choice["type"] != "choice" {
		t.Fatalf("type = %v, want choice", choice["type"])
	}

	options := choice["options"].([]interface{})
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}

	// Option A (brave)
	optA := options[0].(map[string]interface{})
	if optA["id"] != "A" {
		t.Errorf("optA.id = %v", optA["id"])
	}
	if optA["mode"] != "brave" {
		t.Errorf("optA.mode = %v", optA["mode"])
	}
	if optA["text"] != "Stand your ground." {
		t.Errorf("optA.text = %v", optA["text"])
	}

	// Check block
	check := optA["check"].(map[string]interface{})
	if check["attr"] != "CHA" {
		t.Errorf("check.attr = %v", check["attr"])
	}
	if check["dc"] != float64(12) {
		t.Errorf("check.dc = %v", check["dc"])
	}

	// No more on_success/on_fail keys — the body is emitted as "steps".
	if _, has := optA["on_success"]; has {
		t.Error("brave option should not have 'on_success' key")
	}
	if _, has := optA["on_fail"]; has {
		t.Error("brave option should not have 'on_fail' key")
	}

	// Brave option body is under "steps".
	stepsA := optA["steps"].([]interface{})
	if len(stepsA) != 1 {
		t.Fatalf("len(optA.steps) = %d, want 1 (the @if tree)", len(stepsA))
	}
	ifStep := stepsA[0].(map[string]interface{})
	if ifStep["type"] != "if" {
		t.Errorf("optA.steps[0].type = %v, want 'if'", ifStep["type"])
	}
	cond := ifStep["condition"].(map[string]interface{})
	if cond["type"] != "check" || cond["result"] != "success" {
		t.Errorf("condition: got %+v, want check/success", cond)
	}

	// then branch should contain concurrent group + dialogue
	thenBranch := ifStep["then"].([]interface{})
	if len(thenBranch) != 2 {
		t.Fatalf("then length = %d, want 2", len(thenBranch))
	}
	if _, ok := thenBranch[0].([]interface{}); !ok {
		t.Fatalf("then[0] should be a concurrent-group array, got %T", thenBranch[0])
	}

	// else branch should contain the narrator (emitted as plain array)
	elseBranch, ok := ifStep["else"].([]interface{})
	if !ok {
		t.Fatalf("else should be array, got %T", ifStep["else"])
	}
	if len(elseBranch) != 1 {
		t.Fatalf("else length = %d, want 1", len(elseBranch))
	}

	// Option B (safe)
	optB := options[1].(map[string]interface{})
	if optB["id"] != "B" {
		t.Errorf("optB.id = %v", optB["id"])
	}
	if optB["mode"] != "safe" {
		t.Errorf("optB.mode = %v", optB["mode"])
	}

	// Safe option should have "steps", NOT "check", "on_success", "on_fail"
	if _, hasCheck := optB["check"]; hasCheck {
		t.Error("safe option should not have 'check'")
	}
	if _, hasOnSuccess := optB["on_success"]; hasOnSuccess {
		t.Error("safe option should not have 'on_success'")
	}
	if _, hasOnFail := optB["on_fail"]; hasOnFail {
		t.Error("safe option should not have 'on_fail'")
	}

	// steps should contain concurrent group
	stepsB := optB["steps"].([]interface{})
	if len(stepsB) != 2 {
		t.Fatalf("len(optB.steps) = %d, want 2 (1 concurrent group + 1 narrator)", len(stepsB))
	}
	// First should be concurrent group
	bGroup, ok := stepsB[0].([]interface{})
	if !ok {
		t.Fatalf("optB.steps[0] should be array, got %T", stepsB[0])
	}
	if len(bGroup) != 2 {
		t.Fatalf("optB.steps[0] length = %d, want 2", len(bGroup))
	}
}

// ---------- Audit F: Null/empty handling ----------

func TestAuditF_GateNull(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate:      nil,
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	s := string(data)

	// gate must be explicitly null, not absent
	if !strings.Contains(s, `"gate": null`) {
		t.Errorf("expected 'gate: null' in output, got:\n%s", s)
	}
}

func TestAuditF_CgShowNoBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.CgShowNode{Name: "window_stare"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	s := string(data)

	// CG with no body should NOT have "steps" key
	if strings.Contains(s, `"steps"`) {
		// Only the top-level steps should be present
		// The cg_show itself should not have "steps"
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		steps := result["steps"].([]interface{})
		cg := steps[0].(map[string]interface{})
		if _, hasSteps := cg["steps"]; hasSteps {
			t.Error("cg_show without body should NOT have 'steps' key")
		}
	}
}

func TestAuditF_IfWithoutElse(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.IfNode{
				Condition: &ast.FlagCondition{Name: "test_flag"},
				Then:      []ast.Node{&ast.NarratorNode{Text: "yes"}},
				Else:      nil,
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	ifNode := steps[0].(map[string]interface{})

	// Should NOT have "else" key at all
	if _, hasElse := ifNode["else"]; hasElse {
		t.Error("if without else should NOT have 'else' key")
	}

	// Must have "then"
	then := ifNode["then"].([]interface{})
	if len(then) != 1 {
		t.Errorf("then length = %d, want 1", len(then))
	}
}

func TestAuditF_PhoneShowNoBody(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body:      []ast.Node{&ast.PhoneShowNode{}},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	phone := steps[0].(map[string]interface{})

	// phone_show without body should NOT have "messages" key
	if _, hasMessages := phone["messages"]; hasMessages {
		t.Error("phone_show without body should NOT have 'messages' key")
	}
}

func TestAuditF_EmptyBodyArray(t *testing.T) {
	// If body is empty slice (not nil), what happens?
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body:      []ast.Node{},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// steps should be an empty array [], not null
	steps := result["steps"]
	if steps == nil {
		t.Fatal("steps should be [] not null for empty body")
	}
	arr, ok := steps.([]interface{})
	if !ok {
		t.Fatalf("steps should be array, got %T", steps)
	}
	if len(arr) != 0 {
		t.Errorf("steps should be empty, got %d items", len(arr))
	}
}

// ---------- Audit G: Type field completeness ----------

func TestAuditG_AllNodeTypesHaveTypeField(t *testing.T) {
	// Every emitXxx function must produce a map with a "type" key.
	// Test each one individually.

	em := New(newMockResolver())

	tests := []struct {
		name     string
		node     ast.Node
		wantType string
	}{
		{"bg", &ast.BgSetNode{Name: "bg1"}, "bg"},
		{"char_show", &ast.CharShowNode{Char: "c", Look: "l", Position: "left"}, "char_show"},
		{"char_hide", &ast.CharHideNode{Char: "c"}, "char_hide"},
		{"char_look", &ast.CharLookNode{Char: "c", Look: "l"}, "char_look"},
		{"char_move", &ast.CharMoveNode{Char: "c", Position: "right"}, "char_move"},
		{"bubble", &ast.CharBubbleNode{Char: "c", BubbleType: "heart"}, "bubble"},
		{"cg_show", &ast.CgShowNode{Name: "cg1"}, "cg_show"},
		{"dialogue", &ast.DialogueNode{Character: "C", Text: "hi"}, "dialogue"},
		{"narrator", &ast.NarratorNode{Text: "n"}, "narrator"},
		{"you", &ast.YouNode{Text: "y"}, "you"},
		{"phone_show", &ast.PhoneShowNode{}, "phone_show"},
		{"phone_hide", &ast.PhoneHideNode{}, "phone_hide"},
		{"text_message", &ast.TextMessageNode{Direction: "from", Char: "c", Content: "hi"}, "text_message"},
		{"music_play", &ast.MusicPlayNode{Track: "t"}, "music_play"},
		{"music_crossfade", &ast.MusicCrossfadeNode{Track: "t"}, "music_crossfade"},
		{"music_fadeout", &ast.MusicFadeoutNode{}, "music_fadeout"},
		{"sfx_play", &ast.SfxPlayNode{Sound: "s"}, "sfx_play"},
		{"minigame", &ast.MinigameNode{ID: "g", Attr: "ATK"}, "minigame"},
		{"choice", &ast.ChoiceNode{Options: []*ast.OptionNode{}}, "choice"},
		{"affection", &ast.AffectionNode{Char: "c", Delta: "+1"}, "affection"},
		{"signal", &ast.SignalNode{Kind: "mark", Event: "E"}, "signal"},
		{"butterfly", &ast.ButterflyNode{Description: "d"}, "butterfly"},
		{"if", &ast.IfNode{
			Condition: &ast.FlagCondition{Name: "f"},
			Then:      []ast.Node{&ast.NarratorNode{Text: "x"}},
		}, "if"},
		{"label", &ast.LabelNode{Name: "L"}, "label"},
		{"pause", &ast.PauseNode{Clicks: 1}, "pause"},
		{"goto", &ast.GotoNode{Name: "L"}, "goto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := em.emitNode(tt.node)
			if step == nil {
				t.Fatal("emitNode returned nil")
			}
			typeVal, ok := step["type"]
			if !ok {
				t.Fatalf("missing 'type' field for %s", tt.name)
			}
			if typeVal != tt.wantType {
				t.Errorf("type = %v, want %q", typeVal, tt.wantType)
			}
		})
	}
}

// ---------- Additional structural checks ----------

func TestAuditG_MinigameStructure(t *testing.T) {
	// Minigame body now uses standard @if (rating.X) { ... } branching.
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				ID:          "qte_challenge",
				Attr:        "ATK",
				Description: "minigame description placeholder",
				Body: []ast.Node{
					&ast.IfNode{
						Condition: &ast.RatingCondition{Grade: "S"},
						Then:      []ast.Node{&ast.NarratorNode{Text: "Perfect!"}},
						Else: []ast.Node{
							&ast.IfNode{
								Condition: &ast.CompoundCondition{
									Op:    "||",
									Left:  &ast.RatingCondition{Grade: "A"},
									Right: &ast.RatingCondition{Grade: "B"},
								},
								Then: []ast.Node{&ast.NarratorNode{Text: "Good."}},
							},
						},
					},
				},
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "test:02"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	mg := steps[0].(map[string]interface{})

	if mg["type"] != "minigame" {
		t.Errorf("type = %v", mg["type"])
	}
	if mg["game_id"] != "qte_challenge" {
		t.Errorf("game_id = %v", mg["game_id"])
	}
	if mg["attr"] != "ATK" {
		t.Errorf("attr = %v", mg["attr"])
	}
	if mg["description"] != "minigame description placeholder" {
		t.Errorf("description = %v", mg["description"])
	}

	// on_results is gone; body is now emitted as "steps".
	if _, has := mg["on_results"]; has {
		t.Error("minigame should not have 'on_results' key; use 'steps' with @if (rating.X)")
	}

	mgSteps, ok := mg["steps"].([]interface{})
	if !ok {
		t.Fatalf("minigame steps: got %T, want []interface{}", mg["steps"])
	}
	if len(mgSteps) != 1 {
		t.Fatalf("minigame steps len: got %d, want 1 (one @if)", len(mgSteps))
	}
	ifStep := mgSteps[0].(map[string]interface{})
	if ifStep["type"] != "if" {
		t.Errorf("minigame step[0] type: got %v, want 'if'", ifStep["type"])
	}
	cond := ifStep["condition"].(map[string]interface{})
	if cond["type"] != "rating" || cond["grade"] != "S" {
		t.Errorf("condition: got %+v, want rating/S", cond)
	}
}

func TestAuditG_ConditionFieldCompleteness(t *testing.T) {
	// Verify each condition type has exactly the right fields
	em := New(newMockResolver())

	t.Run("choice", func(t *testing.T) {
		c := em.emitCondition(&ast.ChoiceCondition{Option: "A", Result: "fail"})
		assertField(t, c, "type", "choice")
		assertField(t, c, "option", "A")
		assertField(t, c, "result", "fail")
	})

	t.Run("flag", func(t *testing.T) {
		c := em.emitCondition(&ast.FlagCondition{Name: "EP01"})
		assertField(t, c, "type", "flag")
		assertField(t, c, "name", "EP01")
	})

	t.Run("comparison", func(t *testing.T) {
		c := em.emitCondition(&ast.ComparisonCondition{
			Left:  ast.ComparisonOperand{Kind: ast.OperandValue, Name: "x"},
			Op:    ">=",
			Right: 5,
		})
		assertField(t, c, "type", "comparison")
		assertField(t, c, "op", ">=")
		assertField(t, c, "right", 5)
		left, ok := c["left"].(map[string]interface{})
		if !ok {
			t.Fatalf("comparison left should be map, got %T", c["left"])
		}
		if left["kind"] != ast.OperandValue {
			t.Errorf("left.kind = %v, want %q", left["kind"], ast.OperandValue)
		}
		if left["name"] != "x" {
			t.Errorf("left.name = %v, want %q", left["name"], "x")
		}
	})

	t.Run("influence", func(t *testing.T) {
		c := em.emitCondition(&ast.InfluenceCondition{Description: "desc"})
		assertField(t, c, "type", "influence")
		assertField(t, c, "description", "desc")
	})

	t.Run("compound", func(t *testing.T) {
		c := em.emitCondition(&ast.CompoundCondition{
			Op:    "&&",
			Left:  &ast.FlagCondition{Name: "a"},
			Right: &ast.FlagCondition{Name: "b"},
		})
		assertField(t, c, "type", "compound")
		assertField(t, c, "op", "&&")
		if _, ok := c["left"].(map[string]interface{}); !ok {
			t.Errorf("compound left should be map, got %T", c["left"])
		}
		if _, ok := c["right"].(map[string]interface{}); !ok {
			t.Errorf("compound right should be map, got %T", c["right"])
		}
	})
}

func TestAuditG_TopLevelFields(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main/bad/001:03",
		Title:     "The End",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{{Target: "main:04"}}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["episode_id"] != "main/bad/001:03" {
		t.Errorf("episode_id = %v", result["episode_id"])
	}
	if result["branch_key"] != "main/bad/001" {
		t.Errorf("branch_key = %v", result["branch_key"])
	}
	if result["seq"] != float64(3) {
		t.Errorf("seq = %v", result["seq"])
	}
	if result["title"] != "The End" {
		t.Errorf("title = %v", result["title"])
	}
}

// ---------- Helpers ----------

func assertType(t *testing.T, val interface{}, wantType string, path string) {
	t.Helper()
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("%s: expected object, got %T", path, val)
	}
	if m["type"] != wantType {
		t.Errorf("%s.type = %v, want %q", path, m["type"], wantType)
	}
}

func assertField(t *testing.T, m map[string]interface{}, key string, want interface{}) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	if got != want {
		t.Errorf("%s = %v, want %v", key, got, want)
	}
}
