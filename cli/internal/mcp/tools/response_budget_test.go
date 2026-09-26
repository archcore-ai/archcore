package tools

// Tests for the search_documents byte budget
// (read-tool-responses-survive-host-truncation.adr): the response fits the
// budget in every mode, a body is shortened before a row is dropped, and the
// size arithmetic agrees with encoding/json.

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"archcore-cli/internal/sync"
)

func TestJSONStringLen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii", "plain words"},
		{"quote and backslash", `say "hi" \ bye`},
		{"newline tab carriage return", "a\nb\tc\rd"},
		{"backspace and form feed", "a\bb\fc"},
		{"other control byte", "a\x01b"},
		{"html characters", "<tag> & more"},
		{"two-byte runes", "Χάρτης εξουσιοδότησης"},
		{"line separators", "a\u2028b\u2029c"},
		{"invalid utf-8", "a\xffb"},
		{"lone continuation byte", "a\x80b"},
		{"encoded replacement character", "a\uFFFDb"},
		{"emoji", "ok 👍"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, want := jsonStringLen(tt.in), marshaledStringLen(t, tt.in); got != want {
				t.Errorf("jsonStringLen(%q) = %d, want %d as encoding/json writes it", tt.in, got, want)
			}
		})
	}
}

func FuzzJSONStringLen(f *testing.F) {
	for _, seed := range []string{"", "plain", "\"\\\n\t", "<&>", "Ελληνικά", "\u2028", "\xff\xfe", "a\x00b"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := jsonStringLen(s), marshaledStringLen(t, s); got != want {
			t.Errorf("jsonStringLen(%q) = %d, want %d", s, got, want)
		}
	})
}

func marshaledStringLen(t *testing.T, s string) int {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return len(data) - len(`""`)
}

func TestCapIndex(t *testing.T) {
	t.Parallel()
	exact := make([]searchIndexEntry, 10)
	used := 0
	for i := range exact {
		exact[i] = searchIndexEntry{Path: fmt.Sprintf(".archcore/row-%d.doc.md", i), Title: strings.Repeat("t", 1000), SourceID: "local"}
		used += indexEntryBytes(t, exact[i])
	}
	exact[len(exact)-1].Title += strings.Repeat("t", searchIndexByteBudget-used)
	over := slices.Clone(exact)
	over[len(over)-1].Title += "t"
	oversized := []searchIndexEntry{
		{Path: ".archcore/big.doc.md", Title: strings.Repeat("t", searchIndexByteBudget), SourceID: "local"},
		exact[0],
	}

	tests := []struct {
		name     string
		index    []searchIndexEntry
		wantKept int
		wantCut  bool
	}{
		{"fits to the byte", exact, 10, false},
		{"one byte over drops the last entry", over, 9, true},
		{"an oversized first entry stays alone", oversized, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kept, cut, err := capIndex(tt.index)
			if err != nil {
				t.Fatal(err)
			}
			if len(kept) != tt.wantKept || cut != tt.wantCut {
				t.Errorf("capIndex kept %d entries, cut=%v; want %d, cut=%v", len(kept), cut, tt.wantKept, tt.wantCut)
			}
		})
	}
}

func indexEntryBytes(t *testing.T, entry searchIndexEntry) int {
	t.Helper()
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return len(data) + len(",")
}

func TestCutForJSONBudget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     string
		budget int
		want   string
	}{
		{"fits whole", "short", 100, "short"},
		{"cuts between runes, never inside one", "ΩΩΩΩ", 5, "ΩΩ"},
		{"counts escapes, not source bytes", "a\n\n\nb", 4, "a\n"},
		{"prefers the last newline in the second half", "first line\nsecond line\nthird", 25, "first line\nsecond line\n"},
		{"ignores a newline in the first half", "a\n" + strings.Repeat("x", 40), 20, "a\\n" + strings.Repeat("x", 17)},
		{"zero budget", "abc", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			want := strings.ReplaceAll(tt.want, "\\n", "\n")
			got := tt.in[:cutForJSONBudget(tt.in, tt.budget)]
			if got != want {
				t.Errorf("cut = %q, want %q", got, want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("cut %q is not valid UTF-8", got)
			}
			if jsonStringLen(got) > tt.budget {
				t.Errorf("cut %q encodes to %d bytes, over the budget %d", got, jsonStringLen(got), tt.budget)
			}
		})
	}
}

// multiByteBody returns a body of about size bytes that holds needle once. Two
// bytes per character is the shape of the recorded incident.
func multiByteBody(needle string, size int) string {
	line := "Γραμμή εγγράφου χωρίς αντιστοιχίες, μόνο όγκος.\n"
	return needle + "\n\n" + strings.Repeat(line, size/len(line))
}

func TestHandleSearchDocuments_ResponseStaysUnderByteBudget(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	for i := range 24 {
		writeFixtureDoc(t, localArch, fmt.Sprintf("big-%02d.doc.md", i),
			fmt.Sprintf("Big Local %02d", i), multiByteBody("needle", 30_000))
	}
	for i := range 6 {
		writeFixtureDoc(t, globalArch, fmt.Sprintf("org-%02d.doc.md", i),
			fmt.Sprintf("Org %02d", i), multiByteBody("needle", 20_000))
	}
	m := sync.NewManifest()
	for i := range 200 {
		name := fmt.Sprintf("cited-%03d.guide.md", i)
		writeFixtureDoc(t, localArch, name, fmt.Sprintf("Cited %03d", i),
			strings.Repeat("see @src/payments/stripe.go and src/payments/ here. ", 12))
		for j := range 8 {
			m.AddRelation(name, fmt.Sprintf("big-%02d.doc.md", j), sync.RelRelated)
		}
	}
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args map[string]any
	}{
		{"full mode default limit", map[string]any{"content": "needle", "mode": "full"}},
		{"full mode incident limit", map[string]any{"content": "needle", "mode": "full", "limit": float64(6)}},
		{"full mode maximum limit", map[string]any{"content": "needle", "mode": "full", "limit": float64(20)}},
		{"snippets default limit", map[string]any{"path_ref": "src/payments/stripe.go"}},
		{"snippets maximum limit", map[string]any{"path_ref": "src/payments/stripe.go", "limit": float64(200)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := callSearch(t, base, tt.args)
			text := searchText(t, res)
			if len(text) > searchResponseByteBudget {
				t.Errorf("response is %d bytes, over the budget %d", len(text), searchResponseByteBudget)
			}
			got := unmarshalSearchEnvelope(t, res)
			if len(got.Results) == 0 {
				t.Fatal("the budget removed every row")
			}
			again := searchText(t, callSearch(t, base, tt.args))
			if again != text {
				t.Error("two identical calls returned different bytes")
			}
		})
	}
}

func TestHandleSearchDocuments_FullModeBodyTruncatedKeepsRow(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	original := multiByteBody("needle needle needle", 24_000)
	writeFixtureDoc(t, localArch, "packages.doc.md", "Auth Packages", original)
	for i := range 5 {
		writeFixtureDoc(t, globalArch, fmt.Sprintf("org-%d.doc.md", i),
			fmt.Sprintf("Org %d", i), multiByteBody("needle", 20_000))
	}

	got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{
		"content": "needle", "mode": "full", "limit": float64(6),
	}))
	if len(got.Results) != 6 || got.Truncated {
		t.Fatalf("page holds %d rows, truncated=%v; want all 6 rows kept and bodies shortened instead",
			len(got.Results), got.Truncated)
	}
	for _, row := range got.Results {
		if !row.BodyTruncated {
			t.Errorf("%s: body_truncated is false for a body that cannot fit", row.Path)
		}
		if !utf8.ValidString(row.Body) {
			t.Errorf("%s: shortened body is not valid UTF-8", row.Path)
		}
		if len(row.Body) < searchBodyFloorBytes {
			t.Errorf("%s: body kept %d bytes, under the floor %d", row.Path, len(row.Body), searchBodyFloorBytes)
		}
	}
	top := got.Results[0]
	if top.SourceID != "local" {
		t.Fatalf("top row is %q, want the local document", top.Path)
	}
	if !strings.HasPrefix(original, top.Body) {
		t.Error("shortened body is not a prefix of the document body")
	}
	if top.BodyBytes != len(original) {
		t.Errorf("body_bytes = %d, want the full body length %d", top.BodyBytes, len(original))
	}
}

func TestHandleSearchDocuments_FullModeSmallBodiesUncut(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	small := "needle\n\nA short body.\n"
	writeFixtureDoc(t, localArch, "small.rule.md", "Small Needle", small)
	writeFixtureDoc(t, localArch, "large-a.doc.md", "Large A", multiByteBody("needle", 60_000))
	writeFixtureDoc(t, localArch, "large-b.doc.md", "Large B", multiByteBody("needle", 60_000))

	res := callSearch(t, base, map[string]any{"content": "needle", "mode": "full"})
	got := unmarshalSearchEnvelope(t, res)
	if len(got.Results) != 3 {
		t.Fatalf("got %d rows, want 3", len(got.Results))
	}
	var large []int
	for _, row := range got.Results {
		if strings.HasSuffix(row.Path, "small.rule.md") {
			if row.BodyTruncated || row.Body != small || row.BodyBytes != 0 {
				t.Errorf("small body arrived as %q (truncated=%v, body_bytes=%d), want it whole and unmarked",
					row.Body, row.BodyTruncated, row.BodyBytes)
			}
			continue
		}
		large = append(large, len(row.Body))
	}
	// Max-min: what the small body leaves unused goes to the large ones, in
	// equal shares.
	if len(large) != 2 || large[0] < 15_000 || large[0] != large[1] {
		t.Errorf("large bodies kept %v bytes, want two equal shares above 15000", large)
	}
	if len(searchText(t, res)) > searchResponseByteBudget {
		t.Errorf("response is over the budget")
	}
}

func TestHandleSearchDocuments_BudgetCutKeepsSourceRepresentation(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	for i := range 60 {
		writeFixtureDoc(t, localArch, fmt.Sprintf("cited-%02d.rule.md", i), fmt.Sprintf("Cited %02d", i),
			strings.Repeat("see @src/payments/stripe.go and src/payments/ in this long sentence of evidence. ", 12))
	}
	writeFixtureDoc(t, globalArch, "org.doc.md", "Org Mention", "one mention of @src/ only")

	got := unmarshalSearchEnvelope(t, callSearch(t, base, map[string]any{
		"path_ref": "src/payments/stripe.go", "limit": float64(50),
	}))
	if !got.Truncated {
		t.Fatal("truncated is false although the budget cannot hold 50 of these rows")
	}
	if len(got.Index) != 50 {
		t.Errorf("index holds %d entries, want the 50 rows the limit admitted", len(got.Index))
	}
	if len(got.Results) >= 50 {
		t.Errorf("results hold %d rows, want fewer than the limit", len(got.Results))
	}
	for i, row := range got.Results {
		if got.Index[i].Path != row.Path && row.SourceID == "local" {
			t.Errorf("results[%d] = %q, want the index order for local rows", i, row.Path)
		}
	}
	globals := 0
	for _, row := range got.Results {
		if row.Global {
			globals++
		}
	}
	if globals != 1 {
		t.Errorf("results carry %d global rows, want the one global match kept through the budget cut", globals)
	}
}
