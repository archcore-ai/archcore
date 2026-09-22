package tools

// Realistic scaling benchmark: builds corpora by replicating this repo's actual
// .archcore/ documents (real size distribution ~5.5 KB avg, real frontmatter and
// @-refs) and seeds relations at the real density (~2 per doc), so search pays
// the real RelationsFor O(K·R) term that the synthetic bench (R=0) hid.
//
//	ARCHCORE_SCALING=1 go test ./internal/mcp/tools/ -run TestRealisticOutputSizes -v
//	go test ./internal/mcp/tools/ -bench BenchmarkRealisticReadTools -benchmem -benchtime=15x

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/sync"
)

type realDoc struct {
	base    string
	content []byte
}

// loadRealDocs reads every *.md under this repo's own .archcore/ (three levels
// up from the package dir). Skips the run if it cannot be located.
func loadRealDocs(tb testing.TB) []realDoc {
	tb.Helper()
	wd, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	realArchcore := filepath.Join(wd, "..", "..", "..", ".archcore")
	if _, statErr := os.Stat(realArchcore); statErr != nil {
		tb.Skipf("real .archcore not found at %s", realArchcore)
	}
	var docs []realDoc
	walkErr := filepath.WalkDir(realArchcore, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		docs = append(docs, realDoc{base: d.Name(), content: data})
		return nil
	})
	if walkErr != nil {
		tb.Fatal(walkErr)
	}
	if len(docs) == 0 {
		tb.Skip("no real docs found")
	}
	return docs
}

// buildRealisticCorpus writes n docs by cycling the real corpus (preserving real
// sizes and type suffixes) and seeds ~2 relations/doc. Returns base + a get path.
func buildRealisticCorpus(tb testing.TB, real []realDoc, n int) (base, getPath string) {
	tb.Helper()
	base = tb.TempDir()
	archcore := filepath.Join(base, ".archcore")
	relPaths := make([]string, n)
	for i := range n {
		src := real[i%len(real)]
		sub := fmt.Sprintf("domain%02d", i%20)
		dir := filepath.Join(archcore, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
		fname := fmt.Sprintf("%05d-%s", i, src.base)
		if err := os.WriteFile(filepath.Join(dir, fname), src.content, 0o644); err != nil {
			tb.Fatal(err)
		}
		relPaths[i] = sub + "/" + fname
	}
	// Real density: ~2 relations per doc (78 docs → 160 relations in this repo).
	m := sync.NewManifest()
	for i := range n {
		m.AddRelation(relPaths[i], relPaths[(i+1)%n], sync.RelationType("related"))
		m.AddRelation(relPaths[i], relPaths[(i+3)%n], sync.RelationType("depends_on"))
	}
	if err := sync.SaveManifest(base, m); err != nil {
		tb.Fatal(err)
	}
	return base, ".archcore/" + relPaths[0]
}

// TestRealisticOutputSizes reports per-tool output bytes (token proxy) and the
// number of content matches, on realistic corpora.
func TestRealisticOutputSizes(t *testing.T) {
	if os.Getenv("ARCHCORE_SCALING") == "" {
		t.Skip("set ARCHCORE_SCALING=1 to run the realistic scaling report")
	}
	real := loadRealDocs(t)
	t.Logf("real corpus: %d docs", len(real))
	t.Logf("%-8s %12s %12s %12s %12s", "N", "list_B", "snip_B", "full_B", "get_B")
	for _, n := range benchTiers {
		base, getPath := buildRealisticCorpus(t, real, n)
		listB := len(mustText(t, HandleListDocuments(StaticRoot(base)), map[string]any{}))
		snipB := len(mustText(t, HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "the"}))
		fullB := len(mustText(t, HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "the", "mode": "full"}))
		getB := len(mustText(t, HandleGetDocument(StaticRoot(base)), map[string]any{"path": getPath}))
		t.Logf("%-8d %12d %12d %12d %12d", n, listB, snipB, fullB, getB)
	}
}

func BenchmarkRealisticReadTools(b *testing.B) {
	real := loadRealDocs(b)
	for _, n := range benchTiers {
		base, getPath := buildRealisticCorpus(b, real, n)
		cases := []struct {
			name string
			h    benchHandler
			args map[string]any
		}{
			{"list", HandleListDocuments(StaticRoot(base)), map[string]any{}},
			{"search-snip", HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "the"}},
			{"search-full", HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "the", "mode": "full"}},
			{"get", HandleGetDocument(StaticRoot(base)), map[string]any{"path": getPath}},
		}
		for _, tt := range cases {
			b.Run(fmt.Sprintf("%s/N=%d", tt.name, n), func(b *testing.B) {
				req := reqWith(tt.args)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					r, err := tt.h(context.Background(), req)
					if err != nil || r.IsError {
						b.Fatal("handler failed")
					}
				}
			})
		}
	}
}
