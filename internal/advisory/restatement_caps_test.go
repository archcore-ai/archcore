package advisory

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	archsync "archcore-cli/internal/sync"
)

// Cap and ordering behavior of the restatement check.
//
// Both ceilings here are budget guards, not presentation choices: the neighbour
// cap bounds how many documents one write reads off disk, and the finding cap
// bounds what the hook prints. Which neighbours survive the cut has to be a
// function of the graph, not of the order relations happened to be added in.

// copiedSet are distinct clauses, each long enough to clear
// minRestatementTokens, so copying the whole set produces one finding per line
// until the cap stops it.
var copiedSet = []string{
	"The operator reviews a queued export batch before the scheduler releases it downstream.",
	"An administrator revokes a session token from the security settings screen without waiting.",
	"The catalog keeps a deleted record visible to its owner for thirty days after removal.",
	"A reviewer receives the finished audit bundle by email instead of returning to the console.",
	"The importer rejects a malformed row and reports the offending line number to the author.",
}

func TestContentNeighboursIsSortedAndCapped(t *testing.T) {
	t.Parallel()
	m := archsync.NewManifest()
	// Insertion order is deliberately not sorted order: the manifest appends,
	// so this is the order a real graph would hand the check.
	for _, target := range []string{"z.prd.md", "m.prd.md", "a.prd.md", "y.prd.md", "b.prd.md", "c.prd.md"} {
		m.Relations = append(m.Relations, archsync.Relation{
			Source: "down.plan.md", Target: target, Type: archsync.RelImplements,
		})
	}

	got := contentNeighbours(m, ".archcore/down.plan.md", maxRestatementTargets)

	if len(got) != maxRestatementTargets {
		t.Fatalf("contentNeighbours() = %d neighbours, want %d", len(got), maxRestatementTargets)
	}
	// The alphabetical head, not the insertion head — otherwise adding an
	// unrelated relation would silently change which documents get compared.
	want := []string{"a.prd.md", "b.prd.md", "c.prd.md", "m.prd.md", "y.prd.md"}
	if !slices.Equal(got, want) {
		t.Errorf("contentNeighbours() = %q, want %q", got, want)
	}
}

func TestContentNeighboursDropsSelfAndNonContentEdges(t *testing.T) {
	t.Parallel()
	m := archsync.NewManifest()
	m.Relations = []archsync.Relation{
		{Source: "down.plan.md", Target: "down.plan.md", Type: archsync.RelImplements},
		{Source: "down.plan.md", Target: "topic.doc.md", Type: archsync.RelRelated},
		{Source: "down.plan.md", Target: "order.doc.md", Type: archsync.RelDependsOn},
		{Source: "down.plan.md", Target: "up.prd.md", Type: archsync.RelImplements},
		{Source: "base.rule.md", Target: "down.plan.md", Type: archsync.RelExtends},
	}

	got := contentNeighbours(m, ".archcore/down.plan.md", maxRestatementTargets)

	want := []string{"base.rule.md", "up.prd.md"}
	if !slices.Equal(got, want) {
		t.Errorf("contentNeighbours() = %q, want %q", got, want)
	}
}

func TestRestatementCapsFindings(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "up.prd.md", numberedDoc(copiedSet...))
	writeArchcoreDoc(t, base, "down.plan.md", numberedDoc(copiedSet...))
	writeManifest(t, base, archsync.Relation{
		Source: "down.plan.md", Target: "up.prd.md", Type: archsync.RelImplements,
	})

	got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md"))

	if len(got) != maxRestatementHits {
		t.Fatalf("Restatement() = %d findings for %d copied statements, want the cap of %d:\n%s",
			len(got), len(copiedSet), maxRestatementHits, strings.Join(got, "\n"))
	}
}

// One statement copied into several linked documents is one defect. Reporting
// it once per neighbour would spend the whole finding cap echoing one line and
// hide the other statements behind it.
func TestRestatementReportsAStatementOnce(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "up1.prd.md", numberedDoc(copiedLine))
	writeArchcoreDoc(t, base, "up2.spec.md", numberedDoc(copiedLine))
	writeArchcoreDoc(t, base, "down.plan.md", numberedDoc(copiedLine))
	writeManifest(t, base,
		archsync.Relation{Source: "down.plan.md", Target: "up1.prd.md", Type: archsync.RelImplements},
		archsync.Relation{Source: "down.plan.md", Target: "up2.spec.md", Type: archsync.RelImplements},
	)

	got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md"))

	if len(got) != 1 {
		t.Fatalf("Restatement() = %d findings for one line copied into two neighbours, want 1:\n%s",
			len(got), strings.Join(got, "\n"))
	}
}

// quote cuts runes, not bytes: a Cyrillic line cut at a byte offset lands
// mid-character and the finding echoes a broken word back at the author.
func TestQuoteCutsRunesNotBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		in           string
		wantRunes    int
		wantEllipsis bool
	}{
		{
			name:      "a short ASCII line is echoed whole",
			in:        "the operator reviews the batch",
			wantRunes: len("the operator reviews the batch"),
		},
		{
			name:      "a line exactly at the ceiling is not cut",
			in:        strings.Repeat("я", maxQuotedLineRunes),
			wantRunes: maxQuotedLineRunes,
		},
		{
			name:         "a long Cyrillic line is cut at the rune ceiling",
			in:           strings.Repeat("я", maxQuotedLineRunes+10),
			wantRunes:    maxQuotedLineRunes,
			wantEllipsis: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := quote(tt.in)

			if !utf8.ValidString(got) {
				t.Fatalf("quote(%d runes) = %q, which is not valid UTF-8", utf8.RuneCountInString(tt.in), got)
			}
			if gotEllipsis := strings.HasSuffix(got, "…"); gotEllipsis != tt.wantEllipsis {
				t.Errorf("quote() ellipsis = %v, want %v", gotEllipsis, tt.wantEllipsis)
			}
			if n := utf8.RuneCountInString(strings.TrimSuffix(got, "…")); n != tt.wantRunes {
				t.Errorf("quote() = %d runes, want %d", n, tt.wantRunes)
			}
		})
	}
}
