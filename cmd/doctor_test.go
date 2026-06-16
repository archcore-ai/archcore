package cmd

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/sync"
)

func TestDoctor_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	out, _ := runCmdInDir(t, dir, "doctor")
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in output, got: %s", out)
	}
}

func TestDoctor_FreeFormDirectoryAllowed(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	// Custom directories are fine — no required subdirs.
	if err := os.MkdirAll(filepath.Join(dir, ".archcore", "auth"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "doctor")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected 'All checks passed', got: %s", out)
	}
}

func TestDoctor_InvalidSettings(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	// Write invalid settings: a KNOWN field in the wrong sync mode still errors
	// (a genuinely-unknown field would now be tolerated, not rejected).
	if err := os.WriteFile(filepath.Join(dir, ".archcore", "settings.json"), []byte(`{"sync":"none","project_id":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "doctor")
	if err == nil {
		t.Fatal("expected error for invalid settings, got nil")
	}
	if !strings.Contains(out, "Settings") {
		t.Errorf("expected 'Settings' label in output, got: %s", out)
	}
	if !strings.Contains(out, "not allowed") {
		t.Errorf("expected 'not allowed' validation error in output, got: %s", out)
	}
}

func TestDoctor_SyncNone_NoServerCheck(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	out, err := runCmdInDir(t, dir, "doctor")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if strings.Contains(out, "reachable") || strings.Contains(out, "unreachable") {
		t.Errorf("none sync should not check server, got: %s", out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected all checks passed, got: %s", out)
	}
}

func TestDoctor_SyncCloud_Reachable(t *testing.T) {
	srv := httptest.NewServer(healthyHandler())
	defer srv.Close()

	orig := config.CloudServerURL
	config.CloudServerURL = srv.URL
	defer func() { config.CloudServerURL = orig }()

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewCloudSettings()); err != nil {
		t.Fatal(err)
	}
	out, err := runCmdInDir(t, dir, "doctor")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "reachable") {
		t.Errorf("expected 'reachable' in output, got: %s", out)
	}
	if !strings.Contains(out, "sync: cloud") {
		t.Errorf("expected 'sync: cloud' in output, got: %s", out)
	}
}

func TestDoctor_SyncOnPrem_Unreachable(t *testing.T) {
	srv := httptest.NewServer(healthyHandler())
	srv.Close() // close immediately

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewOnPremSettings(srv.URL)); err != nil {
		t.Fatal(err)
	}
	out, err := runCmdInDir(t, dir, "doctor")
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
	if !strings.Contains(out, "unreachable") {
		t.Errorf("expected 'unreachable' in output, got: %s", out)
	}
}

func TestDoctor_WithInvalidDocuments(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	// Write a doc with bad naming (no type segment).
	writeDoc(t, dir, "knowledge", "readme.md", "---\ntitle: T\nstatus: draft\n---\n")

	out, err := runCmdInDir(t, dir, "doctor")
	if err == nil {
		t.Fatal("expected error for invalid documents, got nil")
	}
	if strings.Contains(out, "All checks passed") {
		t.Errorf("expected issues, but got 'All checks passed': %s", out)
	}
}

// Test the underlying doctor flow directly for completeness.
func TestDoctor_AllChecksPass(t *testing.T) {
	srv := httptest.NewServer(healthyHandler())
	defer srv.Close()

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewOnPremSettings(srv.URL)); err != nil {
		t.Fatal(err)
	}

	// Use runInit to also verify integration.
	_, err := runInit(context.Background(), dir, config.NewOnPremSettings(srv.URL))
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}

	out, err := runCmdInDir(t, dir, "doctor")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected 'All checks passed', got: %s", out)
	}
}

func TestDoctor_FixRemovesDanglingRelations(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	// Create only source file, not target — so the relation is dangling.
	writeDoc(t, dir, "", "a.adr.md", validFrontmatter)
	data := `{"version":1,"files":{},"relations":[
		{"source":"a.adr.md","target":"nonexistent.prd.md","type":"related"},
		{"source":"a.adr.md","target":"also-gone.rfc.md","type":"implements"}
	]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "doctor", "--fix")
	if err != nil {
		t.Fatalf("expected no error with --fix, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Removed 2 orphaned relation(s)") {
		t.Errorf("expected 'Removed 2 orphaned relation(s)' in output, got: %s", out)
	}
	if !strings.Contains(out, "Sync manifest valid") {
		t.Errorf("expected 'Sync manifest valid' in output, got: %s", out)
	}

	// Verify the manifest on disk has no relations left.
	m, loadErr := sync.LoadManifest(dir)
	if loadErr != nil {
		t.Fatalf("LoadManifest after fix: %v", loadErr)
	}
	if len(m.Relations) != 0 {
		t.Errorf("expected 0 relations after fix, got %d", len(m.Relations))
	}
}

func TestDoctor_FixKeepsValidRelations(t *testing.T) {
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	writeDoc(t, dir, "", "a.adr.md", validFrontmatter)
	writeDoc(t, dir, "", "b.prd.md", validFrontmatter)
	data := `{"version":1,"files":{},"relations":[
		{"source":"a.adr.md","target":"b.prd.md","type":"implements"},
		{"source":"a.adr.md","target":"gone.rfc.md","type":"related"}
	]}`
	if err := os.WriteFile(filepath.Join(dir, ".archcore", sync.ManifestFile), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmdInDir(t, dir, "doctor", "--fix")
	if err != nil {
		t.Fatalf("expected no error with --fix, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Removed 1 orphaned relation(s)") {
		t.Errorf("expected 'Removed 1 orphaned relation(s)' in output, got: %s", out)
	}

	// Verify the valid relation is still there.
	m, loadErr := sync.LoadManifest(dir)
	if loadErr != nil {
		t.Fatalf("LoadManifest after fix: %v", loadErr)
	}
	if len(m.Relations) != 1 {
		t.Fatalf("expected 1 relation after fix, got %d", len(m.Relations))
	}
	if m.Relations[0].Target != "b.prd.md" {
		t.Errorf("expected kept relation target b.prd.md, got %s", m.Relations[0].Target)
	}
}
