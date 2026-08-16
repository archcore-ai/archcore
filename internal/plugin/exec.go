package plugin

import (
	"context"
	"os/exec"
	"time"
)

// The subprocess seam of the plugin surface. It copies the shape of
// @internal/git/git.go — a lookPath seam, exec.CommandContext, a WaitDelay so a
// grandchild holding the pipes cannot outlive the deadline, and a capped writer
// — and differs from it in two deliberate ways.
//
// git discards stderr because its stderr embeds absolute paths and every caller
// there has a fallback. This surface discards it for a stronger reason: nothing
// may ever print it. A host CLI's stderr routinely carries a path or a token,
// and both specs answer a failed command with the command line instead — so a
// captured stderr is not a reader that has yet to be written, it is output the
// specs forbid surfacing. readListing parses Stdout alone and CommandFailed
// reports the command line, so the buffer had no reachable consumer.
//
// Leaving cmd.Stderr nil routes the child's stderr to the null device, which
// cannot block however much the host writes; the capped writer was never what
// kept the child moving.
//
// git's timeouts are constants because a hook budget is fixed. These two are
// package-level vars because a test has to reach them: proving that the step
// bound skips the remaining hosts means expiring it in milliseconds, and no
// test may spend the real budget to do it.

// commandTimeout bounds one host command. It protects the per-host share of the
// step budget, so a host CLI that blocks on a prompt or stalls on the network
// cannot spend the whole step alone — updating-the-plugin.spec, Surface.
var commandTimeout = 30 * time.Second

// stepTimeout bounds the whole plugin step. It protects the latency budget the
// step is allowed inside `archcore update` and `archcore init`; once it
// elapses the remaining hosts are skipped in silence rather than delaying the
// command that hosts the step — updating-the-plugin.spec, Surface.
var stepTimeout = 120 * time.Second

// commandWaitDelay bounds how long Wait blocks on inherited pipes after the
// context is cancelled. It protects the command timeout itself: without it a
// grandchild holding a pipe open outlives the deadline that killed its parent,
// and the bound the specs state stops being a bound.
const commandWaitDelay = 100 * time.Millisecond

// maxCommandOutput bounds what one host command may hand back on stdout. It
// protects the memory of a process that does not choose how much a host CLI
// prints. Output that exceeds the cap is reported as truncated, and the listing
// parse counts a truncated answer as no answer rather than parsing a prefix, so
// an oversized stream fails instead of silently shortening —
// bounded-and-deterministic-output.rule.
const maxCommandOutput = 1 << 20

// lookPath is the seam tests swap to place a host CLI on PATH or take it off.
// The resolved path is what the command actually runs, so swapping this one
// variable redirects both the evidence question and the execution.
//
// On Windows it answers only for an extension %PATHEXT% lists, so a host
// shipped as a PowerShell script or as a shim outside that set reads as absent
// even while the user has it. CI runs Linux only, so that branch ships
// unexercised — the fixture host CLIs are shell scripts and the tests that use
// them skip. It is left as it is because the way it fails is the way the whole
// surface fails: an unresolved binary is no evidence, and no evidence prints
// the exact command instead of running one. The cost is a printed line on a
// machine that could have acted; the alternative — guessing at extensions —
// risks running the wrong program.
var lookPath = exec.LookPath

// runCommand is the seam every subprocess of this package goes through. Neither
// the collector nor the executor calls exec directly, so one test double counts
// the subprocesses a path starts — which is how PrintOnly is proved to run none
// rather than merely to print the same lines.
var runCommand = execCommand

// commandOutcome is what one host command produced. Failed collapses the three
// ways a command can fail — it never started, it exited nonzero, a deadline
// killed it — because every caller treats them alike: both specs answer all
// three with the exact command line and a move to the next host, and none of
// them with a parse.
type commandOutcome struct {
	Stdout string
	// Truncated reports that stdout exceeded maxCommandOutput and bytes were
	// dropped. It tracks stdout alone: stderr is not captured, and folding a
	// second stream in here once meant a host writing a megabyte of progress
	// noise to stderr made a perfectly good listing on stdout read as "the
	// plugin is not installed" — the silent verdict that skips the host.
	Truncated bool
	Failed    bool
}

// execCommand runs one host command bounded by commandTimeout and returns its
// captured streams.
//
// It returns no error on purpose. A failed host command is data on this surface,
// not an exception: every tier of both specs answers one with an output line and
// a move to the next host, so an error return would only be unwrapped back into
// this same struct at the single call site that reads it.
func execCommand(ctx context.Context, c Command) commandOutcome {
	if c.Name == "" {
		return commandOutcome{Failed: true}
	}
	path, err := lookPath(c.Name)
	if err != nil {
		return commandOutcome{Failed: true}
	}

	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, c.Args...)
	cmd.WaitDelay = commandWaitDelay

	var stdout capWriter
	cmd.Stdout = &stdout
	// cmd.Stderr stays nil: os/exec connects it to the null device, which no
	// amount of host output can block on.

	runErr := cmd.Run()
	return commandOutcome{
		Stdout:    string(stdout.buf),
		Truncated: stdout.truncated,
		Failed:    runErr != nil,
	}
}

// capWriter keeps at most maxCommandOutput bytes and discards the rest. Write
// never fails, so an oversized stream costs the child nothing: the reader marks
// the result truncated and the caller decides what a truncated answer means.
type capWriter struct {
	buf       []byte
	truncated bool
}

// Write is truncated only when bytes were actually dropped. Testing the buffer
// length instead marked a stream of exactly maxCommandOutput bytes truncated
// with nothing discarded, and readListing throws a truncated answer away — so a
// listing that fit the cap precisely was refused intact.
func (w *capWriter) Write(p []byte) (int, error) {
	room := maxCommandOutput - len(w.buf)
	if room > 0 {
		w.buf = append(w.buf, p[:min(room, len(p))]...)
	}
	if len(p) > max(room, 0) {
		w.truncated = true
	}
	return len(p), nil
}
