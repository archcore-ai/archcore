package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

const testDoc = "---\ntitle: Original Title\nstatus: draft\n---\n\n## Context\nOriginal body."

func TestHandleUpdateDocument_TitleOnly(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/knowledge/my-adr.adr.md",
		"title": "New Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `title: "New Title"`) {
		t.Error("title not updated")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("status should be preserved")
	}
	if !strings.Contains(content, "Original body.") {
		t.Error("body should be preserved")
	}
}

func TestHandleUpdateDocument_StatusOnly(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   ".archcore/knowledge/my-adr.adr.md",
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `title: "Original Title"`) {
		t.Error("title should be preserved")
	}
	if !strings.Contains(content, "status: accepted") {
		t.Error("status not updated")
	}
	if !strings.Contains(content, "Original body.") {
		t.Error("body should be preserved")
	}
}

func TestHandleUpdateDocument_BrokenFrontmatterFails(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	brokenDoc := "---\ntitle: [broken\nstatus: draft\ntags:\n  - keep-me\n---\n\nBody must survive."
	writeDoc(t, base, "knowledge", "broken.adr.md", brokenDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   ".archcore/knowledge/broken.adr.md",
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for document with invalid frontmatter YAML")
	}
	msg := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(msg, "not valid YAML") {
		t.Errorf("error %q should mention invalid YAML", msg)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "broken.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != brokenDoc {
		t.Error("file must be byte-identical after a rejected update")
	}
}

func TestHandleUpdateDocument_ContentOnly(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":    ".archcore/knowledge/my-adr.adr.md",
		"content": "## Updated\nNew body here.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `title: "Original Title"`) {
		t.Error("title should be preserved")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("status should be preserved")
	}
	if !strings.Contains(content, "New body here.") {
		t.Error("content not updated")
	}
	if strings.Contains(content, "Original body.") {
		t.Error("old body should be replaced")
	}
}

func TestHandleUpdateDocument_AllFields(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":    ".archcore/knowledge/my-adr.adr.md",
		"title":   "All New",
		"status":  "accepted",
		"content": "Completely new content.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &info); err != nil {
		t.Fatal(err)
	}
	if info["title"] != "All New" {
		t.Errorf("title = %q, want %q", info["title"], "All New")
	}
	if info["status"] != "accepted" {
		t.Errorf("status = %q, want %q", info["status"], "accepted")
	}
	if info["category"] != "knowledge" {
		t.Errorf("category = %q, want %q", info["category"], "knowledge")
	}
	if info["type"] != "adr" {
		t.Errorf("type = %q, want %q", info["type"], "adr")
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `title: "All New"`) {
		t.Error("title not updated in file")
	}
	if !strings.Contains(content, "status: accepted") {
		t.Error("status not updated in file")
	}
	if !strings.Contains(content, "Completely new content.") {
		t.Error("content not updated in file")
	}
}

func TestHandleUpdateDocument_InvalidStatus(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   ".archcore/knowledge/my-adr.adr.md",
		"status": "proposed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid status")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "invalid status") {
		t.Errorf("error message = %q, want it to mention invalid status", text)
	}
}

func TestHandleUpdateDocument_PathTraversal(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   "../etc/passwd",
		"status": "hacked",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
	// Rejected by validateArchcorePath, not by an incidental filesystem miss.
	if got := resultText(t, result); !strings.Contains(got, "invalid path") {
		t.Errorf("error = %q, want a path-guard rejection (\"invalid path: ...\")", got)
	}
}

func TestHandleUpdateDocument_NonArchcorePath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   "src/main.go",
		"status": "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for non-.archcore path")
	}
}

func TestHandleUpdateDocument_MissingFile(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   ".archcore/knowledge/nonexistent.adr.md",
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing file")
	}
}

func TestHandleUpdateDocument_NoFieldsProvided(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error when no update fields provided")
	}
}

func TestHandleUpdateDocument_ContentWithFrontmatter(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	// Simulate an AI agent passing content that already includes frontmatter.
	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":    ".archcore/knowledge/my-adr.adr.md",
		"content": "---\ntitle: Ignored Title\nstatus: accepted\n---\n\n## Updated\nNew body here.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Must NOT contain duplicate frontmatter.
	if strings.Count(content, "---\n") > 2 {
		t.Errorf("duplicate frontmatter detected:\n%s", content)
	}
	// Original title/status should be preserved since only content was updated.
	if !strings.Contains(content, `title: "Original Title"`) {
		t.Error("title should be preserved when only content is updated")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("status should be preserved when only content is updated")
	}
	if !strings.Contains(content, "New body here.") {
		t.Error("body content not updated")
	}
}

func TestHandleUpdateDocument_CategoryDerivedFromType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	// Place a PRD (vision type) in a custom "auth" directory — category must be "vision", not "auth".
	writeDoc(t, base, "auth", "oauth.prd.md", "---\ntitle: OAuth PRD\nstatus: draft\n---\n\nBody.")

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/auth/oauth.prd.md",
		"title": "Updated OAuth PRD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &info); err != nil {
		t.Fatal(err)
	}
	if info["category"] != "vision" {
		t.Errorf("category = %q, want %q (should be derived from type, not directory)", info["category"], "vision")
	}
	if info["type"] != "prd" {
		t.Errorf("type = %q, want %q", info["type"], "prd")
	}
}

func TestHandleUpdateDocument_RootLevelDoc(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	// Doc at .archcore/ root (no subdirectory).
	writeDoc(t, base, "", "my-idea.idea.md", "---\ntitle: My Idea\nstatus: draft\n---\n\nBody.")

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":   ".archcore/my-idea.idea.md",
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &info); err != nil {
		t.Fatal(err)
	}
	if info["category"] != "vision" {
		t.Errorf("category = %q, want %q", info["category"], "vision")
	}
	if info["status"] != "accepted" {
		t.Errorf("status = %q, want %q", info["status"], "accepted")
	}
}

const testDocWithTags = "---\ntitle: Original Title\nstatus: draft\ntags:\n  - backend\n  - infra\n---\n\n## Context\nOriginal body."

func TestHandleUpdateDocument_AddTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
		"tags": []any{"frontend", "auth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "tags:\n  - \"auth\"\n  - \"frontend\"\n") {
		t.Errorf("expected tags block, got:\n%s", content)
	}
	if !strings.Contains(content, "Original body.") {
		t.Error("body should be preserved")
	}
}

func TestHandleUpdateDocument_ReplaceTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDocWithTags)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
		"tags": []any{"new-tag"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "tags:\n  - \"new-tag\"\n") {
		t.Errorf("expected new tags, got:\n%s", content)
	}
	if strings.Contains(content, "backend") || strings.Contains(content, "infra") {
		t.Error("old tags should be removed")
	}
}

func TestHandleUpdateDocument_ClearTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDocWithTags)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
		"tags": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "tags:") {
		t.Errorf("expected no tags block after clearing, got:\n%s", content)
	}
}

func TestHandleUpdateDocument_PreserveTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDocWithTags)

	// Update only the title, tags should be preserved.
	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/knowledge/my-adr.adr.md",
		"title": "New Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	data, err := os.ReadFile(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "tags:\n  - \"backend\"\n  - \"infra\"\n") {
		t.Errorf("expected existing tags to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, `title: "New Title"`) {
		t.Error("title should be updated")
	}
}

func TestHandleUpdateDocument_InvalidTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
		"tags": []any{"INVALID"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid tags")
	}
}

func TestHandleUpdateDocument_ReadPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	path := filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/knowledge/my-adr.adr.md",
		"title": "New Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for unreadable file")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "permission denied") {
		t.Errorf("message = %q, want permission-denied class", msg)
	}
}

func TestHandleUpdateDocument_TagsOnly(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", testDoc)

	// Provide only tags — should be accepted as a valid update.
	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
		"tags": []any{"new-tag"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &info); err != nil {
		t.Fatal(err)
	}
	tagsVal, ok := info["tags"]
	if !ok {
		t.Fatal("expected tags in response")
	}
	arr := tagsVal.([]any)
	if len(arr) != 1 || arr[0] != "new-tag" {
		t.Errorf("tags = %v, want [new-tag]", arr)
	}
}

const retainedFrontmatterDoc = `---
zeta: "false"
title: Original Title
nothing: null
status: draft
enabled: true
count: 17
fraction: 1.25
tags: [zebra, alpha]
items: [one, 2, false]
settings: &settings
  nested: {url: "https://example.test/a#b", values: [null, true]}
copy: *settings
multiline: |
  First line.
  Second line.
alpha: !!str 123
---

Original body.
`

func decodeFrontmatterValues(t *testing.T, data string) (map[string]any, []string, string) {
	t.Helper()
	parts := strings.SplitN(data, "---", 3)
	if len(parts) != 3 {
		t.Fatalf("no delimited frontmatter: %q", data)
	}
	var values map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &values); err != nil {
		t.Fatalf("decode persisted YAML: %v", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(parts[1]), &root); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for i := 0; i < len(root.Content[0].Content); i += 2 {
		key := root.Content[0].Content[i].Value
		if key != "title" && key != "status" && key != "tags" {
			keys = append(keys, key)
		}
	}
	return values, keys, strings.TrimSpace(parts[2])
}

func TestHandleUpdateDocument_PreservesUnknownFrontmatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       map[string]any
		wantTitle  string
		wantStatus string
		wantTags   []any
		wantBody   string
	}{
		{name: "body", args: map[string]any{"content": "Revised body."}, wantTitle: "Original Title", wantStatus: "draft", wantTags: []any{"zebra", "alpha"}, wantBody: "Revised body."},
		{name: "title", args: map[string]any{"title": "Revised Title"}, wantTitle: "Revised Title", wantStatus: "draft", wantTags: []any{"zebra", "alpha"}, wantBody: "Original body."},
		{name: "status", args: map[string]any{"status": "accepted"}, wantTitle: "Original Title", wantStatus: "accepted", wantTags: []any{"zebra", "alpha"}, wantBody: "Original body."},
		{name: "replace tags", args: map[string]any{"tags": []string{"new", "label"}}, wantTitle: "Original Title", wantStatus: "draft", wantTags: []any{"label", "new"}, wantBody: "Original body."},
		{name: "clear tags", args: map[string]any{"tags": []string{}}, wantTitle: "Original Title", wantStatus: "draft", wantBody: "Original body."},
		{name: "embedded metadata cannot replace retained values", args: map[string]any{"content": "---\nzeta: poisoned\n---\n\nRevised body."}, wantTitle: "Original Title", wantStatus: "draft", wantTags: []any{"zebra", "alpha"}, wantBody: "Revised body."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			writeDoc(t, base, "", "material.evidence.md", retainedFrontmatterDoc)
			want, wantKeys, _ := decodeFrontmatterValues(t, retainedFrontmatterDoc)
			tt.args["path"] = ".archcore/material.evidence.md"
			for pass := range 2 {
				result, err := callTool(HandleUpdateDocument(StaticRoot(base)), tt.args)
				if err != nil || result.IsError {
					t.Fatalf("update %d: result=%+v err=%v", pass, result, err)
				}
				data, err := os.ReadFile(filepath.Join(base, ".archcore", "material.evidence.md"))
				if err != nil {
					t.Fatal(err)
				}
				got, keys, body := decodeFrontmatterValues(t, string(data))
				if !reflect.DeepEqual(keys, wantKeys) {
					t.Errorf("retained order = %v, want %v", keys, wantKeys)
				}
				for _, key := range wantKeys {
					value, ok := got[key]
					if !ok || !reflect.DeepEqual(value, want[key]) {
						t.Errorf("retained %s = %#v (present %v), want %#v", key, value, ok, want[key])
					}
				}
				if got["title"] != tt.wantTitle || got["status"] != tt.wantStatus || body != tt.wantBody {
					t.Errorf("owned values = %#v / body %q", got, body)
				}
				if tt.wantTags == nil {
					if _, ok := got["tags"]; ok {
						t.Error("tags were not cleared")
					}
				} else if !reflect.DeepEqual(got["tags"], tt.wantTags) {
					t.Errorf("tags = %#v, want %#v", got["tags"], tt.wantTags)
				}
				if strings.Index(string(data), "zeta:") < strings.Index(string(data), "status:") {
					t.Error("extras precede owned fields")
				}
			}
		})
	}
}

func TestHandleUpdateDocument_UnpreservableFrontmatterUnchanged(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, input, want string }{
		{name: "alias to reconstructed owned field", input: "---\ntitle: &title Original\nstatus: draft\ncustom: *title\n---\n\nBody", want: "frontmatter cannot be preserved"},
		{name: "shadowed owned anchor", input: "---\nfirst: &name Earlier\ntitle: &name Later\ncustom: *name\n---\n\nBody", want: "frontmatter cannot be preserved"},
		{name: "preserved status needs YAML quoting", input: "---\ntitle: Original\nstatus: \"draft: bad\"\ncustom: retained\n---\n\nBody", want: "frontmatter cannot be preserved"},
		{name: "reconstructed alias error hides document content", input: "---\ntitle: Original\nstatus: \"*sensitiveAnchor\"\ncustom: retained\n---\n\nBody", want: "frontmatter cannot be preserved"},
		{name: "unknown anchor", input: "---\ntitle: Original\ncustom: *missing\n---\n\nBody", want: "not valid YAML"},
		{name: "duplicate unknown key", input: "---\ntitle: Original\ncustom: first\ncustom: second\n---\n\nBody", want: "not valid YAML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			writeDoc(t, base, "", "material.evidence.md", tt.input)
			result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{"path": ".archcore/material.evidence.md", "title": "Changed"})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Fatal("unsafe metadata rewrite succeeded")
			}
			msg := result.Content[0].(mcp.TextContent).Text
			if !strings.Contains(msg, tt.want) || !strings.Contains(msg, "manually") || strings.Contains(msg, base) || strings.Contains(msg, "sensitiveAnchor") || strings.Contains(msg, "repair YAML aliases") {
				t.Errorf("error = %q", msg)
			}
			data, err := os.ReadFile(filepath.Join(base, ".archcore", "material.evidence.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.input {
				t.Errorf("refusal changed original file: %q", data)
			}
		})
	}
}

func TestHandleUpdateDocument_MergedTagsCanBeCleared(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	input := "---\n<<: {title: Original, status: draft, tags: [old], custom: {nested: true}}\n---\n\nBody"
	writeDoc(t, base, "", "material.evidence.md", input)
	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{"path": ".archcore/material.evidence.md", "tags": []string{}})
	if err != nil || result.IsError {
		t.Fatalf("clear merged tags: %+v / %v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(base, ".archcore", "material.evidence.md"))
	if err != nil {
		t.Fatal(err)
	}
	values, _, _ := decodeFrontmatterValues(t, string(data))
	if tags, ok := values["tags"].([]any); !ok || len(tags) != 0 {
		t.Errorf("merged tags survived clearing: %#v", values["tags"])
	}
	if !reflect.DeepEqual(values["custom"], map[string]any{"nested": true}) {
		t.Errorf("merged metadata changed: %#v", values)
	}
}
