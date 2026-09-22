package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A failed write must not corrupt the user's file: writeFileAtomic renames a
// sibling temp into place, so if the temp cannot even be created the original
// content survives byte-for-byte. Before the atomic change, os.WriteFile
// truncated the target first — a mid-write failure left it half-written.
func TestUpsertFencedBlock_FailedWriteLeavesOriginalIntact(t *testing.T) {
	// Not parallel: toggles directory permissions.
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory write permissions; cannot force a write failure")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	const original = "# precious user content\n"
	writeFile(t, path, original)

	// Strip write permission from the directory so the sibling temp file cannot
	// be created — the failure mode we want to prove is survivable.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := upsertFencedBlock(path); err == nil {
		t.Fatal("expected write to fail in an unwritable directory")
	}

	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != original {
		t.Errorf("original content was clobbered by a failed write:\n got: %q\nwant: %q", got, original)
	}
}

// os.WriteFile keeps an existing file's permission bits (its mode arg only
// applies on create). The atomic temp+rename must not regress that — a file the
// user chmod'd to 0600 stays 0600 after the nudge is upserted.
func TestUpsertFencedBlock_PreservesExistingPermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	writeFile(t, path, "# mine\n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsertFencedBlock: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("permissions = %o, want 0600 (atomic write must preserve existing mode)", got)
	}
	if !strings.Contains(readFile(t, path), instructionsMarkerStart) {
		t.Error("managed block was not written")
	}
}

// A symlinked instruction file must be written THROUGH to its target (matching
// os.WriteFile), so the link is preserved rather than replaced by a regular
// file — otherwise a repo that symlinks CLAUDE.md/AGENTS.md would silently lose
// the link on the first nudge write.
func TestUpsertFencedBlock_WritesThroughSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "real-AGENTS.md")
	writeFile(t, target, "# mine\n")

	link := filepath.Join(dir, "AGENTS.md")
	if err := os.Symlink("real-AGENTS.md", link); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}

	if err := upsertFencedBlock(link); err != nil {
		t.Fatalf("upsertFencedBlock: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced by a regular file; the write must go through the link")
	}
	if !strings.Contains(readFile(t, target), instructionsMarkerStart) {
		t.Error("managed block did not land in the symlink target")
	}
}

// A DANGLING symlink cannot be written through — its target does not exist, so
// EvalSymlinks fails and the link path itself is materialized as a regular file
// carrying the block (the behavior writeFileAtomic's doc promises). This pins
// that documented edge so it can't regress into e.g. writing to the missing
// target or erroring.
func TestUpsertFencedBlock_DanglingSymlinkBecomesRegularFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	link := filepath.Join(dir, "AGENTS.md")
	if err := os.Symlink("nonexistent-target.md", link); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}

	if err := upsertFencedBlock(link); err != nil {
		t.Fatalf("upsertFencedBlock: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("dangling symlink should have been materialized as a regular file")
	}
	if !strings.Contains(readFile(t, link), instructionsMarkerStart) {
		t.Error("managed block did not land at the (former) link path")
	}
	// The dangling target must not have been created behind the link.
	if _, err := os.Lstat(filepath.Join(dir, "nonexistent-target.md")); !os.IsNotExist(err) {
		t.Errorf("write should not have created the dangling target, lstat err = %v", err)
	}
}
