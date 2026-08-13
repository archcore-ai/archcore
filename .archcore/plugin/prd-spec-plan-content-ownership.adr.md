---
title: "Content-Kind Ownership Across prd, spec, and plan — One Owning Document per Statement"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
  - "skills"
---

## Context

The `sdd` track produces four documents on one topic — `idea`, `prd`, `spec`, `plan` — and they restate each other. Measured on this corpus on 2026-08-13:

- `track-layer-extension.idea`, `.prd`, and `.plan` carry the same statement three times ("frame → gather → conclude, produces `rnd` with Goal, Questions, Approach, Findings, Recommendation, Next Action"), and one line is verbatim in both the `prd` Goals section and the `plan` Acceptance Criteria section.
- 6 of the 8 requirements in `plugin-v2.prd` reappear as numbered behavior in `command-surface-v2.spec`.
- Across the 5 PRDs in this repository and the CLI repository, 2 carry BCP 14 modals and 1 carries 4 EARS clauses — `cli-update-analytics.prd` states 8 numbered behaviors that its own `cli-update-telemetry.spec` also carries.

Four mechanisms produced this:

1. `prd` was the one type in the track with no content contract. The boundary existed only in `spec-contract.md`, which tells a `spec` what to leave to the `prd` and never the reverse.
2. Every gate exit check verified that a section is present. No check asked whether the section repeats the document upstream of it.
3. `sdd.decompose` asks zero questions and composes from upstream drafts, so restatement was the only output it could produce.
4. `skip_when` asked whether a document on the topic exists, never whether content remains to add. The compression path that let a feature-scoped request go straight to the `spec` lived in `skills/plan/references/feature-flow.md`, which the v2 track cutover deleted; only the 40-line target survived into `sdd.require`.

A fifth mechanism was latent: the CLI `prd` template carried 12 top-level sections — including `Solution Overview`, `Constraints`, `Timeline`, and an `FR-###` table with acceptance criteria — against the 4 sections `sdd.require` checks. Skills compose bodies themselves, so the template reached a document only through the empty-content path of `create_document`.

## Decision

Assign every kind of statement produced on one topic to exactly one owning document, record the assignment in a new content contract, `plugins/archcore/skills/_shared/prd-contract.md`, and check its mechanical half in the post-tool-use hook rather than at the gate alone.

1. The contract carries the routing gate in the `prd` → `spec` / `plan` / `adr` direction, mirroring the gate `spec-contract.md` already carries in the opposite direction.
2. The contract carries the content-kind ownership table: a statement written into one document is not written into a second, and a downstream gate that takes over a kind edits the upstream statement down to the kind that document owns, in the same `update_document` call that closes the gate.
3. A `prd` requirement states an outcome. EARS clause order and BCP 14 modals stay out of a `prd`; they are `spec` notation. The ISO requirement types keep their own notation.
4. The coverage category `Completion Signals` reaches three different sections — `prd` Goals and Success Metrics, `plan` Acceptance Criteria, `spec` Conformance — and coverage at one gate does not close the category at another. Recorded as rule 4 of the ownership table; `coverage-taxonomy.md` carries a pointer to it rather than a second copy.
5. `sdd.require`, `sdd.design`, and `sdd.decompose` each gain a blocking exit check that applies the ownership table by reference, per the gate-contract rule against restating a shared contract inside a gate. `sdd.design`'s Purpose states that the gate also calls `update_document` on the `prd`, because the gate record template has no field for a document a gate edits rather than produces.
6. `sdd.require` regains the compression path as a `skip_when` condition: a feature-scoped request whose problem and goals are already recorded in an `idea`, `rnd`, or `adr` continues at `sdd.design`. `sdd.design` and `sdd.decompose` accept that upstream document in place of the `prd`.
7. The CLI `prd` template collapses to the four canonical sections, and `templates.RequiredSections` gains a `prd` entry.
8. The post-tool-use precision hook checks four mechanical properties per write, all advisory: the mandatory sections; a BCP 14 modal anywhere in a `prd` body; a numbered line opening with `WHEN`, `WHILE`, or `IF`; and a heading listed in the new `templates.ForeignSections` canon, which names the type that owns it.
9. The same hook checks ownership rule 1 across documents: for each `implements` or `extends` edge touching the written document, it compares list statements against the neighbour's body and reports an overlap of 85% of content words or more. Thresholds are `minRestatementTokens = 6` and `restatementThreshold = 0.85`, calibrated on this corpus — the measured copied line scores 1.0 and the closest genuine `prd`/`spec` pair scores 0.67.

## Alternatives Considered

1. **Non-redundancy exit checks in the gates, with no contract** — rejected. `gate-contract.md` forbids restating a shared contract's rules inside a gate, so three gates would each carry their own copy of the boundary, which is the drift this decision removes.
2. **Merge `prd` and `spec` into one per-feature document** — rejected in `no-frd-type-prd-scope-rule.adr` as a flow-composition question rather than a type question, and re-rejected here: one document would blur the vision and knowledge categories that the write-affinity model in `command-surface-v2.spec` depends on.
3. **Leave the CLI template and rely on the track alone** — rejected. `create_document` falls back to the template when content is empty, so the 12-section shape stayed reachable and disagreed with the check `sdd.require` runs.
4. **Leave every ownership check to the composing agent** — rejected. The measured failure was a copied sentence, which a script decides without judgement, and an advisory hook costs a glance when it is wrong. A semantic-similarity threshold low enough to catch paraphrase was rejected with it: two documents on one topic share vocabulary, so that setting reports the boundary working as noise.

## Consequences

- Every gate that produces a second document on a topic has one objective question to answer: which document owns this kind of statement.
- A feature-scoped request with its "why" recorded upstream now produces two documents instead of four.
- `prd-contract.md` is the fifth shared content contract; the shared-asset count in `plugin-architecture.spec` moves from 4 to 5.
- The precision hook gains five finding classes, all advisory: a missing mandatory section, a BCP 14 modal, an EARS clause, a foreign heading, and a restated statement. The hook never blocks.
- The restatement check reads at most 5 neighbours per write, and only across `implements` and `extends`. A `related` edge is an association, so an overlap across it reports nothing.
- Tradeoff: near-verbatim restatement is now decided by a script; **paraphrased** restatement is not, and stays with the composing agent at the gate exit checks. A paraphrase is where the two documents differ legitimately, so the boundary between the two cases is the boundary between a copy and a rewrite. [assumption] The effect of either half on restatement rates has not been measured over time.
- Tradeoff: 2 of the 5 PRDs already in these repositories will report modal findings on their next update. The finding names the fix and does not block it.
- One fact is recorded in one place, including inside this change: the ownership table is the single record of the per-type section assignment, and `coverage-taxonomy.md` and the contract's own Forbidden list point at it instead of repeating it.
- Documents accepted before this decision are not retro-fitted. `track-layer-extension.prd` and `plugin-v2.prd` keep their recorded content.

## Superseded when

- `get_type_schema` ships per `get-type-schema-mcp-tool.idea` and the CLI templates become the single machine-readable section canon, at which point `prd-contract.md` keeps only the routing gate, the ownership table, and the compression path.
- The restatement check is measured against a body of real writes and its threshold proves either noisy or inert, which would retune `restatementThreshold` or withdraw the check.
