package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"archcore-cli/templates"
)

func setupTestArchcore(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	return base
}

func writeDoc(t *testing.T, base, subdir, filename, content string) {
	t.Helper()
	dir := filepath.Join(base, ".archcore", subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanDocuments_Empty(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestScanDocuments_FindsDocs(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "use-postgres.adr.md", "---\ntitle: Use PostgreSQL\nstatus: accepted\n---\n\nbody")
	writeDoc(t, base, "vision", "my-plan.plan.md", "---\ntitle: My Plan\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}

	// Find the ADR.
	var adr *LocalDocument
	for i := range docs {
		if docs[i].Type == "adr" {
			adr = &docs[i]
		}
	}
	if adr == nil {
		t.Fatal("adr not found")
	}
	if adr.Title != "Use PostgreSQL" {
		t.Errorf("title = %q, want %q", adr.Title, "Use PostgreSQL")
	}
	if adr.Status != "accepted" {
		t.Errorf("status = %q, want %q", adr.Status, "accepted")
	}
	if adr.Slug != "use-postgres" {
		t.Errorf("slug = %q, want %q", adr.Slug, "use-postgres")
	}
	// Category is virtual — derived from type, not directory.
	if adr.Category != "knowledge" {
		t.Errorf("category = %q, want %q", adr.Category, "knowledge")
	}
	if adr.Content != "" {
		t.Error("content should be empty in scan results")
	}
}

func TestScanDocuments_CustomDirectory(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "auth", "jwt-strategy.adr.md", "---\ntitle: JWT\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "payments", "stripe.prd.md", "---\ntitle: Stripe\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}

	// Find the ADR — should have virtual category "knowledge".
	for _, doc := range docs {
		if doc.Type == "adr" {
			if doc.Category != "knowledge" {
				t.Errorf("adr category = %q, want %q", doc.Category, "knowledge")
			}
			if doc.Path != ".archcore/auth/jwt-strategy.adr.md" {
				t.Errorf("adr path = %q", doc.Path)
			}
		}
		if doc.Type == "prd" {
			if doc.Category != "vision" {
				t.Errorf("prd category = %q, want %q", doc.Category, "vision")
			}
		}
	}
}

func TestScanDocuments_NestedDirectory(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "infrastructure/k8s", "migration.adr.md", "---\ntitle: K8s Migration\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Path != ".archcore/infrastructure/k8s/migration.adr.md" {
		t.Errorf("path = %q", docs[0].Path)
	}
}

func TestScanDocuments_SkipsNonMd(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "readme.txt", "not a doc")
	writeDoc(t, base, "knowledge", "real.rfc.md", "---\ntitle: Real\nstatus: draft\n---\n")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
}

func TestScanDocuments_NoArchcoreDir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestReadDocumentContent(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	content := "---\ntitle: My ADR\nstatus: accepted\n---\n\n## Context\nSome context."
	writeDoc(t, base, "knowledge", "my-adr.adr.md", content)

	doc, err := ReadDocumentContent(base, ".archcore/knowledge/my-adr.adr.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "My ADR" {
		t.Errorf("title = %q, want %q", doc.Title, "My ADR")
	}
	if doc.Status != "accepted" {
		t.Errorf("status = %q, want %q", doc.Status, "accepted")
	}
	if doc.Content != content {
		t.Errorf("content mismatch")
	}
	if doc.Type != "adr" {
		t.Errorf("type = %q, want %q", doc.Type, "adr")
	}
	if doc.Category != "knowledge" {
		t.Errorf("category = %q, want %q", doc.Category, "knowledge")
	}
}

func TestReadDocumentContent_NotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	_, err := ReadDocumentContent(base, ".archcore/knowledge/nonexistent.adr.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestScanDocuments_FilesInRoot(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "my-doc.adr.md", "---\ntitle: Root Doc\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Path != ".archcore/my-doc.adr.md" {
		t.Errorf("path = %q, want %q", docs[0].Path, ".archcore/my-doc.adr.md")
	}
	if docs[0].Category != "knowledge" {
		t.Errorf("category = %q, want %q", docs[0].Category, "knowledge")
	}
	if docs[0].Slug != "my-doc" {
		t.Errorf("slug = %q, want %q", docs[0].Slug, "my-doc")
	}
}

func TestScanDocuments_SkipsHiddenDirectories(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, ".git", "config.adr.md", "---\ntitle: Hidden\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "visible", "real.adr.md", "---\ntitle: Real\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (hidden dir skipped), got %d", len(docs))
	}
	if docs[0].Slug != "real" {
		t.Errorf("slug = %q, want %q", docs[0].Slug, "real")
	}
}

func TestScanDocuments_SkipsMetaFiles(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// Write meta files that should be ignored.
	if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(`{"sync":"none"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".archcore", ".sync-state.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDoc(t, base, "", "real.adr.md", "---\ntitle: Real\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (meta files skipped), got %d", len(docs))
	}
}

func TestScanDocuments_UnknownDocType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// File with no type segment -- ExtractDocType returns "".
	writeDoc(t, base, "", "readme.md", "---\ntitle: Readme\nstatus: draft\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Type != "" {
		t.Errorf("type = %q, want empty string", docs[0].Type)
	}
	// Unknown type defaults to "knowledge" via CategoryForType.
	if docs[0].Category != "knowledge" {
		t.Errorf("category = %q, want %q", docs[0].Category, "knowledge")
	}
}

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no frontmatter",
			input: "## Context\nSome body.",
			want:  "## Context\nSome body.",
		},
		{
			name:  "with frontmatter",
			input: "---\ntitle: My Title\nstatus: draft\n---\n\n## Context\nSome body.",
			want:  "## Context\nSome body.",
		},
		{
			name:  "frontmatter only no body",
			input: "---\ntitle: My Title\nstatus: draft\n---",
			want:  "",
		},
		{
			name:  "frontmatter with trailing newline",
			input: "---\ntitle: My Title\nstatus: draft\n---\n",
			want:  "",
		},
		{
			name:  "windows line endings",
			input: "---\r\ntitle: My Title\r\nstatus: draft\r\n---\r\n\r\n## Context\r\nBody.",
			want:  "## Context\nBody.",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "dashes in body only",
			input: "Some text\n---\nMore text",
			want:  "Some text\n---\nMore text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripFrontmatter(tt.input)
			if got != tt.want {
				t.Errorf("stripFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDocType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"use-postgres.adr.md", "adr"},
		{"my-rfc.rfc.md", "rfc"},
		{"simple.md", ""},
		{"multi.part.rule.md", "rule"},
		{"", ""},
		{".md", ""},
		{"no-extension", ""},
		{"dots.in.slug.adr.md", "adr"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := templates.ExtractDocType(tt.input)
			if got != tt.want {
				t.Errorf("ExtractDocType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"use-postgres.adr.md", "use-postgres"},
		{"my-rfc.rfc.md", "my-rfc"},
		{"simple.md", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := templates.ExtractSlug(tt.input)
			if got != tt.want {
				t.Errorf("ExtractSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tags    []string
		wantErr bool
		errHint string
	}{
		{name: "valid tags", tags: []string{"frontend", "auth", "team-platform"}, wantErr: false},
		{name: "colon separator", tags: []string{"team:payments"}, wantErr: false},
		{name: "pipe separator", tags: []string{"some|flag"}, wantErr: false},
		{name: "underscore", tags: []string{"snake_case"}, wantErr: false},
		{name: "uppercase with underscore hint", tags: []string{"Payment_Team"}, wantErr: true, errHint: "payment_team"},
		{name: "complex tag", tags: []string{"team:frontend|web"}, wantErr: false},
		{name: "uppercase hint", tags: []string{"Frontend"}, wantErr: true, errHint: "did you mean"},
		{name: "digit start", tags: []string{"1invalid"}, wantErr: true},
		{name: "single char", tags: []string{"a"}, wantErr: false},
		{name: "hyphen only", tags: []string{"-"}, wantErr: true},
		{name: "empty tag", tags: []string{""}, wantErr: true},
		{name: "space in tag", tags: []string{"has space"}, wantErr: true},
		{name: "uppercase letters", tags: []string{"UPPER"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateTags(tt.tags)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for tags %v", tt.tags)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.errHint != "" && err != nil && !strings.Contains(err.Error(), tt.errHint) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errHint)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{name: "sort", tags: []string{"z", "a", "m"}, want: []string{"a", "m", "z"}},
		{name: "dedup", tags: []string{"a", "b", "a"}, want: []string{"a", "b"}},
		{name: "nil input", tags: nil, want: nil},
		{name: "empty input", tags: []string{}, want: nil},
		{name: "single", tags: []string{"x"}, want: []string{"x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeTags(tt.tags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeTags(%v) = %v, want %v", tt.tags, got, tt.want)
			}
		})
	}
}

func TestNormalizeTags_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	input := []string{"z", "a", "m"}
	original := []string{"z", "a", "m"}
	_ = normalizeTags(input)
	if !reflect.DeepEqual(input, original) {
		t.Errorf("normalizeTags mutated input: got %v, want %v", input, original)
	}
}

func TestBuildDocumentFile_WithTags(t *testing.T) {
	t.Parallel()
	result := buildDocumentFile("Test Title", "draft", []string{"auth", "frontend"}, "## Body\nContent.")
	if !strings.Contains(result, "tags:\n  - \"auth\"\n  - \"frontend\"\n") {
		t.Errorf("expected YAML tags block, got:\n%s", result)
	}
	if !strings.Contains(result, `title: "Test Title"`) {
		t.Error("missing title")
	}
	if !strings.Contains(result, "status: draft") {
		t.Error("missing status")
	}
	if !strings.Contains(result, "## Body\nContent.") {
		t.Error("missing body")
	}

	// No tags case.
	noTags := buildDocumentFile("T", "draft", nil, "Body")
	if strings.Contains(noTags, "tags:") {
		t.Error("should not contain tags block when tags is nil")
	}
}

func TestScanDocuments_WithTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "tagged.adr.md", "---\ntitle: Tagged\nstatus: draft\ntags:\n  - auth\n  - backend\n---\n\nbody")

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if !reflect.DeepEqual(docs[0].Tags, []string{"auth", "backend"}) {
		t.Errorf("tags = %v, want [auth backend]", docs[0].Tags)
	}
}

func TestReadDocumentContent_WithTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "tagged.adr.md", "---\ntitle: Tagged\nstatus: draft\ntags:\n  - frontend\n---\n\nbody")

	doc, err := ReadDocumentContent(base, ".archcore/knowledge/tagged.adr.md")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc.Tags, []string{"frontend"}) {
		t.Errorf("tags = %v, want [frontend]", doc.Tags)
	}
}

func TestValidateArchcorePath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
		errMatch string
	}{
		{name: "forward slashes", input: ".archcore/features/lfrom/x.doc.md", want: ".archcore/features/lfrom/x.doc.md"},
		{name: "redundant segments cleaned", input: ".archcore/a/./b/../c.md", want: ".archcore/a/c.md"},
		// Regression: filepath.Clean on Windows re-introduces backslashes, which
		// broke the second prefix check. The cleaned output must always use
		// forward slashes, regardless of platform.
		{name: "output uses forward slashes", input: ".archcore/a/b.md", want: ".archcore/a/b.md"},
		{name: "missing prefix", input: "features/lfrom/x.doc.md", wantErr: true, errMatch: "must start with"},
		{name: "traversal escape", input: ".archcore/../etc/passwd", wantErr: true, errMatch: "must be relative"},
		{name: "absolute unix", input: "/etc/passwd", wantErr: true, errMatch: "must be relative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateArchcorePath(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%q)", got)
				}
				if tc.errMatch != "" && !strings.Contains(err.Error(), tc.errMatch) {
					t.Errorf("error = %q, want substring %q", err.Error(), tc.errMatch)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
