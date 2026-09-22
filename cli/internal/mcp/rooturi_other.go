//go:build !windows

package mcp

// stripURIDrivePrefix returns path unchanged: no drive letter exists to strip.
//
// This is the weaker variant. Its residual invariant is that a URI path here is
// already the local path, so "/C:/repo" is a directory literally named "C:" at
// the filesystem root rather than a drive — the Windows variant would strip it
// into a relative path that acceptRoot then refuses.
func stripURIDrivePrefix(path string) string {
	return path
}
