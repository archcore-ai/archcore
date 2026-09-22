package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCline_WriteMCPConfig_NoOp(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(Cline)

	// Should not create any file, just return nil.
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	// No MCP config file should be created.
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if e.Name() == ".clinerules" || e.Name() == "mcp.json" {
			t.Errorf("unexpected file created: %s", e.Name())
		}
	}
}

func TestCline_Detect_True(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, ".clinerules"), 0o755)

	if !ByID(Cline).DetectFn(base) {
		t.Error("expected detection")
	}
}

func TestCline_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(Cline).DetectFn(base) {
		t.Error("expected no detection")
	}
}
