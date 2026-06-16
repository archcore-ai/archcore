package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSettings writes baseDir/.archcore/settings.json with the given JSON body.
func seedSettings(t *testing.T, baseDir, body string) {
	t.Helper()
	dir := filepath.Join(baseDir, ".archcore")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestConfigSet_PreservesUnknownField_EndToEnd is the core write-preserving
// acceptance: an older-but-tolerant binary running `config set` must not drop a
// field it does not recognize (e.g. `globals` added by a newer version).
func TestConfigSet_PreservesUnknownField_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	seedSettings(t, dir, `{"sync":"none","language":"en","future_flag":true}`)

	if _, err := runCmdInDir(t, dir, "config", "set", "language", "ru"); err != nil {
		t.Fatalf("config set: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, ".archcore", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("re-parse saved settings: %v\n%s", err, raw)
	}
	if m["language"] != "ru" {
		t.Errorf("language = %v, want ru", m["language"])
	}
	if _, ok := m["future_flag"]; !ok {
		t.Errorf("unknown field future_flag was dropped on config set:\n%s", raw)
	}
}

// TestConfigGet_StdoutCleanWithUnknownField verifies the unknown-field warning
// goes to stderr, leaving `config get` stdout machine-readable. runCmdInDir
// captures only stdout, so a clean value proves the warning did not land there.
func TestConfigGet_StdoutCleanWithUnknownField(t *testing.T) {
	dir := t.TempDir()
	seedSettings(t, dir, `{"sync":"none","language":"ru","future_flag":true}`)

	out, err := runCmdInDir(t, dir, "config", "get", "language")
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if strings.TrimSpace(out) != "ru" {
		t.Errorf("stdout = %q, want exactly \"ru\" (warning must go to stderr)", out)
	}
}
