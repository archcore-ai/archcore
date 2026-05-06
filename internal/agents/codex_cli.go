package agents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func codexCLIAgent() *Agent {
	return &Agent{
		ID:          CodexCLI,
		DisplayName: "Codex CLI",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, ".codex", "config.toml")
		},
		WriteMCPConfig: func(baseDir string) error {
			return writeCodexCLIMCPConfig(baseDir)
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".codex"))
		},
	}
}

const codexArchcoreBlock = `
[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
`

func writeCodexCLIMCPConfig(baseDir string) error {
	codexDir := filepath.Join(baseDir, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return fmt.Errorf("creating .codex/ directory: %w", err)
	}

	configPath := filepath.Join(codexDir, "config.toml")

	var content string
	data, err := os.ReadFile(configPath)
	if err == nil {
		content = string(data)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	if strings.Contains(content, "[mcp_servers.archcore]") {
		return nil // already configured
	}

	content = strings.TrimRight(content, "\n") + codexArchcoreBlock

	return os.WriteFile(configPath, []byte(content), 0o644)
}
