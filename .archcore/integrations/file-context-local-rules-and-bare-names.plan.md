---
title: "File Context: Local Pathless Rules Before Global Ones, and Bare File-Name Collisions"
status: draft
tags:
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
---

## Goal

Let local pathless rules reach the pre-edit context, and stop a common file name from matching hundreds of documents.

Measured on 2026-10-15 with `search_documents` `for_path` on this repository, 287 referenced files plus 200 random code files:

- The general part shows 8 rows, all from the `archcore` global source, and omits 5. No local pathless `rule` or `cpat` reaches an edit.
- Cause: general rows sort by type priority, then by path, and `../global/` sorts before `.archcore/` (the `fc.General` sort in @cli/internal/advisory/code_alignment.go).
- `bin/archcore` collects 214 candidate documents through its bare name; `settings.json` files collect 45 each. The `genericFileStems` list in @cli/internal/advisory/code_alignment.go covers neither name.
- 79 of the 287 referenced files already hit the cap of 5 matched rows.
- 72 bare-name rows on 16 unique-name files match only inside a longer name: `document.go` inside `create_document.go`.

A local document is authoritative over a global one on the same topic (@plugin/plugins/archcore/skills/_shared/globals.md); the current sort hides every local pathless rule behind global ones.

## Tasks

### Phase 0 — Measurement

1. [x] Add the delivery measurement script and a `test-delivery-bench` target. @plugin/test/behavioral/delivery-bench.py, @plugin/Makefile
2. [x] Run it on this repository before the change and keep the output beside the bench results.

### Phase 1 — Contract

3. [x] Add behavior 17: a local general row ranks above a global one at equal type priority.
4. [x] Add to the spec the bare-name rule the engine already applies through `genericFileStems` (behavior 18).
5. [x] Decide the bare-name rule: a base name another tracked file carries matches only by full path (behavior 19).

### Phase 2 — Engine

6. [x] Sort general rows local before global at equal type priority. @cli/internal/advisory/code_alignment.go
7. [x] Skip the bare-name match when `git ls-files` lists another file with that base name. @cli/internal/advisory/code_alignment.go, @cli/internal/git/git.go
8. [x] Test that two local pathless rules appear beside ten global ones. @cli/internal/advisory/code_alignment_test.go
9. [x] Test that a base name two tracked files share gives no row by name alone. @cli/internal/advisory/code_alignment_test.go
10. [x] Test that a project outside git keeps only the generic-stem rule. @cli/internal/advisory/code_alignment_test.go
11. [x] Keep the hook and `for_path` parity test green. @cli/internal/mcp/integration/

### Phase 3 — Canon and measurement

12. [x] Update the advisory subsystem doc for the general-row order and the bare-name rule. @.archcore/architecture/advisory-subsystem.doc.md
13. [x] Rerun the delivery measurement and compare it with the Phase 0 output.

### Phase 4 — Name boundary and shared-name rules

14. [x] Add behavior 20 and the name-boundary invariant to the file-context spec.
15. [x] Count a bare name only where no file-name character or `.` precedes it. @cli/internal/advisory/code_alignment.go
16. [x] Give reason `kind` to an accepted rule naming a shared base name. @cli/internal/advisory/code_alignment.go
17. [x] Test the boundary and the shared-name rule. @cli/internal/advisory/code_alignment_test.go
18. [x] Rerun the delivery measurement against the HEAD build.

## Acceptance Criteria

- `for_path` on @plugin/plugins/archcore/skills/_shared/gate-contract.md lists at least one local pathless rule in the general part; on 2026-10-15 it lists none.
- At equal type priority, no global general row precedes a local one.
- `for_path` on @cli/examples/01-minimal/.archcore/settings.json returns no row matched only by the bare name `settings.json`; on 2026-10-15 it collected 45 candidate documents.
- `for_path` on @plugin/plugins/archcore/bin/session-start returns the same matched rows as on 2026-10-15; no other tracked file carries that name.
- The delivery measurement reports 0 bare-name rows on files whose base name another tracked file carries.
- The hook text stays within 2 048 runes, and the `for_path` response within 8 000 characters.
- `go test ./...` and `golangci-lint run ./...` pass in `cli/`.

## Dependencies

- The user confirmed both spec-wrong verdicts on 2026-10-15; the spec draft carries behaviors 17 and 18.
- The user chose the uniqueness rule on 2026-10-15; @.archcore/integrations/bare-file-name-matches-only-when-unique.adr.md records it with the rejected alternatives.
- On 2026-10-16 the user chose the name boundary (Phase 4) and the `kind` reason for rules on a shared name.
- `bin/archcore` is a build output ignored by `.gitignore`; agents do not edit it, so its 214 candidates do not decide task 5.
- The engine shipped in commit f426f8b under the agent-context-delivery plan, which awaits closeout; this plan does not reopen it.
- Task 6 lands before task 8; task 7 lands before tasks 9 and 10.

## Declared Delta

- `creates=[]`; `modifies=[file-context-resolution]`; `retires=[]`; `decision=[bare-name rule]`; `intent_gap=no`.
- Route: amendment, size M. M=stone: the accepted `search_documents` spec depends on file-context resolution, and agents on every host read its result. R=external-contract: the order of the hook text changes for those agents. One raise step covers both.
- Π: machine for the measurement and the sort code; user for the bare-name rule, chosen on 2026-10-15.
- Verdicts: file-context-resolution was spec-wrong on the general-row tie-break and on the unstated bare-name rule; the spec was edited on 2026-10-15 after the user's confirmation. Behavior 19 and failure rule 5 were added the same day after the user chose the uniqueness rule.
