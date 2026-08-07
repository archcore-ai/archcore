---
title: "User-Invoked Skills — Seven-Command Surface Specification"
status: rejected
tags:
  - "commands"
  - "plugin"
---

## Purpose & Scope

This spec defines the contract for user-invoked skills: discoverability, naming, argument handling, and behavior when a user invokes them as slash commands. The current surface is **7 commands**, all auto-invocable, fixed by `skill-surface-collapse.adr`. Normative for the external-facing command surface; `skills-system.spec` remains normative for internal skill structure. Depended on by users of the `/` menu on all four hosts and by the `commands/*.md` wrappers. Out of scope: MCP tools, and the prior tiered intent/track/utility structure, which is superseded — every remaining skill is intent-class.

Host surfacing differs. Claude Code and Cursor surface user-invoked workflows directly from skills. Codex CLI requires `commands/*.md` wrappers — thin host-adapter shims that delegate to `skills/<name>/SKILL.md`. GitHub Copilot CLI surfaces skills directly, but its manifest gives the `commands` field no default path, so `.plugin/plugin.json` names `./commands/` explicitly; without that pointer the wrappers do not load there. On Copilot a skill takes precedence over a command of the same name, so a wrapper is a fallback surface rather than the primary one. The skill file is the single behavioral source of truth on every host.

## Surface

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

Total visible: **7 commands**. Total skills on disk: **7**. No hidden surface, no utility-only flag. A user describes intent in natural language and the model routes; an explicit `/archcore:<name>` invocation behaves identically.

| Command | Description (in skill picker) | Argument | Behavior |
|---|---|---|---|
| `/archcore:init` | First-time onboarding — detect scale + shape, compose a full first-day seed (stack rule, run guide, data-model, integrations, config, entry points, architecture overview, hotspot specs) and import agent files | `[--mode=small\|medium\|large] [--domain=<slug>] [--refresh]` | Detect → compose → one preview → single `confirm` → create + wire relations. Nothing written before confirm. Idempotent (skip-on-exists). Also installs host wiring (project MCP config, SessionStart hook, usage hint) matching `archcore init` — see `host-wiring-parity.adr`. See `magic-first-day-init.adr`. |
| `/archcore:capture` | Document a module / component / system | `[topic]` | Routes to adr / spec / doc / guide based on context |
| `/archcore:decide` | Record a decision (ADR) or draft a proposal (RFC); optional standard cascade | `[topic]` | Creates adr or rfc; offers optional CPAT → rule → guide continuation |
| `/archcore:plan` | Plan a feature or initiative end-to-end | `[topic] [--product\|--sources\|--iso\|--feature]` | Routes to single plan, or one of four flows: product (idea→prd→plan), sources (mrd→brd→urd), iso (brs→strs→syrs→srs), feature (prd→spec→plan→task-type) |
| `/archcore:audit` | Documentation health — dashboard (default), `--deep` audit, or `--drift` detection | `[--deep\|--drift] [category, tag, or scope]` | Default: compact dashboard. `--deep`: coverage gaps + recommendations. `--drift`: code/cascade/temporal staleness with assisted fix |
| `/archcore:context` | Surface rules / decisions for a code area or pickup | `[path, topic, --git-changes]` | search_documents-backed grouped markdown; `--git-changes` derives scope from the working tree |
| `/archcore:help` | Guide to Archcore commands and capabilities | — | Command catalogue, onboarding cues |

**Document-type access.** Every Archcore document type is reachable through an intent skill that inlines its creation, or through a direct `mcp__archcore__create_document(type=<any>)` call:

- `adr`, `rule`, `guide`, `cpat`, `rfc` → `/archcore:decide`
- `adr`, `spec`, `doc`, `guide` → `/archcore:capture`
- `idea`, `prd`, `plan`, `task-type` → `/archcore:plan` (product and feature flows)
- `mrd`, `brd`, `urd` → `/archcore:plan --sources`
- `brs`, `strs`, `syrs`, `srs` → `/archcore:plan --iso`
- `rule`, `spec`, `doc` → `/archcore:init` also composes these in the first-day seed; it is the only skill that seeds documents in bulk, behind a single preview and confirm

The full mapping lives in `skills-system.spec` under "Document-type coverage".

**Naming.** Every command carries the `archcore:` plugin prefix and an action verb or a clear noun: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. There are no sub-namespaces — Claude Code uses a single colon as the plugin separator.

**Arguments.** Every command accepts the arguments documented in its `argument-hint:` frontmatter. With an argument (`/archcore:plan auth-redesign`) the topic arrives as `$ARGUMENTS` and scopes the work and the duplicate check; without one the skill asks an initial question to establish topic and scope. Mode flags (`--deep`, `--drift`, `--product`, `--sources`, `--iso`, `--feature`, `--git-changes`, `--mode`, `--domain`, `--refresh`) select a mode inside a single skill.

- `/archcore:audit` treats a non-flag argument as a scope filter (tag, category, type) for `--deep` and `--drift`. The default short mode is project-wide and ignores filters by design.
- `/archcore:plan` treats `--product`, `--sources`, `--iso`, and `--feature` as flow selectors, and uses the topic argument to scope the documents.
- `/archcore:context` additionally accepts `--git-changes` — working-tree scope (staged, unstaged, and untracked against HEAD, minus `.archcore/`) — which replaces path and topic classification with a git-derived path set: one `search_documents` call per changed directory, deduped and capped. It short-circuits to an empty state when git is unavailable. The agent MAY invoke `--git-changes` proactively, but only once per task over a dirty working tree.
- `/archcore:init` accepts `--mode=small|medium|large` to force the detected scale, `--domain=<slug>` for a large-repo re-run scoped to one domain, and `--refresh` to top up an already-seeded repository with facts that appeared since.

**Discoverability.** All four hosts show the 7 skills in a flat list, supported by `/archcore:help`, by the SessionStart empty-state nudge that points a fresh repository at `/archcore:init`, and by auto-invocation from natural phrasing. The `/archcore:help` output structure:

```
## Quick Start (most users start here)
/archcore:init       — seed a new repo (facts + hotspot specs + linked overview, one preview/confirm)
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

1. A command MUST be invokable without knowledge of Archcore internals.
2. A command MUST route to the correct type, flow, or analysis without asking the user to pick a document type.
3. A creation command MUST carry an inline creation recipe for each document type it produces.
4. WHEN a creation command runs, the command MUST call `list_documents` before `create_document`.
5. WHEN a creation command has created a document, the command MUST suggest `add_relation` calls.
6. An analysis command (`audit`, `context`) MUST gather its data through MCP read tools.
7. The `plan` skill MUST hold its per-flow logic in `skills/plan/references/<flow>.md`.
8. WHEN the user invokes `/archcore:audit` without an argument, the skill MUST produce the short dashboard.
9. WHEN the user invokes `/archcore:audit` with `--deep` or `--drift`, the skill MUST switch to that mode.
10. WHILE `/archcore:init` awaits the user's confirmation, the skill MUST NOT create a document.
11. WHILE `/archcore:init` awaits the user's confirmation, the skill MUST NOT write a host-wiring file outside `.archcore/`.
12. WHEN `/archcore:init` finds an artifact already present, the skill MUST skip it and show that skip in the preview.
13. WHEN `/archcore:init` runs on a fully seeded repository without `--refresh` and without `--domain`, the skill MUST exit early.

## Constraints & Invariants

- Constraint: the visible palette MUST hold exactly 7 commands. An eighth skill requires a new ADR.
- Constraint: every command is `archcore:<name>`; sub-namespaces are unavailable because Claude Code uses a single colon as the plugin separator.
- Constraint: a command asks at most one scope-confirmation question before execution. `/archcore:init` is the exception: it presents one preview manifest and proceeds on a single `confirm`, `edit`, or `cancel`.
- Constraint: a flow step inside `plan` asks at most 1–2 content questions per document step.
- Constraint: a host that provides no default `commands/` path MUST point at the wrappers from its manifest. On Copilot a missing pointer removes the entire `/archcore:*` surface, which `@test/structure/copilot-plugin.bats` pins.
- Invariant: every skill in the palette is auto-invocable, carrying no `disable-model-invocation`.
- Invariant: every skill description enumerates trigger phrases and anti-triggers in the `Activate when X. Do NOT activate for Y.` format.
- Invariant: every creation command checks duplicates first and suggests relations afterwards.
- Invariant: every analysis command gathers data through MCP read tools before producing output.
- Invariant: `help` reflects the current 7-command surface and names direct-MCP access for any document type.
- Invariant: every Archcore document type has at least one intent path that can create it.
- Invariant: the set of `commands/*.md` wrappers matches the set of `skills/<name>/` directories exactly. A wrapper naming a removed skill would surface a `/archcore:<name>` entry that dead-ends.

## Failure Behavior

1. IF the MCP server is unavailable, THEN the command MUST inform the user with install and init instructions.
2. IF `create_document` fails on a duplicate filename, THEN the command MUST suggest an alternative slug.
3. IF intent routing is ambiguous, THEN the command MUST ask one scope question.
4. IF routing stays ambiguous after that question, THEN the command MUST fall back to `/archcore:capture` behavior.
5. IF git is unavailable during `/archcore:audit --drift`, THEN the skill MUST skip code-drift analysis and run the cascade and temporal analyses only.

## Conformance

A user-invoked skill or its command wrapper is conformant when:

1. Its behavior resides at `skills/<name>/SKILL.md`, and `<name>` is one of `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Codex and Copilot may also expose a matching `commands/<name>.md` wrapper — required on Codex, which does not surface skills directly, and a fallback on Copilot, which does.
2. Its description uses the `Activate when X. Do NOT activate for Y.` trigger format.
3. It performs document operations through MCP tools only.
4. It checks duplicates before creation and suggests relations afterwards, where it is a creation command.
5. It gathers data through MCP read tools, where it is an analysis command.
6. Its argument handling matches its `argument-hint:` frontmatter.
