// Package git runs the handful of read-only git queries the CLI needs. Every
// call is bounded: a hook runs inside a one-second budget, and a git invocation
// that blocks (a credential helper waiting on a tty, a network remote) must
// never be what spends it.
package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// callTimeout bounds a single git invocation. Every query here is local and
// metadata-only, so anything slower is a hang, not slow work.
const callTimeout = 500 * time.Millisecond

// waitDelay bounds how long Wait blocks on inherited pipes after the context is
// cancelled. Without it a grandchild holding the pipe open outlives the timeout.
const waitDelay = 100 * time.Millisecond

// ErrGitAbsent reports that no git executable is on PATH. It is distinct from
// "not a repository" so a caller can tell "no git installed" from "not a repo"
// instead of collapsing both into an empty result.
var ErrGitAbsent = errors.New("git executable not found in PATH")

// lookPath is a seam so tests can simulate a machine without git.
var lookPath = exec.LookPath

// maxOutput bounds what one git query may return. ChangedSince over a long gap
// lists every changed file, and the caller only needs the count and the
// top-level directory names — so a truncated answer is a slightly shorter list,
// not a failure.
const maxOutput = 1 << 20

// capWriter keeps at most maxOutput bytes and discards the rest. Write never
// fails, so an oversized result truncates instead of killing the child.
type capWriter struct {
	buf       []byte
	truncated bool
}

func (w *capWriter) Write(p []byte) (int, error) {
	if room := maxOutput - len(w.buf); room > 0 {
		w.buf = append(w.buf, p[:min(room, len(p))]...)
	}
	if len(w.buf) >= maxOutput {
		w.truncated = true
	}
	return len(p), nil
}

// run executes one git query in dir and returns its trimmed stdout, plus
// whether the output hit the size cap. stderr is discarded: every caller either
// has a usable result or falls back, and git's stderr embeds absolute paths.
func run(ctx context.Context, dir string, args ...string) (string, bool, error) {
	if _, err := lookPath("git"); err != nil {
		return "", false, ErrGitAbsent
	}

	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.WaitDelay = waitDelay

	var out capWriter
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out.buf)), out.truncated, nil
}

// DetectRepoURL returns the URL of the "origin" remote for the git repository
// at dir, or an empty string if detection fails (not a git repo, no remote,
// git absent). The empty-string contract is load-bearing: sync omits RepoURL
// from its payload rather than failing when detection is impossible.
func DetectRepoURL(ctx context.Context, dir string) string {
	out, _, err := run(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return out
}

// CurrentBranch returns the checked-out branch name. A detached HEAD yields
// ErrDetachedHead so a caller can render the state rather than print "HEAD".
func CurrentBranch(ctx context.Context, dir string) (string, error) {
	out, _, err := run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", ErrDetachedHead
	}
	return out, nil
}

// ErrDetachedHead reports that the working tree has no checked-out branch.
var ErrDetachedHead = errors.New("detached HEAD")

// LastCommitTouching returns the hash of the most recent commit that changed
// anything under pathspec. An empty string with a nil error means the pathspec
// has no history yet — an ordinary state, not a failure.
func LastCommitTouching(ctx context.Context, dir, pathspec string) (string, error) {
	out, _, err := run(ctx, dir, "log", "-1", "--format=%H", "--", pathspec)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ChangedSince lists files changed between sha and HEAD. Each element of
// excludes is passed as a git pathspec (for example ":(exclude).archcore/"), so
// a caller can ask "what changed outside the documentation?" in one query.
// Returns nil when nothing changed.
func ChangedSince(ctx context.Context, dir, sha string, excludes ...string) ([]string, error) {
	args := append([]string{"diff", "--name-only", sha + "..HEAD", "--"}, excludes...)
	out, truncated, err := run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	names := strings.Split(out, "\n")
	if truncated && len(names) > 0 {
		// The cap can land mid-path, so the last element may be a fragment.
		names = names[:len(names)-1]
	}
	return names, nil
}
