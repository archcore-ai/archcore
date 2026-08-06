---
title: "Optimize MCP Tool Definitions and System Prompt for Token Efficiency"
status: draft
tags:
  - "cli"
---

## Idea

Reduce token consumption of the MCP server by deduplicating tool descriptions, trimming redundant response fields, and consolidating disambiguation rules — without sacrificing document classification accuracy.

Original baseline: **~5,600 tokens/session** fixed overhead (system instructions ~2,900 + tool schemas ~2,100 + session context ~600). Realistic savings: **800–1,300 tokens/session (15–23%)**.

## Status

**Partially shipped.** The "safe" subset landed in commit `05d1af3 fix: safe token optimizations in MCP tools` (2026-05-12). Two further reductions landed later with the prompts removal and the session-recap work. The remaining open items are response-shape changes.

### Shipped (safe)

- **Compressed `create_document` description** (partial Item 1). The 18 document types are still enumerated in the tool description, but each entry is now one or two lines instead of a full multi-line section list. Section detail is delegated to the auto-generated template. The `tags` and `content` parameter descriptions were also tightened, with longer reference text pushed to server instructions. Net effect: roughly half the tokens of the original create_document schema, without removing type guidance.
- **`nearby_documents` cap** (Item 5). `populateNearbyDocuments` in `@internal/mcp/tools/create_document.go` sorts results alphabetically and caps at `maxNearbyDocuments = 5`. Bounds the response size in large directories.
- **Instruction trim** (partial Item 4). The `REQUIREMENTS TRACKS`, `RESEARCH GATE`, and `WORKFLOW PROMPTS` sections were cut from `mcpServerInstructions` when the MCP track prompts were removed — about 1,900 bytes. `REQUIREMENTS LAYERS` stays, because the `add_relation` description refers to it. See the ADR on removing the MCP track prompts.
- **Prompts capability removed.** The server no longer declares prompts, so the prompt list is not part of session overhead at all.
- **Bounded session-start context.** The recap is now capped at 24 document lines regardless of corpus size, so the session-context component no longer scales with the repository. See the SessionStart context output contract.

### Not shipped (deferred)

- **Item 1 (full)** — Removing the type list from `create_document` entirely and pointing to system instructions. The compressed version was chosen instead because tool-schema-weighted models still benefit from in-schema type names. Full deduplication remains a candidate if A/B testing shows the compressed list is also expendable.
- **Item 2 — Remove `filename` + `slug` from `list_documents` response.** Both fields are still on the document model (`docs.Document` in `@internal/docs/document.go`, aliased as `LocalDocument`). Mechanically derivable from `path`. ~430 tokens / 50 docs.
- **Item 3 — Strip frontmatter from `content` in `get_document` response.** Still returned verbatim. Contract change — needs a migration plan (or a `raw` opt-in parameter) before shipping. ~10–14 tokens/call.
- **Item 4 (remainder)** — Merging `TYPE SELECTION RULES`, `REQUIREMENTS LAYERS`, and `WHEN TO CREATE` into one compact reference table. ~150–200 tokens.

## Value

- Lower per-request cost for every user interacting via MCP (tokens are paid on every LLM call)
- Faster time-to-first-token (smaller context = less prefill)
- More headroom in context window for actual user work
- No degradation in document quality or type selection accuracy

## Current Token Budget (pre-`05d1af3` baseline)

| Component | Tokens | % |
|---|---|---|
| System instructions (`mcpServerInstructions`) | ~2,900 | 47% |
| 8 tool schemas combined | ~2,100 | 34% |
| → `create_document` alone | ~920 | 15% |
| Session-start context (doc list + relations) | ~600 | 10% |
| **Total baseline** | **~5,600** | **100%** |

The baseline is stale in two directions and has not been re-measured: the server now exposes eleven
tools rather than eight, and the instruction and session-context components both shrank. Re-baseline
before deciding whether the remaining items are worth pursuing.

### System Prompt Breakdown (baseline)

| Section | Tokens | % of instructions |
|---|---|---|
| TYPE SELECTION RULES (20 disambiguation pairs) | ~970 | 34% |
| Requirements layers + tracks | ~440 | 15% |
| WHEN TO CREATE (18 bullets) | ~370 | 13% |
| Everything else (workflow, examples, status) | ~1,120 | 38% |

The tracks half of row 2 is gone; layers remain.

### Generation Impact (output tokens)

| Scenario | Input overhead | Avg output/doc | 10 documents |
|---|---|---|---|
| Current detailed tool descriptions | ~6,000 | ~1,200 | ~18,000 |
| Minimal tool descriptions | ~200 | ~900 | ~9,200 |
| Delta | +5,800 | +300/doc | +8,800 (+95%) |

Minimal descriptions yield 25–40% error rate (wrong type, missing sections). The detailed form is a deliberate trade-off: roughly twice the tokens for correctness.

## Possible Implementation

### 1. Deduplicate `create_document` description (~500–700 tokens saved) — *partial*
The tool description re-lists all 18 types with required sections (~700 tokens) that already exist in system instructions. The compressed shipped version keeps a one-line summary per type; a future change could replace it with a single reference: "See system instructions for required sections per type."

### 2. Remove `filename` + `slug` from `list_documents` response (~430 tokens per 50 docs) — *open*
Both are mechanically derivable from `path`. Strip from the JSON response.

### 3. Strip frontmatter from `content` in `get_document` response (~10–14 tokens/call) — *open*
`title` and `status` are already decoded as separate JSON fields. The raw frontmatter inside `content` is redundant.

### 4. Consolidate the remaining disambiguation blocks (~150–200 tokens) — *partial*
The tracks and research-gate blocks are already cut. `TYPE SELECTION RULES`, `REQUIREMENTS LAYERS`, and `WHEN TO CREATE` still cover overlapping ground and could merge into one compact table.

### 5. Cap `nearby_documents` in `create_document` response (up to ~286 tokens saved) — *shipped*
Capped at 5 entries, sorted alphabetically.

## Risks and Constraints

- **Deduplication risk**: Some LLMs weight tool-level descriptions higher than system instructions. Removing type info from `create_document` may reduce accuracy for models that prioritize tool schemas over system prompt. Needs A/B testing. The compressed form is a hedge against this risk.
- **Stripping frontmatter from get_document**: Changes the contract — agents that parse `content` expecting frontmatter will break. Requires a migration path or a `raw` parameter.
- **Consolidating disambiguation rules**: The current verbose format is optimized for LLM comprehension. A compact table may reduce readability for the model. Test with edge cases (brs vs brd, strs vs urd).
- **Instruction cuts must respect cross-references**: `REQUIREMENTS LAYERS` survived the trim because the `add_relation` description points at it. Check for such a reference before removing a section.
