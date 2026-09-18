package tools

// Properties of the search_documents envelope
// (read-tool-responses-survive-host-truncation.adr, search-documents.spec §11):
//
//   - the keys serialize as coverage, hits, truncated, index, results in every
//     mode, so a host preview shows the summary first;
//   - hits carries the key set of coverage under every source scope;
//   - index lists the admitted rows, and results is an ordered part of it;
//   - two identical calls return identical bytes;
//   - the tool description names every field the response can carry.

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/sync"
)

// topLevelKeys returns the keys of a JSON object in wire order, which
// json.Unmarshal into a struct or a map cannot show.
func topLevelKeys(t *testing.T, text string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(text))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("response does not open a JSON object: %v %v", tok, err)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		key, ok := tok.(string)
		if !ok {
			t.Fatalf("key token is %T, want a string", tok)
		}
		keys = append(keys, key)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			t.Fatal(err)
		}
	}
	return keys
}

func rawEnvelope(t *testing.T, text string) map[string]json.RawMessage {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

// envelopeFixture holds three sources: local and org match "needle", and team
// is searched without a match.
func envelopeFixture(t *testing.T) string {
	t.Helper()
	base, localArch, globalArchs := manySourceFixture(t, "org", "team")
	for i := range 3 {
		writeFixtureDoc(t, localArch, fmt.Sprintf("local-%d.rule.md", i), fmt.Sprintf("Local Needle %d", i),
			"see @src/payments/ for details\n")
	}
	writeFixtureDoc(t, localArch, "silent.doc.md", "Silent", "Nothing here.\n")
	for i := range 2 {
		writeFixtureDoc(t, globalArchs["org"], fmt.Sprintf("org-%d.adr.md", i), fmt.Sprintf("Org %d", i),
			"the needle sits in the body\n")
	}
	writeFixtureDoc(t, globalArchs["org"], "org-silent.doc.md", "Org Silent", "Nothing here.\n")
	writeFixtureDoc(t, globalArchs["team"], "team-silent.doc.md", "Team Silent", "Nothing here.\n")

	m := sync.NewManifest()
	m.AddRelation("local-0.rule.md", "local-1.rule.md", sync.RelRelated)
	m.AddRelation("local-2.rule.md", "local-0.rule.md", sync.RelImplements)
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestSearchDocuments_EnvelopeKeyOrder(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)
	want := []string{"coverage", "hits", "truncated", "index", "results"}

	tests := []struct {
		name string
		args map[string]any
	}{
		{"verified absence", map[string]any{"content": "zephyrite"}},
		{"metadata only", map[string]any{"types": []string{"rule"}}},
		{"snippets", map[string]any{"content": "needle"}},
		{"full", map[string]any{"content": "needle", "mode": "full"}},
		{"path_ref in one source", map[string]any{"path_ref": "src/payments/", "source": "local"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			text := searchText(t, callSearch(t, base, tt.args))
			if got := topLevelKeys(t, text); !slices.Equal(got, want) {
				t.Errorf("keys arrive as %v, want %v; a host preview must reach hits and index before any row", got, want)
			}
		})
	}
}

// TestSearchDocuments_VerifiedAbsenceCarriesEmptyArrays reads the raw text: a
// decoded nil slice and an empty one look the same. It fails against the
// current handler, which serializes "results":null because fitSearchPage
// returns a nil page for zero rows (search-documents.spec §11.1).
func TestSearchDocuments_VerifiedAbsenceCarriesEmptyArrays(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)

	tests := []struct {
		name string
		args map[string]any
	}{
		{"snippets", map[string]any{"content": "zephyrite"}},
		{"full", map[string]any{"content": "zephyrite", "mode": "full"}},
		{"metadata only", map[string]any{"types": []string{"prd"}}},
	}
	want := map[string]string{
		"coverage":  `{"local":4,"org":3,"team":1}`,
		"hits":      `{"local":0,"org":0,"team":0}`,
		"truncated": `false`,
		"index":     `[]`,
		"results":   `[]`,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := rawEnvelope(t, searchText(t, callSearch(t, base, tt.args)))
			for key, wantValue := range want {
				if got := string(raw[key]); got != wantValue {
					t.Errorf("%s = %s, want %s; a client that iterates the array fails on null, and a missing zero reads as a skipped source",
						key, got, wantValue)
				}
			}
		})
	}
}

func TestSearchDocuments_HitsFollowTheSourceScope(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)

	tests := []struct {
		name         string
		source       string
		wantCoverage map[string]int
		wantHits     map[string]int
	}{
		{"no scope", "", map[string]int{"local": 4, "org": 3, "team": 1}, map[string]int{"local": 3, "org": 2, "team": 0}},
		{"local", "local", map[string]int{"local": 4}, map[string]int{"local": 3}},
		{"global", "global", map[string]int{"org": 3, "team": 1}, map[string]int{"org": 2, "team": 0}},
		{"declared id with matches", "org", map[string]int{"org": 3}, map[string]int{"org": 2}},
		{"declared id without a match", "team", map[string]int{"team": 1}, map[string]int{"team": 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := map[string]any{"content": "needle", "limit": float64(1)}
			if tt.source != "" {
				args["source"] = tt.source
			}
			got := unmarshalSearchEnvelope(t, callSearch(t, base, args))
			if !maps.Equal(got.Coverage, tt.wantCoverage) {
				t.Errorf("coverage = %v, want %v", got.Coverage, tt.wantCoverage)
			}
			if !maps.Equal(got.Hits, tt.wantHits) {
				t.Errorf("hits = %v, want %v: the count before the limit cut, with zero for a searched source that held nothing",
					got.Hits, tt.wantHits)
			}
		})
	}
}

func indexOf(rows []searchResult) []searchIndexEntry {
	entries := make([]searchIndexEntry, len(rows))
	for i, row := range rows {
		entries[i] = searchIndexEntry{Path: row.Path, Title: row.Title, SourceID: row.SourceID}
	}
	return entries
}

func isOrderedPartOf(part, whole []searchIndexEntry) bool {
	next := 0
	for _, entry := range whole {
		if next < len(part) && part[next] == entry {
			next++
		}
	}
	return next == len(part)
}

func TestSearchDocuments_IndexMirrorsResults(t *testing.T) {
	t.Parallel()
	evidence := strings.Repeat("see @src/payments/stripe.go and src/payments/ in this long sentence of evidence. ", 12)

	t.Run("every row of an untruncated page, in order", func(t *testing.T) {
		t.Parallel()
		base := envelopeFixture(t)
		for _, mode := range []string{"snippets", "full"} {
			got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{"content": "needle", "mode": mode, "limit": float64(4)}))
			if got.Truncated || len(got.Results) != 4 {
				t.Fatalf("%s: page holds %d rows, truncated=%v; want 4 rows", mode, len(got.Results), got.Truncated)
			}
			if !slices.Equal(got.Index, indexOf(got.Results)) {
				t.Errorf("%s: index = %v, want the identity of each row in row order", mode, got.Index)
			}
		}
	})

	t.Run("a truncated page of one source is the head of the index", func(t *testing.T) {
		t.Parallel()
		base, localArch, _ := twoSourceFixture(t)
		for i := range 60 {
			writeFixtureDoc(t, localArch, fmt.Sprintf("cited-%02d.rule.md", i), fmt.Sprintf("Cited %02d", i), evidence)
		}
		got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{"path_ref": "src/payments/stripe.go", "limit": float64(60)}))
		if !got.Truncated || len(got.Index) != 60 || len(got.Results) == 0 {
			t.Fatalf("page holds %d rows of %d, truncated=%v; want a truncated page under an index of 60",
				len(got.Results), len(got.Index), got.Truncated)
		}
		if !slices.Equal(got.Index[:len(got.Results)], indexOf(got.Results)) {
			t.Error("results are not the first index entries; the agent cannot tell which admitted rows it still has to read")
		}
	})

	t.Run("a truncated page of two sources is an ordered part of the index", func(t *testing.T) {
		t.Parallel()
		base, localArch, globalArch := twoSourceFixture(t)
		for i := range 60 {
			writeFixtureDoc(t, localArch, fmt.Sprintf("cited-%02d.rule.md", i), fmt.Sprintf("Cited %02d", i), evidence)
		}
		writeFixtureDoc(t, globalArch, "org.doc.md", "Org Mention", "one mention of @src/ only")
		got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{"path_ref": "src/payments/stripe.go", "limit": float64(50)}))
		if !got.Truncated {
			t.Fatal("the fixture fits the budget, so it does not reach the cut")
		}
		if !isOrderedPartOf(indexOf(got.Results), got.Index) {
			t.Errorf("results %v are not an ordered part of the index", rowPaths(got.Results))
		}
		seen := map[searchIndexEntry]bool{}
		for _, entry := range got.Index {
			if seen[entry] {
				t.Errorf("index lists %q twice", entry.Path)
			}
			seen[entry] = true
		}
	})
}

func TestSearchDocuments_IdenticalCallsAreByteIdentical(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)

	tests := []struct {
		name string
		args map[string]any
	}{
		{"snippets by relevance", map[string]any{"content": "needle"}},
		{"snippets by mtime", map[string]any{"content": "needle", "sort": "mtime"}},
		{"full", map[string]any{"content": "needle", "mode": "full", "limit": float64(20)}},
		{"any with a folded word", map[string]any{"content": "src-payments zephyrite", "match": "any"}},
		{"exact", map[string]any{"content": "the needle", "match": "exact"}},
		{"path_ref with relations", map[string]any{"path_ref": "src/payments/"}},
		{"metadata only", map[string]any{"status": "accepted", "mode": "full"}},
		{"verified absence", map[string]any{"content": "zephyrite"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			first := searchText(t, callSearch(t, base, tt.args))
			for range 3 {
				if again := searchText(t, callSearch(t, base, tt.args)); again != first {
					t.Fatalf("an identical call returned different bytes:\n%s\n%s", first, again)
				}
			}
		})
	}
}

func TestSearchDocuments_EmptySortMeansRelevance(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)

	byDefault := searchText(t, callSearch(t, base, map[string]any{"content": "needle"}))
	empty := searchText(t, callSearch(t, base, map[string]any{"content": "needle", "sort": ""}))

	if empty != byDefault {
		t.Errorf("sort=\"\" returned\n%s\nwant the relevance order\n%s", empty, byDefault)
	}
}

// wireFieldNames walks a response type and returns the json name of every
// field it can serialize.
func wireFieldNames(root reflect.Type) []string {
	seen := map[reflect.Type]bool{}
	names := map[string]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		switch typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			walk(typ.Elem())
		case reflect.Struct:
			if seen[typ] || typ.PkgPath() != root.PkgPath() {
				return
			}
			seen[typ] = true
			for i := range typ.NumField() {
				field := typ.Field(i)
				name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				if !field.IsExported() || name == "" || name == "-" {
					continue
				}
				names[name] = true
				walk(field.Type)
			}
		default:
		}
	}
	walk(root)
	return slices.Sorted(maps.Keys(names))
}

// searchFieldsWithoutMention exempts a wire field from the description. The
// value is the reason.
var searchFieldsWithoutMention = map[string]string{
	"type":                     "row metadata that list_documents and get_document carry under the same name",
	"mtime":                    "row metadata that list_documents and get_document carry under the same name",
	"tags":                     "row metadata that list_documents and get_document carry under the same name",
	"read_only":                "restates source_kind, which the description tells the agent to read",
	"incoming_relations":       "named as a group: \"5 relations per direction\"",
	"outgoing_relations":       "named as a group: \"5 relations per direction\"",
	"incoming_relations_total": "named as a group: \"the relation totals\"",
	"outgoing_relations_total": "named as a group: \"the relation totals\"",
	"kind":                     "a part of one matches entry; the description names matches",
	"ref":                      "a part of one matches entry; the description names matches",
	"specificity":              "a part of one matches entry; the description names matches",
	"excerpt":                  "a part of one matches entry; the description names excerpts as the snippets output",
}

// TestNewSearchDocumentsTool_DescriptionNamesEveryWireField reads the field
// list from the response types, so a field added to the wire cannot ship
// without a mention or a recorded reason
// (registry-agreement-and-test-seams.guide).
func TestNewSearchDocumentsTool_DescriptionNamesEveryWireField(t *testing.T) {
	t.Parallel()
	description := NewSearchDocumentsTool().Description
	fields := wireFieldNames(reflect.TypeFor[searchDocumentsResult]())
	if len(fields) == 0 || description == "" {
		t.Fatalf("read %d wire fields and a description of %d bytes, so this test proves nothing", len(fields), len(description))
	}
	mentions := func(name string) bool {
		return regexp.MustCompile(`(^|[^a-z_])` + regexp.QuoteMeta(name) + `([^a-z_]|$)`).MatchString(description)
	}

	for _, name := range fields {
		_, exempt := searchFieldsWithoutMention[name]
		switch {
		case !exempt && !mentions(name):
			t.Errorf("the description does not mention %q; an agent meets the field with no contract for it", name)
		case exempt && mentions(name):
			t.Errorf("the description mentions %q, so its exemption is stale", name)
		}
	}
	for name := range searchFieldsWithoutMention {
		if !slices.Contains(fields, name) {
			t.Errorf("exemption %q names no wire field", name)
		}
	}
	envelope := reflect.TypeFor[searchDocumentsResult]()
	for i := range envelope.NumField() {
		name, _, _ := strings.Cut(envelope.Field(i).Tag.Get("json"), ",")
		if reason, exempt := searchFieldsWithoutMention[name]; exempt {
			t.Errorf("envelope key %q is exempt (%s); the summary keys are what a truncated response leaves the agent", name, reason)
		}
	}
}

// TestSearchDocuments_ResponseKeysAreKnownWireFields guards the reflection
// walk above: a key on the wire that the walk cannot see escapes the
// description check.
func TestSearchDocuments_ResponseKeysAreKnownWireFields(t *testing.T) {
	t.Parallel()
	base := envelopeFixture(t)
	fields := wireFieldNames(reflect.TypeFor[searchDocumentsResult]())

	for _, args := range []map[string]any{
		{"content": "needle", "mode": "full", "limit": float64(20)},
		{"path_ref": "src/payments/"},
	} {
		raw := rawEnvelope(t, searchText(t, callSearch(t, base, args)))
		keys := map[string]bool{}
		for key, value := range raw {
			keys[key] = true
			if key == "index" || key == "results" {
				var rows any
				if err := json.Unmarshal(value, &rows); err != nil {
					t.Fatal(err)
				}
				collectObjectKeys(rows, keys)
			}
		}
		if !keys["outgoing_relations"] || !keys["matches"] {
			t.Fatalf("response keys %v lack the row fields, so this test proves nothing", slices.Sorted(maps.Keys(keys)))
		}
		for key := range keys {
			if !slices.Contains(fields, key) {
				t.Errorf("response carries key %q, which the field walk did not find", key)
			}
		}
	}
}

func collectObjectKeys(value any, into map[string]bool) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectObjectKeys(item, into)
		}
	case map[string]any:
		for key, item := range typed {
			into[key] = true
			collectObjectKeys(item, into)
		}
	}
}
