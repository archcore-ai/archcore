package docs

import (
	"path/filepath"
	"strings"
)

// RelativeToBase converts a host-supplied path to a baseDir-relative, slash-
// separated path. It reports false when the path lies outside the project,
// which the guard treats as none of its business.
func RelativeToBase(baseDir, p string) (string, bool) {
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(filepath.Clean(p)), true
	}
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
