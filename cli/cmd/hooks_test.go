package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

func setupArchcoreDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

func TestRunHooksInstallForAgent_AlsoInstallsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runHooksInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	// .mcp.json should exist.
	data, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	if _, ok := raw["mcpServers"]; !ok {
		t.Error("mcpServers missing from .mcp.json after hooks install")
	}
}

// TestRunHooksInstallAutoDetect_InstallsForPickedAgents pins the auto-detect
// path: agents chosen through the picker get hooks and MCP config installed.
// Not parallel: withPickAgents swaps package-level seams.
func TestRunHooksInstallAutoDetect_InstallsForPickedAgents(t *testing.T) {
	base := setupArchcoreDir(t)
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}})

	if err := runHooksInstallAutoDetect(base); err != nil {
		t.Fatalf("runHooksInstallAutoDetect: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".claude", "settings.json")); err != nil {
		t.Error("hooks must be installed for the picked agent")
	}
	if _, err := os.Stat(filepath.Join(base, ".mcp.json")); err != nil {
		t.Error("MCP config must be installed for the picked agent")
	}
}

// TestRunHooksInstallAutoDetect_PickerFailureIsSoft pins the warn-and-continue
// branch: a picker failure prints a hint and exits zero. Not parallel (seams).
func TestRunHooksInstallAutoDetect_PickerFailureIsSoft(t *testing.T) {
	base := setupArchcoreDir(t)
	withPickAgentsFn(t, func() (agentSelection, error) { return agentSelection{}, errors.New("tty unavailable") })

	if err := runHooksInstallAutoDetect(base); err != nil {
		t.Fatalf("picker failure must be soft, got: %v", err)
	}
}

// TestRunHooksInstallForAgent_UnknownAgent pins the unknown-id error path.
func TestRunHooksInstallForAgent_UnknownAgent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	err := runHooksInstallForAgent(base, "no-such-agent")
	if err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("error = %v, want unknown-agent error", err)
	}
}

// TestRunHooksInstallForAgent_HooklessAgentGetsMCP pins that a hookless agent
// (Cline) still proceeds to MCP config install, matching auto-detect.
func TestRunHooksInstallForAgent_HooklessAgentGetsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	if err := runHooksInstallForAgent(base, agents.Cline); err != nil {
		t.Fatalf("hookless agent install must succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Error("no hooks file may be created for a hookless agent")
	}
}
