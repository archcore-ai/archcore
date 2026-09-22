---
title: "Release the Plugin and the CLI Together From One Tag on the Monorepo"
status: draft
tags:
  - "architecture"
  - "component:cli"
  - "component:plugin"
  - "release"
  - "update"
---

## Context

On 2026-09-22 the owner decided that `archcore-ai/cli` will be archived with its CI stopped, and that the plugin and the CLI release together under one version. Before this decision the root @.github/workflows/release.yml published only the exported plugin tree on every `v*` tag, the CLI publication workflow copied to `cli/.github/workflows/` did not execute in this repository, and the two CLI commits after the import (a5ac3fb, d6dd53e) had no publication path. The plugin last shipped as v0.10.0 with its four manifests still at 0.9.4, and the CLI last shipped as v0.8.7 from commit 87a1b5d, which is the exact import point of `cli/`.

## Decision

Publish the exported plugin tree and the GoReleaser-built CLI archives from one `vX.Y.Z` tag on this repository in one Release workflow (@.github/workflows/release.yml), with the four plugin manifests and the CLI `main.version` equal to that tag, starting at v0.10.1 (owner confirmation 2026-09-22).

## Alternatives Considered

1. Mirror `cli/` into `archcore-ai/cli` with `git subtree push` and keep its tag-driven workflows. A local `git subtree split --prefix=cli dev` on 2026-09-22 fast-forwards from 87a1b5d with two commits, so the mirror works mechanically. Ruled out because the owner archives that repository and its CI stops running.
2. Independent version trains in this repository with prefixed tags `cli/v*` and `plugin/v*`. Rejected because GoReleaser's monorepo tag filtering (`monorepo.tag_prefix`, `monorepo.dir`) is a GoReleaser Pro feature (goreleaser.com/customization/monorepo, read 2026-09-22), and because one `releases/latest` redirect serves both trains, so the resolver in @cli/internal/update/update.go reads a plugin tag as the CLI version.
3. Build the CLI in this repository and publish its releases to a separate release-only repository through `release.github.owner` and `release.github.name`. Rejected because it keeps two `latest` pointers and two repositories to redirect while the owner wants one synchronized release.
4. One bridge release to `archcore-ai/cli` before archiving, so installs at or below 0.8.7 learn the new home through `archcore update`. Declined by the owner on 2026-09-22; older installs move through a fresh run of the archcore.ai installer.

## Consequences

### Enabled

- [expected] One tag produces one GitHub Release that carries the plugin tree on `main` and 9 CLI assets (6 archives, `checksums.txt`, `install.sh`, `install.ps1`), so `https://github.com/archcore-ai/plugin/releases/latest` always resolves to a release with CLI archives.
- [expected] The v0.10.0 defect, a tag ahead of the manifests, becomes a workflow failure: the `verify-version` job compares the tag with the four manifests before any test or publish step.
- The two unreleased CLI commits ship in v0.10.1, including the dormant source migration of @cli/internal/plugin/source_migration.go.
- The installer mechanics stay unchanged: @cli/install.sh and @cli/install.ps1 resolve the tag from the `GITHUB_REPO/releases/latest` redirect, download `releases/download/v<version>/<archive>` and `checksums.txt`, accept an `ARCHCORE_VERSION` pin, and carry one `__POSTHOG_KEY__` placeholder that the landing deploy substitutes. Only `GITHUB_REPO` changed, to `archcore-ai/plugin`.

### Costs and boundaries

- Every installed CLI at or below 0.8.7 resolves updates from `archcore-ai/cli/releases/latest`. After the archive those installs report "up to date" on `archcore update` and on the SessionStart version check; unattended update does not run in 0.8.7 (`RunUnattended` in @cli/internal/update/policy.go has no caller outside tests). Accepted by the owner on 2026-09-22.
- The redirect resolver halts at the first 3xx and requires a `/releases/tag/` path. A renamed repository answers first with a 301 to the new name, so a binary that carries the old name fails its update check after the rename with a clean error. The later rename to `archcore-ai/archcore` therefore ships a release carrying the new name before the rename, or a resolver that follows one same-path 301.
- The official-build marker and `POSTHOG_KEY` injection are gated on `github.repository_id` 1201781375 instead of the repository name, so a rename keeps them and a fork loses them. The `POSTHOG_KEY` and `POSTHOG_HOST` variables were copied into this repository on 2026-09-22; without them `@cli/scripts/assert-not-inert.sh` fails the release.
- `archcore-ai/landing` fetches `install.sh` and `install.ps1` from `raw.githubusercontent.com/archcore-ai/cli/refs/heads/main` on an `installer-updated` dispatch, and its nightly `install-stats.yml` counts asset downloads with `CLI_REPO: archcore-ai/cli`. Until the fetch base, the stats repository, and the dispatcher move, archcore.ai serves installers that install 0.8.7 from the archived repository, and the download counts stop growing.
- The first republish of the installers on archcore.ai waits for v0.10.1: an installer pointing at `archcore-ai/plugin` before that release resolves `latest` to v0.10.0, which carries no CLI assets, and every install fails.
- The CLI version jumps from 0.8.7 to 0.10.1. The plugin gates `cli-gte 0.7.0` in @plugin/plugins/archcore/skills/init/SKILL.md keep passing.
- The plugin CI in @.github/workflows/test.yml downloads CLI 0.8.6 from `archcore-ai/cli/releases/download`. [assumption] An archived repository keeps serving existing assets, so the pin works until it moves to this repository's v0.10.1 assets.

## Superseded when

- A CLI-only or plugin-only hotfix is required more than twice in one calendar month.
- A package-manager channel (Homebrew tap, Scoop manifest, winget package) requires a version scheme that differs from the shared tag.