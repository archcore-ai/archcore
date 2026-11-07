//go:build unix

package agents

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The plain os.WriteFile this writer replaced honored the umask on create; a
// CreateTemp plus chmod would publish a new .codex/config.toml world-readable.
func TestCodexCLI_WriteMCPConfig_NewFileHonorsTheUmask(t *testing.T) {
	// Not parallel: the umask is process-global.
	old := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(old) })

	base := t.TempDir()
	if err := ByID(CodexCLI).WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}
	info, err := os.Stat(filepath.Join(base, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config.toml published at %#o under umask 077, want 0600", got)
	}
}
