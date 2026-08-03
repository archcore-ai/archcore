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

// mcpWriteMode controls behavior when an archcore entry already exists.
type mcpWriteMode int

const (
	// mcpKeepExisting — install semantics: an existing archcore entry is never
	// rewritten (the user may have customized it; drift is doctor's job).
	mcpKeepExisting mcpWriteMode = iota
	// mcpConverge — doctor --fix semantics: an existing archcore entry that
	// differs from the desired one is updated in place. Foreign servers and
	// unknown keys are still round-tripped untouched.
	mcpConverge
)

// writeMCPConfig is the shared implementation for writing MCP config files.
// It merges an "archcore" entry under serversKey, round-tripping everything
// else (unknown keys, other servers, key order) as opaque RawMessage, and
// writes atomically. Returns whether the file was changed.
func writeMCPConfig(filePath, serversKey string, entry any, mode mcpWriteMode) (bool, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// A target that exists but is not valid strict JSON is backed up as .bak
	// with a visible warning and rewritten fresh (backup-invalid-configs.adr).
	// The install aborts if the backup itself cannot be written.
	doc, backedUp, err := jsonfile.ReadOrBackup(filePath)
	if err != nil {
		return false, err
	}
	if backedUp {
		fmt.Println(display.WarnLine(fmt.Sprintf("Corrupted %s backed up, starting fresh", filePath)))
	}

	servers := jsonfile.NewDoc()
	if err := jsonfile.UnmarshalSection(doc, serversKey, servers); err != nil {
		return false, err
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return false, err
	}

	if existing, exists := servers.Get("archcore"); exists {
		if mode == mcpKeepExisting {
			return false, nil // already configured — no write
		}
		// mcpConverge: merge, don't replace — only the fields archcore owns
		// (those present in the desired entry) are overwritten; a user-added
		// field like "env" on the archcore entry survives the converge.
		merged, changed := mergeEntryFields(existing, entryJSON)
		if !changed {
			return false, nil
		}
		entryJSON = merged
	}
	servers.Set("archcore", json.RawMessage(entryJSON))

	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return false, err
	}
	doc.Set(serversKey, json.RawMessage(serversJSON))

	if err := jsonfile.SaveDoc(filePath, doc); err != nil {
		return false, err
	}
	return true, nil
}

// mergeEntryFields overlays the desired entry's fields onto the existing
// archcore entry, preserving any extra fields the user added (e.g. "env").
// Key order of the existing entry is kept; desired-only keys append. A
// non-object existing entry is replaced wholesale. Reports whether the merge
// result differs semantically from the existing entry.
func mergeEntryFields(existing, desired json.RawMessage) (json.RawMessage, bool) {
	entry := jsonfile.NewDoc()
	if json.Unmarshal(existing, entry) != nil {
		return desired, !jsonEqual(existing, desired)
	}
	fields := jsonfile.NewDoc()
	if json.Unmarshal(desired, fields) != nil {
		return desired, !jsonEqual(existing, desired)
	}
	for pair := fields.Oldest(); pair != nil; pair = pair.Next() {
		entry.Set(pair.Key, pair.Value)
	}
	merged, err := json.Marshal(entry)
	if err != nil {
		return desired, !jsonEqual(existing, desired)
	}
	return merged, !jsonEqual(existing, merged)
}

// jsonEqual reports semantic equality of two JSON values (key order and
// whitespace insensitive).
func jsonEqual(a, b json.RawMessage) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	ac, errA := json.Marshal(av)
	bc, errB := json.Marshal(bv)
	return errA == nil && errB == nil && string(ac) == string(bc)
}

var (
	standardMCPEntry = mcpServerEntry{
		Command: "archcore",
		Args:    []string{"mcp"},
	}
	// cursorMCPEntry passes the project root explicitly via
	// `--project ${workspaceFolder}`: Cursor does not guarantee the cwd it
	// spawns stdio MCP servers with (forum #99215; see
	// host-cwd-misrouting.adr), and it documents ${workspaceFolder}
	// interpolation in project-level `args`. resolveProjectRoot gives
	// --project top precedence, so the server root no longer depends on
	// spawn cwd.
	cursorMCPEntry = mcpServerEntry{
		Command: "archcore",
		Args:    []string{"mcp", "--project", "${workspaceFolder}"},
	}
)

// WriteStandardMCPJSON writes or merges an archcore entry into a standard
// mcpServers JSON config file (used by Claude Code, Gemini CLI, Roo Code).
func WriteStandardMCPJSON(filePath string) error {
	_, err := writeMCPConfig(filePath, "mcpServers", standardMCPEntry, mcpKeepExisting)
	return err
}

// WriteCursorMCPJSON writes or merges an archcore entry into Cursor's
// .cursor/mcp.json (see cursorMCPEntry for why its args differ).
func WriteCursorMCPJSON(filePath string) error {
	_, err := writeMCPConfig(filePath, "mcpServers", cursorMCPEntry, mcpKeepExisting)
	return err
}

// ConvergeStandardMCPJSON and ConvergeCursorMCPJSON are the doctor --fix
// counterparts of the writers above: an existing archcore entry that drifted
// from the desired shape (e.g. a Cursor config written before
// --project ${workspaceFolder} existed) is updated in place. Foreign servers
// and unknown keys stay untouched. They report whether the file changed.

func ConvergeStandardMCPJSON(filePath string) (bool, error) {
	return writeMCPConfig(filePath, "mcpServers", standardMCPEntry, mcpConverge)
}

func ConvergeCursorMCPJSON(filePath string) (bool, error) {
	return writeMCPConfig(filePath, "mcpServers", cursorMCPEntry, mcpConverge)
}
