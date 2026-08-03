---
title: "Context Skill Implementation Plan — Phase 1 of JTBD #1"
status: accepted
tags:
  - "commands"
  - "onboarding"
  - "plugin"
  - "skills"
---

**Status — realized, Phase 1.** Shipped in commit `3dccbd5` at plugin version 0.3.0. Delivered: `skills/context/SKILL.md` as the pull-mode skill with its scope classifier across path, topic, and pickup, plus guide routing, a top-5 cap per group, and a classification footer; anti-trigger bullets in the sibling skills; README hero copy aligned with a `/context` demo prompt added; and the CLI `search_documents` MCP tool consumed by the skill, having shipped earlier in CLI 0.1.7. The push counterpart shipped separately and is recorded in `pre-code-hook-implementation.plan`; together they close the JTBD #1 repo-alignment gap.

**Naming note.** The `context` skill survived the later `skill-surface-collapse.adr` consolidation and remains one of the 7 visible commands, but several siblings did not. Read every `/archcore:review` below as `/archcore:audit`, `/archcore:review --deep` as `/archcore:audit --deep`, `/archcore:actualize` as `/archcore:audit --drift`, `/archcore:standard` as `/archcore:decide`, and `/archcore:bootstrap` as `/archcore:init`. `/archcore:status` had already become the default short mode of the inspection skill and `/archcore:graph` had already been removed by `merge-review-status-remove-graph.adr`.

**Reference-section note (2026-05-20).** Step 3 grouping in the skill gained a **Reference** section surfacing `doc`, `rfc`, and any orphan `guide` — a guide present in the search results but not inlined under a rule, ADR, or spec by the Step 4 routing. This closed a gap where the most relevant content match could be dropped silently because its type was outside the original allow-list, which was observed when a `doc` topped relevance for a topic query and never reached the output. Read the acceptance criterion naming the rule, adr, spec, and cpat groups as including a Reference group as well.

**Deferred and non-blocking.** Snapshot tests with fixture repositories under `tests/fixtures/context/`; a CLI MCP-instructions nudge steering models toward the skill; and the `/archcore:align` push command, which is superseded by the shipped hook and this skill, recorded as rejected in `code-alignment-intent-skill.idea`.

## Goal

Ship `/archcore:context` as the user-facing pull-mode entry point for JTBD #1, repo alignment at coding time, backed by the CLI's `search_documents` MCP tool. Close the implementation gap for on-demand code-area lookup and session pickup without touching the pre-mutation hooks, which are deferred to Phase 2.

Scope is the plugin repository only. The CLI side is complete: the `search_documents` tool landed with 27 green tests covering the path and content filters, relevance and mtime sorting in Go, manifest relation enrichment, lazy body loading, UTF-8-safe excerpts, and the URL-reject heuristic.

## Architecture

The chosen shape is a search primitive plus a markdown skill. The CLI owns a generic `search_documents` primitive — filters and ranking in Go, body scan, manifest enrichment — reusable by hooks, sub-agents, and future push skills. The plugin owns the `context` skill as pure markdown that classifies scope, calls the primitive, and groups and renders the results. The separation is that what to search lives in Go, where it is stable and testable, while how to show it lives in markdown, where it evolves without a CLI release. Ranking stays deterministic in Go, so the skill does not re-sort — it groups by type, truncates to the top 5, and renders.

## Tasks

### Phase 1 — ship, blocking for release

**1. Create `skills/context/SKILL.md`.** Its frontmatter carries `name: context`, an argument hint of `[file, directory, or topic; leave empty for current-focus pickup]`, and a description whose triggers include "what rules apply to X", "before I refactor Z", "pick up where we left off", and "show me the decisions for X", with a do-not list routing creation, planning, and audits away.

Its body classifies scope — empty input means pickup, input containing a slash or naming an existing directory means path, and anything else means topic. Path mode calls the primitive with `path_ref`, a limit of 50, and relevance sorting, then groups by type and truncates each section to the top 5. Topic mode does the same with `content`. Pickup mode makes two primitive calls, one for drafts and one for recent accepted documents with a 30-day window falling back to 90, rendering In Progress, Recent Decisions, and Recent Rules. Guide routing checks each top-5 rule, ADR, and spec for an incoming `guide` linked by `implements` or `related` and inlines it as an indented bullet, tracking the inlined set so a non-inlined guide lands in Reference rather than being dropped. An empty section emits no header. A classification footer records the mode for observability. And a disambiguation note states that the skill is unrelated to the AI context window or session state, so it is not mis-invoked for chat-memory topics.

**2. Add anti-trigger bullets to the sibling skills.** Two bullets go into each "Not X" list: reading applicable rules, ADRs, and specs before coding routes to `context`, and picking up where work left off routes to `context`. The original list covered 6 siblings; under the current surface it covers 4. The purpose is to stop those skills catching pull-intent phrasing.

**3. Align the README copy.** Add a `/context` demo prompt to the try-these section, and soften the hero claim of "on every request, across sessions" to language matching the Phase 1 delivery.

### Phase 1.5 — follow-up, non-blocking

**4.** Extend the `search_documents` paragraph in the MCP server instructions to prefer the plugin skill for an interactive user-facing code-area summary.

**5.** Add two or three fixture repositories under `tests/fixtures/context/`, run the skill in a harness, and assert the markdown against a snapshot.

### Phase 2 — deferred

The pre-mutation hook for source-file edits, which is push-mode context injection, is tracked in `pre-code-context-injection.idea` and reuses the search primitive directly with no skill.

## Acceptance Criteria

**The skill.** `skills/context/SKILL.md` exists and is picked up by plugin auto-discovery. A path query returns the rule, adr, spec, and cpat groups plus a Reference group for `doc`, `rfc`, and any orphan guide, sorted by specificity then type then mtime, at 5 per section. A topic query returns content-match groups with title and body excerpts, with `doc` and `rfc` matches surfacing in Reference rather than being dropped. An argument-free invocation returns In Progress, Recent Decisions, and Recent Rules, with the 30-day to 90-day fallback when the first pass is empty. Guide routing inlines a guide under its parent where the relation exists and places an orphan guide in Reference. No header renders for an empty group, and the classification footer is always present.

**Routing precision.** "What rules apply to `src/payments/`" reaches path mode; "before I touch the billing flow" reaches path or topic; "pick up where I left off" reaches pickup; "show me the decisions for `src/payments/`" reaches path; "how many docs do we have" reaches the audit dashboard; "audit docs health" reaches `--deep`; "check for stale docs" reaches `--drift`; "document the auth module" reaches `capture`; "we decided on PostgreSQL" reaches `decide`; "plan the auth redesign" reaches `plan`; "establish a standard" reaches the `decide` continuation; and "context window" or "session state" activates nothing, per the disambiguation note.

**Siblings and README.** Each sibling intent skill lists the two new anti-trigger bullets, the try-these section includes the `/context` demo prompt, and the hero claim matches the Phase 1 delivery with automatic injection marked as upcoming.

## Dependencies

- The CLI `search_documents` tool, shipped.
- No new plugin manifest entries.
- No new hooks in Phase 1.

## Pre-merge checklist

1. The skill file is present and its frontmatter parses.
2. No YAML frontmatter error exists across the modified skill files.
3. A manual routing test confirms activation and non-activation match expectations.
4. A manual execution verifies the output shape on a non-trivial repository.
5. An anti-regression pass confirms the sibling intent skills still work.
6. The README renders cleanly on GitHub.
7. This plan is linked in the relation graph.
8. No direct write to `.archcore/` occurs; every document operation goes through MCP.
9. The plugin version is bumped.
10. The commit messages follow the existing style.

## Post-merge smoke tests

Run these against this repository. `/archcore:context skills/` should surface the skill-system rules, ADRs, and specs. `/archcore:context rules/` should surface `mcp-only-operations.rule` and `skill-file-structure.rule`. `/archcore:context "intent-based skill"` should find the intent-based skill architecture ADR. An argument-free invocation should show the draft plans plus the recent accepted rules and ADRs. And in a repository where a `doc` tops relevance for the queried topic, confirm it appears in the Reference section rather than being filtered out, repeating for an `rfc` covering the topic and for a `guide` linked to no rule, ADR, or spec.
