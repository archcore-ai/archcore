package testsupport

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// hostCLIs are the coding-agent CLIs the plugin delivery path executes. Running
// one of these for real installs, updates or uninstalls a plugin on the machine
// running the suite.
//
// It repeats the CLI column of internal/plugin's host table rather than reading
// it, because internal/plugin's own TestMain calls into this package and the
// import would cycle. A host added to that table and not to this list escapes
// the trap silently, so the two are checked against each other in
// isolate_test.go, which may import internal/plugin because nothing imports it
// back.
var hostCLIs = []string{"claude", "codex", "copilot", "cursor"}

// Isolation is a live ambient-state isolation, returned by IsolateAmbientState.
type Isolation struct {
	root string
	trap string
}

// IsolateAmbientState points the home and XDG environment variables a test
// could inherit from the developer's machine at a fresh temporary directory,
// and arms a trap for any test that reaches a host CLI without isolating PATH
// itself. Call it from TestMain and pass the result of m.Run to Finish.
//
// The working directory is deliberately not moved. `go test` runs each package
// in its own source directory, and the tests that parse this repository's own
// source with go/parser depend on that; a helper that resolves a project root
// from os.Getwd() therefore still lands in the real repository, and a test that
// needs a different root passes one explicitly or calls t.Chdir.
//
// This exists because of a real incident. A test ran `archcore init --agent
// claude-code` with neither HOME nor PATH overridden; on a machine with Claude
// Code installed it executed three real host commands and wrote a marketplace
// entry into the developer's own ~/.claude/settings.json. The suite stayed green
// throughout — nothing about a passing run revealed that the machine had changed.
//
// Two separate defences, because the incident had two halves:
//
//   - HOME, USERPROFILE, XDG_STATE_HOME and XDG_CONFIG_HOME move to a temporary
//     directory, so a write that escapes a test's own t.TempDir lands somewhere
//     harmless. A test that wants a specific home still overrides these with
//     t.Setenv, including to the empty string for the unresolvable-directory
//     cases in update and xdg.
//
//     These four are the whole set: no other ambient variable is touched, and
//     the working directory stays where `go test` put it.
//
//   - Stand-in host CLIs go first on PATH. They run no host command; they record
//     the attempt and fail. The recording is the point. internal/plugin's
//     execCommand captures a child's stderr into a capped buffer and reports a
//     failure as data rather than an error, so a stand-in that only printed a
//     warning would be swallowed and the run would stay green. Finish reads the
//     record instead, which no production error handling can absorb.
//
// PATH keeps its remaining entries: the fixture host CLIs are /bin/sh scripts
// that call cat and sleep, and tests that need a real host CLI put their own
// fixture on a PATH of their own.
func IsolateAmbientState() *Isolation {
	root, err := os.MkdirTemp("", "archcore-isolated")
	if err != nil {
		// This runs inside TestMain, before any test exists to fail. A panic
		// here answers "the suite cannot be isolated" with a goroutine dump;
		// the one line below is what a reader needs.
		fmt.Fprintln(os.Stderr, "FAIL: creating the isolation root:", err)
		os.Exit(1)
	}

	iso := &Isolation{root: root, trap: filepath.Join(root, "host-invocations.log")}

	home := filepath.Join(root, "home")
	mustMkdir(home)
	// os.UserHomeDir reads USERPROFILE on windows and HOME everywhere else;
	// setting only one leaves the other platform inheriting the real directory.
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("USERPROFILE", home)

	state := filepath.Join(root, "state")
	mustMkdir(state)
	_ = os.Setenv("XDG_STATE_HOME", state)

	config := filepath.Join(root, "config")
	mustMkdir(config)
	_ = os.Setenv("XDG_CONFIG_HOME", config)

	iso.armHostTrap()
	return iso
}

// armHostTrap puts a recording stand-in for every host CLI first on PATH.
//
// POSIX only: the stand-ins are shell scripts, and a windows run would need
// .cmd shims plus a PATHEXT that lists them. The repository's other host
// fixtures skip on windows for the same reason, and CI runs Linux.
func (iso *Isolation) armHostTrap() {
	if runtime.GOOS == "windows" {
		return
	}
	bin := filepath.Join(iso.root, "bin")
	mustMkdir(bin)

	for _, host := range hostCLIs {
		// The trap path is single-quoted because a temporary directory may
		// contain characters the shell would otherwise split on.
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s %%s\\n' %s \"$*\" >> '%s'\nexit 1\n",
			host, iso.trap)
		if err := os.WriteFile(filepath.Join(bin, host), []byte(script), 0o755); err != nil {
			panic("writing the " + host + " stand-in: " + err.Error())
		}
	}
	_ = os.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Finish removes the isolation root and returns the exit code TestMain should
// use. It turns a recorded host-CLI invocation into a failing run even when
// every test passed, because that is the exact shape the original incident had.
func (iso *Isolation) Finish(code int) int {
	record, err := os.ReadFile(iso.trap)
	if err == nil && len(record) > 0 {
		fmt.Fprintf(os.Stderr, "\nFAIL: a test executed a host CLI without isolating PATH.\n\n%s\n"+
			"Running a host CLI for real installs, updates or uninstalls a plugin on this\n"+
			"machine. Isolate PATH before exercising a delivery entry point — in cmd that is\n"+
			"isolatePluginRun(t), which points PATH at an empty directory and returns it so a\n"+
			"test can stage its own stand-in host.\n", string(record))
		if code == 0 {
			code = 1
		}
	}
	_ = os.RemoveAll(iso.root)
	return code
}

// Root is the directory every isolated path lives under. A test asserting that
// some path is isolated compares against this rather than against the real home,
// which is no longer observable from inside an isolated process —
// cmd/isolation_guard_test.go is the caller outside this package.
func (iso *Isolation) Root() string { return iso.root }

// hostTrapRecord returns the host-CLI invocations recorded so far, one per line.
// It exists so a test can prove the trap is armed rather than assume it, and it
// is unexported because only this package's own tests read it.
func (iso *Isolation) hostTrapRecord() []string {
	record, err := os.ReadFile(iso.trap)
	if err != nil || len(record) == 0 {
		return nil
	}
	return strings.Split(strings.TrimRight(string(record), "\n"), "\n")
}

// AssertNoRealHome fails when the environment still resolves to a home
// directory outside the isolation root — the state in which a stray write
// reaches the developer's own files.
func AssertNoRealHome(tb testing.TB, iso *Isolation) {
	tb.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		return // an unresolvable home is the safe end of this range
	}
	// The separator matters: a bare prefix test also accepts a sibling whose
	// name merely starts with the root's, and /tmp/archcore-isolatedXYZ-real is
	// exactly as unisolated as the developer's own home.
	if home != iso.root && !strings.HasPrefix(home, iso.root+string(os.PathSeparator)) {
		tb.Errorf("os.UserHomeDir() = %q, outside the isolation root %q", home, iso.root)
	}
}

func mustMkdir(dir string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic("creating " + dir + ": " + err.Error())
	}
}
