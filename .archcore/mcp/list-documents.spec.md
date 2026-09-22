---
title: "list_documents MCP Tool Contract"
status: draft
tags:
  - "component:cli"
  - "globals"
  - "mcp"
---

## Purpose & Scope

This specification is normative for the `list_documents` MCP tool: its filters, its paging, its interleaved order, its response envelope, and its byte budget. LLM agents, the plugin skills, and any third-party MCP client depend on it to discover documents and to obtain paths for `get_document`.

It does not cover body search (`search-documents.spec`), the source annotation fields of a document row (`global-sources.spec`), or the scan and its cache (`@cli/internal/docs/scan.go`).

Clauses 9, 14, 15, and 16 state behavior that `host-truncation-safe-read-tools.plan` Phase 5 adds; the code at commit `7a0150f` serializes `documents` first and has no byte budget.

## Surface

- Tool: `list_documents`, read-only — `NewListDocumentsTool`, `HandleListDocuments` in `@cli/internal/mcp/tools/list_documents.go`.
- Inputs: `types` (string[]), `category`, `status`, `tags` (string[], OR), `source`, `limit`, `offset`. Every input is optional.
- Output envelope: `listDocumentsResult` in the same file — `by_source`, `total`, `offset`, `returned`, `truncated`, `documents`.
- Row: `LocalDocument`, an alias of `docs.Document` — `@cli/internal/docs/document.go`. A row carries no body.
- Order: `interleaveBySource` in the same file.
- Source scope: `validSourceScope` and `sourceAdmits` in `@cli/internal/mcp/tools/search_documents.go`, shared with `search_documents`.

## Normative Behavior

1. The handler MUST apply the filters `source`, `types`, `category`, `status`, and `tags` with AND semantics.
2. WHEN `types` holds more than one value, the handler MUST admit a document whose type equals any of them.
3. WHEN `tags` holds more than one value, the handler MUST admit a document that carries at least one of them.
4. The handler MUST resolve `source` exactly as `search_documents` does: `local`, `global`, `__global__`, or a declared global source id.
5. WHEN `limit` is `0` or absent, the handler MUST use 100.
6. WHEN `limit` exceeds 500, the handler MUST use 500.
7. WHEN `offset` exceeds the filtered total, the handler MUST return an empty page with `offset` equal to the total.
8. The handler MUST count `by_source` and `total` over the full filtered set, before the page cut.
9. The handler MUST serialize the envelope keys in the order `by_source`, `total`, `offset`, `returned`, `truncated`, `documents`.
10. WHEN the filtered set spans more than one source, the handler MUST place one document of each source first, in scan order of the sources.
11. After those first rows, the handler MUST interleave the sources in proportion to their remaining document counts.
12. WHEN the filtered set holds one source, the handler MUST keep scan order.
13. The handler MUST set `truncated` to `true` exactly when `offset + returned` is less than `total`.
14. The handler MUST keep the serialized response at or under 40,000 bytes (`listResponseByteBudget`).
15. WHEN the page exceeds the byte budget, the handler MUST remove rows from the tail of the page and report the smaller `returned`.
16. The handler MUST keep at least one row of a non-empty page.

## Constraints & Invariants

| Constraint | Value | Rationale |
| --- | --- | --- |
| Default limit | 100 | Covers a typical project in one call. |
| Maximum limit | 500 | Caps the row count a caller can request. |
| Response byte budget | 40,000 bytes | The same host limit that bounds `search_documents`; two recorded listings of 92,660 and 99,041 bytes were stored by the host instead of shown. |

- The handler is read-only: no filesystem write and no manifest mutation.
- Two identical calls against an unchanged tree MUST produce byte-identical JSON.
- `documents` MUST serialize as `[]`, never `null`.
- `offset + returned` MUST be the `offset` of the next page, with or without a byte-budget cut.
- WHEN a source has at least one filtered document, the first page of an unscoped listing MUST carry a row of that source, bounded by the page size.
- A row MUST carry `source_id` and `source_kind`; `global` and `read_only` appear when true.
- A document path MUST begin with `.archcore/` for a local document; a mounted global renders with its declared relative prefix.

## Failure Behavior

1. IF a `types` value is not a registered document type, THEN the handler MUST return the error `invalid type "<value>" (valid: ...)`.
2. IF `category` is not a registered category, THEN the handler MUST return the error `invalid category "<value>" (valid: ...)`.
3. IF `status` is not a registered status, THEN the handler MUST return the error `invalid status "<value>" (valid: ...)`.
4. IF a `tags` value fails tag validation, THEN the handler MUST return the validation message.
5. IF `limit` is negative, THEN the handler MUST return the error `limit must be non-negative`.
6. IF `offset` is negative, THEN the handler MUST return the error `offset must be non-negative`.
7. IF `source` names no known scope, THEN the handler MUST return the error `invalid source "<value>" (valid: "local", "global", or a declared global source id)`.
8. IF the scan fails, THEN the handler MUST return `scanning documents: <I/O class>` with no absolute path.
9. The handler MUST validate every input before it scans.

Every failure above is a tool error result with a nil Go error. Validation failures are not retriable; a scan failure is.

## Conformance

An implementation conforms when it satisfies every MUST above, every invariant, and every failure rule. The acceptance harness is `@cli/internal/mcp/tools/list_documents_test.go` and `@cli/internal/mcp/tools/list_interleave_test.go`.

Given 500 local documents and 500 documents in one global source,
When a caller lists with no arguments,
Then the first 2,048 bytes carry `by_source` for both sources, and the page holds rows of both.
