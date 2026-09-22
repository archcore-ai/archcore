package xdg

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStateDir(t *testing.T) {
	tests := []struct {
		name  string
		state string // XDG_STATE_HOME
		home  string // HOME
		want  func(state, home string) string
	}{
		{
			name:  "XDG_STATE_HOME wins",
			state: "/xdg-state",
			home:  "/home/user",
			want:  func(state, _ string) string { return filepath.Join(state, "archcore") },
		},
		{
			name:  "empty XDG_STATE_HOME falls back to the home default",
			state: "",
			home:  "/home/user",
			want:  func(_, home string) string { return filepath.Join(home, ".local", "state", "archcore") },
		},
		{
			name:  "neither resolves",
			state: "",
			home:  "",
			want:  func(_, _ string) string { return "" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Every case that leaves XDG_STATE_HOME empty reaches
			// os.UserHomeDir, which reads USERPROFILE on windows and ignores
			// the HOME this test sets — both the fallback path and the
			// unresolvable one.
			if tt.state == "" && runtime.GOOS == "windows" {
				t.Skip("os.UserHomeDir does not read HOME on windows")
			}
			t.Setenv("XDG_STATE_HOME", tt.state)
			t.Setenv("HOME", tt.home)

			if got, want := StateDir(), tt.want(tt.state, tt.home); got != want {
				t.Errorf("StateDir() = %q, want %q", got, want)
			}
		})
	}
}

// TestStateDir_MatchesInstallerLayout pins the one property that makes the
// shared install-id work: the path this package builds is the path
// install_id_path() in install.sh builds, so the installer and the CLI name the
// same machine.
func TestStateDir_MatchesInstallerLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)

	// install.sh: printf '%s/archcore/install-id' "${XDG_STATE_HOME:-...}"
	want := filepath.Join(root, "archcore", "install-id")
	if got := filepath.Join(StateDir(), "install-id"); got != want {
		t.Errorf("install-id path = %q, want %q", got, want)
	}
}

// TestStateDir_CreatesNothing: resolving a path must not touch the filesystem.
// Callers create the directory only once they have decided to write.
func TestStateDir_CreatesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)

	_ = StateDir()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("StateDir() created %d entries under the state root, want 0", len(entries))
	}
}
