package tools

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

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
		originalSource, originalTarget := source, target

		if filepath.IsAbs(source) || filepath.IsAbs(target) {
			return errorResult("relation paths must be relative and within .archcore/"), nil
		}
		if strings.Contains(source, "..") {
			return errorResult("source path must not contain '..'"), nil
		}
		if strings.Contains(target, "..") {
			return errorResult("target path must not contain '..'"), nil
		}

		var removed bool
		baseDir := root.Root(ctx)
		globals, guardFail := loadGlobalsFailClosed(baseDir)
		if guardFail != nil {
			return guardFail, nil
		}
		endpoints := []string{source, target}
		for i, endpoint := range endpoints {
			cleaned, err := guardWritablePath(baseDir, ".archcore/"+endpoint, globals)
			if err != nil {
				switch {
				case errors.Is(err, errPathReadOnlyGlobal):
					return errorResult("cannot remove a relation involving a read-only global source document — relations connect local documents only"), nil
				case errors.Is(err, errPathNotDocument):
					return errorResult("relation endpoints must be .md document files"), nil
				default:
					return errorResult(err.Error()), nil
				}
			}
			endpoints[i] = normalizeRelPath(cleaned)
		}
		source, target = endpoints[0], endpoints[1]

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
		data, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(data)), nil
	}
}
