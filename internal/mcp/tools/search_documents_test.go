package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// unmarshalSearch decodes a tool result into []searchResult. It fails the test
// if decoding fails or the result is marked as an error.
func unmarshalSearch(t *testing.T, result *mcp.CallToolResult) []searchResult {
	t.Helper()
	if result == nil {
		t.Fatal("nil result")
	}
	if result.IsError {
		var text string
		if len(result.Content) > 0 {
			if tc, ok := mcp.AsTextContent(result.Content[0]); ok {
				text = tc.Text
			}
		}
		t.Fatalf("unexpected error result: %s", text)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("unexpected content type %T", result.Content[0])
	}
	var out []searchResult
	if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
		t.Fatalf("unmarshal: %v\npayload: %s", err, tc.Text)
	}
	return out
}

// setMtime changes the mtime on a written document.
func setMtime(t *testing.T, base, relPath string, mtime time.Time) {
	t.Helper()
	full := filepath.Join(base, relPath)
	if err := os.Chtimes(full, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestHandleSearchDocuments_EmptyFilters(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for empty filters")
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("unexpected content type %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, "at least one filter") {
		t.Errorf("error text = %q, want mention of filters", tc.Text)
	}
}

func TestHandleSearchDocuments_PathRefExplicit(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "money.rule.md",
		"---\ntitle: Money Arithmetic\nstatus: accepted\n---\n\nmonetary amounts in @src/payments/ MUST use Decimal.")
	writeDoc(t, base, "knowledge", "auth.rule.md",
		"---\ntitle: Auth\nstatus: accepted\n---\n\nno ref here")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "@src/payments/",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Title != "Money Arithmetic" {
		t.Errorf("title = %q", got[0].Title)
	}
	if len(got[0].Matches) != 1 {
		t.Fatalf("expected 1 match record, got %d", len(got[0].Matches))
	}
	m := got[0].Matches[0]
	if m.Kind != "path_ref_explicit" {
		t.Errorf("kind = %q, want path_ref_explicit", m.Kind)
	}
	if m.Specificity != 2 {
		t.Errorf("specificity = %d, want 2", m.Specificity)
	}
	if !strings.Contains(m.Excerpt, "@src/payments/") {
		t.Errorf("excerpt should contain @src/payments/: %q", m.Excerpt)
	}
}

func TestHandleSearchDocuments_PathRefBareTrailingSlash(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "api.rule.md",
		"---\ntitle: API Rules\nstatus: accepted\n---\n\nHandlers in src/api/ MUST validate inputs.")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "src/api/",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Matches[0].Kind != "path_ref_mention" {
		t.Errorf("kind = %q, want path_ref_mention", got[0].Matches[0].Kind)
	}
}

func TestHandleSearchDocuments_PathRefBareSourceExtension(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "stripe.rule.md",
		"---\ntitle: Stripe\nstatus: accepted\n---\n\nrefer to src/payments/stripe.ts for details")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "src/payments/stripe.ts",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Matches[0].Specificity != 3 {
		t.Errorf("specificity = %d, want 3", got[0].Matches[0].Specificity)
	}
}

func TestHandleSearchDocuments_PathRefRejectsURLLike(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// "example/com" has one slash and a non-source extension segment ("com"
	// has no extension at all). Should be rejected.
	writeDoc(t, base, "knowledge", "linkrot.doc.md",
		"---\ntitle: Linkrot\nstatus: accepted\n---\n\nsee docs.example/com for more info")
	// Similarly, "cd/oldpath" — single slash, no extension.
	writeDoc(t, base, "knowledge", "cmd.doc.md",
		"---\ntitle: Cmd\nstatus: accepted\n---\n\nrun cd/oldpath to move")

	// Search for the substring that would match if the regex were too loose.
	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "example/com",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 0 {
		t.Errorf("expected 0 matches (URL-like rejected), got %d: %+v", len(got), got)
	}

	result2, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "cd/oldpath",
	})
	if err != nil {
		t.Fatal(err)
	}
	got2 := unmarshalSearch(t, result2)
	if len(got2) != 0 {
		t.Errorf("expected 0 matches (single-slash no-ext rejected), got %d", len(got2))
	}
}

func TestHandleSearchDocuments_ContentTitleHit(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "money.rule.md",
		"---\ntitle: Money Arithmetic in Decimals\nstatus: accepted\n---\n\nuse Decimal for money.")
	writeDoc(t, base, "knowledge", "other.rule.md",
		"---\ntitle: Other\nstatus: accepted\n---\n\nnothing about money here either ... actually, body mentions money.")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "Money Arithmetic",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Matches[0].Specificity != 3 {
		t.Errorf("title hit specificity = %d, want 3", got[0].Matches[0].Specificity)
	}
}

func TestHandleSearchDocuments_ContentBodyHit(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "money.rule.md",
		"---\ntitle: Finance\nstatus: accepted\n---\n\nMoney Arithmetic belongs here.")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "Money Arithmetic",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Matches[0].Specificity != 1 {
		t.Errorf("body hit specificity = %d, want 1", got[0].Matches[0].Specificity)
	}
}

func TestHandleSearchDocuments_ContentCaseInsensitive(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "money.rule.md",
		"---\ntitle: Money Arithmetic in Decimals\nstatus: accepted\n---\n\nuse Decimal.")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "MONEY arithmetic",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
}

func TestHandleSearchDocuments_TypesAndPathRefCombined(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "rule-payments.rule.md",
		"---\ntitle: Rule Payments\nstatus: accepted\n---\n\nuse @src/payments/ always")
	writeDoc(t, base, "knowledge", "adr-payments.adr.md",
		"---\ntitle: ADR Payments\nstatus: accepted\n---\n\nwe picked @src/payments/")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "@src/payments/",
		"types":    []any{"rule"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 (AND), got %d", len(got))
	}
	if got[0].Type != "rule" {
		t.Errorf("type = %q", got[0].Type)
	}
}

func TestHandleSearchDocuments_StatusAndMtimeAfterRelative(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "old.adr.md",
		"---\ntitle: Old\nstatus: accepted\n---\n\nbody")
	writeDoc(t, base, "knowledge", "recent.adr.md",
		"---\ntitle: Recent\nstatus: accepted\n---\n\nbody")
	writeDoc(t, base, "knowledge", "draft.adr.md",
		"---\ntitle: Draft\nstatus: draft\n---\n\nbody")

	now := time.Now()
	setMtime(t, base, ".archcore/knowledge/old.adr.md", now.Add(-90*24*time.Hour))
	setMtime(t, base, ".archcore/knowledge/recent.adr.md", now.Add(-1*24*time.Hour))
	setMtime(t, base, ".archcore/knowledge/draft.adr.md", now.Add(-1*24*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"status":      "accepted",
		"mtime_after": "30d",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %+v", len(got), got)
	}
	if got[0].Title != "Recent" {
		t.Errorf("title = %q, want Recent", got[0].Title)
	}
}

func TestHandleSearchDocuments_MtimeAfterISO8601(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "old.adr.md",
		"---\ntitle: Old\nstatus: accepted\n---\n\nbody")
	writeDoc(t, base, "knowledge", "new.adr.md",
		"---\ntitle: New\nstatus: accepted\n---\n\nbody")

	cutoff := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	setMtime(t, base, ".archcore/knowledge/old.adr.md", cutoff.Add(-24*time.Hour))
	setMtime(t, base, ".archcore/knowledge/new.adr.md", cutoff.Add(24*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"status":      "accepted",
		"mtime_after": cutoff.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Title != "New" {
		t.Errorf("title = %q, want New", got[0].Title)
	}
}

func TestHandleSearchDocuments_MtimeAfterInvalid(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"status":      "accepted",
		"mtime_after": "not-a-date",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid mtime_after")
	}
}

func TestHandleSearchDocuments_LimitTruncation(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for i, slug := range []string{"a", "b", "c", "d", "e"} {
		writeDoc(t, base, "knowledge", slug+".rule.md",
			"---\ntitle: T"+slug+"\nstatus: accepted\n---\n\nbody")
		// Give each a distinct mtime so ordering is deterministic.
		setMtime(t, base, ".archcore/knowledge/"+slug+".rule.md",
			time.Now().Add(-time.Duration(5-i)*time.Hour))
	}

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"limit": float64(2),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 2 {
		t.Fatalf("expected 2 (limit), got %d", len(got))
	}
	if got[0].Title != "Te" {
		t.Errorf("got[0].Title = %q, want %q (newest by mtime)", got[0].Title, "Te")
	}
	if got[1].Title != "Td" {
		t.Errorf("got[1].Title = %q, want %q (second-newest by mtime)", got[1].Title, "Td")
	}
}

func TestHandleSearchDocuments_SortRelevanceTypePriority(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// Two docs, same specificity (both body mentions of "alpha"), different
	// types — rule outranks plan.
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: RuleDoc\nstatus: accepted\n---\n\nalpha is in the body")
	writeDoc(t, base, "vision", "b.plan.md",
		"---\ntitle: PlanDoc\nstatus: accepted\n---\n\nalpha is in the body")

	// Make plan NEWER than rule — so pure mtime order would invert them.
	// Relevance must still put rule first.
	setMtime(t, base, ".archcore/knowledge/a.rule.md", time.Now().Add(-2*time.Hour))
	setMtime(t, base, ".archcore/vision/b.plan.md", time.Now().Add(-1*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "alpha",
		"sort":    "relevance",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Type != "rule" {
		t.Errorf("sort=relevance first type = %q, want rule", got[0].Type)
	}
	if got[1].Type != "plan" {
		t.Errorf("sort=relevance second type = %q, want plan", got[1].Type)
	}
}

func TestHandleSearchDocuments_SortMtimeIgnoresTypePriority(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: RuleDoc\nstatus: accepted\n---\n\nalpha is in the body")
	writeDoc(t, base, "vision", "b.plan.md",
		"---\ntitle: PlanDoc\nstatus: accepted\n---\n\nalpha is in the body")

	// plan newer than rule; sort=mtime must put plan first.
	setMtime(t, base, ".archcore/knowledge/a.rule.md", time.Now().Add(-2*time.Hour))
	setMtime(t, base, ".archcore/vision/b.plan.md", time.Now().Add(-1*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "alpha",
		"sort":    "mtime",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Type != "plan" {
		t.Errorf("sort=mtime first type = %q, want plan", got[0].Type)
	}
}

// TestHandleSearchDocuments_SortRelevanceExtendedTypes exercises the newly
// ranked types in typePriority (rfc, doc) that previously fell back to
// typePriorityDefault. rfc is rank 3, doc is rank 17 — rfc must win on
// relevance even when doc is newer.
func TestHandleSearchDocuments_SortRelevanceExtendedTypes(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rfc.md",
		"---\ntitle: RfcDoc\nstatus: accepted\n---\n\nalpha is in the body")
	writeDoc(t, base, "knowledge", "b.doc.md",
		"---\ntitle: DocDoc\nstatus: accepted\n---\n\nalpha is in the body")

	// doc newer than rfc — pure mtime would invert them.
	setMtime(t, base, ".archcore/knowledge/a.rfc.md", time.Now().Add(-2*time.Hour))
	setMtime(t, base, ".archcore/knowledge/b.doc.md", time.Now().Add(-1*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "alpha",
		"sort":    "relevance",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Type != "rfc" {
		t.Errorf("sort=relevance first type = %q, want rfc (rank 3 vs doc rank 17)", got[0].Type)
	}
	if got[1].Type != "doc" {
		t.Errorf("sort=relevance second type = %q, want doc", got[1].Type)
	}
}

func TestHandleSearchDocuments_IncomingRelations(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "target.rule.md",
		"---\ntitle: Target Rule\nstatus: accepted\n---\n\nuse @src/payments/ always")
	writeDoc(t, base, "knowledge", "guide.guide.md",
		"---\ntitle: Guide\nstatus: accepted\n---\n\nhow-to content")

	m := sync.NewManifest()
	m.AddRelation("knowledge/guide.guide.md", "knowledge/target.rule.md", sync.RelImplements)
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "@src/payments/",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if len(got[0].IncomingRelations) != 1 {
		t.Fatalf("expected 1 incoming relation, got %d", len(got[0].IncomingRelations))
	}
	if got[0].IncomingRelations[0].Path != ".archcore/knowledge/guide.guide.md" {
		t.Errorf("incoming path = %q", got[0].IncomingRelations[0].Path)
	}
	if got[0].IncomingRelations[0].Type != "implements" {
		t.Errorf("incoming type = %q", got[0].IncomingRelations[0].Type)
	}
}

func TestHandleSearchDocuments_PureMetadataReturnsEmptyMatches(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "only.rule.md",
		"---\ntitle: Only\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Matches == nil {
		t.Error("matches should be non-nil empty slice")
	}
	if len(got[0].Matches) != 0 {
		t.Errorf("matches should be empty, got %d", len(got[0].Matches))
	}
}

func TestHandleSearchDocuments_LimitClampedToMax(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"limit": float64(999),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Expect success, not error (limit is clamped, not rejected).
	if result.IsError {
		t.Error("unexpected error on over-max limit (should clamp)")
	}
	got := unmarshalSearch(t, result)
	if len(got) > searchMaxLimit {
		t.Errorf("snippets mode clamped to %d, got %d", searchMaxLimit, len(got))
	}
}

func TestHandleSearchDocuments_NegativeLimitRejected(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"limit": float64(-5),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for negative limit")
	}
}

func TestHandleSearchDocuments_NoManifestOK(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nuse @src/payments/")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "@src/payments/",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].IncomingRelations == nil || got[0].OutgoingRelations == nil {
		t.Error("relations slices should be non-nil (empty) even without manifest")
	}
}

func TestHandleSearchDocuments_ExcerptRuneSafe(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// Cyrillic body where the match window will fall inside a multibyte rune
	// unless buildExcerpt snaps to rune boundaries.
	writeDoc(t, base, "knowledge", "ru.rule.md",
		"---\ntitle: Правило\nstatus: accepted\n---\n\n"+
			"В этом документе мы обсуждаем важный KEYWORD и контекст вокруг него — лишние слова, чтобы excerpt уходил за границу руны.")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "KEYWORD",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	excerpt := got[0].Matches[0].Excerpt
	if !utf8.ValidString(excerpt) {
		t.Errorf("excerpt is not valid UTF-8: %q", excerpt)
	}
	if !strings.Contains(excerpt, "KEYWORD") {
		t.Errorf("excerpt missing match token: %q", excerpt)
	}
}

func TestHandleSearchDocuments_LazyBodyLoad(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// Pure metadata query (types + status) must not depend on doc bodies.
	// We verify indirectly: the doc has a body referencing @src/payments/
	// but the query does not supply path_ref, so no match records should be
	// emitted (matches must be empty), and the filter still works.
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody mentions @src/payments/ but query ignores it")
	writeDoc(t, base, "knowledge", "b.rule.md",
		"---\ntitle: B\nstatus: draft\n---\n\nwhatever")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types":  []any{"rule"},
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1 match (accepted rule), got %d", len(got))
	}
	if len(got[0].Matches) != 0 {
		t.Errorf("pure metadata query should yield empty matches, got %d", len(got[0].Matches))
	}
}

func TestHandleSearchDocuments_UnknownTypeIsError(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"nonexistent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// A typo'd type previously matched nothing, indistinguishable from "no
	// documents" — the sibling category/status filters always errored.
	if !result.IsError {
		t.Fatal("expected error for unknown type filter")
	}
	if msg := resultText(t, result); !strings.Contains(msg, `invalid type "nonexistent"`) {
		t.Errorf("message = %q, want invalid-type error", msg)
	}
}

func TestHandleSearchDocuments_EmptyBodyNoMatch(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\n")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"path_ref": "@src/payments/",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 0 {
		t.Errorf("expected 0 matches for empty body, got %d", len(got))
	}
}

func TestHandleSearchDocuments_RelevanceMtimeTiebreak(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "older.rule.md",
		"---\ntitle: Older\nstatus: accepted\n---\n\nalpha is in the body")
	writeDoc(t, base, "knowledge", "newer.rule.md",
		"---\ntitle: Newer\nstatus: accepted\n---\n\nalpha is in the body")

	setMtime(t, base, ".archcore/knowledge/older.rule.md", time.Now().Add(-2*time.Hour))
	setMtime(t, base, ".archcore/knowledge/newer.rule.md", time.Now().Add(-1*time.Hour))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "alpha",
		"sort":    "relevance",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Title != "Newer" {
		t.Errorf("got[0].Title = %q, want %q (newer wins tiebreak)", got[0].Title, "Newer")
	}
	if got[1].Title != "Older" {
		t.Errorf("got[1].Title = %q, want %q", got[1].Title, "Older")
	}
}

func TestHandleSearchDocuments_MtimeAfterZeroDays(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")
	setMtime(t, base, ".archcore/knowledge/a.adr.md", time.Now().Add(-time.Minute))

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"mtime_after": "0d",
		"status":      "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result on mtime_after=0d")
	}
	got := unmarshalSearch(t, result)
	if len(got) != 0 {
		t.Errorf("expected 0 matches for mtime_after=0d, got %d", len(got))
	}
}

// --- mode=full tests ---

func TestHandleSearchDocuments_FullModeBodyPresentPureMetadata(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: Payments Rule\nstatus: accepted\n---\n\nuse Decimal for all monetary amounts")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "full",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Body == "" {
		t.Error("mode=full with types-only filter must populate Body")
	}
	if !strings.Contains(got[0].Body, "Decimal") {
		t.Errorf("Body should contain document content, got: %q", got[0].Body)
	}
	// Frontmatter must be stripped.
	if strings.Contains(got[0].Body, "---") {
		t.Errorf("Body should not contain frontmatter delimiters, got: %q", got[0].Body)
	}
}

func TestHandleSearchDocuments_FullModeBodyPresentWithContent(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: Decimal Rule\nstatus: accepted\n---\n\nuse Decimal for monetary amounts")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"content": "Decimal",
		"mode":    "full",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Body == "" {
		t.Error("mode=full with content filter must populate Body")
	}
}

func TestHandleSearchDocuments_SnippetsModeNoBody(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody text here")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "snippets",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Body != "" {
		t.Errorf("mode=snippets must not populate Body, got: %q", got[0].Body)
	}
	// Verify Body is absent from JSON output (omitempty).
	tc, _ := result.Content[0].(interface{ GetText() string })
	_ = tc
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(interface{ GetText() string }); ok {
			if strings.Contains(textContent.GetText(), `"body"`) {
				t.Error("Body field must be absent from JSON in snippets mode")
			}
		}
	}
}

func TestHandleSearchDocuments_FullModeDefaultLimitIsThree(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for _, slug := range []string{"a", "b", "c", "d", "e"} {
		writeDoc(t, base, "knowledge", slug+".rule.md",
			"---\ntitle: T"+slug+"\nstatus: accepted\n---\n\nbody")
	}

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "full",
		// limit omitted — must default to searchFullDefaultLimit=3
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != searchFullDefaultLimit {
		t.Errorf("full mode default limit = %d, got %d", searchFullDefaultLimit, len(got))
	}
}

func TestHandleSearchDocuments_FullModeDefaultLimitOverridable(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for _, slug := range []string{"a", "b", "c", "d", "e"} {
		writeDoc(t, base, "knowledge", slug+".rule.md",
			"---\ntitle: T"+slug+"\nstatus: accepted\n---\n\nbody")
	}

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "full",
		"limit": float64(5),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 5 {
		t.Errorf("expected 5 (explicit limit), got %d", len(got))
	}
}

func TestHandleSearchDocuments_FullModeLimitClampedTo20(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "full",
		"limit": float64(999),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("unexpected error on over-max limit in full mode (should clamp)")
	}
	got := unmarshalSearch(t, result)
	if len(got) > searchFullMaxLimit {
		t.Errorf("full mode clamped to %d, got %d", searchFullMaxLimit, len(got))
	}
}

func TestHandleSearchDocuments_FullModeFrontmatterOnlyDocEmptyBody(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// Document has frontmatter but no body content.
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: Empty Body\nstatus: accepted\n---\n")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "full",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if strings.Contains(got[0].Body, "---") {
		t.Errorf("Body must not contain raw frontmatter, got: %q", got[0].Body)
	}
}

func TestHandleSearchDocuments_UnknownModeFallsBackToSnippets(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody text")

	result, err := callTool(HandleSearchDocuments(base), map[string]any{
		"types": []any{"rule"},
		"mode":  "unknown_value",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("unknown mode must not return an error (fallback to snippets)")
	}
	got := unmarshalSearch(t, result)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Body != "" {
		t.Errorf("unknown mode must behave as snippets (no Body), got: %q", got[0].Body)
	}
}
