// Package decompiler reconstructs MSS source and asset mappings from compiled
// player JSON.
package decompiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// EpisodeFile is one reconstructed MSS source file.
type EpisodeFile struct {
	Name   string
	Source []byte
}

// Warning records a lossy or suspicious recovery detail.
type Warning struct {
	Message string
}

// Result is the full output of a decompile pass.
type Result struct {
	Episodes []EpisodeFile
	Mapping  []byte
	Warnings []Warning
}

type assetRecord struct {
	kind    string
	segment string
	char    string
	name    string
	url     string
}

type decompiler struct {
	records  []assetRecord
	warnings []Warning
}

type assetMapping struct {
	BaseURL string      `json:"base_url"`
	Assets  assetGroups `json:"assets"`
}

type assetGroups struct {
	Bg         map[string]string            `json:"bg"`
	Characters map[string]map[string]string `json:"characters"`
	Music      map[string]string            `json:"music"`
	Sfx        map[string]string            `json:"sfx"`
	Cg         map[string]string            `json:"cg"`
	Minigames  map[string]string            `json:"minigames"`
}

// Decompile converts compiled MSS JSON into MSS source files and a recovered
// asset mapping. The input may be a single episode object, an array of episode
// objects, or an object with an "episodes" array.
func Decompile(data []byte) (*Result, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var root interface{}
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	episodes, err := extractEpisodes(root)
	if err != nil {
		return nil, err
	}

	d := &decompiler{}
	result := &Result{Episodes: make([]EpisodeFile, 0, len(episodes))}
	usedNames := map[string]int{}
	for i, ep := range episodes {
		source, err := d.writeEpisode(ep)
		if err != nil {
			return nil, fmt.Errorf("episode %d: %w", i+1, err)
		}
		name := uniqueEpisodeFileName(episodeFileName(ep, i), usedNames)
		result.Episodes = append(result.Episodes, EpisodeFile{Name: name, Source: source})
	}

	mapping, err := d.writeMapping()
	if err != nil {
		return nil, err
	}
	result.Mapping = mapping
	result.Warnings = d.warnings
	return result, nil
}

func extractEpisodes(root interface{}) ([]map[string]interface{}, error) {
	switch v := root.(type) {
	case []interface{}:
		return episodeArray(v)
	case map[string]interface{}:
		if _, ok := v["steps"]; ok {
			return []map[string]interface{}{v}, nil
		}
		if raw, ok := v["episodes"].([]interface{}); ok {
			return episodeArray(raw)
		}
	}
	return nil, fmt.Errorf("json does not look like compiled MSS output")
}

func episodeArray(raw []interface{}) ([]map[string]interface{}, error) {
	episodes := make([]map[string]interface{}, 0, len(raw))
	for i, item := range raw {
		ep, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("episode %d is not an object", i+1)
		}
		episodes = append(episodes, ep)
	}
	if len(episodes) == 0 {
		return nil, fmt.Errorf("json contains no episodes")
	}
	return episodes, nil
}

func episodeFileName(ep map[string]interface{}, idx int) string {
	id := stringValue(ep["episode_id"])
	if id == "" {
		id = fmt.Sprintf("episode_%02d", idx+1)
	}
	return sanitizeFileName(id) + ".mss.md"
}

func uniqueEpisodeFileName(name string, used map[string]int) string {
	used[name]++
	if used[name] == 1 {
		return name
	}
	return strings.TrimSuffix(name, ".mss.md") + fmt.Sprintf("_%d.mss.md", used[name])
}

func sanitizeFileName(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.'
		if ok {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_.")
	if out == "" {
		return "episode"
	}
	return out
}

type sourceWriter struct {
	b strings.Builder
}

func (w *sourceWriter) line(indent int, format string, args ...interface{}) {
	for i := 0; i < indent; i++ {
		w.b.WriteString("  ")
	}
	if len(args) > 0 {
		w.b.WriteString(fmt.Sprintf(format, args...))
	} else {
		w.b.WriteString(format)
	}
	w.b.WriteByte('\n')
}

func (w *sourceWriter) blank() {
	w.b.WriteByte('\n')
}

func (d *decompiler) writeEpisode(ep map[string]interface{}) ([]byte, error) {
	id := stringValue(ep["episode_id"])
	if id == "" {
		id = episodeIDFromParts(ep)
	}
	if id == "" {
		return nil, fmt.Errorf("missing episode_id")
	}

	title := stringValue(ep["title"])
	if title == "" {
		title = "Recovered Episode"
	}

	var w sourceWriter
	w.line(0, "@episode %s %s {", id, quote(title))
	w.blank()

	if steps, ok := ep["steps"].([]interface{}); ok {
		if err := d.writeSteps(&w, steps, 1); err != nil {
			return nil, err
		}
	}

	if gate, ok := ep["gate"]; ok && gate != nil {
		w.blank()
		if err := d.writeGate(&w, gate, 1); err != nil {
			return nil, err
		}
	}

	if ending, ok := ep["ending"].(map[string]interface{}); ok && len(ending) > 0 {
		kind := stringValue(ending["type"])
		if kind != "" {
			w.blank()
			w.line(1, "@ending %s", kind)
		}
	}

	w.line(0, "}")
	return []byte(w.b.String()), nil
}

func episodeIDFromParts(ep map[string]interface{}) string {
	branch := stringValue(ep["branch_key"])
	seq, ok := intValue(ep["seq"])
	if branch == "" || !ok || seq <= 0 {
		return ""
	}
	return fmt.Sprintf("%s:%02d", branch, seq)
}

func (d *decompiler) writeSteps(w *sourceWriter, raw []interface{}, indent int) error {
	for _, item := range raw {
		switch v := item.(type) {
		case []interface{}:
			for i, grouped := range v {
				step, ok := grouped.(map[string]interface{})
				if !ok {
					return fmt.Errorf("concurrent group contains a non-object step")
				}
				if err := d.writeStep(w, step, indent, i > 0); err != nil {
					return err
				}
			}
		case map[string]interface{}:
			if err := d.writeStep(w, v, indent, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("steps contains a non-object item")
		}
	}
	return nil
}

func (d *decompiler) writeStep(w *sourceWriter, step map[string]interface{}, indent int, concurrent bool) error {
	typ := stringValue(step["type"])
	prefix := "@"
	if concurrent && canUseConcurrentPrefix(typ) {
		prefix = "&"
	}

	switch typ {
	case "bg":
		name := stringValue(step["name"])
		d.record("bg", "bg", "", name, stringValue(step["url"]))
		w.line(indent, "%sbg set %s%s", prefix, name, suffix(" ", stringValue(step["transition"])))
	case "char_show":
		char := stringValue(step["character"])
		look := stringValue(step["look"])
		d.record("characters", "characters", char, look, stringValue(step["url"]))
		w.line(indent, "%s%s show %s at %s%s", prefix, char, look, stringValue(step["position"]), suffix(" ", stringValue(step["transition"])))
	case "char_hide":
		w.line(indent, "%s%s hide%s", prefix, stringValue(step["character"]), suffix(" ", stringValue(step["transition"])))
	case "char_look":
		char := stringValue(step["character"])
		look := stringValue(step["look"])
		d.record("characters", "characters", char, look, stringValue(step["url"]))
		w.line(indent, "%s%s look %s%s", prefix, char, look, suffix(" ", stringValue(step["transition"])))
	case "char_move":
		w.line(indent, "%s%s move to %s", prefix, stringValue(step["character"]), stringValue(step["position"]))
	case "bubble":
		w.line(indent, "%s%s bubble %s", prefix, stringValue(step["character"]), stringValue(step["bubble_type"]))
	case "cg_show":
		return d.writeCg(w, step, indent, prefix)
	case "dialogue":
		w.line(indent, "%s: %s", strings.ToUpper(stringValue(step["character"])), stringValue(step["text"]))
	case "narrator":
		w.line(indent, "NARRATOR: %s", stringValue(step["text"]))
	case "you":
		w.line(indent, "YOU: %s", stringValue(step["text"]))
	case "phone_show":
		w.line(indent, "%sphone show {", prefix)
		if messages, ok := step["messages"].([]interface{}); ok {
			if err := d.writeSteps(w, messages, indent+1); err != nil {
				return err
			}
		}
		w.line(indent, "}")
	case "phone_hide":
		w.line(indent, "%sphone hide", prefix)
	case "text_message":
		w.line(indent, "%stext %s %s: %s", prefix, stringValue(step["direction"]), strings.ToUpper(stringValue(step["character"])), stringValue(step["text"]))
	case "music_play":
		name := stringValue(step["name"])
		d.record("music", "music", "", name, stringValue(step["url"]))
		w.line(indent, "%smusic play %s", prefix, name)
	case "music_crossfade":
		name := stringValue(step["name"])
		d.record("music", "music", "", name, stringValue(step["url"]))
		w.line(indent, "%smusic crossfade %s", prefix, name)
	case "music_fadeout":
		w.line(indent, "%smusic fadeout", prefix)
	case "sfx_play":
		name := stringValue(step["name"])
		d.record("sfx", "sfx", "", name, stringValue(step["url"]))
		w.line(indent, "%ssfx play %s", prefix, name)
	case "minigame":
		return d.writeMinigame(w, step, indent, prefix)
	case "choice":
		return d.writeChoice(w, step, indent, prefix)
	case "affection":
		delta, _ := intValue(step["delta"])
		w.line(indent, "%saffection %s %s", prefix, stringValue(step["character"]), signed(delta))
	case "signal":
		w.line(indent, "%s", d.signalSource(prefix, step))
	case "butterfly":
		w.line(indent, "%sbutterfly %s", prefix, quote(stringValue(step["description"])))
	case "achievement":
		d.writeAchievement(w, step, indent, prefix)
	case "if":
		return d.writeIf(w, step, indent, prefix)
	case "label":
		w.line(indent, "%slabel %s", prefix, stringValue(step["name"]))
	case "goto":
		w.line(indent, "%sgoto %s", prefix, stringValue(step["target"]))
	case "pause":
		clicks, ok := intValue(step["clicks"])
		if !ok || clicks < 1 {
			clicks = 1
		}
		w.line(indent, "%spause for %d", prefix, clicks)
	default:
		d.warn("skipped unsupported step type %q", typ)
	}
	return nil
}

func canUseConcurrentPrefix(typ string) bool {
	switch typ {
	case "bg", "char_show", "char_hide", "char_look", "char_move", "bubble",
		"phone_hide", "music_play", "music_crossfade", "music_fadeout",
		"sfx_play", "affection", "signal", "butterfly", "achievement",
		"label", "goto", "pause":
		return true
	default:
		return false
	}
}

func (d *decompiler) writeCg(w *sourceWriter, step map[string]interface{}, indent int, prefix string) error {
	name := stringValue(step["name"])
	d.record("cg", "cg", "", name, stringValue(step["url"]))

	duration := stringValue(step["duration"])
	if duration == "" {
		duration = "medium"
		d.warn("cg %q missing duration; used %q", name, duration)
	}
	content := stringValue(step["content"])
	if content == "" {
		content = fmt.Sprintf("Recovered CG content for %s.", name)
		d.warn("cg %q missing content; used a placeholder", name)
	}

	w.line(indent, "%scg show %s%s {", prefix, name, suffix(" ", stringValue(step["transition"])))
	w.line(indent+1, "duration: %s", duration)
	w.line(indent+1, "content: %s", quote(content))
	if steps, ok := step["steps"].([]interface{}); ok && len(steps) > 0 {
		if err := d.writeSteps(w, steps, indent+1); err != nil {
			return err
		}
	}
	w.line(indent, "}")
	return nil
}

func (d *decompiler) writeMinigame(w *sourceWriter, step map[string]interface{}, indent int, prefix string) error {
	id := stringValue(step["game_id"])
	d.record("minigames", "minigames", "", id, stringValue(step["game_url"]))

	attr := stringValue(step["attr"])
	if attr == "" {
		attr = "DEX"
		d.warn("minigame %q missing attr; used %q", id, attr)
	}

	w.line(indent, "%sminigame %s %s %s {", prefix, id, attr, quote(stringValue(step["description"])))
	if steps, ok := step["steps"].([]interface{}); ok {
		if err := d.writeSteps(w, steps, indent+1); err != nil {
			return err
		}
	}
	w.line(indent, "}")
	return nil
}

func (d *decompiler) writeChoice(w *sourceWriter, step map[string]interface{}, indent int, prefix string) error {
	w.line(indent, "%schoice {", prefix)
	options, ok := step["options"].([]interface{})
	if !ok || len(options) == 0 {
		return fmt.Errorf("choice has no options")
	}
	for _, raw := range options {
		opt, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("choice option is not an object")
		}
		mode := stringValue(opt["mode"])
		if mode == "" {
			mode = "safe"
		}
		w.line(indent+1, "@option %s %s %s {", stringValue(opt["id"]), mode, quote(stringValue(opt["text"])))
		if check, ok := opt["check"].(map[string]interface{}); ok {
			dc, _ := intValue(check["dc"])
			w.line(indent+2, "check {")
			w.line(indent+3, "attr: %s", stringValue(check["attr"]))
			w.line(indent+3, "dc: %d", dc)
			w.line(indent+2, "}")
		}
		if steps, ok := opt["steps"].([]interface{}); ok {
			if err := d.writeSteps(w, steps, indent+2); err != nil {
				return err
			}
		}
		w.line(indent+1, "}")
	}
	w.line(indent, "}")
	return nil
}

func (d *decompiler) writeAchievement(w *sourceWriter, step map[string]interface{}, indent int, prefix string) {
	w.line(indent, "%sachievement %s {", prefix, stringValue(step["achievement_id"]))
	w.line(indent+1, "name: %s", quote(stringValue(step["name"])))
	w.line(indent+1, "rarity: %s", stringValue(step["rarity"]))
	w.line(indent+1, "description: %s", quote(stringValue(step["description"])))
	w.line(indent, "}")
}

func (d *decompiler) writeIf(w *sourceWriter, step map[string]interface{}, indent int, prefix string) error {
	cond, err := conditionSource(step["condition"])
	if err != nil {
		return err
	}
	w.line(indent, "%sif (%s) {", prefix, cond)
	if thenSteps, ok := step["then"].([]interface{}); ok {
		if err := d.writeSteps(w, thenSteps, indent+1); err != nil {
			return err
		}
	}
	w.line(indent, "}")

	if elseRaw, ok := step["else"]; ok && elseRaw != nil {
		w.line(indent, "@else {")
		switch elseVal := elseRaw.(type) {
		case []interface{}:
			if err := d.writeSteps(w, elseVal, indent+1); err != nil {
				return err
			}
		case map[string]interface{}:
			if err := d.writeStep(w, elseVal, indent+1, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("if else branch is not a step or step list")
		}
		w.line(indent, "}")
	}
	return nil
}

func (d *decompiler) signalSource(prefix string, step map[string]interface{}) string {
	switch stringValue(step["kind"]) {
	case "mark":
		event := stringValue(step["event"])
		if isIdentifier(event) {
			return fmt.Sprintf("%ssignal mark %s", prefix, event)
		}
		return fmt.Sprintf("%ssignal mark %s", prefix, quote(event))
	case "int":
		name := stringValue(step["name"])
		op := stringValue(step["op"])
		value, _ := intValue(step["value"])
		if op == "=" {
			return fmt.Sprintf("%ssignal int %s = %d", prefix, name, value)
		}
		if op == "-" {
			return fmt.Sprintf("%ssignal int %s -%d", prefix, name, value)
		}
		return fmt.Sprintf("%ssignal int %s +%d", prefix, name, value)
	default:
		return fmt.Sprintf("%ssignal mark %s", prefix, quote("UNKNOWN_SIGNAL"))
	}
}

type gateRoute struct {
	condition string
	target    string
}

func (d *decompiler) writeGate(w *sourceWriter, raw interface{}, indent int) error {
	routes, err := flattenGate(raw)
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("gate has no routes")
	}

	w.line(indent, "@gate {")
	for i, route := range routes {
		switch {
		case route.condition != "" && i == 0:
			w.line(indent+1, "@if (%s):", route.condition)
			w.line(indent+2, "@next %s", route.target)
		case route.condition != "":
			w.line(indent+1, "@else @if (%s):", route.condition)
			w.line(indent+2, "@next %s", route.target)
		case i == 0:
			w.line(indent+1, "@next %s", route.target)
		default:
			w.line(indent+1, "@else:")
			w.line(indent+2, "@next %s", route.target)
		}
	}
	w.line(indent, "}")
	return nil
}

func flattenGate(raw interface{}) ([]gateRoute, error) {
	gate, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("gate is not an object")
	}

	var routes []gateRoute
	for {
		target := stringValue(gate["next"])
		if condRaw, ok := gate["if"]; ok && condRaw != nil {
			cond, err := conditionSource(condRaw)
			if err != nil {
				return nil, err
			}
			routes = append(routes, gateRoute{condition: cond, target: target})
			elseRaw, ok := gate["else"]
			if !ok || elseRaw == nil {
				break
			}
			next, ok := elseRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("gate else branch is not an object")
			}
			gate = next
			continue
		}
		routes = append(routes, gateRoute{target: target})
		break
	}
	return routes, nil
}

func conditionSource(raw interface{}) (string, error) {
	cond, ok := raw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("condition is not an object")
	}

	switch stringValue(cond["type"]) {
	case "choice":
		return fmt.Sprintf("%s.%s", stringValue(cond["option"]), stringValue(cond["result"])), nil
	case "flag":
		return stringValue(cond["name"]), nil
	case "influence":
		return fmt.Sprintf("influence %s", quote(stringValue(cond["description"]))), nil
	case "comparison":
		left, ok := cond["left"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("comparison condition has no left operand")
		}
		right, _ := intValue(cond["right"])
		return fmt.Sprintf("%s %s %d", operandSource(left), stringValue(cond["op"]), right), nil
	case "compound":
		left, err := conditionSource(cond["left"])
		if err != nil {
			return "", err
		}
		right, err := conditionSource(cond["right"])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, stringValue(cond["op"]), right), nil
	case "check":
		return fmt.Sprintf("check.%s", stringValue(cond["result"])), nil
	case "rating":
		return fmt.Sprintf("rating.%s", stringValue(cond["grade"])), nil
	default:
		return "", fmt.Errorf("unsupported condition type %q", stringValue(cond["type"]))
	}
}

func operandSource(left map[string]interface{}) string {
	switch stringValue(left["kind"]) {
	case "affection":
		return "affection." + stringValue(left["char"])
	default:
		return stringValue(left["name"])
	}
}

func (d *decompiler) record(kind, segment, char, name, url string) {
	if name == "" || url == "" {
		return
	}
	d.records = append(d.records, assetRecord{
		kind:    kind,
		segment: segment,
		char:    char,
		name:    name,
		url:     url,
	})
}

func (d *decompiler) writeMapping() ([]byte, error) {
	mapping := assetMapping{
		BaseURL: deriveBaseURL(d.records),
		Assets: assetGroups{
			Bg:         map[string]string{},
			Characters: map[string]map[string]string{},
			Music:      map[string]string{},
			Sfx:        map[string]string{},
			Cg:         map[string]string{},
			Minigames:  map[string]string{},
		},
	}

	for _, rec := range d.records {
		rel := relativeAssetPath(rec.url, mapping.BaseURL, rec.segment)
		switch rec.kind {
		case "bg":
			setString(mapping.Assets.Bg, rec.name, rel, d)
		case "characters":
			if mapping.Assets.Characters[rec.char] == nil {
				mapping.Assets.Characters[rec.char] = map[string]string{}
			}
			setString(mapping.Assets.Characters[rec.char], rec.name, rel, d)
		case "music":
			setString(mapping.Assets.Music, rec.name, rel, d)
		case "sfx":
			setString(mapping.Assets.Sfx, rec.name, rel, d)
		case "cg":
			setString(mapping.Assets.Cg, rec.name, rel, d)
		case "minigames":
			setString(mapping.Assets.Minigames, rec.name, rel, d)
		}
	}

	out, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal mapping: %w", err)
	}
	out = append(out, '\n')
	return out, nil
}

func setString(m map[string]string, key, value string, d *decompiler) {
	if old, ok := m[key]; ok && old != value {
		d.warn("asset %q has conflicting URLs; kept %q and ignored %q", key, old, value)
		return
	}
	m[key] = value
}

func deriveBaseURL(records []assetRecord) string {
	var candidates []string
	for _, rec := range records {
		if rec.url == "" || rec.segment == "" {
			continue
		}
		marker := "/" + rec.segment + "/"
		if idx := strings.Index(rec.url, marker); idx >= 0 {
			candidates = append(candidates, strings.TrimRight(rec.url[:idx], "/"))
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	first := candidates[0]
	allSame := true
	for _, c := range candidates[1:] {
		if c != first {
			allSame = false
			break
		}
	}
	if allSame {
		return first
	}
	return trimToURLDirectory(commonPrefix(candidates))
}

func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) && prefix != "" {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func trimToURLDirectory(prefix string) string {
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return ""
	}
	schemeIdx := strings.Index(prefix, "://")
	searchStart := 0
	if schemeIdx >= 0 {
		searchStart = schemeIdx + len("://")
	}
	idx := strings.LastIndex(prefix[searchStart:], "/")
	if idx < 0 {
		return prefix
	}
	return prefix[:searchStart+idx]
}

func relativeAssetPath(url, baseURL, segment string) string {
	if baseURL != "" && strings.HasPrefix(url, baseURL+"/") {
		return strings.TrimPrefix(url, baseURL+"/")
	}
	marker := "/" + segment + "/"
	if idx := strings.Index(url, marker); idx >= 0 {
		return strings.TrimPrefix(url[idx+1:], "/")
	}
	return strings.TrimLeft(url, "/")
}

func (d *decompiler) warn(format string, args ...interface{}) {
	d.warnings = append(d.warnings, Warning{Message: fmt.Sprintf(format, args...)})
}

func suffix(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
}

func quote(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	// The current MSS lexer does not implement string escapes; a raw double
	// quote would terminate the token. Preserve a valid source file by
	// normalising embedded quotes to apostrophes.
	s = strings.ReplaceAll(s, `"`, `'`)
	return `"` + s + `"`
}

func signed(n int) string {
	if n >= 0 {
		return fmt.Sprintf("+%d", n)
	}
	return strconv.Itoa(n)
}

func stringValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func intValue(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		n, err := strconv.Atoi(t.String())
		if err == nil {
			return n, true
		}
		f, err := strconv.ParseFloat(t.String(), 64)
		if err != nil {
			return 0, false
		}
		return int(f), true
	default:
		return 0, false
	}
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '/' || r == ':') {
			return false
		}
	}
	return true
}
