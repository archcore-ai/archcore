package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/templates"
)

// Frontmatter holds the parsed frontmatter fields sent per file.
type Frontmatter struct {
	Title  string              `json:"title"`
	Status templates.DocStatus `json:"status,omitempty"`
	Tags   []string            `json:"tags,omitempty"`
}

// FileEntry is a single file in the sync request body.
type FileEntry struct {
	Path        string             `json:"path"`
	SHA256      string             `json:"sha256"`
	DocType     string             `json:"doc_type,omitempty"`
	Category    templates.Category `json:"category,omitempty"`
	Frontmatter Frontmatter        `json:"frontmatter"`
	Content     string             `json:"content"`
}

// Payload is the full request body for POST /sync.
type Payload struct {
	ProjectID   *int        `json:"project_id,omitempty"`
	ProjectName *string     `json:"project_name,omitempty"`
	RepoURL     *string     `json:"repo_url,omitempty"`
	Created     []FileEntry `json:"created"`
	Modified    []FileEntry `json:"modified"`
	Deleted     []string    `json:"deleted"`
}

// ParseFrontmatter extracts title, status, and tags from raw document content.
func ParseFrontmatter(content string) (title string, status templates.DocStatus, tags []string) {
	fm, _ := templates.SplitDocument([]byte(content))
	return fm.Title, fm.Status, fm.Tags
}

// validateRelPath checks that a relative path does not escape the base directory.
func validateRelPath(relPath string) error {
	cleaned := filepath.Clean(relPath)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("invalid path %q: must be relative and within .archcore/", relPath)
	}
	return nil
}

// BuildPayload constructs the sync payload from diff entries.
// It reads file content for created/modified files, parses frontmatter,
// and collects deleted paths.
func BuildPayload(baseDir string, entries []DiffEntry) (*Payload, error) {
	payload := &Payload{
		Created:  []FileEntry{},
		Modified: []FileEntry{},
		Deleted:  []string{},
	}

	for _, e := range entries {
		if e.Action == ActionUnchanged {
			continue
		}

		if err := validateRelPath(e.RelPath); err != nil {
			return nil, err
		}

		switch e.Action {
		case ActionCreated, ActionModified:
			absPath := filepath.Join(baseDir, ".archcore", e.RelPath)
			content, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("reading %s for sync payload: %w", e.RelPath, err)
			}

			title, status, tags := ParseFrontmatter(string(content))

			if status != "" && !templates.IsValidStatus(status) {
				return nil, fmt.Errorf("file %s has invalid status %q (valid values: %s)",
					e.RelPath, status, strings.Join(templates.ValidStatusStrings(), ", "))
			}

			filename := filepath.Base(e.RelPath)
			docType := templates.ExtractDocType(filename)
			category := templates.CategoryForType(templates.DocumentType(docType))

			fe := FileEntry{
				Path:    e.RelPath,
				SHA256:  e.Hash,
				DocType: docType,
				Category: category,
				Frontmatter: Frontmatter{
					Title:  title,
					Status: status,
					Tags:   tags,
				},
				Content: string(content),
			}

			if e.Action == ActionCreated {
				payload.Created = append(payload.Created, fe)
			} else {
				payload.Modified = append(payload.Modified, fe)
			}

		case ActionDeleted:
			payload.Deleted = append(payload.Deleted, e.RelPath)
		}
	}

	return payload, nil
}
