package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexCLI_WriteMCPConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(CodexCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[mcp_servers.archcore]") {
		t.Error("missing [mcp_servers.archcore] section")
	}
	if !strings.Contains(content, `command = "archcore"`) {
		t.Error("missing command line")
	}
	if !strings.Contains(content, `args = ["mcp"]`) {
		t.Error("missing args line")
	}
}

func TestCodexCLI_WriteMCPConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(CodexCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("first WriteMCPConfig: %v", err)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("second WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	count := strings.Count(content, "[mcp_servers.archcore]")
	if count != 1 {
		t.Errorf("expected 1 archcore block, got %d", count)
	}
}

func TestCodexCLI_WriteMCPConfig_AppendsToTOML(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	codexDir := filepath.Join(base, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := `[model]
name = "gpt-4"

[mcp_servers.other]
command = "other"
`
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(CodexCLI)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `[model]`) {
		t.Error("existing [model] section lost")
	}
	if !strings.Contains(content, `[mcp_servers.other]`) {
		t.Error("existing [mcp_servers.other] section lost")
	}
	if !strings.Contains(content, `[mcp_servers.archcore]`) {
		t.Error("archcore section not added")
	}
}

func TestCodexCLI_WriteMCPConfig_EmptyFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	codexDir := filepath.Join(base, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(CodexCLI)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "[mcp_servers.archcore]") {
		t.Error("archcore section not added to empty file")
	}
}

func TestCodexCLI_Detect_True(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".codex"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if !ByID(CodexCLI).DetectFn(base) {
		t.Error("expected detection")
	}
}

func TestCodexCLI_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(CodexCLI).DetectFn(base) {
		t.Error("expected no detection")
	}
}
