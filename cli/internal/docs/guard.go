package docs

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"
)

// Classified write-path failures. Callers map them to their own per-surface
// messages via errors.Is; lexical failures keep ValidateArchcorePath's own text.
// None of these embed filesystem paths.
var (
	ErrPathReadOnlyGlobal = errors.New("path is in read-only global space")
	ErrPathNotDocument    = errors.New("path is not a document file")
	ErrPathEscapes        = errors.New("invalid path: resolves outside .archcore/")
)

// GuardWritablePath validates relPath (".archcore/…"-prefixed) as a mutable
// local document target. Validation layers, in order:
//  1. lexical: ValidateArchcorePath (unchanged semantics).
//  2. document-only: the basename must end in ".md" and must not be a meta
//     file (templates.SkipFiles) — settings.json and .sync-state.json are not
//     documents and must never be rewritten or removed.
//  3. reserved global tree, case-folded (isReservedGlobalDirFold).
//  4. declared global sources, exact and case-folded (fail-closed).
//  5. symlink containment: the deepest existing ancestor of the target must
//     resolve inside the real .archcore/ root — the write-side mirror of the
//     ValidateReadPath hardening, run BEFORE any MkdirAll so a symlinked
//     directory can never route writes outside the tree.
//
// Returns the cleaned path or a classified error.
//
// Both the MCP write tools and the pre-tool-use hook call this, so a direct
// editor write and an MCP mutation are judged by exactly the same rules.
func GuardWritablePath(baseDir, relPath string, globals []config.GlobalSource) (string, error) {
	cleaned, err := ValidateArchcorePath(relPath)
	if err != nil {
		return "", err
	}
	base := path.Base(cleaned)
	if !strings.HasSuffix(base, ".md") || templates.SkipFiles[base] {
		return "", ErrPathNotDocument
	}
	if isReservedGlobalDirFold(cleaned) || IsGlobalPath(baseDir, cleaned, globals) || isGlobalPathFold(baseDir, cleaned, globals) {
		return "", ErrPathReadOnlyGlobal
	}
	if err := checkSymlinkContainment(baseDir, cleaned); err != nil {
		return "", err
	}
	return cleaned, nil
}

// checkSymlinkContainment verifies that the deepest existing ancestor of
// relPath resolves inside the real .archcore/ root after evaluating symlinks.
// For update/remove that ancestor is the file itself; for create it is the
// closest existing parent directory, which catches a repo-shipped symlink
// (.archcore/x -> /elsewhere) before MkdirAll would follow it. The read guard
// reuses it so get_document can't read a file outside .archcore/ through a
// symlinked document. Errors never embed absolute paths
// (no-absolute-paths-in-mcp-errors.rule).
func checkSymlinkContainment(baseDir, relPath string) error {
	root := filepath.Join(baseDir, ".archcore")
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no .archcore/ yet — nothing to escape from
		}
		return errors.New("invalid path: cannot resolve .archcore/")
	}
	probe := filepath.Join(baseDir, filepath.FromSlash(relPath))
	for {
		real, err := filepath.EvalSymlinks(probe)
		if err == nil {
			if real != realRoot && !strings.HasPrefix(real, realRoot+string(filepath.Separator)) {
				return ErrPathEscapes
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return errors.New("invalid path: cannot resolve path")
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return nil // walked to filesystem root without finding anything
		}
		probe = parent
	}
}

// ValidateArchcorePath normalises and validates a document path.
// It returns the cleaned path or an error if the path is invalid.
//
// Uses path.Clean (POSIX, forward-slash) rather than filepath.Clean because on
// Windows filepath.Clean would re-introduce backslashes after ToSlash, breaking
// the subsequent ".archcore/" prefix check.
func ValidateArchcorePath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", errors.New("invalid path: must be relative and within .archcore/")
	}
	relPath = filepath.ToSlash(relPath)
	if !strings.HasPrefix(relPath, ".archcore/") {
		return "", errors.New("invalid path: must start with \".archcore/\"")
	}
	cleaned := path.Clean(relPath)
	if strings.HasPrefix(cleaned, "..") || !strings.HasPrefix(cleaned, ".archcore/") {
		return "", errors.New("invalid path: must be relative and within .archcore/")
	}
	return cleaned, nil
}

// ValidateReadPath validates the path argument of the read surfaces (get_document).
//
// It accepts everything ValidateArchcorePath accepts — local documents and in-tree
// globals, all under the primary's .archcore/ — with identical behavior. It
// ADDITIONALLY accepts a document that resolves strictly inside a declared external
// global source (one whose path is "../…" or absolute, so its documents render with
// a leading ".." that ValidateArchcorePath rejects). Only reads of read-only globals
// take this branch; the write path keeps the strict ValidateArchcorePath and never
// reaches it, so a "../" global stays unwritable and non-linkable.
//
// The external-global branch is hardened, defense in depth:
//  1. the path must be relative — list_documents only ever returns relative paths;
//  2. only ".md" document files are readable (exactly what the scan surfaces);
//  3. it must resolve, lexically (".." collapsed by filepath.Join), strictly within a
//     declared global root — blocking "../"-traversal escapes;
//  4. after evaluating symlinks, the real file must STILL sit inside the real global
//     root — blocking symlink escapes out of the read-only mount.
//
// A path under a declared global that points at a missing file returns fs.ErrNotExist
// so the caller reports an ordinary "document not found". Errors never embed an
// absolute path (see no-absolute-paths-in-mcp-errors.rule).
func ValidateReadPath(baseDir, relPath string, globals []config.GlobalSource) (string, error) {
	// Local documents and in-tree globals: strict lexical validation, then a
	// symlink-containment check so a symlinked document (.archcore/x.adr.md ->
	// /etc/passwd) can't leak a file outside .archcore/ through get_document.
	// This mirrors the write guard; the external-global branch below runs its
	// own EvalSymlinks containment.
	if cleaned, err := ValidateArchcorePath(relPath); err == nil {
		if err := checkSymlinkContainment(baseDir, cleaned); err != nil {
			return "", err
		}
		return cleaned, nil
	}

	// External-global read candidate. Tighten every axis before allowing it.
	if filepath.IsAbs(relPath) {
		return "", errors.New("invalid path: must be relative and within .archcore/")
	}
	rel := path.Clean(filepath.ToSlash(relPath))
	if !strings.HasSuffix(rel, ".md") {
		return "", errors.New("invalid path: only .md global documents are readable")
	}

	target := filepath.ToSlash(filepath.Join(baseDir, rel))
	for _, gs := range globals {
		root := filepath.ToSlash(filepath.Clean(resolveGlobalPath(baseDir, gs.Path)))
		if target != root && !strings.HasPrefix(target, root+"/") {
			continue // not under this global
		}
		// Lexically inside a declared global. Harden against symlink escape: the
		// real file must remain inside the real global root.
		realRoot, err := filepath.EvalSymlinks(filepath.FromSlash(root))
		if err != nil {
			return "", errors.New("invalid path: global source is not accessible")
		}
		realFile, err := filepath.EvalSymlinks(filepath.FromSlash(target))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fs.ErrNotExist // missing document → ordinary not-found
			}
			return "", errors.New("invalid path: cannot resolve global document")
		}
		if realFile != realRoot && !strings.HasPrefix(realFile, realRoot+string(filepath.Separator)) {
			return "", errors.New("invalid path: resolves outside the global source")
		}
		return rel, nil
	}
	return "", errors.New("invalid path: must start with \".archcore/\"")
}
