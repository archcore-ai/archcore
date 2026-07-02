package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"archcore-cli/internal/config"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// DocumentRelation represents one side of a relation for enriched output.
type DocumentRelation struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// EnrichedDocument extends LocalDocument with relation information.
type EnrichedDocument struct {
	LocalDocument
	OutgoingRelations []DocumentRelation `json:"outgoing_relations,omitempty"`
	IncomingRelations []DocumentRelation `json:"incoming_relations,omitempty"`
}

// LocalDocument represents a document discovered in .archcore/.
type LocalDocument struct {
	Path       string              `json:"path"`                // relative: ".archcore/auth/jwt-strategy.adr.md"
	Category   templates.Category  `json:"category"`            // virtual: vision, knowledge, experience (derived from type)
	Type       string              `json:"type"`                // adr, rfc, rule...
	Filename   string              `json:"filename"`            // "jwt-strategy.adr.md"
	Slug       string              `json:"slug"`                // "jwt-strategy"
	Title      string              `json:"title,omitempty"`     // from frontmatter
	Status     templates.DocStatus `json:"status,omitempty"`    // from frontmatter
	Tags       []string            `json:"tags,omitempty"`      // from frontmatter
	ModTime    time.Time           `json:"mtime,omitzero"`      // file modification time
	Content    string              `json:"content,omitempty"`   // full markdown (optional)
	SourceID   string              `json:"source_id"`           // "local" or global source id
	SourceKind string              `json:"source_kind"`         // "local" or "global"
	Global     bool                `json:"global,omitempty"`    // true for mounted global sources
	ReadOnly   bool                `json:"read_only,omitempty"` // true for mounted global sources
}

// ScanDocuments discovers all .md files recursively inside .archcore/.
// Global sources declared in settings.json are scanned read-only.
func ScanDocuments(baseDir string) ([]LocalDocument, error) {
	return scanDocuments(baseDir, false)
}

// ScanDocumentsFull mirrors ScanDocuments but also populates the Content field
// for every document. Frontmatter is parsed from the same bytes read from disk,
// so this performs no extra I/O compared to ScanDocuments.
func ScanDocumentsFull(baseDir string) ([]LocalDocument, error) {
	return scanDocuments(baseDir, true)
}

// resolveGlobalPath resolves a global source path to an absolute directory.
// It delegates to config.ResolveGlobalPath so the scan, the read-path validation,
// and the MCP startup check all resolve paths identically.
func resolveGlobalPath(baseDir, gsPath string) string {
	return config.ResolveGlobalPath(baseDir, gsPath)
}

// ScanLocalDocuments discovers only the primary project's own documents, never
// touching declared global sources. Unlike ScanDocuments it cannot fail because a
// global is missing — it never reads one. CLI surfaces that operate on local
// documents only (status, SessionStart context) use this so an unreachable global
// degrades to local-only instead of blanking the whole result.
func ScanLocalDocuments(baseDir string) ([]LocalDocument, error) {
	return scanLocalDocuments(baseDir, false)
}

// scanLocalDocuments performs phase 1 of the scan: the primary's own documents.
// It skips the reserved global/ mount directory and any document that falls under
// a declared global source (surfaced read-only in phase 2 instead), so a document
// is never scanned as both local and global. A missing .archcore/ yields (nil, nil).
func scanLocalDocuments(baseDir string, includeContent bool) ([]LocalDocument, error) {
	archcoreDir := filepath.Join(baseDir, ".archcore")
	allGlobals := config.ReadGlobals(baseDir)
	var docs []LocalDocument

	err := templates.WalkArchcoreFilesSkipping(archcoreDir, []string{"global"}, func(p string, d fs.DirEntry) error {
		relPath, _ := filepath.Rel(baseDir, p)
		relPath = filepath.ToSlash(relPath)
		if isGlobalPath(baseDir, relPath, allGlobals) {
			return nil
		}
		doc := buildDoc(baseDir, p, d, includeContent)
		doc.SourceID = "local"
		doc.SourceKind = "local"
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

// scanDocuments walks .archcore/ in two phases:
//  1. Local documents — skips the global/ subdirectory entirely (scanLocalDocuments).
//  2. Global sources declared in settings.json — each source is walked
//     independently and documents are tagged read-only.
func scanDocuments(baseDir string, includeContent bool) ([]LocalDocument, error) {
	docs, err := scanLocalDocuments(baseDir, includeContent)
	if err != nil {
		return nil, err
	}

	// Phase 2: mounted global sources declared in settings.json.
	allGlobals := config.ReadGlobals(baseDir)
	seen := make(map[string]string, len(allGlobals)) // resolved dir -> first source id
	for _, gs := range allGlobals {
		globalDir := resolveGlobalPath(baseDir, gs.Path)
		if firstID, dup := seen[globalDir]; dup {
			return nil, fmt.Errorf("global sources %q and %q resolve to the same path %q", firstID, gs.ID, gs.Path)
		}
		seen[globalDir] = gs.ID
		if dirErr := config.CheckGlobalDir(baseDir, globalDir); dirErr != nil {
			return nil, errors.New(config.DescribeGlobalDirError(gs, dirErr))
		}
		walkErr := templates.WalkArchcoreFilesSkipping(globalDir, nil, func(p string, d fs.DirEntry) error {
			// Mount only recognized document types: a misconfigured path (e.g. a
			// parent directory) must not surface stray .md files as malformed docs.
			if !templates.IsValidType(templates.ExtractDocType(d.Name())) {
				return nil
			}
			doc := buildDoc(baseDir, p, d, includeContent)
			doc.SourceID = gs.ID
			doc.SourceKind = "global"
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

	return docs, nil
}

// buildDoc constructs a LocalDocument from a filesystem path during a walk.
// absPath is the absolute path to the file; baseDir is the project root used
// to compute the relative path stored in LocalDocument.Path. An unreadable file
// still yields a document populated from its filename-derived fields.
func buildDoc(baseDir, absPath string, d fs.DirEntry, includeContent bool) LocalDocument {
	name := d.Name()

	docType := templates.ExtractDocType(name)
	category := templates.CategoryForType(templates.DocumentType(docType))
	slug := templates.ExtractSlug(name)

	data, readErr := os.ReadFile(absPath)
	var fm templates.Frontmatter
	if readErr == nil {
		// A YAML parse error is deliberately ignored: a malformed document is
		// still indexed with empty metadata (search-documents.spec contract).
		fm, _, _ = templates.SplitDocument(data)
	}

	var modTime time.Time
	if info, infoErr := d.Info(); infoErr == nil {
		modTime = info.ModTime()
	}

	relPath, _ := filepath.Rel(baseDir, absPath)
	relPath = filepath.ToSlash(relPath)

	doc := LocalDocument{
		Path:     relPath,
		Category: category,
		Type:     docType,
		Filename: name,
		Slug:     slug,
		Title:    fm.Title,
		Status:   fm.Status,
		Tags:     fm.Tags,
		ModTime:  modTime,
	}
	if includeContent && readErr == nil {
		doc.Content = string(data)
	}
	return doc
}

// matchGlobal returns the id of the declared global source that contains relPath
// (relative to baseDir), and whether a match was found. Both relPath and each
// gs.Path are anchored to baseDir before comparison, so embedded, "../"-relative,
// and absolute global paths are all handled in the same coordinate space.
func matchGlobal(baseDir, relPath string, globals []config.GlobalSource) (string, bool) {
	target := filepath.ToSlash(filepath.Join(baseDir, relPath))
	for _, gs := range globals {
		prefix := filepath.ToSlash(resolveGlobalPath(baseDir, gs.Path))
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return gs.ID, true
		}
	}
	return "", false
}

// isGlobalPath reports whether relPath (relative to baseDir) falls under any
// declared global source. Used by the write handlers to reject mutations of
// read-only global documents.
func isGlobalPath(baseDir, relPath string, globals []config.GlobalSource) bool {
	_, ok := matchGlobal(baseDir, relPath, globals)
	return ok
}

// isReservedGlobalDir reports whether relPath sits in reserved global mount space:
// any directory named "global" under .archcore/, at any depth, or anything inside
// one. This matches the any-depth skip the local scan applies in
// WalkArchcoreFilesSkipping (skipDirs=["global"]) so the read scan and the write
// guard agree on what is reserved — a "global" segment hides a document from the
// scan AND makes it read-only, never one without the other. Read-only regardless of
// whether the directory is declared in settings.json. The match is on whole path
// segments, so a sibling like ".archcore/global-ish/" is not reserved.
func isReservedGlobalDir(relPath string) bool {
	rp := filepath.ToSlash(relPath)
	if !strings.HasPrefix(rp, ".archcore/") {
		return false
	}
	return slices.Contains(strings.Split(rp, "/"), "global")
}

// isReadOnlyGlobalPath reports whether relPath is read-only global space: either a
// declared global source or the reserved .archcore/global/ tree. Every write and
// relation guard uses this single predicate so "is this global?" has one answer.
func isReadOnlyGlobalPath(baseDir, relPath string, globals []config.GlobalSource) bool {
	return isReservedGlobalDir(relPath) || isGlobalPath(baseDir, relPath, globals)
}

// annotateSource fills SourceID, SourceKind, Global, and ReadOnly on doc by
// matching doc.Path against declared global sources in settings.json.
//
// A declared source is matched first so it keeps its own id (a global vendored
// in-tree under .archcore/global/<id> is both reserved AND declared, and must
// report <id>, not the sentinel). Content in the reserved global/ tree that is NOT
// declared (isReservedGlobalDir) is still read-only global space — the write guards
// treat it as read-only (isReadOnlyGlobalPath) — so it is annotated global with the
// reserved "__global__" source id, keeping the read label consistent with the write
// reality. The sentinel carries underscores, which globalIDRe forbids in a declared
// id, so it can never collide with a real source id. Everything else is local.
func annotateSource(doc *LocalDocument, baseDir string) {
	if id, ok := matchGlobal(baseDir, doc.Path, config.ReadGlobals(baseDir)); ok {
		doc.SourceID = id
		doc.SourceKind = "global"
		doc.Global = true
		doc.ReadOnly = true
		return
	}
	if isReservedGlobalDir(doc.Path) {
		doc.SourceID = "__global__"
		doc.SourceKind = "global"
		doc.Global = true
		doc.ReadOnly = true
		return
	}
	doc.SourceID = "local"
	doc.SourceKind = "local"
}

// ReadDocumentContent reads a single document fully from a relative path.
func ReadDocumentContent(baseDir, relPath string) (LocalDocument, error) {
	absPath := filepath.Join(baseDir, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return LocalDocument{}, err
	}

	filename := filepath.Base(relPath)
	docType := templates.ExtractDocType(filename)
	category := templates.CategoryForType(templates.DocumentType(docType))
	fm, _, _ := templates.SplitDocument(data)
	slug := templates.ExtractSlug(filename)

	var modTime time.Time
	if info, statErr := os.Stat(absPath); statErr == nil {
		modTime = info.ModTime()
	}

	return LocalDocument{
		Path:     relPath,
		Category: category,
		Type:     docType,
		Filename: filename,
		Slug:     slug,
		Title:    fm.Title,
		Status:   fm.Status,
		Tags:     fm.Tags,
		ModTime:  modTime,
		Content:  string(data),
	}, nil
}

// validateTags checks that every tag matches the required format.
func validateTags(tags []string) error {
	for _, tag := range tags {
		if !templates.TagRe.MatchString(tag) {
			hint := strings.ToLower(tag)
			if hint != tag && templates.TagRe.MatchString(hint) {
				return fmt.Errorf("invalid tag %q — did you mean %q?", tag, hint)
			}
			return fmt.Errorf("invalid tag %q — must be lowercase alphanumeric with hyphens, underscores, colons, or pipes (e.g. \"frontend\", \"team-platform\", \"team:payments\", \"payment_team\", \"some|flag\")", tag)
		}
	}
	return nil
}

// parseTags validates and normalizes a tag list. Returns nil for empty input.
func parseTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	if err := validateTags(tags); err != nil {
		return nil, err
	}
	return normalizeTags(tags), nil
}

// normalizeTags sorts and deduplicates tags. Returns nil for empty input.
// Always operates on a clone to avoid mutating the caller's slice.
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := slices.Clone(tags)
	slices.Sort(out)
	out = slices.Compact(out)
	return out
}

func errorResult(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(msg)
}

// sanitizeError builds a client-safe error message "<action>: <detail>". When
// err (anywhere in its chain) is a *fs.PathError or *os.LinkError, detail is a
// fixed I/O class instead of err.Error(), which embeds an absolute filesystem
// path (no-absolute-paths-in-mcp-errors.rule). Any other error keeps its own
// text — validation errors are built from relative manifest/settings keys and
// are safe and valuable diagnostics.
func sanitizeError(action string, err error) string {
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	if errors.As(err, &pathErr) || errors.As(err, &linkErr) {
		return action + ": " + describeIOClass(err)
	}
	return action + ": " + err.Error()
}

// describeIOClass maps an OS-level error to a short, path-free description.
func describeIOClass(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "file not found"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.Is(err, fs.ErrExist):
		return "file already exists"
	default:
		return "file system error"
	}
}

// buildDocumentFile reconstructs a full document file from frontmatter fields and body.
func buildDocumentFile(title string, status templates.DocStatus, tags []string, body string) string {
	var buf strings.Builder
	buf.WriteString("---\n")
	fmt.Fprintf(&buf, "title: %q\n", title)
	fmt.Fprintf(&buf, "status: %s\n", status)
	if len(tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range tags {
			fmt.Fprintf(&buf, "  - %q\n", tag)
		}
	}
	buf.WriteString("---\n\n")
	buf.WriteString(body)
	return buf.String()
}

// stripFrontmatter removes YAML frontmatter from content if present.
// This prevents duplicate frontmatter when callers (e.g. AI agents)
// include frontmatter in the content parameter despite the tool
// description specifying body-only content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		return content
	}
	_, body, _ := templates.SplitDocument([]byte(content))
	return body
}

// validateArchcorePath normalises and validates a document path.
// It returns the cleaned path or an error if the path is invalid.
//
// Uses path.Clean (POSIX, forward-slash) rather than filepath.Clean because on
// Windows filepath.Clean would re-introduce backslashes after ToSlash, breaking
// the subsequent ".archcore/" prefix check.
func validateArchcorePath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("invalid path: must be relative and within .archcore/")
	}
	relPath = filepath.ToSlash(relPath)
	if !strings.HasPrefix(relPath, ".archcore/") {
		return "", fmt.Errorf("invalid path: must start with \".archcore/\"")
	}
	cleaned := path.Clean(relPath)
	if strings.HasPrefix(cleaned, "..") || !strings.HasPrefix(cleaned, ".archcore/") {
		return "", fmt.Errorf("invalid path: must be relative and within .archcore/")
	}
	return cleaned, nil
}

// validateReadPath validates the path argument of the read tools (get_document).
//
// It accepts everything validateArchcorePath accepts — local documents and in-tree
// globals, all under the primary's .archcore/ — with identical behavior. It
// ADDITIONALLY accepts a document that resolves strictly inside a declared external
// global source (one whose path is "../…" or absolute, so its documents render with
// a leading ".." that validateArchcorePath rejects). Only reads of read-only globals
// take this branch; the write tools keep the strict validateArchcorePath and never
// reach it, so a "../" global stays unwritable and non-linkable.
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
func validateReadPath(baseDir, relPath string, globals []config.GlobalSource) (string, error) {
	// Local documents and in-tree globals: unchanged, strict validation.
	if cleaned, err := validateArchcorePath(relPath); err == nil {
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
