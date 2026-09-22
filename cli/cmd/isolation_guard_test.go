package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"archcore-cli/internal/testsupport"
)

// This file guards the guard. TestMain's isolation is the reason the delivery
// entry points in this package can be tested at all without changing the machine
// running the suite, and nothing about a green run would otherwise reveal that
// the isolation had stopped being applied.

// TestIsolation_HomeResolvesInsideTheIsolationRoot: the delivery path writes
// under $HOME — a plugin marketplace entry goes to $HOME/.claude/settings.json.
// If the real home is still resolvable, that write lands on the developer.
func TestIsolation_HomeResolvesInsideTheIsolationRoot(t *testing.T) {
	testsupport.AssertNoRealHome(t, isolation)
}

// TestIsolation_AmbientStateVarsPointAtTheIsolationRoot walks the four variables
// that decide where ambient state lands. HOME and USERPROFILE are both checked
// because os.UserHomeDir reads a different one per platform.
//
// The assertion is containment in the isolation root, not inequality with the
// real home: once HOME is overridden the real home is no longer observable from
// inside this process, so comparing against it would compare the isolated value
// with itself and pass no matter what.
func TestIsolation_AmbientStateVarsPointAtTheIsolationRoot(t *testing.T) {
	root := isolation.Root()

	for _, name := range []string{"HOME", "USERPROFILE", "XDG_STATE_HOME", "XDG_CONFIG_HOME"} {
		t.Run(name, func(t *testing.T) {
			got := os.Getenv(name)
			if got == "" {
				t.Fatalf("%s is unset, so the tests inherit the machine's default", name)
			}
			if !strings.HasPrefix(got, root) {
				t.Fatalf("%s = %q, outside the isolation root %q", name, got, root)
			}
			if _, err := os.Stat(got); err != nil {
				t.Errorf("%s = %q, which does not exist: %v", name, got, err)
			}
		})
	}
}

// TestIsolation_HostCLIsResolveToAStandIn: a host CLI reached from a test must
// never be the real one. Running the real `claude` installs, updates or
// uninstalls a plugin on this machine — which is what happened once already.
func TestIsolation_HostCLIsResolveToAStandIn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in host CLIs are #!/bin/sh scripts")
	}

	for _, host := range []string{"claude", "codex", "copilot", "cursor"} {
		t.Run(host, func(t *testing.T) {
			resolved, err := exec.LookPath(host)
			if err != nil {
				t.Fatalf("LookPath(%s) = %v, want the stand-in", host, err)
			}
			root := isolation.Root()
			if !strings.HasPrefix(resolved, root) {
				t.Errorf("LookPath(%s) = %q, outside the isolation root %q — a test that "+
					"runs this would change the machine", host, resolved, root)
			}
		})
	}
}
