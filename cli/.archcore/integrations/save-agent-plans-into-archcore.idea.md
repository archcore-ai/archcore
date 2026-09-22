---
title: "Capture Agent Plan-Mode Plans as .archcore/*.plan.md"
status: draft
tags:
  - "integrations"
  - "mcp"
---

## Idea

When an agent runs in plan mode it already produces a plan artifact, but that artifact
lives in host-owned, usually ephemeral space (Claude Code presents it via `ExitPlanMode`
and does not persist it by default; Cursor writes it to the global `~/.cursor/plans/`;
Codex renders a transient `update_plan` checklist). Archcore should offer to route that
plan into a first-class `.archcore/<slug>.plan.md` document instead of (or in addition to)
the host's own storage.

Archcore already has the destination type: `plan` (virtual category `vision`, sections
**Goal / Tasks / Acceptance Criteria / Dependencies**). Capturing the plan makes it
git-native, versioned, relatable (`plan` → `prd` / `spec` / `adr` via relations), visible
at the next `SessionStart`, and syncable to the server / GraphRAG.

The write path is always the existing `create_document(type:'plan')` MCP primitive — no new
MCP tool. It is also not an MCP prompt: the server declares no prompts capability at all
(see the ADR on removing the MCP track prompts). What differs per host is only how the plan
is *observed* and how the capture is *triggered*.

## Value

- Turns an ephemeral / host-local plan into durable, versioned project context.
- Lands directly in archcore's `plan` template with room to link the driving `prd`/`spec`.
- Plans surface at the next session start and flow through sync — the same context loop as
  every other archcore document.
- Reinforces "one product, two entry points": a plugin-free baseline in the CLI, richer
  capture in the plugin.

## Host Feasibility (researched 2026-07-06)

No host except Claude Code exposes a clean, deterministic "hook on the plan tool". The only
truly cross-host path is a soft nudge → `create_document`, because all hosts support MCP.

| Host | Plan mode | How the plan surfaces | Interception mechanism | Verdict |
|------|-----------|-----------------------|------------------------|---------|
| Claude Code | yes (Shift+Tab) | tool call `ExitPlanMode`, input `{"plan": "<md>"}` | `PreToolUse` matched on tool name → write file / inject / deny | **STRONG** |
| Cursor | yes (Shift+Tab) | writes `.plan.md` natively to `~/.cursor/plans/`; repo `<repo>/.cursor/plans/` on "Save to Workspace" | no plan hook; FS-harvest of the emitted `.plan.md`, or nudge → MCP | **PARTIAL** |
| Copilot | yes (VS Code Plan agent / `manage_todo_list`) | VS Code: tool call `manage_todo_list`; CLI: inline text; cloud: PR checklist | VS Code only: `PostToolUse` on `manage_todo_list` (**Preview**); CLI/cloud: nudge | **PARTIAL** (STRONG only in VS Code) |
| OpenCode | yes (Tab: `plan`/`build`) | inline message + optional `todowrite` | plugin `event` bus (`todo.updated`/`message.updated`) with FS access writes the file | **PARTIAL** (via an OpenCode plugin) |
| Codex CLI | yes (Shift+Tab / `/plan`) | tool call `update_plan` (checklist, not a file) | hooks do **not** cover `update_plan` (proposal issue #24547); deterministic only via a `PLANS.md` file + `apply_patch` `PostToolUse` | **PARTIAL** (native plan = instruction-only) |

Key takeaways:

- Codex ships a full hooks framework (`SessionStart`/`PreToolUse`/`PostToolUse`/`Stop`/…),
  but `PreToolUse`/`PostToolUse` fire only for Bash, `apply_patch`, and MCP calls — the native
  `update_plan` tool is not covered.
- OpenCode's capture lives in its plugin runtime, and MCP tool calls do **not** trigger those
  plugin hooks (open issue #2319), so a plugin cannot even observe the agent calling archcore's
  MCP tools — it can only watch native tools / the event bus.
- Cursor already uses the exact `.plan.md` suffix and can save the plan into the repo, making
  it the closest fit for an FS-harvest import.

## Possible Implementation

Layered: a plugin-free baseline first, richer UX on top. Ordered by cost.

**Phase 1 — instruction nudge (CLI, universal, plugin-free).**
Extend the usage-nudge content (`@internal/agents/instructions.go`) so that after planning /
before a multi-step change the agent persists its plan via `create_document(type:'plan')` and,
if applicable, links it to the driving `prd`/`spec`. Rides the existing
`archcore instructions install`; ships to all eight instruction hosts; zero new code paths.
Soft (compliance-dependent) but the only path that lands directly in archcore format on every
host. This is the baseline the four non-Claude hosts rely on.

**Phase 2 — deterministic hook capture (CLI, Claude Code only).**
Add an opt-in `PreToolUse` handler matched on `ExitPlanMode`. The handler reads `{"plan": ...}`
from the payload and writes a draft `.archcore/<slug>.plan.md` (or injects "call
`create_document`"). This is a deterministic tool-name match, **not** the natural-language
keyword matching that got Stop/UserPromptSubmit removed (`disable-stop-and-prompt-hooks`), so
that rationale does not apply.

The cost of this phase dropped substantially after the guardrail repatriation: the CLI now
runs `PreToolUse` on five hosts with a shared dialect table, a payload decoder, and a
per-(host, event) install path. Capture would add a matcher and a handler branch, not a new
hook path. What remains host-specific is the `ExitPlanMode` tool name itself — only Claude Code
offers a clean deterministic plan-tool payload, so do not build equivalents elsewhere.

**Phase 3 — plugin capture (plugin entry point).**
The plugin is the natural home for interception on the hosts whose surfaces are plugin-like:
ship an OpenCode plugin that watches `todo.updated`/`message.updated` under the `plan` agent,
and add the richer capture flow the bare nudge cannot guarantee — normalize the raw plan into
the Goal/Tasks/Acceptance-Criteria/Dependencies template, propose a relation to the driving
doc, and confirm before writing.

**Optional — FS-harvest import (CLI, Cursor-first).**
An out-of-band `archcore import-plans`-style command / watcher that reads `<repo>/.cursor/plans/*.plan.md`
(and, by extension, Codex `PLANS.md`, Visual Studio `.copilot/plans/*.md`) and normalizes them
into `.archcore/<slug>.plan.md` with proper frontmatter. Not a hook; a file importer. Attractive
because Cursor already emits the same `.plan.md` convention. Note that this would grow the
public command surface, which the current design deliberately holds at nine commands.

## Risks

- **No uniform mechanism.** Only Claude Code gives a first-class deterministic hook; every other
  host needs a different, weaker path. Resist the temptation to build five bespoke handlers — the
  nudge is the portable baseline.
- **Do not add an MCP tool.** A `save_plan` tool breaks the "tools stay CRUD primitives" stance.
  Write via `create_document(type:'plan')`. An MCP prompt is not an option either: the server
  declares no prompts capability.
- **No auto-relations.** Per `no-auto-relations-on-create-document`, the link from a captured plan
  to its driving `prd`/`spec` must be proposed/confirmed, never written silently.
- **Plan spam.** Trivial "let me plan this small edit" plans should not litter `.archcore/`. Gate
  capture behind confirm / a size threshold; default to `draft` status. This is the dominant risk:
  a corpus that accumulates executed plans has to be pruned by hand, because a finished plan holds
  no durable knowledge that an `adr`, `spec`, or `guide` does not hold better.
- **Preview / churn.** Copilot VS Code hooks and the `manage_todo_list` payload are Preview; Codex
  `update_plan` lifecycle hooks are a proposal (#24547). Anything built on them may shift.
- **FS-harvest fragility.** Cursor's default plan dir is global (`~/.cursor/plans/`), snapshots can
  multiply per plan, and there is no archcore frontmatter — the importer must own slug derivation
  and dedup.

## Open Questions

- Capture status: `draft` vs `accepted` on write.
- Slug derivation from the plan title; dedup / update-in-place when a plan is re-planned.
- How to propose the relation to the driving document without violating no-auto-relations.
- Whether the optional FS-harvest importer is worth building before Phase 1/2 land.
- **Lifecycle after execution.** Nothing in this idea says what happens to a captured plan once
  its tasks are done. Without an answer, capture trades an ephemeral artifact for a permanent one
  that outlives its usefulness.

## Related

- `.archcore/integrations/instruction-nudge-on-init.adr.md` — the nudge mechanism Phase 1 extends.
- `.archcore/integrations/disable-stop-and-prompt-hooks.adr.md` — why the keyword-matching hooks were
  removed, and why the `ExitPlanMode` match does not fall under that rationale.
- `.archcore/integrations/hook-guardrails-in-the-cli.adr.md` — the repatriation that made a
  `PreToolUse` handler cheap to add.
- `.archcore/integrations/hook-runtime.spec.md` — the contract any new handler must satisfy.
- `.archcore/integrations/supported-ai-agents.doc.md` — host registry.
- `.archcore/integrations/agent-hooks-integration.doc.md` — integration surfaces.
- Global `architecture/one-product-two-entry-points` (read-only) — CLI baseline vs plugin runtime
  framing; referenced in prose, not linked (globals are not related to).
