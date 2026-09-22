package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"archcore-cli/internal/projectroot"
)

// resolveProjectRoot picks the project root from (in order):
//  1. flagValue if non-empty
//  2. envValue if non-empty
//  3. os.Getwd()
//
// The returned path is absolute. Errors if the source cannot be made absolute,
// does not exist, or is not a directory.
//
// The plugin-cache guard applies only to the implicit sources (env, cwd) —
// those are the ones a host can misroute. An explicit --project flag states
// user intent and is trusted as-is, which also keeps it usable as the
// recovery path the guard's own error message recommends.
//
// envValue is passed in (not read inside) so the caller controls precedence
// visibly and tests do not have to mutate process env.
func resolveProjectRoot(flagValue, envValue string) (string, error) {
	source := flagValue
	explicit := flagValue != ""
	if !explicit {
		if envValue != "" {
			source = envValue
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to determine working directory: %w", err)
			}
			source = cwd
		}
	}

	abs, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project root %q: %w", source, err)
	}

	if !explicit && projectroot.IsPluginCachePath(abs) {
		return "", fmt.Errorf(
			"refusing project root %q: path is inside an AI-host plugin install cache, not a user project (the host likely misrouted the working directory — pass --project to the real project root, or register a project-level server: archcore init --agent <agent> --project <path>)", abs)
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

// rootIsPinned reports whether the served root was stated rather than inferred.
// It mirrors resolveProjectRoot's precedence over the same two inputs: either
// source states user intent, and a stated root is served for the process
// lifetime without ever querying the client (project-root-resolution.spec §1
// and §2).
func rootIsPinned(flagValue, envValue string) bool {
	return flagValue != "" || envValue != ""
}
