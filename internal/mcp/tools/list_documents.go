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

const (
	// listDefaultLimit bounds an unfiltered listing to roughly 10K tokens at
	// the measured ~98 tokens per document row.
	listDefaultLimit = 100
	listMaxLimit     = 500
)

// listDocumentsResult is the response envelope. A bare array cannot carry a
// truncation signal, so the page metadata rides alongside the documents.
// BySource counts the full filtered set per source id, so a truncation that
// removes one source's rows stays visible (global-recall-guarantees.rfc).
type listDocumentsResult struct {
	Documents []LocalDocument `json:"documents"`
	Total     int             `json:"total"`
	Offset    int             `json:"offset"`
	Returned  int             `json:"returned"`
	Truncated bool            `json:"truncated"`
	BySource  map[string]int  `json:"by_source"`
}

// NewListDocumentsTool returns the tool definition for list_documents.
func NewListDocumentsTool() mcp.Tool {
	return mcp.NewTool("list_documents",
		mcp.WithDescription(`Discover and filter documents in the .archcore/ knowledge base.

Call this tool FIRST before reading or creating any document. Use it to:
- Check whether a document on a given topic already exists (prevents duplicates)
- Get valid file paths required by get_document
- Browse what documentation is available by type, category, or status

Returns: JSON {"documents": [...], "total": N, "offset": N, "returned": N, "truncated": bool, "by_source": {...}}. Each document carries path, title, type, category, status, tags (when present), and source_kind — the listing covers the local project AND every mounted read-only global source, interleaved so every source appears on the first page. "by_source" counts the full filtered set per source (e.g. {"local": 102, "org": 42}); compare it with the page to see what a truncation dropped. When "truncated" is true there are more matches beyond this page — narrow the filters, scope with "source", or request the next page with "offset".

Use the returned paths directly as input to get_document. Do not construct paths manually.`),
		mcp.WithArray("types",
			mcp.Description("Filter by one or more document types. Valid values: adr, rfc, rule, guide, doc, spec, prd, idea, plan, rnd, task-type, cpat, mrd, brd, urd, brs, strs, syrs, srs. Example: [\"adr\", \"rule\"] returns only decision records and standards."),
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
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of documents to return. Default 100, max 500 (values above the cap are clamped; 0 or omitted maps to the default)."),
		),
		mcp.WithNumber("offset",
			mcp.Description("Number of matching documents to skip before the returned page. Default 0. Use with \"truncated\" to page through large result sets."),
		),
		mcp.WithString("source",
			mcp.Description("Scope the listing to one source: \"local\" (the project's own documents), \"global\" (every mounted global source), or a declared global source id."),
		),
		mcp.WithTitleAnnotation("List Documents"),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

// HandleListDocuments handles the list_documents tool call.
func HandleListDocuments(root RootProvider) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rawTypes := request.GetStringSlice("types", nil)
		types := make([]templates.DocumentType, len(rawTypes))
		for i, t := range rawTypes {
			types[i] = templates.DocumentType(t)
		}
		for _, tp := range types {
			if !templates.IsValidType(string(tp)) {
				return errorResult(fmt.Sprintf("invalid type %q (valid: %s)", tp, strings.Join(templates.ValidTypes(), ", "))), nil
			}
		}
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

		// Pagination, mirroring search_documents' limit semantics: 0/omitted →
		// default, above the cap → silent clamp, negative → error.
		limit := request.GetInt("limit", 0)
		if limit < 0 {
			return errorResult("limit must be non-negative"), nil
		}
		if limit == 0 {
			limit = listDefaultLimit
		}
		if limit > listMaxLimit {
			limit = listMaxLimit
		}
		offset := request.GetInt("offset", 0)
		if offset < 0 {
			return errorResult("offset must be non-negative"), nil
		}
		sourceFilter := strings.TrimSpace(request.GetString("source", ""))
		baseDir := root.Root(ctx)

		// Validate the scope before scanning ("validate then act"): a typo'd
		// source must fail loudly, not return an empty page that an agent could
		// read as an empty corpus.
		if !validSourceScope(baseDir, sourceFilter) {
			return errorResult(fmt.Sprintf("invalid source %q (valid: \"local\", \"global\", or a declared global source id)", sourceFilter)), nil
		}

		docs, err := scanDocuments(baseDir)
		if err != nil {
			return errorResult(sanitizeError("scanning documents", err)), nil
		}

		bySource := make(map[string]int)
		var filtered []LocalDocument
		for _, doc := range docs {
			if !sourceAdmits(sourceFilter, doc.SourceID, doc.SourceKind) {
				continue
			}
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
			bySource[doc.SourceID]++
			filtered = append(filtered, doc)
		}

		// Interleave sources so a page cut can never remove one source
		// entirely — pure walk order made every global row unreachable once the
		// local corpus outgrew the page (global-recall-guarantees.rfc).
		filtered = interleaveBySource(filtered)

		total := len(filtered)
		if offset > total {
			offset = total
		}
		end := min(offset+limit, total)
		page := filtered[offset:end]
		if page == nil {
			page = []LocalDocument{}
		}

		data, err := json.Marshal(listDocumentsResult{
			Documents: page,
			Total:     total,
			Offset:    offset,
			Returned:  len(page),
			Truncated: offset+len(page) < total,
			BySource:  bySource,
		})
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// interleaveBySource reorders the filtered documents into a deterministic
// sequence that every page cut samples fairly: first one leading document per
// source in scan order (local first, then declaration order), then a smooth
// weighted round-robin proportional to each source's remaining count. Within a
// source the walk order is preserved. A single-source corpus returns unchanged,
// so a project without globals keeps today's ordering exactly.
func interleaveBySource(docs []LocalDocument) []LocalDocument {
	// The common case is a single-source corpus; detect it without building
	// the groups, so a project without globals pays one comparison per row.
	multiSource := false
	for i := 1; i < len(docs); i++ {
		if docs[i].SourceID != docs[0].SourceID {
			multiSource = true
			break
		}
	}
	if !multiSource {
		return docs
	}

	order := make([]string, 0, 4)
	groups := make(map[string][]LocalDocument, 4)
	for _, d := range docs {
		if _, ok := groups[d.SourceID]; !ok {
			order = append(order, d.SourceID)
		}
		groups[d.SourceID] = append(groups[d.SourceID], d)
	}

	out := make([]LocalDocument, 0, len(docs))
	next := make(map[string]int, len(order))

	// Seed: every source's first document, in scan order — the floor-of-one
	// guarantee for the first page.
	for _, id := range order {
		out = append(out, groups[id][0])
		next[id] = 1
	}

	// Smooth weighted round-robin over the remainder, weights = remaining
	// counts. Iteration follows the order slice, never a map, so ties resolve
	// to the earlier source and the sequence is deterministic.
	weights := make(map[string]int, len(order))
	for _, id := range order {
		weights[id] = len(groups[id]) - 1
	}
	current := make(map[string]int, len(order))
	for len(out) < len(docs) {
		best := ""
		activeTotal := 0
		for _, id := range order {
			if next[id] >= len(groups[id]) {
				continue
			}
			current[id] += weights[id]
			activeTotal += weights[id]
			if best == "" || current[id] > current[best] {
				best = id
			}
		}
		if best == "" {
			break
		}
		current[best] -= activeTotal
		out = append(out, groups[best][next[best]])
		next[best]++
	}
	return out
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
