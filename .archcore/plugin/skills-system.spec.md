---
title: "Skills System Specification — Seven Auto-Invocable Intent Skills"
status: rejected
tags:
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines how skills are structured, discovered, and used inside the Archcore plugin on Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI. Skills form a single tier of auto-invocable intent skills; per-type elicitation and per-flow orchestration live inline inside those intents, with no per-document-type skills and no separate track tier. Normative for every file in `skills/` — the 7 intent skills plus the shared runtime asset directory `skills/_shared/` — covering naming, content structure, invocation triggers, the relationship to MCP tools, and the per-flow reference structure used by `plan`, `audit`, and `init`. Depended on by `skill-file-structure.rule`, which derives from it, and by `plugin-architecture.spec`, which defines how skills interact with other components. Out of scope: agents (subagents), and the external command surface, which `commands-system.spec` governs.

The surface is fixed by `skill-surface-collapse.adr`, which supersedes the track tier of `intent-based-skill-architecture.adr`, the standalone `actualize`/`review` split of `merge-review-status-remove-graph.adr`, and the mainstream/niche stratification of `inverted-invocation-policy.adr`.

## Surface

**Skills (7).** The visible `/` palette holds exactly 7 commands. Each skill maps to a distinct user intent; no two skills anti-trigger each other. All 7 auto-invoke from user phrasing.

| Directory | Skill | User intent | Modes / continuations |
|---|---|---|---|
| `skills/init/` | init | Seed an empty `.archcore/` on first install — scale-detect (small/medium/large), compose a full first-day seed (stack rule, run guide, data-model, integrations, config, entry points, hotspot specs, linked architecture overview) and import agent files | detect → compose → one preview → single `confirm` → create + wire relations; after confirm also installs host wiring — the same files `archcore init` writes for the detected host — via `install_host_config` MCP tool → `archcore init --agent <host> --project <root>` → printed manual command as last resort (`host-wiring-parity.adr`); idempotent (skip-on-exists); `--mode` / `--domain` / `--refresh` (the latter also retrofits host wiring on pre-wiring repos); see `magic-first-day-init.adr` |
| `skills/capture/` | capture | Document a module/component/system | routes to adr / spec / doc / guide |
| `skills/decide/` | decide | Record a decision (ADR) or draft a proposal (RFC); optional standard cascade | ADR → optional CPAT (for code-pattern changes) → optional rule → optional guide |
| `skills/plan/` | plan | Plan a feature or initiative end-to-end | routes to single plan, or one of the multi-doc flows via references: product (idea→prd→plan), sources (mrd→brd→urd), iso (brs→strs→syrs→srs), feature (prd→spec→plan→task-type) |
| `skills/audit/` | audit | Documentation health and drift | three modes: default short dashboard, `--deep` coverage audit, `--drift` code/cascade/temporal staleness |
| `skills/context/` | context | Surface rules/decisions for a code area or pickup | search_documents-backed grouped markdown; `--git-changes` derives the path set from the working tree |
| `skills/help/` | help | Navigate the system | command catalogue, onboarding cues |

**Shared runtime assets (`skills/_shared/`).** Plain-markdown assets loaded at runtime before a skill composes a document. They ship with the plugin, and skill instructions reference plugin-internal paths only — never the consumer's `.archcore/`.

| Asset | Path | Loaded by | Purpose |
|---|---|---|---|
| `precision-rules.md` | `skills/_shared/precision-rules.md` | `decide`, `capture`, `init`, `plan` | Forbidden vagueness lexicon, imperative phrasing, no-cross-document-section rule, `[assumption]` marker conventions |
| `adr-contract.md` | `skills/_shared/adr-contract.md` | `decide`, `capture` (ADR) | Mandatory sections + bad/good examples for ADR content per MADR 4.0 |
| `spec-contract.md` | `skills/_shared/spec-contract.md` | `capture` (spec), `init` (hotspot specs) | Mandatory sections + "when NOT to write a spec" for spec content |
| `rule-contract.md` | `skills/_shared/rule-contract.md` | `decide` (rule), `init` (cross-cutting rules) | Mandatory rule body: RFC 2119 statement, applies-to scope, rationale, Good/Bad examples, enforcement |

**Per-flow references.** Where a skill supports several multi-document flows or heavy detection logic, the per-flow content lives in markdown that `SKILL.md` loads on demand. This keeps each `SKILL.md` inside its line budget while preserving per-flow elicitation behind a single intent entry point.

| Reference | Path | Used by | Content |
|---|---|---|---|
| `product-flow.md` | `skills/plan/references/product-flow.md` | `plan` | idea → prd → plan cascade |
| `sources-flow.md` | `skills/plan/references/sources-flow.md` | `plan` | mrd → brd → urd cascade |
| `iso-flow.md` | `skills/plan/references/iso-flow.md` | `plan` | brs → strs → syrs → srs cascade |
| `feature-flow.md` | `skills/plan/references/feature-flow.md` | `plan` | prd → spec → plan → task-type cascade |
| `continuations.md` | `skills/decide/references/continuations.md` | `decide` | ADR → CPAT → rule → guide continuation logic |
| `drift-detection.md` | `skills/audit/lib/drift-detection.md` | `audit` (drift mode) | code-drift, cascade, temporal staleness protocols |
| `detect-*.md`, `extract-*.md`, `compose-overview.md` | `skills/init/lib/*.md` | `init` | scale/stack/domain/module/entry-point/hotspot/data-model/integration/config detection catalogs + capstone composer, read lazily in the Detect/Compose sub-phases |

**File location and frontmatter.** Each skill resides at `skills/<name>/SKILL.md`, where `<name>` is one of `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.

```yaml
---
name: <skill-name>
argument-hint: "[topic, mode, or description]"
description: <What this skill does. Activate when X. Do NOT activate for Y (use /archcore:other).>
---
```

**Content structure.** Every skill file carries these five sections in order: (1) title and one-liner in user terms; (2) *When to Use* — the natural-language signals that lead here, contrasted with adjacent skills; (3) *Routing Table* — a decision tree mapping input or arguments to types, flows, or analysis modes, each branch terminating in a named outcome, with at most one clarifying question when input is ambiguous between two paths; (4) *Execution* — gather data, confirm scope, run the core steps, summarize; (5) *Result* — what was created or found, and the recommended next actions. A flow-style skill may replace the routing table with a numbered step or phase sequence where each step has a deterministic branch set.

**Host surfacing.** Claude Code, Cursor, and GitHub Copilot CLI surface skills directly in their `/` menus. Codex CLI does not and reaches the same seven through `commands/*.md` wrappers. Copilot loads those wrappers behind the skills, and only because `.plugin/plugin.json` points at `commands/` explicitly — that field has no default path on Copilot.

**Document-type coverage.** Every type is reachable through an intent skill, and `mcp__archcore__create_document(type=<any>)` remains a direct path that bypasses skill mediation.

| Type | Reached via |
|---|---|
| adr | `/archcore:decide`, `/archcore:capture` |
| rfc | `/archcore:decide` (open-proposal branch) |
| rule | `/archcore:decide` (continuation), `/archcore:capture` (when codifying observed practice), `/archcore:init` (stack + cross-cutting rules in the first-day seed) |
| guide | `/archcore:capture`, `/archcore:decide` (continuation), `/archcore:init` (run guide) |
| doc | `/archcore:capture`, `/archcore:init` (data-model, integrations, config, entry points, top-level map, architecture overview) |
| spec | `/archcore:capture`, `/archcore:init` (hotspot specs) |
| prd | `/archcore:plan` (product or feature flow) |
| idea | `/archcore:plan` (product flow) |
| plan | `/archcore:plan` (any flow) |
| task-type | `/archcore:plan` (feature flow) |
| cpat | `/archcore:decide` (continuation, optional) |
| mrd, brd, urd | `/archcore:plan` (sources flow) |
| brs, strs, syrs, srs | `/archcore:plan` (iso flow) |

## Normative Behavior

1. Every skill MUST be auto-invocable.
2. A skill MUST NOT carry `disable-model-invocation`.
3. A skill MUST provide an explicit routing table with bounded decision branches.
4. A flow-style skill MAY substitute a numbered step or phase sequence with deterministic branches per step for the routing table.
5. A skill description MUST enumerate triggers and anti-triggers in the `Activate when X. Do NOT activate for Y.` format.
6. A skill MUST default to the minimum viable path.
7. WHEN a skill needs to expand beyond that path, the skill MUST ask a binary scope question. The `init` skill is exempt: it composes the full seed for the detected scale and gates it behind one preview and confirm.
8. A creation-oriented skill MUST carry an inline creation recipe — question, sections, create, relate — for each document type it produces.
9. WHERE a flow has several steps, the skill MAY hold per-flow content in `skills/<name>/references/<flow>.md` or `skills/<name>/lib/*.md` and load it on demand.
10. WHEN the user confirms the init plan and the installed CLI version is at least v0.6.4, the `init` skill MUST install host wiring for the detected host through the first available path: the `install_host_config` MCP tool, else `archcore init --agent <host> --project <root>`, else a printed ready-to-run manual command.
11. WHILE `init` awaits the user's confirmation, the skill MUST NOT write a host config.
12. IF `bin/detect-host` returns `__UNKNOWN__`, THEN the `init` skill MUST ask the user which host to wire, offering all four. A GitHub Copilot CLI session always lands here — that host sets no environment marker a hook-less helper can read — and on Copilot the wiring is not optional: without it no MCP server exists at all (`copilot-mcp-architecture.adr`).
13. An analysis skill (`audit`, `context`) MUST gather data through `list_documents`, `get_document`, and `list_relations`.
14. An analysis skill MAY use git, Grep, or Glob for cross-referencing.
15. A skill MUST perform document operations through MCP tools.
16. A skill MUST NOT instruct a direct Write or Edit against a `.archcore/` markdown file.
17. A skill MUST name each MCP tool by its exact name.
18. A skill MUST provide guidance around the template rather than the template itself.

## Constraints & Invariants

- Constraint: a `SKILL.md` MUST NOT exceed 300 lines. Named exception: `skills/init/SKILL.md` MAY reach 450 lines, because it is the only skill carrying a multi-phase gated pipeline (detect → compose → preview → confirm → create) plus host wiring, and further extraction into `lib/` would split one ordered flow across files.
- Constraint: a `SKILL.md` MUST NOT include a code block longer than 20 lines.
- Constraint: a per-flow reference file under `skills/<name>/references/`, `skills/audit/lib/`, or `skills/init/lib/` MUST NOT exceed 200 lines.
- Constraint: a skill MUST NOT reference internal CLI implementation detail; only the MCP tool interface is available to it.
- Constraint: a skill MUST NOT embed a full document template.
- Constraint: a skill MUST NOT contain a host-conditional instruction (`host-adapter-contract.spec`); host differences belong in `bin/` and in the per-host configs.
- Invariant: exactly 7 skills exist on disk — `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help` — and all 7 are visible in `/`, with no hidden skill.
- Invariant: no skill carries `disable-model-invocation: true`.
- Invariant: every skill has a routing table or a numbered step sequence with deterministic branches.
- Invariant: every creation skill references `create_document` in its workflow.
- Invariant: every analysis skill references `list_documents` and `list_relations` in its workflow.
- Invariant: no skill instructs a direct Write or Edit against a `.archcore/` file.
- Invariant: every Archcore document type has at least one intent path that creates it.

## Failure Behavior

1. IF the MCP tools are unavailable, THEN the skill MUST inform the user with install and init instructions.
2. IF `create_document` fails on a duplicate filename, THEN the skill MUST suggest an alternative filename.
3. IF intent routing stays ambiguous after one scope question, THEN the skill MUST fall back to the most general path.
4. IF a multi-step flow finds an existing document mid-flow, THEN the skill MUST skip that document and resume. The `init` skill applies this as skip-on-exists across its whole seed in one pass.
5. IF git is unavailable during `audit --drift`, THEN the skill MUST skip code-drift analysis and run the cascade and temporal analyses only.

## Conformance

A skill file is conformant when:

1. It resides at `skills/<name>/SKILL.md` and `<name>` is one of the 7 canonical skill names.
2. Its frontmatter carries `name` and `description` and no `disable-model-invocation` flag.
3. It contains all 5 required sections: title, when-to-use, routing table or step sequence, execution, result.
4. It references the MCP tools its workflow uses, by exact name.
5. It stays inside its line limit — 300 lines for a `SKILL.md`, 450 for `init` under the named exception, 200 for a reference file.
6. It embeds no full template content.
