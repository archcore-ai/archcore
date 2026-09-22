package wiring

import (
	"os"
	"path/filepath"
	"testing"
)

func setupArchcoreDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return base
}
