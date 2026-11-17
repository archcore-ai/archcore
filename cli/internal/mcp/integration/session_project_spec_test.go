package integration

// Normative properties of session-project-is-nearest-store.adr, pinned here:
//
//  1. A session below a project root reads and writes that project.
//  2. A session at a project root answers exactly as a session below it does,
//     and carries no field the rule added.
//  3. A root without a store refuses every document write and creates nothing.
//  4. A read on a root without a store succeeds and says "initialized": false.
//  5. init_project from below a root returns the project above; a nested store
//     takes an explicit project.
//  6. A store that appears above the session folder is served by the next call.
//  7. A reported or pinned folder is judged by the project it belongs to.
//  8. Every registered tool is classified for a root without a store.
//  9. A store without settings.json above the session folder, the trace of an
//     implicit create by an earlier CLI, does not capture the folder.

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	mcpserver "archcore-cli/internal/mcp"
)

// storesUnder lists every .archcore directory below dir, as paths relative to
// it. It is the oracle for "this call created no store".
func storesUnder(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".archcore" {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return relErr
			}
			found = append(found, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(found)
	return found
}

func TestSessionProject_BelowARootReadsAndWritesTheProject(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	deep := w.plain(t, "a/src/deep")
	writeFixtureFile(t, filepath.Join(a, "src", "x.go"), "package src\n")
	before := storesUnder(t, w.dir)

	root := newTestClient(t, a)
	createADR(t, root, "", "seed")

	below := newTestClient(t, deep)
	listed := mustCallTool(t, below, "list_documents", nil)
	if got := projectOf(t, listed); got != a {
		t.Fatalf("a session below the root served %q, want the project above %q", got, a)
	}
	if docs := decodeListedDocs(t, listed); len(docs) != 1 {
		t.Fatalf("a session below the root listed %d documents, want the project's 1", len(docs))
	}
	// Property 2: the two sessions answer with the same bytes.
	for name, args := range map[string]map[string]any{
		"list_documents":   nil,
		"search_documents": {"content": "seed"},
		"list_relations":   nil,
		"get_document":     {"path": ".archcore/seed.adr.md"},
	} {
		atRoot := firstText(mustCallTool(t, root, name, args))
		if got := firstText(mustCallTool(t, below, name, args)); got != atRoot {
			t.Errorf("%s differs between the root and a folder below it:\n below: %s\n  root: %s", name, got, atRoot)
		}
		if strings.Contains(atRoot, `"initialized"`) {
			t.Errorf("%s on a project with a store carries the initialized field: %s", name, atRoot)
		}
	}

	created := mustCallTool(t, below, "create_document", adrArgs("from-below", nil))
	if got := projectOf(t, created); got != a {
		t.Errorf("create_document from below the root served %q, want %q", got, a)
	}
	if !documentExists(t, a, filepath.Join(".archcore", "from-below.adr.md")) {
		t.Error("the document did not land in the project above the session folder")
	}
	mustCallTool(t, below, "update_document", map[string]any{"path": ".archcore/from-below.adr.md", "title": "Renamed"})
	mustCallTool(t, below, "add_relation", map[string]any{"source": ".archcore/from-below.adr.md", "target": ".archcore/seed.adr.md", "type": "related"})
	if got := projectOf(t, mustCallTool(t, below, "search_documents", map[string]any{"for_path": "src/x.go"})); got != a {
		t.Errorf("a relative for_path from below the root served %q, want %q", got, a)
	}
	if after := storesUnder(t, w.dir); !slices.Equal(after, before) {
		t.Errorf("stores under the workspace changed from %v to %v: a session below a root created one", before, after)
	}
}

func TestSessionProject_NoStoreRefusesWritesAndCreatesNothing(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	w.project(t, "sibling")
	folder := w.plain(t, "plain/work")
	before := storesUnder(t, w.dir)
	c := newTestClient(t, folder)

	doc := ".archcore/x.adr.md"
	writes := map[string]map[string]any{
		"create_document": adrArgs("x", nil),
		"update_document": {"path": doc, "title": "X"},
		"remove_document": {"path": doc},
		"add_relation":    {"source": doc, "target": ".archcore/y.adr.md", "type": "related"},
		"remove_relation": {"source": doc, "target": ".archcore/y.adr.md", "type": "related"},
	}
	writes["add_relation with an absolute source"] = map[string]any{"source": filepath.Join(folder, doc), "target": ".archcore/y.adr.md", "type": "related"}
	for name, args := range writes {
		tool, _, _ := strings.Cut(name, " ")
		expectRefusal(t, c, w, tool, args, "call init_project after the user asks for it, or pass project")
	}
	if after := storesUnder(t, w.dir); !slices.Equal(after, before) {
		t.Fatalf("stores under the workspace changed from %v to %v: a refused write created one", before, after)
	}

	// Property 4: the reads answer as before, and say why they are empty.
	for name, args := range map[string]map[string]any{
		"list_documents":   nil,
		"search_documents": {"content": "anything"},
		"list_relations":   nil,
	} {
		text := firstText(mustCallTool(t, c, name, args))
		if !strings.HasSuffix(text, `,"initialized":false}`) {
			t.Errorf("%s on a root without a store = %s, want it to end with \"initialized\":false", name, text)
		}
	}
	if text := expectToolError(t, c, "get_document", map[string]any{"path": doc}); !strings.Contains(text, "document not found") {
		t.Errorf("get_document on a root without a store = %q, want the unchanged not-found error", text)
	}

	// The documented flow is unchanged: init_project, then the write.
	initialized := decodeJSON[map[string]any](t, mustCallTool(t, c, "init_project", nil))
	if initialized["already_initialized"] != false || initialized["project"] != folder {
		t.Fatalf("init_project = %v, want a new store in the session folder", initialized)
	}
	if _, has := initialized["initialized"].(bool); !has || initialized["initialized"] != true {
		t.Errorf("init_project lost its own initialized field: %v", initialized)
	}
	mustCallTool(t, c, "create_document", adrArgs("x", nil))
	if text := firstText(mustCallTool(t, c, "list_documents", nil)); strings.Contains(text, `"initialized"`) {
		t.Errorf("list_documents after init_project still carries the initialized field: %s", text)
	}
}

func TestSessionProject_InitFromBelowReturnsTheProjectAbove(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	pkg := w.plain(t, "a/packages/foo")
	c := newTestClient(t, pkg)

	got := decodeJSON[map[string]any](t, mustCallTool(t, c, "init_project", nil))
	if got["already_initialized"] != true || got["project"] != a {
		t.Fatalf("init_project from below a root = %v, want the existing project %q", got, a)
	}
	if stores := storesUnder(t, w.dir); !slices.Equal(stores, []string{"a/.archcore"}) {
		t.Fatalf("stores = %v: init_project from below a root created a nested store", stores)
	}

	nested := decodeJSON[map[string]any](t, mustCallTool(t, c, "init_project", map[string]any{"project": pkg}))
	if nested["already_initialized"] != false || nested["project"] != pkg {
		t.Fatalf("init_project with an explicit project = %v, want a new store in %q", nested, pkg)
	}
	// Property 6, the other direction: the nested store is now the nearest one.
	if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != pkg {
		t.Errorf("after the nested store appeared the session served %q, want %q", got, pkg)
	}
}

func TestSessionProject_AStoreThatAppearsAboveIsServedByTheNextCall(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	parent := w.plain(t, "repo")
	folder := w.plain(t, "repo/pkg")
	c := newTestClient(t, folder)

	expectRefusal(t, c, w, "create_document", adrArgs("early", nil), "init_project")
	mustCallTool(t, c, "init_project", map[string]any{"project": parent})
	created := mustCallTool(t, c, "create_document", adrArgs("late", nil))
	if got := projectOf(t, created); got != parent {
		t.Errorf("the call after the store appeared served %q, want %q", got, parent)
	}
	if stores := storesUnder(t, w.dir); !slices.Equal(stores, []string{"repo/.archcore"}) {
		t.Errorf("stores = %v, want the one the user asked for", stores)
	}
}

func TestSessionProject_AFolderEnteredThroughALinkBelongsWhereTheLinkLeads(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	w.plain(t, "a/src")
	link := filepath.Join(w.dir, "link-into-src")
	if err := os.Symlink(filepath.Join(a, "src"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	c := newTestClient(t, link)
	if got := projectOf(t, mustCallTool(t, c, "create_document", adrArgs("via-link", nil))); got != a {
		t.Errorf("a session entered through a link served %q, want %q", got, a)
	}
	if stores := storesUnder(t, w.dir); !slices.Equal(stores, []string{"a/.archcore"}) {
		t.Errorf("stores = %v: the write created a store beside the link target", stores)
	}
}

func TestSessionProject_ReportedAndPinnedFoldersResolveToTheirProject(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	aSrc := w.plain(t, "a/src")
	bSrc := w.plain(t, "b/src")
	plain := w.plain(t, "plain")

	t.Run("a reported folder below another project moves the session there", func(t *testing.T) {
		t.Parallel()
		handler := &fakeRoots{}
		handler.set(bSrc)
		var warnings bytes.Buffer
		c := newRootsClient(t, a, handler, &warnings)
		if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != b {
			t.Errorf("served %q, want %q; warnings: %q", got, b, warnings.String())
		}
		if warnings.Len() != 0 {
			t.Errorf("a folder below a project drew a warning: %q", warnings.String())
		}
	})
	t.Run("two reported folders of one project are one candidate", func(t *testing.T) {
		t.Parallel()
		handler := &fakeRoots{}
		handler.set(bSrc, b)
		var warnings bytes.Buffer
		c := newRootsClient(t, a, handler, &warnings)
		if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != b {
			t.Errorf("served %q, want %q; warnings: %q", got, b, warnings.String())
		}
		if strings.Contains(warnings.String(), "several usable") {
			t.Errorf("two folders of one project were read as two projects: %q", warnings.String())
		}
	})
	t.Run("a reported folder of the served project changes nothing", func(t *testing.T) {
		t.Parallel()
		handler := &fakeRoots{}
		handler.set(aSrc)
		var warnings bytes.Buffer
		c := newRootsClient(t, a, handler, &warnings)
		if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != a {
			t.Errorf("served %q, want %q", got, a)
		}
		if warnings.Len() != 0 {
			t.Errorf("a folder of the served project drew a warning: %q", warnings.String())
		}
	})
	t.Run("a session started below a root keeps its project when the host reports both", func(t *testing.T) {
		t.Parallel()
		handler := &fakeRoots{}
		handler.set(aSrc, a)
		c := newRootsClient(t, aSrc, handler, &bytes.Buffer{})
		if got := projectOf(t, mustCallTool(t, c, "create_document", adrArgs("both-reported", nil))); got != a {
			t.Errorf("served %q, want %q", got, a)
		}
	})
	t.Run("a reported folder without a project keeps the current one", func(t *testing.T) {
		t.Parallel()
		handler := &fakeRoots{}
		handler.set(plain)
		var warnings bytes.Buffer
		c := newRootsClient(t, a, handler, &warnings)
		if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != a {
			t.Errorf("served %q, want %q", got, a)
		}
		if !strings.Contains(warnings.String(), "no .archcore/") {
			t.Errorf("warnings = %q, want the refusal reason", warnings.String())
		}
	})
	t.Run("a pinned folder below a root serves the project", func(t *testing.T) {
		t.Parallel()
		c := newPinnedClient(t, bSrc, &fakeRoots{}, &bytes.Buffer{})
		if got := projectOf(t, mustCallTool(t, c, "list_documents", nil)); got != b {
			t.Errorf("served %q, want %q", got, b)
		}
	})
	// Cleanup runs after the parallel subtests above, so it judges them all.
	t.Cleanup(func() {
		if stores := storesUnder(t, w.dir); len(stores) != 2 {
			t.Errorf("stores = %v, want the two projects only", stores)
		}
	})
}

func TestSessionProject_AStoreWithoutSettingsDoesNotCaptureTheFolderBelow(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	stray := w.plain(t, "work/.archcore")
	writeFixtureFile(t, filepath.Join(stray, "old.adr.md"), "---\ntitle: Old\nstatus: draft\n---\n")
	repo := w.plain(t, "work/repo")
	c := newTestClient(t, repo)

	listed := mustCallTool(t, c, "list_documents", nil)
	if got := projectOf(t, listed); got != repo {
		t.Errorf("a session below a store without settings.json served %q, want its own folder %q", got, repo)
	}
	if !strings.HasSuffix(firstText(listed), `,"initialized":false}`) {
		t.Errorf("list_documents = %s, want the uninitialized answer", firstText(listed))
	}
	expectRefusal(t, c, w, "create_document", adrArgs("x", nil), "init_project")
	if stores := storesUnder(t, w.dir); !slices.Equal(stores, []string{"work/.archcore"}) {
		t.Errorf("stores = %v, want the earlier store only", stores)
	}
}

// TestSessionProject_EveryToolIsClassifiedForARootWithoutAStore reads the
// tools off a built server and calls each one on a root without a store. A
// tool the server gains must land in one of the three classes on purpose.
func TestSessionProject_EveryToolIsClassifiedForARootWithoutAStore(t *testing.T) {
	t.Parallel()
	refuses := map[string]bool{
		"create_document": true, "update_document": true, "remove_document": true,
		"add_relation": true, "remove_relation": true,
	}
	answers := map[string]bool{"list_documents": true, "search_documents": true, "get_document": true, "list_relations": true}
	creates := map[string]bool{"init_project": true, "install_host_config": true}

	w := newWorkspace(t)
	srv := mcpserver.NewServer(w.plain(t, "probe"), "test",
		mcpserver.WithHostWiring(func(string, string, bool) ([]byte, error) { return []byte(`{}`), nil }))
	registered := srv.ListTools()
	if len(registered) == 0 {
		t.Fatal("the server registered no tools, so this test proves nothing")
	}
	for name := range registered {
		classes := 0
		for _, class := range []map[string]bool{refuses, answers, creates} {
			if class[name] {
				classes++
			}
		}
		if classes != 1 {
			t.Errorf("%s is in %d classes, want exactly 1: decide what it does on a root without .archcore/", name, classes)
		}
	}
	for _, class := range []map[string]bool{refuses, answers, creates} {
		for name := range class {
			if _, ok := registered[name]; !ok {
				t.Errorf("%s is classified here and the server does not register it", name)
			}
		}
	}

	for name := range refuses {
		folder := w.plain(t, "refuses-"+name)
		c := newTestClient(t, folder)
		text := expectToolError(t, c, name, map[string]any{
			"type": "adr", "filename": "x", "path": ".archcore/x.adr.md",
			"source": ".archcore/x.adr.md", "target": ".archcore/y.adr.md",
		})
		if !strings.Contains(text, "holds no .archcore/ directory") {
			t.Errorf("%s on a root without a store = %q, want the no-store refusal", name, text)
		}
		if stores := storesUnder(t, folder); len(stores) != 0 {
			t.Errorf("%s created %v on a root without a store", name, stores)
		}
	}
}
