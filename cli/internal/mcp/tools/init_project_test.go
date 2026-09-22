package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func resultText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if len(r.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("result content is %T, want mcp.TextContent", r.Content[0])
	}
	return tc.Text
}

func readOnDiskSettings(t *testing.T, base string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, ".archcore", "settings.json"))
	if err != nil {
		t.Fatalf("reading settings.json: %v", err)
	}
	var onDisk map[string]any
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	return onDisk
}

func TestHandleInitProject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		args            map[string]any
		wantErr         bool
		errContains     string
		wantSync        string
		wantLanguage    string // "" means expect absent from on-disk file
		wantArchcoreURL string
	}{
		{
			name:     "fresh init defaults",
			args:     nil,
			wantSync: "none",
		},
		{
			name:         "with language",
			args:         map[string]any{"language": "ru"},
			wantSync:     "none",
			wantLanguage: "ru",
		},
		{
			name:     "english default not persisted",
			args:     map[string]any{"language": "en"},
			wantSync: "none",
		},
		{
			name:     "sync cloud",
			args:     map[string]any{"sync_mode": "cloud"},
			wantSync: "cloud",
		},
		{
			name:            "on-prem with url",
			args:            map[string]any{"sync_mode": "on-prem", "archcore_url": "https://archcore.example.com"},
			wantSync:        "on-prem",
			wantArchcoreURL: "https://archcore.example.com",
		},
		{
			name:        "on-prem requires url",
			args:        map[string]any{"sync_mode": "on-prem"},
			wantErr:     true,
			errContains: "archcore_url",
		},
		{
			name:        "unknown sync mode",
			args:        map[string]any{"sync_mode": "bogus"},
			wantErr:     true,
			errContains: "unknown sync_mode",
		},
		{
			name:        "language with spaces rejected",
			args:        map[string]any{"language": "en US"},
			wantErr:     true,
			errContains: "spaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()

			result, err := callTool(HandleInitProject(StaticRoot(base)), tt.args)
			if err != nil {
				t.Fatalf("handler returned Go error: %v", err)
			}
			if result.IsError != tt.wantErr {
				t.Fatalf("IsError = %v, wantErr %v; body: %s", result.IsError, tt.wantErr, resultText(t, result))
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(resultText(t, result), tt.errContains) {
					t.Errorf("error %q should contain %q", resultText(t, result), tt.errContains)
				}
				if _, err := os.Stat(filepath.Join(base, ".archcore", "settings.json")); err == nil {
					t.Error("settings.json should not exist after failed init")
				}
				return
			}

			onDisk := readOnDiskSettings(t, base)
			if onDisk["sync"] != tt.wantSync {
				t.Errorf("sync = %v, want %v", onDisk["sync"], tt.wantSync)
			}
			if tt.wantLanguage == "" {
				if got, ok := onDisk["language"]; ok {
					t.Errorf("language should not be persisted; got %v", got)
				}
			} else if onDisk["language"] != tt.wantLanguage {
				t.Errorf("language = %v, want %v", onDisk["language"], tt.wantLanguage)
			}
			if tt.wantArchcoreURL != "" && onDisk["archcore_url"] != tt.wantArchcoreURL {
				t.Errorf("archcore_url = %v, want %v", onDisk["archcore_url"], tt.wantArchcoreURL)
			}
		})
	}
}

func TestHandleInitProject_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	if _, err := callTool(HandleInitProject(StaticRoot(base)), map[string]any{"language": "ru"}); err != nil {
		t.Fatalf("first init: %v", err)
	}

	result, err := callTool(HandleInitProject(StaticRoot(base)), map[string]any{"language": "en"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}

	var payload struct {
		AlreadyInitialized bool           `json:"already_initialized"`
		Settings           map[string]any `json:"settings"`
	}
	if err := json.Unmarshal([]byte(resultText(t, result)), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.AlreadyInitialized {
		t.Error("already_initialized should be true on second call")
	}
	if payload.Settings["language"] != "ru" {
		t.Errorf("idempotent call clobbered language: got %v, want ru", payload.Settings["language"])
	}
}

func TestHandleInitProject_CorruptSettingsNotOverwritten(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	path := filepath.Join(base, ".archcore", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte("{not json")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleInitProject(StaticRoot(base)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected error for corrupt settings; body: %s", resultText(t, result))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(corrupt) {
		t.Errorf("corrupt settings.json was overwritten: got %q, want %q", got, corrupt)
	}
}
