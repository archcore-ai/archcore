// Package stamp records "has this already happened recently?" for the hook
// surfaces.
//
// Several hook surfaces need "has this already happened recently?" with wildly
// different windows: SessionStart dedupes a double-registered hook within
// seconds, a tool-level event dedupes within one tool call, and the staleness
// advisory rate-limits itself to once a day. They share the atomic-claim
// mechanism and differ only in window and namespace.
//
// Each scope owns its own directory. That is not tidiness — a sweep deletes
// every stamp older than its window, so a 10-minute scope sharing a directory
// with a 24-hour scope would silently erase the day-long rate limit on its
// first run.
package stamp

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// DirFor returns the state directory for one dedup scope, honoring
// XDG_STATE_HOME. An empty result disables dedup for that scope (fail open).
func DirFor(scope string) string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "archcore", scope)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "archcore", scope)
}

// PathFor maps an arbitrary dedup key to a filesystem-safe path.
func PathFor(stampDir, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(stampDir, fmt.Sprintf("%x", sum[:16]))
}

// Fresh reports whether a stamp for key exists within window.
func Fresh(stampDir, key string, window time.Duration) bool {
	info, err := os.Stat(PathFor(stampDir, key))
	return err == nil && time.Since(info.ModTime()) < window
}

// stale reports whether the file at path exists and is older than window. A
// missing file is not stale: there is nothing to reclaim.
func stale(path string, window time.Duration) bool {
	info, err := os.Stat(path)
	return err == nil && time.Since(info.ModTime()) >= window
}

// lockSuffix names the sibling file that serializes a stale-stamp reclaim. It
// is created in the same directory, so sweep clears an abandoned one on the
// same schedule as the stamps themselves.
const lockSuffix = ".lock"

// Claim atomically claims the stamp for key: exactly one of any number of
// concurrent callers wins (O_CREATE|O_EXCL), which is what makes the dedup hold
// when two hook entries fire in parallel for the same event. Best-effort by
// design: any filesystem failure other than "fresh stamp exists" claims (fails
// open) — dedup must never break the hook. The winner also sweeps its scope.
func Claim(stampDir, key string, window time.Duration) bool {
	if os.MkdirAll(stampDir, 0o755) != nil {
		return true // fail open
	}
	path := PathFor(stampDir, key)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		_ = f.Close()
		sweep(stampDir, window)
		return true
	}
	if !errors.Is(err, fs.ErrExist) {
		return true // fail open
	}
	if Fresh(stampDir, key, window) {
		return false // fresh stamp — a peer already emitted
	}
	return reclaim(stampDir, path, window)
}

// reclaim replaces a stale stamp with a fresh one and lets exactly one caller
// through.
//
// Remove-then-retry is not enough on its own. Two callers that both find the
// stamp stale can both remove it and both create it: the event is emitted twice,
// and the second removal deletes a stamp the first caller had already published,
// so a third caller inside the window finds nothing and emits again. The check
// and the replacement have to be one step, and no portable filesystem call does
// "replace this file only if it is still the one I looked at".
//
// A sibling lock file created with O_EXCL serializes the whole sequence instead,
// and the reclaim refreshes the stamp's timestamp rather than replacing the
// file. Removing it first would not help: the gap between the remove and the
// create is exactly where another caller's plain O_EXCL create — the fast path
// above, which holds no lock — succeeds and hands out a second claim.
//
// A lock left behind by a killed process is itself older than window and gets
// cleared — by the next caller here, and by sweep.
func reclaim(stampDir, path string, window time.Duration) bool {
	lock := path + lockSuffix
	lf, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		if !stale(lock, window) {
			return false // a peer is reclaiming right now, and it will emit
		}
		_ = os.Remove(lock) // abandoned by a process that died mid-reclaim
		lf, err = os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return false // a peer took it first
		}
	}
	if err != nil {
		return true // fail open: dedup must never break the hook
	}
	defer func() {
		_ = lf.Close()
		_ = os.Remove(lock)
	}()

	// Under the lock. A peer may have reclaimed between our staleness check and
	// this point, in which case the stamp is fresh (or already swept away by
	// that peer) and there is nothing left to take.
	if !stale(path, window) {
		return false
	}
	now := time.Now()
	if os.Chtimes(path, now, now) != nil {
		return true // fail open
	}
	sweep(stampDir, window)
	return true
}

// sweep removes stamps older than window, best-effort. It only ever
// touches stampDir, so one scope can never expire another scope's stamps.
func sweep(stampDir string, window time.Duration) {
	entries, err := os.ReadDir(stampDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && time.Since(info.ModTime()) >= window {
			_ = os.Remove(filepath.Join(stampDir, e.Name()))
		}
	}
}
