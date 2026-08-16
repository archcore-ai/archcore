package update

// The unattended policy is the one path in this repository that replaces a
// binary with nobody watching, so what these tests hold down is mostly what it
// must NOT do: refuse before it touches the filesystem, claim before it reports
// anything, and never turn a refusal into a failure event.
//
// Two conventions run through the file.
//
// Every test that must reach past the CI condition calls clearCIEnv. This suite
// itself runs on CI runners, where CI=true, and without that call every such
// test would pass by refusing for the wrong reason — green, and blind to the
// whole policy below condition 3.
//
// Every wire string is written out as a literal. "cli_updated", "auto",
// "not_writable" and the five stage values are a contract with PostHog and with
// the landing repository's event map; reading the constants back would let a
// rename sail through here and empty a dashboard instead.

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"archcore-cli/internal/stamp"
	"archcore-cli/internal/telemetry"
)

// policyUpdater is the Updater every policy test builds. Only three of its
// fields ever differ between them — the version the machine runs, the binary it
// points at, and the client it talks to — and the repository and binary name
// were spelled out at all twenty call sites.
func policyUpdater(version, execPath string, client *http.Client) *Updater {
	return &Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
		ExecPath:       execPath,
	}
}

// --- helpers -----------------------------------------------------------------

const (
	// originalBinary is what the fake install holds before an attempt. A test
	// that must leave the binary untouched compares against it byte for byte.
	originalBinary = "old binary"

	// healthyPayload passes healthProbe: it starts, prints nothing and exits 0.
	healthyPayload = "#!/bin/sh\nexit 0\n"
)

// withOfficialBuild sets the release marker for one test and restores it.
//
// The marker is a package variable, so no test in this file may call
// t.Parallel: a parallel sibling would observe the mutation and refuse (or fail
// to refuse) for a reason that has nothing to do with what it asserts.
func withOfficialBuild(t *testing.T, value string) {
	t.Helper()
	previous := officialBuild
	officialBuild = value
	t.Cleanup(func() { officialBuild = previous })
}

// clearCIEnv empties every CI variable for the duration of the test. Mandatory
// in any test that must reach past condition 3 — see the file comment.
func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, name := range telemetry.CIVars {
		t.Setenv(name, "")
	}
}

// unattendedEnv puts one test on a machine the policy would act on: an official
// build, no CI variable, no telemetry opt-out, and a state directory of its own
// so nothing here reads or writes the developer's real one.
//
// HOME is redirected as well as XDG_STATE_HOME. xdg.StateDir falls back to
// $HOME/.local/state whenever XDG_STATE_HOME is empty, so a test that clears the
// first variable — or a change to that precedence — would otherwise write claim
// stamps and an install-id into the developer's own state directory, where they
// silently suppress the next real update for a day.
func unattendedEnv(t *testing.T) {
	t.Helper()
	withOfficialBuild(t, "release")
	clearCIEnv(t)
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("ARCHCORE_TELEMETRY_OPTOUT", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// fakeInstall creates a binary to replace and returns its directory and path,
// both resolved through EvalSymlinks. The resolution matters: on macOS a temp
// directory lives under a symlinked /var, and the policy keys its claim on the
// resolved path — an assertion built on the unresolved one would look for a
// stamp that was never written there.
func fakeInstall(t *testing.T) (dir, binary string) {
	t.Helper()

	binary = filepath.Join(t.TempDir(), "archcore")
	if err := os.WriteFile(binary, []byte(originalBinary), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(binary)
	if err != nil {
		t.Fatalf("resolving fake binary: %v", err)
	}
	return filepath.Dir(resolved), resolved
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// noNetwork returns a client that fails the test on any request. A refusal that
// still resolves the latest release has leaked a network call into a path the
// spec says contacts nothing.
func noNetwork(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected network request to %s", r.URL)
		return nil, errors.New("no network expected in this test")
	})}
}

// releaseHost serves one whole release: the /releases/latest redirect
// CheckLatest reads, plus the archive and checksums.txt Apply downloads. It
// counts every request, so "downloads nothing" is an assertion and not a hope.
type releaseHost struct {
	url  string
	hits atomic.Int64
}

func newReleaseHost(t *testing.T, tag string, payload []byte) *releaseHost {
	t.Helper()

	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	archive := createTarGz(t, map[string][]byte{"archcore": payload})
	if runtime.GOOS == "windows" {
		archive = createZip(t, map[string][]byte{"archcore.exe": payload})
	}
	checksums := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), archiveName)

	host := &releaseHost{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host.hits.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/"+tag)
			w.WriteHeader(http.StatusFound)
		case strings.HasSuffix(r.URL.Path, archiveName):
			_, _ = w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	host.url = srv.URL
	return host
}

func (h *releaseHost) client() *http.Client {
	return &http.Client{Transport: &rewriteTransport{target: h.url}}
}

func (h *releaseHost) requests() int64 { return h.hits.Load() }

// capturedEvent is one payload the endpoint accepted.
type capturedEvent struct {
	name  string
	props map[string]any
}

// telemetryRecorder is a local endpoint plus the client that posts to it. It
// records what actually crossed the wire rather than what the policy meant to
// send, so a property dropped by encoding is still visible here.
type telemetryRecorder struct {
	client *telemetry.Client

	mu     sync.Mutex
	events []capturedEvent
}

func newTelemetryRecorder(t *testing.T) *telemetryRecorder {
	t.Helper()

	rec := &telemetryRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Event      string         `json:"event"`
			Properties map[string]any `json:"properties"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "undecodable payload", http.StatusBadRequest)
			return
		}
		rec.mu.Lock()
		rec.events = append(rec.events, capturedEvent{name: payload.Event, props: payload.Properties})
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	rec.client = &telemetry.Client{
		Version:  "v1.0.0",
		Key:      "phc_test_key",
		Endpoint: srv.URL,
		StateDir: t.TempDir(),
	}
	return rec
}

func (r *telemetryRecorder) captured() []capturedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]capturedEvent(nil), r.events...)
}

// only returns the single recorded event, failing when the count is not one.
// Every terminal state of an attempt produces at most one event, so "exactly
// one" is the assertion nearly every test wants.
func (r *telemetryRecorder) only(t *testing.T) capturedEvent {
	t.Helper()
	events := r.captured()
	if len(events) != 1 {
		t.Fatalf("recorded %d event(s), want exactly 1: %v", len(events), names(events))
	}
	return events[0]
}

func (r *telemetryRecorder) assertSilent(t *testing.T) {
	t.Helper()
	if events := r.captured(); len(events) != 0 {
		t.Errorf("recorded %v, want no event at all", names(events))
	}
}

func names(events []capturedEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.name)
	}
	return out
}

// assertProp checks one property against a written-out expected value.
func assertProp(t *testing.T, ev capturedEvent, key string, want any) {
	t.Helper()
	got, ok := ev.props[key]
	if !ok {
		t.Errorf("%s carries no %q property", ev.name, key)
		return
	}
	if got != want {
		t.Errorf("%s[%q] = %v, want %v", ev.name, key, got, want)
	}
}

func assertNoProp(t *testing.T, ev capturedEvent, key string) {
	t.Helper()
	if value, ok := ev.props[key]; ok {
		t.Errorf("%s carries %q = %v, want it omitted", ev.name, key, value)
	}
}

// assertCarriesNothingPrivate fails when any property value contains one of the
// forbidden fragments. No property may carry an error message, a path, a
// directory name, a user name, a host name or repository data, and the usual way
// that breaks is a well-meant "detail" field holding err.Error().
func assertCarriesNothingPrivate(t *testing.T, ev capturedEvent, forbidden ...string) {
	t.Helper()
	for key, value := range ev.props {
		text, ok := value.(string)
		if !ok {
			continue
		}
		for _, fragment := range forbidden {
			if fragment != "" && strings.Contains(text, fragment) {
				t.Errorf("%s[%q] = %q leaks %q", ev.name, key, text, fragment)
			}
		}
	}
}

// assertNoClaim fails when a stamp exists in claimDir. A refusal before the
// claim must leave the directory untouched — a stamp written by a run that
// refused would block the next legitimate attempt for a whole window.
func assertNoClaim(t *testing.T, claimDir string) {
	t.Helper()
	entries, err := os.ReadDir(claimDir)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatalf("reading claim directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("claim directory holds %d entr(ies), want none: a refusal takes no claim", len(entries))
	}
}

// assertClaimHeld fails when the claim for target is gone from claimDir.
//
// The claim is never released, on any exit path. An attempt that failed halfway
// — or that died outright — must not repeat on the next process that starts, so
// the window is the only thing that ends it. A tidy-looking release on a failure
// branch turns a machine that fails at 3 a.m. into a machine that retries the
// same download on every MCP server start until morning —
// unattended-update.spec, Failure Behavior 5.
func assertClaimHeld(t *testing.T, claimDir, target string) {
	t.Helper()
	if _, err := os.Stat(stamp.PathFor(claimDir, target)); err != nil {
		t.Errorf("the claim for the binary is gone after the attempt: %v", err)
	}
}

// seedCache writes a freshness-cache entry. An empty latest writes the failure
// stamp, exactly as a failed lookup does.
func seedCache(t *testing.T, path, latest string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(latest+"\n"), 0o644); err != nil {
		t.Fatalf("seeding cache: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.Base(path), err)
	}
	return string(data)
}

// requireOwnedPermissions skips a test that depends on a directory mode the
// running user cannot be denied. Root ignores the permission bits, and Windows
// does not model them the way chmod does.
func requireOwnedPermissions(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not modeled by chmod on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses the permission bits this test denies")
	}
}

// denyWrites makes dir read-only for the rest of the test and restores it
// afterwards, so t.TempDir's own cleanup can still remove the tree.
func denyWrites(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("denying writes on %s: %v", filepath.Base(dir), err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

// requiresExec skips a test that stages a real executable and runs it. The
// health probe starts a child process, which needs a POSIX shell.
func requiresExec(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to stage a probe-able binary")
	}
}

// --- the constants the policy is defined by ----------------------------------

// The ceiling is written out here rather than compared against itself, because
// every other test in this file would stay green at any value. It is the budget
// a caller lends the policy: the MCP server's background slot must come back,
// and 120 s is what unattended-update.spec commits to.
func TestUnattendedCeiling_IsTheDocumentedBudget(t *testing.T) {
	if unattendedCeiling != 120*time.Second {
		t.Errorf("unattendedCeiling = %s, want 120s", unattendedCeiling)
	}
}

// The release marker is absent unless the release workflow injects one, and any
// value it injects counts.
//
// The first subtest is the one that guards the fork condition, and it reads the
// package variable rather than any code path: `go build`, `go install`, a fork
// and a CI build are inert by construction only while the default is empty. A
// default with any value in it would arm this policy in every build in the
// world, which is the takeover the condition exists to prevent.
//
// The rest pin presence, not content. The workflow decides what to put there,
// and a policy that parsed the value would turn a formatting change in a YAML
// file into a silent loss of unattended updates for a whole release —
// unattended-update.spec §3.
func TestIsOfficialBuild(t *testing.T) {
	t.Run("a build nobody marked", func(t *testing.T) {
		if officialBuild != "" {
			t.Errorf("officialBuild = %q by default, want it empty until a release injects one", officialBuild)
		}
		if isOfficialBuild() {
			t.Error("isOfficialBuild() = true with no marker injected")
		}
	})

	// "0" and "false" are in the set deliberately: they are the values a reader
	// is most tempted to treat as "not a release".
	for _, marker := range []string{"1", "0", "false", "release", "v1.2.3"} {
		t.Run("marked with "+marker, func(t *testing.T) {
			withOfficialBuild(t, marker)
			if !isOfficialBuild() {
				t.Errorf("isOfficialBuild() = false for marker %q, want presence alone to count", marker)
			}
		})
	}
}

// The CI set is shared with install.sh's is_ci(). A machine the installer calls
// CI and the policy does not is a machine that self-updates inside a container
// image build.
func TestUnattendedCIVars_AreTheInstallerSet(t *testing.T) {
	want := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "JENKINS_URL", "TEAMCITY_VERSION"}
	if len(telemetry.CIVars) != len(want) {
		t.Fatalf("telemetry.CIVars = %v, want %v", telemetry.CIVars, want)
	}
	for i, name := range want {
		if telemetry.CIVars[i] != name {
			t.Errorf("telemetry.CIVars[%d] = %q, want %q", i, telemetry.CIVars[i], name)
		}
	}
}

// FailureStage reads the stage off the typed error and never off the message.
// The default is `check`, because the only untagged error an attempt can reach
// telemetry with came from resolving the latest tag.
func TestFailureStage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"tagged check", stageErr(StageCheck, errors.New("boom")), "check"},
		{"tagged download", stageErr(StageDownload, errors.New("boom")), "download"},
		{"tagged checksum", stageErr(StageChecksum, errors.New("boom")), "checksum"},
		{"tagged extract", stageErr(StageExtract, errors.New("boom")), "extract"},
		{"tagged replace", stageErr(StageReplace, errors.New("boom")), "replace"},
		{"untagged defaults to check", errors.New("boom"), "check"},
		{"wrapped tag is still found", fmt.Errorf("outer: %w", stageErr(StageExtract, errors.New("boom"))), "extract"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FailureStage(tt.err); got != tt.want {
				t.Errorf("FailureStage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- the silent refusals -----------------------------------------------------

// A binary this project did not build refuses before anything else: no claim
// file, no network, no event. The claim assertion is what pins the ordering —
// every later condition runs after the claim is taken.
func TestRunUnattended_UnofficialBuildRefusesFirst(t *testing.T) {
	unattendedEnv(t)
	withOfficialBuild(t, "")

	rec := newTelemetryRecorder(t)
	_, binary := fakeInstall(t)
	claimDir := t.TempDir()

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: filepath.Join(t.TempDir(), "last-update-check"),
		ClaimDir:  claimDir,
	})

	if result.Updated {
		t.Error("a build without the release marker replaced itself")
	}
	assertNoClaim(t, claimDir)
	rec.assertSilent(t)
	if got := readFile(t, binary); got != originalBinary {
		t.Errorf("binary content = %q, want %q untouched", got, originalBinary)
	}
}

// A development build refuses and takes no claim. Without this condition
// NeedsUpdate's "dev is always behind" would make every locally built binary
// replace itself with a release.
func TestRunUnattended_DevVersionRefuses(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	_, binary := fakeInstall(t)
	claimDir := t.TempDir()

	for _, version := range []string{"dev", "vdev"} {
		t.Run(version, func(t *testing.T) {
			result := RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater(version, binary, noNetwork(t)),
				Version:   version,
				Telemetry: rec.client,
				CachePath: filepath.Join(t.TempDir(), "last-update-check"),
				ClaimDir:  claimDir,
			})

			if result.Updated {
				t.Error("a development build replaced itself")
			}
			assertNoClaim(t, claimDir)
			rec.assertSilent(t)
		})
	}
}

// Any one CI variable refuses on its own. A CI runner is discarded minutes
// later, so an update there costs a download and buys nothing.
func TestRunUnattended_CIRefuses(t *testing.T) {
	for _, ciVar := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "JENKINS_URL", "TEAMCITY_VERSION"} {
		t.Run(ciVar, func(t *testing.T) {
			unattendedEnv(t)
			t.Setenv(ciVar, "true")

			rec := newTelemetryRecorder(t)
			_, binary := fakeInstall(t)
			claimDir := t.TempDir()

			result := RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
				Version:   "v1.0.0",
				Telemetry: rec.client,
				CachePath: filepath.Join(t.TempDir(), "last-update-check"),
				ClaimDir:  claimDir,
			})

			if result.Updated {
				t.Errorf("%s=true did not stop the policy", ciVar)
			}
			assertNoClaim(t, claimDir)
			rec.assertSilent(t)
		})
	}
}

// The conformance sentence: given a state directory that is read-only, the
// policy refuses at the claim, contacts no network and sends no event. Both
// subtests leave ClaimDir and CachePath empty, so the defaults — the paths a
// real caller gets — are what is under test.
func TestRunUnattended_UnusableStateDirectoryRefuses(t *testing.T) {
	run := func(t *testing.T) UnattendedResult {
		t.Helper()
		rec := newTelemetryRecorder(t)
		_, binary := fakeInstall(t)

		result := RunUnattended(context.Background(), UnattendedOptions{
			Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
			Version:   "v1.0.0",
			Telemetry: rec.client,
		})
		rec.assertSilent(t)
		if got := readFile(t, binary); got != originalBinary {
			t.Errorf("binary content = %q, want %q untouched", got, originalBinary)
		}
		return result
	}

	t.Run("read-only state directory", func(t *testing.T) {
		requireOwnedPermissions(t)
		unattendedEnv(t)

		root := t.TempDir()
		denyWrites(t, root)
		t.Setenv("XDG_STATE_HOME", root)

		if run(t).Updated {
			t.Error("the policy replaced the binary with no claim to stand on")
		}
	})

	t.Run("no state directory resolves at all", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("os.UserHomeDir does not read HOME on windows")
		}
		unattendedEnv(t)
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", "")

		if run(t).Updated {
			t.Error("the policy replaced the binary with no state directory at all")
		}
	})
}

// A latest tag that does not parse as semver refuses with no event. NeedsUpdate
// once fell back to string inequality here, which in an unattended path turns an
// odd tag into a replacement or a downgrade.
func TestRunUnattended_UnparseableTagRefusesSilently(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "nightly", []byte(healthyPayload))
	_, binary := fakeInstall(t)
	cachePath := filepath.Join(t.TempDir(), "last-update-check")

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, host.client()),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: cachePath,
		ClaimDir:  t.TempDir(),
	})

	if result.Updated {
		t.Error("an unparseable tag was installed")
	}
	rec.assertSilent(t)
	if got := readFile(t, binary); got != originalBinary {
		t.Errorf("binary content = %q, want %q untouched", got, originalBinary)
	}
	// One request — the version lookup — and nothing downloaded after it.
	if got := host.requests(); got != 1 {
		t.Errorf("release host saw %d request(s), want exactly the version lookup", got)
	}
	// The lookup succeeded, so the cache still holds its result: refusing to act
	// on a tag is not a reason to make the next caller pay for the lookup again.
	if got := strings.TrimSpace(readFile(t, cachePath)); got != "nightly" {
		t.Errorf("cache = %q, want the resolved tag %q", got, "nightly")
	}
}

// No refusal may emit cli_update_failed. A refusal means a condition was not
// met; a failure means an attempted step did not complete, and conflating them
// makes the failure series unreadable.
func TestRunUnattended_NoRefusalReportsAFailure(t *testing.T) {
	tests := []struct {
		name    string
		marker  string
		version string
		ciVar   string
		tag     string
	}{
		{name: "unofficial build", marker: "", version: "v1.0.0", tag: "v2.0.0"},
		{name: "development build", marker: "release", version: "dev", tag: "v2.0.0"},
		{name: "ci runner", marker: "release", version: "v1.0.0", ciVar: "GITHUB_ACTIONS", tag: "v2.0.0"},
		{name: "unparseable tag", marker: "release", version: "v1.0.0", tag: "nightly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unattendedEnv(t)
			withOfficialBuild(t, tt.marker)
			if tt.ciVar != "" {
				t.Setenv(tt.ciVar, "1")
			}

			rec := newTelemetryRecorder(t)
			host := newReleaseHost(t, tt.tag, []byte(healthyPayload))
			_, binary := fakeInstall(t)

			RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater(tt.version, binary, host.client()),
				Version:   tt.version,
				Telemetry: rec.client,
				CachePath: filepath.Join(t.TempDir(), "last-update-check"),
				ClaimDir:  t.TempDir(),
			})

			for _, ev := range rec.captured() {
				t.Errorf("a refusal emitted %q, want no event", ev.name)
			}
		})
	}
}

// --- the claim ---------------------------------------------------------------

// Two callers that start at the same moment produce exactly one attempt. The
// cache is pre-seeded with a fresh newer version, so nothing but the claim
// separates them: without it both would stage over the same file at once.
func TestRunUnattended_ConcurrentCallersMakeOneAttempt(t *testing.T) {
	requiresExec(t)
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "v2.0.0", []byte(healthyPayload))
	_, binary := fakeInstall(t)

	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")
	seedCache(t, cachePath, "v2.0.0")

	const callers = 2
	results := make([]UnattendedResult, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater("v1.0.0", binary, host.client()),
				Version:   "v1.0.0",
				Telemetry: rec.client,
				CachePath: cachePath,
				ClaimDir:  claimDir,
			})
		}()
	}
	wg.Wait()

	updated := 0
	for _, r := range results {
		if r.Updated {
			updated++
		}
	}
	if updated != 1 {
		t.Errorf("%d of %d callers replaced the binary, want exactly 1", updated, callers)
	}

	events := rec.captured()
	if len(events) != 1 || events[0].name != "cli_updated" {
		t.Errorf("recorded %v, want exactly one cli_updated", names(events))
	}
	if got := readFile(t, binary); got != healthyPayload {
		t.Errorf("binary content = %q, want the downloaded payload", got)
	}
}

// The claim window is the cache TTL: one constant governs both. A stamp just
// inside the window refuses; the same stamp just outside it lets the next
// attempt through.
//
// Both ages are derived from CacheTTL and sit one minute from it, which is what
// makes this an equality rather than a range. A window declared as its own
// constant, or as the TTL plus a grace period, is what the spec forbids and what
// a looser margin would wave through: the two would then drift apart release by
// release with every test still green — unattended-update.spec.
func TestRunUnattended_ClaimWindowIsTheCacheTTL(t *testing.T) {
	tests := []struct {
		name      string
		stampAge  time.Duration
		wantHits  int64
		wantSkip  bool
		wantEvent string
	}{
		{
			name:      "inside the window the claim is held",
			stampAge:  CacheTTL - time.Minute,
			wantHits:  0,
			wantSkip:  true,
			wantEvent: "",
		},
		{
			name:      "past the window the claim is reclaimed",
			stampAge:  CacheTTL + time.Minute,
			wantHits:  1,
			wantSkip:  false,
			wantEvent: "cli_update_skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unattendedEnv(t)

			rec := newTelemetryRecorder(t)
			// The running version and the served tag are equal, so a caller that
			// gets past the claim reports skipped(current) and one that does not
			// reports nothing.
			host := newReleaseHost(t, "v1.0.0", []byte(healthyPayload))
			_, binary := fakeInstall(t)

			claimDir := t.TempDir()
			stampPath := stamp.PathFor(claimDir, binary)
			if err := os.WriteFile(stampPath, nil, 0o644); err != nil {
				t.Fatalf("seeding the claim: %v", err)
			}
			ageFile(t, stampPath, tt.stampAge)

			RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater("v1.0.0", binary, host.client()),
				Version:   "v1.0.0",
				Telemetry: rec.client,
				CachePath: filepath.Join(t.TempDir(), "last-update-check"),
				ClaimDir:  claimDir,
			})

			if got := host.requests(); got != tt.wantHits {
				t.Errorf("release host saw %d request(s), want %d", got, tt.wantHits)
			}
			if tt.wantSkip {
				rec.assertSilent(t)
				return
			}
			if ev := rec.only(t); ev.name != tt.wantEvent {
				t.Errorf("recorded %q, want %q", ev.name, tt.wantEvent)
			}
		})
	}
}

// The claim is keyed by the resolved binary path, not by the path the caller
// happened to be invoked through. An install reached through
// /usr/local/bin/archcore and the same install reached through its real path
// would otherwise take two separate claims and both replace one file, which is
// the outcome exclusivity exists to forbid — unattended-update.spec §6.
func TestRunUnattended_ClaimKeyIsTheResolvedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink needs a privilege this test cannot assume on windows")
	}
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	_, binary := fakeInstall(t)

	link := filepath.Join(t.TempDir(), "archcore")
	if err := os.Symlink(binary, link); err != nil {
		t.Fatalf("linking to the install: %v", err)
	}

	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")
	// Already current, so the run stops at the comparison: the claim is taken,
	// nothing is downloaded, and no network is touched.
	seedCache(t, cachePath, "v1.0.0")

	RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", link, noNetwork(t)),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: cachePath,
		ClaimDir:  claimDir,
	})

	// The skip proves the run got past the claim, so the stamp below is this
	// attempt's own and not the absence of one.
	if ev := rec.only(t); ev.name != "cli_update_skipped" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_skipped")
	}
	assertClaimHeld(t, claimDir, binary)
	if _, err := os.Stat(stamp.PathFor(claimDir, link)); err == nil {
		t.Error("the claim is keyed by the link, so a second path to one install would take a second claim")
	}
}

// --- the cache ---------------------------------------------------------------

// A fresh failure stamp is stale to this policy: it records that the network was
// unreachable a moment ago and nothing at all about the release. Reading it as
// an answer would report `current` for a machine that never compared anything.
func TestRunUnattended_FreshFailureStampLeadsToALookup(t *testing.T) {
	requiresExec(t)
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "v2.0.0", []byte(healthyPayload))
	_, binary := fakeInstall(t)

	cachePath := filepath.Join(t.TempDir(), "last-update-check")
	seedCache(t, cachePath, "") // the failure stamp, written moments ago

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, host.client()),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: cachePath,
		ClaimDir:  t.TempDir(),
	})

	if host.requests() == 0 {
		t.Fatal("a fresh failure stamp suppressed the lookup entirely")
	}
	if !result.Updated || result.NewVersion != "v2.0.0" {
		t.Errorf("result = %+v, want an update to v2.0.0", result)
	}
	if ev := rec.only(t); ev.name != "cli_updated" {
		t.Errorf("recorded %q, want %q — a failure stamp must never read as current", ev.name, "cli_updated")
	}
	if got := strings.TrimSpace(readFile(t, cachePath)); got != "v2.0.0" {
		t.Errorf("cache = %q, want it refreshed to %q", got, "v2.0.0")
	}
}

// A fresh cache holding the running version reports skipped(current) once, and
// the claim keeps the second caller in the same window silent. That pairing is
// what makes the series count machines rather than MCP server starts.
func TestRunUnattended_CurrentVersionSkipsOncePerWindow(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	_, binary := fakeInstall(t)
	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")
	seedCache(t, cachePath, "v1.0.0")

	call := func() UnattendedResult {
		return RunUnattended(context.Background(), UnattendedOptions{
			// A fresh cache answers the question, so any request at all is a
			// network call this path is not allowed to make.
			Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
			Version:   "v1.0.0",
			Telemetry: rec.client,
			CachePath: cachePath,
			ClaimDir:  claimDir,
		})
	}

	if call().Updated {
		t.Error("the policy replaced a binary that was already current")
	}

	ev := rec.only(t)
	if ev.name != "cli_update_skipped" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_skipped")
	}
	assertProp(t, ev, "reason", "current")
	assertProp(t, ev, "trigger", "auto")

	// The second caller falls inside the same claim window.
	call()
	if events := rec.captured(); len(events) != 1 {
		t.Errorf("recorded %v across two calls, want exactly one skip per window", names(events))
	}
}

// --- the reportable outcomes -------------------------------------------------

// An install directory the user cannot write reports skipped(not_writable) and
// downloads nothing. A root-owned install directory is the supported way to opt
// a machine out of self-update, so this runs on every attempt and must stay
// cheap.
//
// The second call is the ordering assertion. Both reportable skips have to be
// evaluated after the claim, so one machine emits at most one
// cli_update_skipped per window however many callers start on it; a
// not_writable check hoisted above the claim reads identically on a single run
// and turns the series into a count of MCP server starts on every root-owned
// install — unattended-update.spec, and cli-update-telemetry.spec.
func TestRunUnattended_UnwritableTargetDirectorySkips(t *testing.T) {
	requireOwnedPermissions(t)
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	// Any request is a failure here: the cache already answers the version
	// question, and the write check runs before the download.
	dir, binary := fakeInstall(t)
	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")
	seedCache(t, cachePath, "v2.0.0")
	denyWrites(t, dir)

	call := func() UnattendedResult {
		return RunUnattended(context.Background(), UnattendedOptions{
			Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
			Version:   "v1.0.0",
			Telemetry: rec.client,
			CachePath: cachePath,
			ClaimDir:  claimDir,
		})
	}

	result := call()

	if result.Updated {
		t.Error("the policy reported an update into a directory it cannot write")
	}

	ev := rec.only(t)
	if ev.name != "cli_update_skipped" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_skipped")
	}
	assertProp(t, ev, "reason", "not_writable")
	assertProp(t, ev, "trigger", "auto")
	assertCarriesNothingPrivate(t, ev, dir, binary, filepath.Base(dir))

	if got := readFile(t, binary); got != originalBinary {
		t.Errorf("binary content = %q, want %q untouched", got, originalBinary)
	}
	// The probe file must not survive a failed write check either.
	if left := stagedFiles(t, dir); len(left) != 0 {
		t.Errorf("staged files left behind: %v", left)
	}
	assertClaimHeld(t, claimDir, binary)

	// The second caller falls inside the same claim window and stays silent.
	call()
	if events := rec.captured(); len(events) != 1 {
		t.Errorf("recorded %v across two calls, want exactly one skip per window", names(events))
	}
}

// A failed version lookup is a failure, not a refusal: it reports the check
// stage, stamps the negative cache, and carries no to_version because no tag
// ever resolved.
func TestRunUnattended_LookupFailureReportsCheckStage(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream is having a day", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, binary := fakeInstall(t)
	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, &http.Client{Transport: &rewriteTransport{target: srv.URL}}),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: cachePath,
		ClaimDir:  claimDir,
	})

	if result.Updated {
		t.Error("a failed lookup still reported an update")
	}
	assertClaimHeld(t, claimDir, binary)

	ev := rec.only(t)
	if ev.name != "cli_update_failed" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_failed")
	}
	assertProp(t, ev, "stage", "check")
	assertProp(t, ev, "trigger", "auto")
	assertNoProp(t, ev, "to_version")
	// The lookup error names a URL and a status; none of it may travel.
	assertCarriesNothingPrivate(t, ev, srv.URL, "status", "500", binary)

	// The failure stamp is what keeps the next caller off a network that just
	// failed — empty content, written now.
	if got := strings.TrimSpace(readFile(t, cachePath)); got != "" {
		t.Errorf("cache = %q, want the failure stamp", got)
	}
}

// The write check answers by creating the very file the staging step will
// write, and it must leave nothing behind. A leftover would survive until some
// later attempt's sweep, and between the two a user reading their install
// directory finds a file with no owner.
func TestTargetDirWritable(t *testing.T) {
	t.Run("writable directory, and the probe is cleaned up", func(t *testing.T) {
		dir, binary := fakeInstall(t)

		if !targetDirWritable(binary) {
			t.Fatal("targetDirWritable() = false for a writable directory")
		}
		if left := stagedFiles(t, dir); len(left) != 0 {
			t.Errorf("the write check left %v behind", left)
		}
		if got := readFile(t, binary); got != originalBinary {
			t.Errorf("binary content = %q, want %q untouched by the check", got, originalBinary)
		}
	})

	t.Run("read-only directory", func(t *testing.T) {
		requireOwnedPermissions(t)
		dir, binary := fakeInstall(t)
		denyWrites(t, dir)

		if targetDirWritable(binary) {
			t.Error("targetDirWritable() = true for a directory that denies writes")
		}
	})
}

// A download that fails after the write check reports the download stage and
// strands nothing in the install directory. This is the one path where no later
// sweep runs inside the same attempt, so the probe file's removal is load-
// bearing here and nowhere else.
func TestRunUnattended_DownloadFailureReportsDownloadStage(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	// The tag resolves and the assets do not: a release whose upload is still
	// in flight looks exactly like this.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v2.0.0")
			w.WriteHeader(http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	dir, binary := fakeInstall(t)
	claimDir := t.TempDir()

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, &http.Client{Transport: &rewriteTransport{target: srv.URL}}),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: filepath.Join(t.TempDir(), "last-update-check"),
		ClaimDir:  claimDir,
	})

	if result.Updated {
		t.Error("a failed download still reported an update")
	}
	assertClaimHeld(t, claimDir, binary)
	if got := readFile(t, binary); got != originalBinary {
		t.Errorf("binary content = %q, want %q untouched", got, originalBinary)
	}
	if left := stagedFiles(t, dir); len(left) != 0 {
		t.Errorf("the attempt stranded %v in the install directory", left)
	}

	ev := rec.only(t)
	if ev.name != "cli_update_failed" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_failed")
	}
	assertProp(t, ev, "stage", "download")
	assertProp(t, ev, "trigger", "auto")
	assertProp(t, ev, "from_version", "v1.0.0")
	assertProp(t, ev, "to_version", "v2.0.0")
	assertCarriesNothingPrivate(t, ev, srv.URL, dir, binary, "404")
}

// A staged binary that cannot report its own version is abandoned before the
// rename: the event names the replace stage and the installed binary is
// byte-identical to what it was.
func TestRunUnattended_ProbeFailureReportsReplaceStage(t *testing.T) {
	requiresExec(t)
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "v2.0.0", []byte("#!/bin/sh\nexit 3\n"))
	dir, binary := fakeInstall(t)
	claimDir := t.TempDir()

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, host.client()),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: filepath.Join(t.TempDir(), "last-update-check"),
		ClaimDir:  claimDir,
	})

	if result.Updated {
		t.Error("a binary that fails its health probe was reported as installed")
	}
	assertClaimHeld(t, claimDir, binary)
	if got := readFile(t, binary); got != originalBinary {
		t.Errorf("binary content = %q, want %q byte-identical", got, originalBinary)
	}
	if left := stagedFiles(t, dir); len(left) != 0 {
		t.Errorf("staged files left after an abandoned attempt: %v", left)
	}

	ev := rec.only(t)
	if ev.name != "cli_update_failed" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_failed")
	}
	assertProp(t, ev, "stage", "replace")
	assertProp(t, ev, "trigger", "auto")
	assertProp(t, ev, "from_version", "v1.0.0")
	assertProp(t, ev, "to_version", "v2.0.0")
	// The probe error names the staged file; the event must not.
	assertCarriesNothingPrivate(t, ev, dir, binary, ".tmp.", "--version")
}

// The successful path end to end: the binary is replaced, one cli_updated
// carries both versions, the cache is refreshed, and the claim stays on disk. A
// released claim would let the next process that starts repeat the attempt.
func TestRunUnattended_SuccessReplacesAndReports(t *testing.T) {
	requiresExec(t)
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "v2.0.0", []byte(healthyPayload))
	dir, binary := fakeInstall(t)

	claimDir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "last-update-check")

	caller := policyUpdater("v1.0.0", binary, host.client())

	result := RunUnattended(context.Background(), UnattendedOptions{
		Updater:   caller,
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: cachePath,
		ClaimDir:  claimDir,
	})

	if !result.Updated || result.NewVersion != "v2.0.0" {
		t.Fatalf("result = %+v, want an update to v2.0.0", result)
	}
	if got := readFile(t, binary); got != healthyPayload {
		t.Errorf("binary content = %q, want the downloaded payload", got)
	}
	if left := stagedFiles(t, dir); len(left) != 0 {
		t.Errorf("staged files left after a successful commit: %v", left)
	}

	ev := rec.only(t)
	if ev.name != "cli_updated" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_updated")
	}
	assertProp(t, ev, "trigger", "auto")
	assertProp(t, ev, "from_version", "v1.0.0")
	assertProp(t, ev, "to_version", "v2.0.0")
	assertCarriesNothingPrivate(t, ev, dir, binary, host.url)

	if got := strings.TrimSpace(readFile(t, cachePath)); got != "v2.0.0" {
		t.Errorf("cache = %q, want it refreshed to %q", got, "v2.0.0")
	}
	if _, err := os.Stat(stamp.PathFor(claimDir, binary)); err != nil {
		t.Errorf("claim stamp is gone after a completed attempt: %v", err)
	}
	// The probe is installed on a copy. A caller whose Updater came back with
	// one would carry it into the typed `archcore update` path, which is
	// specified to run without a probe.
	if caller.PreCommitProbe != nil {
		t.Error("the policy installed the health probe on the caller's Updater")
	}
}

// TestRunUnattended_TelemetryOptOutLeavesTheUpdateWorking pins the release's
// most easily-broken promise: the two telemetry opt-outs stop events and do not
// stop updates. There is deliberately no variable that disables unattended
// update, and the honest way to say so on the privacy page depends on this
// staying true.
//
// It is the pair of assertions that makes the test worth having. Replacement
// alone would pass on a build where the opt-out never worked; silence alone
// would pass on one where the opt-out disabled the whole path. A future
// refactor that folds the two decisions together — reading DO_NOT_TRACK
// anywhere near the policy's conditions — fails here whichever way it leans.
//
// unattendedEnv sets both variables to "" for isolation; each subtest then sets
// exactly one truthy, so a guard that only ever reads the other is not covered
// by its sibling — unattended-update.spec, cli-update-telemetry.spec.
func TestRunUnattended_TelemetryOptOutLeavesTheUpdateWorking(t *testing.T) {
	for _, optOut := range []string{"DO_NOT_TRACK", "ARCHCORE_TELEMETRY_OPTOUT"} {
		t.Run(optOut, func(t *testing.T) {
			requiresExec(t)
			unattendedEnv(t)
			t.Setenv(optOut, "1")

			rec := newTelemetryRecorder(t)
			host := newReleaseHost(t, "v2.0.0", []byte(healthyPayload))
			_, binary := fakeInstall(t)

			result := RunUnattended(context.Background(), UnattendedOptions{
				Updater:   policyUpdater("v1.0.0", binary, host.client()),
				Version:   "v1.0.0",
				Telemetry: rec.client,
				CachePath: filepath.Join(t.TempDir(), "last-update-check"),
				ClaimDir:  t.TempDir(),
			})

			if !result.Updated || result.NewVersion != "v2.0.0" {
				t.Fatalf("%s=1 blocked the update: result = %+v", optOut, result)
			}
			if got := readFile(t, binary); got != healthyPayload {
				t.Errorf("binary content = %q, want the downloaded payload", got)
			}
			rec.assertSilent(t)
		})
	}
}

// TestRunUnattended_TelemetryOutlivesTheCeiling covers the send site's use of
// context.WithoutCancel. The context here is already cancelled, which is what a
// run that hit the 120 s ceiling looks like from the send site: the lookup fails
// immediately, and the cli_update_failed reporting it must still be delivered.
// Without context.WithoutCancel the failure disappears in the same moment it
// happens, and the failure series reads as a machine that never tried.
func TestRunUnattended_TelemetryOutlivesTheCeiling(t *testing.T) {
	unattendedEnv(t)

	rec := newTelemetryRecorder(t)
	host := newReleaseHost(t, "v2.0.0", []byte(healthyPayload))
	_, binary := fakeInstall(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	RunUnattended(ctx, UnattendedOptions{
		Updater:   policyUpdater("v1.0.0", binary, host.client()),
		Version:   "v1.0.0",
		Telemetry: rec.client,
		CachePath: filepath.Join(t.TempDir(), "last-update-check"),
		ClaimDir:  t.TempDir(),
	})

	ev := rec.only(t)
	if ev.name != "cli_update_failed" {
		t.Fatalf("recorded %q, want %q", ev.name, "cli_update_failed")
	}
	assertProp(t, ev, "stage", "check")
	assertProp(t, ev, "trigger", "auto")
	assertCarriesNothingPrivate(t, ev, "context canceled", binary)
}

// A nil telemetry client is the shape an inert build takes, and a nil Updater is
// a caller that cannot replace anything. Neither may fault the host: this policy
// runs inside a background goroutine of a long-lived server.
func TestRunUnattended_NilDependenciesDoNotPanic(t *testing.T) {
	unattendedEnv(t)

	t.Run("nil telemetry client", func(t *testing.T) {
		_, binary := fakeInstall(t)
		cachePath := filepath.Join(t.TempDir(), "last-update-check")
		seedCache(t, cachePath, "v1.0.0")

		result := RunUnattended(context.Background(), UnattendedOptions{
			Updater:   policyUpdater("v1.0.0", binary, noNetwork(t)),
			Version:   "v1.0.0",
			CachePath: cachePath,
			ClaimDir:  t.TempDir(),
		})
		if result.Updated {
			t.Error("a current version was reported as updated")
		}
	})

	t.Run("nil updater", func(t *testing.T) {
		claimDir := t.TempDir()
		result := RunUnattended(context.Background(), UnattendedOptions{
			Version:   "v1.0.0",
			CachePath: filepath.Join(t.TempDir(), "last-update-check"),
			ClaimDir:  claimDir,
		})
		if result.Updated {
			t.Error("a caller with no updater reported an update")
		}
		assertNoClaim(t, claimDir)
	})
}
