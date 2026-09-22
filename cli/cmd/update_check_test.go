package cmd

// Tests for `archcore update --check` — the quiet freshness probe consumed by
// the plugin's session-start advisory. Contract: one line "update available:
// vX" when behind, silence when current or on ANY failure, result cached with
// a TTL, and never a non-zero exit.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"archcore-cli/internal/update"
)

// checkUpdater builds an Updater whose github.com requests — the
// /releases/latest redirect lookup — hit srv instead.
func checkUpdater(version string, srv *httptest.Server) *update.Updater {
	client := &http.Client{Transport: &testRewriteTransport{target: srv.URL}}
	return &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
	}
}

// releaseServer mimics github.com/OWNER/REPO/releases/latest, which answers
// with a 302 to the tag page rather than a JSON body.
func releaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/"+tag)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestUpdateCheck_ReportsNewerVersion(t *testing.T) {
	t.Parallel()
	srv := releaseServer(t, "v9.9.9")
	cache := filepath.Join(t.TempDir(), "last-update-check")
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv), cache)

	got := out.String()
	if got != "update available: v9.9.9\n" {
		t.Errorf("output = %q, want single 'update available: v9.9.9' line", got)
	}
	if data, err := os.ReadFile(cache); err != nil || strings.TrimSpace(string(data)) != "v9.9.9" {
		t.Errorf("cache not written: err=%v data=%q", err, data)
	}
}

func TestUpdateCheck_SilentWhenCurrent(t *testing.T) {
	t.Parallel()
	srv := releaseServer(t, "v0.5.7")
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv),
		filepath.Join(t.TempDir(), "last-update-check"))

	if out.Len() != 0 {
		t.Errorf("must be silent when up to date, got %q", out.String())
	}
}

func TestUpdateCheck_SilentOnNetworkFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv),
		filepath.Join(t.TempDir(), "last-update-check"))

	if out.Len() != 0 {
		t.Errorf("must be silent on API failure, got %q", out.String())
	}
}

func TestUpdateCheck_FreshCacheSkipsNetwork(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v9.9.9")
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	cache := filepath.Join(t.TempDir(), "last-update-check")
	if err := os.WriteFile(cache, []byte("v8.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv), cache)

	if n := requests.Load(); n != 0 {
		t.Errorf("fresh cache must skip the network, saw %d requests", n)
	}
	if out.String() != "update available: v8.0.0\n" {
		t.Errorf("output = %q, want cached v8.0.0 advisory", out.String())
	}
}

// A failed check must be negative-cached: within the failure TTL no further
// network requests are made (hooks must not re-pay the probe timeout), and
// once the failure stamp expires the check fetches again.
func TestUpdateCheck_FailureIsNegativeCached(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cache := filepath.Join(t.TempDir(), "last-update-check")
	u := checkUpdater("v0.5.7", srv)
	var out bytes.Buffer

	// First call: hits the network, fails silently, writes the failure stamp.
	runUpdateCheck(context.Background(), &out, "v0.5.7", u, cache)
	if n := requests.Load(); n == 0 {
		t.Fatal("first call must attempt the network")
	}
	if out.Len() != 0 {
		t.Fatalf("failure must be silent, got %q", out.String())
	}
	after := requests.Load()

	// Second call within the failure TTL: no new request.
	runUpdateCheck(context.Background(), &out, "v0.5.7", u, cache)
	if n := requests.Load(); n != after {
		t.Errorf("fresh failure stamp must skip the network, saw %d extra request(s)", n-after)
	}

	// Expired failure stamp: fetches again.
	old := time.Now().Add(-update.CacheFailureTTL - time.Minute)
	if err := os.Chtimes(cache, old, old); err != nil {
		t.Fatal(err)
	}
	runUpdateCheck(context.Background(), &out, "v0.5.7", u, cache)
	if n := requests.Load(); n == after {
		t.Error("expired failure stamp must refetch")
	}
	if out.Len() != 0 {
		t.Errorf("still-failing check must stay silent, got %q", out.String())
	}
}

// probeReturnBound is what "cheap enough for a session-start hook" means in
// wall-clock terms: the deadline in cmd is 2 s, and a probe still running an
// order of magnitude past it is holding the hook open, whatever the constant
// says. It is a literal rather than a multiple of updateCheckTimeout so that
// raising that constant fails here instead of silently widening this bound.
const probeReturnBound = 20 * time.Second

// A host that accepts the connection and then never answers is the case the
// negative cache cannot help with: nothing is cached yet, so the probe pays the
// full stall. `archcore update --check` runs inside the plugin's session-start
// advisory, so the call must come back on its own deadline, stay silent, and
// leave the failure stamp that spares the next session the same wait.
//
// Deleting the deadline leaves every other test in this file green — they all
// answer immediately — and turns a github.com outage into a hook that never
// returns.
func TestUpdateCheck_HangingHostReturnsOnItsOwnDeadline(t *testing.T) {
	t.Parallel()

	// The handler holds the request until the test is over. Cleanups run LIFO,
	// so releasing is registered last and runs before srv.Close, which waits on
	// the handler.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	cache := filepath.Join(t.TempDir(), "last-update-check")
	// The buffer stays inside the goroutine and is handed over on the channel:
	// after a timeout the probe is still running, and a buffer shared with it
	// would be a data race rather than a failure.
	done := make(chan string, 1)
	go func() {
		var out bytes.Buffer
		runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv), cache)
		done <- out.String()
	}()

	select {
	case out := <-done:
		if out != "" {
			t.Errorf("a stalled probe must stay silent, got %q", out)
		}
		if latest, fresh := update.ReadCachedLatest(cache); latest != "" || !fresh {
			t.Errorf("cache after a stall = (%q, %v), want the failure stamp (\"\", true)", latest, fresh)
		}
	case <-time.After(probeReturnBound):
		t.Fatalf("the probe did not return within %s: --check is unbounded and would hold a session-start hook open for as long as the host stalls", probeReturnBound)
	}
}

func TestUpdateCheck_ExpiredCacheRefetches(t *testing.T) {
	t.Parallel()
	srv := releaseServer(t, "v9.9.9")
	cache := filepath.Join(t.TempDir(), "last-update-check")
	if err := os.WriteFile(cache, []byte("v8.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-update.CacheTTL - time.Minute)
	if err := os.Chtimes(cache, old, old); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv), cache)

	if out.String() != "update available: v9.9.9\n" {
		t.Errorf("expired cache must refetch: output = %q", out.String())
	}
}

// The command must write the cache internal/update names, not one of its own.
// The unattended policy reads that same file to decide whether to look up a
// release at all, so a second derivation here would give one machine two
// caches and cost the policy the network budget the cache exists to save —
// unattended-update.spec.
func TestUpdateCheck_CommandUsesTheCanonicalCachePath(t *testing.T) {
	// Not parallel: the cache path is derived from XDG_STATE_HOME.
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	srv := releaseServer(t, "v9.9.9")

	client := &http.Client{Transport: &testRewriteTransport{target: srv.URL}}
	cmd := newUpdateCmdWithClient("v0.5.7", client)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--check"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update --check must always exit 0, got: %v", err)
	}
	if got := out.String(); got != "update available: v9.9.9\n" {
		t.Errorf("output = %q, want the advisory line", got)
	}

	cached, fresh := update.ReadCachedLatest(update.CachePath())
	if cached != "v9.9.9" || !fresh {
		t.Errorf("canonical cache = (%q, %v), want (\"v9.9.9\", true)", cached, fresh)
	}
	if _, err := os.Stat(filepath.Join(state, "archcore", "last-update-check")); err != nil {
		t.Errorf("cache is not at the shared state path: %v", err)
	}
}

func TestUpdateCheck_CommandExitsZeroAndQuietOnFailure(t *testing.T) {
	// End-to-end through cobra: --check with an erroring API must exit 0.
	// Not parallel: XDG_STATE_HOME env mutation isolates the real cache path.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: &testRewriteTransport{target: srv.URL}}
	cmd := newUpdateCmdWithClient("v0.5.7", client)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--check"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update --check must always exit 0, got: %v", err)
	}
	// The other half of this test's name, and the half a hook actually reads:
	// an advisory that prints a stack trace or a "could not check" line on every
	// offline session is worse than one that prints nothing.
	if got := out.String(); got != "" {
		t.Errorf("a failed --check must print nothing, got %q", got)
	}
}
