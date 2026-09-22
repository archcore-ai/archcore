package wiring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// cursorHooksConfig is the decode shape tests use to assert on the written
// .cursor/hooks.json. Production code round-trips the file as opaque
// RawMessage (hooks_install.go) and no longer needs a typed config.
type cursorHooksConfig struct {
	Version int                          `json:"version"`
	Hooks   map[string][]cursorHookEntry `json:"hooks"`
}

func TestCursorWriteHooksConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("InstallCursorHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var cfg cursorHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	for _, ev := range []string{"sessionStart"} {
		entries, ok := cfg.Hooks[ev]
		if !ok {
			t.Errorf("missing hook event %s", ev)
			continue
		}
		if len(entries) != 1 {
			t.Errorf("event %s: want 1 entry, got %d", ev, len(entries))
			continue
		}
		if entries[0].Type != "command" {
			t.Errorf("event %s: type = %q, want 'command'", ev, entries[0].Type)
		}
	}
}

func TestCursorWriteHooksConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("first InstallCursorHooks: %v", err)
	}
	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("second InstallCursorHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var cfg cursorHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, ev := range []string{"sessionStart"} {
		if len(cfg.Hooks[ev]) != 1 {
			t.Errorf("event %s: want 1 entry after idempotent install, got %d", ev, len(cfg.Hooks[ev]))
		}
	}
}

func TestCursorWriteHooksConfig_MergesExisting(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	cursorDir := filepath.Join(base, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := cursorHooksConfig{
		Version: 1,
		Hooks: map[string][]cursorHookEntry{
			"sessionStart": {{Command: "echo hello", Type: "command"}},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "hooks.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("InstallCursorHooks: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(cursorDir, "hooks.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var cfg cursorHooksConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// sessionStart should have 2 entries: existing + archcore.
	if len(cfg.Hooks["sessionStart"]) != 2 {
		t.Errorf("sessionStart: want 2 entries, got %d", len(cfg.Hooks["sessionStart"]))
	}
}

func TestCursorWriteHooksConfig_CorruptedJSON(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	cursorDir := filepath.Join(base, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corrupted := []byte("not json")
	hooksPath := filepath.Join(cursorDir, "hooks.json")
	if err := os.WriteFile(hooksPath, corrupted, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// .bak should contain original corrupted content.
	bakData, err := os.ReadFile(hooksPath + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(bakData) != string(corrupted) {
		t.Errorf("bak content = %q, want %q", bakData, corrupted)
	}

	// hooks.json should be valid with hooks installed.
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg cursorHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(cfg.Hooks) == 0 {
		t.Error("hooks map empty after recovery")
	}
}
