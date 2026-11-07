//go:build !windows

package cmd

import "os"

// defaultIsInteractive reports whether prompts can be shown. It probes /dev/tty
// because huh opens it explicitly; stdin being a char device is not enough.
func defaultIsInteractive() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
