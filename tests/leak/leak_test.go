// Package leak holds tests that enforce architectural invariants by grepping
// the source tree. These tests run as part of the regular `go test ./...`
// pass and break the build if a forbidden symbol or env-var name shows up
// outside its allowed home.
//
// The leaked-host-env-vars test is the contract guarantee that the CLI
// remains plugin-agnostic: it must NEVER read or reference host-specific
// environment variables (CLAUDE_*, CODEX_*, WORKSPACE_*, CURSOR_*,
// CLAUDE_PLUGIN_*). Translating those signals into ARCHCORE_BASE_DIR is
// the plugin's job, not the CLI's. Without this gate, accidental
// "convenience" reads in CLI code would couple the CLI to specific hosts
// and break the integration contract.
package leak

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// forbiddenTokens are host-specific env var names and plugin paths that must
// not appear anywhere in the CLI source. Plugin code lives in a separate
// repository; the CLI must remain unaware of host integrations.
var forbiddenTokens = []*regexp.Regexp{
	regexp.MustCompile(`\bCLAUDE_PROJECT_DIR\b`),
	regexp.MustCompile(`\bCLAUDE_PLUGIN_ROOT\b`),
	regexp.MustCompile(`\bCLAUDE_PLUGIN_DATA\b`),
	regexp.MustCompile(`\bCODEX_PROJECT_DIR\b`),
	regexp.MustCompile(`\bCODEX_PLUGIN_ROOT\b`),
	regexp.MustCompile(`\bCODEX_PLUGIN_DATA\b`),
	regexp.MustCompile(`\bCURSOR_PLUGIN_ROOT\b`),
	regexp.MustCompile(`\bWORKSPACE_FOLDER_PATHS\b`),
	regexp.MustCompile(`\$\{workspaceFolder\}`),
}

// TestNoHostSpecificEnvVarsInCLI walks the CLI source tree (cmd/ and
// internal/) and fails if any forbidden token appears. Test files are
// scanned too so misguided "test the host integration" code is caught early.
//
// Excluded paths:
//   - reference-materials/ (vendored upstream code, not ours)
//   - vendor/ (third-party deps)
//   - tests/leak/ itself (the regex literals here would self-match)
func TestNoHostSpecificEnvVarsInCLI(t *testing.T) {
	root := repoRoot(t)

	scanDirs := []string{"cmd", "internal", "main.go"}
	skipDirs := map[string]bool{
		"reference-materials": true,
		"vendor":              true,
		filepath.Join("tests", "leak"): true,
	}

	hits := map[string][]string{}

	for _, sub := range scanDirs {
		path := filepath.Join(root, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("skipping %s: %v", path, err)
			continue
		}

		scanFile := func(p string) {
			if !strings.HasSuffix(p, ".go") {
				return
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return
			}
			content := string(data)
			rel, _ := filepath.Rel(root, p)
			for _, re := range forbiddenTokens {
				if re.MatchString(content) {
					hits[rel] = append(hits[rel], re.String())
				}
			}
		}
		walk := func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				rel, _ := filepath.Rel(root, p)
				if skipDirs[rel] {
					return fs.SkipDir
				}
				return nil
			}
			scanFile(p)
			return nil
		}

		if !info.IsDir() {
			scanFile(path)
			continue
		}
		if err := filepath.WalkDir(path, walk); err != nil {
			t.Fatalf("walk %s: %v", path, err)
		}
	}

	if len(hits) > 0 {
		var b strings.Builder
		b.WriteString("CLI must remain plugin-agnostic — host-specific tokens detected.\n\n")
		b.WriteString("Plugin layer (separate repo) translates host signals to ARCHCORE_BASE_DIR;\n")
		b.WriteString("the CLI must only know about ARCHCORE_BASE_DIR + --base-dir.\n\n")
		for file, tokens := range hits {
			b.WriteString(file)
			b.WriteString(":\n")
			for _, tok := range tokens {
				b.WriteString("  - ")
				b.WriteString(tok)
				b.WriteString("\n")
			}
		}
		t.Fatal(b.String())
	}
}

// repoRoot walks up from the current package looking for go.mod, returning
// the absolute path. Avoids hardcoding paths in CI.
func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find go.mod walking up from %s", cwd)
	return ""
}
