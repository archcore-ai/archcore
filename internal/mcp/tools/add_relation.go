package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func normalizeRelPath(p string) string {
	return strings.TrimPrefix(p, ".archcore/")
}

func NewAddRelationTool() mcp.Tool {
	return mcp.NewTool("add_relation",
		mcp.WithDescription(`Add a directed relation between two documents in the .archcore/ knowledge base.

Relations are stored in the sync manifest and represent semantic links between documents.

Relation types:
  related     — general association (e.g., two ADRs on the same topic)
  implements  — source implements what target specifies (e.g., plan implements prd)
  extends     — source builds upon target (e.g., rfc extends an existing adr)
  depends_on  — source requires target to proceed (e.g., plan depends_on adr)

For requirements-layer guidance (Sources vs Specifications, ISO cascade), see server instructions REQUIREMENTS LAYERS section.

Both source and target must be existing documents. Paths can be given with or without the ".archcore/" prefix.`),
		mcp.WithString("source",
			mcp.Description("Path to the source document (e.g. \"auth/jwt-strategy.adr.md\" or \".archcore/auth/jwt-strategy.adr.md\")"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description(`Path to the target document (e.g. "payments/stripe.adr.md" or ".archcore/payments/stripe.adr.md")`),
			mcp.Required(),
		),
		mcp.WithString("type",
			mcp.Description("Semantic type: related (general), implements (source fulfills target), extends (source builds on target), depends_on (source requires target)"),
			mcp.Required(),
			mcp.Enum(sync.ValidRelationTypes()...),
		),
		mcp.WithTitleAnnotation("Add Relation"),
		mcp.WithReadOnlyHintAnnotation(false),
	)
}

func HandleAddRelation(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		if source == target {
			return errorResult("source and target must be different documents"), nil
		}

		// Relations connect local documents only. A read-only global source — a
		// declared source or anything in the reserved .archcore/global/ tree —
		// must never be a relation endpoint, in either direction. Fail closed if
		// settings.json cannot be read so a corrupt config can't slip an edge in.
		globals, gErr := config.LoadGlobals(baseDir)
		if gErr != nil {
			return errorResult("cannot verify global sources: settings.json is unreadable"), nil
		}
		if isReadOnlyGlobalPath(baseDir, ".archcore/"+source, globals) ||
			isReadOnlyGlobalPath(baseDir, ".archcore/"+target, globals) {
			return errorResult("cannot add a relation involving a read-only global source document — relations connect local documents only"), nil
		}

		// Verify both documents exist.
		if _, err := ReadDocumentContent(baseDir, ".archcore/"+source); err != nil {
			return errorResult("source document not found: .archcore/" + source), nil
		}
		if _, err := ReadDocumentContent(baseDir, ".archcore/"+target); err != nil {
			return errorResult("target document not found: .archcore/" + target), nil
		}

		m, err := sync.LoadManifest(baseDir)
		if err != nil {
			return errorResult("loading manifest: " + err.Error()), nil
		}

		added := m.AddRelation(source, target, sync.RelationType(relType))

		if added {
			if err := sync.SaveManifest(baseDir, m); err != nil {
				return errorResult("saving manifest: " + err.Error()), nil
			}
		}

		result := map[string]any{
			"source": source,
			"target": target,
			"type":   relType,
			"added":  added,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
