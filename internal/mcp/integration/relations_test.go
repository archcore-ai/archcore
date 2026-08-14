package integration

import (
	"strings"
	"testing"

	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/sync"
)

// listRelationsResponse mirrors the JSON shape returned by list_relations.
type listRelationsResponse struct {
	Relations []sync.Relation `json:"relations"`
}

// searchHit is a minimal mirror of the unexported searchResult type returned
// by search_documents — we decode only the fields the assertions touch.
type searchHit struct {
	Path              string                   `json:"path"`
	Title             string                   `json:"title"`
	Type              string                   `json:"type"`
	IncomingRelations []tools.DocumentRelation `json:"incoming_relations"`
	OutgoingRelations []tools.DocumentRelation `json:"outgoing_relations"`
}

func findHit(t *testing.T, hits []searchHit, path string) searchHit {
	t.Helper()
	for _, h := range hits {
		if h.Path == path {
			return h
		}
	}
	t.Fatalf("no search hit for path %q in %d results", path, len(hits))
	return searchHit{}
}

// TestAddRelationCrossTool exercises the headline cross-tool composition:
// create A + create B → add_relation(A→B, implements) → search_documents
// surfaces the relation on both sides with the correct .archcore/-prefixed
// paths. Also corroborates against the on-disk manifest, which must store
// paths WITHOUT the .archcore/ prefix (per add_relation.go:14).
//
// What this catches that unit tests don't: drift between add_relation's
// path-stripping and search_documents' path-reconstructing — both unit
// suites pass against their own hand-written manifests, but a regression
// in either direction silently breaks the loop in production.
func TestAddRelationCrossTool(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	c := newTestClient(t, base)

	pathA := createADR(t, c, "", "alpha")
	pathB := createADR(t, c, "", "beta")

	addRes := mustCallTool(t, c, "add_relation", map[string]any{
		"source": pathA,
		"target": pathB,
		"type":   "implements",
	})
	addPayload := decodeJSON[map[string]any](t, addRes)
	if got, _ := addPayload["added"].(bool); !got {
		t.Errorf("add_relation added = %v, want true", addPayload["added"])
	}

	// Client-side assertion: search_documents surfaces the relation on both
	// sides with the .archcore/ prefix on the relation paths.
	searchRes := mustCallTool(t, c, "search_documents", map[string]any{
		"types": []string{"adr"},
	})
	hits := decodeJSON[struct {
		Results []searchHit `json:"results"`
	}](t, searchRes).Results
	if len(hits) != 2 {
		t.Fatalf("search_documents returned %d hits, want 2", len(hits))
	}

	hitA := findHit(t, hits, pathA)
	hitB := findHit(t, hits, pathB)

	if len(hitA.OutgoingRelations) != 1 {
		t.Fatalf("doc A outgoing = %d, want 1: %+v", len(hitA.OutgoingRelations), hitA.OutgoingRelations)
	}
	if hitA.OutgoingRelations[0].Path != pathB {
		t.Errorf("doc A outgoing path = %q, want %q", hitA.OutgoingRelations[0].Path, pathB)
	}
	if hitA.OutgoingRelations[0].Type != "implements" {
		t.Errorf("doc A outgoing type = %q, want implements", hitA.OutgoingRelations[0].Type)
	}

	if len(hitB.IncomingRelations) != 1 {
		t.Fatalf("doc B incoming = %d, want 1: %+v", len(hitB.IncomingRelations), hitB.IncomingRelations)
	}
	if hitB.IncomingRelations[0].Path != pathA {
		t.Errorf("doc B incoming path = %q, want %q", hitB.IncomingRelations[0].Path, pathA)
	}

	// Manifest-side assertion: paths must be stored relative to .archcore/
	// (no prefix). This is the canonical place a mismatch between the
	// two normalization sites would surface.
	m := loadManifest(t, base)
	if len(m.Relations) != 1 {
		t.Fatalf("manifest has %d relations, want 1: %+v", len(m.Relations), m.Relations)
	}
	wantSrc := strings.TrimPrefix(pathA, ".archcore/")
	wantTgt := strings.TrimPrefix(pathB, ".archcore/")
	if m.Relations[0].Source != wantSrc {
		t.Errorf("manifest source = %q, want %q (no .archcore/ prefix)", m.Relations[0].Source, wantSrc)
	}
	if m.Relations[0].Target != wantTgt {
		t.Errorf("manifest target = %q, want %q (no .archcore/ prefix)", m.Relations[0].Target, wantTgt)
	}
	if m.Relations[0].Type != sync.RelImplements {
		t.Errorf("manifest type = %q, want implements", m.Relations[0].Type)
	}
}

// TestRemoveDocumentCleansUpRelations proves that remove_document, when
// invoked through the real handler path, both deletes the file AND triggers
// CleanupRelations() with manifest persistence. Failing either step orphans
// edges in the manifest — silent, never user-visible until a sync diff
// catches it days later.
func TestRemoveDocumentCleansUpRelations(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	c := newTestClient(t, base)

	pathA := createADR(t, c, "", "removed")
	pathB := createADR(t, c, "", "kept")

	mustCallTool(t, c, "add_relation", map[string]any{
		"source": pathA,
		"target": pathB,
		"type":   "implements",
	})

	// Sanity: the relation is in the manifest before removal.
	if got := loadManifest(t, base).Relations; len(got) != 1 {
		t.Fatalf("precondition: manifest relations = %d, want 1", len(got))
	}

	removeRes := mustCallTool(t, c, "remove_document", map[string]any{
		"path": pathA,
	})
	removePayload := decodeJSON[map[string]any](t, removeRes)
	if got, _ := removePayload["relations_removed"].(float64); got != 1 {
		t.Errorf("remove_document relations_removed = %v, want 1", removePayload["relations_removed"])
	}

	// Client assertion: list_relations reports nothing.
	listRes := mustCallTool(t, c, "list_relations", nil)
	listPayload := decodeJSON[listRelationsResponse](t, listRes)
	if len(listPayload.Relations) != 0 {
		t.Errorf("list_relations after remove = %d, want 0: %+v", len(listPayload.Relations), listPayload.Relations)
	}

	// Manifest assertion: the on-disk manifest is also clean. Both sites
	// must agree — disagreement means CleanupRelations ran but SaveManifest
	// did not, which is the exact failure mode this test exists to catch.
	if got := loadManifest(t, base).Relations; len(got) != 0 {
		t.Errorf("manifest relations after remove = %d, want 0: %+v", len(got), got)
	}
}

// TestAddRelationAcceptsBothPathForms verifies the documented contract that
// add_relation accepts paths with or without the ".archcore/" prefix and
// produces byte-identical manifest state in both cases (per add_relation.go
// description). A regression here would mean agents calling with one form
// vs. the other end up with different graphs — confusing to debug and
// invisible to per-tool unit tests.
func TestAddRelationAcceptsBothPathForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		source string
		target string
	}{
		{"with_prefix", ".archcore/alpha.adr.md", ".archcore/beta.adr.md"},
		{"without_prefix", "alpha.adr.md", "beta.adr.md"},
	}

	const wantSrc = "alpha.adr.md"
	const wantTgt = "beta.adr.md"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			base := initArchcore(t)
			c := newTestClient(t, base)

			// Create the two docs at .archcore/ root so the same paths
			// (with/without prefix) work for both cases.
			createADR(t, c, "", "alpha")
			createADR(t, c, "", "beta")

			mustCallTool(t, c, "add_relation", map[string]any{
				"source": tc.source,
				"target": tc.target,
				"type":   "related",
			})

			snap := loadManifest(t, base).Relations
			if len(snap) != 1 {
				t.Fatalf("%s: manifest has %d relations, want 1", tc.name, len(snap))
			}
			if snap[0].Source != wantSrc {
				t.Errorf("%s: manifest source = %q, want %q (prefix must be stripped)", tc.name, snap[0].Source, wantSrc)
			}
			if snap[0].Target != wantTgt {
				t.Errorf("%s: manifest target = %q, want %q (prefix must be stripped)", tc.name, snap[0].Target, wantTgt)
			}
			if snap[0].Type != sync.RelRelated {
				t.Errorf("%s: manifest type = %q, want related", tc.name, snap[0].Type)
			}
		})
	}
}

// TestRemoveRelationUndoesAddRelation pairs add and remove on the same
// edge and asserts both client and manifest see no relation. Catches
// off-by-one / index-mismatch bugs in remove_relation that don't surface
// in unit tests starting from a hand-crafted clean manifest — the relation
// has to come from a real add_relation call to expose them.
func TestRemoveRelationUndoesAddRelation(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	c := newTestClient(t, base)

	pathA := createADR(t, c, "", "src")
	pathB := createADR(t, c, "", "tgt")

	mustCallTool(t, c, "add_relation", map[string]any{
		"source": pathA,
		"target": pathB,
		"type":   "depends_on",
	})
	if got := loadManifest(t, base).Relations; len(got) != 1 {
		t.Fatalf("precondition: manifest relations = %d, want 1", len(got))
	}

	removeRes := mustCallTool(t, c, "remove_relation", map[string]any{
		"source": pathA,
		"target": pathB,
		"type":   "depends_on",
	})
	removePayload := decodeJSON[map[string]any](t, removeRes)
	if got, _ := removePayload["removed"].(bool); !got {
		t.Errorf("remove_relation removed = %v, want true", removePayload["removed"])
	}

	listRes := mustCallTool(t, c, "list_relations", nil)
	listPayload := decodeJSON[listRelationsResponse](t, listRes)
	if len(listPayload.Relations) != 0 {
		t.Errorf("list_relations after remove = %d, want 0", len(listPayload.Relations))
	}
	if got := loadManifest(t, base).Relations; len(got) != 0 {
		t.Errorf("manifest relations after remove = %d, want 0", len(got))
	}
}
