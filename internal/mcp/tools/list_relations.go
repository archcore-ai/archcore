package tools

import (
	"context"
	"encoding/json"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func NewListRelationsTool() mcp.Tool {
	return mcp.NewTool("list_relations",
		mcp.WithDescription(`List document relations stored in the sync manifest.

Use to see the full relation graph across all documents (omit "path"), or audit all relations for a specific document. Note: get_document also returns relations for a single document — use list_relations for a broader view or when you haven't loaded the document yet.

Optionally filter by a specific document path (with or without ".archcore/" prefix).`),
		mcp.WithString("path",
			mcp.Description("Optional document path to filter relations for. Returns all relations if omitted."),
		),
		mcp.WithTitleAnnotation("List Relations"),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func HandleListRelations(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := request.GetString("path", "")

		baseDir := root.Root(ctx)
		// A present-but-invalid manifest is a tool error — silently reporting an
		// empty graph hides real relation state (missing file = empty manifest).
		m, err := sharedManifestStore.load(baseDir)
		if err != nil {
			return errorResult(sanitizeError("loading manifest", err)), nil
		}

		var relations []sync.Relation
		if path == "" {
			relations = m.Relations
		} else {
			path = normalizeRelPath(path)
			out, in := m.RelationsFor(path)
			seen := make(map[string]bool)
			for _, r := range out {
				key := r.Source + "|" + r.Target + "|" + string(r.Type)
				if !seen[key] {
					relations = append(relations, r)
					seen[key] = true
				}
			}
			for _, r := range in {
				key := r.Source + "|" + r.Target + "|" + string(r.Type)
				if !seen[key] {
					relations = append(relations, r)
					seen[key] = true
				}
			}
		}

		if relations == nil {
			relations = []sync.Relation{}
		}

		result := map[string]any{"relations": relations}
		data, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(data)), nil
	}
}
