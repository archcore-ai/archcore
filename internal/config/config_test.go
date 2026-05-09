package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitDir(t *testing.T) {
	base := t.TempDir()
	if err := InitDir(base); err != nil {
		t.Fatalf("InitDir: %v", err)
	}
	p := filepath.Join(base, dirName)
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf(".archcore/ not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".archcore/ is not a directory")
	}
}

func TestInitDir_Idempotent(t *testing.T) {
	base := t.TempDir()
	if err := InitDir(base); err != nil {
		t.Fatalf("first InitDir: %v", err)
	}
	if err := InitDir(base); err != nil {
		t.Fatalf("second InitDir: %v", err)
	}
}

func TestDirExists(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, base string)
		want  bool
	}{
		{
			name:  "no directory",
			setup: func(t *testing.T, base string) {},
			want:  false,
		},
		{
			name: "after InitDir",
			setup: func(t *testing.T, base string) {
				if err := InitDir(base); err != nil {
					t.Fatalf("InitDir: %v", err)
				}
			},
			want: true,
		},
		{
			name: "file instead of dir",
			setup: func(t *testing.T, base string) {
				if err := os.WriteFile(filepath.Join(base, dirName), []byte("x"), 0o644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			tt.setup(t, base)
			if got := DirExists(base); got != tt.want {
				t.Errorf("DirExists = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Constructors ---

func TestNewNoneSettings(t *testing.T) {
	s := NewNoneSettings()
	if s.Sync != SyncTypeNone {
		t.Errorf("Sync = %q, want %q", s.Sync, SyncTypeNone)
	}
	if s.ProjectID != nil {
		t.Error("ProjectID should be nil")
	}
	if s.ArchcoreURL != "" {
		t.Error("ArchcoreURL should be empty")
	}
}

func TestNewCloudSettings(t *testing.T) {
	s := NewCloudSettings()
	if s.Sync != SyncTypeCloud {
		t.Errorf("Sync = %q, want %q", s.Sync, SyncTypeCloud)
	}
	if s.ProjectID != nil {
		t.Error("ProjectID should be nil")
	}
}

func TestNewOnPremSettings(t *testing.T) {
	s := NewOnPremSettings("http://my-server:8080")
	if s.Sync != SyncTypeOnPrem {
		t.Errorf("Sync = %q, want %q", s.Sync, SyncTypeOnPrem)
	}
	if s.ArchcoreURL != "http://my-server:8080" {
		t.Errorf("ArchcoreURL = %q, want %q", s.ArchcoreURL, "http://my-server:8080")
	}
}

// --- ServerURL ---

func TestServerURL(t *testing.T) {
	tests := []struct {
		name string
		s    *Settings
		want string
	}{
		{"none", NewNoneSettings(), ""},
		{"cloud", NewCloudSettings(), CloudServerURL},
		{"on-prem", NewOnPremSettings("http://my:8080"), "http://my:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.ServerURL(); got != tt.want {
				t.Errorf("ServerURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Validate ---

func TestValidate(t *testing.T) {
	pid := 42
	tests := []struct {
		name    string
		s       Settings
		wantErr bool
	}{
		{"none valid", Settings{Sync: SyncTypeNone}, false},
		{"cloud valid nil pid", Settings{Sync: SyncTypeCloud}, false},
		{"cloud valid with pid", Settings{Sync: SyncTypeCloud, ProjectID: &pid}, false},
		{"on-prem valid", Settings{Sync: SyncTypeOnPrem, ArchcoreURL: "http://x:8080"}, false},
		{"on-prem valid with pid", Settings{Sync: SyncTypeOnPrem, ProjectID: &pid, ArchcoreURL: "http://x:8080"}, false},
		{"none with language", Settings{Sync: SyncTypeNone, Language: "ru"}, false},
		{"cloud with language", Settings{Sync: SyncTypeCloud, Language: "ja"}, false},
		{"on-prem with language", Settings{Sync: SyncTypeOnPrem, ArchcoreURL: "http://x:8080", Language: "de"}, false},
		{"language with spaces", Settings{Sync: SyncTypeNone, Language: "en US"}, true},
		{"none with pid", Settings{Sync: SyncTypeNone, ProjectID: &pid}, true},
		{"none with url", Settings{Sync: SyncTypeNone, ArchcoreURL: "http://x"}, true},
		{"cloud with url", Settings{Sync: SyncTypeCloud, ArchcoreURL: "http://x"}, true},
		{"on-prem no url", Settings{Sync: SyncTypeOnPrem}, true},
		{"unknown sync", Settings{Sync: "magic"}, true},
		{"empty sync", Settings{Sync: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- MarshalJSON ---

func TestMarshalJSON(t *testing.T) {
	pid := 42
	tests := []struct {
		name string
		s    Settings
		want string
	}{
		{
			"none",
			Settings{Sync: SyncTypeNone},
			`{"sync":"none"}`,
		},
		{
			"cloud nil pid",
			Settings{Sync: SyncTypeCloud},
			`{"sync":"cloud"}`,
		},
		{
			"cloud with pid",
			Settings{Sync: SyncTypeCloud, ProjectID: &pid},
			`{"sync":"cloud","project_id":42}`,
		},
		{
			"on-prem nil pid",
			Settings{Sync: SyncTypeOnPrem, ArchcoreURL: "http://x:8080"},
			`{"sync":"on-prem","archcore_url":"http://x:8080"}`,
		},
		{
			"on-prem with pid",
			Settings{Sync: SyncTypeOnPrem, ProjectID: &pid, ArchcoreURL: "http://x:8080"},
			`{"sync":"on-prem","project_id":42,"archcore_url":"http://x:8080"}`,
		},
		{
			"none with language",
			Settings{Sync: SyncTypeNone, Language: "ru"},
			`{"sync":"none","language":"ru"}`,
		},
		{
			"cloud with language",
			Settings{Sync: SyncTypeCloud, Language: "ja"},
			`{"sync":"cloud","language":"ja"}`,
		},
		{
			"on-prem with language",
			Settings{Sync: SyncTypeOnPrem, ArchcoreURL: "http://x:8080", Language: "de"},
			`{"sync":"on-prem","archcore_url":"http://x:8080","language":"de"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.s)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON =\n  %s\nwant\n  %s", got, tt.want)
			}
		})
	}
}

func TestMarshalJSON_UnknownSync(t *testing.T) {
	s := Settings{Sync: "magic"}
	_, err := json.Marshal(s)
	if err == nil {
		t.Fatal("expected error for unknown sync type")
	}
}

// --- UnmarshalJSON ---

func TestUnmarshalJSON_Valid(t *testing.T) {
	pid := 42
	tests := []struct {
		name     string
		input    string
		wantSync SyncType
		wantPID  *int
		wantURL  string
		wantLang string
	}{
		{
			"none",
			`{"sync":"none"}`,
			SyncTypeNone, nil, "", "",
		},
		{
			"cloud no pid",
			`{"sync":"cloud"}`,
			SyncTypeCloud, nil, "", "",
		},
		{
			"cloud null pid",
			`{"sync":"cloud","project_id":null}`,
			SyncTypeCloud, nil, "", "",
		},
		{
			"cloud with pid",
			`{"sync":"cloud","project_id":42}`,
			SyncTypeCloud, &pid, "", "",
		},
		{
			"on-prem no pid",
			`{"sync":"on-prem","archcore_url":"http://x:8080"}`,
			SyncTypeOnPrem, nil, "http://x:8080", "",
		},
		{
			"on-prem null pid",
			`{"sync":"on-prem","project_id":null,"archcore_url":"http://x:8080"}`,
			SyncTypeOnPrem, nil, "http://x:8080", "",
		},
		{
			"on-prem with pid",
			`{"sync":"on-prem","project_id":42,"archcore_url":"http://x:8080"}`,
			SyncTypeOnPrem, &pid, "http://x:8080", "",
		},
		{
			"none with language",
			`{"sync":"none","language":"ru"}`,
			SyncTypeNone, nil, "", "ru",
		},
		{
			"cloud with language",
			`{"sync":"cloud","language":"ja"}`,
			SyncTypeCloud, nil, "", "ja",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Settings
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if s.Sync != tt.wantSync {
				t.Errorf("Sync = %q, want %q", s.Sync, tt.wantSync)
			}
			if (s.ProjectID == nil) != (tt.wantPID == nil) {
				t.Errorf("ProjectID nil = %v, want %v", s.ProjectID == nil, tt.wantPID == nil)
			} else if s.ProjectID != nil && *s.ProjectID != *tt.wantPID {
				t.Errorf("ProjectID = %d, want %d", *s.ProjectID, *tt.wantPID)
			}
			if s.ArchcoreURL != tt.wantURL {
				t.Errorf("ArchcoreURL = %q, want %q", s.ArchcoreURL, tt.wantURL)
			}
			if s.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", s.Language, tt.wantLang)
			}
		})
	}
}

func TestUnmarshalJSON_Rejection(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"none with project_id", `{"sync":"none","project_id":null}`},
		{"none with archcore_url", `{"sync":"none","archcore_url":"http://x"}`},
		{"cloud with archcore_url", `{"sync":"cloud","project_id":null,"archcore_url":"http://x"}`},
		{"on-prem missing archcore_url", `{"sync":"on-prem","project_id":null}`},
		{"on-prem empty archcore_url", `{"sync":"on-prem","project_id":null,"archcore_url":""}`},
		{"unknown sync type", `{"sync":"magic"}`},
		{"missing sync", `{"project_id":null}`},
		{"unknown field", `{"sync":"none","extra":true}`},
		{"invalid JSON", `{invalid`},
		{"project_id as string", `{"sync":"cloud","project_id":"42"}`},
		{"language empty", `{"sync":"none","language":""}`},
		{"language with spaces", `{"sync":"none","language":"en US"}`},
		{"language as number", `{"sync":"none","language":42}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Settings
			if err := json.Unmarshal([]byte(tt.input), &s); err == nil {
				t.Errorf("expected error for input: %s", tt.input)
			}
		})
	}
}

// --- Roundtrip ---

func TestRoundtrip(t *testing.T) {
	pid := 7
	tests := []struct {
		name string
		s    Settings
	}{
		{"none", Settings{Sync: SyncTypeNone}},
		{"cloud nil pid", Settings{Sync: SyncTypeCloud}},
		{"cloud with pid", Settings{Sync: SyncTypeCloud, ProjectID: &pid}},
		{"on-prem nil pid", Settings{Sync: SyncTypeOnPrem, ArchcoreURL: "http://x:8080"}},
		{"on-prem with pid", Settings{Sync: SyncTypeOnPrem, ProjectID: &pid, ArchcoreURL: "http://x:8080"}},
		{"none with language", Settings{Sync: SyncTypeNone, Language: "ru"}},
		{"cloud with language", Settings{Sync: SyncTypeCloud, Language: "ja"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.s)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got Settings
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got.Sync != tt.s.Sync {
				t.Errorf("Sync = %q, want %q", got.Sync, tt.s.Sync)
			}
			if (got.ProjectID == nil) != (tt.s.ProjectID == nil) {
				t.Errorf("ProjectID nil mismatch")
			} else if got.ProjectID != nil && *got.ProjectID != *tt.s.ProjectID {
				t.Errorf("ProjectID = %d, want %d", *got.ProjectID, *tt.s.ProjectID)
			}
			if got.ArchcoreURL != tt.s.ArchcoreURL {
				t.Errorf("ArchcoreURL = %q, want %q", got.ArchcoreURL, tt.s.ArchcoreURL)
			}
			if got.Language != tt.s.Language {
				t.Errorf("Language = %q, want %q", got.Language, tt.s.Language)
			}
		})
	}
}

// --- Save/Load integration ---

func TestSaveAndLoad(t *testing.T) {
	pid := 99
	tests := []struct {
		name string
		s    *Settings
	}{
		{"none", NewNoneSettings()},
		{"cloud nil pid", NewCloudSettings()},
		{"cloud with pid", &Settings{Sync: SyncTypeCloud, ProjectID: &pid}},
		{"on-prem", NewOnPremSettings("http://internal:8080")},
		{"on-prem with pid", &Settings{Sync: SyncTypeOnPrem, ProjectID: &pid, ArchcoreURL: "http://internal:8080"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			if err := Save(base, tt.s); err != nil {
				t.Fatalf("Save: %v", err)
			}
			loaded, err := Load(base)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if loaded.Sync != tt.s.Sync {
				t.Errorf("Sync = %q, want %q", loaded.Sync, tt.s.Sync)
			}
			if (loaded.ProjectID == nil) != (tt.s.ProjectID == nil) {
				t.Errorf("ProjectID nil mismatch")
			} else if loaded.ProjectID != nil && *loaded.ProjectID != *tt.s.ProjectID {
				t.Errorf("ProjectID = %d, want %d", *loaded.ProjectID, *tt.s.ProjectID)
			}
			if loaded.ArchcoreURL != tt.s.ArchcoreURL {
				t.Errorf("ArchcoreURL = %q, want %q", loaded.ArchcoreURL, tt.s.ArchcoreURL)
			}
		})
	}
}

func TestSave_RejectsInvalid(t *testing.T) {
	base := t.TempDir()
	bad := &Settings{Sync: SyncTypeNone, ArchcoreURL: "http://x"}
	if err := Save(base, bad); err == nil {
		t.Fatal("expected error saving invalid settings")
	}
	// File should not have been written.
	_, err := os.Stat(settingsPath(base))
	if err == nil {
		t.Fatal("settings file should not exist after failed save")
	}
}

func TestLoad_NoFile(t *testing.T) {
	base := t.TempDir()
	_, err := Load(base)
	if err == nil {
		t.Fatal("expected error for missing settings file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(base)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoad_RejectsInvalidSettings(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write a JSON with unknown field.
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"sync":"none","extra":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(base)
	if err == nil {
		t.Fatal("expected error for invalid settings")
	}
}

func TestSave_CreatesDir(t *testing.T) {
	base := t.TempDir()
	s := NewNoneSettings()
	if err := Save(base, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(base, dirName))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".archcore is not a directory")
	}
}
