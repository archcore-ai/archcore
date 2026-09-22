package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"archcore-cli/internal/plugin"
)

// TestHostTrapRecordsTheAttemptAndFailsTheRun pins the property the whole guard
// rests on: reaching a host CLI must turn a passing run into a failing one.
//
// Asserting on Finish rather than on the stand-in's exit status is deliberate.
// internal/plugin's execCommand captures a child's streams into a capped buffer
// and reports the failure as data, not as an error, so a guard that only made
// the child exit nonzero would be absorbed by the code under test and the run
// would stay green — the exact shape of the incident this prevents.
func TestHostTrapRecordsTheAttemptAndFailsTheRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in host CLIs are #!/bin/sh scripts")
	}

	root := t.TempDir()
	iso := &Isolation{root: root, trap: filepath.Join(root, "host-invocations.log")}

	// Registers the current PATH for restoration, so armHostTrap's os.Setenv is
	// undone when this test ends.
	t.Setenv("PATH", os.Getenv("PATH"))
	iso.armHostTrap()

	if record := iso.hostTrapRecord(); len(record) != 0 {
		t.Fatalf("a freshly armed trap holds %v, want nothing", record)
	}

	// Resolved through PATH rather than by path, so this also proves the
	// stand-in shadows whatever host CLI the machine really has.
	resolved, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("LookPath(claude) after arming: %v", err)
	}
	if !strings.HasPrefix(resolved, root) {
		t.Fatalf("LookPath(claude) = %q, want the stand-in under %q", resolved, root)
	}
	if err := exec.Command(resolved, "plugin", "install", "archcore").Run(); err == nil {
		t.Error("the stand-in host CLI exited zero, want a failure")
	}

	record := iso.hostTrapRecord()
	if len(record) != 1 {
		t.Fatalf("trap recorded %v, want exactly one invocation", record)
	}
	if want := "claude plugin install archcore"; record[0] != want {
		t.Errorf("trap recorded %q, want %q", record[0], want)
	}

	if code := iso.Finish(0); code == 0 {
		t.Error("Finish returned 0 after a recorded host invocation, want a failing code")
	}
}

// TestFinishKeepsAQuietRunPassing: the guard may not manufacture a failure out
// of a run that reached no host CLI.
func TestFinishKeepsAQuietRunPassing(t *testing.T) {
	root := t.TempDir()
	iso := &Isolation{root: root, trap: filepath.Join(root, "host-invocations.log")}

	if code := iso.Finish(0); code != 0 {
		t.Errorf("Finish(0) on a quiet run = %d, want 0", code)
	}
}

// TestFinishPreservesAFailingCode: a real test failure must not be masked by
// the guard's own exit-code handling.
func TestFinishPreservesAFailingCode(t *testing.T) {
	root := t.TempDir()
	iso := &Isolation{root: root, trap: filepath.Join(root, "host-invocations.log")}

	if code := iso.Finish(3); code != 3 {
		t.Errorf("Finish(3) = %d, want the original 3", code)
	}
}

// TestHostCLIsCoversEveryPluginHost closes the one gap the duplication leaves.
//
// hostCLIs repeats the CLI column of internal/plugin's host table because that
// package's own TestMain calls IsolateAmbientState, so reading the table from
// isolate.go would cycle the import. Nothing stops a fifth host being added to
// the table and not to the list, and a host missing from the list is a host
// whose real binary a test can still execute — silently, on the developer's own
// machine, which is the incident the trap exists to prevent.
//
// The cycle does not reach here: internal/plugin imports nothing from this
// package outside its TestMain, and no test file is imported by anything.
func TestHostCLIsCoversEveryPluginHost(t *testing.T) {
	t.Parallel()

	listed := make(map[string]bool, len(hostCLIs))
	for _, cli := range hostCLIs {
		listed[cli] = true
	}

	checked := 0
	for _, spec := range plugin.Specs() {
		// A host with no CLI has no binary to execute and needs no stand-in.
		if spec.CLI == "" {
			continue
		}
		checked++
		if !listed[spec.CLI] {
			t.Errorf("host %q runs %q, which hostCLIs does not trap — a test could execute the real binary",
				spec.Host, spec.CLI)
		}
	}
	// Without this the test passes on an empty table, which is what a renamed
	// accessor or an emptied registry would produce.
	if checked == 0 {
		t.Fatal("no host with a CLI was checked — plugin.Specs() answered nothing")
	}
}
