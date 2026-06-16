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

**Partially shipped** in commit `05d1af3 fix: safe token optimizations in MCP tools` (2026-05-12). The "safe" subset — changes that do not alter the response contract or risk type-selection accuracy — has landed. The riskier items (response-shape changes and full system-prompt consolidation) remain open and are tracked below.

### Shipped (safe)

- **Compressed `create_document` description** (partial Item 1). The 18 document types are still enumerated in the tool description, but each entry is now one or two lines instead of a full multi-line section list. Section detail is delegated to the auto-generated template. The `tags` and `content` parameter descriptions were also tightened, with longer reference text pushed to server instructions. Net effect: roughly half the tokens of the original create_document schema, without removing type guidance.
- **`nearby_documents` cap** (Item 5). `populateNearbyDocuments` in `internal/mcp/tools/create_document.go` now sorts results alphabetically and caps at `maxNearbyDocuments = 5`. Bounds the response size in large directories.

### Not shipped (deferred)

- **Item 1 (full)** — Removing the type list from `create_document` entirely and pointing to system instructions. The compressed version was chosen instead because tool-schema-weighted models still benefit from in-schema type names. Full deduplication remains a candidate if A/B testing shows the compressed list is also expendable.
- **Item 2 — Remove `filename` + `slug` from `list_documents` response.** Both fields are still in `LocalDocument` (`internal/mcp/tools/common.go`). Mechanically derivable from `path`. ~430 tokens / 50 docs.
- **Item 3 — Strip frontmatter from `content` in `get_document` response.** Still returned verbatim. Contract change — needs a migration plan (or a `raw` opt-in parameter) before shipping. ~10–14 tokens/call.
- **Item 4 — Consolidate requirements tracks + layers + type selection in `mcpServerInstructions`.** `internal/mcp/server.go` was not touched in `05d1af3`. ~150–200 tokens.

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

Post-`05d1af3` figures have not been re-measured. Re-baseline before deciding whether the remaining items are worth pursuing.

### System Prompt Breakdown

| Section | Tokens | % of instructions |
|---|---|---|
| TYPE SELECTION RULES (20 disambiguation pairs) | ~970 | 34% |
| Requirements layers + tracks | ~440 | 15% |
| WHEN TO CREATE (18 bullets) | ~370 | 13% |
| Everything else (workflow, examples, status) | ~1,120 | 38% |

### Generation Impact (output tokens)

| Scenario | Input overhead | Avg output/doc | 10 documents |
|---|---|---|---|
| Current detailed prompts | ~6,000 | ~1,200 | ~18,000 |
| Minimal prompts | ~200 | ~900 | ~9,200 |
| Delta | +5,800 | +300/doc | +8,800 (+95%) |

Minimal prompts yield 25–40% error rate (wrong type, missing sections). Current prompts are a deliberate trade-off: ~2x tokens for correctness.

## Possible Implementation

### 1. Deduplicate `create_document` description (~500–700 tokens saved) — *partial: compressed in `05d1af3`*
The tool description re-lists all 18 types with required sections (~700 tokens) that already exist in system instructions. The compressed shipped version keeps a one-line summary per type; a future change could replace with a single reference: "See system instructions for required sections per type."

### 2. Remove `filename` + `slug` from `list_documents` response (~430 tokens per 50 docs) — *open*
Both are mechanically derivable from `path`. Strip from JSON response.

### 3. Strip frontmatter from `content` in `get_document` response (~10–14 tokens/call) — *open*
`title` and `status` are already decoded as separate JSON fields. The raw frontmatter inside `content` is redundant.

### 4. Consolidate requirements tracks + layers + type selection (~150–200 tokens) — *open*
Three overlapping blocks cover similar disambiguation ground. Merge into a single compact reference table.

### 5. Cap `nearby_documents` in `create_document` response (up to ~286 tokens saved) — *shipped in `05d1af3`*
Capped at 5 entries, sorted alphabetically.

## Risks and Constraints

- **Deduplication risk**: Some LLMs weight tool-level descriptions higher than system instructions. Removing type info from `create_document` may reduce accuracy for models that prioritize tool schemas over system prompt. Needs A/B testing. The compressed form shipped in `05d1af3` is a hedge against this risk.
- **Stripping frontmatter from get_document**: Changes the contract — agents that parse `content` expecting frontmatter will break. Requires migration path or a `raw` parameter.
- **Consolidating disambiguation rules**: The current verbose format is optimized for LLM comprehension. A compact table may reduce readability for the model. Test with edge cases (brs vs brd, strs vs urd).
- **Session-start context scales with repo size**: The document list and relations summary grow linearly. This analysis covers the fixed prompt; the scaling overhead is a separate concern.
