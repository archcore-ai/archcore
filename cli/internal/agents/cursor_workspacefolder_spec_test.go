package agents

// TDD spec for the planned cwd-independence hardening of the Cursor MCP
// config: Cursor does not guarantee the working directory it spawns MCP
// servers with, so `archcore mcp` must be told the project root explicitly.
// The .cursor/mcp.json entry must therefore carry
//
//	"args": ["mcp", "--project", "${workspaceFolder}"]
//
// (Cursor substitutes ${workspaceFolder} with the open workspace path; the
// CLI's resolveProjectRoot gives --project top precedence, so the server root
// no longer depends on spawn cwd.)
//
// Implemented: cursorAgent() delegates to WriteCursorMCPJSON, which emits
// args ["mcp", "--project", "${workspaceFolder}"]. Companion invariant
// (covered by existing tests, must keep passing): Claude Code's .mcp.json
// keeps plain ["mcp"] — ${workspaceFolder} is Cursor syntax only
// (mcp_helpers_test.go asserts args == ["mcp"] for the standard writer).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCursor_WriteMCPConfig_PassesWorkspaceFolderProject_Spec(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(Cursor)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	entry, ok := raw.MCPServers["archcore"]
	if !ok {
		t.Fatalf("missing archcore entry in mcpServers:\n%s", data)
	}
	if entry.Command != "archcore" {
		t.Errorf("command = %q, want %q", entry.Command, "archcore")
	}
	want := []string{"mcp", "--project", "${workspaceFolder}"}
	if len(entry.Args) != len(want) {
		t.Fatalf("args = %v, want %v", entry.Args, want)
	}
	for i := range want {
		if entry.Args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q (full: %v)", i, entry.Args[i], want[i], entry.Args)
		}
	}
}

// Converge must MERGE, not replace: a user-customized field on the archcore
// entry (e.g. "env") survives while the fields archcore owns (command/args)
// are updated to the desired shape. Ownership contract of doctor --fix.
func TestConvergeCursorMCPJSON_PreservesUserFieldsOnEntry(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	path := filepath.Join(base, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// A pre---project entry the user extended with env + an absolute command.
	seed := `{
  "mcpServers": {
    "archcore": {
      "command": "archcore",
      "args": ["mcp"],
      "env": {"ARCHCORE_LOG": "debug"}
    },
    "other": {"command": "other-tool"}
  }
}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := ConvergeCursorMCPJSON(path)
	if err != nil {
		t.Fatalf("ConvergeCursorMCPJSON: %v", err)
	}
	if !changed {
		t.Fatal("drifted args must report changed")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	entry := raw.MCPServers["archcore"]
	if got, want := entry.Args, []string{"mcp", "--project", "${workspaceFolder}"}; len(got) != len(want) || got[1] != want[1] {
		t.Errorf("args must converge to %v, got %v", want, got)
	}
	if entry.Env["ARCHCORE_LOG"] != "debug" {
		t.Errorf("user env field must survive converge:\n%s", data)
	}
	if _, ok := raw.MCPServers["other"]; !ok {
		t.Errorf("foreign server must survive converge:\n%s", data)
	}

	// Second converge: already at desired shape, env still present → no-op.
	changed, err = ConvergeCursorMCPJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("second converge must be a no-op")
	}
}
