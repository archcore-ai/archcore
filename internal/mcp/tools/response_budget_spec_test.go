package tools

// Properties of the search_documents byte budget
// (read-tool-responses-survive-host-truncation.adr, search-documents.spec §8 and §12):
//
//   - every response fits searchResponseByteBudget, except a page of one row
//     whose own fields exceed it;
//   - rows leave the page only as far as the budget requires, and every
//     matching source stays on the shorter page;
//   - bodies share the spare bytes max-min, and a shortened body is a marked,
//     rune-safe prefix of the document body.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"archcore-cli/internal/docs"
)

// manySourceFixture builds a project that declares one global source per id.
func manySourceFixture(t *testing.T, ids ...string) (base, localArch string, globalArchs map[string]string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "project")
	localArch = filepath.Join(base, ".archcore")
	if err := os.MkdirAll(localArch, 0o755); err != nil {
		t.Fatal(err)
	}
	globalArchs = make(map[string]string, len(ids))
	declared := make([]string, 0, len(ids))
	for _, id := range ids {
		dir := filepath.Join(root, id, ".archcore")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		globalArchs[id] = dir
		declared = append(declared, fmt.Sprintf(`{"id":%q,"path":"../%s/.archcore"}`, id, id))
	}
	settings := `{"sync":"none","globals":[` + strings.Join(declared, ",") + `]}`
	if err := os.WriteFile(filepath.Join(localArch, "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	return base, localArch, globalArchs
}

func writeTaggedFixtureDoc(t *testing.T, dir, name, title string, tags []string, body string) {
	t.Helper()
	var front strings.Builder
	front.WriteString("---\ntitle: \"" + title + "\"\nstatus: accepted\n")
	if len(tags) > 0 {
		front.WriteString("tags:\n")
		for _, tag := range tags {
			front.WriteString("  - \"" + tag + "\"\n")
		}
	}
	front.WriteString("---\n\n")
	if err := os.WriteFile(filepath.Join(dir, name), []byte(front.String()+body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func numberedTags(n int) []string {
	tags := make([]string, n)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag-number-%05d", i)
	}
	return tags
}

// budgetOpening makes every matrix document answer both the content query and
// the path_ref query.
const budgetOpening = "needle, see @src/payments/stripe.go and src/payments/ here.\n\n"

func filledBody(line string, size int) string {
	return budgetOpening + strings.Repeat(line, size/len(line)+1)
}

// budgetCorpus is one corpus shape of the budget matrix. bodies maps a file
// name to the body written for it.
type budgetCorpus struct {
	base    string
	sources int
	bodies  map[string]string
}

func (c *budgetCorpus) add(t *testing.T, dir, name, title string, tags []string, body string) {
	t.Helper()
	writeTaggedFixtureDoc(t, dir, name, title, tags, body)
	c.bodies[name] = body
}

func buildBudgetCorpus(t *testing.T, shape string) *budgetCorpus {
	t.Helper()
	corpus := &budgetCorpus{sources: 2, bodies: map[string]string{}}
	base, localArch, globalArch := twoSourceFixture(t)
	corpus.base = base

	fill := func(locals, globals int, title func(i int) string, tags []string, body func(i int) string) {
		for i := range locals {
			corpus.add(t, localArch, fmt.Sprintf("local-%03d.doc.md", i), title(i), tags, body(i))
		}
		for i := range globals {
			corpus.add(t, globalArch, fmt.Sprintf("org-%03d.adr.md", i), "Org "+title(i), tags, body(i))
		}
	}
	plainTitle := func(i int) string { return fmt.Sprintf("Document %03d", i) }

	switch shape {
	case "ascii":
		fill(24, 6, plainTitle, nil, func(int) string {
			return filledBody("A plain line of filler text without any match in it.\n", 30_000)
		})
	case "two-byte runes":
		fill(24, 6, plainTitle, nil, func(int) string {
			return filledBody("Γραμμή εγγράφου χωρίς αντιστοιχίες, μόνο όγκος.\n", 30_000)
		})
	case "six-byte escapes":
		fill(12, 4, plainTitle, nil, func(int) string {
			return filledBody("<&>\x01\u2028\u2029\xff \"quoted\\\" <tag>\n", 12_000)
		})
	case "many tiny bodies":
		fill(260, 40, plainTitle, nil, func(int) string { return budgetOpening })
	case "one giant body":
		fill(3, 2, plainTitle, nil, func(int) string { return budgetOpening + "Short.\n" })
		corpus.add(t, localArch, "giant.rule.md", "Needle Giant", nil,
			filledBody("Πολύ μεγάλο έγγραφο, γραμμή προς γραμμή.\n", 2_000_000))
	case "long titles and many tags":
		fill(40, 10, func(i int) string {
			return fmt.Sprintf("%03d ", i) + strings.Repeat("Μεγάλος τίτλος ", 9)
		}, numberedTags(40), func(int) string {
			return filledBody("Filler line for a tagged document.\n", 5_000)
		})
	case "eight global sources":
		ids := []string{"org-a", "org-b", "org-c", "org-d", "org-e", "org-f", "org-g", "org-h"}
		manyBase, manyLocal, globalArchs := manySourceFixture(t, ids...)
		corpus.base, corpus.sources = manyBase, len(ids)+1
		body := filledBody("Γραμμή κοινού εγγράφου χωρίς αντιστοιχίες.\n", 12_000)
		for i := range 4 {
			corpus.add(t, manyLocal, fmt.Sprintf("local-%03d.doc.md", i), plainTitle(i), nil, body)
		}
		for _, id := range ids {
			for i := range 4 {
				corpus.add(t, globalArchs[id], fmt.Sprintf("%s-%03d.adr.md", id, i), id+" "+plainTitle(i), nil, body)
			}
		}
	default:
		t.Fatalf("unknown corpus shape %q", shape)
	}
	return corpus
}

// asDecoded returns s as a client reads it after the JSON round trip, which
// turns each invalid byte into U+FFFD.
func asDecoded(t *testing.T, s string) string {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var out string
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestHandleSearchDocuments_ByteBudgetHoldsAcrossCorpusShapes pins
// search-documents.spec §8.5 as a property over mode, limit, and corpus shape.
// No row of these corpora exceeds the budget alone, so no response may.
func TestHandleSearchDocuments_ByteBudgetHoldsAcrossCorpusShapes(t *testing.T) {
	t.Parallel()
	shapes := []string{
		"ascii", "two-byte runes", "six-byte escapes", "many tiny bodies",
		"one giant body", "long titles and many tags", "eight global sources",
	}
	calls := []struct {
		name string
		args map[string]any
	}{
		{"snippets default limit", map[string]any{"content": "needle"}},
		{"snippets limit 1", map[string]any{"content": "needle", "limit": float64(1)}},
		{"snippets limit 200", map[string]any{"content": "needle", "limit": float64(200)}},
		{"snippets path_ref limit 200", map[string]any{"path_ref": "src/payments/stripe.go", "limit": float64(200)}},
		{"snippets both filters by mtime", map[string]any{"content": "needle", "path_ref": "src/payments/", "sort": "mtime", "limit": float64(200)}},
		{"full default limit", map[string]any{"content": "needle", "mode": "full"}},
		{"full limit 1", map[string]any{"content": "needle", "mode": "full", "limit": float64(1)}},
		{"full incident limit", map[string]any{"content": "needle", "match": "exact", "mode": "full", "limit": float64(6)}},
		{"full limit 20", map[string]any{"content": "needle", "mode": "full", "limit": float64(20)}},
		{"full limit clamped from 200", map[string]any{"path_ref": "src/payments/stripe.go", "mode": "full", "limit": float64(200)}},
		{"full metadata only by mtime", map[string]any{"status": "accepted", "mode": "full", "sort": "mtime", "limit": float64(20)}},
	}
	for _, shape := range shapes {
		t.Run(shape, func(t *testing.T) {
			t.Parallel()
			corpus := buildBudgetCorpus(t, shape)
			for _, tt := range calls {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					res := callSearch(t, corpus.base, tt.args)
					text := searchText(t, res)
					if len(text) > searchResponseByteBudget {
						t.Errorf("response is %d bytes, over the budget %d; a host would store it and show the agent a preview",
							len(text), searchResponseByteBudget)
					}
					if again := searchText(t, callSearch(t, corpus.base, tt.args)); again != text {
						t.Error("two identical calls returned different bytes")
					}
					assertBudgetedPage(t, corpus, tt.args, text, unmarshalSearchEnvelope(t, res))
				})
			}
		})
	}
}

func assertBudgetedPage(t *testing.T, corpus *budgetCorpus, args map[string]any, text string, got searchDocumentsResult) {
	t.Helper()
	if len(got.Results) == 0 {
		t.Fatal("the budget removed every row")
	}
	if got.Truncated != (len(got.Results) < len(got.Index)) {
		t.Errorf("truncated = %v with %d rows for %d index entries; the flag must say exactly when rows left the page",
			got.Truncated, len(got.Results), len(got.Index))
	}
	if len(got.Coverage) != corpus.sources {
		t.Fatalf("coverage names %d sources, want %d", len(got.Coverage), corpus.sources)
	}
	onPage := map[string]bool{}
	for _, row := range got.Results {
		onPage[row.SourceID] = true
	}
	matching := 0
	for _, count := range got.Hits {
		if count > 0 {
			matching++
		}
	}
	if want := min(matching, len(got.Results)); len(onPage) != want {
		t.Errorf("page of %d rows carries %d sources, want %d; the budget cut dropped a matching source",
			len(got.Results), len(onPage), want)
	}

	if args["mode"] != "full" {
		for _, field := range []string{`"body":`, `"body_truncated":`, `"body_bytes":`} {
			if strings.Contains(text, field) {
				t.Errorf("snippets response carries %s", field)
			}
		}
		return
	}
	for _, row := range got.Results {
		written, ok := corpus.bodies[filepath.Base(row.Path)]
		if !ok {
			t.Fatalf("row %q is not a fixture document", row.Path)
		}
		whole := asDecoded(t, written)
		if !strings.HasPrefix(whole, row.Body) {
			t.Errorf("%s: body is not a prefix of the document; an agent would read text the file does not hold", row.Path)
		}
		if !utf8.ValidString(row.Body) {
			t.Errorf("%s: body is not valid UTF-8", row.Path)
		}
		shortened := len(row.Body) < len(whole)
		if row.BodyTruncated != shortened {
			t.Errorf("%s: body_truncated = %v for a body of %d of %d bytes; an unmarked short body could be written back",
				row.Path, row.BodyTruncated, len(row.Body), len(whole))
		}
		wantBytes := 0
		if shortened {
			wantBytes = len(written)
		}
		if row.BodyBytes != wantBytes {
			t.Errorf("%s: body_bytes = %d, want %d", row.Path, row.BodyBytes, wantBytes)
		}
	}
}

// TestHandleSearchDocuments_OneOversizedRowIsTheOnlyOverflow pins
// search-documents.spec §8.8 from both sides: the oversized row stays alone
// when it ranks first, and it leaves the page when it ranks last.
func TestHandleSearchDocuments_OneOversizedRowIsTheOnlyOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		oversizedBody string
		otherTitle    string
		mode          string
		wantTopSuffix string
		wantOverflow  bool
	}{
		{"oversized row ranks first, snippets", "needle\n", "Other", "snippets", "oversized.doc.md", true},
		{"oversized row ranks first, full", "needle\n" + strings.Repeat("filler line\n", 500), "Other", "full", "oversized.doc.md", true},
		{"oversized row ranks last, snippets", "needle\n", "Needle Other", "snippets", "other.doc.md", false},
		{"oversized row ranks last, full", "needle\n", "Needle Other", "full", "other.doc.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, localArch, _ := twoSourceFixture(t)
			oversizedTitle := "Needle Oversized"
			if !tt.wantOverflow {
				oversizedTitle = "Oversized"
			}
			writeTaggedFixtureDoc(t, localArch, "oversized.doc.md", oversizedTitle, numberedTags(3000), tt.oversizedBody)
			writeFixtureDoc(t, localArch, "other.doc.md", tt.otherTitle, "needle in the body\n")

			res := callSearch(t, base, map[string]any{"content": "needle", "mode": tt.mode})
			text := searchText(t, res)
			got := unmarshalSearchEnvelope(t, res)

			if len(got.Results) != 1 || !got.Truncated || len(got.Index) != 2 {
				t.Fatalf("page holds %d rows, truncated=%v, index=%d; want one row, truncated, and both rows in the index",
					len(got.Results), got.Truncated, len(got.Index))
			}
			if !strings.HasSuffix(got.Results[0].Path, tt.wantTopSuffix) {
				t.Errorf("kept row is %q, want %q: rows leave from the tail only", got.Results[0].Path, tt.wantTopSuffix)
			}
			if overflow := len(text) > searchResponseByteBudget; overflow != tt.wantOverflow {
				t.Errorf("response is %d bytes, overflow=%v, want overflow=%v", len(text), overflow, tt.wantOverflow)
			}
			if tt.mode == "full" && tt.wantOverflow {
				row := got.Results[0]
				if !row.BodyTruncated || jsonStringLen(row.Body) > searchBodyFloorBytes {
					t.Errorf("oversized row kept a body of %d encoded bytes, truncated=%v; want it cut to the floor %d",
						jsonStringLen(row.Body), row.BodyTruncated, searchBodyFloorBytes)
				}
			}
		})
	}
}

// TestHandleSearchDocuments_IndexStaysInsideByteBudget: 200 index entries with
// long titles once exceeded the budget although the one row kept was small.
// searchIndexByteBudget now cuts the index from the tail and marks the
// response truncated.
func TestHandleSearchDocuments_IndexStaysInsideByteBudget(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	for i := range 200 {
		writeFixtureDoc(t, localArch, fmt.Sprintf("doc-%03d.doc.md", i),
			strings.Repeat("Ω", 70)+fmt.Sprintf(" needle %03d", i), "Body.\n")
	}

	res := callSearch(t, base, map[string]any{"content": "needle", "limit": float64(200)})
	text := searchText(t, res)
	got := unmarshalSearchEnvelope(t, res)

	rowBytes, err := json.Marshal(got.Results)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > searchResponseByteBudget {
		t.Errorf("response is %d bytes, over the budget %d, while its %d rows take %d bytes; the index alone overflows the host limit",
			len(text), searchResponseByteBudget, len(got.Results), len(rowBytes))
	}
}

func budgetRow(path, sourceID string, score int, title, body string) searchResult {
	kind := docs.SourceKindLocal
	if sourceID != "local" {
		kind = docs.SourceKindGlobal
	}
	return searchResult{
		Path: path, Title: title, Type: "doc", SourceID: sourceID, SourceKind: kind,
		Matches:           []searchMatch{},
		IncomingRelations: []DocumentRelation{},
		OutgoingRelations: []DocumentRelation{},
		Body:              body,
		score:             score,
	}
}

// bodilessRowBytes is what the rows add to a response: each row and the comma
// after it.
func bodilessRowBytes(t *testing.T, rows []searchResult) (total, last int) {
	t.Helper()
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		last = len(data) + len(",")
		total += last
	}
	return total, last
}

func rowPaths(rows []searchResult) []string {
	paths := make([]string, len(rows))
	for i, row := range rows {
		paths[i] = row.Path
	}
	return paths
}

func TestFitSearchPage_DropsOnlyAsManyRowsAsNeeded(t *testing.T) {
	t.Parallel()
	ranked := make([]searchResult, 10)
	for i := range ranked {
		ranked[i] = budgetRow(fmt.Sprintf(".archcore/row-%02d.doc.md", i), "local", 100-i,
			strings.Repeat("t", 900+i), "")
	}
	all, last := bodilessRowBytes(t, ranked)
	exactFit := searchResponseByteBudget - all

	tests := []struct {
		name          string
		pageLen       int
		headBytes     int
		wantRows      int
		wantTruncated bool
	}{
		{"fits to the byte", 10, exactFit, 10, false},
		{"one byte over drops one row", 10, exactFit + 1, 9, true},
		{"over by the last row drops it alone", 10, exactFit + last, 9, true},
		{"one byte past the last row drops two", 10, exactFit + last + 1, 8, true},
		{"nothing fits, one row stays", 10, searchResponseByteBudget, 1, true},
		{"page shorter than the ranking, fits", 4, 0, 4, false},
		{"room to spare", 10, 0, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, truncated, err := fitSearchPage(slices.Clone(ranked), tt.pageLen, tt.headBytes, "relevance")
			if err != nil {
				t.Fatal(err)
			}
			if len(page) != tt.wantRows {
				t.Errorf("page holds %d rows, want %d; a row left the page that the budget had room for, or stayed over it",
					len(page), tt.wantRows)
			}
			if truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTruncated)
			}
			if want := rowPaths(ranked[:len(page)]); !slices.Equal(rowPaths(page), want) {
				t.Errorf("page = %v, want the top rows %v: rows leave from the tail", rowPaths(page), want)
			}
		})
	}
}

func TestFitSearchPage_KeepsEverySourceOnTheShorterPage(t *testing.T) {
	t.Parallel()
	var ranked []searchResult
	for i := range 9 {
		ranked = append(ranked, budgetRow(fmt.Sprintf(".archcore/local-%d.doc.md", i), "local", 100-i,
			strings.Repeat("t", 900), ""))
	}
	ranked = append(ranked, budgetRow("../org/.archcore/org.doc.md", "org", 1, strings.Repeat("t", 900), ""))
	all, last := bodilessRowBytes(t, ranked)

	page, truncated, err := fitSearchPage(ranked, len(ranked), searchResponseByteBudget-all+5*last+1, "relevance")
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 4 || !truncated {
		t.Fatalf("page holds %d rows, truncated=%v; want 4 rows, truncated", len(page), truncated)
	}
	want := []string{".archcore/local-0.doc.md", ".archcore/local-1.doc.md", ".archcore/local-2.doc.md", "../org/.archcore/org.doc.md"}
	if got := rowPaths(page); !slices.Equal(got, want) {
		t.Errorf("page = %v, want %v; the global source lost its only row to the budget cut", got, want)
	}
}

func TestFitSearchPage_SecondCallReturnsTheSamePage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		pageLen  int
		knownGap string
	}{
		{"page is the whole ranking", 6, ""},
		{"page is shorter than the ranking", 4, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.knownGap != "" {
				t.Skip(tt.knownGap)
			}
			ranked := make([]searchResult, 6)
			for i := range ranked {
				body := strings.Repeat("Γραμμή σώματος εγγράφου.\n", 400*(i+1))
				ranked[i] = budgetRow(fmt.Sprintf(".archcore/row-%d.doc.md", i), "local", 100-i, "Row", body)
			}

			first, firstTruncated, err := fitSearchPage(ranked, tt.pageLen, 500, "relevance")
			if err != nil {
				t.Fatal(err)
			}
			first = slices.Clone(first)
			second, secondTruncated, err := fitSearchPage(ranked, tt.pageLen, 500, "relevance")
			if err != nil {
				t.Fatal(err)
			}

			if firstTruncated != secondTruncated || !reflect.DeepEqual(first, second) {
				t.Errorf("second call returned a different page; the first call changed its input")
			}
			cut := 0
			for _, row := range first {
				if row.BodyTruncated {
					cut++
				}
			}
			if cut == 0 {
				t.Fatal("no body was shortened, so the fixture does not reach the code that writes to the rows")
			}
		})
	}
}

func TestFitSearchPage_BodilessRowsGainNoBodyFields(t *testing.T) {
	t.Parallel()
	ranked := make([]searchResult, 80)
	for i := range ranked {
		ranked[i] = budgetRow(fmt.Sprintf(".archcore/row-%02d.doc.md", i), "local", 100-i, strings.Repeat("t", 900), "")
	}
	page, truncated, err := fitSearchPage(ranked, len(ranked), 0, "relevance")
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("the fixture fits, so it does not reach the cut")
	}
	for _, row := range page {
		if row.Body != "" || row.BodyTruncated || row.BodyBytes != 0 {
			t.Errorf("%s: body fields set on a snippets row (truncated=%v, body_bytes=%d)", row.Path, row.BodyTruncated, row.BodyBytes)
		}
	}
}

// TestFitSearchPage_FullPageFitsWithEveryBodyAtItsShare measures the marshaled
// page: the 64 reserved bytes per body must cover the fields a cut adds.
func TestFitSearchPage_FullPageFitsWithEveryBodyAtItsShare(t *testing.T) {
	t.Parallel()
	const headBytes = 2000
	ranked := make([]searchResult, 20)
	for i := range ranked {
		ranked[i] = budgetRow(fmt.Sprintf(".archcore/row-%02d.doc.md", i), "local", 100-i, "Row",
			strings.Repeat("<&> γραμμή \u2028\n", 900_000/20))
	}
	page, truncated, err := fitSearchPage(ranked, len(ranked), headBytes, "relevance")
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(page) != 20 {
		t.Fatalf("page holds %d rows, truncated=%v; want all 20 rows with shortened bodies", len(page), truncated)
	}
	data, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if total := headBytes + len(data); total > searchResponseByteBudget {
		t.Errorf("head and rows take %d bytes, over the budget %d", total, searchResponseByteBudget)
	}
}

// TestFitSearchPage_ReportsARowItCannotMeasure uses a year that
// time.Time.MarshalJSON refuses, the one way a row fails to serialize.
func TestFitSearchPage_ReportsARowItCannotMeasure(t *testing.T) {
	t.Parallel()
	row := budgetRow(".archcore/late.doc.md", "local", 1, "Late", "")
	row.ModTime = time.Date(10_000, time.January, 1, 0, 0, 0, 0, time.UTC)

	page, _, err := fitSearchPage([]searchResult{row}, 1, 0, "relevance")

	if err == nil || !strings.Contains(err.Error(), ".archcore/late.doc.md") {
		t.Errorf("err = %v, want an error that names the row; a swallowed error would ship a page measured as zero bytes", err)
	}
	if page != nil {
		t.Errorf("page = %v, want no page beside the error", rowPaths(page))
	}
}

func TestShareBodies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		bodySizes []int
		spare     int
		wantSizes []int
	}{
		{"grant equal to the body leaves it whole", []int{2000}, 976, []int{2000}},
		{"grant one byte short cuts it", []int{2000}, 975, []int{1999}},
		{"equal bodies get equal shares", []int{10_000, 10_000, 10_000}, 3000, []int{2024, 2024, 2024}},
		{"a small body arrives whole and frees its surplus", []int{10_000, 1500, 10_000}, 3000, []int{2286, 1500, 2286}},
		{"a body under the floor is never cut", []int{500, 10_000}, 0, []int{500, 1024}},
		{"a deficit comes out of the floor of the one forced row", []int{5000}, -300, []int{724}},
		{"rows without a body take no share", []int{0, 10_000, 10_000}, 2000, []int{0, 2024, 2024}},
		{"no spare for one large body among small ones", []int{100, 200, 10_000}, 10, []int{100, 200, 1034}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, encoded := sharedPage(tt.bodySizes)

			shareBodies(page, encoded, tt.spare)

			for i, row := range page {
				if len(row.Body) != tt.wantSizes[i] {
					t.Errorf("row %d kept %d bytes, want %d", i, len(row.Body), tt.wantSizes[i])
				}
				wasCut := tt.wantSizes[i] < tt.bodySizes[i]
				wantBytes := 0
				if wasCut {
					wantBytes = tt.bodySizes[i]
				}
				if row.BodyTruncated != wasCut || row.BodyBytes != wantBytes {
					t.Errorf("row %d: body_truncated=%v body_bytes=%d, want %v and %d; the marks must follow the cut exactly",
						i, row.BodyTruncated, row.BodyBytes, wasCut, wantBytes)
				}
			}
		})
	}
}

func TestShareBodies_RemainderStaysInsideTheSpare(t *testing.T) {
	t.Parallel()
	for _, spare := range []int{3000, 3001, 3002, 7, 1, 0} {
		t.Run(fmt.Sprintf("spare %d", spare), func(t *testing.T) {
			t.Parallel()
			page, encoded := sharedPage([]int{10_000, 10_000, 10_000})

			shareBodies(page, encoded, spare)

			granted, smallest, largest := 0, len(page[0].Body), len(page[0].Body)
			for _, row := range page {
				granted += len(row.Body) - searchBodyFloorBytes
				smallest, largest = min(smallest, len(row.Body)), max(largest, len(row.Body))
			}
			if granted != spare {
				t.Errorf("bodies took %d bytes beyond the floor, want the whole spare %d and not a byte more", granted, spare)
			}
			if largest-smallest > 1 {
				t.Errorf("shares range from %d to %d bytes, want equal bodies within one byte of each other", smallest, largest)
			}
		})
	}
}

// sharedPage builds rows with ASCII bodies without a line break, so a cut lands
// exactly on its grant.
func sharedPage(bodySizes []int) ([]searchResult, map[string]int) {
	page := make([]searchResult, len(bodySizes))
	encoded := make(map[string]int, len(bodySizes))
	for i, size := range bodySizes {
		page[i] = budgetRow(fmt.Sprintf(".archcore/row-%d.doc.md", i), "local", 0, "Row", strings.Repeat("x", size))
		if size > 0 {
			encoded[page[i].Path] = size
		}
	}
	return page, encoded
}

func TestCutForJSONBudget_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     string
		budget int
		want   string
	}{
		{"budget under a two-byte rune", "Ωabc", 1, ""},
		{"budget under a six-byte escape", "<abc", 5, ""},
		{"budget under an escaped newline", "\nabc", 1, ""},
		{"negative budget", "abc", -1, ""},
		{"empty input", "", 10, ""},
		{"only newlines", "\n\n\n\n\n\n", 7, "\n\n\n"},
		{"newline exactly at the half", "aaaa\nbbbbbbbbbbbb", 9, "aaaa\n"},
		{"newline one byte before the half", "aaaa\nbbbbbbbbbbbb", 11, "aaaa\nbbbbb"},
		{"crlf stays together", "line one\r\nline two\r\nline three", 27, "line one\r\nline two\r\n"},
		{"cut between cr and lf falls back a line", "line one\r\nline two\r\nline three", 23, "line one\r\n"},
		{"four-byte rune does not fit", "ab👍cd", 5, "ab"},
		{"four-byte rune fits", "ab👍cd", 6, "ab👍"},
		{"line separator costs six", "a\u2028b", 6, "a"},
		{"line separator fits at seven", "a\u2028b", 7, "a\u2028"},
		{"invalid byte costs six", "a\xffb", 6, "a"},
		{"invalid byte fits at seven", "a\xffb", 7, "a\xff"},
		{"exact fit returns the whole input", "a<b", 8, "a<b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.in[:cutForJSONBudget(tt.in, tt.budget)]
			if got != tt.want {
				t.Errorf("cut = %q, want %q", got, tt.want)
			}
			if size := marshaledStringLen(t, got); size > max(tt.budget, 0) {
				t.Errorf("cut %q encodes to %d bytes, over the budget %d", got, size, tt.budget)
			}
		})
	}
}

// largestFittingPrefix is the oracle of FuzzCutForJSONBudget. It asks
// encoding/json for the size of each rune and shares no code with jsonRuneLen.
func largestFittingPrefix(t *testing.T, s string, budget int) int {
	t.Helper()
	end, used := 0, 0
	for end < len(s) {
		_, width := utf8.DecodeRuneInString(s[end:])
		used += marshaledStringLen(t, s[end:end+width])
		if used > budget {
			break
		}
		end += width
	}
	return end
}

func FuzzCutForJSONBudget(f *testing.F) {
	seeds := []struct {
		in     string
		budget int
	}{
		{"", 0}, {"plain", 3}, {"ΩΩΩΩ", 5}, {"a\n\n\nb", 4}, {"first\nsecond\nthird", 14},
		{"<&>", 7}, {"a\u2028b", 6}, {"\xff\xfe", 6}, {"ab👍cd", 5}, {"line\r\nline", 7}, {"abc", -4},
	}
	for _, seed := range seeds {
		f.Add(seed.in, seed.budget)
	}
	f.Fuzz(func(t *testing.T, s string, budget int) {
		n := cutForJSONBudget(s, budget)
		if n < 0 || n > len(s) {
			t.Fatalf("cut = %d, outside the input of %d bytes", n, len(s))
		}
		kept := s[:n]
		if size := marshaledStringLen(t, kept); size > max(budget, 0) {
			t.Errorf("cut %q encodes to %d bytes, over the budget %d", kept, size, budget)
		}
		if utf8.ValidString(s) && !utf8.ValidString(kept) {
			t.Errorf("cut %q splits a rune of valid input", kept)
		}
		largest := largestFittingPrefix(t, s, budget)
		want := largest
		if largest < len(s) {
			if i := strings.LastIndexByte(s[:largest], '\n'); i >= largest/2 {
				want = i + 1
			}
		}
		if n != want {
			t.Errorf("cut = %d, want %d: the largest fitting prefix is %d bytes", n, want, largest)
		}
	})
}
