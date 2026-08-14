package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// twoSourceFixture builds a project with a local .archcore and one declared
// global source ("org", mounted from a sibling directory), and returns the
// project base directory plus both .archcore directories. Tests write their
// own documents into the returned directories.
func twoSourceFixture(t *testing.T) (base, localArch, globalArch string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "project")
	localArch = filepath.Join(base, ".archcore")
	globalArch = filepath.Join(root, "org", ".archcore")
	for _, dir := range []string{localArch, globalArch} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	settings := `{"sync":"none","globals":[{"id":"org","path":"../org/.archcore"}]}`
	if err := os.WriteFile(filepath.Join(localArch, "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	return base, localArch, globalArch
}
