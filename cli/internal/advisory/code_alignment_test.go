package advisory

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Case names mirror test/unit/check-code-alignment.bats in the plugin.

func writeAlignmentDoc(t *testing.T, base, relPath, title, body string) {
	t.Helper()
	full := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\ntitle: %q\nstatus: accepted\n---\n\n%s\n", title, body)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodeAlignment_SilentCases(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		setup    func(t *testing.T, base string)
		env      map[string]string
	}{
		{name: "no file_path", filePath: ""},
		{name: "non-source-root path", filePath: "docs/readme.md"},
		{name: ".archcore/*.md path", filePath: ".archcore/knowledge/a.adr.md"},
		{name: "a file named like a root but not inside one", filePath: "src"},
		{
			name:     "escape hatch disables injection",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "knowledge/api.rule.md", "API Rule", "Applies to src/api/ handlers.")
			},
			env: map[string]string{"ARCHCORE_DISABLE_INJECTION": "1"},
		},
		{
			name:     "source edit with no matching docs",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "knowledge/unrelated.adr.md", "Unrelated", "Nothing about that tree.")
			},
		},
		{
			name:     "types outside the allowlist are ignored",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "vision/roadmap.plan.md", "Roadmap", "Touches src/api/ eventually.")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := setupArchcoreDir(t)
			if tt.setup != nil {
				tt.setup(t, base)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := CodeAlignment(base, tt.filePath); got != "" {
				t.Errorf("CodeAlignment = %q, want empty", got)
			}
		})
	}
}

func TestCodeAlignment_InjectsMatchingDocs(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/api.rule.md", "API Handler Rule", "Everything under src/api/ returns an envelope.")

	got := CodeAlignment(base, "src/api/handlers.go")

	for _, want := range []string{"[Archcore Context] Read these before editing src/api/handlers.go:", "rule: API Handler Rule", "[.archcore/knowledge/api.rule.md]"} {
		if !strings.Contains(got, want) {
			t.Errorf("advisory missing %q:\n%s", want, got)
		}
	}
}

// TestCodeAlignment_LongerPrefixWins: a document about the exact package must
// outrank one about the whole tree, or the specific advice is never seen.
func TestCodeAlignment_LongerPrefixWins(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/broad.rule.md", "Broad", "Applies across src/.")
	writeAlignmentDoc(t, base, "knowledge/narrow.rule.md", "Narrow", "Applies to src/api/handlers/ only.")

	got := CodeAlignment(base, "src/api/handlers/users.go")

	narrow := strings.Index(got, "Narrow")
	broad := strings.Index(got, "Broad")
	if narrow < 0 || broad < 0 {
		t.Fatalf("expected both documents:\n%s", got)
	}
	if narrow > broad {
		t.Errorf("the more specific document ranked below the broader one:\n%s", got)
	}
}

// TestCodeAlignment_TypePriority: at equal specificity a rule constrains an
// edit more than a guide does.
func TestCodeAlignment_TypePriority(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/z.guide.md", "Some Guide", "Working in src/api/.")
	writeAlignmentDoc(t, base, "knowledge/a.rule.md", "Some Rule", "Working in src/api/.")

	got := CodeAlignment(base, "src/api/handlers.go")

	// Presence first: strings.Index returns -1 for an absent document, and
	// "-1 > 0" is false — so comparing indices alone passes just as happily when
	// rules stopped being injected at all.
	rule := strings.Index(got, "Some Rule")
	guide := strings.Index(got, "Some Guide")
	if rule < 0 || guide < 0 {
		t.Fatalf("expected both documents:\n%s", got)
	}
	if rule > guide {
		t.Errorf("guide outranked rule at equal specificity:\n%s", got)
	}
}

// TestCodeAlignment_SourceRootsAreNormalized: config validation accepts "./src"
// and "src/", but document paths are slash-separated and unprefixed. Those roots
// used to validate cleanly and then match nothing, so the advisory went silent
// for a settings.json that looked correct and reported no error. Normalization
// at load makes the accepted set and the matching set the same set.
//
// The Windows separator is normalized by the same filepath.ToSlash and is not
// tabled here: on Unix a backslash is an ordinary filename character, so
// converting it would break a directory legitimately named that way.
func TestCodeAlignment_SourceRootsAreNormalized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		root string // as written in settings.json
		file string // an edit that must produce an injection
	}{
		{name: "bare", root: "backend", file: "backend/api/h.go"},
		{name: "dot-slash prefixed", root: "./backend", file: "backend/api/h.go"},
		{name: "trailing slash", root: "backend/", file: "backend/api/h.go"},
		{name: "dot-slash and trailing slash", root: "./backend/", file: "backend/api/h.go"},
		{name: "nested root", root: "./backend/api/", file: "backend/api/h.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			settings := `{"sync":"none","codeAlignment":{"sourceRoots":["` + tt.root + `"]}}`
			if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
				t.Fatal(err)
			}
			writeAlignmentDoc(t, base, "knowledge/be.rule.md", "Backend Rule", "Applies to backend/api/ code.")

			if got := CodeAlignment(base, tt.file); !strings.Contains(got, "Backend Rule") {
				t.Errorf("root %q produced no injection for %s:\n%s", tt.root, tt.file, got)
			}
		})
	}
}

// TestCodeAlignment_MatchedCapStatesRemainder pins file-context-resolution.spec
// behavior 10: the cut rows are counted and the call that returns them is named.
func TestCodeAlignment_MatchedCapStatesRemainder(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	for i := range 8 {
		writeAlignmentDoc(t, base, fmt.Sprintf("knowledge/r-%d.rule.md", i), fmt.Sprintf("Rule %d", i), "Applies to src/api/ code.")
	}

	got := CodeAlignment(base, "src/api/handlers.go")

	if n := strings.Count(got, "\n- "); n != MaxContextMatched {
		t.Errorf("injected %d documents, want %d:\n%s", n, MaxContextMatched, got)
	}
	if !strings.Contains(got, `… and 3 more — search_documents(path_ref="src/api/handlers.go")`) {
		t.Errorf("remainder line missing:\n%s", got)
	}
}

// TestCodeAlignment_SourceRootsOverride pins that settings.json replaces the
// defaults rather than extending them.
func TestCodeAlignment_SourceRootsOverride(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	settings := `{"sync":"none","codeAlignment":{"sourceRoots":["backend"]}}`
	if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAlignmentDoc(t, base, "knowledge/be.rule.md", "Backend Rule", "Applies to backend/api/ code.")
	writeAlignmentDoc(t, base, "knowledge/fe.rule.md", "Src Rule", "Applies to src/api/ code.")
	// Both rules name a real directory, so neither counts as general.
	mkdirs(t, base, "backend/api", "src/api")

	if got := CodeAlignment(base, "backend/api/h.go"); !strings.Contains(got, "Backend Rule") {
		t.Errorf("declared root produced no injection:\n%s", got)
	}
	if got := CodeAlignment(base, "src/api/h.go"); got != "" {
		t.Errorf("a default root survived the override:\n%s", got)
	}
}

// TestCodeAlignment_RejectedExcluded: a refused decision is not a constraint.
func TestCodeAlignment_RejectedExcluded(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	full := filepath.Join(base, ".archcore", "knowledge", "old.rule.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: \"Refused Rule\"\nstatus: rejected\n---\n\nApplies to src/api/ code.\n"
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := CodeAlignment(base, "src/api/h.go"); got != "" {
		t.Errorf("a rejected document was injected:\n%s", got)
	}
}

func TestCodeAlignment_ExcludesResearchVocabulary(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"research", "evidence"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			writeAlignmentDoc(t, base, "world."+typ+".md", "World", "Applies to src/api/ handlers.")
			writeAlignmentDoc(t, base, "local.rule.md", "Local API Rule", "Applies to src/api/ handlers.")
			got := CodeAlignment(base, "src/api/handlers.go")
			if !strings.Contains(got, "local.rule.md") || strings.Contains(got, "world."+typ+".md") {
				t.Errorf("CodeAlignment = %q; want local rule only", got)
			}
		})
	}
}

// TestCodeAlignment_ScenarioRankedBetweenSpecAndGuide pins
// scenario-and-journey-advisory-canon.spec §15: a scenario reaches the edit
// through its Anchors line, below the spec it illustrates and above a guide.
func TestCodeAlignment_ScenarioRankedBetweenSpecAndGuide(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/a.guide.md", "Some Guide", "Working in src/api/.")
	writeAlignmentDoc(t, base, "knowledge/b.scenario.md", "Some Scenario", "Anchors: src/api/")
	writeAlignmentDoc(t, base, "knowledge/c.spec.md", "Some Spec", "Working in src/api/.")

	got := CodeAlignment(base, "src/api/handlers.go")

	spec := strings.Index(got, "Some Spec")
	scenario := strings.Index(got, "Some Scenario")
	guide := strings.Index(got, "Some Guide")
	if spec < 0 || scenario < 0 || guide < 0 {
		t.Fatalf("expected all three documents:\n%s", got)
	}
	if !(spec < scenario && scenario < guide) {
		t.Errorf("order is not spec, scenario, guide:\n%s", got)
	}
}

func TestCodeAlignment_ExcludesJourney(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "path.journey.md", "Path", "Applies to src/api/ handlers.")
	writeAlignmentDoc(t, base, "local.rule.md", "Local API Rule", "Applies to src/api/ handlers.")
	got := CodeAlignment(base, "src/api/handlers.go")
	if !strings.Contains(got, "local.rule.md") || strings.Contains(got, "path.journey.md") {
		t.Errorf("CodeAlignment = %q; want local rule only", got)
	}
}

func mkdirs(t *testing.T, base string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(base, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeStatusDoc(t *testing.T, base, relPath, title, status, body string) {
	t.Helper()
	writeArchcoreDoc(t, base, relPath, fmt.Sprintf("---\ntitle: %q\nstatus: %s\n---\n\n%s\n", title, status, body))
}

// TestResolveFileContext_Ranking pins file-context-resolution.spec behaviors
// 1–7: reason, depth, status, type, mentions, then path.
func TestResolveFileContext_Ranking(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		file   string
		setup  func(t *testing.T, base string)
		want   []string // matched paths, in order
		reason ContextReason
	}{
		{
			name: "a doc naming the file outranks a rule naming its directory",
			file: "src/app/_components/FullLayout.tsx",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "a.rule.md", "Dir Rule", "Applies to src/app/_components/ code.")
				writeAlignmentDoc(t, base, "z.doc.md", "Seam", "FullLayout.tsx renders the shell.")
			},
			want:   []string{".archcore/z.doc.md", ".archcore/a.rule.md"},
			reason: ReasonFile,
		},
		{
			name: "a full path outranks a bare file name",
			file: "src/app/FullLayout.tsx",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "a.rule.md", "Name", "FullLayout.tsx only.")
				writeAlignmentDoc(t, base, "b.doc.md", "Path", "See src/app/FullLayout.tsx.")
			},
			want:   []string{".archcore/b.doc.md", ".archcore/a.rule.md"},
			reason: ReasonFile,
		},
		{
			name: "accepted outranks draft at equal match",
			file: "src/api/h.go",
			setup: func(t *testing.T, base string) {
				writeStatusDoc(t, base, "a.spec.md", "Draft", "draft", "Covers src/api/.")
				writeStatusDoc(t, base, "b.spec.md", "Accepted", "accepted", "Covers src/api/.")
			},
			want:   []string{".archcore/b.spec.md", ".archcore/a.spec.md"},
			reason: ReasonDirectory,
		},
		{
			name: "more mentions outrank fewer at equal type",
			file: "src/api/h.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "a.adr.md", "Once", "Covers src/api/.")
				writeAlignmentDoc(t, base, "b.adr.md", "Twice", "Covers src/api/ and src/api/ again.")
			},
			want:   []string{".archcore/b.adr.md", ".archcore/a.adr.md"},
			reason: ReasonDirectory,
		},
		{
			name: "a doc ranks below a guide",
			file: "src/api/h.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "a.doc.md", "Doc", "Covers src/api/.")
				writeAlignmentDoc(t, base, "b.guide.md", "Guide", "Covers src/api/.")
			},
			want:   []string{".archcore/b.guide.md", ".archcore/a.doc.md"},
			reason: ReasonDirectory,
		},
		{
			name: "a generic file name does not count as naming the file",
			file: "src/app/page.tsx",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "a.doc.md", "Other Page", "Another page.tsx elsewhere.")
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			tt.setup(t, base)

			fc, err := ResolveFileContext(base, tt.file)
			if err != nil {
				t.Fatalf("ResolveFileContext: %v", err)
			}
			var got []string
			for _, row := range fc.Matched {
				got = append(got, row.Doc.Path)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("matched = %v, want %v", got, tt.want)
			}
			if len(fc.Matched) > 0 && fc.Matched[0].Reason != tt.reason {
				t.Errorf("first reason = %q, want %q", fc.Matched[0].Reason, tt.reason)
			}
		})
	}
}

// TestResolveFileContext_GeneralRules pins pathless-rules-are-general.adr and
// behavior 7: only an accepted rule or cpat naming no path of this project.
func TestResolveFileContext_GeneralRules(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	mkdirs(t, base, "lib/shared")
	writeAlignmentDoc(t, base, "guard.rule.md", "Guard Clauses", "Return early.")
	writeAlignmentDoc(t, base, "tests.rule.md", "Test Titles", "Use @testing-library/react in component tests.")
	writeAlignmentDoc(t, base, "scoped.rule.md", "Scoped", "Only for lib/shared/ helpers.")
	writeAlignmentDoc(t, base, "naming.cpat.md", "Naming Change", "Handlers are named handle*.")
	writeAlignmentDoc(t, base, "why.adr.md", "Decision", "No path here.")
	writeStatusDoc(t, base, "draft.rule.md", "Draft Rule", "draft", "No path here.")

	fc, err := ResolveFileContext(base, "src/api/h.go")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, row := range fc.General {
		got = append(got, row.Doc.Path)
	}
	want := []string{".archcore/guard.rule.md", ".archcore/tests.rule.md", ".archcore/naming.cpat.md"}
	if !slices.Equal(got, want) {
		t.Errorf("general = %v, want %v", got, want)
	}
}

// TestCodeAlignment_OutsideSourceRoots pins behavior 11: a root config file
// still gets the documents naming it and the general rules.
func TestCodeAlignment_OutsideSourceRoots(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "vitest.doc.md", "Vitest Config", "Settings live in vitest.config.mjs.")
	writeAlignmentDoc(t, base, "guard.rule.md", "Guard Clauses", "Return early.")

	got := CodeAlignment(base, "vitest.config.mjs")

	for _, want := range []string{"Vitest Config", "Rules that name no path apply to every file:", "Guard Clauses"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
}

// TestCodeAlignment_StaysInsideTheRuneBudget pins the hook constraint: long
// titles drop general rows first, and the remainder is stated.
func TestCodeAlignment_StaysInsideTheRuneBudget(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	long := strings.Repeat("Очень длинный заголовок правила ", 6)
	for i := range 12 {
		writeAlignmentDoc(t, base, fmt.Sprintf("g-%02d.rule.md", i), fmt.Sprintf("%s %d", long, i), "Return early.")
	}
	for i := range 5 {
		writeAlignmentDoc(t, base, fmt.Sprintf("m-%d.rule.md", i), fmt.Sprintf("%s m%d", long, i), "Applies to src/api/ code.")
	}

	got := CodeAlignment(base, "src/api/h.go")

	if n := len([]rune(got)); n > maxAlignmentRunes {
		t.Errorf("hook text holds %d runes, cap %d", n, maxAlignmentRunes)
	}
	if !strings.Contains(got, `list_documents(types=["rule","cpat"], status="accepted")`) {
		t.Errorf("general remainder line missing:\n%s", got)
	}
}

// TestResolveFileContext_Kind: an accepted rule or cpat naming the file's
// compound extension governs every file of that kind, between file and
// directory matches.
func TestCountStandalone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		content string
		want    int
	}{
		{"Name tests *.test.tsx", 1},
		{"Use `.test.tsx` files", 1},
		{".test.tsx at the start", 1},
		{"See `Form.test.tsx`", 0},
		{"`a-b.test.tsx` and `*.test.tsx`", 1},
		{"no extension here", 0},
	}
	for _, tt := range tests {
		if got := countStandalone(tt.content, ".test.tsx"); got != tt.want {
			t.Errorf("countStandalone(%q) = %d, want %d", tt.content, got, tt.want)
		}
	}
}

func TestResolveFileContext_Kind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		file string
		want map[string]ContextReason // path -> reason; absent paths must not be matched
	}{
		{
			name: "a test file gets the rule naming .test.tsx as kind",
			file: "src/components/Foo/Foo.test.tsx",
			want: map[string]ContextReason{".archcore/tests.rule.md": ReasonKind, ".archcore/dir.rule.md": ReasonDirectory},
		},
		{
			name: "a plain component file does not",
			file: "src/components/Foo/Foo.tsx",
			want: map[string]ContextReason{".archcore/dir.rule.md": ReasonDirectory},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			mkdirs(t, base, "src/components")
			writeAlignmentDoc(t, base, "tests.rule.md", "Test Files", "Name tests *.test.tsx next to their sources.")
			writeAlignmentDoc(t, base, "tests.adr.md", "Test Decision", "We chose *.test.tsx files.")
			writeAlignmentDoc(t, base, "promo.rule.md", "Promo Flow", "Covered by `PromoForm.test.tsx` and `PromoQuery.test.tsx`.")
			writeAlignmentDoc(t, base, "dir.rule.md", "Components", "Applies to src/components/Foo/ code.")

			fc, err := ResolveFileContext(base, tt.file)
			if err != nil {
				t.Fatal(err)
			}
			got := make(map[string]ContextReason)
			var order []ContextReason
			for _, row := range fc.Matched {
				got[row.Doc.Path] = row.Reason
				order = append(order, row.Reason)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("matched = %v, want %v", got, tt.want)
			}
			if !slices.IsSortedFunc(order, func(a, b ContextReason) int { return reasonRank[a] - reasonRank[b] }) {
				t.Errorf("reasons out of order: %v", order)
			}
		})
	}
}

func TestResolveFileContext_RefusesPathsItCannotServe(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	tests := []struct {
		name string
		path string
		want error
	}{
		{name: "relative escape", path: "../other/x.go", want: ErrOutsideProject},
		{name: "absolute outside", path: filepath.Join(filepath.Dir(base), "x.go"), want: ErrOutsideProject},
		{name: "archcore document", path: ".archcore/a.rule.md", want: ErrArchcoreDocPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ResolveFileContext(base, tt.path); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
