package docs

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"
)

// resolveGlobalPath resolves a global source path to an absolute directory.
// It delegates to config.ResolveGlobalPath so the scan, the read-path validation,
// and the MCP startup check all resolve paths identically.
func resolveGlobalPath(baseDir, gsPath string) string {
	return config.ResolveGlobalPath(baseDir, gsPath)
}

// globalRoots holds the declared global sources with their paths already
// resolved. Built once per scan and consulted per file: resolving inside the
// walk re-derived every prefix for every document.
//
// §F: GlobalInspection owns `g` and GlobalState owns `st` in this package, so
// the receiver here is `gr`.
type globalRoots struct {
	ids      []string
	prefixes []string
}

// resolveGlobalRoots anchors every declared source to baseDir once.
func resolveGlobalRoots(baseDir string, globals []config.GlobalSource) globalRoots {
	if len(globals) == 0 {
		return globalRoots{}
	}
	gr := globalRoots{
		ids:      make([]string, len(globals)),
		prefixes: make([]string, len(globals)),
	}
	for i, gs := range globals {
		gr.ids[i] = gs.ID
		gr.prefixes[i] = filepath.ToSlash(resolveGlobalPath(baseDir, gs.Path))
	}
	return gr
}

// match returns the id of the source containing target, an absolute
// slash-separated path.
func (gr globalRoots) match(target string) (string, bool) {
	for i, prefix := range gr.prefixes {
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return gr.ids[i], true
		}
	}
	return "", false
}

// contains reports whether target falls under any declared source.
func (gr globalRoots) contains(target string) bool {
	_, ok := gr.match(target)
	return ok
}

// matchGlobal returns the id of the declared global source that contains relPath
// (relative to baseDir), and whether a match was found. Both relPath and each
// gs.Path are anchored to baseDir before comparison, so embedded, "../"-relative,
// and absolute global paths are all handled in the same coordinate space.
//
// One-shot: callers on the write path resolve a single document. The scan
// resolves the roots once instead (resolveGlobalRoots).
func matchGlobal(baseDir, relPath string, globals []config.GlobalSource) (string, bool) {
	if len(globals) == 0 {
		return "", false
	}
	return resolveGlobalRoots(baseDir, globals).match(filepath.ToSlash(filepath.Join(baseDir, relPath)))
}

// IsGlobalPath reports whether relPath (relative to baseDir) falls under any
// declared global source. Used by the write handlers to reject mutations of
// read-only global documents.
func IsGlobalPath(baseDir, relPath string, globals []config.GlobalSource) bool {
	_, ok := matchGlobal(baseDir, relPath, globals)
	return ok
}

// IsReservedGlobalDir reports whether relPath sits in reserved global mount space:
// any directory named "global" under .archcore/, at any depth, or anything inside
// one. This matches the any-depth skip the local scan applies in
// WalkArchcoreFilesSkipping (skipDirs=["global"]) so the read scan and the write
// guard agree on what is reserved — a "global" segment hides a document from the
// scan AND makes it read-only, never one without the other. Read-only regardless of
// whether the directory is declared in settings.json. The match is on whole path
// segments, so a sibling like ".archcore/global-ish/" is not reserved.
func IsReservedGlobalDir(relPath string) bool {
	rp := filepath.ToSlash(relPath)
	if !strings.HasPrefix(rp, ".archcore/") {
		return false
	}
	return slices.Contains(strings.Split(rp, "/"), "global")
}

// isReservedGlobalDirFold is IsReservedGlobalDir with case-insensitive segment
// matching. On case-insensitive filesystems (APFS, NTFS) ".archcore/Global/x"
// resolves to the reserved global/ tree on disk, so the write guard must fold
// case or the read-only invariant is bypassable. The read path keeps exact
// matching — folding there would reclassify scan results on case-sensitive
// filesystems.
func isReservedGlobalDirFold(relPath string) bool {
	rp := filepath.ToSlash(relPath)
	if !strings.HasPrefix(rp, ".archcore/") {
		return false
	}
	return slices.ContainsFunc(strings.Split(rp, "/"), func(seg string) bool {
		return strings.EqualFold(seg, "global")
	})
}

// isGlobalPathFold mirrors IsGlobalPath with case-insensitive comparison, for
// the same case-insensitive-filesystem bypass. Fail-closed: on a case-sensitive
// filesystem this may reject a local directory differing from a declared global
// only by case — acceptable for a write guard.
func isGlobalPathFold(baseDir, relPath string, globals []config.GlobalSource) bool {
	if len(globals) == 0 {
		return false
	}
	target := strings.ToLower(filepath.ToSlash(filepath.Join(baseDir, relPath)))
	for _, gs := range globals {
		prefix := strings.ToLower(filepath.ToSlash(resolveGlobalPath(baseDir, gs.Path)))
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

// IsExternalGlobalDocument reports whether p is a document file inside a
// declared global source that ValidateArchcorePath can never accept — one
// mounted from outside the store, so its documents render with a leading "..".
//
// Such a source is unaddressable through the MCP write tools: every path they
// take must start with ".archcore/", so the tools refuse it outright and
// GuardWritablePath is never reached. The pre-write hook is handed a
// host-supplied absolute path instead, so without this check a declared
// external global stays editable straight from the editor while every in-tree
// global is protected — and the two surfaces stop agreeing on the same path
// (global-sources.spec, globals-are-read-only-everywhere.rule).
//
// p may be absolute or baseDir-relative. Matching is case-folded for the reason
// isGlobalPathFold folds: on APFS or NTFS a differently-cased prefix reaches the
// same directory. Meta files and non-".md" files are excluded so the verdict
// matches GuardWritablePath's ErrPathNotDocument step — a stray file inside a
// global mount is none of the guard's business.
func IsExternalGlobalDocument(baseDir, p string, globals []config.GlobalSource) bool {
	if len(globals) == 0 || p == "" {
		return false
	}
	base := path.Base(filepath.ToSlash(p))
	if !strings.HasSuffix(base, ".md") || templates.SkipFiles[base] {
		return false
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	target := strings.ToLower(filepath.ToSlash(filepath.Clean(p)))
	for _, gs := range globals {
		prefix := strings.ToLower(filepath.ToSlash(filepath.Clean(resolveGlobalPath(baseDir, gs.Path))))
		if strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

// AnnotateSource fills SourceID, SourceKind, Global, and ReadOnly on doc by
// matching doc.Path against declared global sources in settings.json.
//
// A declared source is matched first so it keeps its own id; undeclared content
// in the reserved tree gets the sentinel. Everything else is local.
func AnnotateSource(doc *Document, baseDir string, globals []config.GlobalSource) {
	if id, ok := matchGlobal(baseDir, doc.Path, globals); ok {
		doc.SourceID = id
		doc.SourceKind = SourceKindGlobal
		doc.Global = true
		doc.ReadOnly = true
		return
	}
	if IsReservedGlobalDir(doc.Path) {
		doc.SourceID = SourceIDReserved
		doc.SourceKind = SourceKindGlobal
		doc.Global = true
		doc.ReadOnly = true
		return
	}
	doc.SourceID = string(SourceKindLocal)
	doc.SourceKind = SourceKindLocal
}
