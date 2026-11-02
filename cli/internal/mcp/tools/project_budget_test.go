package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForPath_ReservesTheProjectField sweeps the row size across the for_path
// ceiling: for every size the handler leaves room for the project field the
// server appends, so the response on the wire never passes forPathByteBudget.
func TestForPath_ReservesTheProjectField(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), strings.Repeat("long-root-segment/", 6)+"p")
	arch := filepath.Join(base, ".archcore")
	if err := os.MkdirAll(filepath.Join(base, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "src", "api.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	window := 0
	for titleLen := 300; titleLen <= 420; titleLen++ {
		if err := os.RemoveAll(arch); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(arch, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 10; i++ {
			name := "rule-" + string(rune('a'+i))
			writeFixtureDoc(t, arch, name+".rule.md", strings.Repeat("t", titleLen), "x\n")
		}
		text := searchText(t, callSearch(t, base, map[string]any{"for_path": "src/api.go"}))
		if len(text)+projectFieldBytes(base) > forPathByteBudget {
			t.Fatalf("title %d: response %d bytes + project field %d > %d", titleLen, len(text), projectFieldBytes(base), forPathByteBudget)
		}
		if len(text) > forPathByteBudget-2*projectFieldBytes(base) {
			window++
		}
	}
	if window == 0 {
		t.Fatal("the sweep never came near the ceiling; the test proves nothing")
	}
}

// sweepNearTheCeiling rewrites docs documents with titles of each length in
// [from, to] and checks that the handler leaves room for the project field the
// server appends. It fails when no response came within twice the field of the
// budget, because a sweep that never reaches the ceiling proves nothing.
func sweepNearTheCeiling(t *testing.T, docs, from, to, budget int, call func(base string) string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), strings.Repeat("long-root-segment/", 6)+"p")
	arch := filepath.Join(base, ".archcore")
	field := projectFieldBytes(base)
	window := 0
	for titleLen := from; titleLen <= to; titleLen++ {
		if err := os.RemoveAll(arch); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(arch, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := range docs {
			writeFixtureDoc(t, arch, fmt.Sprintf("doc-%03d.adr.md", i), strings.Repeat("t", titleLen), "alpha\n")
		}
		text := call(base)
		if len(text)+field > budget {
			t.Fatalf("title %d: response %d bytes + project field %d > %d", titleLen, len(text), field, budget)
		}
		if len(text) > budget-2*field {
			window++
		}
	}
	if window == 0 {
		t.Fatal("the sweep never came near the ceiling; the test proves nothing")
	}
}

func TestListDocuments_ReservesTheProjectField(t *testing.T) {
	t.Parallel()
	sweepNearTheCeiling(t, 100, 380, 480, listResponseByteBudget, func(base string) string {
		res, err := HandleListDocuments(StaticRoot(base))(t.Context(), reqWith(nil))
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}
		return searchText(t, res)
	})
}

func TestSearchDocuments_ReservesTheProjectField(t *testing.T) {
	t.Parallel()
	sweepNearTheCeiling(t, 60, 600, 760, searchResponseByteBudget, func(base string) string {
		return searchText(t, callSearch(t, base, map[string]any{"content": "alpha", "limit": 60}))
	})
}
