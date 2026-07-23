---
title: "Bump Plugin Version Across All Host Manifests"
status: accepted
tags:
  - "multi-host"
  - "plugin"
---

## Pattern

The plugin version is declared in **four per-host manifests** and nowhere else. On every release, the same version string is bumped in all four, kept **byte-identical**, then the merge commit is tagged `vX.Y.Z`.

The four files (plugin root is `plugins/archcore/`):

- `plugins/archcore/.claude-plugin/plugin.json`
- `plugins/archcore/.cursor-plugin/plugin.json`
- `plugins/archcore/.codex-plugin/plugin.json`
- `plugins/archcore/.plugin/plugin.json`

Set the same new `version` in all four. Do not touch anything else — the marketplace catalogs, MCP configs, and hook JSON carry no plugin version.

**Version source of truth is the latest git tag, not the manifest.** Compute the next version relative to the highest semantic version tag (`git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n 1`, strip the leading `v`), apply the requested step (default `patch`, else `minor` / `major` / an explicit `X.Y.Z`), and write the result. Deriving from the tag rather than the manifest reconciles drift when a tag was cut without a manifest bump.

## Before

`plugins/archcore/.claude-plugin/plugin.json` (and the `.cursor-plugin` / `.codex-plugin` / `.plugin` siblings):

```json
{
  "name": "archcore",
  "description": "Make your AI agent code with your project's architecture, rules, and decisions.",
  "version": "0.4.18",
  ...
}
```

## After

All four manifests, identically (latest tag `v0.4.18` → patch step → `0.4.19`):

```json
{
  "name": "archcore",
  "description": "Make your AI agent code with your project's architecture, rules, and decisions.",
  "version": "0.4.19",
  ...
}
```

Only the `version` line changes. `name` and `description` MUST remain byte-identical across all four (cross-host parity), so leave them untouched.

## Scope

Exactly the four `plugin.json` manifests under `plugins/archcore/{.claude-plugin,.cursor-plugin,.codex-plugin,.plugin}/`. Nothing else in the repo declares the plugin version:

- Marketplace catalogs (`.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`, `.agents/plugins/marketplace.json`) carry no `version` field — do not add one.
- `hooks/cursor.hooks.json` has `"version": 1` — that is the **hook-schema** version, not the plugin version. Do not touch.
- `.archcore/.sync-state.json` `"version": 1` is a sync-manifest schema version. Do not touch.
- No `package.json`, CHANGELOG, or hardcoded source constant carries the plugin version (the bundled launcher and its `bin/CLI_VERSION` were removed — see `remove-bundled-launcher-global-cli.idea`).

The bump edits files only. Merging, tagging (`git tag vX.Y.Z && git push origin vX.Y.Z`), and publishing are separate manual steps — see the release runbook (`docs/release.md`, "Cutting a release").

## Rationale

- **Single source, four copies.** Each host loads its own manifest; there is no shared version file. The version is duplicated by necessity, so the bump must fan out to all four or a host ships stale.
- **Parity is test-enforced.** `test/structure/json-configs.bats` asserts the version matches across Claude Code and Cursor; `test/structure/codex-plugin.bats` asserts the Codex manifest metadata (incl. `.version`) matches Claude Code; `test/structure/copilot-plugin.bats` does the same for Copilot. Bumping one host and forgetting another turns CI red — this is the single most common release regression.
- **Tag-relative stepping avoids drift decisions.** The git tag drives the release workflow (`.github/workflows/release.yml` triggers on `v*`). Computing the next version from the latest tag keeps the manifest and the tag lineage aligned, even if a previous tag was cut on a docs commit without a manifest bump.
- **Mechanical and low-risk, but easy to do incompletely.** The change is four one-line edits — cheap to script, but the "identical across four files" invariant is exactly what a human eye skips. A dedicated flow (the `/bump-plugin-version` skill) removes that failure mode.

## Enforcement going forward

- Any version bump MUST update all four manifests to the same value in one change.
- Never bump the version directly on `main` — `main` is synthesized from a tagged `dev` commit (`docs/release.md`).
- The `/bump-plugin-version` local skill is the canonical way to perform this pattern: it reads this cpat, derives the next version from the latest tag, and edits the four manifests. Prefer it over hand-editing.
- After bumping, verify parity — `verify-plugin-integrity` Section 4 (cross-host consistency) checks `name`/`description`/`version` across the four manifests.

## Edge cases

- **Tag ahead of manifest.** A tag can exist that is newer than the manifest `version` (a tag cut without a bump). Because the next version is derived from the tag, not the manifest, the bump naturally moves forward from the true release marker rather than repeating a stale manifest value.
- **No tags yet.** If the semantic-version tag query finds nothing, fall back to the current manifest `version` (or seed `0.0.0`) and step from there; surface the fallback to the user.
- **First-digit/pre-1.0 semantics.** These are ordinary semver bumps — `minor` resets patch to 0, `major` resets minor and patch to 0. Pre-1.0 (`0.y.z`) is treated as normal semver here; a `major` step goes `0.y.z → 1.0.0` only when explicitly requested.
- **Explicit version.** When an exact `X.Y.Z` is provided, write it verbatim (no tag arithmetic), but still enforce it across all four files and that it is greater than the latest tag.
