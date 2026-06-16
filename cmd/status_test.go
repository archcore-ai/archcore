package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/sync"
)

// runCmdInDir executes an archcore subcommand in dir and returns captured stdout + error.
func runCmdInDir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer os.Chdir(orig)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	root := NewRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	execErr := root.Execute()
	w.Close()
	os.Stdout = oldStdout

	var out bytes.Buffer
	out.ReadFrom(r)
	return out.String(), execErr
}

// writeDoc creates a .md file with given content inside .archcore/<subdir>/.
// Creates the subdirectory if it doesn't exist.
func writeDoc(t *testing.T, dir, subdir, filename, content string) {
	t.Helper()
	d := filepath.Join(dir, ".archcore", subdir)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, filename)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const validFrontmatter = "---\ntitle: Test Doc\nstatus: draft\n---\n\nBody.\n"

func initValidDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStatus_NoArchcoreDir(t *testing.T) {
	dir := t.TempDir()
	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got: %s", out)
	}
}

func TestStatus_CustomDirectoryAllowed(t *testing.T) {
	dir := initValidDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".archcore", "auth"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeDoc(t, dir, "auth", "jwt.adr.md", validFrontmatter)

	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error for custom directory, got: %v", err)
	}
}

func TestStatus_ValidStructureAndFiles(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "use-postgres.adr.md", validFrontmatter)

	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "exists") {
		t.Errorf("expected 'exists' checks in output, got: %s", out)
	}
}

func TestStatus_BadFilename_NoTypeSegment(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "readme.md", validFrontmatter)

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "<slug>.<type>.md") {
		t.Errorf("expected '<slug>.<type>.md' hint, got: %s", out)
	}
}

func TestStatus_BadSlug_Uppercase(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "MyFeature.adr.md", validFrontmatter)

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "lowercase") {
		t.Errorf("expected 'lowercase' in output, got: %s", out)
	}
}

func TestStatus_UnknownDocumentType(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "my-feature.banana.md", validFrontmatter)

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected 'unknown' in output, got: %s", out)
	}
}

func TestStatus_AnyDirectoryForAnyType(t *testing.T) {
	dir := initValidDir(t)
	// task-type in any directory is fine — categories are virtual.
	writeDoc(t, dir, "vision", "my-task.task-type.md", validFrontmatter)

	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestStatus_MissingFrontmatter(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "my-doc.adr.md", "# No frontmatter\n")

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "missing YAML frontmatter") {
		t.Errorf("expected 'missing YAML frontmatter', got: %s", out)
	}
}

func TestStatus_MissingTitle(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "my-doc.adr.md", "---\nstatus: draft\n---\n")

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "missing required field") {
		t.Errorf("expected 'missing required field', got: %s", out)
	}
}

func TestStatus_MissingStatus(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "my-doc.adr.md", "---\ntitle: Hello\n---\n")

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "missing required field") {
		t.Errorf("expected 'missing required field', got: %s", out)
	}
}

func TestStatus_NoManifestFile(t *testing.T) {
	dir := initValidDir(t)
	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No sync manifest") {
		t.Errorf("expected 'No sync manifest' in output, got: %s", out)
	}
}

func TestStatus_ValidManifest(t *testing.T) {
	dir := initValidDir(t)
	data := `{"version":1,"files":{"vision/test.md":"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"}}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Sync manifest valid") {
		t.Errorf("expected 'Sync manifest valid' in output, got: %s", out)
	}
	if !strings.Contains(out, "1 file(s) tracked, 0 relation(s)") {
		t.Errorf("expected '1 file(s) tracked, 0 relation(s)' in output, got: %s", out)
	}
}

func TestStatus_CorruptManifest(t *testing.T) {
	dir := initValidDir(t)
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte("{truncated"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in output, got: %s", out)
	}
	if !strings.Contains(out, "Delete .archcore/.sync-state.json") {
		t.Errorf("expected delete hint in output, got: %s", out)
	}
}

func TestStatus_InvalidHashInManifest(t *testing.T) {
	dir := initValidDir(t)
	data := `{"version":1,"files":{"vision/test.md":"short"}}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(out, "not valid SHA-256") {
		t.Errorf("expected 'not valid SHA-256' in output, got: %s", out)
	}
}

func TestStatus_DeeplyNestedFiles(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "infrastructure/k8s/prod", "migration.adr.md", validFrontmatter)

	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error for deeply nested file, got: %v", err)
	}
}

func TestStatus_FileInRoot(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "", "root-doc.adr.md", validFrontmatter)

	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error for file in .archcore/ root, got: %v", err)
	}
}

func TestStatus_NonMdFilesIgnored(t *testing.T) {
	dir := initValidDir(t)
	// Write a non-.md file that should be silently skipped.
	d := filepath.Join(dir, ".archcore", "knowledge")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "notes.txt"), []byte("not a doc"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Also write a valid doc so we get meaningful output.
	writeDoc(t, dir, "knowledge", "real.adr.md", validFrontmatter)

	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error (non-.md files ignored), got: %v", err)
	}
}

func TestStatus_EmptyArchcoreDir(t *testing.T) {
	dir := initValidDir(t)
	// .archcore/ exists but has no documents.
	_, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error for empty .archcore/ dir, got: %v", err)
	}
}

func TestStatus_ManifestWithCustomDirPath(t *testing.T) {
	dir := initValidDir(t)
	// Manifest with a custom directory path (not vision/knowledge/experience).
	data := `{"version":1,"files":{"auth/jwt-strategy.adr.md":"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"}}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error for custom-dir manifest path, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Sync manifest valid") {
		t.Errorf("expected 'Sync manifest valid', got: %s", out)
	}
}

func TestStatus_ManifestWithValidRelations(t *testing.T) {
	dir := initValidDir(t)
	// Create actual documents referenced by the relation.
	writeDoc(t, dir, "", "a.adr.md", validFrontmatter)
	writeDoc(t, dir, "", "b.prd.md", validFrontmatter)
	data := `{"version":1,"files":{},"relations":[{"source":"a.adr.md","target":"b.prd.md","type":"implements"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "1 relation(s)") {
		t.Errorf("expected '1 relation(s)' in output, got: %s", out)
	}
}

func TestStatus_ManifestWithDanglingRelation(t *testing.T) {
	dir := initValidDir(t)
	// Only create source, not target.
	writeDoc(t, dir, "", "a.adr.md", validFrontmatter)
	data := `{"version":1,"files":{},"relations":[{"source":"a.adr.md","target":"nonexistent.prd.md","type":"related"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error for dangling relation, got nil")
	}
	if !strings.Contains(out, "does not exist") {
		t.Errorf("expected 'does not exist' in output, got: %s", out)
	}
	if !strings.Contains(out, "archcore doctor --fix") {
		t.Errorf("expected 'archcore doctor --fix' hint in output, got: %s", out)
	}
}

func TestStatus_ManifestWithInvalidRelationType(t *testing.T) {
	dir := initValidDir(t)
	data := `{"version":1,"files":{},"relations":[{"source":"a.adr.md","target":"b.prd.md","type":"blocks"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error for invalid relation type, got nil")
	}
	if !strings.Contains(out, "invalid type") {
		t.Errorf("expected 'invalid type' in output, got: %s", out)
	}
}

func TestStatus_ManifestWithDuplicateRelation(t *testing.T) {
	dir := initValidDir(t)
	data := `{"version":1,"files":{},"relations":[
		{"source":"a.adr.md","target":"b.prd.md","type":"related"},
		{"source":"a.adr.md","target":"b.prd.md","type":"related"}
	]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatal("expected error for duplicate relation, got nil")
	}
	if !strings.Contains(out, "duplicate") {
		t.Errorf("expected 'duplicate' in output, got: %s", out)
	}
}

// TestStatus_ExcludesGlobalTagsFromHygiene verifies tag hygiene ignores mounted
// read-only globals: a tag that exists only upstream must not be flagged in the
// consumer's status, which cannot fix it.
func TestStatus_ExcludesGlobalTagsFromHygiene(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "local.rule.md",
		"---\ntitle: Local\nstatus: accepted\ntags:\n  - shared\n---\n\nbody\n")
	// In-tree global carrying a tag that exists ONLY in the global.
	writeDoc(t, dir, "global/company/knowledge", "company.rule.md",
		"---\ntitle: Company\nstatus: accepted\ntags:\n  - globalonly\n---\n\nbody\n")
	settings := filepath.Join(dir, ".archcore", "settings.json")
	if err := os.WriteFile(settings, []byte(`{"sync":"none","globals":[{"id":"company","path":".archcore/global/company"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err != nil {
		t.Fatalf("status error: %v\noutput: %s", err, out)
	}
	if strings.Contains(out, "globalonly") {
		t.Errorf("global-only tag must be excluded from tag hygiene, got:\n%s", out)
	}
}

// TestStatus_MissingGlobalFailsStatus guards the "mandatory globals are loud, not
// silent" invariant on the status surface: a declared-but-absent global must make
// `archcore status` report a visible failure and exit non-zero, while local structural
// checks still run. The status path uses the global-aware tools.ScanDocuments (not
// ScanLocalDocuments) precisely so this failure surfaces; a refactor swapping it would
// drop the invariant silently and fail this test. No absolute path may leak.
func TestStatus_MissingGlobalFailsStatus(t *testing.T) {
	dir := initValidDir(t)
	writeDoc(t, dir, "knowledge", "local.rule.md",
		"---\ntitle: Local\nstatus: accepted\n---\n\nbody\n")
	// Declare a global whose directory does not exist on disk.
	settings := filepath.Join(dir, ".archcore", "settings.json")
	if err := os.WriteFile(settings, []byte(`{"sync":"none","globals":[{"id":"company","path":"../missing/.archcore"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "status")
	if err == nil {
		t.Fatalf("status must exit non-zero when a declared global is missing; output:\n%s", out)
	}
	for _, want := range []string{
		"error scanning documents for tag check:", // the surfacing FailLine
		"company",              // the missing global's id
		"../missing/.archcore", // the relative declared path, verbatim
		"1 issue(s) found",     // exactly one issue counted + summary printed
		".archcore/ exists",    // local structural checks ran first (degrade-but-loud)
	} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q; got:\n%s", want, out)
		}
	}
	// No absolute-path leak: the temp dir's absolute root must never appear.
	if strings.Contains(out, dir) {
		t.Errorf("status output leaked an absolute path (%q); got:\n%s", dir, out)
	}
}
