package update

// The freshness cache: the one place that answers "what was the latest release
// the last time anyone looked?", so a hook-driven `archcore update --check`, the
// doctor advisory and the unattended policy share one answer and one network
// budget instead of each keeping a copy — unattended-update.spec.
//
// The cache holds two kinds of entry in one file. A version string is a real
// answer. Empty content is a failure stamp: a recent lookup failed, and until
// the shorter window lapses no caller pays the probe timeout again. The stamp
// is why the file's own mtime, not the content, decides freshness.

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"archcore-cli/internal/jsonfile"
	"archcore-cli/internal/xdg"
)

const (
	// CacheTTL is how long a resolved version stays fresh.
	//
	// The unattended policy's claim window is this same constant, which is what
	// unattended-update.spec requires: one constant governs both, and the
	// resulting worst case is ~48 h from release to replacement.
	CacheTTL = 24 * time.Hour

	// CacheFailureTTL is the negative-cache window. It is shorter than CacheTTL
	// because a failure stamp records nothing about the release, only that the
	// network was unreachable a moment ago: it protects the hook budget without
	// letting one blip hide a release for a day.
	CacheFailureTTL = time.Hour

	// cacheFileName is the entry inside the shared state directory.
	cacheFileName = "last-update-check"

	// maxCacheBytes bounds what a read admits into memory. The content is one
	// version tag, so a file past this ceiling is not something this package
	// wrote; a hook that runs on every session start must not be able to load a
	// grown or corrupted file instead — bounded-and-deterministic-output.rule.
	maxCacheBytes = 4 << 10
)

// CachePath returns the freshness-cache file. It returns an empty string when
// no state directory resolves; every function here treats that as "no cache"
// and degrades to a network lookup rather than failing a command.
func CachePath() string {
	dir := xdg.StateDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, cacheFileName)
}

// ReadCachedLatest returns the cached version and whether the cache is fresh.
// Empty content is a failure stamp with its own shorter window: latest == ""
// with fresh == true means "a recent lookup failed, skip the network".
//
// Freshness comes from the mtime rather than from a timestamp inside the file,
// so a failure stamp needs no content at all and a partially written file can
// never read as a fresher one.
func ReadCachedLatest(path string) (latest string, fresh bool) {
	if path == "" {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxCacheBytes {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	latest = strings.TrimSpace(string(data))
	window := CacheTTL
	if latest == "" {
		window = CacheFailureTTL
	}
	return latest, time.Since(info.ModTime()) < window
}

// WriteCachedLatest stores latest, best-effort: a cache that cannot be written
// costs a network lookup next time and nothing else, so no caller is told.
// An empty latest writes the failure stamp.
//
// The write goes through jsonfile.WriteAtomic, whose temp name is per-attempt:
// the unattended policy refreshes this file from a background goroutine while
// separate `archcore update --check` processes rewrite it, and a shared temp
// name across processes would let one truncate another's and publish a torn
// file — choosing-an-atomic-write.rule.
func WriteCachedLatest(path, latest string) {
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	_ = jsonfile.WriteAtomic(path, []byte(latest+"\n"))
}
