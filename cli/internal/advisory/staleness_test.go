package advisory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/stamp"
)

func TestStalenessAdvisory_SilentCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{
			name:  "not a git repository",
			setup: func(t *testing.T) string { return setupArchcoreDir(t) },
		},
		{
			name: "no documentation history",
			setup: func(t *testing.T) string {
				base := stalenessRepo(t)
				writeAt(t, base, "src/main.go", "package main\n")
				gitCommit(t, base, "code only")
				return base
			},
		},
		{
			name: "nothing changed since the documentation commit",
			setup: func(t *testing.T) string {
				base := stalenessRepo(t)
				writeAt(t, base, "src/main.go", "package main\n")
				writeAt(t, base, ".archcore/knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nSee src/ for details.\n")
				gitCommit(t, base, "code and docs")
				return base
			},
		},
		{
			name: "source moved but no document mentions it",
			setup: func(t *testing.T) string {
				base := stalenessRepo(t)
				writeAt(t, base, ".archcore/knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nUnrelated body.\n")
				gitCommit(t, base, "docs")
				writeAt(t, base, "vendor/lib.go", "package lib\n")
				gitCommit(t, base, "vendor only")
				return base
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := tt.setup(t)
			if got := Staleness(context.Background(), base, t.TempDir(), scanCorpus(t, base)); got != "" {
				t.Errorf("Staleness = %q, want empty", got)
			}
		})
	}
}

func TestStalenessAdvisory_ReportsDrift(t *testing.T) {
	t.Parallel()
	base := stalenessRepo(t)
	writeAt(t, base, "src/auth.go", "package src\n")
	writeAt(t, base, ".archcore/knowledge/auth.adr.md",
		"---\ntitle: \"Auth\"\nstatus: accepted\n---\n\nImplemented in src/auth.go.\n")
	gitCommit(t, base, "code and docs")

	writeAt(t, base, "src/auth.go", "package src // rewritten\n")
	gitCommit(t, base, "code moved on")

	got := Staleness(context.Background(), base, t.TempDir(), scanCorpus(t, base))

	for _, want := range []string{"[Archcore Staleness]", "auth.adr.md", "references src/"} {
		if !strings.Contains(got, want) {
			t.Errorf("advisory missing %q:\n%s", want, got)
		}
	}
	// The plugin's command palette is being redesigned; the CLI must not name a
	// command it does not own and cannot keep correct.
	if strings.Contains(got, "/archcore:") {
		t.Errorf("advisory names a plugin command:\n%s", got)
	}
}

// TestStalenessAdvisory_RateLimited pins the 24-hour budget. Drift is a slow
// signal; repeated every session it trains the reader to skip it.
func TestStalenessAdvisory_RateLimited(t *testing.T) {
	t.Parallel()
	base := stalenessRepo(t)
	writeAt(t, base, "src/auth.go", "package src\n")
	writeAt(t, base, ".archcore/knowledge/auth.adr.md",
		"---\ntitle: \"Auth\"\nstatus: accepted\n---\n\nImplemented in src/auth.go.\n")
	gitCommit(t, base, "code and docs")
	writeAt(t, base, "src/auth.go", "package src // rewritten\n")
	gitCommit(t, base, "code moved on")

	stampDir := t.TempDir()

	if first := Staleness(context.Background(), base, stampDir, scanCorpus(t, base)); first == "" {
		t.Fatal("first call produced no advisory")
	}
	if second := Staleness(context.Background(), base, stampDir, scanCorpus(t, base)); second != "" {
		t.Errorf("second call within the window returned %q, want empty", second)
	}

	// Age the stamp past the window; the advisory returns.
	key := "staleness\x00" + base
	aged := time.Now().Add(-stalenessWindow - time.Hour)
	if err := os.Chtimes(stamp.PathFor(stampDir, key), aged, aged); err != nil {
		t.Fatal(err)
	}
	if third := Staleness(context.Background(), base, stampDir, scanCorpus(t, base)); third == "" {
		t.Error("advisory did not return after the window expired")
	}
}

// TestStalenessAdvisory_StampSurvivesSessionSweep is the cross-scope guarantee:
// the session scope sweeps on a 10-minute window and must not expire the
// day-long staleness budget living in its own directory.
func TestStalenessAdvisory_StampSurvivesSessionSweep(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stalenessDir := filepath.Join(root, "staleness-stamps")
	sessionDir := filepath.Join(root, "session-stamps")

	base := stalenessRepo(t)
	writeAt(t, base, "src/auth.go", "package src\n")
	writeAt(t, base, ".archcore/knowledge/auth.adr.md",
		"---\ntitle: \"Auth\"\nstatus: accepted\n---\n\nImplemented in src/auth.go.\n")
	gitCommit(t, base, "code and docs")
	writeAt(t, base, "src/auth.go", "package src // rewritten\n")
	gitCommit(t, base, "code moved on")

	if first := Staleness(context.Background(), base, stalenessDir, scanCorpus(t, base)); first == "" {
		t.Fatal("first call produced no advisory")
	}
	// Age it past the session window but well inside its own.
	key := "staleness\x00" + base
	aged := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stamp.PathFor(stalenessDir, key), aged, aged); err != nil {
		t.Fatal(err)
	}

	stamp.Claim(sessionDir, "some-session", 10*time.Minute)

	if again := Staleness(context.Background(), base, stalenessDir, scanCorpus(t, base)); again != "" {
		t.Error("the session scope's sweep expired the 24-hour staleness stamp")
	}
}
