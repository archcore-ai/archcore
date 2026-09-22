package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"archcore-cli/internal/display"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/telemetry"
	"archcore-cli/internal/update"

	"github.com/spf13/cobra"
)

// --- update --check ----------------------------------------------------------
//
// A cheap, quiet freshness probe designed for hook/advisory use (the plugin's
// session-start advisory shells out to it): result read from and written to the
// shared freshness cache in internal/update, network request bounded by a short
// timeout, every failure silent. Output contract: exactly one line "update
// available: vX.Y.Z" when behind, nothing when current-or-unknown; exit code is
// always 0.
//
// The cache itself — its path and its two windows — belongs to internal/update,
// because the unattended policy reads the same file and a policy that reached
// back into cmd would be an import cycle. What stays here is the timeout: it
// bounds this command's network call, not the cache.

// updateCheckTimeout must absorb a cold TLS handshake to github.com; the
// negative cache keeps the worst case to one stall per failure window.
const updateCheckTimeout = 2 * time.Second

// runUpdateCheck implements `archcore update --check`. Never fails: a network
// or cache problem just means no output. A failed fetch is negative-cached
// (empty stamp) so hooks don't re-pay the timeout until the failure window
// lapses.
func runUpdateCheck(ctx context.Context, w io.Writer, version string, u *update.Updater, cachePath string) {
	latest, fresh := update.ReadCachedLatest(cachePath)
	if !fresh {
		checkCtx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
		defer cancel()
		fetched, err := u.CheckLatest(checkCtx)
		if err != nil {
			update.WriteCachedLatest(cachePath, "") // failure stamp
			return                                  // silent by contract — this runs inside hooks
		}
		latest = fetched
		update.WriteCachedLatest(cachePath, latest)
	}
	if latest != "" && update.NeedsUpdate(version, latest) {
		fmt.Fprintf(w, "update available: %s\n", latest)
	}
}

// --- the plugin step ----------------------------------------------------------
//
// After the binary phase, `archcore update` refreshes the Archcore plugin on
// each host that has it. The step is the update verb of the shared plugin core
// in @cmd/plugin_run.go, which `archcore plugin update` reaches through its own
// command: one planner, one executor, two entry points that differ only in the
// lines around the run — updating-the-plugin.spec.
//
// The step has exactly two call sites, and where it is NOT called is the larger
// half of the contract:
//   - not under --check (§4). --check is the hook-facing probe on a 2 s budget,
//     and a 120 s plugin step behind an advisory that fires on every session
//     start would be paid on every session start.
//   - not after a binary phase that failed (§3), on either return.

// runPluginUpdateStep refreshes the plugin on each host whose own listing names
// it. Everything else — which hosts, which tier, what to print — belongs to the
// planner and the shared core.
//
// It returns nothing, which is §13 as a signature rather than a convention: the
// step must not change the exit code of `archcore update`, and an outcome it
// cannot hand back is an outcome no call site can accidentally return. It sends
// no telemetry event either (§14), which is what keeps the one-event-per-
// invocation invariant of cli-update-telemetry.spec intact.
//
// It prints no preamble. A machine that never installed the plugin must see no
// plugin output at all, and a header would be output. It hands over no commands
// for a host the step bound cut it short of either — Failure Behavior 5 of
// updating-the-plugin.spec prints nothing for the hosts an update skipped, which
// is what keeps a step running behind another command from growing output on a
// slow machine.
//
// What it does owe is the overlap notice, on the same terms as every other entry
// point that mutates a plugin: a refreshed plugin puts a second set of hooks into
// the session the user already has open, and the command that caused it does not
// change that — plugin-delivery.spec §15.
func runPluginUpdateStep(ctx context.Context, out io.Writer) {
	outcome := runPluginActions(ctx, out, pluginRunOptions{Verb: plugin.VerbUpdate})
	reportSelfCausedPluginConflict(out, plugin.VerbUpdate, outcome)
}

// --- telemetry on the manual path --------------------------------------------
//
// An `archcore update` a user typed reports its outcome, so how far a release
// actually reaches is measurable. Every property here is graded `manual`, which
// is what keeps evidence of user intent apart from evidence that the unattended
// mechanism ran — cli-update-telemetry.spec §10.

// The event names and the trigger come from internal/telemetry, which owns the
// wire vocabulary. This file once declared its own untyped copies beside the
// unattended policy's untyped copies — two spellings of one PostHog contract,
// with nothing that could fail a build when they drifted.
const (
	telemetryEventUpdated      = telemetry.EventUpdated
	telemetryEventUpdateFailed = telemetry.EventUpdateFailed

	// telemetryTriggerManual grades an invocation a user typed.
	telemetryTriggerManual = telemetry.TriggerManual
)

const (
	// telemetryNotice is printed once the endpoint has accepted a manual event,
	// so a user learns something left the machine in the same run it left. It
	// names an opt-out variable and the privacy page, and repeats the wording
	// install.sh prints for the install ping so the two read as one policy —
	// cli-update-telemetry.spec §13.
	//
	// "an event", not "a successful update": a delivered cli_update_failed
	// carries the install identifier and the same machine facts off the box as
	// cli_updated does, so the run that sends one owes the reader the same
	// notice. Both manual call sites therefore print it, and neither prints it
	// when nothing was delivered — §14.
	telemetryNotice = "Anonymous update event sent (no personal data). Opt out with DO_NOT_TRACK=1 — https://archcore.ai/privacy"
)

// disclose prints the notice when the endpoint accepted the event, and nothing
// when it did not — cli-update-telemetry.spec §13, §14.
//
// Every manual call site passes its Capture result through here, which is what
// keeps the one-notice invariant readable: the command sends at most one event,
// and this is the only place the notice is rendered.
func disclose(out io.Writer, sent bool) {
	if sent {
		fmt.Fprintln(out, display.Dim.Render("  "+telemetryNotice))
	}
}

// telemetryStage names the failed step for the event's `stage` property.
//
// The reading itself is update.FailureStage: the unattended policy reports the
// same property off the same typed errors, and this file carried a
// byte-identical copy of the function and its rationale. The rule is that the
// message never travels — it routinely holds a path, an archive name or a URL,
// and the spec forbids transmitting any of them — so the stage is read off the
// typed error and `check` is the honest default for an untagged one.
func telemetryStage(err error) string { return update.FailureStage(err) }

// updateDeps builds what a typed `archcore update` runs with: the updater that
// talks to the release host, and the sender that reports the outcome.
//
// It is a variable so a test can run newUpdateCmd itself against its own two
// servers. Nothing else here can: every other test builds the command through
// buildUpdateCmd and therefore supplies its own sender, which leaves the one
// line that wires the real one unexercised. That line fails silently — a
// command built without a sender behaves exactly like an inert build, with no
// event, no disclosure and no error — so a release could ship with the manual
// path dark and every test still green.
var updateDeps = func(version string) (*update.Updater, *telemetry.Client) {
	return update.NewUpdater(version, "archcore-ai/cli", "archcore"), telemetry.NewClient(version)
}

func newUpdateCmd(version string) *cobra.Command {
	u, tel := updateDeps(version)
	return buildUpdateCmd(version, u, tel)
}

// newUpdateCmdWithClient creates an update command that uses a custom HTTP
// client. This is used for testing to inject a mock server. It sends no
// telemetry: Capture is nil-receiver safe, so a nil client is the "telemetry
// off" case a test gets for free.
func newUpdateCmdWithClient(version string, client *http.Client) *cobra.Command {
	u := &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
	}
	return buildUpdateCmd(version, u, nil)
}

// buildUpdateCmd wires the update command. A nil tel means no events, which is
// the shape a test and an inert build both take.
func buildUpdateCmd(version string, u *update.Updater, tel *telemetry.Client) *cobra.Command {
	var checkFlag bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update archcore to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Every human-readable line goes to the command's writer, not to the
			// process stdout: a caller that redirects the command — the root's
			// SetOut, a test buffer — must see all of the output or none of it.
			out := cmd.OutOrStdout()

			if checkFlag {
				// --check is the hook-facing probe and sends nothing, so an
				// advisory that runs on every session start cannot inflate the
				// series — cli-update-telemetry.spec §7.
				runUpdateCheck(ctx, out, version, u, update.CachePath())
				return nil
			}

			fmt.Fprintln(out, display.Banner(version))
			fmt.Fprintln(out)
			fmt.Fprintln(out, display.Dim.Render("  Checking for updates..."))

			latest, err := u.CheckLatest(ctx)
			if err != nil {
				fmt.Fprintln(out, display.FailLine("Could not check for updates"))
				fmt.Fprintln(out, display.HintLine(err.Error()))
				// No to_version: the tag never resolved, and a placeholder
				// would read in the data as a version this run really aimed at
				// — cli-update-telemetry.spec.
				disclose(out, tel.Capture(ctx, telemetryEventUpdateFailed, map[string]any{
					"trigger": telemetryTriggerManual,
					"stage":   telemetryStage(err),
				}))
				// Updating is this command's sole job — scripts must see a
				// non-zero exit on failure. The details are already printed, so
				// signal exit-only (mirrors status/doctor) and let main avoid a
				// second stderr copy.
				return ErrAlreadyReported
			}

			fmt.Fprintln(out, display.CheckLine(fmt.Sprintf("Current: %s", version)))
			fmt.Fprintln(out, display.CheckLine(fmt.Sprintf("Latest:  %s", latest)))

			if !update.NeedsUpdate(version, latest) {
				fmt.Fprintln(out)
				fmt.Fprintln(out, display.CheckLine(fmt.Sprintf("Already up to date (%s)", version)))
				// A current binary is not a current plugin: the two are
				// released apart, so the step runs on this branch as much as
				// on the one that replaced the binary —
				// updating-the-plugin.spec §2.
				runPluginUpdateStep(ctx, out)
				// Nothing was updated and nothing failed, so this invocation
				// has no outcome to report — cli-update-telemetry.spec §6.
				return nil
			}

			fmt.Fprintln(out)

			archive := update.ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintln(out, display.Dim.Render(fmt.Sprintf("  Downloading %s...", archive)))

			if err := u.Apply(ctx, latest); err != nil {
				fmt.Fprintln(out, display.FailLine("Update failed"))
				fmt.Fprintln(out, display.HintLine(err.Error()))
				disclose(out, tel.Capture(ctx, telemetryEventUpdateFailed, map[string]any{
					"trigger":      telemetryTriggerManual,
					"stage":        telemetryStage(err),
					"from_version": version,
					"to_version":   latest,
				}))
				return ErrAlreadyReported
			}

			fmt.Fprintln(out, display.CheckLine("Checksum verified"))
			fmt.Fprintln(out, display.CheckLine(fmt.Sprintf("Updated to %s", latest)))

			// The disclosure is gated on delivery, so a machine that sent
			// nothing — opted out, inert build, endpoint down — is told nothing
			// was sent — cli-update-telemetry.spec §14.
			disclose(out, tel.Capture(ctx, telemetryEventUpdated, map[string]any{
				"trigger":      telemetryTriggerManual,
				"from_version": version,
				"to_version":   latest,
			}))

			// Last, after the binary phase reported its own result —
			// updating-the-plugin.spec §1. It follows the disclosure rather
			// than preceding it so the event this run sent stays attached to
			// the lines that describe it; the step itself sends none.
			runPluginUpdateStep(ctx, out)

			return nil
		},
	}

	cmd.Flags().BoolVar(&checkFlag, "check", false,
		"quietly report whether a newer version exists (cached 24h, short network timeout, always exit 0) — designed for hooks/advisories")
	return cmd
}
