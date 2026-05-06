package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	}
}

// openCodeMCPEntry is the format OpenCode uses for MCP servers.
type openCodeMCPEntry struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command"`
	Cwd         string            `json:"cwd,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

func writeOpenCodeMCPConfig(baseDir string) error {
	configPath := filepath.Join(baseDir, "opencode.json")

	var raw map[string]json.RawMessage
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing %s: %w", configPath, err)
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		raw = make(map[string]json.RawMessage)
	} else {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	var mcpSection map[string]json.RawMessage
	if mcpRaw, ok := raw["mcp"]; ok {
		if err := json.Unmarshal(mcpRaw, &mcpSection); err != nil {
			return fmt.Errorf("parsing mcp section: %w", err)
		}
	} else {
		mcpSection = make(map[string]json.RawMessage)
	}

	// Skip only when both `cwd` and `environment.ARCHCORE_BASE_DIR` already
	// match baseDir — anything stale (project moved, missing env) is rewritten
	// so re-running `archcore mcp install` heals past installs automatically.
	if existingRaw, exists := mcpSection["archcore"]; exists {
		var existing map[string]any
		if json.Unmarshal(existingRaw, &existing) == nil {
			if archcoreEntryUpToDate(existing, baseDir, "environment") {
				return nil
			}
		}
	}

	entry := openCodeMCPEntry{
		Type:        "local",
		Command:     []string{"archcore", "mcp"},
		Cwd:         baseDir,
		Environment: map[string]string{"ARCHCORE_BASE_DIR": baseDir},
	}
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	mcpSection["archcore"] = json.RawMessage(entryJSON)

	mcpJSON, err := json.Marshal(mcpSection)
	if err != nil {
		return err
	}
	raw["mcp"] = json.RawMessage(mcpJSON)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(configPath, out, 0o644)
}
