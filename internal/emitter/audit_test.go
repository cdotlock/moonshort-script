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
	// MSS-equivalent:
	//   @bg set school_classroom
	//   &music calm_morning
	//   &malia neutral_phone
	//   NARRATOR: Hello.
	//   @malia worried
	//   &josie cheerful_wave
	//   YOU: Thinking.

	musicNode := &ast.MusicSetNode{Name: "calm_morning"}
	musicNode.SetConcurrent(true)
	maliaShow := &ast.CharShowNode{Char: "malia", Look: "neutral_phone"}
	maliaShow.SetConcurrent(true)
	josieShow := &ast.CharShowNode{Char: "josie", Look: "cheerful_wave"}
	josieShow.SetConcurrent(true)

	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.BgSetNode{Name: "school_classroom"},
			musicNode,
			maliaShow,
			&ast.NarratorNode{Text: "Hello."},
			&ast.CharShowNode{Char: "malia", Look: "worried"},
			josieShow,
			&ast.YouNode{Text: "Thinking."},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
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
	//   [0] = array of 3 (bg + music + char_show)
	//   [1] = object narrator
	//   [2] = array of 2 (char_show + char_show)  // pose-swap + concurrent show
	//   [3] = object you
	if len(steps) != 4 {
		t.Fatalf("len(steps) = %d, want 4", len(steps))
	}

	group0, ok := steps[0].([]interface{})
	if !ok {
		t.Fatalf("steps[0] should be array, got %T", steps[0])
	}
	if len(group0) != 3 {
		t.Fatalf("steps[0] length = %d, want 3", len(group0))
	}
	assertType(t, group0[0], "bg", "steps[0][0]")
	assertType(t, group0[1], "music", "steps[0][1]")
	assertType(t, group0[2], "char_show", "steps[0][2]")

	narr, ok := steps[1].(map[string]interface{})
	if !ok {
		t.Fatalf("steps[1] should be object, got %T", steps[1])
	}
	assertType(t, narr, "narrator", "steps[1]")
	if narr["text"] != "Hello." {
		t.Errorf("steps[1].text = %v, want 'Hello.'", narr["text"])
	}

	group2, ok := steps[2].([]interface{})
	if !ok {
		t.Fatalf("steps[2] should be array, got %T", steps[2])
	}
	if len(group2) != 2 {
		t.Fatalf("steps[2] length = %d, want 2", len(group2))
	}
	// Both are char_show — pose swap + adjacent show, same node type.
	assertType(t, group2[0], "char_show", "steps[2][0]")
	assertType(t, group2[1], "char_show", "steps[2][1]")

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
	//   @else @if (affection.easton >= 5): @next route:01
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
					Leaf:      &ast.NextLeaf{Target: "bad:01"},
				},
				{
					Condition: &ast.ComparisonCondition{
						Left:  &ast.ComparisonOperand{Kind: ast.OperandAffection, Char: "easton"},
						Op:    ">=",
						Right: &ast.ComparisonOperand{Kind: ast.OperandLiteral, Value: 5},
					},
					Leaf: &ast.NextLeaf{Target: "route:01"},
				},
				{
					Leaf: &ast.NextLeaf{Target: "main:02"},
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

	// Level 2: gate.else.if.type == "comparison"
	gateElse := gate["else"].(map[string]interface{})
	elseIf := gateElse["if"].(map[string]interface{})
	if elseIf["type"] != "comparison" {
		t.Errorf("gate.else.if.type = %v, want 'comparison'", elseIf["type"])
	}
	left := elseIf["left"].(map[string]interface{})
	if left["kind"] != "affection" || left["char"] != "easton" {
		t.Errorf("gate.else.if.left = %#v, want affection/easton", left)
	}
	right := elseIf["right"].(map[string]interface{})
	if right["kind"] != "literal" || right["value"].(float64) != 5 {
		t.Errorf("gate.else.if.right = %#v, want literal 5", right)
	}
	if gateElse["next"] != "route:01" {
		t.Errorf("gate.else.next = %v, want 'route:01'", gateElse["next"])
	}

	// Level 3: gate.else.else.next == "main:02"
	fallback := gateElse["else"].(map[string]interface{})
	if fallback["next"] != "main:02" {
		t.Errorf("gate.else.else.next = %v, want 'main:02'", fallback["next"])
	}
	if _, hasIf := fallback["if"]; hasIf {
		t.Error("gate.else.else should NOT have 'if' key (fallback)")
	}
}

// TestAuditB_GateEndLeaves verifies @end leaves render `end: <type>` in
// the gate JSON — both single conditional, single mixed, and within a
// nested if/else chain alongside @next leaves.
func TestAuditB_GateEndLeaves(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01", Title: "T",
		Body: []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate: &ast.GateBlock{
			Routes: []*ast.GateRoute{
				{
					Condition: &ast.FlagCondition{Name: "WIN"},
					Leaf:      &ast.EndLeaf{Type: ast.EndingComplete},
				},
				{
					Condition: &ast.FlagCondition{Name: "LOSE"},
					Leaf:      &ast.EndLeaf{Type: ast.EndingBad},
				},
				{Leaf: &ast.NextLeaf{Target: "main:02"}},
			},
		},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	gate := result["gate"].(map[string]interface{})
	if gate["end"] != "complete" {
		t.Errorf("gate.end = %v, want complete", gate["end"])
	}
	if _, has := gate["next"]; has {
		t.Errorf("gate root should not have next when leaf is end")
	}

	mid := gate["else"].(map[string]interface{})
	if mid["end"] != "bad_ending" {
		t.Errorf("gate.else.end = %v, want bad_ending", mid["end"])
	}

	tail := mid["else"].(map[string]interface{})
	if tail["next"] != "main:02" {
		t.Errorf("gate.else.else.next = %v, want main:02", tail["next"])
	}
	if _, has := tail["end"]; has {
		t.Errorf("fallback should not have end when leaf is next")
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
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
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
	idx := strings.Index(s, `"else"`)
	if idx < 0 {
		t.Fatal("no else in output")
	}
	afterElse := s[idx+len(`"else"`):]
	afterElse = strings.TrimSpace(afterElse)
	if !strings.HasPrefix(afterElse, ": {") && !strings.HasPrefix(afterElse, ":{") {
		colonIdx := strings.Index(afterElse, ":")
		if colonIdx >= 0 {
			afterColon := strings.TrimSpace(afterElse[colonIdx+1:])
			if strings.HasPrefix(afterColon, "[") {
				t.Error("first else should be bare object { not array [")
			}
		}
	}
}

// ---------- Audit D: Character name consistency (lowercase emitter scope) ----------

func TestAuditD_CharacterNamesAlwaysLowercase(t *testing.T) {
	// Test node types that emit "character" field through the emitter.
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.DialogueNode{Character: "JOSIE", Text: "Hi."},
			&ast.CharShowNode{Char: "mauricio", Look: "neutral_smirk"},
			&ast.CharBubbleNode{Char: "josie", BubbleType: "heart"},
			&ast.AffectionNode{Char: "easton", Delta: "+2"},
			&ast.PhoneShowNode{Body: []ast.Node{
				&ast.TextMessageNode{Direction: "from", Char: "MARK", Content: "Hey"},
			}},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// Walk and check every "character" field. Only DialogueNode and
	// TextMessageNode pass through lowercase normalization in the
	// emitter; others rely on parser-side normalization. The test
	// authors them lowercase to keep the invariant clear.
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

// TestAuditD_EmitterLowercaseScope clarifies which emit functions lowercase
// the character name vs. passing it through. The emitter only lowercases
// DialogueNode and TextMessageNode; CharShowNode / CharBubbleNode /
// AffectionNode pass through (the parser normalises them).
func TestAuditD_EmitterLowercaseScope(t *testing.T) {
	tests := []struct {
		name     string
		node     ast.Node
		wantChar string
	}{
		{"dialogue_upper", &ast.DialogueNode{Character: "JOSIE", Text: "x"}, "josie"},
		{"dialogue_lower", &ast.DialogueNode{Character: "josie", Text: "x"}, "josie"},
		{"char_show_lower", &ast.CharShowNode{Char: "josie", Look: "a"}, "josie"},
		{"char_show_upper", &ast.CharShowNode{Char: "JOSIE", Look: "a"}, "JOSIE"},
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
	musicNode := &ast.MusicSetNode{Name: "calm_morning"}
	musicNode.SetConcurrent(true)
	josieNode := &ast.CharShowNode{Char: "josie", Look: "cheerful_wave"}
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
									&ast.CharShowNode{Char: "malia", Look: "worried"},
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
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
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

	check := optA["check"].(map[string]interface{})
	if check["attr"] != "CHA" {
		t.Errorf("check.attr = %v", check["attr"])
	}
	if check["dc"] != float64(12) {
		t.Errorf("check.dc = %v", check["dc"])
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
	if _, hasCheck := optB["check"]; hasCheck {
		t.Error("safe option should not have 'check'")
	}

	stepsB := optB["steps"].([]interface{})
	if len(stepsB) != 2 {
		t.Fatalf("len(optB.steps) = %d, want 2", len(stepsB))
	}
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

	if !strings.Contains(s, `"gate": null`) {
		t.Errorf("expected 'gate: null' in output, got:\n%s", s)
	}
}

// TestAuditF_CgShowLeafShape verifies cg_show is a leaf — no `steps`,
// `duration`, `transition`, or `body` keys.
func TestAuditF_CgShowLeafShape(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.CgShowNode{Name: "window_stare", Content: "story prose"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})
	cg := steps[0].(map[string]interface{})

	for _, key := range []string{"steps", "duration", "transition", "body"} {
		if _, has := cg[key]; has {
			t.Errorf("cg_show should not carry %q (leaf step)", key)
		}
	}
	if cg["content"] != "story prose" {
		t.Errorf("cg_show.content = %v, want %q", cg["content"], "story prose")
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
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	ifNode := steps[0].(map[string]interface{})

	if _, hasElse := ifNode["else"]; hasElse {
		t.Error("if without else should NOT have 'else' key")
	}

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
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	phone := steps[0].(map[string]interface{})

	if _, hasMessages := phone["messages"]; hasMessages {
		t.Error("phone_show without body should NOT have 'messages' key")
	}
}

func TestAuditF_EmptyBodyArray(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body:      []ast.Node{},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

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

// TestAuditG_AllNodeTypesHaveTypeField verifies that every emit* function
// produces a map with the expected "type" key — covering exactly the
// current AST node set.
func TestAuditG_AllNodeTypesHaveTypeField(t *testing.T) {
	em := New(newMockResolver())

	tests := []struct {
		name     string
		node     ast.Node
		wantType string
	}{
		{"bg", &ast.BgSetNode{Name: "bg1"}, "bg"},
		{"char_show", &ast.CharShowNode{Char: "c", Look: "l"}, "char_show"},
		{"bubble", &ast.CharBubbleNode{Char: "c", BubbleType: "heart"}, "bubble"},
		{"cg_show", &ast.CgShowNode{Name: "cg1", Content: "x"}, "cg_show"},
		{"dialogue", &ast.DialogueNode{Character: "C", Text: "hi"}, "dialogue"},
		{"narrator", &ast.NarratorNode{Text: "n"}, "narrator"},
		{"you", &ast.YouNode{Text: "y"}, "you"},
		{"phone_show", &ast.PhoneShowNode{}, "phone_show"},
		{"text_message", &ast.TextMessageNode{Direction: "from", Char: "c", Content: "hi"}, "text_message"},
		{"music", &ast.MusicSetNode{Name: "t"}, "music"},
		{"music_stop", &ast.MusicStopNode{}, "music_stop"},
		{"sfx", &ast.SfxNode{Name: "s"}, "sfx"},
		{"minigame", &ast.MinigameNode{Name: "g", Description: "d"}, "minigame"},
		{"trick", &ast.TrickNode{Type: ast.TrickTap, Prompt: "tap."}, "trick"},
		{"choice", &ast.ChoiceNode{Options: []*ast.OptionNode{}}, "choice"},
		{"affection", &ast.AffectionNode{Char: "c", Delta: "+1"}, "affection"},
		{"signal_mark", &ast.SignalNode{Kind: ast.SignalKindMark, Event: "E"}, "signal"},
		{"signal_int", &ast.SignalNode{Kind: ast.SignalKindInt, Name: "x", Op: ast.SignalOpAdd, Value: 1}, "signal"},
		{"butterfly", &ast.ButterflyNode{Description: "d"}, "butterfly"},
		{"achievement", &ast.AchievementNode{ID: "X", Name: "n", Rarity: ast.RarityRare, Description: "d"}, "achievement"},
		{"if", &ast.IfNode{
			Condition: &ast.FlagCondition{Name: "f"},
			Then:      []ast.Node{&ast.NarratorNode{Text: "x"}},
		}, "if"},
		{"pause", &ast.PauseNode{}, "pause"},
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

// ---------- Audit G+: structural shapes ----------

func TestAuditG_MinigameStructure(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.MinigameNode{
				Name:        "qte_challenge",
				Description: "minigame description placeholder",
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
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
	if mg["name"] != "qte_challenge" {
		t.Errorf("name = %v", mg["name"])
	}
	if mg["description"] != "minigame description placeholder" {
		t.Errorf("description = %v", mg["description"])
	}
	if mg["game_url"] != "https://cdn.test/minigames/qte_challenge/index.html" {
		t.Errorf("game_url = %v", mg["game_url"])
	}

	for _, key := range []string{"attr", "game_id", "on_results", "steps"} {
		if _, has := mg[key]; has {
			t.Errorf("minigame must not carry legacy %q key", key)
		}
	}
}

func TestAuditG_TrickStructure(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01",
		Title:     "T",
		Body: []ast.Node{
			&ast.TrickNode{
				Type:   ast.TrickHold,
				Prompt: "Hold your breath.",
			},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}

	em := New(newMockResolver())
	data, _ := em.Emit(ep)

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	steps := result["steps"].([]interface{})
	trick := steps[0].(map[string]interface{})

	if trick["type"] != "trick" {
		t.Errorf("type = %v", trick["type"])
	}
	if trick["trick_type"] != "hold" {
		t.Errorf("trick_type = %v, want hold", trick["trick_type"])
	}
	if trick["prompt"] != "Hold your breath." {
		t.Errorf("prompt = %v", trick["prompt"])
	}
	for _, key := range []string{"url", "game_url", "steps", "attr"} {
		if _, has := trick[key]; has {
			t.Errorf("trick must not carry %q key", key)
		}
	}
}

// TestAuditG_MusicAndSfxStructure verifies the renamed step types and
// fields: music uses "name" (not "track"), sfx uses "name" (not "sound"),
// and music_stop is a separate leaf type.
func TestAuditG_MusicAndSfxStructure(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01", Title: "T",
		Body: []ast.Node{
			&ast.MusicSetNode{Name: "calm_morning"},
			&ast.MusicStopNode{},
			&ast.SfxNode{Name: "phone_buzz"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	steps := result["steps"].([]interface{})

	music := steps[0].(map[string]interface{})
	if music["type"] != "music" {
		t.Errorf("music type = %v, want music", music["type"])
	}
	if music["name"] != "calm_morning" {
		t.Errorf("music.name = %v, want calm_morning", music["name"])
	}
	if _, has := music["track"]; has {
		t.Error("music must not carry legacy 'track' key")
	}

	stop := steps[1].(map[string]interface{})
	if stop["type"] != "music_stop" {
		t.Errorf("music_stop type = %v, want music_stop", stop["type"])
	}

	sfx := steps[2].(map[string]interface{})
	if sfx["type"] != "sfx" {
		t.Errorf("sfx type = %v, want sfx", sfx["type"])
	}
	if sfx["name"] != "phone_buzz" {
		t.Errorf("sfx.name = %v, want phone_buzz", sfx["name"])
	}
	if _, has := sfx["sound"]; has {
		t.Error("sfx must not carry legacy 'sound' key")
	}
}

// TestAuditG_BubbleStructure verifies the renamed bubble step type
// (formerly char_bubble): "type": "bubble" with "character" and
// "bubble_type" fields.
func TestAuditG_BubbleStructure(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "test:01", Title: "T",
		Body: []ast.Node{
			&ast.CharBubbleNode{Char: "josie", BubbleType: "heart"},
		},
		Gate: &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("test:02")}},
	}
	em := New(newMockResolver())
	data, _ := em.Emit(ep)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	bubble := result["steps"].([]interface{})[0].(map[string]interface{})

	if bubble["type"] != "bubble" {
		t.Errorf("type = %v, want bubble", bubble["type"])
	}
	if bubble["character"] != "josie" {
		t.Errorf("character = %v", bubble["character"])
	}
	if bubble["bubble_type"] != "heart" {
		t.Errorf("bubble_type = %v", bubble["bubble_type"])
	}
}

func TestAuditG_ConditionFieldCompleteness(t *testing.T) {
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
			Left:  &ast.ComparisonOperand{Kind: ast.OperandValue, Name: "x"},
			Op:    ">=",
			Right: &ast.ComparisonOperand{Kind: ast.OperandLiteral, Value: 5},
		})
		assertField(t, c, "type", "comparison")
		assertField(t, c, "op", ">=")
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
		// Right must now also be a map (operand AST), not an int.
		right, ok := c["right"].(map[string]interface{})
		if !ok {
			t.Fatalf("comparison right should be map, got %T", c["right"])
		}
		if right["kind"] != ast.OperandLiteral {
			t.Errorf("right.kind = %v, want %q", right["kind"], ast.OperandLiteral)
		}
		if right["value"] != 5 {
			t.Errorf("right.value = %v, want 5", right["value"])
		}
	})

	t.Run("check", func(t *testing.T) {
		c := em.emitCondition(&ast.CheckCondition{Result: "success"})
		assertField(t, c, "type", "check")
		assertField(t, c, "result", "success")
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

// TestAuditG_OperandShapes verifies the five operand kinds each render
// with the correct discriminator field set.
func TestAuditG_OperandShapes(t *testing.T) {
	em := New(newMockResolver())

	t.Run("literal", func(t *testing.T) {
		m := em.emitOperand(&ast.ComparisonOperand{Kind: ast.OperandLiteral, Value: -3})
		if m["kind"] != "literal" {
			t.Errorf("kind = %v, want literal", m["kind"])
		}
		if m["value"] != -3 {
			t.Errorf("value = %v, want -3", m["value"])
		}
	})

	t.Run("affection", func(t *testing.T) {
		m := em.emitOperand(&ast.ComparisonOperand{Kind: ast.OperandAffection, Char: "easton"})
		if m["kind"] != "affection" {
			t.Errorf("kind = %v, want affection", m["kind"])
		}
		if m["char"] != "easton" {
			t.Errorf("char = %v, want easton", m["char"])
		}
	})

	t.Run("value", func(t *testing.T) {
		m := em.emitOperand(&ast.ComparisonOperand{Kind: ast.OperandValue, Name: "rejections"})
		if m["kind"] != "value" {
			t.Errorf("kind = %v, want value", m["kind"])
		}
		if m["name"] != "rejections" {
			t.Errorf("name = %v, want rejections", m["name"])
		}
	})

	t.Run("max", func(t *testing.T) {
		m := em.emitOperand(&ast.ComparisonOperand{
			Kind: ast.OperandMax,
			Args: []*ast.ComparisonOperand{
				{Kind: ast.OperandAffection, Char: "easton"},
				{Kind: ast.OperandAffection, Char: "diego"},
			},
		})
		if m["kind"] != "max" {
			t.Errorf("kind = %v, want max", m["kind"])
		}
		args := m["args"].([]interface{})
		if len(args) != 2 {
			t.Fatalf("args length = %d, want 2", len(args))
		}
	})

	t.Run("min", func(t *testing.T) {
		m := em.emitOperand(&ast.ComparisonOperand{
			Kind: ast.OperandMin,
			Args: []*ast.ComparisonOperand{
				{Kind: ast.OperandValue, Name: "rejections"},
				{Kind: ast.OperandLiteral, Value: 10},
			},
		})
		if m["kind"] != "min" {
			t.Errorf("kind = %v, want min", m["kind"])
		}
		args := m["args"].([]interface{})
		if len(args) != 2 {
			t.Fatalf("args length = %d, want 2", len(args))
		}
	})
}

func TestAuditG_TopLevelFields(t *testing.T) {
	ep := &ast.Episode{
		BranchKey: "main/bad/001:03",
		Title:     "The End",
		Body:      []ast.Node{&ast.NarratorNode{Text: "Hi."}},
		Gate:      &ast.GateBlock{Routes: []*ast.GateRoute{nextLeaf("main:04")}},
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
