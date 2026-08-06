package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewServer_HasTools(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestBuildInstructions_DefaultEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"", "en"} {
		result := buildInstructions(lang)
		if result != mcpServerInstructions {
			t.Errorf("buildInstructions(%q): expected base instructions unchanged", lang)
		}
		if strings.Contains(result, "LANGUAGE REQUIREMENT") {
			t.Errorf("buildInstructions(%q): should not contain LANGUAGE REQUIREMENT", lang)
		}
	}
}

func TestBuildInstructions_NonEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"ru", "ja", "de"} {
		result := buildInstructions(lang)
		if !strings.HasPrefix(result, mcpServerInstructions) {
			t.Errorf("buildInstructions(%q): should start with base instructions", lang)
		}
		if !strings.Contains(result, "LANGUAGE REQUIREMENT") {
			t.Errorf("buildInstructions(%q): should contain LANGUAGE REQUIREMENT", lang)
		}
		if !strings.Contains(result, lang) {
			t.Errorf("buildInstructions(%q): should contain the language code", lang)
		}
	}
}

func TestNewServer_WithLanguageSetting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(base, ".archcore", "settings.json"),
		[]byte(`{"sync":"none","language":"ru"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

// TestBuildInstructions_TrackSectionsRemoved guards the layer boundary from
// both sides. Track orchestration moved to the plugin, so the instructions must
// not describe it — but the cut sat next to sections that carry knowledge about
// document TYPES, which stays here. Asserting only the removal would let a
// careless edit take REQUIREMENTS LAYERS with it and no test would notice;
// asserting only the survivors would let the track prose creep back.
func TestBuildInstructions_TrackSectionsRemoved(t *testing.T) {
	t.Parallel()

	// Track orchestration and the prompt surface that drove it.
	removed := []string{
		"REQUIREMENTS TRACKS",
		"RESEARCH GATE",
		"WORKFLOW PROMPTS",
		"iso_track",
		"sources_track",
		"product_track",
		"standard_track",
		"architecture_track",
	}
	// Type knowledge, relation conventions, and status semantics stay.
	kept := []string{
		"TYPE SELECTION RULES",
		"REQUIREMENTS LAYERS",
		"DOCUMENT RELATIONS",
		"VALID STATUS VALUES",
		"TAGS:",
		// The rnd verdict mapping outlived the section it used to live in.
		"first-class outcome",
	}

	for _, lang := range []string{"", "en", "ru"} {
		result := buildInstructions(lang)
		for _, s := range removed {
			if strings.Contains(result, s) {
				t.Errorf("buildInstructions(%q): still contains removed marker %q", lang, s)
			}
		}
		for _, s := range kept {
			if !strings.Contains(result, s) {
				t.Errorf("buildInstructions(%q): lost surviving section %q", lang, s)
			}
		}
	}
}

func TestNewServer_MissingSettings_FallsBack(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// No .archcore/settings.json — server should still create successfully.
	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}
