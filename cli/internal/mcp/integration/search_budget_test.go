package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// hostPreviewBytes is the prefix one host showed the agent after it stored an
	// oversized result (read-tool-responses-survive-host-truncation.adr).
	hostPreviewBytes = 2048
	// searchByteBudget restates tools.searchResponseByteBudget, which this package
	// cannot import. The wire ceiling is the contract a host depends on.
	searchByteBudget = 40_000
)

type budgetedRow struct {
	Path          string `json:"path"`
	SourceID      string `json:"source_id"`
	Body          string `json:"body"`
	BodyTruncated bool   `json:"body_truncated"`
	BodyBytes     int    `json:"body_bytes"`
}

type budgetedSearch struct {
	Coverage  map[string]int `json:"coverage"`
	Hits      map[string]int `json:"hits"`
	Truncated bool           `json:"truncated"`
	Index     []struct {
		Path string `json:"path"`
	} `json:"index"`
	Results []budgetedRow `json:"results"`
}

func longMultiByteBody(opening string, size int) string {
	line := "Γραμμή εγγράφου χωρίς αντιστοιχίες, μόνο όγκος.\n"
	return opening + "\n\n" + strings.Repeat(line, size/len(line))
}

// TestSearchBudget_IncidentCallSurvivesAHostPreview replays the recorded call
// through the real server: documents written by create_document and a mounted
// global, read back by search_documents, then recovered by get_document.
func TestSearchBudget_IncidentCallSurvivesAHostPreview(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	base := filepath.Join(parent, "primary")
	writeFixtureFile(t, filepath.Join(base, ".archcore", "settings.json"),
		`{"sync":"none","globals":[{"id":"org","path":"../org-global/.archcore"}]}`)
	for i := range 5 {
		writeFixtureFile(t, filepath.Join(parent, "org-global", ".archcore", fmt.Sprintf("org-%d.doc.md", i)),
			fmt.Sprintf("---\ntitle: \"Org Document %d\"\nstatus: accepted\n---\n\n", i)+
				longMultiByteBody("The platform adopts acme-id-sdk.", 22_000))
	}
	c := newTestClient(t, base)

	localBody := longMultiByteBody("Install acme-id-sdk first. "+strings.Repeat("acme-id-sdk ", 8), 24_000)
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "doc", "filename": "packages", "title": "Auth Packages", "directory": "auth", "content": localBody,
	})
	mustCallTool(t, c, "create_document", map[string]any{
		"type": "doc", "filename": "unrelated", "title": "Unrelated", "content": "Nothing on the subject.\n",
	})

	res := mustCallTool(t, c, "search_documents", map[string]any{
		"content": "acme-id-sdk", "match": "exact", "mode": "full", "limit": 6,
	})
	text := firstText(res)

	if len(text) > searchByteBudget {
		t.Errorf("response is %d bytes, over the budget %d; the host would store it and show a preview", len(text), searchByteBudget)
	}
	if len(text) <= hostPreviewBytes {
		t.Fatalf("response is %d bytes, so the fixture does not reach past a host preview", len(text))
	}
	preview := text[:hostPreviewBytes]
	for _, want := range []string{`"coverage":{"local":2,"org":5}`, `"hits":{"local":1,"org":5}`, `org-4.doc.md`} {
		if !strings.Contains(preview, want) {
			t.Errorf("first %d bytes lack %s; an agent shown only the preview answers without the global source", hostPreviewBytes, want)
		}
	}

	got := decodeJSON[budgetedSearch](t, res)
	if len(got.Results) != 6 || got.Truncated || len(got.Index) != 6 {
		t.Fatalf("page holds %d rows of %d, truncated=%v; want all six rows with shortened bodies",
			len(got.Results), len(got.Index), got.Truncated)
	}
	top := got.Results[0]
	if top.SourceID != "local" || !top.BodyTruncated {
		t.Fatalf("top row = %s (source %s, body_truncated=%v); want the local document with a shortened body",
			top.Path, top.SourceID, top.BodyTruncated)
	}
	if !strings.HasPrefix(localBody, top.Body) || top.BodyBytes <= len(top.Body) {
		t.Errorf("shortened body of %d bytes (body_bytes=%d) is not a marked prefix of the document", len(top.Body), top.BodyBytes)
	}

	for _, row := range got.Results {
		if !row.BodyTruncated {
			continue
		}
		doc := decodeJSON[struct {
			Content string `json:"content"`
		}](t, mustCallTool(t, c, "get_document", map[string]any{"path": row.Path}))
		if !strings.Contains(doc.Content, row.Body) {
			t.Errorf("%s: get_document does not hold the shortened body", row.Path)
		}
		if len(doc.Content) < row.BodyBytes || len(doc.Content) <= len(row.Body) {
			t.Errorf("%s: get_document returned %d bytes for body_bytes=%d and a shortened body of %d; the recovery path must return the rest",
				row.Path, len(doc.Content), row.BodyBytes, len(row.Body))
		}
	}
}
