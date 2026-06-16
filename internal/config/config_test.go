package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestValidate_Globals(t *testing.T) {
	tests := []struct {
		name    string
		globals []GlobalSource
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid embedded path",
			globals: []GlobalSource{{ID: "company", Path: ".archcore/global/company"}},
		},
		{
			name:    "valid external path",
			globals: []GlobalSource{{ID: "platform", Path: "vendor/platform"}},
		},
		{
			name:    "valid external archcore path",
			globals: []GlobalSource{{ID: "corp", Path: "../corp/.archcore"}},
		},
		{
			name:    "empty id",
			globals: []GlobalSource{{ID: "", Path: ".archcore/global/company"}},
			wantErr: true, errMsg: `"id" must not be empty`,
		},
		{
			name:    "id with spaces",
			globals: []GlobalSource{{ID: "my company", Path: ".archcore/global/company"}},
			wantErr: true, errMsg: "lowercase alphanumeric with hyphens",
		},
		{
			name:    "uppercase id",
			globals: []GlobalSource{{ID: "Company", Path: ".archcore/global/company"}},
			wantErr: true, errMsg: "lowercase alphanumeric with hyphens",
		},
		{
			name:    "reserved id local",
			globals: []GlobalSource{{ID: "local", Path: ".archcore/global/local"}},
			wantErr: true, errMsg: "reserved",
		},
		{
			name:    "empty path",
			globals: []GlobalSource{{ID: "company", Path: ""}},
			wantErr: true, errMsg: `"path" must not be empty`,
		},
		{
			name:    "sibling path with parent traversal is valid",
			globals: []GlobalSource{{ID: "company-global", Path: "../company-global/.archcore"}},
			wantErr: false,
		},
		{
			name: "duplicate ids",
			globals: []GlobalSource{
				{ID: "company", Path: ".archcore/global/company"},
				{ID: "company", Path: ".archcore/global/other"},
			},
			wantErr: true, errMsg: "duplicate id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Settings{Sync: SyncTypeNone, Globals: tt.globals}
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
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
		// NOTE: a genuinely-unknown field (e.g. "extra") is no longer rejected — it
		// is tolerated and captured into Extra. See TestUnmarshalJSON_TolerateUnknownScalar.
		// A KNOWN field in the wrong sync mode (above) still errors.
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
	// A KNOWN field in the wrong sync mode is a misconfiguration and must still be
	// rejected (project_id is not valid for sync "none").
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"sync":"none","project_id":1}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(base)
	if err == nil {
		t.Fatal("expected error for invalid settings")
	}
}

// TestLoad_TolerantOfUnknownField is the forward-compatibility flip: a field this
// binary does not recognize (as a newer archcore would add) no longer fails Load.
func TestLoad_TolerantOfUnknownField(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"sync":"none","future":1}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	s, err := Load(base)
	if err != nil {
		t.Fatalf("Load should tolerate an unknown field, got: %v", err)
	}
	if _, ok := s.Extra["future"]; !ok {
		t.Errorf("unknown field 'future' should be captured into Extra, got Extra=%v", s.Extra)
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

// --- Forward-compatible parsing: tolerate + preserve unknown fields ---

func TestUnmarshalJSON_TolerateUnknownScalar(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"sync":"none","flag":true}`), &s); err != nil {
		t.Fatalf("unknown field should be tolerated, got: %v", err)
	}
	if s.Sync != SyncTypeNone {
		t.Errorf("Sync = %q, want none", s.Sync)
	}
	if len(s.Extra) != 1 {
		t.Fatalf("Extra len = %d, want 1 (Extra=%v)", len(s.Extra), s.Extra)
	}
	if got := string(s.Extra["flag"]); got != "true" {
		t.Errorf("Extra[flag] = %q, want true", got)
	}
}

func TestUnmarshalJSON_TolerateUnknownObjectArrayNull(t *testing.T) {
	tests := []struct {
		name, input, key, want string
	}{
		{"object", `{"sync":"none","obj":{"a":1}}`, "obj", `{"a":1}`},
		{"array", `{"sync":"none","arr":[1,2]}`, "arr", `[1,2]`},
		{"null", `{"sync":"none","n":null}`, "n", `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Settings
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatalf("should tolerate %s value: %v", tt.name, err)
			}
			if got := string(s.Extra[tt.key]); got != tt.want {
				t.Errorf("Extra[%s] = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSON_MultipleUnknown_CapturedAll(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"sync":"none","a":1,"b":2}`), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(s.Extra) != 2 || s.Extra["a"] == nil || s.Extra["b"] == nil {
		t.Errorf("Extra = %v, want both a and b captured", s.Extra)
	}
}

func TestUnmarshalJSON_NoExtra_NilMap(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"sync":"none","language":"ru"}`), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if s.Extra != nil {
		t.Errorf("Extra should be nil when no unknown fields, got %v", s.Extra)
	}
}

// TestUnmarshalJSON_KnownFieldWrongMode_StillRejected pins the union-knownFields
// requirement: a field this binary KNOWS but that is invalid for the declared sync
// mode must still hard-error (and not be captured into Extra), unchanged.
func TestUnmarshalJSON_KnownFieldWrongMode_StillRejected(t *testing.T) {
	tests := []struct{ name, input string }{
		{"archcore_url under none", `{"sync":"none","archcore_url":"http://x"}`},
		{"project_id under none", `{"sync":"none","project_id":1}`},
		{"archcore_url under cloud", `{"sync":"cloud","archcore_url":"http://x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Settings
			err := json.Unmarshal([]byte(tt.input), &s)
			if err == nil {
				t.Fatalf("expected error for %s", tt.input)
			}
			if !strings.Contains(err.Error(), "is not allowed for sync type") {
				t.Errorf("error %q should keep the existing wrong-mode message", err)
			}
			if len(s.Extra) != 0 {
				t.Errorf("known wrong-mode field must not land in Extra, got %v", s.Extra)
			}
		})
	}
}

// TestUnmarshalJSON_KnownNeverInExtra ensures recognized fields are decoded into
// typed fields, never captured into Extra.
func TestUnmarshalJSON_KnownNeverInExtra(t *testing.T) {
	var s Settings
	input := `{"sync":"on-prem","project_id":7,"archcore_url":"http://x","language":"en","globals":[{"id":"a","path":"p"}]}`
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if s.Extra != nil {
		t.Errorf("no known field should be in Extra, got %v", s.Extra)
	}
}

func TestUnmarshalJSON_MalformedUnknownValueStillErrors(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"sync":"none","x":}`), &s); err == nil {
		t.Fatal("malformed JSON must still error (outer parse fails before capture)")
	}
}

func TestUnmarshalJSON_UnknownSyncStillErrors(t *testing.T) {
	var s Settings
	err := json.Unmarshal([]byte(`{"sync":"magic","x":1}`), &s)
	if err == nil || !strings.Contains(err.Error(), "unknown sync type") {
		t.Fatalf("want unknown-sync-type error, got: %v", err)
	}
}

func TestMarshalJSON_ByteIdenticalWhenNoExtra(t *testing.T) {
	want := `{"sync":"none","language":"ru"}`
	nilExtra := Settings{Sync: SyncTypeNone, Language: "ru"}
	emptyExtra := Settings{Sync: SyncTypeNone, Language: "ru", Extra: map[string]json.RawMessage{}}
	for _, s := range []Settings{nilExtra, emptyExtra} {
		got, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if string(got) != want {
			t.Errorf("Marshal = %s, want byte-identical %s", got, want)
		}
	}
}

func TestMarshalJSON_PreservesExtra(t *testing.T) {
	s := Settings{Sync: SyncTypeNone, Extra: map[string]json.RawMessage{"new_thing": json.RawMessage(`{"k":1}`)}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if string(m["sync"]) != `"none"` {
		t.Errorf("sync = %s, want \"none\"", m["sync"])
	}
	if string(m["new_thing"]) != `{"k":1}` {
		t.Errorf("new_thing = %s, want {\"k\":1}", m["new_thing"])
	}
}

func TestMarshalJSON_ExtraDeterministicAndDisjoint(t *testing.T) {
	s := Settings{Sync: SyncTypeNone, Language: "en", Extra: map[string]json.RawMessage{
		"b": json.RawMessage(`1`), "a": json.RawMessage(`2`),
	}}
	b1, _ := json.Marshal(s)
	b2, _ := json.Marshal(s)
	if string(b1) != string(b2) {
		t.Errorf("Marshal not deterministic: %s vs %s", b1, b2)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b1, &m); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	for _, k := range []string{"sync", "language", "a", "b"} {
		if _, ok := m[k]; !ok {
			t.Errorf("key %q missing from merged output %s", k, b1)
		}
	}
}

func TestMarshalJSON_IndentedPreservesExtra(t *testing.T) {
	s := Settings{Sync: SyncTypeNone, Extra: map[string]json.RawMessage{"future": json.RawMessage(`true`)}}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if !strings.Contains(string(b), "\n  ") {
		t.Errorf("output not 2-space indented:\n%s", b)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if string(m["future"]) != "true" {
		t.Errorf("future = %s, want true", m["future"])
	}
}

func TestRoundtrip_PreservesExtra(t *testing.T) {
	s := Settings{Sync: SyncTypeNone, Language: "ru", Extra: map[string]json.RawMessage{"future": json.RawMessage(`{"x":1}`)}}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Language != "ru" {
		t.Errorf("Language = %q, want ru", got.Language)
	}
	if string(got.Extra["future"]) != `{"x":1}` {
		t.Errorf("Extra[future] = %s, want {\"x\":1}", got.Extra["future"])
	}
}

func TestSaveAndLoad_PreservesUnknownField(t *testing.T) {
	base := t.TempDir()
	s := &Settings{Sync: SyncTypeNone, Language: "en", Extra: map[string]json.RawMessage{"future_flag": json.RawMessage(`true`)}}
	if err := Save(base, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(settingsPath(base))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "future_flag") {
		t.Errorf("saved file should physically contain future_flag:\n%s", raw)
	}
	loaded, err := Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Language != "en" {
		t.Errorf("Language = %q, want en", loaded.Language)
	}
	if string(loaded.Extra["future_flag"]) != "true" {
		t.Errorf("Extra[future_flag] = %s, want true", loaded.Extra["future_flag"])
	}
}

func TestUnknownFieldNames_Sorted(t *testing.T) {
	s := Settings{Extra: map[string]json.RawMessage{"b": json.RawMessage(`1`), "a": json.RawMessage(`2`)}}
	got := s.UnknownFieldNames()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("UnknownFieldNames = %v, want [a b]", got)
	}
	if names := (&Settings{}).UnknownFieldNames(); names != nil {
		t.Errorf("UnknownFieldNames with no Extra = %v, want nil", names)
	}
}
