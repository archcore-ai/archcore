---
title: "Fix Plan: Mounted Global Sources Prototype"
status: accepted
tags:
  - "config"
  - "globals"
  - "integrations"
  - "mcp"
  - "prototype"
---

## Resolution

Superseded by the settings.json-only model. Problems 1 to 5 are resolved. Problem 6
(lifecycle and distribution) and problem 7 (version and plugin compatibility) remain open.

The canonical documents now are the ADR on declaring globals in `settings.json`, the global
sources contract, the rules on declaring globals and on local-over-global precedence, the ADR
on mandatory sources, and the vendoring guide. All are linked in the relation graph.

What changed since this plan was written: the launch-flag MVP (`--project` plus a `global: true`
marker) was built and then removed in favor of declaring globals only in the consumer's
`settings.json`. `path` now points directly at the global's `.archcore` directory (problem 5),
`isGlobalPath` computes real path prefixes in absolute space (problem 1), `annotateSource`
shares the same resolver (problem 2), `GlobalSource` is validated (problem 3), and the
per-entry `required` flag was dropped, so every declared global is mandatory and enforced at
scan time and at MCP startup (problem 4).

## Goal

Fix the engineering problems found during the review of the Mounted Global Sources prototype.
Problems 1 to 4 ship in one PR; problems 5 and 6 need their own decisions.

## Context

The prototype lives in `@internal/mcp/tools/`, `@internal/config/config.go`, and
`@templates/templates.go`. The fixture is `@examples/07-local-overrides-global/`, with the shared
source `@examples/_global_/company-standards/`. The originating RFC is the file
`Mounted Global Sources.pdf` in the project root.

## Problems and fix plan

### Problem 1 — `isGlobalPath` worked only for the `.archcore/global/` convention (HIGH) ✅ resolved

`isGlobalPath` checked for the substring `.archcore/global/` in the path, so it did not fire for
`gs.Path = "../company-repo"`. Implemented: `isGlobalPathAbs(baseDir, relPath, globals)` computes
the real path prefixes from `gs.Path` in absolute space (`common.go`).

### Problem 2 — `annotateSource` and `scanDocuments` used divergent logic (MEDIUM) ✅ resolved

Implemented: `annotateSource` uses the same `resolveGlobalPath` resolver, which removes the divergence.

### Problem 3 — `GlobalSource` was not validated (MEDIUM) ✅ resolved

Implemented: `Settings.Validate()` rejects an empty, malformed, reserved (`local`), or duplicate
`id`, and an empty `path`. The `../` prohibition was dropped deliberately, because cross-project
references are the intended use.

### Problem 4 — `required: true` was a dead field (LOW/MEDIUM) ✅ resolved

Implemented, and later refined: the `required` field was removed, so every global is mandatory. A
missing source is an error in `scanDocuments` at scan time and in `checkGlobals` at MCP server
startup. The related ADR on mandatory globals records the decision.

### Problem 5 — Double nesting in `path` (DESIGN) ✅ resolved

Decided and implemented: `path` points directly at the source's `.archcore` directory, with no
auto-appended segment. Example: `path = "../company-standards/.archcore"`.

### Problem 6 — No lifecycle management (OPERATIONAL) ⏳ deferred

A global has to be cloned and updated by hand. There is no `archcore globals pull`, no lockfile,
no mounted-source reporting in `archcore status`, and no broken-global check in `archcore doctor`.
Status: a separate milestone, not started. The interim manual distribution is described in the
related vendoring guide.

### Problem 7 — Version and plugin compatibility for globals (HIGH) 🔧 in progress

`globals` lives in `settings.json`, and `Settings.UnmarshalJSON` used to reject unknown fields, so
already released older CLIs hit `field "globals" is not allowed` and broke every config-loading
path: `mcp`, `hooks`, `doctor`, and `status`. This hit users who received `globals` in the config
but had not updated the CLI.

The asymmetry: a new CLI with an older plugin is safe, because the plugin is globals-agnostic and
picks the feature up transparently. A new plugin with an older CLI breaks only in the "old CLI with
`globals` in the config" cell, and the cause is the CLI and the config, not the plugin version.

The work is split into two execution plans, both linked to this document: the CLI plan for
forward-compatible config parsing (soft-ignore plus release sequencing), and the plugin plan
(agnosticism invariant, compatibility advisory, grep hygiene).

## Tasks

- [x] Refactor `isGlobalPath` to take `[]GlobalSource` and compute the real prefixes
- [x] Unify `annotateSource` with the same resolver — problem 2
- [x] Validate `GlobalSource` in `Settings.Validate()` — problem 3
- [x] Decide on the `required` field: removed, every global is mandatory — problem 4
- [x] Decide the design question in problem 5: `path` points at the `.archcore` directory
- [ ] Lifecycle management (`archcore globals pull` and related) — separate milestone, problem 6
- [ ] Version and plugin compatibility — two linked plans, problem 7

## Acceptance Criteria

- Problems 1 to 5 are closed in code and covered by tests. ✅
- Problem 6 stays open and is addressed by a separate milestone.
- Problem 7 stays open and is addressed by the two linked plans.

## Dependencies

- The settings.json declaration model — the related ADR.
- Mandatory status for every declared source — the related ADR.
- The CLI and plugin compatibility execution plans — the related plans.
