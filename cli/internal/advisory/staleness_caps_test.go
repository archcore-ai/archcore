package advisory

import (
	"fmt"
	"strings"
	"testing"

	"archcore-cli/internal/docs"
)

// Cap behavior of the staleness correlation.
//
// Three budgets bound what one session start can print: documents per changed
// directory, total lines, and how many directories are correlated at all. None
// of them was pinned — no case built a corpus large enough to reach a cap, so
// moving any of the numbers changed the advisory and failed nothing.
//
// These cases assert the literal budgets rather than the constants. Comparing
// rendered output against the constant follows it wherever it moves, which is
// the same trap TestSessionRecap_BudgetMatchesTheSpec calls out for the recap.

// stalenessDoc builds a corpus document whose body mentions dir + "/".
func stalenessDoc(path, dir string) docs.Document {
	return docs.Document{
		Path:    path,
		Content: fmt.Sprintf("Body referencing %s/impl.go for context.\n", dir),
	}
}

// changedIn returns n changed-file paths under dir.
func changedIn(dir string, n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = fmt.Sprintf("%s/file%d.go", dir, i)
	}
	return out
}

// TestCorrelateStaleness_PerDirectoryCap: one directory with more matching
// documents than the budget reports exactly the budget.
func TestCorrelateStaleness_PerDirectoryCap(t *testing.T) {
	t.Parallel()
	const budget = 5 // maxStalenessDocsPerDir

	var corpus []docs.Document
	for i := range 7 {
		corpus = append(corpus, stalenessDoc(fmt.Sprintf(".archcore/d%d.adr.md", i), "src"))
	}

	got := correlateStaleness(corpus, changedIn("src", 2))

	if len(got) != budget {
		t.Errorf("reported %d documents for one directory, want the %d-document budget:\n%s",
			len(got), budget, strings.Join(got, "\n"))
	}
}

// TestCorrelateStaleness_TotalLineCap: the per-directory budget alone would let
// three directories print fifteen lines, so the total budget has to bind first.
func TestCorrelateStaleness_TotalLineCap(t *testing.T) {
	t.Parallel()
	const budget = 10 // maxStalenessLines

	var corpus []docs.Document
	var changed []string
	for _, dir := range []string{"alpha", "beta", "gamma"} {
		changed = append(changed, changedIn(dir, 2)...)
		for i := range 7 {
			corpus = append(corpus, stalenessDoc(fmt.Sprintf(".archcore/%s-%d.adr.md", dir, i), dir))
		}
	}

	got := correlateStaleness(corpus, changed)

	if len(got) != budget {
		t.Errorf("reported %d lines across three directories, want the %d-line budget:\n%s",
			len(got), budget, strings.Join(got, "\n"))
	}
}

// TestCorrelateStaleness_DirectoryCap covers the directory budget in both
// directions. Only the least-changed directory carries a matching document, so
// it is reported exactly when it survives truncation — which makes the case fail
// whether the budget is lowered or raised.
func TestCorrelateStaleness_DirectoryCap(t *testing.T) {
	t.Parallel()
	const budget = 12 // maxStalenessDirs

	tests := []struct {
		name      string
		dirs      int
		wantLines int
	}{
		{name: "at the budget", dirs: budget, wantLines: 1},
		{name: "one over the budget", dirs: budget + 1, wantLines: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Every directory but the last changes twice; the last changes once, so
			// it sorts last and is the first one truncation drops.
			var changed []string
			for i := range tt.dirs - 1 {
				changed = append(changed, changedIn(fmt.Sprintf("dir%02d", i), 2)...)
			}
			last := fmt.Sprintf("dir%02d", tt.dirs-1)
			changed = append(changed, changedIn(last, 1)...)

			// The corpus mentions only the least-changed directory.
			corpus := []docs.Document{stalenessDoc(".archcore/only.adr.md", last)}

			got := correlateStaleness(corpus, changed)

			if len(got) != tt.wantLines {
				t.Errorf("reported %d lines for %d directories, want %d (budget %d):\n%s",
					len(got), tt.dirs, tt.wantLines, budget, strings.Join(got, "\n"))
			}
		})
	}
}
