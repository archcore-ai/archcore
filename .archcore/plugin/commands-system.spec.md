---
title: "User-Invoked Skills — Seven-Command Surface Specification"
status: accepted
tags:
  - "commands"
  - "plugin"
---

## Purpose

Define the contract for user-invoked skills: their discoverability, naming, argument handling, and behavior when users invoke them via slash commands. The current surface is **7 commands**, all auto-invocable, governed by `skill-surface-collapse.adr.md`. The prior tiered structure (intent/track/utility) is superseded — every remaining skill is intent-class.

Note: Claude Code and Cursor surface user-invoked workflows directly from skills. Codex CLI requires `commands/*.md` wrappers — thin host-adapter shims that delegate to `skills/<name>/SKILL.md`. The skill file remains the single behavioral source of truth.

## Scope

This specification covers the user-invoked surface of the plugin: how users discover, invoke, and interact with the 7 skills that appear in the `/` menu. It does not cover MCP tools.

## Authority

This specification is the authoritative reference for user-invoked skill behavior. `skills-system.spec.md` defines internal skill structure; this spec defines the external-facing contract.

## Subject

### Visible `/` palette (7 commands)

```
┌──────────────────────────────────────────────────────┐
│  /archcore: PALETTE — 7 auto-invocable skills        │
│                                                      │
│  /archcore:init       "set up archcore"              │
│  /archcore:capture    "document this"                │
│  /archcore:decide     "record this decision"         │
│  /archcore:plan       "plan this feature"            │
│  /archcore:audit      "show status / find drift"     │
│  /archcore:context    "what rules apply here?"       │
│  /archcore:help       "what can I do?"               │
└──────────────────────────────────────────────────────┘
```

Total visible: **7 commands**. Total skills on disk: **7**. No hidden surface, no utility-only flag.

### Command reference

All commands are auto-invocable. The user describes intent in natural language and the model routes; an explicit `/archcore:<name>` invocation works identically.

| Command | Description (in skill picker) | Argument | Behavior |
|---|---|---|---|
| `/archcore:init` | First-time onboarding — seed `.archcore/` with stack rule + run guide + scale-appropriate extras | — | Three-step flow, each step accept/edit/skip. Idempotent. |
| `/archcore:capture` | Document a module / component / system | `[topic]` | Routes to adr / spec / doc / guide based on context |
| `/archcore:decide` | Record a decision (ADR) or draft a proposal (RFC); optional standard cascade | `[topic]` | Creates adr or rfc; offers optional CPAT → rule → guide continuation |
| `/archcore:plan` | Plan a feature or initiative end-to-end | `[topic] [--product\|--sources\|--iso\|--feature]` | Routes to single plan, or one of four flows: product (idea→prd→plan), sources (mrd→brd→urd), iso (brs→strs→syrs→srs), feature (prd→spec→plan→task-type) |
| `/archcore:audit` | Documentation health — dashboard (default), `--deep` audit, or `--drift` detection | `[--deep\|--drift] [category, tag, or scope]` | Default: compact dashboard. `--deep`: coverage gaps + recommendations. `--drift`: code/cascade/temporal staleness with assisted fix |
| `/archcore:context` | Surface rules / decisions for a code area or pickup | `[path or topic]` | search_documents-backed grouped markdown |
| `/archcore:help` | Guide to Archcore commands and capabilities | — | Command catalogue, onboarding cues |

### Document-type access

Every Archcore document type is reachable via:

1. An intent skill that inlines its creation:
   - `adr`, `rule`, `guide`, `cpat`, `rfc` → `/archcore:decide`
   - `adr`, `spec`, `doc`, `guide` → `/archcore:capture`
   - `idea`, `prd`, `plan`, `task-type` → `/archcore:plan` (product/feature flows)
   - `mrd`, `brd`, `urd` → `/archcore:plan --sources`
   - `brs`, `strs`, `syrs`, `srs` → `/archcore:plan --iso`
2. A direct MCP call — `mcp__archcore__create_document(type=<any>)`.

The full mapping is in `skills-system.spec.md` → "Document-type coverage".

## Contract Surface

### Naming Conventions

- All commands use the `archcore:` plugin prefix.
- Commands use **action verbs or clear nouns**: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.
- No sub-namespaces (no `archcore:flow:iso` or similar) — Claude Code uses a single colon as plugin separator.

### Argument Handling

All commands accept arguments documented in their `argument-hint:` frontmatter.

- **With argument**: `/archcore:plan auth-redesign` — the topic is passed as `$ARGUMENTS`, skill uses it to scope work and check for duplicates.
- **Without argument**: `/archcore:plan` — skill asks an initial question to establish topic/scope.

Mode flags (`--deep`, `--drift`, `--product`, `--sources`, `--iso`, `--feature`) select between modes within a single skill.

The `/archcore:audit` command treats a non-flag argument as a **scope filter** (tag, category, type) for `--deep` and `--drift` modes. The default short mode is project-wide and ignores filters by design.

The `/archcore:plan` command treats a flag argument (`--product`, `--sources`, `--iso`, `--feature`) as a flow selector and uses the topic argument to scope the documents.

The `/archcore:init` command takes no argument; its flow is deterministic (three sequential steps).

### Discoverability

Claude Code, Cursor, and Codex CLI all show the 7 skills in a flat list. Discoverability is supported by:

1. **`/archcore:help`** — explains the active surface and routes users to the right command.
2. **SessionStart empty-state nudge** — on fresh repos, the session-start hook points users at `/archcore:init` so onboarding is self-routing.
3. **Natural conversation** — every skill is auto-invocable. The model picks the right one from user phrasing without an explicit `/` invocation.

The `/archcore:help` output structure:

```
## Quick Start (most users start here)
/archcore:init       — seed a new repo (stack rule, run guide, optional extras)
/archcore:capture    — document a module or component
/archcore:plan       — plan a feature end-to-end (single plan or full flow)
/archcore:decide     — record a decision (ADR) or draft a proposal (RFC)
/archcore:audit      — dashboard (default), `--deep` audit, or `--drift` detection
/archcore:context    — rules/decisions for a code area or pickup
/archcore:help       — this guide

## Direct document creation
For any document type, call mcp__archcore__create_document with the matching `type` parameter.

Tip: just describe what you need in natural language.
The right skill auto-invokes from the phrasing.
```

## Normative Behavior

- All commands MUST be invokable without knowledge of Archcore internals.
- All commands MUST route to the correct types/flows/analysis without forcing the user to pick a document type.
- Creation commands MUST be self-contained with inline creation recipes per document type they produce (check duplicates → ask questions → create → suggest relations).
- The `plan` skill MUST hold per-flow logic in `skills/plan/references/<flow>.md` rather than spawning new top-level skills.
- All creation commands MUST call `list_documents` before `create_document` to prevent duplicates.
- All creation commands MUST suggest `add_relation` calls after document creation.
- Analysis commands (`audit`, `context`) MUST use MCP read tools for data gathering.
- `/archcore:audit` MUST default to the short dashboard when invoked without arguments and switch modes when `--deep` or `--drift` is present.
- `/archcore:init` MUST be idempotent: each step detects existing artifacts and asks before regenerating; re-runs on a partially-initialized repo only offer the missing steps.

## Constraints

- No sub-namespaces. All commands are `archcore:<name>`.
- The visible palette MUST be exactly 7 commands. Adding an eighth skill requires a new ADR.
- Commands ask at most one scope-confirmation question before starting execution (`/archcore:init` is the exception: it runs its three-step flow with per-step accept/edit/skip).
- Flow steps within `plan` ask at most 1–2 content questions per document step.

## Invariants

- Every skill in the palette is auto-invocable (no `disable-model-invocation`).
- Every skill description enumerates trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y." format.
- Every creation command checks for duplicates first and suggests relations after.
- Every analysis command gathers data via MCP read tools before producing output.
- The `help` command accurately reflects the current 7-command surface and notes direct-MCP access for any document type.
- Every Archcore document type has at least one intent path that can create it.

## Error Handling

- If MCP server is unavailable, inform user with install/init instructions.
- If `create_document` fails due to duplicate filename, suggest an alternative slug.
- If intent routing is ambiguous, ask one scope question. If still ambiguous, default to `/archcore:capture` behavior.
- If git is unavailable for `/archcore:audit --drift`, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A user-invoked skill or Codex command wrapper conforms to this specification if:

1. Its behavior resides at `skills/<name>/SKILL.md` and `<name>` is one of: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Codex may also expose a matching `commands/<name>.md` wrapper.
2. Its description uses the "Activate when X. Do NOT activate for Y." trigger format.
3. It uses MCP tools exclusively for document operations.
4. Creation commands check for duplicates before creation and suggest relations after.
5. Analysis commands gather data via MCP read tools.
6. Its argument handling matches its `argument-hint:` frontmatter.
