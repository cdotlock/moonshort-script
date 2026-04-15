// Package resolver maps semantic asset names to full OSS URLs via a YAML mapping file.
package resolver

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// mappingFile mirrors the top-level structure of the YAML mapping file.
type mappingFile struct {
	BaseURL string                 `yaml:"base_url"`
	Assets  map[string]interface{} `yaml:"assets"`
}

// Resolver resolves semantic asset names to full URLs using a YAML mapping.
type Resolver struct {
	BaseURL    string
	Bg         map[string]string            // name -> relative path
	Characters map[string]map[string]string // char -> pose -> relative path
	Music      map[string]string
	Sfx        map[string]string
	Cg         map[string]string
	Minigames  map[string]string
}

// LoadMapping reads and parses a YAML mapping file, returning a ready Resolver.
func LoadMapping(path string) (*Resolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("resolver: read mapping file: %w", err)
	}

	var mf mappingFile
	if err := yaml.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("resolver: parse mapping file: %w", err)
	}

	r := &Resolver{
		BaseURL:    mf.BaseURL,
		Bg:         make(map[string]string),
		Characters: make(map[string]map[string]string),
		Music:      make(map[string]string),
		Sfx:        make(map[string]string),
		Cg:         make(map[string]string),
		Minigames:  make(map[string]string),
	}

	// Parse each asset category from the raw map.
	if bg, ok := mf.Assets["bg"]; ok {
		r.Bg = toStringMap(bg)
	}
	if chars, ok := mf.Assets["characters"]; ok {
		r.Characters = toNestedStringMap(chars)
	}
	if music, ok := mf.Assets["music"]; ok {
		r.Music = toStringMap(music)
	}
	if sfx, ok := mf.Assets["sfx"]; ok {
		r.Sfx = toStringMap(sfx)
	}
	if cg, ok := mf.Assets["cg"]; ok {
		r.Cg = toStringMap(cg)
	}
	if mg, ok := mf.Assets["minigames"]; ok {
		r.Minigames = toStringMap(mg)
	}

	return r, nil
}

// ResolveBg returns the full URL for a background asset.
func (r *Resolver) ResolveBg(name string) (string, error) {
	rel, ok := r.Bg[name]
	if !ok {
		return "", fmt.Errorf("resolver: unknown bg asset %q", name)
	}
	return r.BaseURL + "/" + rel, nil
}

// ResolveCharacter returns the full URL for a character pose/expression.
func (r *Resolver) ResolveCharacter(char, poseExpr string) (string, error) {
	poses, ok := r.Characters[char]
	if !ok {
		return "", fmt.Errorf("resolver: unknown character %q", char)
	}
	rel, ok := poses[poseExpr]
	if !ok {
		return "", fmt.Errorf("resolver: unknown pose %q for character %q", poseExpr, char)
	}
	return r.BaseURL + "/" + rel, nil
}

// ResolveMusic returns the full URL for a music track.
func (r *Resolver) ResolveMusic(name string) (string, error) {
	rel, ok := r.Music[name]
	if !ok {
		return "", fmt.Errorf("resolver: unknown music asset %q", name)
	}
	return r.BaseURL + "/" + rel, nil
}

// ResolveSfx returns the full URL for a sound effect.
func (r *Resolver) ResolveSfx(name string) (string, error) {
	rel, ok := r.Sfx[name]
	if !ok {
		return "", fmt.Errorf("resolver: unknown sfx asset %q", name)
	}
	return r.BaseURL + "/" + rel, nil
}

// ResolveCg returns the full URL for a CG illustration.
func (r *Resolver) ResolveCg(name string) (string, error) {
	rel, ok := r.Cg[name]
	if !ok {
		return "", fmt.Errorf("resolver: unknown cg asset %q", name)
	}
	return r.BaseURL + "/" + rel, nil
}

// ResolveMinigame returns the full URL for a minigame entry point.
func (r *Resolver) ResolveMinigame(gameID string) (string, error) {
	rel, ok := r.Minigames[gameID]
	if !ok {
		return "", fmt.Errorf("resolver: unknown minigame %q", gameID)
	}
	return r.BaseURL + "/" + rel, nil
}

// toStringMap converts an interface{} (expected map[string]interface{}) to map[string]string.
func toStringMap(v interface{}) map[string]string {
	m := make(map[string]string)
	raw, ok := v.(map[string]interface{})
	if !ok {
		return m
	}
	for k, val := range raw {
		if s, ok := val.(string); ok {
			m[k] = s
		}
	}
	return m
}

// toNestedStringMap converts a two-level nested map to map[string]map[string]string.
func toNestedStringMap(v interface{}) map[string]map[string]string {
	m := make(map[string]map[string]string)
	raw, ok := v.(map[string]interface{})
	if !ok {
		return m
	}
	for k, val := range raw {
		m[k] = toStringMap(val)
	}
	return m
}
