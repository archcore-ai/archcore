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
	Path     string              `json:"path"`              // relative: ".archcore/auth/jwt-strategy.adr.md"
	Category templates.Category  `json:"category"`          // virtual: vision, knowledge, experience (derived from type)
	Type     string              `json:"type"`              // adr, rfc, rule...
	Filename string              `json:"filename"`          // "jwt-strategy.adr.md"
	Slug     string              `json:"slug"`              // "jwt-strategy"
	Title    string              `json:"title,omitempty"`   // from frontmatter
	Status   templates.DocStatus `json:"status,omitempty"`  // from frontmatter
	Tags     []string            `json:"tags,omitempty"`    // from frontmatter
	ModTime  time.Time           `json:"mtime,omitzero"`    // file modification time
	Content  string              `json:"content,omitempty"` // full markdown (optional)
}

// ScanDocuments discovers all .md files recursively inside .archcore/.
func ScanDocuments(baseDir string) ([]LocalDocument, error) {
	return scanDocuments(baseDir, false)
}

// ScanDocumentsFull mirrors ScanDocuments but also populates the Content field
// for every document. Frontmatter is parsed from the same bytes read from disk,
// so this performs no extra I/O compared to ScanDocuments.
func ScanDocumentsFull(baseDir string) ([]LocalDocument, error) {
	return scanDocuments(baseDir, true)
}

// scanDocuments walks .archcore/ once. When includeContent is false the
// frontmatter parse is still fed from the full-file read (same as the previous
// behavior), but the body is discarded.
func scanDocuments(baseDir string, includeContent bool) ([]LocalDocument, error) {
	archcoreDir := filepath.Join(baseDir, ".archcore")
	var docs []LocalDocument

	err := templates.WalkArchcoreFiles(archcoreDir, func(path string, d fs.DirEntry) error {
		name := d.Name()

		docType := templates.ExtractDocType(name)
		category := templates.CategoryForType(templates.DocumentType(docType))
		slug := templates.ExtractSlug(name)

		data, readErr := os.ReadFile(path)
		var fm templates.Frontmatter
		if readErr == nil {
			fm, _ = templates.SplitDocument(data)
		}

		var modTime time.Time
		if info, infoErr := d.Info(); infoErr == nil {
			modTime = info.ModTime()
		}

		relPath, _ := filepath.Rel(baseDir, path)
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
		docs = append(docs, doc)
		return nil
	})

	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return docs, err
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
	fm, _ := templates.SplitDocument(data)
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
	_, body := templates.SplitDocument([]byte(content))
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
