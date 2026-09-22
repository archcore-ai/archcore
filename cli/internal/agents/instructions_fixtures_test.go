package agents

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShippedInstructionFixtures pins every instruction file shipped in the
// repo — the examples/ gallery and the repo root's own AGENTS.md / CLAUDE.md —
// to the REAL writer output:
//
//  1. marker counts are balanced (a lone end-less <!-- archcore:start --> is
//     unrecoverable: upsertFencedBlock deliberately treats it as user content,
//     so a corrupted fixture would ship forever and spread by copy);
//  2. every marker pair forms a well-formed managed span;
//  3. every managed block is byte-identical to instructionsFencedBlock — the
//     fixtures track the writer, so a nudge-body change fails here until
//     scripts/regen-examples.sh is re-run;
//  4. example fixtures are PURE block files (block + trailing newline, no
//     stray content) — exactly what a fresh `archcore init` writes.
//
// This guard exists because the entire gallery once shipped with a doubled,
// half-stale block (2 starts / 1 end) that install/remove could not heal.
func TestShippedInstructionFixtures(t *testing.T) {
	t.Parallel()
	// Tests run with CWD = package dir (internal/agents); the repo root is two
	// levels up (same convention as internal/mcp/tools/examples_smoke_test.go).
	repoRoot := filepath.Join("..", "..")

	var files []string
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		p := filepath.Join(repoRoot, rel)
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}

	examplesRoot := filepath.Join(repoRoot, "examples")
	if _, err := os.Stat(examplesRoot); err != nil {
		t.Skipf("examples/ not found at %s: %v", examplesRoot, err)
	}
	walkErr := filepath.WalkDir(examplesRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return nil
		case d.Name() == "AGENTS.md", d.Name() == "CLAUDE.md", d.Name() == "GEMINI.md":
			files = append(files, p)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk examples: %v", walkErr)
	}
	if len(files) == 0 {
		t.Fatal("no instruction files found — wrong repoRoot?")
	}

	for _, path := range files {
		path := path
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			content := string(data)

			starts := strings.Count(content, instructionsMarkerStart)
			ends := strings.Count(content, instructionsMarkerEnd)
			if starts != ends {
				t.Fatalf("unbalanced markers: %d start / %d end — an end-less orphan "+
					"block is unrecoverable by install/remove; regenerate via "+
					"scripts/regen-examples.sh", starts, ends)
			}

			spans := findManagedSpans(content)
			if len(spans) != starts {
				t.Fatalf("%d marker pair(s) but only %d well-formed managed span(s) — "+
					"nested or malformed markers", starts, len(spans))
			}

			for i, span := range spans {
				if got := content[span[0]:span[1]]; got != instructionsFencedBlock {
					t.Errorf("managed block %d drifted from the writer output "+
						"(instructionsFencedBlock); re-run scripts/regen-examples.sh.\ngot:\n%s", i, got)
				}
			}

			// Example fixtures are pure writer output; the repo root's own
			// files legitimately carry user content around the block.
			inExamples := strings.Contains(filepath.ToSlash(path), "/examples/")
			if inExamples && content != instructionsFencedBlock+"\n" {
				t.Errorf("example fixture is not a pure managed-block file — "+
					"unexpected content outside the block:\n%s", content)
			}
		})
	}
}
