package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// mcpServerEntry represents an MCP server configuration.
type mcpServerEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// vscodeMCPEntry represents an MCP server entry in VS Code format (used by Copilot).
type vscodeMCPEntry struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// writeMCPConfig is the shared implementation for writing MCP config files.
// It reads/creates a JSON file, merges an "archcore" entry under the given
// serversKey, and writes it back.
func writeMCPConfig(filePath, serversKey string, entry any) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	var raw map[string]json.RawMessage
	data, err := os.ReadFile(filePath)
	if err == nil {
		if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
			_ = os.WriteFile(filePath+".bak", data, 0o644)
			raw = make(map[string]json.RawMessage)
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		raw = make(map[string]json.RawMessage)
	} else {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	var servers map[string]json.RawMessage
	if serversRaw, ok := raw[serversKey]; ok {
		if unmarshalErr := json.Unmarshal(serversRaw, &servers); unmarshalErr != nil {
			return fmt.Errorf("parsing %s section: %w", serversKey, unmarshalErr)
		}
	} else {
		servers = make(map[string]json.RawMessage)
	}

	if _, exists := servers["archcore"]; exists {
		return nil // already configured
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	servers["archcore"] = json.RawMessage(entryJSON)

	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	raw[serversKey] = json.RawMessage(serversJSON)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(filePath, out, 0o644)
}

// WriteStandardMCPJSON writes or merges an archcore entry into a standard
// mcpServers JSON config file (used by Claude Code, Cursor, Roo Code).
func WriteStandardMCPJSON(filePath string) error {
	return writeMCPConfig(filePath, "mcpServers", mcpServerEntry{
		Command: "archcore",
		Args:    []string{"mcp"},
	})
}

// WriteVSCodeMCPJSON writes or merges an archcore entry into a VS Code-style
// MCP config file (uses "servers" key + "type": "stdio"), used by GitHub Copilot.
func WriteVSCodeMCPJSON(filePath string) error {
	return writeMCPConfig(filePath, "servers", vscodeMCPEntry{
		Type:    "stdio",
		Command: "archcore",
		Args:    []string{"mcp"},
	})
}
