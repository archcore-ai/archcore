package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
)

// hookWorkspace is a folder of sibling projects: a and b hold .archcore/, a/app
// is a project nested in a, and plain holds none.
type hookWorkspace struct {
	dir, a, b, nested, plain string
}

func newHookWorkspace(t *testing.T) hookWorkspace {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := hookWorkspace{
		dir:    dir,
		a:      filepath.Join(dir, "a"),
		b:      filepath.Join(dir, "b"),
		nested: filepath.Join(dir, "a", "app"),
		plain:  filepath.Join(dir, "plain"),
	}
	for _, p := range []string{
		filepath.Join(w.a, ".archcore", "knowledge"),
		filepath.Join(w.b, ".archcore", "knowledge"),
		filepath.Join(w.nested, ".archcore"),
		filepath.Join(w.a, "src"),
		filepath.Join(w.b, "src", "api"),
		filepath.Join(w.plain, "src"),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func writePayload(path string, extra string) string {
	quoted, _ := json.Marshal(path)
	return `{"tool_name":"Write","tool_input":{"file_path":` + string(quoted) + `}` + extra + `}`
}

func sessionPayload(t *testing.T, path string) string {
	t.Helper()
	id, _ := json.Marshal(t.Name())
	return writePayload(path, `,"session_id":`+string(id))
}

func TestHookRoots_RootOf(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	link := filepath.Join(w.dir, "a-link")
	if err := os.Symlink(w.a, link); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, session, target, want string
	}{
		{"relative target", w.a, ".archcore/x.adr.md", w.a},
		{"relative target that escapes into a sibling project", w.a, "../b/.archcore/x.adr.md", w.b},
		{"relative target that escapes outside every project", w.a, "../plain/src/x.go", w.a},
		{"empty target", w.a, "", w.a},
		{"absolute target in the session project", w.a, filepath.Join(w.a, "src", "x.go"), w.a},
		{"absolute target in a sibling project", w.a, filepath.Join(w.b, ".archcore", "x.adr.md"), w.b},
		{"absolute target in a nested project", w.a, filepath.Join(w.nested, ".archcore", "x.md"), w.nested},
		{"absolute target outside every project", w.a, filepath.Join(w.plain, "src", "x.go"), w.a},
		{"new directories under a sibling project", w.a, filepath.Join(w.b, "new1", "new2", "new3", "x.go"), w.b},
		{"new directories under the session project", w.a, "new1/new2/x.go", w.a},
		{"session without .archcore, target in a project", w.plain, filepath.Join(w.b, "src", "x.go"), w.b},
		{"session spelled through a symlink", link, filepath.Join(w.a, "src", "x.go"), link},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			roots := newHookRoots(tt.session)
			for range 2 { // the second answer comes from the memo
				if got := roots.rootOf(tt.target); got != tt.want {
					t.Fatalf("rootOf(%q) = %q, want %q", tt.target, got, tt.want)
				}
			}
		})
	}
}

// TestPreToolUse_GuardsEveryProject holds hook-runtime.spec §14–§15 and the
// write-guard invariant of hooks-validation-system.spec: a direct write to any
// project's .archcore/ is denied, wherever the session started.
func TestPreToolUse_GuardsEveryProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	storeLink := filepath.Join(w.a, "src", "b-store")
	if err := os.Symlink(filepath.Join(w.b, ".archcore"), storeLink); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{filepath.Join(w.a, ".archcore", "own.adr.md"), filepath.Join(w.b, ".archcore", "knowledge", "linked.adr.md")} {
		if err := os.WriteFile(doc, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	siblingDocLink := filepath.Join(w.a, "src", "sibling-doc.md")
	ownDocLink := filepath.Join(w.a, "src", "own-doc.md")
	if err := os.Symlink(filepath.Join(w.b, ".archcore", "knowledge", "linked.adr.md"), siblingDocLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(w.a, ".archcore", "own.adr.md"), ownDocLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, session, target string
		wantDeny              bool
		wantHint              string // the root a deny names, "" for none
	}{
		{"relative document in the session project", w.a, ".archcore/knowledge/x.adr.md", true, ""},
		{"absolute document in the session project", w.a, filepath.Join(w.a, ".archcore", "knowledge", "x.adr.md"), true, ""},
		{"document in a sibling project", w.a, filepath.Join(w.b, ".archcore", "knowledge", "x.adr.md"), true, w.b},
		{"relative document in a sibling project", w.a, "../b/.archcore/knowledge/x.adr.md", true, w.b},
		{"document through a link into a sibling store", w.a, filepath.Join(storeLink, "knowledge", "x.adr.md"), true, w.b},
		{"file link to a sibling document", w.a, siblingDocLink, true, w.b},
		{"file link to a document of the session project", w.a, ownDocLink, true, ""},
		{"relative file link to a sibling document", w.a, "src/sibling-doc.md", true, w.b},
		{"document under new directories of a sibling store", w.a, filepath.Join(w.b, ".archcore", "n1", "n2", "n3", "x.adr.md"), true, w.b},
		{"document in a nested project", w.a, filepath.Join(w.nested, ".archcore", "x.spec.md"), true, w.nested},
		{"document of a project, session without .archcore", w.plain, filepath.Join(w.b, ".archcore", "x.adr.md"), true, w.b},
		{"document of the parent, session in the nested project", w.nested, filepath.Join(w.a, ".archcore", "x.adr.md"), true, w.a},
		{"settings.json of a sibling stays writable", w.a, filepath.Join(w.b, ".archcore", "settings.json"), false, ""},
		{"sync state of a sibling stays writable", w.a, filepath.Join(w.b, ".archcore", ".sync-state.json"), false, ""},
		{"source file of a sibling", w.a, filepath.Join(w.b, "src", "x.go"), false, ""},
		{"README of a sibling", w.a, filepath.Join(w.b, "README.md"), false, ""},
		{"file outside every project", w.a, filepath.Join(w.plain, "notes.md"), false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, host := range []string{"claude-code", "codex-cli", "cursor", "copilot", "opencode", "gemini-cli"} {
				dec := preToolUseHandler(bg(), reqFor(t, host, tt.session, writePayload(tt.target, "")))
				if dec.deny != tt.wantDeny {
					t.Fatalf("%s: deny = %v, want %v (reason %q)", host, dec.deny, tt.wantDeny, dec.reason)
				}
				if !tt.wantDeny {
					continue
				}
				if !strings.HasPrefix(dec.reason, writeGuardReason) {
					t.Errorf("%s: reason %q does not lead with the MCP-only reason", host, dec.reason)
				}
				hint := `project="` + tt.wantHint + `"`
				if tt.wantHint == "" && strings.Contains(dec.reason, "project=") {
					t.Errorf("%s: a session-project deny carries a project hint: %q", host, dec.reason)
				}
				if tt.wantHint != "" && !strings.Contains(dec.reason, hint) {
					t.Errorf("%s: reason %q does not name %s", host, dec.reason, hint)
				}
			}
		})
	}
}

// TestPreToolUse_SymlinkedProject: a target written through a symlink to
// another project is still denied, and the deny and the note name the evaluated
// root, the spelling the MCP server answers with.
func TestPreToolUse_SymlinkedProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	link := filepath.Join(w.dir, "b-link")
	if err := os.Symlink(w.b, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, host := range []string{"claude-code", "codex-cli", "cursor", "copilot", "opencode", "gemini-cli"} {
		dec := preToolUseHandler(bg(), reqFor(t, host, w.a, writePayload(filepath.Join(link, ".archcore", "knowledge", "x.adr.md"), "")))
		if !dec.deny || !strings.Contains(dec.reason, `project="`+w.b+`"`) {
			t.Errorf("%s: a document through a symlink = %+v; want a deny naming %s", host, dec, w.b)
		}
	}
	first := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(link, "src", "x.go"))))
	if !strings.Contains(first.context, "project at "+w.b+",") {
		t.Fatalf("note through a symlink = %q; want the evaluated root %s", first.context, w.b)
	}
	second := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.b, "src", "y.go"))))
	if strings.Contains(second.context, "[Archcore Project]") {
		t.Errorf("two spellings of one project got two notes: %q", second.context)
	}
}

// TestPreToolUse_SessionProjectThroughASymlink: a write into the session's own
// store spelled through a symlink of its root is denied with the plain reason.
// Compared as written the target sat outside the project and was allowed
// (host-probe-protocol.spec run of 2026-10-30).
func TestPreToolUse_SessionProjectThroughASymlink(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	link := filepath.Join(w.dir, "a-link")
	if err := os.Symlink(w.a, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, tt := range []struct{ session, target string }{
		{w.a, filepath.Join(link, ".archcore", "knowledge", "x.adr.md")},
		{link, filepath.Join(w.a, ".archcore", "knowledge", "x.adr.md")},
		{w.a, filepath.Join(link, ".archcore", "new-dir", "x.adr.md")},
	} {
		for _, host := range []string{"claude-code", "codex-cli", "cursor", "copilot", "opencode", "gemini-cli"} {
			dec := preToolUseHandler(bg(), reqFor(t, host, tt.session, writePayload(tt.target, "")))
			if !dec.deny || strings.Contains(dec.reason, "project=") {
				t.Errorf("%s: session %q, target %q = %+v; want the plain deny", host, tt.session, tt.target, dec)
			}
		}
	}
}

// TestPreToolUse_ProjectTouchAdvisoryFromAPatch: Codex edits every file with
// apply_patch, so a patch target in another project carries the note once per
// project, and a patch inside the session's project carries none
// (host-probe-protocol.spec run of 2026-10-30).
func TestPreToolUse_ProjectTouchAdvisoryFromAPatch(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	patch := func(paths ...string) string {
		text := "*** Begin Patch\n"
		for _, p := range paths {
			text += "*** Add File: " + p + "\n+x\n"
		}
		quoted, _ := json.Marshal(text + "*** End Patch")
		id, _ := json.Marshal(t.Name())
		return `{"tool_name":"apply_patch","session_id":` + string(id) + `,"tool_input":{"command":` + string(quoted) + `}}`
	}
	first := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, patch(filepath.Join(w.b, "src", "x.go"), filepath.Join(w.b, "src", "y.go"), filepath.Join(w.nested, "z.go"))))
	if first.deny || strings.Count(first.context, "[Archcore Project]") != 2 ||
		!strings.Contains(first.context, "project at "+w.b+",") || !strings.Contains(first.context, "project at "+w.nested+",") {
		t.Fatalf("patch into two other projects = %+v; want one note for each", first)
	}
	again := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, patch(filepath.Join(w.b, "src", "w.go"))))
	if strings.Contains(again.context, "[Archcore Project]") {
		t.Errorf("a second patch into the same project repeated the note: %q", again.context)
	}
	own := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, patch("src/x.go", filepath.Join(w.a, "src", "y.go"))))
	if strings.Contains(own.context, "[Archcore Project]") {
		t.Errorf("a patch inside the session's project got a note: %q", own.context)
	}
}

// TestPreToolUse_SessionProjectDenyIsUnchanged pins backward compatibility:
// the deny for the session's own project is byte-identical to the one before
// call roots existed.
func TestPreToolUse_SessionProjectDenyIsUnchanged(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	dec := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, writePayload(".archcore/knowledge/x.adr.md", "")))
	if !dec.deny || dec.reason != writeGuardReason {
		t.Errorf("decision = %+v, want the unchanged MCP-only deny", dec)
	}
}

// TestPreToolUse_PatchIntoAnotherProject: an apply_patch call that names an
// absolute target in another project's store is denied as a whole.
func TestPreToolUse_PatchIntoAnotherProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	target := filepath.Join(w.b, ".archcore", "x.rule.md")
	patch := "*** Begin Patch\n*** Update File: src/ok.go\n*** Update File: " + target + "\n*** End Patch"
	quoted, _ := json.Marshal(patch)
	payload := `{"tool_name":"apply_patch","tool_input":{"input":` + string(quoted) + `}}`

	dec := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, payload))
	if !dec.deny || !strings.Contains(dec.reason, `project="`+w.b+`"`) {
		t.Errorf("decision = %+v, want a deny naming project B", dec)
	}

	ok := "*** Begin Patch\n*** Update File: src/ok.go\n*** Update File: " + filepath.Join(w.b, "src", "x.go") + "\n*** End Patch"
	quoted, _ = json.Marshal(ok)
	if dec := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, `{"tool_name":"apply_patch","tool_input":{"input":`+string(quoted)+`}}`)); dec.deny {
		t.Errorf("a patch of source files was denied: %q", dec.reason)
	}
}

// TestPreToolUse_ProjectTouchAdvisory holds hook-runtime.spec §16: one note per
// session and foreign project, none for the session's own project.
func TestPreToolUse_ProjectTouchAdvisory(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	inB := filepath.Join(w.b, "src", "x.go")
	note := `[Archcore Project] This file belongs to the Archcore project at ` + w.b

	first := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, inB)))
	if first.deny || !strings.Contains(first.context, note) || !strings.Contains(first.context, `project="`+w.b+`"`) {
		t.Fatalf("first touch = %+v, want the project note", first)
	}
	second := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.b, "src", "y.go"))))
	if strings.Contains(second.context, "[Archcore Project]") {
		t.Errorf("second touch of the same project repeated the note: %q", second.context)
	}
	nested := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.nested, "main.go"))))
	if !strings.Contains(nested.context, "project at "+w.nested) {
		t.Errorf("a second foreign project got no note: %q", nested.context)
	}
	own := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.a, "src", "x.go"))))
	if strings.Contains(own.context, "[Archcore Project]") {
		t.Errorf("the session's own project got a note: %q", own.context)
	}
	relative := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, "src/x.go")))
	if strings.Contains(relative.context, "[Archcore Project]") {
		t.Errorf("a relative path got a note: %q", relative.context)
	}
	outside := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.plain, "x.go"))))
	if strings.Contains(outside.context, "[Archcore Project]") {
		t.Errorf("a file outside every project got a note: %q", outside.context)
	}
}

// TestPreToolUse_ProjectTouchAdvisoryPerHost: the stamp is keyed by session,
// root, and host (hook-runtime.spec §16), so a second host gets its own note.
func TestPreToolUse_ProjectTouchAdvisoryPerHost(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	inB := filepath.Join(w.b, "src", "x.go")
	for _, host := range []string{"claude-code", "codex-cli"} {
		if dec := preToolUseHandler(bg(), reqFor(t, host, w.a, sessionPayload(t, inB))); !strings.Contains(dec.context, "[Archcore Project]") {
			t.Errorf("%s: the first touch in this session got no project note: %q", host, dec.context)
		}
	}
}

func TestPreToolUse_ProjectTouchAdvisoryWithoutSessionID(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	for range 2 {
		dec := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, writePayload(filepath.Join(w.b, "src", "x.go"), "")))
		if !strings.Contains(dec.context, "[Archcore Project]") {
			t.Fatalf("without a session id the note must fail open, got %q", dec.context)
		}
	}
}

func TestPreToolUse_ProjectTouchAdvisorySkippedOnCopilot(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	dec := preToolUseHandler(bg(), reqFor(t, "copilot", w.a, sessionPayload(t, filepath.Join(w.b, "src", "x.go"))))
	if dec.deny || dec.context != "" {
		t.Errorf("copilot decision = %+v, want a silent allow", dec)
	}
}

// TestPreToolUse_CodeAlignmentFromTheFilesProject: the documents injected
// before an edit come from the project that holds the file.
func TestPreToolUse_CodeAlignmentFromTheFilesProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	writeAlignmentHandlerDoc(t, w.b, "knowledge/api.rule.md", "B API Rule", "Applies to src/api/ code.")
	writeAlignmentHandlerDoc(t, w.a, "knowledge/api.rule.md", "A API Rule", "Applies to src/api/ code.")
	if err := os.MkdirAll(filepath.Join(w.a, "src", "api"), 0o755); err != nil {
		t.Fatal(err)
	}

	dec := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, sessionPayload(t, filepath.Join(w.b, "src", "api", "h.go"))))
	if !strings.Contains(dec.context, "B API Rule") || strings.Contains(dec.context, "A API Rule") {
		t.Errorf("context = %q, want B's rule and not A's", dec.context)
	}
	own := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, writePayload("src/api/h.go", "")))
	if !strings.Contains(own.context, "A API Rule") {
		t.Errorf("a relative edit lost the session project's context: %q", own.context)
	}
}

// TestPostToolUse_ValidatesInTheNamedProject holds hook-runtime.spec §17.
func TestPostToolUse_ValidatesInTheNamedProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	writeArchcoreDoc(t, w.b, "knowledge/bad.adr.md", "no frontmatter at all\n")

	payload := func(project string) *hookPayload {
		quoted, _ := json.Marshal(project)
		return decodeHookPayload(strings.NewReader(
			`{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/knowledge/bad.adr.md","project":` + string(quoted) + `}}`))
	}
	req := func(project string) hookRequest {
		return hookRequest{baseDir: w.a, dialect: hookDialects[0], event: eventPostToolUse, payload: payload(project)}
	}

	if dec := postToolUseHandler(bg(), req(w.b)); !strings.Contains(dec.context, "[Archcore Validation]") {
		t.Errorf("project B: context = %q, want B's validation finding", dec.context)
	}
	// A relative value that does reach B from the process's working directory:
	// a relative project names no root (call-root-resolution.spec, the project
	// form), so it must not move validation even when the directory exists.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The kernel resolves ".." against the physical directory, so a cwd reached
	// through a symlink would give a relative path that misses B.
	if cwd, err = filepath.EvalSymlinks(cwd); err != nil {
		t.Fatal(err)
	}
	relativeB, err := filepath.Rel(cwd, w.b)
	if err != nil {
		t.Fatal(err)
	}
	if !config.DirExists(relativeB) {
		t.Fatalf("relative path %q does not reach project B; the case would prove nothing", relativeB)
	}
	for _, project := range []string{"", "b", relativeB, w.plain, filepath.Join(w.dir, "missing")} {
		if dec := postToolUseHandler(bg(), req(project)); strings.Contains(dec.context, "[Archcore Validation]") {
			t.Errorf("project %q: validated B instead of the working directory: %q", project, dec.context)
		}
	}
}

// TestPostToolUse_ValidatesInTheServedProject holds hook-runtime.spec §17 and
// Failure Behavior 7: each root in the order is taken only when the roots
// before it fail, and a write no project holds is allowed silently. Every
// project carries its own broken document, so a finding names the root used.
func TestPostToolUse_ValidatesInTheServedProject(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	roots := map[string]string{"a": w.a, "b": w.b, "nested": w.nested}
	for name, root := range roots {
		writeArchcoreDoc(t, root, "knowledge/bad-"+name+".adr.md", "no frontmatter at all\n")
	}

	quote := func(s string) string {
		quoted, _ := json.Marshal(s)
		return string(quoted)
	}
	resultText := func(project string) string {
		return quote(`{"path":".archcore/knowledge/bad-b.adr.md","project":` + quote(project) + `}`)
	}
	claude := func(project string) string {
		return `,"tool_response":[{"type":"text","text":` + resultText(project) + `}]`
	}
	codex := func(project string) string {
		return `,"tool_response":{"content":[{"type":"text","text":` + resultText(project) + `}]}`
	}
	copilot := func(project string) string {
		return `,"toolResult":{"resultType":"success","textResultForLlm":` + resultText(project) + `}`
	}
	cursor := func(project string) string {
		return `,"result_json":` + resultText(project)
	}

	tests := []struct {
		name, cwd, project, response string
		want                         string // the project whose finding is reported; "" means a silent allow
	}{
		{"served root wins over the working directory", w.nested, "", claude(w.b), "b"},
		{"served root wins over the project argument", w.a, w.a, claude(w.b), "b"},
		{"served root without a store falls to the project argument", filepath.Join(w.plain, "src"), w.b, claude(w.plain), "b"},
		{"relative served root falls to the project argument", filepath.Join(w.plain, "src"), w.b, claude("b"), "b"},
		{"served root without a store falls to the working directory", filepath.Join(w.b, "src"), "", claude(w.plain), "b"},
		{"working directory below the root", filepath.Join(w.b, "src", "api"), "", "", "b"},
		{"working directory inside a nested project", w.nested, "", "", "nested"},
		{"no project above the working directory", filepath.Join(w.plain, "src"), "", "", ""},
		{"codex result", w.a, "", codex(w.b), "b"},
		{"copilot result", w.a, "", copilot(w.b), "b"},
		{"cursor result", w.a, "", cursor(w.b), "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := `{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/knowledge/bad-b.adr.md","project":` +
				quote(tt.project) + `}` + tt.response + `}`
			req := hookRequest{baseDir: tt.cwd, dialect: hookDialects[0], event: eventPostToolUse, payload: decodeHookPayload(strings.NewReader(raw))}
			dec := postToolUseHandler(bg(), req)
			if tt.want == "" && (dec.context != "" || dec.deny) {
				t.Fatalf("decision = %+v, want a silent allow", dec)
			}
			for name := range roots {
				if got := strings.Contains(dec.context, "bad-"+name+".adr.md"); got != (name == tt.want) {
					t.Errorf("reported project %s = %v, want %v; context = %q", name, got, name == tt.want, dec.context)
				}
			}
		})
	}
}

func TestServedProject(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, payload, want string }{
		// The payload Claude Code sent on 2026-11-05, kept as
		// test/fixtures/stdin/claude-code/posttooluse-mcp-list-documents.json in
		// the plugin, so a change of shape is caught on both sides.
		{"claude-code captured", `{"hook_event_name":"PostToolUse","tool_name":"mcp__plugin_archcore_archcore__list_documents","tool_input":{},` +
			`"tool_response":[{"type":"text","text":"{\"by_source\":{},\"total\":0,\"offset\":0,\"returned\":0,\"truncated\":false,\"documents\":[],\"project\":\"/work/sub\",\"source\":\"local\"}"}],` +
			`"mcp_server":{"name":"plugin:archcore:archcore","source":"plugin"}}`, "/work/sub"},
		{"codex result object", `{"tool_response":{"content":[{"type":"text","text":"{\"project\":\"/w/b\"}"}]}}`, "/w/b"},
		{"copilot text result", `{"toolResult":{"resultType":"success","textResultForLlm":"{\"project\":\"/w/b\"}"}}`, "/w/b"},
		{"cursor result payload", `{"result_json":"{\"project\":\"/w/b\"}"}`, "/w/b"},
		{"cursor result object", `{"result_json":"{\"content\":[{\"type\":\"text\",\"text\":\"{\\\"project\\\":\\\"/w/b\\\"}\"}]}"}`, "/w/b"},
		{"last project key wins", `{"tool_response":[{"type":"text","text":"{\"project\":\"/w/a\",\"project\":\"/w/b\"}"}]}`, "/w/b"},
		{"two content blocks", `{"tool_response":[{"type":"text","text":"{\"project\":\"/w/b\"}"},{"type":"text","text":"{}"}]}`, ""},
		{"text not JSON", `{"tool_response":[{"type":"text","text":"OK"}]}`, ""},
		{"error result", `{"tool_response":{"content":[{"type":"text","text":"project holds no .archcore/ directory"}],"isError":true}}`, ""},
		{"cursor plain output", `{"tool_output":"OK"}`, ""},
		{"no response", `{"tool_input":{"project":"/w/b"}}`, ""},
	} {
		if got := decodeHookPayload(strings.NewReader(tt.payload)).servedProject(); got != tt.want {
			t.Errorf("%s: servedProject = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestToolProject(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ payload, want string }{
		{`{"tool_input":{"project":"/w/b"}}`, "/w/b"},
		{`{"toolArgs":{"project":"/w/b"}}`, "/w/b"},
		{`{"tool_args":{"project":"/w/b"}}`, "/w/b"},
		{`{"tool_input":"{\"project\":\"/w/b\"}"}`, "/w/b"},
		{`{"tool_input":{"path":".archcore/x.md"}}`, ""},
		{`{}`, ""},
	} {
		if got := decodeHookPayload(strings.NewReader(tt.payload)).toolProject(); got != tt.want {
			t.Errorf("toolProject(%s) = %q, want %q", tt.payload, got, tt.want)
		}
	}
}

// TestSessionStart_OutsideAProject holds hooks-validation-system.spec, Hook 1
// phase 2: no advice to create a store, the reachable projects listed, and the
// project argument named.
func TestSessionStart_OutsideAProject(t *testing.T) {
	// Not parallel: sessionContextOf captures the process stdout.
	w := newHookWorkspace(t)

	tests := []struct {
		name, dir string
		want      []string
		dontWant  []string
	}{
		{
			name:     "workspace parent",
			dir:      w.dir,
			want:     []string{"NO PROJECT HERE", "PROJECTS BELOW: " + w.a + ", " + w.b, "project=<its absolute root>"},
			dontWant: []string{w.nested, "PROJECT ABOVE", "No Archcore project was found"},
		},
		{
			name:     "a folder inside a project",
			dir:      filepath.Join(w.a, "src"),
			want:     []string{"NO PROJECT HERE", "PROJECT ABOVE: " + w.a},
			dontWant: []string{"PROJECTS BELOW", "No Archcore project was found"},
		},
		{
			name: "a folder far from every project",
			dir:  w.plain,
			want: []string{"NO PROJECT HERE", "No Archcore project was found near this folder."},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionContextOf(t, "claude-code", tt.dir, "test")
			for _, s := range tt.want {
				if !strings.Contains(got, s) {
					t.Errorf("recap lacks %q:\n%s", s, got)
				}
			}
			for _, s := range append(tt.dontWant, "use create_document to record", "init_project") {
				if strings.Contains(got, s) {
					t.Errorf("recap carries %q:\n%s", s, got)
				}
			}
		})
	}
}

// TestSessionStart_InsideAProjectIsUnchanged pins backward compatibility: a
// project with .archcore/ gets the corpus line, never the inventory.
func TestSessionStart_InsideAProjectIsUnchanged(t *testing.T) {
	w := newHookWorkspace(t)
	got := sessionContextOf(t, "claude-code", w.a, "test")
	if !strings.Contains(got, "CORPUS: no documents yet — use create_document to record the first one.") {
		t.Errorf("an empty project lost its corpus line:\n%s", got)
	}
	if strings.Contains(got, "NO PROJECT HERE") || strings.Contains(got, "PROJECTS BELOW") {
		t.Errorf("an initialized project got the inventory:\n%s", got)
	}
}

func TestWriteProjectInventory_CapsTheList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for i := range inventoryShown + 3 {
		if err := os.MkdirAll(filepath.Join(dir, "p"+string(rune('a'+i)), ".archcore"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var b strings.Builder
	writeProjectInventory(&b, dir)
	got := b.String()
	line := got[strings.Index(got, "PROJECTS BELOW"):]
	line = line[:strings.Index(line, "\n")]
	if !strings.HasSuffix(line, " … and 3 more") || strings.Count(line, ", ") != inventoryShown-1 {
		t.Errorf("inventory did not cap at %d and state the rest:\n%s", inventoryShown, got)
	}
}

func TestWriteProjectInventory_DepthBound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	third := filepath.Join(dir, "x", "y", "third")
	fourth := filepath.Join(dir, "x", "y", "z", "fourth")
	for _, p := range []string{third, fourth} {
		if err := os.MkdirAll(filepath.Join(p, ".archcore"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var b strings.Builder
	writeProjectInventory(&b, dir)
	got := b.String()
	if !strings.Contains(got, "PROJECTS BELOW: "+third+" (the search stopped at its bound)\n") {
		t.Errorf("want the third-level project listed and the bound stated:\n%s", got)
	}
	if strings.Contains(got, fourth) {
		t.Errorf("a fourth-level project was listed:\n%s", got)
	}
}

func TestWriteProjectInventory_DirectoryBound(t *testing.T) {
	t.Parallel()
	// The walk reads the session folder, then one directory per empty child, so
	// 1999 children fit the 2000-directory bound and 2000 do not.
	for _, tt := range []struct {
		name     string
		children int
		wantNote bool
	}{
		{name: "inside the bound", children: 1999},
		{name: "past the bound", children: 2000, wantNote: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			project := filepath.Join(dir, "p")
			if err := os.MkdirAll(filepath.Join(project, ".archcore"), 0o755); err != nil {
				t.Fatal(err)
			}
			for i := range tt.children {
				if err := os.Mkdir(filepath.Join(dir, fmt.Sprintf("d%04d", i)), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			var b strings.Builder
			writeProjectInventory(&b, dir)
			got := b.String()
			if note := strings.Contains(got, "PROJECTS BELOW: "+project+" (the search stopped at its bound)\n"); note != tt.wantNote {
				t.Errorf("bound note = %v, want %v:\n%s", note, tt.wantNote, got)
			}
			if !strings.Contains(got, "PROJECTS BELOW: "+project) {
				t.Errorf("the project is not listed:\n%s", got)
			}
		})
	}
}

// TestPreToolUse_CodeAlignmentFromThePatchsFirstTarget: a Codex edit names its
// files only inside the patch, so the advisory aligns on the first of them
// (hook-runtime.spec §18) and on no later one.
func TestPreToolUse_CodeAlignmentFromThePatchsFirstTarget(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	writeAlignmentHandlerDoc(t, w.a, "knowledge/api.rule.md", "A API Rule", "Applies to src/api/ code.")
	writeAlignmentHandlerDoc(t, w.b, "knowledge/api.rule.md", "B API Rule", "Applies to src/api/ code.")
	if err := os.MkdirAll(filepath.Join(w.a, "src", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	patch := func(files ...string) string {
		body := "*** Begin Patch\n"
		for _, f := range files {
			body += "*** Update File: " + f + "\n@@\n-a\n+b\n"
		}
		quoted, _ := json.Marshal(body + "*** End Patch\n")
		return `{"tool_name":"apply_patch","tool_input":{"command":` + string(quoted) + `}}`
	}

	for _, tt := range []struct {
		name, payload, want, dontWant string
	}{
		{name: "relative first target", payload: patch("src/api/h.go", "README.md"), want: "A API Rule"},
		{name: "first target in another project", payload: patch(filepath.Join(w.b, "src", "api", "h.go")), want: "B API Rule", dontWant: "A API Rule"},
		{name: "a later target is not aligned", payload: patch("README.md", "src/api/h.go"), dontWant: "A API Rule"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dec := preToolUseHandler(bg(), reqFor(t, "codex-cli", w.a, tt.payload))
			if dec.deny {
				t.Fatalf("a source patch was denied: %+v", dec)
			}
			if tt.want != "" && !strings.Contains(dec.context, tt.want) {
				t.Errorf("context = %q, want %q", dec.context, tt.want)
			}
			if tt.dontWant != "" && strings.Contains(dec.context, tt.dontWant) {
				t.Errorf("context = %q, want no %q", dec.context, tt.dontWant)
			}
		})
	}
}

// TestPreToolUse_DenyNamesTheSource: a direct write to a document of a project
// the session declares as a source is denied, and the deny names that project
// and the source id (source-mounts.spec, behavior 13).
func TestPreToolUse_DenyNamesTheSource(t *testing.T) {
	t.Parallel()
	w := newHookWorkspace(t)
	settings := `{"sync":"none","globals":[{"id":"shared","path":"../b/.archcore"}]}`
	if err := os.WriteFile(filepath.Join(w.a, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{filepath.Join(w.b, ".archcore", "knowledge", "x.adr.md"), "../b/.archcore/knowledge/x.adr.md"} {
		dec := preToolUseHandler(bg(), reqFor(t, "claude-code", w.a, writePayload(target, "")))
		if !dec.deny {
			t.Fatalf("%s: a write into a source store was allowed", target)
		}
		for _, want := range []string{`project="` + w.b + `"`, `source "shared"`} {
			if !strings.Contains(dec.reason, want) {
				t.Errorf("%s: reason %q does not name %s", target, dec.reason, want)
			}
		}
	}
}
