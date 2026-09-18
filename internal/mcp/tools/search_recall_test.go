package tools

// Tests for the recall guarantees (global-recall-guarantees.rfc): tokenized
// matching, the coverage envelope, the source scope, and per-source
// representation on the cut page.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/docs"

	"github.com/mark3labs/mcp-go/mcp"
)

// searchFixture builds a project with local documents and one declared global
// source. Both hold a rule on the same topic under different phrasing — the
// live reproduction from the RFC.
func searchFixture(t *testing.T) string {
	t.Helper()
	base, localArch, globalArch := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "compat.rule.md",
		"Compatibility Contract Between the CLI and the Plugin",
		"## Rule\n\nThe plugin ships its own schedule.\n")
	writeFixtureDoc(t, globalArch, "compat.rule.md",
		"Plugin / CLI Compatibility Across Independent Release Trains",
		"## Rule\n\nEach train releases alone.\n")
	return base
}

func writeFixtureDoc(t *testing.T, dir, name, title, body string) {
	t.Helper()
	content := "---\ntitle: \"" + title + "\"\nstatus: accepted\n---\n\n" + body
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSearchDocuments_AllModeMatchesAcrossWordGaps: the RFC's reproduced miss —
// "plugin compatibility" must match "Plugin / CLI Compatibility" and the local
// title too, because every token occurs regardless of order and distance.
func TestSearchDocuments_AllModeMatchesAcrossWordGaps(t *testing.T) {
	t.Parallel()
	base := searchFixture(t)

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "plugin compatibility"}))

	if len(got) != 2 {
		t.Fatalf("got %d results, want both same-topic rules; results=%+v", len(got), got)
	}
	kinds := map[docs.SourceKind]bool{}
	for _, r := range got {
		kinds[r.SourceKind] = true
	}
	if !kinds[docs.SourceKindLocal] || !kinds[docs.SourceKindGlobal] {
		t.Errorf("want one local and one global result, got %v", kinds)
	}
}

// TestSearchDocuments_GlobalMtimeIsNotARankingSignal pins the effective-mtime
// key (search-documents.spec §7.1): a global document's mtime is its clone
// date, so on an otherwise full tie the local document ranks first even when
// the global file is newer on disk.
func TestSearchDocuments_GlobalMtimeIsNotARankingSignal(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	// Same type, same title hit, same score — only source and mtime differ.
	writeFixtureDoc(t, localArch, "retry.rule.md", "Retry Policy Local", "Body.\n")
	writeFixtureDoc(t, globalArch, "retry.rule.md", "Retry Policy Global", "Body.\n")

	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(localArch, "retry.rule.md"), old, old); err != nil {
		t.Fatal(err)
	}
	// The global stays freshly written — a clone-date artifact by construction.

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "retry policy"}))
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	if got[0].SourceKind != docs.SourceKindLocal {
		t.Errorf("first result is %s (%s), want the local document despite the older mtime",
			got[0].SourceKind, got[0].Path)
	}
}

// TestSearchDocuments_MatchModes: exact keeps the substring semantics, any
// admits a single hit, all requires every token.
func TestSearchDocuments_MatchModes(t *testing.T) {
	t.Parallel()
	base := searchFixture(t)

	tests := []struct {
		name    string
		args    map[string]any
		wantLen int
	}{
		{"exact misses across the gap", map[string]any{"content": "plugin compatibility", "match": "exact"}, 0},
		{"exact hits the literal phrase", map[string]any{"content": "Compatibility Across", "match": "exact"}, 1},
		{"all drops a doc missing one token", map[string]any{"content": "plugin schedule"}, 1},
		{"any keeps a doc with one token", map[string]any{"content": "schedule trains", "match": "any"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, tt.args))
			if len(got) != tt.wantLen {
				t.Errorf("got %d results, want %d; results=%+v", len(got), tt.wantLen, got)
			}
		})
	}
}

// TestSearchDocuments_CoverageNamesEverySearchedSource: the envelope reports
// scanned counts per source, and an empty result keeps them — a verified
// absence, not an unfalsifiable blank.
func TestSearchDocuments_CoverageNamesEverySearchedSource(t *testing.T) {
	t.Parallel()
	base := searchFixture(t)

	env := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{"content": "zephyrite-absent-term"}))

	if len(env.Results) != 0 {
		t.Fatalf("got %d results, want 0", len(env.Results))
	}
	if env.Coverage["local"] != 1 || env.Coverage["org"] != 1 {
		t.Errorf("coverage = %v, want local:1 org:1 even on an empty result", env.Coverage)
	}
}

// TestSearchDocuments_SourceScope: source narrows both the results and the
// coverage to the scoped source.
func TestSearchDocuments_SourceScope(t *testing.T) {
	t.Parallel()
	base := searchFixture(t)

	tests := []struct {
		source       string
		wantKind     docs.SourceKind
		wantCoverage map[string]int
	}{
		{"local", docs.SourceKindLocal, map[string]int{"local": 1}},
		{"global", docs.SourceKindGlobal, map[string]int{"org": 1}},
		{"org", docs.SourceKindGlobal, map[string]int{"org": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			t.Parallel()
			env := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{
				"content": "compatibility", "source": tt.source,
			}))
			if len(env.Results) != 1 || env.Results[0].SourceKind != tt.wantKind {
				t.Fatalf("source=%q: results=%+v, want one %s row", tt.source, env.Results, tt.wantKind)
			}
			if len(env.Coverage) != len(tt.wantCoverage) {
				t.Errorf("coverage = %v, want %v", env.Coverage, tt.wantCoverage)
			}
			for k, v := range tt.wantCoverage {
				if env.Coverage[k] != v {
					t.Errorf("coverage[%s] = %d, want %d", k, env.Coverage[k], v)
				}
			}
		})
	}
}

// TestSearchDocuments_SourceRepresentationSurvivesTheCut: with many local
// matches and a small limit, a matching global source keeps its top row on the
// page instead of being evicted wholesale — including in mode=full.
func TestSearchDocuments_SourceRepresentationSurvivesTheCut(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	// 10 local matches with title hits — each outscores the global body hit.
	for i := range 10 {
		writeFixtureDoc(t, localArch, "local-"+string(rune('a'+i))+".rule.md",
			"Retry Policy "+strings.Repeat("x", i+1), "Local body.\n")
	}
	writeFixtureDoc(t, globalArch, "org-retry.doc.md",
		"Org Guidance", "The retry policy for services is exponential backoff.\n")

	for _, mode := range []string{"snippets", "full"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, map[string]any{
				"content": "retry policy", "mode": mode, "limit": float64(3),
			}))
			if len(got) != 3 {
				t.Fatalf("got %d results, want 3", len(got))
			}
			globals := 0
			for _, r := range got {
				if r.Global {
					globals++
				}
			}
			if globals != 1 {
				t.Errorf("page carries %d global rows, want exactly the top global match kept; results=%+v", globals, got)
			}
		})
	}
}

// hostPreviewBytes is the prefix a host showed the agent when it stored an
// oversized result in a file (read-tool-responses-survive-host-truncation.adr).
const hostPreviewBytes = 2048

// TestSearchDocuments_EnvelopeLeadsWithCoverageHitsIndex replays the shape of
// the recorded incident: one local row ranks first, five global rows follow,
// and every body is larger than the preview.
func TestSearchDocuments_EnvelopeLeadsWithCoverageHitsIndex(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	filler := strings.Repeat("Κείμενο εγγράφου χωρίς αντιστοιχίες. ", 200)
	writeFixtureDoc(t, localArch, "packages.doc.md", "Auth Packages",
		strings.Repeat("needle ", 9)+filler)
	for i := range 5 {
		writeFixtureDoc(t, globalArch, "org-"+string(rune('a'+i))+".doc.md",
			"Org Document "+string(rune('A'+i)), "needle "+filler)
	}

	text := searchText(t, callSearch(t, base, map[string]any{
		"content": "needle", "mode": "full", "limit": float64(6),
	}))
	if len(text) <= hostPreviewBytes {
		t.Fatalf("response is %d bytes, so the fixture does not reach past the preview", len(text))
	}
	preview := text[:hostPreviewBytes]

	for _, want := range []string{
		`"coverage":{"local":1,"org":5}`,
		`"hits":{"local":1,"org":5}`,
		`"index":[`,
		`org-e.doc.md`,
	} {
		if !strings.Contains(preview, want) {
			t.Errorf("first %d bytes lack %s; an agent shown only the preview cannot tell that the global source matched", hostPreviewBytes, want)
		}
	}
	if results := strings.Index(text, `"results":`); results < strings.Index(text, `"index":`) {
		t.Errorf("results start at byte %d, before the index", results)
	}
}

// TestSearchDocuments_HitsCountBeforeThePageCut: hits reports what each source
// held, the index lists what the page kept, and a searched source without a
// match still appears with zero.
func TestSearchDocuments_HitsCountBeforeThePageCut(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	for i := range 10 {
		writeFixtureDoc(t, localArch, "local-"+string(rune('a'+i))+".rule.md",
			"Retry Policy "+strings.Repeat("x", i+1), "Local body.\n")
	}
	writeFixtureDoc(t, globalArch, "org-retry.doc.md",
		"Org Guidance", "The retry policy for services is exponential backoff.\n")

	t.Run("both sources match", func(t *testing.T) {
		t.Parallel()
		got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{
			"content": "retry policy", "limit": float64(3),
		}))
		if got.Hits["local"] != 10 || got.Hits["org"] != 1 {
			t.Errorf("hits = %v, want local=10 org=1", got.Hits)
		}
		if len(got.Index) != len(got.Results) {
			t.Fatalf("index holds %d entries for %d rows", len(got.Index), len(got.Results))
		}
		for i, row := range got.Results {
			entry := got.Index[i]
			if entry.Path != row.Path || entry.Title != row.Title || entry.SourceID != row.SourceID {
				t.Errorf("index[%d] = %+v, want the identity of row %q", i, entry, row.Path)
			}
		}
	})

	t.Run("a searched source without a match reports zero", func(t *testing.T) {
		t.Parallel()
		got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{
			"content": "exponential backoff",
		}))
		if hits, ok := got.Hits["local"]; !ok || hits != 0 {
			t.Errorf("hits = %v, want an explicit local=0", got.Hits)
		}
		if got.Hits["org"] != 1 {
			t.Errorf("hits = %v, want org=1", got.Hits)
		}
	})
}

// TestSearchDocuments_SlugIsAMatchField: a document named for the query is
// found even when its title and body spell the subject another way
// (search-matches-the-slug-and-folds-separators.adr).
func TestSearchDocuments_SlugIsAMatchField(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "acme-id-sdk.doc.md", "The Identity Client", "A server-side client.\n")
	writeFixtureDoc(t, localArch, "unrelated.doc.md", "Unrelated", "Nothing to see.\n")

	for _, match := range []string{"exact", "all", "any"} {
		t.Run(match, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "acme-id-sdk", "match": match}))
			if len(got) != 1 || !strings.HasSuffix(got[0].Path, "acme-id-sdk.doc.md") {
				t.Fatalf("results = %+v, want the document whose slug is the query", got)
			}
			if m := got[0].Matches[0]; m.Specificity != 3 || m.Excerpt != "acme-id-sdk" {
				t.Errorf("match = %+v, want specificity 3 with the slug as the excerpt", m)
			}
		})
	}
}

// TestSearchDocuments_SeparatorFoldingConverges: the spellings of one compound
// name reach one document set.
func TestSearchDocuments_SeparatorFoldingConverges(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "client.doc.md", "The Identity Client", "Install `@acme-id/sdk` first.\n")
	writeFixtureDoc(t, globalArch, "org-client.adr.md", "Shared Client", "We adopt acme-id-sdk for every web app.\n")
	writeFixtureDoc(t, localArch, "decoy.doc.md", "Decoy", "The acme team owns an identity sdk.\n")

	var want []string
	for _, query := range []string{"acme-id-sdk", "@acme-id/sdk", "acme-id sdk", "ACME_ID.SDK", "acme-id:sdk"} {
		var paths []string
		for _, r := range unmarshalSearch(t, callSearch(t, base, map[string]any{"content": query})) {
			paths = append(paths, r.Path)
		}
		slices.Sort(paths)
		if len(paths) != 2 {
			t.Errorf("query %q matched %v, want the two documents that name the package", query, paths)
		}
		if want == nil {
			want = paths
		} else if !slices.Equal(paths, want) {
			t.Errorf("query %q matched %v, want the same set as the first spelling %v", query, paths, want)
		}
	}
}

func TestSearchDocuments_ExactModeDoesNotFold(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "client.doc.md", "The Identity Client", "Install `@acme-id/sdk` first.\n")

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "acme-id-sdk", "match": "exact"}))
	if len(got) != 0 {
		t.Errorf("exact matched %+v, want a literal substring search that finds nothing", got)
	}
}

func TestSearchDocuments_SeparatorOnlyWords(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "billing.doc.md", "Billing", "Body.\n")

	t.Run("a query of separators alone is refused", func(t *testing.T) {
		t.Parallel()
		res := callSearch(t, base, map[string]any{"content": "-- // @"})
		if !res.IsError {
			t.Fatal("expected the at-least-one-word error")
		}
	})
	t.Run("a separator word beside a real word is dropped", func(t *testing.T) {
		t.Parallel()
		got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "-- billing"}))
		if len(got) != 1 {
			t.Errorf("got %d results, want the document matched on the remaining word", len(got))
		}
	})
}

func searchText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || result.IsError || len(result.Content) == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("unexpected content type %T", result.Content[0])
	}
	return tc.Text
}

// TestSearchDocuments_InvalidInputs: an unknown source scope and a wordless
// content query fail loudly instead of returning an empty page that reads as a
// verified absence; a whitespace-only exact query keeps the substring
// semantics and stays a non-error.
func TestSearchDocuments_InvalidInputs(t *testing.T) {
	t.Parallel()
	base := searchFixture(t)

	tests := []struct {
		name    string
		args    map[string]any
		wantErr string // empty: must NOT be an error result
	}{
		{"unknown source id", map[string]any{"content": "compatibility", "source": "orgg"}, `invalid source "orgg"`},
		{"whitespace content all", map[string]any{"content": "   "}, "content must contain at least one word"},
		{"whitespace content any", map[string]any{"content": " \t ", "match": "any"}, "content must contain at least one word"},
		{"whitespace content exact stays substring", map[string]any{"content": "   ", "match": "exact"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := callSearch(t, base, tt.args)
			if tt.wantErr == "" {
				if res.IsError {
					t.Fatalf("want a normal result, got an error: %+v", res.Content)
				}
				return
			}
			if !res.IsError {
				t.Fatal("want an error result")
			}
			tc, ok := mcp.AsTextContent(res.Content[0])
			if !ok {
				t.Fatalf("unexpected content type %T", res.Content[0])
			}
			if !strings.Contains(tc.Text, tt.wantErr) {
				t.Errorf("error = %q, want substring %q", tc.Text, tt.wantErr)
			}
		})
	}
}

// callSearch invokes the handler with args and returns the raw tool result.
func callSearch(t *testing.T, base string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := HandleSearchDocuments(StaticRoot(base))(t.Context(), reqWith(args))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return res
}
