package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// mcpServerEntry represents an MCP server configuration. cwd and env are
// emitted unconditionally so the host launches archcore in the resolved
// project — without them, hosts default cwd to the workspace folder (which
// is sometimes $HOME under user-scope MCP installs) and the CLI's project
// resolution gets confused. ARCHCORE_BASE_DIR is the canonical signal.
type mcpServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// vscodeMCPEntry represents an MCP server entry in VS Code format (used by Copilot).
type vscodeMCPEntry struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// archcoreEntryUpToDate reports whether an existing archcore MCP entry
// already targets baseDir. The entry is considered fresh iff `cwd` equals
// baseDir AND the env map under envKey contains ARCHCORE_BASE_DIR=baseDir.
// envKey differs by host: standard JSON / VS Code use "env"; OpenCode uses
// "environment".
func archcoreEntryUpToDate(existing map[string]any, baseDir, envKey string) bool {
	cwd, ok := existing["cwd"].(string)
	if !ok || cwd != baseDir {
		return false
	}
	envRaw, ok := existing[envKey].(map[string]any)
	if !ok {
		return false
	}
	got, ok := envRaw["ARCHCORE_BASE_DIR"].(string)
	if !ok {
		return false
	}
	return got == baseDir
}

// writeMCPConfig is the shared implementation for writing MCP config files.
// It reads/creates a JSON file, merges an "archcore" entry under the given
// serversKey, and writes it back. baseDir is used to detect stale entries
// from prior installs and force a rewrite when cwd or env drifted.
func writeMCPConfig(filePath, serversKey, baseDir string, entry any) error {
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

	// If an archcore entry already exists, only skip when it is already
	// up-to-date — both `cwd` and `env.ARCHCORE_BASE_DIR` must match the
	// current baseDir. Anything stale (project moved, missing env, old
	// shape) gets rewritten so a re-run of `archcore mcp install` heals
	// past installs automatically.
	if existingRaw, exists := servers["archcore"]; exists {
		var existing map[string]any
		if json.Unmarshal(existingRaw, &existing) == nil {
			if archcoreEntryUpToDate(existing, baseDir, "env") {
				return nil
			}
		}
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
//
// baseDir is emitted as both `cwd` and `env.ARCHCORE_BASE_DIR` so the host
// launches the MCP server in the project — and even if the host overrides
// cwd, the env var anchors the CLI to the right directory.
func WriteStandardMCPJSON(filePath, baseDir string) error {
	return writeMCPConfig(filePath, "mcpServers", baseDir, mcpServerEntry{
		Command: "archcore",
		Args:    []string{"mcp"},
		Cwd:     baseDir,
		Env:     map[string]string{"ARCHCORE_BASE_DIR": baseDir},
	})
}

// WriteVSCodeMCPJSON writes or merges an archcore entry into a VS Code-style
// MCP config file (uses "servers" key + "type": "stdio"), used by GitHub Copilot.
func WriteVSCodeMCPJSON(filePath, baseDir string) error {
	return writeMCPConfig(filePath, "servers", baseDir, vscodeMCPEntry{
		Type:    "stdio",
		Command: "archcore",
		Args:    []string{"mcp"},
		Cwd:     baseDir,
		Env:     map[string]string{"ARCHCORE_BASE_DIR": baseDir},
	})
}
