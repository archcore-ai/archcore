package wiring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeminiCLIWriteHooksConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatalf("InstallGeminiCLIHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var hooks map[string][]hookMatcher
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatalf("Unmarshal hooks: %v", err)
	}

	for _, ev := range []string{"SessionStart"} {
		matchers, ok := hooks[ev]
		if !ok {
			t.Errorf("missing hook event %s", ev)
			continue
		}
		if len(matchers) != 1 {
			t.Errorf("event %s: want 1 matcher, got %d", ev, len(matchers))
		}
	}
}

func TestGeminiCLIWriteHooksConfig_MergeWithMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	geminiDir := filepath.Join(base, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Pre-populate with mcpServers.
	existing := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"command": "other"},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(geminiDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatalf("InstallGeminiCLIHooks: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(geminiDir, "settings.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// mcpServers should be preserved.
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("existing mcpServers key was lost")
	}
	// hooks should be added.
	if _, ok := raw["hooks"]; !ok {
		t.Error("hooks key missing")
	}
}

func TestGeminiCLIWriteHooksConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatalf("first InstallGeminiCLIHooks: %v", err)
	}
	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatalf("second InstallGeminiCLIHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var hooks map[string][]hookMatcher
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatalf("Unmarshal hooks: %v", err)
	}

	for _, ev := range []string{"SessionStart"} {
		if len(hooks[ev]) != 1 {
			t.Errorf("event %s: want 1 matcher after idempotent install, got %d", ev, len(hooks[ev]))
		}
	}
}

func TestGeminiCLIWriteHooksConfig_CorruptedJSON(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	geminiDir := filepath.Join(base, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corrupted := []byte("not json")
	settingsPath := filepath.Join(geminiDir, "settings.json")
	if err := os.WriteFile(settingsPath, corrupted, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// .bak should contain original corrupted content.
	bakData, err := os.ReadFile(settingsPath + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(bakData) != string(corrupted) {
		t.Errorf("bak content = %q, want %q", bakData, corrupted)
	}

	// settings.json should be valid with hooks installed.
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["hooks"]; !ok {
		t.Error("'hooks' key missing after recovery")
	}
}
