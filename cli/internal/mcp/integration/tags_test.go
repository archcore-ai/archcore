package integration

import (
	"slices"
	"testing"

	"archcore-cli/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/client"
)

// TestUpdateDocumentTagSemantics covers the three-way tag update contract
// from tag-update-semantics.rule, end-to-end through the
// real server: create_document writes initial tags → update_document
// applies the semantic → list_documents and get_document confirm the
// observable result.
//
// The three semantics:
//
//	omit_preserves: no "tags" key in update args      → existing tags unchanged
//	empty_clears:   "tags": []                         → all tags removed
//	array_replaces: "tags": [new...]                   → exact replacement
//
// What this catches: a regression in update_document.go's tagsProvided
// presence check (lines 76–84) — the bug pattern is "always treat tags as
// provided", which collapses omit_preserves into empty_clears and is
// invisible to single-handler unit tests that don't compose create→update.
func TestUpdateDocumentTagSemantics(t *testing.T) {
	t.Parallel()

	t.Run("omit_preserves", func(t *testing.T) {
		t.Parallel()
		base := initArchcore(t)
		c := newTestClient(t, base)

		path := createADRWithTags(t, c, "alpha", []string{"auth", "backend"})

		// Update title only — tags must be preserved.
		mustCallTool(t, c, "update_document", map[string]any{
			"path":  path,
			"title": "Renamed Alpha",
		})

		// list_documents with tag filter still finds the doc.
		filtered := listByTag(t, c, "auth")
		if len(filtered) != 1 || filtered[0].Path != path {
			t.Errorf("after omit-update: list_documents(tags=[auth]) = %v, want 1 hit at %q", filtered, path)
		}

		// get_document confirms both tags survived in the order set on disk.
		doc := getDoc(t, c, path)
		want := []string{"auth", "backend"}
		if !slices.Equal(doc.Tags, want) {
			t.Errorf("get_document tags = %v, want %v (omitted update must preserve)", doc.Tags, want)
		}
		if doc.Title != "Renamed Alpha" {
			t.Errorf("get_document title = %q, want %q", doc.Title, "Renamed Alpha")
		}
	})

	t.Run("empty_clears", func(t *testing.T) {
		t.Parallel()
		base := initArchcore(t)
		c := newTestClient(t, base)

		path := createADRWithTags(t, c, "beta", []string{"auth"})

		// Empty array MUST clear all tags.
		mustCallTool(t, c, "update_document", map[string]any{
			"path": path,
			"tags": []string{},
		})

		// list_documents(tags=[auth]) now returns nothing.
		filtered := listByTag(t, c, "auth")
		if len(filtered) != 0 {
			t.Errorf("after empty-update: list_documents(tags=[auth]) = %d hits, want 0", len(filtered))
		}

		// get_document: Tags slice must be empty.
		doc := getDoc(t, c, path)
		if len(doc.Tags) != 0 {
			t.Errorf("get_document tags = %v, want empty", doc.Tags)
		}
	})

	t.Run("array_replaces", func(t *testing.T) {
		t.Parallel()
		base := initArchcore(t)
		c := newTestClient(t, base)

		path := createADRWithTags(t, c, "gamma", []string{"auth"})

		// Replacing should remove "auth" and add "frontend".
		mustCallTool(t, c, "update_document", map[string]any{
			"path": path,
			"tags": []string{"frontend"},
		})

		if got := listByTag(t, c, "auth"); len(got) != 0 {
			t.Errorf("after replace-update: list_documents(tags=[auth]) = %d hits, want 0", len(got))
		}
		if got := listByTag(t, c, "frontend"); len(got) != 1 || got[0].Path != path {
			t.Errorf("after replace-update: list_documents(tags=[frontend]) = %v, want 1 hit at %q", got, path)
		}

		doc := getDoc(t, c, path)
		want := []string{"frontend"}
		if !slices.Equal(doc.Tags, want) {
			t.Errorf("get_document tags = %v, want %v", doc.Tags, want)
		}
	})
}

// createADRWithTags wraps create_document for the tag scenarios so each
// sub-test can express setup in one call.
func createADRWithTags(t *testing.T, c *client.Client, slug string, tags []string) string {
	t.Helper()
	res := mustCallTool(t, c, "create_document", map[string]any{
		"type":     "adr",
		"filename": slug,
		"title":    "Doc " + slug,
		"tags":     tags,
	})
	payload := decodeJSON[map[string]any](t, res)
	path, _ := payload["path"].(string)
	if path == "" {
		t.Fatalf("create_document(%s) returned empty path", slug)
	}
	return path
}

// listByTag filters list_documents by a single tag and returns the matching
// docs. Encapsulates the args shape so the per-scenario assertions stay
// focused on the round-trip.
func listByTag(t *testing.T, c *client.Client, tag string) []tools.LocalDocument {
	t.Helper()
	res := mustCallTool(t, c, "list_documents", map[string]any{
		"tags": []string{tag},
	})
	return decodeListDocuments(t, res)
}

// getDoc fetches a single document and decodes the EnrichedDocument payload.
func getDoc(t *testing.T, c *client.Client, path string) tools.EnrichedDocument {
	t.Helper()
	res := mustCallTool(t, c, "get_document", map[string]any{
		"path": path,
	})
	return decodeJSON[tools.EnrichedDocument](t, res)
}
