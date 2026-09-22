//go:build windows

package mcp

import "regexp"

// windowsDriveURIRe matches the "/C:/…" shape a file:// URI carries on Windows,
// where the drive letter must lose its leading separator.
var windowsDriveURIRe = regexp.MustCompile(`^/[A-Za-z]:`)

// stripURIDrivePrefix removes the separator a file:// URI puts in front of a
// Windows drive letter, so "file:///C:/repo" yields "C:/repo".
func stripURIDrivePrefix(path string) string {
	if windowsDriveURIRe.MatchString(path) {
		return path[1:]
	}
	return path
}
