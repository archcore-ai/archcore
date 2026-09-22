// Package xdg resolves the shared state directory for the CLI.
//
// One directory holds every piece of state the CLI keeps between invocations
// and outside any project: the hook dedup stamps, the update-check cache, and
// the install-id the installer scripts write. install.sh derives that directory
// as ${XDG_STATE_HOME:-$HOME/.local/state}/archcore, so the Go side must derive
// it the same way. A second derivation means a machine with two identities and
// a cache the installer cannot see.
//
// This package is a leaf: it imports only the standard library, so stamp,
// telemetry and update can all depend on it without a cycle.
//
// Failure bias: an unresolvable directory returns an empty string, not an
// error. What a caller does with that string is the caller's own decision, and
// the callers no longer agree. A dedup stamp, a cache and an anonymous analytics
// id degrade quietly and carry on; the fail-closed claim that guards unattended
// update refuses instead, because a state directory it cannot resolve is a state
// directory in which exclusivity cannot be established. The bias holds for both:
// no caller may fail a command because a home directory is missing, and a
// refusal to replace the binary fails no command.
package xdg

import (
	"os"
	"path/filepath"
)

// dirName is the per-tool directory inside the XDG state root.
const dirName = "archcore"

// StateDir returns ${XDG_STATE_HOME:-$HOME/.local/state}/archcore. It returns
// an empty string when neither XDG_STATE_HOME nor the home directory resolves;
// callers treat that as "no state directory" and degrade.
func StateDir() string {
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, dirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", dirName)
}
