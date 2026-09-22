---
title: "Recall Guarantees for Global Content in the MCP Read Tools"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Summary

A mounted global source is searchable today only by exact substring, so a query that misses the author's phrasing misses the document. This RFC proposes four changes that make global content reachable from the local project in every case: tokenized all-words matching in `search_documents`, per-source representation on every result page, a coverage envelope that names what was searched, and a `source` filter on both read tools. The changes amend @.archcore/mcp/search-documents.spec.md and add the first paging contract for `list_documents`.

Implemented 2026-08-14: @internal/mcp/tools/search_documents.go, @internal/mcp/tools/list_documents.go, tests in @internal/mcp/tools/search_recall_test.go and @internal/mcp/tools/list_interleave_test.go; the amended contract is in @.archcore/mcp/search-documents.spec.md.

## Motivation

Second user report against globals: the global explained how a subsystem works, the local project held nothing on it, and the agent answered "no data". The first report and the measured page-cut thresholds are in @.archcore/globals/global-discovery-gap.idea.md.

Live reproduction on this repository, 2026-08-14. The topic "plugin/CLI compatibility" is owned by a local rule ("Compatibility Contract Between the Archcore CLI and the Archcore Plugin") and a global rule ("Plugin / CLI Compatibility Across Independent Release Trains"). Four natural phrasings of one question return disjoint sets:

| Query | Local rule | Global rule | Returned instead |
| --- | --- | --- | --- |
| `compatibility contract` | hit (title) | miss | — |
| `plugin compatibility` | miss | miss | one body mention in an unrelated doc |
| `release trains` | miss | hit (title) | — |
| `independent release` | miss | hit (title) | — |

The engine does not skip globals — the scan covers the full mounted corpus (@internal/mcp/tools/search_documents.go). The misses come from four mechanisms:

1. Exact-substring matching: `strings.Contains` over lowercased title and body; "plugin compatibility" does not match "Plugin / CLI Compatibility" (@internal/mcp/tools/search_documents.go `extractContentMatch`, removed with this change).
2. Vocabulary asymmetry: the session recap shows the agent every local title and nothing of the global corpus, so query terms come from local and user vocabulary — the disclosure block narrows this (@.archcore/globals/session-globals-disclosure.spec.md).
3. Page cuts: with 500 or more local documents, `list_documents` returns no global row at any limit (measured in @.archcore/globals/global-discovery-gap.idea.md).
4. Ranking keys unrelated to relevance: binary specificity, type priority, and clone-date mtime (same idea, Measurements).

An empty result is indistinguishable from an unsearched corpus: the bare array carries no record of what was scanned, so the agent cannot tell absence from invisibility.

## Detailed Design

### 1. Tokenized matching (`search_documents`)

- The `content` query splits on whitespace into tokens; a document matches when every token occurs case-insensitively in its title or body — order-independent and gap-tolerant.
- Under this rule "plugin compatibility" matches both rules above: each contains "plugin" and "compatibility".
- A `match` parameter selects `exact` (today's substring), `all` (new default), or `any`. A single-token query behaves identically under `exact` and `all`.
- Scoring replaces the binary 3/1 specificity: per-token field weights (title above heading above body) summed with a capped occurrence count; ties keep type priority; mtime moves last and is ignored for global sources, whose mtime is a clone-date artifact.

### 2. Per-source representation (both tools)

- When a source has at least one match, the default page carries its top match — local rows can no longer evict an entire source. In `mode=full`, one of the three slots is reserved for the top global match when one exists.
- The `list_documents` default page interleaves per-source quotas proportional to source share with a floor of one row; a `by_source` count map in the envelope names what each source holds.

### 3. Coverage envelope (`search_documents`)

- The bare array becomes `{"results": [...], "coverage": {"local": 102, "archcore": 42}}` — each source id mapped to its scanned document count.
- Empty `results` next to a visible `coverage` is a verified absence; today's empty array is unfalsifiable.
- This is a wire-shape break; the amended spec carries the conformance note.

### 4. Source filter (both tools)

- A `source` parameter (`local`, `global`, or a declared id) scopes a call. The instructions layer adds: an empty local result with globals mounted means retry broadened or scoped to `global`.

### Interlock with the disclosure block

@.archcore/globals/session-globals-disclosure.spec.md tells the agent that a global exists and shows its top-level vocabulary; this RFC makes the query side forgiving enough that approximate vocabulary still lands. Disclosure without tokenized matching still demands phrase guessing; tokenized matching without disclosure still demands knowing there is something to search for.

### Version-skew obligation on the plugin side

An older CLI silently ignores the `match` and `source` parameters (the handler reads known argument keys only) and returns the bare array, so a plugin that teaches the agent these capabilities against an old CLI degrades silently — the agent believes it searched with all-words semantics and coverage when it did not. A plugin release that teaches `match`, `source`, `coverage`, or the `GLOBALS` block gates that teaching on the CLI version that shipped them, through the plugin's existing `cli-gte` mechanism. The obligation is plugin-side and tracked in the plugin repository; it is recorded here because this RFC creates it.

## Drawbacks

- The envelope change breaks any consumer parsing the bare array. Verified against the installed plugin 0.7.4 (2026-08-14): no plugin file claims a response shape, and the plugin's `bin/` scripts splice only the session-start hook envelope by anchor — no code parses read-tool output. The remaining exposure is external MCP clients, which are not enumerable; the release notes carry the migration note.
- The `all` default widens result sets for multi-word queries; where today's exact phrase was intended, precision drops — `match: "exact"` remains available.
- A reserved slot can push a higher-scored local row off a full page; bounded to one row per source.
- Tokenized scoring costs one pass per token plus a frequency count instead of one `strings.Index` call. Measured 2026-08-14 against `BenchmarkReadToolsScaling` at baseline 27f7959, benchtime 10x: search snippets 54.3 ms → 61.8 ms at N=10000 (+13%), 15.6 ms → 17.7 ms at N=3000; `list_documents` at parity after the single-source fast path (2.59 ms → 2.52 ms at N=1000). The scan still dominates the call (@.archcore/mcp/read-path-scan-performance.idea.md); the +13% buys the heading tier and the frequency signal.

## Alternatives

- Semantic retrieval (embeddings, GraphRAG): the real fix for vocabulary and cross-language queries, out of local-binary scope; tracked in @.archcore/sync/falkordb-graphrag-hybrid-indexing.idea.md. Tokenized matching is the step that fits a local CLI.
- Instructions-only (teach the agent to retry with synonyms): keeps recall probabilistic; rejected as the sole fix.
- OR-matching as default: highest recall, noisy pages; kept as opt-in `match: "any"`.
- English plural folding ("train" / "trains"): helps morphology misses, risks folding code identifiers; deferred until tokenized matching proves insufficient. [assumption]