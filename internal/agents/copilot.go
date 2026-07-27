package agents

import "path/filepath"

// copilotMCPPath is the workspace-root .mcp.json — the ONLY project-level MCP
// source Copilot CLI reads. It is deliberately the same file claude-code
// writes: both hosts key on "mcpServers" and accept a bare {command, args}
// stdio entry, so one file serves both and writeMCPConfig merges idempotently.
//
// Not .vscode/mcp.json (what this agent wrote before): Copilot CLI dropped
// that source in v1.0.37 (github/copilot-cli#3019), so it is dead config for
// the CLI and belongs to VS Code alone — a surface copilot-adapter-design.adr
// explicitly scopes out. Not .github/mcp.json either: the config-dir docs list
// it, but it has never been read as a workspace source
// (github/copilot-cli#1886, still open; confirmed by the maintainer closing
// #1291). Copilot CLI discovers .mcp.json from the working directory up to the
// git root, so a repo-root file covers monorepo layouts too.
const copilotMCPPath = ".mcp.json"

func copilotAgent() *Agent {
	return &Agent{
		ID:          Copilot,
		DisplayName: "GitHub Copilot",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, copilotMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteStandardMCPJSON(filepath.Join(baseDir, copilotMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return fileExists(filepath.Join(baseDir, ".github", "copilot-instructions.md"))
		},
		InstructionsPath:   agentsMDInstructionsPath,
		WriteInstructions:  writeAgentsMDInstructions,
		RemoveInstructions: removeAgentsMDInstructions,
	}
}
