---
title: "Listing Evidence Counts Only an Installed Plugin"
status: draft
tags:
  - "cli"
  - "integrations"
  - "update"
---

## Goal

Make the plugin surface read a host listing the way the rest of the surface already reads on-disk
evidence: an entry counts as the Archcore plugin only when the host reports the plugin itself as
installed. Today `objectNamesPlugin` and `findPluginEntry` in `@internal/plugin/evidence.go` count an
entry the host marks `"installed": false`, and count a bare marketplace id carried in an identity
value. The key reading in the same file and `registryNamesPlugin` in `@internal/plugin/registry.go`
already exclude both.

The read is a guard: its result decides whether a mutating host command runs
(`@.archcore/code-quality/fail-open-or-fail-closed-reads.rule.md`, requirement 2). An ambiguous entry
currently permits the command instead of refusing it.

The defect is latent, not live. Codex 0.147 lists uninstalled marketplace plugins only under
`--available`, and the CLI passes `codex plugin list --json` without it — verified 2026-08-17: 0
entries without the flag, 180 with it. The invariant of `@.archcore/update/updating-the-plugin.spec.md`
therefore rests today on a command line whose installed-only property no document states. This plan
hardens the parser and records the missing property.

## Tasks

### Phase 1 — The parser guard

1. Read the `installed` boolean of a listing entry in `objectNamesPlugin`; a `false` answers "not installed" — `@internal/plugin/evidence.go`.
2. Narrow the identity-value match to the plugin itself; exclude the bare marketplace id, as the key reading does — `@internal/plugin/evidence.go`.
3. Apply the same boundary to the plain-text fields in `parseTextListing` — `@internal/plugin/evidence.go`.
4. Stop the walk from descending into an entry the host already answered for — `@internal/plugin/evidence.go`.
5. Name the guard classification at the branch that decides absence, per `@.archcore/code-quality/fail-open-or-fail-closed-reads.rule.md` — `@internal/plugin/evidence.go`.

### Phase 2 — Tests

6. Add a JSON case: an entry under `available` carrying `"installed": false` answers not listed — `@internal/plugin/evidence_test.go`.
7. Add a JSON case: an identity value equal to the marketplace id answers not listed — `@internal/plugin/evidence_test.go`.
8. Add a text case: a marketplace-named line without an installed plugin answers not listed — `@internal/plugin/evidence_test.go`.
9. Add a planner case: that evidence plans no update action and no already-installed report — `@internal/plugin/plan_test.go`.

### Phase 3 — The governing documents

10. State in the Surface evidence line that each listing command enumerates installed plugins only — `@.archcore/update/updating-the-plugin.spec.md`.
11. Define what a listing showing the plugin means, beside requirements 6 and 7 — `@.archcore/update/updating-the-plugin.spec.md`.
12. Carry the same definition into the host-evidence line that requirements 9 and 25 read — `@.archcore/integrations/plugin-delivery.spec.md`.

### Phase 4 — Verification

13. Run `go test ./internal/plugin/... ./cmd/...`.
14. Read the two plugin-repository smoke tests for expectations this change moves — `archcore-ai/plugin`, `test/integration/codex-plugin-smoke.bats`, `test/integration/copilot-plugin-smoke.bats`.

## Acceptance Criteria

- The Phase 2 cases fail against the parser as it stands and pass after Phase 1 — eight subtests across three test functions.
- `go test ./internal/plugin/... ./cmd/...` passes.
- The Surface table and the requirement pair of `@.archcore/update/updating-the-plugin.spec.md` carry the installed-only property, so a later edit that adds a flag like `--available` reads as a change to a stated property.
- The branch that decides absence in `@internal/plugin/evidence.go` names its guard classification and cites the rule slug.
- No task changes a frozen identifier, a host command line, or any behavior outside the plugin surface.

## Dependencies

- `@.archcore/update/updating-the-plugin.spec.md` and `@.archcore/integrations/plugin-delivery.spec.md` are `accepted`; Phase 3 edits both.
- `@.archcore/integrations/plugin-cli-compatibility.rule.md` requirement 3 permits this surface to read host plugin state, and requirement 11 freezes the three identifiers this plan leaves untouched.
- `@.archcore/code-quality/fail-open-or-fail-closed-reads.rule.md` supplies the guard classification Phase 1 applies.
- The host listing schemas are another project's to change; the walk stays tolerant of the envelope — `@internal/plugin/evidence.go` records that as an assumption.
- [assumption] The Copilot text listing prints installed plugins only; the flag survey that confirmed the Codex behavior has no Copilot equivalent.
- Phase 4 task 14 reads a second repository and changes nothing in it.

## Declared Delta

- Route: `amendment` (size M). Base label S from a non-empty `modifies`; raised one step because the touched zone is `stone` — three governing documents carry `status: accepted` and the shipped engine consumes them.
- Δ: `creates=[]`; `modifies=[plugin-presence-evidence]`; `retires=[]`; `decision=none`; `intent_gap=no`.
- Π: `machine` for the parser behavior (probe over a copy of the package), the host behavior (`codex plugin list --help`, live listings of four hosts, 2026-08-17), and the document claims; `user` for the scope, which the request text fixed.
- M: `stone`. R: `external-contract` — the read depends on host CLI output another project owns.
- Verdicts on `plugin-presence-evidence`: `code-wrong` for the parser, resolved by Phase 1 and Phase 2; `spec-wrong` for the undefined term and the unstated command property, resolved by Phase 3.
- Unplanned Δ, recorded at review: task 4 was not in the declared list. Implementation surfaced a second path to the same defect — the walk descended into an entry it had just rejected and read the id it was recognized by a second time, as a bare string with no installation flag beside it.
