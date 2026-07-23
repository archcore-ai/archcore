---
title: "How to Create a New Release"
status: accepted
tags:
  - "release"
---

## Prerequisites

- Push access to the `archcore-ai/cli` repository
- All changes merged to `main`
- Tests passing on `main`

## Steps

1. **Ensure you're on the latest main**

   ```bash
   git checkout main
   git pull origin main
   ```

2. **Choose a version following semver**

   - `v1.0.0` — first stable release or breaking changes
   - `v1.1.0` — new features, backwards compatible
   - `v1.1.1` — bug fixes only

3. **Create and push the tag**

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. **Monitor the release workflow**

   The GitHub Actions workflow (`.github/workflows/release.yml`) triggers automatically. It will:
   - Run `go test ./...`
   - Build binaries for darwin/linux/windows × amd64/arm64 (6 platforms)
   - Create a GitHub Release with archives and `checksums.txt`

   Watch progress at: `https://github.com/archcore-ai/cli/actions`

5. **Verify the release**

   ```bash
   # Check the GitHub Release page has all 7 assets (6 archives + checksums.txt)
   gh release view v1.0.0

   # Test the install script (macOS/Linux)
   ARCHCORE_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/archcore-ai/cli/main/install.sh | bash

   # Test the install script (Windows PowerShell)
   $env:ARCHCORE_VERSION="v1.0.0"; irm https://raw.githubusercontent.com/archcore-ai/cli/main/install.ps1 | iex

   # Verify the installed binary
   archcore --version
   # Expected: archcore 1.0.0 (commit: <sha>)
   ```

## Coordinated Release with the Claude Plugin

The Claude plugin's `SKILL.md` pins a **minimum CLI version** (currently `v0.6.0`) and gates `/archcore:init` on it: an older `archcore --version` makes the plugin skip the `install_host_config` MCP tool and ask the user to update. That constant and the actual CLI release tag must agree, and the order of operations matters.

**Rollout order — CLI first, plugin second:**

1. Tag and release the CLI with **exactly** the version the plugin's SKILL.md constant names. If the tag ends up different (hotfix bump, renumbering), update the SKILL.md constant atomically with the plugin release instead — never let them diverge.
2. Verify the release assets are live before touching the plugin: `gh release view vX.Y.Z` shows all 7 assets and `archcore update` on an older install sees the new version.
3. Only then release the plugin (from its `dev` branch).

**Why a mismatch is a real failure, in both directions:**

- Tag **behind** the constant (e.g. constant says `v0.6.0`, CLI shipped as `v0.6.1` only): the gate rejects users running a perfectly capable CLI.
- Feature lands in a **later** tag than the constant (constant `v0.6.0`, but the tool only exists from `v0.6.1`): the gate admits CLIs without the tool and the plugin fails on an unknown tool call mid-flow.

## Verification

- GitHub Release page shows 6 archives (4 `.tar.gz` for darwin/linux amd64+arm64, 2 `.zip` for windows amd64+arm64) plus `checksums.txt`
- `archcore --version` on installed binary shows correct version and commit
- Install script succeeds on a clean macOS/Linux machine
- `install.ps1` succeeds on a clean Windows machine
- When the release changes the plugin contract: the plugin's SKILL.md minimum-version constant equals the tag just pushed

See [Release Infrastructure Overview](release-infrastructure.doc.md) for the full build matrix, artifact naming, and update paths.

## Common Issues

- **Workflow fails at test step** — Fix the tests on `main`, delete the tag (`git push origin :v1.0.0 && git tag -d v1.0.0`), then re-tag after fixing.
- **GoReleaser fails** — Check `.goreleaser.yaml` syntax. Run `goreleaser check` locally if available.
- **Wrong commit tagged** — Delete the remote tag, re-tag the correct commit, and push again.
- **install.sh / install.ps1 can't find the release** — Ensure the tag follows the `v*` pattern (e.g. `v1.0.0`, not `1.0.0`).
- **Windows zips missing from release** — Confirm `.goreleaser.yaml` `format_overrides` still maps `windows` to `zip`; otherwise GoReleaser will fall back to `.tar.gz` and `install.ps1` won't find the expected archives.
