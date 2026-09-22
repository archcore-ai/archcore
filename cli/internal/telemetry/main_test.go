package telemetry

import (
	"os"
	"testing"

	"archcore-cli/internal/testsupport"
)

// TestMain makes ambient-state isolation the default for this package rather
// than something each test has to remember. Telemetry resolves the shared
// install id under the state directory.
//
// A test that needs a specific home or state directory — including an empty
// one — still overrides it with t.Setenv. See
// testsupport.IsolateAmbientState for the incident this prevents.
func TestMain(m *testing.M) {
	iso := testsupport.IsolateAmbientState()
	os.Exit(iso.Finish(m.Run()))
}
