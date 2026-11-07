---
title: "search_documents MCP Tool Contract"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Purpose

This specification defines the contract of the `search_documents` MCP tool (@cli/internal/mcp/tools/search_documents.go, @cli/internal/mcp/tools/response_budget.go). It is normative for the tool and for every MCP client that depends on its response shape. Where the implementation diverges, this document governs until it is amended.

Decided elsewhere: the recall design (tokenized matching, coverage, representation) in @.archcore/mcp/global-recall-guarantees.rfc.md; the byte budget, evidence caps, and envelope order in @.archcore/mcp/read-tool-responses-survive-host-truncation.adr.md; the slug field and separator folding in @.archcore/mcp/search-matches-the-slug-and-folds-separators.adr.md; near misses in @.archcore/mcp/empty-all-words-search-partial-matches.rfc.md.

Out of scope: a persistent path index, semantic or embedding search.

## Definitions

| Term | Definition |
| --- | --- |
| Separator | One of `-`, `_`, `/`, `.`, `@`, `:`. |
| Match token | One whitespace-separated word with its separators folded (§6.1) under `all`/`any`, or the whole query under `exact`. |
| Admitted row | A row that survives the `limit` cut of §8. |
| Near miss | A document that passes the filters and holds at least half, but not all, of the distinct words of an empty all-words query (§13). |
| Byte budget | The ceiling on the serialized response: 40,000 bytes (`searchResponseByteBudget`). |
| Score | `100 × (path-ref maximum specificity + Σ content-token specificities) + capped occurrence count`. Not serialized. |

## Contract Surface

Annotated `ReadOnlyHint: true`. At least one of `path_ref`, `content`, `types`, `status`, or `for_path` MUST be provided.

### Inputs

| Name | Description |
| --- | --- |
| `path_ref` | Path reference to match in document bodies. A leading `@` is stripped. |
| `for_path` | One repository file. Returns the file-context result: the ranked documents that constrain an edit to it, then the general rules (@.archcore/integrations/file-context-resolution.spec.md). An absolute path outside the session root derives the call root from the nearest ancestor holding `.archcore/`. |
| `project` | Absolute root of the project to search, per @.archcore/mcp/call-root-resolution.spec.md. |
| `content` | Case-insensitive word search against `title + slug + body`. Separators fold under `all` and `any`. No stemming, no fuzzy matching. |
| `match` | `all` (default; every word occurs, any order), `any` (at least one word), `exact` (the whole string as one substring). |
| `source` | `local`, `global` (every mounted global source), or a declared global source id. Empty admits every source. |
| `types`, `status`, `mtime_after` | Metadata filters. `mtime_after` takes RFC3339 or `<N>h` / `<N>d`. |
| `sort` | `relevance` (default) or `mtime`. |
| `mode` | `snippets` (default) or `full`, which also inlines each matched body. |
| `limit` | `snippets`: default 50, max 200. `full`: default 3, max 20. Above the cap is clamped; `0` maps to the default. Under `for_path` the file-context caps replace it; §1.2 still applies. |

### Outputs

A JSON object whose keys serialize in this order: `coverage`, `hits`, `truncated`, `index`, `near_misses`, `results`. The summary comes first because a host that stores an oversized result shows the caller only its first bytes.

- `coverage` maps each searched source id to its scanned document count, computed after the `source` scope and before the query filters. An empty `results` next to a populated `coverage` is a verified absence: under `all`, no passing document holds every query word.
- `hits` maps each searched source id to its matched count before the `limit` cut.
- `truncated` is `true` when the byte budget returned fewer rows, `index` entries, or near misses than were admitted.
- `index` lists `path`, `title`, and `source_id` of the admitted rows in §7 order, inside its own byte ceiling (§8.10).
- `near_misses` lists up to 3 documents with the words each lacks (§13). Present only under §13.1.
- `project` is the absolute root the call resolved (@.archcore/mcp/call-root-resolution.spec.md).
- `omitted` appears only under `for_path` when a cap cut rows, with `more_matched` and `more_general` naming the call that returns them.

Each `searchResult` row carries `path`, `title`, `type`, `status`, `mtime`, `tags`, `source_id`, `source_kind` (`local` or `global`), `source_root` (global rows), `matches`, `reason` (only under `for_path`), `matches_total`, `body`, `body_truncated`, `body_bytes`, and the relation arrays with their `_total` fields. Field names and omit rules follow the struct in @cli/internal/mcp/tools/search_documents.go. `matches` is always an array. Each `Match` has `kind` (`path_ref_explicit`, `path_ref_mention`, `content`), `ref`, `specificity`, and a UTF-8-safe `excerpt` of about 120 characters.

## Normative Behavior

### §1 Filter and parameter validation

1. The handler MUST reject a call with no filter: `specify at least one filter (path_ref, content, types, status, or for_path)`.
2. The handler MUST reject `limit < 0`.
3. The handler MUST treat `limit == 0` as omitted.
4. The handler MUST clamp `limit` above the mode cap without an error.
5. The handler MUST accept RFC3339 and `<N>h` / `<N>d` for `mtime_after` and reject anything else with `invalid mtime_after: <reason>`.
6. The handler MUST map any `mode` other than `full` to `snippets`.
7. The handler MUST map any `match` other than `exact` or `any` to `all`.
8. The handler MUST resolve `source` as: empty admits all; `local` and `global` admit by `source_kind`; a declared global source id admits its documents.
9. WHEN `source` carries any other value, the handler MUST reject the call with `invalid source "<value>" (valid: "local", "global", or a declared global source id)`. The declared-id check is an advisory, fail-open read.
10. WHEN `match` is `all` or `any` and `content` tokenizes to zero words, the handler MUST reject the call with `content must contain at least one word`.
11. WHEN `for_path` is set together with `path_ref` or `content`, the handler MUST reject the call.
12. WHEN `for_path` is set, the handler MUST order and cap `results` as the file-context spec defines and apply `source`, `types`, and `status` as filters on its rows.
13. WHEN `project` is set, the handler MUST search that root and none other.
14. WHEN `for_path` is absolute and `project` is absent, the handler MUST search the nearest ancestor of `for_path` that holds `.archcore/`.
15. WHEN `for_path` is absolute and `project` is set, the handler MUST reject the call if `for_path` lies outside `project`.

### §2 Document loading

A pure-metadata `snippets` query does not retain bodies; any `path_ref`, `content`, or `full` query does. The I/O cost is the same in both cases.

### §3 Manifest loading

1. A missing `.sync-state.json` yields an empty manifest and empty relation arrays.
2. IF the manifest is present but invalid, THEN the handler MUST fail with `loading manifest: <reason>`. A silent empty-relations fallback would hide real graph state.

### §4 Source scope and metadata filters (AND semantics)

1. `source` applies first. A document admitted here MUST be counted in `coverage` under its `source_id` whatever the later filters do.
2. `types`, `status`, and `mtime_after` (strictly after) then exclude documents that fail them.

### §5 Path-reference extraction

1. The handler MUST extract explicit references with `@[\w./\-_]+` and bare candidates with `[\w\-_]+/[\w\-_./]+`.
2. The handler MUST drop a bare candidate contained in an explicit one.
3. The handler MUST keep a bare candidate only when it ends with `/`, holds at least two `/`, or ends in a source extension.
4. Specificity is the number of left-aligned `/`-segments the candidate shares with the normalized filter. Specificity 0 MUST NOT match.
5. The handler MUST order hits by specificity DESC, explicit before mention, then offset ASC.
6. The handler MUST emit at most 5 hits (`searchMatchCap`).
7. The score MUST take the maximum specificity across a document's hits, not their sum. The cap of §5.6 MUST NOT change the score.
8. WHEN §5.6 or §6.6 cuts hits, the handler MUST set `matches_total` to the full count.

### §6 Content matching

1. The handler MUST lowercase the query and, under `all` and `any`, split it on whitespace and fold each separator into one space. Under `exact`, the whole query is one token and nothing folds.
2. Under `all` and `any`, the handler MUST compare each token against the title, the slug, and the body folded the same way.
3. Tiers: a title hit is 3; a slug hit is 3; a heading-line hit is 2; any other body hit is 1; absent is 0. The slug is a match field in every mode.
4. Under `all` and `exact`, a tier-0 token MUST exclude the document. Under `any`, the document matches when at least one token hits.
5. Each hitting token MUST emit a `content` match with an excerpt around its first occurrence.
6. WHEN a document has more than 5 content matches, the handler MUST keep the 5 of highest specificity.
7. The handler MUST count token occurrences across title and body, capped at 20 (`contentFreqCap`), as the low-order score component.
8. WHILE `match` is `all` and no document has matched, the handler MUST count the distinct query words at tier 1 or above per document.

### §7 Ranking

1. `sort=relevance` MUST order by: score DESC; type priority ASC (`rule=1, adr=2, spec=3, cpat=4, guide=5, plan=6, idea=7, other=100`); effective mtime DESC; `path` ASC. A global document's effective mtime is the zero time, because a vendored clone date is not a relevance signal, so on a full tie a local document ranks first (@.archcore/globals/local-overrides-source.rule.md).
2. `sort=mtime` MUST order purely by `mtime` DESC.
3. The sort MUST be stable and deterministic across runs.

### §8 Truncation and per-source representation

1. After sorting, the handler MUST cut the results to at most `limit` rows.
2. WHEN the cut removes every row of a source that has a match, the handler MUST swap that source's top row over the lowest-ranked page row whose source keeps more than one row.
3. Sources claim their swap in rank order of their top row. WHEN every page row is its source's last, the handler MUST stop swapping.
4. The handler MUST re-sort the page with the §7 keys after the swaps.
5. The handler MUST keep the serialized response at or under the byte budget.
6. WHEN the rows do not fit with each body at its floor (§12.4), the handler MUST remove rows from the tail and set `truncated`.
7. After each removal, the handler MUST re-apply §8.2–§8.4.
8. The handler MUST keep at least one row. A single row whose fields other than `body` exceed the budget is the only response that may exceed it.
9. `index` MUST list the admitted rows in §7 order, including rows §8.6 removed from `results`.
10. WHEN the index exceeds 20,000 bytes (`searchIndexByteBudget`), the handler MUST drop entries from the tail, keep one, and set `truncated`.

### §9 Excerpts

Excerpts are 120-character windows snapped to rune boundaries, so they are valid UTF-8, with whitespace runs collapsed and `...` marking a cut.

### §10 Relations enrichment

1. The handler MUST populate both relation arrays from the manifest and serialize an empty array as `[]`, never `null`.
2. The handler MUST order each array by `path` ASC, then `type` ASC, and emit at most 5 per array (`searchRelationCap`).
3. WHEN §10.2 cuts an array, the handler MUST set its `_total` field. `get_document` returns the full graph.

### §11 Response shape invariants

`results` and `index` are empty arrays, never `null`, when nothing matches, and `coverage` still carries the scanned counts. `coverage` and `hits` hold one entry per source the scope admits. Every row carries `source_id` and `source_kind`.

### §12 Full-mode body

1. In `full` mode each row MUST carry `body` with the frontmatter stripped. In `snippets` mode `body` MUST be omitted.
2. Full mode MUST NOT change matching, ranking, or representation.
3. WHEN the bodies do not fit the budget, the handler MUST shorten them by max-min fair share: a body smaller than an equal share arrives whole.
4. The handler MUST NOT remove a row to make room for a body while every body can keep a share of 1,024 encoded bytes (`searchBodyFloorBytes`).
5. A shortened `body` MUST be a rune-aligned prefix and SHOULD end at the last line break in the second half of its share.
6. A row with a shortened `body` MUST carry `body_truncated: true` and `body_bytes`. A caller MUST read the whole document through `get_document` before it writes through `update_document`.

### §13 Near misses on an empty all-words response

In this section, n is the number of distinct folded query words.

1. WHEN `match` is `all`, `content` holds at least two distinct folded words, and every `hits` value is 0, the handler MUST compute near misses.
2. The handler MUST admit only a document that passes §4 and §5 and holds ⌈n/2⌉ or more distinct words.
3. The handler MUST order near misses by the §6.8 count DESC, then the §7.1 keys, whatever `sort` holds.
4. The handler MUST keep at most 3 (`searchNearMissCap`) and apply the §8.2–§8.4 representation swap to them.
5. WHEN §13.1 holds and none qualifies, the handler MUST serialize `near_misses` as `[]`. Otherwise it MUST omit the field.
6. Each near miss MUST carry only `path`, `title`, `source_id`, and `missing`. It MUST NOT count toward `hits` or `index`.
7. IF the response exceeds the budget, THEN the handler MUST remove near misses from the tail and set `truncated`.

## Constraints & Invariants

| Constraint | Value | Rationale |
| --- | --- | --- |
| Response byte budget | 40,000 bytes | One host stored results from about 50,000 characters upward and refused larger ones; bytes bound characters from above. |
| Index ceiling | 20,000 bytes | 200 admitted rows with long titles otherwise fill the response by themselves (44,198 bytes measured). |
| Match cap | 5 per filter | One document cited one path 39 times; evidence was 60–70% of an overflowed response. |
| Relation cap | 5 per direction | A hub document's row otherwise grows with its degree. |
| Path-ref score | maximum specificity | Repeating a path is not a relevance signal. |
| Occurrence cap | 20 | A term-stuffed body must not outrank a structural hit. |
| Near-miss threshold | ⌈n/2⌉ of n words | Equals "every word except one" at two and three words and still admits long queries. |

- The handler is read-only: no filesystem writes, no manifest mutations.
- Two identical calls against an unchanged tree MUST produce identical output.
- The response MUST NOT exceed the byte budget, except the single-row case of §8.8.
- `near_misses` appears only beside an empty `results`.
- One document yields at most one row; several candidates collapse into `matches`.
- WHEN a source has a matching document, the truncated page MUST carry at least that source's top match, within the §8.3 slot rule.
- Consumers MUST tell local from global by `source_id` and `source_kind`, not by path.

## Failure Behavior

Each failure below is an MCP error that holds no absolute path: a missing filter; a negative `limit`; an invalid `mtime_after`; an unknown `source`; a zero-word `content` under `all`/`any`; a present-but-invalid manifest (`loading manifest: <reason>`); an I/O error during scan (`scanning documents: <I/O class>`). A missing manifest is not an error. Malformed frontmatter indexes the document with an empty title and status. See @.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md.

## Conformance

An implementation conforms when it satisfies every MUST in §§1–13, the invariants, and the failure rows, and passes @cli/internal/mcp/tools/search_documents_test.go, @cli/internal/mcp/tools/search_recall_test.go, @cli/internal/mcp/tools/response_budget_test.go, and @cli/internal/mcp/tools/search_near_misses_spec_test.go.

Example: given `content: "plugin compatibility"`, a document that holds both words in any order and any distance matches. `match="exact"` returns zero rows because the phrase appears in no title.
