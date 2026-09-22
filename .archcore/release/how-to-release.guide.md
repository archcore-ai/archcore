---
title: "How to Create a New Release"
status: accepted
tags:
  - "component:cli"
  - "release"
---

## Purpose

Publish one release of the plugin and the CLI from one git tag on `archcore-ai/archcore`, with the tag, the four plugin manifests, and the CLI version equal.

The former `archcore-ai/cli` channel receives no further releases, and this repository was renamed from `archcore-ai/plugin` on 2026-09-22; GitHub redirects the old address for git and web traffic. The workflow copies that lived under `cli/.github/workflows/` were removed the same day.

## Prerequisites

- Push access to the `archcore-ai/archcore` repository.
- Every intended change merged to `dev`, with `Plugin Tests` and `CLI Tests` green on `dev`.
- Repository variables `POSTHOG_KEY` and `POSTHOG_HOST` present in `archcore-ai/archcore` (copied 2026-09-22). Without them the GoReleaser post-build hook @cli/scripts/assert-not-inert.sh fails the release.

## Procedure

1. Check out `dev` and pull.

   ```bash
   git checkout dev
   git pull origin dev
   ```

2. Choose one semver version for both components, higher than the latest `v*` tag.

   - `v1.0.0` — first stable release, or a breaking change
   - `v1.1.0` — new features, backwards compatible
   - `v1.1.1` — bug fixes only

3. Set the four manifests under `plugin/plugins/archcore/` to that version with `/bump-plugin-version X.Y.Z`.

4. Commit the bump on `dev` and push it.

5. Wait for `Plugin Tests` and `CLI Tests` on that commit.

6. Create and push the tag.

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

7. Monitor the `Release` run at `https://github.com/archcore-ai/archcore/actions`.

   Expected result: `verify-version`, `test-plugin`, `test-cli`, `publish-plugin`, and `publish-cli` complete in that order (@.github/workflows/release.yml).

8. Verify the published release.

   ```bash
   gh release view vX.Y.Z
   git fetch origin main && git ls-tree --name-only origin/main
   ARCHCORE_VERSION=vX.Y.Z bash cli/install.sh
   archcore --version
   ```

   Expected result: 9 assets on the release page (6 archives, `checksums.txt`, `install.sh`, `install.ps1`), the plugin layout on `main`, and `vX.Y.Z` from `archcore --version`.

Run the installer from the checkout, not from `archcore.ai`. The checkout copy carries the `__POSTHOG_KEY__` placeholder, so a release check installs without reporting an install event. See install-script-usage.guide.md.

## One version for both components

The plugin's `cli-gte` constants in `skills/init/SKILL.md` and `bin/cli-gte` name the minimum CLI a feature needs. WHEN a plugin change needs a new CLI capability, set that constant to the release version that ships the capability; both halves ship in the same tag, so the constant and the tag never diverge.

## Sequencing a release that adds a settings.json field

`Settings` parsing is forward-compatible: the tolerant parser captures an unknown config field in `Settings.Extra` with a soft warning instead of a hard error. Tolerance protects only binaries that already ship it. A CLI released before the tolerant parser hard-fails on any unknown field.

Requirement: a release that introduces a new `settings.json` field MUST ship no earlier than the release that carries the tolerant parser, which shipped with the globals rollout.

When planning such a release:

1. Confirm that the previous released version already tolerates unknown fields. Every release from the globals rollout onward does; keep this check for a long-lived maintenance branch.
2. Ship the parser and validation change in an earlier, separate release from the feature that writes the new field, so a user on the intermediate version upgrades smoothly in either order.
3. Do not backport a new config field to a branch whose parser is still strict.

## Publishing an installer change

The installers live at @cli/install.sh and @cli/install.ps1 on `dev`. They resolve the newest release from `https://github.com/archcore-ai/archcore/releases/latest`, so an installer change needs no release of its own.

1. Merge the change to `dev`. `CLI Install Smoke` (@.github/workflows/cli-install-smoke.yml) runs on the push and installs the latest release on Windows, Ubuntu, macOS, Alpine, and a dash-only Debian.
2. If the smoke run is red, fix it before publication.
3. Trigger the landing deploy. [assumption] The dispatcher from this repository is pending (plan `release/unified-release-cutover`, task 19); until it exists, run the landing deploy by hand.
4. The landing deploy re-fetches both installers, substitutes the PostHog key, and republishes them.

## Verification

- The GitHub Release page shows 6 archives (4 `.tar.gz` for darwin and linux on amd64 and arm64, 2 `.zip` for windows on amd64 and arm64), `checksums.txt`, `install.sh`, and `install.ps1`.
- `archcore --version` on the installed binary prints the tag, for example `v0.10.2`.
- `main` carries only the exported plugin layout.
- The install script succeeds on a clean macOS or Linux machine.
- `install.ps1` succeeds on a clean Windows machine.

The related reference document on release infrastructure carries the full build matrix, artifact naming, and update paths.

## Troubleshooting

- `verify-version` fails: a manifest differs from the tag. Fix the manifests on `dev`, delete the tag, and tag the corrected commit.

  Warning: deleting a pushed tag rewrites published state. Do it only while no GitHub Release exists for the tag.

  ```bash
  git push origin :vX.Y.Z && git tag -d vX.Y.Z
  ```

- A test job fails: nothing was published. Fix on `dev`, delete the tag as above, and re-tag.
- `publish-cli` fails after `publish-plugin`: `main` already carries the new tree. Fix the cause, delete the partial GitHub Release if one exists while keeping the tag, and re-run the failed job. GoReleaser replaces conflicting assets on a re-run (`release.replace_existing_artifacts` in @cli/.goreleaser.yaml).
- GoReleaser fails: check the syntax of @cli/.goreleaser.yaml, and run `goreleaser check` locally when the binary is available.
- `install.sh` or `install.ps1` cannot find the release: confirm that the tag follows the `v*` pattern, for example `v1.0.0` rather than `1.0.0`.
- The Windows zips are missing from the release: confirm that `format_overrides` in @cli/.goreleaser.yaml still maps `windows` to `zip`.
- archcore.ai still serves an old script: check the landing deploy's installer fetch path and its last run.