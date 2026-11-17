//go:build !darwin

package update

// checkOSMinimum accepts every binary outside macOS. Residual invariant: a
// release that raises the minimum Linux kernel or Windows version is stopped
// only by the `--version` run of healthProbe, so it must fail at startup there.
func checkOSMinimum(string) error { return nil }
