package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"archcore-cli/internal/config"
	"archcore-cli/templates"
)

// scanCount records how many corpus walks this process has performed.
var scanCount atomic.Uint64

// ScanCount reports how many corpus scans this process has performed. It exists
// so a test can assert that a code path does no scanning — the write guard is
// required to reach its verdict without one, and there is no other way to
// observe that from outside the package. Production never reads it.
func ScanCount() uint64 { return scanCount.Load() }

// scanOptions is what one walk needs to know. Collected into a struct because
// the three flags are independent and a positional list of them reads wrong at
// every call site.
type scanOptions struct {
	// includeContent populates Document.Content. The file is read either way —
	// frontmatter needs it — so this only controls whether the body is kept.
	includeContent bool
	// types restricts the walk to these document types. A nil map accepts
	// everything. Filtering happens on the filename, before the file is opened.
	types map[templates.DocumentType]bool
	// globals are the declared global sources, loaded once per request.
	globals []config.GlobalSource
}

// Scan discovers all .md files recursively inside .archcore/.
// Global sources declared in settings.json are scanned read-only.
func Scan(baseDir string) ([]Document, error) {
	return scanAll(baseDir, scanOptions{globals: config.ReadGlobals(baseDir)})
}

// ScanFull mirrors Scan but also populates the Content field for every document.
// Frontmatter is parsed from the same bytes read from disk, so this performs no
// extra I/O compared to Scan.
func ScanFull(baseDir string) ([]Document, error) {
	return scanAll(baseDir, scanOptions{includeContent: true, globals: config.ReadGlobals(baseDir)})
}

// ScanTypes mirrors ScanFull but visits only documents of the given types.
//
// The type is derived from the filename, so a document outside the set is never
// opened, parsed, or cached. A caller that keeps a handful of types out of the
// full vocabulary pays for a walk instead of a read of the whole corpus.
func ScanTypes(baseDir string, types map[templates.DocumentType]bool) ([]Document, error) {
	return scanAll(baseDir, scanOptions{includeContent: true, types: types, globals: config.ReadGlobals(baseDir)})
}

// ScanLocal discovers only the primary project's own documents, never touching
// declared global sources. Unlike Scan it cannot fail because a global is
// missing — it never reads one. Surfaces that operate on local documents only
// (status, session context, hook injection) use this so an unreachable global
// degrades to local-only instead of blanking the whole result.
//
// includeContent populates Document.Content from the same bytes already read, so
// a content-bearing local scan costs no extra I/O over a metadata-only one.
func ScanLocal(baseDir string, includeContent bool) ([]Document, error) {
	return scanLocal(baseDir, scanOptions{includeContent: includeContent, globals: config.ReadGlobals(baseDir)})
}

// ScanLocalTypes is ScanTypes restricted to the primary project, used where a
// broken global must degrade to local-only rather than blank the result.
func ScanLocalTypes(baseDir string, types map[templates.DocumentType]bool) ([]Document, error) {
	return scanLocal(baseDir, scanOptions{includeContent: true, types: types, globals: config.ReadGlobals(baseDir)})
}

// scanLocal performs phase 1 of the scan: the primary's own documents.
// It skips the reserved global/ mount directory and any document that falls under
// a declared global source (surfaced read-only in phase 2 instead), so a document
// is never scanned as both local and global. A missing .archcore/ yields (nil, nil).
func scanLocal(baseDir string, opt scanOptions) ([]Document, error) {
	scanCount.Add(1)
	archcoreDir := filepath.Join(baseDir, ".archcore")
	roots := resolveGlobalRoots(baseDir, opt.globals)
	var docs []Document

	err := templates.WalkArchcoreFilesSkipping(archcoreDir, []string{"global"}, func(p string, d fs.DirEntry) error {
		if !opt.accepts(d.Name()) {
			return nil
		}
		if roots.contains(filepath.ToSlash(p)) {
			return nil // surfaced read-only in phase 2, never as a local document
		}
		doc := buildDoc(baseDir, p, d, opt.includeContent)
		doc.SourceID = string(SourceKindLocal)
		doc.SourceKind = SourceKindLocal
		docs = append(docs, doc)
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// accepts reports whether a filename's document type is in scope for this walk.
func (o scanOptions) accepts(filename string) bool {
	if o.types == nil {
		return true
	}
	return o.types[templates.DocumentType(templates.ExtractDocType(filename))]
}

// scanAll walks .archcore/ in two phases:
//  1. Local documents — skips the global/ subdirectory entirely (scanLocal).
//  2. Global sources declared in settings.json — each source is walked
//     independently and documents are tagged read-only.
func scanAll(baseDir string, opt scanOptions) ([]Document, error) {
	docs, err := scanLocal(baseDir, opt)
	if err != nil {
		return nil, err
	}

	// Phase 2: mounted global sources declared in settings.json. Every declared
	// source is mandatory, so an unusable one fails the scan rather than
	// silently yielding a smaller corpus.
	seen := make(map[string]string, len(opt.globals)) // resolved dir -> first source id
	for _, gs := range opt.globals {
		gsDir := resolveGlobalPath(baseDir, gs.Path)
		if firstID, dup := seen[gsDir]; dup {
			return nil, fmt.Errorf("global sources %q and %q resolve to the same path %q", firstID, gs.ID, gs.Path)
		}
		seen[gsDir] = gs.ID
		if dirErr := config.CheckGlobalDir(baseDir, gsDir); dirErr != nil {
			return nil, errors.New(config.DescribeGlobalDirError(gs, dirErr))
		}
		walkErr := templates.WalkArchcoreFilesSkipping(gsDir, nil, func(p string, d fs.DirEntry) error {
			// Mount only recognized document types: a misconfigured path (e.g. a
			// parent directory) must not surface stray .md files as malformed docs.
			if !templates.IsValidType(templates.ExtractDocType(d.Name())) || !opt.accepts(d.Name()) {
				return nil
			}
			doc := buildDoc(baseDir, p, d, opt.includeContent)
			doc.SourceID = gs.ID
			doc.SourceKind = SourceKindGlobal
			doc.Global = true
			doc.ReadOnly = true
			docs = append(docs, doc)
			return nil
		})
		if walkErr != nil {
			// Never surface the raw walk error — it embeds an absolute path
			// (no-absolute-paths-in-mcp-errors.rule).
			return nil, fmt.Errorf("global source %q at %q is not readable", gs.ID, gs.Path)
		}
	}

	// Amortized cache hygiene: entries for files no longer enumerated (deleted,
	// renamed) are dropped once the cache outgrows the corpus. The key set is
	// built only when a sweep will actually run — it is a full-corpus allocation
	// for a sweep that almost never fires.
	if sharedScanCache.needsPrune(len(docs)) {
		seenPaths := make(map[string]bool, len(docs))
		for _, doc := range docs {
			seenPaths[filepath.Join(baseDir, filepath.FromSlash(doc.Path))] = true
		}
		sharedScanCache.prune(seenPaths)
	}

	return docs, nil
}

// buildDoc constructs a Document from a filesystem path during a walk.
// absPath is the absolute path to the file; baseDir is the project root used
// to compute the relative path stored in Document.Path. An unreadable file
// still yields a document populated from its filename-derived fields.
//
// Reads go through sharedScanCache: on a (mtime, size) hit the file is neither
// re-read nor re-parsed — the walk's DirEntry.Info() supplies the key, so a
// warm scan costs walk+stat instead of ReadFile+YAML per document.
func buildDoc(baseDir, absPath string, d fs.DirEntry, includeContent bool) Document {
	name := d.Name()

	docType := templates.DocumentType(templates.ExtractDocType(name))
	category := templates.CategoryForType(docType)
	slug := templates.ExtractSlug(name)

	var modTime time.Time
	var size int64
	haveInfo := false
	if info, infoErr := d.Info(); infoErr == nil {
		modTime = info.ModTime()
		size = info.Size()
		haveInfo = true
	}

	var fm templates.Frontmatter
	var content string
	readOK := false
	if haveInfo {
		if e, ok := sharedScanCache.lookup(absPath, modTime, size, includeContent); ok {
			fm, content, readOK = e.fm, e.content, true
		}
	}
	if !readOK {
		if data, readErr := os.ReadFile(absPath); readErr == nil {
			// A YAML parse error is deliberately ignored: a malformed document
			// is still indexed with empty metadata (search-documents.spec).
			fm, _, _ = templates.SplitDocument(data)
			content = string(data)
			readOK = true
			if haveInfo {
				e := docCacheEntry{modTime: modTime, size: size, fm: fm}
				if includeContent {
					e.content, e.hasContent = content, true
				}
				sharedScanCache.store(absPath, e)
			}
		}
	}

	relPath, _ := filepath.Rel(baseDir, absPath)
	relPath = filepath.ToSlash(relPath)

	doc := Document{
		Path:     relPath,
		Category: category,
		Type:     docType,
		Filename: name,
		Slug:     slug,
		Title:    fm.Title,
		Status:   fm.Status,
		// Cloned, not aliased: a cache hit would otherwise hand every Document
		// the same backing array as the cache, and the MCP worker pool makes any
		// later in-place sort a silent race.
		Tags:    slices.Clone(fm.Tags),
		ModTime: modTime,
	}
	if includeContent && readOK {
		doc.Content = content
	}
	return doc
}
