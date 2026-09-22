package docs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
)

// The guard is the single answer to "may this path be written?", shared by the
// MCP write tools and the pre-tool-use hook. These tests exercise it directly,
// without a protocol layer, so a hook regression cannot hide behind a passing
// MCP test.

func TestValidateArchcorePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		relPath     string
		want        string
		errContains string
	}{
		{name: "plain document", relPath: ".archcore/knowledge/a.adr.md", want: ".archcore/knowledge/a.adr.md"},
		{name: "root level document", relPath: ".archcore/a.adr.md", want: ".archcore/a.adr.md"},
		{name: "redundant segments are cleaned", relPath: ".archcore/./knowledge/../a.adr.md", want: ".archcore/a.adr.md"},
		{name: "absolute path rejected", relPath: "/etc/passwd", errContains: "must be relative"},
		{name: "missing prefix rejected", relPath: "knowledge/a.adr.md", errContains: "must start with"},
		{name: "traversal out of tree rejected", relPath: ".archcore/../../etc/passwd", errContains: "must be relative"},
		{name: "empty path rejected", relPath: "", errContains: "must start with"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateArchcorePath(tt.relPath)
			if tt.errContains != "" {
				if err == nil {
					t.Fatalf("ValidateArchcorePath(%q) = %q, want error", tt.relPath, got)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ValidateArchcorePath(%q) = %q, want %q", tt.relPath, got, tt.want)
			}
		})
	}
}

func TestGuardWritablePath(t *testing.T) {
	t.Parallel()
	globals := []config.GlobalSource{{ID: "company", Path: ".archcore/global/company"}}

	tests := []struct {
		name    string
		relPath string
		wantErr error
	}{
		{name: "local document is writable", relPath: ".archcore/knowledge/a.adr.md"},
		{name: "nested local document is writable", relPath: ".archcore/a/b/c.rule.md"},
		{name: "settings.json is not a document", relPath: ".archcore/settings.json", wantErr: ErrPathNotDocument},
		{name: "sync state is not a document", relPath: ".archcore/.sync-state.json", wantErr: ErrPathNotDocument},
		{name: "non-markdown is not a document", relPath: ".archcore/knowledge/a.txt", wantErr: ErrPathNotDocument},
		{name: "reserved global tree is read-only", relPath: ".archcore/global/x.rule.md", wantErr: ErrPathReadOnlyGlobal},
		{name: "nested reserved global is read-only", relPath: ".archcore/a/global/b.rule.md", wantErr: ErrPathReadOnlyGlobal},
		{name: "case-variant reserved global is read-only", relPath: ".archcore/Global/x.rule.md", wantErr: ErrPathReadOnlyGlobal},
		{name: "declared global source is read-only", relPath: ".archcore/global/company/x.rule.md", wantErr: ErrPathReadOnlyGlobal},
		{name: "global-ish sibling stays writable", relPath: ".archcore/global-ish/x.rule.md"},
		{name: "global in filename stays writable", relPath: ".archcore/knowledge/global.rule.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := GuardWritablePath(base, tt.relPath, globals)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GuardWritablePath(%q) error = %v, want %v", tt.relPath, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GuardWritablePath(%q) unexpected error: %v", tt.relPath, err)
			}
			if got == "" {
				t.Errorf("GuardWritablePath(%q) returned an empty path", tt.relPath)
			}
		})
	}
}

func TestNormalizeRelPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "strips the archcore prefix", in: ".archcore/knowledge/a.adr.md", want: "knowledge/a.adr.md"},
		{name: "leaves a manifest path alone", in: "knowledge/a.adr.md", want: "knowledge/a.adr.md"},
		{name: "strips only the leading prefix", in: ".archcore/.archcore/a.adr.md", want: ".archcore/a.adr.md"},
		{name: "empty stays empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeRelPath(tt.in); got != tt.want {
				t.Errorf("NormalizeRelPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
