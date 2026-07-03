package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

// copilotHooksConfig is the decode shape tests use to assert on the written
// .github/hooks/archcore.json. Production code round-trips the file as opaque
// RawMessage (cmd/hooks_install.go) and no longer needs a typed config.
type copilotHooksConfig struct {
	Version int                           `json:"version"`
	Hooks   map[string][]copilotHookEntry `json:"hooks"`
}

func TestCopilotWriteHooksConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runCopilotHooksInstall(base); err != nil {
		t.Fatalf("runCopilotHooksInstall: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".github", "hooks", "archcore.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var cfg copilotHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	entries, ok := cfg.Hooks["sessionStart"]
	if !ok {
		t.Fatal("missing hook event sessionStart")
	}
	if len(entries) != 1 {
		t.Fatalf("sessionStart: want 1 entry, got %d", len(entries))
	}
	if entries[0].Type != "command" {
		t.Errorf("sessionStart: type = %q, want 'command'", entries[0].Type)
	}
	if entries[0].Bash != "archcore hooks copilot session-start" {
		t.Errorf("sessionStart: bash = %q, want 'archcore hooks copilot session-start'", entries[0].Bash)
	}
}

func TestCopilotWriteHooksConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runCopilotHooksInstall(base); err != nil {
		t.Fatalf("first runCopilotHooksInstall: %v", err)
	}
	if err := runCopilotHooksInstall(base); err != nil {
		t.Fatalf("second runCopilotHooksInstall: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".github", "hooks", "archcore.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var cfg copilotHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(cfg.Hooks["sessionStart"]) != 1 {
		t.Errorf("sessionStart: want 1 entry after idempotent install, got %d", len(cfg.Hooks["sessionStart"]))
	}
}

func TestCopilotWriteHooksConfig_MergesExisting(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	hooksDir := filepath.Join(base, ".github", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := copilotHooksConfig{
		Version: 1,
		Hooks: map[string][]copilotHookEntry{
			"sessionStart": {{Type: "command", Bash: "echo hello"}},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "archcore.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := runCopilotHooksInstall(base); err != nil {
		t.Fatalf("runCopilotHooksInstall: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(hooksDir, "archcore.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var cfg copilotHooksConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// sessionStart should have 2 entries: existing + archcore.
	if len(cfg.Hooks["sessionStart"]) != 2 {
		t.Errorf("sessionStart: want 2 entries, got %d", len(cfg.Hooks["sessionStart"]))
	}
}

func TestCopilotWriteHooksConfig_CorruptedJSON(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	hooksDir := filepath.Join(base, ".github", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corrupted := []byte("not json")
	hooksPath := filepath.Join(hooksDir, "archcore.json")
	if err := os.WriteFile(hooksPath, corrupted, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runCopilotHooksInstall(base); err != nil {
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

	// archcore.json should be valid with hooks installed.
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg copilotHooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(cfg.Hooks) == 0 {
		t.Error("hooks map empty after recovery")
	}
}

func TestRunHooksInstallForAgent_CopilotAlsoInstallsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runHooksInstallForAgent(base, agents.Copilot); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	// .vscode/mcp.json should exist with "servers" key.
	data, err := os.ReadFile(filepath.Join(base, ".vscode", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile .vscode/mcp.json: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["servers"]; !ok {
		t.Error("missing 'servers' in .vscode/mcp.json")
	}
}
