package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveProjectRoot covers the parallel-safe scenarios of
// resolveProjectRoot. The two cwd-dependent cases (fallback to cwd, relative
// path resolution) need t.Chdir and therefore cannot run in parallel; they
// live in their own standalone tests below.
func TestResolveProjectRoot(t *testing.T) {
	t.Parallel()

	type tcResult struct {
		flagValue       string
		envValue        string
		wantBase        string // expected absolute path; empty + !wantErr means assert IsAbs only
		wantErr         bool
		errContains     string
		errContainsPath string // assert this path appears in the error message
	}

	tests := []struct {
		name  string
		setup func(t *testing.T) tcResult
	}{
		{
			name: "flag overrides env",
			setup: func(t *testing.T) tcResult {
				t.Helper()
				flagDir := t.TempDir()
				envDir := t.TempDir()
				wantAbs, err := filepath.Abs(flagDir)
				if err != nil {
					t.Fatalf("filepath.Abs(%q): %v", flagDir, err)
				}
				return tcResult{flagValue: flagDir, envValue: envDir, wantBase: wantAbs}
			},
		},
		{
			name: "env used when flag empty",
			setup: func(t *testing.T) tcResult {
				t.Helper()
				envDir := t.TempDir()
				wantAbs, err := filepath.Abs(envDir)
				if err != nil {
					t.Fatalf("filepath.Abs(%q): %v", envDir, err)
				}
				return tcResult{flagValue: "", envValue: envDir, wantBase: wantAbs}
			},
		},
		{
			name: "non-existent path returns descriptive error",
			setup: func(t *testing.T) tcResult {
				t.Helper()
				missing := filepath.Join(t.TempDir(), "does-not-exist")
				wantAbs, err := filepath.Abs(missing)
				if err != nil {
					t.Fatalf("filepath.Abs(%q): %v", missing, err)
				}
				return tcResult{
					flagValue:       missing,
					wantErr:         true,
					errContains:     "does not exist",
					errContainsPath: wantAbs,
				}
			},
		},
		{
			name: "path is regular file returns 'not a directory' error",
			setup: func(t *testing.T) tcResult {
				t.Helper()
				file := filepath.Join(t.TempDir(), "regular-file")
				if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
					t.Fatalf("os.WriteFile: %v", err)
				}
				return tcResult{
					flagValue:   file,
					wantErr:     true,
					errContains: "is not a directory",
				}
			},
		},
		{
			name: "both inputs empty falls back to abs cwd",
			setup: func(t *testing.T) tcResult {
				t.Helper()
				// wantBase intentionally empty — assert filepath.IsAbs only,
				// since we cannot safely t.Chdir in a parallel test.
				return tcResult{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.setup(t)

			got, err := resolveProjectRoot(c.flagValue, c.envValue)

			if (err != nil) != c.wantErr {
				t.Fatalf("resolveProjectRoot(%q, %q) error = %v, wantErr %v",
					c.flagValue, c.envValue, err, c.wantErr)
			}

			if c.wantErr {
				if c.errContains != "" && !strings.Contains(err.Error(), c.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), c.errContains)
				}
				if c.errContainsPath != "" && !strings.Contains(err.Error(), c.errContainsPath) {
					t.Errorf("error %q should contain path %q", err.Error(), c.errContainsPath)
				}
				return
			}

			if c.wantBase != "" {
				if got != c.wantBase {
					t.Errorf("resolveProjectRoot(%q, %q) = %q, want %q",
						c.flagValue, c.envValue, got, c.wantBase)
				}
				return
			}

			if !filepath.IsAbs(got) {
				t.Errorf("expected absolute path, got %q", got)
			}
		})
	}
}

// TestResolveProjectRoot_FallsBackToCwd verifies the cwd fallback. Cannot run
// in parallel because it calls t.Chdir.
func TestResolveProjectRoot_FallsBackToCwd(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	got, err := resolveProjectRoot("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compare against os.Getwd() after Chdir to handle macOS /var → /private/var
	// canonicalization — the resolver and the assertion must derive their paths
	// the same way.
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q (should fall back to cwd)", got, want)
	}
}

// TestResolveProjectRoot_RelativePath verifies that a relative flag value is
// resolved against the current working directory. Cannot run in parallel
// because it calls t.Chdir.
func TestResolveProjectRoot_RelativePath(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q): %v", child, err)
	}
	t.Chdir(parent)

	got, err := resolveProjectRoot("child", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "child" {
		t.Errorf("got base %q, want %q", filepath.Base(got), "child")
	}
	// Resolver runs filepath.Abs against current cwd, so the resolved path
	// shares the same canonical prefix as os.Getwd().
	parentCanonical, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	wantChild := filepath.Join(parentCanonical, "child")
	if got != wantChild {
		t.Errorf("got %q, want %q", got, wantChild)
	}
}
