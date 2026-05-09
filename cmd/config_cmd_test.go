package cmd

import (
	"os"
	"strings"
	"testing"

	"archcore-cli/internal/config"
)

func setupConfigTest(t *testing.T, s *config.Settings) string {
	t.Helper()
	dir := t.TempDir()
	if err := config.InitDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, s); err != nil {
		t.Fatal(err)
	}
	return dir
}

// --- getSettingsValue ---

func TestGetSettingsValue_Sync(t *testing.T) {
	tests := []struct {
		name string
		s    *config.Settings
		want string
	}{
		{"none", config.NewNoneSettings(), "none"},
		{"cloud", config.NewCloudSettings(), "cloud"},
		{"on-prem", config.NewOnPremSettings("http://x"), "on-prem"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSettingsValue(tt.s, "sync")
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSettingsValue_ProjectID(t *testing.T) {
	pid := 42
	tests := []struct {
		name    string
		s       *config.Settings
		want    string
		wantErr bool
	}{
		{"cloud null", config.NewCloudSettings(), "null", false},
		{"cloud with pid", &config.Settings{Sync: "cloud", ProjectID: &pid}, "42", false},
		{"none errors", config.NewNoneSettings(), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSettingsValue(tt.s, "project_id")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSettingsValue_ArchcoreURL(t *testing.T) {
	tests := []struct {
		name    string
		s       *config.Settings
		want    string
		wantErr bool
	}{
		{"on-prem", config.NewOnPremSettings("http://x:8080"), "http://x:8080", false},
		{"cloud errors", config.NewCloudSettings(), "", true},
		{"none errors", config.NewNoneSettings(), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSettingsValue(tt.s, "archcore_url")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSettingsValue_UnknownKey(t *testing.T) {
	s := config.NewNoneSettings()
	_, err := getSettingsValue(s, "unknown")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

// --- setSettingsValue ---

func TestSetSettingsValue_SyncType(t *testing.T) {
	// Switch from cloud to none — should reset fields.
	pid := 42
	s := &config.Settings{Sync: "cloud", ProjectID: &pid}
	if err := setSettingsValue(s, "sync", "none"); err != nil {
		t.Fatal(err)
	}
	if s.Sync != "none" {
		t.Errorf("Sync = %q, want none", s.Sync)
	}
	if s.ProjectID != nil {
		t.Error("ProjectID should be nil after switching to none")
	}
}

func TestSetSettingsValue_SyncToOnPrem_NoURL(t *testing.T) {
	s := config.NewCloudSettings()
	err := setSettingsValue(s, "sync", "on-prem")
	if err == nil {
		t.Fatal("expected error switching to on-prem without archcore_url")
	}
	// Verify the error message guides the user to use 'config set archcore_url'.
	want := "archcore config set archcore_url"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error message %q should contain %q", err.Error(), want)
	}
}

func TestSetSettingsValue_SyncToOnPrem_WithURL(t *testing.T) {
	s := &config.Settings{Sync: "on-prem", ArchcoreURL: "http://x:8080"}
	// Switching to on-prem when already on-prem with URL is fine.
	if err := setSettingsValue(s, "sync", "on-prem"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetSettingsValue_SyncInvalid(t *testing.T) {
	s := config.NewNoneSettings()
	if err := setSettingsValue(s, "sync", "magic"); err == nil {
		t.Fatal("expected error for invalid sync type")
	}
}

func TestSetSettingsValue_ProjectID(t *testing.T) {
	s := config.NewCloudSettings()

	// Set to number.
	if err := setSettingsValue(s, "project_id", "42"); err != nil {
		t.Fatal(err)
	}
	if s.ProjectID == nil || *s.ProjectID != 42 {
		t.Errorf("ProjectID = %v, want 42", s.ProjectID)
	}

	// Set to null.
	if err := setSettingsValue(s, "project_id", "null"); err != nil {
		t.Fatal(err)
	}
	if s.ProjectID != nil {
		t.Error("ProjectID should be nil after setting to null")
	}
}

func TestSetSettingsValue_ProjectID_NoneSync(t *testing.T) {
	s := config.NewNoneSettings()
	if err := setSettingsValue(s, "project_id", "42"); err == nil {
		t.Fatal("expected error setting project_id when sync=none")
	}
}

func TestSetSettingsValue_ProjectID_InvalidValue(t *testing.T) {
	s := config.NewCloudSettings()
	if err := setSettingsValue(s, "project_id", "abc"); err == nil {
		t.Fatal("expected error for non-numeric project_id")
	}
}

func TestSetSettingsValue_ArchcoreURL(t *testing.T) {
	s := config.NewOnPremSettings("http://old:8080")
	if err := setSettingsValue(s, "archcore_url", "http://new:9090/"); err != nil {
		t.Fatal(err)
	}
	if s.ArchcoreURL != "http://new:9090" {
		t.Errorf("ArchcoreURL = %q, want %q", s.ArchcoreURL, "http://new:9090")
	}
}

func TestSetSettingsValue_ArchcoreURL_SwitchesToOnPrem(t *testing.T) {
	tests := []struct {
		name    string
		initial *config.Settings
	}{
		{"from cloud", config.NewCloudSettings()},
		{"from none", config.NewNoneSettings()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			if err := setSettingsValue(s, "archcore_url", "http://x:8080"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Sync != config.SyncTypeOnPrem {
				t.Errorf("Sync = %q, want %q", s.Sync, config.SyncTypeOnPrem)
			}
			if s.ArchcoreURL != "http://x:8080" {
				t.Errorf("ArchcoreURL = %q, want %q", s.ArchcoreURL, "http://x:8080")
			}
		})
	}
}

func TestSetSettingsValue_ArchcoreURL_Empty(t *testing.T) {
	s := config.NewOnPremSettings("http://old:8080")
	if err := setSettingsValue(s, "archcore_url", ""); err == nil {
		t.Fatal("expected error for empty archcore_url")
	}
}

func TestGetSettingsValue_Language(t *testing.T) {
	tests := []struct {
		name string
		s    *config.Settings
		want string
	}{
		{"default when empty", config.NewNoneSettings(), "en"},
		{"set value", &config.Settings{Sync: "none", Language: "ru"}, "ru"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSettingsValue(tt.s, "language")
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSettingsValue_Language(t *testing.T) {
	s := config.NewNoneSettings()
	if err := setSettingsValue(s, "language", "ru"); err != nil {
		t.Fatal(err)
	}
	if s.Language != "ru" {
		t.Errorf("Language = %q, want %q", s.Language, "ru")
	}
}

func TestSetSettingsValue_Language_Empty(t *testing.T) {
	s := config.NewNoneSettings()
	if err := setSettingsValue(s, "language", ""); err == nil {
		t.Fatal("expected error for empty language")
	}
}

func TestConfigSetLanguage_Persists(t *testing.T) {
	dir := setupConfigTest(t, config.NewNoneSettings())

	s, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := setSettingsValue(s, "language", "ja"); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, s); err != nil {
		t.Fatal(err)
	}

	s2, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Language != "ja" {
		t.Errorf("Language = %q, want %q", s2.Language, "ja")
	}
}

func TestSetSettingsValue_UnknownKey(t *testing.T) {
	s := config.NewNoneSettings()
	if err := setSettingsValue(s, "unknown", "val"); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

// --- runConfig CLI flow tests ---

func TestRunConfig_NoArgs_PrintsSync(t *testing.T) {
	dir := setupConfigTest(t, config.NewNoneSettings())
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	cmd := newConfigCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfig_GetProjectID_ComingSoon(t *testing.T) {
	dir := setupConfigTest(t, config.NewNoneSettings())
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"get", "project_id"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for coming soon guard")
	}
	if !strings.Contains(err.Error(), "not available yet") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not available yet")
	}
}

func TestRunConfig_SetSync_ComingSoon(t *testing.T) {
	dir := setupConfigTest(t, config.NewNoneSettings())
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"set", "sync", "cloud"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for coming soon guard")
	}
	if !strings.Contains(err.Error(), "not available yet") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not available yet")
	}
}

// --- Integration: set + save + load ---

func TestConfigSetAndLoad(t *testing.T) {
	dir := setupConfigTest(t, config.NewCloudSettings())

	// Load, set project_id, save, reload.
	s, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := setSettingsValue(s, "project_id", "77"); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, s); err != nil {
		t.Fatal(err)
	}

	s2, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s2.ProjectID == nil || *s2.ProjectID != 77 {
		t.Errorf("ProjectID = %v, want 77", s2.ProjectID)
	}
}

func TestConfigSetArchcoreURL_SwitchesAndPersists(t *testing.T) {
	// Start with sync=none settings.
	dir := setupConfigTest(t, config.NewNoneSettings())

	s, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Setting archcore_url should auto-switch to on-prem.
	if err := setSettingsValue(s, "archcore_url", "http://localhost:8080"); err != nil {
		t.Fatalf("setSettingsValue: %v", err)
	}
	if s.Sync != config.SyncTypeOnPrem {
		t.Fatalf("Sync = %q, want %q", s.Sync, config.SyncTypeOnPrem)
	}

	// Should save and reload successfully.
	if err := config.Save(dir, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	s2, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s2.Sync != config.SyncTypeOnPrem {
		t.Errorf("Sync = %q, want %q", s2.Sync, config.SyncTypeOnPrem)
	}
	if s2.ArchcoreURL != "http://localhost:8080" {
		t.Errorf("ArchcoreURL = %q, want %q", s2.ArchcoreURL, "http://localhost:8080")
	}
}
