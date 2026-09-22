package plugin

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain makes ambient-state isolation the default for this package rather
// than something each test has to remember. The plugin engine executes host
// CLIs and reads $HOME/.claude/settings.json, so an unisolated test both runs
// real host commands and writes to the developer's own config.
//
// A test that needs a specific home or state directory — including an empty
// one — still overrides it with t.Setenv. See
// testsupport.IsolateAmbientState for the incident this prevents.
func TestMain(m *testing.M) {
	iso := testsupport.IsolateAmbientState()
	os.Exit(iso.Finish(m.Run()))
}
