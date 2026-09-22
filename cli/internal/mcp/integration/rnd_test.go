package integration

import (
	"testing"

	"archcore-cli/internal/mcp/tools"
)

// TestRejectedRnDStaysVisible pins a load-bearing invariant of the rnd
// (research) type: a document concluded as "rejected" — "we investigated and
// decided not to" — is the most valuable thing an rnd preserves, so it must
// stay discoverable. No read path applies a *default* status filter, so both
// list_documents (no filter) and search_documents (type filter, NOT status)
// must surface a rejected rnd. A future change that started hiding rejected
// docs by default would defeat the type's whole point; this test catches it.
func TestRejectedRnDStaysVisible(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	c := newTestClient(t, base)

	// A research investigation that concluded "stop" → status rejected.
	createPayload := decodeJSON[map[string]any](t, mustCallTool(t, c, "create_document", map[string]any{
		"type":     "rnd",
		"filename": "evaluate-graphql",
		"title":    "Evaluate GraphQL for the public API",
		"status":   "rejected",
	}))
	wantPath, _ := createPayload["path"].(string)
	if wantPath == "" {
		t.Fatal("create_document(rnd) returned empty path")
	}
	if got, _ := createPayload["category"].(string); got != "vision" {
		t.Errorf("rnd category = %q, want vision", got)
	}

	// list_documents with NO status filter must include the rejected rnd.
	docs := decodeListDocuments(t, mustCallTool(t, c, "list_documents", nil))
	if !rejectedRnDVisible(docs, wantPath) {
		t.Errorf("list_documents (no status filter) hid the rejected rnd %q; got %+v", wantPath, docs)
	}

	// search_documents filtered by TYPE (not status) must also surface it.
	hits := decodeJSON[struct {
		Results []map[string]any `json:"results"`
	}](t, mustCallTool(t, c, "search_documents", map[string]any{
		"types": []string{"rnd"},
	})).Results
	found := false
	for _, h := range hits {
		if p, _ := h["path"].(string); p == wantPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("search_documents (type filter, no status filter) hid the rejected rnd %q; got %d hits", wantPath, len(hits))
	}
}

// rejectedRnDVisible reports whether docs contains the rejected rnd at path.
func rejectedRnDVisible(docs []tools.LocalDocument, path string) bool {
	for _, d := range docs {
		if d.Path == path && d.Type == "rnd" && string(d.Status) == "rejected" {
			return true
		}
	}
	return false
}
