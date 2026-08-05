---
title: "Measure Installs from the Installer, With the PostHog Key Injected at Landing Deploy Time"
status: accepted
tags:
  - "release"
  - "telemetry"
---

## Context

The project had no reliable install count.

### Current State

Before this decision the only signal was the GitHub release asset `download_count`, read by hand. That number is heavily contaminated. Measured across all 36 releases at the time of writing:

| Signal | Total |
|---|---|
| Archive downloads, raw | ~3175 |
| `checksums.txt` downloads | 538 |
| `linux_amd64` archives alone | 2732, or 86 % of raw |

Both `install.sh` and `archcore update` download the platform archive **and** `checksums.txt` on every run. The `checksums.txt` count is therefore a denoised proxy for real installer runs, and the archive count is the raw figure. The gap between 538 and 3175 is scanners, mirrors, and CI.

### Problem Statement

An npm distribution channel was proposed first, on the theory that npm download counts would supply the metric. Evidence rejected it. npm documents its own counts as "naïve by design … simply a count of the number of HTTP 200 responses we served that were tarball files", and states that "only if your package is getting > 50 downloads/day can you be sure you're seeing signal instead of noise". At 7–32 installer runs per release the project sits two orders of magnitude below that floor. npm performs no bot filtering, and exposes one integer per package with no second signal to cross-check. A seven-package publishing pipeline with immutable versions was not justified to obtain a less trustworthy number than the project already had.

## Decision

Measure installs in two independent ways, both reporting into the PostHog project that already serves archcore.ai.

1. **An installer beacon.** `install.sh` and `install.ps1` send one anonymous event per run — `cli_installed` on success, `cli_install_failed` with a `stage` category on failure — to `https://ph.archcore.ai/i/v0/e/`, the existing first-party ingestion proxy.
2. **A release-counter bridge.** `.github/workflows/install-stats.yml` in `archcore-ai/landing` reports the GitHub `download_count` totals to PostHog on a daily schedule, sending both the denoised `checksums.txt` figure and the raw archive figure so the noise floor stays visible on the same chart.

The two measure different populations — the bridge counts asset fetches including automation, the beacon counts consenting machines — so each is the other's sanity check.

**The PostHog key is never committed to this repository.** Both scripts carry a `__POSTHOG_KEY__` placeholder. The landing deploy workflow substitutes `vars.POSTHOG_KEY` while it copies the scripts into `public/`.

### Rationale

- **The key injection makes the repository copy inert by construction.** A clone, a fork, or `install-smoke.yml` sends nothing, because the guard requires a `phc_` prefix. No opt-out plumbing is needed in CI to keep test runs out of the data.
- **archcore.ai becomes the only place that can turn analytics on**, which matches where the privacy policy is published and where the key already lives for the site bundle.
- **The `phc_` guard, rather than a comparison against the placeholder text**, means the substitution cannot rewrite its own off-switch.
- **The bridge needs no state.** It reports a cumulative gauge, so a missed or re-run job can never double-count.

### Implementation Notes

- The `distinct_id` is a random 32-character hexadecimal value stored at `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-id` on Unix and `%USERPROFILE%\.local\state\archcore\install-id` on Windows. `install.ps1` deliberately uses the non-idiomatic `.local\state` path to match `updateCheckCachePath` in `@cmd/update.go`, so the CLI's own telemetry can adopt the same identifier later and join "installed" to "used" without a second one. `[guid]::NewGuid().ToString('N')` produces the same 32-hex format `install.sh` writes, so one machine that has used both installers counts once.
- **An opt-out leaves no trace.** `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` are checked before the identifier file is created, not after.
- The beacon can never fail an install: bounded timeouts, discarded output, `|| true` in bash, and a `try`/`catch` in PowerShell. The PowerShell guard matters because `$ErrorActionPreference` is `Stop` script-wide — an unguarded web call there would abort a *successful* install, which is exactly how `Test-Install` once failed.
- Failure events carry a `stage` category only. Error messages, file paths, directory names, user names, and hostnames are never transmitted.
- The installer prints a one-line notice when it sends an event, so disclosure sits next to the act and not only in the policy.
- The landing deploy fails when the placeholder count in a synced script is not exactly 1. A renamed placeholder would produce an installer that reports nothing — a state indistinguishable from "nobody is installing".
- Bridge events set `$process_person_profile: false`. There is no person, and anonymous events are both cheaper and the honest representation.
- **Publication is gated on the installer tests.** Installer changes reach `main` by direct push as well as by pull request, so `install-smoke.yml` triggers on both, and `notify-landing.yml` is chained to a successful `Install Smoke` run through `workflow_run` instead of firing on the push. Without the chain, the redeploy dispatch — one fast curl — beat a multi-minute Windows matrix, and archcore.ai could republish an installer before anything had verified it. The beacon lives in exactly the PowerShell path that has no other gate.

## Alternatives Considered

### Alternative 1: npm channel for the download statistics

Rejected on npm's own published caveats — see Problem Statement. An npm channel may still be justified later as a *distribution* decision for the JavaScript ecosystem, but not as a measurement one. Note also that `npx` suits this CLI poorly: `archcore init` writes host configs and hooks that invoke a bare `archcore`, while npx puts the binary on `PATH` only for the executed process.

### Alternative 2: The bridge alone, with no beacon

Requires no privacy change at all. Rejected as insufficient: it yields an aggregate trend with a noise floor and cannot distinguish unique machines, fresh installs from upgrades, or real users from CI.

### Alternative 3: Opt-in beacon

Honest, and useless for this metric — install-time opt-in rates are near zero.

### Alternative 4: An endpoint on archcore.ai that forwards to PostHog

Not available. archcore.ai is static GitHub Pages with no server, no edge functions, and no middleware, per the hosting ADR. The existing `ph.archcore.ai` proxy is the only first-party ingestion path.

## Consequences

### Positive

- Install counts become answerable from the same place as every other archcore metric, with two mutually-checking signals.
- The repository copy of each installer is provably inert, so contributors and forks cannot pollute the data.
- The `install-id` contract is now fixed, so CLI telemetry can adopt it without introducing a second identifier.
- Direct pushes of installer changes are now tested and gated, which they were not before.

### Negative

- **A published privacy promise was reversed.** `/privacy` previously stated "No telemetry. We do not collect usage analytics, crash reports, identifiers, or any data from the plugin or CLI." The page now scopes that promise to the installed tools and documents the installer beacon separately. Any future change to what the beacon sends must update that copy in the same change.
- `ph.archcore.ai` resolves to Vercel, the platform the project migrated away from because it is unreliable in Russia. Beacons from affected users fail silently, so install geography is systematically skewed. The bridge is unaffected and partly compensates.
- The beacon adds up to 3 s to an install on a network where the proxy is unreachable.
- The bridge's historical mode reports a per-release total *as of the day it runs*. The GitHub API exposes no historical series, so a true daily backfill is impossible; charted at publish dates it answers "which releases got picked up", not "installs per week".
- **A red or flaky `Install Smoke` run now blocks installer publication**, where previously the redeploy fired regardless. `Notify Landing` keeps its `workflow_dispatch` as the manual override.
- There is no instant kill switch for the beacon. Turning it off means reverting the substitution step and redeploying the landing, roughly two to three minutes. Clearing `POSTHOG_KEY` is not an option — that fails the site build by design.

### Changes Made

- @install.sh — telemetry constants, `telemetry_enabled`, `install_id_path`, `resolve_install_id`, `send_event`; `error_exit` reports a failure stage; `main` advances `STAGE` and sends `cli_installed`
- @install.ps1 — the same, as `Test-TelemetryEnabled`, `Get-InstallIdPath`, `Get-InstallId`, `Send-TelemetryEvent`
- @.github/workflows/install-smoke.yml — `DO_NOT_TRACK: "1"` as a second line of defence; added a `push` trigger on `main` so a direct push is tested too
- @.github/workflows/notify-landing.yml — retriggered from `push` to `workflow_run` on `Install Smoke`, with a guard on conclusion, upstream event, and branch
- `archcore-ai/landing` — key substitution in the installer sync step of `deploy.yml`; new `install-stats.yml`; `cli_installed`, `cli_install_failed`, `release_downloads_sampled`, and `release_downloads_recorded` declared in `src/lib/analytics/events.ts` under `ExternalAnalyticsEventMap`; `src/pages/privacy.tsx` rewritten
- Updated install-script-usage.guide.md with the analytics contract, the opt-out procedure, and the placeholder rules
- Updated release-infrastructure.doc.md with the two workflow roles, the analytics section, and the variables table
- Updated how-to-release.guide.md with the installer publication path and the warning to keep release checks on `raw.githubusercontent.com`
