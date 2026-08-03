---
title: "Multi-Host Compatibility Layer Specification"
status: rejected
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Purpose & Scope

This spec is **rejected and superseded**. It described the bundled CLI launcher architecture, and no part of it is normative. It is retained as a tombstone so that a reader who arrives from an old reference learns where the current contract lives instead of re-deriving a removed design.

Superseded by `remove-bundled-launcher-global-cli.idea`, which replaced the launcher with a globally installed `archcore` on PATH.

## Surface

The removed design covered: POSIX, Windows CMD, and PowerShell launchers; the resolution order `$ARCHCORE_BIN` → PATH → plugin cache → GitHub download; checksum verification; the `bin/CLI_VERSION` pin; host-specific MCP wiring through `.mcp.json` and `.codex.mcp.json` pointing at the launcher; CWD rebase for Codex; and the `ARCHCORE_BIN` / `ARCHCORE_SKIP_DOWNLOAD` / `ARCHCORE_HIDE_EMPTY_NUDGE` environment contract.

Plugin v0.4.0 removed every one of those mechanics. The current shape is described by `multi-host-plugin-architecture.adr` for the shared-core and per-host-adapter split, which still governs; by `component-registry.doc` for the current `bin/`, hooks, MCP, and manifest layout; by `plugin-development.guide` for the current prerequisites and MCP wiring; and by `codex-local-plugin-testing.guide` for the current Codex packaging contract.

## Normative Behavior

This document states no requirement. A reader looking for a binding requirement about host wiring MUST consult `host-adapter-contract.spec`.

## Constraints & Invariants

- Invariant: the body of this document stays a tombstone. The launcher-era contract is recoverable from git history, which is its source of truth.

## Failure Behavior

- IF a document or a script still references the launcher mechanics described above, THEN the author MUST treat that reference as stale and repoint it at `host-adapter-contract.spec`.

## Conformance

Nothing conforms to this document. It is retained for navigation only.
