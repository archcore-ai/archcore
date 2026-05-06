package agents

import "path/filepath"

var copilotMCPPath = filepath.Join(".vscode", "mcp.json")

func copilotAgent() *Agent {
	return &Agent{
		ID:          Copilot,
		DisplayName: "GitHub Copilot",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, copilotMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteVSCodeMCPJSON(filepath.Join(baseDir, copilotMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return fileExists(filepath.Join(baseDir, ".github", "copilot-instructions.md"))
		},
	}
}
