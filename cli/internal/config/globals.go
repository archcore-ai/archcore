package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"archcore-cli/internal/git"
)

// Sentinel errors classifying why a declared global source's resolved directory
// cannot be used. Callers turn these into user-facing messages with
// DescribeGlobalDirError, which is built from the source's id and declared
// (relative) path — never from the resolved absolute path
// (see no-absolute-paths-in-mcp-errors.rule).
var (
	ErrGlobalMissing     = errors.New("global source directory does not exist")
	ErrGlobalNotDir      = errors.New("global source path is not a directory")
	ErrGlobalUnreadable  = errors.New("global source directory is not readable")
	ErrGlobalSelfOverlap = errors.New("global source resolves to the project's own .archcore")
)

// ResolveGlobalPath returns the absolute, cleaned directory a global source path
// points at. This is the single resolution point shared by the MCP scan, the
// read-path validation, and the startup check.
//
// An absolute path is cleaned and returned as-is. A relative path that stays
// inside baseDir resolves against baseDir: such a source is version-controlled,
// so a git worktree must read its own branch's copy. A relative path that
// escapes baseDir ("../company/.archcore") resolves against the repository's
// main checkout, because a worktree does not share the main checkout's parent
// directory and would otherwise resolve a declared source to nothing
// (relative-globals-resolve-from-main-checkout.adr).
//
// The main checkout is looked up once per baseDir and memoized. A project that
// declares no escaping path never triggers the lookup.
func ResolveGlobalPath(baseDir, gsPath string) string {
	if filepath.IsAbs(gsPath) {
		return filepath.Clean(gsPath)
	}
	base := filepath.Clean(baseDir)
	inTree, escapes := resolveInTree(base, gsPath)
	if !escapes {
		return inTree
	}
	return anchoredAt(mainCheckoutFor(base), gsPath, inTree)
}

// ResolveGlobalPathFrom is the pure core of ResolveGlobalPath: it performs no
// I/O and runs no subprocess, so the caller supplies the main checkout. An empty
// mainCheckout selects today's behavior — resolution against baseDir alone.
//
// baseDir must be an absolute path. escapesDir compares lexically, so a relative
// baseDir such as "." reports every in-tree path as escaping and the anchor
// would be applied where it must not be. ResolveGlobalPath satisfies this by
// construction; a direct caller must too.
func ResolveGlobalPathFrom(baseDir, mainCheckout, gsPath string) string {
	if filepath.IsAbs(gsPath) {
		return filepath.Clean(gsPath)
	}
	base := filepath.Clean(baseDir)
	inTree, escapes := resolveInTree(base, gsPath)
	if !escapes {
		return inTree
	}
	return anchoredAt(mainCheckout, gsPath, inTree)
}

// resolveInTree resolves gsPath against base and reports whether the result
// leaves base. Both functions above need the resolution and the classification,
// and computing them once is what keeps ResolveGlobalPath from joining and
// classifying the same pair twice.
func resolveInTree(base, gsPath string) (resolved string, escapes bool) {
	resolved = filepath.Clean(filepath.Join(base, gsPath))
	return resolved, escapesDir(base, resolved)
}

// anchoredAt resolves gsPath against anchor, falling back to the in-tree
// resolution when no anchor is usable.
func anchoredAt(anchor, gsPath, inTree string) string {
	if anchor == "" {
		return inTree
	}
	return filepath.Clean(filepath.Join(filepath.Clean(anchor), gsPath))
}

// escapesDir reports whether resolved falls outside baseDir. Both arguments must
// already be cleaned absolute paths.
func escapesDir(baseDir, resolved string) bool {
	return resolved != baseDir && !strings.HasPrefix(resolved, baseDir+string(filepath.Separator))
}

// mainCheckoutCache memoizes the resolution anchor per project root. The lookup
// costs two git subprocesses, the answer cannot change while the process serves
// one checkout, and the MCP server resolves global paths on every read.
var (
	mainCheckoutMu    sync.Mutex
	mainCheckoutCache = map[string]string{}
)

// lookupWorktreeRoots is a seam so tests can drive resolution without building a
// git repository. Production runs the bounded queries in internal/git.
var lookupWorktreeRoots = git.WorktreeRoots

// mainCheckoutFor returns the directory an escaping relative path resolves
// against, or "" when today's resolution against the project root stands.
//
// The anchor is the project root's own position inside the main checkout, not
// the main checkout itself: a project may sit below its working tree root
// (examples/05-global-single-source in this repository does), and such a project
// declares its path relative to itself. Mapping the position keeps a nested
// project resolving exactly as before while a worktree of the whole repository
// moves onto the main checkout.
//
// The anchor is accepted only when it holds a .archcore/ directory. That check
// rejects the answer git gives inside a submodule, where the reported worktree
// is <super>/.git/modules/<name> rather than the real checkout.
//
// The lock is released before the lookup and retaken to publish. Holding it
// across deriveAnchor would put two git subprocesses inside a mutex that every
// concurrent MCP tool call contends for (process-and-concurrency-model.spec §1).
// Two callers racing on a cold key may both derive; the answer is the same for
// both, so the cost is one duplicate lookup rather than a serialized handler.
func mainCheckoutFor(baseDir string) string {
	mainCheckoutMu.Lock()
	cached, ok := mainCheckoutCache[baseDir]
	mainCheckoutMu.Unlock()
	if ok {
		return cached
	}

	anchor := deriveAnchor(baseDir)

	mainCheckoutMu.Lock()
	defer mainCheckoutMu.Unlock()
	if cached, ok := mainCheckoutCache[baseDir]; ok {
		return cached // another caller published first; one answer per key
	}
	mainCheckoutCache[baseDir] = anchor
	return anchor
}

// deriveAnchor computes the anchor for baseDir, or "" when none is usable: a
// non-git directory, a machine without git, a bare repository, a submodule, or a
// project root outside the reported working tree.
//
// Advisory, not a guard (fail-open-or-fail-closed-reads.rule §4): every failure
// below returns "", which resolves the declared path against the project root —
// the behavior that stood before worktree awareness existed. A worktree whose
// git query fails reads its own copy rather than nothing.
//
// The lookup takes context.Background() rather than the caller's context. The
// memo sits behind ResolveGlobalPath, whose signature carries no context and is
// called from the read path of every MCP tool; threading one through would
// change that signature everywhere for a query internal/git already bounds at
// callTimeout (relative-globals-resolve-from-main-checkout.adr).
func deriveAnchor(baseDir string) string {
	roots, err := lookupWorktreeRoots(context.Background(), baseDir)
	if err != nil {
		return ""
	}
	// git answers with symlink-evaluated paths (/private/var on macOS), while
	// baseDir may still carry the symlinked spelling (/var). Relativizing across
	// the two spellings yields a "../../.." that means nothing, so both sides are
	// evaluated before they are compared.
	current, main := realPath(roots.Current), realPath(roots.Main)
	if current == main {
		return "" // not a linked worktree; resolution against the project root stands
	}
	rel, err := filepath.Rel(current, realPath(baseDir))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	anchor := filepath.Clean(filepath.Join(main, rel))
	if !holdsArchcoreDir(anchor) {
		return ""
	}
	return anchor
}

// realPath returns dir with symlinks evaluated, or the cleaned dir when it
// cannot be evaluated (it does not exist yet, or a component is unreadable).
func realPath(dir string) string {
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(dir)
}

// holdsArchcoreDir reports whether dir contains a readable .archcore directory.
func holdsArchcoreDir(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, dirName))
	return err == nil && info.IsDir()
}

// CheckGlobalDir classifies a global source's resolved directory at the
// filesystem level, returning one of the ErrGlobal* sentinels (match with
// errors.Is) or nil when the directory exists, is a readable directory, and does
// not overlap the primary's own .archcore tree.
//
// It deliberately does NOT inspect document contents: an existing, readable
// directory holding zero archcore documents returns nil here. "Empty" is a
// warning the scan/report layer surfaces, not a hard error.
//
// resolvedDir must come from ResolveGlobalPath(baseDir, gs.Path). The returned
// error never embeds resolvedDir, so callers can format a message from the
// declared id/path without leaking an absolute path.
func CheckGlobalDir(baseDir, resolvedDir string) error {
	// Self-overlap: a global may not resolve to the primary's own .archcore or an
	// ancestor of it, which would re-mount the primary's local documents as
	// read-only globals. In-tree vendored globals (descendants such as
	// .archcore/global/<id>) are allowed and intentionally not flagged.
	archcoreDir := filepath.Clean(filepath.Join(baseDir, dirName))
	if resolvedDir == archcoreDir ||
		strings.HasPrefix(archcoreDir, resolvedDir+string(filepath.Separator)) {
		return ErrGlobalSelfOverlap
	}

	info, err := os.Stat(resolvedDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrGlobalMissing
		}
		return ErrGlobalUnreadable
	}
	if !info.IsDir() {
		return ErrGlobalNotDir
	}

	// Readability probe. os.Stat succeeds on a directory the process cannot read
	// (it only needs traverse permission on the parent), but the scan walk would
	// then fail at runtime — so a permission problem must be caught here for
	// startup and runtime to agree. An empty-but-readable directory returns
	// io.EOF, which is not an error.
	f, err := os.Open(resolvedDir)
	if err != nil {
		return ErrGlobalUnreadable
	}
	defer func() { _ = f.Close() }() // read-only handle
	if _, err := f.Readdirnames(1); err != nil && !errors.Is(err, io.EOF) {
		return ErrGlobalUnreadable
	}
	return nil
}

// DescribeGlobalDirError maps a CheckGlobalDir sentinel to a user-facing message
// built only from the source's id and declared path. Returns "" for a nil error.
// This is the single source of truth for these message strings, shared by the
// MCP scan (internal/mcp/tools) and the startup/status surfaces.
func DescribeGlobalDirError(gs GlobalSource, err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrGlobalMissing):
		return fmt.Sprintf("global source %q not found at %q", gs.ID, gs.Path)
	case errors.Is(err, ErrGlobalNotDir):
		return fmt.Sprintf("global source %q at %q is not a directory", gs.ID, gs.Path)
	case errors.Is(err, ErrGlobalUnreadable):
		return fmt.Sprintf("global source %q at %q is not readable", gs.ID, gs.Path)
	case errors.Is(err, ErrGlobalSelfOverlap):
		return fmt.Sprintf("global source %q at %q resolves to the project's own .archcore", gs.ID, gs.Path)
	default:
		return fmt.Sprintf("global source %q at %q is not usable", gs.ID, gs.Path)
	}
}
