//go:build windows

package cmd

import (
	"os"

	"golang.org/x/sys/windows"
)

// defaultIsInteractive reports whether prompts can be shown. Windows has no
// /dev/tty: huh reads CONIN$, and stdin must be the console too, because a CI
// runner attaches a console nobody types into while it redirects stdin.
func defaultIsInteractive() bool {
	var mode uint32
	if windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode) != nil {
		return false
	}
	f, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
