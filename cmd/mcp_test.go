package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

func TestRunMCPInstall_NoArchcoreDir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	err := runMCPInstallForAgent(base, agents.ClaudeCode)
	if err == nil {
		t.Fatal("expected error without .archcore/")
	}
	if got := err.Error(); got != ".archcore/ not found — run 'archcore init' first" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRunMCPInstall_FreshInstall(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runMCPInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("runMCPInstallForAgent: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}

	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' in mcpServers")
	}

	var entry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Command != "archcore" {
		t.Errorf("command = %q, want %q", entry.Command, "archcore")
	}
	if len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("args = %v, want [mcp]", entry.Args)
	}
}

func TestRunMCPInstall_Idempotent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runMCPInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := runMCPInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("second install: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}

	if len(servers) != 1 {
		t.Errorf("expected 1 server entry, got %d", len(servers))
	}
}

func TestRunMCPInstall_MergesExisting(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	// Pre-populate .mcp.json with an existing server.
	existing := map[string]any{
		"mcpServers": map[string]any{
			"other-tool": map[string]any{
				"command": "other-tool",
				"args":    []string{"serve"},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(base, ".mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runMCPInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("runMCPInstallForAgent: %v", err)
	}

	resultData, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(resultData, &raw); err != nil {
		t.Fatal(err)
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}

	if _, ok := servers["other-tool"]; !ok {
		t.Error("existing 'other-tool' was lost during merge")
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' after install")
	}
}

func TestMCPInstall_AgentFlag_Valid(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runMCPInstallForAgent(base, agents.Cursor); err != nil {
		t.Fatalf("runMCPInstallForAgent: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("missing mcpServers in .cursor/mcp.json")
	}
}

func TestMCPInstall_AgentFlag_Invalid(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	err := runMCPInstallForAgent(base, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid agent")
	}
}

func TestMCPInstall_AutoDetect(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatalf("MkdirAll .cursor: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".roo"), 0o755); err != nil {
		t.Fatalf("MkdirAll .roo: %v", err)
	}

	if err := runMCPInstallAutoDetect(base); err != nil {
		t.Fatalf("runMCPInstallAutoDetect: %v", err)
	}

	// Both .cursor/mcp.json and .roo/mcp.json should exist.
	for _, path := range []string{".cursor/mcp.json", ".roo/mcp.json"} {
		if _, err := os.Stat(filepath.Join(base, path)); err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
		}
	}
}

func TestMCPInstall_NoAgents_NoInstallInNonInteractive(t *testing.T) {
	base := setupArchcoreDir(t)
	origIsInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origIsInteractive }()

	if err := runMCPInstallAutoDetect(base); err != nil {
		t.Fatalf("runMCPInstallAutoDetect: %v", err)
	}

	// No agent detected + non-interactive env → nothing should be installed.
	if _, err := os.Stat(filepath.Join(base, ".mcp.json")); err == nil {
		t.Error("expected no .mcp.json when no agent detected in non-interactive env")
	}
}

func TestMCPInstall_MultipleAgents(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	if err := os.MkdirAll(filepath.Join(base, ".claude"), 0o755); err != nil {
		t.Fatalf("MkdirAll .claude: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".gemini"), 0o755); err != nil {
		t.Fatalf("MkdirAll .gemini: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".codex"), 0o755); err != nil {
		t.Fatalf("MkdirAll .codex: %v", err)
	}

	if err := runMCPInstallAutoDetect(base); err != nil {
		t.Fatalf("runMCPInstallAutoDetect: %v", err)
	}

	// Claude Code .mcp.json.
	if _, err := os.Stat(filepath.Join(base, ".mcp.json")); err != nil {
		t.Error("expected .mcp.json")
	}
	// Gemini CLI .gemini/settings.json.
	if _, err := os.Stat(filepath.Join(base, ".gemini", "settings.json")); err != nil {
		t.Error("expected .gemini/settings.json")
	}
	// Codex CLI .codex/config.toml.
	if _, err := os.Stat(filepath.Join(base, ".codex", "config.toml")); err != nil {
		t.Error("expected .codex/config.toml")
	}
}

// writeMCPSettings writes baseDir/.archcore/settings.json with the given JSON
// body, creating the .archcore directory if needed.
func writeMCPSettings(t *testing.T, baseDir, body string) {
	t.Helper()
	dir := filepath.Join(baseDir, ".archcore")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckGlobals_MissingSourceFails covers the startup fail-fast (spec §6.2):
// a declared global whose directory is absent must abort MCP startup with a
// message naming the source and how to fix it. Without this guard the server
// would serve a silently-incomplete context.
func TestCheckGlobals_MissingSourceFails(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeMCPSettings(t, base,
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)

	err := checkGlobals(base)
	if err == nil {
		t.Fatal("checkGlobals should fail when a declared global is absent")
	}
	if !strings.Contains(err.Error(), "company") {
		t.Errorf("error %q should name the missing global id", err)
	}
	if !strings.Contains(err.Error(), "clone it") {
		t.Errorf("error %q should hint how to fix it (per spec §6.2)", err)
	}
}

// TestCheckGlobals_PresentSourcePasses verifies the happy path: a declared
// global resolved via a "../"-relative path that exists on disk passes startup.
func TestCheckGlobals_PresentSourcePasses(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	base := filepath.Join(parent, "primary")
	// The sibling global the settings point at must exist on disk.
	if err := os.MkdirAll(filepath.Join(parent, "company", ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeMCPSettings(t, base,
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)

	if err := checkGlobals(base); err != nil {
		t.Errorf("checkGlobals should pass when the global exists, got %v", err)
	}
}

// TestCheckGlobals_NoGlobalsPasses verifies that a project declaring no globals
// is never blocked at startup.
func TestCheckGlobals_NoGlobalsPasses(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeMCPSettings(t, base, `{"sync":"none"}`)

	if err := checkGlobals(base); err != nil {
		t.Errorf("checkGlobals with no globals should be nil, got %v", err)
	}
}

// TestCheckGlobals_ToleratesUnknownField is the forward-compatibility guard at
// MCP startup: a settings.json carrying a field this binary does not recognize
// (as a newer archcore would add) must NOT abort startup — it is tolerated.
// (Distinct from TestCheckGlobals_InvalidSettingsFails, where the JSON itself is
// malformed and still fails.)
func TestCheckGlobals_ToleratesUnknownField(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeMCPSettings(t, base, `{"sync":"none","futurefield":1}`)

	if err := checkGlobals(base); err != nil {
		t.Errorf("checkGlobals must tolerate an unknown field, got %v", err)
	}
}

// TestCheckGlobals_InvalidSettingsFails covers Fix 1: a present-but-invalid
// settings.json must abort MCP startup rather than start with globals silently
// dropped from the read path.
func TestCheckGlobals_InvalidSettingsFails(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeMCPSettings(t, base, `{not valid json`)

	err := checkGlobals(base)
	if err == nil {
		t.Fatal("checkGlobals should fail on an invalid settings.json")
	}
	if !strings.Contains(err.Error(), "settings.json") {
		t.Errorf("error %q should mention settings.json", err)
	}
}
