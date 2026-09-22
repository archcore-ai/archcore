package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCursor_WriteMCPConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(Cursor)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// Assert the archcore entry itself was written — not merely that the
	// mcpServers key exists. A no-op assignment would leave the key present
	// but the map empty, which a bare key check cannot detect.
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing archcore entry in mcpServers")
	}
}

func TestCursor_WriteMCPConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(Cursor)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("first WriteMCPConfig: %v", err)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("second WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}
}

func TestCursor_WriteMCPConfig_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cursorDir := filepath.Join(base, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"command": "other"},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "mcp.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(Cursor)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(cursorDir, "mcp.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}

	if _, ok := servers["other"]; !ok {
		t.Error("existing server lost")
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("archcore not added")
	}
}

func TestCursor_Detect_True(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if !ByID(Cursor).DetectFn(base) {
		t.Error("expected detection")
	}
}

func TestCursor_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(Cursor).DetectFn(base) {
		t.Error("expected no detection")
	}
}
