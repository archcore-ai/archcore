package git

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain detaches the git subprocesses this package spawns from the
// developer's global configuration. A global log.showSignature corrupts
// `git log --format=%H`, and commit.gpgsign fails the helpers' commits.
func TestMain(m *testing.M) {
	testsupport.IsolateGit()
	os.Exit(m.Run())
}
