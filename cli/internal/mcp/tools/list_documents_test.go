package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
