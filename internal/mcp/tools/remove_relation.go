package tools

import (
	"context"
	"encoding/json"
	"strings"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func NewRemoveRelationTool() mcp.Tool {
	return mcp.NewTool("remove_relation",
		mcp.WithDescription(`Remove a directed relation between two documents.

Use when a relation is no longer accurate — for example, when a document is superseded or a dependency is resolved.

Both source and target paths can be given with or without the ".archcore/" prefix. The relation type must match exactly.

Returns: {"removed": true} if found and removed, {"removed": false} if not found.`),
		mcp.WithString("source",
			mcp.Description("Path to the source document"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description("Path to the target document"),
			mcp.Required(),
		),
		mcp.WithString("type",
			mcp.Description("Relation type to remove"),
			mcp.Required(),
			mcp.Enum(sync.ValidRelationTypes()...),
		),
		mcp.WithTitleAnnotation("Remove Relation"),
		mcp.WithReadOnlyHintAnnotation(false),
	)
}

func HandleRemoveRelation(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		source, err := request.RequireString("source")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		target, err := request.RequireString("target")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		relType, err := request.RequireString("type")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		if !sync.IsValidRelationType(relType) {
			return errorResult("invalid relation type: " + relType), nil
		}

		source = normalizeRelPath(source)
		target = normalizeRelPath(target)

		if strings.Contains(source, "..") {
			return errorResult("source path must not contain '..'"), nil
		}
		if strings.Contains(target, "..") {
			return errorResult("target path must not contain '..'"), nil
		}

		var removed bool
		baseDir := root.Root(ctx)

		if err := sharedManifestStore.mutate(baseDir, func(m *sync.Manifest) bool {
			removed = m.RemoveRelation(source, target, sync.RelationType(relType))
			return removed
		}); err != nil {
			return errorResult(sanitizeError("updating manifest", err)), nil
		}

		result := map[string]any{
			"source":  source,
			"target":  target,
			"type":    relType,
			"removed": removed,
		}
		data, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(data)), nil
	}
}
