//go:build windows

package mcp

import "os"

// shieldStdout on Windows keeps the Go-level shield only: os.Stdout is
// pointed at os.Stderr so fmt-style prints from tool executors go to stderr,
// while the returned protocol stream keeps the real stdout. The OS handle for
// stdout is NOT rerouted — cgo, raw writes to fd 1, and child processes
// inheriting the standard handle could still reach the protocol stream, so
// tool executors must not spawn children on Windows.
//
// RunStdio's background task widens that residual: it is not a tool executor,
// it outlives the shield, and through the unattended policy it does exec a
// child — the health probe on the staged binary. Here that child is contained
// by the probe leaving Stdout nil, which os/exec resolves to the null device,
// and not by this shield. A probe changed to hand the staged binary our streams
// would put a downloaded binary's first line into the host's frame stream on
// Windows alone, where no test can see it — mcp-background-update.spec §6.
func shieldStdout() (protocolOut *os.File, restore func()) {
	protocolOut = os.Stdout
	os.Stdout = os.Stderr
	return protocolOut, func() { os.Stdout = protocolOut }
}
