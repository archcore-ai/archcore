package tools

// Properties of near misses on an empty all-words response
// (search-documents.spec §13, empty-all-words-search-partial-matches.rfc):
//
//   - an all-words query of two or more distinct words that matches nothing
//     names the documents holding at least half of the words, each with the
//     words it lacks in query order;
//   - repeated words count once;
//   - the field is [] when the trigger holds and nothing qualifies, and absent
//     for one word, for any and exact, and beside a non-empty results;
//   - the filters bind a near miss as they bind a result;
//   - more matched words rank first, the cap holds whatever limit says, and a
//     source with a near miss keeps one row;
//   - equal word counts fall back to the relevance keys whatever sort says, and
//     the page is re-sorted after the source swaps;
//   - a row carries path, title, source_id, and missing, and nothing counts it
//     toward hits or index;
//   - near misses leave from the tail when the response would exceed the byte
//     budget, and truncated then reads true.

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// nearMissFixture holds the buried decision of the integration-bench task
// review-seam-buried: the ADR lacks "source" and any path reference.
func nearMissFixture(t *testing.T) string {
	t.Helper()
	base, localArch, _ := manySourceFixture(t)
	writeFixtureDoc(t, localArch, "delivery-policy-seam.adr.md",
		"Keep Delivery Fees Behind the DeliveryPolicy Interface",
		"## Context\n\nPartners plug a fee rule into the delivery module under the partner contract.\n\n"+
			"## Consequences\n\nPrice lookup is not covered by this decision.\n")
	writeFixtureDoc(t, localArch, "quote-fields.doc.md", "Quote Dictionary Fields",
		"The delivery fee is in cents.\n")
	writeFixtureDoc(t, localArch, "no-network.rule.md", "Application Code Makes No Network Calls",
		"## Rule\n\nNo sockets.\n")
	return base
}

// nearMisses returns the decoded near_misses and whether the key is present.
func nearMisses(t *testing.T, base string, args map[string]any) ([]searchNearMiss, bool) {
	t.Helper()
	raw := rawEnvelope(t, searchText(t, callSearch(t, base, args)))
	value, present := raw["near_misses"]
	if !present {
		return nil, false
	}
	var rows []searchNearMiss
	if err := json.Unmarshal(value, &rows); err != nil {
		t.Fatal(err)
	}
	return rows, true
}

func TestSearchDocuments_NearMissesNameTheMissingWords(t *testing.T) {
	t.Parallel()
	base := nearMissFixture(t)
	adr := filepath.ToSlash(filepath.Join(".archcore", "delivery-policy-seam.adr.md"))

	tests := []struct {
		name        string
		content     string
		wantMissing []string
	}{
		{"one word from a second topic", "delivery policy price source", []string{"source"}},
		{"eight words, six held", "delivery policy price source partner contract interface removal", []string{"source", "removal"}},
		{"two languages", "delivery policy доставка политика", []string{"доставка", "политика"}},
		{"repeated words count once", "source source delivery", []string{"source"}},
		{"a compound word keeps its parts", "partner source-policy", []string{"source policy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rows, present := nearMisses(t, base, map[string]any{"content": tt.content})
			if !present {
				t.Fatal("near_misses is absent from an empty all-words response of two or more words")
			}
			if len(rows) == 0 || rows[0].Path != adr {
				t.Fatalf("near_misses = %+v, want the ADR first", rows)
			}
			if !slices.Equal(rows[0].Missing, tt.wantMissing) {
				t.Errorf("missing = %q, want %q", rows[0].Missing, tt.wantMissing)
			}
			if rows[0].Title == "" || rows[0].SourceID != "local" {
				t.Errorf("row = %+v, want its title and source_id", rows[0])
			}
			for _, row := range rows[1:] {
				if row.Path == filepath.ToSlash(filepath.Join(".archcore", "no-network.rule.md")) {
					t.Errorf("a document holding none of the words is a near miss: %+v", row)
				}
			}
		})
	}
}

func TestSearchDocuments_NearMissesBelowHalfAreDropped(t *testing.T) {
	t.Parallel()
	base := nearMissFixture(t)
	quote := filepath.ToSlash(filepath.Join(".archcore", "quote-fields.doc.md"))

	rows, present := nearMisses(t, base, map[string]any{"content": "delivery policy price source partner contract interface removal"})
	if !present {
		t.Fatal("near_misses is absent")
	}
	for _, row := range rows {
		if row.Path == quote {
			t.Errorf("a document holding 1 of 8 words is a near miss: %+v", row)
		}
	}
}

func TestSearchDocuments_NearMissesPresence(t *testing.T) {
	t.Parallel()
	base := nearMissFixture(t)

	tests := []struct {
		name        string
		args        map[string]any
		wantPresent bool
	}{
		{"nothing qualifies", map[string]any{"content": "zephyrite quuxite"}, true},
		{"filters exclude the only candidate", map[string]any{"content": "delivery source", "types": []string{"rule"}}, true},
		{"path_ref excludes the only candidate", map[string]any{"content": "delivery source", "path_ref": "src/other/"}, true},
		{"one word", map[string]any{"content": "zephyrite"}, false},
		{"one distinct word", map[string]any{"content": "source source"}, false},
		{"any", map[string]any{"content": "zephyrite quuxite", "match": "any"}, false},
		{"exact", map[string]any{"content": "delivery source", "match": "exact"}, false},
		{"results not empty", map[string]any{"content": "delivery fee"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rows, present := nearMisses(t, base, tt.args)
			if present != tt.wantPresent {
				t.Fatalf("near_misses present = %v, want %v", present, tt.wantPresent)
			}
			if present && len(rows) != 0 {
				t.Errorf("near_misses = %+v, want []", rows)
			}
		})
	}
}

func TestSearchDocuments_NearMissesRankCapAndRepresentation(t *testing.T) {
	t.Parallel()
	base, localArch, globalArchs := manySourceFixture(t, "org")
	writeFixtureDoc(t, localArch, "a.adr.md", "A", "alpha beta gamma\n")
	writeFixtureDoc(t, localArch, "b.adr.md", "B", "alpha beta\n")
	writeFixtureDoc(t, localArch, "c.adr.md", "C", "alpha gamma\n")
	writeFixtureDoc(t, localArch, "d.adr.md", "D", "beta gamma\n")
	writeFixtureDoc(t, localArch, "e.adr.md", "E", "alpha\n")
	writeFixtureDoc(t, globalArchs["org"], "g.adr.md", "G", "alpha delta\n")
	now := time.Now().Truncate(time.Second)
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		setMtime(t, base, filepath.Join(".archcore", name+".adr.md"), now)
	}

	rows, present := nearMisses(t, base, map[string]any{"content": "alpha beta gamma delta", "limit": float64(1)})
	if !present {
		t.Fatal("near_misses is absent")
	}
	if len(rows) != 3 {
		t.Fatalf("got %d near misses with limit 1, want the cap of 3 (search-documents.spec §13.4): %+v", len(rows), rows)
	}
	if rows[0].Title != "A" || !slices.Equal(rows[0].Missing, []string{"delta"}) {
		t.Errorf("first row = %+v, want A, the document holding three words", rows[0])
	}
	if rows[1].Title != "B" {
		t.Errorf("second row = %+v, want B: equal word counts fall back to the relevance keys, path last", rows[1])
	}
	if rows[2].SourceID != "org" {
		t.Errorf("rows = %+v, want the org near miss swapped in over the lowest local row", rows)
	}
}

func TestSearchDocuments_NearMissRowShape(t *testing.T) {
	t.Parallel()
	base := nearMissFixture(t)

	for _, mode := range []string{"snippets", "full"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			text := searchText(t, callSearch(t, base, map[string]any{"content": "delivery policy price source", "mode": mode}))
			raw := rawEnvelope(t, text)
			var rows []map[string]json.RawMessage
			if err := json.Unmarshal(raw["near_misses"], &rows); err != nil || len(rows) == 0 {
				t.Fatalf("near_misses = %s (%v), want at least one row", raw["near_misses"], err)
			}
			for _, row := range rows {
				var keys []string
				for key := range row {
					keys = append(keys, key)
				}
				slices.Sort(keys)
				if want := []string{"missing", "path", "source_id", "title"}; !slices.Equal(keys, want) {
					t.Errorf("row keys = %v, want exactly %v", keys, want)
				}
			}
			if got := string(raw["hits"]) + string(raw["index"]) + string(raw["truncated"]); got != `{"local":0}[]false` {
				t.Errorf("hits, index, truncated = %s, want a near miss counted in none of them", got)
			}
			want := []string{"coverage", "hits", "truncated", "index", "near_misses", "results"}
			if got := topLevelKeys(t, text); !slices.Equal(got, want) {
				t.Errorf("keys arrive as %v, want %v", got, want)
			}
		})
	}
}

func TestSearchDocuments_NearMissesRankByScoreWhateverSort(t *testing.T) {
	t.Parallel()
	base, localArch, _ := manySourceFixture(t)
	writeFixtureDoc(t, localArch, "a.adr.md", "A", "alpha\n")
	writeFixtureDoc(t, localArch, "z.adr.md", "Alpha Zeta", "Nothing else.\n")
	now := time.Now().Truncate(time.Second)
	setMtime(t, base, filepath.Join(".archcore", "a.adr.md"), now)
	setMtime(t, base, filepath.Join(".archcore", "z.adr.md"), now.Add(-time.Hour))

	for _, sort := range []string{"relevance", "mtime"} {
		t.Run(sort, func(t *testing.T) {
			t.Parallel()
			rows, _ := nearMisses(t, base, map[string]any{"content": "alpha beta", "sort": sort})
			if len(rows) != 2 || rows[0].Title != "Alpha Zeta" {
				t.Errorf("near_misses = %+v, want the title hit first: the score outranks mtime and path, and sort does not apply", rows)
			}
		})
	}
}

func TestSearchDocuments_NearMissesReSortAfterTwoSwaps(t *testing.T) {
	t.Parallel()
	base, localArch, globalArchs := manySourceFixture(t, "org", "team")
	now := time.Now().Truncate(time.Second)
	for _, name := range []string{"l1", "l2", "l3"} {
		writeFixtureDoc(t, localArch, name+".adr.md", strings.ToUpper(name), "alpha beta gamma\n")
		setMtime(t, base, filepath.Join(".archcore", name+".adr.md"), now)
	}
	writeFixtureDoc(t, globalArchs["org"], "g1.adr.md", "Alpha", "beta\n")
	writeFixtureDoc(t, globalArchs["team"], "g2.adr.md", "G2", "alpha beta\n")

	rows, _ := nearMisses(t, base, map[string]any{"content": "alpha beta gamma delta"})
	var sources []string
	for _, row := range rows {
		sources = append(sources, row.SourceID)
	}
	if want := []string{"local", "org", "team"}; !slices.Equal(sources, want) || rows[0].Title != "L1" {
		t.Errorf("near_misses = %+v, want L1, then the org title hit, then team: each missing source swaps in, then the page is re-sorted", rows)
	}
}

func TestSearchDocuments_NearMissesFitTheByteBudget(t *testing.T) {
	t.Parallel()
	base, localArch, _ := manySourceFixture(t)
	for _, name := range []string{"a", "b", "c"} {
		writeFixtureDoc(t, localArch, name+".adr.md", strings.ToUpper(name), "alpha\n")
	}
	long := strings.Repeat("q", 25_000)

	text := searchText(t, callSearch(t, base, map[string]any{"content": "alpha " + long}))
	raw := rawEnvelope(t, text)
	var rows []searchNearMiss
	if err := json.Unmarshal(raw["near_misses"], &rows); err != nil {
		t.Fatal(err)
	}
	if len(text) > searchResponseByteBudget {
		t.Errorf("response is %d bytes, over the %d-byte budget", len(text), searchResponseByteBudget)
	}
	if len(rows) != 1 || string(raw["truncated"]) != "true" {
		t.Errorf("got %d near misses and truncated=%s, want the tail cut to one row and truncated=true", len(rows), raw["truncated"])
	}
}

func TestSearchDocuments_NearMissesFillTheByteBudgetExactly(t *testing.T) {
	t.Parallel()
	base, localArch, _ := manySourceFixture(t)
	writeFixtureDoc(t, localArch, "a.adr.md", "A", "alpha\n")
	query := func(wordLen int) map[string]any {
		return map[string]any{"content": "alpha " + strings.Repeat("q", wordLen)}
	}
	probe := len(searchText(t, callSearch(t, base, query(1))))

	text := searchText(t, callSearch(t, base, query(1+searchResponseByteBudget-probe)))
	raw := rawEnvelope(t, text)
	var rows []searchNearMiss
	if err := json.Unmarshal(raw["near_misses"], &rows); err != nil {
		t.Fatal(err)
	}
	if len(text) != searchResponseByteBudget {
		t.Fatalf("response is %d bytes, want exactly the %d-byte budget", len(text), searchResponseByteBudget)
	}
	if len(rows) != 1 || string(raw["truncated"]) != "false" {
		t.Errorf("got %d near misses and truncated=%s, want the row kept at the exact budget", len(rows), raw["truncated"])
	}
}
