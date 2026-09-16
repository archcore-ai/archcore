# Scenario and Journey Plugin Runtime — Design

Date: 2026-09-16. Status: approved design (approach A, full package in one release).
Reader: the implementer of the plugin half of the actor-subject vocabulary and the reviewer of that work.

## Purpose

Deliver the plugin runtime for the two actor-subject document types that CLI v0.8.4 ships: `scenario` (knowledge) and `journey` (vision). After this work, the four plugin commands produce, read, and verify both types, and refuse to write them on an engine below 0.8.4.

The behavior is recorded in four draft specs and one plan under `.archcore/plugin/`; this design records the approach, the settled choices, and the component boundaries that the specs assume. When this file and a spec differ, the spec wins and this file is corrected.

- Outcomes: `.archcore/plugin/scenario-and-journey-runtime.prd.md`
- Behavior: `.archcore/plugin/actor-subject-content-contracts.spec.md`, `.archcore/plugin/illustrate-instrument.spec.md`, `.archcore/plugin/scenario-evidence-in-describe-and-closeout.spec.md`, `.archcore/plugin/actor-subject-compatibility.spec.md`
- Order of work: `.archcore/plugin/scenario-and-journey-runtime.plan.md`

## Context

- The vocabulary is defined in `concepts/scenario-and-journey-types` (archcore global source, `draft`), grounded on `concepts/bdd-document-type-evaluation` and `concepts/bdd-and-adjacent-practices`. These documents stay in the global source and are not imported into this repository.
- The engine half shipped in CLI v0.8.4, tag on commit `2a8f6e4` in the `cli` repository: registry of 23 types, templates, precision canon F6, `MaxBodyLines` table with `spec`, `scenario`, and `journey` at 120, `scenario` in the injection allowlist below `spec`, `.feature` as a source extension. The CLI-side contracts are `scenario-and-journey-types.spec` and `scenario-and-journey-advisory-canon.spec` under `document-types/` in that repository.
- The plugin runtime today knows 21 types. The precedent for a vocabulary release is the research vocabulary (CLI 0.8.3): a compatibility file, a probe helper, argument-hint entries, and edits to the covering specs.

## Approach

Approach A: all four capabilities in one release. Alternatives considered:

- B, two releases (readable half first, producing half second): rejected because it needs two plugin version bumps, two passes over `track-layer.spec`, and leaves a state where the types exist but `/archcore:plan` produces neither.
- C, contracts and gate only: rejected because PRD requirements 2 and 4 stay unmet and the RFC's Runtime section stays open.

Phases 1–4 of the plan are independent, so pull requests can be cut per phase inside one branch without an intermediate release.

## Settled choices

Two choices the global RFC leaves open were settled on 2026-09-16:

1. `document journey` enters `sdd.require` in callable mode and produces only the `journey`, no `prd`. This mirrors `document research`, which files a ready vision artifact through a plan-side instrument. Rejected: no `document journey` entry (diverges from the RFC's argument-hint clause); a describe-track gate (describe documents existing code, a journey records intent before code).
2. The engine gate lives in a separate file, `skills/_shared/actor-subject-compatibility.md`, at minimum 0.8.4, on the structure of `research-compatibility.md`. Rejected: one merged `vocabulary-compatibility.md` (renames a file 9 sources and two accepted specs reference); a second section inside the research file (file name stops matching its content).

Both choices are recorded as ADRs in `.archcore/` after this design is approved (integration rule 7).

## Components

Four units, one spec each. Each unit names what it does, how it is used, and what it depends on.

### 1. Content contracts

- Does: tells a composing skill what a `scenario` and a `journey` hold before it writes. Two new files, `skills/_shared/scenario-contract.md` and `skills/_shared/journey-contract.md`, at section parity with `spec-contract.md`: what it is, routing gate, when not to write, mandatory sections, notation (F6), body cap with an "Over the cap" section naming the actor, status, forbidden in the body, enforcement, rationale, examples.
- Used by: `plan` at `sdd.illustrate` and `sdd.require`; `document` on the describe track.
- Depends on: the engine canon in `@../cli/templates/precision.go`; when the contract and the CLI hook disagree, the CLI is right.
- Canon hooks edited: `precision-rules.md` rules 6 and 7 (both types claim-recording, F6 profile); `prd-contract.md` ownership table (three rows: intended user path and header → `journey`; anchored flow → `scenario` Flows; example with data → `scenario` Examples); `spec-contract.md` Conformance (an example past five lines goes to a linked `scenario`).

### 2. Illustrate instrument

- Does: produces one `scenario` per qualifying capability at a new gate `sdd.illustrate`, sequenced after that capability's `sdd.design`; makes `sdd.require` produce a `journey` beside the `prd` under the same condition; adds an advisory check at `sdd.design` that reports each Normative Behavior clause no example illustrates.
- Illustrate condition: the capability's Δ names a user-facing surface (UI, conversational skill, operator-facing flow), or grounding finds `features/*.feature` or a BDD runner in the test-runner slot of `grounding/detect-stack.md`. The conductor reads Δ and grounding only; it never asks whether to illustrate.
- Relations: `scenario depends_on spec` (the cascade edge); `scenario implements journey` when a journey exists; `journey related prd`. No edge from `spec` to `scenario`.
- Exit checks at `sdd.illustrate`: blocking — every cited clause number exists in the `spec`; every Flows subsection carries an `Anchors:` line; the body is within the cap or split by actor. Per-gate question maximum: 2.
- Depends on: the registry and package contribution in `delta-routing.md`; the gate contract; the compatibility probe (a failed probe drops the instrument and reports once).
- Recorded exception: `sdd.require` producing a second type (`journey`) is a single-type-production exception on the pattern of `decision.cascade`.

### 3. Scenario evidence in describe and closeout

- Does: `describe.read` records `features/*.feature` files as evidence for Failure Behavior and Conformance; `describe.draft` may produce a `scenario` beside the `spec` (`depends_on` → spec, feature files cited in `Anchors:`); `closeout.verify` reports readiness (each example run, confirmed, or unconfirmed) and coverage (each `spec` clause without an example; each cited feature file absent from the branch; each feature file on the branch that no spec cites); `closeout.accept` names the readiness result when it offers a scenario transition.
- Used by: `document` and `review` users.
- Depends on: the verdict contract for a scenario whose `Anchors:` paths changed in the diff; the probe (a failed probe keeps the legacy scope filter and skips both checks).
- Invariants: both reports are advisory and block nothing; the runtime executes no feature file and no example; no feature file is copied into `.archcore/`.

### 4. Compatibility gate and command entries

- Does: gates the two names on CLI 0.8.4 through `bin/cli-gte 0.8.4`; adds `scenario` and `journey` to the `/archcore:document` argument hint; routes `document scenario` to `describe.read` with the type settled and `document journey` to callable `sdd.require`; adds both types to the grounding filters of `plan`, `document`, and `review` when the probe returns `yes`; names both types and the file in the three agent instruction files.
- Probe trigger: a request or grounding result names either type, or the route engages the illustrate instrument. Otherwise the probe is skipped.
- Fallback: explicit type on a failed probe → report the required version, exit without a write; illustrate on a failed probe → drop the instrument, report once; never convert an existing artifact of either type.
- Depends on: the CLI type contract for old-engine behavior (an older engine skips `.scenario.md` and `.journey.md` in the scan and reports an invalid type in `status`). Relation values are unchanged, so the shared-manifest hazard of the research release does not recur.
- `plan` exposes no new entry; `journey` comes from `sdd.require` and `scenario` from the illustrate instrument.

## Data flow

- `/archcore:plan`: grounding → probe (only under the trigger) → Δ, Π, M, R → package; per capability `sdd.design` → `sdd.illustrate`; `sdd.require` adds the journey under the illustrate condition; `sdd.decompose` records the package in the plan.
- `/archcore:document`: argument hint → probe → `describe.read` (scenario) or callable `sdd.require` (journey) → draft with relations.
- `/archcore:review` closeout: branch state → `closeout.verify` reads the scoped `plan`, its `implements` chain, and every `scenario` and `spec` the diff references → readiness and coverage in the running report → `closeout.merge` and `closeout.accept` under per-document confirmation.

## Error handling

| Condition | Behavior |
|---|---|
| Probe returns `no` or `__NO_CLI__`, explicit type | Report "Scenario and journey require Archcore CLI 0.8.4; skipping actor-subject documents." once; no write. |
| Probe fails, illustrate engaged | Instrument dropped; one report; the rest of the package proceeds. |
| Cited clause number absent from the `spec` | Blocking stop at `sdd.illustrate`; the number is reported. |
| Flows subsection without `Anchors:` | Advisory finding at gate close; the write proceeds. |
| Scenario over 120 lines, actor boundary clear | Split into one document per actor, `related` between parts, each `depends_on` the spec. |
| Scenario over 120 lines, no clear boundary | Kept whole; excess reported; no normative content deleted. |
| `spec` without numbered clauses | Coverage not computable for that spec; reported. |
| No runner and no confirmation | Every example reported as unconfirmed. |
| Server rejects a type after a `yes` probe | Mismatch reported; the operation stops; no retry under another type. |
| Teammate on an older engine | Files skipped in the scan, reported in `status`; the local probe cannot verify teammates. |

## Testing

- Structure: `test/structure/actor-subject-contracts.bats` (contract files exist, canon hooks present) and `test/structure/actor-subject-compat.bats` (argument-hint parity across hosts, agent references, compatibility file references resolve); existing `delta-routing.bats` and `track-goldens.bats` cover the registry row and gate record shapes.
- Unit: existing `test/unit/cli-gte.bats` covers the helper.
- Integration: `test/integration/actor-subject-vocabulary.bats` against a real CLI 0.8.4 stdio MCP — 23 types in the `create_document` enum, `scenario` in knowledge, `journey` in vision, template sections as the CLI contract lists them. The pinned CLI in `Makefile` and the CI workflow rises to 0.8.4.
- Behavioral: two new traces under `test/behavioral/fixtures/` (a user-facing capability; a repository with `features/*.feature`); the 40 existing traces keep their route.
- Negative check: remove the `illustrate` registry row, confirm the structure suite fails, restore by editing.
- Manual: draft one document of each type from the contracts in a scratch project; the CLI hook reports only the placeholder-body finding.

## Canon updates

Four accepted specs receive point edits after the user confirms each one, per the `spec-wrong` verdicts recorded in the plan's Declared Delta: `track-layer.spec` (catalog line), `delta-routing-instruments.spec` (registry entry, 23-type invariant), `command-surface-v2.spec` (`document` expert names), `agent-system.spec` (type lists, probe duty). Two docs are updated without a verdict: `delta-routing-type-engagement.doc` (two rows, count) and `delta-routing-compatibility.doc` (0.8.4 row).

## Out of scope

The docs site, the "Archcore + Cucumber" integration recipe, global shared-context updates on RFC acceptance, per-type body caps for the other 21 types, execution of any scenario, and a merged vocabulary compatibility file.

## Open items

- Release waits for the global RFC to reach `accepted` (adoption step 3). Implementation proceeds before that on the user's instruction of 2026-09-16.
- [assumption] The `v0.8.4` tag is published as a GitHub release; not verified from this repository.
- Coverage per clause reads clause numbers as written; stable clause identifiers (open item of `spec-single-narrative-ears-bcp14.adr`) are not required for this release.
