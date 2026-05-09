package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewListDocumentsTool returns the tool definition for list_documents.
func NewListDocumentsTool() mcp.Tool {
	return mcp.NewTool("list_documents",
		mcp.WithDescription(`Discover and filter documents in the .archcore/ knowledge base.

Call this tool FIRST before reading or creating any document. Use it to:
- Check whether a document on a given topic already exists (prevents duplicates)
- Get valid file paths required by get_document
- Browse what documentation is available by type, category, or status

Returns: a JSON array of documents, each with path, title, type, category, status, and tags (when present). Returns an empty array if no documents match.

Use the returned paths directly as input to get_document. Do not construct paths manually.`),
		mcp.WithArray("types",
			mcp.Description("Filter by one or more document types. Valid values: adr, rfc, rule, guide, doc, spec, prd, idea, plan, task-type, cpat, mrd, brd, urd, brs, strs, syrs, srs. Example: [\"adr\", \"rule\"] returns only decision records and standards."),
			mcp.WithStringItems(),
		),
		mcp.WithString("category",
			mcp.Description(`Filter by virtual category (derived from document type, not directory). Use "knowledge" for decisions/standards/guides/specs/docs/proposals, "vision" for requirements/ideas/plans, "experience" for task patterns and code pattern changes.`),
			mcp.Enum(templates.ValidCategoryStrings()...),
		),
		mcp.WithString("status",
			mcp.Description("Filter by frontmatter status field. Valid values: draft, accepted, rejected."),
			mcp.Enum(templates.ValidStatusStrings()...),
		),
		mcp.WithArray("tags",
			mcp.Description("Filter by tags. Returns documents that have at least one of the specified tags (OR semantics)."),
			mcp.WithStringItems(),
		),
		mcp.WithTitleAnnotation("List Documents"),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

// HandleListDocuments handles the list_documents tool call.
func HandleListDocuments(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docs, err := ScanDocuments(baseDir)
		if err != nil {
			return nil, fmt.Errorf("scanning documents: %w", err)
		}

		types := request.GetStringSlice("types", nil)
		category := templates.Category(request.GetString("category", ""))
		if category != "" && !templates.IsValidCategory(category) {
			return errorResult(fmt.Sprintf("invalid category %q (valid: %s)", category, strings.Join(templates.ValidCategoryStrings(), ", "))), nil
		}
		status := templates.DocStatus(request.GetString("status", ""))
		if status != "" && !templates.IsValidStatus(status) {
			return errorResult(fmt.Sprintf("invalid status %q (valid: %s)", status, strings.Join(templates.ValidStatusStrings(), ", "))), nil
		}
		filterTags := request.GetStringSlice("tags", nil)
		if len(filterTags) > 0 {
			if err := validateTags(filterTags); err != nil {
				return errorResult(err.Error()), nil
			}
		}

		var filtered []LocalDocument
		for _, doc := range docs {
			if len(types) > 0 && !slices.Contains(types, doc.Type) {
				continue
			}
			if category != "" && doc.Category != category {
				continue
			}
			if status != "" && doc.Status != status { // exact match: enum constraint guarantees lowercase
				continue
			}
			if len(filterTags) > 0 && !hasAnyTag(doc.Tags, filterTags) {
				continue
			}
			filtered = append(filtered, doc)
		}

		data, err := json.Marshal(filtered)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// hasAnyTag returns true if docTags contains at least one of filterTags (OR semantics).
func hasAnyTag(docTags, filterTags []string) bool {
	for _, ft := range filterTags {
		if slices.Contains(docTags, ft) {
			return true
		}
	}
	return false
}
