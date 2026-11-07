//go:build unix

package sync

import (
	"os"
	"path/filepath"
	"testing"
)

// An endpoint the process cannot stat is not a missing one: dropping its
// relations on EACCES would turn a permission problem into data loss.
func TestMissingEnds_UnreadableDirectoryCountsAsPresent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory modes")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "a.adr.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	got := MissingEnds(dir, []Relation{{Source: "locked/a.adr.md", Target: "locked/b.adr.md", Type: RelRelated}})
	if len(got) != 0 {
		t.Errorf("MissingEnds = %v, want none: EACCES is not absence", got)
	}
}
