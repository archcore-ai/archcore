package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	archsync "archcore-cli/internal/sync"
)

// mockSyncClient implements syncClient for tests.
type mockSyncClient struct {
	called         bool
	payload        *archsync.Payload
	resp           *api.SyncResponse
	projectCreated bool
	err            error
}

func (m *mockSyncClient) Sync(_ context.Context, payload *archsync.Payload) (*api.SyncResponse, bool, error) {
	m.called = true
	m.payload = payload
	return m.resp, m.projectCreated, m.err
}

// setupSyncTestDir creates a .archcore directory with settings and optional document files.
func setupSyncTestDir(t *testing.T) string {
	t.Helper()
	baseDir := t.TempDir()
	if err := config.InitDir(baseDir); err != nil {
		t.Fatalf("InitDir: %v", err)
	}
	pid := 1
	s := config.NewCloudSettings()
	s.ProjectID = &pid
	if err := config.Save(baseDir, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return baseDir
}

// setupSyncTestDirNoPID creates a .archcore directory with cloud sync but no project_id.
func setupSyncTestDirNoPID(t *testing.T) string {
	t.Helper()
	baseDir := t.TempDir()
	if err := config.InitDir(baseDir); err != nil {
		t.Fatalf("InitDir: %v", err)
	}
	s := config.NewCloudSettings()
	if err := config.Save(baseDir, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return baseDir
}

func writeSyncDoc(t *testing.T, baseDir, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(baseDir, ".archcore", relPath)
	os.MkdirAll(filepath.Dir(absPath), 0o755)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

func testPreconditions(baseDir string) *syncPreconditions {
	pid := 1
	return &syncPreconditions{
		Settings:  &config.Settings{Sync: config.SyncTypeCloud, ProjectID: &pid},
		ProjectID: &pid,
		ServerURL: "http://localhost",
		BaseDir:   baseDir,
	}
}

func testPreconditionsNoPID(baseDir string) *syncPreconditions {
	return &syncPreconditions{
		Settings:  &config.Settings{Sync: config.SyncTypeCloud},
		ProjectID: nil,
		ServerURL: "http://localhost",
		BaseDir:   baseDir,
	}
}

func TestCheckSyncPreconditions(t *testing.T) {
	pid := 42

	tests := []struct {
		name        string
		setup       func(t *testing.T, baseDir string)
		wantErr     bool
		errContains string
		wantPID     *int
	}{
		{
			name:        "no .archcore dir",
			setup:       func(t *testing.T, baseDir string) {},
			wantErr:     true,
			errContains: ".archcore/",
		},
		{
			name: "sync mode none",
			setup: func(t *testing.T, baseDir string) {
				config.InitDir(baseDir)
				config.Save(baseDir, config.NewNoneSettings())
			},
			wantErr:     true,
			errContains: "sync is disabled",
		},
		{
			name: "valid cloud with project_id",
			setup: func(t *testing.T, baseDir string) {
				config.InitDir(baseDir)
				s := config.NewCloudSettings()
				s.ProjectID = &pid
				config.Save(baseDir, s)
			},
			wantErr: false,
			wantPID: &pid,
		},
		{
			name: "valid cloud without project_id",
			setup: func(t *testing.T, baseDir string) {
				config.InitDir(baseDir)
				config.Save(baseDir, config.NewCloudSettings())
			},
			wantErr: false,
			wantPID: nil,
		},
		{
			name: "valid on-prem with project_id",
			setup: func(t *testing.T, baseDir string) {
				config.InitDir(baseDir)
				s := config.NewOnPremSettings("http://server:8080")
				s.ProjectID = &pid
				config.Save(baseDir, s)
			},
			wantErr: false,
			wantPID: &pid,
		},
		{
			name: "valid on-prem without project_id",
			setup: func(t *testing.T, baseDir string) {
				config.InitDir(baseDir)
				config.Save(baseDir, config.NewOnPremSettings("http://server:8080"))
			},
			wantErr: false,
			wantPID: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			tt.setup(t, baseDir)

			pre, err := checkSyncPreconditions(baseDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if (pre.ProjectID == nil) != (tt.wantPID == nil) {
				t.Errorf("ProjectID nil = %v, want %v", pre.ProjectID == nil, tt.wantPID == nil)
			} else if pre.ProjectID != nil && *pre.ProjectID != *tt.wantPID {
				t.Errorf("ProjectID = %d, want %d", *pre.ProjectID, *tt.wantPID)
			}
			if pre.ServerURL == "" {
				t.Error("ServerURL should not be empty")
			}
		})
	}
}

func TestCheckSyncPreconditions_CloudServerURL(t *testing.T) {
	pid := 1
	baseDir := t.TempDir()
	config.InitDir(baseDir)
	s := config.NewCloudSettings()
	s.ProjectID = &pid
	config.Save(baseDir, s)

	pre, err := checkSyncPreconditions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pre.ServerURL != config.CloudServerURL {
		t.Errorf("ServerURL = %q, want %q", pre.ServerURL, config.CloudServerURL)
	}
}

func TestCheckSyncPreconditions_OnPremServerURL(t *testing.T) {
	pid := 1
	baseDir := t.TempDir()
	config.InitDir(baseDir)
	s := config.NewOnPremSettings("http://custom:9090")
	s.ProjectID = &pid
	config.Save(baseDir, s)

	pre, err := checkSyncPreconditions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pre.ServerURL != "http://custom:9090" {
		t.Errorf("ServerURL = %q, want %q", pre.ServerURL, "http://custom:9090")
	}
}

func TestRunSync_DryRun_DoesNotUpdateManifest(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/test.adr.md", "# Test ADR")

	mock := &mockSyncClient{}
	flags := &syncFlags{DryRun: true, CI: true}

	err := doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	// API should not have been called.
	if mock.called {
		t.Error("Sync should not be called in dry-run mode")
	}

	// Manifest should not exist (first sync, never saved).
	manifestPath := filepath.Join(baseDir, ".archcore", ".sync-state.json")
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("manifest should not exist after dry-run")
	}
}

func TestRunSync_Force_ResyncsUnchangedFiles(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/test.adr.md", "# Test ADR")

	// Create a manifest that already has the file with the correct hash,
	// so a normal sync would consider it unchanged.
	currentFiles, err := archsync.ScanFiles(baseDir)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}
	m := archsync.NewManifest()
	for _, f := range currentFiles {
		m.Files[f.RelPath] = f.Hash
	}
	if err := archsync.SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 1,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/test.adr.md", Action: "modified"}},
		},
	}
	flags := &syncFlags{Force: true, CI: true}

	err = doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if !mock.called {
		t.Fatal("Sync should be called with --force even when files are unchanged")
	}
	if len(mock.payload.Modified) == 0 {
		t.Fatal("payload should contain modified files when --force is used")
	}
}

func TestRunSync_Force_DetectsDeletions(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/existing.adr.md", "# Existing")

	// Manifest has an extra file that no longer exists on disk.
	m := archsync.NewManifest()
	m.Files["vision/existing.adr.md"] = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	m.Files["vision/deleted.adr.md"] = "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5"
	if err := archsync.SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 1,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/existing.adr.md", Action: "modified"}},
			Deleted:   []string{"vision/deleted.adr.md"},
		},
	}
	flags := &syncFlags{Force: true, CI: true}

	err := doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if !mock.called {
		t.Fatal("Sync should be called")
	}

	// Expect modified files and deleted paths.
	if len(mock.payload.Modified) == 0 {
		t.Error("expected modified files in payload")
	}
	hasDeleted := false
	for _, d := range mock.payload.Deleted {
		if d == "vision/deleted.adr.md" {
			hasDeleted = true
		}
	}
	if !hasDeleted {
		t.Error("expected deleted.adr.md in payload with --force")
	}

	// Verify manifest no longer has the deleted file.
	loaded, err := archsync.LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := loaded.Files["vision/deleted.adr.md"]; ok {
		t.Error("deleted file should be removed from manifest after sync")
	}
}

func TestRunSync_NoChanges_ShortCircuit(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/test.adr.md", "# Test ADR")

	// Create manifest that exactly matches on-disk state.
	currentFiles, err := archsync.ScanFiles(baseDir)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}
	m := archsync.NewManifest()
	for _, f := range currentFiles {
		m.Files[f.RelPath] = f.Hash
	}
	if err := archsync.SaveManifest(baseDir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	mock := &mockSyncClient{}
	flags := &syncFlags{CI: true}

	err = doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if mock.called {
		t.Error("Sync should not be called when there are no changes")
	}
}

func TestRunSync_EndToEnd(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/plan.plan.md", "# Plan")
	writeSyncDoc(t, baseDir, "knowledge/adr.adr.md", "# ADR")

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 1,
			Accepted: []api.SyncAcceptedEntry{
				{Path: "vision/plan.plan.md", Action: "created"},
				{Path: "knowledge/adr.adr.md", Action: "created"},
			},
		},
	}
	flags := &syncFlags{CI: true}

	err := doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	// Verify API was called with both files.
	if !mock.called {
		t.Fatal("Sync should be called")
	}
	if len(mock.payload.Created) != 2 {
		t.Fatalf("payload has %d created files, want 2", len(mock.payload.Created))
	}

	// Verify both files have content.
	for _, f := range mock.payload.Created {
		if f.Content == "" {
			t.Errorf("file %s should have content", f.Path)
		}
	}

	// Verify project_id was set on payload.
	if mock.payload.ProjectID == nil || *mock.payload.ProjectID != 1 {
		t.Errorf("payload ProjectID = %v, want 1", mock.payload.ProjectID)
	}

	// Verify manifest was updated.
	loaded, err := archsync.LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(loaded.Files) != 2 {
		t.Errorf("manifest has %d files, want 2", len(loaded.Files))
	}
}

func TestRunSync_AutoCreateProject(t *testing.T) {
	baseDir := setupSyncTestDirNoPID(t)
	writeSyncDoc(t, baseDir, "vision/plan.plan.md", "# Plan")

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 99,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/plan.plan.md", Action: "created"}},
		},
		projectCreated: true,
	}
	flags := &syncFlags{CI: true}

	pre := testPreconditionsNoPID(baseDir)
	pre.Settings = &config.Settings{Sync: config.SyncTypeCloud}

	err := doSync(context.Background(), baseDir, flags, pre, mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if !mock.called {
		t.Fatal("Sync should be called")
	}

	// Payload should have project_name, not project_id.
	if mock.payload.ProjectID != nil {
		t.Errorf("payload ProjectID should be nil, got %v", mock.payload.ProjectID)
	}
	if mock.payload.ProjectName == nil {
		t.Fatal("payload ProjectName should be set")
	}

	// Verify project_id was persisted to settings.
	loaded, err := config.Load(baseDir)
	if err != nil {
		t.Fatalf("Load settings: %v", err)
	}
	if loaded.ProjectID == nil || *loaded.ProjectID != 99 {
		t.Errorf("settings ProjectID = %v, want 99", loaded.ProjectID)
	}
}

func TestRunSync_PayloadHasFrontmatter(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	content := "---\ntitle: My ADR\nstatus: accepted\n---\n\n# Body"
	writeSyncDoc(t, baseDir, "vision/test.adr.md", content)

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 1,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/test.adr.md", Action: "created"}},
		},
	}
	flags := &syncFlags{CI: true}

	err := doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if len(mock.payload.Created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(mock.payload.Created))
	}
	fe := mock.payload.Created[0]
	if fe.Frontmatter.Title != "My ADR" {
		t.Errorf("frontmatter title = %q, want %q", fe.Frontmatter.Title, "My ADR")
	}
	if fe.Frontmatter.Status != "accepted" {
		t.Errorf("frontmatter status = %q, want %q", fe.Frontmatter.Status, "accepted")
	}
	if fe.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
}

func TestDeriveProjectName(t *testing.T) {
	got := deriveProjectName("/home/user/my-project")
	if got != "my-project" {
		t.Errorf("deriveProjectName = %q, want %q", got, "my-project")
	}
}

func TestRunSync_AutoCreate_NoGitRepo_RepoURLNil(t *testing.T) {
	baseDir := setupSyncTestDirNoPID(t)
	writeSyncDoc(t, baseDir, "vision/plan.plan.md", "# Plan")

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 99,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/plan.plan.md", Action: "created"}},
		},
		projectCreated: true,
	}
	flags := &syncFlags{CI: true}

	pre := testPreconditionsNoPID(baseDir)
	pre.Settings = &config.Settings{Sync: config.SyncTypeCloud}

	err := doSync(context.Background(), baseDir, flags, pre, mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if mock.payload.RepoURL != nil {
		t.Errorf("RepoURL should be nil for non-git dir, got %q", *mock.payload.RepoURL)
	}
}

func TestRunSync_AutoCreate_WithGitRepo_RepoURLPopulated(t *testing.T) {
	baseDir := setupSyncTestDirNoPID(t)
	writeSyncDoc(t, baseDir, "vision/plan.plan.md", "# Plan")

	// Initialize a git repo with a remote in the test directory.
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/repo.git"},
	} {
		cmd := exec.Command("git", append([]string{"-C", baseDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 99,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/plan.plan.md", Action: "created"}},
		},
		projectCreated: true,
	}
	flags := &syncFlags{CI: true}

	pre := testPreconditionsNoPID(baseDir)
	pre.Settings = &config.Settings{Sync: config.SyncTypeCloud}

	err := doSync(context.Background(), baseDir, flags, pre, mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if mock.payload.RepoURL == nil {
		t.Fatal("RepoURL should be set for git dir with origin remote")
	}
	want := "https://github.com/example/repo.git"
	if *mock.payload.RepoURL != want {
		t.Errorf("RepoURL = %q, want %q", *mock.payload.RepoURL, want)
	}
}

func TestRunSync_ExistingProject_NoRepoURL(t *testing.T) {
	baseDir := setupSyncTestDir(t)
	writeSyncDoc(t, baseDir, "vision/plan.plan.md", "# Plan")

	// Initialize a git repo — RepoURL should still be nil because project_id is set.
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/repo.git"},
	} {
		cmd := exec.Command("git", append([]string{"-C", baseDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	mock := &mockSyncClient{
		resp: &api.SyncResponse{
			ProjectID: 1,
			Accepted:  []api.SyncAcceptedEntry{{Path: "vision/plan.plan.md", Action: "created"}},
		},
	}
	flags := &syncFlags{CI: true}

	err := doSync(context.Background(), baseDir, flags, testPreconditions(baseDir), mock)
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	if mock.payload.RepoURL != nil {
		t.Errorf("RepoURL should be nil for existing project, got %q", *mock.payload.RepoURL)
	}
}
