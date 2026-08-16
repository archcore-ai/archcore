package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewServer_HasTools(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestBuildInstructions_DefaultEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"", "en"} {
		result := buildInstructions(lang, nil)
		if result != mcpServerInstructions {
			t.Errorf("buildInstructions(%q): expected base instructions unchanged", lang)
		}
		if strings.Contains(result, "LANGUAGE REQUIREMENT") {
			t.Errorf("buildInstructions(%q): should not contain LANGUAGE REQUIREMENT", lang)
		}
	}
}

func TestBuildInstructions_NonEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"ru", "ja", "de"} {
		result := buildInstructions(lang, nil)
		if !strings.HasPrefix(result, mcpServerInstructions) {
			t.Errorf("buildInstructions(%q): should start with base instructions", lang)
		}
		if !strings.Contains(result, "LANGUAGE REQUIREMENT") {
			t.Errorf("buildInstructions(%q): should contain LANGUAGE REQUIREMENT", lang)
		}
		if !strings.Contains(result, lang) {
			t.Errorf("buildInstructions(%q): should contain the language code", lang)
		}
	}
}

func TestNewServer_WithLanguageSetting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(base, ".archcore", "settings.json"),
		[]byte(`{"sync":"none","language":"ru"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

// TestBuildInstructions_TrackSectionsRemoved guards the layer boundary from
// both sides. Track orchestration moved to the plugin, so the instructions must
// not describe it — but the cut sat next to sections that carry knowledge about
// document TYPES, which stays here. Asserting only the removal would let a
// careless edit take REQUIREMENTS LAYERS with it and no test would notice;
// asserting only the survivors would let the track prose creep back.
func TestBuildInstructions_TrackSectionsRemoved(t *testing.T) {
	t.Parallel()

	// Track orchestration and the prompt surface that drove it.
	removed := []string{
		"REQUIREMENTS TRACKS",
		"RESEARCH GATE",
		"WORKFLOW PROMPTS",
		"iso_track",
		"sources_track",
		"product_track",
		"standard_track",
		"architecture_track",
	}
	// Type knowledge, relation conventions, and status semantics stay.
	kept := []string{
		"TYPE SELECTION RULES",
		"REQUIREMENTS LAYERS",
		"DOCUMENT RELATIONS",
		"VALID STATUS VALUES",
		"TAGS:",
		// The rnd verdict mapping outlived the section it used to live in.
		"first-class outcome",
	}

	for _, lang := range []string{"", "en", "ru"} {
		result := buildInstructions(lang, nil)
		for _, s := range removed {
			if strings.Contains(result, s) {
				t.Errorf("buildInstructions(%q): still contains removed marker %q", lang, s)
			}
		}
		for _, s := range kept {
			if !strings.Contains(result, s) {
				t.Errorf("buildInstructions(%q): lost surviving section %q", lang, s)
			}
		}
	}
}

func TestNewServer_MissingSettings_FallsBack(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// No .archcore/settings.json — server should still create successfully.
	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

// runStdioTimeout bounds every wait in the RunStdio tests. They drive a real
// server loop, so a regression must surface as a failure, never as a suite that
// hangs until the CI job is killed.
const runStdioTimeout = 10 * time.Second

// The three tests below drive RunStdio in this process, which swaps fd 1
// process-wide for the duration. None of them may call t.Parallel(): Go releases
// parallel tests only after the serial ones finish, and that ordering is what
// keeps the swap from landing under another test's output.

// The background task must receive the session's own context. The trigger's
// cancellation story — a host that closes stdio before the delay elapses gets no
// update attempt at all — is unenforceable if RunStdio hands the task a detached
// context: the task would outlive the session and reach the policy anyway.
// mcp-background-update.spec §9.
func TestRunStdio_BackgroundTaskInheritsSessionContext(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskCtx := make(chan context.Context, 1)
	served := make(chan error, 1)
	go func() {
		served <- RunStdio(ctx, dir, "test",
			func(c context.Context) { taskCtx <- c })
	}()

	var got context.Context
	select {
	case got = <-taskCtx:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio never started the background task")
	}

	cancel()

	select {
	case <-got.Done():
	case <-time.After(runStdioTimeout):
		t.Fatal("cancelling RunStdio's context left the task's context live")
	}
	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return after its context was cancelled")
	}
}

// Every existing caller and embedder passes no background task, so a nil task
// must be skipped rather than invoked — a nil call would take the whole process
// down with the session. The byte-level half of "serves exactly as today" is
// TestRunStdio_StdoutIdenticalWithAndWithoutBackgroundTask.
func TestRunStdio_WithoutABackgroundTask_ReturnsWithoutPanicking(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	served := make(chan error, 1)
	go func() { served <- RunStdio(ctx, dir, "test", nil) }()

	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return under a cancelled context")
	}
}

// RunStdio delegates option application to NewServer, so each option runs once.
// The hazard this pins is the shape RunStdio used to have: it applied the
// options itself to reach a field NewServer ignored, and the obvious repair —
// keep the NewServer call and apply them again — would run every
// side-effecting option twice.
func TestRunStdio_AppliesEachOptionOnce(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	applied := make(chan struct{}, 8)
	count := ServerOption(func(*serverConfig) { applied <- struct{}{} })

	served := make(chan error, 1)
	go func() { served <- RunStdio(ctx, dir, "test", nil, count, count) }()

	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return under a cancelled context")
	}
	if got := len(applied); got != 2 {
		t.Errorf("two options passed: applied %d times, want 2", got)
	}
}

// TestRunStdio_CancelsTheBackgroundTaskWhenItReturns closes the lifetime gap.
//
// Listen returns on stdin EOF as well as on cancellation, and EOF is how a host
// normally ends a session. Before RunStdio derived the task's context, a task
// started under a still-live parent context kept running after RunStdio
// returned and restore() closed the protocol stream — with a context that still
// reported itself live. `archcore mcp` exits immediately so nothing showed, but
// the in-process tests and any embedder kept the goroutine.
//
// The task is cancelled, never joined: mcp-background-update.spec §10 says a
// session must not wait on an attempt.
func TestRunStdio_CancelsTheBackgroundTaskWhenItReturns(t *testing.T) {
	dir := t.TempDir()

	taskCtx := make(chan context.Context, 1)
	served := make(chan error, 1)
	go func() {
		// stdin is already at EOF under `go test`, so Listen returns on its own
		// with the parent context still live — the exact shape that leaked.
		served <- RunStdio(context.Background(), dir, "test",
			func(c context.Context) { taskCtx <- c })
	}()

	var got context.Context
	select {
	case got = <-taskCtx:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio never started the background task")
	}
	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return on stdin EOF")
	}

	select {
	case <-got.Done():
	case <-time.After(runStdioTimeout):
		t.Error("RunStdio returned but left the background task's context live")
	}
}
