package update

// The unattended-update policy: every condition that must hold before this
// process replaces its own binary with nobody watching — unattended-update.spec.
// The trigger supplies the moment; this file supplies the conditions, in the
// fixed order §1 and §2 require.
//
// Three properties shape the code below and are easy to erase in a refactor:
//
//  1. Every refusal before the claim is silent. A refusal is not a failure: an
//     event sent for one would count forks, development builds and CI runners as
//     machines that tried and could not, and `cli_update_failed` would stop
//     meaning "an attempted step did not complete" —
//     cli-update-telemetry.spec.
//  2. Both reportable skips sit after the claim. That is what makes "at most one
//     cli_update_skipped per machine per window" true however many callers start
//     — unattended-update.spec.
//  3. No property carries an error message, a path, a directory name, a user
//     name, a host name or repository data. The stage travels as a typed value
//     through StageOf; err.Error() never enters a property.
//
// The policy writes to no stream and never terminates, restarts or re-execs its
// caller: `archcore mcp` owns stdout as a protocol stream, and the running
// process keeps executing the image it started with.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"archcore-cli/internal/stamp"
	"archcore-cli/internal/telemetry"
)

// unattendedCeiling bounds the whole run. It protects the caller that started
// the policy — today the MCP server's background slot — from being held open by
// a stalled network for as long as the transport allows. Two minutes covers a
// version lookup, an archive download and a checksum fetch on a slow link
// [assumption] — unattended-update.spec.
//
// It does not bound the staged write, the health probe or the rename: a ceiling
// that tears the replace step defeats the atomicity contract, so healthProbe
// takes its bound from context.WithoutCancel and atomicReplace never reads the
// deadline at all.
const unattendedCeiling = 120 * time.Second

// claimScope names the stamp directory this policy claims in. It is its own
// scope because a sweep deletes every stamp older than its window, so sharing a
// directory with the session or staleness scopes would erase a day-long claim on
// their first run — see package stamp.
const claimScope = "update-stamps"

// devVersion is what a binary built without release ldflags reports. NeedsUpdate
// treats it as always behind, so without the refusal at §4 every locally built
// binary would replace itself with a release.
const devVersion = "dev"

// The wire vocabulary lives in internal/telemetry, typed, because the typed
// `archcore update` sends against the same contract and each path once carried
// its own untyped copy of these names — cli-update-telemetry.spec. Local
// aliases keep the call sites below short.
const (
	triggerAuto       = telemetry.TriggerAuto
	reasonCurrent     = telemetry.SkipReasonCurrent
	reasonNotWritable = telemetry.SkipReasonNotWritable
)

// UnattendedResult reports what the policy did. A refusal, a skip and a failure
// are all the zero value: no caller acts differently on them, and the difference
// that matters is already recorded in telemetry.
type UnattendedResult struct {
	Updated    bool
	NewVersion string
}

// UnattendedOptions carries what one unattended attempt needs.
type UnattendedOptions struct {
	Updater   *Updater
	Version   string            // the running version, already cleaned
	Telemetry *telemetry.Client // nil-safe
	CachePath string            // "" uses CachePath()
	ClaimDir  string            // "" uses the default scope directory
}

// RunUnattended evaluates the unattended-update policy and, when every condition
// holds, replaces the binary. It writes to no stream, never terminates or
// restarts its caller, and never runs a plugin command.
func RunUnattended(ctx context.Context, opts UnattendedOptions) UnattendedResult {
	// The zero value is every outcome other than a completed replacement.
	var none UnattendedResult

	// 1. An unofficial build refuses before anything else, so a fork never
	// reaches the filesystem, the network or the identifier file —
	// unattended-update.spec §3.
	if !isOfficialBuild() {
		return none
	}

	// 2. A development build refuses next — §4.
	if stripV(opts.Version) == devVersion {
		return none
	}

	// 3. CI refuses third — §5.
	if runningInCI() {
		return none
	}

	// 4. The claim. It is keyed by the resolved binary path, so two installs on
	// one machine hold separate claims and a symlink cannot buy a second one —
	// §6. A resolution failure refuses: without a path there is nothing to key
	// on and nothing to replace.
	target, err := resolveTarget(opts.Updater)
	if err != nil {
		return none
	}
	claimDir := opts.ClaimDir
	if claimDir == "" {
		claimDir = stamp.DirFor(claimScope)
	}
	// ClaimFailClosed, not Claim: a claim that cannot be established must read
	// as "a peer may hold it", because two callers that both win race the same
	// rename. The claim is never released — an attempt that crashes after this
	// line must not repeat on the next process that starts, so the window, not
	// the exit path, is what ends it (Failure Behavior 5).
	if !stamp.ClaimFailClosed(claimDir, target, CacheTTL) {
		return none
	}

	// 5. The ceiling opens here rather than at the top: everything above is
	// local and instant, and a deadline that started before the claim would be
	// spent on nothing.
	ctx, cancel := context.WithTimeout(ctx, unattendedCeiling)
	defer cancel()

	// 6. The cache before any network call — §7.
	cachePath := opts.CachePath
	if cachePath == "" {
		cachePath = CachePath()
	}
	latest, fresh := ReadCachedLatest(cachePath)
	// A fresh failure stamp — empty content inside the negative window — counts
	// as stale here. The negative cache exists to protect the 2 s hook budget,
	// which this path does not share: it runs at most once per claim window and
	// under its own ceiling. Reading a stamp as an answer would let one network
	// blip report `current` for a machine that never compared anything, which is
	// the one thing that event may never mean — §8.
	if latest == "" || !fresh {
		fetched, checkErr := opts.Updater.CheckLatest(ctx)
		if checkErr != nil {
			WriteCachedLatest(cachePath, "") // failure stamp
			// No to_version: the tag never resolved, and a placeholder would
			// read in the data as a version this run really aimed at.
			opts.send(ctx, telemetry.EventUpdateFailed, map[string]any{
				"trigger": triggerAuto,
				"stage":   FailureStage(checkErr),
			})
			return none
		}
		latest = fetched
		WriteCachedLatest(cachePath, latest)
	}

	// 7. The comparison. NewerSemver rather than NeedsUpdate: NeedsUpdate falls
	// back to "dev is always behind", and an unparseable version on either side
	// must refuse here rather than turn an odd tag into a downgrade — §12.
	newer, ok := NewerSemver(opts.Version, latest)
	if !ok {
		return none
	}
	if !newer {
		// The only place this event may be sent: after a real comparison of two
		// parsed versions — §11 and the invariant that guards it.
		opts.send(ctx, telemetry.EventUpdateSkipped, map[string]any{
			"trigger": triggerAuto,
			"reason":  reasonCurrent,
		})
		return none
	}

	// 8. Write access before the download — §13. A machine whose install
	// directory is root-owned is the supported way to opt a machine out of
	// self-update, so this is a routine outcome and not a failure: it must cost
	// no bandwidth and report as a skip.
	if !targetDirWritable(target) {
		opts.send(ctx, telemetry.EventUpdateSkipped, map[string]any{
			"trigger": triggerAuto,
			"reason":  reasonNotWritable,
		})
		return none
	}

	// 9. The replacement runs on a copy of the caller's Updater. Installing the
	// probe on the caller's value would leave it installed for whatever else
	// holds that pointer — the typed `archcore update` path deliberately runs
	// without one, and the policy must not reach across and change it.
	attempt := *opts.Updater
	attempt.PreCommitProbe = healthProbe
	// The resolved path, so the file the claim names and the file Apply replaces
	// are provably the same one.
	attempt.ExecPath = target

	if err := attempt.Apply(ctx, latest); err != nil {
		opts.send(ctx, telemetry.EventUpdateFailed, map[string]any{
			"trigger":      triggerAuto,
			"stage":        FailureStage(err),
			"from_version": opts.Version,
			"to_version":   latest,
		})
		return none
	}

	// 10. The replacement is done; the running process keeps executing the image
	// it started with, and the new version takes effect at the next launch.
	opts.send(ctx, telemetry.EventUpdated, map[string]any{
		"trigger":      triggerAuto,
		"from_version": opts.Version,
		"to_version":   latest,
	})
	return UnattendedResult{Updated: true, NewVersion: latest}
}

// send is the one place an unattended event is emitted, and the one place
// context.WithoutCancel is applied.
//
// Telemetry has to outlive the ceiling. The events that matter most are the ones
// sent on the way out of a run that ran long, and a Capture inheriting an
// expired deadline would drop exactly the cli_update_failed that reports the
// ceiling — the failure would become invisible in the same moment it happened.
// Capture keeps its own short timeouts, so nothing here is unbounded.
// The receiver is `op` rather than the §F single letter: `u` belongs to
// *Updater throughout this package, and §F's edge case asks for the shortest
// unambiguous prefix when two types in one package share an initial.
func (op UnattendedOptions) send(ctx context.Context, event telemetry.Event, props map[string]any) {
	// Capture is nil-safe, so a caller that sends no telemetry needs no branch.
	op.Telemetry.Capture(context.WithoutCancel(ctx), event, props)
}

// FailureStage names the failed step for a telemetry event's `stage` property.
//
// It reads the stage off the typed error rather than off the message, because
// the message routinely holds a path, an archive name or a URL and none of those
// may leave the machine. An untagged error can only have come from resolving the
// latest tag, so `check` is the honest default; the five stage values are the
// whole vocabulary the dashboards group on — cli-update-telemetry.spec.
//
// It is exported because the typed `archcore update` reports the same property
// off the same errors and carried a byte-identical private copy of this
// function, rationale included.
func FailureStage(err error) string {
	if stage, ok := StageOf(err); ok {
		return string(stage)
	}
	return string(StageCheck)
}

// runningInCI reports whether this process is on a CI runner.
//
// The decision it feeds is this file's own — a CI runner's filesystem is
// discarded minutes later, so replacing a binary there buys nothing
// (unattended-update.spec §5) — but the set of variables that means "CI" is one
// fact shared with the telemetry grader and the plugin step, so it is read from
// one declaration rather than copied here.
func runningInCI() bool { return telemetry.DetectedCI() }

// resolveTarget returns the resolved path of the running binary: the ExecPath
// seam first, then os.Executable, then EvalSymlinks.
//
// The symlink resolution is what makes the claim key stable. An install reached
// through /usr/local/bin/archcore and the same install reached through its real
// path would otherwise take two different claims and both replace one file.
func resolveTarget(u *Updater) (string, error) {
	if u == nil {
		// A caller with no updater cannot replace anything. Refusing beats a nil
		// dereference in a background goroutine that must not fault its host.
		return "", errors.New("no updater")
	}
	path := u.ExecPath
	if path == "" {
		var err error
		path, err = os.Executable()
		if err != nil {
			return "", err
		}
	}
	return filepath.EvalSymlinks(path)
}

// targetDirWritable reports whether this process may create the file the
// replacement is about to stage.
//
// It creates that exact name — "<base>.tmp.<pid>", the one stageBinary writes —
// rather than a probe of its own. Same directory, same name, same mode: a probe
// that passes and a staging that then fails on permissions cannot disagree, and
// the name is already the one sweepAttemptLeftovers collects, so a process
// killed between this check and the download leaves nothing a later attempt does
// not clear — unattended-update.spec §14.
//
// This writes no content and is not an atomic publish, so it is not a fourth
// temp-file-plus-rename — choosing-an-atomic-write.rule.
func targetDirWritable(target string) bool {
	probe := target + stagedSuffix + strconv.Itoa(os.Getpid())
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return true
}
