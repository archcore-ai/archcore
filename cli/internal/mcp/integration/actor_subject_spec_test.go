// Actor-subject vocabulary properties pinned through the MCP protocol, per
// scenario-and-journey-types.spec:
// 1. Both types support create, get, update, list, search, and remove (§9).
// 2. Creation emits the registered template and infers no relation.
// 3. Category, type, and tag filters retain records in every supported status.
// 4. A scenario depends_on spec edge round-trips through the manifest.
package integration

import (
	"strings"
	"testing"

	"archcore-cli/internal/mcp/tools"
	"archcore-cli/templates"
)

func TestActorSubjectVocabulary_DocumentLifecycle(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, typ, category string }{
		{name: "journey in vision", typ: "journey", category: "vision"},
		{name: "scenario in knowledge", typ: "scenario", category: "knowledge"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := initArchcore(t)
			c := newTestClient(t, base)
			created := decodeJSON[map[string]any](t, mustCallTool(t, c, "create_document", map[string]any{
				"type": tt.typ, "filename": "refund", "title": "Refund", "tags": []string{"actor:buyer"},
			}))
			path, ok := created["path"].(string)
			if !ok || path != ".archcore/refund."+tt.typ+".md" || created["category"] != tt.category {
				t.Fatalf("created = %v", created)
			}
			doc := decodeJSON[tools.EnrichedDocument](t, mustCallTool(t, c, "get_document", map[string]any{"path": path}))
			_, body, err := templates.SplitDocument([]byte(doc.Content))
			if err != nil {
				t.Fatal(err)
			}
			if body != templates.GenerateTemplate(templates.DocumentType(tt.typ)) {
				t.Error("MCP creation bypassed the template")
			}
			if len(loadManifest(t, base).Relations) != 0 {
				t.Error("create_document inferred relations")
			}
			for _, status := range []string{"draft", "accepted", "rejected"} {
				mustCallTool(t, c, "update_document", map[string]any{"path": path, "status": status})
				for _, args := range []map[string]any{nil, {"category": tt.category}, {"types": []string{tt.typ}}, {"tags": []string{"actor:buyer"}}} {
					docs := decodeListDocuments(t, mustCallTool(t, c, "list_documents", args))
					if len(docs) != 1 || docs[0].Path != path || string(docs[0].Status) != status || string(docs[0].Category) != tt.category {
						t.Fatalf("list %v: %+v", args, docs)
					}
				}
				hits := decodeJSON[struct {
					Results []searchHit `json:"results"`
				}](t, mustCallTool(t, c, "search_documents", map[string]any{"types": []string{tt.typ}})).Results
				if len(hits) != 1 || hits[0].Path != path || hits[0].Type != tt.typ {
					t.Errorf("search = %+v", hits)
				}
			}
			mustCallTool(t, c, "update_document", map[string]any{"path": path, "content": "Incomplete section content."})
			doc = decodeJSON[tools.EnrichedDocument](t, mustCallTool(t, c, "get_document", map[string]any{"path": path}))
			if !strings.Contains(doc.Content, "Incomplete section content.") {
				t.Error("precision rejected a write")
			}
			mustCallTool(t, c, "remove_document", map[string]any{"path": path})
			if docs := decodeListDocuments(t, mustCallTool(t, c, "list_documents", nil)); len(docs) != 0 {
				t.Errorf("removed record remains: %+v", docs)
			}
		})
	}
}

func TestActorSubjectVocabulary_ScenarioDependsOnSpec(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	c := newTestClient(t, base)
	mustCallTool(t, c, "create_document", map[string]any{"type": "spec", "filename": "refund", "title": "Refund"})
	mustCallTool(t, c, "create_document", map[string]any{"type": "scenario", "filename": "refund", "title": "Refund"})
	mustCallTool(t, c, "add_relation", map[string]any{
		"source": "refund.scenario.md", "target": "refund.spec.md", "type": "depends_on",
	})
	doc := decodeJSON[tools.EnrichedDocument](t, mustCallTool(t, c, "get_document", map[string]any{"path": ".archcore/refund.spec.md"}))
	if len(doc.IncomingRelations) != 1 || doc.IncomingRelations[0].Path != ".archcore/refund.scenario.md" || doc.IncomingRelations[0].Type != "depends_on" {
		t.Errorf("incoming relations = %+v, want the scenario's depends_on edge", doc.IncomingRelations)
	}
}
