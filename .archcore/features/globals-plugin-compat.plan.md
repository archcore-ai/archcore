---
title: "Plan: Plugin Compatibility for the Globals Rollout (Plugin)"
status: accepted
tags:
  - "cli"
  - "globals"
  - "integrations"
  - "mcp"
---

## Goal

Keep the plugin compatible with every CLI version, old and new, so that it never creates the
dangerous "old CLI with globals" cell, and so that it nudges users who have not updated the CLI.

The plugin code lives in the separate `archcore/plugin` repository (`plugins/archcore`). The
context is problem 7 of the umbrella prototype plan, linked in the relation graph.

## Status

Revised 2026-07-24: partially implemented — 3 of 5 tasks done.

The open tasks — the compatibility advisory in `bin/session-start`, and `--exclude-dir` in
`bin/check-staleness` and `bin/check-code-alignment` — live in the separate `archcore/plugin`
repository and cannot be verified from this repository. This plan is coordination-only: the
checkboxes close when the plugin work merges.

## Decision

### A. Plugin-agnosticism invariant — four rules

1. The plugin never writes `globals` into `settings.json`, including from `/archcore:init`.
2. The executable path uses stable CLI verbs only (`mcp`, `hooks`, `doctor`, `--version`). A new
   command or flag appears in hint text only.
3. The new MCP fields (`source_kind`, `read_only`, `source_id`) are optional. In `bin/`, the
   executable code, logic does not branch on their presence, because an older CLI does not send
   them. Skills MAY read them and surface globals, but every clause is data-gated: when the fields
   are absent, behavior is identical to the path without globals.
4. A version mismatch produces a nudge, never a hard block.

Revision 2026-06-17: the plugin understands globals. The invariant is no longer "zero mentions of
globals in `skills/`" but the property set above: `bin/` stays agnostic; skills read the optional
fields only behind a data gate, where absent means the same behavior as without globals; and the
plugin never writes `globals`. The property-based check `test/structure/cli-compat-invariant.bats`
in the `archcore/plugin` repository guards it.

Reason for the reversal: the fields are live on all three read tools, per §4.1 of the related
global sources contract, and the plugin is their intended consumer. The blunt `grep=0` proxy
blocked capability without adding real protection.

### B. Compatibility advisory in bin/session-start

Local detection with no network: `grep -q '"globals"' .archcore/settings.json`, plus the fact that
`archcore hooks … session-start` failed, or that `archcore --version` is below the minimum. On a
mismatch, the plugin does not propagate a cryptic crash; it prints "update the archcore CLI:
`archcore update`" and lets the session continue. The advisory fires once, rate-limited through a
stamp file, following the `check-staleness` pattern. The mechanism is version-general, not
globals-specific.

### C. Grep hygiene for filesystem-scanning scripts

`bin/check-staleness` (`grep -rl … .archcore/`) and `bin/check-code-alignment`
(`grep -rlF … .archcore --include='*.md'`) walk the whole `.archcore/` tree, including an in-tree
`.archcore/global/…`, and emit nudges about read-only globals. Add `--exclude-dir=global`, or skip
the declared globals, or both. The change is soft and blocks nothing, but it removes the noise.

## Tasks

- [x] Record the agnosticism invariant as the property-based check `cli-compat-invariant.bats` in
  the `archcore/plugin` repository: `bin/` at hard zero on the new fields, skills data-gated, and
  `globals` never written
- [ ] `bin/session-start`: compatibility advisory — local detection plus a rate-limited nudge to update the CLI
- [ ] `bin/check-staleness`: skip `.archcore/global` and the declared globals
- [ ] `bin/check-code-alignment`: skip `.archcore/global` and the declared globals
- [x] Review: no skill and no file under `bin/` writes `globals`; `bin/` does not branch on
  `source_kind` or `read_only`; skills read them only behind an absent-guard (`skills/_shared/globals.md`
  plus the context, audit, capture, decide, and plan clauses)

## Acceptance Criteria

- A new plugin on an older CLI without globals behaves as before.
- On the "old CLI with globals in the config" combination, the user sees one clear nudge instead of a cryptic crash.
- `check-staleness` and `check-code-alignment` emit no nudges about in-tree globals.
- `bin/` contains no `source_kind`, `read_only`, or `source_id`, and never writes `globals`.
- Every skill that reads those fields carries an absent-default guard, so an older CLI without the fields behaves as before.

## Dependencies

- The nudge points at a CLI update, so it depends on the related CLI plan for forward-compatible config parsing.
- The globals decision: the related ADR on declaring global sources in `settings.json`.
- The code lives in the separate `archcore/plugin` repository; this plan is coordination-only.
