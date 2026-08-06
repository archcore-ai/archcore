package advisory

import (
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/docs"
	"archcore-cli/internal/testsupport"
)

// setupArchcoreDir creates a project with an empty store and returns its root.
func setupArchcoreDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

// writeArchcoreDoc writes content at a slash-separated path under .archcore/.
func writeArchcoreDoc(t *testing.T, base, relPath, content string) {
	t.Helper()
	full := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// stalenessRepo is a project that is also a git repository.
func stalenessRepo(t *testing.T) string {
	t.Helper()
	return testsupport.NewGitRepo(t, setupArchcoreDir(t))
}

func writeAt(t *testing.T, base, relPath, content string) {
	t.Helper()
	testsupport.WriteFile(t, base, relPath, content)
}

func gitCommit(t *testing.T, base, msg string) {
	t.Helper()
	testsupport.GitCommit(t, base, msg)
}

// scanCorpus mirrors what the session-start path hands the advisory: the local
// documents, with bodies, scanned once.
func scanCorpus(t *testing.T, base string) []docs.Document {
	t.Helper()
	corpus, err := docs.ScanLocal(base, true)
	if err != nil {
		t.Fatalf("ScanLocal: %v", err)
	}
	return corpus
}
