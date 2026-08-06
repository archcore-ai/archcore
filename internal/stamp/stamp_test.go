package stamp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestClaimStamp_ScopesDoNotExpireEachOther pins the reason each dedup scope
// owns a directory. A sweep deletes everything older than ITS window, so a
// 10-minute scope sharing storage with a 24-hour scope would wipe the day-long
// rate limit on its first run — the failure would be silent, and the only
// symptom a staleness advisory that reappears every session.
func TestClaimStamp_ScopesDoNotExpireEachOther(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	shortDir := filepath.Join(root, "session-stamps")
	longDir := filepath.Join(root, "staleness-stamps")

	if !Claim(longDir, "daily", 24*time.Hour) {
		t.Fatal("first claim in the long scope was refused")
	}
	// Age the long-scope stamp well past the short window but inside its own.
	aged := time.Now().Add(-time.Hour)
	if err := os.Chtimes(PathFor(longDir, "daily"), aged, aged); err != nil {
		t.Fatal(err)
	}

	// A short-scope claim sweeps its own directory only.
	if !Claim(shortDir, "session", 10*time.Minute) {
		t.Fatal("first claim in the short scope was refused")
	}

	if !Fresh(longDir, "daily", 24*time.Hour) {
		t.Error("the 24h stamp was expired by the 10-minute scope's sweep")
	}
	if Claim(longDir, "daily", 24*time.Hour) {
		t.Error("the 24h rate limit did not hold after a short-scope claim")
	}
}

func TestClaimStamp_Window(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		age       time.Duration
		window    time.Duration
		wantClaim bool
	}{
		{name: "fresh stamp suppresses", age: time.Minute, window: 10 * time.Minute, wantClaim: false},
		{name: "expired stamp is reclaimed", age: 20 * time.Minute, window: 10 * time.Minute, wantClaim: true},
		{name: "stamp inside a long window suppresses", age: 2 * time.Hour, window: 24 * time.Hour, wantClaim: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if !Claim(dir, "k", tt.window) {
				t.Fatal("first claim was refused")
			}
			aged := time.Now().Add(-tt.age)
			if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
				t.Fatal(err)
			}
			if got := Claim(dir, "k", tt.window); got != tt.wantClaim {
				t.Errorf("Claim(age=%v, window=%v) = %v, want %v", tt.age, tt.window, got, tt.wantClaim)
			}
		})
	}
}

// TestClaimStamp_ReclaimRefreshesInPlace pins the mechanism that makes the
// reclaim safe, deterministically — the concurrency test below can only sample
// the race, and a rare window is not something to leave to sampling.
//
// A stale stamp used to be reclaimed by removing it and creating it again. The
// gap between those two calls is exactly where another caller's plain O_EXCL
// create succeeds: it holds no lock, sees no file, and takes a second claim for
// the same event — after which the loser's own removal can delete the stamp the
// winner just published. Refreshing the timestamp in place closes the gap
// because the file never stops existing.
func TestClaimStamp_ReclaimRefreshesInPlace(t *testing.T) {
	t.Parallel()
	const window = 10 * time.Minute
	dir := t.TempDir()

	if !Claim(dir, "k", window) {
		t.Fatal("first claim was refused")
	}
	path := PathFor(dir, "k")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	aged := time.Now().Add(-2 * window)
	if err := os.Chtimes(path, aged, aged); err != nil {
		t.Fatal(err)
	}

	if !Claim(dir, "k", window) {
		t.Fatal("a stale stamp was not reclaimed")
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("no stamp after the reclaim: %v", err)
	}
	if !os.SameFile(before, after) {
		t.Error("the reclaim replaced the stamp file instead of refreshing it in place")
	}
	if !after.ModTime().After(aged) {
		t.Error("the reclaimed stamp is still stale")
	}
}

// TestClaimStamp_StaleReclaimHasOneWinner is the point of the whole mechanism:
// concurrent callers agree on who emits, and the winner's stamp survives.
//
// Sampled rather than proved — the window this closes opens rarely, and pushing
// the sample far enough to hit it reliably costs seconds. The test above pins
// the mechanism; this one guards the property callers actually depend on.
func TestClaimStamp_StaleReclaimHasOneWinner(t *testing.T) {
	t.Parallel()
	const (
		rounds  = 200
		callers = 8
		window  = 10 * time.Minute
	)
	dir := t.TempDir()

	for round := range rounds {
		key := fmt.Sprintf("k-%d", round)
		if !Claim(dir, key, window) {
			t.Fatalf("round %d: first claim was refused", round)
		}
		aged := time.Now().Add(-2 * window)
		if err := os.Chtimes(PathFor(dir, key), aged, aged); err != nil {
			t.Fatal(err)
		}

		var (
			done  sync.WaitGroup
			won   atomic.Int64
			start = make(chan struct{})
		)
		for range callers {
			done.Go(func() {
				<-start // release them together, so they collide
				if Claim(dir, key, window) {
					won.Add(1)
				}
			})
		}
		close(start)
		done.Wait()

		if n := won.Load(); n != 1 {
			t.Fatalf("round %d: %d callers claimed a single stale stamp, want exactly 1", round, n)
		}
		if !Fresh(dir, key, window) {
			t.Fatalf("round %d: the winner's stamp is gone — a loser removed it", round)
		}
	}
}

// TestClaimStamp_AbandonedLockIsCleared: the reclaim lock is a file, so a
// process killed mid-reclaim leaves one behind. If nothing cleared it, that key
// would be permanently un-claimable and the event would never fire again.
func TestClaimStamp_AbandonedLockIsCleared(t *testing.T) {
	t.Parallel()
	const window = 10 * time.Minute
	dir := t.TempDir()

	if !Claim(dir, "k", window) {
		t.Fatal("first claim was refused")
	}
	aged := time.Now().Add(-2 * window)
	if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
		t.Fatal(err)
	}
	lock := PathFor(dir, "k") + lockSuffix
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	// A lock younger than the window belongs to a live peer: back off.
	if Claim(dir, "k", window) {
		t.Error("claimed while a live peer held the reclaim lock")
	}

	if err := os.Chtimes(lock, aged, aged); err != nil {
		t.Fatal(err)
	}
	if !Claim(dir, "k", window) {
		t.Error("an abandoned reclaim lock left the key permanently un-claimable")
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Error("the reclaim lock was not released")
	}
}

// TestClaimStamp_FailsOpen: dedup must never be what breaks a hook.
func TestClaimStamp_FailsOpen(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	if !Claim(filepath.Join(parent, "unwritable"), "k", time.Minute) {
		t.Error("Claim on an unwritable directory = false, want true (fail open)")
	}
}
