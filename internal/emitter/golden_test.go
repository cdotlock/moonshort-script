package emitter_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cdotlock/moonshort-script/internal/emitter"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/parser"
	"github.com/cdotlock/moonshort-script/internal/resolver"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func TestGoldenEp01(t *testing.T) {
	srcBytes, err := os.ReadFile(testdataPath("ep01.md"))
	if err != nil {
		t.Fatalf("read ep01.md: %v", err)
	}
	expectedBytes, err := os.ReadFile(testdataPath("ep01_output.json"))
	if err != nil {
		t.Fatalf("read ep01_output.json: %v", err)
	}

	l := lexer.New(string(srcBytes))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	res, err := resolver.LoadMapping(testdataPath("mapping.json"))
	if err != nil {
		t.Fatalf("load mapping: %v", err)
	}

	em := emitter.New(res)
	actual, err := em.Emit(ep)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	// Compare as normalized JSON
	var expectedJSON, actualJSON interface{}
	if err := json.Unmarshal(expectedBytes, &expectedJSON); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}
	if err := json.Unmarshal(actual, &actualJSON); err != nil {
		t.Fatalf("unmarshal actual: %v", err)
	}

	expectedNorm, _ := json.Marshal(expectedJSON)
	actualNorm, _ := json.Marshal(actualJSON)

	if string(expectedNorm) != string(actualNorm) {
		t.Error("golden output mismatch -- run `mss compile testdata/ep01.md --assets testdata/mapping.json > testdata/ep01_output.json` to update")
	}
}
