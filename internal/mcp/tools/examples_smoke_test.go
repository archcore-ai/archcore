package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/config"
)

// TestExamples_ValidAndGlobalsResolve smoke-tests the shipped examples/ fixtures
// that the globals docs now cite as canonical (.archcore/globals/*). For every
// .archcore/settings.json under examples/: the settings must parse and validate,
// ScanDocuments must succeed (which resolves and walks any declared globals), and
// any example declaring globals must actually surface at least one global
// document. A typo or unresolvable path in a shipped example fails here instead
// of silently misleading a user who opens it.
func TestExamples_ValidAndGlobalsResolve(t *testing.T) {
	// Tests run with CWD = package dir (internal/mcp/tools); examples/ is three
	// levels up at the repo root.
	examplesRoot := filepath.Join("..", "..", "..", "examples")
	if _, err := os.Stat(examplesRoot); err != nil {
		t.Skipf("examples/ not found at %s: %v", examplesRoot, err)
	}

	var baseDirs []string
	walkErr := filepath.WalkDir(examplesRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// A project base dir is the parent of the ".archcore" dir that holds
		// settings.json.
		if !d.IsDir() && d.Name() == "settings.json" && filepath.Base(filepath.Dir(p)) == ".archcore" {
			baseDirs = append(baseDirs, filepath.Dir(filepath.Dir(p)))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk examples: %v", walkErr)
	}
	if len(baseDirs) == 0 {
		t.Fatal("no example .archcore/settings.json found — wrong examplesRoot?")
	}

	for _, base := range baseDirs {
		t.Run(filepath.ToSlash(base), func(t *testing.T) {
			settings, err := config.Load(base)
			if err != nil {
				t.Fatalf("config.Load: %v", err)
			}
			if err := settings.Validate(); err != nil {
				t.Fatalf("settings.Validate: %v", err)
			}

			docs, err := ScanDocuments(base)
			if err != nil {
				t.Fatalf("ScanDocuments (a declared global likely does not resolve): %v", err)
			}

			if len(settings.Globals) > 0 {
				globalCount := 0
				for _, doc := range docs {
					if doc.SourceKind == "global" {
						globalCount++
					}
				}
				if globalCount == 0 {
					t.Errorf("example declares %d global source(s) but ScanDocuments surfaced no global documents",
						len(settings.Globals))
				}
			}
		})
	}
}
