//go:build unix && !linux

package mcp

import "syscall"

// dup2 makes newfd refer to the same file as oldfd.
func dup2(oldfd, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
