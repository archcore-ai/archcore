//go:build windows

package mcp

import "os"

// shieldStdout on Windows keeps the Go-level shield only: os.Stdout is
// pointed at os.Stderr so fmt-style prints from tool executors go to stderr,
// while the returned protocol stream keeps the real stdout. The OS handle for
// stdout is NOT rerouted — cgo, raw writes to fd 1, and child processes
// inheriting the standard handle could still reach the protocol stream, so
// tool executors must not spawn children on Windows.
func shieldStdout() (protocolOut *os.File, restore func()) {
	protocolOut = os.Stdout
	os.Stdout = os.Stderr
	return protocolOut, func() { os.Stdout = protocolOut }
}
