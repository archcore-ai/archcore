---
title: "Plugin Evolution History — Thin Wrapper to Four Commands over Gated Tracks"
status: accepted
tags:
  - "plugin"
  - "reference"
---

## Overview

A dated record of how the plugin reached its current shape. This document replaces the retired development roadmap: every phase below is delivered, and the current-state truth lives in `plugin-architecture.spec`, `command-surface-v2.spec`, `track-layer.spec`, `hooks-validation-system.spec`, and `component-registry.doc`. Read this for the "how it got here", never for the "what it is now".

## Content

**Phase 1 — documentation (dogfooding).** The project documented itself with Archcore's own document types before any skill shipped.

**Phase 2–3 — the skill build-out and first collapse.** The skill surface peaked at 34 directories across intent, track, type, and utility tiers, then consolidated in recorded steps: `remove-document-type-skills.adr` removed the type layer (34 → 18); `merge-review-status-remove-graph.adr` merged `status` into `review` and removed `graph` (18 → 16); `skill-surface-collapse.adr` removed the track tier, merged `actualize` into `audit`, renamed `bootstrap` to `init`, merged `standard` into `decide`, and removed `verify` (16 → 7). Two agents shipped: the read/write `archcore-assistant` and the read-only `archcore-auditor`.

**Phase 4 — shell hooks.** Six hook entries per host implemented the write guard, code-alignment injection, post-mutation validation, cascade, and precision checks as POSIX shell under `bin/`. This layer was later repatriated into the CLI (see the v2 cutover below).

**Phase 5 — multi-host support.** Adapter layers for Cursor, Codex CLI, and GitHub Copilot CLI: per-host manifests, hooks configs, marketplace catalogs, command wrappers, TOML agent variants, and the stdin normalization library. The bundled CLI launcher shipped briefly and was removed in v0.4.0 (2026-05-12) after eight bug classes — the CLI installs globally from the official installer (`remove-bundled-launcher-global-cli.idea`). MCP isolation followed: Claude Code reads `.claude.mcp.json` through a manifest pointer, Cursor and Copilot get project-level wiring only (`cursor-mcp-architecture.adr`, `copilot-mcp-architecture.adr`).

**Phase 6 — zero-content onboarding.** The SessionStart empty-state nudge and the `/archcore:init` first-day seed, later expanded to scale modes and full host wiring (`magic-first-day-init.adr`, `host-wiring-parity.adr`).

**v2 — four commands over gated tracks (2026-08).** `four-command-palette.adr` collapsed the seven skills to `init / plan / document / review`, with flow logic moved into gated track files under `skills/_shared/tracks/` (`track-layer.spec`) and a bounded elicitation contract (`bounded-elicitation.adr`). `cli-owns-layers-4-5.adr` moved the hook policy into the CLI: from CLI v0.7.0 and plugin 0.7.0 the plugin ships three thin launchers (`session-start`, `pre-tool-use`, `post-tool-use`) and the CLI executes the write guard, code-alignment injection, validation, cascade, precision, staleness, and the ranked session recap. `/archcore:context` and `/archcore:help` were removed (`remove-context-command.adr`); rejected documents left agent steering.

## Examples

Command-surface trajectory, by count of user-facing skills: 34 → 18 → 16 → 7 → 4. Hook-entry trajectory, per host config: 6 shell entries → 3 launcher entries backed by the CLI.