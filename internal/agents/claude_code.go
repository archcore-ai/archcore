package agents

import "path/filepath"

const claudeCodeMCPPath = ".mcp.json"

func claudeCodeAgent() *Agent {
	return &Agent{
		ID:          ClaudeCode,
		DisplayName: "Claude Code",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, claudeCodeMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteStandardMCPJSON(filepath.Join(baseDir, claudeCodeMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".claude"))
		},
	}
}
