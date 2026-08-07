---
title: "Magic First-Day Init Implementation Plan"
status: rejected
tags:
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Rewrite `/archcore:init` to deliver a populated, navigable `.archcore/` on first run, per `magic-first-day-init.adr`. Replace the earlier behavior — seeding 0 to 4 documents plus a non-binding propose list — with a single-pass flow of **detect → compose → one preview → one confirm → create and wire relations**, adding fact-extractive document types and confirmed hotspot-spec synthesis.

A pre-implementation prompt-engineering review found the design sound and quantified the cost: a medium-repository run roughly doubles, from about 21.5k to about 46.6k tokens. Its fixes are folded into the flow below — chiefly deferring hotspot source reads behind the confirm gate, splitting detection and composition into separate catalog-loading sub-phases, making the capstone safe under precision Rule 5, and gating `init_project()` correctly.

A later universality pass, recorded below and as task M8, widened the seed beyond DB-backed server applications so it is substantive for any repository shape — scripts, libraries and SDKs, SPAs, ML, mobile, games, and agent-plugin or Markdown-tooling repositories.

## Scope

In scope: the new init `SKILL.md` flow built around two ordered sub-phases plus one preview and one confirm; three new extractive detectors and one capstone composer under `skills/_shared/grounding/`; a new `skills/_shared/rule-contract.md` for the Tier-2 cross-cutting rule bodies; promoting the entry-point inventory and the top-level map to every applicable mode, so scale changes breadth rather than presence; shifting hotspots from a proposed to-do to a stub in the preview and a full `spec` after confirm; folding the agent-file import into the main flow while preserving its high-cost gate; auto-wiring relations among the seeded documents; and closing the deferred test fixtures of `bootstrap-scale-modes.plan`.

Out of scope: per-language deep parsers and call-graph analysis, so detection stays heuristic and data-table driven; auto-running `document` or `plan` for a non-hotspot target, which stays organic; and any change to another skill beyond registration updates.

## The new init flow

| Step | Behavior |
|---|---|
| **Pre-flight** | The CLI check, `init_project()`, and the empty-repo gate, which exits and creates nothing when there is no source. Carve-out: `init_project`, `list_documents`, and `get_document` are read-only infrastructure and run before the confirm gate; only `create_document` and `add_relation` are gated. |
| **Detect** | Load only the detection catalogs — scale, domains, modules, stack, data-model, integrations, config, run-instructions, entry-points, surface, hotspots, cross-cutting, agent-files. Make one filesystem pass, batching the shared manifest reads across the stack, data-model, integrations, and config detectors. Collect signals, emit a one-line progress note, and stop referencing detection catalogs. No composition catalog is loaded yet, and nothing is written. |
| **Compose** | Load the composition catalogs — the precision rules and the spec, rule, overview, and routing contracts. Compose the Tier-1 fact documents in full, which is cheap because they are extractive; compose Tier-2 stubs only, carrying the title plus the qualifying LOC and test ratio, with no source read; compose the capstone; and assemble the planned relation edges. |
| **Preview** | One manifest grouped by tier, one line each. Each Tier-2 stub shows its hotspot path, LOC, test ratio, and estimated synthesis cost. The agent-file import shows its size and cost tier, with the high-cost gate preserved. The manifest closes with the planned relation count and the total token estimate. |
| **Confirm** | `confirm` proceeds; `edit` deselects items and proceeds with the rest; `cancel` fires zero `create_document` and `add_relation` calls, leaving no partial state. |
| **Create and wire** | Create the Tier-1 documents. For each surviving Tier-2 spec, read its source and companion tests *now* and compose the full body under the spec contract, composing the cross-cutting rules under the rule contract. Create the capstone, then add the planned relation edges, rolling forward on an individual failure. |
| **Result** | A closing message: editing a file under a hotspot path auto-injects context; CLI hooks and command grounding pull it on demand (context removed under v2); and `/archcore:review` now runs against a non-empty graph. |

## New detectors

- **`detect-data-model.md`** reads `schema.prisma`, Drizzle schema files, TypeORM entities, Sequelize models, SQLAlchemy declarative bases, Django models, Ecto, ActiveRecord, and GORM structs, `*.sql` migrations, and `*.proto`. It outputs one `doc` listing entities, tables, and key relations by name only, capped at 40 lines.
- **`detect-integrations.md`** maps an allowlist of SDK dependencies to an external service — payments, AWS, telephony, email, LLM providers, Redis, Kafka, and queues. An unknown dependency is omitted rather than guessed. It outputs one `doc` capped at 15 lines.
- **`detect-config.md`** reads `.env.example` and `.env.sample`, config-schema libraries, and settings modules. It opens with a bolded security rule stating that the detector MUST NOT output the value of any environment variable — names and purpose only — with a Bad example. It outputs one `doc` capped at 20 lines.
- **`detect-surface.md`**, added in the universality pass, captures the role-based public surface the entry-point inventory does not cover: web routes and pages, a library's exported API, a multi-command CLI's catalog, an agent plugin's skills and commands, and mobile screens. It carries names plus a one-line purpose, skips when the entry points already enumerate the surface, and is capped at 25 lines.

The four manifest-driven detectors share one manifest parse, so no detector re-reads `package.json` or a schema file; `detect-surface` reads the route, export, command, and skill declaration sites instead.

## Capstone composer

`compose-overview.md` assembles the `architecture-overview` doc from a structural-facts orientation line — module and domain counts, language, framework, ORM, all extracted rather than free prose — plus an index table keyed by document type and topic rather than by `.archcore/` path, since precision Rule 5 keeps links in the relation graph alone. It emits a `related` edge from the overview to each seeded document and is capped hard at 150 lines.

## Synthesis contracts

Add `skills/_shared/rule-contract.md`, covering the applies-to scope, the normative statements, and enforcement, so a Tier-2 cross-cutting rule has a body contract analogous to the spec contract. Compose every Tier-2 hotspot artifact as a `spec` only: the type hints inside `detect-hotspots.md` serve solely to filter candidates, dropping those that fail the spec contract's "when NOT to write a spec" gate, and never to switch the document type.

## Universality follow-up

A later pass widened init beyond DB-backed server applications so the seed is substantive for any repository shape, including this plugin, on which init dogfoods.

- **The public-surface fact** is a Tier-1 extractive document carrying the seed for a library, SPA, or plugin repository with no server to enumerate.
- **The hotspot test-independent fallback** gives ranking a tests-aware primary tier and a fallback tier over fan-in, public surface, size, and churn, which fills the top-N when no tests exist so a test-less repository still gets real specs instead of an empty pool. Primary behavior is unchanged where tests are present.
- **The instruction-modules exception** counts instruction files as modules, narrowly, when a repository's product *is* prompt or instruction content — an agent or LLM plugin, or Markdown tooling with skills, commands, agents, a plugin manifest, and little traditional source — so scale and hotspots stay meaningful. An ordinary repository that merely contains documentation is unaffected.
- **Agent-plugin recognition** in the stack detector and the source-signal gate covers the plugin manifests.
- Detectors generalized in this pass lead concept-first with non-exhaustive example lists and a positive-evidence guardrail, which `@test/structure/init-skill.bats` guards against regression.

## Mode breadth

Small, medium, and large scale how much is seeded rather than whether. Tier 1 facts appear in every mode when detected. The Tier-2 hotspot spec count is 3 for small, 5 for medium, and 3 per selected domain for large, and large mode keeps its domain-selection dialog before composing stubs.

## Token-control measures

The stub-before and body-after-confirm split defers about 9.9k input tokens on a medium repository behind the gate, and a deselected spec's source is never read. The two-sub-phase catalog loading never holds the full catalog set at once, since the composition contracts load only after detection completes. The manifest batch parse reads the shared manifests once across four detectors. The per-document line caps, with the capstone at 150 lines, keep a created document from padding toward the blob threshold. And the informed deselect shows per-item synthesis cost in the preview, so `edit` is a real budget lever rather than a blind toggle.

## Tasks

**M1 — new extractive detectors.**
- [ ] Add `detect-data-model.md` with multi-ORM coverage, fixtures, and its 40-line cap.
- [ ] Add `detect-integrations.md` with its allowlist, fixtures, and 15-line cap.
- [ ] Add `detect-config.md` with fixtures, its 20-line cap, the bolded never-values security rule, and a Bad example.
- [ ] Add the manifest-batch read step shared across the four manifest-driven detectors.

**M2 — capstone and wiring.**
- [ ] Add `compose-overview.md` with its type-and-topic index, its structural-facts orientation, and its 150-line cap, enumerating no paths.
- [ ] Add the relation-wiring rules table: overview to all, data-model to integrations, specs to the top-level map and entry points, and imported rules to the stack rule.

**M3 — Tier-2 synthesis, split into stub and body.**
- [ ] Compose the stubs before the preview from the hotspot LOC and test ratio, with no source read.
- [ ] Compose the full bodies after confirm under the spec contract, skipping deselected items.
- [ ] Add the rule contract and route the Tier-2 cross-cutting rules through it.
- [ ] Enforce specs-only, using the hotspot type hints to filter rather than to switch type.

**M4 — the `SKILL.md` rewrite.**
- [ ] Implement the pre-flight carve-out, leaving `init_project` and the read-only calls ungated while gating creation and relations.
- [ ] Implement detect and compose as ordered sub-phases with explicit catalog-load boundaries, preserving the lazy-reading discipline.
- [ ] Implement the preview manifest format with per-item cost lines and the three confirm outcomes.
- [ ] Guarantee zero creation and relation calls on `cancel`.

**M5 — the agent-file import in the main flow.**
- [ ] Move the import into compose and preserve the high-cost gate as a preview line item.

**M6 — tests, closing the deferred item of `bootstrap-scale-modes.plan`.**
- [ ] Add fixtures for a small TypeScript SDK, a medium service, a large pnpm monorepo, a small Python repository, and a small Go repository.
- [ ] Add `test/structure/init-modes.bats` covering the preview manifest and the seeded-document set per mode.
- [ ] Add the security test asserting that the config document contains no environment-variable value.

**M7 — documentation and specification updates.**
- [ ] Update `commands-system.spec`, `skills-system.spec`, and `plugin-architecture.spec` for the new init contract.

**M8 — the universality pass. Done.**
- [x] Add `detect-surface.md` with its 25-line cap, wired into detect, compose, preview, create, and both the overview index and its relations.
- [x] Add the hotspot test-independent fallback tier.
- [x] Add the instruction-modules exception and agent-plugin manifest recognition.
- [x] Extend the structure test guards to cover surface and hotspots.

## Acceptance Criteria

1. On a small TypeScript SDK, init previews and then, on confirm, creates the stack rule, the run guide, the data-model, integrations, and config documents where detected, the architecture overview, and up to the per-mode hotspot specs, all with relations.
2. `confirm` creates the whole set; `edit` removes the selected items and never reads a deselected spec's source; and `cancel` fires zero creation and relation calls, though `init_project` may have run.
3. After init, command grounding (formerly `/archcore:context`, removed under v2) surfaces the generated spec, the graph is non-empty, and editing a matching file injects context.
4. No created document exceeds 200 lines, the capstone stays within 150, the capstone body enumerates no `.archcore/` path, and `bin/check-precision` raises no error on the seeded synthesis.
5. The config document carries variable names and purpose only, with no values, and the security test passes.
6. The empty-repo gate still exits without creating anything, and every seed is idempotent on re-run.
7. The agent-file import runs in the main flow, appears in the preview, and respects the high-cost gate.
8. A library, SPA, or agent-plugin repository with no tests still seeds a public-surface document and at least one hotspot spec through the fallback tier, rather than an empty Tier-2 pool.

## Dependencies

- `magic-first-day-init.adr`, which this plan implements.
- `bootstrap-scale-modes.plan`, which this extends, reusing its scale detection and existing detector catalogs.
- The spec contract and precision rules as synthesis contracts, plus the new rule contract added here.
- The existing detectors for scale, stack, domains, modules, entry points, surface, hotspots, cross-cutting rules, agent files, and extract routing.
- The pre-implementation prompt-engineering review, for the token budget and prompt-quality findings.

## Risks

- **Synthesis quality** is mitigated by the single-confirm preview with deselection, so a weak spec is dropped before its source is read.
- **Roughly double token cost on a medium repository** is mitigated by the stub-and-body split keeping peak spend behind the gate, the per-mode Tier-2 cap, and the per-item cost shown in the preview.
- **Detector false positives** in the data-model and integrations detectors are mitigated by allowlist-driven conservatism, omitting an unknown dependency rather than guessing.
- **Config value leakage into git** is mitigated by the bolded security rule and the dedicated security test.
- **Lazy-reading regression** is prevented by the detect-and-compose sub-phase boundary, since the composition contracts are not loaded during detection.
- **Preview overwhelm on a large repository** is mitigated by grouping by tier, capping the hotspot stubs, and summarizing the import yield.
- **Universality over-fit and a false surface** are mitigated because `detect-surface` is role-based and skips when no named surface exists or the entry points already cover it, the hotspot fallback leaves the tests-aware primary tier unchanged, and the instruction-modules exception is gated on positive evidence across all three signals.
