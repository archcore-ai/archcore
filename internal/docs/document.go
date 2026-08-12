// Package docs owns the .archcore/ document domain: the document model, the
// filesystem scan, the global-source predicates, and the path guards.
//
// It carries no MCP dependency. The MCP tool layer (internal/mcp/tools) wraps it
// with protocol concerns, and the hook commands (cmd) use it directly — a hook
// enforces the same write guard and reads the same corpus as an MCP tool, so both
// must share one implementation rather than two that drift.
package docs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"archcore-cli/templates"
)

// SourceKind says whether a document is the project's own or comes from a
// mounted read-only source. A closed set, so it carries a type (§G).
type SourceKind string

// Source annotation vocabulary (wire values; see global-sources.spec §4).
const (
	SourceKindLocal  SourceKind = "local"
	SourceKindGlobal SourceKind = "global"
)

// SourceIDReserved marks undeclared content in the reserved global/ tree.
// Underscores are invalid in a declared id, so it can never collide. It is a
// source *id*, not a kind, so it stays a plain string.
const SourceIDReserved = "__global__"

// DocumentRelation represents one side of a relation for enriched output.
type DocumentRelation struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// EnrichedDocument extends Document with relation information.
type EnrichedDocument struct {
	Document
	OutgoingRelations []DocumentRelation `json:"outgoing_relations,omitempty"`
	IncomingRelations []DocumentRelation `json:"incoming_relations,omitempty"`
}

// Document represents a document discovered in .archcore/.
type Document struct {
	Path       string                 `json:"path"`                // relative: ".archcore/auth/jwt-strategy.adr.md"
	Category   templates.Category     `json:"category"`            // virtual: vision, knowledge, experience (derived from type)
	Type       templates.DocumentType `json:"type"`                // adr, rfc, rule...
	Filename   string                 `json:"filename"`            // "jwt-strategy.adr.md"
	Slug       string                 `json:"slug"`                // "jwt-strategy"
	Title      string                 `json:"title,omitempty"`     // from frontmatter
	Status     templates.DocStatus    `json:"status,omitempty"`    // from frontmatter
	Tags       []string               `json:"tags,omitempty"`      // from frontmatter
	ModTime    time.Time              `json:"mtime,omitzero"`      // file modification time
	Content    string                 `json:"content,omitempty"`   // full markdown (optional)
	SourceID   string                 `json:"source_id"`           // "local" or global source id
	SourceKind SourceKind             `json:"source_kind"`         // "local" or "global"
	Global     bool                   `json:"global,omitempty"`    // true for mounted global sources
	ReadOnly   bool                   `json:"read_only,omitempty"` // true for mounted global sources
}

// IsLocal reports whether the document belongs to the primary project rather
// than to a mounted global source.
func (d Document) IsLocal() bool { return d.SourceKind == SourceKindLocal }

// InAgentContext reports whether the document belongs in context pushed at an
// agent. A rejected document is excluded from the push; the read tools still
// return it (session-start-context.spec behavior 4).
func (d Document) InAgentContext() bool {
	return d.Status != templates.StatusRejected
}

// NormalizeRelPath strips the ".archcore/" prefix, converting a tool-facing
// document path to the manifest-relative form relations are stored in.
func NormalizeRelPath(p string) string {
	return strings.TrimPrefix(p, ".archcore/")
}

// ReadDocumentContent reads a single document fully from a relative path.
func ReadDocumentContent(baseDir, relPath string) (Document, error) {
	absPath := filepath.Join(baseDir, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Document{}, err
	}

	filename := filepath.Base(relPath)
	docType := templates.DocumentType(templates.ExtractDocType(filename))
	category := templates.CategoryForType(docType)
	fm, _, _ := templates.SplitDocument(data)
	slug := templates.ExtractSlug(filename)

	var modTime time.Time
	if info, statErr := os.Stat(absPath); statErr == nil {
		modTime = info.ModTime()
	}

	return Document{
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

// WriteFileAtomic writes data to absPath via a temp file + rename so a crash
// mid-write can never leave a truncated document (mirrors sync.SaveManifest),
// then drops the cached parse.
//
// The invalidation lives here rather than at the call sites: a write that
// leaves a stale entry behind makes a long-lived server serve the old parse
// until the mtime moves, and the pairing is only enforceable where the write is.
func WriteFileAtomic(absPath string, data []byte) error {
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, absPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	InvalidateCache(absPath)
	return nil
}
