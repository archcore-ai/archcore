---
title: "Context Skill Implementation Plan — Phase 1 of JTBD #1"
status: accepted
tags:
  - "commands"
  - "onboarding"
  - "plugin"
  - "skills"
---

## Status — Realized (Phase 1)

Shipped in commit `3dccbd5` (feat: new skill context), plugin version 0.3.0.

Delivered:

- `skills/context/SKILL.md` — pull-mode skill with scope classifier (path / topic / pickup), guide-routing, top-5 per group, classification footer.
- Anti-trigger bullets added to 6 sibling skills (capture, decide, standard, plan, review, actualize).
- README hero copy aligned; `/context` demo prompt added to "Try these 3 prompts first".
- CLI `search_documents` MCP tool consumed by the skill (shipped earlier in CLI 0.1.7).

The push counterpart (`bin/check-code-alignment`) shipped separately in commit `87d384c` — see `pre-code-hook-implementation.plan.md`. Together they close the JTBD #1 repo-alignment gap (pull + push).

Deferred (non-blocking, tracked here for follow-up):

- Snapshot tests with fixture `.archcore/` repos under `tests/fixtures/context/`.
- CLI MCP-instructions nudge to steer models toward the skill when appropriate.
- `/archcore:align` push-mode command — **superseded** by the shipped hook + /context skill. See `code-alignment-intent-skill.idea.md` (rejected).

## Goal

Ship `/archcore:context` as the user-facing pull-mode entry point for JTBD #1 ("repo-alignment at coding time"), backed by the CLI's `search_documents` MCP tool. Close the JTBD-implementation gap for on-demand code-area lookup and session pickup, without touching PreToolUse hooks (deferred to Phase 2).

Scope: plugin repo only. CLI side is complete (`search_documents` tool landed with 27 green tests — path_ref/content filters, sort="relevance"|"mtime" in Go, manifest relation enrichment, lazy body load, UTF-8 safe excerpts, URL-reject regex heuristic).

## Architecture — Alternative C (search primitive + markdown skill)

- CLI: generic `search_documents` primitive (filters + ranking in Go, body scan, manifest enrichment). Reusable by hooks, sub-agents, future `/align` push skill.
- Plugin: `/archcore:context` skill is pure markdown — classifies scope, calls the primitive, groups/renders results.
- Separation: "what to search" lives in Go (stable, testable). "How to show" lives in markdown (evolves without CLI release).
- Ranking stays deterministic (Go), so the skill does not re-sort — it groups by type, truncates top-5, renders.

## Tasks

### Phase 1 — Ship (blocking for release)

**1. Create `skills/context/SKILL.md`**

Frontmatter (final, post-prompt-engineer review):

- `name: context`
- `argument-hint: "[file, directory, or topic; leave empty for current-focus pickup]"`
- `description`: trigger phrases include "what rules apply to X", "before I refactor Z", "pick up where we left off", "where is the payments work right now", "what was I working on in X", "show me the decisions/rules/specs for X". DO-NOT list routes creation/planning/audits/graph/status away.

Body sections:
- **Classify scope** — empty/whitespace → pickup; contains `/` OR is an existing repo directory → path; otherwise → topic.
- **Path mode** — `search_documents(path_ref, limit=50, sort="relevance")`, group by type (rule/adr/spec/cpat/plan-draft/idea-draft), truncate each section to top-5, render.
- **Topic mode** — same but `content="<scope>"`.
- **Pickup mode** — two primitive calls: drafts + recent-accepted (30d → fallback 90d). Render as In Progress / Recent Decisions / Recent Rules.
- **Guide routing** — for each rule/adr/spec top-5, check `incoming_relations` for a `guide` linked via `implements`/`related`; inline as indented bullet.
- **Empty-header suppression** — do NOT emit a section header if its array is empty (override of the earlier "always render header with `_none_`" idea — found to be noisy in review).
- **Classification footer** — `_Classified as: <mode>._` for observability.
- **Disambiguation note** — "Not related to the AI context window or session state" in body, so the skill does not get mis-invoked for chat memory topics.

**2. Anti-trigger bullets in 6 sibling skills**

Add to each "Not X:" list in:
- `skills/capture/SKILL.md`
- `skills/decide/SKILL.md`
- `skills/standard/SKILL.md`
- `skills/plan/SKILL.md`
- `skills/review/SKILL.md`
- `skills/actualize/SKILL.md`

The two bullets:
- Reading applicable rules/ADRs/specs before coding → `/archcore:context`
- Picking up where work left off → `/archcore:context`

Purpose: stop these skills from catching "pull"-intent phrases.

**3. README.md copy alignment**

- Add a `/context` demo-prompt to "Try these 3 prompts first" (now 4 prompts, or replace #1 since it's vague).
- Soften "on every request, across sessions" in the hero section — replace with language that matches the Phase 1 delivery ("on demand with `/archcore:context`, ...auto-injection on edits is coming").

### Phase 1.5 — Follow-up (non-blocking)

**4. CLI MCP instructions nudge**

In `internal/mcp/server.go`, extend the `search_documents` paragraph with: "For an interactive user-facing code-area summary, prefer the `/archcore:context` plugin skill which composes `search_documents` with sensible defaults." One commit in cli-repo, separate PR.

**5. Snapshot tests**

Two or three fixture `.archcore/` repos under `tests/fixtures/context/`. Run the skill in a harness and assert markdown matches a snapshot. Covers grouping order, top-5 truncation, guide routing, empty-section suppression, classification footer.

### Phase 2 — Deferred (tracked as separate idea/plan)

- PreToolUse hook for source-file edits — push-mode context injection. See `pre-code-context-injection.idea.md`. Will reuse `search_documents` directly (no skill).
- `/archcore:align` push-mode command — see `code-alignment-intent-skill.idea.md`.

## Acceptance Criteria

**SKILL.md**
- `skills/context/SKILL.md` exists, picked up by plugin auto-discovery (no manifest registration needed).
- `/archcore:context src/payments/` returns rule+adr+spec+cpat groups sorted by specificity→type→mtime, top-5 per section.
- `/archcore:context "money rounding"` returns content-match groups with title/body excerpts.
- `/archcore:context` (no argument) returns In Progress + Recent Decisions + Recent Rules, with 30d→90d fallback when first pass is empty.
- Guide routing: when a rule/adr/spec has an incoming `guide` via `implements` or `related`, guide appears as an indented bullet below the parent.
- No section header is rendered when its group is empty (other than the classification footer).
- Classification footer is always present.

**Routing precision (manual test matrix — 14 cases)**
- "what rules apply to src/payments/" → `/archcore:context` path mode
- "before I touch the billing flow" → `/archcore:context` path or topic
- "pick up where I left off" → `/archcore:context` pickup mode
- "where is the payments work right now" → `/archcore:context` pickup
- "show me the decisions for src/payments/" → `/archcore:context` path
- "how many docs do we have" → `/archcore:status` (NOT context)
- "draw the graph" → `/archcore:graph` (NOT context)
- "review docs health" → `/archcore:review` (NOT context)
- "check for stale docs" → `/archcore:actualize` (NOT context)
- "document the auth module" → `/archcore:capture` (NOT context)
- "we decided on PostgreSQL" → `/archcore:decide` (NOT context)
- "plan the auth redesign" → `/archcore:plan` (NOT context)
- "make this a standard" → `/archcore:standard` (NOT context)
- "context window" / "session state" → no activation (disambig note)

**Sibling anti-trigger**
- Each of the 6 skills lists the 2 new "Not X:" bullets referencing `/archcore:context`.

**README**
- "Try these" section includes a `/context` demo-prompt.
- Hero overclaim softened to match Phase 1 delivery; PreToolUse auto-injection is explicitly marked as upcoming.

## Dependencies

- CLI `search_documents` tool — SHIPPED (see `search_documents.go`, `search_documents_test.go`). No further CLI work required for Phase 1.
- No new plugin manifest entries (Claude + Cursor manifests point at `skills/` directory; auto-discovery).
- No new hooks.

## Pre-merge validity checklist

1. `skills/context/SKILL.md` present, frontmatter parses (name, description, argument-hint).
2. No YAML frontmatter errors across all modified SKILL.md files (lint: plugin test suite).
3. Manual routing test — run 14 trigger phrases above in Claude Code + Cursor, confirm activation / non-activation matches expectations.
4. Manual skill execution — `/archcore:context src/payments/`, `/archcore:context "money rounding"`, `/archcore:context` on a non-trivial `.archcore/` repo; verify output shape.
5. Anti-regression — run `/archcore:status`, `/archcore:graph`, `/archcore:review`, `/archcore:actualize` to confirm siblings still work after edits.
6. README renders cleanly on GitHub (no broken code fences, valid markdown).
7. Plan doc (this file) links in the graph (relations to jtbd-alignment-analysis, code-alignment-intent-skill, inverted-invocation-policy, pre-code-context-injection, intent-skill-implementation).
8. No direct writes to `.archcore/` — all doc ops via MCP.
9. Plugin version bumped in `.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json` (0.2.3 → 0.3.0 for new skill).
10. Commit messages follow existing style (e.g., `feat: add /archcore:context skill`, `docs: align JTBD #1 copy with Phase 1 delivery`).

## Post-merge smoke tests (this repo)

Run in Claude Code against this plugin repo:

- `/archcore:context skills/` — should surface skill-system-related rules/adrs/specs.
- `/archcore:context rules/` — should surface mcp-only-operations.rule, skill-file-structure.rule.
- `/archcore:context "intent-based skill"` — should find intent-based-skill-architecture.adr.
- `/archcore:context` with no argument — should show draft plans (this one, scenario-track-skills-implementation.plan) + recent accepted rules/ADRs.
