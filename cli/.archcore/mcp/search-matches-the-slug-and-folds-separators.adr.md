---
title: "search_documents Matches the Slug and Folds Separators in Content Words"
status: draft
tags:
  - "globals"
  - "mcp"
---

## Context

On 2026-09-18, against a project of 276 documents (239 local, 37 global), a query written as the repository name of one package (`acme-id-sdk` in this record) returned 9 rows and missed the local document whose file name is that same word, because its body spells the package `@acme-id/sdk`; the package spelling returned 18 rows. `scoreContent` read the title and the body only, and the handler split the query on whitespace only (`@internal/mcp/tools/search_documents.go`), so two spellings of one name reached two document sets and the file name never counted. The agent in that session found the missed document through `ls`, not through the search.

## Decision

`search_documents` reads the document slug as a match field at specificity 3 in every match mode, and under `match=all` and `match=any` it folds the separators `-`, `_`, `/`, `.`, `@`, and `:` to one space, byte for byte, in both the query words and the searched text.

## Alternatives Considered

1. Split a compound word into separate tokens (`acme`, `id`, `sdk`) — rejected because the score sums per-token specificity, so one compound would score three times, and two-letter tokens such as `id` match unrelated documents.
2. Renumber the tiers to 4/3/2/1 so the slug gets its own value — rejected because `specificity` is a wire field and its values balance content hits against `path_ref` segment counts.
3. Directory segments as a match field — rejected because a query such as `mcp` or `globals` would match every document in that directory or mount.
4. Plural and stem folding — deferred, as the recall RFC already deferred it, because it risks folding code identifiers [assumption].

## Consequences

Positive:

- `acme-id-sdk` and `@acme-id/sdk` return one document set: 19 rows on the measured corpus, byte-identical responses, where the sets held 9 and 18 rows before. The missed local document ranks fourth. The two-word form `acme-id sdk` stays a wider all-words query (22 rows), because its words may sit apart.
- A document whose file name equals the query surfaces even when its body uses another spelling; `match=exact` stays literal and gains the slug field alone.
- Folding replaces one byte with one byte, so excerpt offsets stay valid, and the token count, the score formula, and the number of `matches` entries do not change.

Negative:

- A short word that hits a slug ranks as high as a title hit.
- A folded word now matches prose where the parts are adjacent: `search_documents` also matches "search documents".
- Folding every searched text cost +22% time and twice the bytes per call in `BenchmarkReadToolsScaling` (search snippets, N=10000). The shipped form folds only when a query word is compound, since a plain word matches the same bytes in either text: +4.5% snippets and +7% full against `7a0150f`, allocations at parity. A compound query still pays one folded copy of each body.
- `TestHandleSearchDocuments_ContentBodyHit` named its fixture `money.rule.md` and queried `money`; the fixture is renamed so the test still pins a body hit (`@internal/mcp/tools/search_documents_test.go`).
- The wire `ref` of a compound word is the folded word, not the typed one.

## Superseded when

- Semantic retrieval ships for the local corpus, which removes the vocabulary problem at its source.
- A transcript scan shows slug-only hits in the top three rows of more than 5% of content searches where the agent then ignores that row.
