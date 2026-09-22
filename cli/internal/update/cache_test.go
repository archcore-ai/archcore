package update

// The freshness cache is read on a hook path and written from a background
// goroutine, so what is pinned here is the degradation: a cache that cannot be
// read or written must cost a network lookup and never an error, and the two
// windows must stay distinguishable — a failure stamp is not a version.

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// ageFile backdates a cache file so a window can be crossed without waiting.
func ageFile(t *testing.T, path string, age time.Duration) {
	t.Helper()
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

func TestCachePath(t *testing.T) {
	t.Run("lives in the shared state directory", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", root)

		want := filepath.Join(root, "archcore", "last-update-check")
		if got := CachePath(); got != want {
			t.Errorf("CachePath() = %q, want %q", got, want)
		}
	})

	t.Run("empty when no state directory resolves", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("os.UserHomeDir does not read HOME on windows")
		}
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", "")

		if got := CachePath(); got != "" {
			t.Errorf("CachePath() = %q, want an empty string", got)
		}
	})

	t.Run("resolving creates nothing", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", root)

		_ = CachePath()

		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("resolving the path created %d entr(ies)", len(entries))
		}
	})
}

// The two windows are written out as literals here, not derived from the
// constants that define them, because every other test in this file measures
// against `CacheTTL ± something` and would stay green at any value at all.
//
// Neither number is a tuning knob. 24 h is also the unattended policy's claim
// window — one constant governs both, and the ~48 h worst case from release to
// replacement is computed from it — so shortening it here silently shortens the
// exclusivity window that keeps concurrent processes from all replacing the
// binary at once. 1 h is the negative-cache window, and it must stay the
// shorter of the two so one network blip cannot hide a release for a day —
// unattended-update.spec.
func TestCacheWindows_AreTheValuesThePolicyDependsOn(t *testing.T) {
	t.Parallel()

	if CacheTTL != 24*time.Hour {
		t.Errorf("CacheTTL = %s, want 24h — it is also the claim window", CacheTTL)
	}
	if CacheFailureTTL != time.Hour {
		t.Errorf("CacheFailureTTL = %s, want 1h", CacheFailureTTL)
	}
	if CacheFailureTTL >= CacheTTL {
		t.Errorf("CacheFailureTTL (%s) must stay shorter than CacheTTL (%s)", CacheFailureTTL, CacheTTL)
	}
}

func TestReadCachedLatest(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		age        time.Duration
		wantLatest string
		wantFresh  bool
	}{
		{
			name:       "a value inside the window is fresh",
			content:    "v9.9.9\n",
			age:        CacheTTL - time.Hour,
			wantLatest: "v9.9.9",
			wantFresh:  true,
		},
		{
			name:       "a value past the window is stale",
			content:    "v9.9.9\n",
			age:        CacheTTL + time.Minute,
			wantLatest: "v9.9.9",
			wantFresh:  false,
		},
		{
			// Empty content is the failure stamp, and it expires on its own
			// shorter window — the whole reason the negative cache does not
			// hide a release for a day.
			name:       "a failure stamp inside the short window is fresh",
			content:    "\n",
			age:        CacheFailureTTL - time.Minute,
			wantLatest: "",
			wantFresh:  true,
		},
		{
			name:       "a failure stamp past the short window is stale",
			content:    "\n",
			age:        CacheFailureTTL + time.Minute,
			wantLatest: "",
			wantFresh:  false,
		},
		{
			// The window a stamp gets must be the short one even when the long
			// one would still cover it; otherwise one blip suppresses the probe
			// for 24 h.
			name:       "a failure stamp is not held for the value window",
			content:    "",
			age:        CacheFailureTTL + time.Minute,
			wantLatest: "",
			wantFresh:  false,
		},
		{
			name:       "surrounding whitespace is not part of the version",
			content:    "  v1.2.3  \n\n",
			age:        time.Minute,
			wantLatest: "v1.2.3",
			wantFresh:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "last-update-check")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			ageFile(t, path, tt.age)

			latest, fresh := ReadCachedLatest(path)
			if latest != tt.wantLatest || fresh != tt.wantFresh {
				t.Errorf("ReadCachedLatest() = (%q, %v), want (%q, %v)",
					latest, fresh, tt.wantLatest, tt.wantFresh)
			}
		})
	}
}

// A file written this second with nothing in it is the case the mtime alone
// cannot tell apart from a fresh version: it must read as a failure stamp, not
// as an empty version this process would then treat as an answer.
func TestReadCachedLatest_FreshEmptyContentIsAFailureStamp(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "last-update-check")
	WriteCachedLatest(path, "")

	latest, fresh := ReadCachedLatest(path)
	if latest != "" {
		t.Errorf("latest = %q, want an empty string", latest)
	}
	if !fresh {
		t.Error("a just-written failure stamp must read as fresh")
	}

	// And it must be the short window that governs it.
	ageFile(t, path, CacheFailureTTL+time.Minute)
	if _, fresh := ReadCachedLatest(path); fresh {
		t.Error("a failure stamp past the short window must read as stale")
	}
}

func TestReadCachedLatest_UnusableCacheIsNotFresh(t *testing.T) {
	t.Run("no path", func(t *testing.T) {
		t.Parallel()
		if latest, fresh := ReadCachedLatest(""); latest != "" || fresh {
			t.Errorf("ReadCachedLatest(\"\") = (%q, %v), want (\"\", false)", latest, fresh)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "absent")
		if latest, fresh := ReadCachedLatest(path); latest != "" || fresh {
			t.Errorf("a missing cache = (%q, %v), want (\"\", false)", latest, fresh)
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" {
			t.Skip("mode bits do not deny a read on windows")
		}
		if os.Geteuid() == 0 {
			t.Skip("root reads a 0o000 file")
		}
		path := filepath.Join(t.TempDir(), "last-update-check")
		if err := os.WriteFile(path, []byte("v9.9.9\n"), 0o000); err != nil {
			t.Fatal(err)
		}
		if latest, fresh := ReadCachedLatest(path); latest != "" || fresh {
			t.Errorf("an unreadable cache = (%q, %v), want (\"\", false)", latest, fresh)
		}
	})

	t.Run("a directory in place of the file", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "last-update-check")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if latest, fresh := ReadCachedLatest(path); latest != "" || fresh {
			t.Errorf("a directory cache = (%q, %v), want (\"\", false)", latest, fresh)
		}
	})

	// Both sides of the ceiling, because only one of them pins which comparison
	// the guard uses. A test that grows the file past the cap alone passes with
	// `>` and with `>=`, and the second reading refuses a cache that fits the
	// ceiling exactly — costing a network lookup on every run of a machine whose
	// cache happened to land on the boundary.
	t.Run("content at and past the ceiling", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name      string
			size      int
			wantFresh bool
		}{
			{name: "exactly the ceiling", size: maxCacheBytes, wantFresh: true},
			{name: "one byte past it", size: maxCacheBytes + 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				path := filepath.Join(t.TempDir(), "last-update-check")
				// A leading "v" keeps the content version-shaped; the rest is
				// padding, and neither side reads past the size check anyway.
				grown := bytes.Repeat([]byte("v"), tt.size)
				if err := os.WriteFile(path, grown, 0o644); err != nil {
					t.Fatal(err)
				}
				latest, fresh := ReadCachedLatest(path)
				if fresh != tt.wantFresh {
					t.Errorf("ReadCachedLatest() fresh = %v, want %v for %d bytes against the %d-byte ceiling",
						fresh, tt.wantFresh, tt.size, maxCacheBytes)
				}
				if !tt.wantFresh && latest != "" {
					t.Errorf("an oversized cache returned %q, want it discarded", latest)
				}
			})
		}
	})
}

func TestWriteCachedLatest(t *testing.T) {
	t.Run("round trips a version", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "last-update-check")

		WriteCachedLatest(path, "v9.9.9")

		latest, fresh := ReadCachedLatest(path)
		if latest != "v9.9.9" || !fresh {
			t.Errorf("read back (%q, %v), want (\"v9.9.9\", true)", latest, fresh)
		}
	})

	t.Run("creates the state directory", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "archcore", "last-update-check")

		WriteCachedLatest(path, "v9.9.9")

		if latest, _ := ReadCachedLatest(path); latest != "v9.9.9" {
			t.Errorf("latest = %q, want v9.9.9 written under a created directory", latest)
		}
	})

	t.Run("leaves no temporary file behind", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "last-update-check")

		WriteCachedLatest(path, "v9.9.9")
		WriteCachedLatest(path, "v9.9.10")

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "last-update-check" {
			t.Errorf("state directory holds %v, want only the cache file", entries)
		}
	})

	// "Silent" is two claims: the call does not fail the process, and it leaves
	// nothing behind. Only the first is visible without assertions, and a write
	// that created a stray directory or replaced the blocking file would have
	// passed a test that merely called it.
	t.Run("an unwritable target is silent", func(t *testing.T) {
		t.Parallel()
		// No path at all, and a path whose parent cannot be created: both are
		// best-effort no-ops, because a cache miss costs a lookup and a failed
		// command costs the user their update.
		WriteCachedLatest("", "v9.9.9")

		dir := t.TempDir()
		blocked := filepath.Join(dir, "file")
		const content = "not a directory"
		if err := os.WriteFile(blocked, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		WriteCachedLatest(filepath.Join(blocked, "archcore", "last-update-check"), "v9.9.9")

		got, err := os.ReadFile(blocked)
		if err != nil {
			t.Fatalf("the blocking file is gone: %v", err)
		}
		if string(got) != content {
			t.Errorf("the blocking file now holds %q, want it untouched at %q", got, content)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "file" {
			t.Errorf("the directory holds %v, want only the blocking file", entries)
		}
	})
}
