package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeError(t *testing.T) {
	t.Parallel()
	pathErr := &fs.PathError{Op: "open", Path: "/Users/someone/.archcore/.sync-state.json", Err: fs.ErrPermission}
	tests := []struct {
		name   string
		action string
		err    error
		want   string
	}{
		{
			name:   "bare path error maps to class",
			action: "loading manifest",
			err:    pathErr,
			want:   "loading manifest: permission denied",
		},
		{
			name:   "wrapped path error maps to class",
			action: "loading manifest",
			err:    fmt.Errorf("reading manifest: %w", pathErr),
			want:   "loading manifest: permission denied",
		},
		{
			name:   "link error maps to class",
			action: "saving manifest",
			err:    &os.LinkError{Op: "rename", Old: "/tmp/a", New: "/tmp/b", Err: fs.ErrExist},
			want:   "saving manifest: file already exists",
		},
		{
			name:   "not-exist class",
			action: "reading doc",
			err:    &fs.PathError{Op: "open", Path: "/abs/doc.md", Err: fs.ErrNotExist},
			want:   "reading doc: file not found",
		},
		{
			name:   "unknown io error falls back to generic class",
			action: "writing doc",
			err:    &fs.PathError{Op: "write", Path: "/abs/doc.md", Err: errors.New("input/output error")},
			want:   "writing doc: file system error",
		},
		{
			name:   "validation text passes through",
			action: "loading manifest",
			err:    errors.New(`invalid manifest: relation[0]: invalid type "bogus"`),
			want:   `loading manifest: invalid manifest: relation[0]: invalid type "bogus"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeError(tt.action, tt.err)
			if got != tt.want {
				t.Errorf("sanitizeError() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "/") && strings.Contains(got, "abs") {
				t.Errorf("sanitized message leaked a path: %q", got)
			}
		})
	}
}

// skipIfNoPermissionErrors skips tests that rely on chmod-induced failures.
func skipIfNoPermissionErrors(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission tests are not portable to Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}
}

// chmodWithRestore chmods path and restores 0o755 on cleanup so t.TempDir
// removal succeeds.
func chmodWithRestore(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o755) })
}

const sanitizeTestDoc = "---\ntitle: Doc\nstatus: draft\n---\n\nbody\n"

func TestHandleAddRelation_UnreadableManifest_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	writeDoc(t, base, "knowledge", "b.adr.md", sanitizeTestDoc)
	manifestFile := filepath.Join(base, ".archcore", ".sync-state.json")
	if err := os.WriteFile(manifestFile, []byte(`{"version":1,"files":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	chmodWithRestore(t, manifestFile, 0o000)

	result, err := callTool(HandleAddRelation(base), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "knowledge/b.adr.md",
		"type":   "related",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unreadable manifest")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "loading manifest: permission denied") {
		t.Errorf("message = %q, want sanitized permission-denied class", msg)
	}
}

func TestHandleAddRelation_CorruptManifest_ValidationTextSurvives(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	writeDoc(t, base, "knowledge", "b.adr.md", sanitizeTestDoc)
	manifestFile := filepath.Join(base, ".archcore", ".sync-state.json")
	corrupt := `{"version":1,"files":{},"relations":[{"source":"x.adr.md","target":"y.adr.md","type":"bogus"}]}`
	if err := os.WriteFile(manifestFile, []byte(corrupt), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleAddRelation(base), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "knowledge/b.adr.md",
		"type":   "related",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for corrupt manifest")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, `invalid type "bogus"`) {
		t.Errorf("message = %q, want validation detail preserved", msg)
	}
}

func TestHandleGetDocument_UnreadableFile_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	chmodWithRestore(t, filepath.Join(base, ".archcore", "knowledge", "a.adr.md"), 0o000)

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/a.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unreadable document")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "permission denied") {
		t.Errorf("message = %q, want permission-denied class", msg)
	}
}

func TestHandleUpdateDocument_UnwritableFile_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	// The write is atomic (tmp + rename), so a read-only FILE is still
	// replaceable; a read-only DIRECTORY blocks creating the temp file.
	chmodWithRestore(t, filepath.Join(base, ".archcore", "knowledge"), 0o555)

	result, err := callTool(HandleUpdateDocument(base), map[string]any{
		"path":  ".archcore/knowledge/a.adr.md",
		"title": "New Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unwritable document")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "writing .archcore/knowledge/a.adr.md: permission denied") {
		t.Errorf("message = %q, want sanitized write error with relative path", msg)
	}
}

func TestHandleCreateDocument_UnwritableDir_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	subDir := filepath.Join(base, ".archcore", "frozen")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chmodWithRestore(t, subDir, 0o555)

	result, err := callTool(HandleCreateDocument(base), map[string]any{
		"type":      "adr",
		"filename":  "new-doc",
		"directory": "frozen",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unwritable directory")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "permission denied") {
		t.Errorf("message = %q, want permission-denied class", msg)
	}
}

func TestHandleListDocuments_UnreadableSubdir_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	writeDoc(t, base, "hidden", "b.adr.md", sanitizeTestDoc)
	chmodWithRestore(t, filepath.Join(base, ".archcore", "hidden"), 0o000)

	result, err := callTool(HandleListDocuments(base), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unreadable subdirectory")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "scanning documents: permission denied") {
		t.Errorf("message = %q, want sanitized scan error", msg)
	}
}

func TestHandleRemoveDocument_UnreadableManifest_NoPathLeak(t *testing.T) {
	skipIfNoPermissionErrors(t)
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", sanitizeTestDoc)
	manifestFile := filepath.Join(base, ".archcore", ".sync-state.json")
	if err := os.WriteFile(manifestFile, []byte(`{"version":1,"files":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	chmodWithRestore(t, manifestFile, 0o000)

	result, err := callTool(HandleRemoveDocument(base), map[string]any{
		"path": ".archcore/knowledge/a.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for unreadable manifest after deletion")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "file deleted but failed to load manifest: permission denied") {
		t.Errorf("message = %q, want sanitized manifest error", msg)
	}
}
