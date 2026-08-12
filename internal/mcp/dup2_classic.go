//go:build unix && !linux

// Package mcp serves the Archcore document tools over the Model Context
// Protocol on stdio. It shields the protocol channel: anything a dependency
// writes to the real stdout would corrupt a JSON-RPC frame, so the descriptor
// is redirected before the server starts.
package mcp

import "syscall"

// dup2 makes newfd refer to the same file as oldfd.
func dup2(oldfd, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
