package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"archcore-cli/internal/display"
	"archcore-cli/internal/docs"
	"archcore-cli/internal/projectroot"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// rootQueryTimeout bounds one roots/list round trip. The transport puts no
// timeout of its own on the request — the caller's context is the only bound —
// and a stalled query occupies one of the server's five tool workers, so the
// bound lives here. It matches internal/git's budget for the same reason: a
// local answer slower than half a second is a hang, not slow work.
const rootQueryTimeout = 500 * time.Millisecond

// RootCacheTTL is how long one root decision stands before the provider asks
// the client again. It trades a round trip per tool call against how quickly a
// worktree switch is noticed: at two seconds a burst of calls costs one query,
// and a switch is picked up by the call after next at the latest. [assumption]
//
// Exported because the in-process integration suite waits out this window and
// must read the value rather than retype it.
const RootCacheTTL = 2 * time.Second

// sessionRootProvider resolves the project root per tool call, following the
// working directories the client reports over roots/list
// (project-root-resolution.spec). It is the tools.RootProvider of a server that
// was not pinned with --project or ARCHCORE_PROJECT_ROOT.
//
// A reported root is untrusted exactly as a spawn-time working directory is:
// it passes the plugin-cache guard, must hold .archcore/, and must carry
// declared global sources that resolve, before the provider will move onto it.
type sessionRootProvider struct {
	pinned bool
	// warnTo receives one line per distinct refusal reason. It is stderr in
	// production: fd 1 carries JSON-RPC frames.
	warnTo io.Writer

	// refreshMu admits one refresh at a time. It is separate from mu because a
	// refresh spans a client round trip and an acceptance walk over every
	// declared global source of every candidate, and mu may not be held across
	// either: every other tool worker reads the root through it.
	refreshMu sync.Mutex

	mu       sync.Mutex
	current  string
	asked    time.Time
	warned   map[string]bool
	queries  int // roots/list requests issued; read by tests
	adoption int // root changes performed; read by tests
}

// newSessionRootProvider returns a provider serving startRoot. WHEN pinned is
// true it never queries the client (project-root-resolution.spec §1–§2).
func newSessionRootProvider(startRoot string, pinned bool, warnTo io.Writer) *sessionRootProvider {
	if warnTo == nil {
		warnTo = os.Stderr
	}
	return &sessionRootProvider{
		pinned:  pinned,
		warnTo:  warnTo,
		current: startRoot,
		warned:  map[string]bool{},
	}
}

// Root returns the project root this tool call operates on. It never fails: a
// client that cannot be reached, a client that reports nothing usable, and a
// candidate the provider refuses all leave the current root in place.
//
// At most one refresh runs at a time, so two concurrent tool calls issue one
// roots/list request between them (project-root-resolution.spec §9). The second
// call does not queue behind the first: it serves the root it already has,
// because a stale-by-one-call root is the same answer the cache window would
// have given it a moment earlier, and waiting would put a filesystem walk on a
// tool call that asked for nothing.
func (p *sessionRootProvider) Root(ctx context.Context) string {
	// pinned is written once, before the provider is published to any caller.
	if p.pinned {
		return p.settledRoot()
	}

	current, fresh := p.cached()
	if fresh {
		return current
	}
	if !p.refreshMu.TryLock() {
		return current // a refresh is already in flight
	}
	defer p.refreshMu.Unlock()

	// The window may have been refilled by the refresh we just missed.
	if current, fresh = p.cached(); fresh {
		return current
	}
	return p.refresh(ctx, current)
}

// settledRoot reads the current root without deciding anything.
func (p *sessionRootProvider) settledRoot() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

// cached reports the current root and whether its decision still stands.
func (p *sessionRootProvider) cached() (root string, fresh bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current, !p.asked.IsZero() && time.Since(p.asked) < RootCacheTTL
}

// refresh asks the client where it is working and settles on the root to serve.
// It runs under refreshMu and holds mu only to read or publish a field, never
// across the query or the acceptance walk.
func (p *sessionRootProvider) refresh(ctx context.Context, current string) string {
	session := server.ClientSessionFromContext(ctx)
	if session == nil {
		return current // no session on this context: nothing to ask
	}
	if info, ok := session.(server.SessionWithClientInfo); ok {
		if caps := info.GetClientCapabilities(); caps.Roots == nil {
			// The host does not implement roots. Asking would buy a guaranteed
			// timeout on every call, so the question is never put.
			p.markAsked()
			return current
		}
	}
	roots, ok := session.(server.SessionWithRoots)
	if !ok {
		p.markAsked()
		return current
	}

	p.countQuery()
	queryCtx, cancel := context.WithTimeout(ctx, rootQueryTimeout)
	defer cancel()
	result, err := roots.ListRoots(queryCtx, mcp.ListRootsRequest{})
	// Stamped after the call, not before: a query that spends the whole
	// rootQueryTimeout would otherwise shorten the next window by its own
	// duration, and project-root-resolution.spec §7 gives a failed query the
	// same lifetime as a successful one.
	p.markAsked()
	if err != nil {
		p.warnOnce("root-query", "the host did not answer roots/list; keeping the project root this server started on")
		return current
	}

	next := p.choose(current, result.Roots)
	p.mu.Lock()
	if next != p.current {
		p.current = next
		p.adoption++
	}
	p.mu.Unlock()
	return next
}

// choose picks the root to serve next. Exactly one acceptable candidate is
// adopted; zero or several leave the current root in place
// (project-root-resolution.spec §11–§14).
func (p *sessionRootProvider) choose(current string, reported []mcp.Root) string {
	var accepted []string
	for _, r := range reported {
		dir, err := fileURIPath(r.URI)
		if err != nil {
			p.warnOnce("root-uri", "the host reported a working directory that is not a usable file:// path; keeping the current project root")
			continue
		}
		if dir == current {
			// Already serving it. Re-running the acceptance checks would walk
			// every declared global source on a call that changes nothing (§10).
			return current
		}
		if reason, ok := acceptRoot(dir); !ok {
			p.warnOnce("root-"+reason, "not switching to the working directory the host reported: "+reason)
			continue
		}
		accepted = append(accepted, dir)
	}

	switch len(accepted) {
	case 0:
		return current
	case 1:
		return accepted[0]
	default:
		p.warnOnce("root-ambiguous", "the host reported several usable Archcore projects; keeping the current project root")
		return current
	}
}

// markAsked opens a fresh cache window.
func (p *sessionRootProvider) markAsked() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.asked = time.Now()
}

// countQuery records one roots/list request.
func (p *sessionRootProvider) countQuery() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queries++
}

// warnOnce writes one line per distinct reason for the life of the process. A
// per-call line would repeat on every tool call of a session that is simply
// running against a host the provider cannot follow.
func (p *sessionRootProvider) warnOnce(reason, message string) {
	p.mu.Lock()
	if p.warned[reason] {
		p.mu.Unlock()
		return
	}
	p.warned[reason] = true
	p.mu.Unlock()

	// The write error is dropped: an unwritable stderr costs the line and
	// nothing else. Rendered as the surrounding startup lines are, because this
	// is the same stderr stream cmd/mcp.go writes its banner to.
	_, _ = fmt.Fprintln(p.warnTo, display.Dim.Render("  "+message))
}

// acceptRoot reports whether dir may become the project root, and names the
// failed check otherwise. The reason never embeds a filesystem path
// (no-absolute-paths-in-mcp-errors.rule); a declared source id is named
// because the user wrote it.
//
// A guard, not an advisory (fail-open-or-fail-closed-reads.rule §4): every stat
// and every globals read below refuses the candidate on error, because a root
// the provider cannot verify is a root it must not move onto.
func acceptRoot(dir string) (reason string, ok bool) {
	if !filepath.IsAbs(dir) {
		return "the reported path is not absolute", false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "the reported path is not an existing directory", false
	}
	if projectroot.IsPluginCachePath(dir) {
		return "the reported path is inside an AI-host plugin install cache, not a user project", false
	}
	archcoreDir, err := os.Stat(filepath.Join(dir, ".archcore"))
	if err != nil || !archcoreDir.IsDir() {
		return "the reported directory holds no .archcore/ directory", false
	}
	inspections, err := docs.InspectGlobals(dir)
	if err != nil {
		return "the reported project has an unreadable .archcore/settings.json", false
	}
	for _, in := range inspections {
		if in.State.Fatal() {
			// The id alone, never in.Message(). GlobalInspection.Path is the
			// declaration verbatim, and config.LoadGlobals returns it verbatim,
			// so a source declared with an absolute path puts that path into the
			// message — which this line may not carry
			// (project-root-resolution.spec §17).
			return fmt.Sprintf("the reported project declares a global source %q that does not resolve", in.ID), false
		}
	}
	return "", true
}

// fileURIPath converts a file:// URI into a local absolute path. Percent
// escapes are decoded by url.Parse; a Windows URI carries the drive letter
// behind a separator ("file:///C:/repo"), which is stripped here.
func fileURIPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", errors.New("not a file URI")
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", errors.New("file URI names a remote host")
	}
	// The drive-letter strip is a build-time split, not a runtime branch
	// (platform-splits-are-files.rule §1): on a POSIX host "/C:/repo" is an
	// ordinary directory name and must survive intact.
	path := stripURIDrivePrefix(u.Path)
	if path == "" {
		return "", errors.New("file URI carries no path")
	}
	return filepath.Clean(filepath.FromSlash(path)), nil
}
