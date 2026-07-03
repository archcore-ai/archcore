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

func TestBuildInstructions_HasWorkflowPromptsBlock(t *testing.T) {
	t.Parallel()
	promptNames := []string{
		"iso_track",
		"sources_track",
		"product_track",
		"standard_track",
		"architecture_track",
	}
	for _, lang := range []string{"", "en", "ru"} {
		result := buildInstructions(lang)
		if !strings.Contains(result, "WORKFLOW PROMPTS") {
			t.Errorf("buildInstructions(%q): missing WORKFLOW PROMPTS block", lang)
		}
		if got := strings.Count(result, "WORKFLOW PROMPTS"); got != 1 {
			t.Errorf("buildInstructions(%q): WORKFLOW PROMPTS appears %d times, want 1", lang, got)
		}
		for _, name := range promptNames {
			if !strings.Contains(result, name) {
				t.Errorf("buildInstructions(%q): missing prompt name %q", lang, name)
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
