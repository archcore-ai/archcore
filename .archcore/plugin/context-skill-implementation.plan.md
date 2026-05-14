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

> **Outcome (2026-05-15):** Shipped. The `context` skill survived the subsequent `skill-surface-collapse.adr.md` consolidation and remains one of the 7 visible commands. The sibling anti-trigger updates below were obsoleted when those siblings were merged or removed: `review` and `actualize` merged into `audit`; `standard` merged into `decide`'s continuation chain; `bootstrap` renamed to `init`.

Shipped in commit `3dccbd5` (feat: new skill context), plugin version 0.3.0.

Delivered:

- `skills/context/SKILL.md` — pull-mode skill with scope classifier (path / topic / pickup), guide-routing, top-5 per group, classification footer.
- Anti-trigger bullets added to sibling skills (capture, decide, plan, audit) — original list of 6 has been consolidated as those skills merged.
- README hero copy aligned; `/context` demo prompt added to "Try these 3 prompts first".
- CLI `search_documents` MCP tool consumed by the skill (shipped earlier in CLI 0.1.7).

The push counterpart (`bin/check-code-alignment`) shipped separately in commit `87d384c` — see `pre-code-hook-implementation.plan.md`. Together they close the JTBD #1 repo-alignment gap (pull + push).

Deferred (non-blocking, tracked here for follow-up):

- Snapshot tests with fixture `.archcore/` repos under `tests/fixtures/context/`.
- CLI MCP-instructions nudge to steer models toward the skill when appropriate.
- `/archcore:align` push-mode command — **superseded** by the shipped hook + /context skill. See `code-alignment-intent-skill.idea.md` (rejected).

> **Note (post-merge cleanup, 2026-05-07):** This plan originally referenced `/archcore:status` and `/archcore:graph` in its routing matrix and anti-regression checklist. Both were removed/merged (see `merge-review-status-remove-graph.adr.md`): `/archcore:status` became the default short mode of `/archcore:review`; `/archcore:graph` was deleted entirely.

> **Note (skill-surface-collapse cleanup, 2026-05-15):** `/archcore:review` was subsequently merged into `/archcore:audit` and `/archcore:actualize` was folded into `/archcore:audit --drift` (see `skill-surface-collapse.adr.md`). References below to `/archcore:review`, `/archcore:review --deep`, `/archcore:actualize`, `/archcore:standard`, and `/archcore:bootstrap` should be read as their successors: `audit` (default), `audit --deep`, `audit --drift`, `decide`, and `init` respectively.

## Goal

Ship `/archcore:context` as the user-facing pull-mode entry point for JTBD #1 ("repo-alignment at coding time"), backed by the CLI's `search_documents` MCP tool. Close the JTBD-implementation gap for on-demand code-area lookup and session pickup, without touching PreToolUse hooks (deferred to Phase 2).

Scope: plugin repo only. CLI side is complete (`search_documents` tool landed with 27 green tests — path_ref/content filters, sort="relevance"|"mtime" in Go, manifest relation enrichment, lazy body load, UTF-8 safe excerpts, URL-reject regex heuristic).

## Architecture — Alternative C (search primitive + markdown skill)

- CLI: generic `search_documents` primitive (filters + ranking in Go, body scan, manifest enrichment). Reusable by hooks, sub-agents, future push skills.
- Plugin: `/archcore:context` skill is pure markdown — classifies scope, calls the primitive, groups/renders results.
- Separation: "what to search" lives in Go (stable, testable). "How to show" lives in markdown (evolves without CLI release).
- Ranking stays deterministic (Go), so the skill does not re-sort — it groups by type, truncates top-5, renders.

## Tasks

### Phase 1 — Ship (blocking for release)

**1. Create `skills/context/SKILL.md`**

Frontmatter:

- `name: context`
- `argument-hint: "[file, directory, or topic; leave empty for current-focus pickup]"`
- `description`: trigger phrases include "what rules apply to X", "before I refactor Z", "pick up where we left off", "where is the payments work right now", "what was I working on in X", "show me the decisions/rules/specs for X". DO-NOT list routes creation/planning/audits away.

Body sections:
- **Classify scope** — empty/whitespace → pickup; contains `/` OR is an existing repo directory → path; otherwise → topic.
- **Path mode** — `search_documents(path_ref, limit=50, sort="relevance")`, group by type (rule/adr/spec/cpat/plan-draft/idea-draft), truncate each section to top-5, render.
- **Topic mode** — same but `content="<scope>"`.
- **Pickup mode** — two primitive calls: drafts + recent-accepted (30d → fallback 90d). Render as In Progress / Recent Decisions / Recent Rules.
- **Guide routing** — for each rule/adr/spec top-5, check `incoming_relations` for a `guide` linked via `implements`/`related`; inline as indented bullet.
- **Empty-header suppression** — do NOT emit a section header if its array is empty.
- **Classification footer** — `_Classified as: <mode>._` for observability.
- **Disambiguation note** — "Not related to the AI context window or session state" in body, so the skill does not get mis-invoked for chat memory topics.

**2. Anti-trigger bullets in sibling skills**

Add to each "Not X:" list in the sibling intent skills. Original list referenced 6 skills (capture, decide, standard, plan, review, actualize); under the current 7-skill surface this becomes 4 (capture, decide, plan, audit).

The two bullets:
- Reading applicable rules/ADRs/specs before coding → `/archcore:context`
- Picking up where work left off → `/archcore:context`

Purpose: stop these skills from catching "pull"-intent phrases.

**3. README.md copy alignment**

- Add a `/context` demo-prompt to "Try these 3 prompts first" (now 4 prompts, or replace #1 since it's vague).
- Soften "on every request, across sessions" in the hero section — replace with language that matches the Phase 1 delivery.

### Phase 1.5 — Follow-up (non-blocking)

**4. CLI MCP instructions nudge**

In `internal/mcp/server.go`, extend the `search_documents` paragraph with: "For an interactive user-facing code-area summary, prefer the `/archcore:context` plugin skill which composes `search_documents` with sensible defaults."

**5. Snapshot tests**

Two or three fixture `.archcore/` repos under `tests/fixtures/context/`. Run the skill in a harness and assert markdown matches a snapshot.

### Phase 2 — Deferred (tracked as separate idea/plan)

- PreToolUse hook for source-file edits — push-mode context injection. See `pre-code-context-injection.idea.md`. Will reuse `search_documents` directly (no skill).

## Acceptance Criteria

**SKILL.md**
- `skills/context/SKILL.md` exists, picked up by plugin auto-discovery.
- `/archcore:context src/payments/` returns rule+adr+spec+cpat groups sorted by specificity→type→mtime, top-5 per section.
- `/archcore:context "money rounding"` returns content-match groups with title/body excerpts.
- `/archcore:context` (no argument) returns In Progress + Recent Decisions + Recent Rules, with 30d→90d fallback when first pass is empty.
- Guide routing: when a rule/adr/spec has an incoming `guide` via `implements` or `related`, guide appears as an indented bullet below the parent.
- No section header is rendered when its group is empty.
- Classification footer is always present.

**Routing precision (manual test matrix)** — historical list referenced removed skills; under the current surface the disambiguation map is:
- "what rules apply to src/payments/" → `/archcore:context` path mode
- "before I touch the billing flow" → `/archcore:context` path or topic
- "pick up where I left off" → `/archcore:context` pickup mode
- "show me the decisions for src/payments/" → `/archcore:context` path
- "how many docs do we have" → `/archcore:audit` (default short mode)
- "audit docs health" → `/archcore:audit --deep`
- "check for stale docs" → `/archcore:audit --drift`
- "document the auth module" → `/archcore:capture`
- "we decided on PostgreSQL" → `/archcore:decide`
- "plan the auth redesign" → `/archcore:plan`
- "establish a standard" → `/archcore:decide` (ADR + rule + guide continuation)
- "context window" / "session state" → no activation (disambig note)

**Sibling anti-trigger**
- Each sibling intent skill lists the 2 new "Not X:" bullets referencing `/archcore:context`.

**README**
- "Try these" section includes a `/context` demo-prompt.
- Hero overclaim softened to match Phase 1 delivery; PreToolUse auto-injection marked as upcoming.

## Dependencies

- CLI `search_documents` tool — SHIPPED.
- No new plugin manifest entries.
- No new hooks (Phase 1).

## Pre-merge validity checklist

1. `skills/context/SKILL.md` present, frontmatter parses.
2. No YAML frontmatter errors across all modified SKILL.md files.
3. Manual routing test — confirm activation / non-activation matches expectations.
4. Manual skill execution — verify output shape on a non-trivial `.archcore/` repo.
5. Anti-regression — run sibling intent skills to confirm they still work.
6. README renders cleanly on GitHub.
7. Plan doc (this file) links in the graph.
8. No direct writes to `.archcore/` — all doc ops via MCP.
9. Plugin version bumped.
10. Commit messages follow existing style.

## Post-merge smoke tests (this repo)

Run in Claude Code against this plugin repo:

- `/archcore:context skills/` — should surface skill-system-related rules/adrs/specs.
- `/archcore:context rules/` — should surface mcp-only-operations.rule, skill-file-structure.rule.
- `/archcore:context "intent-based skill"` — should find intent-based-skill-architecture.adr.
- `/archcore:context` with no argument — should show draft plans + recent accepted rules/ADRs.
