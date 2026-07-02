package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// resolveProjectRoot picks the project root from (in order):
//  1. flagValue if non-empty
//  2. envValue if non-empty
//  3. os.Getwd()
//
// The returned path is absolute. Errors if the source cannot be made absolute,
// does not exist, or is not a directory.
//
// envValue is passed in (not read inside) so the caller controls precedence
// visibly and tests do not have to mutate process env.
func resolveProjectRoot(flagValue, envValue string) (string, error) {
	var source string
	switch {
	case flagValue != "":
		source = flagValue
	case envValue != "":
		source = envValue
	default:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to determine working directory: %w", err)
		}
		source = cwd
	}

	abs, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project root %q: %w", source, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("project root does not exist: %q", abs)
		}
		return "", fmt.Errorf("failed to stat project root %q: %w", abs, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory: %q", abs)
	}

	return abs, nil
}
