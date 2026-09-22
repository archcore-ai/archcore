package tools

// Synthetic scaling benchmark for the read path (list/search/get) across corpus
// size tiers. Not part of normal CI runs — invoke explicitly:
//
//	go test ./internal/mcp/tools/ -run TestScalingOutputSizes -v
//	go test ./internal/mcp/tools/ -bench BenchmarkReadToolsScaling -benchmem -benchtime=20x
//
// All docs are local; scan cost is identical for a global doc (same walk+read
// path), so N-scaling here is also the global-growth curve. Per-tier deltas for
// globals/relations/relevance are reasoned separately from the code.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type benchHandler = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

var benchTiers = []int{10, 50, 100, 300, 1000, 3000, 10000}

const benchBodyTarget = 1800 // ~1.8 KB body — representative archcore rule/adr

func benchBody(i int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This document number %d covers lorem ipsum dolor sit amet. ", i)
	fmt.Fprintf(&b, "It references @src/module%d/handler.go and @internal/pkg%d/ in passing. ", i, i%17)
	const filler = "Consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. "
	for b.Len() < benchBodyTarget {
		b.WriteString(filler)
	}
	return b.String()
}

func benchDoc(i int) string {
	return fmt.Sprintf("---\ntitle: %q\nstatus: accepted\ntags:\n  - bench\n  - tier-%d\n---\n\n%s",
		fmt.Sprintf("Document %d Title", i), i%5, benchBody(i))
}

// buildLocalCorpus writes n local docs spread across 20 subdirectories.
func buildLocalCorpus(tb testing.TB, n int) string {
	tb.Helper()
	base := tb.TempDir()
	archcore := filepath.Join(base, ".archcore")
	for i := range n {
		sub := filepath.Join(archcore, fmt.Sprintf("domain%02d", i%20))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			tb.Fatal(err)
		}
		f := filepath.Join(sub, fmt.Sprintf("doc-%05d.rule.md", i))
		if err := os.WriteFile(f, []byte(benchDoc(i)), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	return base
}

func reqWith(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

func mustText(tb testing.TB, h benchHandler, args map[string]any) string {
	tb.Helper()
	r, err := h(context.Background(), reqWith(args))
	if err != nil {
		tb.Fatalf("handler error: %v", err)
	}
	if r.IsError {
		tc, _ := mcp.AsTextContent(r.Content[0])
		tb.Fatalf("error result: %s", tc.Text)
	}
	tc, ok := mcp.AsTextContent(r.Content[0])
	if !ok {
		tb.Fatalf("unexpected content type %T", r.Content[0])
	}
	return tc.Text
}

// TestScalingOutputSizes reports the marshaled output size per tool per tier.
// Output bytes / 4 ≈ tokens delivered to the model.
func TestScalingOutputSizes(t *testing.T) {
	if os.Getenv("ARCHCORE_SCALING") == "" {
		t.Skip("set ARCHCORE_SCALING=1 to run the scaling report")
	}
	getPath := ".archcore/domain00/doc-00000.rule.md"
	t.Logf("%-8s %12s %12s %12s %12s", "N", "list_B", "snip_B", "full_B", "get_B")
	for _, n := range benchTiers {
		base := buildLocalCorpus(t, n)
		listB := len(mustText(t, HandleListDocuments(StaticRoot(base)), map[string]any{}))
		snipB := len(mustText(t, HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "lorem"}))
		fullB := len(mustText(t, HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "lorem", "mode": "full"}))
		getB := len(mustText(t, HandleGetDocument(StaticRoot(base)), map[string]any{"path": getPath}))
		t.Logf("%-8d %12d %12d %12d %12d", n, listB, snipB, fullB, getB)
	}
}

func BenchmarkReadToolsScaling(b *testing.B) {
	getPath := ".archcore/domain00/doc-00000.rule.md"
	for _, n := range benchTiers {
		base := buildLocalCorpus(b, n)
		cases := []struct {
			name string
			h    benchHandler
			args map[string]any
		}{
			{"list", HandleListDocuments(StaticRoot(base)), map[string]any{}},
			{"search-snip", HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "lorem"}},
			{"search-full", HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "lorem", "mode": "full"}},
			{"get", HandleGetDocument(StaticRoot(base)), map[string]any{"path": getPath}},
		}
		for _, tt := range cases {
			b.Run(fmt.Sprintf("%s/N=%d", tt.name, n), func(b *testing.B) {
				req := reqWith(tt.args)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r, err := tt.h(context.Background(), req)
					if err != nil || r.IsError {
						b.Fatal("handler failed")
					}
				}
			})
		}
	}
}
