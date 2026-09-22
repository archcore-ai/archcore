package cmd

// SessionStart dedup by session id (handleSessionStartDeduped in
// hook_session_start.go).
//
// Motivation: a project-level hook installed by `archcore init --agent`
// coexisting with a plugin-shipped hook fires twice for one SessionStart
// event, and both entries delegate to this binary — so the suppression lives
// here, protecting every plugin/CLI version combination. The key folds in the
// event source (startup/resume/clear/compact) so a legitimate re-injection
// after a compact is not suppressed by the startup stamp, and the host id so
// two leaves reading the same stdin do not suppress each other.

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
	first, banner, emitted := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir)
	if !emitted {
		t.Fatal("first call must emit")
	}
	if !strings.Contains(first, "[Archcore") {
		t.Errorf("output missing the session context:\n%s", first)
	}
	if banner == "" {
		t.Error("first call must carry the connected banner")
	}

	// 2. Repeat for the SAME session: nothing emitted, and no error — hosts must
	// never see a failing hook here, just silence.
	if _, _, again := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir); again {
		t.Error("repeat call for the same session must emit nothing")
	}

	// 3. A DIFFERENT session emits again — dedup is per-session, not global.
	if _, _, other := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s2", stampDir); !other {
		t.Error("a different session id must emit (s1's stamp must not suppress s2)")
	}
}

func TestHandleSessionStart_EmptyKeyFailsOpen(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	for i := 1; i <= 2; i++ {
		if _, _, emitted := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "", stampDir); !emitted {
			t.Errorf("call %d with an empty session id must emit (no dedup key, fail open)", i)
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
		if _, _, emitted := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir); !emitted {
			t.Errorf("call %d must emit — dedup is best-effort and fails open", i)
		}
	}
}

func TestHandleSessionStart_ExpiredStampReEmits(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()

	if _, _, emitted := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir); !emitted {
		t.Fatal("first call must emit")
	}
	// Age the stamp beyond the window: the next call must emit again. The
	// on-disk stamp key is the host key scoped by project root.
	stamp := sessionStampPath(stampDir, "s1\x00"+base)
	old := time.Now().Add(-sessionStampWindow - time.Minute)
	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatalf("aging stamp: %v", err)
	}

	if _, _, emitted := handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir); !emitted {
		t.Error("an expired stamp must not suppress emission")
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
	results := make([]bool, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			_, _, results[i] = handleSessionStartDeduped(bg(), base, "v0.0.0-test", "s1", stampDir)
		})
	}
	close(start)
	wg.Wait()

	emitted := 0
	for _, ok := range results {
		if ok {
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

	if _, _, emitted := handleSessionStartDeduped(bg(), baseA, "v0.0.0-test", "s1", stampDir); !emitted {
		t.Fatal("project A must emit")
	}
	if _, _, emitted := handleSessionStartDeduped(bg(), baseB, "v0.0.0-test", "s1", stampDir); !emitted {
		t.Error("the same session id in a different project must emit (stamps are project-scoped)")
	}
}

func TestDeriveDedupKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		host    string
		payload string
		want    string
	}{
		{name: "claude/codex session_id", host: "claude-code", payload: `{"session_id":"abc","source":"startup"}`, want: "abc\x00startup\x00claude-code"},
		{name: "cursor conversation_id", host: "cursor", payload: `{"conversation_id":"conv1"}`, want: "conv1\x00\x00cursor"},
		{name: "session_id wins over conversation_id", host: "claude-code", payload: `{"session_id":"abc","conversation_id":"conv1"}`, want: "abc\x00\x00claude-code"},
		{name: "no id at all fails open", host: "claude-code", payload: `{"source":"startup"}`, want: ""},
		{name: "source differentiates compact re-fire", host: "claude-code", payload: `{"session_id":"abc","source":"compact"}`, want: "abc\x00compact\x00claude-code"},
		{name: "unparsable payload fails open", host: "claude-code", payload: `not json`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deriveDedupKey(reqFor(t, tt.host, "", tt.payload)); got != tt.want {
				t.Errorf("deriveDedupKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDeriveDedupKey_HostSeparatesTheLeaves is the C3 contract: Copilot reads
// .claude/settings.json as well as its own config, so one session start can run
// two leaves with byte-identical stdin. Without the host in the key they race
// for one stamp and the loser stays silent — and because each speaks a different
// dialect, the wrong winner leaves the session with no readable context at all.
func TestDeriveDedupKey_HostSeparatesTheLeaves(t *testing.T) {
	t.Parallel()
	const payload = `{"session_id":"s1","source":"startup"}`

	copilot := deriveDedupKey(reqFor(t, "copilot", "/p", payload))
	claude := deriveDedupKey(reqFor(t, "claude-code", "/p", payload))

	if copilot == claude {
		t.Errorf("both hosts derived the same dedup key %q; one would suppress the other", copilot)
	}
}

// TestHandleSessionStart_BothHostsEmitForOneSession is the same contract one
// layer down: with per-host keys each leaf emits in its own dialect.
func TestHandleSessionStart_BothHostsEmitForOneSession(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	stampDir := t.TempDir()
	const payload = `{"session_id":"s1","source":"startup"}`

	copilotKey := deriveDedupKey(reqFor(t, "copilot", base, payload))
	claudeKey := deriveDedupKey(reqFor(t, "claude-code", base, payload))

	if _, _, emitted := handleSessionStartDeduped(bg(), base, "v", copilotKey, stampDir); !emitted {
		t.Fatal("the copilot leaf must emit")
	}
	if _, _, emitted := handleSessionStartDeduped(bg(), base, "v", claudeKey, stampDir); !emitted {
		t.Error("the claude-code leaf was suppressed by the copilot stamp")
	}
}
