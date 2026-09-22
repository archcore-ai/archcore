---
title: "Scenario and Journey Vocabulary: Two Types, One Advisory Canon, One Extension"
status: draft
tags:
  - "component:cli"
  - "document-types"
  - "golang"
  - "mcp"
---

## Goal

Deliver the CLI portion of `concepts/scenario-and-journey-types` [global · archcore · read-only] in one release: two document types with templates and MCP exposure, their precision and advisory treatment, and the `.feature` source extension. The linked PRD owns outcomes; two linked specs own behavior.

Grounding used branch `main`, HEAD `bda0242`, clean working tree, on 2026-09-15. The user asked on the same day to implement without committing. Phases 1 to 4 are complete in the working tree, uncommitted; phase 5 is open.

## Declared Delta

| Field | Value |
|---|---|
| `creates` | `scenario-and-journey-types`; `scenario-and-journey-advisory-canon` |
| `modifies` | path-reference detection in `search_documents`: `.feature` joins the source-extension list; verdict `ok` against @.archcore/mcp/search-documents.spec.md, which cites the list by path and stays true |
| `retires` | None |
| `decision` | None; the user settled two open points on 2026-09-15: `scenario` joins the injection allowlist below `spec`, and the body caps become one per-type table |
| `intent_gap` | No; the global RFC and rnd record the intent, both `draft` |
| Π | `machine`: registry, precision canon, advisory engines, the research-vocabulary precedent; `user`: the two settled points |
| M | `stone`: accepted procedure doc, search contract, and naming rule cover the zone; the plugin consumes the registry |
| R | `external-contract`: MCP type enum and plugin compatibility |
| Route | `umbrella`, base L, raised once to XL by `stone` and `external-contract` |
| Instruments | intent, contract twice, verdict on the one `modifies` entry, decompose |

Two capabilities with different consumer sets justify the umbrella: MCP clients rely on the registry; the hook path and the plugin gates rely on the checks. The precedent @.archcore/document-types/research-and-evidence-vocabulary.plan.md folded both into one spec at 75 lines; the F6 checks push this pair past the 120-line cap, so the contract splits by sub-surface.

## Tasks

### Phase 1 — types and MCP surface

1. [x] Register `TypeScenario` and `TypeJourney` with their categories in @cli/templates/templates.go.
2. [x] Add both names to `ValidTypes` in @cli/templates/templates.go.
3. [x] Add `generateJourneyTemplate` and `generateScenarioTemplate` in @cli/templates/templates.go.
4. [x] Wire both generators into `GenerateTemplate` in @cli/templates/templates.go.
5. [x] Add `.feature` to the source-extension list in @cli/templates/source_extensions.go.
6. [x] Extend the type description block in @cli/internal/mcp/tools/create_document.go.
7. [x] Extend the valid-values description in @cli/internal/mcp/tools/list_documents.go.
8. [x] Add the type list rows, the WHEN TO CREATE rows, the three selection rules, the relation conventions, and the status meanings in @cli/internal/mcp/server.go.
9. [x] Cover registration, categories, counts, and template sections in @cli/templates/templates_test.go.
10. [x] Cover `.feature` in @cli/templates/source_extensions_test.go.

### Phase 2 — precision canon

11. [x] Add required sections for both types in @cli/templates/precision.go.
12. [x] Add both types to the prose-profile table in @cli/templates/precision.go.
13. [x] Register Flows and Journeys as step sections and, with Examples, as observation sections in @cli/templates/precision.go.
14. [x] Add the foreign-heading rows in both directions in @cli/templates/precision.go.
15. [x] Add the per-type cap table `MaxBodyLines` fed by `MaxSpecBodyLines` in @cli/templates/precision.go.
16. [x] Collect Given/When/Then lines in Flows, Journeys, and Examples as steps in @cli/internal/advisory/precision.go.
17. [x] Add the Anchors-line check per Flows subsection in @cli/internal/advisory/precision.go.
18. [x] Add the actor-subject check over Flows and Journeys steps in @cli/internal/advisory/precision.go.
19. [x] Route the body-cap check through the per-type table in @cli/internal/advisory/precision.go.
20. [x] Cover every new finding and every non-finding in @cli/internal/advisory/precision_actor_subject_test.go.

### Phase 3 — advisory engines and ranking

21. [x] Add `scenario` to the injection ranking below `spec` and above `guide` in @cli/internal/advisory/code_alignment.go.
22. [x] Pin `scenario` inclusion, its rank, and `journey` exclusion in @cli/internal/advisory/code_alignment_test.go.
23. [x] Pin default search ranking and the `.feature` mention in @cli/internal/mcp/tools/search_documents_test.go.
24. [x] Pin cascade behavior for `scenario depends_on spec` in @cli/cmd/hook_post_tool_use_test.go.
25. [x] Pin restatement over `scenario implements journey` in @cli/internal/advisory/restatement_test.go.

### Phase 4 — integration and documentation

26. [x] Add lifecycle scenarios for both types in @cli/internal/mcp/integration/actor_subject_spec_test.go.
27. [x] Update the type inventory and counts in @.archcore/dir/categories-and-document-types.doc.md.
28. [x] Update the category table and the add-a-type procedure in @.archcore/cli-ui/building-the-cli.doc.md.
29. [x] Update the injected-type sentence and ranking in @.archcore/architecture/advisory-subsystem.doc.md.
30. [x] Update the registry count in @.archcore/dir/free-form-directory-rules.rule.md.
31. [x] Update the type counts and tables in @cli/README.md.
32. [x] Run `go test ./...`, `golangci-lint run ./...`, and `go build -o archcore .` from the repository root.

### Phase 5 — release handoff

33. [ ] Confirm the global RFC reached `accepted` before tagging a release.
34. [ ] Prepare release notes naming the vocabulary version and the older-binary skip behavior.
35. [ ] Record the shipped engine version for the plugin compatibility probe.

## Verification record

On 2026-09-15 the full Go suite passed, `golangci-lint run ./...` reported 0 issues, and `go build` succeeded. A deliberate removal of `.feature` from the extension list failed four tests before restoration. The benchmark corpus in @cli/internal/testsupport/corpus.go gained both types so it still spans both halves of the injection allowlist.

On 2026-09-16 a validation pass closed three parser defects: a fenced block under Flows or Actors read as structure, an Actors row without outer pipes named no actor, and an unnumbered observation line under Flows or Journeys escaped the step checks. A health check then injected 26 faults one at a time in an isolated worktree; four survived until four regression cases were added, and `TestBuildCorpus_SpansBothHalvesOfTheAllowlist` in @cli/internal/advisory/bench_test.go now pins the corpus against the injection ranking. The suite, the linter, and the build passed after every step.

## Acceptance Criteria

1. The two linked contract suites pass against the final implementation commit.
2. A generated `scenario` and a generated `journey` each report only the placeholder-body finding in the hook.
3. The in-process scenarios exercise both types through the registered MCP surface.
4. The full Go suite, the linter, and the build complete without findings.
5. Public inventories match the 23-type registry.
6. Implementation review reconciles all three declared delta entries through `/archcore:review`.

## Dependencies

The accepted document-model decision keeps templates in @cli/templates/ and document operations in @cli/internal/docs/. The procedure for adding a type is recorded in the linked building doc. Phase 1 precedes phase 2, which precedes phase 3; phase 4 verifies all three; phase 5 waits for the RFC status in the global repository.

Older binaries skip the new files and report them in `status` (@cli/internal/docs/scan.go, @cli/cmd/status.go). No data migration is needed. The sync server's acceptance of the two new type strings is not verified here. [assumption] The server accepts any type string the client sends.
