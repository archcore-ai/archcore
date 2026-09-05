package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteStandardMCPJSON_PreservesExistingArchcore pins the actual contract of
// the "already configured" guard: a second write must NOT overwrite an existing
// (possibly user-customized) archcore entry. The idempotency tests only assert
// len(servers)==1, which a guard-less re-marshal of the same key also satisfies —
// so they cannot detect the guard being removed. This can.
func TestWriteStandardMCPJSON_PreservesExistingArchcore(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	const sentinel = "SENTINEL-DO-NOT-OVERWRITE"
	seed := `{"mcpServers":{"archcore":{"command":"` + sentinel + `"}}}`
	if err := os.WriteFile(filePath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), sentinel) {
		t.Errorf("existing archcore entry was overwritten; file = %s", data)
	}
}

func TestWriteStandardMCPJSON_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	data, err := os.ReadFile(filePath)
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

func TestWriteStandardMCPJSON_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, err := os.ReadFile(filePath)
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

	if len(servers) != 1 {
		t.Errorf("expected 1 server entry, got %d", len(servers))
	}
}

func TestWriteStandardMCPJSON_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	existing := map[string]any{
		"mcpServers": map[string]any{
			"other-tool": map[string]any{
				"command": "other-tool",
				"args":    []string{"serve"},
			},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	result, err2 := os.ReadFile(filePath)
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}

	if _, ok := servers["other-tool"]; !ok {
		t.Error("existing 'other-tool' was lost during merge")
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' after install")
	}
}

func TestWriteStandardMCPJSON_CreatesDirs(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".cursor", "mcp.json")

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestWriteStandardMCPJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	corrupted := []byte("not json")
	if err := os.WriteFile(filePath, corrupted, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// .bak should contain original corrupted content.
	bakData, err := os.ReadFile(filePath + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(bakData) != string(corrupted) {
		t.Errorf("bak content = %q, want %q", bakData, corrupted)
	}

	// .mcp.json should now be valid with archcore entry.
	data, err := os.ReadFile(filePath)
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
		t.Error("missing 'archcore' in mcpServers after recovery")
	}
}

// TestWriteStandardMCPJSON_NullServersSection pins the `"mcpServers": null`
// handling (previously a nil-map assignment panic).
func TestWriteStandardMCPJSON_NullServersSection(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	if err := os.WriteFile(filePath, []byte(`{"mcpServers": null}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatalf("null servers section must not fail: %v", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Servers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Servers["archcore"]; !ok {
		t.Error("archcore entry must be added over a null section")
	}
}

// TestWriteStandardMCPJSON_PreservesTopLevelKeyOrder pins order preservation:
// the previous map-based rewrite alphabetized the user's keys.
func TestWriteStandardMCPJSON_PreservesTopLevelKeyOrder(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	input := `{"zeta": 1, "alpha": {"x": true}, "mcpServers": {}}`
	if err := os.WriteFile(filePath, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStandardMCPJSON(filePath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	zi, ai, mi := strings.Index(text, `"zeta"`), strings.Index(text, `"alpha"`), strings.Index(text, `"mcpServers"`)
	if !(zi >= 0 && zi < ai && ai < mi) {
		t.Errorf("top-level key order not preserved: zeta@%d alpha@%d mcpServers@%d\n%s", zi, ai, mi, text)
	}
}

// TestWriteStandardMCPJSON_BackupWriteFailureAborts pins the abort policy from
// backup-invalid-configs.adr: if the .bak cannot be written, the original
// corrupted file must stay untouched and the install must fail.
func TestWriteStandardMCPJSON_BackupWriteFailureAborts(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	original := `{corrupted`
	if err := os.WriteFile(filePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filePath+".bak", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := WriteStandardMCPJSON(filePath); err == nil {
		t.Fatal("expected error when the backup cannot be written")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Error("original corrupted file must stay untouched when backup fails")
	}
}

// --- Copilot MCP path -------------------------------------------------------
//
// Copilot CLI reads exactly one project-level MCP source: the workspace-root
// .mcp.json, keyed "mcpServers". These tests pin that, because the two paths
// this agent must NOT use are both plausible-looking and both wrong:
// .vscode/mcp.json (dropped upstream in v1.0.37, github/copilot-cli#3019) and
// .github/mcp.json (listed in GitHub's config-dir docs but never read —
// github/copilot-cli#1886). Either one silently yields a Copilot session with
// no archcore tools, which is invisible until a user trips it.

func TestCopilotAgent_WritesWorkspaceRootMCPJSON(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := copilotAgent()

	want := filepath.Join(base, ".mcp.json")
	if got := agent.MCPConfigPath(base); got != want {
		t.Errorf("MCPConfigPath = %q, want %q", got, want)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	for _, dead := range []string{
		filepath.Join(base, ".vscode", "mcp.json"),
		filepath.Join(base, ".github", "mcp.json"),
	} {
		if _, err := os.Stat(dead); !os.IsNotExist(err) {
			t.Errorf("%s must not be written — Copilot CLI does not read it", dead)
		}
	}

	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["servers"]; ok {
		t.Error(`"servers" is the VS Code key; Copilot CLI reads "mcpServers"`)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}
	var entry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Command != "archcore" || len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("entry = %+v, want {archcore [mcp]}", entry)
	}
}

// Copilot and claude-code deliberately target the SAME file. Wiring both must
// converge on one archcore entry, in either order — if it ever duplicated or
// fought, a repo wired for both hosts would flip shape on every init.
func TestCopilotAndClaudeCodeShareOneMCPEntry(t *testing.T) {
	t.Parallel()
	for _, order := range [][]*Agent{
		{copilotAgent(), claudeCodeAgent()},
		{claudeCodeAgent(), copilotAgent()},
	} {
		base := t.TempDir()
		if order[0].MCPConfigPath(base) != order[1].MCPConfigPath(base) {
			t.Fatalf("expected a shared MCP path, got %q and %q",
				order[0].MCPConfigPath(base), order[1].MCPConfigPath(base))
		}
		for _, a := range order {
			if err := a.WriteMCPConfig(base); err != nil {
				t.Fatalf("%s WriteMCPConfig: %v", a.ID, err)
			}
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
		if len(servers) != 1 {
			t.Errorf("%s then %s: expected 1 server entry, got %d",
				order[0].ID, order[1].ID, len(servers))
		}
	}
}
