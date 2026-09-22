package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"
)

// GlobalState classifies a declared global source for the startup, status, and
// session-start reporting surfaces.
type GlobalState int

const (
	// GlobalOK: directory exists, is readable, and holds at least one document.
	GlobalOK GlobalState = iota
	// GlobalMissing: the resolved directory does not exist.
	GlobalMissing
	// GlobalNotDir: the resolved path exists but is not a directory.
	GlobalNotDir
	// GlobalUnreadable: the directory exists but cannot be read.
	GlobalUnreadable
	// GlobalSelfOverlap: the path resolves to the project's own .archcore or an ancestor.
	GlobalSelfOverlap
	// GlobalDuplicate: an earlier declared source resolves to the same directory.
	GlobalDuplicate
	// GlobalEmpty: the directory is readable but contains no archcore documents.
	GlobalEmpty
)

// Fatal reports whether the state must prevent serving (the MCP server refuses to
// start; status counts it as an issue). GlobalOK and GlobalEmpty are not fatal —
// an empty source is surfaced as a warning, never a hard failure.
// §F edge case: GlobalInspection owns the g receiver in this package;
// GlobalState methods use st.
func (st GlobalState) Fatal() bool {
	switch st {
	case GlobalOK, GlobalEmpty:
		return false
	default:
		return true
	}
}

// GlobalInspection is the classification of one declared global source. It
// carries only the declared (relative) path and id — never an absolute path — so
// callers can render a message without leaking a filesystem path.
type GlobalInspection struct {
	ID    string
	Path  string
	State GlobalState
	Docs  int
	// DocsByCategory counts mounted documents per virtual category, derived
	// from the filename type suffix during the same walk that counts Docs.
	// Filled only for a source the walk completed (GlobalOK or GlobalEmpty);
	// zero-read by construction (session-globals-disclosure.spec).
	DocsByCategory map[templates.Category]int
	// TopDirs counts mounted documents per top-level directory under the
	// source root, from the same walk. A root-level document carries no entry.
	TopDirs map[string]int
}

// Message returns a user-facing description of a non-OK inspection, built from
// the declared id and path. The directory-level states reuse
// config.DescribeGlobalDirError so the wording matches the MCP scan exactly.
func (g GlobalInspection) Message() string {
	gs := config.GlobalSource{ID: g.ID, Path: g.Path}
	switch g.State {
	case GlobalMissing:
		return config.DescribeGlobalDirError(gs, config.ErrGlobalMissing)
	case GlobalNotDir:
		return config.DescribeGlobalDirError(gs, config.ErrGlobalNotDir)
	case GlobalUnreadable:
		return config.DescribeGlobalDirError(gs, config.ErrGlobalUnreadable)
	case GlobalSelfOverlap:
		return config.DescribeGlobalDirError(gs, config.ErrGlobalSelfOverlap)
	case GlobalDuplicate:
		return fmt.Sprintf("global source %q at %q duplicates an earlier source's path", g.ID, g.Path)
	case GlobalEmpty:
		return fmt.Sprintf("global source %q at %q contains no documents", g.ID, g.Path)
	default:
		return ""
	}
}

// InspectGlobals classifies every declared global source for the startup, status,
// and session-start surfaces. It is the single source of truth for missing /
// not-a-directory / unreadable / self-overlap / duplicate / empty detection.
//
// The returned error is non-nil only when settings.json is present but invalid
// (fail closed) — a missing settings.json yields no inspections and no error.
func InspectGlobals(baseDir string) ([]GlobalInspection, error) {
	globals, err := config.LoadGlobals(baseDir)
	if err != nil {
		return nil, err
	}

	out := make([]GlobalInspection, 0, len(globals))
	seen := make(map[string]struct{}, len(globals)) // resolved dir set
	for _, gs := range globals {
		in := GlobalInspection{ID: gs.ID, Path: gs.Path}
		resolved := config.ResolveGlobalPath(baseDir, gs.Path)

		if _, dup := seen[resolved]; dup {
			in.State = GlobalDuplicate
			out = append(out, in)
			continue
		}
		seen[resolved] = struct{}{}

		if dirErr := config.CheckGlobalDir(baseDir, resolved); dirErr != nil {
			in.State = stateForDirErr(dirErr)
			out = append(out, in)
			continue
		}

		counts, walkErr := countGlobalDocs(resolved)
		if walkErr != nil {
			// A nested directory the walk cannot enter. CheckGlobalDir only
			// probes the top level, so this is the first place the failure is
			// visible — and deriving a state from the partial count would report
			// OK or empty for a source the runtime scan then refuses outright.
			in.State = GlobalUnreadable
			out = append(out, in)
			continue
		}
		in.Docs = counts.total
		in.DocsByCategory = counts.byCategory
		in.TopDirs = counts.topDirs
		if in.Docs == 0 {
			in.State = GlobalEmpty
		}
		out = append(out, in)
	}
	return out, nil
}

// stateForDirErr maps a config.CheckGlobalDir sentinel to a GlobalState.
func stateForDirErr(err error) GlobalState {
	switch {
	case errors.Is(err, config.ErrGlobalMissing):
		return GlobalMissing
	case errors.Is(err, config.ErrGlobalNotDir):
		return GlobalNotDir
	case errors.Is(err, config.ErrGlobalSelfOverlap):
		return GlobalSelfOverlap
	default:
		return GlobalUnreadable
	}
}

// globalDocCounts is what one source-count walk produces: the mount total plus
// the filename-derived breakdowns the session-start disclosure renders.
type globalDocCounts struct {
	total      int
	byCategory map[templates.Category]int
	topDirs    map[string]int
}

// countGlobalDocs counts the recognized-type documents a global directory would
// mount, using the same valid-type filter as the scan, and breaks the count down
// by virtual category and by top-level directory. Every field is derived from
// the filename and the directory name — no document is opened
// (session-globals-disclosure.spec).
//
// The walk error is returned, not swallowed: CheckGlobalDir opens only the top
// directory, so an unreadable subdirectory reaches this walk unclassified. The
// runtime scan walks the same tree and fails on it, and the startup gate and the
// scan have to classify a source identically (global-sources.spec) — a discarded
// error is what let `status` print a healthy source whose every read then failed.
func countGlobalDocs(resolvedDir string) (globalDocCounts, error) {
	counts := globalDocCounts{
		byCategory: make(map[templates.Category]int),
		topDirs:    make(map[string]int),
	}
	err := templates.WalkArchcoreFilesSkipping(resolvedDir, nil, func(p string, d fs.DirEntry) error {
		docType := templates.DocumentType(templates.ExtractDocType(d.Name()))
		if !templates.IsValidType(string(docType)) {
			return nil
		}
		counts.total++
		counts.byCategory[templates.CategoryForType(docType)]++
		if rel, relErr := filepath.Rel(resolvedDir, p); relErr == nil {
			if top, _, nested := strings.Cut(filepath.ToSlash(rel), "/"); nested {
				counts.topDirs[top]++
			}
		}
		return nil
	})
	if err != nil {
		return globalDocCounts{}, err
	}
	return counts, nil
}
