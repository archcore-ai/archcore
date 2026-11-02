---
title: "Rollout: Global Sources as a Read Policy — Addressed Writes, One Address per Document, No Reserved Directory"
status: accepted
tags:
  - "component:cli"
  - "component:plugin"
  - "config"
  - "globals"
  - "hooks"
  - "mcp"
  - "release"
---

## Goal

Ship the source-mounts contract in v0.15.0 together with the `project` argument: a declared global source is a read policy, every row and write response carries `project` and `source`, no document is read-only, and `.archcore/global/` no longer exists. Done when the Go and bats suites are green with exactly the intentional failures of the acceptance criteria rewritten, the live probes P0 and A–D pass on Claude Code, Codex, and OpenCode against the built CLI, and the six superseded documents carry their new status.

## Tasks

### Phase 1 — Model and guards

1. Remove `Global` and `ReadOnly` from `Document`, add `Project`, drop `SourceIDReserved`. @cli/internal/docs/document.go.
2. Delete `IsReservedGlobalDir`, `isReservedGlobalDirFold`, `isGlobalPathFold`, `IsExternalGlobalDocument`; reduce `AnnotateSource` to "which declared source owns this root". @cli/internal/docs/globals.go.
3. Reduce `GuardWritablePath` to the lexical, document-only, and symlink layers; remove the external-global branch of `ValidateReadPath`; add the `..` refusal that names the owning project. @cli/internal/docs/guard.go.
4. Drop the `global` skip and the declared-global skip from the local phase of the scan; keep the two phases and the valid-type filter. @cli/internal/docs/scan.go, @cli/templates/templates.go (`WalkArchcoreFilesSkipping` callers).
5. Add the validation "a source path under the declaring `.archcore/` is fatal" to `CheckGlobalDir` and its message with the migration target; drop the self-overlap case it subsumes. @cli/internal/config/globals.go, @cli/internal/docs/inspect.go.
6. Remove the global exclusion from the sync file scan; the invariant of task 5 makes it unreachable. @cli/internal/sync/hash.go.

### Phase 2 — Tools and server

7. Compute `source` per call: `"local"` for the session root, the declared id when the call root equals a resolved source of the session root, absent otherwise; append it beside `project`. @cli/internal/mcp/tools/project.go, @cli/internal/mcp/tools/docs_bridge.go.
8. Emit `project` and `source` on every row of `list_documents`, `search_documents`, `get_document`, and the file-context rows; delete the `Global` and `ReadOnly` row fields. @cli/internal/mcp/tools/list_documents.go, @cli/internal/mcp/tools/search_documents.go, @cli/internal/mcp/tools/search_file_context.go, @cli/internal/mcp/tools/get_document.go.
9. Remove `loadGlobalsFailClosed` and the `read-only global source` messages from the three write tools and both relation tools; keep the one-root check of `add_relation` and `remove_relation`. @cli/internal/mcp/tools/create_document.go, @cli/internal/mcp/tools/update_document.go, @cli/internal/mcp/tools/remove_document.go, @cli/internal/mcp/tools/add_relation.go, @cli/internal/mcp/tools/remove_relation.go.
10. Rewrite the `GLOBAL SOURCES` sentence of the `initialize` instructions: sources are read with the session root, a write to a source passes its `project`, the response names it in `source`. @cli/internal/mcp/server.go, under the 2,048-character cap.
11. Rewrite the tool descriptions that say "read-only global source". @cli/internal/mcp/tools/add_relation.go, @cli/internal/mcp/tools/remove_relation.go, @cli/internal/mcp/tools/list_documents.go, @cli/internal/mcp/tools/search_documents.go.

### Phase 3 — Hooks and instructions

12. Delete the `IsExternalGlobalDocument` branch and `writeGuardUnverifiableReason`; add `source` to the deny reason beside `project`. @cli/cmd/hook_write_guard.go, @cli/cmd/hook_project_root.go (`withProjectHint`).
13. Change the `[global]` marker of the code-alignment row to `[source:<id>]`. @cli/internal/advisory/code_alignment.go.
14. Render the `GLOBALS` block as `id → root` with counts; keep the block name. @cli/cmd/hooks_common.go.
15. Rewrite the managed-block sentence on global sources and bump `InstructionsBlockVersion` to 3. @cli/internal/agents/instructions.go.

### Phase 4 — Plugin

16. Rewrite the write convention of the globals asset: a write to a source passes `project`, the report names `source`; drop every `read_only` and `global: true` branch, keep `source_kind`. @plugin/plugins/archcore/skills/_shared/globals.md, @plugin/plugins/archcore/skills/_shared/target-project.md.
17. Replace "never modify a global document" in the skills, tracks, and agents with the rule of task 16. @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/review/SKILL.md, @plugin/plugins/archcore/skills/init/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/decision.md, @plugin/plugins/archcore/skills/_shared/tracks/research.md, @plugin/plugins/archcore/skills/_shared/grounding/convert-routing.md, @plugin/plugins/archcore/agents/archcore-assistant.md, @plugin/plugins/archcore/agents/archcore-auditor.md, @plugin/plugins/archcore/copilot-agents/.
18. Remove example 08 and the `.archcore/global/` steps of the vendoring guide; the guide's new procedure places a vendored source in a directory with its own `.archcore/`. @cli/examples/08-global-in-archcore/, @cli/examples/README.md, @.archcore/globals/vendoring-a-global.guide.md.

### Phase 5 — Tests, probes, documents

19. Rewrite the intentional failures of the acceptance criteria: each read-only refusal becomes a success through `project`, each reserved-directory case is deleted, the `..` write case asserts the owning-project message. Files: @cli/internal/mcp/tools/globals_test.go, @cli/internal/mcp/tools/globals_reserved_test.go, @cli/internal/mcp/tools/guard_writable_path_test.go, @cli/internal/docs/guard_test.go, @cli/cmd/hook_write_guard_test.go, @cli/internal/mcp/integration/globals_test.go, @cli/cmd/hooks_common_test.go, @cli/cmd/hooks_globals_block_test.go, @plugin/test/integration/search-response-shape.bats, @plugin/test/structure/cli-compat-invariant.bats, @plugin/test/structure/agents.bats.
20. Add the new cases: `source` on rows for a session root, a declared source, and an undeclared project; a write and a relation inside a source through `project`; the startup refusal for a path under the declaring `.archcore/`; the hook deny naming `source`. @cli/internal/mcp/integration/globals_test.go, @cli/cmd/hook_write_guard_test.go, @cli/internal/config/globals_test.go, @plugin/test/integration/cross-project.bats.
21. Run the live probes on the three hosts per the host probe protocol and record the rows. @.archcore/plugin/host-probe-protocol.spec.md, @plugin/test/probe/mkprobe.
22. Amend the two specs this plan keeps: the session disclosure spec for the `id → root` table, and the call-root spec for the `source` field and the `..` refusal. @.archcore/globals/session-globals-disclosure.spec.md, @.archcore/mcp/call-root-resolution.spec.md.
23. On acceptance: set `rejected` (the corpus marks a superseded document this way, with an incoming `supersedes` edge) on `global-sources.spec`, `globals-are-read-only-everywhere.rule`, `local-overrides-global.rule`, `declaring-global-sources.rule`; set `rejected` on `writable-global-sources.idea`; set `accepted` on the three new documents. Status changes wait for the user's acceptance.

## Acceptance Criteria

- Every tool call without `project` on a project that declares no source returns the same rows as 0.14.4 plus `project` and `source: "local"`; the read-only fields are absent.
- On a project that declares a source, the rows of that source carry `source_kind: "global"`, `source_id: <id>`, `source: <id>`, and a path under `.archcore/` with the source's `project`; no row path starts with `..`.
- `update_document(project=<root of G>, …)` from a session in A, where A declares G as `org`, changes the file and returns `source: "org"`.
- A write through `../g/.archcore/x.md` from the same session fails with a message naming the owning project and `project`.
- A `Write` into any document of G from the same session is denied with a reason naming `update_document`, `project`, and `org`.
- An MCP server for a project whose `globals` path lies under its own `.archcore/` refuses to start with a message naming the id and the migration target.
- The manifest stays at version 1; `add_relation` across two roots fails as in 0.15.0 before this plan.
- `source_kind`, the `source` filter, `coverage`, `hits`, and `by_source` are unchanged against the 0.14.4 oracles.
- Intentional failures, and no others, in the test run before task 19: `TestScanDocuments_SkipsGlobalDirWithoutDeclaration`, `TestScanDocuments_InTreeGlobalOutsideReservedDir_NotDoubleScanned`, `TestScanDocuments_SelfOverlapFails`, `TestScanDocuments_SkipsNestedGlobalDir`, `TestCreateDocument_RejectsGlobalPath`, `TestUpdateDocument_RejectsGlobalPath`, `TestRemoveDocument_RejectsGlobalPath`, `TestCreateDocument_RejectsReservedGlobalDir`, `TestUpdateDocument_RejectsNestedReservedGlobalDir`, `TestCreateDocument_RejectsNestedReservedGlobalDir`, `TestRemoveDocument_RejectsNestedReservedGlobalDir`, `TestCreateDocument_FailsClosedOnUnreadableSettings`, `TestUpdateDocument_FailsClosedOnUnreadableSettings`, `TestRemoveDocument_FailsClosedOnUnreadableSettings`, `TestIsGlobalPath_LookupBased`, `TestIsReservedGlobalDir`, `TestGetDocument_ReservedDirIsReadOnly`, `TestAddRelation_RejectsGlobalEndpoint`, `TestAddRelation_RejectsNestedReservedGlobalEndpoint`, `TestGuardWritablePath` (both packages), `TestHandleCreateDocument_CaseVariantGlobalBlocked`, `TestHandleUpdateDocument_CaseVariantGlobalBlocked`, `TestValidateReadPath`, `TestValidateReadPath_MultipleGlobals`, `TestGlobals_WriteToGlobalRejected`, `TestGlobals_GetExternalGlobalReadable`, `TestGlobals_RelationToGlobalRejected`, `TestWriteGuard_DeclaredExternalGlobalIsBlocked`, `TestWriteGuard_CaseVariantGlobalBlocked`, `TestWriteGuard_UnreadableSettingsFailsClosedOutsideTheProject`, `TestWriteGuard_UnreadableSettingsStillAllowsRepairingIt`, `TestBuildSessionContext_ExcludesGlobals`, the `read_only` assertion of `search-response-shape.bats`, the field list of `cli-compat-invariant.bats`, and the auditor source check of `agents.bats`. A failure outside this list is an uncounted case and blocks the release.
- The three host probe rows of this run carry `pass` in every column.
- `golangci-lint run ./...` is clean.

## Dependencies

- @.archcore/globals/source-mounts.spec.md — the contract this plan ships.
- @.archcore/globals/global-sources-are-a-read-policy.adr.md — the decision and the user's choices of 2026-11-02.
- @.archcore/globals/local-overrides-source.rule.md — the agent-side convention the plugin tasks implement.
- @.archcore/mcp/call-root-resolution.spec.md — the `project` argument this plan builds on; amended by task 22.
- @.archcore/globals/session-globals-disclosure.spec.md — amended by task 22.
- @.archcore/plugin/host-probe-protocol.spec.md — task 21.
- Ordering: phase 1 before phase 2; task 7 before task 8 and task 12; task 15 before task 17; task 19 after phases 1–4; task 23 last, after the user's acceptance.

## Declared Delta

- create adr globals/global-sources-are-a-read-policy — done
- create spec globals/source-mounts — done
- create rule globals/local-overrides-source — done
- create plan globals/source-model-rollout — this document
- amend spec globals/session-globals-disclosure (the `id → root` table) — done, task 22
- amend spec mcp/call-root-resolution (`source` field, `..` refusal) — done, task 22
- amend guide globals/vendoring-a-global (no reserved directory) — done, task 18
- retire spec globals/global-sources — done, superseded by source-mounts, task 23
- retire rule globals/globals-are-read-only-everywhere — done, superseded by local-overrides-source, task 23
- retire rule globals/local-overrides-global — done, superseded by local-overrides-source, task 23
- retire rule globals/declaring-global-sources — done, folded into source-mounts, task 23
- retire idea globals/writable-global-sources — done, rejected, realized through `project` instead, task 23

## Outcome

- Shipped in v0.15.0, tag on commit 5e9d7fe4; the Release run published the plugin, `@archcore/opencode-plugin@0.15.0`, and the CLI.
- The user accepted the five implementation deviations recorded in globals/global-sources-are-a-read-policy.adr, Clarifications. Acceptance criterion 2 holds in that amended form: a global row keeps its 0.14 read path.
- Acceptance criterion 10 is met for Claude Code and Codex. The OpenCode row is `deferred:free-model-limit`; the user chose to release without it. [EVIDENCE REQUIRED: an OpenCode probe row of P0 and A–D against CLI v0.15.0]
- none: cross-project relation edges — ruled out 2026-11-02
