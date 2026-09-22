package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testDoc = "---\ntitle: \"A\"\nstatus: draft\ntags:\n  - one\n  - two\n---\n\nBody text.\n"

// newProject writes n documents under dir/.archcore/knowledge/ and returns dir.
func newProject(t *testing.T, docs map[string]string) string {
	t.Helper()
	base := t.TempDir()
	for rel, content := range docs {
		full := filepath.Join(base, ".archcore", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

// absOf returns the absolute path of a document inside the project.
func absOf(base, rel string) string {
	return filepath.Join(base, ".archcore", filepath.FromSlash(rel))
}

// TestScanCache_MetadataScanRetainsNoBodies: a metadata-only scan still has to
// read each file to parse its frontmatter, but it must not keep the body.
//
// The MCP server is long-lived and list_documents is its most-called tool, so a
// cache that retains every body holds the whole corpus in memory for the life of
// the process even though nothing asked for content.
func TestScanCache_MetadataScanRetainsNoBodies(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.adr.md": testDoc})

	if _, err := Scan(base); err != nil { // metadata only
		t.Fatal(err)
	}

	e, ok := sharedScanCache.entries[absOf(base, "knowledge/a.adr.md")]
	if !ok {
		t.Fatal("the metadata scan cached nothing at all — frontmatter should still be cached")
	}
	if e.content != "" {
		t.Errorf("a metadata-only scan retained the document body (%d bytes)", len(e.content))
	}
	if e.fm.Title != "A" {
		t.Errorf("frontmatter was not cached: title = %q", e.fm.Title)
	}
}

// TestScanCache_ContentScanServesContent is the counterpart: asking for content
// must still get it, on a cold cache and on one warmed by a metadata scan.
func TestScanCache_ContentScanServesContent(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.adr.md": testDoc})

	if _, err := Scan(base); err != nil { // warm the cache without bodies
		t.Fatal(err)
	}
	got, err := ScanFull(base)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("scanned %d documents, want 1", len(got))
	}
	if got[0].Content == "" {
		t.Error("a content scan after a metadata scan returned an empty body")
	}
}

// TestBuildDoc_TagsAreNotAliasedToTheCache: every Document built from a cache
// hit used to share one backing array with the cache and with each other. No
// caller mutates tags today, but the MCP server serves tools/call on a worker
// pool, so an in-place sort added later would be a silent data race and would
// corrupt what the next reader sees.
func TestBuildDoc_TagsAreNotAliasedToTheCache(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.adr.md": testDoc})

	first, err := Scan(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(first[0].Tags) != 2 {
		t.Fatalf("expected one document with two tags, got %+v", first)
	}
	first[0].Tags[0] = "MUTATED"

	second, err := Scan(base)
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Tags[0] == "MUTATED" {
		t.Error("mutating a returned document's tags changed the cached copy")
	}
}

// TestScanCache_InvalidationKey pins what makes a cached parse stale. Both
// halves matter: an edit that preserves the size still changes the mtime, and
// an edit inside the same coarse mtime tick still changes the size.
func TestScanCache_InvalidationKey(t *testing.T) {
	tests := []struct {
		name     string
		rewrite  string
		sameSize bool
	}{
		{name: "size changes", rewrite: testDoc + "More text appended.\n"},
		{name: "size is preserved but content differs", rewrite: strings.Repeat("x", len(testDoc)), sameSize: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetCache()
			base := newProject(t, map[string]string{"knowledge/a.adr.md": testDoc})
			abs := absOf(base, "knowledge/a.adr.md")

			if _, err := ScanFull(base); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, []byte(tt.rewrite), 0o644); err != nil {
				t.Fatal(err)
			}
			// Push the mtime forward so the change is visible on filesystems with
			// coarse timestamps.
			future := time.Now().Add(2 * time.Second)
			if err := os.Chtimes(abs, future, future); err != nil {
				t.Fatal(err)
			}

			got, err := ScanFull(base)
			if err != nil {
				t.Fatal(err)
			}
			if got[0].Content != tt.rewrite {
				t.Errorf("stale content served after rewrite:\ngot  %q\nwant %q", got[0].Content, tt.rewrite)
			}
		})
	}
}

// TestInvalidateCache_ForcesAReRead covers the belt-and-braces path the write
// handlers rely on when a filesystem's mtime granularity is too coarse to show
// an edit made moments after the read.
func TestInvalidateCache_ForcesAReRead(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.adr.md": testDoc})
	abs := absOf(base, "knowledge/a.adr.md")

	if _, err := ScanFull(base); err != nil {
		t.Fatal(err)
	}
	if _, ok := sharedScanCache.entries[abs]; !ok {
		t.Fatal("nothing was cached")
	}

	InvalidateCache(abs)

	if _, ok := sharedScanCache.entries[abs]; ok {
		t.Error("the entry survived invalidation")
	}
}

// TestScanCache_PruneOnlyOnceOutgrown: pruning on every scan would make a
// long-lived server re-read the corpus whenever two projects share the process.
// The cache drops absent entries only after it has outgrown the current corpus.
func TestScanCache_PruneOnlyOnceOutgrown(t *testing.T) {
	tests := []struct {
		name    string
		entries int
		corpus  int
		want    bool
	}{
		{name: "cache smaller than the corpus", entries: 3, corpus: 5},
		{name: "cache at twice the corpus", entries: 10, corpus: 5},
		{name: "cache past twice the corpus", entries: 11, corpus: 5, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := scanCache{entries: map[string]docCacheEntry{}}
			for i := range tt.entries {
				c.store(filepath.Join("p", string(rune('a'+i))), docCacheEntry{})
			}
			if got := c.needsPrune(tt.corpus); got != tt.want {
				t.Errorf("needsPrune(%d) with %d entries = %v, want %v", tt.corpus, tt.entries, got, tt.want)
			}
		})
	}
}
