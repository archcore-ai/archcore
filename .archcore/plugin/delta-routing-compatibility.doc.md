---
title: "Delta-Routing Compatibility — Version-Skew Risk Register for Shipped Plugin and CLI"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Overview

This register covers version-skew risk when delta routing ships: repositories where an older plugin release or an older `archcore` CLI coexists with the new conductor — a teammate on the previous plugin release, a host adapter awaiting reinstall, a CI machine with a pinned CLI. Baseline facts: routing logic ships in skill files, byte-identical across hosts within one release (host-adapter-contract invariant); the CLI PostToolUse hook never blocks a write, and a CLI predating a check reports nothing for it; a hook launcher below CLI 0.7.0 exits 0 without output.

## Content

| # | Surface | Shipped-version behavior | Risk under delta routing | Containment |
|---|---|---|---|---|
| 1 | MCP status enum | `create_document`, `update_document`, `list_documents`, `search_documents` schemas enum `draft`/`accepted`/`rejected` | an `archived` value is refused at the tool boundary by every shipped CLI — discharge cannot ship plugin-side first | the `archived` rfc lands in the CLI first; the plugin gates discharge behind a `cli-gte`-style version probe; until then discharge reports only |
| 2 | Plugin version skew in one team | the fixed routing table runs from the installed release's skill files | an old-release teammate routes the same corpus through the fixed cascade — a null-route task still yields `idea` + `prd`; duplication, not corruption | corpus-compatibility invariant: the conductor emits only shipped types, statuses, and relation types; old tracks' `skip_when` closes gates on conductor-produced docs, bounding duplication |
| 3 | In-flight draft state blocks | resume rules re-enter at the earliest unmet gate; an unknown stage falls back to entry-condition evaluation | an old release resuming a conductor-written draft ignores the `route:` and `delta:` fields and degrades the computed route to the fixed cascade | phase 1 keeps `gate:` values within the existing `<track>.<stage>` names; in the other direction the conductor recomputes the route on resume |
| 4 | Expert aliases | the `plan` argument-hint names `sdd`, `sources`, `iso`, `research`; host command descriptions and user habits reference them | removing an alias breaks recorded invocations and adapter command descriptions | aliases stay valid and map to computed-era paths (conductor spec constraint) |
| 5 | Older CLI hook validation | the PostToolUse leaf reports findings per its own version and always exits 0 | conductor-produced documents get no new findings on an old CLI; enforcement stays prompt-side, as today | behavioral routing tests live plugin-side (bench-derived), independent of CLI version |
| 6 | CLI below 0.7.0 | hook launchers exit 0 without output | no new risk — validation is absent today on those installs | none needed |
| 7 | Planned status-transition guard | closeout confirms each transition in chat; the server sees only the status write | a strict server-side guard would refuse legitimate accepts from old plugin releases that pass no confirmation payload | advisory-then-enforce rollout, version-gated the same way as row 1 |
| 8 | Sync manifest growth | 750 relations today; umbrella routes add edges per capability | branch-merge conflicts on the manifest amplify — the shared-mutable-file failure mode the enforcement audit records | raise `cli-path-index.rfc` priority; no manifest format change rides with delta routing itself |

Held invariants that keep the skew safe: the 19 document types are unchanged, the relation type set is unchanged, and no file format changes — an old release reads every conductor-produced document as an ordinary document of its type.

## Examples

- Mixed team: dev A (new release) plans "CSV export" → capability route, `spec` + `plan`. Dev B (old release) later runs `/archcore:plan` on the same topic: `sdd.design` and `sdd.decompose` close through `skip_when` on A's documents; `sdd.frame` still opens and adds an `idea`, and `sdd.require` at worst a compressed `prd` — row 2's bounded duplication, no corruption.