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

// checkUpdater builds an Updater whose GitHub API calls hit srv.
func checkUpdater(version string, srv *httptest.Server) *update.Updater {
	client := &http.Client{Transport: &testRewriteTransport{target: srv.URL}}
	return &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
	}
}

func releaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name": "` + tag + `"}`))
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
		_, _ = w.Write([]byte(`{"tag_name": "v9.9.9"}`))
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
	old := time.Now().Add(-updateCheckFailureTTL - time.Minute)
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

func TestUpdateCheck_ExpiredCacheRefetches(t *testing.T) {
	t.Parallel()
	srv := releaseServer(t, "v9.9.9")
	cache := filepath.Join(t.TempDir(), "last-update-check")
	if err := os.WriteFile(cache, []byte("v8.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-updateCheckTTL - time.Minute)
	if err := os.Chtimes(cache, old, old); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	runUpdateCheck(context.Background(), &out, "v0.5.7", checkUpdater("v0.5.7", srv), cache)

	if out.String() != "update available: v9.9.9\n" {
		t.Errorf("expired cache must refetch: output = %q", out.String())
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
}
