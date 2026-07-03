package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"archcore-cli/internal/display"
	"archcore-cli/internal/jsonfile"
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

// corruptPolicy controls what happens when the target file exists but is not
// valid strict JSON.
type corruptPolicy int

const (
	// corruptBackupAndReset backs the original up as .bak with a visible
	// warning and starts fresh (backup-invalid-configs.adr.md). The install
	// aborts if the backup itself cannot be written.
	corruptBackupAndReset corruptPolicy = iota
	// corruptSkipInstall leaves the file untouched and prints manual install
	// instructions. Used for JSONC-capable targets (.vscode/mcp.json), where
	// "invalid strict JSON" is usually a perfectly valid JSONC config whose
	// other MCP servers must not be silently replaced.
	corruptSkipInstall
)

// writeMCPConfig is the shared implementation for writing MCP config files.
// It merges an "archcore" entry under serversKey, round-tripping everything
// else (unknown keys, other servers, key order) as opaque RawMessage, and
// writes atomically. An already-configured file is never rewritten.
func writeMCPConfig(filePath, serversKey string, entry any, policy corruptPolicy) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	var doc *jsonfile.Doc
	switch policy {
	case corruptBackupAndReset:
		var backedUp bool
		var err error
		doc, backedUp, err = jsonfile.ReadOrBackup(filePath)
		if err != nil {
			return err
		}
		if backedUp {
			fmt.Println(display.WarnLine(fmt.Sprintf("Corrupted %s backed up, starting fresh", filePath)))
		}
	default: // corruptSkipInstall
		var err error
		doc, err = jsonfile.Read(filePath)
		if err != nil {
			entryJSON, _ := json.Marshal(entry)
			fmt.Println(display.WarnLine(fmt.Sprintf(
				"%s is not valid strict JSON (VS Code configs may contain comments) — left untouched", filePath)))
			fmt.Println(display.HintLine(fmt.Sprintf(
				"Add archcore manually under %q: \"archcore\": %s", serversKey, entryJSON)))
			return nil
		}
	}

	servers := jsonfile.NewDoc()
	if err := jsonfile.UnmarshalSection(doc, serversKey, servers); err != nil {
		return err
	}

	if _, exists := servers.Get("archcore"); exists {
		return nil // already configured — no write
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	servers.Set("archcore", json.RawMessage(entryJSON))

	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	doc.Set(serversKey, json.RawMessage(serversJSON))

	return jsonfile.SaveDoc(filePath, doc)
}

// WriteStandardMCPJSON writes or merges an archcore entry into a standard
// mcpServers JSON config file (used by Claude Code, Cursor, Roo Code).
func WriteStandardMCPJSON(filePath string) error {
	return writeMCPConfig(filePath, "mcpServers", mcpServerEntry{
		Command: "archcore",
		Args:    []string{"mcp"},
	}, corruptBackupAndReset)
}

// WriteVSCodeMCPJSON writes or merges an archcore entry into a VS Code-style
// MCP config file (uses "servers" key + "type": "stdio"), used by GitHub Copilot.
func WriteVSCodeMCPJSON(filePath string) error {
	return writeMCPConfig(filePath, "servers", vscodeMCPEntry{
		Type:    "stdio",
		Command: "archcore",
		Args:    []string{"mcp"},
	}, corruptSkipInstall)
}
