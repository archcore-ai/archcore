---
title: "Host-Truncation-Safe Read Tools: Envelope, Byte Budget, Wording, Recall"
status: draft
tags:
  - "globals"
  - "golang"
  - "integrations"
  - "mcp"
---

## Goal

Make a global match reach the agent even when the host cuts the tool result. On 2026-09-18 a `search_documents` call found five global documents and the agent saw none of them: the host kept 2 KB of a 70,058-byte response. The same overflow hit 43 of 590 recorded `search_documents` and `list_documents` calls. The two decisions this plan carries out are recorded in `read-tool-responses-survive-host-truncation.adr` and `search-matches-the-slug-and-folds-separators.adr`.

Each phase ships alone. Phase 1 gives the agent the source summary inside any preview. Phases 2 and 3 remove the overflow. Phase 4 removes the wording that led the agent into it. Phases 5 and 6 cover the list tool and the recall gap. Phase 7 is a separate release of the plugin repository.

## Tasks

### Phase 1 — Head-first envelope

1. Add a failing test that reads the first 2,048 bytes of a two-source response — `@internal/mcp/tools/search_recall_test.go`.
2. Reorder `searchDocumentsResult` to `coverage`, `hits`, `truncated`, `index`, `results` — `@internal/mcp/tools/search_documents.go`.
3. Count `hits` per source where a row is appended; give every covered source without a match the value 0.
4. Add `searchIndexEntry` with `path`, `title`, and `source_id`, one entry per page row.
5. Amend the Outputs section, §11, and the three examples — `@.archcore/mcp/search-documents.spec.md`.

### Phase 2 — Evidence caps

6. Collect path-reference candidates; order them by specificity, then explicit before mention, then byte offset — `@internal/mcp/tools/search_documents.go`.
7. Keep `searchMatchCap` entries per filter group; build excerpts for kept entries only; emit `matches_total`.
8. Order each relation array by path, then type; keep `searchRelationCap` entries; emit both totals.
9. Update `TestHandleSearchDocuments_PathRefRepetitionIsNotRelevance`, which asserts ten matches — `@internal/mcp/tools/search_documents_test.go`.
10. Amend §5.6, §5.7, §10, and the Constraints table — `@.archcore/mcp/search-documents.spec.md`.

### Phase 3 — Byte budget

11. Add `jsonStringSize` with a table test and a fuzz test against `json.Marshal` — new `@internal/mcp/tools/response_budget.go`.
12. Declare `searchResponseByteBudget` and `searchBodyFloorBytes`; each doc comment names the budget it protects — `@internal/mcp/tools/search_documents.go`.
13. Measure each page row once with an empty body; allocate the bodies by max-min fair share.
14. Cut each body on a rune boundary, at the last newline when it falls in the second half; emit `body_truncated` and `body_bytes`.
15. Shorten the page from the tail when bare rows alone exceed the budget; re-run `ensureSourceRepresentation`; set `truncated`.
16. Add the incident regression test: six bodies of two-byte characters, two sources, `mode=full`, `limit=6` — new `@internal/mcp/tools/response_budget_test.go`.
17. Amend §8.1, §12.1, the `body` Outputs row, Constraints, and Compatibility — `@.archcore/mcp/search-documents.spec.md`.
18. Append a 2026-09 addendum on the shortened body — `@.archcore/mcp/search-documents-matching-not-presentation.adr.md`.

### Phase 4 — Wording

19. Replace `searchDocumentsDescription` and the `content`, `mode`, and `limit` parameter texts; keep the text host-neutral — `@internal/mcp/tools/search_documents.go`.
20. Replace the GLOBAL SOURCES paragraph in `buildInstructions` — `@internal/mcp/server.go`.
21. Amend clause 13 to the new precedence sentence — `@.archcore/globals/session-globals-disclosure.spec.md`.
22. Change `globalsPrecedenceLine` and its test to the amended sentence — `@cmd/hooks_common.go`, `@cmd/hooks_globals_block_test.go`.

### Phase 5 — list_documents

23. Record the `list_documents` contract as a `spec` through `/archcore:document`; no spec covers the tool today.
24. Reorder `listDocumentsResult` to `by_source`, `total`, `offset`, `returned`, `truncated`, `documents` — `@internal/mcp/tools/list_documents.go`.
25. Add `listResponseByteBudget` with a tail cut that `offset` recovers; add both tests — `@internal/mcp/tools/list_interleave_test.go`.

### Phase 6 — Recall

26. Add the slug to `scoreContent` after the title, at specificity 3, in every match mode — `@internal/mcp/tools/search_documents.go`.
27. Fold the six separators in query words and searched text under `all` and `any`; reject a word that folds to nothing.
28. Add the slug, convergence, exact-mode, and separator-only tests; update `TestHandleSearchDocuments_ContentBodyHit` — `@internal/mcp/tools/search_recall_test.go`, `@internal/mcp/tools/search_documents_test.go`.
29. Amend §6 and the "Match token" definition — `@.archcore/mcp/search-documents.spec.md`.
30. Run `BenchmarkReadToolsScaling` before and after the change; record both figures in the second ADR.

### Phase 7 — Plugin (repository `archcore-ai/plugin`, its own release)

31. Add the section "Large or truncated results" before "Reading convention" — `plugins/archcore/skills/_shared/globals.md`.
32. Extend the current-versus-older CLI bullet with `hits`, `index`, and `body_truncated`, read when present.
33. Mirror the retry wording into `agents/archcore-assistant.md`, `agents/archcore-assistant.toml`, and `copilot-agents/archcore-assistant.agent.md`.

### Phase 8 — Verification

34. Run `go test ./...` and `golangci-lint run ./...`.
35. Replay the incident call against the project that reported it; record the byte count and the first 2,048 bytes.
36. Run the five wording scenarios over an extended `examples/07-local-overrides-global/` fixture; record each transcript verdict.
37. Write the release note: shortened `body`, capped `matches` and relations, new envelope fields.

## Acceptance Criteria

- The Phase 1 test and the incident regression test fail at commit `7a0150f` and pass after their phases.
- The replayed incident call returns at most 40,000 bytes, and its first 2,048 bytes carry a `hits` entry for the global source.
- The call `path_ref: "internal/mcp/server.go"`, `limit: 50` in this repository, 129,797 characters in the recorded session, returns under the budget.
- A query that equals a document's slug returns that document when its body spells the name with other separators.
- No existing wire field is renamed or removed; a client that reads `results` and `coverage` parses the new response unchanged.
- `searchDocumentsDescription` is shorter than its 2,342 characters at `7a0150f`.
- `BenchmarkReadToolsScaling` search snippets at N=10000 regresses by less than 15% against `7a0150f` [assumption: the threshold repeats the cost the tokenized scoring was accepted at].
- `go test ./...` and `golangci-lint run ./...` pass.

## Dependencies

- `search-documents.spec`, `session-globals-disclosure.spec`, and the matching-primitive ADR are `accepted`; tasks 5, 10, 17, 18, 21, and 29 edit them and each edit waits for the user's confirmation.
- Task 23 precedes tasks 24 and 25: a change to an uncovered capability gets its covering spec first.
- Phase 3 depends on Phase 2: without the caps, bare rows of a `path_ref` page exceed the budget and task 15 would cut rows the caps would have kept.
- Phase 7 follows a CLI release that contains Phases 1–3; the plugin text stays valid against an older CLI because it reads the new fields only when present.
- `bounded-and-deterministic-output.rule` clauses 1, 2, 3, 4, and 6 bind every new constant and every cut in Phases 2, 3, and 5.
- [assumption] The Claude Code inline limit is near 50,000 characters for the preview path and 25,000 tokens for the error path; the first figure is measured from transcripts, the second is inferred. Other hosts are unmeasured.
- Out of this plan: a ceiling for `get_document`. It has none, 4 of 13 recorded previews came from it, and a shortened body there can reach `update_document`.
- Reported by the code read, not verified, not in scope: §1.5 of the search contract lists `w`, `mo`, and `y` for `mtime_after` while `parseMtimeAfter` may accept fewer units; the `typePriority` values may differ from §7.

## Declared Delta

- Route: `amendment` (size M). Base label S from a non-empty `modifies`; raised one step because the zone is `stone` and R carries `external-contract`.
- Δ: `creates=[]`; `modifies=[search-documents-response, search-documents-matching, list-documents-envelope, globals-precedence-wording]`; `retires=[]`; `decision=[byte budget and head-first envelope, slug field and separator folding]`; `intent_gap=no`.
- Π: `machine` for every need — the stored incident response, the replay on the reporting project (CLI v0.8.5, byte-identical), the scan of 909 transcripts, the code read, and two agent design reports; `user` for the scope, which the request fixed as all of P0–P5.
- M: `stone` — two accepted specs, one accepted ADR, and one accepted rule cover the zone. R: `external-contract` — the MCP wire shape and host-side limits that another project owns.
- Verdicts: `search-documents-response` — `code-wrong` against `bounded-and-deterministic-output.rule` clause 1, and `spec-wrong` for §12.1, §5.6, §5.7, §8.1, and §10, which require the unbounded output. `search-documents-matching` — `spec-wrong` for §6. `globals-precedence-wording` — `spec-wrong` for clause 13 of `session-globals-disclosure.spec`. `list-documents-envelope` — no covering spec; task 23 creates it.
- Progress, 2026-09-18: Phases 1–4 and 6 are implemented on branch `fix/host-truncation-safe-read-tools`; tasks 5, 10, 17, 18, 21, 22, 23, and 29 are done; Phase 7 is implemented on plugin branch `fix/large-result-guidance`. Open: tasks 24, 25, 30 (figures recorded in the second ADR), and Phase 8.
- Unplanned Δ: `index` lists every admitted row while the budget shortens `results`, because the caps alone left a 50-row `path_ref` page at 90,979 bytes. Folding became lazy after the first form regressed the benchmark by 22%. The sample output in the docs repository (`guides/connect-your-agent.mdx`) repeats the old precedence sentence and needs the amended one at release.
- Deviation: sequencing rule 11 asks for the covering `list_documents` spec before the package closes; this plan defers it to task 23 so the package stays on the incident.
