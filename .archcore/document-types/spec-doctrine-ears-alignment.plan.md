---
title: "Fix Plan: Align Spec Doctrine with the Plugin EARS + BCP 14 Contract"
status: accepted
tags:
  - "document-types"
---

## Goal

Bring spec authoring in this repository into alignment with the canonical contract defined in the Archcore plugin repo (`skills/_shared/spec-contract.md` plus the ADR "Spec — Single Narrative, Generalized Sections, EARS + BCP 14 Notation"): one form — six fixed sections — with Normative/Failure Behavior lines in EARS clause order carrying BCP 14 modals, and bodies capped at 80 lines. Close the three delivery gaps that produced non-conformant specs on 2026-07-10: a stale local guide that wins via archcore-first search priority, the direct-MCP creation path that bypasses the plugin contract, and the absence of advisory lint.

## Tasks

### Phase 1 — doctrine in this repo

1. Rewrite `writing-spec-documents.guide.md`: six sections, notation block mirrored from the plugin contract, 80-line body cap, forbidden-content list, and a source-of-truth note (the plugin contract is canonical; when it changes, update the guide and @templates/templates.go together). — **done 2026-07-10**
2. Update `spec-type-usage.rule.md`: broaden "one concrete technical boundary" to "a boundary or a feature/subsystem others rely on"; refresh the Enforcement bullet to the six-section template shipped by @templates/templates.go. — **done 2026-07-10**

### Phase 2 — align the 2026-07-10 specs

3. `jsonfile-config-surgery.spec.md`: error paths as IF…THEN in Failure Behavior, compound clauses split, caller obligations demoted to recovery notes, body ≤ 80 lines. — **done, then removed**
4. `settings-json.spec.md`: fold Authority/Subject/Definitions into Purpose & Scope / Surface; drop the EARS pattern-taxonomy subsections; SHALL → MUST/SHOULD/MAY grading; strip rationale tails; fix the WHERE misclassifications; body ≤ 80 lines. — **done, then removed**

Note: both specs were aligned on 2026-07-10 and deleted from the corpus by the owner the same day. The phase's conventions are carried forward by the guide (task 1), the tool description (task 5), and the lint (task 6); any future spec on these subjects starts from the six-section template.

### Phase 3 — close the direct-MCP path (CLI code)

5. Add one sentence to the `content` parameter description in @internal/mcp/tools/create_document.go: spec normative lines use EARS clause order with BCP 14 modals. Update co-located tests. — **done 2026-07-10**

### Phase 4 — advisory lint (plugin repo)

6. Extend `bin/check-precision` in the plugin repo with spec-only advisory findings: `SHALL` in the body → suggest BCP 14 modals; body over 80 lines → cite the cap (120 for init-synthesized flagship specs). Soft warnings, exit 0, firing on create/update only — consistent with the plugin's migrate-on-next-edit policy. Add bats unit tests. — **done 2026-07-10**

Out of scope: migrating existing spec corpora in either repo (legacy headings stay accepted by check-precision); blocking notation validation; changes to the plugin contract itself.

## Acceptance Criteria

- The local guide and rule teach only the six-section EARS + BCP 14 form and name the plugin contract as the canonical source. — met
- The 2026-07-10 specs conform: six sections, BCP 14 modals (no SHALL), IF…THEN failure clauses, bodies ≤ 80 lines, check-precision silent on update. — met at completion; the documents were subsequently removed by the owner
- The `create_document` content description states the spec notation; `go test ./...` passes. — met
- Plugin `check-precision` warns (advisory) on SHALL and on >80-line spec bodies; the plugin test suite passes; untouched legacy specs stay silent until edited. — met

## Dependencies

- Plugin repo contract files (read-only inputs): `skills/_shared/spec-contract.md` and the ADR spec-single-narrative-ears-bcp14.
- Already landed in this repo: template and type-label alignment from commit 2721bf4 (templates.go, server.go, create_document.go).