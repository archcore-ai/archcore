package tools

// Tests for the slug match field and separator folding
// (search-matches-the-slug-and-folds-separators.adr, search-documents.spec §6).

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"archcore-cli/internal/docs"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestSearchDocuments_TitleHitWinsTheExcerptOverTheSlug(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "billing-api.doc.md", "Billing API Reference", "Endpoints.\n")

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "billing"}))
	if len(got) != 1 || len(got[0].Matches) != 1 {
		t.Fatalf("results = %+v, want one row with one match", got)
	}
	if m := got[0].Matches[0]; m.Specificity != 3 || m.Excerpt != "Billing API Reference" {
		t.Errorf("match = %+v, want the title as the excerpt: the title is the field a reader recognizes", m)
	}
}

// TestSearchDocuments_SlugHitOutranksABodyHit: a slug hit scores as a title
// hit, so it ranks above a body hit of a document type with a higher priority.
func TestSearchDocuments_SlugHitOutranksABodyHit(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "acme-gateway.doc.md", "Edge Proxy", "Routes traffic.\n")
	writeFixtureDoc(t, localArch, "notes.rule.md", "Notes", "The gateway restarts nightly. The gateway logs to disk.\n")

	got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "gateway"}))
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if !strings.HasSuffix(got[0].Path, "acme-gateway.doc.md") {
		t.Errorf("first row is %q, want the document named for the query", got[0].Path)
	}
}

func TestSearchDocuments_SlugMatchesInEveryModeAndCase(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "acme-id-sdk.doc.md", "The Identity Client", "A server-side client.\n")
	writeFixtureDoc(t, globalArch, "org-billing-sdk.adr.md", "Shared Invoicing", "One decision.\n")
	writeFixtureDoc(t, localArch, "unrelated.doc.md", "Unrelated", "Nothing to see.\n")

	tests := []struct {
		name     string
		args     map[string]any
		wantSlug string
		wantKind docs.SourceKind
		wantRef  string
	}{
		{"exact, uppercase query", map[string]any{"content": "ACME-ID-SDK", "match": "exact"}, "acme-id-sdk", docs.SourceKindLocal, "acme-id-sdk"},
		{"exact, part of the slug", map[string]any{"content": "Id-Sdk", "match": "exact"}, "acme-id-sdk", docs.SourceKindLocal, "id-sdk"},
		{"all, other separators", map[string]any{"content": "Acme_Id.Sdk"}, "acme-id-sdk", docs.SourceKindLocal, "acme id sdk"},
		{"any, beside an absent word", map[string]any{"content": "zephyrite acme-id-sdk", "match": "any"}, "acme-id-sdk", docs.SourceKindLocal, "acme id sdk"},
		{"exact, global document", map[string]any{"content": "org-billing-sdk", "match": "exact"}, "org-billing-sdk", docs.SourceKindGlobal, "org-billing-sdk"},
		{"all, global document in the global scope", map[string]any{"content": "billing-sdk", "source": "global"}, "org-billing-sdk", docs.SourceKindGlobal, "billing sdk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, tt.args))
			if len(got) != 1 || got[0].SourceKind != tt.wantKind || !strings.Contains(got[0].Path, tt.wantSlug+".") {
				t.Fatalf("results = %+v, want the %s document %q alone", got, tt.wantKind, tt.wantSlug)
			}
			want := searchMatch{Kind: matchKindContent, Ref: tt.wantRef, Specificity: 3, Excerpt: tt.wantSlug}
			if len(got[0].Matches) != 1 || got[0].Matches[0] != want {
				t.Errorf("matches = %+v, want %+v", got[0].Matches, want)
			}
		})
	}
}

func TestSearchDocuments_OneWordInTheSlugAnotherInTheBody(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "acme-gateway.doc.md", "Edge Proxy", "Handles throttling for every tenant.\n")
	writeFixtureDoc(t, localArch, "gateway-notes.doc.md", "Loose Notes", "Nothing on the other word.\n")

	t.Run("all keeps the document that holds both words", func(t *testing.T) {
		t.Parallel()
		got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "gateway throttling"}))
		if len(got) != 1 || !strings.HasSuffix(got[0].Path, "acme-gateway.doc.md") {
			t.Fatalf("results = %+v, want the one document with a slug hit and a body hit", got)
		}
		if len(got[0].Matches) != 2 {
			t.Fatalf("matches = %+v, want one per query word", got[0].Matches)
		}
		slugHit := searchMatch{Kind: matchKindContent, Ref: "gateway", Specificity: 3, Excerpt: "acme-gateway"}
		if got[0].Matches[0] != slugHit {
			t.Errorf("matches[0] = %+v, want %+v", got[0].Matches[0], slugHit)
		}
		bodyHit := got[0].Matches[1]
		if bodyHit.Ref != "throttling" || bodyHit.Specificity != 1 || !strings.Contains(bodyHit.Excerpt, "Handles throttling for every tenant.") {
			t.Errorf("matches[1] = %+v, want the body word at specificity 1 with its sentence", bodyHit)
		}
	})
	t.Run("any keeps the slug-only document too", func(t *testing.T) {
		t.Parallel()
		got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": "gateway throttling", "match": "any"}))
		if len(got) != 2 {
			t.Errorf("got %d rows, want both documents", len(got))
		}
	})
}

func TestSearchDocuments_SeparatorOnlyWordsInAnyMode(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "billing.doc.md", "Billing", "Body.\n")
	writeFixtureDoc(t, localArch, "unrelated.doc.md", "Unrelated", "Nothing -- to // see.\n")

	tests := []struct {
		name     string
		content  string
		wantErr  string
		wantRefs []string
	}{
		{"separator words beside a real word are dropped", "-- billing @ ::", "", []string{"billing"}},
		{"separators around a word are trimmed", "--billing--", "", []string{"billing"}},
		{"separators alone are refused", "-- // @ _ . :", "content must contain at least one word", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := callSearch(t, base, map[string]any{"content": tt.content, "match": "any"})
			if tt.wantErr != "" {
				if !res.IsError {
					t.Fatal("want an error: an empty word matches every document, and the page would read as a finding")
				}
				if tc, ok := mcp.AsTextContent(res.Content[0]); !ok || !strings.Contains(tc.Text, tt.wantErr) {
					t.Errorf("error = %+v, want %q", res.Content[0], tt.wantErr)
				}
				return
			}
			got := unmarshalSearch(t, res)
			if len(got) != 1 || !strings.HasSuffix(got[0].Path, "billing.doc.md") {
				t.Fatalf("results = %+v, want the billing document alone", got)
			}
			if refs := matchRefs(got[0].Matches); !slices.Equal(refs, tt.wantRefs) {
				t.Errorf("match refs = %v, want %v", refs, tt.wantRefs)
			}
		})
	}
}

func TestSearchDocuments_ExactModeKeepsSeparatorsLiteral(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	writeFixtureDoc(t, localArch, "client.doc.md", "The Identity Client", "Install `@acme-id/sdk` first.\n")
	writeFixtureDoc(t, globalArch, "org-client.adr.md", "Shared Client", "We adopt acme-id-sdk for every web app.\n")

	tests := []struct {
		name      string
		content   string
		wantPaths []string
		wantRef   string
	}{
		{"scoped spelling finds the scoped text alone", "@Acme-ID/sdk", []string{"client.doc.md"}, "@acme-id/sdk"},
		{"hyphen spelling finds the hyphen text alone", "acme-id-sdk", []string{"org-client.adr.md"}, "acme-id-sdk"},
		{"space spelling finds neither", "acme id sdk", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": tt.content, "match": "exact"}))
			var names []string
			for _, row := range got {
				names = append(names, row.Path[strings.LastIndex(row.Path, "/")+1:])
				if row.Matches[0].Ref != tt.wantRef {
					t.Errorf("ref = %q, want the literal query %q", row.Matches[0].Ref, tt.wantRef)
				}
			}
			if !slices.Equal(names, tt.wantPaths) {
				t.Errorf("exact matched %v, want %v: exact is a literal substring search", names, tt.wantPaths)
			}
		})
	}
}

// TestSearchDocuments_FoldedHitKeepsTheSourceExcerpt: the text ahead of the hit
// holds two-byte runes and more separators than half an excerpt window, so a
// fold that moved offsets would cut the window beside the hit.
func TestSearchDocuments_FoldedHitKeepsTheSourceExcerpt(t *testing.T) {
	t.Parallel()
	base, localArch, _ := twoSourceFixture(t)
	var lead strings.Builder
	for i := range 40 {
		fmt.Fprintf(&lead, "Ενότητα %d.%d: βήμα-προς-βήμα, βλ. v1.2/beta_%d.\n", i, i+1, i)
	}
	body := lead.String() + "Εγκαταστήστε το πακέτο `@acme-id/sdk` πρώτα.\n" + strings.Repeat("Το κείμενο συνεχίζεται. ", 40)
	writeFixtureDoc(t, localArch, "client.doc.md", "Πελάτης ταυτοποίησης", body)

	for _, query := range []string{"acme-id-sdk", "@acme-id/sdk", "ACME_ID:SDK"} {
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			got := unmarshalSearch(t, callSearch(t, base, map[string]any{"content": query}))
			if len(got) != 1 || len(got[0].Matches) != 1 {
				t.Fatalf("results = %+v, want one row with one match", got)
			}
			m := got[0].Matches[0]
			if m.Ref != "acme id sdk" || m.Specificity != 1 {
				t.Errorf("match = %+v, want the folded word as ref at body specificity", m)
			}
			if !strings.Contains(m.Excerpt, "Εγκαταστήστε το πακέτο `@acme-id/sdk` πρώτα.") {
				t.Errorf("excerpt = %q, want the source text around the hit with its separators intact", m.Excerpt)
			}
			if !utf8.ValidString(m.Excerpt) || !strings.HasPrefix(m.Excerpt, "...") {
				t.Errorf("excerpt = %q, want valid UTF-8 that opens with the cut mark", m.Excerpt)
			}
		})
	}
}

// TestSearchDocuments_SimpleWordIgnoresSeparatorFolding pins the equivalence
// that lets scoreContent skip the fold: a word without a separator matches the
// same bytes in folded and unfolded text.
func TestSearchDocuments_SimpleWordIgnoresSeparatorFolding(t *testing.T) {
	t.Parallel()
	base, localArch, globalArch := twoSourceFixture(t)
	var lead strings.Builder
	for i := range 40 {
		fmt.Fprintf(&lead, "Ενότητα %d.%d: βήμα-προς-βήμα, βλ. v1.2/beta_%d.\n", i, i+1, i)
	}
	writeFixtureDoc(t, localArch, "client.doc.md", "Πελάτης", lead.String()+"Εγκαταστήστε `@acme-id/sdk` πρώτα.\n")
	writeFixtureDoc(t, globalArch, "org-client.adr.md", "Shared Client", "## The sdk-first rule\n\nWe adopt acme-id-sdk.\n")
	resultsOf := func(args map[string]any) string {
		return string(rawEnvelope(t, searchText(t, callSearch(t, base, args)))["results"])
	}

	folding := resultsOf(map[string]any{"content": "SDK"})
	if literal := resultsOf(map[string]any{"content": "SDK", "match": "exact"}); folding != literal {
		t.Errorf("match=all returned\n%s\nmatch=exact returned\n%s\nwant identical rows for a word without a separator", folding, literal)
	}
	if rows := rawRows(t, `{"results":`+folding+`}`); len(rows) != 2 {
		t.Fatalf("got %d rows, want both documents, or the comparison proves nothing", len(rows))
	}

	writeFixtureDoc(t, localArch, "decoy.doc.md", "Decoy: a_b-c/d.e@f", "Only separators here: - _ / . @ :\n")
	if after := resultsOf(map[string]any{"content": "SDK"}); after != folding {
		t.Errorf("a document full of separators changed the rows of an unrelated query:\n%s\n%s", folding, after)
	}
}

func TestScoreContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		title       string
		slug        string
		body        string
		tokens      []string
		matchMode   contentMatchMode
		wantFound   bool
		wantRefs    []string
		wantSpecSum int
		wantFreq    int
	}{
		{"no tokens", "Needle", "needle", "needle", nil, matchModeAll, false, nil, 0, 0},
		{"occurrences are capped", "Guide", "guide", strings.Repeat("needle ", 50), []string{"needle"}, matchModeAll, true, []string{"needle"}, 1, contentFreqCap},
		{"all drops a document that lacks one token", "Guide", "guide", "needle", []string{"needle", "absent"}, matchModeAll, false, nil, 0, 0},
		{"any skips the absent token", "Guide", "guide", "needle", []string{"absent", "needle"}, matchModeAny, true, []string{"needle"}, 1, 1},
		{"any without a hit", "Guide", "guide", "body", []string{"absent", "missing"}, matchModeAny, false, nil, 0, 0},
		{"a slug hit adds no occurrence", "Guide", "needle-guide", "body", []string{"needle"}, matchModeExact, true, []string{"needle"}, 3, 0},
		{"a compound token meets three spellings", "Acme_ID SDK", "acme-id-sdk", "## Install\n\nuse @acme-id/sdk", []string{"acme id sdk"}, matchModeAll, true, []string{"acme id sdk"}, 3, 2},
		{"exact does not fold the text", "Acme_ID SDK", "guide", "use @acme-id/sdk", []string{"acme id sdk"}, matchModeExact, false, nil, 0, 0},
		{"every token adds its specificity and occurrences", "Needle Guide", "guide", "## Hay\n\nneedle hay", []string{"needle", "hay"}, matchModeAll, true, []string{"needle", "hay"}, 5, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matches, specSum, freq, found, _ := scoreContent(tt.title, tt.slug, tt.body, tt.tokens, tt.matchMode, false)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if refs := matchRefs(matches); !slices.Equal(refs, tt.wantRefs) && len(refs)+len(tt.wantRefs) > 0 {
				t.Errorf("match refs = %v, want %v", refs, tt.wantRefs)
			}
			if specSum != tt.wantSpecSum || freq != tt.wantFreq {
				t.Errorf("specificity sum = %d, occurrences = %d, want %d and %d: both feed the ranking score",
					specSum, freq, tt.wantSpecSum, tt.wantFreq)
			}
		})
	}
}
