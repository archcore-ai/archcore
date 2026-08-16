// Package stamp records "has this already happened recently?" across every
// process on one machine.
//
// The surfaces that need that answer want wildly different windows: SessionStart
// dedupes a double-registered hook within seconds, a tool-level event dedupes
// within one tool call, the staleness advisory rate-limits itself to once a day,
// and unattended update allows one replacement per binary path per day. They
// share the atomic-claim mechanism and differ only in window and namespace.
//
// Each scope owns its own directory. That is not tidiness — a sweep deletes
// every stamp older than its window, so a 10-minute scope sharing a directory
// with a 24-hour scope would silently erase the day-long rate limit on its
// first run.
//
// The mechanism is shared; the failure bias belongs to the entry point, because
// its two kinds of caller want opposite answers when the filesystem itself
// fails. Claim claims anyway: dedup exists to spare a reader a repeated line,
// and it must never be the reason a hook does not run. ClaimFailClosed refuses
// instead: it decides whether this process may replace the binary, and under the
// fail-open bias an unwritable state directory lets every concurrent process
// replace it at once — unattended-update.spec §6.
package stamp

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"archcore-cli/internal/xdg"
)

// DirFor returns the state directory for one dedup scope, honoring
// XDG_STATE_HOME. An empty result means no state directory could be resolved:
// Claim then claims and ClaimFailClosed refuses, per each entry point's bias.
func DirFor(scope string) string {
	root := xdg.StateDir()
	if root == "" {
		return ""
	}
	return filepath.Join(root, scope)
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
	return claim(stampDir, key, window, true)
}

// ClaimFailClosed claims the stamp for key through the same mechanism and the
// opposite bias: an unresolved (empty) stampDir and every filesystem failure
// refuse. It guards binary replacement, where a claim that cannot be established
// must read as "a peer may hold it" — two callers that both win race the same
// rename, which is the outcome exclusivity exists to forbid —
// unattended-update.spec §6. The bias is in the name so a call site reads
// correctly without opening this package — fail-open-or-fail-closed-reads.rule §4.
func ClaimFailClosed(stampDir, key string, window time.Duration) bool {
	return claim(stampDir, key, window, false)
}

// claim is the mechanism both entry points share. failOpen is the answer at
// every point where the filesystem failed rather than answered — the cases where
// the claim is neither refused by a peer nor taken, and only the caller's bias
// can decide. Passing it in keeps that decision at the entry point instead of
// spread over the error branches, where the two biases had to be kept in step by
// hand.
func claim(stampDir, key string, window time.Duration, failOpen bool) bool {
	if stampDir == "" {
		// No state directory resolved, so no claim is possible. Decided here to
		// keep a fail-closed caller off the filesystem entirely; MkdirAll("")
		// reaches the same answer, but only by way of an error.
		return failOpen
	}
	if os.MkdirAll(stampDir, 0o755) != nil {
		return failOpen
	}
	path := PathFor(stampDir, key)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		_ = f.Close()
		sweep(stampDir, window)
		return true
	}
	if !errors.Is(err, fs.ErrExist) {
		return failOpen
	}
	if Fresh(stampDir, key, window) {
		return false // fresh stamp — a peer already emitted
	}
	return reclaim(stampDir, path, window, failOpen)
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
// A lock left behind by a killed process is itself older than window. The next
// caller here steals it — see stealAbandonedLock — and sweep clears one that no
// caller reaches.
//
// Holding the lock is not the same as still holding it. Recovery renames a lock
// aside, and a caller whose staleness check was overtaken lands that rename on a
// live lock instead, so a holder's lock can be taken out from under it mid-
// reclaim. That is why the last thing checked before the stamp is touched is
// whether this caller is still the holder — see lockHeld.
func reclaim(stampDir, path string, window time.Duration, failOpen bool) bool {
	lock := path + lockSuffix
	lf, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		if !stale(lock, window) {
			return false // a peer is reclaiming right now, and it will emit
		}
		// The lock was abandoned by a process that died mid-reclaim. Taking it
		// has to be one step: a bare remove followed by a create lets two
		// callers interleave and both come away holding it.
		if !stealLock(lock, window) {
			return false // a peer took it first, or is holding a live one
		}
		lf, err = os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return false // a peer took it first
		}
	}
	if err != nil {
		return failOpen // the lock is unusable, so exclusivity is unestablished
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
	if !lockHeld(lf, lock) {
		return false // our lock was taken; a peer may be inside the window
	}
	now := time.Now()
	if os.Chtimes(path, now, now) != nil {
		// The stamp stays stale, so a later caller would claim it again.
		return failOpen
	}
	sweep(stampDir, window)
	return true
}

// lockHeld is the seam reclaim confirms it still holds its lock through. Like
// stealLock it exists because the interleave it guards cannot be scheduled from a
// test: the lock has to be taken away between this caller's exclusive create and
// its next syscall. A stub answering "no" is the only way to reach the branch,
// and without it the check would be indistinguishable from dead code.
var lockHeld = holdsLock

// holdsLock reports whether lock still names the file this caller created.
//
// The recovery path renames an abandoned lock aside, and the staleness check that
// authorizes it is a separate call — so a caller descheduled between the two
// lands its rename on whatever occupies the name by then, which may be a lock a
// peer published in the meantime. That peer keeps working, the name is free, and
// the next caller creates a lock of its own, finds the stamp still stale, and
// refreshes it alongside the peer: two attempts in one window, across concurrent
// processes, which is the invariant unattended update rests on —
// unattended-update.spec.
//
// No rename can be prevented, so the holder is what detects it. The open file is
// the caller's proof of what it created; comparing it against the name says
// whether the name still leads there. A holder that finds it no longer does gives
// up rather than repairing, because it cannot tell whether the caller that took
// its lock is already inside the window, and the answer that adds no second
// winner is the safe one under both biases. Giving up costs only this attempt:
// the stamp is left untouched, so the next caller still finds it stale and
// reclaims it in turn.
func holdsLock(lf *os.File, lock string) bool {
	held, err := lf.Stat()
	if err != nil {
		return false
	}
	named, err := os.Stat(lock)
	return err == nil && os.SameFile(held, named)
}

// stealSeq makes a steal name unique inside this process. The pid alone is not:
// two goroutines can reclaim the same key at once, and the tests do exactly that.
var stealSeq atomic.Uint64

// stealLock is the seam reclaim recovers an abandoned lock through. It exists
// because no test can prove that recovery is atomic from the outside: the
// interleave it forbids needs a process killed inside a window a test cannot
// schedule, and every observable outcome of a bare remove-then-create matches
// the steal's. A counting stub is the only thing that can tell "reclaim takes
// the lock in one step" from "reclaim removes it and hopes".
var stealLock = stealAbandonedLock

// stealAbandonedLock takes an abandoned lock away from every other caller at
// once, and reports whether this caller is the one that took the abandoned lock.
//
// Removing the lock and creating a new one is not one step. Two callers
// interleaved as remove-create-remove-create both come away holding "the" lock,
// because the second removal deleted the first caller's — after which both pass
// the under-lock recheck and both claim the same window. That is the outcome
// exclusivity exists to forbid: for a hook it duplicates a line, and for
// unattended update it is two processes racing one binary rename —
// unattended-update.spec.
//
// Renaming the lock aside is one step, and a path can be renamed away exactly
// once, so a caller that arrives after the winner finds nothing to take. It gets
// fs.ErrNotExist and must refuse rather than retry, because a retry lands back
// in the same race. A failed steal is a refusal under both biases — it cannot be
// told apart from a peer holding the lock, which the fail-open entry point
// already refuses on today.
//
// The rename alone would not be enough, because the staleness check that sent us
// here is a separate call: a peer can recover the lock and publish its own in
// between, and the rename would then land on that live lock instead of the
// abandoned one — the same two holders, one step later. So what the rename
// actually took is checked afterwards, on the side copy: it is the same file and
// it carries the same modification time, and only an abandoned lock is older
// than window. Taking a live peer's lock is not a win.
//
// Refusing is only half the answer, and it is the half a return value can carry:
// the rename has already happened, so the peer is left working under a name that
// holds nothing. The side copy is removed either way, and a lock taken by mistake
// is not put back. Restoring it looks tidier and is worse: by then the peer may
// have finished and released, and the restored file would sit at the lock's name
// with a fresh timestamp, holding the key un-claimable for a whole window with no
// process behind it. What covers the peer instead is the peer itself — it checks
// that it still holds its lock before it touches the stamp, and gives up when it
// does not. See lockHeld; the two halves are one guard and neither works alone.
//
// The lock keeps its name on disk, which two concurrent CLI versions depend on.
// Every exit path clears the side copy; a caller killed mid-steal leaves one, and
// sweep collects it on the usual schedule because it inherited the abandoned
// lock's modification time. A leftover whose name a later attempt reuses is
// replaced by that attempt's rename.
func stealAbandonedLock(lock string, window time.Duration) bool {
	side := fmt.Sprintf("%s.steal.%d.%d", lock, os.Getpid(), stealSeq.Add(1))
	if os.Rename(lock, side) != nil {
		return false
	}
	abandoned := stale(side, window)
	_ = os.Remove(side)
	return abandoned
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
