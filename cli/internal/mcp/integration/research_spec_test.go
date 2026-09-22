// Research vocabulary properties pinned through the MCP protocol:
// 1. Registered tool enums match the document and relation registries.
// 2. Both types support create, get, update, list, search, and remove.
// 3. Category, type, and tag filters retain records in every supported status.
// 4. Incomplete sections remain writable, and creation infers no relations.
// See research-and-evidence-types.spec and evidential-and-temporal-relations.spec.
package integration

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/sync"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestResearchVocabulary_RegisteredSchemas(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, initArchcore(t))
	listed, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) == 0 {
		t.Fatal("server exposes no tools")
	}
	expected := map[string][]string{
		"create_document": templates.ValidTypes(),
		"add_relation":    sync.ValidRelationTypes(),
		"remove_relation": sync.ValidRelationTypes(),
	}
	for _, tool := range listed.Tools {
		want, ok := expected[tool.Name]
		if !ok {
			continue
		}
		data, err := json.Marshal(tool.InputSchema.Properties["type"])
		if err != nil {
			t.Fatal(err)
		}
		var property struct {
			Enum []string `json:"enum"`
		}
		if err := json.Unmarshal(data, &property); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(property.Enum, want) {
			t.Errorf("%s enum = %v, registry = %v", tool.Name, property.Enum, want)
		}
		delete(expected, tool.Name)
	}
	if len(expected) != 0 {
		t.Errorf("missing registered tools: %v", expected)
	}
}

func TestResearchVocabulary_DocumentLifecycle(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, typ, category string }{
		{name: "research in vision", typ: "research", category: "vision"},
		{name: "evidence in knowledge", typ: "evidence", category: "knowledge"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := initArchcore(t)
			c := newTestClient(t, base)
			created := decodeJSON[map[string]any](t, mustCallTool(t, c, "create_document", map[string]any{
				"type": tt.typ, "filename": "territory", "title": "Territory", "tags": []string{"source:primary"},
			}))
			path, ok := created["path"].(string)
			if !ok || path != ".archcore/territory."+tt.typ+".md" || created["category"] != tt.category {
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
				for _, args := range []map[string]any{nil, {"category": tt.category}, {"types": []string{tt.typ}}, {"tags": []string{"source:primary"}}} {
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
