package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGlobalPath(t *testing.T) {
	t.Parallel()
	base := filepath.FromSlash("/a/b")
	tests := []struct {
		name, in, want string
	}{
		{"relative", "c/.archcore", filepath.FromSlash("/a/b/c/.archcore")},
		{"parent-relative", "../c/.archcore", filepath.FromSlash("/a/c/.archcore")},
		{"absolute", filepath.FromSlash("/x/y"), filepath.FromSlash("/x/y")},
		{"absolute-uncleaned", filepath.FromSlash("/x/y/../z"), filepath.FromSlash("/x/z")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveGlobalPath(base, tc.in); got != tc.want {
				t.Errorf("ResolveGlobalPath(%q, %q) = %q, want %q", base, tc.in, got, tc.want)
			}
		})
	}
}

func TestCheckGlobalDir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	archcore := filepath.Join(base, dirName)
	if err := os.MkdirAll(archcore, 0o755); err != nil {
		t.Fatal(err)
	}

	// A readable, empty directory is OK at this layer — "empty of documents" is a
	// reporting-layer warning, not a CheckGlobalDir error.
	emptyDir := filepath.Join(base, "empty-global")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckGlobalDir(base, emptyDir); err != nil {
		t.Errorf("CheckGlobalDir(empty readable dir) = %v, want nil", err)
	}

	// Missing.
	if err := CheckGlobalDir(base, filepath.Join(base, "nope")); !errors.Is(err, ErrGlobalMissing) {
		t.Errorf("missing dir: got %v, want ErrGlobalMissing", err)
	}

	// Not a directory (a file).
	file := filepath.Join(base, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckGlobalDir(base, file); !errors.Is(err, ErrGlobalNotDir) {
		t.Errorf("file path: got %v, want ErrGlobalNotDir", err)
	}

	// Self-overlap: the primary's own .archcore, and an ancestor of it.
	if err := CheckGlobalDir(base, archcore); !errors.Is(err, ErrGlobalSelfOverlap) {
		t.Errorf("own .archcore: got %v, want ErrGlobalSelfOverlap", err)
	}
	if err := CheckGlobalDir(base, base); !errors.Is(err, ErrGlobalSelfOverlap) {
		t.Errorf("ancestor of .archcore: got %v, want ErrGlobalSelfOverlap", err)
	}

	// A descendant of .archcore (legit in-tree vendoring) is NOT self-overlap.
	vendored := filepath.Join(archcore, "global", "company")
	if err := os.MkdirAll(vendored, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckGlobalDir(base, vendored); err != nil {
		t.Errorf("vendored descendant: got %v, want nil", err)
	}
}

func TestCheckGlobalDir_Unreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file permissions are not enforced")
	}
	t.Parallel()
	base := t.TempDir()
	noperm := filepath.Join(base, "noperm")
	if err := os.MkdirAll(noperm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(noperm, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noperm, 0o755) }) // restore so TempDir cleanup works

	if err := CheckGlobalDir(base, noperm); !errors.Is(err, ErrGlobalUnreadable) {
		t.Errorf("unreadable dir: got %v, want ErrGlobalUnreadable", err)
	}
}

func TestDescribeGlobalDirError(t *testing.T) {
	t.Parallel()
	gs := GlobalSource{ID: "company", Path: "../company/.archcore"}
	tests := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{ErrGlobalMissing, `global source "company" not found at "../company/.archcore"`},
		{ErrGlobalNotDir, `global source "company" at "../company/.archcore" is not a directory`},
		{ErrGlobalUnreadable, `global source "company" at "../company/.archcore" is not readable`},
		{ErrGlobalSelfOverlap, `global source "company" at "../company/.archcore" resolves to the project's own .archcore`},
	}
	for _, tc := range tests {
		if got := DescribeGlobalDirError(gs, tc.err); got != tc.want {
			t.Errorf("DescribeGlobalDirError(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
