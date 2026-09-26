package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/internal/docs"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewUpdateDocumentTool returns the tool definition for update_document.
func NewUpdateDocumentTool() mcp.Tool {
	return mcp.NewTool("update_document",
		mcp.WithDescription(`Update an existing document in the .archcore/ knowledge base.

Before updating, confirm the intended changes with the user if possible.

Call list_documents first to get the document's path, then pass that path to get_document to review current content before updating.

You can update any combination of: title, status, tags, and either content or edits (not both). Fields not provided are left unchanged. Pass an empty tags array to clear all tags.

To change part of the body, pass edits instead of content: only the changed text is sent, not the whole body.

Returns: JSON with the path of the updated file, its type, category, title, status, and tags (when present).`),
		mcp.WithString("path",
			mcp.Description("Relative path to the document from the project root. Must be obtained from list_documents — do not construct this manually. Example: \".archcore/knowledge/use-postgres.adr.md\""),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("New title for the document frontmatter. If omitted, the existing title is preserved."),
		),
		mcp.WithString("status",
			mcp.Description("New document status. Valid values: draft, accepted, rejected."),
			mcp.Enum(templates.ValidStatusStrings()...),
		),
		mcp.WithString("content",
			mcp.Description("New markdown body for the document. Replaces everything after the frontmatter. If omitted, the existing body is preserved. Cannot be combined with edits."),
		),
		mcp.WithArray("edits",
			mcp.Description("Exact-match replacements applied in order to the markdown body (frontmatter excluded). Each old_string must occur exactly once in the body as the previous edits left it. If any edit fails, nothing is written. Cannot be combined with content."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"old_string": map[string]any{"type": "string", "description": "Exact text to replace, copied from get_document. Must match exactly once."},
					"new_string": map[string]any{"type": "string", "description": "Replacement text. An empty string deletes old_string."},
				},
				"required": []string{"old_string", "new_string"},
			}),
		),
		mcp.WithArray("tags",
			mcp.Description(`New tags for the document. Format per server instructions (TAGS section), e.g. "frontend", "team:payments". Pass an empty array to clear all tags; omit to preserve existing.`),
			mcp.WithStringItems(),
		),
		mcp.WithTitleAnnotation("Update Document"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

// HandleUpdateDocument handles the update_document tool call.
func HandleUpdateDocument(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		relPath, err := request.RequireString("path")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		baseDir := root.Root(ctx)
		// Validate path.
		globals, guardFail := loadGlobalsFailClosed(baseDir)
		if guardFail != nil {
			return guardFail, nil
		}
		relPath, err = guardWritablePath(baseDir, relPath, globals)
		if err != nil {
			switch {
			case errors.Is(err, errPathReadOnlyGlobal):
				return errorResult("cannot update a read-only global source document"), nil
			case errors.Is(err, errPathNotDocument):
				return errorResult("invalid path: not a document — only .md document files can be updated"), nil
			default:
				return errorResult(err.Error()), nil
			}
		}

		// Require at least one update field.
		newTitle := request.GetString("title", "")
		newStatus := templates.DocStatus(request.GetString("status", ""))
		newContent := request.GetString("content", "")

		var newTags []string
		tagsProvided := false
		if _, ok := request.GetArguments()["tags"]; ok {
			tagsProvided = true
			var tagErr error
			newTags, tagErr = parseTags(request.GetStringSlice("tags", nil))
			if tagErr != nil {
				return errorResult(tagErr.Error()), nil
			}
		}

		var edits []bodyEdit
		editsRaw, editsProvided := request.GetArguments()["edits"]
		if editsProvided {
			if newContent != "" {
				return errorResult("pass either content or edits, not both"), nil
			}
			if edits, err = parseEdits(editsRaw); err != nil {
				return errorResult(err.Error()), nil
			}
		}

		if newTitle == "" && newStatus == "" && newContent == "" && !tagsProvided && !editsProvided {
			return errorResult("at least one of title, status, content, edits, or tags must be provided"), nil
		}

		if newStatus != "" && !templates.IsValidStatus(newStatus) {
			return errorResult(fmt.Sprintf("invalid status %q (valid: %s)", newStatus, strings.Join(templates.ValidStatusStrings(), ", "))), nil
		}

		// Read existing file.
		absPath := filepath.Join(baseDir, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return errorResult(fmt.Sprintf("document not found: %s", relPath)), nil
			}
			return errorResult(sanitizeError("reading "+relPath, err)), nil
		}

		// Parse existing document. A broken frontmatter block must fail the
		// update: rebuilding the file from a zero Frontmatter would silently
		// erase the document's title, status, and tags.
		existingFM, existingBody, fmErr := templates.SplitDocument(data)
		if fmErr != nil {
			return errorResult(fmt.Sprintf("cannot update %s: existing frontmatter is not valid YAML — fix the file manually before updating", relPath)), nil
		}

		// Apply updates.
		title := existingFM.Title
		if newTitle != "" {
			title = newTitle
		}
		status := existingFM.Status
		if newStatus != "" {
			status = newStatus
		}
		body := existingBody
		if newContent != "" {
			body = stripFrontmatter(newContent)
		}
		if editsProvided {
			if body, err = applyEdits(body, edits); err != nil {
				return errorResult(err.Error()), nil
			}
		}
		// Preserved tags keep their on-disk order — normalizing them here
		// would reorder the user's tags on a title-only update. Provided tags
		// are already validated and normalized by parseTags.
		tags := existingFM.Tags
		if tagsProvided {
			tags = newTags
		}

		// Reconstruct the file.
		existingFM.Title = title
		existingFM.Status = status
		existingFM.Tags = tags
		fileContent, err := buildDocumentFile(existingFM, body)
		if err != nil {
			// YAML errors can quote document content; omit it from the refusal
			// to satisfy document-update-frontmatter.spec's path-disclosure constraint.
			return errorResult(fmt.Sprintf("cannot update %s: frontmatter cannot be preserved — repair frontmatter manually before updating", relPath)), nil
		}

		if err := writeFileAtomic(absPath, []byte(fileContent)); err != nil {
			return errorResult(sanitizeError("writing "+relPath, err)), nil
		}
		docs.InvalidateCache(absPath)

		// Derive category from document type, not directory.
		filename := filepath.Base(relPath)
		docType := templates.ExtractDocType(filename)
		category := templates.CategoryForType(templates.DocumentType(docType))

		result := map[string]any{
			"path":     relPath,
			"category": category,
			"type":     docType,
			"title":    title,
			"status":   status,
		}
		if len(tags) > 0 {
			result["tags"] = tags
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

type bodyEdit struct{ old, new string }

func parseEdits(raw any) ([]bodyEdit, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, errors.New("edits must be a non-empty array of {old_string, new_string} objects")
	}
	edits := make([]bodyEdit, 0, len(items))
	for i, item := range items {
		m, _ := item.(map[string]any)
		oldText, _ := m["old_string"].(string)
		if oldText == "" {
			return nil, fmt.Errorf("edits[%d]: old_string must be a non-empty string", i)
		}
		newText, isString := m["new_string"].(string)
		if !isString {
			return nil, fmt.Errorf("edits[%d]: new_string must be a string", i)
		}
		// get_document returns the file's raw bytes, while SplitDocument hands
		// applyEdits a body with CRLF folded to LF; fold the edit the same way.
		edits = append(edits, bodyEdit{
			old: strings.ReplaceAll(oldText, "\r\n", "\n"),
			new: strings.ReplaceAll(newText, "\r\n", "\n"),
		})
	}
	return edits, nil
}

// Each old text must match exactly once in the body as the previous edits left
// it, so an edit never lands on the wrong occurrence. The caller writes nothing
// on error, so a failed batch leaves the document untouched.
func applyEdits(body string, edits []bodyEdit) (string, error) {
	for i, e := range edits {
		switch n := strings.Count(body, e.old); n {
		case 1:
			body = strings.Replace(body, e.old, e.new, 1)
		case 0:
			return "", fmt.Errorf("edits[%d]: old_string not found in the document body — call get_document and copy the text exactly", i)
		default:
			return "", fmt.Errorf("edits[%d]: old_string matches %d places — include more surrounding text so it matches once", i, n)
		}
	}
	return body, nil
}
