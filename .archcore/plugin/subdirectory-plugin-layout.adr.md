---
title: "Plugin Content Relocated to plugins/archcore/ for Multi-Host Marketplace Discovery"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

`codex plugin marketplace add archcore-ai/plugin` registered the marketplace but Codex never discovered the `archcore` plugin, and `codex plugin add archcore@archcore-plugins` failed with `plugin 'archcore' was not found` — GitHub issue #2, reported on Codex CLI 0.137.0 on Windows and reproduced locally on 0.135.0. The root cause was that all three host marketplace catalogs pointed the plugin `source` at the repository root, as `"./"` or `"."`, while Codex requires `source.path` to be a dedicated subdirectory containing `.codex-plugin/plugin.json` and does not scan the marketplace root even when the manifest physically exists there, which was verified against a cached snapshot. Claude Code and Cursor tolerate both root and subdirectory sources, which is why only Codex broke. The plugin's shared content — `skills/`, `bin/`, `hooks/`, `agents/`, `commands/`, `rules/` — is auto-discovered by each host at its own plugin root, so it could not sit at the repo root for Claude Code and Cursor and in a subdirectory for Codex without duplication or symlinks, and symlinks break on Windows, the reporter's platform.

A documentation cross-check established the subdirectory as the canonical layout for every host: Codex (`developers.openai.com/codex/plugins/build`) requires `source.path` of `./<subdir>` and does not support the marketplace root; Claude Code (`code.claude.com/docs/en/plugin-marketplaces`) resolves a relative `source: "./plugins/<name>"` against the marketplace root and treats subdirectory sources as first-class; Cursor (`github.com/cursor/plugins`, `cursor/plugin-template`) documents `plugins/<name>/.cursor-plugin/plugin.json` with `source: "./plugins/<name>"` as canonical and tolerates the root form.

## Decision

Relocate the entire plugin into the dedicated subdirectory **`plugins/archcore/`** and point all three marketplace catalogs at it, keeping one copy of every shared component as the single source of truth.

Moved under `plugins/archcore/`, as host-runtime-loaded content: `.codex-plugin/plugin.json`, `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `skills/`, `agents/`, `commands/`, `hooks/`, `bin/`, `assets/`, `rules/`, `.codex.mcp.json`, and `.mcp.json`. Kept at the repo root: the three marketplace catalogs (`.agents/plugins/marketplace.json`, `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`), each with `source` and `path` set to `./plugins/archcore`, plus the dev tooling and non-runtime files `test/`, `Makefile`, `.github/`, `.gitmodules`, `.archcore/`, `reference-materials/`, `docs/`, `README.md`, `LICENSE`, and `NOTICE`.

This is the canonical "catalog at root, plugin manifest in subdirectory" pattern shared by all three hosts. `docs/`, `README.md`, `LICENSE`, and `NOTICE` stay at the root because no host loads them at runtime — `termsOfServiceURL` is an absolute GitHub blob URL — so leaving them avoids churn while every reference stays valid.

## Alternatives Considered

1. **A Codex-only generated subdirectory**, keeping Claude Code and Cursor at the root and copying shared content into `plugins/archcore/` through a build step — rejected because the copy must be committed, since Codex git-clones the repository and reads `path`, so it becomes a permanent drift surface plus a CI sync check and build machinery, an ongoing cost incurred to avoid a one-time, test-covered migration.
2. **Symlinking the shared content into a Codex subdirectory** — rejected because symlinks break on Windows, the reporter's platform, and in archive-based installs.
3. **Leaving the layout and documenting the limitation** — rejected because it leaves the Codex install broken, which is the actual bug.

## Consequences

- The fix itself is three catalog `source` and `path` edits; the bulk of the change was relocating the content those catalogs point at, through `git mv`, so history is preserved.
- Regression coverage was added: a structure-level guard asserts that every catalog `source` resolves to a subdirectory that is not the marketplace root — manifest presence alone is insufficient, because it passed under the bug — and the Codex integration smoke test now runs the real `marketplace add → plugin list → plugin add` cycle instead of a symlinked fake, which is what let the bug ship green.
- The dev `.archcore/` sits at the repo root, outside `plugins/archcore/`, so it is naturally excluded from the per-host plugin install subtree, with the release strip remaining as a second line of defense.
- Tradeoff: the test harness now needs a `REPO_ROOT` distinct from `PLUGIN_ROOT`, which is `…/plugins/archcore`. Tests for root-staying artifacts use `REPO_ROOT`; everything else rides along with `PLUGIN_ROOT`.
- Tradeoff: release synthesis for `dev → main` had to be updated, so the `.archcore`-reference grep guard scans `plugins/archcore/{skills,agents,commands,rules,hooks,bin}` plus `README.md`.
- The version was bumped to 0.4.9.
- The directory trees in `multi-host-plugin-architecture.adr` and `component-registry.doc` describe the layout question as this decision settles it.

## Superseded when

- Codex adds support for a marketplace-root `source.path`, which would remove the constraint that forced the relocation.
- A second plugin ships from this repository, which would make `plugins/archcore/` one of several roots and reopen the catalog-per-plugin question.
