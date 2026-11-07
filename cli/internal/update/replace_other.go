//go:build !windows

package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// commitStaged renames the staged file over target, removing it on any failure
// so an abandoned attempt leaves nothing behind.
func commitStaged(target, tmpPath string) error {
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp to target: %w", err)
	}
	// writeBinarySynced made the contents durable; this makes the directory
	// entry that points at them durable too. Best-effort on purpose: the binary
	// is already in place, so a failure here must not report a completed update
	// as failed.
	_ = syncDir(filepath.Dir(target))

	return nil
}

// syncDir fsyncs a directory so a rename inside it survives a power loss.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
