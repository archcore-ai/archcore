---
title: "Command Entry Grammar — Mode Words on Four Commands and Type Selection at Gates"
status: draft
tags:
  - "commands"
  - "plugin"
  - "skills"
---

## Goal

Move the four commands to one entry form, `/archcore:<command> [mode] [subject]`, with the document type selected at a gate. The linked decisions own the choice; `command-surface-v2.spec` and `track-layer.spec` own the behavior. The change ships in the same release as the scenario and journey runtime, because both edit the `document` entry surface.

Grounding used branch `dev`, HEAD `6c235ce`, with the scenario and journey runtime uncommitted in the working tree, on 2026-09-16. During implementation the maintainer committed that runtime as `d408a51`; this plan's changes stay uncommitted on top of it. The first MCP server of the session ran an earlier binary and rejected the type `scenario`; after the maintainer restarted it on CLI 0.8.4, the scenario write succeeded.

## Declared Delta

| Field | Value |
|---|---|
| `creates` | None |
| `modifies` | command entry surface — `command-surface-v2.spec`, verdict `spec-wrong`; gate-level type selection and the executor map — `track-layer.spec`, verdict `spec-wrong`; journey production — `delta-routing-instruments.spec`, verdict `spec-wrong`; routing table and review modes — `plugin-architecture.spec`, verdict `spec-wrong`; drift mode name — `actualize-system.spec`, verdict `spec-wrong`; actor-subject entries — `actor-subject-compatibility.spec` (draft), verdict `spec-wrong`; register rows 17, 19, 25 — `track-handoffs.spec` (draft), verdict `spec-wrong`. The user confirmed the accepted edits on 2026-09-16; all seven are applied. |
| `retires` | the entries `document adr`, `rfc`, `rule`, `spec`, `doc`, `guide`, `research`, `evidence`, `scenario`, `journey` as type names; `review --drift`, `review --deep`; `init --refresh`, `init --domain=<slug>`, `init --mode=<scale>`; the five route names as `plan` entries; the callable `sdd.require` entry from `document` |
| `decision` | `command-entry-grammar.adr`; `journey-produced-only-on-plan.adr`; route names dropped as entries, confirmed by the user on 2026-09-16 |
| `intent_gap` | No |
| Π | `machine`: the argument hints in the four `SKILL.md` files, `delta-routing.md` Expert invocation map, the gate records of `decision.md`, `describe.md`, `research.md`, the 2026-09-07 revision recorded in the global RFC `concepts/research-and-evidence-types`; `user`: route names, confirmation of accepted edits |
| M | `stone`: accepted `command-surface-v2.spec`, `track-layer.spec`, `plugin-architecture.spec` cover the zone |
| R | `external-contract`: the user-facing command surface; recorded invocations break |
| Route | `amendment`, base S, raised once to M by `stone` and `external-contract` |
| Instruments | decision; verdicts on the seven `modifies` entries; illustrate (the command surface is a conversational skill surface) — `command-entry-grammar.scenario`; decompose |

## Tasks

### Phase 1 — records

1. [x] Restart the MCP server on CLI 0.8.4, then create `plugin/command-entry-grammar.scenario.md` with `depends_on` → `command-surface-v2.spec`.
2. [x] Update the accepted docs through `update_document`: `delta-routing-type-engagement.doc`, `delta-routing-compatibility.doc` (row 4, new row 9, actor-subject rows), `component-registry.doc` (skills table, examples), `skill-file-structure.rule` (items 19–21, hint examples).
3. [x] Set status `rejected` on `document-journey-callable-intent-entry.adr` (user confirmed on 2026-09-16) and remove the `implements` edge from `actor-subject-compatibility.spec`.
4. [x] Update `research-runtime-category-and-evidence-entry.adr` (draft): name `document research` with one material as the evidence entry.

### Phase 2 — command surface

5. [x] Set the argument hints in `@plugins/archcore/skills/document/SKILL.md` and `@plugins/archcore/commands/document.md` to `[decision|code|research] [subject]`; rewrite the routing table, Step 2 as a mode map, and the description.
6. [x] Set the hints in `@plugins/archcore/skills/plan/SKILL.md` and `@plugins/archcore/commands/plan.md` to `[sdd|sources|iso|research] [topic]`; remove the document-type clause of step 2.
7. [x] Set the hints in `@plugins/archcore/skills/review/SKILL.md` and `@plugins/archcore/commands/review.md` to `[drift|deep|closeout|experience] [path, tag, or scope]`; replace `--drift`, `--deep`, and the named-track row.
8. [x] Set the hints in `@plugins/archcore/skills/init/SKILL.md` and `@plugins/archcore/commands/init.md`; replace `--refresh` and `--domain` in the skill, `@plugins/archcore/skills/_shared/grounding/detect-domains.md`, and `@plugins/archcore/skills/_shared/grounding/detect-hotspots.md`.
9. [x] Edit `@plugins/archcore/skills/_shared/delta-routing.md`: remove route names from the Expert invocation map; name `document research` as the evidence entry.

### Phase 3 — gates

10. [x] Edit `decision.classify` in `@plugins/archcore/skills/_shared/tracks/decision.md`: add the standard outcome per track-layer behaviors 33 and 34.
11. [x] Edit `research.frame` in `@plugins/archcore/skills/_shared/tracks/research.md`: route one supplied material to gather.
12. [x] Edit `describe.draft` in `@plugins/archcore/skills/_shared/tracks/describe.md`: a type name in the subject settles the type; no `journey`.
13. [x] Edit `@plugins/archcore/skills/_shared/tracks/sdd.md`: remove the callable `document journey` entry from `sdd.require`.
14. [x] Edit `@plugins/archcore/skills/_shared/tracks/actualize.md` and `@plugins/archcore/skills/_shared/tracks/requirements-cascade.md`: replace flag and expert-invocation wording.
15. [x] Remove `<track>.<stage>` addresses from the Result sections of the `plan`, `document`, and `review` skills; `init` runs no track.

### Phase 4 — shared text, agents, docs

16. [x] Edit `@plugins/archcore/skills/_shared/actor-subject-compatibility.md`, `@plugins/archcore/skills/_shared/research-compatibility.md`, and `@plugins/archcore/skills/_shared/journey-contract.md`: drop the retired entries.
17. [x] Check the three agent files: they name no command entry; no edit was needed.
18. [x] Edit `@README.md` and `@.claude/skills/verify-plugin-integrity/SKILL.md`; `@.claude/skills/bump-plugin-version/SKILL.md` carries no entry form.

### Phase 5 — tests

19. [x] Rewrite the entry rows of `@test/fixtures/routing/fixtures.tsv` to mode words; add one row per `review` mode and per `init` mode.
20. [x] Update `@test/structure/actor-subject-compat.bats`, `@test/structure/agent-contracts.bats`, `@test/structure/delta-routing.bats`, `@test/structure/research-track.bats`, and `@test/integration/research-agent.bats`; `@test/unit/session-start-emit-matrix.bats` matched only a comment.
21. [x] Add `@test/structure/command-grammar.bats`: mode lists, no type or route name as a mode, flags only as settings, retired forms, the gate-address rule, the mode-to-track maps, and the standard branch of `decision.classify`.
22. [x] Regenerate `@test/fixtures/goldens/decision.golden`, `@test/fixtures/goldens/research.golden`, and `@test/fixtures/goldens/sdd.golden`.
23. [x] Add `@test/behavioral/document-bench.sh` with fixtures in `@test/behavioral/fixtures/document-bench.tsv`, its harness test `@test/unit/document-bench.bats`, and the target `make test-document-bench`; record the result below.
24. [x] Run `make all` and the integration suite against CLI 0.8.4.

### Phase 6 — prompt review follow-ups

25. [x] Rename the `init` setting `--mode` to `--scale` in `@plugins/archcore/skills/init/SKILL.md`, `@plugins/archcore/commands/init.md`, and `@plugins/archcore/skills/_shared/grounding/detect-scale.md`.
26. [x] Add the description-parity test to `@test/structure/command-grammar.bats`: each command and skill description names every mode of its hint.
27. [x] Add a no-arguments row to the `plan` and `document` routing tables and a plain-run sentence to `init`; pin all four in `@test/structure/command-grammar.bats` and add bench row 21.

### Phase 7 — host skill selection

28. [x] Add `@test/behavioral/skill-bench.sh` with fixtures in `@test/behavioral/fixtures/skill-bench.tsv`, its harness test `@test/unit/skill-bench.bats`, and the target `make test-skill-bench`.
29. [x] Separate `sources` from `research` in the `plan` descriptions of `@plugins/archcore/skills/plan/SKILL.md` and `@plugins/archcore/commands/plan.md`.

## Verification record — 2026-09-16

- `make all`: 613 tests pass after tasks 25–29. `make test-integration` on CLI 0.8.4: 16 of 16 pass, twice.
- `archcore doctor` after the scenario write: all checks pass, 646 relations.
- Test health check in an isolated worktree, 21 fault probes plus 5 re-probes, one change per probe, bytes restored and the suite green after each: every entry-surface probe was detected. Three semantic probes first survived — `code` mapped to the decision track, a gate address in the `plan` Result section, and the standard-branch condition of `decision.classify` inverted. `command-grammar.bats` tests 5–8 were added, and all three probes are now detected. A `sdd.md` of exactly 300 lines passes and 301 lines fails the cap test. A description without the `closeout` mode fails the parity test.
- Not probed: `@test/integration/research-agent.bats` runs a live model; `@test/integration/actor-subject-vocabulary.bats` pins the external CLI and had baseline and reliability runs only.
- Document bench on the session's default model: 19 of 20 fixtures on the first run; 13 of 14 requests without a mode word classified correctly. The miss, row 4 ("we accepted the proposal" with an `rfc` draft), returned `rfc`; after the `document` mode table named the `adr` that `decision.resolve` records, row 4 passed 3 of 3 runs. Row 21, an invocation with no arguments, passed 2 of 2 runs.
- Plan route bench after the change: 44 of 44 fixtures pass.
- Host skill-selection bench (`claude -p` with the plugin loaded, only the Skill tool, hooks off, built-in skills competing): 23 of 23 messages that name no command selected the expected skill on the first run, including 3 negative messages that selected no archcore skill. With the mode word also checked, row 3 ("I need market research before we plan") passed `research` instead of `sources`; after task 29 it passed `sources` 3 of 3 runs. Row 12 referred to an attached file that the message did not carry and selected no skill in 1 of 3 runs; the fixture now carries the material inline. The final full run passed 23 of 23 with modes checked on 11 rows.
- Prompt review of the skill text: the definitions of expert invocation, a request that names a type, and investigation versus one external material moved into `@plugins/archcore/skills/_shared/gate-contract.md`; `plan` and `document` descriptions now separate a new investigation from a finished report; `decision.classify` states the standard branch as two positive conditions and asks when two or more local `adr` documents match.
- Limits: every bench ran on one model, once per fixture except the reruns named above; the skill bench measures Claude Code only, not Cursor, Codex, or Copilot routing.

## Acceptance Criteria

- `make all` and the integration suite pass on CLI 0.8.4.
- A search of `plugins/` for `document adr`, `document evidence`, `document journey`, `--drift`, `--deep`, `--refresh`, `--domain`, and `--mode` returns no entry usage.
- The document bench and the host skill-selection bench results are recorded in the verification record.
- The release notes name every retired invocation form.

## Dependencies

- The scenario and journey runtime plan: this plan edits the same entry surface and ships in the same release.
- The global RFCs in the `archcore` global source stay unchanged; the plugin departs from their entry clauses, as `command-entry-grammar.adr` records.
