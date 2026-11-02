package projectroot

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Nearest returns the closest directory at or above path that holds a
// .archcore/ directory: the project a file belongs to
// (call-root-resolution.spec §5, §13). The walk is bounded by the depth of path.
//
// An advisory read: a directory the process may not search ends the walk with
// no project, so the caller falls back to its working directory instead of a
// farther ancestor (call-root-resolution.spec, Failure Behavior 8).
func Nearest(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	dir := filepath.Clean(path)
	for {
		info, err := os.Stat(filepath.Join(dir, ".archcore"))
		if err == nil && info.IsDir() {
			return dir, true
		}
		if errors.Is(err, fs.ErrPermission) {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Canonical returns dir with its symlinks evaluated, the one spelling a root of
// another project carries on the wire and in every cache key
// (call-root-resolution.spec, the invariant that two spellings of one directory
// are one root). An advisory read: a path that cannot be evaluated is returned
// as written.
func Canonical(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return dir
}

// SameDir reports whether a and b name one directory, either as written or
// after symlinks are evaluated (call-root-resolution.spec, the invariant that
// two spellings of one directory are one root). An advisory read: a path whose
// symlinks cannot be evaluated is compared as written.
func SameDir(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	realA, errA := filepath.EvalSymlinks(a)
	realB, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && realA == realB
}

// EvalExisting evaluates the symlinks of the deepest existing ancestor of p and
// keeps the rest as written, because the file a write targets often does not
// exist yet. The walk is bounded by the depth of p. An advisory read: a path
// with no evaluable ancestor is returned as written.
func EvalExisting(p string) string {
	dir, rest := filepath.Clean(p), ""
	for {
		if real, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return p
		}
		rest = filepath.Join(filepath.Base(dir), rest)
		dir = parent
	}
}
