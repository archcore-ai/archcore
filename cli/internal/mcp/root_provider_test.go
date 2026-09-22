package mcp

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestFileURIPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, uri, want string
		wantErr         bool
	}{
		{name: "posix path", uri: "file:///home/u/repo", want: filepath.FromSlash("/home/u/repo")},
		{name: "percent escape", uri: "file:///home/u/my%20repo", want: filepath.FromSlash("/home/u/my repo")},
		{name: "localhost host", uri: "file://localhost/home/u/repo", want: filepath.FromSlash("/home/u/repo")},
		{name: "trailing slash cleaned", uri: "file:///home/u/repo/", want: filepath.FromSlash("/home/u/repo")},
		{name: "remote host rejected", uri: "file://elsewhere/home/u/repo", wantErr: true},
		{name: "non-file scheme rejected", uri: "https://example.com/repo", wantErr: true},
		{name: "empty path rejected", uri: "file://", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := fileURIPath(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("fileURIPath(%q) = %q, want an error", tt.uri, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("fileURIPath(%q): %v", tt.uri, err)
			}
			if got != tt.want {
				t.Errorf("fileURIPath(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestAcceptRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns the candidate directory; wantOK says whether it passes,
		// and wantReason is a fragment of the refusal.
		setup      func(t *testing.T) string
		wantOK     bool
		wantReason string
	}{
		{
			name: "project with a resolvable global",
			setup: func(t *testing.T) string {
				parent := t.TempDir()
				base := filepath.Join(parent, "primary")
				mkdirAll(t, filepath.Join(base, ".archcore"))
				mkdirAll(t, filepath.Join(parent, "company", ".archcore"))
				writeFile(t, filepath.Join(base, ".archcore", "settings.json"),
					`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
				return base
			},
			wantOK: true,
		},
		{
			name: "project with no globals",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				mkdirAll(t, filepath.Join(base, ".archcore"))
				return base
			},
			wantOK: true,
		},
		{
			name:       "relative path",
			setup:      func(t *testing.T) string { return "relative/dir" },
			wantReason: "not absolute",
		},
		{
			name:       "missing directory",
			setup:      func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent") },
			wantReason: "not an existing directory",
		},
		{
			name: "plugin install cache",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), ".claude", "plugins", "cached")
				mkdirAll(t, filepath.Join(dir, ".archcore"))
				return dir
			},
			wantReason: "plugin install cache",
		},
		{
			name:       "no .archcore directory",
			setup:      func(t *testing.T) string { return t.TempDir() },
			wantReason: "no .archcore/",
		},
		{
			name: "declared global does not resolve",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				mkdirAll(t, filepath.Join(base, ".archcore"))
				writeFile(t, filepath.Join(base, ".archcore", "settings.json"),
					`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
				return base
			},
			wantReason: `"company"`,
		},
		{
			name: "unreadable settings.json",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				mkdirAll(t, filepath.Join(base, ".archcore"))
				writeFile(t, filepath.Join(base, ".archcore", "settings.json"), `{not valid json`)
				return base
			},
			wantReason: "settings.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)
			reason, ok := acceptRoot(dir)
			if ok != tt.wantOK {
				t.Fatalf("acceptRoot(%q) ok = %v, want %v (reason %q)", dir, ok, tt.wantOK, reason)
			}
			if tt.wantReason != "" && !strings.Contains(reason, tt.wantReason) {
				t.Errorf("acceptRoot reason = %q, want it to contain %q", reason, tt.wantReason)
			}
		})
	}
}

// TestAcceptRoot_ReasonCarriesNoAbsolutePath keeps the refusal lines inside
// no-absolute-paths-in-mcp-errors.rule and project-root-resolution.spec §17:
// the reason names the check and the declared source id, never where anything
// lives.
//
// The absolute-declaration case is the one that regressed. The reason used to
// be GlobalInspection.Message(), which quotes the declared path — harmless
// while every fixture declared a relative one, and a leak the moment a user
// wrote "/Users/me/company/.archcore" into settings.json.
func TestAcceptRoot_ReasonCarriesNoAbsolutePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// declaration is the globals entry written into settings.json, and
		// leaked is the path the reason must not contain. An empty leaked means
		// the candidate directory itself.
		declaration func(parent string) (settings, leaked string)
	}{
		{
			name: "relative declaration",
			declaration: func(string) (string, string) {
				return `{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`, ""
			},
		},
		{
			name: "absolute declaration",
			declaration: func(parent string) (string, string) {
				missing := filepath.Join(parent, "company", ".archcore")
				return `{"sync":"none","globals":[{"id":"company","path":"` +
					filepath.ToSlash(missing) + `"}]}`, missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parent := t.TempDir()
			base := filepath.Join(parent, "primary")
			mkdirAll(t, filepath.Join(base, ".archcore"))
			settings, leaked := tt.declaration(parent)
			writeFile(t, filepath.Join(base, ".archcore", "settings.json"), settings)

			reason, ok := acceptRoot(base)
			if ok {
				t.Fatalf("acceptRoot accepted a project whose global does not resolve")
			}
			if strings.Contains(reason, base) {
				t.Errorf("reason %q embeds the absolute candidate path", reason)
			}
			if leaked != "" && strings.Contains(reason, filepath.ToSlash(leaked)) {
				t.Errorf("reason %q embeds the declared absolute source path", reason)
			}
			if !strings.Contains(reason, `"company"`) {
				t.Errorf("reason %q does not name the declared source id", reason)
			}
		})
	}
}

// TestSessionRootProvider_PinnedNeverAsks covers project-root-resolution.spec
// §1 and §2: an explicit root is served for the process lifetime and the client
// is never queried. A pinned provider must reach that answer without a session
// on the context at all.
func TestSessionRootProvider_PinnedNeverAsks(t *testing.T) {
	t.Parallel()
	var warnings bytes.Buffer
	p := newSessionRootProvider("/pinned/root", true, &warnings)

	for range 3 {
		if got := p.Root(context.Background()); got != "/pinned/root" {
			t.Fatalf("Root() = %q, want the pinned root", got)
		}
	}
	if p.queries != 0 {
		t.Errorf("pinned provider issued %d roots/list requests, want 0", p.queries)
	}
	if warnings.Len() != 0 {
		t.Errorf("pinned provider wrote %q to stderr, want nothing", warnings.String())
	}
}

// TestSessionRootProvider_NoSessionKeepsTheStartRoot covers the failure row for
// a context carrying no client session: serve the current root, say nothing.
func TestSessionRootProvider_NoSessionKeepsTheStartRoot(t *testing.T) {
	t.Parallel()
	var warnings bytes.Buffer
	p := newSessionRootProvider("/start/root", false, &warnings)

	if got := p.Root(context.Background()); got != "/start/root" {
		t.Errorf("Root() = %q, want the start-time root", got)
	}
	if p.queries != 0 {
		t.Errorf("provider issued %d roots/list requests without a session, want 0", p.queries)
	}
	if warnings.Len() != 0 {
		t.Errorf("provider wrote %q to stderr, want nothing", warnings.String())
	}
}

// TestSessionRootProvider_WarnsOncePerReason covers
// project-root-resolution.spec §16: a session running against a host the
// provider cannot follow must not repeat the same line on every tool call.
func TestSessionRootProvider_WarnsOncePerReason(t *testing.T) {
	t.Parallel()
	var warnings bytes.Buffer
	p := newSessionRootProvider("/start/root", false, &warnings)

	for range 3 {
		p.warnOnce("root-query", "the host did not answer roots/list")
	}
	p.warnOnce("root-ambiguous", "several usable projects")

	if got := strings.Count(warnings.String(), "did not answer"); got != 1 {
		t.Errorf("the same reason appeared %d times, want 1", got)
	}
	if got := strings.Count(warnings.String(), "several usable"); got != 1 {
		t.Errorf("the second reason appeared %d times, want 1", got)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeSession answers roots/list without a transport, so the provider's decision
// path can be driven directly. It deliberately does not implement
// server.SessionWithClientInfo: the capability check is skipped and the query is
// put, which is the path these tests are about.
type fakeSession struct {
	dirs []string
	// block, when non-nil, holds ListRoots until it is closed. It is how a test
	// keeps one refresh in flight while another call arrives.
	block chan struct{}
	calls atomic.Int64
}

func (s *fakeSession) Initialize()                                         {}
func (s *fakeSession) Initialized() bool                                   { return true }
func (s *fakeSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return nil }
func (s *fakeSession) SessionID() string                                   { return "fake" }

func (s *fakeSession) ListRoots(ctx context.Context, _ mcp.ListRootsRequest) (*mcp.ListRootsResult, error) {
	s.calls.Add(1)
	if s.block != nil {
		select {
		case <-s.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	roots := make([]mcp.Root, 0, len(s.dirs))
	for _, d := range s.dirs {
		roots = append(roots, mcp.Root{URI: "file:///" + strings.TrimPrefix(filepath.ToSlash(d), "/")})
	}
	return &mcp.ListRootsResult{Roots: roots}, nil
}

// sessionContext puts sess where server.ClientSessionFromContext finds it.
func sessionContext(sess server.ClientSession) context.Context {
	return server.NewMCPServer("test", "0.0.0").WithContext(context.Background(), sess)
}

// newAcceptableProject returns a directory acceptRoot will accept.
func newAcceptableProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, ".archcore"))
	return dir
}

// TestSessionRootProvider_AdoptsASingleAcceptableCandidate covers
// project-root-resolution.spec §11 and §12, and pins the adoption counter: a
// second call inside the cache window must not re-adopt the root it is already
// serving.
func TestSessionRootProvider_AdoptsASingleAcceptableCandidate(t *testing.T) {
	t.Parallel()
	target := newAcceptableProject(t)
	sess := &fakeSession{dirs: []string{target}}
	var warnings bytes.Buffer
	p := newSessionRootProvider(t.TempDir(), false, &warnings)
	ctx := sessionContext(sess)

	if got := p.Root(ctx); got != target {
		t.Fatalf("Root() = %q, want the reported candidate %q (warnings %q)", got, target, warnings.String())
	}
	if p.adoption != 1 {
		t.Fatalf("adoption = %d after the first switch, want 1", p.adoption)
	}
	if got := p.Root(ctx); got != target {
		t.Fatalf("Root() = %q on the second call, want %q", got, target)
	}
	if p.adoption != 1 {
		t.Errorf("adoption = %d after a cached second call, want 1", p.adoption)
	}
	if got := sess.calls.Load(); got != 1 {
		t.Errorf("roots/list was requested %d times inside one cache window, want 1", got)
	}
}

// TestSessionRootProvider_ConcurrentCallsIssueOneQuery covers
// project-root-resolution.spec §9. The second call must not queue behind the refresh: a refresh spans a client round trip
// and a walk of every declared global source, and a tool call that asked for
// nothing must not pay for it. It serves the root it already has instead.
func TestSessionRootProvider_ConcurrentCallsIssueOneQuery(t *testing.T) {
	t.Parallel()
	start := t.TempDir()
	target := newAcceptableProject(t)
	release := make(chan struct{})
	sess := &fakeSession{dirs: []string{target}, block: release}
	p := newSessionRootProvider(start, false, io.Discard)
	ctx := sessionContext(sess)

	var wg sync.WaitGroup
	wg.Add(1)
	var refreshed string
	go func() {
		defer wg.Done()
		refreshed = p.Root(ctx)
	}()

	// Wait for the refresh to be inside ListRoots, so the second call is
	// guaranteed to arrive while it is in flight.
	waitFor(t, func() bool { return sess.calls.Load() == 1 })

	concurrent := make(chan string, 1)
	go func() { concurrent <- p.Root(ctx) }()
	select {
	case got := <-concurrent:
		if got != start {
			t.Errorf("the concurrent call returned %q, want the root it already had (%q)", got, start)
		}
	case <-time.After(rootQueryTimeout):
		t.Error("the concurrent call queued behind the in-flight refresh instead of serving its current root")
	}

	close(release)
	wg.Wait()

	if refreshed != target {
		t.Errorf("the refreshing call returned %q, want the adopted candidate %q", refreshed, target)
	}
	if got := sess.calls.Load(); got != 1 {
		t.Errorf("two concurrent calls issued %d roots/list requests, want 1", got)
	}
}

// waitFor blocks until cond holds, and fails the test if it never does.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(time.Millisecond)
	}
}
