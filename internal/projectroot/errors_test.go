package projectroot

import (
	"errors"
	"testing"
)

func TestResolveError_UnwrapsToSentinel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
	}{
		{"NoProject", ErrNoProject},
		{"Home", ErrBaseDirHome},
		{"NotExist", ErrBaseDirNotExist},
		{"System", ErrBaseDirSystem},
		{"NoMarkers", ErrBaseDirNoMarkers},
		{"TildeExpand", ErrTildeExpand},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			re := &ResolveError{Sentinel: c.err, Code: "TEST"}
			if !errors.Is(re, c.err) {
				t.Errorf("errors.Is(re, %v) = false", c.err)
			}
		})
	}
}

func TestResolveError_HasStableCode(t *testing.T) {
	t.Parallel()
	codes := []string{
		CodeNotProject,
		CodeHomeRefused,
		CodeNoProject,
		CodePathInvalid,
		CodeTildeExpand,
	}
	wants := []string{
		"ERR_NOT_PROJECT",
		"ERR_HOME_REFUSED",
		"ERR_NO_PROJECT",
		"ERR_PATH_INVALID",
		"ERR_TILDE_EXPAND",
	}
	for i, code := range codes {
		if code != wants[i] {
			t.Errorf("code[%d] = %q, want %q", i, code, wants[i])
		}
	}
}

func TestFormatError_Goldens(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  *ResolveError
		want string
	}{
		{
			name: "not_project",
			err: &ResolveError{
				Sentinel: ErrBaseDirNoMarkers,
				Code:     CodeNotProject,
				Path:     "/Users/x/empty",
				Source:   SourceFlag,
				Reason:   "no project markers found in this directory",
				Fix:      "expected one of: .archcore, .git, go.mod, package.json, pyproject.toml, Cargo.toml, pom.xml, composer.json, Gemfile",
			},
			want: `--- archcore error ---
code    : ERR_NOT_PROJECT
message : base dir has no project markers
path    : /Users/x/empty
source  : flag
detail  : no project markers found in this directory
hint    : expected one of: .archcore, .git, go.mod, package.json, pyproject.toml, Cargo.toml, pom.xml, composer.json, Gemfile
---------------------
`,
		},
		{
			name: "home_refused",
			err: &ResolveError{
				Sentinel: ErrBaseDirHome,
				Code:     CodeHomeRefused,
				Path:     "/Users/x",
				Source:   SourceCwd,
				Reason:   "directory matches $HOME; refusing under strict mode",
				Fix:      "cd into a project directory; or set ARCHCORE_LEGACY_BASE_DIR=1 to bypass (not recommended)",
			},
			want: `--- archcore error ---
code    : ERR_HOME_REFUSED
message : base dir is $HOME
path    : /Users/x
source  : cwd
detail  : directory matches $HOME; refusing under strict mode
hint    : cd into a project directory; or set ARCHCORE_LEGACY_BASE_DIR=1 to bypass (not recommended)
---------------------
`,
		},
		{
			name: "no_project",
			err: &ResolveError{
				Sentinel: ErrNoProject,
				Code:     CodeNoProject,
				Path:     "/Users/x/random",
				Source:   SourceCwd,
				Reason:   "no project markers found from /Users/x/random (walked up 8 levels)",
				Fix:      "cd into a project directory, or pass --base-dir <path>, or set ARCHCORE_BASE_DIR",
			},
			want: `--- archcore error ---
code    : ERR_NO_PROJECT
message : no project root found
path    : /Users/x/random
source  : cwd
detail  : no project markers found from /Users/x/random (walked up 8 levels)
hint    : cd into a project directory, or pass --base-dir <path>, or set ARCHCORE_BASE_DIR
---------------------
`,
		},
		{
			name: "path_invalid",
			err: &ResolveError{
				Sentinel: ErrBaseDirNotExist,
				Code:     CodePathInvalid,
				Path:     "/no/such/path",
				Source:   SourceFlag,
				Reason:   "stat: open /no/such/path: no such file or directory",
				Fix:      "create the directory or fix the path",
			},
			want: `--- archcore error ---
code    : ERR_PATH_INVALID
message : base dir does not exist
path    : /no/such/path
source  : flag
detail  : stat: open /no/such/path: no such file or directory
hint    : create the directory or fix the path
---------------------
`,
		},
		{
			name: "tilde_expand",
			err: &ResolveError{
				Sentinel: ErrTildeExpand,
				Code:     CodeTildeExpand,
				Reason:   "$HOME not set",
				Fix:      "use an absolute path or set $HOME",
			},
			want: `--- archcore error ---
code    : ERR_TILDE_EXPAND
message : tilde expansion failed
detail  : $HOME not set
hint    : use an absolute path or set $HOME
---------------------
`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := FormatError(c.err)
			if got != c.want {
				t.Errorf("FormatError mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

func TestFormatError_NilOrUnknown(t *testing.T) {
	t.Parallel()
	if FormatError(nil) != "" {
		t.Errorf("FormatError(nil) should be empty")
	}
	if FormatError(errors.New("plain")) != "" {
		t.Errorf("FormatError(non-ResolveError) should be empty")
	}
}

func TestMarkers_ReturnsCopy(t *testing.T) {
	t.Parallel()
	a := Markers()
	b := Markers()
	if &a[0] == &b[0] {
		t.Errorf("Markers() should return a fresh copy each call")
	}
	a[0] = "MUTATED"
	if Markers()[0] == "MUTATED" {
		t.Errorf("Markers() leaked internal state")
	}
}
