package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testHash1 = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	testHash2 = "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5"
)

func TestLoadManifest(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, baseDir string)
		wantFiles int
		wantErr   bool
	}{
		{
			name: "no manifest file returns empty",
			setup: func(t *testing.T, baseDir string) {
				os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)
			},
			wantFiles: 0,
		},
		{
			name: "valid manifest with two files",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				data := `{"version":1,"files":{
					"vision/adr-001.md":"` + testHash1 + `",
					"knowledge/rfc-001.md":"` + testHash2 + `"
				}}`
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(data), 0o644)
			},
			wantFiles: 2,
		},
		{
			name: "corrupted JSON returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte("{invalid"), 0o644)
			},
			wantErr: true,
		},
		{
			name: "null files field returns empty map",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{"version":1}`), 0o644)
			},
			wantFiles: 0,
		},
		{
			name: "unsupported version returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{"version":99,"files":{}}`), 0o644)
			},
			wantErr: true,
		},
		{
			name: "empty file returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(""), 0o644)
			},
			wantErr: true,
		},
		{
			name: "truncated JSON returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{"version":1,"files":{`), 0o644)
			},
			wantErr: true,
		},
		{
			name: "invalid hash returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				data := `{"version":1,"files":{"vision/test.md":"short"}}`
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(data), 0o644)
			},
			wantErr: true,
		},
		{
			name: "unknown root field returns error",
			setup: func(t *testing.T, baseDir string) {
				dir := filepath.Join(baseDir, ".archcore")
				os.MkdirAll(dir, 0o755)
				data := `{"version":1,"files":{},"extra":"bad"}`
				os.WriteFile(filepath.Join(dir, ManifestFile), []byte(data), 0o644)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			tt.setup(t, baseDir)
			m, err := LoadManifest(baseDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadManifest error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(m.Files) != tt.wantFiles {
				t.Errorf("got %d files, want %d", len(m.Files), tt.wantFiles)
			}
		})
	}
}

func TestSaveManifest_Roundtrip(t *testing.T) {
	baseDir := t.TempDir()
	os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)

	m := NewManifest()
	m.Files["vision/test.md"] = testHash1

	if err := SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	hash, ok := loaded.Files["vision/test.md"]
	if !ok {
		t.Fatal("expected vision/test.md in loaded manifest")
	}
	if hash != testHash1 {
		t.Errorf("Hash = %q, want %q", hash, testHash1)
	}
}

func TestSaveManifest_Atomic(t *testing.T) {
	baseDir := t.TempDir()
	os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)

	m := NewManifest()
	if err := SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	// Temp file should not exist after save.
	tmpPath := manifestPath(baseDir) + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file should not exist after save")
	}

	// Manifest file should exist.
	if _, err := os.Stat(manifestPath(baseDir)); err != nil {
		t.Errorf("manifest file should exist after save: %v", err)
	}
}

func TestValidateManifestJSON(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantIssues int
		wantSubstr string
	}{
		{
			name:       "valid JSON",
			data:       `{"version":1,"files":{"vision/test.md":"` + testHash1 + `"}}`,
			wantIssues: 0,
		},
		{
			name:       "empty file",
			data:       "",
			wantIssues: 1,
			wantSubstr: "empty",
		},
		{
			name:       "truncated JSON",
			data:       `{"version":1,"files":{`,
			wantIssues: 1,
			wantSubstr: "invalid JSON",
		},
		{
			name:       "empty object",
			data:       `{}`,
			wantIssues: 0,
		},
		{
			name:       "version as string",
			data:       `{"version":"1","files":{}}`,
			wantIssues: 1,
			wantSubstr: "wrong type",
		},
		{
			name:       "version as null",
			data:       `{"version":null,"files":{}}`,
			wantIssues: 1,
			wantSubstr: "null",
		},
		{
			name:       "unknown root field",
			data:       `{"version":1,"files":{},"extra":"bad"}`,
			wantIssues: 1,
			wantSubstr: "unknown root field",
		},
		{
			name:       "null hash",
			data:       `{"version":1,"files":{"vision/test.md":null}}`,
			wantIssues: 1,
			wantSubstr: "null hash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateManifestJSON([]byte(tt.data))
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
			if tt.wantSubstr != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", tt.wantSubstr, issues)
				}
			}
		})
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name       string
		manifest   *Manifest
		wantIssues int
		wantSubstr string
	}{
		{
			name: "valid manifest",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md": testHash1,
				},
			},
			wantIssues: 0,
		},
		{
			name: "empty files map",
			manifest: &Manifest{
				Version: 1,
				Files:   map[string]string{},
			},
			wantIssues: 0,
		},
		{
			name: "wrong version",
			manifest: &Manifest{
				Version: 99,
				Files:   map[string]string{},
			},
			wantIssues: 1,
			wantSubstr: "unsupported version",
		},
		{
			name: "path traversal",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"../etc/passwd": testHash1,
				},
			},
			wantIssues: 1,
			wantSubstr: "invalid path",
		},
		{
			name: "absolute path",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"/etc/passwd": testHash1,
				},
			},
			wantIssues: 1,
			wantSubstr: "invalid path",
		},
		{
			name: "path in custom directory",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"auth/jwt-strategy.adr.md": testHash1,
				},
			},
			wantIssues: 0,
		},
		{
			name: "empty hash",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md": "",
				},
			},
			wantIssues: 1,
			wantSubstr: "hash is empty",
		},
		{
			name: "short hash",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md": "abc123",
				},
			},
			wantIssues: 1,
			wantSubstr: "not valid SHA-256",
		},
		{
			name: "uppercase hash",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md": "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2",
				},
			},
			wantIssues: 1,
			wantSubstr: "not valid SHA-256",
		},
		{
			name: "non-hex hash",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md": "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
				},
			},
			wantIssues: 1,
			wantSubstr: "not valid SHA-256",
		},
		{
			name: "too many entries",
			manifest: func() *Manifest {
				m := &Manifest{Version: 1, Files: make(map[string]string)}
				for i := 0; i < 10001; i++ {
					key := fmt.Sprintf("vision/file-%05d.md", i)
					m.Files[key] = testHash1
				}
				return m
			}(),
			wantIssues: 1, // just the "too many" issue (not counting per-file)
			wantSubstr: "too many",
		},
		{
			name: "multiple issues combined",
			manifest: &Manifest{
				Version: 99,
				Files: map[string]string{
					"other/test.md": "short",
				},
			},
			wantIssues: 2, // wrong version + bad hash
		},
		{
			name: "file at root (no directory)",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"my-doc.adr.md": testHash1,
				},
			},
			wantIssues: 0,
		},
		{
			name: "deeply nested path",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"infrastructure/k8s/prod/migration.adr.md": testHash1,
				},
			},
			wantIssues: 0,
		},
		{
			name: "mixed root and nested",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"root-doc.adr.md":           testHash1,
					"auth/jwt.adr.md":           testHash2,
					"infra/k8s/deploy.guide.md": testHash1,
				},
			},
			wantIssues: 0,
		},
		{
			name: "double-slash path",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision//test.md": testHash1,
				},
			},
			wantIssues: 1,
			wantSubstr: "empty path segment",
		},
		{
			name: "trailing slash",
			manifest: &Manifest{
				Version: 1,
				Files: map[string]string{
					"vision/test.md/": testHash1,
				},
			},
			wantIssues: 1,
			wantSubstr: "trailing slash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateManifest(tt.manifest)
			if tt.name == "too many entries" {
				// Only check that the "too many" issue is present.
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, "too many") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected 'too many' issue, got %v", issues)
				}
				return
			}
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
			if tt.wantSubstr != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", tt.wantSubstr, issues)
				}
			}
		})
	}
}

func TestLoadManifest_FreeFormRootPath(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, ".archcore")
	os.MkdirAll(dir, 0o755)
	data := `{"version":1,"files":{"my-doc.adr.md":"` + testHash1 + `"}}`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(data), 0o644)

	m, err := LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := m.Files["my-doc.adr.md"]; !ok {
		t.Error("expected root-level file in manifest")
	}
}

func TestSaveManifest_FreeFormPaths(t *testing.T) {
	baseDir := t.TempDir()
	os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)

	m := NewManifest()
	m.Files["root-doc.adr.md"] = testHash1
	m.Files["auth/jwt.adr.md"] = testHash2
	m.Files["infra/k8s/deploy.guide.md"] = testHash1

	if err := SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(loaded.Files) != 3 {
		t.Errorf("got %d files, want 3", len(loaded.Files))
	}
	for _, p := range []string{"root-doc.adr.md", "auth/jwt.adr.md", "infra/k8s/deploy.guide.md"} {
		if _, ok := loaded.Files[p]; !ok {
			t.Errorf("missing expected path %q", p)
		}
	}
}

func TestValidateManifestJSON_Relations(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantIssues int
		wantSubstr string
	}{
		{
			name:       "relations as string",
			data:       `{"version":1,"files":{},"relations":"bad"}`,
			wantIssues: 1,
			wantSubstr: "relations must be an array",
		},
		{
			name:       "relations as empty array",
			data:       `{"version":1,"files":{},"relations":[]}`,
			wantIssues: 0,
		},
		{
			name:       "relations with valid entry",
			data:       `{"version":1,"files":{},"relations":[{"source":"a.adr.md","target":"b.prd.md","type":"implements"}]}`,
			wantIssues: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateManifestJSON([]byte(tt.data))
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
			if tt.wantSubstr != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", tt.wantSubstr, issues)
				}
			}
		})
	}
}

func TestValidateManifest_Relations(t *testing.T) {
	tests := []struct {
		name       string
		relations  []Relation
		wantIssues int
		wantSubstr string
	}{
		{
			name:       "valid relation",
			relations:  []Relation{{Source: "a.adr.md", Target: "b.prd.md", Type: RelImplements}},
			wantIssues: 0,
		},
		{
			name:       "invalid type",
			relations:  []Relation{{Source: "a.adr.md", Target: "b.prd.md", Type: "blocks"}},
			wantIssues: 1,
			wantSubstr: "invalid type",
		},
		{
			name:       "source equals target",
			relations:  []Relation{{Source: "a.adr.md", Target: "a.adr.md", Type: RelRelated}},
			wantIssues: 1,
			wantSubstr: "source and target are the same",
		},
		{
			name: "duplicate relation",
			relations: []Relation{
				{Source: "a.adr.md", Target: "b.prd.md", Type: RelRelated},
				{Source: "a.adr.md", Target: "b.prd.md", Type: RelRelated},
			},
			wantIssues: 1,
			wantSubstr: "duplicate relation",
		},
		{
			name:       "source path traversal",
			relations:  []Relation{{Source: "../etc/passwd", Target: "b.prd.md", Type: RelRelated}},
			wantIssues: 1,
			wantSubstr: "source",
		},
		{
			name:       "target path traversal",
			relations:  []Relation{{Source: "a.adr.md", Target: "../etc/passwd", Type: RelRelated}},
			wantIssues: 1,
			wantSubstr: "target",
		},
		{
			name: "too many relations",
			relations: func() []Relation {
				rels := make([]Relation, 50001)
				for i := range rels {
					rels[i] = Relation{
						Source: fmt.Sprintf("src-%d.adr.md", i),
						Target: fmt.Sprintf("tgt-%d.prd.md", i),
						Type:   RelRelated,
					}
				}
				return rels
			}(),
			wantIssues: 1,
			wantSubstr: "too many relations",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{
				Version:   1,
				Files:     map[string]string{},
				Relations: tt.relations,
			}
			issues := ValidateManifest(m)
			if tt.name == "too many relations" {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", tt.wantSubstr, issues)
				}
				return
			}
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
			if tt.wantSubstr != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, tt.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing %q, got %v", tt.wantSubstr, issues)
				}
			}
		})
	}
}

func TestManifest_AddRelation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *Manifest)
		source  string
		target  string
		relType RelationType
		want    bool
		wantLen int
	}{
		{
			name:    "add to empty",
			setup:   func(m *Manifest) {},
			source:  "a.adr.md",
			target:  "b.prd.md",
			relType: RelImplements,
			want:    true,
			wantLen: 1,
		},
		{
			name: "duplicate returns false",
			setup: func(m *Manifest) {
				m.AddRelation("a.adr.md", "b.prd.md", RelImplements)
			},
			source:  "a.adr.md",
			target:  "b.prd.md",
			relType: RelImplements,
			want:    false,
			wantLen: 1,
		},
		{
			name: "different type adds",
			setup: func(m *Manifest) {
				m.AddRelation("a.adr.md", "b.prd.md", RelImplements)
			},
			source:  "a.adr.md",
			target:  "b.prd.md",
			relType: RelRelated,
			want:    true,
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManifest()
			tt.setup(m)
			got := m.AddRelation(tt.source, tt.target, tt.relType)
			if got != tt.want {
				t.Errorf("AddRelation = %v, want %v", got, tt.want)
			}
			if len(m.Relations) != tt.wantLen {
				t.Errorf("len(Relations) = %d, want %d", len(m.Relations), tt.wantLen)
			}
		})
	}
}

func TestManifest_RemoveRelation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *Manifest)
		source  string
		target  string
		relType RelationType
		want    bool
	}{
		{
			name: "remove existing",
			setup: func(m *Manifest) {
				m.AddRelation("a.adr.md", "b.prd.md", RelImplements)
			},
			source:  "a.adr.md",
			target:  "b.prd.md",
			relType: RelImplements,
			want:    true,
		},
		{
			name: "remove non-existent",
			setup: func(m *Manifest) {
				m.AddRelation("a.adr.md", "b.prd.md", RelImplements)
			},
			source:  "x.adr.md",
			target:  "y.prd.md",
			relType: RelImplements,
			want:    false,
		},
		{
			name:    "remove from empty",
			setup:   func(m *Manifest) {},
			source:  "a.adr.md",
			target:  "b.prd.md",
			relType: RelImplements,
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManifest()
			tt.setup(m)
			got := m.RemoveRelation(tt.source, tt.target, tt.relType)
			if got != tt.want {
				t.Errorf("RemoveRelation = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManifest_RelationsFor(t *testing.T) {
	m := NewManifest()
	m.AddRelation("a.adr.md", "b.prd.md", RelImplements)
	m.AddRelation("c.rfc.md", "a.adr.md", RelDependsOn)
	m.AddRelation("d.doc.md", "e.guide.md", RelRelated)

	tests := []struct {
		name         string
		path         string
		wantOutgoing int
		wantIncoming int
	}{
		{"path as source", "a.adr.md", 1, 1},
		{"path as target only", "b.prd.md", 0, 1},
		{"path with no relations", "z.md", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, in := m.RelationsFor(tt.path)
			if len(out) != tt.wantOutgoing {
				t.Errorf("outgoing = %d, want %d", len(out), tt.wantOutgoing)
			}
			if len(in) != tt.wantIncoming {
				t.Errorf("incoming = %d, want %d", len(in), tt.wantIncoming)
			}
		})
	}
}

func TestManifest_CleanupRelations(t *testing.T) {
	tests := []struct {
		name         string
		relations    []Relation
		files        []string // files to create on disk (relative to archcoreDir)
		wantRemoved  int
		wantRelCount int
	}{
		{
			name: "no dangling relations",
			relations: []Relation{
				{Source: "a.adr.md", Target: "b.prd.md", Type: RelImplements},
			},
			files:        []string{"a.adr.md", "b.prd.md"},
			wantRemoved:  0,
			wantRelCount: 1,
		},
		{
			name: "source missing",
			relations: []Relation{
				{Source: "gone.adr.md", Target: "b.prd.md", Type: RelImplements},
			},
			files:        []string{"b.prd.md"},
			wantRemoved:  1,
			wantRelCount: 0,
		},
		{
			name: "target missing",
			relations: []Relation{
				{Source: "a.adr.md", Target: "gone.prd.md", Type: RelRelated},
			},
			files:        []string{"a.adr.md"},
			wantRemoved:  1,
			wantRelCount: 0,
		},
		{
			name: "both missing",
			relations: []Relation{
				{Source: "gone1.adr.md", Target: "gone2.prd.md", Type: RelRelated},
			},
			files:        []string{},
			wantRemoved:  1,
			wantRelCount: 0,
		},
		{
			name: "mixed valid and dangling",
			relations: []Relation{
				{Source: "a.adr.md", Target: "b.prd.md", Type: RelImplements},
				{Source: "a.adr.md", Target: "gone.rfc.md", Type: RelRelated},
				{Source: "c.doc.md", Target: "a.adr.md", Type: RelDependsOn},
			},
			files:        []string{"a.adr.md", "b.prd.md", "c.doc.md"},
			wantRemoved:  1,
			wantRelCount: 2,
		},
		{
			name:         "empty relations",
			relations:    nil,
			files:        []string{},
			wantRemoved:  0,
			wantRelCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archcoreDir := t.TempDir()
			for _, f := range tt.files {
				p := filepath.Join(archcoreDir, f)
				os.MkdirAll(filepath.Dir(p), 0o755)
				if err := os.WriteFile(p, []byte("content"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			m := NewManifest()
			m.Relations = tt.relations
			removed := m.CleanupRelations(archcoreDir)
			if removed != tt.wantRemoved {
				t.Errorf("CleanupRelations removed %d, want %d", removed, tt.wantRemoved)
			}
			if len(m.Relations) != tt.wantRelCount {
				t.Errorf("len(Relations) = %d, want %d", len(m.Relations), tt.wantRelCount)
			}
		})
	}
}

func TestIsValidRelationType(t *testing.T) {
	for _, rt := range []string{"related", "implements", "extends", "depends_on"} {
		if !IsValidRelationType(rt) {
			t.Errorf("expected %q to be valid", rt)
		}
	}
	if IsValidRelationType("blocks") {
		t.Error("expected 'blocks' to be invalid")
	}
}

func TestLoadSaveManifest_WithRelations(t *testing.T) {
	baseDir := t.TempDir()
	os.MkdirAll(filepath.Join(baseDir, ".archcore"), 0o755)

	m := NewManifest()
	m.Files["vision/test.md"] = testHash1
	m.AddRelation("vision/test.md", "knowledge/impl.md", RelImplements)
	m.AddRelation("vision/test.md", "knowledge/ref.md", RelRelated)

	if err := SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(loaded.Relations) != 2 {
		t.Fatalf("got %d relations, want 2", len(loaded.Relations))
	}
	if loaded.Relations[0].Source != "vision/test.md" {
		t.Errorf("relation[0].Source = %q", loaded.Relations[0].Source)
	}
	if loaded.Relations[0].Type != RelImplements {
		t.Errorf("relation[0].Type = %q", loaded.Relations[0].Type)
	}
}

func TestManifest_SevenRelationValues(t *testing.T) {
	t.Parallel()
	want := []string{"related", "implements", "extends", "depends_on", "supports", "contradicts", "supersedes"}
	got := ValidRelationTypes()
	if len(got) != len(want) || len(validRelationTypes) != len(want) {
		t.Fatalf("relation registries = %v / %v", got, validRelationTypes)
	}
	for i, value := range want {
		if got[i] != value || !IsValidRelationType(value) {
			t.Errorf("relation %q missing from registry", value)
		}
	}
	for _, value := range append(want, "blocks") {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
				t.Fatal(err)
			}
			input := fmt.Sprintf(`{"version":1,"files":{},"relations":[{"source":"one.evidence.md","target":"two.research.md","type":%q}]}`, value)
			path := filepath.Join(base, ".archcore", ManifestFile)
			if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
				t.Fatal(err)
			}
			m, err := LoadManifest(base)
			if value == "blocks" {
				if err == nil {
					t.Fatal("unknown relation accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(m.Relations) != 1 || string(m.Relations[0].Type) != value {
				t.Fatalf("relations = %+v", m.Relations)
			}
			if err := SaveManifest(base, m); err != nil {
				t.Fatal(err)
			}
			m, err = LoadManifest(base)
			if err != nil || string(m.Relations[0].Type) != value {
				t.Fatalf("reload = %+v / %v", m, err)
			}
		})
	}
}
