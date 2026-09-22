package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := []byte("hello world\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	// Compute expected SHA-256 independently.
	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestHashFile_NotFound(t *testing.T) {
	_, err := HashFile("/nonexistent/file.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestHashFile_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("consistent content"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash1, _ := HashFile(path)
	hash2, _ := HashFile(path)
	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %q != %q", hash1, hash2)
	}
}

func TestHashFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	// SHA-256 of empty input.
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestScanFiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, baseDir string)
		wantCount int
		wantPaths []string
	}{
		{
			name: "empty directories",
			setup: func(t *testing.T, baseDir string) {
				for _, cat := range []string{"vision", "knowledge", "experience"} {
					os.MkdirAll(filepath.Join(baseDir, ".archcore", cat), 0o755)
				}
			},
			wantCount: 0,
		},
		{
			name: "files across categories",
			setup: func(t *testing.T, baseDir string) {
				for _, cat := range []string{"vision", "knowledge", "experience"} {
					os.MkdirAll(filepath.Join(baseDir, ".archcore", cat), 0o755)
				}
				os.WriteFile(filepath.Join(baseDir, ".archcore", "vision", "plan-001.plan.md"), []byte("plan"), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "knowledge", "rfc-001.rfc.md"), []byte("rfc"), 0o644)
			},
			wantCount: 2,
			wantPaths: []string{"vision/plan-001.plan.md", "knowledge/rfc-001.rfc.md"},
		},
		{
			name: "includes nested subdirectories",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore", "vision", "nested"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "vision", "adr-001.adr.md"), []byte("adr"), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "vision", "nested", "hidden.adr.md"), []byte("hidden"), 0o644)
			},
			wantCount: 2,
			wantPaths: []string{"vision/adr-001.adr.md", "vision/nested/hidden.adr.md"},
		},
		{
			name: "missing category dir is not an error",
			setup: func(t *testing.T, baseDir string) {
				// Only create vision, not knowledge/experience.
				os.MkdirAll(filepath.Join(baseDir, ".archcore", "vision"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "vision", "adr-001.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
		},
		{
			name: "files in archcore root",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "root-doc.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
			wantPaths: []string{"root-doc.adr.md"},
		},
		{
			name: "skips hidden directories",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore", ".git"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", ".git", "config.md"), []byte("git"), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "visible.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
			wantPaths: []string{"visible.adr.md"},
		},
		{
			name: "skips meta files",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)
				// A valid settings.json: ScanFiles now reads the globals list
				// fail-closed, so an unparseable one is a refusal, not a skip.
				os.WriteFile(filepath.Join(baseDir, ".archcore", "settings.json"), []byte(`{"sync":"none"}`), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", ".sync-state.json"), []byte(`{}`), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "real.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
			wantPaths: []string{"real.adr.md"},
		},
		{
			name: "custom directory names",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore", "auth"), 0o755)
				os.MkdirAll(filepath.Join(baseDir, ".archcore", "payments"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "auth", "jwt.adr.md"), []byte("adr"), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "payments", "stripe.prd.md"), []byte("prd"), 0o644)
			},
			wantCount: 2,
			wantPaths: []string{"auth/jwt.adr.md", "payments/stripe.prd.md"},
		},
		{
			name: "deeply nested directories",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore", "infrastructure", "k8s", "prod"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "infrastructure", "k8s", "prod", "migration.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
			wantPaths: []string{"infrastructure/k8s/prod/migration.adr.md"},
		},
		{
			name: "skips non-md files",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "notes.txt"), []byte("text"), 0o644)
				os.WriteFile(filepath.Join(baseDir, ".archcore", "real.adr.md"), []byte("adr"), 0o644)
			},
			wantCount: 1,
			wantPaths: []string{"real.adr.md"},
		},
		{
			name: "no .archcore directory at all",
			setup: func(t *testing.T, baseDir string) {
				// Do not create .archcore.
			},
			wantCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			tt.setup(t, baseDir)
			files, err := ScanFiles(baseDir)
			if err != nil {
				t.Fatalf("ScanFiles: %v", err)
			}
			if len(files) != tt.wantCount {
				t.Errorf("got %d files, want %d", len(files), tt.wantCount)
			}
			if tt.wantPaths != nil {
				gotPaths := make(map[string]bool)
				for _, f := range files {
					gotPaths[f.RelPath] = true
				}
				for _, want := range tt.wantPaths {
					if !gotPaths[want] {
						t.Errorf("missing expected path %q", want)
					}
				}
			}
		})
	}
}

// TestScanFiles_SkipsGlobalSpace pins that sync never pushes read-only global
// space: the reserved global/ tree and documents under a declared in-tree
// global source are excluded from the scan.
func TestScanFiles_SkipsGlobalSpace(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		abs := filepath.Join(baseDir, ".archcore", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("knowledge/local.adr.md", "---\ntitle: L\nstatus: draft\n---\n\nbody")
	write("global/corp/x.rule.md", "---\ntitle: G\nstatus: accepted\n---\n\nbody")
	write("vendored/y.rule.md", "---\ntitle: V\nstatus: accepted\n---\n\nbody")
	settings := `{"sync": "none", "globals": [{"id": "vendored", "path": ".archcore/vendored"}]}`
	if err := os.WriteFile(filepath.Join(baseDir, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ScanFiles(baseDir)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("ScanFiles returned %d files, want 1 (only the local doc): %+v", len(files), files)
	}
	if files[0].RelPath != "knowledge/local.adr.md" {
		t.Errorf("RelPath = %q, want the local document", files[0].RelPath)
	}
}

// TestScanFiles_UnreadableSettingsRefuses: the globals list is the only thing
// keeping a read-only mount out of the push. Read through the degrading
// variant, an unparseable settings.json reported that no globals are declared,
// and every mounted global document was scanned as a local document of this
// project. `archcore sync` refuses earlier on the same file, so this is the
// second line rather than the first — and the reason ScanFiles is safe to call
// from anywhere.
func TestScanFiles_UnreadableSettingsRefuses(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"),
		[]byte(`{"sync":"none","globals":[`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ScanFiles(base); err == nil {
		t.Fatal("ScanFiles accepted an unparseable settings.json; a mounted global would be pushed as a local document")
	}
}
