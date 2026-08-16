package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/config"
	"archcore-cli/internal/update"
)

// stubUnattended swaps the trigger's two package seams for the duration of one
// test: the delay so a case costs milliseconds instead of a minute, and the
// policy so nothing claims a stamp, reaches github.com or replaces a binary.
// The returned pointer holds every UnattendedOptions the trigger passed.
//
// It mutates package state, so a test that calls it must not call t.Parallel().
func stubUnattended(t *testing.T, delay time.Duration, res update.UnattendedResult) *[]update.UnattendedOptions {
	t.Helper()

	oldDelay, oldRun := backgroundUpdateDelay, runUnattendedUpdate
	t.Cleanup(func() { backgroundUpdateDelay, runUnattendedUpdate = oldDelay, oldRun })

	var calls []update.UnattendedOptions
	backgroundUpdateDelay = delay
	runUnattendedUpdate = func(_ context.Context, opts update.UnattendedOptions) update.UnattendedResult {
		calls = append(calls, opts)
		return res
	}
	return &calls
}

// The value the binary ships with, which no other test can see: every case here
// shrinks the delay through the same package variable and restores it, and a
// seam a test left set reads exactly like a deliberate edit to the default. At
// zero the attempt contends with the host's initialize round trip — the one
// phase of a session where latency is visible — and a session shorter than the
// delay stops being the no-op the spec promises (mcp-background-update.spec §3
// and its no-zero-delay constraint).
//
// Not parallel: it reads state the tests around it mutate.
func TestBackgroundUpdateDelay_IsSixtySeconds(t *testing.T) {
	if backgroundUpdateDelay != 60*time.Second {
		t.Errorf("backgroundUpdateDelay = %s, want 60s", backgroundUpdateDelay)
	}
}

// One run of the task is one attempt. The other half of "at most once per server
// process" — that RunStdio starts the task exactly once — belongs to internal/mcp.
func TestBackgroundUpdateTask_RunsThePolicyOnce(t *testing.T) {
	calls := stubUnattended(t, time.Millisecond, update.UnattendedResult{})

	stdout, _ := captureOutput(t, func() {
		backgroundUpdateTask("v1.0.0")(context.Background())
	})

	if len(*calls) != 1 {
		t.Fatalf("policy invocations = %d, want 1", len(*calls))
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty — fd 1 is the JSON-RPC frame stream", stdout)
	}
}

// The version the policy compares and reports must be the cleaned one. root.go
// hands `mcp` the RAW version (every other command is constructed with the
// cleaned one), so a "+"-decorated build would reach NewerSemver unparseable and
// refuse silently, forever — the failure this test exists to catch.
func TestBackgroundUpdateTask_PassesTheCleanedVersionToThePolicy(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"decorated release refuses semver comparison uncleaned", "1.2.3+dirty", "v1.2.3"},
		{"unprefixed tag", "1.2.3", "v1.2.3"},
		{"already cleaned tag is unchanged", "v1.2.3", "v1.2.3"},
		{"dev survives so the policy still refuses a development build", "dev", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := stubUnattended(t, time.Millisecond, update.UnattendedResult{})

			backgroundUpdateTask(tt.raw)(context.Background())

			if len(*calls) != 1 {
				t.Fatalf("policy invocations = %d, want 1", len(*calls))
			}
			opts := (*calls)[0]
			if opts.Version != tt.want {
				t.Errorf("opts.Version = %q, want %q (raw %q)", opts.Version, tt.want, tt.raw)
			}
			// The updater carries the same spelling: it is what CheckLatest
			// compares against and what the archive URL is built from.
			if opts.Updater == nil {
				t.Fatal("opts.Updater = nil, want the updater updateDeps builds")
			}
			if opts.Updater.CurrentVersion != tt.want {
				t.Errorf("opts.Updater.CurrentVersion = %q, want %q (raw %q)",
					opts.Updater.CurrentVersion, tt.want, tt.raw)
			}
			// The sender too. Capture is nil-safe, so an unwired one is silent
			// rather than fatal: the unattended path would replace the binary
			// and report nothing, and no test would fail. It is the same
			// failure TestNewUpdateCmd_ReportsThroughTheWiredSender catches on
			// the manual path.
			if opts.Telemetry == nil {
				t.Error("opts.Telemetry = nil, want the sender updateDeps builds — an unattended update would report nothing")
			}
		})
	}
}

// A session that ends inside the delay spends no claim stamp and says nothing.
func TestBackgroundUpdateTask_CancelledContextSkipsThePolicy(t *testing.T) {
	// An hour, so the cancelled context is the only ready case in the select.
	calls := stubUnattended(t, time.Hour, update.UnattendedResult{Updated: true, NewVersion: "v9.9.9"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, stderr := captureOutput(t, func() {
		backgroundUpdateTask("v1.0.0")(ctx)
	})

	if len(*calls) != 0 {
		t.Errorf("policy invocations = %d, want 0", len(*calls))
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
}

func TestBackgroundUpdateTask_ReplacementWritesOneStderrLine(t *testing.T) {
	stubUnattended(t, time.Millisecond, update.UnattendedResult{Updated: true, NewVersion: "v9.9.9"})

	stdout, stderr := captureOutput(t, func() {
		backgroundUpdateTask("v1.0.0")(context.Background())
	})

	if stdout != "" {
		t.Errorf("stdout = %q, want empty — fd 1 is the JSON-RPC frame stream", stdout)
	}
	if !strings.HasSuffix(stderr, "\n") {
		t.Fatalf("stderr = %q, want one terminated line", stderr)
	}
	if lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n"); len(lines) != 1 {
		t.Fatalf("stderr lines = %d, want exactly 1: %q", len(lines), stderr)
	}
	if !strings.Contains(stderr, "v9.9.9") {
		t.Errorf("stderr = %q, want it to name the new version v9.9.9", stderr)
	}
}

// mcpTriggerTimeout bounds the wait for an attempt that a correctly wired
// command starts within a millisecond. It is a ceiling on a failing run, not a
// budget: the passing path never spends it.
const mcpTriggerTimeout = 10 * time.Second

// The trigger only exists if `archcore mcp` wires it. Every test above drives
// the closure directly, so a build that dropped the option from the command
// would serve documents perfectly, pass the whole suite, and never attempt an
// update — mcp-background-update.spec §1. It is the same class of failure the
// release gate assert-not-inert.sh covers on the ldflags side, and nothing else
// here can see it.
func TestMCPCmd_StartsTheBackgroundUpdateTrigger(t *testing.T) {
	// No env slip may reach the developer's real state directory: the policy is
	// stubbed, but the updater and the telemetry client the command builds are
	// the real ones.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}

	// A channel rather than stubUnattended's slice: the attempt runs on its own
	// goroutine here, so the recording has to synchronise with the assertion.
	attempted := make(chan update.UnattendedOptions, 1)
	oldDelay, oldRun := backgroundUpdateDelay, runUnattendedUpdate
	t.Cleanup(func() { backgroundUpdateDelay, runUnattendedUpdate = oldDelay, oldRun })
	backgroundUpdateDelay = time.Millisecond
	runUnattendedUpdate = func(_ context.Context, opts update.UnattendedOptions) update.UnattendedResult {
		attempted <- opts
		return update.UnattendedResult{}
	}

	// The session stays open until the attempt has been observed. RunStdio ends
	// the task's context when it returns, which is what §9 and the conformance
	// clause ask for — a session that closes before the delay elapses makes no
	// attempt — so a test that EOFs stdin first would be racing the very
	// cancellation the design specifies, and could only pass on a leak.
	stdin := withOpenStdin(t)

	cmd := newMCPCmd("v1.2.3")
	cmd.SetArgs([]string{"--project", dir})

	// Bounded like the RunStdio tests in internal/mcp: this drives a real server
	// loop, and a regression must fail the run rather than hang it holding the
	// process-wide stdout swap open.
	var execErr error
	captureOutput(t, func() {
		served := make(chan error, 1)
		go func() { served <- cmd.Execute() }()

		select {
		case opts := <-attempted:
			// The command hands the trigger its own version, not the server's.
			if opts.Version != "v1.2.3" {
				t.Errorf("opts.Version = %q, want %q", opts.Version, "v1.2.3")
			}
		case <-time.After(mcpTriggerTimeout):
			t.Error("`archcore mcp` served a session without starting an update attempt — mcp-background-update.spec §1")
		}

		// Only now does the session end, the way a host closing stdio ends one.
		_ = stdin.Close()
		select {
		case execErr = <-served:
		case <-time.After(mcpTriggerTimeout):
			t.Fatal("the mcp session did not return on stdin EOF")
		}
	})
	if execErr != nil {
		t.Fatalf("mcp session: %v", execErr)
	}
}

// The other half of the same wiring, and the constraint the spec states
// outright: "a session shorter than the delay produces no attempt". One-shot
// agent runs and host capability probes take this path on every machine, and a
// trigger that outlived its session would update binaries on all of them —
// mcp-background-update.spec §9 and its conformance clause.
func TestMCPCmd_ASessionShorterThanTheDelayStartsNoAttempt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}

	attempted := make(chan update.UnattendedOptions, 1)
	oldDelay, oldRun := backgroundUpdateDelay, runUnattendedUpdate
	t.Cleanup(func() { backgroundUpdateDelay, runUnattendedUpdate = oldDelay, oldRun })
	// Long enough that the session below cannot outlive it.
	backgroundUpdateDelay = time.Hour
	runUnattendedUpdate = func(_ context.Context, opts update.UnattendedOptions) update.UnattendedResult {
		attempted <- opts
		return update.UnattendedResult{}
	}

	withStdin(t, "")
	cmd := newMCPCmd("v1.2.3")
	cmd.SetArgs([]string{"--project", dir})

	var execErr error
	captureOutput(t, func() {
		served := make(chan error, 1)
		go func() { served <- cmd.Execute() }()
		select {
		case execErr = <-served:
		case <-time.After(mcpTriggerTimeout):
			t.Fatal("the mcp session did not return on stdin EOF")
		}
	})
	if execErr != nil {
		t.Fatalf("mcp session: %v", execErr)
	}

	// The goroutine is not joined, so give a leaked one room to fire.
	select {
	case opts := <-attempted:
		t.Errorf("a session that ended before the delay invoked the policy with %+v", opts)
	case <-time.After(100 * time.Millisecond):
	}
}

// A project whose declared global is missing never starts a server, so it never
// starts an attempt either — mcp-background-update.spec, Failure Behavior 5.
// The guard is positional: checkGlobals returns before RunStdio is reached, and
// a trigger moved ahead of it (into RunE, or into the option list built before
// the checks) would update binaries on machines whose session refused to run.
func TestMCPCmd_BrokenGlobalMountStartsNoUpdateAttempt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	writeMCPSettings(t, dir, `{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)

	// Zero delay: an attempt that the command wrongly started has nothing left
	// to wait for, so the grace period below is measuring scheduling and not a
	// timer.
	attempted := make(chan update.UnattendedOptions, 1)
	oldDelay, oldRun := backgroundUpdateDelay, runUnattendedUpdate
	t.Cleanup(func() { backgroundUpdateDelay, runUnattendedUpdate = oldDelay, oldRun })
	backgroundUpdateDelay = 0
	runUnattendedUpdate = func(_ context.Context, opts update.UnattendedOptions) update.UnattendedResult {
		attempted <- opts
		return update.UnattendedResult{}
	}

	withStdin(t, "")
	cmd := newMCPCmd("v1.2.3")
	cmd.SetArgs([]string{"--project", dir})

	var execErr error
	captureOutput(t, func() { execErr = cmd.Execute() })
	if execErr == nil {
		t.Fatal("the command served a project with a missing global mount")
	}

	select {
	case <-attempted:
		t.Fatal("an update attempt started for a session that never began — mcp-background-update.spec, Failure Behavior 5")
	case <-time.After(noAttemptGrace):
	}
}

// noAttemptGrace bounds a negative check: long enough for a wrongly started
// goroutine with a zero delay to reach the stub, short enough to keep a passing
// run cheap.
const noAttemptGrace = 250 * time.Millisecond

// Every outcome other than a completed replacement is silent: the policy has
// already recorded it in telemetry, and the host's log gets nothing.
func TestBackgroundUpdateTask_QuietWhenNothingWasReplaced(t *testing.T) {
	stubUnattended(t, time.Millisecond, update.UnattendedResult{})

	stdout, stderr := captureOutput(t, func() {
		backgroundUpdateTask("v1.0.0")(context.Background())
	})

	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
}
