//go:build linux

package mcp

import "syscall"

// dup2 makes newfd refer to the same file as oldfd. Newer Linux ports
// (arm64, riscv64, loong64) ship only dup3(2), so route through it.
func dup2(oldfd, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
