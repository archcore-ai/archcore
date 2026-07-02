package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"archcore-cli/internal/config"
	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewGetDocumentTool returns the tool definition for get_document.
func NewGetDocumentTool() mcp.Tool {
	return mcp.NewTool("get_document",
		mcp.WithDescription(`Read the full content of a single .archcore/ document by its file path.

Call this tool AFTER list_documents has returned a valid path. Do not guess or construct paths — only use paths returned by list_documents.

Returns: the document's YAML frontmatter (title, type, status), its full Markdown body, and any outgoing_relations and incoming_relations from the knowledge graph.

Use this tool when you need to:
- Read the reasoning or content of a specific document
- Verify what a document says before creating a related one
- Retrieve a document to summarize or reference in a response`),
		mcp.WithString("path",
			mcp.Description(`Relative path to the document from the project root. Must be obtained from list_documents — do not construct this manually. Example: ".archcore/knowledge/use-postgres.adr.md"`),
			mcp.Required(),
		),
		mcp.WithTitleAnnotation("Get Document"),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

// HandleGetDocument handles the get_document tool call.
func HandleGetDocument(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		reqPath, err := request.RequireString("path")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		// Validate path safety. Reads additionally allow a document that resolves
		// inside a declared read-only external global source; the write tools keep
		// the strict validateArchcorePath and never reach this relaxation.
		cleanPath, err := validateReadPath(baseDir, reqPath, config.ReadGlobals(baseDir))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return errorResult("document not found: " + reqPath), nil
			}
			return errorResult(err.Error()), nil
		}

		doc, err := ReadDocumentContent(baseDir, cleanPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return errorResult("document not found: " + cleanPath), nil
			}
			return errorResult(sanitizeError("reading "+cleanPath, err)), nil
		}

		// Annotate source metadata based on path and declared globals.
		annotateSource(&doc, baseDir)

		enriched := EnrichedDocument{LocalDocument: doc}

		// Try to load relations from manifest.
		relPath := normalizeRelPath(cleanPath)
		if m, mErr := sync.LoadManifest(baseDir); mErr == nil {
			outgoing, incoming := m.RelationsFor(relPath)
			for _, r := range outgoing {
				enriched.OutgoingRelations = append(enriched.OutgoingRelations, DocumentRelation{
					Path: ".archcore/" + r.Target,
					Type: string(r.Type),
				})
			}
			for _, r := range incoming {
				enriched.IncomingRelations = append(enriched.IncomingRelations, DocumentRelation{
					Path: ".archcore/" + r.Source,
					Type: string(r.Type),
				})
			}
		}

		data, err := json.Marshal(enriched)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
