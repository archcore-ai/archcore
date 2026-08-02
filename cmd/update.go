package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"archcore-cli/internal/display"
	"archcore-cli/internal/update"

	"github.com/spf13/cobra"
)

// --- update --check ----------------------------------------------------------
//
// A cheap, quiet freshness probe designed for hook/advisory use (the plugin's
// session-start advisory shells out to it): result cached in XDG state with a
// 24h TTL, network request bounded by a short timeout, every failure silent.
// Output contract: exactly one line "update available: vX.Y.Z" when behind,
// nothing when current-or-unknown; exit code is always 0.

const (
	updateCheckTTL = 24 * time.Hour
	// updateCheckFailureTTL is the negative-cache window: a failed check is
	// stamped (empty cache content) so sessions on a slow or offline network
	// don't pay the probe timeout on every hook invocation.
	updateCheckFailureTTL = time.Hour
	// updateCheckTimeout must absorb a cold TLS handshake to github.com;
	// the negative cache keeps the worst case to one stall per failure TTL.
	updateCheckTimeout = 2 * time.Second
)

// updateCheckCachePath returns the cache file, honoring XDG_STATE_HOME.
func updateCheckCachePath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "archcore", "last-update-check")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "archcore", "last-update-check")
}

// readCachedLatest returns the cached latest-version string and whether the
// cache is fresh. Empty content is a failure stamp (negative cache) with its
// own, shorter TTL: latest == "" with fresh == true means "recent check
// failed, skip the network silently".
func readCachedLatest(path string) (latest string, fresh bool) {
	if path == "" {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	latest = strings.TrimSpace(string(data))
	ttl := updateCheckTTL
	if latest == "" {
		ttl = updateCheckFailureTTL
	}
	return latest, time.Since(info.ModTime()) < ttl
}

// writeCachedLatest stores latest, best-effort.
func writeCachedLatest(path, latest string) {
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	_ = os.WriteFile(path, []byte(latest+"\n"), 0o644)
}

// runUpdateCheck implements `archcore update --check`. Never fails: a network
// or cache problem just means no output. A failed fetch is negative-cached
// (empty stamp) so hooks don't re-pay the timeout until the failure TTL lapses.
func runUpdateCheck(ctx context.Context, w io.Writer, version string, u *update.Updater, cachePath string) {
	latest, fresh := readCachedLatest(cachePath)
	if !fresh {
		checkCtx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
		defer cancel()
		fetched, err := u.CheckLatest(checkCtx)
		if err != nil {
			writeCachedLatest(cachePath, "") // failure stamp
			return                           // silent by contract — this runs inside hooks
		}
		latest = fetched
		writeCachedLatest(cachePath, latest)
	}
	if latest != "" && update.NeedsUpdate(version, latest) {
		fmt.Fprintf(w, "update available: %s\n", latest)
	}
}

func newUpdateCmd(version string) *cobra.Command {
	u := update.NewUpdater(version, "archcore-ai/cli", "archcore")
	return buildUpdateCmd(version, u)
}

// newUpdateCmdWithClient creates an update command that uses a custom HTTP
// client. This is used for testing to inject a mock server.
func newUpdateCmdWithClient(version string, client *http.Client) *cobra.Command {
	u := &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
	}
	return buildUpdateCmd(version, u)
}

func buildUpdateCmd(version string, u *update.Updater) *cobra.Command {
	var checkFlag bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update archcore to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if checkFlag {
				runUpdateCheck(ctx, cmd.OutOrStdout(), version, u, updateCheckCachePath())
				return nil
			}

			fmt.Println(display.Banner(version))
			fmt.Println()
			fmt.Println(display.Dim.Render("  Checking for updates..."))

			latest, err := u.CheckLatest(ctx)
			if err != nil {
				fmt.Println(display.FailLine("Could not check for updates"))
				fmt.Println(display.HintLine(err.Error()))
				// Updating is this command's sole job — scripts must see a
				// non-zero exit on failure. The details are already printed, so
				// signal exit-only (mirrors status/doctor) and let main avoid a
				// second stderr copy.
				return ErrAlreadyReported
			}

			fmt.Println(display.CheckLine(fmt.Sprintf("Current: %s", version)))
			fmt.Println(display.CheckLine(fmt.Sprintf("Latest:  %s", latest)))

			if !update.NeedsUpdate(version, latest) {
				fmt.Println()
				fmt.Println(display.CheckLine(fmt.Sprintf("Already up to date (%s)", version)))
				return nil
			}

			fmt.Println()

			archive := update.ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
			fmt.Println(display.Dim.Render(fmt.Sprintf("  Downloading %s...", archive)))

			if err := u.Apply(ctx, latest); err != nil {
				fmt.Println(display.FailLine("Update failed"))
				fmt.Println(display.HintLine(err.Error()))
				return ErrAlreadyReported
			}

			fmt.Println(display.CheckLine("Checksum verified"))
			fmt.Println(display.CheckLine(fmt.Sprintf("Updated to %s", latest)))

			return nil
		},
	}

	cmd.Flags().BoolVar(&checkFlag, "check", false,
		"quietly report whether a newer version exists (cached 24h, short network timeout, always exit 0) — designed for hooks/advisories")
	return cmd
}
