package tools

// Tests for the evidence caps of a search_documents row
// (read-tool-responses-survive-host-truncation.adr, search-documents.spec
// §5.6-§5.9, §6.6, §10).

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/sync"
)

// rawRows decodes the rows without the response types, so an omitted key and a
// zero value stay apart.
func rawRows(t *testing.T, text string) []map[string]json.RawMessage {
	t.Helper()
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(rawEnvelope(t, text)["results"], &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func matchRefs(matches []searchMatch) []string {
	refs := make([]string, len(matches))
	for i, m := range matches {
		refs[i] = m.Ref
	}
	return refs
}

func TestHandleSearchDocuments_MatchCapBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mentions    int
		wantMatches int
		wantTotal   string
	}{
		{"one under the cap", 4, 4, ""},
		{"at the cap", 5, 5, ""},
		{"one over the cap", 6, 5, "6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, localArch, _ := twoSourceFixture(t)
			writeFixtureDoc(t, localArch, "cited.doc.md", "Cited", strings.Repeat("see @src/payments/ again. ", tt.mentions))

			res := callSearch(t, base, map[string]any{"path_ref": "src/payments/"})
			rows := rawRows(t, searchText(t, res))
			got := unmarshalSearch(t, res)
			if len(rows) != 1 || len(got) != 1 {
				t.Fatalf("got %d rows, want 1", len(got))
			}
			if len(got[0].Matches) != tt.wantMatches {
				t.Errorf("row carries %d matches, want %d", len(got[0].Matches), tt.wantMatches)
			}
			if total := string(rows[0]["matches_total"]); total != tt.wantTotal {
				t.Errorf("matches_total = %q, want %q: the total appears only beside a cut array", total, tt.wantTotal)
			}
		})
	}
}

func TestHandleSearchDocuments_BothFiltersCapIndependently(t *testing.T) {
	t.Parallel()
	words := []string{"alpha", "bravo", "gamma", "delta", "echo", "foxtrot", "golf"}
	tests := []struct {
		name            string
		mentions        int
		queryWords      int
		wantPathMatches int
		wantWordMatches int
		wantTotal       int
	}{
		{"both groups over the cap", 8, 7, 5, 5, 15},
		{"only path_ref over the cap", 8, 2, 5, 2, 10},
		{"only content over the cap", 3, 7, 3, 5, 10},
		{"neither group over the cap", 3, 2, 3, 2, 0},
		{"both groups at the cap", 5, 5, 5, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, localArch, _ := twoSourceFixture(t)
			writeFixtureDoc(t, localArch, "cited.doc.md", "Cited",
				strings.Join(words, " ")+". "+strings.Repeat("see @src/payments/ again. ", tt.mentions))

			got := unmarshalSearch(t, callSearch(t, base, map[string]any{
				"path_ref": "src/payments/", "content": strings.Join(words[:tt.queryWords], " "),
			}))
			if len(got) != 1 {
				t.Fatalf("got %d rows, want 1", len(got))
			}
			pathMatches, wordMatches := 0, 0
			for _, m := range got[0].Matches {
				if m.Kind == matchKindContent {
					wordMatches++
				} else {
					pathMatches++
				}
			}
			if pathMatches != tt.wantPathMatches || wordMatches != tt.wantWordMatches {
				t.Errorf("row carries %d path_ref and %d content matches, want %d and %d: one filter must not use the other's cap",
					pathMatches, wordMatches, tt.wantPathMatches, tt.wantWordMatches)
			}
			if got[0].MatchesTotal != tt.wantTotal {
				t.Errorf("matches_total = %d, want %d: the hits of both filters", got[0].MatchesTotal, tt.wantTotal)
			}
			if first := got[0].Matches[0].Kind; first == matchKindContent {
				t.Errorf("matches[0] is %q, want the path_ref evidence ahead of the content evidence", first)
			}
		})
	}
}

// TestHandleSearchDocuments_ContentCapKeepsOrder pins search-documents.spec
// §6.6: query-word order at or under the cap, specificity order over it.
func TestHandleSearchDocuments_ContentCapKeepsOrder(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "words.doc.md", "Alpha Bravo",
		"## Gamma Delta\n\nThe words echo, foxtrot and golf sit in the prose.\n")

	tests := []struct {
		name      string
		query     string
		wantRefs  []string
		wantSpecs []int
		wantTotal int
	}{
		{"at the cap keeps query-word order", "echo gamma alpha foxtrot delta",
			[]string{"echo", "gamma", "alpha", "foxtrot", "delta"}, []int{1, 2, 3, 1, 2}, 0},
		{"over the cap keeps the most specific words", "echo gamma alpha foxtrot delta bravo golf",
			[]string{"alpha", "bravo", "gamma", "delta", "echo"}, []int{3, 3, 2, 2, 1}, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": tt.query}))
			if len(got) != 1 {
				t.Fatalf("got %d rows, want 1", len(got))
			}
			if refs := matchRefs(got[0].Matches); !slices.Equal(refs, tt.wantRefs) {
				t.Errorf("matches = %v, want %v", refs, tt.wantRefs)
			}
			for i, m := range got[0].Matches {
				if i < len(tt.wantSpecs) && m.Specificity != tt.wantSpecs[i] {
					t.Errorf("matches[%d] %q has specificity %d, want %d", i, m.Ref, m.Specificity, tt.wantSpecs[i])
				}
			}
			if got[0].MatchesTotal != tt.wantTotal {
				t.Errorf("matches_total = %d, want %d", got[0].MatchesTotal, tt.wantTotal)
			}
		})
	}
}

// TestHandleSearchDocuments_ContentCapDoesNotChangeTheScore: the wide document
// wins on its sixth and seventh title words, which the cap removes from the
// evidence. A score read from the capped evidence ties, and the rule type then
// ranks first.
func TestHandleSearchDocuments_ContentCapDoesNotChangeTheScore(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "wide.doc.md", "Alpha Bravo Gamma Delta Echo Foxtrot Golf", "Body.\n")
	writeFixtureDoc(t, localArch, "narrow.rule.md", "Alpha Bravo Gamma Delta Echo Foxtrot", "The golf golf.\n")

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "alpha bravo gamma delta echo foxtrot golf"}))
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if !strings.HasSuffix(got[0].Path, "wide.doc.md") {
		t.Errorf("first row is %q, want the document with seven title hits; the cap changed the ranking", got[0].Path)
	}
	for _, row := range got {
		if len(row.Matches) != searchMatchCap || row.MatchesTotal != 7 {
			t.Errorf("%s carries %d matches and matches_total %d, want %d and 7", row.Path, len(row.Matches), row.MatchesTotal, searchMatchCap)
		}
	}
}

// TestRankPathRefs_TieBreaking pins search-documents.spec §5.6. The first
// mention sits ahead of the explicit reference in the body, so body order alone
// would rank it first.
func TestRankPathRefs_TieBreaking(t *testing.T) {
	t.Parallel()
	body := "src/payments/ first, then @src/payments/ explicit, then src/payments/ again, " +
		"@src/ least, and @src/payments/stripe.go last, plus docs/guide.md elsewhere."

	hits := rankPathRefs(body, "@src/payments/stripe.go")

	type rankedRef struct {
		raw         string
		kind        string
		specificity int
	}
	want := []rankedRef{
		{"@src/payments/stripe.go", refKindExplicit, 3},
		{"@src/payments/", refKindExplicit, 2},
		{"src/payments/", refKindMention, 2},
		{"src/payments/", refKindMention, 2},
		{"@src/", refKindExplicit, 1},
	}
	got := make([]rankedRef, len(hits))
	for i, h := range hits {
		got[i] = rankedRef{h.Raw, h.Kind, h.specificity}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ranked refs = %v, want %v; the cap would keep weaker evidence", got, want)
	}
	if hits[2].Start >= hits[3].Start {
		t.Errorf("equal mentions arrive at offsets %d then %d, want body order", hits[2].Start, hits[3].Start)
	}
	if again := rankPathRefs(body, "@src/payments/stripe.go"); !slices.EqualFunc(hits, again, func(a, b pathRefHit) bool { return a == b }) {
		t.Error("two calls ranked the same body differently")
	}
}

// TestRankPathRefs_OrdersALargeBody checks the whole order on more hits than
// the stable sort handles in one insertion block, where the comparison runs in
// both directions.
func TestRankPathRefs_OrdersALargeBody(t *testing.T) {
	t.Parallel()
	spellings := []string{"src/payments/", "@src/", "@src/payments/stripe.go", "src/payments/stripe.go", "@src/payments/", "docs/other/"}
	var body strings.Builder
	related := 0
	for i := range 60 {
		spelling := spellings[(i*7+i/3)%len(spellings)]
		if strings.Contains(spelling, "src/") {
			related++
		}
		fmt.Fprintf(&body, "line %d cites %s here.\n", i, spelling)
	}

	hits := rankPathRefs(body.String(), "src/payments/stripe.go")

	if len(hits) != related || related < 40 {
		t.Fatalf("got %d hits, want the %d references that share a segment with the filter", len(hits), related)
	}
	for i := 1; i < len(hits); i++ {
		prev, next := hits[i-1], hits[i]
		switch {
		case prev.specificity != next.specificity:
			if prev.specificity < next.specificity {
				t.Errorf("hits[%d] has specificity %d after %d", i, next.specificity, prev.specificity)
			}
		case prev.Kind != next.Kind:
			if next.Kind == refKindExplicit {
				t.Errorf("hits[%d] is an explicit reference after a mention of equal specificity", i)
			}
		case prev.Start >= next.Start:
			t.Errorf("hits[%d] at offset %d follows offset %d, want body order among equals", i, next.Start, prev.Start)
		}
	}
}

func relationFixture(t *testing.T, relations int) (base string) {
	t.Helper()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "hub.rule.md", "Hub", "use @src/payments/ always\n")
	m := sync.NewManifest()
	for i := range relations {
		name := fmt.Sprintf("spoke-%d.guide.md", i)
		writeFixtureDoc(t, localArch, name, fmt.Sprintf("Spoke %d", i), "how-to content\n")
		m.AddRelation("hub.rule.md", name, sync.RelRelated)
		m.AddRelation(name, "hub.rule.md", sync.RelImplements)
	}
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestHandleSearchDocuments_RelationCapBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		relations     int
		wantRelations int
		wantTotal     string
	}{
		{"at the cap", 5, 5, ""},
		{"one over the cap", 6, 5, "6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := relationFixture(t, tt.relations)

			rows := rawRows(t, searchText(t, callSearch(t, base, map[string]any{"path_ref": "src/payments/"})))
			if len(rows) != 1 {
				t.Fatalf("got %d rows, want the hub alone", len(rows))
			}
			for _, direction := range []string{"incoming_relations", "outgoing_relations"} {
				var relations []DocumentRelation
				if err := json.Unmarshal(rows[0][direction], &relations); err != nil {
					t.Fatal(err)
				}
				if len(relations) != tt.wantRelations {
					t.Errorf("%s holds %d entries, want %d", direction, len(relations), tt.wantRelations)
				}
				if total := string(rows[0][direction+"_total"]); total != tt.wantTotal {
					t.Errorf("%s_total = %q, want %q: the total appears only beside a cut array", direction, total, tt.wantTotal)
				}
			}
		})
	}
}

// TestHandleSearchDocuments_RelationsSortByPathThenType writes the manifest
// against the wire order, so an unsorted array or a sort on the path alone
// shows.
func TestHandleSearchDocuments_RelationsSortByPathThenType(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "hub.rule.md", "Hub", "use @src/payments/ always\n")
	writeFixtureDoc(t, localArch, "a.guide.md", "Guide A", "how-to content\n")
	writeFixtureDoc(t, localArch, "b.guide.md", "Guide B", "how-to content\n")
	m := sync.NewManifest()
	for _, name := range []string{"b.guide.md", "a.guide.md"} {
		for _, relType := range []sync.RelationType{sync.RelRelated, sync.RelImplements, sync.RelExtends} {
			m.AddRelation("hub.rule.md", name, relType)
			m.AddRelation(name, "hub.rule.md", relType)
		}
	}
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"path_ref": "src/payments/"}))
	if len(got) != 1 {
		t.Fatalf("got %d rows, want the hub alone", len(got))
	}
	want := []DocumentRelation{
		{Path: ".archcore/a.guide.md", Type: "extends"},
		{Path: ".archcore/a.guide.md", Type: "implements"},
		{Path: ".archcore/a.guide.md", Type: "related"},
		{Path: ".archcore/b.guide.md", Type: "extends"},
		{Path: ".archcore/b.guide.md", Type: "implements"},
	}
	for direction, relations := range map[string][]DocumentRelation{
		"incoming": got[0].IncomingRelations, "outgoing": got[0].OutgoingRelations,
	} {
		if !slices.Equal(relations, want) {
			t.Errorf("%s = %v, want %v; the cap must keep the same five edges on every call", direction, relations, want)
		}
	}
	if got[0].IncomingRelationsTotal != 6 || got[0].OutgoingRelationsTotal != 6 {
		t.Errorf("totals = %d incoming, %d outgoing, want 6 and 6", got[0].IncomingRelationsTotal, got[0].OutgoingRelationsTotal)
	}
}

func TestHandleSearchDocuments_GlobalRowKeepsEmptyRelationArrays(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "local.rule.md", "Local", "use @src/payments/ always\n")
	writeFixtureDoc(t, globalArch, "org.rule.md", "Org", "use @src/payments/ always\n")

	for _, mode := range []string{"snippets", "full"} {
		rows := rawRows(t, searchText(t, callSearch(t, base, map[string]any{"path_ref": "src/payments/", "mode": mode})))
		if len(rows) != 2 {
			t.Fatalf("%s: got %d rows, want one per source", mode, len(rows))
		}
		for _, row := range rows {
			for _, direction := range []string{"incoming_relations", "outgoing_relations"} {
				if value := string(row[direction]); value != "[]" {
					t.Errorf("%s: %s of %s = %s, want []; a client that iterates the array fails on null",
						mode, direction, row["path"], value)
				}
				if _, present := row[direction+"_total"]; present {
					t.Errorf("%s: %s carries %s_total with nothing cut", mode, row["path"], direction)
				}
			}
		}
	}
}
