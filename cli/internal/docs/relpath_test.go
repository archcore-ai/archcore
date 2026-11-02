package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelativeToBase(t *testing.T) {
	t.Parallel()
	w, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(w, "project")
	other := filepath.Join(w, "other")
	for _, d := range []string{filepath.Join(project, ".archcore", "knowledge"), other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(w, "project-link")
	if err := os.Symlink(project, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	otherLink := filepath.Join(w, "other-link")
	if err := os.Symlink(other, otherLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, base, path, want string
		inside                 bool
	}{
		{"relative path", project, ".archcore/x.adr.md", ".archcore/x.adr.md", true},
		{"absolute path inside", project, filepath.Join(project, ".archcore", "x.adr.md"), ".archcore/x.adr.md", true},
		{"absolute path outside", project, filepath.Join(other, "x.go"), "", false},
		{"escape through ..", project, filepath.Join(project, "..", "other", "x.go"), "", false},
		{"existing file through a symlink of the base", project, filepath.Join(link, ".archcore", "knowledge"), ".archcore/knowledge", true},
		{"new file through a symlink of the base", project, filepath.Join(link, ".archcore", "knowledge", "new.adr.md"), ".archcore/knowledge/new.adr.md", true},
		{"new directory through a symlink of the base", project, filepath.Join(link, ".archcore", "new", "x.adr.md"), ".archcore/new/x.adr.md", true},
		{"base spelled through a symlink, path real", link, filepath.Join(project, ".archcore", "x.adr.md"), ".archcore/x.adr.md", true},
		{"symlink to another directory stays outside", project, filepath.Join(otherLink, "x.go"), "", false},
		{"missing base stays outside", filepath.Join(w, "missing"), filepath.Join(project, "x.go"), "", false},
		{"the parent of the base stays outside", project, w, "", false},
		{"a name that starts with .. stays inside", project, filepath.Join(project, "..x"), "..x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, inside := RelativeToBase(tt.base, tt.path)
			if inside != tt.inside || got != tt.want {
				t.Errorf("RelativeToBase(%q, %q) = %q, %v; want %q, %v", tt.base, tt.path, got, inside, tt.want, tt.inside)
			}
		})
	}
}
