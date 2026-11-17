//go:build windows

package cmd

import (
	"os"
	"testing"
)

// A console with stdin redirected (a CI step, `archcore init < nul`) is not a
// place to open a picker: the probe must say no even though CONIN$ opens.
func TestDefaultIsInteractive_RedirectedStdinIsNotInteractive(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = devNull.Close() })
	orig := os.Stdin
	os.Stdin = devNull
	t.Cleanup(func() { os.Stdin = orig })

	if defaultIsInteractive() {
		t.Error("defaultIsInteractive() = true with stdin redirected to NUL")
	}
}
