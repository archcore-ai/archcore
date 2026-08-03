---
title: "How to Create a New Release"
status: accepted
tags:
  - "release"
---

## Purpose

Publish a new Archcore CLI release from a git tag, and keep the release aligned with the Claude plugin and with `settings.json` parser compatibility.

## Prerequisites

- Push access to the `archcore-ai/cli` repository.
- Every intended change merged to `main`.
- Tests passing on `main`.

## Procedure

1. Check out the latest `main`.

   ```bash
   git checkout main
   git pull origin main
   ```

2. Choose a version that follows semver.

   - `v1.0.0` — first stable release, or a breaking change
   - `v1.1.0` — new features, backwards compatible
   - `v1.1.1` — bug fixes only

3. Create and push the tag.

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. Monitor the release workflow at `https://github.com/archcore-ai/cli/actions`.

   The workflow in `@.github/workflows/release.yml` triggers on the tag push. It runs `go test ./...`, builds binaries for darwin, linux, and windows on amd64 and arm64 (6 platforms), and creates a GitHub Release with the archives and `checksums.txt`.

5. Verify the published release.

   ```bash
   # The GitHub Release page carries 7 assets (6 archives + checksums.txt)
   gh release view v1.0.0

   # Install script (macOS/Linux)
   ARCHCORE_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/archcore-ai/cli/main/install.sh | bash

   # Install script (Windows PowerShell)
   $env:ARCHCORE_VERSION="v1.0.0"; irm https://raw.githubusercontent.com/archcore-ai/cli/main/install.ps1 | iex

   # Installed binary
   archcore --version
   ```

   Expected result: `archcore 1.0.0 (commit: <sha>)`.

## Coordinated release with the Claude plugin

The plugin's `SKILL.md` pins a minimum CLI version — currently `v0.6.1`, the `CLAUDE.md` and `AGENTS.md` nudge layout — and gates `/archcore:init` on it. An older `archcore --version` makes the plugin skip the `install_host_config` MCP tool and ask the user to update. The constant and the CLI release tag must agree, and the order matters.

Rollout order — CLI first, plugin second:

1. Tag and release the CLI with exactly the version that the plugin's `SKILL.md` constant names.
2. IF the tag ends up different, for example after a hotfix bump or a renumbering, THEN update the `SKILL.md` constant together with the plugin release, so the two never diverge.
3. Verify that the release assets are live before touching the plugin: `gh release view vX.Y.Z` shows all 7 assets, and `archcore update` on an older install sees the new version.
4. Release the plugin from its `dev` branch.

A mismatch fails in both directions:

- The tag is behind the constant — the constant says `v0.6.0` while the CLI shipped only as `v0.6.1`: the gate rejects users who run a capable CLI.
- The feature lands in a later tag than the constant — the constant says `v0.6.0` while the tool exists only from `v0.6.1`: the gate admits CLIs without the tool, and the plugin fails on an unknown tool call mid-flow.

## Sequencing a release that adds a settings.json field

`Settings` parsing is forward-compatible: the tolerant parser captures an unknown config field in `Settings.Extra` with a soft warning instead of a hard error. Tolerance protects only binaries that already ship it. A CLI released before the tolerant parser hard-fails on any unknown field.

Requirement: a release that introduces a new `settings.json` field MUST ship no earlier than the release that carries the tolerant parser, which shipped with the globals rollout.

When planning such a release:

1. Confirm that the previous released version already tolerates unknown fields. Every release from the globals rollout onward does; keep this check for a long-lived maintenance branch.
2. Ship the parser and validation change in an earlier, separate release from the feature that writes the new field, so a user on the intermediate version upgrades smoothly in either order.
3. Do not backport a new config field to a branch whose parser is still strict.

## Verification

- The GitHub Release page shows 6 archives (4 `.tar.gz` for darwin and linux on amd64 and arm64, 2 `.zip` for windows on amd64 and arm64) plus `checksums.txt`.
- `archcore --version` on the installed binary shows the expected version and commit.
- The install script succeeds on a clean macOS or Linux machine.
- `install.ps1` succeeds on a clean Windows machine.
- WHEN the release changes the plugin contract, the plugin's `SKILL.md` minimum-version constant equals the tag that was pushed.

The related reference document on release infrastructure carries the full build matrix, artifact naming, and update paths.

## Troubleshooting

- The workflow fails at the test step. Fix the tests on `main`, then delete the tag and re-tag after the fix.

  Warning: deleting a pushed tag rewrites published release state. Confirm that no release assets have been consumed before running it.

  ```bash
  git push origin :v1.0.0 && git tag -d v1.0.0
  ```

- GoReleaser fails. Check the syntax of `@.goreleaser.yaml`, and run `goreleaser check` locally when the binary is available.
- The wrong commit was tagged. Delete the remote tag, re-tag the correct commit, and push again.
- `install.sh` or `install.ps1` cannot find the release. Confirm that the tag follows the `v*` pattern, for example `v1.0.0` rather than `1.0.0`.
- The Windows zips are missing from the release. Confirm that `format_overrides` in `@.goreleaser.yaml` still maps `windows` to `zip`. Otherwise GoReleaser falls back to `.tar.gz` and `install.ps1` does not find the expected archives.
