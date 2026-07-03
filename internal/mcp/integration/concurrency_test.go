package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	gosync "sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"archcore-cli/internal/sync"
)

// writeConcDoc creates a minimal valid document file directly on disk.
func writeConcDoc(t *testing.T, base, relPath string) {
	t.Helper()
	abs := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: Doc\nstatus: draft\n---\n\nbody\n"
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestConcurrentAddRelations_NoLostUpdates pins the manifestStore serialization:
// mcp-go dispatches tools/call on a worker pool, so unserialized
// load-modify-save of .sync-state.json would lose relations (last save wins).
// Every concurrent add must land in the manifest and report added=true.
func TestConcurrentAddRelations_NoLostUpdates(t *testing.T) {
	base := initArchcore(t)
	const n = 8
	writeConcDoc(t, base, "knowledge/hub.adr.md")
	for i := range n {
		writeConcDoc(t, base, fmt.Sprintf("knowledge/doc-%d.adr.md", i))
	}
	c := newTestClient(t, base)

	var wg gosync.WaitGroup
	addedCh := make(chan bool, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := mcp.CallToolRequest{}
			req.Params.Name = "add_relation"
			req.Params.Arguments = map[string]any{
				"source": fmt.Sprintf("knowledge/doc-%d.adr.md", i),
				"target": "knowledge/hub.adr.md",
				"type":   "related",
			}
			result, err := c.CallTool(context.Background(), req)
			if err != nil {
				t.Errorf("add_relation transport error: %v", err)
				return
			}
			if result.IsError {
				t.Errorf("add_relation handler error: %s", firstText(result))
				return
			}
			var payload struct {
				Added bool `json:"added"`
			}
			if err := json.Unmarshal([]byte(firstText(result)), &payload); err != nil {
				t.Errorf("decoding add_relation result: %v", err)
				return
			}
			addedCh <- payload.Added
		}(i)
	}
	wg.Wait()
	close(addedCh)

	for added := range addedCh {
		if !added {
			t.Error("every distinct concurrent add_relation must report added=true")
		}
	}

	m, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Relations) != n {
		t.Errorf("manifest holds %d relations, want %d — concurrent update lost", len(m.Relations), n)
	}
}

// TestConcurrentAddRemoveRelations_Converges mixes concurrent adds and removes
// of disjoint edges and verifies the survivors are exactly the added-only set.
func TestConcurrentAddRemoveRelations_Converges(t *testing.T) {
	base := initArchcore(t)
	const n = 6
	writeConcDoc(t, base, "knowledge/hub.adr.md")
	for i := range n {
		writeConcDoc(t, base, fmt.Sprintf("knowledge/doc-%d.adr.md", i))
	}
	c := newTestClient(t, base)

	// Seed n pre-existing edges sequentially.
	for i := range n {
		mustCallTool(t, c, "add_relation", map[string]any{
			"source": fmt.Sprintf("knowledge/doc-%d.adr.md", i),
			"target": "knowledge/hub.adr.md",
			"type":   "related",
		})
	}

	// Concurrently: remove the even edges, add "extends" edges for odd docs.
	var wg gosync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := mcp.CallToolRequest{}
			if i%2 == 0 {
				req.Params.Name = "remove_relation"
				req.Params.Arguments = map[string]any{
					"source": fmt.Sprintf("knowledge/doc-%d.adr.md", i),
					"target": "knowledge/hub.adr.md",
					"type":   "related",
				}
			} else {
				req.Params.Name = "add_relation"
				req.Params.Arguments = map[string]any{
					"source": fmt.Sprintf("knowledge/doc-%d.adr.md", i),
					"target": "knowledge/hub.adr.md",
					"type":   "extends",
				}
			}
			result, err := c.CallTool(context.Background(), req)
			if err != nil {
				t.Errorf("transport error: %v", err)
				return
			}
			if result.IsError {
				t.Errorf("handler error: %s", firstText(result))
			}
		}(i)
	}
	wg.Wait()

	m, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	// Survivors: n/2 "related" edges (odd docs) + n/2 "extends" edges (odd docs).
	want := n/2 + n/2
	if len(m.Relations) != want {
		t.Errorf("manifest holds %d relations, want %d", len(m.Relations), want)
	}
}
