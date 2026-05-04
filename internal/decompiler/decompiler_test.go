package decompiler

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/emitter"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/parser"
	"github.com/cdotlock/moonshort-script/internal/resolver"
	"github.com/cdotlock/moonshort-script/internal/validator"
)

func testdataPath(parts ...string) string {
	all := append([]string{"..", "..", "testdata"}, parts...)
	return filepath.Join(all...)
}

func TestDecompileRoundTripSingleEpisode(t *testing.T) {
	original, err := os.ReadFile(testdataPath("ep01_output.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Decompile(original)
	if err != nil {
		t.Fatalf("decompile: %v", err)
	}
	if len(result.Episodes) != 1 {
		t.Fatalf("episodes: got %d, want 1", len(result.Episodes))
	}
	if !bytes.Contains(result.Episodes[0].Source, []byte("@episode main:01 \"Butterfly\"")) {
		t.Fatalf("decompiled source missing episode header:\n%s", result.Episodes[0].Source)
	}
	if !bytes.Contains(result.Mapping, []byte(`"base_url": "https://oss.mobai.com/novel_001"`)) {
		t.Fatalf("mapping did not recover base_url:\n%s", result.Mapping)
	}

	tmp := t.TempDir()
	mappingPath := filepath.Join(tmp, "assests_mapping.json")
	if err := os.WriteFile(mappingPath, result.Mapping, 0644); err != nil {
		t.Fatalf("write mapping: %v", err)
	}

	res, err := resolver.LoadMapping(mappingPath)
	if err != nil {
		t.Fatalf("load mapping: %v", err)
	}
	compiled, err := compileSource(result.Episodes[0].Source, res)
	if err != nil {
		t.Fatalf("compile decompiled source: %v\n%s", err, result.Episodes[0].Source)
	}

	assertJSONEqual(t, original, compiled)
}

func TestDecompileArrayNamesEpisodes(t *testing.T) {
	ep1, err := os.ReadFile(testdataPath("feature_parade", "ep01_output.json"))
	if err != nil {
		t.Fatalf("read ep1: %v", err)
	}
	ep2, err := os.ReadFile(testdataPath("feature_parade", "ep02_output.json"))
	if err != nil {
		t.Fatalf("read ep2: %v", err)
	}
	input := append([]byte("["), ep1...)
	input = append(input, ',')
	input = append(input, ep2...)
	input = append(input, ']')

	result, err := Decompile(input)
	if err != nil {
		t.Fatalf("decompile: %v", err)
	}
	if got, want := len(result.Episodes), 2; got != want {
		t.Fatalf("episodes: got %d, want %d", got, want)
	}
	if result.Episodes[0].Name == result.Episodes[1].Name {
		t.Fatalf("episode names should be unique, got %q", result.Episodes[0].Name)
	}
	if !bytes.Contains(result.Mapping, []byte(`"characters"`)) {
		t.Fatalf("mapping missing character assets:\n%s", result.Mapping)
	}
}

func compileSource(source []byte, res emitter.AssetResolver) ([]byte, error) {
	l := lexer.New(string(source))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		return nil, err
	}
	if errs := validator.Validate(ep); len(errs) > 0 {
		return nil, errs[0]
	}
	return emitter.New(res).Emit(ep)
}

func assertJSONEqual(t *testing.T, expected, actual []byte) {
	t.Helper()
	var expectedJSON interface{}
	if err := json.Unmarshal(expected, &expectedJSON); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}
	var actualJSON interface{}
	if err := json.Unmarshal(actual, &actualJSON); err != nil {
		t.Fatalf("unmarshal actual: %v", err)
	}
	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		expectedNorm, _ := json.MarshalIndent(expectedJSON, "", "  ")
		actualNorm, _ := json.MarshalIndent(actualJSON, "", "  ")
		t.Fatalf("json mismatch\nexpected:\n%s\nactual:\n%s", expectedNorm, actualNorm)
	}
}
