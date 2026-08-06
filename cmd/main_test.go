package cmd

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain isolates the two pieces of ambient state this package's tests would
// otherwise inherit from the developer's machine.
//
// XDG_STATE_HOME: buildSessionContext and the session-start leaf resolve their
// stamp directory from it, falling back to $HOME. Without an override every
// test that builds a session context reads — and, once a test runs inside a git
// repository, writes — the developer's real ~/.local/state/archcore.
//
// Git: internal/git spawns git with this process's environment, so a global
// commit.gpgsign or core.hooksPath changes what the code under test observes.
func TestMain(m *testing.M) {
	testsupport.IsolateGit()

	stateDir, err := os.MkdirTemp("", "archcore-state")
	if err != nil {
		panic("creating temporary state dir: " + err.Error())
	}
	os.Setenv("XDG_STATE_HOME", stateDir)

	code := m.Run()
	os.RemoveAll(stateDir)
	os.Exit(code)
}
