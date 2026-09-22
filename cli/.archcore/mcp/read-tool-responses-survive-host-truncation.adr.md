---
title: "Read-Tool Responses Stay Under a Byte Budget and Lead With Their Summary"
status: draft
tags:
  - "globals"
  - "mcp"
  - "performance"
---

## Context

On 2026-09-18 the call `search_documents` with one compound package name as `content`, `match: "exact"`, `mode: "full"`, and `limit: 6`, in a project that mounts one global source, returned 70,058 bytes (50,775 characters); Claude Code stored the result in a file and showed the agent a 2 KB preview that held one local row, while the five global rows began at character 11,237 and `coverage` sat in the last 37 characters, so the agent answered without any global content. A scan of 909 Claude Code transcripts on the same machine found 43 host overflows in 590 `search_documents` and `list_documents` calls: 9 previews, and 34 errors (`exceeds maximum allowed tokens`) that carry no result content; the largest inline result was 48,211 characters and the smallest stored one 50.5 KB. `HandleSearchDocuments` bounds the row count and nothing else (`@internal/mcp/tools/search_documents.go`): `body` is 93% of a full-mode overflow, and `matches` (up to 39 per document) plus relation arrays are 80–96% of a `path_ref` overflow. Clause 1 of the bounded-output rule already requires a named ceiling on an MCP tool response.

## Decision

Every `search_documents` and `list_documents` response stays at or under a named byte budget — `searchResponseByteBudget` = 40,000 bytes — by capping `matches` and each relation array at 5 entries per row with a total beside them, by shortening full-mode bodies under a max-min fair share with `body_truncated` and `body_bytes` on the row, and by serializing `coverage`, `hits`, and `index` ahead of `results`.

## Alternatives Considered

1. Return an MCP error when the response exceeds the budget — rejected because the host already does this, and in 27 of 29 recorded overflowed search calls the agent never opened the stored output; an error delivers zero rows.
2. Drop tail rows until the response fits — rejected because the dropped rows in the incident were the five global rows, and the per-source representation invariant of the search contract would stop holding.
3. Lower the row limits only (full mode default 3, maximum 20) — ruled out because the row count does not bound bytes: the default `mode=full` call on the same corpus returned 40,928 bytes from three rows, and one global body alone is 22,598 bytes.
4. Teach the agent through instructions to open the stored output — rejected as the sole fix because the error format shows the agent no evidence that a global matched, so the instruction has nothing to trigger on.
5. A budget counted in tokens — ruled out because a local binary carries no tokenizer; bytes bound characters from above and follow the token count of a two-byte script closer than characters do (70,058 bytes against 50,775 characters in the incident).
6. Paged bodies through MCP resource links — deferred because host support for resource links in tool results is unverified [assumption].

## Consequences

Positive:

- The replayed incident call returns 39,564 bytes instead of 70,058 and keeps 6 of 6 rows, 5 of them with a shortened body; its first 2,048 bytes carry `hits` for both sources and all six `index` entries (measured 2026-09-18 on the branch build).
- A recorded `path_ref` overflow in this repository falls from 130,138 to 39,313 bytes. The caps alone reach 90,979 bytes, because a bare row weighs about 1.8 KB; the budget then keeps 18 of the 50 admitted rows in `results`, and `index` still lists all 50.
- A row is never removed to make room for a body; tail rows leave `results` only when bare rows alone exceed the budget. The envelope then carries `truncated: true`, `index` keeps every admitted row, and every matching source keeps its top row.
- The static tool description shrinks from 2,342 to about 1,615 characters [expected].

Negative:

- `body` in `mode=full` stops being the whole document in every case, which narrows §12.1 of the search contract and the 2026-06 addendum of the matching-primitive ADR ("unmodified body").
- A consumer that writes a shortened `body` back through `update_document` would delete the rest of the document; `body_truncated` and `body_bytes` are the only guard, and the tool description names `get_document` as the source for a write.
- `matches` no longer lists every path-reference hit; §5.6 and §5.7 of the search contract change, and `matches_total` carries the count.
- With `limit=3` each body receives about 12,500 bytes; in this repository the p90 body is 11.7 KB and the largest 39 KB, so about 10% of bodies arrive shortened.
- `index` costs about 140 bytes per row, about 7 KB at 50 rows, inside the budget [expected].
- The plugin teaches `hits`, `index`, and `body_truncated` as "read them when present" and adds no `cli-gte` gate; this departs from the version-skew obligation the recall RFC recorded, on the ground that the fields are additive on a read response.

## Superseded when

- A supported host documents a tool-result limit below 40,000 bytes, or a transcript scan after release shows more than 1% of read-tool calls overflowing.
- MCP hosts render resource links or paged tool results, so a body can arrive outside the tool result.
- `get_document` gains its own ceiling; it has none today, and 4 of the 13 recorded previews came from it.
