package cmd

import (
	"fmt"
	"sort"
	"strings"

	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/sync"
	"archcore-cli/templates"
)

// maxSessionTags caps the number of tags emitted in SessionStart context.
// Capped at 20 (top-N by frequency) to limit static token overhead per session
// while preserving enough coverage for projects with rich tag namespaces.
const maxSessionTags = 20

// buildSessionContext generates the session-start context string
// that is injected into agents at session start.
func buildSessionContext(baseDir string) (string, int) {
	docs, err := tools.ScanDocuments(baseDir)
	var scanWarning string
	if err != nil {
		// A declared global source is missing or unreadable. Don't blank the whole
		// session context — that would also drop every LOCAL document. Degrade to a
		// local-only scan (which never fails on a missing global) and surface the
		// problem, honoring "a missing mandatory global is loud, not silent".
		scanWarning = err.Error()
		docs, _ = tools.ScanLocalDocuments(baseDir)
	}
	docs = localDocuments(docs) // globals are MCP-read-only; not surfaced in session context

	var b strings.Builder
	b.WriteString("[Archcore — Git-native context for AI coding agents]\n")
	b.WriteString("Git-native context for AI coding agents.\n")
	b.WriteString("You have MCP tools available: list_documents, get_document, create_document, update_document, add_relation, remove_relation, list_relations.\n")
	if scanWarning != "" {
		fmt.Fprintf(&b, "\n⚠ %s — context limited to local documents; clone the source or fix .archcore/settings.json.\n", scanWarning)
	}

	// Pre-group documents by category.
	docsByCategory := make(map[templates.Category][]tools.LocalDocument, 3)
	for _, doc := range docs {
		docsByCategory[doc.Category] = append(docsByCategory[doc.Category], doc)
	}

	// Existing documents by category.
	b.WriteString("\nEXISTING DOCUMENTS:\n")
	for _, cat := range []templates.Category{templates.CategoryKnowledge, templates.CategoryVision, templates.CategoryExperience} {
		fmt.Fprintf(&b, "  [%s]\n", cat)
		catDocs := docsByCategory[cat]
		if len(catDocs) == 0 {
			b.WriteString("    (none)\n")
			continue
		}
		for _, doc := range catDocs {
			titlePart := ""
			if doc.Title != "" {
				titlePart = fmt.Sprintf(" — %q", doc.Title)
			}
			fmt.Fprintf(&b, "    - %s%s\n", doc.Filename, titlePart)
		}
	}

	// Aggregate tag frequencies and emit top tags.
	tagFreq := make(map[string]int)
	for _, doc := range docs {
		for _, tag := range doc.Tags {
			tagFreq[tag]++
		}
	}
	if len(tagFreq) > 0 {
		type tagCount struct {
			tag   string
			count int
		}
		sorted := make([]tagCount, 0, len(tagFreq))
		for tag, count := range tagFreq {
			sorted = append(sorted, tagCount{tag, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].count != sorted[j].count {
				return sorted[i].count > sorted[j].count
			}
			return sorted[i].tag < sorted[j].tag
		})
		limit := min(maxSessionTags, len(sorted))
		tags := make([]string, limit)
		for i := range limit {
			tags[i] = sorted[i].tag
		}
		fmt.Fprintf(&b, "\nEXISTING TAGS: %s\n", strings.Join(tags, ", "))
	}

	// Document relations summary.
	if m, mErr := sync.LoadManifest(baseDir); mErr == nil && len(m.Relations) > 0 {
		fmt.Fprintf(&b, "\nDOCUMENT RELATIONS: %d relation(s) stored.\n", len(m.Relations))
		b.WriteString("  Use list_relations, add_relation, remove_relation MCP tools to manage.\n")
	}

	b.WriteString("\nRefer to MCP server instructions for document types, workflow rules, and usage guidance.\n")

	return b.String(), len(docs)
}

// localDocuments returns only documents owned by the primary project, dropping
// mounted read-only global sources. Globals are surfaced exclusively through the
// MCP read tools (list/get/search), never in CLI status or session-start context.
func localDocuments(docs []tools.LocalDocument) []tools.LocalDocument {
	out := make([]tools.LocalDocument, 0, len(docs))
	for _, d := range docs {
		if d.SourceKind == "local" {
			out = append(out, d)
		}
	}
	return out
}
