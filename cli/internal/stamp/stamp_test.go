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

// TestClaimStamp_OnDiskLayoutIsACrossProcessContract pins the two names the
// claim occupies on disk, as literals.
//
// Every other test in this file derives the expected path from PathFor and the
// expected lock from lockSuffix — the same code under test — so all of them stay
// green through a changed digest or a renamed suffix. The names are not internal:
// they are how two processes that never speak agree on who holds a claim, and
// unattended update makes different versions concurrent on one machine by
// construction, because the process that replaces the binary keeps running beside
// the ones that start after it. A version that computed a different stamp path,
// or a different lock name for the same key, would share a window with the old
// one and both would claim it — unattended-update.spec §6.
//
// The two behavioral assertions are what make this bite: each arranges a file at
// the literal name and requires the claim to see it.
func TestClaimStamp_OnDiskLayoutIsACrossProcessContract(t *testing.T) {
	t.Parallel()
	const (
		window = 10 * time.Minute
		// sha256("k"), first 16 bytes, hex.
		stampName = "8254c329a92850f6d539dd376f4816ee"
		lockName  = stampName + ".lock"
	)

	dir := t.TempDir()
	if got, want := PathFor(dir, "k"), filepath.Join(dir, stampName); got != want {
		t.Errorf("PathFor() = %q, want %q — a changed stamp name abandons every stamp already on disk", got, want)
	}

	t.Run("a stamp at the literal name suppresses", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, stampName), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if Claim(dir, "k", window) {
			t.Error("Claim() = true over a fresh stamp at the documented name: this build cannot see another build's claim")
		}
	})

	t.Run("a lock at the literal name holds off a reclaim", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		aged := time.Now().Add(-2 * window)
		if err := os.WriteFile(filepath.Join(dir, stampName), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filepath.Join(dir, stampName), aged, aged); err != nil {
			t.Fatal(err)
		}
		// A live lock: a peer is reclaiming the stale stamp right now.
		if err := os.WriteFile(filepath.Join(dir, lockName), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if Claim(dir, "k", window) {
			t.Error("Claim() = true while a lock at the documented name was held: this build cannot see another build's reclaim")
		}
	})
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

// TestClaimStamp_WinningClaimSweepsExpiredStamps pins the only thing that bounds
// a scope's directory.
//
// The session scope keys by session id, so its keys are never claimed twice: no
// key of it ever reaches the reclaim path, and the sweep the winner runs after
// creating its stamp is the sole reader that ever deletes one. Without it the
// directory grows by a file per session forever, and nothing else in this file
// notices — the sibling test above asserts a sweep does NOT cross scopes, which
// stays true when the sweep stops running at all.
func TestClaimStamp_WinningClaimSweepsExpiredStamps(t *testing.T) {
	t.Parallel()
	const window = 10 * time.Minute
	dir := t.TempDir()

	if !Claim(dir, "spent", window) {
		t.Fatal("first claim was refused")
	}
	spent := PathFor(dir, "spent")
	aged := time.Now().Add(-2 * window)
	if err := os.Chtimes(spent, aged, aged); err != nil {
		t.Fatal(err)
	}

	// A claim of an unrelated key: the only kind this scope ever sees again.
	if !Claim(dir, "fresh", window) {
		t.Fatal("a claim of an unclaimed key was refused")
	}

	if _, err := os.Stat(spent); !os.IsNotExist(err) {
		t.Errorf("the expired stamp survived a winning claim (err=%v): the scope grows without bound", err)
	}
	if !Fresh(dir, "fresh", window) {
		t.Error("the sweep took the stamp the winner had just published")
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

// TestClaimStamp_RecoversAnAbandonedLockThroughTheAtomicSteal pins the one thing
// about recovery no outcome test can see. A bare os.Remove followed by an
// exclusive create produces the same result as the steal in every arrangement a
// test can build: the difference only appears when two callers interleave inside
// a window scheduling cannot reach, and that interleave is what lets both of
// them hold the lock and, under ClaimFailClosed, race one binary rename —
// unattended-update.spec. Counting the seam is what tells the two apart, so a
// change that disconnects reclaim from the steal fails here rather than shipping
// as dead code.
func TestClaimStamp_RecoversAnAbandonedLockThroughTheAtomicSteal(t *testing.T) {
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
	if err := os.Chtimes(lock, aged, aged); err != nil {
		t.Fatal(err)
	}

	calls := 0
	original := stealLock
	stealLock = func(l string, w time.Duration) bool {
		calls++
		return original(l, w)
	}
	t.Cleanup(func() { stealLock = original })

	if !Claim(dir, "k", window) {
		t.Fatal("an abandoned lock left the key un-claimable")
	}
	if calls != 1 {
		t.Errorf("reclaim called the atomic steal %d time(s), want 1: an abandoned lock must be taken in one step, not removed and recreated", calls)
	}
}

// TestClaimStamp_RefusesWhenTheStealIsLost proves the refusal the steal's return
// value carries. A caller that loses the race for an abandoned lock must stop,
// not retry: a retry lands back in the race it just lost.
func TestClaimStamp_RefusesWhenTheStealIsLost(t *testing.T) {
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
	if err := os.Chtimes(lock, aged, aged); err != nil {
		t.Fatal(err)
	}

	original := stealLock
	stealLock = func(string, time.Duration) bool { return false }
	t.Cleanup(func() { stealLock = original })

	if Claim(dir, "k", window) {
		t.Error("Claim() = true after losing the steal: the winner is already inside the window")
	}
	if ClaimFailClosed(dir, "k", window) {
		t.Error("ClaimFailClosed() = true after losing the steal: two processes would race one binary rename")
	}
}

// TestClaimStamp_AbandonedLockHasOneWinner races the recovery path itself, which
// the concurrency tests never enter: they start from a clean directory, so no
// caller ever finds a lock to clear.
//
// Recovery is where exclusivity is most exposed. Every racing caller sees the
// same abandoned lock, every one of them clears it, and every one of them then
// retries the exclusive create — so all but one land on the "a peer took it
// first" refusal at once. That refusal is load-bearing for binary replacement:
// if it read as a win, one process killed mid-reclaim would let every caller that
// follows claim the same window and race the same rename —
// unattended-update.spec §6.
//
// A rare failure here is not flake, and must not be retried away. Recovery took
// the abandoned lock by removing it and creating a new one, and two callers
// interleaved as remove-create-remove-create both ended up holding a lock: the
// second one deleted the first one's. Recovery is a rename now — see
// TestStealAbandonedLock_HasOneWinner for the mechanism, which this test can
// only sample.
func TestClaimStamp_AbandonedLockHasOneWinner(t *testing.T) {
	t.Parallel()
	const (
		rounds  = 100
		callers = 8
		window  = 10 * time.Minute
	)
	dir := t.TempDir()

	for round := range rounds {
		key := fmt.Sprintf("k-%d", round)
		if !ClaimFailClosed(dir, key, window) {
			t.Fatalf("round %d: first claim was refused", round)
		}
		aged := time.Now().Add(-2 * window)
		if err := os.Chtimes(PathFor(dir, key), aged, aged); err != nil {
			t.Fatal(err)
		}
		// Left behind by a process killed between taking the lock and
		// refreshing the stamp.
		lock := PathFor(dir, key) + lockSuffix
		if err := os.WriteFile(lock, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(lock, aged, aged); err != nil {
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
				if ClaimFailClosed(dir, key, window) {
					won.Add(1)
				}
			})
		}
		close(start)
		done.Wait()

		// One caller past the race is what makes this exact. A racing caller may
		// legitimately come away with nothing: recovery renames a lock aside, so
		// a caller can have its own lock taken by a straggler whose staleness
		// check ran before that lock existed, and the only safe answer then is to
		// give up — see TestClaimStamp_GivesUpWhenItsLockIsTakenAway. What must
		// never happen is a window that goes to two callers, or a window that
		// goes to nobody at all: giving up leaves the stamp stale, so the claim
		// below finds it and takes it.
		if n := won.Load() + settle(t, dir, key, window); n != 1 {
			t.Fatalf("round %d: %d callers claimed a key whose lock was abandoned, want exactly 1", round, n)
		}
		if _, err := os.Stat(lock); !os.IsNotExist(err) {
			t.Fatalf("round %d: the reclaim lock was not released", round)
		}
	}
}

// settle claims once with no contention and reports the claim as a count, so a
// caller of it can add the uncontended answer to the contended one.
//
// A race can only be asserted on a total. Counting the racers alone asserts too
// much — a caller that finds its lock taken away has to give up, and a round it
// leaves empty is correct. Counting them and the settling claim together asserts
// exactly what the mechanism promises: the window goes to one caller, and it is
// never dropped on the floor.
func settle(t *testing.T, dir, key string, window time.Duration) int64 {
	t.Helper()
	if ClaimFailClosed(dir, key, window) {
		return 1
	}
	return 0
}

// TestStealAbandonedLock_HasOneWinner pins the recovery step itself, which the
// end-to-end race above can only sample: the interleaving that breaks it is a
// few instructions wide, and a 7000-round probe did not reproduce it on the
// machine this was written on.
//
// The property holds regardless of scheduling, which is what makes this test
// worth more than the sample. A path can be renamed away exactly once, so every
// caller but one gets fs.ErrNotExist. Remove-then-create had no such guarantee:
// two callers interleaved as remove-create-remove-create both came away holding
// a lock, and both then passed the under-lock recheck and claimed the same
// window — unattended-update.spec.
func TestStealAbandonedLock_HasOneWinner(t *testing.T) {
	t.Parallel()
	const (
		rounds  = 200
		callers = 8
		window  = 10 * time.Minute
	)
	dir := t.TempDir()
	lock := PathFor(dir, "k") + lockSuffix

	for round := range rounds {
		// Left behind by a process killed between taking the lock and
		// refreshing the stamp.
		if err := os.WriteFile(lock, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		aged := time.Now().Add(-2 * window)
		if err := os.Chtimes(lock, aged, aged); err != nil {
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
				if stealAbandonedLock(lock, window) {
					won.Add(1)
				}
			})
		}
		close(start)
		done.Wait()

		if n := won.Load(); n != 1 {
			t.Fatalf("round %d: %d callers stole one abandoned lock, want exactly 1", round, n)
		}
		if _, err := os.Stat(lock); !os.IsNotExist(err) {
			t.Fatalf("round %d: the abandoned lock is still in place (err=%v)", round, err)
		}
	}

	// The steal renames the lock aside before it removes it. A side copy that
	// outlives the steal is a file no reader of this directory expects, and it
	// would accumulate one per recovery until a sweep with a matching window ran.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("%d file(s) survived %d steals, want 0: the side copy is not cleaned up", len(entries), rounds)
	}
}

// TestStealAbandonedLock_RefusesALockThatTurnedOver is the interleave a rename
// does not close on its own, arranged deterministically: the lock the rename
// lands on is not the one the staleness check looked at.
//
// The check and the steal are two calls, and a peer fits a whole recovery
// between them — it takes the abandoned lock, publishes its own, and starts
// working under it. Renaming that live lock aside and calling it a win is the
// original bug wearing a rename: two callers hold "the" lock, the second one
// having taken the first one's away — unattended-update.spec. The arrangement
// stands in for the peer by leaving a lock that is younger than the window,
// which is the one thing an abandoned lock can never be.
//
// Refusing is only half the answer, and it is the half a return value can carry:
// the rename has already happened, and the peer whose lock it took keeps working.
// The other half is TestClaimStamp_GivesUpWhenItsLockIsTakenAway, where that peer
// finds out. Neither half holds the invariant alone.
func TestStealAbandonedLock_RefusesALockThatTurnedOver(t *testing.T) {
	t.Parallel()
	const window = 10 * time.Minute
	dir := t.TempDir()
	lock := PathFor(dir, "k") + lockSuffix
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if stealAbandonedLock(lock, window) {
		t.Error("stealAbandonedLock() = true over a live peer's lock: the caller that published it is holding it")
	}

	// The taken lock is not put back. The sibling test below is the reason: a
	// restored lock outlives the peer that published it, and nothing clears it
	// for a whole window.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the refused steal left %d file(s), want 0: the side copy is not cleaned up", len(entries))
	}
}

// TestClaimStamp_GivesUpWhenItsLockIsTakenAway is the other half of the refusal
// above, and the half that actually holds the invariant.
//
// A caller inside the reclaim can have its lock renamed away by a straggler whose
// staleness check ran before that lock existed. The straggler refuses, but the
// rename already happened: the holder is working under a name that leads nowhere,
// the next caller finds the name free, creates a lock of its own, sees the stamp
// still stale — the holder has not refreshed it yet — and refreshes it too. Two
// attempts in one window, across concurrent processes, is the invariant
// unattended update rests on — unattended-update.spec.
//
// Nothing can stop the rename, so the holder is what notices. This drives the
// notice through its seam, because the theft has to land between the holder's
// exclusive create and its next syscall, and no test can ask for that. The stamp
// is the second assertion and the load-bearing one: giving up means giving up
// before touching it, so the next caller still finds it stale and reclaims it.
//
// Not parallel: it replaces the lockHeld seam.
func TestClaimStamp_GivesUpWhenItsLockIsTakenAway(t *testing.T) {
	const window = 10 * time.Minute
	dir := t.TempDir()

	if !Claim(dir, "k", window) {
		t.Fatal("first claim was refused")
	}
	aged := time.Now().Add(-2 * window)
	if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
		t.Fatal(err)
	}

	original := lockHeld
	lockHeld = func(*os.File, string) bool { return false }
	t.Cleanup(func() { lockHeld = original })

	if Claim(dir, "k", window) {
		t.Error("Claim() = true after its lock was taken away: the caller that took it may already be inside the window")
	}
	if ClaimFailClosed(dir, "k", window) {
		t.Error("ClaimFailClosed() = true after its lock was taken away: two processes would race one binary rename")
	}
	if Fresh(dir, "k", window) {
		t.Error("the stamp was refreshed by a caller that had lost its lock: a caller that gives up must leave the stamp for the next one")
	}
}

// TestHoldsLock covers the three answers the holder's check can reach, since the
// interleave that produces the last two cannot be scheduled from a test.
func TestHoldsLock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// arrange runs after the lock is created and returns what the check
		// should answer for the state it leaves behind.
		arrange func(t *testing.T, lock string)
		want    bool
	}{
		{
			name:    "the file this caller created is still at the name",
			arrange: func(*testing.T, string) {},
			want:    true,
		},
		{
			name: "a straggler renamed the lock away and nothing replaced it",
			arrange: func(t *testing.T, lock string) {
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "a later caller published its own lock at the freed name",
			arrange: func(t *testing.T, lock string) {
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(lock, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lock := PathFor(t.TempDir(), "k") + lockSuffix
			lf, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = lf.Close() }()
			tt.arrange(t, lock)

			if got := holdsLock(lf, lock); got != tt.want {
				t.Errorf("holdsLock() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClaimStamp_DoesNotDisturbALiveLock pins the guard that keeps a live peer's
// lock out of the steal path in the first place.
//
// The steal refuses a live lock and puts it back, so an outcome test cannot see
// the difference between "never touched it" and "took it and repaired it". The
// repair is a fallback for a race, not a licence to rename a lock this caller can
// already see is fresh: a repaired lock is a lock that was briefly absent, and
// the one that is never taken never was.
func TestClaimStamp_DoesNotDisturbALiveLock(t *testing.T) {
	const window = 10 * time.Minute
	dir := t.TempDir()

	if !Claim(dir, "k", window) {
		t.Fatal("first claim was refused")
	}
	aged := time.Now().Add(-2 * window)
	if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
		t.Fatal(err)
	}
	// Fresh: a peer is reclaiming right now.
	if err := os.WriteFile(PathFor(dir, "k")+lockSuffix, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	original := stealLock
	stealLock = func(l string, w time.Duration) bool {
		calls++
		return original(l, w)
	}
	t.Cleanup(func() { stealLock = original })

	if Claim(dir, "k", window) {
		t.Error("Claim() = true while a peer held a live reclaim lock")
	}
	if calls != 0 {
		t.Errorf("reclaim called the steal %d time(s) on a live lock, want 0: only an abandoned lock may be taken", calls)
	}
}

// TestClaimStamp_RefusesWhenAPeerRelocksAfterTheSteal covers the last gap in the
// recovery: this caller wins the steal, and a peer publishes its own lock at the
// freed name before this caller's exclusive create reaches it.
//
// The create then fails with fs.ErrExist, which is a peer holding the lock and
// not a filesystem that failed to answer — so it must refuse under BOTH biases.
// Letting it fall through to the fail-open answer is what makes this worth
// pinning: the assertion is on Claim, because ClaimFailClosed refuses either way
// and would pass over the defect.
//
// Not parallel: it replaces the stealLock seam.
func TestClaimStamp_RefusesWhenAPeerRelocksAfterTheSteal(t *testing.T) {
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
	if err := os.Chtimes(lock, aged, aged); err != nil {
		t.Fatal(err)
	}

	original := stealLock
	// A won steal that leaves the name occupied stands in for the peer: it is
	// what this caller sees when one publishes between the rename and the create.
	stealLock = func(string, time.Duration) bool { return true }
	t.Cleanup(func() { stealLock = original })

	if Claim(dir, "k", window) {
		t.Error("Claim() = true with a lock at the name after the steal: the peer that published it is inside the window")
	}
}

// TestClaimStamp_ReclaimSweepsTheScope pins the sweep on the path that the
// longest-lived scope actually takes.
//
// The sibling test above covers the sweep a first claim runs. The update scope
// never reaches it twice: it keys by binary path, so after the first day every
// claim finds a stamp and goes through the reclaim instead, and the sweep there
// is the only reader that ever deletes anything in that directory again. A side
// copy left by a caller killed mid-steal inherits the abandoned lock's
// timestamp, so nothing else collects it.
func TestClaimStamp_ReclaimSweepsTheScope(t *testing.T) {
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
	// Left by a process killed between the steal's rename and its removal.
	orphan := PathFor(dir, "k") + lockSuffix + ".steal.1.1"
	if err := os.WriteFile(orphan, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphan, aged, aged); err != nil {
		t.Fatal(err)
	}

	if !Claim(dir, "k", window) {
		t.Fatal("a stale stamp was not reclaimed")
	}

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("the orphaned side copy survived a reclaim (err=%v): a scope that only ever reclaims grows without bound", err)
	}
}

// TestClaimStamp_RecoveryLeavesNoLockBehind is the sibling of the test above,
// and the reason a mistaken steal removes rather than restores.
//
// Recovery races callers whose staleness checks and renames interleave freely,
// so a caller can take a lock a peer published a moment earlier. Putting that
// lock back resurrects it: the peer that owned it may already have released, its
// own removal having found nothing, and the restored file then holds the key
// un-claimable until a sweep with a matching window runs. This asserts the end
// state instead of the step — after every caller returns, no lock survives.
func TestClaimStamp_RecoveryLeavesNoLockBehind(t *testing.T) {
	t.Parallel()
	const (
		rounds  = 100
		callers = 8
		window  = 10 * time.Minute
	)
	dir := t.TempDir()

	for round := range rounds {
		key := fmt.Sprintf("k-%d", round)
		if !ClaimFailClosed(dir, key, window) {
			t.Fatalf("round %d: first claim was refused", round)
		}
		aged := time.Now().Add(-2 * window)
		if err := os.Chtimes(PathFor(dir, key), aged, aged); err != nil {
			t.Fatal(err)
		}
		lock := PathFor(dir, key) + lockSuffix
		if err := os.WriteFile(lock, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(lock, aged, aged); err != nil {
			t.Fatal(err)
		}

		var (
			done  sync.WaitGroup
			start = make(chan struct{})
		)
		for range callers {
			done.Go(func() {
				<-start // release them together, so they collide
				ClaimFailClosed(dir, key, window)
			})
		}
		close(start)
		done.Wait()

		if _, err := os.Stat(lock); !os.IsNotExist(err) {
			t.Fatalf("round %d: a lock survived the recovery (err=%v): the key is un-claimable until a sweep clears it", round, err)
		}
		// The racers may all have given up, which is allowed; what is not allowed
		// is a directory a later caller cannot claim through.
		if settle(t, dir, key, window) == 0 && !Fresh(dir, key, window) {
			t.Fatalf("round %d: no stamp survived the recovery", round)
		}
	}
}

// TestClaimStamp_UnstealableLockRefusesUnderBothBiases pins the one filesystem
// failure the fail-open entry point does not claim through.
//
// A steal that does not land carries no information about who holds the lock:
// the same refusal covers a peer that got there first and a directory that will
// not accept the rename. Claiming on it would hand the same window to every
// caller at once, which is what exclusivity forbids for a binary replacement —
// unattended-update.spec §6. The refusal is also what the recovery path already
// did before the steal replaced the remove, so the fail-open bias is unchanged.
func TestClaimStamp_UnstealableLockRefusesUnderBothBiases(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("root writes through the read-only directory this case needs")
	}
	const window = 10 * time.Minute
	dir := t.TempDir()

	aged := time.Now().Add(-2 * window)
	for _, path := range []string{PathFor(dir, "k"), PathFor(dir, "k") + lockSuffix} {
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, aged, aged); err != nil {
			t.Fatal(err)
		}
	}
	readOnlyDir(t, dir)

	if ClaimFailClosed(dir, "k", window) {
		t.Error("ClaimFailClosed() = true, want false: a lock this caller could not take may still be held")
	}
	if Claim(dir, "k", window) {
		t.Error("Claim() = true, want false: a failed steal is not a win under either bias")
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

// rootOwnedFile is a path that exists on macOS and Linux and belongs to root, so
// this user may read its timestamps but not set them. That is the only portable
// arrangement in which os.Stat succeeds and os.Chtimes fails, which is what the
// timestamp-refresh branch of the reclaim needs. The tests that use it verify
// the denial first and never modify the file.
const rootOwnedFile = "/etc/hosts"

// skipUnlessTimestampsDenied verifies the precondition rather than assuming it:
// on a machine where this user may set the file's timestamps, the arrangement
// below would exercise the success path and the test would pass for the wrong
// reason. The probe writes the file's own modification time back, and a zero
// access time leaves that field alone, so a machine that permits it is unchanged.
func skipUnlessTimestampsDenied(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Skipf("%s is absent on this machine", path)
	}
	if os.Chtimes(path, time.Time{}, info.ModTime()) == nil {
		t.Skipf("this user may set the timestamps of %s", path)
	}
}

// symlinkOrSkip skips on a host that does not grant this process the right to
// create symlinks, rather than reporting the arrangement's absence as a failure.
func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
}

// readOnlyDir makes dir unwritable for the rest of the test and restores it
// afterwards, so t.TempDir can still remove it.
func readOnlyDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

// TestClaimFailClosed_RefusesOnFilesystemFailure covers every point where the
// filesystem fails rather than answers — the four returns that Claim answers with
// "you won" and the unresolved state directory. Binary replacement claims through
// this entry point, and a claim it cannot establish must read as "a peer may hold
// it": two winners race the same rename — unattended-update.spec §6.
//
// Each case asserts both entry points on one arrangement. The fail-open answer is
// what proves the branch was reached at all: a refusal alone is also what a fresh
// peer stamp produces, so without it a case that stopped earlier than intended
// would still pass.
func TestClaimFailClosed_RefusesOnFilesystemFailure(t *testing.T) {
	t.Parallel()
	const window = time.Minute

	tests := []struct {
		name       string
		needsWrite bool // an arrangement root would write through
		arrange    func(t *testing.T) string
	}{
		{
			name: "no state directory resolved",
			arrange: func(t *testing.T) string {
				return ""
			},
		},
		{
			name: "state directory cannot be created because its parent is a file",
			arrange: func(t *testing.T) string {
				parent := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(parent, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(parent, "scope")
			},
		},
		{
			name:       "state directory cannot be created in a read-only parent",
			needsWrite: true,
			arrange: func(t *testing.T) string {
				parent := t.TempDir()
				readOnlyDir(t, parent)
				return filepath.Join(parent, "scope")
			},
		},
		{
			name:       "stamp cannot be created in a read-only state directory",
			needsWrite: true,
			arrange: func(t *testing.T) string {
				dir := t.TempDir()
				readOnlyDir(t, dir)
				return dir
			},
		},
		{
			name:       "reclaim lock cannot be created in a read-only state directory",
			needsWrite: true,
			arrange: func(t *testing.T) string {
				dir := t.TempDir()
				aged := time.Now().Add(-2 * window)
				if err := os.WriteFile(PathFor(dir, "k"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
					t.Fatal(err)
				}
				readOnlyDir(t, dir)
				return dir
			},
		},
		{
			name:       "stale stamp cannot have its timestamp refreshed",
			needsWrite: true,
			arrange: func(t *testing.T) string {
				skipUnlessTimestampsDenied(t, rootOwnedFile)
				info, err := os.Stat(rootOwnedFile)
				if err != nil {
					t.Fatal(err)
				}
				if time.Since(info.ModTime()) < window {
					t.Skipf("%s was modified inside the claim window", rootOwnedFile)
				}
				dir := t.TempDir()
				symlinkOrSkip(t, rootOwnedFile, PathFor(dir, "k"))
				return dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.needsWrite && os.Getuid() == 0 {
				t.Skip("root writes through the arrangement this case needs")
			}
			dir := tt.arrange(t)

			if ClaimFailClosed(dir, "k", window) {
				t.Error("ClaimFailClosed() = true, want false: a claim that cannot be established must refuse")
			}
			if !Claim(dir, "k", window) {
				t.Error("Claim() = false, want true: the case did not reach the branch it targets, so the refusal above proves nothing")
			}
		})
	}
}

// TestClaimFailClosed_Window pins that the inverted bias changed the failure
// answers only: on a working filesystem both entry points behave identically.
func TestClaimFailClosed_Window(t *testing.T) {
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
			if !ClaimFailClosed(dir, "k", tt.window) {
				t.Fatal("first claim was refused")
			}
			aged := time.Now().Add(-tt.age)
			if err := os.Chtimes(PathFor(dir, "k"), aged, aged); err != nil {
				t.Fatal(err)
			}
			if got := ClaimFailClosed(dir, "k", tt.window); got != tt.wantClaim {
				t.Errorf("ClaimFailClosed(age=%v, window=%v) = %v, want %v", tt.age, tt.window, got, tt.wantClaim)
			}
		})
	}
}

// TestClaimFailClosed_HasOneWinner is the invariant the update policy rests on:
// at most one attempt per binary path per window, across every concurrent process
// on the machine — unattended-update.spec. Both entries into the claim are raced,
// because the fresh path and the stale reclaim take different routes to it.
func TestClaimFailClosed_HasOneWinner(t *testing.T) {
	t.Parallel()
	const (
		rounds  = 100
		callers = 8
		window  = 10 * time.Minute
	)

	for _, tt := range []struct {
		name  string
		stale bool
	}{
		{name: "no stamp yet"},
		{name: "stale stamp", stale: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for round := range rounds {
				key := fmt.Sprintf("k-%d", round)
				if tt.stale {
					if !ClaimFailClosed(dir, key, window) {
						t.Fatalf("round %d: first claim was refused", round)
					}
					aged := time.Now().Add(-2 * window)
					if err := os.Chtimes(PathFor(dir, key), aged, aged); err != nil {
						t.Fatal(err)
					}
				}

				var (
					done  sync.WaitGroup
					won   atomic.Int64
					start = make(chan struct{})
				)
				for range callers {
					done.Go(func() {
						<-start // release them together, so they collide
						if ClaimFailClosed(dir, key, window) {
							won.Add(1)
						}
					})
				}
				close(start)
				done.Wait()

				if n := won.Load(); n != 1 {
					t.Fatalf("round %d: %d callers claimed the same key, want exactly 1", round, n)
				}
				if !Fresh(dir, key, window) {
					t.Fatalf("round %d: the winner's stamp is gone — a loser removed it", round)
				}
			}
		})
	}
}

// TestDirFor pins the layout across the move to internal/xdg. The path was
// derived inline here before, and three other places in the repo derive the
// same directory; a scope that moved even one level would silently abandon
// every stamp already on disk and let a rate-limited advisory fire again on
// every machine that upgraded.
//
// Not parallel: it sets XDG_STATE_HOME.
func TestDirFor(t *testing.T) {
	t.Run("XDG_STATE_HOME wins", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", root)

		want := filepath.Join(root, "archcore", "session-stamps")
		if got := DirFor("session-stamps"); got != want {
			t.Errorf("DirFor() = %q, want %q", got, want)
		}
	})

	// The home-directory fallback and the unresolvable case are internal/xdg's
	// contract and are covered there; what this file owns is the scope segment
	// appended to it.

	t.Run("resolving creates nothing", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", root)

		_ = DirFor("session-stamps")

		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("DirFor() created %d entries, want 0", len(entries))
		}
	})
}
