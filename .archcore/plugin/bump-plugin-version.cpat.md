---
title: "Bump Plugin Version Across All Host Manifests"
status: accepted
tags:
  - "multi-host"
  - "plugin"
---

## Pattern

The plugin version is declared in **four per-host manifests** and nowhere else. On every release the same version string is bumped in all four, kept **byte-identical**, and the merge commit is then tagged `vX.Y.Z`.

The four files, with the plugin root at `plugins/archcore/`:

- `plugins/archcore/.claude-plugin/plugin.json`
- `plugins/archcore/.cursor-plugin/plugin.json`
- `plugins/archcore/.codex-plugin/plugin.json`
- `plugins/archcore/.plugin/plugin.json`

Set the same new `version` in all four and change nothing else. The marketplace catalogs, the MCP configs, and the hook JSON carry no plugin version.

The version source of truth is the latest git tag rather than the manifest. Compute the next version from the highest semantic-version tag with `git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n 1`, strip the leading `v`, apply the requested step — `patch` by default, otherwise `minor`, `major`, or an explicit `X.Y.Z` — and write the result. Deriving from the tag reconciles drift when a tag was cut without a manifest bump.

## Before

`plugins/archcore/.claude-plugin/plugin.json`, and its `.cursor-plugin`, `.codex-plugin`, and `.plugin` siblings:

```json
{
  "name": "archcore",
  "description": "Make your AI agent code with your project's architecture, rules, and decisions.",
  "version": "0.4.18",
  ...
}
```

## After

All four manifests, identically, for latest tag `v0.4.18` and a patch step:

```json
{
  "name": "archcore",
  "description": "Make your AI agent code with your project's architecture, rules, and decisions.",
  "version": "0.4.19",
  ...
}
```

Only the `version` line changes. `name` and `description` stay byte-identical across all four for cross-host parity, so leave them untouched.

## Scope

Exactly the four `plugin.json` manifests under `plugins/archcore/{.claude-plugin,.cursor-plugin,.codex-plugin,.plugin}/`. Nothing else in the repository declares the plugin version:

- The marketplace catalogs `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`, and `.agents/plugins/marketplace.json` carry no `version` field. Do not add one.
- `hooks/cursor.hooks.json` carries `"version": 1`, which is the hook-schema version rather than the plugin version. Do not touch it.
- `.archcore/.sync-state.json` carries `"version": 1`, which is the sync-manifest schema version. Do not touch it.
- No `package.json`, changelog, or hardcoded source constant carries the plugin version. The bundled launcher and its `bin/CLI_VERSION` were removed — see `remove-bundled-launcher-global-cli.idea`.

The bump edits files only. Merging, tagging with `git tag vX.Y.Z && git push origin vX.Y.Z`, and publishing are separate manual steps, documented in `docs/release.md` under "Cutting a release".

## Rationale

- **Single source, four copies.** Each host loads its own manifest and no shared version file exists, so the version is duplicated by necessity and the bump must fan out to all four or a host ships stale.
- **Parity is test-enforced.** `@test/structure/json-configs.bats` asserts the version matches across Claude Code and Cursor; `@test/structure/codex-plugin.bats` asserts the Codex manifest metadata, including `.version`, matches Claude Code; `@test/structure/copilot-plugin.bats` does the same for Copilot. Bumping one host and forgetting another turns CI red. [assumption] This is the most frequent release regression; no incident count has been recorded.
- **Tag-relative stepping avoids drift decisions.** The git tag drives the release workflow, which `.github/workflows/release.yml` triggers on `v*`. Computing the next version from the latest tag keeps the manifest and the tag lineage aligned even when a previous tag was cut on a docs commit without a manifest bump.
- **Mechanical, low-risk, and easy to do incompletely.** The change is four one-line edits, cheap to script, but the "identical across four files" invariant is what a human eye skips. A dedicated flow removes that failure mode.

## Enforcement going forward

- A version bump MUST update all four manifests to the same value in one change.
- A version bump MUST NOT land directly on `main`. `main` is synthesized from a tagged `dev` commit, per `docs/release.md`.
- The `/bump-plugin-version` local skill is the canonical way to perform this pattern: it reads this document, derives the next version from the latest tag, and edits the four manifests. Prefer it over hand-editing.
- After bumping, run `make test-structure`. The three parity tests named under Rationale check `name`, `description`, and `version` across the four manifests.

## Edge cases

- **Tag ahead of manifest.** A tag can be newer than the manifest `version`, when a tag was cut without a bump. Because the next version derives from the tag rather than the manifest, the bump moves forward from the true release marker instead of repeating a stale manifest value.
- **No tags yet.** If the semantic-version tag query returns nothing, fall back to the current manifest `version`, or seed `0.0.0`, step from there, and tell the user which fallback was used.
- **Pre-1.0 semantics.** These are ordinary semver bumps: `minor` resets patch to 0, and `major` resets minor and patch to 0. A `0.y.z` version is treated as normal semver, so a `major` step goes `0.y.z → 1.0.0` only when explicitly requested.
- **Explicit version.** When an exact `X.Y.Z` is supplied, write it verbatim with no tag arithmetic, still enforce it across all four files, and confirm it is greater than the latest tag.
