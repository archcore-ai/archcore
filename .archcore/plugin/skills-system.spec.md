---
title: "Skills System Specification — Seven Auto-Invocable Intent Skills"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Purpose

Define the contract for how skills are structured, discovered, and used within the Archcore Plugin (Claude Code, Cursor, Codex CLI). Skills are organized as a single tier of auto-invocable intent skills. Per-type elicitation and per-flow orchestration live inline within these intents — there are no per-document-type skills and no separate track tier.

The current surface is governed by `skill-surface-collapse.adr.md`, which supersedes the track tier from `intent-based-skill-architecture.adr.md`, the standalone `actualize`/`review` split from `merge-review-status-remove-graph.adr.md`, and the mainstream/niche type-skill stratification from `inverted-invocation-policy.adr.md`.

## Scope

This specification covers all skill files in the `skills/` directory: **7 intent skills** plus the shared runtime asset directory `skills/_shared/`. It defines naming convention, content structure, invocation triggers, relationship to MCP tools, and the per-flow reference asset structure used by `plan` and `audit`. It does not cover agents (subagents).

## Authority

This specification is the authoritative reference for skill files in the plugin. The Skill File Structure Standard (rule) derives from this specification. The Plugin Architecture spec defines how skills interact with other components. The current invocation policy is "every skill auto-invocable; no `disable-model-invocation` flags" per `skill-surface-collapse.adr.md`.

## Subject

### Skills (7) — Auto-Invocable Intent Skills

The visible `/` palette is exactly 7 commands. Each skill maps to a clearly distinct user intent — no two skills anti-trigger each other. All 7 are auto-invocable; the model picks them up from user phrasing.

| Directory | Skill | User intent | Modes / continuations |
|---|---|---|---|
| `skills/init/` | init | Seed an empty `.archcore/` on first install — scale-detect (small/medium/large), seed stack rule + run guide + scale-appropriate extras | three-step accept/edit/skip flow; idempotent |
| `skills/capture/` | capture | Document a module/component/system | routes to adr / spec / doc / guide |
| `skills/decide/` | decide | Record a decision (ADR) or draft a proposal (RFC); optional standard cascade | ADR → optional CPAT (for code-pattern changes) → optional rule → optional guide |
| `skills/plan/` | plan | Plan a feature or initiative end-to-end | routes to single plan, or one of the multi-doc flows via references: product (idea→prd→plan), sources (mrd→brd→urd), iso (brs→strs→syrs→srs), feature (prd→spec→plan→task-type) |
| `skills/audit/` | audit | Documentation health and drift | three modes: default short dashboard, `--deep` coverage audit, `--drift` code/cascade/temporal staleness |
| `skills/context/` | context | Surface rules/decisions for a code area or pickup | search_documents-backed grouped markdown |
| `skills/help/` | help | Navigate the system | command catalogue, onboarding cues |

### Shared Runtime Assets (`skills/_shared/`)

Plain-markdown assets loaded at runtime by skills before composing documents. They ship with the plugin; skill instructions reference plugin-internal paths only (never the consumer's `.archcore/`).

| Asset | Path | Loaded by | Purpose |
|---|---|---|---|
| `precision-rules.md` | `skills/_shared/precision-rules.md` | `decide`, `capture` | Forbidden vagueness lexicon, imperative phrasing, `[assumption]` marker conventions |
| `adr-contract.md` | `skills/_shared/adr-contract.md` | `decide`, `capture` (ADR) | Mandatory sections + bad/good examples for ADR content per MADR 4.0 |

Companion ADR: `precision-over-coverage.adr.md` documents the design rationale.

### Per-Flow References (`skills/plan/references/` and `skills/audit/lib/`)

Where a skill needs to support multiple multi-document flows, the per-flow content lives in markdown references that the SKILL.md loads on demand:

| Reference | Path | Used by | Content |
|---|---|---|---|
| `product-flow.md` | `skills/plan/references/product-flow.md` | `plan` | idea → prd → plan cascade |
| `sources-flow.md` | `skills/plan/references/sources-flow.md` | `plan` | mrd → brd → urd cascade |
| `iso-flow.md` | `skills/plan/references/iso-flow.md` | `plan` | brs → strs → syrs → srs cascade |
| `feature-flow.md` | `skills/plan/references/feature-flow.md` | `plan` | prd → spec → plan → task-type cascade |
| `continuations.md` | `skills/decide/references/continuations.md` | `decide` | ADR → CPAT → rule → guide continuation logic |
| `drift-detection.md` | `skills/audit/lib/drift-detection.md` | `audit` (drift mode) | code-drift, cascade, temporal staleness protocols |

This pattern keeps each SKILL.md under the 200-line budget while preserving rich per-flow elicitation behind a single intent entry point.

### Document-type coverage

Every Archcore document type is reachable through an intent skill or directly via MCP. There are no per-type skills.

| Type | Reached via |
|---|---|
| adr | `/archcore:decide`, `/archcore:capture` |
| rfc | `/archcore:decide` (open-proposal branch) |
| rule | `/archcore:decide` (continuation), `/archcore:capture` (when codifying observed practice) |
| guide | `/archcore:capture`, `/archcore:decide` (continuation) |
| doc | `/archcore:capture` |
| spec | `/archcore:capture` |
| prd | `/archcore:plan` (product or feature flow) |
| idea | `/archcore:plan` (product flow) |
| plan | `/archcore:plan` (any flow) |
| task-type | `/archcore:plan` (feature flow) |
| cpat | `/archcore:decide` (continuation, optional) |
| mrd, brd, urd | `/archcore:plan` (sources flow) |
| brs, strs, syrs, srs | `/archcore:plan` (iso flow) |

For any type, `mcp__archcore__create_document(type=<any>)` remains a direct path that bypasses skill mediation entirely.

## Contract Surface

### File Location

Each skill resides at `skills/<name>/SKILL.md` where `<name>` is one of: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.

### SKILL.md Frontmatter

Every skill is auto-invocable. No `disable-model-invocation` flag.

```yaml
---
name: <skill-name>
argument-hint: "[topic, mode, or description]"
description: <What this skill does. Activate when X. Do NOT activate for Y (use /archcore:other).>
---
```

### Skill Content Structure

Every skill file MUST contain these sections in order:

1. **Title and one-liner** — What this skill does, in user terms.
2. **When to Use** — Natural-language signals that lead to this skill. Contrast with adjacent skills.
3. **Routing Table** — Explicit decision tree mapping user input or arguments to document types, flows, or analysis modes. Each branch terminates in a named outcome. Maximum one clarifying question when input is ambiguous between two paths. Flow-style skills (`init`) may replace the routing table with a numbered step sequence as long as each step has a deterministic set of branches.
4. **Execution** — Step-by-step flow:
   - Step 1: Gather data (list_documents, list_relations, git log as needed)
   - Step 2: Scope confirmation (one `AskUserQuestion` if `$ARGUMENTS` is ambiguous)
   - Steps 3–N: Core execution (document creation, analysis, or reporting). Creation steps include per-type elicitation inline: question → compose sections → create_document → add_relation. Flow-style execution may load a reference under `skills/<name>/references/` (or `skills/audit/lib/`) for the chosen flow.
   - Final step: Summary and suggested next actions
5. **Result** — Summary of what was created or found, and recommended next actions.

Note: creation-oriented skills (`init`, `capture`, `decide`, `plan`) include inline creation recipes for the document types they can produce. Analysis-oriented skills (`audit`, `context`) include analysis logic. The `audit` skill has three output modes (short dashboard / `--deep` / `--drift`) inside one skill. The `help` skill includes the command catalogue.

## Normative Behavior

- All skills MUST be auto-invocable. No skill carries `disable-model-invocation`.
- Skills MUST contain explicit routing tables with bounded decision branches (flow-style skills like `init` may substitute a numbered step sequence with deterministic branches per step).
- Skill descriptions MUST enumerate triggers and anti-triggers using the "Activate when X. Do NOT activate for Y." format.
- Skills MUST default to minimum viable path. Expansion requires a binary scope question.
- Creation-oriented skills MUST be self-contained with inline creation recipes (question + sections + create + relate per document type produced). Where a flow has multiple steps, per-flow content MAY live in `skills/<name>/references/<flow>.md` and be loaded on demand.
- Analysis skills (`audit`, `context`) MUST use MCP read tools (`list_documents`, `get_document`, `list_relations`) and MAY use git/Grep/Glob for cross-referencing.
- All skills MUST use MCP tools for document operations. MUST NOT instruct direct Write/Edit to `.archcore/*.md`.
- Skills MUST reference MCP tools by exact name.
- Skills provide guidance around the template, not the template itself.

## Constraints

- Skill files must not exceed 300 lines.
- Skill files must not include code blocks longer than 20 lines.
- Per-flow reference files (under `skills/<name>/references/` or `skills/audit/lib/`) must not exceed 200 lines each.
- Skills must not reference internal CLI implementation details — only the MCP tool interface.
- Skills must not embed full document templates.

## Invariants

- There are exactly 7 skills on disk: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.
- Total skills visible in `/`: 7. No hidden skills.
- No skill has `disable-model-invocation: true`.
- Every skill has a routing table section or a numbered step sequence with deterministic branches.
- Every creation skill references `create_document` in its workflow.
- Every analysis skill references `list_documents` and `list_relations` in its workflow.
- No skill instructs direct Write/Edit to `.archcore/` files.
- Every Archcore document type has at least one intent path that creates it.

## Error Handling

- If MCP tools are unavailable, skills inform the user with install/init instructions.
- If `create_document` fails (duplicate filename), skills suggest an alternative filename.
- If intent routing is ambiguous after one scope question, default to the most general path.
- If a multi-step flow detects existing documents mid-flow, skip already-created documents and resume.
- If git is unavailable for `audit --drift`, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A skill file conforms to this specification if:

1. It resides at the correct path (`skills/<name>/SKILL.md`) and `<name>` is one of the 7 canonical skill names.
2. It has valid frontmatter with `name` and `description` fields and no `disable-model-invocation` flag.
3. It contains all 5 required sections (title, when-to-use, routing-table-or-step-sequence, execution, result).
4. It references appropriate MCP tools in its workflow.
5. It stays within its line limit (300 SKILL.md, 200 references).
6. It does not embed full template content.
7. Its description follows the "Activate when X. Do NOT activate for Y." format.
