package agents

import (
	"path/filepath"
)

func openCodeAgent() *Agent {
	return &Agent{
		ID:          OpenCode,
		DisplayName: "OpenCode",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, "opencode.json")
		},
		WriteMCPConfig: func(baseDir string) error {
			return writeOpenCodeMCPConfig(baseDir)
		},
		DetectFn: func(baseDir string) bool {
			return fileExists(filepath.Join(baseDir, "opencode.json")) ||
				dirExists(filepath.Join(baseDir, ".opencode"))
		},
		InstructionsPath:   agentsMDInstructionsPath,
		WriteInstructions:  writeAgentsMDInstructions,
		RemoveInstructions: removeAgentsMDInstructions,
	}
}

// openCodeMCPEntry is the format OpenCode uses for MCP servers.
type openCodeMCPEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
}

func writeOpenCodeMCPConfig(baseDir string) error {
	_, err := writeMCPConfig(filepath.Join(baseDir, "opencode.json"), "mcp", openCodeMCPEntry{
		Type:    "local",
		Command: []string{"archcore", "mcp"},
	}, corruptBackupAndReset, mcpKeepExisting)
	return err
}
