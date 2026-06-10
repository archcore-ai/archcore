---
title: "Plugin Content Relocated to plugins/archcore/ for Multi-Host Marketplace Discovery"
status: accepted
---

## Context

`codex plugin marketplace add archcore-ai/plugin` registered the marketplace but Codex never discovered the `archcore` plugin; `codex plugin add archcore@archcore-plugins` failed with `plugin 'archcore' was not found` (GitHub issue #2, reported on Codex CLI 0.137.0/Windows, reproduced locally on 0.135.0).

Root cause: all three host marketplace catalogs pointed the plugin `source` at the repository **root** (`"./"` / `"."`). Codex **requires** `source.path` to be a dedicated **subdirectory** containing `.codex-plugin/plugin.json`; it does not scan the marketplace root for a plugin, even when the manifest physically exists there (verified: the cached snapshot had `.codex-plugin/plugin.json` at root and was still not discovered). Claude Code and Cursor tolerate both root and subdirectory `source`, which is why only Codex broke.

The plugin's shared content (`skills/`, `bin/`, `hooks/`, `agents/`, `commands/`, `rules/`) is auto-discovered by each host at its own plugin root. It therefore cannot live at the repo root for Claude/Cursor **and** in a subdirectory for Codex without duplicating or symlinking — and symlinks break on Windows (the reporter's platform).

Documentation cross-check (least-risk basis — subdirectory is the documented/canonical layout for every host):
- **Codex** (`developers.openai.com/codex/plugins/build`): `source.path` must be `./<subdir>`; marketplace root is unsupported.
- **Claude Code** (`code.claude.com/docs/en/plugin-marketplaces`): relative `source: "./plugins/<name>"` resolves against the marketplace root; a `metadata.pluginRoot` shortcut exists. Subdirectory sources are first-class.
- **Cursor** (`github.com/cursor/plugins`, `cursor/plugin-template`): canonical tree is `plugins/<name>/.cursor-plugin/plugin.json` with `source: "./plugins/<name>"`; root (`"./"`) is also tolerated.

## Decision

Relocate the entire plugin into a dedicated subdirectory **`plugins/archcore/`** and point all three marketplace catalogs at it. Single source of truth — one copy of every shared component.

- Moved under `plugins/archcore/` (host-runtime-loaded content): `.codex-plugin/plugin.json`, `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `skills/`, `agents/`, `commands/`, `hooks/`, `bin/`, `assets/`, `rules/`, `.codex.mcp.json`, `.mcp.json`.
- Kept at the **repo root**: the three marketplace catalogs (`.agents/plugins/marketplace.json`, `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`), each with `source`/`path` = `./plugins/archcore`; plus dev tooling and non-runtime files (`test/`, `Makefile`, `.github/`, `.gitmodules`, `.archcore/`, `reference-materials/`, `docs/`, `README.md`, `LICENSE`, `NOTICE`).

This is the canonical "catalog at root, plugin manifest in subdirectory" pattern shared by all three hosts. `docs/`, `README.md`, `LICENSE`, `NOTICE` stay at the root because they are not loaded by any host at runtime (e.g. `termsOfServiceURL` is an absolute GitHub blob URL), so leaving them avoids churn while keeping every reference valid.

## Alternatives

- **Codex-only generated subdirectory** — keep Claude/Cursor at the root and copy shared content into `plugins/archcore/` for Codex via a build step. Rejected: the copy must be committed (Codex git-clones the repo and reads `path`), so it becomes a permanent drift surface plus a CI sync-check and build machinery — ongoing cost to avoid a one-time, test-covered migration.
- **Symlink the shared content into a Codex subdirectory** — rejected: symlinks break on Windows (the reporter's platform) and in archive-based installs.
- **Leave the layout, document the limitation** — rejected: it leaves Codex install broken, which is the actual bug.

## Consequences

- The fix itself is the three catalog `source`/`path` edits; the bulk of the change is relocating the content the catalogs now point at (`git mv`, history preserved).
- Test harness gains a `REPO_ROOT` (repo root) distinct from `PLUGIN_ROOT` (now `…/plugins/archcore`). Tests for root-staying artifacts (catalogs, `README.md`, `docs/`, `.github/`) use `REPO_ROOT`; everything else rides along with `PLUGIN_ROOT`.
- Release synthesis (`dev → main`) updated: the `.archcore`-reference grep guard now scans `plugins/archcore/{skills,agents,commands,rules,hooks,bin}` + `README.md`. The dev `.archcore/` sits at the repo root, outside `plugins/archcore/`, so it is naturally excluded from the per-host plugin install subtree (the strip remains the belt-and-suspenders).
- Regression coverage added: a structure-level guard asserts every catalog `source` resolves to a subdirectory that is **not** the marketplace root (manifest-presence alone is insufficient — it passed under the bug); and the Codex integration smoke test now runs the real `marketplace add → plugin list → plugin add` cycle instead of a symlinked fake (the fake is what let the bug ship green).
- Version bumped to 0.4.9.
- The directory trees in `multi-host-plugin-architecture.adr` and `component-registry.doc` (which show the old root layout) are superseded by this ADR for the layout question; full text sync of those docs is follow-up.
