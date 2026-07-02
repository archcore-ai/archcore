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

	"archcore-cli/internal/config"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewUpdateDocumentTool returns the tool definition for update_document.
func NewUpdateDocumentTool() mcp.Tool {
	return mcp.NewTool("update_document",
		mcp.WithDescription(`Update an existing document in the .archcore/ knowledge base.

Before updating, confirm the intended changes with the user if possible.

Call list_documents first to get the document's path, then pass that path to get_document to review current content before updating.

You can update any combination of: title, status, content, or tags. Fields not provided are left unchanged. Pass an empty tags array to clear all tags.

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
			mcp.Description("New markdown body for the document. Replaces everything after the frontmatter. If omitted, the existing body is preserved."),
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
func HandleUpdateDocument(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		relPath, err := request.RequireString("path")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		// Validate path.
		relPath, err = validateArchcorePath(relPath)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		globals, gErr := config.LoadGlobals(baseDir)
		if gErr != nil {
			return errorResult("cannot verify global sources: settings.json is unreadable"), nil
		}
		if isReadOnlyGlobalPath(baseDir, relPath, globals) {
			return errorResult("cannot update a read-only global source document"), nil
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

		if newTitle == "" && newStatus == "" && newContent == "" && !tagsProvided {
			return errorResult("at least one of title, status, content, or tags must be provided"), nil
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
			return nil, fmt.Errorf("reading %s: %w", relPath, err)
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
		tags := existingFM.Tags
		if tagsProvided {
			tags = newTags
		}
		tags = normalizeTags(tags)

		// Reconstruct the file.
		fileContent := buildDocumentFile(title, status, tags, body)

		if err := os.WriteFile(absPath, []byte(fileContent), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}

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
