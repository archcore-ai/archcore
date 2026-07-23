package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"archcore-cli/internal/display"

	"github.com/spf13/cobra"
)

// hookInput represents the JSON payload the host sends via stdin. cwd
// resolves the project; session_id (Claude Code, Codex) / conversation_id
// (Cursor) and source key the SessionStart dedup. Other fields are ignored.
type hookInput struct {
	CWD            string `json:"cwd"`
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	Source         string `json:"source"`
}

// dedupKey derives the SessionStart dedup key. session_id wins (Claude Code,
// Codex); Cursor sends conversation_id instead. The event source
// (startup/resume/clear/compact on Claude Code) is folded in so a legitimate
// re-injection after e.g. a compact — where the earlier context was
// summarized away — is NOT suppressed by the startup stamp. Empty when the
// host sent no id: dedup then fails open (always emit).
func (in *hookInput) dedupKey() string {
	id := in.SessionID
	if id == "" {
		id = in.ConversationID
	}
	if id == "" {
		return ""
	}
	return id + "\x00" + in.Source
}

// hookOutput is the JSON response written to stdout.
type hookOutput struct {
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
	SystemMessage      string         `json:"systemMessage,omitempty"`
}

// resolveBaseDir returns the base directory from hook input, falling back to cwd.
func resolveBaseDir(input *hookInput) (string, error) {
	if input.CWD != "" {
		return input.CWD, nil
	}
	return os.Getwd()
}

// readHookInput parses the hook input JSON from a reader.
func readHookInput(r io.Reader) (*hookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}
	return &input, nil
}

func newHooksClaudeCodeCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-code",
		Short: "Handle Claude Code hook events",
	}
	cmd.AddCommand(
		newSessionStartCmd(version),
	)
	return cmd
}

func newSessionStartCmd(version string) *cobra.Command {
	return newSessionStartHookCmd("session-start", "Handle SessionStart hook event", version)
}

// --- Hook command factories ---

// newSessionStartHookCmd creates a session-start hook command (shared across agents).
func newSessionStartHookCmd(use, short, version string) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Short:  short,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readHookInput(os.Stdin)
			if err != nil {
				return err
			}
			baseDir, err := resolveBaseDir(input)
			if err != nil {
				return err
			}
			out, err := handleSessionStartDeduped(baseDir, version, input.dedupKey(), defaultSessionStampDir())
			if err != nil {
				return err
			}
			if len(out) == 0 {
				return nil
			}
			_, err = os.Stdout.Write(out)
			return err
		},
	}
}

// --- SessionStart dedup ------------------------------------------------------
//
// A session can receive the same SessionStart context twice: a project-level
// hook installed by `archcore init --agent` coexisting with a plugin-shipped
// hook fires both entries for one event, and both delegate to this binary.
// The stamp dedup makes the second (and any near-simultaneous) invocation for
// the same session emit nothing — regardless of which plugin/CLI versions are
// in play, since the suppression lives here in the shared binary.

// sessionStampWindow bounds both suppression and stamp retention: a stamp
// fresher than the window suppresses re-emission; anything older is swept.
// The double-fire this defends against happens within one event (seconds),
// so a short window dedupes it while letting genuinely later re-fires
// (same session, same source, minutes apart) emit again.
const sessionStampWindow = 10 * time.Minute

// defaultSessionStampDir returns the stamp directory, honoring XDG_STATE_HOME.
func defaultSessionStampDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "archcore", "session-stamps")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "archcore", "session-stamps")
}

// sessionStampPath maps an arbitrary dedup key to a filesystem-safe path.
func sessionStampPath(stampDir, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(stampDir, fmt.Sprintf("session-%x", sum[:16]))
}

// sessionStampFresh reports whether a stamp for key exists within the window.
func sessionStampFresh(stampDir, key string) bool {
	info, err := os.Stat(sessionStampPath(stampDir, key))
	return err == nil && time.Since(info.ModTime()) < sessionStampWindow
}

// claimSessionStamp atomically claims the stamp for key: exactly one of any
// number of concurrent callers wins (O_CREATE|O_EXCL), which is what makes
// the dedup hold when a project-level hook and a plugin hook fire in
// parallel for the same event. Best-effort by design: any filesystem failure
// other than "fresh stamp exists" claims (fails open) — dedup must never
// break the hook. The winner also sweeps expired stamps.
func claimSessionStamp(stampDir, key string) bool {
	if os.MkdirAll(stampDir, 0o755) != nil {
		return true // fail open
	}
	path := sessionStampPath(stampDir, key)
	for range 2 {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			sweepExpiredStamps(stampDir)
			return true
		}
		if !errors.Is(err, fs.ErrExist) {
			return true // fail open
		}
		if sessionStampFresh(stampDir, key) {
			return false // fresh stamp — a peer already emitted
		}
		// Stale stamp: reclaim by removing and retrying the exclusive
		// create once; losing that retry race means a peer reclaimed it.
		_ = os.Remove(path)
	}
	return false
}

// sweepExpiredStamps removes stamps older than the window, best-effort.
func sweepExpiredStamps(stampDir string) {
	entries, err := os.ReadDir(stampDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && time.Since(info.ModTime()) >= sessionStampWindow {
			_ = os.Remove(filepath.Join(stampDir, e.Name()))
		}
	}
}

// handleSessionStartDeduped wraps handleSessionStart with the stamp dedup.
// The host-derived key is scoped to the project by folding in baseDir — one
// session touching two projects within the window must get both contexts.
// An empty key or stampDir fails open (always emit); a suppressed repeat
// returns empty output with nil error — hosts must never see a failing hook
// here, just silence.
func handleSessionStartDeduped(baseDir, version, key, stampDir string) ([]byte, error) {
	dedup := key != "" && stampDir != ""
	if dedup {
		key += "\x00" + baseDir
		if !claimSessionStamp(stampDir, key) {
			return nil, nil
		}
	}
	out, err := handleSessionStart(baseDir, version)
	if err != nil {
		if dedup {
			// Don't let a failed build suppress the retry within the window.
			_ = os.Remove(sessionStampPath(stampDir, key))
		}
		return nil, err
	}
	return out, nil
}

// --- Session Start Handler (Claude Code adapter) ---

func handleSessionStart(baseDir, version string) ([]byte, error) {
	ctx, docCount := buildSessionContext(baseDir)
	output := hookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": ctx,
		},
		SystemMessage: display.HookConnectedLine(version, docCount),
	}
	return json.Marshal(output)
}
