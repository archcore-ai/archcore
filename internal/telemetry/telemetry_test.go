package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const testKey = "phc_test_key"

// capturedEvent is the PostHog payload shape install.sh posts. Decoding into it
// is the assertion that the Go client did not drift from the installer.
type capturedEvent struct {
	APIKey     string         `json:"api_key"`
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Properties map[string]any `json:"properties"`
}

// received is one request as the endpoint saw it.
type received struct {
	method      string
	path        string
	contentType string
	event       capturedEvent
}

// recorder is an endpoint that records what it received. The handler runs on
// the server's goroutine, so every field crossing to the test goroutine is held
// under the mutex.
type recorder struct {
	server *httptest.Server

	mu     sync.Mutex
	status int
	calls  int
	last   received
}

func newRecorder(t *testing.T) *recorder {
	t.Helper()
	r := &recorder{status: http.StatusOK}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got := received{
			method:      req.Method,
			path:        req.URL.Path,
			contentType: req.Header.Get("Content-Type"),
		}
		if err := json.NewDecoder(req.Body).Decode(&got.event); err != nil {
			t.Errorf("decoding the payload: %v", err)
		}

		r.mu.Lock()
		r.calls++
		r.last = got
		status := r.status
		r.mu.Unlock()

		w.WriteHeader(status)
	}))
	t.Cleanup(r.server.Close)
	return r
}

// setStatus fixes the status the endpoint answers with.
func (r *recorder) setStatus(status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// received returns the last request. It is only read after Capture has
// returned, so there is no in-flight request to miss.
func (r *recorder) received() received {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last
}

// endpoint mirrors the production path, so a test can assert the client posts
// to /i/v0/e/ rather than to the server root.
func (r *recorder) endpoint() string { return r.server.URL + "/i/v0/e/" }

// client builds a Client pointed at this recorder. The five-field literal was
// written out at every call site, and StateDir is the one that must never be
// forgotten: a client without it falls through to the process state directory,
// which TestMain shares across the whole package — so one test's install-id
// would decide another's distinct_id.
func (r *recorder) client(version, key, stateDir string) *Client {
	return &Client{
		Version:    version,
		Key:        key,
		Endpoint:   r.endpoint(),
		HTTPClient: r.server.Client(),
		StateDir:   stateDir,
	}
}

// frozenCIVars is the CI set install.sh tests, spelled out rather than read
// from CIVars. A test that cleared whatever the implementation happens to list
// would keep passing after a variable was dropped from the set, and would then
// be reporting `ci=false` from a runner that sets it.
var frozenCIVars = []string{
	"CI",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"BUILDKITE",
	"JENKINS_URL",
	"TEAMCITY_VERSION",
}

// clearEnv neutralizes every environment variable this package reads. CI
// runners set CI=true, which would otherwise flip the `ci` property and, for a
// guard test, hide the property under test behind an unrelated value.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("ARCHCORE_TELEMETRY_OPTOUT", "")
	for _, name := range frozenCIVars {
		t.Setenv(name, "")
	}
}

// TestFrozenWireIdentifiers pins the values that leave the process. Every one of
// them is shared with something outside this package — the PostHog project, the
// event map in archcore-ai/landing, install.sh and install.ps1 — so a rename
// here is a silent break there rather than a compile error. Asserting them
// against the package's own constants would pin nothing at all.
func TestFrozenWireIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"capture endpoint", defaultEndpoint, "https://ph.archcore.ai/i/v0/e/"},
		{"$lib", libName, "archcore-cli"},
		{"source", sourceCLI, "cli"},
		{"project key prefix", keyPrefix, "phc_"},
		{"identifier file name", installIDFile, "install-id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("= %q, want %q", tt.got, tt.want)
			}
		})
	}

	if len(CIVars) != len(frozenCIVars) {
		t.Fatalf("CIVars = %v, want the installer's set %v", CIVars, frozenCIVars)
	}
	for i, want := range frozenCIVars {
		if CIVars[i] != want {
			t.Errorf("CIVars[%d] = %q, want %q", i, CIVars[i], want)
		}
	}
}

// TestDefaultTransportHonorsProxyEnvironment: install.sh reaches the endpoint
// through curl, which reads http_proxy/https_proxy/no_proxy. A machine that
// egresses only through a proxy must not go silent when the sender changes from
// curl to Go — that would drop every event from a whole class of machines while
// the installer kept reporting from them, and add a connect stall to each
// update on top.
//
// A nil check alone is not enough: a non-nil Proxy that resolves nothing passes
// it and still leaves every proxied machine unable to reach the endpoint, which
// is the whole failure. The resolver's identity is checked too.
func TestDefaultTransportHonorsProxyEnvironment(t *testing.T) {
	transport, ok := defaultHTTPClient().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("default transport is %T, want *http.Transport", defaultHTTPClient().Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("the default transport resolves no proxy; a proxied machine cannot reach the endpoint")
	}

	// Identity, not behavior: http.ProxyFromEnvironment reads the environment
	// once per process and caches the answer, so a t.Setenv here would be
	// ignored on any run where something resolved a proxy first — the test
	// would pass or fail on test ordering. Comparing the function pointer
	// against the standard resolver settles the same question deterministically,
	// and a hand-rolled Proxy that answered nil would fail it.
	want := reflect.ValueOf(http.ProxyFromEnvironment).Pointer()
	if got := reflect.ValueOf(transport.Proxy).Pointer(); got != want {
		t.Error("the default transport's Proxy is not http.ProxyFromEnvironment; " +
			"a resolver that ignores http_proxy/https_proxy/no_proxy silences every proxied machine")
	}
}

// assertNoStateWritten pins the property that makes an opt-out honest: a
// machine that opted out leaves no file behind.
func assertNoStateWritten(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("the state directory holds %v, want it untouched", names)
	}
}

func TestCapture_Guards(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		doNotTrack string
		optOut     string
		wantSent   bool
	}{
		{name: "no key: an unofficial build is inert", key: "", wantSent: false},
		{name: "placeholder key is not a project key", key: "__POSTHOG_KEY__", wantSent: false},
		{name: "DO_NOT_TRACK=1 opts out", key: testKey, doNotTrack: "1", wantSent: false},
		{name: "DO_NOT_TRACK=true opts out", key: testKey, doNotTrack: "true", wantSent: false},
		{name: "DO_NOT_TRACK=0 does not opt out", key: testKey, doNotTrack: "0", wantSent: true},
		{name: "ARCHCORE_TELEMETRY_OPTOUT=1 opts out", key: testKey, optOut: "1", wantSent: false},
		{name: "ARCHCORE_TELEMETRY_OPTOUT=yes opts out", key: testKey, optOut: "yes", wantSent: false},
		{name: "ARCHCORE_TELEMETRY_OPTOUT=0 does not opt out", key: testKey, optOut: "0", wantSent: true},
		{name: "key and no opt-out sends", key: testKey, wantSent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DO_NOT_TRACK", tt.doNotTrack)
			t.Setenv("ARCHCORE_TELEMETRY_OPTOUT", tt.optOut)

			rec := newRecorder(t)
			dir := t.TempDir()
			c := rec.client("1.2.3", tt.key, dir)

			got := c.Capture(context.Background(), "cli_updated", map[string]any{"trigger": "manual"})

			if got != tt.wantSent {
				t.Errorf("Capture() = %v, want %v", got, tt.wantSent)
			}
			if n := rec.count(); (n > 0) != tt.wantSent {
				t.Errorf("endpoint received %d requests, want sent=%v", n, tt.wantSent)
			}
			if !tt.wantSent {
				assertNoStateWritten(t, dir)
			}
		})
	}
}

func TestCapture_Payload(t *testing.T) {
	tests := []struct {
		name  string
		ciEnv string
		props map[string]any
		want  map[string]any
	}{
		{
			name:  "common properties",
			props: map[string]any{"trigger": "manual", "from_version": "1.0.0", "to_version": "1.1.0"},
			want: map[string]any{
				"$lib":         "archcore-cli",
				"$lib_version": "1.0.0",
				"source":       "cli",
				"ci":           false,
				"trigger":      "manual",
				"from_version": "1.0.0",
				"to_version":   "1.1.0",
			},
		},
		{
			name:  "a CI variable sets ci",
			ciEnv: "GITHUB_ACTIONS",
			props: map[string]any{"trigger": "auto"},
			want: map[string]any{
				"ci":      true,
				"trigger": "auto",
			},
		},
		{
			name:  "a caller property overrides a common one",
			props: map[string]any{"trigger": "manual", "source": "mcp"},
			want: map[string]any{
				"source":  "mcp",
				"trigger": "manual",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			if tt.ciEnv != "" {
				t.Setenv(tt.ciEnv, "true")
			}

			rec := newRecorder(t)
			c := rec.client("1.0.0", testKey, t.TempDir())

			if !c.Capture(context.Background(), "cli_updated", tt.props) {
				t.Fatal("Capture() = false, want true")
			}

			got := rec.received()
			if got.method != http.MethodPost {
				t.Errorf("method = %q, want %q", got.method, http.MethodPost)
			}
			if got.path != "/i/v0/e/" {
				t.Errorf("path = %q, want %q", got.path, "/i/v0/e/")
			}
			if got.contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", got.contentType, "application/json")
			}
			if got.event.APIKey != testKey {
				t.Errorf("api_key = %q, want %q", got.event.APIKey, testKey)
			}
			if got.event.Event != "cli_updated" {
				t.Errorf("event = %q, want %q", got.event.Event, "cli_updated")
			}
			if !isInstallID(got.event.DistinctID) {
				t.Errorf("distinct_id = %q, want 32 lowercase hex characters", got.event.DistinctID)
			}
			for k, want := range tt.want {
				if have := got.event.Properties[k]; have != want {
					t.Errorf("properties[%q] = %v, want %v", k, have, want)
				}
			}
			for _, k := range []string{"os", "arch"} {
				if have, _ := got.event.Properties[k].(string); have == "" {
					t.Errorf("properties[%q] is empty, want the build platform", k)
				}
			}
		})
	}
}

// TestCapture_InstallIDIsStable: two events from one machine must name one
// machine. An id regenerated per event would count every invocation as a new
// install.
func TestCapture_InstallIDIsStable(t *testing.T) {
	clearEnv(t)
	rec := newRecorder(t)
	dir := t.TempDir()
	c := rec.client("1.0.0", testKey, dir)

	if !c.Capture(context.Background(), "cli_updated", nil) {
		t.Fatal("first Capture() = false, want true")
	}
	first := rec.received().event.DistinctID

	if !c.Capture(context.Background(), "cli_update_failed", map[string]any{"stage": "download"}) {
		t.Fatal("second Capture() = false, want true")
	}
	if second := rec.received().event.DistinctID; second != first {
		t.Errorf("distinct_id changed between events: %q then %q", first, second)
	}

	data, err := os.ReadFile(filepath.Join(dir, installIDFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != first {
		t.Errorf("the stored id is %q, want the sent id %q", strings.TrimSpace(string(data)), first)
	}
}

// TestCapture_ReusesInstallerID is the shared-identifier invariant: install.sh
// writes this file first, and the CLI must adopt what it finds rather than
// reformat it. A rewrite would split one machine into two distinct_ids and
// break every join between "installed" and "updated".
func TestCapture_ReusesInstallerID(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantID   string // empty: the client must mint a fresh id
		wantFile string // empty: the file must be left byte-identical
	}{
		{
			name:    "installer format with a trailing newline",
			content: "0123456789abcdef0123456789abcdef\n",
			wantID:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:    "no trailing newline",
			content: "fedcba9876543210fedcba9876543210",
			wantID:  "fedcba9876543210fedcba9876543210",
		},
		{
			name:    "surrounding whitespace is trimmed",
			content: "  abcdef01abcdef01abcdef01abcdef01  \n",
			wantID:  "abcdef01abcdef01abcdef01abcdef01",
		},
		{
			name:    "content that is not an identifier is replaced",
			content: "NOT-AN-ID\n",
		},
		// A hex run of the wrong length is a partial write, not an identifier.
		// createInstallID's fs.ErrExist branch rewrites this file with a plain
		// os.WriteFile, and the exclusive-create branch has a gap between the
		// create and the first Write; a reader landing in either window sees a
		// prefix. Taking one as real gave the machine a second distinct_id —
		// exactly what the exclusive create exists to prevent.
		{
			name:    "a short hex prefix is a partial write, not an id",
			content: "abc123\n",
		},
		{
			name:    "one character short of an id",
			content: "0123456789abcdef0123456789abcde\n",
		},
		{
			name:    "one character long",
			content: "0123456789abcdef0123456789abcdef0\n",
		},
		{
			name:    "an empty write in progress",
			content: "\n",
		},
		{
			name:    "an empty file is replaced",
			content: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			rec := newRecorder(t)
			dir := t.TempDir()
			path := filepath.Join(dir, installIDFile)
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			c := rec.client("1.0.0", testKey, dir)
			if !c.Capture(context.Background(), "cli_updated", nil) {
				t.Fatal("Capture() = false, want true")
			}

			sent := rec.received().event.DistinctID
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantID == "" {
				if !isInstallID(sent) {
					t.Errorf("distinct_id = %q, want a fresh 32-character hex id", sent)
				}
				if strings.TrimSpace(string(after)) != sent {
					t.Errorf("the stored id is %q, want the sent id %q", strings.TrimSpace(string(after)), sent)
				}
				return
			}
			if sent != tt.wantID {
				t.Errorf("distinct_id = %q, want %q", sent, tt.wantID)
			}
			if string(after) != tt.content {
				t.Errorf("the id file was rewritten as %q, want it left as %q", string(after), tt.content)
			}
		})
	}
}

// TestCapture_UsesXDGStateDirByDefault covers the seam the production build
// takes: an empty StateDir resolves through internal/xdg, which is the path
// install.sh writes.
func TestCapture_UsesXDGStateDirByDefault(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)

	rec := newRecorder(t)
	c := &Client{
		Version:    "1.0.0",
		Key:        testKey,
		Endpoint:   rec.endpoint(),
		HTTPClient: rec.server.Client(),
	}
	if !c.Capture(context.Background(), "cli_updated", nil) {
		t.Fatal("Capture() = false, want true")
	}

	data, err := os.ReadFile(filepath.Join(root, "archcore", installIDFile))
	if err != nil {
		t.Fatalf("no install-id under the XDG state directory: %v", err)
	}
	if sent := rec.received().event.DistinctID; strings.TrimSpace(string(data)) != sent {
		t.Errorf("the stored id is %q, want the sent id %q", strings.TrimSpace(string(data)), sent)
	}
}

// TestCapture_UnavailableIdentifierSkips: an id that cannot be persisted would
// differ on the next run, so the event is dropped rather than sent under an
// unstable identity.
func TestCapture_UnavailableIdentifierSkips(t *testing.T) {
	clearEnv(t)
	rec := newRecorder(t)

	// A regular file where a directory has to be: MkdirAll fails for any user,
	// root included.
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	c := rec.client("1.0.0", testKey, filepath.Join(blocked, "state"))
	if c.Capture(context.Background(), "cli_updated", nil) {
		t.Error("Capture() = true, want false when the identifier cannot be created")
	}
	if n := rec.count(); n != 0 {
		t.Errorf("endpoint received %d requests, want 0", n)
	}
}

// TestCapture_UnresolvableStateDirectorySkips covers the other early return in
// installID: stateDir() answers "" when no XDG_STATE_HOME is set and the home
// directory cannot be resolved either.
//
// It is a different branch from the one above, which fails at MkdirAll after a
// directory has been named. This one never names a directory at all, and a
// regression that only guarded the MkdirAll would send an event under a
// one-off identifier — counting the machine again on every run.
func TestCapture_UnresolvableStateDirectorySkips(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir does not read HOME on windows")
	}
	clearEnv(t)
	rec := newRecorder(t)
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")

	// No StateDir override, so the client falls through to xdg.StateDir().
	c := &Client{
		Version:    "1.0.0",
		Key:        testKey,
		Endpoint:   rec.endpoint(),
		HTTPClient: rec.server.Client(),
	}
	if c.Capture(context.Background(), EventUpdated, nil) {
		t.Error("Capture() = true, want false when no state directory resolves")
	}
	if n := rec.count(); n != 0 {
		t.Errorf("endpoint received %d requests, want 0", n)
	}
}

// TestCapture_TransportFailures: every delivery failure is the same outcome to
// the caller — false, no error, no stall.
func TestCapture_TransportFailures(t *testing.T) {
	t.Run("non-2xx status", func(t *testing.T) {
		clearEnv(t)
		rec := newRecorder(t)
		rec.setStatus(http.StatusInternalServerError)

		c := rec.client("1.0.0", testKey, t.TempDir())
		if c.Capture(context.Background(), "cli_updated", nil) {
			t.Error("Capture() = true on a 500 response, want false")
		}
		if n := rec.count(); n != 1 {
			t.Errorf("endpoint received %d requests, want 1", n)
		}
	})

	t.Run("a hanging endpoint is bounded", func(t *testing.T) {
		clearEnv(t)
		const bound = 200 * time.Millisecond

		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			<-release
		}))
		// Closed before the server, so the handler returns and Close does not
		// wait on it.
		defer srv.Close()
		defer close(release)

		client := srv.Client()
		client.Timeout = bound

		c := &Client{
			Version:    "1.0.0",
			Key:        testKey,
			Endpoint:   srv.URL + "/i/v0/e/",
			HTTPClient: client,
			StateDir:   t.TempDir(),
		}

		start := time.Now()
		got := c.Capture(context.Background(), "cli_updated", nil)
		elapsed := time.Since(start)

		if got {
			t.Error("Capture() = true against a hanging endpoint, want false")
		}
		if elapsed > 10*bound {
			t.Errorf("Capture() blocked for %v, want it bounded near %v", elapsed, bound)
		}
	})

	t.Run("an unreachable endpoint", func(t *testing.T) {
		clearEnv(t)
		c := &Client{
			Version:  "1.0.0",
			Key:      testKey,
			Endpoint: "http://127.0.0.1:1/i/v0/e/",
			HTTPClient: &http.Client{
				Timeout: time.Second,
			},
			StateDir: t.TempDir(),
		}
		if c.Capture(context.Background(), "cli_updated", nil) {
			t.Error("Capture() = true against an unreachable endpoint, want false")
		}
	})

	t.Run("a cancelled context", func(t *testing.T) {
		clearEnv(t)
		rec := newRecorder(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		c := rec.client("1.0.0", testKey, t.TempDir())
		if c.Capture(ctx, "cli_updated", nil) {
			t.Error("Capture() = true with a cancelled context, want false")
		}
	})
}

// TestCapture_BoundedWithoutAnInjectedTimeout covers the bound the production
// path actually relies on. The hanging-endpoint case above injects a client
// carrying its own 200 ms timeout, so it proves that client's timeout works and
// says nothing about this package's. Here the caller supplies an unbounded
// context and either the package's own client or one with no timeout at all —
// the two shapes an `archcore update` can meet — and the send must still return
// well inside the contract's total timeout.
//
// The subtests run in parallel against one hanging endpoint, so the whole test
// costs one timeout rather than two. clearEnv runs on the parent: t.Setenv
// cannot be called from a parallel subtest, and the parent's restore waits for
// them to finish.
func TestCapture_BoundedWithoutAnInjectedTimeout(t *testing.T) {
	clearEnv(t)

	var (
		mu       sync.Mutex
		requests int
	)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		<-release
	}))
	// Released before Close, so the handlers return and Close does not wait on
	// them. Both run after the parallel subtests have finished.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	tests := []struct {
		name   string
		client *http.Client
	}{
		{name: "the package's own client", client: nil},
		{name: "an injected client with no timeout", client: &http.Client{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Client{
				Version:    "1.0.0",
				Key:        testKey,
				Endpoint:   srv.URL + "/i/v0/e/",
				HTTPClient: tt.client,
				StateDir:   t.TempDir(),
			}

			start := time.Now()
			got := c.Capture(context.Background(), "cli_updated", nil)
			elapsed := time.Since(start)

			if got {
				t.Error("Capture() = true against a hanging endpoint, want false")
			}
			// Without this the test would pass on a build that refuses at a
			// guard and never opens a connection, which bounds nothing.
			mu.Lock()
			seen := requests
			mu.Unlock()
			if seen == 0 {
				t.Fatal("the endpoint saw no request; the send was refused before the timeout could apply")
			}
			if elapsed > 2*totalTimeout {
				t.Errorf("Capture() blocked for %v, want it bounded by the %v total timeout", elapsed, totalTimeout)
			}
		})
	}
}

// TestCreateInstallID_LoserAdoptsTheWinnersID: two invocations can reach the
// creation path at once on a fresh machine — the MCP trigger and a typed
// `archcore update`, or the CLI and install.sh. The second must not overwrite
// the first, or one machine reports two distinct_ids and every join between
// "installed" and "updated" loses a row.
func TestCreateInstallID_LoserAdoptsTheWinnersID(t *testing.T) {
	const (
		winner = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		loser  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	path := filepath.Join(t.TempDir(), installIDFile)

	if got := createInstallID(path, winner); got != winner {
		t.Fatalf("first createInstallID() = %q, want %q", got, winner)
	}
	if got := createInstallID(path, loser); got != winner {
		t.Errorf("second createInstallID() = %q, want the stored %q", got, winner)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != winner+"\n" {
		t.Errorf("the id file holds %q, want %q", string(data), winner+"\n")
	}
}

// TestCapture_NilClient: a caller with telemetry switched off holds a nil
// Client and still calls Capture, so the call site needs no branch.
func TestCapture_NilClient(t *testing.T) {
	clearEnv(t)
	var c *Client
	if c.Capture(context.Background(), "cli_updated", map[string]any{"trigger": "manual"}) {
		t.Error("Capture() on a nil Client = true, want false")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("1.2.3")
	if c.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", c.Version, "1.2.3")
	}
	if want := "https://ph.archcore.ai/i/v0/e/"; c.endpoint() != want {
		t.Errorf("endpoint() = %q, want %q", c.endpoint(), want)
	}
	if c.key() != apiKey {
		t.Errorf("key() = %q, want the injected package key %q", c.key(), apiKey)
	}
}

// TestPackageKeyIsInertInThisRepository: the key must never be committed. A
// checked-in key would make every developer build, every fork and every CI run
// report events.
func TestPackageKeyIsInertInThisRepository(t *testing.T) {
	if strings.HasPrefix(apiKey, keyPrefix) {
		t.Error("the package key is a live project key; it must arrive through an -X ldflag at release time")
	}
}
