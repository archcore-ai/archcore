---
title: "Plugin Source Migration During the Repository Cutover"
status: draft
tags:
  - "component:cli"
  - "integrations"
  - "update"
---

## Purpose & Scope

The plugin update executor prepares installed Claude Code and Codex marketplaces for the coordinated repository rename. Dependents: manual `archcore update` and `archcore plugin update`. Fresh installation, unattended CLI updates and marketplace identifiers retain their existing contracts.

## Surface

- Active delivery is `archcore-ai/archcore` since `RepoID` changed on 2026-09-22 (unified release v0.10.2); GitHub redirects the legacy address `archcore-ai/plugin`.
- Marketplace `archcore-plugins`, plugin `archcore@archcore-plugins`, public `main` and `plugins/archcore` retain their spelling.
- Claude source declarations: known marketplaces, user settings and settings files of listed, resolved project/local installations.
- Codex source declaration: the native `[marketplaces.archcore-plugins]` table, holding `source_type` and `source`, plus the refresh fields `last_updated` and `last_revision` when a refresh wrote them, and `sparse_paths` when the install or this migration wrote it.
- Codex sparse paths: `[".agents/plugins", "plugins/archcore"]`. Codex `marketplace upgrade` reads `sparse_paths` from this table and then clones without file contents.
- Cursor retains its UI update path. Copilot recognizes both legacy and canonical direct-install directory names.
- Implementation: @cli/internal/plugin/source_migration.go.

## Normative Behavior

1. WHEN the active repository is canonical, the executor MUST prepare source migration before an explicitly planned update runs.
2. WHEN the active repository remains legacy, the executor MUST leave source declarations unchanged.
3. WHEN a source is a recognized unpinned official address, the migrator MUST replace only its repository locator.
4. WHEN a source carries custom fetch fields, the migrator MUST preserve that declaration.
5. WHEN Claude declarations disagree with the supported official source shapes, the migrator MUST leave that host's declarations unchanged.
6. BEFORE changing a declaration, the migrator MUST verify the canonical marketplace name and plugin path over a bounded read.
7. BEFORE replacing a settings file, the migrator MUST save its original bytes in a restricted sibling backup.
8. WHEN migration encounters an unreadable or concurrently changed file, the executor MUST report failure without running that host's update commands.
9. WHEN a repeated migration finds canonical sources, the migrator MUST perform no write.
10. WHEN the Codex table names an official source and has no `sparse_paths`, the migrator MUST add the Codex sparse paths, and WHEN it already has `sparse_paths`, the migrator MUST keep its value.

## Constraints & Invariants

1. The migrator MUST NOT remove a marketplace, uninstall a plugin, rewrite managed declarations or user-selected refs, or send a telemetry event.
2. The migration probe MUST finish within 10 seconds inside the existing whole-step deadline.
3. The migrator MUST reject config files over 1 MiB and catalogs over 64 KiB.
4. The Claude migrator MUST refuse more than 66 candidate settings paths rather than migrate a prefix.
5. The Codex migrator MUST preserve unsupported TOML shapes instead of rewriting the whole configuration.

## Failure Behavior

1. IF the canonical catalog is unavailable or differs in identity, THEN the executor MUST leave the legacy settings intact.
2. IF backup creation fails, THEN the migrator MUST leave the affected settings file intact.
3. IF a later file write fails, THEN the migrator MUST retain completed edits and their backups for a resumable retry.
4. IF one host fails migration, THEN the executor MUST continue with other hosts inside the remaining deadline.

## Conformance

Source migration is not evidence of a successful public GitHub rename.

The local native-host probe on 2026-09-22 (Claude Code 2.1.278, Codex CLI 0.155.1) showed that both hosts reject `marketplace add` for an existing name with another source. Codex marketplace remove temporarily hides installed plugins. Direct declaration changes followed by native refresh preserve the plugin ID; Claude updated 0.9.4 to 0.9.5 and Codex updated 0.9.5 to 0.9.6, both retaining `enabled = false`. The probe used isolated homes and local Git fixtures, not user settings or production repositories.

A check on 2026-09-24 (Codex CLI 0.155.1) found a user table that carried `last_updated` and `last_revision`, which the Codex CLI's own `marketplace add` and `marketplace upgrade` do not write. In an isolated home, `marketplace upgrade` with a stale `last_revision` fetched the head of the default branch, so those two fields pin no revision.

Remaining release gate: old-to-new remote update on all four hosts, actual GitHub redirects, supported historical host versions and Cursor's existing marketplace card.
