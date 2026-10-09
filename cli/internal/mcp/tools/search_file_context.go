package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"archcore-cli/internal/advisory"
	"archcore-cli/internal/docs"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// forPathByteBudget keeps a for_path response inline on the strictest probed
// host: Copilot in VS Code stores a tool result over 8 × 1 024 characters in a
// file the agent may never open (file-context-resolution.spec).
const forPathByteBudget = 8_000

const moreGeneralCall = `list_documents(types=["rule","cpat"], status="accepted")`

// searchFileContext serves search_documents for_path: the file-context result
// the pre-edit hook renders, filtered by source, types, and status, then capped
// (search-documents.spec §1.13–§1.14).
func searchFileContext(baseDir, forPath, sourceFilter string, types []templates.DocumentType, status templates.DocStatus) (*mcp.CallToolResult, error) {
	fc, err := advisory.ResolveFileContext(baseDir, forPath)
	switch {
	case errors.Is(err, advisory.ErrOutsideProject):
		return errorResult("for_path must name a file inside the project"), nil
	case errors.Is(err, advisory.ErrArchcoreDocPath):
		return errorResult("for_path must name a source file, not an .archcore document; read documents with get_document"), nil
	case err != nil:
		return errorResult(sanitizeError("resolving file context", err)), nil
	}

	coverage := make(map[string]int)
	for sourceID, n := range fc.Scanned {
		kind := docs.SourceKindGlobal
		if sourceID == string(docs.SourceKindLocal) {
			kind = docs.SourceKindLocal
		}
		if sourceAdmits(sourceFilter, sourceID, kind) {
			coverage[sourceID] = n
		}
	}
	hits := make(map[string]int, len(coverage))
	for sourceID := range coverage {
		hits[sourceID] = 0
	}
	admit := func(rows []advisory.ContextRow) []advisory.ContextRow {
		var out []advisory.ContextRow
		for _, row := range rows {
			d := row.Doc
			if !sourceAdmits(sourceFilter, d.SourceID, d.SourceKind) ||
				(len(types) > 0 && !slices.Contains(types, d.Type)) ||
				(status != "" && d.Status != status) {
				continue
			}
			hits[d.SourceID]++
			out = append(out, row)
		}
		return out
	}
	matched, general := admit(fc.Matched), admit(fc.General)
	shownMatched := min(len(matched), advisory.MaxContextMatched)
	shownGeneral := min(len(general), advisory.MaxContextGeneral)

	truncated := false
	for {
		response := fileContextResponse(fc.Rel, coverage, hits, matched, general, shownMatched, shownGeneral, truncated)
		data, err := json.Marshal(response)
		if err != nil {
			return nil, fmt.Errorf("marshaling file context: %w", err)
		}
		if len(data) <= forPathByteBudget || shownMatched+shownGeneral <= 1 {
			return mcp.NewToolResultText(string(data)), nil
		}
		truncated = true
		if shownGeneral > 0 {
			shownGeneral--
		} else {
			shownMatched--
		}
	}
}

func fileContextResponse(rel string, coverage, hits map[string]int, matched, general []advisory.ContextRow, shownMatched, shownGeneral int, truncated bool) searchDocumentsResult {
	rows := append(slices.Clone(matched[:shownMatched]), general[:shownGeneral]...)
	response := searchDocumentsResult{
		Coverage:  coverage,
		Hits:      hits,
		Truncated: truncated,
		Index:     make([]searchIndexEntry, 0, len(rows)),
		Results:   make([]searchResult, 0, len(rows)),
	}
	for _, row := range rows {
		d := row.Doc
		response.Index = append(response.Index, searchIndexEntry{Path: d.Path, Title: d.Title, SourceID: d.SourceID})
		// Relations stay empty under for_path: five per direction per row would
		// spend the byte budget that keeps the result inline.
		response.Results = append(response.Results, searchResult{
			Path: d.Path, Title: d.Title, Type: d.Type, Status: d.Status, ModTime: d.ModTime, Tags: d.Tags,
			SourceID: d.SourceID, SourceKind: d.SourceKind, Global: d.Global, ReadOnly: d.ReadOnly,
			Matches: []searchMatch{}, Reason: row.Reason,
			IncomingRelations: []DocumentRelation{}, OutgoingRelations: []DocumentRelation{},
		})
	}
	omitted := fileContextOmission{Matched: len(matched) - shownMatched, General: len(general) - shownGeneral}
	if omitted.Matched > 0 {
		omitted.MoreMatched = fmt.Sprintf("search_documents(path_ref=%q)", rel)
	}
	if omitted.General > 0 {
		omitted.MoreGeneral = moreGeneralCall
	}
	if omitted != (fileContextOmission{}) {
		response.Omitted = &omitted
	}
	return response
}
