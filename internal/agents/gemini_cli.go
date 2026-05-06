package agents

import "path/filepath"

var geminiCLIMCPPath = filepath.Join(".gemini", "settings.json")

func geminiCLIAgent() *Agent {
	return &Agent{
		ID:          GeminiCLI,
		DisplayName: "Gemini CLI",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, geminiCLIMCPPath)
		},
		WriteMCPConfig: func(baseDir string) error {
			return WriteStandardMCPJSON(filepath.Join(baseDir, geminiCLIMCPPath))
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".gemini"))
		},
	}
}
