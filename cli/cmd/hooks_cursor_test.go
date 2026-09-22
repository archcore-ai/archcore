package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

func TestRunHooksInstallForAgent_CursorAlsoInstallsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runHooksInstallForAgent(base, agents.Cursor); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	// .cursor/mcp.json should exist.
	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile .cursor/mcp.json: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("missing mcpServers in .cursor/mcp.json")
	}
}
