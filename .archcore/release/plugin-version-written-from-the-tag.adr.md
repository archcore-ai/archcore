---
title: "The Release Writes the Plugin Version From the Tag; dev Manifests Stay at 0.0.0"
status: draft
tags:
  - "component:plugin"
  - "release"
---

## Context

Two of the five tags `v0.10.0` to `v0.10.4` were pushed ahead of the four plugin manifests: `v0.10.0` shipped with the manifests at `0.9.4`, and the first `v0.10.4` Release run (Actions run 35992806707, 2026-09-24) failed in `verify-version` because the manifests still read `0.10.3`. The version lived in five places, the tag and four `plugin.json` files, and only a manual `/bump-plugin-version` step kept them equal. The gate stopped the second case before publication, but recovery meant deleting a pushed tag and tagging again (@plugin/docs/release.md, Recovery).

## Decision

The Release workflow writes the tag's version into the four published plugin manifests through `scripts/export-plugin.sh OUTPUT VERSION`, the four source manifests on `dev` carry the placeholder `0.0.0`, and `verify-version` runs `scripts/check-release-tag.sh`, which accepts only the next patch, minor, or major version after the highest other `vX.Y.Z` tag.

## Alternatives Considered

1. A `make release VERSION=X.Y.Z` script that bumps the four manifests, commits, tags, and pushes in one command — rejected because it keeps five copies of the version, and a hand-typed `git tag` still skips the script.
2. A local `.githooks/pre-push` hook that compares a pushed tag with the manifests — rejected because `core.hooksPath` is opt-in per clone, `git push --no-verify` skips the hook, and the hook duplicates the manifest loop of @.github/workflows/release.yml.
3. release-please with a release pull request that bumps the manifests and creates the tag — rejected because it adds an external bot and a pull-request step to the release path, and its default `CHANGELOG.md` output is forbidden by `AGENTS.md`.

## Consequences

### Positive

- A tag ahead of the manifests cannot occur: no source file carries a release version. @plugin/test/structure/release-blocklist.bats checks that the export writes the given version into all four manifests and changes no other line.
- A release is one command, `git tag vX.Y.Z && git push origin vX.Y.Z`. The bump commit and the `/bump-plugin-version` skill are removed.
- `verify-version` rejects a mistyped tag: a skip (`v0.10.40` after `v0.10.4`), a repeat, or a step back. @plugin/test/structure/release-tag.bats covers these cases. The old gate caught them only through a manifest mismatch.
- [expected] A local install from `dev` at `0.0.0` never shares a version-keyed cache directory, such as `~/.codex/plugins/cache/<marketplace>/archcore/<version>/`, with a released install.

### Negative

- The manifests on `dev` no longer show the released version. A plugin loaded from a `dev` checkout reports `0.0.0` to its host.
- `verify-version` refuses a deliberate version skip. A skip needs a change to @scripts/check-release-tag.sh.
- A re-run of an older tag's Release run fails once a newer tag exists, because the newer tag becomes the previous release.
- @scripts/export-plugin.sh depends on `jq` to confirm the written version.

## Superseded when

- A host installs the plugin from the `dev` source tree instead of the generated `main` tree, so the `0.0.0` placeholder reaches users.
- A second release train with its own version line, such as a CLI-only hotfix line, is introduced; @scripts/check-release-tag.sh rejects its tags as a step back.