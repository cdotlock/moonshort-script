package resolver

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "mapping.yaml")
}

func TestLoadMapping(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}
	if r.BaseURL != "https://oss.mobai.com/novel_001" {
		t.Errorf("BaseURL = %q, want %q", r.BaseURL, "https://oss.mobai.com/novel_001")
	}
	if len(r.Bg) == 0 {
		t.Error("Bg map is empty")
	}
	if len(r.Characters) == 0 {
		t.Error("Characters map is empty")
	}
	if len(r.Music) == 0 {
		t.Error("Music map is empty")
	}
	if len(r.Sfx) == 0 {
		t.Error("Sfx map is empty")
	}
	if len(r.Cg) == 0 {
		t.Error("Cg map is empty")
	}
	if len(r.Minigames) == 0 {
		t.Error("Minigames map is empty")
	}
}

func TestResolveBg(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	got, err := r.ResolveBg("school_classroom")
	if err != nil {
		t.Fatalf("ResolveBg(school_classroom) error: %v", err)
	}
	want := "https://oss.mobai.com/novel_001/bg/school_classroom.png"
	if got != want {
		t.Errorf("ResolveBg(school_classroom) = %q, want %q", got, want)
	}
}

func TestResolveCharacter(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	got, err := r.ResolveCharacter("mauricio", "neutral_smirk")
	if err != nil {
		t.Fatalf("ResolveCharacter error: %v", err)
	}
	want := "https://oss.mobai.com/novel_001/characters/mauricio_neutral_smirk.png"
	if got != want {
		t.Errorf("ResolveCharacter = %q, want %q", got, want)
	}
}

func TestResolveMusic(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	got, err := r.ResolveMusic("calm_morning")
	if err != nil {
		t.Fatalf("ResolveMusic error: %v", err)
	}
	want := "https://oss.mobai.com/novel_001/music/calm_morning.mp3"
	if got != want {
		t.Errorf("ResolveMusic = %q, want %q", got, want)
	}
}

func TestResolveMissing(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	if _, err := r.ResolveBg("nonexistent"); err == nil {
		t.Error("ResolveBg(nonexistent) expected error, got nil")
	}
	if _, err := r.ResolveCharacter("nobody", "smile"); err == nil {
		t.Error("ResolveCharacter(nobody, smile) expected error, got nil")
	}
	if _, err := r.ResolveCharacter("mauricio", "nonexistent_pose"); err == nil {
		t.Error("ResolveCharacter(mauricio, nonexistent_pose) expected error, got nil")
	}
	if _, err := r.ResolveMusic("nonexistent"); err == nil {
		t.Error("ResolveMusic(nonexistent) expected error, got nil")
	}
	if _, err := r.ResolveSfx("nonexistent"); err == nil {
		t.Error("ResolveSfx(nonexistent) expected error, got nil")
	}
	if _, err := r.ResolveCg("nonexistent"); err == nil {
		t.Error("ResolveCg(nonexistent) expected error, got nil")
	}
	if _, err := r.ResolveMinigame("nonexistent"); err == nil {
		t.Error("ResolveMinigame(nonexistent) expected error, got nil")
	}
}

func TestResolveMinigame(t *testing.T) {
	r, err := LoadMapping(testdataPath())
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	got, err := r.ResolveMinigame("qte_challenge")
	if err != nil {
		t.Fatalf("ResolveMinigame error: %v", err)
	}
	want := "https://oss.mobai.com/novel_001/minigames/qte_challenge/index.html"
	if got != want {
		t.Errorf("ResolveMinigame = %q, want %q", got, want)
	}
}
