package cmd

// Tests for doctor --fix host-wiring convergence: drifted artifacts written
// by an older CLI are updated in place; foreign content survives.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A Cursor MCP config written before the --project ${workspaceFolder} change
// must be converged to the current shape; a foreign server must survive.
func TestConvergeHostWiring_UpdatesDriftedCursorMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".cursor/mcp.json", `{
  "mcpServers": {
    "archcore": {"command": "archcore", "args": ["mcp"]},
    "other": {"command": "other-tool", "args": ["serve"]}
  }
}`)

	failures := convergeHostWiring(base, []string{"cursor"})

	if failures != 0 {
		t.Fatalf("convergeHostWiring failures = %d", failures)
	}
	data, err := os.ReadFile(configPathFor(base, ".cursor/mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "${workspaceFolder}") {
		t.Errorf("drifted cursor entry must be converged to --project ${workspaceFolder}:\n%s", data)
	}
	if !strings.Contains(string(data), "other-tool") {
		t.Errorf("foreign server must survive convergence:\n%s", data)
	}
}

// A current-shape config is left byte-identical (no gratuitous rewrite).
func TestConvergeHostWiring_NoChangeWhenCurrent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	// First converge establishes the current shape (creates all artifacts).
	if failures := convergeHostWiring(base, []string{"cursor"}); failures != 0 {
		t.Fatalf("initial converge failures = %d", failures)
	}
	path := configPathFor(base, ".cursor/mcp.json")
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if failures := convergeHostWiring(base, []string{"cursor"}); failures != 0 {
		t.Fatalf("second converge failures = %d", failures)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("second converge must be a no-op:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// A stale hook command from an older CLI is healed during convergence.
func TestConvergeHostWiring_HealsStaleHookCommand(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks session-start"}]}
    ]
  }
}`)

	failures := convergeHostWiring(base, []string{"claude-code"})

	if failures != 0 {
		t.Fatalf("convergeHostWiring failures = %d", failures)
	}
	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Scoped to the event under repair: archcore now installs three events, so
	// counting across the file would measure that instead of the ownership
	// contract this test is about.
	var doc struct {
		Hooks map[string][]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing config: %v\n%s", err, data)
	}
	owned := 0
	for _, entry := range doc.Hooks["SessionStart"] {
		if strings.Contains(string(entry), "archcore hooks ") {
			owned++
		}
	}
	if owned != 1 {
		t.Errorf("want exactly 1 archcore entry under SessionStart after convergence, got %d:\n%s", owned, data)
	}
	if !strings.Contains(string(data), "archcore hooks claude-code session-start") {
		t.Errorf("stale hook command must be updated:\n%s", data)
	}
}

// Unknown agent id under --fix --agent is a failure, not a silent skip.
func TestConvergeHostWiring_UnknownAgent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if failures := convergeHostWiring(base, []string{"nope"}); failures != 1 {
		t.Errorf("unknown agent must count as a failure, got %d", failures)
	}
	if _, err := os.Stat(filepath.Join(base, ".claude")); !os.IsNotExist(err) {
		t.Error("unknown agent must not create artifacts")
	}
}

// No detected agents and no explicit ids → clean no-op.
func TestConvergeHostWiring_NothingDetectedNoOp(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if failures := convergeHostWiring(base, nil); failures != 0 {
		t.Errorf("no detected agents must be a clean no-op, got %d failures", failures)
	}
}
