package cmd

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// isolation is the package-wide ambient-state isolation, kept so the guard
// tests in isolation_guard_test.go can inspect it rather than assume it.
var isolation *testsupport.Isolation

// TestMain isolates the ambient state this package's tests would otherwise
// inherit from the developer's machine.
//
// The state directory: buildSessionContext and the session-start leaf resolve
// their stamp directory from XDG_STATE_HOME, falling back to $HOME. Without an
// override every test that builds a session context reads — and, once a test
// runs inside a git repository, writes — the developer's real
// ~/.local/state/archcore.
//
// The home directory and PATH: this package holds the delivery entry points,
// `archcore init --agent` and `archcore plugin`, which write under $HOME and
// execute host CLIs. See IsolateAmbientState for the incident that makes both
// defences necessary.
//
// Git: internal/git spawns git with this process's environment, so a global
// commit.gpgsign or core.hooksPath changes what the code under test observes.
func TestMain(m *testing.M) {
	testsupport.IsolateGit()
	isolation = testsupport.IsolateAmbientState()
	os.Exit(isolation.Finish(m.Run()))
}
