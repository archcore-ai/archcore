package cmd

// SessionStart dedup by session id (implemented — handleSessionStartDeduped
// in hooks_claude_code.go).
//
// Motivation: a project-level hook installed by `archcore init --agent`
// coexisting with a plugin-shipped hook fires twice for one SessionStart
// event, and both entries delegate to this binary — so the suppression lives
// here, protecting every plugin/CLI version combination. The dedup key folds
// in the event source (startup/resume/clear/compact) so a legitimate
// re-injection after a compact is not suppressed by the startup stamp, and
// the stamp window is short (sessionStampWindow) so genuinely later re-fires
// emit again.

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHandleSessionStart_DedupesBySessionID_Spec(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	// 1. First call for session "s1": full context emitted.
	first, err := handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("first call must emit output")
	}
	if !strings.Contains(string(first), `"hookEventName":"SessionStart"`) {
		t.Errorf("output missing SessionStart envelope:\n%s", first)
	}
	if !strings.Contains(string(first), "additionalContext") {
		t.Errorf("output missing additionalContext:\n%s", first)
	}

	// 2. Repeat for the SAME session: empty output, nil error — hosts must
	// never see a failing hook here, just silence.
	second, err := handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir)
	if err != nil {
		t.Fatalf("repeat call must not error: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("repeat call for same session must emit nothing, got:\n%s", second)
	}

	// 3. A DIFFERENT session emits again — dedup is per-session, not global.
	other, err := handleSessionStartDeduped(base, "v0.0.0-test", "s2", stampDir)
	if err != nil {
		t.Fatalf("other-session call: %v", err)
	}
	if len(other) == 0 {
		t.Error("different session id must emit output (stamp from s1 must not suppress s2)")
	}
}

func TestHandleSessionStart_EmptyKeyFailsOpen(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	for i := 1; i <= 2; i++ {
		out, err := handleSessionStartDeduped(base, "v0.0.0-test", "", stampDir)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if len(out) == 0 {
			t.Errorf("call %d with empty session id must emit (no dedup key, fail open)", i)
		}
	}
}

func TestHandleSessionStart_UnwritableStampDirFailsOpen(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	// A stamp dir under a read-only parent: MkdirAll fails, dedup disengages.
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	stampDir := filepath.Join(parent, "stamps")

	for i := 1; i <= 2; i++ {
		out, err := handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir)
		if err != nil {
			t.Fatalf("call %d must stay exit-0 with unwritable stamp dir: %v", i, err)
		}
		if len(out) == 0 {
			t.Errorf("call %d must emit — dedup is best-effort and fails open", i)
		}
	}
}

func TestHandleSessionStart_ExpiredStampReEmits(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	if _, err := handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir); err != nil {
		t.Fatal(err)
	}
	// Age the stamp beyond the window: the next call must emit again. The
	// on-disk stamp key is the host key scoped by project root.
	stamp := sessionStampPath(stampDir, "s1\x00"+base)
	old := time.Now().Add(-sessionStampWindow - time.Minute)
	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatalf("aging stamp: %v", err)
	}

	out, err := handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("expired stamp must not suppress emission")
	}
}

// The motivating double-fire (project-level hook + plugin hook for one
// SessionStart event) is near-simultaneous: both processes run concurrently.
// The stamp claim must be atomic — exactly one of N parallel invocations for
// the same key may emit.
func TestHandleSessionStart_ConcurrentDoubleFire_ExactlyOneEmits(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	const n = 8
	outs := make([][]byte, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			outs[i], errs[i] = handleSessionStartDeduped(base, "v0.0.0-test", "s1", stampDir)
		})
	}
	close(start)
	wg.Wait()

	emitted := 0
	for i := range n {
		if errs[i] != nil {
			t.Errorf("call %d errored: %v", i, errs[i])
		}
		if len(outs[i]) > 0 {
			emitted++
		}
	}
	if emitted != 1 {
		t.Errorf("exactly one concurrent invocation must emit, got %d of %d", emitted, n)
	}
}

// One host session can touch two projects within the window (multi-root
// workspace, workspace switch reusing the conversation id). The stamp key is
// scoped by project root, so project A's stamp must never suppress project B.
func TestHandleSessionStart_SameSessionDifferentProjectsBothEmit(t *testing.T) {
	t.Parallel()
	baseA := setupArchcoreDir(t)
	baseB := setupArchcoreDir(t)
	stampDir := t.TempDir()

	outA, err := handleSessionStartDeduped(baseA, "v0.0.0-test", "s1", stampDir)
	if err != nil {
		t.Fatalf("project A: %v", err)
	}
	if len(outA) == 0 {
		t.Fatal("project A must emit")
	}

	outB, err := handleSessionStartDeduped(baseB, "v0.0.0-test", "s1", stampDir)
	if err != nil {
		t.Fatalf("project B: %v", err)
	}
	if len(outB) == 0 {
		t.Error("same session id in a different project must emit (stamps are project-scoped)")
	}
}

func TestHookInput_DedupKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   hookInput
		want string
	}{
		{"claude/codex session_id", hookInput{SessionID: "abc", Source: "startup"}, "abc\x00startup"},
		{"cursor conversation_id", hookInput{ConversationID: "conv1"}, "conv1\x00"},
		{"session_id wins over conversation_id", hookInput{SessionID: "abc", ConversationID: "conv1"}, "abc\x00"},
		{"no id at all fails open", hookInput{Source: "startup"}, ""},
		{"source differentiates compact re-fire", hookInput{SessionID: "abc", Source: "compact"}, "abc\x00compact"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.in.dedupKey(); got != tt.want {
				t.Errorf("dedupKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
