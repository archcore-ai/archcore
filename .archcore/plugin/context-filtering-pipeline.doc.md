---
title: "/archcore:context — Filtering Pipeline"
status: accepted
tags:
  - "commands"
  - "plugin"
  - "skills"
---

## Overview

This document records which documents `/archcore:context` surfaces and which it drops, across the two layers of its pipeline. Use it as a lookup when a relevant document does not appear in `/context` output and you want to know why, when you are tuning the skill or adding a document type, or when you are debugging an unexpected ordering of results.

Sources of truth: `internal/mcp/tools/search_documents.go` in the `archcore-ai/cli` repository for Layer 1, and `@plugins/archcore/skills/context/SKILL.md` for Layer 2. The skill markdown is canonical for every rendering decision.

## Pipeline

**Layer 1 — MCP `search_documents`, in the CLI.** Returns every document type, because `/context` passes no `types` filter. Sorts by relevance — `max_specificity` descending, then `typeRank` ascending, then `mtime` descending — with a default limit of 50.

**Layer 2 — `/archcore:context` steps 3 and 4, in the markdown skill.** Applies a type allow-list in path and topic modes, caps each section at the top 5, and renders the markdown surface. Pickup mode bypasses the allow-list and uses its own fixed sections.

## Layer 1 — MCP `search_documents`

### Inputs the skill passes

| Mode | Filter |
|---|---|
| path | `path_ref="<normalized>"`, `limit=50`, `sort="relevance"` |
| topic | `content="<argument>"`, `limit=50`, `sort="relevance"` |
| pickup | drafts: `types=["plan","idea"]`, `status="draft"`, `sort="mtime"`; recent-accepted: `types=["adr","rule"]`, `status="accepted"`, `mtime_after="30d"` (fallback `90d`) |

Content search is a strict substring match, with no stemming and no fuzzy matching; the skill retries once with shorter or alternate phrasing when the first call returns empty. Path mode normalizes `\` to `/` and strips a trailing `/` before sending.

### Sort keys in relevance mode

1. `max_specificity` descending. A content match in the title scores `3`; a content match in the body only scores `1`; a `path_ref` match scores the number of `/`-separated segments shared between the reference and the query, so `src/payments/stripe.ts` against `src/payments/` scores `2`.
2. `typeRank` ascending, per the table below.
3. `mtime` descending.

`typeRank` is a tiebreaker only. It never filters anything out by itself.

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

## Layer 2 — Step 3 grouping, in path and topic modes

The skill takes the relevance-sorted list and slots each document into a section:

| Section | Types | Cap |
|---|---|---|
| Rules | `rule` | top 5 |
| Decisions | `adr` | top 5 |
| Specs | `spec` | top 5 |
| Patterns | `cpat` | top 5 |
| Reference | `doc`, `rfc`, orphan `guide` (see Step 4) | top 5 |
| In Progress | `plan` / `idea` with `status=draft` | top 5 |

Three groups are never rendered in path or topic mode: an accepted `plan` or `idea`, because once shipped they leave the surface; `task-type`, which is experiential and out of scope for "before you touch code" context; and the vision and requirements types `prd`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, and `srs`, which describe intent rather than normative knowledge.

An empty section is suppressed: no header is rendered when its array is empty.

## Layer 2 — Step 4 guide routing and the orphan-guide concept

`guide` is handled in two passes.

1. **Inlined guide.** For each item in the Rules, Decisions, and Specs sections, the skill walks its `incoming_relations`. A `guide` pointing at that item through `implements` or `related` is rendered as an indented bullet under the parent, marked 📖. The skill tracks the set of inlined guide paths to avoid double-counting.
2. **Orphan guide.** Any `guide` returned by `search_documents` and absent from the inlined set falls through to the Reference section.

The effect is that a guide whose normative parent matched the same query stays attached to it, while a standalone guide still surfaces — lower in the output — instead of being dropped silently, which was the behavior before 2026-05-20.

## Pickup mode, invoked with no argument

Pickup uses its own fixed sections rather than the Step 3 allow-list, driven by two parallel `search_documents` calls:

| Section | Source call |
|---|---|
| In Progress | `types=["plan","idea"]`, `status="draft"`, sort by mtime |
| Recent Decisions | `types=["adr"]` from the recent-accepted call (30d → 90d fallback) |
| Recent Rules | `types=["rule"]` from the same call |

Pickup therefore surfaces no `doc`, `rfc`, or orphan guide, which is intentional: it answers "what work is current?" rather than "what knowledge applies?".

## Examples

The three examples below are non-normative illustrations of the tables above.

**Topic query `recaptcha`.** `search_documents(content="recaptcha")` returns, in order: `recaptcha-handling.doc.md` on a title match with `specificity=3` and `typeRank=17`; `error-handling.rule.md` on a body match with `specificity=1` and `typeRank=1`; `auth-popup-unit-coverage.plan.md` on a body match with `specificity=1`, `typeRank=7`, and `status=draft`; and `auth-provider-decomposition.idea.md` on a body match with `specificity=1`, `typeRank=8`, and `status=draft`. After Step 3 the surface reads: Rules holds `error-handling.rule.md`; Reference holds `recaptcha-handling.doc.md`; In Progress holds the plan and the idea. Before 2026-05-20 the Reference section did not exist and `recaptcha-handling.doc.md` was dropped silently despite being the top relevance hit.

**Path query `src/auth/popup/`.** The same allow-list applies. A `doc` referencing that path, through `@src/auth/popup/...` or a qualified bare mention, lands in Reference, while rules and ADRs go to their normative sections.

**An accepted `plan` on the same topic.** Dropped at Step 3, because `/context` answers what knowledge applies to a code area rather than what was done about it. An accepted plan is a historical record, discoverable through `/archcore:audit` or `list_documents`.

## Maintenance hooks

- **Adding a document type on the CLI side.** Set its `typePriority` in `search_documents.go` for a deterministic tiebreak, then decide its Layer 2 fate in the skill's Step 3 — allow-list, Reference, or drop.
- **Changing what `/context` surfaces** is a skill-only change in markdown and needs no CLI release. Update the Step 3 table, the Step 5 render template, and the README commands-table copy together; the post-merge notes in `context-skill-implementation.plan` hold the audit trail.
