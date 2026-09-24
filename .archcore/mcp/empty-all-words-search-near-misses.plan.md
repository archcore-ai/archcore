---
title: "Near Misses for Empty All-Words Searches — Contract, Matching, Wording, Verification"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "mcp"
---

## Goal

Ship the near-miss data and the corrected absence wording of `search_documents` in one CLI and plugin release, after the maintainer accepts the near-misses RFC and its addendum to the matching-primitive ADR.

## Tasks

### Phase 1 — Accept the design

1. Get the maintainer's acceptance of the near-misses RFC — @.archcore/mcp/empty-all-words-search-partial-matches.rfc.md.
2. Append the RFC's addendum text to the matching-primitive ADR — @.archcore/mcp/search-documents-matching-not-presentation.adr.md.
3. Amend the Outputs section, §6, §7, §8, and §11.1 of the search contract per the RFC — @.archcore/mcp/search-documents.spec.md.

### Phase 2 — Matching

4. Count distinct folded query words per document inside the existing matching loop — @cli/internal/mcp/tools/search_documents.go.
5. Collect documents that pass every filter and match at least half of the distinct words — @cli/internal/mcp/tools/search_documents.go.
6. Declare the near-miss cap as a named constant whose doc comment names the budget it protects — @cli/internal/mcp/tools/search_documents.go.
7. Build rows with `path`, `title`, `source_id`, and `missing`; order them with a stable tie key, then cut — @cli/internal/mcp/tools/search_documents.go.
8. Apply `ensureSourceRepresentation` to the near-miss rows — @cli/internal/mcp/tools/search_documents.go.
9. Serialize `near_misses` between `index` and `results`; emit `[]` when the trigger holds and nothing qualifies — @cli/internal/mcp/tools/search_documents.go.

### Phase 3 — Wording

10. Replace the absence sentence in `searchDocumentsDescription`; name `near_misses` and `missing` — @cli/internal/mcp/tools/search_documents.go.
11. Move the all-words sentence from the GLOBAL SOURCES paragraph into the base instructions — @cli/internal/mcp/server.go.
12. Name `near_misses` as the first step of the empty-result retry ladder — @plugin/plugins/archcore/skills/_shared/globals.md.
13. Mirror that step in @plugin/plugins/archcore/agents/archcore-assistant.md, @plugin/plugins/archcore/agents/archcore-assistant.toml, and @plugin/plugins/archcore/copilot-agents/archcore-assistant.agent.md.

### Phase 4 — Tests and measurement

14. Add tests for the trigger, half rule, deduplication, compound and Cyrillic words, order, presence, and filters — @cli/internal/mcp/tools/search_recall_test.go.
15. Update `TestSearchDocuments_EnvelopeKeyOrder` and the description test to the new key order and sentence — @cli/internal/mcp/tools/search_envelope_spec_test.go.
16. Add the empty two-word case `lorem zephyrite` to `BenchmarkReadToolsScaling` — @cli/internal/mcp/tools/scaling_bench_test.go.
17. Record the benchmark before and after the change — @cli/internal/mcp/tools/scaling_bench_test.go.

### Phase 5 — Verification and release

18. Run `go test ./...` and `golangci-lint run ./...` in `cli/`.
19. Replay the failing run-1 conversation ten times with the new response; record each outcome.
20. Rerun `review-seam-buried` with the `ponytail-archcore-lean` arm through the recipe-bench launcher.
21. Write the release note for `near_misses`, the narrowed absence wording, and the moved instruction sentence.
22. After the release, add the empty-query stop share to the transcript scan.

No target in this repository was found for tasks 19 to 22. Tasks 19 and 20 run in integration-bench, where experiment `20260923T153932Z-98ef5bbd20` holds the conversation and the recipe-bench launcher lives. Grounding found no release-note file for task 21, and `AGENTS.md` forbids a `CHANGELOG.md`. Task 22 runs in a local transcript-scan script.

## Acceptance Criteria

- Each edit to an accepted document in tasks 2 and 3 carries the maintainer's confirmation.
- `go test ./...` and `golangci-lint run ./...` pass in `cli/`.
- The existing envelope and recall tests for responses with rows pass without edits to their expected output.
- The replay record and the benchmark figures are stored with their sample counts and compared against the PRD metrics before the release.
- The release note names every new wire field and every changed instruction sentence.
- The CLI change and the plugin text ship from the same tag.

## Dependencies

- The RFC is draft. Phase 2 starts after task 1. The ADR and the spec are accepted, and tasks 2 and 3 edit them only with the maintainer's confirmation.
- The truncation plan edits the same description, envelope, and ADR. Its shipped head-first envelope, byte budget, and slug folding are the base of Phase 2. [assumption] Its wording phase has shipped, so task 10 starts from the current `searchDocumentsDescription`.
- `bounded-and-deterministic-output.rule` clauses 1 to 4 and 6 bind the cap, the order, and the contract statement of tasks 3, 6, and 7.
- The plugin reads `near_misses` only when present, so the plugin text stays valid against an older CLI.
- integration-bench and the transcript scan live outside this repository.

## Declared Delta

- Route: `sdd` expert invocation, without route computation. Instruments: intent produced the PRD; contract skipped, because the accepted search contract covers the capability and task 3 amends it; illustrate not engaged, because the surface is an MCP response, not a user-facing flow; decompose produced this plan.
- Δ: creates=[]; modifies=[`search_documents` empty all-words response]; retires=[]; decision=[near-miss design in the draft RFC, plus the ADR addendum]; intent_gap=yes, recorded in the PRD.
- Π: machine — the RFC, the search contract, the code, and the bench runs; user — four answers under the PRD's Clarifications.
- M: stone — the search contract and the matching-primitive ADR are accepted. R: external-contract — external MCP clients parse the response shape.