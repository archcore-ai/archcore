package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/docs"
	"archcore-cli/internal/sync"
)

func TestBuildSessionContext_Empty(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	ctx, _ := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "Archcore") {
		t.Error("missing Archcore header")
	}
	// An empty corpus states so and points at the way out, rather than printing
	// three empty category headings the model has to interpret.
	if !strings.Contains(ctx, "CORPUS: no documents yet") {
		t.Errorf("empty corpus should say so; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "create_document") {
		t.Errorf("empty corpus should name the tool that fixes it; ctx=%q", ctx)
	}
	for _, absent := range []string{"IN PROGRESS", "RECENTLY ACCEPTED"} {
		if strings.Contains(ctx, absent) {
			t.Errorf("empty corpus should not render the %q block; ctx=%q", absent, ctx)
		}
	}
}

func TestBuildSessionContext_WithDocs(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	knowledgeDoc := filepath.Join(base, ".archcore", "knowledge", "use-postgres.adr.md")
	if err := os.WriteFile(knowledgeDoc, []byte("---\ntitle: Use PostgreSQL\nstatus: accepted\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	visionDoc := filepath.Join(base, ".archcore", "vision", "mvp.plan.md")
	if err := os.WriteFile(visionDoc, []byte("---\ntitle: MVP Plan\nstatus: draft\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx, _ := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "use-postgres.adr.md") {
		t.Error("context missing knowledge doc")
	}
	if !strings.Contains(ctx, "mvp.plan.md") {
		t.Error("context missing vision doc")
	}
	// session-start-context.spec §12 and §13.
	if !strings.HasPrefix(ctx, "[Archcore — Git-native context for AI coding agents]\n"+agents.ContextAddress+"\n") {
		t.Error("the context address does not follow the header line")
	}
	if strings.Contains(ctx, "Refer to MCP server instructions") {
		t.Error("the context still closes with a pointer to the server instructions")
	}
}

func TestBuildSessionContext_WithRelations(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	m := sync.NewManifest()
	m.AddRelation("knowledge/a.adr.md", "vision/b.prd.md", sync.RelImplements)
	m.AddRelation("knowledge/a.adr.md", "knowledge/c.rfc.md", sync.RelRelated)
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	ctx, _ := buildSessionContext(bg(), base)
	if !strings.Contains(ctx, "DOCUMENT RELATIONS") {
		t.Error("expected DOCUMENT RELATIONS section")
	}
	if !strings.Contains(ctx, "2 relation(s)") {
		t.Error("expected '2 relation(s)' count")
	}
}

func TestBuildSessionContext_MentionsRelationTools(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	ctx, _ := buildSessionContext(bg(), base)
	if !strings.Contains(ctx, "add_relation") {
		t.Error("expected 'add_relation' in context")
	}
	if !strings.Contains(ctx, "list_relations") {
		t.Error("expected 'list_relations' in context")
	}
	if !strings.Contains(ctx, "remove_relation") {
		t.Error("expected 'remove_relation' in context")
	}
}

func TestBuildSessionContext_WithTags(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	doc := filepath.Join(base, ".archcore", "knowledge", "tagged.adr.md")
	if err := os.WriteFile(doc, []byte("---\ntitle: Tagged Doc\nstatus: draft\ntags:\n  - frontend\n  - auth\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc2 := filepath.Join(base, ".archcore", "vision", "tagged2.prd.md")
	if err := os.WriteFile(doc2, []byte("---\ntitle: Another\nstatus: draft\ntags:\n  - frontend\n  - backend\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, _ := buildSessionContext(bg(), base)
	if !strings.Contains(ctx, "EXISTING TAGS:") {
		t.Error("expected EXISTING TAGS section")
	}
	if !strings.Contains(ctx, "frontend") {
		t.Error("expected 'frontend' tag in context")
	}
	if !strings.Contains(ctx, "auth") {
		t.Error("expected 'auth' tag in context")
	}
	if !strings.Contains(ctx, "backend") {
		t.Error("expected 'backend' tag in context")
	}
}

func TestBuildSessionContext_NoTags(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	doc := filepath.Join(base, ".archcore", "knowledge", "untagged.adr.md")
	if err := os.WriteFile(doc, []byte("---\ntitle: Untagged\nstatus: draft\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, _ := buildSessionContext(bg(), base)
	if strings.Contains(ctx, "EXISTING TAGS:") {
		t.Error("should not show EXISTING TAGS when no tags exist")
	}
}

// TestBuildSessionContext_ExcludesGlobals verifies mounted read-only globals do
// not leak into the session-start context — they are surfaced only through the
// MCP read tools, never in CLI-injected context.
func TestBuildSessionContext_ExcludesGlobals(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	localDoc := filepath.Join(base, ".archcore", "knowledge", "local-rule.rule.md")
	if err := os.WriteFile(localDoc, []byte("---\ntitle: Local Rule\nstatus: accepted\ntags:\n  - localtag\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	globalDoc := filepath.Join(base, "vendor", "company", "knowledge", "company-rule.rule.md")
	if err := os.MkdirAll(filepath.Dir(globalDoc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalDoc, []byte("---\ntitle: Company Rule\nstatus: accepted\ntags:\n  - globaltag\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settings, []byte(`{"sync":"none","globals":[{"id":"company","path":"vendor/company"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, _ := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "local-rule.rule.md") {
		t.Error("local doc should appear in session context")
	}
	if strings.Contains(ctx, "company-rule.rule.md") {
		t.Error("global doc must NOT appear in session context")
	}
	if strings.Contains(ctx, "globaltag") {
		t.Error("global-only tag must NOT appear in session context")
	}
}

// TestBuildSessionContext_MissingGlobalDegradesToLocal guards the regression where a
// declared-but-absent global made ScanDocuments error and the hook blanked the whole
// context — dropping every LOCAL doc too, silently. The hook must instead keep the
// local docs and surface a visible warning (a missing global is loud, not
// silent), without leaking an absolute path.
func TestBuildSessionContext_MissingGlobalDegradesToLocal(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	localDoc := filepath.Join(base, ".archcore", "knowledge", "local-rule.rule.md")
	if err := os.WriteFile(localDoc, []byte("---\ntitle: Local Rule\nstatus: accepted\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Declare a global that is not on disk.
	settings := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settings, []byte(`{"sync":"none","globals":[{"id":"company","path":"../missing/.archcore"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, n := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "local-rule.rule.md") {
		t.Error("local doc must survive a missing global, not be blanked")
	}
	if n.local != 1 {
		t.Errorf("doc count = %d, want 1 surviving local doc", n.local)
	}
	if !strings.Contains(ctx, "⚠") || !strings.Contains(ctx, `"company"`) {
		t.Errorf("expected a visible warning naming the missing global; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "../missing/.archcore") {
		t.Error("warning should cite the declared relative path, not an absolute one")
	}
}

func TestBuildSessionContext_PlanProgress(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	tracked := "---\ntitle: Checks\nstatus: draft\n---\n## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - proof: cache test @ e67a927\n\n2. [ ] T2 Add the cap\n\n   - proof: —\n"
	legacy := "---\ntitle: Old\nstatus: draft\n---\n## Tasks\n\n1. Add the model.\n\n## Acceptance Criteria\n\n- [ ] Criterion 1\n"
	accepted := strings.Replace(tracked, "status: draft", "status: accepted", 1)
	for name, body := range map[string]string{"checks.plan.md": tracked, "old.plan.md": legacy, "done.plan.md": accepted} {
		if err := os.WriteFile(filepath.Join(base, ".archcore", name), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	ctx, _ := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, `.archcore/checks.plan.md — "Phase 1: Cache": 1 of 2 proven; next: "T2 Add the cap"`) {
		t.Errorf("missing progress line for the proof-tracked plan; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "a checked task counts only with `proof: <check> @ <sha>`, where <sha> is the first commit that passes the check and ends the line") {
		t.Errorf("the block header must name the proof form; ctx=%q", ctx)
	}
	if _, block, _ := strings.Cut(ctx, "PLAN PROGRESS"); strings.Contains(block, "old.plan.md") || strings.Contains(block, "done.plan.md") {
		t.Errorf("a legacy plan must not get a progress line; ctx=%q", ctx)
	}
}

func TestBuildSessionContext_PlanProgressCapAndOrder(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"a", "b", "c", "d", "e"} {
		path := filepath.Join(base, ".archcore", name+".plan.md")
		body := "---\ntitle: " + name + "\nstatus: draft\n---\n## Tasks\n\n1. [ ] T1 Do " + name + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		mtime := start.Add(time.Duration(i) * time.Hour)
		if name == "e" {
			mtime = start.Add(3 * time.Hour) // ties with d; the path breaks the tie
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
	}

	ctx, _ := buildSessionContext(bg(), base)

	_, block, _ := strings.Cut(ctx, "PLAN PROGRESS")
	d, e, c := strings.Index(block, "d.plan.md"), strings.Index(block, "e.plan.md"), strings.Index(block, "c.plan.md")
	if d < 0 || e < 0 || c < 0 || d > e || e > c {
		t.Errorf("want d, e, c newest first; block=%q", block)
	}
	if strings.Contains(block, "a.plan.md") || strings.Contains(block, "b.plan.md") {
		t.Errorf("want only %d lines; block=%q", maxPlanProgressLines, block)
	}
	if !strings.Contains(block, "… and 2 more") {
		t.Errorf("want an overflow line; block=%q", block)
	}
}

func TestPlanProgressText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   docs.PlanProgress
		want string
	}{
		{"all proven", docs.PlanProgress{}, "every task proven; ready for closeout"},
		{"claims and waiting", docs.PlanProgress{Phase: "Phase 1", Total: 2, Claimed: 2, Waiting: 1}, `"Phase 1": 0 of 2 proven; 2 checked without a proof in that form; waiting on 1`},
		{"unexpanded phase", docs.PlanProgress{Phase: "Phase 2: Index"}, `"Phase 2: Index": expand this phase`},
		{"quoted and truncated", docs.PlanProgress{Phase: "P\nX", Total: 1, Next: strings.Repeat("é", 130)}, `"P\nX": 0 of 1 proven; next: "` + strings.Repeat("é", 120) + `…"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := planProgressText(tt.in); got != tt.want {
				t.Errorf("planProgressText = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildSessionContext_NoPlanProgressForLegacyPlans(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	legacy := "---\ntitle: Old\nstatus: draft\n---\n## Tasks\n\n1. Add the model.\n\n## Acceptance Criteria\n\n- [x] Criterion 1\n"
	if err := os.WriteFile(filepath.Join(base, ".archcore", "old.plan.md"), []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx, _ := buildSessionContext(bg(), base)

	if strings.Contains(ctx, "PLAN PROGRESS") {
		t.Errorf("a corpus of legacy plans must render no PLAN PROGRESS block; ctx=%q", ctx)
	}
}
