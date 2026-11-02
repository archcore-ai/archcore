package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

// realPath returns dir with symlinks evaluated: on macOS t.TempDir sits under
// /var, a symlink to /private/var, and the resolver serves the evaluated path.
func realPath(t *testing.T, dir string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", dir, err)
	}
	return real
}

// callRootFixture lays out a workspace: two projects, a nested project inside
// the first, a directory without .archcore/, a plugin cache that holds
// .archcore/, a project whose global source is misconfigured, and a project
// whose settings.json is unreadable.
type callRootFixture struct {
	workspace, a, nested, b, plain, cache, fatal, unreadable string
}

func newCallRootFixture(t *testing.T) callRootFixture {
	t.Helper()
	w := realPath(t, t.TempDir())
	f := callRootFixture{
		workspace:  w,
		a:          filepath.Join(w, "a"),
		nested:     filepath.Join(w, "a", "app"),
		b:          filepath.Join(w, "b"),
		plain:      filepath.Join(w, "plain"),
		cache:      filepath.Join(w, ".claude", "plugins", "cache", "archcore"),
		fatal:      filepath.Join(w, "fatal"),
		unreadable: filepath.Join(w, "unreadable"),
	}
	for _, dir := range []string{f.a, f.nested, f.b, f.cache, f.fatal, f.unreadable} {
		mkdirAll(t, filepath.Join(dir, ".archcore"))
	}
	mkdirAll(t, filepath.Join(f.a, "src"))
	mkdirAll(t, filepath.Join(f.nested, "src"))
	mkdirAll(t, filepath.Join(f.b, "src"))
	mkdirAll(t, filepath.Join(f.plain, "src"))
	writeFile(t, filepath.Join(f.fatal, ".archcore", "settings.json"),
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
	mkdirAll(t, filepath.Join(w, "company"))
	writeFile(t, filepath.Join(w, "company", ".archcore"), "not a directory")
	writeFile(t, filepath.Join(f.unreadable, ".archcore", "settings.json"), "{not json")
	return f
}

func requestWith(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

func TestResolveProjectArgument(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	link := filepath.Join(f.workspace, "b-link")
	if err := os.Symlink(f.b, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	tests := []struct {
		name         string
		project      string
		initializing bool
		want         string
		wantRefusal  string
	}{
		{name: "a project with .archcore", project: f.b, want: f.b},
		{name: "trailing separator", project: f.b + "/", want: f.b},
		{name: "symlink to a project serves the target", project: link, want: f.b},
		{name: "nested project", project: f.nested, want: f.nested},
		{name: "relative path", project: "b", wantRefusal: "must be an absolute path"},
		{name: "dot-relative path", project: "./b", wantRefusal: "must be an absolute path"},
		{name: "missing directory", project: filepath.Join(f.workspace, "missing"), wantRefusal: "is not an existing directory"},
		{name: "a file, not a directory", project: filepath.Join(f.workspace, "company", ".archcore"), wantRefusal: "not .archcore/ itself"},
		{name: "the .archcore directory itself", project: filepath.Join(f.b, ".archcore"), wantRefusal: "not .archcore/ itself"},
		{name: "plugin install cache", project: f.cache, wantRefusal: "plugin install cache"},
		{name: "no .archcore", project: f.plain, wantRefusal: "call init_project with the same project argument"},
		{name: "misconfigured global source", project: f.fatal, wantRefusal: `global source "company"`},
		{name: "unreadable settings.json", project: f.unreadable, wantRefusal: "unreadable .archcore/settings.json"},
		{name: "init_project accepts a directory without .archcore", project: f.plain, initializing: true, want: f.plain},
		{name: "init_project still refuses a plugin cache", project: f.cache, initializing: true, wantRefusal: "plugin install cache"},
		{name: "init_project still refuses a missing directory", project: filepath.Join(f.workspace, "missing"), initializing: true, wantRefusal: "is not an existing directory"},
		{name: "init_project still refuses a relative path", project: "plain", initializing: true, wantRefusal: "must be an absolute path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, refusal := resolveProjectArgument(tt.project, tt.initializing)
			if tt.wantRefusal != "" {
				if got != "" || !strings.Contains(refusal, tt.wantRefusal) {
					t.Fatalf("resolveProjectArgument(%q) = %q, %q; want a refusal containing %q", tt.project, got, refusal, tt.wantRefusal)
				}
				if strings.Contains(refusal, f.workspace) {
					t.Errorf("refusal %q embeds an absolute path", refusal)
				}
				return
			}
			if refusal != "" || got != tt.want {
				t.Fatalf("resolveProjectArgument(%q) = %q, %q; want %q", tt.project, got, refusal, tt.want)
			}
		})
	}
}

func TestResolveForPathRoot(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	uninitialized := filepath.Join(f.plain)

	tests := []struct {
		name        string
		session     string
		forPath     string
		want        string
		wantRefusal string
	}{
		{name: "empty for_path keeps the session root", session: f.a, forPath: "", want: f.a},
		{name: "relative for_path keeps the session root", session: f.a, forPath: "src/x.go", want: f.a},
		{name: "relative for_path that escapes stays with the session root", session: f.a, forPath: "../b/src/x.go", want: f.a},
		{name: "absolute for_path inside the session root", session: f.a, forPath: filepath.Join(f.a, "src", "x.go"), want: f.a},
		{name: "absolute for_path in another project", session: f.a, forPath: filepath.Join(f.b, "src", "x.go"), want: f.b},
		{name: "absolute for_path in a nested project", session: f.a, forPath: filepath.Join(f.nested, "src", "x.go"), want: f.nested},
		{name: "absolute for_path in the parent of a nested session", session: f.nested, forPath: filepath.Join(f.a, "src", "x.go"), want: f.a},
		{name: "absolute for_path outside every project", session: f.a, forPath: filepath.Join(f.plain, "src", "x.go"), wantRefusal: "pass project"},
		{name: "uninitialized session root answers as before", session: uninitialized, forPath: filepath.Join(f.plain, "src", "x.go"), want: uninitialized},
		{name: "absolute for_path inside a plugin cache", session: f.a, forPath: filepath.Join(f.cache, "skills", "x.md"), wantRefusal: "plugin install cache"},
		{name: "absolute for_path in a project with a broken global", session: f.a, forPath: filepath.Join(f.fatal, "x.go"), wantRefusal: `global source "company"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, refusal := resolveForPathRoot(tt.session, tt.forPath)
			if tt.wantRefusal != "" {
				if got != "" || !strings.Contains(refusal, tt.wantRefusal) {
					t.Fatalf("resolveForPathRoot(%q) = %q, %q; want a refusal containing %q", tt.forPath, got, refusal, tt.wantRefusal)
				}
				if strings.Contains(refusal, f.workspace) {
					t.Errorf("refusal %q embeds an absolute path", refusal)
				}
				return
			}
			if refusal != "" || got != tt.want {
				t.Fatalf("resolveForPathRoot(%q) = %q, %q; want %q", tt.forPath, got, refusal, tt.want)
			}
		})
	}
}

// TestResolveForPathRoot_KeepsTheSessionSpelling: a for_path written through
// the evaluated path of the session root still serves the session root as the
// session spells it, so one project keeps one cache key.
func TestResolveForPathRoot_KeepsTheSessionSpelling(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	link := filepath.Join(f.workspace, "a-link")
	if err := os.Symlink(f.a, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, refusal := resolveForPathRoot(link, filepath.Join(f.a, "src", "x.go"))
	if refusal != "" || got != link {
		t.Errorf("resolveForPathRoot = %q, %q; want the session spelling %q", got, refusal, link)
	}
}

// TestResolveForPathRoot_AnotherProjectIsCanonical: a for_path written through
// a symlink to another project serves the evaluated root, the same root a
// project argument through that symlink serves, so one project keeps one
// spelling on the wire and one cache key.
func TestResolveForPathRoot_AnotherProjectIsCanonical(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	link := filepath.Join(f.workspace, "b-link")
	if err := os.Symlink(f.b, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	byPath, refusal := resolveForPathRoot(f.a, filepath.Join(link, "src", "x.go"))
	if refusal != "" || byPath != f.b {
		t.Fatalf("resolveForPathRoot through a symlink = %q, %q; want %q", byPath, refusal, f.b)
	}
	byProject, refusal := resolveProjectArgument(link, false)
	if refusal != "" || byProject != byPath {
		t.Errorf("project through the symlink = %q, %q; want the for_path root %q", byProject, refusal, byPath)
	}
}

func TestRelationRootRefusal(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	inA := filepath.Join(f.a, ".archcore", "x.adr.md")
	inB := filepath.Join(f.b, ".archcore", "y.spec.md")

	tests := []struct {
		name           string
		root           string
		source, target string
		wantRefused    bool
	}{
		{name: "two relative paths", root: f.a, source: "x.adr.md", target: ".archcore/y.spec.md"},
		{name: "absolute paths under the call root", root: f.a, source: inA, target: filepath.Join(f.a, ".archcore", "z.md")},
		{name: "absolute source in another project", root: f.a, source: inB, target: "x.adr.md", wantRefused: true},
		{name: "absolute target in another project", root: f.a, source: "x.adr.md", target: inB, wantRefused: true},
		{name: "source in A and target in B", root: f.a, source: inA, target: inB, wantRefused: true},
		{name: "absolute path in a nested project", root: f.a, source: filepath.Join(f.nested, ".archcore", "n.md"), target: "x.adr.md", wantRefused: true},
		{name: "absolute path outside every project", root: f.a, source: filepath.Join(f.plain, "n.md"), target: "x.adr.md", wantRefused: true},
		{name: "project B with B's absolute paths", root: f.b, source: inB, target: filepath.Join(f.b, ".archcore", "w.md")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			refusal := relationRootRefusal(tt.root, requestWith(map[string]any{"source": tt.source, "target": tt.target}))
			if tt.wantRefused != (refusal != "") {
				t.Fatalf("relationRootRefusal = %q, want refused %v", refusal, tt.wantRefused)
			}
			if tt.wantRefused && refusal != crossProjectRelationRefusal {
				t.Errorf("refusal = %q, want %q", refusal, crossProjectRelationRefusal)
			}
		})
	}
}

// TestCallRootResolver_Precedence holds call-root-resolution.spec §1, §2, §5,
// §7: project, then an absolute for_path, then the session root.
func TestCallRootResolver_Precedence(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	r := callRootResolver{session: tools.StaticRoot(f.a)}

	tests := []struct {
		name        string
		tool        string
		args        map[string]any
		want        string
		wantRefusal string
	}{
		{name: "no arguments serve the session root", tool: "list_documents", args: nil, want: f.a},
		{name: "project overrides the session root", tool: "list_documents", args: map[string]any{"project": f.b}, want: f.b},
		{name: "blank project is absent", tool: "list_documents", args: map[string]any{"project": "  "}, want: f.a},
		{name: "project wins over an absolute for_path", tool: "search_documents", args: map[string]any{"project": f.b, "for_path": filepath.Join(f.nested, "x.go")}, want: f.b},
		{name: "absolute for_path selects its project", tool: "search_documents", args: map[string]any{"for_path": filepath.Join(f.b, "src", "x.go")}, want: f.b},
		{name: "for_path is read only by search_documents", tool: "list_documents", args: map[string]any{"for_path": filepath.Join(f.b, "src", "x.go")}, want: f.a},
		{name: "init_project with project without .archcore", tool: "init_project", args: map[string]any{"project": f.plain}, want: f.plain},
		{name: "another tool with project without .archcore", tool: "create_document", args: map[string]any{"project": f.plain}, wantRefusal: "init_project"},
		{name: "relation across projects", tool: "add_relation", args: map[string]any{"source": filepath.Join(f.b, ".archcore", "y.md"), "target": "x.md"}, wantRefusal: "relation stays inside one project"},
		{name: "relation across projects is refused by remove_relation too", tool: "remove_relation", args: map[string]any{"project": f.b, "source": filepath.Join(f.a, ".archcore", "y.md"), "target": "x.md"}, wantRefusal: "relation stays inside one project"},
		{name: "relation inside project B", tool: "add_relation", args: map[string]any{"project": f.b, "source": "y.md", "target": "x.md"}, want: f.b},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, refusal := r.resolve(context.Background(), tt.tool, requestWith(tt.args))
			if tt.wantRefusal != "" {
				if !strings.Contains(refusal, tt.wantRefusal) {
					t.Fatalf("resolve = %q, %q; want a refusal containing %q", got, refusal, tt.wantRefusal)
				}
				return
			}
			if refusal != "" || got != tt.want {
				t.Fatalf("resolve = %q, %q; want %q", got, refusal, tt.want)
			}
		})
	}
}

// settledOnly is a session provider whose Root must not run: deciding the
// session root may wait on roots/list.
type settledOnly struct {
	t    *testing.T
	root string
}

func (s settledOnly) Root(context.Context) string {
	s.t.Error("a call with project decided the session root")
	return s.root
}

func (s settledOnly) settledRoot() string { return s.root }

// TestCallRootResolver_ProjectDoesNotDecideTheSessionRoot holds §2: a call with
// project compares against the settled session root and sends no roots/list.
func TestCallRootResolver_ProjectDoesNotDecideTheSessionRoot(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	r := callRootResolver{session: settledOnly{t: t, root: f.a}}
	for _, project := range []string{f.a, f.b} {
		if got, refusal := r.resolve(context.Background(), "list_documents", requestWith(map[string]any{"project": project})); refusal != "" || got != project {
			t.Errorf("project %q: resolve = %q, %q", project, got, refusal)
		}
	}
}

// TestCallRootResolver_ProjectNamingTheSessionKeepsItsSpelling: a session
// root spelled through a symlink and a project argument naming the same
// directory answer with one spelling.
func TestCallRootResolver_ProjectNamingTheSessionKeepsItsSpelling(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	link := filepath.Join(f.workspace, "a-link")
	if err := os.Symlink(f.a, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r := callRootResolver{session: tools.StaticRoot(link)}
	for _, project := range []string{f.a, link, link + "/"} {
		got, refusal := r.resolve(context.Background(), "list_documents", requestWith(map[string]any{"project": project}))
		if refusal != "" || got != link {
			t.Errorf("project %q: resolve = %q, %q; want the session spelling %q", project, got, refusal, link)
		}
	}
	got, _ := r.resolve(context.Background(), "list_documents", requestWith(map[string]any{"project": f.b}))
	if got != f.b {
		t.Errorf("another project = %q, want %q", got, f.b)
	}
}

// TestCallRootResolver_WrapServesOneRoot holds §8, §10, and §17: the handler
// sees the root the resolver settled, the response names it, and a following
// call without project is back on the session root.
func TestCallRootResolver_WrapServesOneRoot(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	session := tools.StaticRoot(f.a)
	root := callRoot{session: session}
	var seen []string
	handle := func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		seen = append(seen, root.Root(ctx))
		return mcp.NewToolResultText(`{"ok":true}`), nil
	}
	wrapped := callRootResolver{session: session}.wrap("list_documents", handle)

	result, err := wrapped(context.Background(), requestWith(map[string]any{"project": f.b}))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := mcp.AsTextContent(result.Content[0])
	if want := `{"ok":true,"project":` + quoteJSON(f.b) + `}`; text.Text != want {
		t.Errorf("response = %s, want %s", text.Text, want)
	}
	if _, err := wrapped(context.Background(), requestWith(nil)); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[0] != f.b || seen[1] != f.a {
		t.Errorf("handler saw roots %q, want [B A]: a project must not outlive its call", seen)
	}
}

func TestCallRootResolver_WrapRefusesWithoutRunningTheHandler(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	ran := false
	handle := func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ran = true
		return mcp.NewToolResultText(`{}`), nil
	}
	wrapped := callRootResolver{session: tools.StaticRoot(f.a)}.wrap("create_document", handle)
	result, err := wrapped(context.Background(), requestWith(map[string]any{"project": f.plain}))
	if err != nil {
		t.Fatal(err)
	}
	if ran {
		t.Error("the handler ran for a refused project")
	}
	text, _ := mcp.AsTextContent(result.Content[0])
	if !result.IsError || strings.Contains(text.Text, `"project":`) || strings.Contains(text.Text, f.workspace) {
		t.Errorf("refusal = %+v, want an error without a project field or an absolute path", result)
	}
	if _, err := os.Stat(filepath.Join(f.plain, ".archcore")); !os.IsNotExist(err) {
		t.Error("a refused call created .archcore/")
	}
}

func TestCallRootResolver_HandlerErrorKeepsItsResult(t *testing.T) {
	t.Parallel()
	f := newCallRootFixture(t)
	handle := func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("bad input"), nil
	}
	result, err := callRootResolver{session: tools.StaticRoot(f.a)}.wrap("get_document", handle)(context.Background(), requestWith(nil))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := mcp.AsTextContent(result.Content[0])
	if text.Text != "bad input" {
		t.Errorf("an error result was rewritten: %q", text.Text)
	}
}

func TestRootRefusal_NamesNoPath(t *testing.T) {
	t.Parallel()
	for _, err := range []error{errRootNotAbsolute, errRootNotDirectory, errRootPluginCache, errRootNoArchcore, errRootSettingsUnreadable, fatalGlobalError{sourceID: "org"}, os.ErrClosed} {
		msg := rootRefusal("project", err)
		if msg == "" || strings.Contains(msg, string(filepath.Separator)+"Users") || strings.Contains(msg, "/tmp") {
			t.Errorf("rootRefusal(%v) = %q", err, msg)
		}
	}
	if got := rootRefusal("project", os.ErrClosed); !strings.Contains(got, "cannot be verified") {
		t.Errorf("an unknown verdict must refuse, got %q", got)
	}
}

func quoteJSON(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}
