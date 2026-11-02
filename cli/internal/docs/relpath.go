package docs

import (
	"path/filepath"
	"strings"

	"archcore-cli/internal/projectroot"
)

// RelativeToBase converts a host-supplied path to a baseDir-relative, slash-
// separated path. It reports false when the path lies outside the project,
// which the guard treats as none of its business.
//
// A path spelled through a symlink of baseDir — /tmp against /private/tmp on
// macOS, or a user's link to a checkout — is compared again with symlinks
// evaluated: compared as written it lands outside, and the write guard then
// allowed a direct write into the store (host-probe-protocol.spec run of
// 2026-10-30). An advisory read: a path that cannot be evaluated stays outside.
func RelativeToBase(baseDir, p string) (string, bool) {
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(filepath.Clean(p)), true
	}
	if rel, ok := relativeWithin(baseDir, p); ok {
		return rel, true
	}
	realBase, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return "", false
	}
	return relativeWithin(realBase, projectroot.EvalExisting(p))
}

func relativeWithin(baseDir, p string) (string, bool) {
	rel, err := filepath.Rel(baseDir, p)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}
