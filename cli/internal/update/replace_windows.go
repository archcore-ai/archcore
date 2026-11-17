//go:build windows

package update

// commitStaged commits through commitViaAside: Windows cannot overwrite a
// running .exe. Residual invariant: Windows exposes no directory handle to
// fsync, so the rename is not made durable against a power loss; a caller must
// not treat a returned nil as crash-durable on this platform.
func commitStaged(target, tmpPath string) error {
	return commitViaAside(target, tmpPath)
}
