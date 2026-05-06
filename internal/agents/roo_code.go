package agents

import "path/filepath"

var rooCodeMCPPath = filepath.Join(".roo", "mcp.json")

func rooCodeAgent() *Agent {
	return &Agent{
		ID:          RooCode,
		DisplayName: "Roo Code",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, rooCodeMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteStandardMCPJSON(filepath.Join(baseDir, rooCodeMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".roo"))
		},
	}
}
