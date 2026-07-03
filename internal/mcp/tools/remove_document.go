package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewRemoveDocumentTool returns the tool definition for remove_document.
func NewRemoveDocumentTool() mcp.Tool {
	return mcp.NewTool("remove_document",
		mcp.WithDescription(`Remove a document from the .archcore/ knowledge base and clean up all its relations.

This operation is permanent — the file is deleted from disk and cannot be recovered.
All relations referencing this document (both outgoing and incoming) are removed
automatically from the manifest.

Before calling this tool: confirm the deletion with the user explicitly. Do not delete
a document based on an inferred intent — always ask.

Prefer update_document over remove_document when the document's history should be preserved:
- A decision is outdated → change status to "rejected" (keeps history)
- A plan is abandoned → change status to "rejected"
- Only delete when the document is genuinely wrong, duplicated, or was created by mistake.

Call list_documents first to get the document's path. Optionally call get_document to
review its content and relations before deleting.

Returns: JSON with path, title, type, category, and relations_removed count.`),
		mcp.WithString("path",
			mcp.Description("Relative path to the document from the project root. Must be obtained from list_documents — do not construct this manually. Example: \".archcore/knowledge/use-postgres.adr.md\""),
			mcp.Required(),
		),
		mcp.WithTitleAnnotation("Remove Document"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
	)
}

// HandleRemoveDocument handles the remove_document tool call.
func HandleRemoveDocument(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		relPath, err := request.RequireString("path")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		// Validate path.
		globals, guardFail := loadGlobalsFailClosed(baseDir)
		if guardFail != nil {
			return guardFail, nil
		}
		relPath, err = guardWritablePath(baseDir, relPath, globals)
		if err != nil {
			switch {
			case errors.Is(err, errPathReadOnlyGlobal):
				return errorResult("cannot remove a read-only global source document"), nil
			case errors.Is(err, errPathNotDocument):
				return errorResult("invalid path: not a document — only .md document files can be removed"), nil
			default:
				return errorResult(err.Error()), nil
			}
		}

		// Read file metadata before deletion.
		doc, err := ReadDocumentContent(baseDir, relPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return errorResult(fmt.Sprintf("document not found: %s", relPath)), nil
			}
			return errorResult(sanitizeError("reading "+relPath, err)), nil
		}

		// Delete the file.
		absPath := filepath.Join(baseDir, relPath)
		if err := os.Remove(absPath); err != nil {
			return errorResult(sanitizeError("removing "+relPath, err)), nil
		}
		sharedScanCache.invalidate(absPath)

		// Clean up relations from manifest. relations_removed counts only the
		// deleted document's own edges; CleanupRelations still opportunistically
		// drops any other stale edge, but those are hygiene, not this deletion.
		relationsRemoved := 0
		archcoreDir := filepath.Join(baseDir, ".archcore")
		docRelPath := normalizeRelPath(relPath)
		if err := sharedManifestStore.mutate(baseDir, func(m *sync.Manifest) bool {
			out, in := m.RelationsFor(docRelPath)
			relationsRemoved = len(out) + len(in)
			return m.CleanupRelations(archcoreDir) > 0
		}); err != nil {
			return errorResult(sanitizeError("file deleted but failed to update manifest", err)), nil
		}

		result := map[string]any{
			"path":              relPath,
			"title":             doc.Title,
			"type":              doc.Type,
			"category":          doc.Category,
			"relations_removed": relationsRemoved,
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
