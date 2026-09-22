package tools

// Tests for the list_documents envelope order and byte budget
// (list-documents.spec, read-tool-responses-survive-host-truncation.adr).

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func listText(t *testing.T, base string, args map[string]any) string {
	t.Helper()
	res, err := HandleListDocuments(StaticRoot(base))(t.Context(), reqWith(args))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	return res.Content[0].(mcp.TextContent).Text
}

func TestHandleListDocuments_EnvelopeLeadsWithCounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		nLocal  int
		nGlobal int
		want    string
	}{
		{"two sources", 120, 30, `{"by_source":{"local":120,"org":30},"total":150,"offset":0,"returned":`},
		{"no documents", 0, 0, `{"by_source":{},"total":0,"offset":0,"returned":0,"truncated":false,"documents":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			text := listText(t, listFixture(t, tt.nLocal, tt.nGlobal), nil)
			if !strings.HasPrefix(text, tt.want) {
				t.Errorf("response starts %.120q, want %q; a host preview must show what each source holds before any row",
					text, tt.want)
			}
		})
	}
}

func TestHandleListDocuments_ResponseStaysUnderByteBudget(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 600, 200)

	for _, limit := range []float64{0, 100, 500} {
		text := listText(t, base, map[string]any{"limit": limit})
		if len(text) > listResponseByteBudget {
			t.Errorf("limit %v: response is %d bytes, over the budget %d", limit, len(text), listResponseByteBudget)
		}
	}

	got := callList(t, base, map[string]any{"limit": float64(500)})
	if got.Returned >= 500 || got.Returned != len(got.Documents) || !got.Truncated {
		t.Fatalf("returned=%d rows=%d truncated=%v; want a page the budget shortened and marked",
			got.Returned, len(got.Documents), got.Truncated)
	}
	globals := 0
	for _, d := range got.Documents {
		if d.Global {
			globals++
		}
	}
	if globals == 0 {
		t.Error("the shortened page holds no global row; the budget cut must keep the interleave")
	}
}

// The next offset is offset+returned with or without a budget cut, so a caller
// that pages never skips and never repeats a document.
func TestHandleListDocuments_BudgetCutKeepsPagingExact(t *testing.T) {
	t.Parallel()
	base := listFixture(t, 300, 100)
	seen := make(map[string]bool)
	offset, calls := 0, 0
	for {
		got := callList(t, base, map[string]any{"limit": float64(500), "offset": float64(offset)})
		calls++
		if got.Returned == 0 && got.Truncated {
			t.Fatal("an empty page reports truncated, so a pager would loop forever")
		}
		for _, d := range got.Documents {
			if seen[d.Path] {
				t.Fatalf("%s arrived twice", d.Path)
			}
			seen[d.Path] = true
		}
		offset += got.Returned
		if !got.Truncated {
			break
		}
	}
	if len(seen) != 400 {
		t.Errorf("paging reached %d documents in %d calls, want all 400", len(seen), calls)
	}
	if calls < 2 {
		t.Errorf("paging took %d call; the fixture must exceed one budget page to prove the cut", calls)
	}
}
