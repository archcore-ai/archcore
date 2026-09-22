//go:build unix

package mcp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// TestShieldHelperProcess is not a test: it is the child half of
// TestShieldStdout_FDLevel, re-executed via the test binary. It applies the
// shield, then writes through every stdout route a tool executor could take —
// a raw syscall on fd 1, a Go-level fmt print, and the protocol stream — and
// exits before the test framework can print PASS to the restored stdout.
func TestShieldHelperProcess(t *testing.T) {
	if os.Getenv("ARCHCORE_SHIELD_HELPER") != "1" {
		t.Skip("helper process for TestShieldStdout_FDLevel")
	}
	protocolOut, _ := shieldStdout()

	_, _ = syscall.Write(1, []byte("RAW.")) // raw fd 1 → must land on stderr
	fmt.Println("GOPRINT")                  // os.Stdout → must land on stderr
	_, _ = fmt.Fprint(protocolOut, "PROTO") // protocol stream → the only stdout bytes
	_ = protocolOut.Sync()
	os.Exit(0)
}

// TestShieldStdout_FDLevel proves the shield holds at the descriptor level:
// after shieldStdout, the parent-observed stdout carries protocol bytes ONLY,
// while raw fd-1 writes and fmt prints are diverted to stderr.
func TestShieldStdout_FDLevel(t *testing.T) {
	t.Parallel()
	cmd := exec.Command(os.Args[0], "-test.run=TestShieldHelperProcess$")
	cmd.Env = append(os.Environ(), "ARCHCORE_SHIELD_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("helper process failed: %v\nstderr:\n%s", err, stderr.String())
	}

	if got := stdout.String(); got != "PROTO" {
		t.Errorf("stdout must carry protocol bytes only, got %q", got)
	}
	if !strings.Contains(stderr.String(), "RAW.") {
		t.Errorf("raw fd-1 write must be diverted to stderr, stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "GOPRINT") {
		t.Errorf("fmt print must be diverted to stderr, stderr:\n%s", stderr.String())
	}
}
