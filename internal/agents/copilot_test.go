package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopilotAgent_MCPWiring pins the registry closures for Copilot: a typo in
// the .vscode/mcp.json wiring would otherwise ship green because only the
// shared helper was tested, not the per-agent hookup.
func TestCopilotAgent_MCPWiring(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	a := ByID(Copilot)
	if a == nil {
		t.Fatal("copilot agent not registered")
	}

	wantPath := filepath.Join(base, ".vscode", "mcp.json")
	if got := a.MCPConfigPath(base); got != wantPath {
		t.Errorf("MCPConfigPath = %q, want %q", got, wantPath)
	}
	if err := a.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("expected %s to exist: %v", wantPath, err)
	}
}
