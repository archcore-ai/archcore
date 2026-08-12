package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
// points at. An absolute path is cleaned and returned as-is; a relative path
// (including "../") is resolved against baseDir. This is the single resolution
// point shared by the MCP scan, the read-path validation, and the startup check.
func ResolveGlobalPath(baseDir, gsPath string) string {
	if filepath.IsAbs(gsPath) {
		return filepath.Clean(gsPath)
	}
	return filepath.Clean(filepath.Join(baseDir, gsPath))
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
