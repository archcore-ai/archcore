//go:build !windows

package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceMigrationPreservesSymlinkAndPermissions(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	settings := paths[1]
	target := filepath.Join(t.TempDir(), "private.json")
	if err := os.Rename(settings, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, settings); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(200, migrationClaudeCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(settings)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink replaced: %v", err)
	}
	info, err = os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions changed: %v", err)
	}
	backups, err := filepath.Glob(target + ".archcore-source-*.bak")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
	info, err = os.Stat(backups[0])
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions=%v err=%v", info, err)
	}
}

func TestReadSourceFileRefusesDanglingSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.Symlink("missing.json", path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readSourceFile(path); err == nil {
		t.Fatal("dangling symlink treated as absent settings")
	}
}

func TestSourceMigrationDeduplicatesSymlinkedDeclarations(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(project, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(paths[1], local); err != nil {
		t.Fatal(err)
	}
	a.Evidence.Installs = []Install{{Scope: ScopeLocal, ProjectPath: project, ProjectPresent: true}}
	calls := 0
	if err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(200, migrationClaudeCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(paths[1] + ".archcore-source-*.bak")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
}
