package projectroot

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// validate runs guards over a resolution. Returns nil on pass, *ResolveError on fail.
//
// Order of checks:
//  1. Existence (always — even in legacy mode)
//  2. legacy bypass for the rest
//  3. HOME equality
//  4. System path
//  5. Marker presence (skipped for ModeInit and ModeHooks-with-explicit-source)
func validate(res *Resolution, o Options, legacy bool) error {
	if res.Path == "" {
		return &ResolveError{
			Sentinel: ErrBaseDirNotExist,
			Code:     CodePathInvalid,
			Reason:   "empty path",
			Fix:      "pass --base-dir <path> or set ARCHCORE_BASE_DIR",
		}
	}

	info, err := o.Stat(res.Path)
	if err != nil {
		return &ResolveError{
			Sentinel: ErrBaseDirNotExist,
			Code:     CodePathInvalid,
			Reason:   fmt.Sprintf("stat: %v", err),
			Fix:      "create the directory or fix the path",
		}
	}
	if !info.IsDir() {
		return &ResolveError{
			Sentinel: ErrBaseDirNotExist,
			Code:     CodePathInvalid,
			Reason:   "path is not a directory",
			Fix:      "use a directory path",
		}
	}

	if legacy {
		return nil
	}

	home, _ := o.Home()
	if home != "" && filepath.Clean(res.Path) == filepath.Clean(home) {
		return &ResolveError{
			Sentinel: ErrBaseDirHome,
			Code:     CodeHomeRefused,
			Reason:   "directory matches $HOME; refusing under strict mode",
			Fix:      "cd into a project directory; or set ARCHCORE_LEGACY_BASE_DIR=1 to bypass (not recommended)",
		}
	}

	if isSystemPath(res.Path) {
		return &ResolveError{
			Sentinel: ErrBaseDirSystem,
			Code:     CodePathInvalid,
			Reason:   "directory is a system path; refusing",
			Fix:      "use a project-local directory",
		}
	}

	if o.Mode == ModeInit {
		return nil
	}
	if o.Mode == ModeHooks && (res.Source == SourceFlag || res.Source == SourceEnv) {
		return nil
	}
	if !hasAnyMarker(res.Path, o) {
		return &ResolveError{
			Sentinel: ErrBaseDirNoMarkers,
			Code:     CodeNotProject,
			Reason:   "no project markers found in this directory",
			Fix:      fmt.Sprintf("expected one of: %s", strings.Join(Markers(), ", ")),
		}
	}
	return nil
}

func hasAnyMarker(dir string, o Options) bool {
	for _, m := range allMarkers {
		if hasMarker(dir, m, o) {
			return true
		}
	}
	return false
}

func isSystemPath(p string) bool {
	p = filepath.Clean(p)
	if p == string(filepath.Separator) {
		return true
	}
	if runtime.GOOS != "windows" {
		for _, sd := range []string{"/tmp", "/var/tmp"} {
			if p == sd {
				return true
			}
		}
	}
	if td := os.TempDir(); td != "" && filepath.Clean(td) == p {
		return true
	}
	return false
}
