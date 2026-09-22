---
title: "Release Infrastructure Overview"
status: accepted
tags:
  - "component:cli"
  - "release"
---

## Overview

The Archcore CLI and the Archcore plugin share one tag-driven release pipeline in `archcore-ai/plugin`. Pushing a `v*` tag triggers @.github/workflows/release.yml, which verifies that the four plugin manifests equal the tag, runs both test suites, regenerates the plugin distribution on `main`, and invokes GoReleaser from `cli/` to build cross-platform binaries and publish the GitHub Release.

The former `archcore-ai/cli` repository published releases up to v0.8.7 (2026-09-21). Its workflow files remain under `cli/.github/workflows/` as migration reference; GitHub does not execute them.

## Content

### Components

| Component | File | Purpose |
|---|---|---|
| Version vars | `@cli/main.go` | `version` with its `dev` default and the build-info fallback |
| Cobra integration | `@cli/cmd/root.go` | `NewRootCmd(version)` sets the `Version` field and the version template |
| GoReleaser config | `@cli/.goreleaser.yaml` | Defines the build matrix, archive naming, checksums, the two ldflags injections, the inertness post-build hook, and the release assets |
| GitHub Actions — release | `@.github/workflows/release.yml` | `verify-version` → `test-plugin` and `test-cli` → `publish-plugin` (export and `main` push) → `publish-cli` (GoReleaser) on a tag push |
| GitHub Actions — CLI tests | `@.github/workflows/cli-test.yml` | gofmt, vet, golangci-lint, `go test ./...`, the inertness self-test, and the examples fixture check on pull requests and `dev` pushes |
| GitHub Actions — installer smoke | `@.github/workflows/cli-install-smoke.yml` | Runs both installers on Windows (PowerShell 5.1 and 7), Ubuntu, macOS, Alpine, and a dash-only Debian on pull requests and `dev` pushes that touch an installer |
| GitHub Actions — landing nudge | `@cli/.github/workflows/notify-landing.yml` | Reference copy only. [assumption] A root dispatcher for this repository is pending (plan `release/unified-release-cutover`, task 19) |
| Install script (Unix) | `@cli/install.sh` | End-user installer for macOS and Linux; downloads `.tar.gz` release artifacts |
| Install script (Windows) | `@cli/install.ps1` | PowerShell installer for Windows amd64 and arm64; downloads `.zip` release artifacts |
| Self-update | `@cli/internal/update/update.go` | In-binary update: check the latest version, download, verify the checksum, replace atomically |
| Update command | `@cli/cmd/update.go` | `archcore update`, plus the cached background version check; names the release repository `archcore-ai/plugin` |

### Build matrix

| OS | Architecture |
|---|---|
| darwin | amd64, arm64 |
| linux | amd64, arm64 |
| windows | amd64, arm64 |

Every build uses `CGO_ENABLED=0` for a static binary and the `-s -w` ldflags to strip debug information.

### Artifact naming

An archive follows the pattern `archcore_<os>_<arch>.tar.gz` for darwin and linux, and `archcore_<os>_<arch>.zip` for windows. Examples: `archcore_darwin_arm64.tar.gz`, `archcore_windows_amd64.zip`. `install.sh` and `archcore update` consume the Unix archives; `install.ps1` consumes the Windows zips.

Every release includes a `checksums.txt` file with SHA-256 hashes for verification, plus `install.sh` and `install.ps1` as extra release files.

Both installers and `archcore update` download `checksums.txt` on every run. Its download count is therefore a denoised proxy for real installer runs, while the archive count is the raw figure. The two differ by roughly a factor of six; see the install analytics ADR.

### Version format

`archcore --version` prints the version with a `v` prefix: the v0.10.1 release prints `v0.10.1`, and a local `make build CLI_VERSION=0.8.7-dev` prints `v0.8.7-dev` (observed 2026-09-22). The template lives in `@cli/cmd/root.go`; a build without an injected version starts from `dev` and takes the module version from Go build info when one is recorded (`resolveVersion` in `@cli/main.go`).

### Version resolution

Both install scripts and `archcore update` resolve "latest" by reading the `Location` header of `https://github.com/archcore-ai/plugin/releases/latest`, a `302` that already carries the tag. The GitHub REST API is avoided deliberately: its 60 requests per hour unauthenticated limit is per IP and breaks teams behind a shared egress address. The related ADR records that decision.

Because every tag releases both components, the redirect always lands on a release that carries CLI assets. The resolver halts at the first 3xx, so a repository rename, which answers with a 301 to the new name first, breaks the check for binaries built before the rename; the unified-release ADR records that consequence.

### Update paths

A user updates the CLI in one of two ways:

1. `archcore update` — the self-update command downloads the release and replaces the binary in place on every supported platform. On Windows, the running `.exe` is renamed to `<binary>.old` before the new file moves in, with a rollback when the second rename fails (`atomicReplace` in `@cli/internal/update/update.go`).
2. Re-running the install script:
   - macOS and Linux: `curl -fsSL https://archcore.ai/install.sh | bash`
   - Windows: `irm https://archcore.ai/install.ps1 | iex`

`go install` is not a channel: the module path in `@cli/go.mod` is `archcore-cli`, not a GitHub path.

### Install analytics

The installers published on archcore.ai send one anonymous event per run. The PostHog key is **not** baked into the scripts in this repository: both carry a `__POSTHOG_KEY__` placeholder, and the `archcore-ai/landing` deploy workflow substitutes the real key while it syncs them into `public/`.

Consequences for this pipeline:

- A script run from a clone, a fork, or `cli-install-smoke.yml` reports nothing, because the guard requires a `phc_` prefix.
- The landing deploy fails when a synced script does not carry exactly one placeholder.
- [assumption] Until the landing deploy fetches the installers from this repository (plan `release/unified-release-cutover`, task 15), archcore.ai serves the copies from `archcore-ai/cli`.

The contract, the event properties, and the opt-out procedure are in install-script-usage.guide.md. The decision and its trade-offs are in the install analytics ADR.

### Secrets and variables

| Name | Location | Required | Purpose |
|---|---|---|---|
| `GITHUB_TOKEN` | this repo (automatic) | yes | Pushes `main` and publishes the release. GitHub Actions provides it; nothing to configure. |
| `POSTHOG_KEY` | this repo (variable) | yes, for telemetry | Public PostHog project key, injected as `-X archcore-cli/internal/telemetry.apiKey` when `github.repository_id` is `1201781375`. `@cli/scripts/assert-not-inert.sh` fails the release when the built binary does not carry it. Copied from `archcore-ai/cli` on 2026-09-22. |
| `POSTHOG_HOST` | this repo (variable) | no | Ingestion host. Present since 2026-09-22 for parity with the landing deploy. |
| `ARCHCORE_OFFICIAL_BUILD` | workflow expression | yes | The official-build marker, `-X archcore-cli/internal/update.officialBuild`, injected only when `github.repository_id` is `1201781375`; a fork builds without it and never self-replaces. |
| `LANDING_DISPATCH_TOKEN` | this repo (secret) | no | [assumption] Pending with the root landing dispatcher; a PAT with `contents: write` on `archcore-ai/landing`. |
| `POSTHOG_KEY` | `archcore-ai/landing` (variable) | yes, for analytics | Substituted into the installers at landing deploy time. A missing or non-`phc_` value fails the landing deploy. |
| `POSTHOG_HOST` | `archcore-ai/landing` (variable) | no | Ingestion host, `https://edge.archcore.ai`. Falls back to the same value when unset. |

The pipeline needs no signing keys and no notarization credentials.

## Examples

Non-normative example — the release artifact listing for v0.10.1 (published 2026-09-22):

```
archcore_darwin_amd64.tar.gz
archcore_darwin_arm64.tar.gz
archcore_linux_amd64.tar.gz
archcore_linux_arm64.tar.gz
archcore_windows_amd64.zip
archcore_windows_arm64.zip
checksums.txt
install.sh
install.ps1
```