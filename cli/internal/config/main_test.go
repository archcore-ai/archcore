package config

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain detaches the git subprocesses this package spawns from the
// developer's global configuration. deriveAnchor shells out to git through
// internal/git, so this package now needs the same isolation internal/git has
// (isolating-the-machine-from-the-test-suite.guide).
func TestMain(m *testing.M) {
	testsupport.IsolateGit()
	os.Exit(m.Run())
}
