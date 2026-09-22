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

// lookPath is a seam so tests can simulate a machine without git. Production
// never reassigns it.
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

// Roots names the two working trees a path can be anchored against: Current is
// the working tree holding the queried directory, Main is the repository's main
// worktree. They are equal outside a linked worktree.
type Roots struct {
	Current string
	Main    string
}

// WorktreeRoots reports both working trees for dir in one call, so a caller that
// must map a path from a linked worktree onto the main checkout gets a
// consistent pair rather than two independently timed answers.
//
// The pair shares one deadline. run bounds each invocation on its own, so
// without the wrapper two queries could spend 2×callTimeout — the whole
// one-second budget the package doc comment says a single hook has.
func WorktreeRoots(ctx context.Context, dir string) (Roots, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	current, err := toplevel(ctx, dir)
	if err != nil {
		return Roots{}, err
	}
	main, err := mainCheckout(ctx, dir)
	if err != nil {
		return Roots{}, err
	}
	return Roots{Current: current, Main: main}, nil
}

// toplevel returns the root of the working tree that holds dir. Inside a linked
// worktree it returns that worktree, not the main checkout.
func toplevel(ctx context.Context, dir string) (string, error) {
	out, _, err := run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errNoWorkingTree
	}
	return out, nil
}

// errNoWorkingTree reports that `rev-parse --show-toplevel` succeeded but named
// no working tree, and errNoMainWorktree that `worktree list --porcelain`
// printed no "worktree " line. They stay separate because they answer about
// different subjects: where the caller stands, and the repository as a whole.
//
// Both are defensive. git fails outright in the states that would produce them
// today — a bare repository makes show-toplevel exit non-zero, and worktree list
// names even a bare repository's own path — so these exist to stop an empty
// answer from being returned as a valid empty path.
//
// Neither is exported: WorktreeRoots is the only caller, and its own caller
// (config.deriveAnchor) treats every failure alike.
var (
	errNoWorkingTree  = errors.New("directory is in no git working tree")
	errNoMainWorktree = errors.New("repository has no main worktree")
)

// mainCheckout returns the working tree of the repository's main worktree — the
// checkout `git worktree add` was run from. Called inside a linked worktree it
// returns the main checkout, not the caller's tree; called inside the main
// checkout it returns that checkout.
//
// The main worktree is the first entry `git worktree list --porcelain` prints,
// which is why this uses that query rather than deriving a path from
// `rev-parse --git-common-dir`: the derivation is the parent of the common
// directory only when that directory is named ".git", which a bare repository
// and `git init --separate-git-dir` both break.
//
// Inside a submodule the answer is wrong — git reports the worktree as
// <super>/.git/modules/<name> while the real checkout is <super>/<name> — so a
// caller MUST validate the returned path before trusting it.
func mainCheckout(ctx context.Context, dir string) (string, error) {
	out, _, err := run(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			return strings.TrimSpace(p), nil
		}
	}
	return "", errNoMainWorktree
}
