package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func callTool(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return handler(context.Background(), req)
}

func TestHandleListDocuments_Empty(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	result, err := callTool(HandleListDocuments(StaticRoot(base)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 0 {
		t.Errorf("expected 0, got %d", len(docs))
	}
}

func TestHandleListDocuments_AllDocs(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: accepted\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), nil)
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 2 {
		t.Errorf("expected 2, got %d", len(docs))
	}
}

func TestHandleListDocuments_FilterByType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"types": []any{"adr"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}
}

func TestHandleListDocuments_FilterByCategory(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"category": "vision",
	})
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Category != "vision" {
		t.Errorf("category = %q, want vision", docs[0].Category)
	}
}

func TestHandleListDocuments_FilterByStatus(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: accepted\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Status != "accepted" {
		t.Errorf("status = %q, want accepted", docs[0].Status)
	}
}

func TestHandleListDocuments_FilterByTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\ntags:\n  - frontend\n  - auth\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\ntags:\n  - backend\n---\n")
	writeDoc(t, base, "vision", "c.prd.md", "---\ntitle: C\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"tags": []any{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}

	// OR semantics: either tag matches.
	result2, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"tags": []any{"frontend", "backend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope2 listDocumentsResult
	if err := json.Unmarshal([]byte(result2.Content[0].(mcp.TextContent).Text), &envelope2); err != nil {
		t.Fatal(err)
	}
	docs2 := envelope2.Documents
	if len(docs2) != 2 {
		t.Fatalf("expected 2 (OR semantics), got %d", len(docs2))
	}
}

func TestHandleListDocuments_InvalidFilterTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"tags": []any{"INVALID"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid filter tags")
	}
}

func TestHandleListDocuments_ScanError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}
	base := t.TempDir()
	archcoreDir := filepath.Join(base, ".archcore")
	subDir := filepath.Join(archcoreDir, "noperm")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(subDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(subDir, 0o755) })

	result, err := callTool(HandleListDocuments(StaticRoot(base)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected tool error when scan fails due to permissions")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "scanning documents: permission denied") {
		t.Errorf("message = %q, want sanitized scan error", msg)
	}
}

func TestHandleListDocuments_TagsAndTypeFilter(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\ntags:\n  - frontend\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\ntags:\n  - frontend\n---\n")

	result, err := callTool(HandleListDocuments(StaticRoot(base)), map[string]any{
		"types": []any{"adr"},
		"tags":  []any{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var envelope listDocumentsResult
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	docs := envelope.Documents
	if len(docs) != 1 {
		t.Fatalf("expected 1 (type+tag filter), got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}
}

// TestHandleListDocuments_Pagination pins the limit/offset contract: default
// 100, cap 500 (silent clamp), 0/omitted → default, negative → error, offset
// past the end → empty page, truncated flag set exactly when more rows remain.
func TestHandleListDocuments_Pagination(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for i := range 7 {
		writeDoc(t, base, "knowledge", fmt.Sprintf("doc-%d.adr.md", i), "---\ntitle: D\nstatus: draft\n---\n")
	}

	decode := func(t *testing.T, result *mcp.CallToolResult) listDocumentsResult {
		t.Helper()
		var envelope listDocumentsResult
		if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope
	}

	tests := []struct {
		name          string
		args          map[string]any
		wantErr       bool
		wantReturned  int
		wantOffset    int
		wantTruncated bool
	}{
		{name: "default returns all under limit", args: nil, wantReturned: 7},
		{name: "limit truncates", args: map[string]any{"limit": 3}, wantReturned: 3, wantTruncated: true},
		{name: "offset pages", args: map[string]any{"limit": 3, "offset": 3}, wantReturned: 3, wantOffset: 3, wantTruncated: true},
		{name: "last page not truncated", args: map[string]any{"limit": 3, "offset": 6}, wantReturned: 1, wantOffset: 6},
		{name: "offset past end yields empty page", args: map[string]any{"offset": 100}, wantReturned: 0, wantOffset: 7},
		{name: "limit above cap is clamped", args: map[string]any{"limit": 100000}, wantReturned: 7},
		{name: "zero limit maps to default", args: map[string]any{"limit": 0}, wantReturned: 7},
		{name: "negative limit is an error", args: map[string]any{"limit": -1}, wantErr: true},
		{name: "negative offset is an error", args: map[string]any{"offset": -1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := callTool(HandleListDocuments(StaticRoot(base)), tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError != tt.wantErr {
				t.Fatalf("IsError = %v, want %v (%s)", result.IsError, tt.wantErr, resultText(t, result))
			}
			if tt.wantErr {
				return
			}
			envelope := decode(t, result)
			if envelope.Total != 7 {
				t.Errorf("total = %d, want 7", envelope.Total)
			}
			if envelope.Returned != tt.wantReturned || len(envelope.Documents) != tt.wantReturned {
				t.Errorf("returned = %d (docs %d), want %d", envelope.Returned, len(envelope.Documents), tt.wantReturned)
			}
			if envelope.Offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", envelope.Offset, tt.wantOffset)
			}
			if envelope.Truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", envelope.Truncated, tt.wantTruncated)
			}
		})
	}
}

// TestHandleListDocuments_EmptyDocumentsIsArray pins that "documents" is [] on
// an empty page, never null (matching the search_documents convention).
func TestHandleListDocuments_EmptyDocumentsIsArray(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	result, err := callTool(HandleListDocuments(StaticRoot(base)), nil)
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, `"documents":[]`) {
		t.Errorf("empty page must serialize documents as [], got: %s", text)
	}
}

func TestHandleListDocuments_FilterPassesOverAnEarlierMismatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args map[string]any
	}{
		{"types", map[string]any{"types": []any{"adr"}}},
		{"tags", map[string]any{"tags": []any{"frontend"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			writeDoc(t, base, "knowledge", "a.rfc.md", "---\ntitle: A\nstatus: draft\n---\n")
			writeDoc(t, base, "knowledge", "b.adr.md", "---\ntitle: B\nstatus: draft\ntags:\n  - frontend\n---\n")

			got := callList(t, base, tt.args)
			if len(got.Documents) != 1 || got.Documents[0].Filename != "b.adr.md" {
				t.Errorf("documents = %v, want only b.adr.md; a mismatch ended the scan", got.Documents)
			}
		})
	}
}

func TestFitListPage(t *testing.T) {
	t.Parallel()
	bySource := map[string]int{"local": 10}
	exact := make([]LocalDocument, 10)
	for i := range exact {
		exact[i] = LocalDocument{Path: fmt.Sprintf(".archcore/row-%d.doc.md", i), Title: strings.Repeat("t", 3000), SourceID: "local"}
	}
	exact[len(exact)-1].Title += strings.Repeat("t", listResponseByteBudget-measuredListBytes(t, bySource, exact))
	over := slices.Clone(exact)
	over[len(over)-1].Title += "t"
	oversized := []LocalDocument{
		{Path: ".archcore/big.doc.md", Title: strings.Repeat("t", listResponseByteBudget), SourceID: "local"},
		exact[0],
	}

	tests := []struct {
		name     string
		page     []LocalDocument
		wantRows int
	}{
		{"fits to the byte", exact, 10},
		{"one byte over drops the last row", over, 9},
		{"an oversized first row stays alone", oversized, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, err := fitListPage(tt.page, bySource)
			if err != nil {
				t.Fatal(err)
			}
			if len(page) != tt.wantRows {
				t.Errorf("page holds %d rows, want %d", len(page), tt.wantRows)
			}
		})
	}
}

func measuredListBytes(t *testing.T, bySource map[string]int, page []LocalDocument) int {
	t.Helper()
	head, err := json.Marshal(listDocumentsResult{BySource: bySource, Total: len(page), Offset: len(page), Returned: len(page), Documents: []LocalDocument{}})
	if err != nil {
		t.Fatal(err)
	}
	const counterSlack = 16
	used := len(head) + counterSlack
	for _, doc := range page {
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		used += len(data) + len(",")
	}
	return used
}

func TestInterleaveBySource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			"a first source with one document still leads",
			[]string{"local/1", "a/1", "a/2", "b/1", "b/2"},
			[]string{"local/1", "a/1", "b/1", "a/2", "b/2"},
		},
		{
			"ties go to the earlier source",
			[]string{"local/1", "local/2", "local/3", "a/1", "a/2", "a/3"},
			[]string{"local/1", "a/1", "local/2", "a/2", "local/3", "a/3"},
		},
		{
			"weights follow the remaining counts",
			[]string{"local/1", "local/2", "local/3", "local/4", "local/5", "a/1", "a/2"},
			[]string{"local/1", "a/1", "local/2", "local/3", "a/2", "local/4", "local/5"},
		},
		{
			"an exhausted source leaves the rotation",
			[]string{"local/1", "local/2", "a/1", "a/2", "b/1", "b/2", "b/3", "b/4", "b/5"},
			[]string{"local/1", "a/1", "b/1", "b/2", "local/2", "b/3", "b/4", "a/2", "b/5"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rows := make([]LocalDocument, len(tt.in))
			for i, path := range tt.in {
				source, _, _ := strings.Cut(path, "/")
				rows[i] = LocalDocument{Path: path, SourceID: source}
			}

			out := interleaveBySource(rows)

			got := make([]string, len(out))
			for i, row := range out {
				got[i] = row.Path
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("interleaveBySource = %v, want %v", got, tt.want)
			}
		})
	}
}
