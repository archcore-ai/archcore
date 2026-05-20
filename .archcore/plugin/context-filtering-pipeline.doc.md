---
title: "/archcore:context — Filtering Pipeline"
status: accepted
tags:
  - "commands"
  - "plugin"
  - "skills"
---

## Overview

This document describes which documents `/archcore:context` surfaces and which it drops, across the two layers of the pipeline. Use it as a lookup when:

- a relevant document doesn't appear in `/context` output and you want to know why,
- you're tuning the skill or adding a new document type,
- you're debugging an unexpected ordering of results.

Source of truth: `cli/internal/mcp/tools/search_documents.go` (Layer 1) and `plugin/skills/context/SKILL.md` (Layer 2). The skill markdown is canonical for rendering decisions.

## Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1 — MCP search_documents (cli/internal/mcp/tools/...)     │
│ Returns ALL document types. /context passes no `types` filter.  │
│ Sorts by relevance: max_specificity DESC → typeRank ASC →       │
│ mtime DESC. Default limit 50.                                   │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2 — /archcore:context Step 3-4 grouping (markdown skill)  │
│ Applies a type allow-list (path & topic modes); pickup mode has │
│ its own fixed sections. Top 5 per section.                      │
└─────────────────────────────────────────────────────────────────┘
                            ↓
                  rendered markdown surface
```

## Layer 1 — MCP `search_documents` (CLI)

### Inputs the skill passes

| Mode | Filter |
|---|---|
| path | `path_ref="<normalized>"`, `limit=50`, `sort="relevance"` |
| topic | `content="<argument>"`, `limit=50`, `sort="relevance"` |
| pickup | drafts: `types=["plan","idea"]`, `status="draft"`, `sort="mtime"`; recent-accepted: `types=["adr","rule"]`, `status="accepted"`, `mtime_after="30d"` (fallback `90d`) |

Notes:
- Content search is strict substring — no stemming, no fuzzy matching. The skill retries once with a shorter or alternate phrasing if the first call returns empty.
- Path mode normalizes `\` → `/` and strips trailing `/` before sending.

### Sort keys (relevance mode)

Ordering, in priority:

1. **`max_specificity` DESC.**
   - Content match in title → `3`.
   - Content match in body only → `1`.
   - `path_ref` match → number of `/`-separated segments shared between the reference and the query (e.g. `src/payments/stripe.ts` ↔ `src/payments/` → `2`).
2. **`typeRank` ASC** (table below).
3. **`mtime` DESC.**

`typeRank` is only a tiebreaker — it never filters anything out by itself.

### Type priority (`typeRank`)

| Rank | Type | Notes |
|---|---|---|
| 1 | `rule` | Highest — normative |
| 2 | `adr` | Decision |
| 3 | `rfc` | Open proposal |
| 4 | `spec` | Contract |
| 5 | `cpat` | Code-pattern change |
| 6 | `guide` | How-to |
| 7 | `plan` | |
| 8 | `idea` | |
| 9 | `prd` | |
| 10–16 | `brs`, `syrs`, `srs`, `strs`, `mrd`, `brd`, `urd` | Requirements (ISO + market/business/user) |
| 17 | `doc` | Reference |
| 18 | `task-type` | |
| 100 | (any unknown) | `typePriorityDefault` |

## Layer 2 — Step 3 grouping (path / topic modes)

The skill takes the relevance-sorted list and slots each document into a section:

| Section | Types | Cap |
|---|---|---|
| Rules | `rule` | top 5 |
| Decisions | `adr` | top 5 |
| Specs | `spec` | top 5 |
| Patterns | `cpat` | top 5 |
| Reference | `doc`, `rfc`, orphan `guide` (see Step 4) | top 5 |
| In Progress | `plan` / `idea` with `status=draft` | top 5 |

**Dropped types** (never rendered in path/topic mode):

- accepted `plan` / `idea` — once shipped they exit the surface,
- `task-type` — experiential, not in scope for "before you touch code" context,
- vision/requirements: `prd`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs` — they describe intent, not normative knowledge.

Empty sections are suppressed: no header is rendered if its array is empty.

## Layer 2 — Step 4 guide routing & orphan-guide concept

`guide` is handled in two passes:

1. **Inlined guide.** For each item in the Rules / Decisions / Specs sections, walk its `incoming_relations`. If a `guide` points at it via `implements` or `related`, the guide is rendered as an indented bullet under the parent (📖). The skill tracks the set of inlined guide paths to avoid double-counting.
2. **Orphan guide.** Any `guide` returned by `search_documents` but **not** in the inlined set falls through to the **Reference** section.

Effect: a guide whose normative parent matched the same query stays attached to it; a standalone guide still surfaces (just lower in the output) instead of being silently dropped, which was the pre-2026-05-20 behavior.

## Pickup mode (no argument)

Pickup has its own fixed sections and does NOT use the Step 3 allow-list. Two `search_documents` calls in parallel:

| Section | Source call |
|---|---|
| In Progress | `types=["plan","idea"]`, `status="draft"`, sort by mtime |
| Recent Decisions | `types=["adr"]` from the recent-accepted call (30d → 90d fallback) |
| Recent Rules | `types=["rule"]` from the same call |

So `doc` / `rfc` / orphan-guide are not surfaced by pickup — that's intentional. Pickup answers "what work is current?", not "what knowledge applies?".

## Examples

### Example 1 — topic query "recaptcha" (post-fix)

`search_documents(content="recaptcha")` returns, ordered:

1. `recaptcha-handling.doc.md` — title match → `specificity=3`, `typeRank=17`.
2. `error-handling.rule.md` — body match → `specificity=1`, `typeRank=1`.
3. `auth-popup-unit-coverage.plan.md` — body match → `specificity=1`, `typeRank=7`, `status=draft`.
4. `auth-provider-decomposition.idea.md` — body match → `specificity=1`, `typeRank=8`, `status=draft`.

After Step 3:

```
## Rules (1)        — error-handling.rule.md
## Reference (1)    — recaptcha-handling.doc.md
## In Progress (2)  — auth-popup-unit-coverage.plan.md, auth-provider-decomposition.idea.md
```

Pre-2026-05-20: Reference did not exist and `recaptcha-handling.doc.md` was silently dropped despite being the top relevance hit.

### Example 2 — path query `src/auth/popup/`

Same allow-list applies. `doc` files referencing that path (via `@src/auth/popup/...` or qualified bare mentions) land in Reference; rules / ADRs go to their normative sections.

### Example 3 — accepted `plan` for the same topic

Dropped at Step 3. Rationale: `/context` is "what knowledge applies to this code area", not "what was done about it". An accepted plan is historical record — discoverable via `/audit` or `list_documents`.

## Maintenance hooks

- **Adding a new document type** (CLI side): set its `typePriority` in `search_documents.go` for deterministic tie-break, and decide its Layer 2 fate in `skills/context/SKILL.md` Step 3 (allow-list, Reference, or drop).
- **Changing what `/context` surfaces** is a skill-only change (markdown) — no CLI release required. Update Step 3 table, Step 5 render template, and the README Commands-table copy together; see `context-skill-implementation.plan.md` post-merge notes for the audit trail.
