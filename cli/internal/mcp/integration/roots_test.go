package integration

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mcpserver "archcore-cli/internal/mcp"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// fakeRoots answers roots/list the way a host does. The reply is swappable so a
// test can move the session into a worktree mid-run, which is the whole point
// of the feature under test.
type fakeRoots struct {
	mu    sync.Mutex
	dirs  []string
	err   error
	hang  time.Duration
	calls int
}

func (f *fakeRoots) set(dirs ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirs = dirs
}

func (f *fakeRoots) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeRoots) ListRoots(ctx context.Context, _ mcp.ListRootsRequest) (*mcp.ListRootsResult, error) {
	f.mu.Lock()
	f.calls++
	dirs, err, hang := f.dirs, f.err, f.hang
	f.mu.Unlock()

	if hang > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(hang):
		}
	}
	if err != nil {
		return nil, err
	}
	roots := make([]mcp.Root, 0, len(dirs))
	for _, d := range dirs {
		// Three slashes, and the path contributes none of its own: on Windows
		// filepath.ToSlash yields "C:/repo" with no leading separator, and
		// "file://C:/repo" parses with "C:" as the URI host — which fileURIPath
		// correctly refuses as remote.
		roots = append(roots, mcp.Root{
			URI:  "file:///" + strings.TrimPrefix(filepath.ToSlash(d), "/"),
			Name: filepath.Base(d),
		})
	}
	return &mcp.ListRootsResult{Roots: roots}, nil
}

// newRootsClient wires a client that declares the roots capability and answers
// roots/list from handler. Both options are load-bearing: the transport option
// registers the session the server needs to reach the client, and the client
// option is what makes Initialize declare the capability.
func newRootsClient(t *testing.T, baseDir string, handler *fakeRoots, warnings *bytes.Buffer) *client.Client {
	t.Helper()
	srv := mcpserver.NewServer(baseDir, "test", mcpserver.WithRootWarnings(warnings))
	tr := transport.NewInProcessTransportWithOptions(srv, transport.WithRootsHandler(handler))
	c := client.NewClient(tr, client.WithRootsHandler(handler))
	startClient(t, c)
	return c
}

// newUndeclaredRootsClient registers a session that can answer roots/list but
// never declares the capability at initialize. The distinction is the point: a
// client with no session at all would exercise the "nothing to ask" path
// instead, and the capability check would go untested.
func newUndeclaredRootsClient(t *testing.T, baseDir string, handler *fakeRoots, warnings *bytes.Buffer) *client.Client {
	t.Helper()
	srv := mcpserver.NewServer(baseDir, "test", mcpserver.WithRootWarnings(warnings))
	tr := transport.NewInProcessTransportWithOptions(srv, transport.WithRootsHandler(handler))
	c := client.NewClient(tr) // no client.WithRootsHandler: capability stays undeclared
	startClient(t, c)
	return c
}

// newPinnedClient is a roots-capable client against a server whose root was
// stated with --project or ARCHCORE_PROJECT_ROOT.
func newPinnedClient(t *testing.T, baseDir string, handler *fakeRoots, warnings *bytes.Buffer) *client.Client {
	t.Helper()
	srv := mcpserver.NewServer(baseDir, "test",
		mcpserver.WithPinnedRoot(), mcpserver.WithRootWarnings(warnings))
	tr := transport.NewInProcessTransportWithOptions(srv, transport.WithRootsHandler(handler))
	c := client.NewClient(tr, client.WithRootsHandler(handler))
	startClient(t, c)
	return c
}

func startClient(t *testing.T, c *client.Client) {
	t.Helper()
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("client.Close: %v", err)
		}
	})
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "archcore-roots-test", Version: "0.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}
}

// newProject creates a directory holding .archcore/ and returns it.
func newProject(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	writeFixtureFile(t, filepath.Join(dir, ".archcore", "settings.json"), `{"sync":"none"}`)
	return dir
}

// documentExists reports whether a tool call wrote rel under base. It is how a
// test names the checkout a write landed in.
func documentExists(t *testing.T, base, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(base, rel))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stat %s: %v", rel, err)
	}
	return err == nil
}

// TestRoots_WriteFollowsTheReportedRoot is issue #31 end to end: the server
// starts on one checkout, the client reports another, and the next write lands
// in the reported one.
func TestRoots_WriteFollowsTheReportedRoot(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	moved := newProject(t, "moved")
	handler := &fakeRoots{}
	handler.set(moved)
	var warnings bytes.Buffer

	c := newRootsClient(t, start, handler, &warnings)
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "adr", "filename": "moved-here", "title": "Moved Here",
	})

	rel := filepath.Join(".archcore", "moved-here.adr.md")
	if !documentExists(t, moved, rel) {
		t.Errorf("document did not land in the reported root; warnings: %q", warnings.String())
	}
	if documentExists(t, start, rel) {
		t.Error("document landed in the start-time root, which the session has left")
	}
}

// TestRoots_FollowsASwitchMidSession covers the sequence the issue describes:
// the session works in one checkout, enters another, and the server moves with
// it on the next call past the cache window.
func TestRoots_FollowsASwitchMidSession(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	moved := newProject(t, "moved")
	handler := &fakeRoots{}
	handler.set(start)
	var warnings bytes.Buffer

	c := newRootsClient(t, start, handler, &warnings)
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "adr", "filename": "before-switch", "title": "Before Switch",
	})
	if !documentExists(t, start, filepath.Join(".archcore", "before-switch.adr.md")) {
		t.Fatal("the first write did not land in the start-time root")
	}

	handler.set(moved)
	waitPastRootCache()

	mustCallTool(t, c, "create_document", map[string]any{
		"type": "adr", "filename": "after-switch", "title": "After Switch",
	})
	if !documentExists(t, moved, filepath.Join(".archcore", "after-switch.adr.md")) {
		t.Errorf("the write after the switch did not follow the session; warnings: %q", warnings.String())
	}
}

func TestRoots_RefusedCandidates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// candidates returns the roots the host reports, given a scratch parent.
		candidates func(t *testing.T) []string
		wantWarn   string
	}{
		{
			name: "directory without .archcore",
			candidates: func(t *testing.T) []string {
				return []string{t.TempDir()}
			},
			wantWarn: "no .archcore/",
		},
		{
			name: "declared global does not resolve",
			candidates: func(t *testing.T) []string {
				dir := newProject(t, "broken")
				writeFixtureFile(t, filepath.Join(dir, ".archcore", "settings.json"),
					`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
				return []string{dir}
			},
			wantWarn: `"company"`,
		},
		{
			name: "plugin install cache",
			candidates: func(t *testing.T) []string {
				dir := filepath.Join(t.TempDir(), ".claude", "plugins", "cached")
				writeFixtureFile(t, filepath.Join(dir, ".archcore", "settings.json"), `{"sync":"none"}`)
				return []string{dir}
			},
			wantWarn: "plugin install cache",
		},
		{
			name: "several usable candidates",
			candidates: func(t *testing.T) []string {
				return []string{newProject(t, "one"), newProject(t, "two")}
			},
			wantWarn: "several usable",
		},
		{
			name:       "no candidate at all",
			candidates: func(t *testing.T) []string { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			start := newProject(t, "start")
			candidates := tt.candidates(t)
			handler := &fakeRoots{}
			handler.set(candidates...)
			var warnings bytes.Buffer

			c := newRootsClient(t, start, handler, &warnings)
			mustCallTool(t, c, "create_document", map[string]any{
				"type": "adr", "filename": "stays-put", "title": "Stays Put",
			})

			if !documentExists(t, start, filepath.Join(".archcore", "stays-put.adr.md")) {
				t.Errorf("the server left the start-time root for a candidate it must refuse; warnings: %q", warnings.String())
			}
			if tt.wantWarn != "" && !strings.Contains(warnings.String(), tt.wantWarn) {
				t.Errorf("warnings = %q, want a line containing %q", warnings.String(), tt.wantWarn)
			}
			// no-absolute-paths-in-mcp-errors.rule covers what the server says
			// about a path, stderr included. The refused candidate is the path a
			// refusal message would most plausibly carry, so it is checked
			// alongside the root the server kept.
			for _, leak := range append(candidates, start) {
				if strings.Contains(warnings.String(), leak) {
					t.Errorf("warnings = %q leak an absolute filesystem path", warnings.String())
				}
			}
		})
	}
}

// TestRoots_NoCapabilityIsNeverAsked covers project-root-resolution.spec §4: a
// host that does not declare roots must not be queried, because the question
// can only time out. The session here answers roots/list if asked, so the call
// count is the assertion that matters — the root staying put would also hold on
// paths that never reach the capability check.
func TestRoots_NoCapabilityIsNeverAsked(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	moved := newProject(t, "moved")
	handler := &fakeRoots{}
	handler.set(moved)
	var warnings bytes.Buffer

	c := newUndeclaredRootsClient(t, start, handler, &warnings)
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "adr", "filename": "no-roots", "title": "No Roots",
	})

	if got := handler.callCount(); got != 0 {
		t.Errorf("roots/list was requested %d times for a host that declares no roots, want 0", got)
	}
	if !documentExists(t, start, filepath.Join(".archcore", "no-roots.adr.md")) {
		t.Error("a session without the roots capability must keep the start-time root")
	}
	if warnings.Len() != 0 {
		t.Errorf("warnings = %q, want nothing for a host that declares no roots", warnings.String())
	}
}

// TestRoots_PinnedServerNeverAsks covers project-root-resolution.spec §1 and §2
// over a live session: the client declares roots and would answer with another
// project, and the pinned server must neither ask nor move.
func TestRoots_PinnedServerNeverAsks(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	moved := newProject(t, "moved")
	handler := &fakeRoots{}
	handler.set(moved)
	var warnings bytes.Buffer

	c := newPinnedClient(t, start, handler, &warnings)
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "adr", "filename": "pinned", "title": "Pinned",
	})

	if got := handler.callCount(); got != 0 {
		t.Errorf("a pinned server requested roots/list %d times, want 0", got)
	}
	if !documentExists(t, start, filepath.Join(".archcore", "pinned.adr.md")) {
		t.Errorf("a pinned server left its root; warnings: %q", warnings.String())
	}
}

// TestRoots_SameRootIsNotRechecked covers project-root-resolution.spec §10: a
// client reporting the root the server already serves must not send it back
// through the acceptance checks.
// The start root here carries a global that does not resolve, so a re-check
// would refuse it and say so — the warning is what makes the skip observable.
func TestRoots_SameRootIsNotRechecked(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	writeFixtureFile(t, filepath.Join(start, ".archcore", "settings.json"),
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
	handler := &fakeRoots{}
	handler.set(start)
	var warnings bytes.Buffer

	c := newRootsClient(t, start, handler, &warnings)
	mustCallTool(t, c, "get_document", map[string]any{"path": ".archcore/settings.json"})

	if warnings.Len() != 0 {
		t.Errorf("warnings = %q, want nothing: the reported root is the one already served", warnings.String())
	}
}

// TestRoots_QueryFailureKeepsTheRoot covers the failure rows: an error and a
// query that never answers both leave the current root in place, and the tool
// call still succeeds.
func TestRoots_QueryFailureKeepsTheRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		prepare func(f *fakeRoots)
	}{
		{
			name:    "transport error",
			prepare: func(f *fakeRoots) { f.err = errors.New("client refused") },
		},
		{
			name:    "query never answers",
			prepare: func(f *fakeRoots) { f.hang = 5 * time.Second },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			start := newProject(t, "start")
			moved := newProject(t, "moved")
			handler := &fakeRoots{}
			handler.set(moved)
			tt.prepare(handler)
			var warnings bytes.Buffer

			c := newRootsClient(t, start, handler, &warnings)
			mustCallTool(t, c, "create_document", map[string]any{
				"type": "adr", "filename": "kept", "title": "Kept",
			})

			if !documentExists(t, start, filepath.Join(".archcore", "kept.adr.md")) {
				t.Error("a failed roots/list must leave the current root in place")
			}
			if !strings.Contains(warnings.String(), "roots/list") {
				t.Errorf("warnings = %q, want a line naming the unanswered query", warnings.String())
			}
		})
	}
}

// TestRoots_QueriedAtMostOncePerWindow covers project-root-resolution.spec §6
// and §8: a burst of tool calls costs one roots/list request, not one per call.
func TestRoots_QueriedAtMostOncePerWindow(t *testing.T) {
	t.Parallel()
	start := newProject(t, "start")
	handler := &fakeRoots{}
	handler.set(start)
	var warnings bytes.Buffer

	c := newRootsClient(t, start, handler, &warnings)
	for range 5 {
		mustCallTool(t, c, "list_documents", map[string]any{})
	}

	if got := handler.callCount(); got != 1 {
		t.Errorf("roots/list was requested %d times for five tool calls, want 1", got)
	}
}

// TestRoots_InitProjectStillWorksOnAnUninitializedStartRoot pins the exemption
// in the acceptance rule: the start-time root stays servable without
// .archcore/, so the guarantee of mcp-server-starts-without-archcore-dir.adr
// survives the containment check a move must pass.
func TestRoots_InitProjectStillWorksOnAnUninitializedStartRoot(t *testing.T) {
	t.Parallel()
	start := t.TempDir() // no .archcore/ anywhere
	handler := &fakeRoots{}
	handler.set(start)
	var warnings bytes.Buffer

	c := newRootsClient(t, start, handler, &warnings)
	mustCallTool(t, c, "init_project", map[string]any{})

	if _, err := os.Stat(filepath.Join(start, ".archcore")); err != nil {
		t.Errorf("init_project did not initialize the start-time root: %v (warnings: %q)", err, warnings.String())
	}
}

// waitPastRootCache sleeps until the provider's cached decision has expired.
// The window is read from the package that owns it, so a change to the constant
// cannot leave this test silently waiting too little.
func waitPastRootCache() {
	time.Sleep(mcpserver.RootCacheTTL + 500*time.Millisecond)
}
