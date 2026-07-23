package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

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
