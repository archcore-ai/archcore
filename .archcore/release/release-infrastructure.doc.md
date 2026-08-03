---
title: "Release Infrastructure Overview"
status: accepted
tags:
  - "release"
---

## Overview

The Archcore CLI uses a tag-driven release pipeline. Pushing a `v*` tag triggers GitHub Actions, which runs the tests and invokes GoReleaser to build cross-platform binaries and publish a GitHub Release.

## Content

### Components

| Component | File | Purpose |
|---|---|---|
| Version vars | `@main.go` | `version` and `commit` variables with dev defaults |
| Cobra integration | `@cmd/root.go` | `NewRootCmd(version, commit)` sets the `Version` field and the version template |
| GoReleaser config | `@.goreleaser.yaml` | Defines the build matrix, archive naming, and checksums |
| GitHub Actions — release | `@.github/workflows/release.yml` | Orchestrates test → build → publish on a tag push |
| GitHub Actions — landing nudge | `@.github/workflows/notify-landing.yml` | On a push to `main` that touches `install.sh` or `install.ps1`, dispatches an `installer-updated` event to `archcore-ai/landing` so archcore.ai republishes the installers |
| Install script (Unix) | `@install.sh` | End-user installer for macOS and Linux; downloads `.tar.gz` release artifacts |
| Install script (Windows) | `@install.ps1` | PowerShell installer for Windows amd64 and arm64; downloads `.zip` release artifacts |
| Self-update | `@internal/update/update.go` | In-binary update: check the latest version, download, verify the checksum, replace atomically |
| Update command | `@cmd/update.go` | `archcore update`, plus the cached background version check |

### Build matrix

| OS | Architecture |
|---|---|
| darwin | amd64, arm64 |
| linux | amd64, arm64 |
| windows | amd64, arm64 |

Every build uses `CGO_ENABLED=0` for a static binary and the `-s -w` ldflags to strip debug information.

### Artifact naming

An archive follows the pattern `archcore_<os>_<arch>.tar.gz` for darwin and linux, and `archcore_<os>_<arch>.zip` for windows. Examples: `archcore_darwin_arm64.tar.gz`, `archcore_windows_amd64.zip`. `install.sh` and `archcore update` consume the Unix archives; `install.ps1` consumes the Windows zips.

Every release includes a `checksums.txt` file with SHA-256 hashes for verification.

### Version format

- Dev build: `archcore dev (commit: none)`
- Release build: `archcore 1.2.3 (commit: abc1234)`

`SetVersionTemplate` on the cobra root command sets this template.

### Version resolution

Both install scripts and `archcore update` resolve "latest" by reading the `Location` header of `https://github.com/archcore-ai/cli/releases/latest`, a `302` that already carries the tag. The GitHub REST API is avoided deliberately: its 60 requests per hour unauthenticated limit is per IP and breaks teams behind a shared egress address. The related ADR records that decision.

### Update paths

A user updates the CLI in one of three ways:

1. `archcore update` — the self-update command downloads the release and replaces the binary in place on every supported platform. On Windows, the running `.exe` is renamed to `<binary>.old` before the new file moves in, with a rollback when the second rename fails (`atomicReplace` in `@internal/update/update.go`).
2. Re-running the install script:
   - macOS and Linux: `curl -fsSL https://archcore.ai/install.sh | bash`
   - Windows: `irm https://archcore.ai/install.ps1 | iex`
3. `go install github.com/archcore-ai/cli@latest`.

### Secrets

| Secret | Required | Purpose |
|---|---|---|
| `GITHUB_TOKEN` | yes | Publishes the release. GitHub Actions provides it automatically; nothing to configure. |
| `LANDING_DISPATCH_TOKEN` | no | A PAT with `contents: write` on `archcore-ai/landing`, used by `notify-landing.yml`. While it is absent, the job emits a warning and exits 0 — archcore.ai still picks the installer up on its next deploy, so a missing secret never turns a CLI push red. |

The pipeline needs no signing keys and no notarization credentials.

## Examples

Non-normative example — the release artifact listing for v1.0.0:

```
archcore_darwin_amd64.tar.gz
archcore_darwin_arm64.tar.gz
archcore_linux_amd64.tar.gz
archcore_linux_arm64.tar.gz
archcore_windows_amd64.zip
archcore_windows_arm64.zip
checksums.txt
```
