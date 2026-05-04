package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cdotlock/moonshort-script/internal/decompiler"
	"github.com/cdotlock/moonshort-script/internal/emitter"
	"github.com/cdotlock/moonshort-script/internal/fixer"
	"github.com/cdotlock/moonshort-script/internal/lexer"
	"github.com/cdotlock/moonshort-script/internal/parser"
	"github.com/cdotlock/moonshort-script/internal/resolver"
	"github.com/cdotlock/moonshort-script/internal/validator"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "compile":
		cmdCompile(os.Args[2:])
	case "decompile":
		cmdDecompile(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	case "fix":
		cmdFix(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "mss - MoonShort Script interpreter")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  mss compile <file.md|dir/> [--assets mapping.json] [-o output.json]")
	fmt.Fprintln(os.Stderr, "  mss decompile <output.json> [-o output-dir]  Rebuild mss.md + assests_mapping.json")
	fmt.Fprintln(os.Stderr, "  mss validate <file.md> [--assets mapping.json]")
	fmt.Fprintln(os.Stderr, "  mss fix <file.md> [-o output.md]     Fix and write (in-place if no -o)")
	fmt.Fprintln(os.Stderr, "  mss fix <file.md> --check            Dry run: report issues, don't write")
	os.Exit(1)
}

// parseFlags extracts --assets and -o values from args, returning the positional arg and flags.
func parseFlags(args []string) (target, assetsPath, outputPath string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--assets":
			if i+1 < len(args) {
				assetsPath = args[i+1]
				i++
			}
		case "-o":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		default:
			if target == "" {
				target = args[i]
			}
		}
	}
	return
}

// noopResolver implements emitter.AssetResolver, returning empty strings for all assets.
type noopResolver struct{}

func (n *noopResolver) ResolveBg(name string) (string, error)              { return "", nil }
func (n *noopResolver) ResolveCharacter(char, look string) (string, error) { return "", nil }
func (n *noopResolver) ResolveMusic(name string) (string, error)           { return "", nil }
func (n *noopResolver) ResolveSfx(name string) (string, error)             { return "", nil }
func (n *noopResolver) ResolveCg(name string) (string, error)              { return "", nil }
func (n *noopResolver) ResolveMinigame(gameID string) (string, error)      { return "", nil }

func loadResolver(assetsPath string) (emitter.AssetResolver, error) {
	if assetsPath == "" {
		return &noopResolver{}, nil
	}
	return resolver.LoadMapping(assetsPath)
}

func cmdCompile(args []string) {
	target, assetsPath, outputPath := parseFlags(args)
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: compile requires a file or directory argument")
		os.Exit(1)
	}

	res, err := loadResolver(assetsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var output []byte

	if info.IsDir() {
		output, err = compileDir(target, res)
	} else {
		output, err = compileFile(target, res)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", outputPath)
	} else {
		fmt.Println(string(output))
	}
}

func cmdDecompile(args []string) {
	target, _, outputDir := parseFlags(args)
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: decompile requires a JSON file argument")
		os.Exit(1)
	}
	if outputDir == "" {
		outputDir = defaultDecompileDir(target)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result, err := decompiler.Decompile(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	for _, ep := range result.Episodes {
		name := ep.Name
		if len(result.Episodes) == 1 {
			name = "mss.md"
		}
		path := filepath.Join(outputDir, name)
		if err := os.WriteFile(path, ep.Source, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	}

	mappingPath := filepath.Join(outputDir, "assests_mapping.json")
	if err := os.WriteFile(mappingPath, result.Mapping, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", mappingPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", mappingPath)

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.Message)
	}
}

func defaultDecompileDir(target string) string {
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "" || base == "." {
		base = "mss"
	}
	return filepath.Join(dir, base+"_decompiled")
}

func compileFile(path string, res emitter.AssetResolver) ([]byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	l := lexer.New(string(src))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Validate -- fail if there are errors; suggest mss fix.
	errs := validator.Validate(ep)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "error: %s: %s\n", path, e.Error())
		}
		return nil, fmt.Errorf("compile %s: validation errors — run `mss fix %s` to attempt auto-repair", path, path)
	}

	em := emitter.New(res)
	data, err := em.Emit(ep)
	if err != nil {
		return nil, fmt.Errorf("emit %s: %w", path, err)
	}

	// Print emitter warnings.
	for _, w := range em.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", path, w.Message)
	}

	return data, nil
}

func compileDir(dir string, res emitter.AssetResolver) ([]byte, error) {
	var results []json.RawMessage

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := compileFile(path, res)
		if err != nil {
			return err
		}
		results = append(results, json.RawMessage(data))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(results, "", "  ")
}

func cmdFix(args []string) {
	target, _, outputPath := parseFlags(args)
	checkOnly := false
	for _, a := range args {
		if a == "--check" {
			checkOnly = true
		}
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: fix requires a file argument")
		os.Exit(1)
	}

	src, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := fixer.Fix(string(src))

	// Print fixes
	for _, f := range result.Fixes {
		fmt.Fprintf(os.Stderr, "fixed: %s\n", f)
	}

	// Print errors
	for _, e := range result.Errors {
		fmt.Fprintf(os.Stderr, "error: %s\n", e)
	}

	if !checkOnly {
		dest := target
		if outputPath != "" {
			dest = outputPath
		}
		if err := os.WriteFile(dest, []byte(result.Fixed), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(1)
		}
		if len(result.Fixes) > 0 || outputPath != "" {
			fmt.Fprintf(os.Stderr, "wrote %s\n", dest)
		}
	}

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

func cmdValidate(args []string) {
	target, assetsPath, _ := parseFlags(args)
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: validate requires a file argument")
		os.Exit(1)
	}

	src, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(src))
	p := parser.New(l)
	ep, err := p.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	errs := validator.Validate(ep)

	// If assets provided, also check resolution.
	if assetsPath != "" {
		res, err := loadResolver(assetsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		em := emitter.New(res)
		_, emitErr := em.Emit(ep)
		if emitErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", emitErr)
			os.Exit(1)
		}
		for _, w := range em.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w.Message)
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s\n", e.Error())
		}
		os.Exit(1)
	}

	fmt.Println("OK")
}
