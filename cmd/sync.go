package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/git"
	archsync "archcore-cli/internal/sync"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// syncClient abstracts the sync API call for testability.
type syncClient interface {
	Sync(ctx context.Context, payload *archsync.SyncPayload) (*api.SyncResponse, bool, error)
}

// syncPreconditions holds validated configuration for a sync operation.
type syncPreconditions struct {
	Settings  *config.Settings
	ProjectID *int // nil = auto-create
	ServerURL string
	BaseDir   string
}

// checkSyncPreconditions validates all requirements before sync can proceed.
func checkSyncPreconditions(baseDir string) (*syncPreconditions, error) {
	if !config.DirExists(baseDir) {
		return nil, fmt.Errorf(".archcore/ directory not found — run 'archcore init' first")
	}

	settings, err := config.Load(baseDir)
	if err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	if settings.Sync == config.SyncTypeNone {
		return nil, fmt.Errorf("sync is disabled — run 'archcore config set sync cloud' or 'archcore init' to configure")
	}

	return &syncPreconditions{
		Settings:  settings,
		ProjectID: settings.ProjectID,
		ServerURL: settings.ServerURL(),
		BaseDir:   baseDir,
	}, nil
}

// deriveProjectName returns the directory name as a fallback project name.
func deriveProjectName(baseDir string) string {
	return filepath.Base(baseDir)
}

type syncFlags struct {
	DryRun bool
	Force  bool
	CI     bool
}

func newSyncCmd() *cobra.Command {
	flags := &syncFlags{}

	cmd := &cobra.Command{
		Use:    "sync",
		Short:  "Push local documents to the Archcore server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("sync is not available yet — this feature is coming soon")
		},
	}

	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Show what would be synced without sending")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Sync all files regardless of change status")
	cmd.Flags().BoolVar(&flags.CI, "ci", false, "Non-interactive mode for CI/CD pipelines")

	return cmd
}

func runSync(cmd *cobra.Command, flags *syncFlags) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 1. Validate preconditions.
	pre, err := checkSyncPreconditions(cwd)
	if err != nil {
		if flags.CI {
			return err
		}
		fmt.Println(display.FailLine(err.Error()))
		return nil
	}

	client := api.NewClient(pre.ServerURL)
	return doSync(cmd.Context(), cwd, flags, pre, client)
}

// doSync contains the core sync logic, separated from cobra and os.Getwd for testability.
func doSync(ctx context.Context, baseDir string, flags *syncFlags, pre *syncPreconditions, client syncClient) error {
	// 2. Load manifest and scan files.
	manifest, err := archsync.LoadManifest(baseDir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	currentFiles, err := archsync.ScanFiles(baseDir)
	if err != nil {
		return fmt.Errorf("scanning files: %w", err)
	}

	// 3. Calculate diff.
	var diffEntries []archsync.DiffEntry
	if flags.Force {
		// Treat all current files as modified (full re-sync).
		currentSet := make(map[string]bool, len(currentFiles))
		for _, f := range currentFiles {
			currentSet[f.RelPath] = true
			diffEntries = append(diffEntries, archsync.DiffEntry{
				RelPath: f.RelPath,
				Action:  archsync.ActionModified,
				Hash:    f.Hash,
			})
		}
		// Detect deletions: files in manifest but no longer on disk.
		for relPath := range manifest.Files {
			if !currentSet[relPath] {
				diffEntries = append(diffEntries, archsync.DiffEntry{
					RelPath: relPath,
					Action:  archsync.ActionDeleted,
				})
			}
		}
	} else {
		diffEntries = archsync.Diff(currentFiles, manifest)
	}

	// 4. Report diff summary.
	created := archsync.FilterByAction(diffEntries, archsync.ActionCreated)
	modified := archsync.FilterByAction(diffEntries, archsync.ActionModified)
	deleted := archsync.FilterByAction(diffEntries, archsync.ActionDeleted)

	if !archsync.HasChanges(diffEntries) {
		fmt.Println(display.CheckLine("Everything is up to date"))
		return nil
	}

	fmt.Println(display.Banner())
	fmt.Println()
	if len(created) > 0 {
		fmt.Println(display.CheckLine(fmt.Sprintf("%d new file(s)", len(created))))
		for _, e := range created {
			fmt.Println(display.HintLine("+ " + e.RelPath))
		}
	}
	if len(modified) > 0 {
		fmt.Println(display.WarnLine(fmt.Sprintf("%d modified file(s)", len(modified))))
		for _, e := range modified {
			fmt.Println(display.HintLine("~ " + e.RelPath))
		}
	}
	if len(deleted) > 0 {
		fmt.Println(display.FailLine(fmt.Sprintf("%d deleted file(s)", len(deleted))))
		for _, e := range deleted {
			fmt.Println(display.HintLine("- " + e.RelPath))
		}
	}
	fmt.Println()

	// 5. Dry-run exit point.
	if flags.DryRun {
		fmt.Println(display.Dim.Render("  Dry run — no changes sent to server."))
		return nil
	}

	// 6. Interactive confirmation (skip in CI mode).
	if !flags.CI {
		var confirm bool
		err := huh.NewConfirm().
			Title("Push these changes to the server?").
			Value(&confirm).
			Run()
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println(display.Dim.Render("  Cancelled."))
			return nil
		}
	}

	// 7. Build payload and send.
	payload, err := archsync.BuildPayload(baseDir, diffEntries)
	if err != nil {
		return fmt.Errorf("building sync payload: %w", err)
	}

	// Set project_id or project_name on the payload.
	if pre.ProjectID != nil {
		payload.ProjectID = pre.ProjectID
	} else {
		name := deriveProjectName(baseDir)
		payload.ProjectName = &name

		if repoURL := git.DetectRepoURL(baseDir); repoURL != "" {
			payload.RepoURL = &repoURL
		}
	}

	resp, projectCreated, err := client.Sync(ctx, payload)
	if err != nil {
		if flags.CI {
			return fmt.Errorf("sync failed: %w", err)
		}
		fmt.Println(display.FailLine("Sync failed"))
		fmt.Println(display.HintLine(err.Error()))
		return nil
	}

	// 7b. If the server auto-created the project, save project_id to settings.
	if projectCreated && resp.ProjectID != 0 {
		newPID := int(resp.ProjectID)
		pre.Settings.ProjectID = &newPID
		if err := config.Save(baseDir, pre.Settings); err != nil {
			return fmt.Errorf("saving project_id to settings: %w", err)
		}
	}

	// 8. Update manifest with new hashes.
	for _, e := range diffEntries {
		switch e.Action {
		case archsync.ActionCreated, archsync.ActionModified:
			manifest.Files[e.RelPath] = e.Hash
		case archsync.ActionDeleted:
			delete(manifest.Files, e.RelPath)
		}
	}
	if err := archsync.SaveManifest(baseDir, manifest); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	// 9. Report success.
	acceptedCount := len(resp.Accepted)
	if len(resp.Errors) > 0 {
		fmt.Println(display.WarnLine(fmt.Sprintf("Synced %d file(s), %d error(s)", acceptedCount, len(resp.Errors))))
		for _, e := range resp.Errors {
			fmt.Println(display.FailLine(fmt.Sprintf("  %s: %s", e.Path, e.Message)))
		}
	} else {
		fmt.Println(display.CheckLine(fmt.Sprintf("Synced %d file(s)", acceptedCount)))
	}
	if projectCreated {
		fmt.Println(display.CheckLine(fmt.Sprintf("Project created (id: %d)", resp.ProjectID)))
	}
	return nil
}
