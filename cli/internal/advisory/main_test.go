package advisory

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain detaches git from the developer's global configuration; the
// staleness advisory shells out to it.
func TestMain(m *testing.M) {
	testsupport.IsolateGit()
	os.Exit(m.Run())
}
