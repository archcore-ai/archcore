---
title: "Plan: Forward-Compatible Config Parsing"
status: accepted
tags:
  - "cli"
  - "config"
  - "mcp"
---

## Goal

Make `.archcore/settings.json` parsing forward-compatible: an older `archcore` binary must not
crash and must not lose functionality when it meets a config field it does not know, added by a
newer CLI. The property is field-agnostic — it applies to every unknown key uniformly, with no
tie to a specific feature.

## Status

Revised 2026-07-24: implemented, except one optional item.

- Implementation: `@internal/config/config.go` (`Settings.Extra`, `knownFields`, `UnmarshalJSON`,
  `MarshalJSON`, `UnknownFieldNames`), plus entry-point warnings in `@cmd/config_warn.go`,
  `@cmd/mcp.go`, `@cmd/config.go`, `@cmd/doctor.go`, and `@cmd/sync.go`.
- The release-sequencing note landed in the release guide on 2026-07-24.
- Open item: the optional `ARCHCORE_AUTO_UPDATE` is not started. Low priority, deferred deliberately.

## Problem

`Settings.UnmarshalJSON` in `@internal/config/config.go` used to reject every unknown field with
`field %q is not allowed for sync type …`. Any new config field therefore broke every config-loading
path on an older CLI: `mcp`, `hooks`, `doctor`, and `status`. Already released strict binaries
cannot be taught otherwise, so the only fix is a tolerant parser shipped from now on, as early
as possible.

## Decision

### A. Soft-ignore unknown fields — three properties, applied uniformly

1. Read-tolerant. An unknown key is captured into `Settings.Extra` and is not an error. The
   field-check loop distinguishes three cases: allowed for the mode — decode it; known but not
   for this mode — keep the existing hard error; unknown to this binary — put it in `Extra`.
2. Keep-serving. After the ignore, the config finishes loading and the known fields populate as usual.
3. Write-preserving. WHEN the CLI writes `settings.json` through `config set`, it preserves the
   unknown keys: the merge-tail in `MarshalJSON` produces byte-identical output while `Extra` is
   empty, and merges with the raw map when it is not. Without this, a tolerant but older CLI would
   silently erase the field.

The warning prints to stderr on user-facing commands only — `mcp` startup, `config`, `doctor`,
and `sync`. `UnmarshalJSON` and `Load` stay silent because they are hot paths. Value validation
for known fields stays strict.

Trade-off: protection against a typo in a field name is lost. This is a deliberate reversal of
the strictness recorded in the related ADR on backing up invalid configs, taken for forward
compatibility.

### B. Release sequencing

The tolerant parser from part A must ship no later than the release that introduces a new config
field, and preferably earlier. It does not repair already released strict binaries: those keep
rejecting the new config, and for them the mitigation sits on the consumer side as a nudge. The
tolerance therefore ships as early as possible, as a separate step.

### C. Optional: ARCHCORE_AUTO_UPDATE, opt-in only

`archcore update` already performs a self-replace. Auto-running it from any hook path is rejected
as a default: it replaces the global binary without asking, does not restart an already running
MCP server (so it only helps the next session), fails in CI, offline, and on a locked file, and
breaks pinned versions. It is acceptable only under an explicit `ARCHCORE_AUTO_UPDATE=1`.

## Tasks

- [x] `config.UnmarshalJSON`: an unknown field is captured into `Extra` instead of raising an error
- [x] `config.UnmarshalJSON`: a known-but-wrong-mode field still errors, and value validation for known fields stays strict
- [x] `Settings.MarshalJSON`: round-trip preservation of unknown fields (merge-tail; byte-identical while `Extra` is empty)
- [x] Entry-point warning on stderr (`@cmd/config_warn.go`; `mcp`, `config`, `doctor`, `sync`)
- [x] Test: the parser accepts a config with an unknown field, populates the known fields, and keeps serving
- [x] Test: `config set <known-field>` does not drop an unrecognized field (round-trip, end-to-end)
- [x] Test (regression): a known-wrong-mode field and a malformed value of a known field still error
- [x] Test: MCP startup (`checkGlobals`) and the in-process server tolerate an unknown field
- [x] Release process: ship the tolerant parser no later than the new config field — recorded in the release guide, section "Sequencing a release that adds a settings.json field" (2026-07-24)
- [ ] Optional: `ARCHCORE_AUTO_UPDATE=1` opt-in self-heal, off by default — deferred, low priority

## Acceptance Criteria

- A tolerant CLI on a config with unknown fields: zero errors, known fields work, and every
  config-loading command (`mcp`, `hooks`, `doctor`, `status`) runs. ✅
- `config set` on any field leaves unrecognized fields in `settings.json` untouched. ✅
- Known fields stay strictly validated: a malformed `project_id` errors. ✅

## Dependencies

- Sequencing: the release with the tolerant parser ships no later than the release with a new config field.
- Touches `@internal/config/config.go` (`UnmarshalJSON`, `MarshalJSON`, `allowedFields`,
  `knownFields`) and the release process described in the related guide.
