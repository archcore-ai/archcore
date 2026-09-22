package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeRecapDoc writes a document with an explicit status and modification time.
func writeRecapDoc(t *testing.T, base, relPath, title, status string, age time.Duration) {
	t.Helper()
	full := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("---\ntitle: %q\nstatus: %s\n---\n\nBody.\n", title, status)
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(full, when, when); err != nil {
		t.Fatal(err)
	}
}

// TestSessionRecap_BudgetMatchesTheSpec asserts the contract value itself.
//
// Every other budget test compares rendered output against maxRecapDocs, so they
// follow the constant wherever it is moved: changing 24 to 25 fails nothing, and
// only an order-of-magnitude drift trips the scale tests. The specified number
// is a contract with the hosts (session-start-context.spec, requirement 1), so a
// change to it has to be a deliberate edit here, not a silent one there.
func TestSessionRecap_BudgetMatchesTheSpec(t *testing.T) {
	t.Parallel()
	const specified = 24 // session-start-context.spec, requirement 1
	if maxRecapDocs != specified {
		t.Errorf("maxRecapDocs = %d, but the spec fixes the recap budget at %d — update the spec with the code, or revert",
			maxRecapDocs, specified)
	}
}

// TestBuildSessionContext_BudgetHoldsAtScale is the point of the recap: session
// cost must be a function of the budget, not of corpus size. The listing this
// replaced grew one line per document, so a large repository spent thousands of
// characters of every session on a directory dump.
func TestBuildSessionContext_BudgetHoldsAtScale(t *testing.T) {
	t.Parallel()

	build := func(n int) string {
		base := setupArchcoreDir(t)
		for i := range n {
			status, age := "draft", time.Duration(i)*time.Minute
			if i%2 == 0 {
				status = "accepted"
			}
			writeRecapDoc(t, base, fmt.Sprintf("knowledge/doc-%04d.adr.md", i), fmt.Sprintf("Doc %d", i), status, age)
		}
		ctx, _ := buildSessionContext(bg(), base)
		return ctx
	}

	small, large := build(300), build(3000)

	// Only the CORPUS counters differ in width between the two corpora.
	const counterSlack = 40
	if diff := len(large) - len(small); diff > counterSlack || diff < -counterSlack {
		t.Errorf("context grew by %d characters from 300 to 3000 documents, want at most %d", diff, counterSlack)
	}

	for _, ctx := range []string{small, large} {
		if got := countRecapLines(ctx); got > maxRecapDocs {
			t.Errorf("recap rendered %d document lines, want at most %d", got, maxRecapDocs)
		}
	}
}

// TestBuildSessionContext_ExcludesRejected pins the push-side rule. A rejected
// document is a road already refused: pushing it reads as guidance. Its
// existence still has to be visible, and its tags are still project vocabulary.
func TestBuildSessionContext_ExcludesRejected(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeRecapDoc(t, base, "knowledge/live.adr.md", "Live Decision", "accepted", time.Hour)
	writeRecapDoc(t, base, "knowledge/refused.adr.md", "Refused Decision", "rejected", time.Hour)

	// Give the rejected document a tag no live document carries.
	refused := filepath.Join(base, ".archcore", "knowledge", "refused.adr.md")
	body := "---\ntitle: \"Refused Decision\"\nstatus: rejected\ntags:\n  - \"deadend\"\n---\n\nBody.\n"
	if err := os.WriteFile(refused, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, count := buildSessionContext(bg(), base)

	if strings.Contains(ctx, "refused.adr.md") {
		t.Errorf("rejected document appeared in a recap block; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "rejected 1") {
		t.Errorf("rejected count missing from CORPUS; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "live.adr.md") {
		t.Errorf("live document missing from recap; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "deadend") {
		t.Errorf("tag carried only by a rejected document should stay in the vocabulary; ctx=%q", ctx)
	}
	if count.local != 2 {
		t.Errorf("doc count = %d, want 2 — the banner reports what the project holds", count.local)
	}
}

// TestBuildSessionContext_NoBranchOutsideGit: an absent line beats one that
// says "unknown".
func TestBuildSessionContext_NoBranchOutsideGit(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeRecapDoc(t, base, "knowledge/a.adr.md", "A", "draft", time.Minute)

	ctx, _ := buildSessionContext(bg(), base)

	if strings.Contains(ctx, "BRANCH:") {
		t.Errorf("non-git directory should render no BRANCH line; ctx=%q", ctx)
	}
}

// TestBuildSessionContext_TruncationIsNamed: a truncated list must not read as
// a complete one.
func TestBuildSessionContext_TruncationIsNamed(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	for i := range 40 {
		writeRecapDoc(t, base, fmt.Sprintf("knowledge/d-%02d.adr.md", i), fmt.Sprintf("D %d", i), "draft", time.Duration(i)*time.Minute)
	}

	ctx, _ := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "more — list_documents(") {
		t.Errorf("truncated block should name what was dropped and how to reach it; ctx=%q", ctx)
	}
}

// countRecapLines counts rendered document lines across both recap blocks.
func countRecapLines(ctx string) int {
	n := 0
	for _, line := range strings.Split(ctx, "\n") {
		if strings.HasPrefix(line, "  - .archcore/") {
			n++
		}
	}
	return n
}
