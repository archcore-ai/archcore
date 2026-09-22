package cmd

// Tests for the doctor cached-update advisory: one line when the shared
// freshness cache already holds a newer release, silence otherwise, and no
// effect on what doctor reports about the project — unattended-update.spec.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/config"
	"archcore-cli/internal/update"
)

// staleAge pushes a cache mtime past the resolved-answer window. A literal,
// because 24 h is the contract a user lives with; the guard in the test fails if
// the constant ever grows past it instead of letting the stale case quietly read
// as fresh.
const staleAge = 25 * time.Hour

func TestReportCachedUpdate_PrintsOnlyForAFreshNewerRelease(t *testing.T) {
	// The function is handed its cache path, but an env slip anywhere under it
	// must not reach the developer's real state directory.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	if staleAge <= update.CacheTTL {
		t.Fatalf("staleAge %s no longer exceeds update.CacheTTL %s — the stale case would read as fresh", staleAge, update.CacheTTL)
	}

	cases := []struct {
		name    string
		current string
		cached  string        // cache file content
		absent  bool          // no cache file at all
		age     time.Duration // how far back the mtime is pushed
		want    string        // the version the line must name; "" means silence
	}{
		{name: "fresh newer release", current: "v0.5.7", cached: "v9.9.9\n", want: "v9.9.9"},
		{name: "cached version equals current", current: "v0.5.7", cached: "v0.5.7\n"},
		{name: "cached version older than current", current: "v0.5.7", cached: "v0.4.0\n"},
		{name: "newer release past the TTL", current: "v0.5.7", cached: "v9.9.9\n", age: staleAge},
		{name: "no cache file", current: "v0.5.7", absent: true},
		// Empty content inside the window is the negative failure stamp, not an
		// answer: nobody resolved a version, so there is nothing to advertise.
		{name: "fresh failure stamp", current: "v0.5.7", cached: "\n"},
		{name: "unparseable cached tag", current: "v0.5.7", cached: "nightly-2026-08-15\n"},
		// A locally built binary reports "dev". NeedsUpdate would call it behind
		// every release and nag on every doctor run.
		{name: "unparseable current version", current: "dev", cached: "v9.9.9\n"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "last-update-check")
			if !tt.absent {
				if err := os.WriteFile(path, []byte(tt.cached), 0o644); err != nil {
					t.Fatal(err)
				}
				if tt.age > 0 {
					old := time.Now().Add(-tt.age)
					if err := os.Chtimes(path, old, old); err != nil {
						t.Fatal(err)
					}
				}
			}

			var out bytes.Buffer
			reportCachedUpdate(&out, tt.current, path)

			got := out.String()
			if tt.want == "" {
				if got != "" {
					t.Fatalf("advisory must stay silent, got %q", got)
				}
				return
			}
			if n := strings.Count(got, "\n"); n != 1 {
				t.Errorf("advisory must be exactly one line, got %d:\n%s", n, got)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("advisory %q does not name the available version %s", got, tt.want)
			}
		})
	}
}

// A cache that cannot be resolved at all — internal/update returns "" for the
// path when no state directory exists — must not stat the working directory or
// print anything.
func TestReportCachedUpdate_SilentWithoutACachePath(t *testing.T) {
	var out bytes.Buffer
	reportCachedUpdate(&out, "v0.5.7", "")
	if out.Len() != 0 {
		t.Errorf("no cache path must mean no advisory, got %q", out.String())
	}
}

// The advisory is not a fault. A project whose only finding is "a newer archcore
// exists" must still pass, or `doctor` in CI starts failing the day a release
// lands.
func TestDoctor_CachedUpdateAdvisoryKeepsAHealthyProjectPassing(t *testing.T) {
	dir := newDoctorProject(t)
	seedFreshCache(t, "v9.9.9")

	out, err := runDoctorVersion(t, dir, "v0.5.7")
	if err != nil {
		t.Fatalf("advisory must not fail a healthy project: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "v9.9.9") {
		t.Fatalf("expected the cached-update advisory in output, got: %s", out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected 'All checks passed', got: %s", out)
	}
}

// The counter the summary reports must read the same with and without the
// advisory: an advisory that incremented it would turn one available release
// into an extra "issue" on every project.
func TestDoctor_CachedUpdateAdvisoryLeavesTheIssueCountAlone(t *testing.T) {
	dir := newDoctorProject(t)
	// A document with no type segment — one real issue for the counter to hold.
	writeDoc(t, dir, "", "readme.md", validFrontmatter)

	withoutOut, withoutErr := runDoctorVersion(t, dir, "v0.5.7")
	seedFreshCache(t, "v9.9.9")
	withOut, withErr := runDoctorVersion(t, dir, "v0.5.7")

	if !strings.Contains(withOut, "v9.9.9") {
		t.Fatalf("the second run must carry the advisory, got: %s", withOut)
	}
	if strings.Contains(withoutOut, "v9.9.9") {
		t.Fatalf("the first run has no cache and must carry no advisory, got: %s", withoutOut)
	}
	if got, want := errText(withErr), errText(withoutErr); got != want {
		t.Errorf("exit with advisory = %q, without = %q", got, want)
	}
	if got, want := doctorSummary(t, withOut), doctorSummary(t, withoutOut); got != want {
		t.Errorf("summary with advisory = %q, without = %q", got, want)
	}
}

// newDoctorProject returns an initialized project that passes every check, with
// the process state directory pointed at a temp dir so no test reads or writes
// the developer's real update cache.
func newDoctorProject(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, config.NewNoneSettings()); err != nil {
		t.Fatal(err)
	}
	return dir
}

// seedFreshCache writes latest into the canonical cache path — the same file the
// update path writes, which is the whole point of the advisory reading it.
func seedFreshCache(t *testing.T, latest string) {
	t.Helper()
	path := update.CachePath()
	if path == "" {
		t.Fatal("no cache path resolved — XDG_STATE_HOME is not set for this test")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(latest+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runDoctorVersion runs the doctor command against dir at a given version.
// Built directly rather than through the root command, whose test version is
// unparseable and would make every advisory case silent.
func runDoctorVersion(t *testing.T, dir, version string) (string, error) {
	t.Helper()
	cmd := newDoctorCmd(version)
	// doctor prints through os.Stdout, so the cobra writer must stay empty.
	// Wiring it to a buffer and then discarding the buffer would hide any line
	// that started going through cmd.OutOrStdout(): the assertion below is what
	// makes the capture the whole output rather than most of it.
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--project", dir})
	// The root command sets both; a standalone command would otherwise render
	// the returned error and the usage block into the buffer, which is cobra's
	// output rather than doctor's and would mask the check below.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = cmd.Execute() })
	if buf.Len() != 0 {
		t.Errorf("doctor wrote %q to the cobra writer; this test reads os.Stdout", buf.String())
	}
	return out, execErr
}

// doctorSummary returns the closing line, the one place the issue counter is
// visible to a reader.
func doctorSummary(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "issue(s) found") || strings.Contains(line, "All checks passed") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatalf("no doctor summary line in output:\n%s", out)
	return ""
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
