package cmd

import (
	"context"
	"time"

	"archcore-cli/internal/display"
	"archcore-cli/internal/stamp"
)

// sessionStampWindow bounds both suppression and stamp retention: a stamp
// fresher than the window suppresses re-emission; anything older is swept. The
// double-fire this defends against happens within one event (seconds), so a
// short window dedupes it while letting genuinely later re-fires emit again.
const sessionStampWindow = 10 * time.Minute

// SessionStart dedup.
//
// A session can receive the same context twice: a project-level hook installed
// by `archcore init --agent` coexisting with a plugin-shipped hook fires both
// entries for one event, and both delegate to this binary. The stamp makes the
// second invocation for the same session emit nothing, whichever plugin and CLI
// versions are in play, because the suppression lives here in the shared binary.

// handleSessionStartDeduped builds the session context unless an equivalent
// invocation already claimed this window.
//
// The host-derived key is scoped to the project by folding in baseDir — one
// session touching two projects within the window must get both contexts. An
// empty key or stampDir fails open (always emit); a suppressed repeat reports
// emitted=false, which the caller turns into silence rather than an error.
func handleSessionStartDeduped(ctx context.Context, baseDir, version, key, stampDir string) (sessionContext, banner string, emitted bool) {
	if key != "" && stampDir != "" {
		key += "\x00" + baseDir
		if !stamp.Claim(stampDir, key, sessionStampWindow) {
			return "", "", false
		}
	}
	text, docCount := buildSessionContext(ctx, baseDir)
	return text, display.HookConnectedLine(version, docCount), true
}

// defaultSessionStampDir is where the session scope keeps its stamps. Each scope
// owns a directory: a sweep expires everything older than its own window, so a
// shared one would let the 10-minute session scope erase the 24-hour staleness
// budget.
func defaultSessionStampDir() string { return stamp.DirFor("session-stamps") }

// sessionStampPath is where one session's stamp lives.
func sessionStampPath(stampDir, key string) string { return stamp.PathFor(stampDir, key) }
