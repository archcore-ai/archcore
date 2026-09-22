//go:build windows

package mcp

import "testing"

// TestFileURIPath_WindowsDrive pins the drive-letter case under the constraint
// that owns it (platform-splits-are-files.rule §7): the URI path carries a
// leading separator the local path must not keep.
func TestFileURIPath_WindowsDrive(t *testing.T) {
	t.Parallel()
	got, err := fileURIPath("file:///C:/repo")
	if err != nil {
		t.Fatalf("fileURIPath: %v", err)
	}
	if want := `C:\repo`; got != want {
		t.Errorf("fileURIPath(file:///C:/repo) = %q, want %q", got, want)
	}
}
