package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStandardMCPJSON_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	data, err := os.ReadFile(filePath)
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

	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' in mcpServers")
	}

	var entry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Command != "archcore" {
		t.Errorf("command = %q, want %q", entry.Command, "archcore")
	}
	if len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("args = %v, want [mcp]", entry.Args)
	}
}

func TestWriteStandardMCPJSON_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, err := os.ReadFile(filePath)
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
		t.Errorf("expected 1 server entry, got %d", len(servers))
	}
}

func TestWriteStandardMCPJSON_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")

	existing := map[string]any{
		"mcpServers": map[string]any{
			"other-tool": map[string]any{
				"command": "other-tool",
				"args":    []string{"serve"},
			},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	result, err2 := os.ReadFile(filePath)
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

	if _, ok := servers["other-tool"]; !ok {
		t.Error("existing 'other-tool' was lost during merge")
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' after install")
	}
}

func TestWriteStandardMCPJSON_CreatesDirs(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".cursor", "mcp.json")

	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestWriteVSCodeMCPJSON_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".vscode", "mcp.json")

	if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteVSCodeMCPJSON: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["servers"], &servers); err != nil {
		t.Fatalf("Unmarshal servers: %v", err)
	}

	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' in servers")
	}

	var entry struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Type != "stdio" {
		t.Errorf("type = %q, want %q", entry.Type, "stdio")
	}
	if entry.Command != "archcore" {
		t.Errorf("command = %q, want %q", entry.Command, "archcore")
	}
	if len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("args = %v, want [mcp]", entry.Args)
	}
}

func TestWriteVSCodeMCPJSON_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".vscode", "mcp.json")

	if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["servers"], &servers); err != nil {
		t.Fatalf("Unmarshal servers: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("expected 1 server entry, got %d", len(servers))
	}
}

func TestWriteVSCodeMCPJSON_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".vscode", "mcp.json")

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := map[string]any{
		"servers": map[string]any{
			"other-tool": map[string]any{
				"type":    "stdio",
				"command": "other-tool",
				"args":    []string{"serve"},
			},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteVSCodeMCPJSON: %v", err)
	}

	result, err2 := os.ReadFile(filePath)
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["servers"], &servers); err != nil {
		t.Fatalf("Unmarshal servers: %v", err)
	}

	if _, ok := servers["other-tool"]; !ok {
		t.Error("existing 'other-tool' was lost during merge")
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' after install")
	}
}

func TestWriteStandardMCPJSON_EmitsCwdAndEnv(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}
	data, _ := os.ReadFile(filePath)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal servers: %v", err)
	}
	var entry struct {
		Cwd string            `json:"cwd"`
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Cwd != base {
		t.Errorf("cwd = %q, want %q", entry.Cwd, base)
	}
	if entry.Env["ARCHCORE_BASE_DIR"] != base {
		t.Errorf("env.ARCHCORE_BASE_DIR = %q, want %q", entry.Env["ARCHCORE_BASE_DIR"], base)
	}
}

func TestWriteVSCodeMCPJSON_EmitsCwdAndEnv(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".vscode", "mcp.json")
	if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteVSCodeMCPJSON: %v", err)
	}
	data, _ := os.ReadFile(filePath)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["servers"], &servers); err != nil {
		t.Fatalf("Unmarshal servers: %v", err)
	}
	var entry struct {
		Type string            `json:"type"`
		Cwd  string            `json:"cwd"`
		Env  map[string]string `json:"env"`
	}
	if err := json.Unmarshal(servers["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Cwd != base {
		t.Errorf("cwd = %q, want %q", entry.Cwd, base)
	}
	if entry.Env["ARCHCORE_BASE_DIR"] != base {
		t.Errorf("env.ARCHCORE_BASE_DIR = %q, want %q", entry.Env["ARCHCORE_BASE_DIR"], base)
	}
}

func TestWriteStandardMCPJSON_RewritesStaleEntry(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	stale := map[string]any{
		"mcpServers": map[string]any{
			"archcore": map[string]any{
				"command": "archcore",
				"args":    []string{"mcp"},
				// no cwd, no env — pre-v0.5.0 shape
			},
		},
	}
	data, _ := json.MarshalIndent(stale, "", "  ")
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("WriteStandardMCPJSON: %v", err)
	}
	updated, _ := os.ReadFile(filePath)
	var raw map[string]json.RawMessage
	json.Unmarshal(updated, &raw)
	var servers map[string]json.RawMessage
	json.Unmarshal(raw["mcpServers"], &servers)
	var entry struct {
		Cwd string `json:"cwd"`
	}
	json.Unmarshal(servers["archcore"], &entry)
	if entry.Cwd != base {
		t.Errorf("stale entry not rewritten: cwd = %q, want %q", entry.Cwd, base)
	}
}

func TestWriteStandardMCPJSON_RewritesWhenStale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		oldBase string
		entry   func(oldBase string) map[string]any
	}{
		{
			name:    "cwd_mismatch",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"command": "archcore",
					"args":    []string{"mcp"},
					"cwd":     oldBase,
					"env":     map[string]any{"ARCHCORE_BASE_DIR": oldBase},
				}
			},
		},
		{
			name:    "env_mismatch",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"command": "archcore",
					"args":    []string{"mcp"},
					// cwd correct, env stale
					"cwd": "PLACEHOLDER",
					"env": map[string]any{"ARCHCORE_BASE_DIR": oldBase},
				}
			},
		},
		{
			name:    "env_missing",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"command": "archcore",
					"args":    []string{"mcp"},
					"cwd":     "PLACEHOLDER",
					// no env at all
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			filePath := filepath.Join(base, ".mcp.json")

			entry := c.entry(c.oldBase)
			if cwd, ok := entry["cwd"].(string); ok && cwd == "PLACEHOLDER" {
				entry["cwd"] = base
			}
			seed := map[string]any{
				"mcpServers": map[string]any{
					"archcore": entry,
				},
			}
			data, err := json.MarshalIndent(seed, "", "  ")
			if err != nil {
				t.Fatalf("MarshalIndent: %v", err)
			}
			if err := os.WriteFile(filePath, data, 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			if err := WriteStandardMCPJSON(filePath, base); err != nil {
				t.Fatalf("WriteStandardMCPJSON: %v", err)
			}

			updated, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(updated, &raw); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			var servers map[string]json.RawMessage
			if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
				t.Fatalf("Unmarshal mcpServers: %v", err)
			}
			var got struct {
				Cwd string            `json:"cwd"`
				Env map[string]string `json:"env"`
			}
			if err := json.Unmarshal(servers["archcore"], &got); err != nil {
				t.Fatalf("Unmarshal entry: %v", err)
			}
			if got.Cwd != base {
				t.Errorf("cwd = %q, want %q", got.Cwd, base)
			}
			if got.Env["ARCHCORE_BASE_DIR"] != base {
				t.Errorf("env.ARCHCORE_BASE_DIR = %q, want %q", got.Env["ARCHCORE_BASE_DIR"], base)
			}
		})
	}
}

func TestWriteVSCodeMCPJSON_RewritesWhenStale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		oldBase string
		entry   func(oldBase string) map[string]any
	}{
		{
			name:    "cwd_mismatch",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"type":    "stdio",
					"command": "archcore",
					"args":    []string{"mcp"},
					"cwd":     oldBase,
					"env":     map[string]any{"ARCHCORE_BASE_DIR": oldBase},
				}
			},
		},
		{
			name:    "env_mismatch",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"type":    "stdio",
					"command": "archcore",
					"args":    []string{"mcp"},
					"cwd":     "PLACEHOLDER",
					"env":     map[string]any{"ARCHCORE_BASE_DIR": oldBase},
				}
			},
		},
		{
			name:    "env_missing",
			oldBase: "/old/project",
			entry: func(oldBase string) map[string]any {
				return map[string]any{
					"type":    "stdio",
					"command": "archcore",
					"args":    []string{"mcp"},
					"cwd":     "PLACEHOLDER",
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			filePath := filepath.Join(base, ".vscode", "mcp.json")
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}

			entry := c.entry(c.oldBase)
			if cwd, ok := entry["cwd"].(string); ok && cwd == "PLACEHOLDER" {
				entry["cwd"] = base
			}
			seed := map[string]any{
				"servers": map[string]any{
					"archcore": entry,
				},
			}
			data, err := json.MarshalIndent(seed, "", "  ")
			if err != nil {
				t.Fatalf("MarshalIndent: %v", err)
			}
			if err := os.WriteFile(filePath, data, 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			if err := WriteVSCodeMCPJSON(filePath, base); err != nil {
				t.Fatalf("WriteVSCodeMCPJSON: %v", err)
			}

			updated, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(updated, &raw); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			var servers map[string]json.RawMessage
			if err := json.Unmarshal(raw["servers"], &servers); err != nil {
				t.Fatalf("Unmarshal servers: %v", err)
			}
			var got struct {
				Cwd string            `json:"cwd"`
				Env map[string]string `json:"env"`
			}
			if err := json.Unmarshal(servers["archcore"], &got); err != nil {
				t.Fatalf("Unmarshal entry: %v", err)
			}
			if got.Cwd != base {
				t.Errorf("cwd = %q, want %q", got.Cwd, base)
			}
			if got.Env["ARCHCORE_BASE_DIR"] != base {
				t.Errorf("env.ARCHCORE_BASE_DIR = %q, want %q", got.Env["ARCHCORE_BASE_DIR"], base)
			}
		})
	}
}

func TestWriteStandardMCPJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	filePath := filepath.Join(base, ".mcp.json")
	corrupted := []byte("not json")
	if err := os.WriteFile(filePath, corrupted, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := WriteStandardMCPJSON(filePath, base); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// .bak should contain original corrupted content.
	bakData, err := os.ReadFile(filePath + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(bakData) != string(corrupted) {
		t.Errorf("bak content = %q, want %q", bakData, corrupted)
	}

	// .mcp.json should now be valid with archcore entry.
	data, err := os.ReadFile(filePath)
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
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing 'archcore' in mcpServers after recovery")
	}
}
