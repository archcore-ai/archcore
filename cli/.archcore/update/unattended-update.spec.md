---
title: "Unattended Update Policy"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "update"
---

## Purpose & Scope

This spec defines **when the CLI may replace its own binary with no user watching**, independent of what triggers it. The trigger supplies the moment; this policy supplies every condition that must hold first. Dependents: the background trigger inside `archcore mcp`, and any orchestrator added later — a devcontainer step, a scheduled job, a command flag.

Out of scope: `archcore update` typed by a user, whose behavior this spec does not change, and `archcore update --check`, which reports and never writes. The plugin-update step of `archcore update` is specified in `.archcore/update/updating-the-plugin.spec.md`.

## Surface

- Policy entry point: an exported function in `internal/update` [planned], callable in-process. This release adds no command flag; the MCP trigger is the only caller.
- Conditions, in the fixed order of Normative Behavior 3–6.
- Official-build marker: a second `-X` ldflags variable [planned], injected only by this repository's release workflow. It is distinct from the telemetry key — a fork may inject its own key legitimately, and telemetry variables govern telemetry only.
- CI variables: the set `is_ci()` enumerates in `@install.sh` — `CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `BUILDKITE`, `JENKINS_URL`, `TEAMCITY_VERSION`.
- Cross-process claim: a fail-closed claim in its own scope [planned], keyed by the resolved binary path, window 24 h. `stamp.Claim` (`@internal/stamp/stamp.go`) supplies the atomic mechanism but not the failure bias — it returns `true` on every filesystem error.
- Freshness cache: one canonical implementation in `internal/update` [planned] — path, the 24 h TTL, the 1 h negative TTL, and an atomic temp-file-and-rename write. `archcore update --check` and the `doctor` advisory call through it; today's copies at `@cmd/update.go:29` move there.
- Replacement: `Updater.Apply` — `@internal/update/update.go` — plus a pre-rename health probe. Its temporary file is `<base>.tmp.<pid>`; the Windows aside name becomes per-attempt, `<target>.old.<pid>`, swept as `<target>.old.*`.
- Health probe: the extracted binary run with `--version`, bounded by 3 s [assumption], before the rename.
- Telemetry: `cli_updated`, `cli_update_failed`, and `cli_update_skipped`, each carrying `trigger` set to `auto`.

## Normative Behavior

1. The CLI MUST evaluate the conditions in requirements 3–6 in the order listed.
2. WHEN a condition in requirements 3–6 holds, the CLI MUST stop without evaluating a later one.
3. IF the binary does not carry the official-build marker, THEN the CLI MUST refuse without sending an event. Rationale: a stamped fork or a repackaged build would otherwise silently replace itself with this project's release — a takeover of a binary this project does not own.
4. IF the running version is `dev`, THEN the CLI MUST refuse without sending an event. Rationale: `NeedsUpdate` treats a development build as always behind, so without this condition every locally built binary replaces itself with a release.
5. IF any CI variable is set, THEN the CLI MUST refuse without sending an event.
6. IF the claim for this binary path cannot be acquired, THEN the CLI MUST refuse without sending an event.
7. WHEN requirements 3–6 all pass, the CLI MUST read the freshness cache before any network call.
8. IF the cache holds a fresh failure stamp, THEN the CLI MUST treat the cache as stale and proceed to the lookup. Rationale: the negative cache protects the 2 s hook budget; the policy runs at most once per claim window under its own ceiling, and a network blip must not masquerade as `current`.
9. WHEN the cache is stale, the CLI MUST resolve the latest version under a bounded timeout.
10. WHEN the lookup succeeds, the CLI MUST refresh the cache with the result.
11. IF the latest version — cached or resolved — is not newer than the running version, THEN the CLI MUST stop and send `cli_update_skipped` carrying `reason` set to `current`.
12. IF the running version or the latest version does not parse as semver, THEN the CLI MUST refuse without sending an event. Rationale: `NeedsUpdate` falls back to string inequality on a parse failure; in an unattended path that turns an odd tag into a replacement or a downgrade.
13. WHEN a newer version exists, the CLI MUST verify write access to the target directory before it downloads anything.
14. The CLI MUST remove temporary files matching `<base>.tmp.*` in the target directory before it writes a new one.
15. WHEN extraction completes, the CLI MUST run the extracted binary with `--version`, bounded by the probe timeout, before the rename.
16. IF the health probe exits nonzero, times out, or fails to start, THEN the CLI MUST abandon the replacement and send `cli_update_failed` with `stage` set to `replace`.
17. WHEN the replacement succeeds, the CLI MUST send `cli_updated` with `trigger` set to `auto`.

## Constraints & Invariants

- Constraint: the policy MUST NOT be reachable from a code path that writes to a protocol stream. Its callers own that discipline; this spec does not grant one.
- Constraint: the policy MUST NOT terminate its caller.
- Constraint: the policy MUST NOT restart or re-exec its caller. Replacement changes the file, never the running process.
- Constraint: the claim MUST fail closed. Rationale: the hook scopes fail open so that dedup never breaks a hook; under the same bias an unwritable state directory lets every concurrent process replace the binary at once, which is the outcome the exclusivity invariant exists to forbid.
- Constraint: the policy MUST evaluate every reportable skip after the claim is acquired. Rationale: both reportable reasons — `current` and `not_writable` — then fall inside one claim window, so a machine emits at most one `cli_update_skipped` per window however many callers start on it.
- Constraint: a telemetry opt-out — `DO_NOT_TRACK` or `ARCHCORE_TELEMETRY_OPTOUT` — MUST NOT disable updates. Those variables govern telemetry only; unattended update has no opt-out surface. A root-owned install directory yields `cli_update_skipped(not_writable)` on every attempt — that placement is the supported operator answer for machines that must not self-update, and the published docs name it.
- Constraint: the policy MUST NOT run any host plugin command. The plugin-update step belongs to `archcore update` typed by a user — `.archcore/update/updating-the-plugin.spec.md`; the unattended path never reaches it.
- Constraint: the claim window MUST equal the cache TTL (24 h). Rationale: one constant governs both, and the resulting worst case is ~48 h from release to replacement — a release published just after a check waits 24 h for the cache and 24 h for the window.
- Constraint: the whole run MUST complete within 120 s. [assumption] that ceiling covers a version lookup, an archive download, and a checksum fetch on a slow link without holding a caller open indefinitely.
- Constraint: the ceiling MUST NOT interrupt the synced write or the rename. A ceiling that tears the replace step defeats the atomicity contract; the ceiling governs the network-bound stages.
- Constraint: on Windows the replace is two renames, and the target name is briefly absent between them. Accepted platform gap: a hook launched inside that window fails to start, and the failure is attributable rather than mysterious because the aside name is per-attempt.
- Constraint: the policy MUST NOT attempt privilege elevation.
- Constraint: `.archcore/settings.json` MUST NOT gate this policy. Rationale: binary replacement is machine-global, and a per-project file cannot govern a machine-global action.
- Constraint: a provenance condition — refusing when a package manager owns the binary — activates with the first package-manager channel. No Homebrew tap, Scoop manifest, or winget package exists today, so the condition has nothing to distinguish against and is not part of this release.
- Invariant: at most one attempt per binary path per claim window, across every concurrent process on the machine.
- Invariant: a refusal and a failure are distinguishable. A refusal means a condition was not met; a failure means an attempted step did not complete.
- Invariant: `archcore update` with no flag is unchanged by this policy, on every machine.
- Invariant: the running process keeps executing the image it started with. The new version takes effect at the next launch of that binary.
- Invariant: `cli_update_skipped(current)` is sent only after an actual version comparison, never from a failure stamp.

## Failure Behavior

1. IF the state directory cannot be created, read, or written, THEN the CLI MUST refuse without sending an event. Exclusivity cannot be established there, and the freshness cache is unreadable in the same condition.
2. IF the version lookup, the download, the checksum, the extraction, or the replacement fails, THEN the CLI MUST send `cli_update_failed` with the stage and with `trigger` set to `auto`.
3. IF the target directory is not writable by the current user, THEN the CLI MUST refuse before downloading and send `cli_update_skipped` carrying `reason` set to `not_writable`.
4. IF the 120 s ceiling elapses, THEN the CLI MUST abandon the attempt and send `cli_update_failed` with the stage it was in.
5. IF an attempt panics after the claim is acquired, THEN the CLI MUST NOT release the claim before its window elapses. Rationale: a crashing attempt would otherwise repeat on every process that starts next.
6. IF the caller exits while a replacement is in flight, THEN the running binary MUST stay intact. The rename is the last step of `atomicReplace`, so an abandoned attempt leaves a temporary file and nothing else.

## Conformance

An implementation is conformant when it evaluates conditions 3–6 in that order, refuses a marker-less or development build before anything else, fails closed on any claim it cannot establish, treats a failure stamp as stale, replaces only on a strictly newer parseable version that passed the health probe, evaluates every reportable skip after the claim is acquired, preserves the one-attempt-per-window invariant across concurrent processes, never restarts its caller, and distinguishes refusals from failures per the failure rules.

Given a state directory that is read-only, when any trigger invokes the policy, then it refuses at requirement 6, contacts no network, and sends no event.