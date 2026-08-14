package tools

// Regression tests for the measured list_documents cutoffs
// (global-discovery-gap.idea): with pure walk order, 100+ local documents
// evicted every global row from the default page, and 500+ made globals
// unreachable at any limit. The interleave closes both.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// listFixture builds a project with nLocal local and nGlobal mounted documents.
func listFixture(t *testing.T, nLocal, nGlobal int) string {
	t.Helper()
	base, localArch, globalArch := twoSourceFixture(t)
	body := "---\ntitle: T\nstatus: accepted\n---\nBody.\n"
	for i := range nLocal {
		if err := os.WriteFile(filepath.Join(localArch, fmt.Sprintf("l-%04d.rule.md", i)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i := range nGlobal {
		if err := os.WriteFile(filepath.Join(globalArch, fmt.Sprintf("g-%04d.doc.md", i)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

func callList(t *testing.T, base string, args map[string]any) listDocumentsResult {
	t.Helper()
	res, err := HandleListDocuments(base)(t.Context(), reqWith(args))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		tc, _ := mcp.AsTextContent(res.Content[0])
		t.Fatalf("error result: %s", tc.Text)
	}
	tc, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("unexpected content type %T", res.Content[0])
	}
	var out listDocumentsResult
	if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// TestListDocuments_GlobalsSurviveThePageCut replays the measured thresholds:
// at every local corpus size the default page carries a proportional global
// share, with at least one global row.
func TestListDocuments_GlobalsSurviveThePageCut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		nLocal, nGlobal int
	}{
		{120, 30}, // old behavior: 0 globals on the default page
		{500, 50}, // old behavior: unreachable at any limit
		{50, 500}, // inverse skew: locals must survive too
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("local=%d global=%d", tt.nLocal, tt.nGlobal), func(t *testing.T) {
			t.Parallel()
			base := listFixture(t, tt.nLocal, tt.nGlobal)
			res := callList(t, base, map[string]any{})

			locals, globals := 0, 0
			for _, d := range res.Documents {
				if d.Global {
					globals++
				} else {
					locals++
				}
			}
			if globals == 0 || locals == 0 {
				t.Fatalf("default page: %d local, %d global — both sources must be represented", locals, globals)
			}
			// Proportional within one row of the exact share, floor 1.
			want := len(res.Documents) * tt.nGlobal / (tt.nLocal + tt.nGlobal)
			if diff := globals - want; diff < -1 || diff > 1 {
				t.Errorf("global rows = %d, want about %d (proportional share)", globals, want)
			}
			if res.BySource["local"] != tt.nLocal || res.BySource["org"] != tt.nGlobal {
				t.Errorf("by_source = %v, want local:%d org:%d", res.BySource, tt.nLocal, tt.nGlobal)
			}
		})
	}
}

// TestListDocuments_TinySourceOnFirstPage pins the floor-of-one seed: a source
// with a single document surfaces at the head of the default page even when the
// local corpus dwarfs it. Pure proportional round-robin without the seed would
// place it near position 201 — past the page cut.
func TestListDocuments_TinySourceOnFirstPage(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 200, 1)

	res := callList(t, base, map[string]any{})

	for i, d := range res.Documents {
		if d.Global {
			if i > 1 {
				t.Errorf("tiny source's row at position %d, want within the seeded head (index <= 1)", i)
			}
			return
		}
	}
	t.Fatalf("tiny global source missing from the default page entirely")
}

// TestListDocuments_SourceScope narrows the listing and the by_source map.
func TestListDocuments_SourceScope(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 5, 3)

	res := callList(t, base, map[string]any{"source": "global"})
	if res.Total != 3 {
		t.Errorf("total = %d, want 3 global docs", res.Total)
	}
	for _, d := range res.Documents {
		if !d.Global {
			t.Errorf("local document %s leaked through source=global", d.Path)
		}
	}
	if len(res.BySource) != 1 || res.BySource["org"] != 3 {
		t.Errorf("by_source = %v, want org:3 only", res.BySource)
	}
}

// TestListDocuments_InvalidSourceIsError: an unknown source scope is an error
// naming the valid forms, not an empty listing.
func TestListDocuments_InvalidSourceIsError(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 1, 1)

	res, err := HandleListDocuments(base)(t.Context(), reqWith(map[string]any{"source": "orgg"}))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !res.IsError {
		t.Fatal("want an error result for an unknown source")
	}
	tc, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("unexpected content type %T", res.Content[0])
	}
	if !strings.Contains(tc.Text, `invalid source "orgg"`) {
		t.Errorf("error = %q, want it to name the invalid source", tc.Text)
	}
}

// TestListDocuments_SingleSourceOrderUnchanged: a project without globals keeps
// the exact walk order — the interleave must be a no-op.
func TestListDocuments_SingleSourceOrderUnchanged(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 10, 0)
	// Drop the globals declaration: pure local corpus.
	settings := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settings, []byte(`{"sync":"none"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	res := callList(t, base, map[string]any{})
	if res.Total != 10 {
		t.Fatalf("total = %d, want 10", res.Total)
	}
	for i, d := range res.Documents {
		want := fmt.Sprintf("l-%04d.rule.md", i)
		if filepath.Base(d.Path) != want {
			t.Fatalf("position %d holds %s, want %s (walk order)", i, filepath.Base(d.Path), want)
		}
	}
}

// TestListDocuments_PagingIsConsistent: offset pages sample the same
// interleaved sequence — no document repeats, none disappears.
func TestListDocuments_PagingIsConsistent(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 30, 20)

	seen := make(map[string]bool)
	for offset := 0; offset < 50; offset += 10 {
		res := callList(t, base, map[string]any{"limit": float64(10), "offset": float64(offset)})
		for _, d := range res.Documents {
			if seen[d.Path] {
				t.Fatalf("document %s appeared on two pages", d.Path)
			}
			seen[d.Path] = true
		}
	}
	if len(seen) != 50 {
		t.Errorf("paged through %d unique documents, want 50", len(seen))
	}
}
