package projectroot

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}
}

func TestNearest(t *testing.T) {
	t.Parallel()
	w := t.TempDir()
	mkdirs(t,
		filepath.Join(w, "a", ".archcore", "mcp"),
		filepath.Join(w, "a", "src", "deep"),
		filepath.Join(w, "a", "app", ".archcore"),
		filepath.Join(w, "a", "app", "src"),
		filepath.Join(w, "plain", "src"),
		filepath.Join(w, "fake", "src"),
	)
	if err := os.WriteFile(filepath.Join(w, "fake", ".archcore"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		path   string
		want   string
		wantOK bool
	}{
		{name: "file in a project", path: filepath.Join(w, "a", "src", "deep", "x.go"), want: filepath.Join(w, "a"), wantOK: true},
		{name: "file that does not exist yet", path: filepath.Join(w, "a", "src", "new", "y.go"), want: filepath.Join(w, "a"), wantOK: true},
		{name: "the project root itself", path: filepath.Join(w, "a"), want: filepath.Join(w, "a"), wantOK: true},
		{name: "a document inside .archcore", path: filepath.Join(w, "a", ".archcore", "mcp", "x.adr.md"), want: filepath.Join(w, "a"), wantOK: true},
		{name: "nested project wins over its parent", path: filepath.Join(w, "a", "app", "src", "z.go"), want: filepath.Join(w, "a", "app"), wantOK: true},
		{name: "document of the nested project", path: filepath.Join(w, "a", "app", ".archcore", "x.spec.md"), want: filepath.Join(w, "a", "app"), wantOK: true},
		{name: "unclean path", path: filepath.Join(w, "a", "src") + "/../src/./x.go", want: filepath.Join(w, "a"), wantOK: true},
		{name: "no project above", path: filepath.Join(w, "plain", "src", "x.go")},
		{name: ".archcore that is a file does not count", path: filepath.Join(w, "fake", "src", "x.go")},
		{name: "empty path", path: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Nearest(tt.path)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Nearest(%q) = %q, %v; want %q, %v", tt.path, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestSameDir(t *testing.T) {
	t.Parallel()
	w := t.TempDir()
	real := filepath.Join(w, "real")
	other := filepath.Join(w, "other")
	mkdirs(t, real, other)
	link := filepath.Join(w, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "identical", a: real, b: real, want: true},
		{name: "trailing separator", a: real + "/", b: real, want: true},
		{name: "symlink and target", a: link, b: real, want: true},
		{name: "different directories", a: real, b: other},
		{name: "missing path", a: filepath.Join(w, "missing"), b: real},
		{name: "two different missing paths", a: filepath.Join(w, "missing"), b: filepath.Join(w, "gone")},
		{name: "missing path with a trailing separator", a: filepath.Join(w, "missing") + "/", b: filepath.Join(w, "missing"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SameDir(tt.a, tt.b); got != tt.want {
				t.Errorf("SameDir(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestBelow(t *testing.T) {
	t.Parallel()
	w := t.TempDir()
	mkdirs(t,
		filepath.Join(w, "a", ".archcore"),
		filepath.Join(w, "a", "app", ".archcore"),
		filepath.Join(w, "b", ".archcore"),
		filepath.Join(w, "group", "c", ".archcore"),
		filepath.Join(w, "group", "deep", "deeper", "d", ".archcore"),
		filepath.Join(w, ".hidden", ".archcore"),
		filepath.Join(w, "node_modules", "pkg", ".archcore"),
		filepath.Join(w, "vendor", "v", ".archcore"),
		filepath.Join(w, "x", ".claude", "plugins", "cache", "p", ".archcore"),
		filepath.Join(w, "y", "plugins", "cache", "q", ".archcore"),
		filepath.Join(w, "plain", "src"),
	)
	if err := os.WriteFile(filepath.Join(w, "plain", ".archcore"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	roots, complete := Below(w, 3, 1000)
	want := []string{filepath.Join(w, "a"), filepath.Join(w, "b"), filepath.Join(w, "group", "c")}
	if !equalStrings(roots, want) {
		t.Errorf("Below = %q, want %q", roots, want)
	}
	if complete {
		t.Error("complete = true, but the depth bound hid group/deep/deeper/d")
	}

	roots, complete = Below(w, 5, 1000)
	if want := append(want, filepath.Join(w, "group", "deep", "deeper", "d")); !equalStrings(roots, want) || !complete {
		t.Errorf("Below(depth 5) = %q, %v; want %q, true", roots, complete, want)
	}

	if roots, complete := Below(w, 5, 1); len(roots) != 2 || complete {
		t.Errorf("Below(maxDirs 1) = %q, %v; want the first level only and incomplete", roots, complete)
	}
	if roots, complete := Below(filepath.Join(w, "missing"), 3, 10); len(roots) != 0 || !complete {
		t.Errorf("Below(missing) = %q, %v", roots, complete)
	}
	if roots, _ := Below(filepath.Join(w, "a"), 3, 10); len(roots) != 1 || roots[0] != filepath.Join(w, "a", "app") {
		t.Errorf("Below(project) = %q, want its nested project", roots)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCanonical(t *testing.T) {
	t.Parallel()
	w, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(w, "real")
	mkdirs(t, real)
	link := filepath.Join(w, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	missing := filepath.Join(w, "missing")

	for in, want := range map[string]string{real: real, link: real, link + "/": real, missing: missing} {
		if got := Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNearest_StopsAtADirectoryItMayNotSearch holds Failure Behavior 8 of
// call-root-resolution.spec: an unsearchable directory ends the walk, so the
// caller falls back to its working directory, not to a farther ancestor.
func TestNearest_StopsAtADirectoryItMayNotSearch(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root searches every directory")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	for _, p := range []string{filepath.Join(dir, ".archcore"), filepath.Join(locked, "sub")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	if root, ok := Nearest(filepath.Join(locked, "sub", "x.md")); ok {
		t.Errorf("Nearest = %q, want no project", root)
	}
}
