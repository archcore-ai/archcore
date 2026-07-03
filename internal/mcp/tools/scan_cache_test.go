package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func listTitles(t *testing.T, base string) map[string]string {
	t.Helper()
	result, err := callTool(HandleListDocuments(base), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("list_documents error: %s", resultText(t, result))
	}
	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	titles := make(map[string]string, len(docs))
	for _, d := range docs {
		titles[d.Path] = d.Title
	}
	return titles
}

// TestScanCache_FreshContentAfterEdit pins cache invalidation on (mtime, size):
// an on-disk edit must be visible on the next scan, warm cache or not.
func TestScanCache_FreshContentAfterEdit(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: Before\nstatus: draft\n---\n\nbody\n")

	titles := listTitles(t, base)
	if titles[".archcore/knowledge/a.adr.md"] != "Before" {
		t.Fatalf("first scan title = %q", titles[".archcore/knowledge/a.adr.md"])
	}

	// Different length guarantees a size change even if mtime granularity is coarse.
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: After Edit Longer\nstatus: draft\n---\n\nbody\n")
	titles = listTitles(t, base)
	if got := titles[".archcore/knowledge/a.adr.md"]; got != "After Edit Longer" {
		t.Errorf("warm scan served stale title %q", got)
	}
}

// TestScanCache_DeleteAndAddVisible pins that adds/removes are detected by
// enumeration — the walk runs every scan, only per-file reads are cached.
func TestScanCache_DeleteAndAddVisible(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody\n")

	if titles := listTitles(t, base); len(titles) != 1 {
		t.Fatalf("first scan returned %d docs", len(titles))
	}

	writeDoc(t, base, "knowledge", "b.adr.md", "---\ntitle: B\nstatus: draft\n---\n\nbody\n")
	if titles := listTitles(t, base); len(titles) != 2 {
		t.Errorf("new document not visible on warm scan: %v", titles)
	}

	if err := os.Remove(filepath.Join(base, ".archcore", "knowledge", "a.adr.md")); err != nil {
		t.Fatal(err)
	}
	titles := listTitles(t, base)
	if _, ok := titles[".archcore/knowledge/a.adr.md"]; ok {
		t.Error("deleted document still visible on warm scan")
	}
	if len(titles) != 1 {
		t.Errorf("scan after delete returned %d docs", len(titles))
	}
}

// TestHandleListDocuments_ConcurrentWarmScans exercises the shared scan cache
// from parallel handlers under -race (the MCP server serves on a worker pool).
func TestHandleListDocuments_ConcurrentWarmScans(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for _, name := range []string{"a.adr.md", "b.rule.md", "c.guide.md"} {
		writeDoc(t, base, "knowledge", name, "---\ntitle: Doc\nstatus: draft\n---\n\nbody\n")
	}
	handler := HandleListDocuments(base)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := callTool(handler, nil)
			if err != nil {
				t.Error(err)
				return
			}
			if result.IsError {
				t.Errorf("list_documents error: %s", result.Content[0].(mcp.TextContent).Text)
			}
		}()
	}
	wg.Wait()
}

// TestPopulateNearbyDocuments_FiltersNonDocuments pins the ReadDir-based
// sibling listing: only valid-typed .md documents are suggested.
func TestPopulateNearbyDocuments_FiltersNonDocuments(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "auth", "existing.adr.md", "---\ntitle: E\nstatus: draft\n---\n\nbody\n")
	writeDoc(t, base, "auth", "notes.txt", "not a doc")
	writeDoc(t, base, "auth", "junk.md", "no type segment")

	result, err := callTool(HandleCreateDocument(base), map[string]any{
		"type":      "rule",
		"filename":  "fresh",
		"directory": "auth",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("create_document error: %s", resultText(t, result))
	}
	var resp struct {
		Nearby []string `json:"nearby_documents"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Nearby) != 1 || resp.Nearby[0] != ".archcore/auth/existing.adr.md" {
		t.Errorf("nearby_documents = %v, want only the valid sibling document", resp.Nearby)
	}
}
