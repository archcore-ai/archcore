package projectroot

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkTree builds a temp directory with the given relative entries.
// Entries ending in "/" are dirs; others are files.
func mkTree(t *testing.T, entries ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, e := range entries {
		full := filepath.Join(root, e)
		if strings.HasSuffix(e, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", full, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir parent of %s: %v", full, err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

func envFn(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func homeFn(p string) func() (string, error) {
	return func() (string, error) { return p, nil }
}

func cwdFn(p string) func() (string, error) {
	return func() (string, error) { return p, nil }
}

// neutralHome returns a func that points to a never-matched temp dir, so
// HOME never collides with the test fixtures.
func neutralHome(t *testing.T) func() (string, error) {
	t.Helper()
	d := t.TempDir()
	return func() (string, error) { return d, nil }
}

func TestResolve_FlagWins(t *testing.T) {
	t.Parallel()
	proj := mkTree(t, ".archcore/", ".git/")
	cwd := mkTree(t, ".git/")

	res, err := Resolve(Options{
		Flag:   proj,
		Mode:   ModeRuntime,
		Getwd:  cwdFn(cwd),
		Home:   neutralHome(t),
		Getenv: envFn(map[string]string{envBaseDir: "/elsewhere"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceFlag {
		t.Errorf("Source = %q, want %q", res.Source, SourceFlag)
	}
	if res.Path != filepath.Clean(proj) {
		t.Errorf("Path = %q, want %q", res.Path, proj)
	}
	if !res.HasArchcore {
		t.Errorf("HasArchcore = false, want true")
	}
}

func TestResolve_EnvWinsOverWalk(t *testing.T) {
	t.Parallel()
	envProj := mkTree(t, ".git/")
	walkProj := mkTree(t, ".git/")

	res, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(walkProj),
		Home:   neutralHome(t),
		Getenv: envFn(map[string]string{envBaseDir: envProj}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceEnv {
		t.Errorf("Source = %q, want %q", res.Source, SourceEnv)
	}
	if res.Path != filepath.Clean(envProj) {
		t.Errorf("Path = %q, want %q", res.Path, envProj)
	}
}

func TestResolve_WalkUpFindsArchcore(t *testing.T) {
	t.Parallel()
	root := mkTree(t, ".archcore/", "sub/sub2/")
	cwd := filepath.Join(root, "sub", "sub2")

	res, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(cwd),
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceWalkArchcore {
		t.Errorf("Source = %q, want %q", res.Source, SourceWalkArchcore)
	}
	if res.Path != filepath.Clean(root) {
		t.Errorf("Path = %q, want %q", res.Path, root)
	}
	if res.WalkDepth != 2 {
		t.Errorf("WalkDepth = %d, want 2", res.WalkDepth)
	}
}

func TestResolve_WalkUpFindsGit(t *testing.T) {
	t.Parallel()
	root := mkTree(t, ".git/", "sub/")
	cwd := filepath.Join(root, "sub")

	res, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(cwd),
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceWalkGit {
		t.Errorf("Source = %q, want %q", res.Source, SourceWalkGit)
	}
}

func TestResolve_WalkUpFindsMarker_Generic(t *testing.T) {
	t.Parallel()
	cases := []string{"go.mod", "package.json", "pyproject.toml", "Cargo.toml", "pom.xml", "composer.json", "Gemfile"}
	for _, marker := range cases {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()
			root := mkTree(t, marker, "sub/")
			cwd := filepath.Join(root, "sub")

			res, err := Resolve(Options{
				Mode:   ModeRuntime,
				Getwd:  cwdFn(cwd),
				Home:   neutralHome(t),
				Getenv: envFn(nil),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Source != SourceWalkMarker {
				t.Errorf("Source = %q, want %q", res.Source, SourceWalkMarker)
			}
			if res.Marker != marker {
				t.Errorf("Marker = %q, want %q", res.Marker, marker)
			}
		})
	}
}

func TestResolve_WalkUpAtHomeReturnsHomeRefused(t *testing.T) {
	t.Parallel()
	// HOME has .git — walk-up must still evaluate it, the validate guard
	// then converts the hit into ERR_HOME_REFUSED (not ERR_NO_PROJECT).
	home := mkTree(t, ".git/", "child/")
	cwd := filepath.Join(home, "child")

	_, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(cwd),
		Home:   homeFn(home),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatalf("expected error when walk-up resolves to HOME, got nil")
	}
	if !errors.Is(err, ErrBaseDirHome) {
		t.Errorf("err is not ErrBaseDirHome: %v", err)
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		t.Fatalf("err is not *ResolveError: %v", err)
	}
	if re.Code != CodeHomeRefused {
		t.Errorf("Code = %q, want %q", re.Code, CodeHomeRefused)
	}
}

func TestResolve_FromHomeWithRepoStyleLayout(t *testing.T) {
	t.Parallel()
	// cwd == HOME, with .git directly under HOME.
	home := mkTree(t, ".git/")

	_, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(home),
		Home:   homeFn(home),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatalf("expected error when cwd==HOME with .git, got nil")
	}
	if !errors.Is(err, ErrBaseDirHome) {
		t.Errorf("err is not ErrBaseDirHome: %v", err)
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		t.Fatalf("err is not *ResolveError: %v", err)
	}
	if re.Code != CodeHomeRefused {
		t.Errorf("Code = %q, want %q", re.Code, CodeHomeRefused)
	}
}

func TestResolve_WalkUpDepthLimit(t *testing.T) {
	t.Parallel()
	// Build 12 nested dirs; marker at the top — beyond the 8-level limit.
	root := mkTree(t, ".archcore/")
	deep := root
	for range 12 {
		deep = filepath.Join(deep, "d")
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(deep),
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatalf("expected error when project is beyond depth limit, got nil")
	}
	if !errors.Is(err, ErrNoProject) {
		t.Errorf("err is not ErrNoProject: %v", err)
	}
}

func TestResolve_NonexistentFlag(t *testing.T) {
	t.Parallel()
	_, err := Resolve(Options{
		Flag:   "/no/such/path",
		Mode:   ModeRuntime,
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatal("expected error for nonexistent flag, got nil")
	}
	if !errors.Is(err, ErrBaseDirNotExist) {
		t.Errorf("err is not ErrBaseDirNotExist: %v", err)
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		t.Fatalf("err is not *ResolveError: %v", err)
	}
	if re.Code != CodePathInvalid {
		t.Errorf("Code = %q, want %q", re.Code, CodePathInvalid)
	}
}

func TestResolve_TildeExpand(t *testing.T) {
	t.Parallel()
	home := mkTree(t, "myproj/.git/")
	flagVal := "~/myproj"

	res, err := Resolve(Options{
		Flag:   flagVal,
		Mode:   ModeRuntime,
		Home:   homeFn(home),
		Getwd:  cwdFn(home),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "myproj")
	if res.Path != want {
		t.Errorf("Path = %q, want %q", res.Path, want)
	}
}

func TestResolve_RelativeFlag(t *testing.T) {
	t.Parallel()
	root := mkTree(t, "myproj/.git/")
	res, err := Resolve(Options{
		Flag:   "./myproj",
		Mode:   ModeRuntime,
		Home:   neutralHome(t),
		Getwd:  cwdFn(root),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "myproj")
	if res.Path != want {
		t.Errorf("Path = %q, want %q", res.Path, want)
	}
}

func TestResolve_ModeInit_AllowsEmpty(t *testing.T) {
	t.Parallel()
	// Empty dir, no markers — ModeInit should accept it via the cwd fallback.
	empty := t.TempDir()
	res, err := Resolve(Options{
		Mode:   ModeInit,
		Getwd:  cwdFn(empty),
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceInitCwd {
		t.Errorf("Source = %q, want %q", res.Source, SourceInitCwd)
	}
	if res.Path != filepath.Clean(empty) {
		t.Errorf("Path = %q, want %q", res.Path, empty)
	}
}

func TestResolve_ModeRuntime_RejectsEmpty(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	_, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  cwdFn(empty),
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatalf("expected error for empty cwd in runtime mode, got nil")
	}
	if !errors.Is(err, ErrNoProject) {
		t.Errorf("err is not ErrNoProject: %v", err)
	}
}

func TestResolve_ModeHooks_LenientWithExplicit(t *testing.T) {
	t.Parallel()
	// Empty dir, but caller passed it via Flag (host-explicit signal).
	// Hooks mode should accept it.
	empty := t.TempDir()
	res, err := Resolve(Options{
		Flag:   empty,
		Mode:   ModeHooks,
		Home:   neutralHome(t),
		Getwd:  cwdFn(empty),
		Getenv: envFn(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != SourceFlag {
		t.Errorf("Source = %q, want %q", res.Source, SourceFlag)
	}
}

func TestResolve_LegacyEscape(t *testing.T) {
	t.Parallel()
	// HOME match would normally be rejected; legacy bypasses it.
	home := mkTree(t)

	res, err := Resolve(Options{
		Flag:   home,
		Mode:   ModeRuntime,
		Home:   homeFn(home),
		Getwd:  cwdFn(home),
		Getenv: envFn(map[string]string{envLegacy: "1"}),
	})
	if err != nil {
		t.Fatalf("unexpected error in legacy mode: %v", err)
	}
	if !res.LegacyMode {
		t.Errorf("LegacyMode = false, want true")
	}
}

func TestResolve_ContextCache(t *testing.T) {
	t.Parallel()
	want := &Resolution{Path: "/p", Source: SourceFlag}
	ctx := WithResolution(context.Background(), want)
	got, ok := From(ctx)
	if !ok || got != want {
		t.Errorf("From() = (%v, %v), want (%v, true)", got, ok, want)
	}
	_, ok = From(context.Background())
	if ok {
		t.Errorf("From() on empty ctx should be false")
	}
}

func TestResolve_RelativeFlagWithGetwdError(t *testing.T) {
	t.Parallel()
	failGetwd := func() (string, error) { return "", os.ErrPermission }
	_, err := Resolve(Options{
		Flag:   "./relative",
		Mode:   ModeRuntime,
		Getwd:  failGetwd,
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatal("expected error for relative flag with failing Getwd, got nil")
	}
	if !errors.Is(err, ErrNoProject) {
		t.Errorf("err is not ErrNoProject: %v", err)
	}
	if errors.Is(err, ErrBaseDirNotExist) {
		t.Errorf("err must not be ErrBaseDirNotExist (false positive): %v", err)
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		t.Fatalf("err is not *ResolveError: %v", err)
	}
	if re.Code != CodeNoProject {
		t.Errorf("Code = %q, want %q", re.Code, CodeNoProject)
	}
}

func TestResolve_GetwdError(t *testing.T) {
	t.Parallel()
	failGetwd := func() (string, error) { return "", os.ErrPermission }
	_, err := Resolve(Options{
		Mode:   ModeRuntime,
		Getwd:  failGetwd,
		Home:   neutralHome(t),
		Getenv: envFn(nil),
	})
	if err == nil {
		t.Fatal("expected error from failing Getwd, got nil")
	}
	if !errors.Is(err, ErrNoProject) {
		t.Errorf("err is not ErrNoProject: %v", err)
	}
}
