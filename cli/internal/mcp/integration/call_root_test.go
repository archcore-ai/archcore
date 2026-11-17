package integration

// In-process coverage of call-root-resolution.spec: the project argument and
// the path-derived root of search_documents, through the real server, the real
// registration, and the real JSON on the wire.

import (
	"bytes"
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	mcpserver "archcore-cli/internal/mcp"
	"archcore-cli/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// workspace is a directory that holds several sibling projects, the shape of
// a session opened in ~/src or in a monorepo parent.
type workspace struct{ dir string }

func newWorkspace(t *testing.T) workspace {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return workspace{dir: dir}
}

// project creates rel under the workspace with an initialized .archcore/.
func (w workspace) project(t *testing.T, rel string) string {
	t.Helper()
	dir := filepath.Join(w.dir, rel)
	writeFixtureFile(t, filepath.Join(dir, ".archcore", "settings.json"), `{"sync":"none"}`)
	return dir
}

// plain creates rel under the workspace without .archcore/.
func (w workspace) plain(t *testing.T, rel string) string {
	t.Helper()
	dir := filepath.Join(w.dir, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// projectOf returns the project field of a successful response.
func projectOf(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	project, _ := decodeJSON[map[string]any](t, result)["project"].(string)
	return project
}

// expectRefusal asserts a tool error that contains want and leaks no path of
// the workspace (no-absolute-paths-in-mcp-errors.rule).
func expectRefusal(t *testing.T, c *client.Client, w workspace, name string, args map[string]any, want string) {
	t.Helper()
	text := expectToolError(t, c, name, args)
	if !strings.Contains(text, want) {
		t.Errorf("%s error = %q, want it to contain %q", name, text, want)
	}
	if strings.Contains(text, w.dir) {
		t.Errorf("%s error %q embeds an absolute path", name, text)
	}
	if strings.Contains(text, `"project"`) {
		t.Errorf("%s error %q carries a project field", name, text)
	}
}

func adrArgs(slug string, extra map[string]any) map[string]any {
	args := map[string]any{"type": "adr", "filename": slug, "title": "Doc " + slug}
	maps.Copy(args, extra)
	return args
}

// TestCallRoot_EveryToolTakesProject holds the Surface of the contract: every
// registered tool, install_host_config included, carries an optional project.
func TestCallRoot_EveryToolTakesProject(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	srv := mcpserver.NewServer(base, "test", mcpserver.WithHostWiring(func(string, string, bool) ([]byte, error) { return []byte(`{}`), nil }))
	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatal(err)
	}
	startClient(t, c)
	listed, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 11 {
		t.Fatalf("server lists %d tools, want 11", len(listed.Tools))
	}
	for _, tool := range listed.Tools {
		if _, ok := tool.InputSchema.Properties[tools.ProjectArgument]; !ok {
			t.Errorf("%s has no project argument", tool.Name)
		}
		if slices.Contains(tool.InputSchema.Required, tools.ProjectArgument) {
			t.Errorf("%s requires project; it must stay optional", tool.Name)
		}
	}
}

// TestCallRoot_NoProjectIsBackwardCompatible holds the first invariant: a call
// without project behaves as before. Every field an existing client reads is
// unchanged; the response gains only the project field, naming the session root.
func TestCallRoot_NoProjectIsBackwardCompatible(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	c := newTestClient(t, a)

	created := mustCallTool(t, c, "create_document", adrArgs("x", nil))
	payload := decodeJSON[map[string]any](t, created)
	if payload["path"] != ".archcore/x.adr.md" || payload["project"] != a {
		t.Fatalf("create_document = %v, want the relative path and project %q", payload, a)
	}
	if !documentExists(t, a, ".archcore/x.adr.md") {
		t.Fatal("the document did not land in the session project")
	}
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"list_documents", nil},
		{"get_document", map[string]any{"path": ".archcore/x.adr.md"}},
		{"search_documents", map[string]any{"types": []string{"adr"}}},
		{"search_documents", map[string]any{"for_path": "src/main.go"}},
		{"list_relations", nil},
		{"update_document", map[string]any{"path": ".archcore/x.adr.md", "title": "Renamed"}},
		{"init_project", nil},
	} {
		if got := projectOf(t, mustCallTool(t, c, call.name, call.args)); got != a {
			t.Errorf("%s project = %q, want the session root %q", call.name, got, a)
		}
	}
	docs := decodeListDocuments(t, mustCallTool(t, c, "list_documents", nil))
	if len(docs) != 1 || docs[0].Path != ".archcore/x.adr.md" {
		t.Errorf("list_documents = %+v, want the one relative document", docs)
	}
}

// TestCallRoot_SessionInAWritesB is the Given/When/Then of the contract and the
// cross-project scenario: a session in A writes, reads, edits, relates, and
// removes documents in B, and A is untouched.
func TestCallRoot_SessionInAWritesB(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	c := newTestClient(t, a)

	created := mustCallTool(t, c, "create_document", adrArgs("x", map[string]any{"project": b}))
	payload := decodeJSON[map[string]any](t, created)
	if payload["path"] != ".archcore/x.adr.md" || payload["project"] != b {
		t.Fatalf("create_document = %v, want path .archcore/x.adr.md and project %q", payload, b)
	}
	if !documentExists(t, b, ".archcore/x.adr.md") || documentExists(t, a, ".archcore/x.adr.md") {
		t.Fatal("the document did not land in B only")
	}
	mustCallTool(t, c, "create_document", adrArgs("y", map[string]any{"project": b}))

	got := mustCallTool(t, c, "get_document", map[string]any{"project": b, "path": ".archcore/x.adr.md"})
	if projectOf(t, got) != b {
		t.Errorf("get_document project = %q, want %q", projectOf(t, got), b)
	}
	expectToolError(t, c, "get_document", map[string]any{"path": ".archcore/x.adr.md"})

	mustCallTool(t, c, "update_document", map[string]any{"project": b, "path": ".archcore/x.adr.md", "title": "Edited in B"})
	body, err := os.ReadFile(filepath.Join(b, ".archcore", "x.adr.md"))
	if err != nil || !strings.Contains(string(body), "Edited in B") {
		t.Errorf("update_document did not edit B: %v %s", err, body)
	}

	mustCallTool(t, c, "add_relation", map[string]any{"project": b, "source": "x.adr.md", "target": "y.adr.md", "type": "related"})
	if len(loadManifest(t, b).Relations) != 1 {
		t.Error("the relation is not in B's manifest")
	}
	if _, err := os.Stat(filepath.Join(a, ".archcore", ".sync-state.json")); err == nil && len(loadManifest(t, a).Relations) != 0 {
		t.Error("the relation leaked into A's manifest")
	}
	rels := decodeJSON[map[string]any](t, mustCallTool(t, c, "list_relations", map[string]any{"project": b}))
	if rels["project"] != b {
		t.Errorf("list_relations project = %v, want %q", rels["project"], b)
	}
	mustCallTool(t, c, "remove_relation", map[string]any{"project": b, "source": "x.adr.md", "target": "y.adr.md", "type": "related"})
	if len(loadManifest(t, b).Relations) != 0 {
		t.Error("remove_relation left the relation in B")
	}

	listedB := decodeListDocuments(t, mustCallTool(t, c, "list_documents", map[string]any{"project": b}))
	listedA := decodeListDocuments(t, mustCallTool(t, c, "list_documents", nil))
	if len(listedB) != 2 || len(listedA) != 0 {
		t.Errorf("list_documents: B has %d, A has %d; want 2 and 0", len(listedB), len(listedA))
	}

	mustCallTool(t, c, "remove_document", map[string]any{"project": b, "path": ".archcore/y.adr.md"})
	if documentExists(t, b, ".archcore/y.adr.md") {
		t.Error("remove_document left the document in B")
	}
	if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != a {
		t.Errorf("after the B calls, a call without project serves %q, want %q", got, a)
	}
}

// TestCallRoot_ProjectOverridesAPinnedRoot holds §2 for the Cursor and Copilot
// shape: a server pinned with --project still reaches another project.
func TestCallRoot_ProjectOverridesAPinnedRoot(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	handler := &fakeRoots{}
	handler.set(b)
	var warnings bytes.Buffer
	c := newPinnedClient(t, a, handler, &warnings)

	mustCallTool(t, c, "create_document", adrArgs("x", map[string]any{"project": b}))
	if !documentExists(t, b, ".archcore/x.adr.md") {
		t.Fatal("the pinned server did not write to the named project")
	}
	if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != a {
		t.Errorf("a call without project serves %q, want the pinned root %q", got, a)
	}
	if handler.callCount() != 0 {
		t.Errorf("a pinned server queried roots/list %d times", handler.callCount())
	}
}

// TestCallRoot_ProjectOverridesTheSessionRoot holds §2 and §17 against a
// server that follows roots/list.
func TestCallRoot_ProjectOverridesTheSessionRoot(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	start := w.project(t, "start")
	reported := w.project(t, "reported")
	other := w.project(t, "other")
	handler := &fakeRoots{}
	handler.set(reported)
	var warnings bytes.Buffer
	c := newRootsClient(t, start, handler, &warnings)

	if got := projectOf(t, mustCallTool(t, c, "list_documents", map[string]any{"project": other})); got != other {
		t.Errorf("project call served %q, want %q", got, other)
	}
	if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != reported {
		t.Errorf("a call without project served %q, want the reported root %q", got, reported)
	}
}

// TestCallRoot_ForPathSelectsItsProject holds §5 and §6 and the reading side of
// the cross-project scenario: a file path names the project whose rules apply.
func TestCallRoot_ForPathSelectsItsProject(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	nested := w.project(t, "a/app")
	for _, p := range []string{a, b, nested} {
		writeFixtureFile(t, filepath.Join(p, "src", "api.go"), "package src\n")
	}
	writeFixtureFile(t, filepath.Join(b, ".archcore", "b-errors.rule.md"),
		"---\ntitle: \"B errors\"\nstatus: accepted\n---\n\n## Rule\n\nWrap errors.\n")
	writeFixtureFile(t, filepath.Join(nested, ".archcore", "app-only.rule.md"),
		"---\ntitle: \"App only\"\nstatus: accepted\n---\n\n## Rule\n\nApp rule.\n")
	c := newTestClient(t, a)

	type row struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	}
	type envelope struct {
		Project string `json:"project"`
		Results []row  `json:"results"`
	}
	for _, tt := range []struct {
		name, forPath, wantProject, wantRow string
	}{
		{"file of B", filepath.Join(b, "src", "api.go"), b, ".archcore/b-errors.rule.md"},
		{"file of a project nested in A", filepath.Join(nested, "src", "api.go"), nested, ".archcore/app-only.rule.md"},
		{"absolute file of A", filepath.Join(a, "src", "api.go"), a, ""},
		{"relative file of A", "src/api.go", a, ""},
	} {
		got := decodeJSON[envelope](t, mustCallTool(t, c, "search_documents", map[string]any{"for_path": tt.forPath}))
		if got.Project != tt.wantProject {
			t.Errorf("%s: project = %q, want %q", tt.name, got.Project, tt.wantProject)
		}
		if tt.wantRow != "" && !slices.ContainsFunc(got.Results, func(r row) bool { return r.Path == tt.wantRow }) {
			t.Errorf("%s: rows %+v lack %s", tt.name, got.Results, tt.wantRow)
		}
		if tt.wantRow == "" && len(got.Results) != 0 {
			t.Errorf("%s: rows %+v leaked from another project", tt.name, got.Results)
		}
	}

	outside := w.plain(t, "notes")
	expectRefusal(t, c, w, "search_documents", map[string]any{"for_path": filepath.Join(outside, "n.md")}, "pass project")
	expectRefusal(t, c, w, "search_documents", map[string]any{"project": b, "for_path": filepath.Join(a, "src", "api.go")}, "for_path must name a file inside the project")
	expectRefusal(t, c, w, "search_documents", map[string]any{"for_path": "../b/src/api.go"}, "for_path must name a file inside the project")
}

// TestCallRoot_Refusals holds Failure Behavior 1–5 and 7 through the server.
func TestCallRoot_Refusals(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	plain := w.plain(t, "plain")
	cache := w.project(t, ".cursor/plugins/archcore")
	fatal := w.project(t, "fatal")
	writeFixtureFile(t, filepath.Join(fatal, ".archcore", "settings.json"),
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
	writeFixtureFile(t, filepath.Join(w.dir, "company", ".archcore"), "not a directory")
	c := newTestClient(t, a)

	for _, tt := range []struct {
		name    string
		project string
		want    string
	}{
		{"relative project", "b", "must be an absolute path"},
		{"missing project", filepath.Join(w.dir, "missing"), "is not an existing directory"},
		{".archcore as project", filepath.Join(b, ".archcore"), "not .archcore/ itself"},
		{"plugin cache", cache, "plugin install cache"},
		{"no .archcore", plain, "init_project"},
		{"misconfigured global", fatal, `global source "company"`},
	} {
		for _, tool := range []string{"list_documents", "create_document", "search_documents", "get_document", "update_document", "remove_document", "add_relation", "remove_relation", "list_relations"} {
			args := map[string]any{"project": tt.project, "type": "adr", "filename": "x", "title": "x", "path": ".archcore/x.adr.md", "source": "x.adr.md", "target": "y.adr.md", "types": []string{"adr"}}
			expectRefusal(t, c, w, tool, args, tt.want)
		}
	}
	if _, err := os.Stat(filepath.Join(plain, ".archcore")); !os.IsNotExist(err) {
		t.Error("a refused call created .archcore/ in a directory without one")
	}
	expectRefusal(t, c, w, "add_relation", map[string]any{
		"source": filepath.Join(a, ".archcore", "x.adr.md"),
		"target": filepath.Join(b, ".archcore", "y.adr.md"),
		"type":   "related",
	}, "relation stays inside one project")
	if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != a {
		t.Errorf("after refusals the session root is %q, want %q", got, a)
	}
}

// TestCallRoot_InitProject holds §12 and the mixed case: only an explicit
// init_project with project creates .archcore/ in a second project.
func TestCallRoot_InitProject(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	fresh := w.plain(t, "fresh")
	cacheDir := w.plain(t, ".claude/plugins/cache/x")
	c := newTestClient(t, a)

	result := decodeJSON[map[string]any](t, mustCallTool(t, c, "init_project", map[string]any{"project": fresh}))
	if result["project"] != fresh || result["already_initialized"] != false {
		t.Fatalf("init_project = %v, want a fresh initialization of %q", result, fresh)
	}
	if _, err := os.Stat(filepath.Join(fresh, ".archcore", "settings.json")); err != nil {
		t.Fatalf("init_project did not create .archcore/ under project: %v", err)
	}
	again := decodeJSON[map[string]any](t, mustCallTool(t, c, "init_project", map[string]any{"project": fresh}))
	if again["already_initialized"] != true {
		t.Errorf("a second init_project = %v, want already_initialized", again)
	}
	mustCallTool(t, c, "create_document", adrArgs("x", map[string]any{"project": fresh}))

	expectRefusal(t, c, w, "init_project", map[string]any{"project": cacheDir}, "plugin install cache")
	expectRefusal(t, c, w, "init_project", map[string]any{"project": filepath.Join(w.dir, "missing")}, "is not an existing directory")
	expectRefusal(t, c, w, "init_project", map[string]any{"project": "fresh"}, "must be an absolute path")
	if _, err := os.Stat(filepath.Join(cacheDir, ".archcore")); !os.IsNotExist(err) {
		t.Error("init_project created .archcore/ inside a plugin cache")
	}
}

// TestCallRoot_SymlinkSpellingsAreOneProject holds §4 and the invariant that
// two spellings of one directory are one root.
func TestCallRoot_SymlinkSpellingsAreOneProject(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	link := filepath.Join(w.dir, "b-link")
	if err := os.Symlink(b, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	c := newTestClient(t, a)

	mustCallTool(t, c, "create_document", adrArgs("x", map[string]any{"project": link}))
	viaReal := decodeListDocuments(t, mustCallTool(t, c, "list_documents", map[string]any{"project": b}))
	if len(viaReal) != 1 {
		t.Fatalf("a document written through the symlink is not listed through the target: %+v", viaReal)
	}
	if got := projectOf(t, mustCallTool(t, c, "list_documents", map[string]any{"project": link})); got != b {
		t.Errorf("project through the symlink = %q, want the evaluated root %q", got, b)
	}
}

// TestCallRoot_ConcurrentCallsKeepTheirRoots holds §8 and the cache invariant
// under load: interleaved calls to several projects never answer from another
// project's scan cache or manifest.
func TestCallRoot_ConcurrentCallsKeepTheirRoots(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	projects := []string{a, w.project(t, "b"), w.project(t, "c")}
	c := newTestClient(t, a)
	for i, p := range projects {
		for j := 0; j <= i; j++ {
			mustCallTool(t, c, "create_document", adrArgs(string(rune('p'+j)), map[string]any{"project": p}))
		}
	}

	var wg sync.WaitGroup
	errs := make(chan string, 90)
	for range 30 {
		for i, p := range projects {
			wg.Go(func() {
				req := mcp.CallToolRequest{}
				req.Params.Name = "list_documents"
				req.Params.Arguments = map[string]any{"project": p}
				res, err := c.CallTool(context.Background(), req)
				if err != nil || res.IsError {
					errs <- "call failed"
					return
				}
				payload := firstText(res)
				if !strings.Contains(payload, `"project":`+quote(p)) || strings.Count(payload, `"path":`) != i+1 {
					errs <- "cross-talk: " + p + " got " + payload
				}
			})
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}

// TestCallRoot_ForPathStaysInsideItsBudget: the project field the server
// appends counts against the 8 000-byte for_path ceiling of
// file-context-resolution.spec, however long the root path is.
func TestCallRoot_ForPathStaysInsideItsBudget(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	deep := w.project(t, strings.Repeat("very-long-directory-name/", 8)+"b")
	writeFixtureFile(t, filepath.Join(deep, "src", "api.go"), "package src\n")
	for i := range 40 {
		name := strings.Repeat("r", 10) + string(rune('a'+i%26)) + string(rune('a'+i/26))
		writeFixtureFile(t, filepath.Join(deep, ".archcore", name+".rule.md"),
			"---\ntitle: \""+strings.Repeat("A long rule title ", 8)+"\"\nstatus: accepted\n---\n\n## Rule\n\nx\n")
	}
	c := newTestClient(t, a)
	for _, args := range []map[string]any{
		{"for_path": filepath.Join(deep, "src", "api.go")},
		{"project": deep, "for_path": "src/api.go"},
	} {
		text := firstText(mustCallTool(t, c, "search_documents", args))
		if len(text) > 8_000 {
			t.Errorf("for_path response is %d bytes, over the 8000-byte ceiling", len(text))
		}
		if !strings.Contains(text, `"project":`+quote(deep)) {
			t.Errorf("for_path response lacks the project field")
		}
	}
}
