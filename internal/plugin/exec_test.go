package plugin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The fixture host CLIs are #!/bin/sh scripts written into t.TempDir() and
// reached by swapping the lookPath seam, which is also what the executor
// resolves the binary through.
//
// CI runs Linux only, so Windows ships unverified here: a shebang means nothing
// there and exec.LookPath wants an executable extension, so both the fixtures
// and the PATH resolution they stand in for would need a different shape.
func skipWithoutPOSIXShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixture host CLIs are #!/bin/sh scripts; CI runs Linux only")
	}
}

// writeHostCLI writes one executable fixture CLI into dir.
func writeHostCLI(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("writing the %s fixture: %v", name, err)
	}
}

// useHostCLIs points the lookPath seam at a directory of fixtures and answers
// "not found" for every other name, so a test states exactly which host CLIs
// the machine has.
func useHostCLIs(t *testing.T, dir string, names ...string) {
	t.Helper()
	present := make(map[string]bool, len(names))
	for _, name := range names {
		present[name] = true
	}
	original := lookPath
	lookPath = func(file string) (string, error) {
		if !present[file] {
			return "", exec.ErrNotFound
		}
		return filepath.Join(dir, file), nil
	}
	t.Cleanup(func() { lookPath = original })
}

// noHostCLIs answers "not found" for every name.
func noHostCLIs(t *testing.T) {
	t.Helper()
	useHostCLIs(t, t.TempDir())
}

// listingScript builds a fixture that answers `<cli> plugin list [--json]` with
// answer and runs mutation for every other subcommand, so one fixture serves
// both the evidence question and the mutating command that follows it.
func listingScript(answer, mutation string) string {
	return "case \"$2\" in\n  list) cat <<'FIXTURE_EOF'\n" + answer +
		"\nFIXTURE_EOF\n    ;;\n  *) " + mutation + " ;;\nesac"
}

// pluginListing is a host answer that names the Archcore plugin, with a version.
const pluginListing = `[{"name":"archcore@archcore-plugins","version":"1.4.0"}]`

// otherPluginListing is a host answer from a machine with plugins, none of them
// this one.
const otherPluginListing = `[{"name":"someone-else@other-marketplace","version":"9.9.9"}]`

// shrinkTimeouts makes the two budgets reachable inside a test. Nothing else in
// the package may change them.
func shrinkTimeouts(t *testing.T, command, step time.Duration) {
	t.Helper()
	originalCommand, originalStep := commandTimeout, stepTimeout
	commandTimeout, stepTimeout = command, step
	t.Cleanup(func() { commandTimeout, stepTimeout = originalCommand, originalStep })
}

// isolateHome points the registry scan at an empty home directory, so a test
// never reads the developer's own host caches.
//
// Both variables are set because os.UserHomeDir reads a different one per
// platform: HOME everywhere but Windows, USERPROFILE there. Setting only HOME
// would leave the registry tests reading the real home on one platform, where
// the developer's own installed plugin decides the verdict.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// setInteractive pins the terminal answer, which decides whether the executor
// appends a host's non-interactive flag. Without it a test would pass or fail
// depending on whether it ran from a terminal.
func setInteractive(t *testing.T, interactive bool) {
	t.Helper()
	original := interactiveSession
	interactiveSession = func() bool { return interactive }
	t.Cleanup(func() { interactiveSession = original })
}

// runRecorder records every command that reaches the subprocess seam. It wraps
// the real seam rather than replacing it, so a test gets the count and the
// fixture's behavior together.
type runRecorder struct {
	commands []Command
}

func (r *runRecorder) lines() []string {
	out := make([]string, 0, len(r.commands))
	for _, c := range r.commands {
		out = append(out, c.String())
	}
	return out
}

func recordRuns(t *testing.T) *runRecorder {
	t.Helper()
	rec := &runRecorder{}
	original := runCommand
	runCommand = func(ctx context.Context, c Command) commandOutcome {
		rec.commands = append(rec.commands, c)
		return original(ctx, c)
	}
	t.Cleanup(func() { runCommand = original })
	return rec
}

// stubRuns replaces the subprocess seam outright, for the cases a fixture
// cannot produce cheaply.
func stubRuns(t *testing.T, answer func(c Command) commandOutcome) {
	t.Helper()
	original := runCommand
	runCommand = func(_ context.Context, c Command) commandOutcome { return answer(c) }
	t.Cleanup(func() { runCommand = original })
}

// TestExecCommandCapturesStdoutAndDiscardsStderr pins which stream survives.
// Both specs answer a failed command with the command line rather than with
// what the host said, and a host CLI's stderr routinely carries a path or a
// token — so stderr has no reachable consumer and is not captured at all. A
// host that writes to it must still exit cleanly, which is the other half of
// this case.
func TestExecCommandCapturesStdoutAndDiscardsStderr(t *testing.T) {
	skipWithoutPOSIXShell(t)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `echo out; echo err >&2`)
	useHostCLIs(t, dir, "claude")

	out := execCommand(t.Context(), Command{Name: "claude", Args: []string{"plugin", "list"}})
	if out.Failed {
		t.Fatalf("a fixture that exits zero reported a failure: %+v", out)
	}
	if strings.TrimSpace(out.Stdout) != "out" {
		t.Errorf("stdout = %q, want %q", out.Stdout, "out")
	}
	if strings.Contains(out.Stdout, "err") {
		t.Errorf("stdout = %q, want the host's stderr kept out of it", out.Stdout)
	}
}

func TestExecCommandReportsANonzeroExit(t *testing.T) {
	skipWithoutPOSIXShell(t)
	dir := t.TempDir()
	writeHostCLI(t, dir, "codex", `echo refused >&2; exit 1`)
	useHostCLIs(t, dir, "codex")

	out := execCommand(t.Context(), Command{Name: "codex", Args: []string{"plugin", "list"}})
	if !out.Failed {
		t.Errorf("a fixture that exits 1 reported success: %+v", out)
	}
}

func TestExecCommandReportsAnAbsentBinary(t *testing.T) {
	noHostCLIs(t)
	out := execCommand(t.Context(), Command{Name: "copilot", Args: []string{"plugin", "list"}})
	if !out.Failed {
		t.Errorf("a command whose binary is not on PATH reported success: %+v", out)
	}
}

// TestExecCommandKillsACommandPastItsTimeout proves the per-command bound is
// real: a host CLI that blocks on a prompt must cost the timeout, not the rest
// of the step.
func TestExecCommandKillsACommandPastItsTimeout(t *testing.T) {
	skipWithoutPOSIXShell(t)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `sleep 30`)
	useHostCLIs(t, dir, "claude")
	shrinkTimeouts(t, 50*time.Millisecond, time.Minute)

	started := time.Now()
	out := execCommand(t.Context(), Command{Name: "claude", Args: []string{"plugin", "update"}})
	elapsed := time.Since(started)

	if !out.Failed {
		t.Errorf("a killed command reported success: %+v", out)
	}
	if elapsed > 10*time.Second {
		t.Errorf("the command took %s, want the timeout to cut it", elapsed)
	}
}

// TestExecCommandRunsTheResolvedPath keeps the lookPath seam load-bearing: the
// resolved path is what runs, so a test that swaps the seam does not also have
// to rewrite PATH.
func TestExecCommandRunsTheResolvedPath(t *testing.T) {
	skipWithoutPOSIXShell(t)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `echo fixture`)
	useHostCLIs(t, dir, "claude")

	out := execCommand(t.Context(), Command{Name: "claude"})
	if strings.TrimSpace(out.Stdout) != "fixture" {
		t.Errorf("stdout = %q, want the fixture's answer", out.Stdout)
	}
}

// TestExecCommandReportsATruncatedStream joins the two halves of the ceiling.
// capWriter reports that it cut, readListing refuses a cut answer, and this is
// the wire between them: without it the seam could hand back a prefix marked
// whole, and a listing that lost its entries would read as "the plugin is not
// installed" — the verdict that silently skips the host.
func TestExecCommandReportsATruncatedStream(t *testing.T) {
	skipWithoutPOSIXShell(t)

	// A kibibyte per line, past the cap in a bounded number of shell builtins.
	const flood = `line=$(printf '%01023d' 0)
i=0
while [ $i -lt 1200 ]; do printf '%s\n' "$line" REDIRECT; i=$((i+1)); done`

	tests := []struct {
		name          string
		redirect      string
		wantTruncated bool
	}{
		{name: "on stdout", redirect: "", wantTruncated: true},
		// The regression this case exists for. Truncated once folded both
		// streams together, so a host writing a megabyte of progress noise to
		// stderr made a perfectly good listing on stdout read as "the plugin is
		// not installed" — readListing throws a truncated answer away, and that
		// verdict silently skips the host.
		{name: "on stderr, which nothing reads", redirect: ">&2", wantTruncated: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHostCLI(t, dir, "claude", strings.Replace(flood, "REDIRECT", tt.redirect, 1))
			useHostCLIs(t, dir, "claude")

			out := execCommand(t.Context(), Command{Name: "claude", Args: []string{"plugin", "list"}})
			if out.Truncated != tt.wantTruncated {
				t.Errorf("Truncated = %v, want %v (%d stdout bytes kept)",
					out.Truncated, tt.wantTruncated, len(out.Stdout))
			}
			if len(out.Stdout) > maxCommandOutput {
				t.Errorf("kept %d stdout bytes, want at most the cap %d",
					len(out.Stdout), maxCommandOutput)
			}
		})
	}
}

// TestCapWriterAcceptsExactlyTheCap is the off-by-one beside the case above.
// Truncated once tested the buffer length, so a stream of exactly
// maxCommandOutput bytes was marked truncated with nothing discarded — and
// readListing refused a listing that fit the ceiling precisely.
func TestCapWriterAcceptsExactlyTheCap(t *testing.T) {
	tests := []struct {
		name          string
		size          int
		wantTruncated bool
	}{
		{name: "one byte under the cap", size: maxCommandOutput - 1, wantTruncated: false},
		{name: "exactly the cap", size: maxCommandOutput, wantTruncated: false},
		{name: "one byte over the cap", size: maxCommandOutput + 1, wantTruncated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w capWriter
			n, err := w.Write(make([]byte, tt.size))
			if err != nil || n != tt.size {
				t.Fatalf("Write() = %d, %v; want %d, nil", n, err, tt.size)
			}
			if w.truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", w.truncated, tt.wantTruncated)
			}
			if want := min(tt.size, maxCommandOutput); len(w.buf) != want {
				t.Errorf("kept %d bytes, want %d", len(w.buf), want)
			}
		})
	}
}

// TestCapWriterTruncatesAcrossWrites covers the same boundary reached in pieces,
// which is how a real child writes.
func TestCapWriterTruncatesAcrossWrites(t *testing.T) {
	var w capWriter
	chunk := make([]byte, maxCommandOutput/2)

	for range 2 {
		if _, err := w.Write(chunk); err != nil {
			t.Fatalf("Write() error: %v", err)
		}
	}
	if w.truncated {
		t.Errorf("two half-cap writes filled the buffer exactly, want no truncation")
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !w.truncated {
		t.Error("a write past a full buffer was not reported as truncated")
	}
	if len(w.buf) != maxCommandOutput {
		t.Errorf("kept %d bytes, want the cap %d", len(w.buf), maxCommandOutput)
	}
}

// TestCapWriterStopsAtTheCap proves the ceiling holds and says so, which is
// what lets the listing parse refuse a truncated answer instead of parsing a
// prefix.
func TestCapWriterStopsAtTheCap(t *testing.T) {
	var w capWriter
	chunk := make([]byte, 64*1024)
	written := 0
	for written < maxCommandOutput+len(chunk) {
		n, err := w.Write(chunk)
		if err != nil || n != len(chunk) {
			t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(chunk))
		}
		written += n
	}
	if len(w.buf) != maxCommandOutput {
		t.Errorf("buffered %d bytes, want the cap %d", len(w.buf), maxCommandOutput)
	}
	if !w.truncated {
		t.Error("the writer reached the cap without reporting it truncated")
	}
}
