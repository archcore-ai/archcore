---
title: "Research and Evidence Vocabulary: Two Types, Three Relations, One Defect Fix"
status: accepted
tags:
  - "document-types"
  - "golang"
  - "mcp"
  - "relations"
---

## Goal

Deliver the CLI portion of the accepted research vocabulary in one release. This plan covers implementation and verification; the linked PRD owns outcomes, and three linked specs own behavior. Product scope follows `concepts/research-and-evidence-types` and `architecture/store-not-method-engine` [global · archcore · read-only].

Implementation and verification are complete in the working tree. The selected release version is `v0.8.3`; publication remains pending. Initial grounding used branch `main`, HEAD `e33df77`, on 2026-09-07; that baseline had no implementation changes.

[assumption] The original 16–22 human-plus-AI hour estimate remains unvalidated. The expanded regression matrix may change it.

## Declared Delta

The package retains the accepted implementation scope and corrects its route calculation.

| Field | Value |
|---|---|
| `creates` | `research-and-evidence-types`; `evidential-and-temporal-relations` |
| `modifies` | `document-update-frontmatter` |
| `retires` | None |
| `decision` | The user revised the categories on 2026-09-07: `research` belongs to vision; `evidence` belongs to knowledge |
| `intent_gap` | No; the accepted plan and global RFC record the outcome |
| Π | `machine`: registry, serializers, manifest, MCP handlers, existing tests, accepted documents |
| M | `stone`: accepted document-model and relation decisions; accepted sync contract |
| R | `external-contract`: MCP type and relation vocabulary |
| Route | `umbrella`, base L, raised once to XL by `stone` and `external-contract` |
| Instruments | Intent → contract per new capability → describe uncovered update contract → existing plan amendment |

Two independently consumed additions require the umbrella route. The previous `capability` route counted them as one despite listing two creates. The frontmatter defect remains an independent modification delivered in the same release.

The update contract remains a draft spec. Its implementation now retains unknown YAML values and extra-key order. Tests prove refusal before writing when owned-anchor reconstruction would change retained metadata.

The sync and search contracts now describe the seven-value implementation. The type contract and inventories reflect the revised categories. The owning global repository records the same decision; release and runtime support remain pending.

## Tasks

Completed tasks below have source or verification evidence in @docs/releases/research-validation.md. Release tasks remain open.

### Phase 1 — document types

This phase establishes vocabulary before relation scenarios depend on the new types.

1. [x] Register the two planned types in @templates/templates.go.
2. [x] Add the research template generator in @templates/templates.go.
3. [x] Add the evidence template generator in @templates/templates.go.
4. [x] Wire both generators into the dispatcher in @templates/templates.go.
5. [x] Register required sections and prose profiles in @templates/precision.go.
6. [x] Update the creation vocabulary in @internal/mcp/tools/create_document.go.
7. [x] Update discovery descriptions in @internal/mcp/tools/list_documents.go.
8. [x] Document type selection, status conventions, and source tags in @internal/mcp/server.go.
9. [x] Extend type-registration and generation coverage in @templates/templates_test.go.
10. [x] Cover missing-section advisories in @internal/advisory/precision_findings_test.go.
11. [x] Pin default ranking behavior in @internal/mcp/tools/search_documents_test.go.
12. [x] Pin CodeAlignment exclusions in @internal/advisory/code_alignment_test.go.

The ranking table in @internal/mcp/tools/search_documents.go and selector in @internal/advisory/code_alignment.go already supply the planned defaults. New tests guard those boundaries.

### Phase 2 — relation vocabulary

This phase extends stored relation values while retaining existing validation boundaries.

13. [x] Extend the relation registry in @internal/sync/manifest.go.
14. [x] Update relation axes and direction descriptions in @internal/mcp/tools/add_relation.go.
15. [x] Update removal vocabulary descriptions in @internal/mcp/tools/remove_relation.go.
16. [x] Extend relation guidance in @internal/mcp/server.go.
17. [x] Expand valid-value and refusal fixtures in @internal/sync/manifest_test.go.
18. [x] Cover relation creation and refusal boundaries in @internal/mcp/tools/add_relation_test.go.
19. [x] Cover deletion of new relation values in @internal/mcp/tools/remove_relation_test.go.
20. [x] Verify relation readback in @internal/mcp/tools/list_relations_test.go and @internal/mcp/tools/get_document_test.go.
21. [x] Pin cascade exclusions in @cmd/hook_post_tool_use_test.go.
22. [x] Pin restatement exclusions in @internal/advisory/restatement_test.go.
23. [x] Add an isolated seven-value manifest fixture in @cmd/doctor_test.go.

The existing exclusion sets remain in @cmd/hook_post_tool_use.go and @internal/advisory/restatement.go. New relations require no content propagation.

### Phase 3 — frontmatter preservation

This phase is independent of phases 1 and 2.

24. [x] Preserve ordered unknown YAML entries during parsing in @templates/templates.go.
25. [x] Carry retained metadata through @internal/mcp/tools/update_document.go.
26. [x] Serialize retained metadata through @internal/mcp/tools/common.go.
27. [x] Adapt the shared serializer caller in @internal/mcp/tools/create_document.go.
28. [x] Replace the ignored-unknown-fields fixture in @templates/templates_test.go.
29. [x] Cover retained YAML serialization in @internal/mcp/tools/common_test.go and refusal through @internal/mcp/tools/update_document_test.go.
30. [x] Exercise the update matrix in @internal/mcp/tools/update_document_test.go.
31. [x] Verify the structured JSON payload boundary in @internal/sync/payload_test.go.

The implementation uses ordered YAML nodes and excludes retained storage from structured JSON. Alias and merge fixtures verify the selected serializer. The storage remains opaque to Archcore configuration.

### Phase 4 — integration and documentation

This phase verifies the assembled behavior and reconciles public descriptions.

32. [x] Extend the lifecycle scenarios in @internal/mcp/integration/ using the existing in-process harness.
33. [x] Exercise relation round trips in @internal/mcp/integration/relations_test.go.
34. [x] Exercise metadata retention in @internal/mcp/integration/forward_compat_test.go.
35. [x] Verify original `rnd` behavior through @internal/mcp/integration/rnd_test.go.
36. [x] Update the type inventory in @.archcore/dir/categories-and-document-types.doc.md.
37. [x] Update the relation contract in @.archcore/sync/sync-engine.spec.md after implementation verification.
38. [x] Update current type counts in @README.md and @.archcore/cli-ui/building-the-cli.doc.md.
39. [x] Correct the stale count in @.archcore/cli/mcp-token-optimization.idea.md.
40. [x] Run the targeted package suites for @templates/, @internal/mcp/, @internal/sync/, @internal/advisory/, and @cmd/.
41. [x] Run `go test ./...` from the repository root.
42. [x] Run `golangci-lint run ./...` against @.golangci.yml.
43. [x] Build the CLI from @main.go.

The in-process harness is @internal/mcp/integration/harness_test.go. Tests use temporary repositories and isolated host state. A native-binary smoke test also verified category filters over stdio and seven-value manifest acceptance.

### Phase 5 — release handoff

This phase prepares the single release after verification.

44. [x] Prepare release notes through the workflow defined in @.github/workflows/release.yml.
45. [x] Select the release tag consumed by @.github/workflows/release.yml and @.goreleaser.yaml.
46. [ ] Publish the verified release through @.github/workflows/release.yml after release authorization.
47. [ ] Record the shipped engine version for the plugin compatibility handoff.

Release notes are prepared in @docs/releases/research-release-candidate.md. The user selected `v0.8.3`; no tag or release was created. Version injection remains tag-driven through @main.go and @.goreleaser.yaml. The plugin minimum-engine gate belongs to the plugin repository.

## Verification record

After the follow-up review, the full Go and race suites passed with `-p 1`; the linter reported zero issues, and vet and the native build passed. Concurrent full runs hit subprocess timeouts in plugin and updater tests; the validation report records that limitation. The final category change was verified through registered MCP tools and the native stdio server. Go review reports no open P0–P3 findings. Archcore review reconciled both creates and the frontmatter modification.

Test-health-check detected 28 distinct deliberate contract faults across the original checks and follow-up review. Every probe was restored and its original test passed. Returning the serializer to owned-fields-only reconstruction reproduced the original metadata-loss defect. Exact diffs and logs are retained in the proof bundle described in @docs/releases/research-validation.md. Follow-up probes pin final YAML validation, private error text, parser/schema ownership agreement, and custom-field retention.

The relation review added path canonicalization, legacy removal compatibility, and invalid-manifest byte assertions. A pre-existing cache-policy test now uses its existing probe seam to avoid testing an unrelated process timeout. These corrections preserve the declared relation and update boundaries.

## Acceptance Criteria

Delivery closes on recorded verification evidence, not on document status alone.

1. The linked contract suites pass against the final implementation commit.
2. The metadata regression fails against the pre-fix implementation and passes after the fix.
3. The in-process scenarios exercise the registered MCP surface through the test client.
4. The full Go suite, linter, and build complete successfully.
5. Public inventories and release notes match the shipped registry.
6. The release handoff names the engine version and the backward-readability limitation.
7. Implementation review reconciles all three declared delta entries through `/archcore:review`.

## Dependencies

The accepted document-model decision assigns templates to @templates/ and document operations to @internal/docs/. The accepted relation-store decision keeps persistence in @internal/sync/manifest.go. Existing graph edges carry these dependencies.

Phases 1 and 2 precede their combined integration scenarios. Phase 3 can proceed independently. Phase 5 depends on all verification in phase 4.

Older binaries reject manifests containing new relation values — @internal/sync/manifest.go. This release adds no data migration, downgrade conversion, or remote-server compatibility guarantee. Plugin routing follows the engine release. The shared category vocabulary was updated before publication at the user's request.

## Clarifications

The plan retains `accepted`; the PRD and specs retain `draft`. The user's category revision is incorporated in local contracts, code, tests, and 12 existing global documents. The owning global repository records these changes in commit `e825034` and has no pending changes. The user selected `v0.8.3`. Publication remains pending.

“Preserve in place” means retain unknown values and their mutual order after owned fields. It does not promise original byte layout or placement between `title`, `status`, and `tags`. This resolves the original plan's conflict between “in place” and “after tags”.
