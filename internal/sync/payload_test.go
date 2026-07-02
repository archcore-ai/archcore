package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/templates"
)

func TestParseFrontmatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		content    string
		wantTitle  string
		wantStatus templates.DocStatus
		wantTags   []string
	}{
		{
			name:       "full frontmatter",
			content:    "---\ntitle: My Document\nstatus: accepted\n---\n\n# Body",
			wantTitle:  "My Document",
			wantStatus: "accepted",
		},
		{
			name:       "title only",
			content:    "---\ntitle: Only Title\n---\n\nBody text",
			wantTitle:  "Only Title",
			wantStatus: "",
		},
		{
			name:       "no frontmatter",
			content:    "# Just Markdown",
			wantTitle:  "",
			wantStatus: "",
		},
		{
			name:       "unclosed frontmatter",
			content:    "---\ntitle: Broken\n",
			wantTitle:  "",
			wantStatus: "",
		},
		{
			name:       "windows line endings",
			content:    "---\r\ntitle: Win Doc\r\nstatus: draft\r\n---\r\n\r\nBody",
			wantTitle:  "Win Doc",
			wantStatus: "draft",
		},
		{
			name:       "empty content",
			content:    "",
			wantTitle:  "",
			wantStatus: "",
		},
		{
			name:       "with tags",
			content:    "---\ntitle: Tagged\nstatus: draft\ntags:\n  - api\n  - backend\n---\n\nBody",
			wantTitle:  "Tagged",
			wantStatus: "draft",
			wantTags:   []string{"api", "backend"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, status, tags := ParseFrontmatter(tt.content)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if len(tt.wantTags) == 0 {
				if len(tags) != 0 {
					t.Errorf("tags = %v, want empty", tags)
				}
			} else {
				if fmt.Sprintf("%v", tags) != fmt.Sprintf("%v", tt.wantTags) {
					t.Errorf("tags = %v, want %v", tags, tt.wantTags)
				}
			}
		})
	}
}

func TestBuildPayload(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "vision")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: ADR 001\nstatus: draft\n---\n\n# ADR 001"
	if err := os.WriteFile(filepath.Join(archDir, "adr-001.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "vision/adr-001.md", Action: ActionCreated, Hash: "abc"},
		{RelPath: "knowledge/removed.md", Action: ActionDeleted},
		{RelPath: "vision/unchanged.md", Action: ActionUnchanged, Hash: "xyz"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	// Created: 1, Deleted: 1, unchanged skipped.
	if len(payload.Created) != 1 {
		t.Fatalf("got %d created, want 1", len(payload.Created))
	}
	if len(payload.Modified) != 0 {
		t.Fatalf("got %d modified, want 0", len(payload.Modified))
	}
	if len(payload.Deleted) != 1 {
		t.Fatalf("got %d deleted, want 1", len(payload.Deleted))
	}

	// Created file should have content and frontmatter.
	created := payload.Created[0]
	if created.Content != content {
		t.Errorf("created content = %q, want %q", created.Content, content)
	}
	if created.SHA256 != "abc" {
		t.Errorf("sha256 = %q, want %q", created.SHA256, "abc")
	}
	if created.Frontmatter.Title != "ADR 001" {
		t.Errorf("frontmatter title = %q, want %q", created.Frontmatter.Title, "ADR 001")
	}
	if created.Frontmatter.Status != "draft" {
		t.Errorf("frontmatter status = %q, want %q", created.Frontmatter.Status, "draft")
	}

	// Deleted should be just the path.
	if payload.Deleted[0] != "knowledge/removed.md" {
		t.Errorf("deleted path = %q, want %q", payload.Deleted[0], "knowledge/removed.md")
	}
}

func TestBuildPayload_Modified(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "vision")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archDir, "doc.md"), []byte("# Doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "vision/doc.md", Action: ActionModified, Hash: "def"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	if len(payload.Modified) != 1 {
		t.Fatalf("got %d modified, want 1", len(payload.Modified))
	}
	if payload.Modified[0].Path != "vision/doc.md" {
		t.Errorf("path = %q, want %q", payload.Modified[0].Path, "vision/doc.md")
	}
}

func TestBuildPayload_MissingFile(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "vision/nonexistent.md", Action: ActionCreated, Hash: "abc"},
	}

	_, err := BuildPayload(baseDir, entries)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBuildPayload_EmptyEntries(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	payload, err := BuildPayload(baseDir, nil)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}
	if len(payload.Created) != 0 || len(payload.Modified) != 0 || len(payload.Deleted) != 0 {
		t.Error("expected empty arrays for nil entries")
	}
}

func TestBuildPayload_AllUnchanged(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	entries := []DiffEntry{
		{RelPath: "vision/a.md", Action: ActionUnchanged, Hash: "aaa"},
		{RelPath: "vision/b.md", Action: ActionUnchanged, Hash: "bbb"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}
	if len(payload.Created) != 0 || len(payload.Modified) != 0 || len(payload.Deleted) != 0 {
		t.Error("expected empty arrays for all unchanged")
	}
}

func TestBuildPayload_InvalidStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  string
		action  DiffAction
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid status on created file",
			status:  "active",
			action:  ActionCreated,
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name:    "invalid status on modified file",
			status:  "archived",
			action:  ActionModified,
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name:    "valid status draft",
			status:  "draft",
			action:  ActionCreated,
			wantErr: false,
		},
		{
			name:    "valid status accepted",
			status:  "accepted",
			action:  ActionModified,
			wantErr: false,
		},
		{
			name:    "valid status rejected",
			status:  "rejected",
			action:  ActionCreated,
			wantErr: false,
		},
		{
			name:    "empty status is allowed",
			status:  "",
			action:  ActionCreated,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			baseDir := t.TempDir()
			archDir := filepath.Join(baseDir, ".archcore", "knowledge")
			if err := os.MkdirAll(archDir, 0o755); err != nil {
				t.Fatal(err)
			}

			var content string
			if tt.status != "" {
				content = fmt.Sprintf("---\ntitle: Test Doc\nstatus: %s\n---\n\n# Body", tt.status)
			} else {
				content = "---\ntitle: Test Doc\n---\n\n# Body"
			}
			if err := os.WriteFile(filepath.Join(archDir, "test.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			entries := []DiffEntry{
				{RelPath: "knowledge/test.md", Action: tt.action, Hash: "abc123"},
			}

			_, err := BuildPayload(baseDir, entries)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error for invalid status, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.status) {
					t.Errorf("error = %q, want it to contain status %q", err.Error(), tt.status)
				}
				if !strings.Contains(err.Error(), "knowledge/test.md") {
					t.Errorf("error = %q, want it to contain file path", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildPayload_InvalidFrontmatterYAML(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "knowledge")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: [broken\nstatus: draft\n---\n\n# Body"
	if err := os.WriteFile(filepath.Join(archDir, "test.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "knowledge/test.md", Action: ActionCreated, Hash: "abc123"},
	}

	_, err := BuildPayload(baseDir, entries)
	if err == nil {
		t.Fatal("expected error for invalid frontmatter YAML, got nil")
	}
	if !strings.Contains(err.Error(), "invalid frontmatter YAML") {
		t.Errorf("error = %q, want it to mention invalid frontmatter YAML", err.Error())
	}
	if !strings.Contains(err.Error(), "knowledge/test.md") {
		t.Errorf("error = %q, want it to contain the relative file path", err.Error())
	}
}

func TestBuildPayload_PathTraversal(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	entries := []DiffEntry{
		{RelPath: "../../etc/passwd", Action: ActionCreated, Hash: "abc"},
	}

	_, err := BuildPayload(baseDir, entries)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("error = %q, should mention invalid path", err.Error())
	}
}

func TestBuildPayload_DocTypeAndCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		relPath      string
		wantDocType  string
		wantCategory templates.Category
	}{
		{
			name:         "adr in custom dir",
			relPath:      "auth/jwt-strategy.adr.md",
			wantDocType:  "adr",
			wantCategory: "knowledge",
		},
		{
			name:         "prd at root",
			relPath:      "product-launch.prd.md",
			wantDocType:  "prd",
			wantCategory: "vision",
		},
		{
			name:         "cpat in subdirectory",
			relPath:      "incidents/outage.cpat.md",
			wantDocType:  "cpat",
			wantCategory: "experience",
		},
		{
			name:         "task-type at root",
			relPath:      "code-review.task-type.md",
			wantDocType:  "task-type",
			wantCategory: "experience",
		},
		{
			name:         "plan in vision dir",
			relPath:      "vision/q1-plan.plan.md",
			wantDocType:  "plan",
			wantCategory: "vision",
		},
		{
			name:         "unknown type defaults to knowledge",
			relPath:      "notes/readme.md",
			wantDocType:  "",
			wantCategory: "knowledge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			absDir := filepath.Join(baseDir, ".archcore", filepath.Dir(tt.relPath))
			if err := os.MkdirAll(absDir, 0o755); err != nil {
				t.Fatal(err)
			}

			content := "---\ntitle: Test\nstatus: draft\n---\n\n# Body"
			if err := os.WriteFile(filepath.Join(baseDir, ".archcore", tt.relPath), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			entries := []DiffEntry{
				{RelPath: tt.relPath, Action: ActionCreated, Hash: "abc123"},
			}

			payload, err := BuildPayload(baseDir, entries)
			if err != nil {
				t.Fatalf("BuildPayload: %v", err)
			}

			if len(payload.Created) != 1 {
				t.Fatalf("got %d created, want 1", len(payload.Created))
			}

			created := payload.Created[0]
			if created.DocType != tt.wantDocType {
				t.Errorf("DocType = %q, want %q", created.DocType, tt.wantDocType)
			}
			if created.Category != tt.wantCategory {
				t.Errorf("Category = %q, want %q", created.Category, tt.wantCategory)
			}
		})
	}
}

func TestBuildPayload_ModifiedDocTypeAndCategory(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "auth")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: JWT Strategy\nstatus: accepted\n---\n\n# JWT"
	if err := os.WriteFile(filepath.Join(archDir, "jwt.adr.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "auth/jwt.adr.md", Action: ActionModified, Hash: "def456"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	if len(payload.Modified) != 1 {
		t.Fatalf("got %d modified, want 1", len(payload.Modified))
	}

	modified := payload.Modified[0]
	if modified.DocType != "adr" {
		t.Errorf("DocType = %q, want %q", modified.DocType, "adr")
	}
	if modified.Category != "knowledge" {
		t.Errorf("Category = %q, want %q", modified.Category, "knowledge")
	}
}

func TestBuildPayload_FileAtRoot(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: Root Doc\nstatus: draft\n---\n\n# Root"
	if err := os.WriteFile(filepath.Join(baseDir, ".archcore", "root-doc.adr.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "root-doc.adr.md", Action: ActionCreated, Hash: "abc"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	if len(payload.Created) != 1 {
		t.Fatalf("got %d created, want 1", len(payload.Created))
	}
	if payload.Created[0].Path != "root-doc.adr.md" {
		t.Errorf("path = %q, want %q", payload.Created[0].Path, "root-doc.adr.md")
	}
	if payload.Created[0].DocType != "adr" {
		t.Errorf("DocType = %q, want %q", payload.Created[0].DocType, "adr")
	}
	if payload.Created[0].Category != "knowledge" {
		t.Errorf("Category = %q, want %q", payload.Created[0].Category, "knowledge")
	}
}

func TestParseFrontmatter_WithTags(t *testing.T) {
	t.Parallel()
	content := "---\ntitle: Tagged\nstatus: draft\ntags:\n  - frontend\n  - auth\n---\n\nBody"
	title, status, tags := ParseFrontmatter(content)
	if title != "Tagged" {
		t.Errorf("title = %q, want %q", title, "Tagged")
	}
	if status != "draft" {
		t.Errorf("status = %q, want %q", status, "draft")
	}
	if len(tags) != 2 || tags[0] != "frontend" || tags[1] != "auth" {
		t.Errorf("tags = %v, want [frontend auth]", tags)
	}
}

func TestParseFrontmatter_WithoutTags(t *testing.T) {
	t.Parallel()
	content := "---\ntitle: No Tags\nstatus: accepted\n---\n\nBody"
	title, status, tags := ParseFrontmatter(content)
	if title != "No Tags" {
		t.Errorf("title = %q, want %q", title, "No Tags")
	}
	if status != "accepted" {
		t.Errorf("status = %q, want %q", status, "accepted")
	}
	if tags != nil {
		t.Errorf("tags = %v, want nil", tags)
	}
}

func TestBuildPayload_WithTags(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "knowledge")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: Tagged Doc\nstatus: draft\ntags:\n  - auth\n  - backend\n---\n\n# Body"
	if err := os.WriteFile(filepath.Join(archDir, "tagged.adr.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "knowledge/tagged.adr.md", Action: ActionCreated, Hash: "abc"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	if len(payload.Created) != 1 {
		t.Fatalf("got %d created, want 1", len(payload.Created))
	}
	created := payload.Created[0]
	if len(created.Frontmatter.Tags) != 2 {
		t.Fatalf("tags = %v, want 2 tags", created.Frontmatter.Tags)
	}
	if created.Frontmatter.Tags[0] != "auth" || created.Frontmatter.Tags[1] != "backend" {
		t.Errorf("tags = %v, want [auth backend]", created.Frontmatter.Tags)
	}
}

func TestBuildPayload_WithoutTags(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	archDir := filepath.Join(baseDir, ".archcore", "knowledge")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: No Tags\nstatus: draft\n---\n\n# Body"
	if err := os.WriteFile(filepath.Join(archDir, "notags.adr.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DiffEntry{
		{RelPath: "knowledge/notags.adr.md", Action: ActionCreated, Hash: "abc"},
	}

	payload, err := BuildPayload(baseDir, entries)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	if len(payload.Created) != 1 {
		t.Fatalf("got %d created, want 1", len(payload.Created))
	}
	if payload.Created[0].Frontmatter.Tags != nil {
		t.Errorf("tags = %v, want nil", payload.Created[0].Frontmatter.Tags)
	}
}
