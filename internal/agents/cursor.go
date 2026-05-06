package agents

import "path/filepath"

var cursorMCPPath = filepath.Join(".cursor", "mcp.json")

func cursorAgent() *Agent {
	return &Agent{
		ID:          Cursor,
		DisplayName: "Cursor",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, cursorMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteStandardMCPJSON(filepath.Join(baseDir, cursorMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".cursor"))
		},
	}
}
