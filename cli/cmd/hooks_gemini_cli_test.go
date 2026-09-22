package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

func TestRunHooksInstallForAgent_GeminiCLIAlsoInstallsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runHooksInstallForAgent(base, agents.GeminiCLI); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Should have mcpServers after hooks install.
	var servers map[string]json.RawMessage
	if serversRaw, ok := raw["mcpServers"]; ok {
		if err := json.Unmarshal(serversRaw, &servers); err != nil {
			t.Fatalf("Unmarshal mcpServers: %v", err)
		}
		if _, ok := servers["archcore"]; !ok {
			t.Error("missing archcore in mcpServers")
		}
	} else {
		t.Error("missing mcpServers key after hooks install")
	}
}
