package docs

import (
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/templates"
)

// TestScanTypes_FiltersBeforeReading: the filter exists to keep the pre-write
// hook off the whole corpus, so it has to reject a document before opening it.
// Filtering after the read would give the same results and none of the benefit.
func TestScanTypes_FiltersBeforeReading(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{
		"knowledge/a.rule.md":  testDoc,
		"knowledge/b.adr.md":   testDoc,
		"vision/c.plan.md":     testDoc,
		"vision/d.idea.md":     testDoc,
		"knowledge/e.guide.md": testDoc,
	})

	got, err := ScanTypes(base, map[templates.DocumentType]bool{
		templates.TypeRule:  true,
		templates.TypeGuide: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("scanned %d documents, want 2 (rule + guide): %+v", len(got), got)
	}
	for _, doc := range got {
		if doc.Type != "rule" && doc.Type != "guide" {
			t.Errorf("type %q is outside the requested set", doc.Type)
		}
		if doc.Content == "" {
			t.Errorf("%s: a filtered scan must still carry content", doc.Path)
		}
	}

	// A rejected document is never opened, so it is never cached.
	for _, rel := range []string{"knowledge/b.adr.md", "vision/c.plan.md", "vision/d.idea.md"} {
		if _, cached := sharedScanCache.entries[absOf(base, rel)]; cached {
			t.Errorf("%s was read despite being filtered out", rel)
		}
	}
}

// TestScanTypes_EmptySetMatchesNothing guards the difference between "no filter"
// (nil) and "a filter that accepts nothing" (empty map).
func TestScanTypes_EmptySetMatchesNothing(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.rule.md": testDoc})

	got, err := ScanTypes(base, map[templates.DocumentType]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("an empty accept-set returned %d documents, want 0", len(got))
	}

	all, err := Scan(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Errorf("an unfiltered scan returned %d documents, want 1", len(all))
	}
}

// TestScanCount_TracksEveryWalk pins the counter the write guard's no-scan
// proof depends on.
func TestScanCount_TracksEveryWalk(t *testing.T) {
	base := newProject(t, map[string]string{"knowledge/a.rule.md": testDoc})

	before := ScanCount()
	if _, err := Scan(base); err != nil {
		t.Fatal(err)
	}
	if got := ScanCount() - before; got != 1 {
		t.Errorf("Scan advanced the counter by %d, want 1", got)
	}

	before = ScanCount()
	if _, err := ScanLocal(base, false); err != nil {
		t.Fatal(err)
	}
	if got := ScanCount() - before; got != 1 {
		t.Errorf("ScanLocal advanced the counter by %d, want 1", got)
	}

	before = ScanCount()
	if _, err := GuardWritablePath(base, ".archcore/knowledge/a.rule.md", nil); err != nil {
		t.Fatal(err)
	}
	if got := ScanCount() - before; got != 0 {
		t.Errorf("the write guard performed %d scans, want 0", got)
	}
}

// TestScanCache_ContentCapDegradesToFrontmatter: past the byte cap an entry is
// still cached for its frontmatter but drops its body, so the cost becomes a
// re-read rather than unbounded growth.
func TestScanCache_ContentCapDegradesToFrontmatter(t *testing.T) {
	c := scanCache{entries: map[string]docCacheEntry{}, contentBytes: maxCachedContentBytes}

	c.store("/p/a.rule.md", docCacheEntry{fm: templates.Frontmatter{Title: "A"}, content: "body", hasContent: true})

	e := c.entries["/p/a.rule.md"]
	if e.hasContent || e.content != "" {
		t.Error("a body was retained past the cap")
	}
	if e.fm.Title != "A" {
		t.Error("frontmatter was dropped along with the body")
	}
}

// TestScanCache_ByteAccountingIsBalanced: store, overwrite, invalidate and
// prune all adjust the same counter, and a drift there would silently disable
// the cap or evict everything.
func TestScanCache_ByteAccountingIsBalanced(t *testing.T) {
	c := scanCache{entries: map[string]docCacheEntry{}}
	withBody := func(s string) docCacheEntry {
		return docCacheEntry{content: s, hasContent: true}
	}

	c.store("/p/a", withBody("12345"))
	c.store("/p/b", withBody("123"))
	if c.contentBytes != 8 {
		t.Fatalf("contentBytes = %d, want 8", c.contentBytes)
	}

	c.store("/p/a", withBody("1")) // overwrite with a shorter body
	if c.contentBytes != 4 {
		t.Errorf("after overwrite contentBytes = %d, want 4", c.contentBytes)
	}

	c.invalidate("/p/b")
	if c.contentBytes != 1 {
		t.Errorf("after invalidate contentBytes = %d, want 1", c.contentBytes)
	}

	c.store("/p/c", withBody("xx"))
	c.store("/p/d", withBody("yy"))
	c.prune(map[string]bool{"/p/a": true})
	if c.contentBytes != 1 {
		t.Errorf("after prune contentBytes = %d, want 1", c.contentBytes)
	}
	if len(c.entries) != 1 {
		t.Errorf("prune left %d entries, want 1", len(c.entries))
	}
}

// TestScanLocal_SkipsTheReservedGlobalTree: a "global" segment hides a document
// from the local scan and makes it read-only — never one without the other.
func TestScanLocal_SkipsTheReservedGlobalTree(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{
		"knowledge/local.rule.md":         testDoc,
		"global/company/x.rule.md":        testDoc,
		"nested/global/deep.rule.md":      testDoc,
		"global-ish/not-reserved.rule.md": testDoc,
	})

	got, err := ScanLocal(base, false)
	if err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, d := range got {
		paths = append(paths, d.Path)
	}
	for _, unwanted := range []string{".archcore/global/company/x.rule.md", ".archcore/nested/global/deep.rule.md"} {
		if slicesContains(paths, unwanted) {
			t.Errorf("reserved global document %s appeared in the local scan", unwanted)
		}
	}
	if !slicesContains(paths, ".archcore/global-ish/not-reserved.rule.md") {
		t.Errorf("a directory merely starting with \"global\" was skipped: %v", paths)
	}
}

func slicesContains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestScanLocal_MissingArchcoreIsNotAnError: a project without .archcore/ is an
// ordinary state for every caller that scans opportunistically.
func TestScanLocal_MissingArchcoreIsNotAnError(t *testing.T) {
	got, err := ScanLocal(t.TempDir(), false)
	if err != nil {
		t.Fatalf("scanning a project without .archcore/ failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("returned %d documents from a project with no store", len(got))
	}
}

// TestWriteFileAtomic_InvalidatesTheCache pins the pairing that used to depend
// on every caller remembering it.
func TestWriteFileAtomic_InvalidatesTheCache(t *testing.T) {
	ResetCache()
	base := newProject(t, map[string]string{"knowledge/a.rule.md": testDoc})
	abs := absOf(base, "knowledge/a.rule.md")

	if _, err := ScanFull(base); err != nil {
		t.Fatal(err)
	}
	if _, ok := sharedScanCache.entries[abs]; !ok {
		t.Fatal("nothing was cached")
	}

	if err := WriteFileAtomic(abs, []byte("---\ntitle: \"B\"\n---\n\nNew body.\n")); err != nil {
		t.Fatal(err)
	}

	if _, ok := sharedScanCache.entries[abs]; ok {
		t.Error("the cached parse survived an atomic write")
	}
}

// TestWriteFileAtomic_LeavesNoTempOnFailure: a rename into a missing directory
// must not strand the temp file next to it.
func TestWriteFileAtomic_LeavesNoTempOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "a.rule.md") // parent does not exist

	if err := WriteFileAtomic(target, []byte("x")); err == nil {
		t.Fatal("expected a write into a missing directory to fail")
	}
	if _, err := os.Stat(target + ".tmp"); err == nil {
		t.Error("the temp file was left behind")
	}
}
