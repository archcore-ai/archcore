// Package tools implements the handlers behind the MCP server's document tools.
// Each file is one tool: its schema constructor and its handler. Document
// reading, writing and path guarding are not implemented here — they come from
// internal/docs through the aliases in docs_bridge.go, so a path the MCP tools
// refuse cannot be reached by editing the file directly.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func NewAddRelationTool() mcp.Tool {
	return mcp.NewTool("add_relation",
		mcp.WithDescription(`Add a directed relation between two documents in the .archcore/ knowledge base.

Before calling, read both documents and call list_relations. Identify the statements that justify the type and direction. For related, name the joint reading task (the concrete task that requires both documents). A shared topic, folder, or tag alone is insufficient. IF the specific type is uncertain, THEN do not use related as a fallback. Do not add a reverse related or an extra related beside a more specific edge solely for navigation. If no supported claim exists, leave the pair unlinked. The tool validates structure, not semantic justification.

Relation axes: structural (related, implements, extends, depends_on), evidential (supports, contradicts), temporal (supersedes).

Relation types:
  related     — general association for a concrete joint reading task (e.g., a guide and its companion reference)
  implements  — source implements what target specifies (e.g., plan implements prd)
  extends     — source builds upon target (e.g., rfc extends an existing adr)
  depends_on  — source requires target to proceed (e.g., plan depends_on adr)
  supports    — material points to the statement it backs
  contradicts — challenger points to the statement it disputes
  supersedes  — newer document points to the older document it replaces

For requirements-layer guidance (Sources vs Specifications, ISO cascade), see server instructions REQUIREMENTS LAYERS section.

Both source and target must be distinct existing local documents. Global sources cannot be relation endpoints. Paths must remain inside the project. The tool mutates the manifest only; it does not change document status or resolve contradictions. An invalid input or manifest leaves the manifest unchanged. Successful calls return whether the relation was added, plus warnings that name a reverse related edge or related beside a more specific edge. A warning never blocks the write. Paths can be given with or without the ".archcore/" prefix.`),
		mcp.WithString("source",
			mcp.Description("Path to the source document (e.g. \"auth/jwt-strategy.adr.md\" or \".archcore/auth/jwt-strategy.adr.md\")"),
			mcp.Required(),
		),
		mcp.WithString("target",
			mcp.Description(`Path to the target document (e.g. "payments/stripe.adr.md" or ".archcore/payments/stripe.adr.md")`),
			mcp.Required(),
		),
		mcp.WithString("type",
			mcp.Description("Semantic type: related (general), implements (source fulfills target), extends (source builds on target), depends_on (source requires target), supports (material backs target), contradicts (challenger disputes target), supersedes (newer replaces older)"),
			mcp.Required(),
			mcp.Enum(sync.ValidRelationTypes()...),
		),
		mcp.WithTitleAnnotation("Add Relation"),
		mcp.WithReadOnlyHintAnnotation(false),
	)
}

func HandleAddRelation(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		if filepath.IsAbs(source) || filepath.IsAbs(target) {
			return errorResult("relation paths must be relative and within .archcore/"), nil
		}
		if strings.Contains(source, "..") {
			return errorResult("source path must not contain '..'"), nil
		}
		if strings.Contains(target, "..") {
			return errorResult("target path must not contain '..'"), nil
		}

		baseDir := root.Root(ctx)
		// Relations connect local documents only. A read-only global source — a
		// declared source or anything in the reserved .archcore/global/ tree —
		// must never be a relation endpoint, in either direction. Fail closed if
		// settings.json cannot be read so a corrupt config can't slip an edge in.
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
					return errorResult("cannot add a relation involving a read-only global source document — relations connect local documents only"), nil
				case errors.Is(err, errPathNotDocument):
					return errorResult("relation endpoints must be .md document files"), nil
				default:
					return errorResult(err.Error()), nil
				}
			}
			endpoints[i] = normalizeRelPath(cleaned)
		}
		source, target = endpoints[0], endpoints[1]
		if source == target {
			return errorResult("source and target must be different documents"), nil
		}

		// Verify both documents exist.
		if _, err := readDocumentContent(baseDir, ".archcore/"+source); err != nil {
			return errorResult("source document not found: .archcore/" + source), nil
		}
		if _, err := readDocumentContent(baseDir, ".archcore/"+target); err != nil {
			return errorResult("target document not found: .archcore/" + target), nil
		}

		var added bool
		var overlaps []sync.RelationOverlap
		if err := sharedManifestStore.mutate(baseDir, func(m *sync.Manifest) bool {
			added = m.AddRelation(source, target, sync.RelationType(relType))
			if added {
				overlaps = m.RelationOverlaps(source, target, sync.RelationType(relType))
			}
			return added
		}); err != nil {
			return errorResult(sanitizeError("updating manifest", err)), nil
		}

		result := map[string]any{
			"source": source,
			"target": target,
			"type":   relType,
			"added":  added,
		}
		if len(overlaps) > 0 {
			result["warnings"] = relationWarnings(overlaps)
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

type relationWarning struct {
	Code    sync.RelationOverlap `json:"code"`
	Message string               `json:"message"`
}

func relationWarnings(overlaps []sync.RelationOverlap) []relationWarning {
	warnings := make([]relationWarning, len(overlaps))
	for i, overlap := range overlaps {
		warnings[i] = relationWarning{Code: overlap, Message: overlapMessage(overlap)}
	}
	return warnings
}

func overlapMessage(overlap sync.RelationOverlap) string {
	switch overlap {
	case sync.OverlapReverseRelated:
		return "a related edge already runs from target to source, and get_document shows it on both documents; call list_relations and keep both edges only when each records a distinct claim"
	case sync.OverlapRelatedBesideSpecific:
		return "this pair now holds related beside a more specific edge; call list_relations and keep related only when it records a claim that the specific edge does not"
	}
	return ""
}
