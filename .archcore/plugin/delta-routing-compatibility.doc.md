---
title: "Delta-Routing Compatibility — Version-Skew Risk Register for Shipped Plugin and CLI"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
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
| 2 | Plugin version skew in one team | the fixed routing table runs from the installed release's skill files | an old-release teammate routes the same corpus through the fixed cascade; duplication, not corruption | the conductor emits only shipped types, statuses, and relation types; old tracks' `skip_when` closes gates on conductor-produced docs, bounding duplication |
| 3 | In-flight draft state blocks | an unknown stage falls back to entry-condition evaluation | an old release resuming a conductor-written draft ignores the `route:` and `delta:` fields | the conductor reads a legacy `delta:` field as the claim ledger, ignores `route:`, and writes `claims:` at the next gate close |
| 5 | Older CLI hook validation | the PostToolUse leaf reports findings per its own version and always exits 0 | conductor-produced documents get no new findings on an old CLI; enforcement stays prompt-side, as today | behavioral routing tests live plugin-side (bench-derived), independent of CLI version |
| 6 | CLI below 0.7.0 | hook launchers exit 0 without output | no new risk — validation is absent today on those installs | none needed |
| 7 | Planned status-transition guard | closeout confirms each transition in chat; the server sees only the status write | a strict server-side guard would refuse legitimate accepts from old plugin releases that pass no confirmation payload | advisory-then-enforce rollout, version-gated the same way as row 1 |
| 8 | Sync manifest growth | 750 relations today; umbrella routes add edges per capability | branch-merge conflicts on the manifest amplify — the shared-mutable-file failure mode the enforcement audit records | no manifest format change rides with delta routing itself; the mitigation for edge growth is not designed here |
| 9 | Retired mode words | only `init` reads its first word as a mode (`import`, `refresh`); older releases read `plan`, `document`, and `review` modes | a recorded invocation with a retired mode word becomes topic text | context resolution reads it as wording, so it degrades to one classifying question, not to a wrong write; the release notes name every retired form |

Historical delta-routing baseline: that release preserved 19 types and four relation values. The research vocabulary changes this assumption: 21 types and seven relations require a supporting CLI. The runtime gate for that vocabulary is @plugin/plugins/archcore/skills/_shared/research-compatibility.md; its minimum is 0.8.3, confirmed against the published [CLI v0.8.3](https://github.com/archcore-ai/cli/releases/tag/v0.8.3).

| Research surface | Current containment |
|---|---|
| New type filters and writes | Probe before the first affected MCP call; only `yes` enables new enums. |
| Old or unrecognized CLI | New investigations use `rnd` and legacy relations; one supplied external material exits without writes. Existing new-type artifacts are never converted. |
| CLI/MCP version mismatch | A rejected new enum stops the affected operation; no retry under a substituted type. |
| Shared manifest | Older binaries reject new relation values. Upgrade all readers before adding them; the local probe cannot verify teammates. |
| Resume | Old `rnd` artifacts retain their type; optional `artifact_type` falls back to the filename. |
| Partial evidence write | Pending state and read-before-retry preserve evidence after edge failure; calls are not atomic. |

The engine release handoff is complete: [CLI v0.8.3](https://github.com/archcore-ai/cli/releases/tag/v0.8.3) ships the vocabulary. The downloaded Darwin arm64 archive matched the GitHub SHA-256 digest and release checksums on 2026-09-07. The native stdio MCP probe passed. No downgrade conversion is supplied.

The actor-subject vocabulary (`scenario`, `journey`) raises the registry to 23 types and adds no relation value. Its runtime gate is @plugin/plugins/archcore/skills/_shared/actor-subject-compatibility.md; its minimum is 0.8.4, published as [CLI v0.8.4](https://github.com/archcore-ai/cli/releases/tag/v0.8.4) on 2026-09-16 with checksums for six platform assets. The plugin's CI pins that release by digest.

| Actor-subject surface | Current containment |
|---|---|
| New type filters and writes | Probe before the first MCP call naming either type; only `yes` enables the names, the `sdd.illustrate` gate, and a `scenario` named in the subject of a `document` request that describes code. |
| Old or unrecognized CLI | A `document` request that describes code and whose subject names `scenario` reports the required version and writes nothing; the illustrate instrument is dropped from the package with one report. Existing artifacts of either type are never converted. |
| Shared corpus, older reader | No relation value changes, so old readers keep reading; they skip `.scenario.md` and `.journey.md` files in the scan and report an invalid type in `status`. |
| Two compatibility files | Research at 0.8.3 and actor-subject at 0.8.4 run independently; a request engaging both runs both probes. |
| Restart | The PATH probe identifies the binary a new server would run; a server started before an upgrade keeps the old engine until restarted. Observed on 2026-09-16: an MCP server started on 0.8.3 rejected `scenario` while `archcore --version` on PATH reported 0.8.4. |

### Audit of 2026-09-07 — old readers against a new-vocabulary corpus

The audit ran CLI 0.8.3 next to 0.8.2, 0.7.3, and 0.6.7 built from their release tags, in isolated macOS arm64 projects, driving the MCP tools and the hook launchers through shell scripts. Every row below is a reproduced observation from that run; live host sessions (Cursor UI, Claude Code, Codex, Copilot) were not exercised, and nothing here describes them.

| Observation | Evidence and limit |
|---|---|
| One new relation blocks old readers | A single `supports` edge made CLI 0.8.2, 0.7.3, and 0.6.7 reject `get_document`, `search_documents`, and `list_relations` — including reads of unrelated ADRs. Read failure is corpus-wide, not per document. |
| Read failure does not disable all writes | The same old CLIs still created and updated legacy-type documents while their reads failed. A teammate on an old binary can keep writing into a corpus they can no longer read. |
| Removal can partially complete | Old `remove_document` deleted the file before reporting the manifest failure; the relation entry stayed behind. The remove call is not atomic across file and manifest on those versions. |
| Old writers drop custom metadata | A title-only `update_document` on the three old CLIs removed custom YAML fields; CLI 0.8.3 preserved them. |
| New types alone degrade discovery | Old CLIs classified `research` as knowledge and omitted it from the vision selection; the document stayed readable. |
| Restart matters | The PATH probe of `research-compatibility.md` identifies the binary a new server would run, not the server already running; an MCP session started before an upgrade keeps the old engine until restarted. |
| Rollout order | Upgrade every shared-corpus reader and writer before introducing a new relation value; the local probe cannot verify teammates. |

### Cursor post-write advisories

The Cursor `afterMCPExecution` payload names the tool bare (`update_document`), and the CLI folds only qualified spellings. Before the plugin-side fix, no Cursor document write received validation, cascade, or precision findings, and nothing marked the session as unprotected. The fix and its payload shapes are specified in `hooks-validation-system.spec`. A user who renames the `archcore` key in Cursor's `mcp.json` gets no translation and no advisory, the same silence as before the fix. Live Cursor sessions were not exercised; the Cursor fixtures are composed from the hooks reference, not captured.

## Examples

- Mixed team: dev A (new release) plans "CSV export" → capability route, `spec` + `plan`. Dev B (old release) later runs `/archcore:plan` on the same topic: `sdd.design` and `sdd.decompose` close through `skip_when` on A's documents; `sdd.frame` still opens and adds an `idea`, and `sdd.require` at worst a compressed `prd` — row 2's bounded duplication, no corruption.
