package integration

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"archcore-cli/internal/mcp/tools"
)

// TestToolRegistrationCanary asserts that the in-process server registers
// exactly the expected tools. A deleted s.AddTool(...) line in server.go
// would slip through every per-tool unit test (which calls handlers
// directly) — this test is the cheapest place to catch it.
func TestToolRegistrationCanary(t *testing.T) {
	t.Parallel()
	expectedTools := []string{
		"add_relation",
		"create_document",
		"get_document",
		"init_project",
		"list_documents",
		"list_relations",
		"remove_document",
		"remove_relation",
		"search_documents",
		"update_document",
	}
	slices.Sort(expectedTools)

	base := initArchcore(t)
	c := newTestClient(t, base)

	res, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	slices.Sort(got)

	if len(got) != len(expectedTools) {
		t.Fatalf("registered %d tools, want %d\ngot:  %v\nwant: %v",
			len(got), len(expectedTools), got, expectedTools)
	}
	for i, name := range expectedTools {
		if got[i] != name {
			t.Errorf("tool[%d] = %q, want %q (full got: %v)", i, got[i], name, got)
		}
	}
}

// TestInitToCreateToGetRoundTrip exercises the headline first-run flow
// through real tool registration + JSON marshalling:
//
//	init_project (no .archcore yet) → list_documents (empty) →
//	create_document → get_document → list_documents (now 1)
//
// What it catches that unit tests can't: any drift between the JSON shape
// create_document writes to disk and the JSON shape get_document returns,
// and any framing/encoding regression on the way through the server layer.
func TestInitToCreateToGetRoundTrip(t *testing.T) {
	t.Parallel()
	// Bare tempdir, no .archcore/ — init_project must create it.
	base := t.TempDir()
	c := newTestClient(t, base)

	// 1. init_project on a fresh project — should report not previously initialized.
	initResult := mustCallTool(t, c, "init_project", map[string]any{
		"sync_mode": "none",
	})
	initPayload := decodeJSON[map[string]any](t, initResult)
	if got, _ := initPayload["initialized"].(bool); !got {
		t.Errorf("init_project initialized = %v, want true", initPayload["initialized"])
	}
	if got, _ := initPayload["already_initialized"].(bool); got {
		t.Errorf("init_project already_initialized = %v, want false", initPayload["already_initialized"])
	}

	// 1b. Idempotency: a second init_project must report already_initialized.
	initResult2 := mustCallTool(t, c, "init_project", map[string]any{"sync_mode": "none"})
	initPayload2 := decodeJSON[map[string]any](t, initResult2)
	if got, _ := initPayload2["already_initialized"].(bool); !got {
		t.Errorf("second init_project already_initialized = %v, want true", initPayload2["already_initialized"])
	}

	// 2. list_documents on freshly initialized project — empty.
	listResult := mustCallTool(t, c, "list_documents", nil)
	docs := decodeListDocuments(t, listResult)
	if len(docs) != 0 {
		t.Fatalf("list_documents on empty project: got %d docs, want 0", len(docs))
	}

	// 3. create_document.
	const (
		wantTitle  = "Use PostgreSQL for primary persistence"
		wantStatus = "draft"
		wantBody   = "## Context\n\nThis is the body.\n"
	)
	createResult := mustCallTool(t, c, "create_document", map[string]any{
		"type":      "adr",
		"filename":  "use-postgres",
		"title":     wantTitle,
		"status":    wantStatus,
		"content":   wantBody,
		"directory": "decisions",
	})
	createPayload := decodeJSON[map[string]any](t, createResult)
	createdPath, _ := createPayload["path"].(string)
	if createdPath == "" {
		t.Fatal("create_document returned empty path")
	}
	if got, _ := createPayload["title"].(string); got != wantTitle {
		t.Errorf("create_document title = %q, want %q", got, wantTitle)
	}
	if got, _ := createPayload["status"].(string); got != wantStatus {
		t.Errorf("create_document status = %q, want %q", got, wantStatus)
	}
	if got, _ := createPayload["type"].(string); got != "adr" {
		t.Errorf("create_document type = %q, want adr", got)
	}
	if got, _ := createPayload["category"].(string); got != "knowledge" {
		t.Errorf("create_document category = %q, want knowledge", got)
	}

	// 4. get_document on the path returned by create_document — round-trip
	// must preserve title/status, and content must include the body we wrote.
	getResult := mustCallTool(t, c, "get_document", map[string]any{
		"path": createdPath,
	})
	got := decodeJSON[tools.EnrichedDocument](t, getResult)
	if got.Title != wantTitle {
		t.Errorf("get_document title = %q, want %q", got.Title, wantTitle)
	}
	if got.Status != wantStatus {
		t.Errorf("get_document status = %q, want %q", got.Status, wantStatus)
	}
	if got.Type != "adr" {
		t.Errorf("get_document type = %q, want adr", got.Type)
	}
	if !strings.Contains(got.Content, "## Context") {
		t.Errorf("get_document content missing body marker; got:\n%s", got.Content)
	}
	if len(got.OutgoingRelations) != 0 || len(got.IncomingRelations) != 0 {
		t.Errorf("get_document on fresh doc: relations should be empty, got out=%v in=%v",
			got.OutgoingRelations, got.IncomingRelations)
	}

	// 5. list_documents now returns the one created doc.
	listResult2 := mustCallTool(t, c, "list_documents", nil)
	docs2 := decodeListDocuments(t, listResult2)
	if len(docs2) != 1 {
		t.Fatalf("list_documents after create: got %d docs, want 1", len(docs2))
	}
	if docs2[0].Path != createdPath {
		t.Errorf("list_documents path = %q, want %q", docs2[0].Path, createdPath)
	}
}
