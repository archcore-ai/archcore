//go:build !windows

package mcp

import "testing"

// TestFileURIPath_DriveShapeIsAPathHere pins the residual invariant of the
// non-Windows stripURIDrivePrefix: "/C:/repo" names a directory, and stripping
// it would turn an absolute path into a relative one that acceptRoot refuses.
func TestFileURIPath_DriveShapeIsAPathHere(t *testing.T) {
	t.Parallel()
	got, err := fileURIPath("file:///C:/repo")
	if err != nil {
		t.Fatalf("fileURIPath: %v", err)
	}
	if want := "/C:/repo"; got != want {
		t.Errorf("fileURIPath(file:///C:/repo) = %q, want %q", got, want)
	}
}
