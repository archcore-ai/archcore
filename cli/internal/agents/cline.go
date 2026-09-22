package agents

import "path/filepath"

func clineAgent() *Agent {
	return &Agent{
		ID:          Cline,
		DisplayName: "Cline",
		MCPConfigPath: func(baseDir string) string {
			return "" // Cline MCP config lives in VS Code globalStorage
		},
		WriteMCPConfig: func(baseDir string) error {
			return nil // no-op — manual install required
		},
		ManualMCPInstallHint: "MCP config is stored in VS Code globalStorage — add manually via Cline MCP settings",
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".clinerules"))
		},
		InstructionsPath:   agentsMDInstructionsPath,
		WriteInstructions:  writeAgentsMDInstructions,
		RemoveInstructions: removeAgentsMDInstructions,
	}
}
