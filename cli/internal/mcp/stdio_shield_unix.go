//go:build unix

package mcp

import (
	"os"
	"syscall"
)

// shieldStdout redirects the process's stdout at the file-descriptor level so
// the JSON-RPC stream owns the only route to the real stdout.
//
// It duplicates fd 1 into a private CLOEXEC descriptor (the returned protocol
// stream), then points fd 1 at stderr. After that, *anything* that writes to
// stdout — fmt via os.Stdout, cgo, raw syscall.Write(1, …), or a child
// process inheriting fd 1 — lands on stderr instead of corrupting protocol
// frames. Child processes cannot reach the protocol stream at all: its
// descriptor is close-on-exec.
//
// If any descriptor operation fails, it degrades to the Go-level-only shield
// (os.Stdout = os.Stderr) and returns the original stdout as the protocol
// stream — the server must still start (fail open).
//
// restore undoes the swap: fd 1 points back at the original stdout and
// os.Stdout is reinstated. A tool-handler goroutine still running after
// restore would print to the restored stdout — the stream is closing then, so
// this is accepted.
func shieldStdout() (protocolOut *os.File, restore func()) {
	goOut := os.Stdout

	dupFD, err := syscall.Dup(1)
	if err != nil {
		os.Stdout = os.Stderr
		return goOut, func() { os.Stdout = goOut }
	}
	syscall.CloseOnExec(dupFD)

	if err := dup2(int(os.Stderr.Fd()), 1); err != nil {
		_ = syscall.Close(dupFD)
		os.Stdout = os.Stderr
		return goOut, func() { os.Stdout = goOut }
	}

	protocolOut = os.NewFile(uintptr(dupFD), "protocol-stdout")
	os.Stdout = os.Stderr
	return protocolOut, func() {
		_ = dup2(dupFD, 1) // fd 1 points back at the original stdout
		os.Stdout = goOut
		_ = protocolOut.Close()
	}
}
