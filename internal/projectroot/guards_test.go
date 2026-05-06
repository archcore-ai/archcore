package projectroot

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGuard_RejectHome(t *testing.T) {
	t.Parallel()
	home := mkTree(t)
	res := &Resolution{Path: home, Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: homeFn(home),
		Stat: os.Stat,
	}), false)
	if err == nil {
		t.Fatal("expected error for HOME, got nil")
	}
	if !errors.Is(err, ErrBaseDirHome) {
		t.Errorf("err is not ErrBaseDirHome: %v", err)
	}
	var re *ResolveError
	if errors.As(err, &re) && re.Code != CodeHomeRefused {
		t.Errorf("Code = %q, want %q", re.Code, CodeHomeRefused)
	}
}

func TestGuard_RejectRoot(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("filesystem root semantics differ on Windows")
	}
	res := &Resolution{Path: "/", Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), false)
	if err == nil {
		t.Fatal("expected error for /, got nil")
	}
	if !errors.Is(err, ErrBaseDirSystem) {
		t.Errorf("err is not ErrBaseDirSystem: %v", err)
	}
}

func TestGuard_RejectTmpVarTmp(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("/tmp checks are POSIX-only")
	}
	for _, p := range []string{"/tmp", "/var/tmp"} {
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			info, err := os.Stat(p)
			if err != nil || !info.IsDir() {
				t.Skipf("%s not a real dir on this host", p)
			}
			res := &Resolution{Path: p, Source: SourceFlag}
			err = validate(res, withDefaults(Options{
				Mode: ModeRuntime,
				Home: neutralHome(t),
				Stat: os.Stat,
			}), false)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", p)
			}
			if !errors.Is(err, ErrBaseDirSystem) {
				t.Errorf("err is not ErrBaseDirSystem: %v", err)
			}
		})
	}
}

func TestGuard_AllowProjectMarker(t *testing.T) {
	t.Parallel()
	cases := []string{".archcore", ".git", "go.mod", "package.json"}
	for _, marker := range cases {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()
			var entry string
			if marker == ".archcore" || marker == ".git" {
				entry = marker + "/"
			} else {
				entry = marker
			}
			dir := mkTree(t, entry)
			res := &Resolution{Path: dir, Source: SourceFlag}
			err := validate(res, withDefaults(Options{
				Mode: ModeRuntime,
				Home: neutralHome(t),
				Stat: os.Stat,
			}), false)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", marker, err)
			}
		})
	}
}

func TestGuard_RejectNoMarkers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res := &Resolution{Path: dir, Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), false)
	if err == nil {
		t.Fatal("expected error for no-markers dir, got nil")
	}
	if !errors.Is(err, ErrBaseDirNoMarkers) {
		t.Errorf("err is not ErrBaseDirNoMarkers: %v", err)
	}
	var re *ResolveError
	if errors.As(err, &re) && re.Code != CodeNotProject {
		t.Errorf("Code = %q, want %q", re.Code, CodeNotProject)
	}
}

func TestGuard_BypassLegacy(t *testing.T) {
	t.Parallel()
	// HOME would normally be rejected, but legacy bypasses HOME/system/marker checks.
	home := mkTree(t)
	res := &Resolution{Path: home, Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: homeFn(home),
		Stat: os.Stat,
	}), true) // legacy=true
	if err != nil {
		t.Errorf("expected nil error in legacy mode, got: %v", err)
	}
}

func TestGuard_LegacyStillChecksExistence(t *testing.T) {
	t.Parallel()
	res := &Resolution{Path: "/no/such/path", Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), true) // legacy=true
	if err == nil {
		t.Fatal("legacy mode should still reject nonexistent paths")
	}
	if !errors.Is(err, ErrBaseDirNotExist) {
		t.Errorf("err is not ErrBaseDirNotExist: %v", err)
	}
}

func TestGuard_InitAllowsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res := &Resolution{Path: dir, Source: SourceCwd}
	err := validate(res, withDefaults(Options{
		Mode: ModeInit,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), false)
	if err != nil {
		t.Errorf("ModeInit should allow empty dir, got: %v", err)
	}
}

func TestGuard_HooksAllowsEmptyWithExplicitSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, src := range []Source{SourceFlag, SourceEnv} {
		t.Run(string(src), func(t *testing.T) {
			t.Parallel()
			res := &Resolution{Path: dir, Source: src}
			err := validate(res, withDefaults(Options{
				Mode: ModeHooks,
				Home: neutralHome(t),
				Stat: os.Stat,
			}), false)
			if err != nil {
				t.Errorf("ModeHooks with %s should allow empty dir, got: %v", src, err)
			}
		})
	}
}

func TestGuard_HooksRejectsEmptyWalkUp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res := &Resolution{Path: dir, Source: SourceCwd}
	err := validate(res, withDefaults(Options{
		Mode: ModeHooks,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), false)
	if err == nil {
		t.Errorf("ModeHooks with implicit source should reject empty dir")
	}
}

func TestGuard_NotADirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "notadir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := &Resolution{Path: file, Source: SourceFlag}
	err := validate(res, withDefaults(Options{
		Mode: ModeRuntime,
		Home: neutralHome(t),
		Stat: os.Stat,
	}), false)
	if err == nil {
		t.Fatal("expected error for file path, got nil")
	}
	if !errors.Is(err, ErrBaseDirNotExist) {
		t.Errorf("err is not ErrBaseDirNotExist: %v", err)
	}
}
