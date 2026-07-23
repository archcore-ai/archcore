package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// hookEntry / hookMatcher are the decode shapes cmd tests use to assert on
// Claude-style hook configs; the production types live in internal/wiring.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

func configPathFor(base, relPath string) string {
	return filepath.Join(base, filepath.FromSlash(relPath))
}

func seedConfig(t *testing.T, base, relPath, content string) string {
	t.Helper()
	path := configPathFor(base, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
