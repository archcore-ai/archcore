package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func NewRemoveRelationTool() mcp.Tool {
	return mcp.NewTool("remove_relation",
		mcp.WithDescription(`Remove a directed relation between two documents.

Use when a relation is no longer accurate — for example, when a document is superseded or a dependency is resolved.

Only local document paths are permitted; global sources and paths outside the project are rejected. The tool mutates the manifest only. A validation failure leaves the manifest unchanged. Missing local documents do not prevent removal of a stored relation.

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
			mcp.Description("Relation type to remove: related, implements, extends, depends_on, supports, contradicts, or supersedes"),
			mcp.Required(),
			mcp.Enum(sync.ValidRelationTypes()...),
		),
		mcp.WithTitleAnnotation("Remove Relation"),
		mcp.WithReadOnlyHintAnnotation(false),
	)
}

func HandleRemoveRelation(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		source, target, relType, refusal := parseRelationArgs(request)
		if refusal != nil {
			return refusal, nil
		}
		originalSource, originalTarget := source, target

		var removed bool
		baseDir := root.Root(ctx)
		source, target, refusal = guardRelationEndpoints(baseDir, source, target)
		if refusal != nil {
			return refusal, nil
		}

		if err := sharedManifestStore.mutate(baseDir, func(m *sync.Manifest) bool {
			removed = m.RemoveRelation(source, target, sync.RelationType(relType))
			// Older binaries stored uncleaned endpoints; their exact triples remain removable.
			if !removed && (source != originalSource || target != originalTarget) {
				removed = m.RemoveRelation(originalSource, originalTarget, sync.RelationType(relType))
				if removed {
					source, target = originalSource, originalTarget
				}
			}
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
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
