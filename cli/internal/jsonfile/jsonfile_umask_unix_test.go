//go:build unix

package jsonfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestWriteAtomic_ACreatedFileHonorsTheUmask pins the half of the mode contract
// that a chmod cannot express.
//
// A user with `umask 077` expects a newly created config file to be owner-only,
// which is what os.WriteFile gave before this helper took over the write. The
// perm therefore has to reach open(2), where the umask applies to it — an
// os.CreateTemp plus chmod bypasses the umask entirely and published every
// user-owned config file this package writes at a world-readable 0o644.
//
// It lives in its own file because syscall.Umask does not exist on windows, and
// a runtime.GOOS skip would still have to compile there.
func TestWriteAtomic_ACreatedFileHonorsTheUmask(t *testing.T) {
	// Not parallel, and no sibling in this file: the umask is process-global.
	old := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(old) })

	path := filepath.Join(t.TempDir(), "fresh.json")
	if err := WriteAtomic(path, []byte("data")); err != nil {
		t.Fatalf("WriteAtomic() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("published at %#o under umask 077, want 0600", got)
	}
}
