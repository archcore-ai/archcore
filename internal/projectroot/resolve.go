package projectroot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mode controls guard relaxation.
type Mode int

const (
	// ModeRuntime is the default: must exist + have project markers
	// (or be the .archcore parent itself).
	ModeRuntime Mode = iota
	// ModeInit allows a missing .archcore/ — the caller is about to
	// create it. Markers are not required; existence still is.
	ModeInit
	// ModeHooks is lenient on markers when source==flag/env, since the
	// host MCP integration explicitly chose this directory.
	ModeHooks
)

// Source labels how the path was resolved. Stable strings — surfaced in
// the stderr banner, the which_project MCP tool, and `archcore where`.
type Source string

const (
	SourceFlag         Source = "flag"
	SourceEnv          Source = "env"
	SourceWalkArchcore Source = "walk-up:.archcore"
	SourceWalkGit      Source = "walk-up:.git"
	SourceWalkMarker   Source = "walk-up:marker"
	SourceCwd          Source = "cwd"
	SourceInitCwd      Source = "init-cwd"
)

// Resolution is the immutable result of a successful resolve.
type Resolution struct {
	Path        string
	Source      Source
	Marker      string
	WalkDepth   int
	LegacyMode  bool
	HasArchcore bool
}

// Options is the input to Resolve. The zero value is valid for ModeRuntime.
//
// The Getwd, Getenv, Home, and Stat fields are test seams: nil means use
// the real syscall. Tests set these directly rather than mutating process
// env or chdir-ing — the package never reads os.Getwd / os.Getenv / etc.
// outside these seams.
type Options struct {
	Flag string
	Mode Mode

	Getwd  func() (string, error)
	Getenv func(string) string
	Home   func() (string, error)
	Stat   func(string) (os.FileInfo, error)
}

const (
	envBaseDir = "ARCHCORE_BASE_DIR"
	envLegacy  = "ARCHCORE_LEGACY_BASE_DIR"
)

// Resolve runs the priority chain (flag → env → walk-up → cwd) and applies
// guards. On error, the returned error is a *ResolveError; callers can
// access its Code field directly or pass through FormatError for the
// stderr block.
func Resolve(opts Options) (*Resolution, error) {
	o := withDefaults(opts)
	legacy := o.Getenv(envLegacy) == "1"

	if o.Flag != "" {
		path, err := expandPath(o.Flag, o)
		if err != nil {
			return nil, err
		}
		return finalize(&Resolution{Path: path, Source: SourceFlag}, o, legacy)
	}

	if v := o.Getenv(envBaseDir); v != "" {
		path, err := expandPath(v, o)
		if err != nil {
			return nil, err
		}
		return finalize(&Resolution{Path: path, Source: SourceEnv}, o, legacy)
	}

	cwd, err := o.Getwd()
	if err != nil {
		return nil, &ResolveError{
			Sentinel: ErrNoProject,
			Code:     CodeNoProject,
			Reason:   fmt.Sprintf("could not determine current directory: %v", err),
			Fix:      "pass --base-dir <path> or set ARCHCORE_BASE_DIR",
		}
	}
	cwd = filepath.Clean(cwd)
	if !filepath.IsAbs(cwd) {
		if abs, absErr := filepath.Abs(cwd); absErr == nil {
			cwd = abs
		}
	}

	if hit, ok := walkUp(cwd, o); ok {
		return finalize(hit, o, legacy)
	}

	if o.Mode == ModeInit {
		return finalize(&Resolution{Path: cwd, Source: SourceInitCwd}, o, legacy)
	}

	return nil, &ResolveError{
		Sentinel: ErrNoProject,
		Code:     CodeNoProject,
		Path:     cwd,
		Source:   SourceCwd,
		Reason:   fmt.Sprintf("no project markers found from %s (walked up %d levels)", cwd, walkDepthLimit),
		Fix:      "cd into a project directory, or pass --base-dir <path>, or set ARCHCORE_BASE_DIR",
	}
}

// expandPath resolves leading ~ and turns relative paths absolute.
func expandPath(p string, o Options) (string, error) {
	if strings.HasPrefix(p, "~") {
		home, err := o.Home()
		if err != nil {
			return "", &ResolveError{
				Sentinel: ErrTildeExpand,
				Code:     CodeTildeExpand,
				Reason:   err.Error(),
				Fix:      "use an absolute path or set $HOME",
			}
		}
		switch {
		case p == "~":
			p = home
		case strings.HasPrefix(p, "~/"):
			p = filepath.Join(home, p[2:])
		}
	}
	if !filepath.IsAbs(p) {
		cwd, err := o.Getwd()
		if err != nil {
			// Same shape as the top-level Getwd failure in Resolve — a
			// failure to determine the cwd means we cannot anchor a relative
			// path, which is morally "no project found", not a missing base
			// directory. Returning ErrBaseDirNotExist here would give callers
			// a false positive on errors.Is(err, ErrBaseDirNotExist).
			return "", &ResolveError{
				Sentinel: ErrNoProject,
				Code:     CodeNoProject,
				Reason:   fmt.Sprintf("could not determine current directory for relative path %q: %v", p, err),
				Fix:      "use an absolute path",
			}
		}
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p), nil
}

// finalize annotates the resolution and runs guards.
func finalize(res *Resolution, o Options, legacy bool) (*Resolution, error) {
	res.LegacyMode = legacy
	res.HasArchcore = hasMarker(res.Path, ".archcore", o)

	if err := validate(res, o, legacy); err != nil {
		var re *ResolveError
		if errors.As(err, &re) {
			if re.Path == "" {
				re.Path = res.Path
			}
			if re.Source == "" {
				re.Source = res.Source
			}
		}
		return nil, err
	}
	return res, nil
}

// hasMarker reports whether dir contains the named marker.
func hasMarker(dir, name string, o Options) bool {
	_, err := o.Stat(filepath.Join(dir, name))
	return err == nil
}

// withDefaults fills nil seams with real syscall implementations.
func withDefaults(o Options) Options {
	if o.Getwd == nil {
		o.Getwd = os.Getwd
	}
	if o.Getenv == nil {
		o.Getenv = os.Getenv
	}
	if o.Home == nil {
		o.Home = os.UserHomeDir
	}
	if o.Stat == nil {
		o.Stat = os.Stat
	}
	return o
}

// --- Context cache: the cmd-layer resolves once per command, sub-checks read it. ---

type ctxKey struct{}

// WithResolution attaches r to ctx so downstream callers can read it via From.
// Keep this scoped to one command invocation; never use as a global cache.
func WithResolution(ctx context.Context, r *Resolution) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

// From retrieves a Resolution previously attached via WithResolution.
func From(ctx context.Context) (*Resolution, bool) {
	r, ok := ctx.Value(ctxKey{}).(*Resolution)
	return r, ok
}
