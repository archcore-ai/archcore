---
title: "Optimize MCP Tool Definitions and System Prompt for Token Efficiency"
status: draft
tags:
  - "cli"
  - "component:cli"
---

## Idea

Reduce token consumption of the MCP server by deduplicating tool descriptions, trimming redundant response fields, and consolidating disambiguation rules — without sacrificing document classification accuracy.

The original token baseline is stale and has not been re-measured. Re-baseline before deciding whether the remaining items are worth pursuing.

## Status

**Partially shipped.** The safe subset landed: a compressed `create_document` description, a `nearby_documents` cap of 5, the removal of the prompts capability, a trimmed server instruction text, and a session recap capped at 24 document lines. The open items are response-shape changes and one description merge.

## Open Items

- **Item 1 (full)**: remove the type list from `create_document` and point to system instructions. The compressed version was kept because tool-schema-weighted models still benefit from in-schema type names. Revisit only if A/B testing shows the compressed list is expendable.
- **Item 2**: remove `filename` and `slug` from the `list_documents` response. Both are still on the document model (`docs.Document` in `@cli/internal/docs/document.go`) and derivable from `path`.
- **Item 3**: strip frontmatter from `content` in the `get_document` response. It is still returned verbatim. This is a contract change and needs a migration plan or a `raw` opt-in parameter.
- **Item 4**: merge the remaining disambiguation blocks (`TYPE SELECTION RULES`, `WHEN TO CREATE`) into one compact table.

## Value

- Lower per-request cost for every user interacting via MCP, since tokens are paid on every LLM call
- Faster time-to-first-token
- More headroom in the context window for user work

## Generation Impact

Minimal tool descriptions yielded a 25–40% error rate (wrong type, missing sections) in an earlier measurement. The detailed form is a deliberate trade-off: roughly twice the tokens for correctness.

## Risks and Constraints

- **Deduplication risk**: Some LLMs weight tool-level descriptions higher than system instructions. Removing type info from `create_document` may reduce accuracy for models that prioritize tool schemas over system prompt. Needs A/B testing. The compressed form is a hedge against this risk.
- **Stripping frontmatter from get_document**: Changes the contract — agents that parse `content` expecting frontmatter will break. Requires a migration path or a `raw` parameter.
- **Consolidating disambiguation rules**: The current verbose format is optimized for LLM comprehension. A compact table may reduce readability for the model. Test with edge cases (brs vs brd, strs vs urd).
- **Instruction cuts must respect cross-references**: Check whether a tool description points at a section before removing it.
