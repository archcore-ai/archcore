package docs

import (
	"path"
	"path/filepath"
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

// index returns the position of the source containing target, an absolute
// slash-separated path.
func (gr globalRoots) index(target string) (int, bool) {
	for i, prefix := range gr.prefixes {
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return i, true
		}
	}
	return 0, false
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
// (source-mounts.spec, behavior 13).
//
// p may be absolute or baseDir-relative. Matching is case-folded: on APFS or
// NTFS a differently-cased prefix reaches the same directory. Meta files and non-".md" files are excluded so the verdict
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

// AnnotateSource fills SourceID, SourceKind, and SourceRoot on doc by matching
// doc.Path against the sources declared in settings.json; everything else is
// local (source-mounts.spec, behaviors 4–5).
func AnnotateSource(doc *Document, baseDir string, globals []config.GlobalSource) {
	if len(globals) > 0 {
		roots := resolveGlobalRoots(baseDir, globals)
		if i, ok := roots.index(filepath.ToSlash(filepath.Join(baseDir, doc.Path))); ok {
			doc.SourceID = roots.ids[i]
			doc.SourceKind = SourceKindGlobal
			doc.SourceRoot = sourceRoot(filepath.FromSlash(roots.prefixes[i]))
			return
		}
	}
	doc.SourceID = string(SourceKindLocal)
	doc.SourceKind = SourceKindLocal
}

// sourceRoot returns the project root of a source whose resolved directory is
// dir: the parent of a directory named .archcore, else "" — a source that is not
// a project's .archcore/ has no project argument that reaches it.
func sourceRoot(dir string) string {
	if filepath.Base(dir) != ".archcore" {
		return ""
	}
	return filepath.Dir(dir)
}

// SourceIDFor returns the id under which the project at baseDir declares the
// project at root as a source. Advisory: an unreadable settings.json declares
// nothing, and the caller then names no source.
func SourceIDFor(baseDir, root string) (string, bool) {
	want := realDir(root)
	for _, gs := range config.ReadGlobals(baseDir) {
		if r := sourceRoot(resolveGlobalPath(baseDir, gs.Path)); r != "" && realDir(r) == want {
			return gs.ID, true
		}
	}
	return "", false
}

// realDir returns dir with symlinks evaluated, or cleaned when it cannot be.
func realDir(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return filepath.Clean(dir)
}
