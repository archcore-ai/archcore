package tools

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"archcore-cli/internal/config"
)

func TestGuardWritablePath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "vendored", Path: ".archcore/vendored"}})
	globals, err := config.LoadGlobals(base)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		relPath string
		wantErr error // nil = must pass; non-nil = errors.Is match
		lexical bool  // true = any error acceptable (validateArchcorePath text)
	}{
		{name: "valid local document", relPath: ".archcore/knowledge/a.adr.md"},
		{name: "valid root document", relPath: ".archcore/a.adr.md"},
		{name: "reserved global exact", relPath: ".archcore/global/x.rule.md", wantErr: errPathReadOnlyGlobal},
		{name: "reserved global capitalized", relPath: ".archcore/Global/x.rule.md", wantErr: errPathReadOnlyGlobal},
		{name: "reserved global mixed case nested", relPath: ".archcore/team/gLoBaL/x.rule.md", wantErr: errPathReadOnlyGlobal},
		{name: "settings.json is not a document", relPath: ".archcore/settings.json", wantErr: errPathNotDocument},
		{name: "sync state is not a document", relPath: ".archcore/.sync-state.json", wantErr: errPathNotDocument},
		{name: "non-md file is not a document", relPath: ".archcore/notes.txt", wantErr: errPathNotDocument},
		{name: "nested meta file is not a document", relPath: ".archcore/sub/settings.json", wantErr: errPathNotDocument},
		{name: "declared global exact", relPath: ".archcore/vendored/x.rule.md", wantErr: errPathReadOnlyGlobal},
		{name: "declared global case variant", relPath: ".archcore/Vendored/x.rule.md", wantErr: errPathReadOnlyGlobal},
		{name: "path traversal", relPath: ".archcore/../etc/passwd.md", lexical: true},
		{name: "absolute path", relPath: "/etc/x.md", lexical: true},
		{name: "outside archcore", relPath: "src/x.md", lexical: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := guardWritablePath(base, tt.relPath, globals)
			switch {
			case tt.lexical:
				if err == nil {
					t.Fatalf("guardWritablePath(%q) = nil error, want lexical rejection", tt.relPath)
				}
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("guardWritablePath(%q) error = %v, want %v", tt.relPath, err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("guardWritablePath(%q) unexpected error: %v", tt.relPath, err)
				}
			}
			if err != nil {
				assertNoAbsPath(t, base, err.Error())
			}
		})
	}
}

func TestGuardWritablePath_SymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not portable to Windows")
	}
	base := setupTestArchcore(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(base, ".archcore", "lnk")); err != nil {
		t.Fatal(err)
	}

	_, err := guardWritablePath(base, ".archcore/lnk/x.adr.md", nil)
	if !errors.Is(err, errPathEscapes) {
		t.Fatalf("error = %v, want errPathEscapes", err)
	}
	assertNoAbsPath(t, base, err.Error())
	assertNoAbsPath(t, outside, err.Error())
}

// TestValidateReadPath_SymlinkEscape guards the read side: a symlinked document
// inside .archcore/ that points at a file outside it must be rejected, so
// get_document / search_documents can never leak a foreign file's contents
// through the MCP server. Mirrors TestGuardWritablePath_SymlinkEscape.
func TestValidateReadPath_SymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not portable to Windows")
	}
	base := setupTestArchcore(t)
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "id_rsa")
	if err := os.WriteFile(secret, []byte("TOP-SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, ".archcore", "knowledge", "leak.adr.md")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}

	_, err := validateReadPath(base, ".archcore/knowledge/leak.adr.md", nil)
	if !errors.Is(err, errPathEscapes) {
		t.Fatalf("error = %v, want errPathEscapes", err)
	}
	assertNoAbsPath(t, base, err.Error())
	assertNoAbsPath(t, outside, err.Error())

	// A legit in-tree document must still pass.
	real := filepath.Join(base, ".archcore", "knowledge", "ok.adr.md")
	if err := os.WriteFile(real, []byte("# ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := validateReadPath(base, ".archcore/knowledge/ok.adr.md", nil); err != nil {
		t.Fatalf("validateReadPath rejected a legit document: %v", err)
	}
}

func TestHandleUpdateDocument_SettingsJSONBlocked(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	settingsPath := filepath.Join(base, ".archcore", "settings.json")
	original := []byte(`{"sync":"none"}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/settings.json",
		"title": "Pwned",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for update of settings.json")
	}
	if msg := resultText(t, result); !strings.Contains(msg, "not a document") {
		t.Errorf("message = %q, want not-a-document rejection", msg)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Error("settings.json must be untouched after a rejected update")
	}
}

func TestHandleRemoveDocument_SyncStateBlocked(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	manifestFile := filepath.Join(base, ".archcore", ".sync-state.json")
	if err := os.WriteFile(manifestFile, []byte(`{"version":1,"files":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/.sync-state.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for removal of .sync-state.json")
	}
	if msg := resultText(t, result); !strings.Contains(msg, "not a document") {
		t.Errorf("message = %q, want not-a-document rejection", msg)
	}
	if _, err := os.Stat(manifestFile); err != nil {
		t.Error(".sync-state.json must still exist after a rejected removal")
	}
}

func TestHandleCreateDocument_CaseVariantGlobalBlocked(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleCreateDocument(StaticRoot(base)), map[string]any{
		"type":      "rule",
		"filename":  "sneaky",
		"directory": "Global/corp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for create under case-variant Global/")
	}
	if msg := resultText(t, result); !strings.Contains(msg, "cannot create document in a read-only global source") {
		t.Errorf("message = %q, want pinned global rejection", msg)
	}
	if _, err := os.Stat(filepath.Join(base, ".archcore", "Global")); !os.IsNotExist(err) {
		t.Error("no directory may be created for a rejected path")
	}
}

func TestHandleUpdateDocument_CaseVariantGlobalBlocked(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "global/corp", "x.rule.md", "---\ntitle: G\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/Global/corp/x.rule.md",
		"title": "Pwned",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for update through case-variant Global/ path")
	}
	if msg := resultText(t, result); !strings.Contains(msg, "cannot update a read-only global source document") {
		t.Errorf("message = %q, want pinned global rejection", msg)
	}
}

func TestHandleCreateDocument_ConcurrentSamePath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	handler := HandleCreateDocument(StaticRoot(base))
	args := map[string]any{
		"type":      "adr",
		"filename":  "raced",
		"directory": "knowledge",
	}

	const workers = 4
	var wg sync.WaitGroup
	errsCh := make(chan bool, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := callTool(handler, args)
			if err != nil {
				t.Error(err)
				return
			}
			errsCh <- result.IsError
		}()
	}
	wg.Wait()
	close(errsCh)

	successes, exists := 0, 0
	for isErr := range errsCh {
		if isErr {
			exists++
		} else {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("exactly one concurrent create must succeed, got %d successes / %d rejections", successes, exists)
	}
}

func TestWriteFileAtomic_RenameFailureCleansTmp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "occupied")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeFileAtomic(target, []byte("data")); err == nil {
		t.Fatal("expected error renaming onto an existing directory")
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file must be removed after a failed rename")
	}
}
