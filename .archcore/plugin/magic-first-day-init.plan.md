---
title: "Magic First-Day Init Implementation Plan"
status: accepted
tags:
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Rewrite `/archcore:init` to deliver a populated, navigable `.archcore/` on first run, per `magic-first-day-init.adr`. Replace the current "seed 0–4 docs + non-binding propose list" behaviour with a single-pass flow — **detect → compose → one preview → one confirm → create + wire-relations** — adding fact-extractive document types and confirmed hotspot-spec synthesis.

A pre-implementation prompt-engineering review (ai-kit:prompt-engineer) found the design sound and quantified cost: a medium-repo run roughly doubles (~21.5k → ~46.6k tokens). Its fixes are folded into the flow and tasks below — chiefly: defer hotspot source reads behind the confirm gate (stubs in preview, bodies after confirm), split detection/composition into separate catalog-loading sub-phases, make the capstone Rule-5-safe, and gate `init_project()` correctly.

A later **universality pass** (recorded under "Universality follow-up" and task M8) widened the seed beyond DB-backed server apps so it is substantive for any repo shape — scripts, libraries/SDKs, SPAs, ML, mobile, games, and agent-plugin / Markdown-tooling repos.

## Scope

In scope:

- New init `SKILL.md` flow built around two ordered sub-phases (detect, compose), one preview, and one confirm.
- Three new extractive detectors + one capstone composer under `skills/init/lib/`.
- A new `skills/_shared/rule-contract.md` for Tier-2 cross-cutting rule bodies.
- Promote entry-point inventory and top-level map to all applicable modes (scale breadth, not presence).
- Shift hotspots from "propose a to-do" to "stub in preview → compose full `spec` after confirm."
- Fold agent-file import into the main flow, preserving the HIGH-cost gate.
- Auto-wire relations among seeded documents.
- Close the deferred test fixtures (M7 of `bootstrap-scale-modes.plan`).

Out of scope:

- Per-language deep parsers / call-graph analysis (detection stays heuristic, data-table driven).
- Auto-running `capture` / `decide` / `plan` for non-hotspot targets — those remain organic.
- Changes to other skills beyond spec/doc registration updates.

## New init flow

| Step | Behaviour |
|---|---|
| **Pre-flight** | CLI check + `init_project()` + empty-repo gate (no source → exit, create nothing). **Carve-out:** `init_project`, `list_documents`, `get_document` are read-only/infra and run before the confirm gate; only `create_document` / `add_relation` are gated. |
| **Detect (sub-phase)** | Load ONLY detection catalogs (scale, domains, modules, stack, **data-model, integrations, config**, run-instructions, entry-points, **surface**, hotspots, cross-cutting, agent-files). One filesystem pass — batch the shared manifest reads (`package.json`, `schema.prisma`, `.env.example`) across stack/data-model/integrations/config. Collect signals, emit a one-line progress note, then stop referencing detection catalogs. No composition catalogs loaded yet. No writes. |
| **Compose (sub-phase)** | Load composition catalogs (`precision-rules`, `spec-contract`, `rule-contract`, `compose-overview`, `extract-routing`). Compose Tier-1 fact docs in full (cheap, extractive); compose Tier-2 **stubs only** (title + qualifying LOC/test-ratio — NO source reads); compose the capstone; assemble the planned relation edges. |
| **Preview** | One manifest, grouped by tier, one line each. For each Tier-2 stub: hotspot path + LOC + test-ratio + estimated synthesis cost (e.g. "~3.3k tokens to read source + tests"). For the agent-file import: size + cost tier (HIGH gate preserved). + planned relation count + total token estimate. |
| **Confirm** | `confirm` → proceed; `edit` → deselect items, then proceed with the rest; `cancel` → zero `create_document`/`add_relation` calls (no partial state). |
| **Create + wire** | Create Tier-1 docs. For each surviving Tier-2 spec, **now** read its source + companion tests and compose the full body under `spec-contract`; compose cross-cutting rules under `rule-contract`. Create the capstone. Then `add_relation` for the planned edges. Roll forward on individual failure. |
| **Result** | Closing message: edit a file under a hotspot path → context auto-injects; `/archcore:context <path>`; `/archcore:audit` for the now-non-empty graph. |

## New detectors (`skills/init/lib/`)

- **`detect-data-model.md`** — sources: `schema.prisma`, Drizzle schema files, TypeORM `@Entity`, Sequelize models, SQLAlchemy `declarative_base`, Django `models.Model`, Ecto/ActiveRecord/GORM structs, `*.sql` migrations, `*.proto`. Output: one `doc` listing entities/tables + key relations (names only). **≤ 40 lines.**
- **`detect-integrations.md`** — allowlist mapping SDK deps → external service (e.g. `stripe`→payments, `@aws-sdk/*`→AWS, `twilio`, `@sendgrid/*`, `openai`/`@anthropic-ai/sdk`→LLM, `ioredis`/`redis`→Redis, `kafkajs`→Kafka, `bullmq`→queue). Unknown deps are omitted, never guessed. Output: one `doc` "External integrations." **≤ 15 lines.**
- **`detect-config.md`** — `.env.example` / `.env.sample`, config-schema libs (`zod`, `convict`, `envalid`), settings modules. **Opens with a bolded security rule: MUST NOT output the value of any environment variable — names and purpose only — with a Bad example.** Output: one `doc` of the config surface. **≤ 20 lines.**
- **`detect-surface.md`** (universality pass) — the role-based public surface the entry-point inventory doesn't cover: web routes/pages, a library's exported API, a multi-command CLI's command catalog, an agent-plugin's skills/commands, mobile screens. Names + one-line purpose only; skips when entry points already enumerate it. Output: one `doc` "Public surface." **≤ 25 lines.**

The manifest-driven detectors (stack, data-model, integrations, config) share one manifest parse (see Detect sub-phase) to avoid re-reading `package.json` / `schema.prisma` per detector; `detect-surface` reads route/export/command/skill declaration sites instead.

## Capstone composer

- **`compose-overview.md`** — assembles the `architecture-overview` doc: a structural-facts orientation line (module/domain counts, language, framework, ORM — extracted, never free prose) + an index table keyed by **document type and topic, not `.archcore/` paths** (Rule 5: links live only in the relation graph). Emits `related` edges from the overview to each seeded doc via `add_relation`. **Hard cap ≤ 150 lines.**

## Synthesis contracts

- Add **`skills/_shared/rule-contract.md`** (Applies-to scope, MUST / MUST NOT statements, Enforcement) so Tier-2 cross-cutting rules have a body contract analogous to `spec-contract.md`.
- Tier-2 hotspot artifacts are composed as **`spec` only**. The `adr` / `task-type` hints in `detect-hotspots.md` are used solely to *filter* candidates (drop those failing `spec-contract`'s "when NOT to write a spec" — e.g. internal helpers with no external consumers), never to switch the document type.

## Universality follow-up (post-implementation)

A later pass widened init beyond DB-backed server apps so the seed is substantive for **any** repo shape — plain script repos, libraries/SDKs, SPAs, ML, mobile, games, and agent-plugin / Markdown-tooling repos (init dogfoods on this very plugin). Changes:

- **Public-surface fact (`detect-surface.md`)** — a Tier-1 extractive fact (above): the outward shape entry points don't cover. It is what carries the seed for library / SPA / plugin repos with no server to enumerate.
- **Hotspot test-independent fallback (`detect-hotspots.md`)** — ranking now has a tests-aware **primary** tier and a **fallback** tier (fan-in / public surface / size / churn) that fills the top-N when no tests exist, so test-less repos (scripts, SPAs, ML, CLIs, agent-plugin tooling) still get real specs instead of an empty pool. Primary behaviour is unchanged when tests are present.
- **Instruction-modules exception (`detect-modules.md`)** — narrowly, when a repo's product IS prompt/instruction content (agent/LLM plugin or Markdown tooling: skills/commands/agents, a plugin manifest, little traditional source), those instruction files count as modules so scale + hotspots are meaningful. Ordinary repos that merely contain docs are unaffected.
- **Agent-plugin recognition** — `detect-stack.md` and the Step 0(b) source-signal gate recognize agent/LLM-plugin manifests (`marketplace.json` / `plugin.json` / `.claude-plugin/*`).
- Detectors generalized in this pass lead concept-first with non-exhaustive example lists and a positive-evidence guardrail; `test/structure/init-skill.bats` guards that invariant against regression for `detect-surface` and `detect-hotspots` too.

## Mode breadth

Small / medium / large still scale *how much* is seeded, not *whether*. Tier 1 facts appear in every mode when detected. Tier-2 hotspot-spec count: **small 3 / medium 5 / large 3 per selected domain** (the existing top-N). Large mode keeps the domain-selection dialog before composing stubs.

## Token-control measures (from the review)

- **Stub-before / body-after-confirm** — defers ~9.9k input tokens (medium repo) behind the gate; deselected specs never read source.
- **Two-sub-phase catalog loading** — never hold the full ~12.7k tokens of catalogs at once; composition contracts load only after detection completes.
- **Manifest batch parse** — one read of shared manifests across the four manifest-driven detectors.
- **Per-doc line caps** + capstone ≤ 150 lines, so created docs can't pad to the blob threshold.
- **Informed deselect** — the preview shows per-item synthesis cost so `edit` is a real budget lever, not a blind toggle.

## Tasks

### M1 — New extractive detectors
- [ ] `detect-data-model.md` (multi-ORM) + fixtures, ≤ 40-line cap.
- [ ] `detect-integrations.md` allowlist + fixtures, ≤ 15-line cap.
- [ ] `detect-config.md` + fixtures, ≤ 20-line cap, with the bolded "never values" security rule and a Bad example.
- [ ] Manifest-batch read step shared across stack/data-model/integrations/config.

### M2 — Capstone + wiring
- [ ] `compose-overview.md`: type/topic index (no path enumeration, Rule-5-safe), structural-facts orientation, ≤ 150-line cap.
- [ ] Relation-wiring rules table (overview→all; data-model→integrations; specs→top-level-map/entry-points; imported rules→stack rule).

### M3 — Tier-2 synthesis (stub + body split)
- [ ] Stub composition (pre-preview, no source reads) from hotspot LOC/test-ratio.
- [ ] Full-body composition (post-confirm) under `spec-contract`, skipping deselected items.
- [ ] Add `skills/_shared/rule-contract.md`; route Tier-2 cross-cutting rules through it.
- [ ] Specs-only rule: use detect-hotspots type hints to filter, not to switch type.

### M4 — SKILL.md rewrite
- [ ] Pre-flight carve-out (`init_project`/read-only ungated; create/relation gated).
- [ ] Detect and Compose as ordered sub-phases with explicit catalog-load boundaries (preserve lazy-reading note).
- [ ] Preview manifest format with per-item cost line items + `confirm`/`edit`/`cancel` semantics.
- [ ] Pre-confirm guarantee: zero `create_document`/`add_relation` on `cancel`.

### M5 — Agent-file import in main flow
- [ ] Move import into Compose; **preserve the HIGH-cost gate** (> 50 KB OR > 5 files OR yield > 8) as a preview line item.

### M6 — Tests (closes deferred M7 of `bootstrap-scale-modes.plan`)
- [ ] Fixtures: small TS SDK, medium service, large pnpm monorepo, small+Python, small+Go.
- [ ] `test/structure/init-modes.bats`: preview manifest + seeded-doc set per mode.
- [ ] Security test: the config doc created by `detect-config` contains no environment-variable values.

### M7 — Doc/spec updates
- [ ] Update `commands-system.spec`, `skills-system.spec`, `plugin-architecture.spec` for the new init contract.

### M8 — Universality pass (done)
- [x] `detect-surface.md` (routes / exported API / CLI commands / plugin skills / mobile screens), ≤ 25-line cap; wired into Detect, Compose, Preview, Create, and the overview index + relations.
- [x] Hotspot test-independent fallback tier in `detect-hotspots.md` (fan-in / public surface / size / churn).
- [x] Instruction-modules exception in `detect-modules.md`; agent-plugin manifests in `detect-stack.md` + Step 0(b) source gate.
- [x] `init-skill.bats` guards: surface in Output / foundation / idempotency-tags / universality; hotspots in universality.

## Acceptance Criteria

1. On a small TS SDK, init previews then (on `confirm`) creates: stack rule, run guide, data-model/integrations/config where detected, architecture-overview, and up to the per-mode hotspot specs — all with relations.
2. `confirm` creates the whole set; `edit` removes selected items (and a deselected Tier-2 spec's source file is never read); `cancel` fires zero `create_document`/`add_relation` calls (`init_project` may have run).
3. After init, `/archcore:context <hotspot path>` returns the generated spec; the graph is non-empty; editing a matching file injects context.
4. No created document exceeds 200 lines (capstone ≤ 150); the capstone body enumerates no `.archcore/` paths; `bin/check-precision` raises no errors on the seeded synthesis.
5. The config doc contains variable names + purpose only, no values (security test passes).
6. The empty-repo gate still exits without creating anything; all seeds idempotent on re-run.
7. `CLAUDE.md` / `AGENTS.md` import runs in the main flow, appears in the preview, and respects the HIGH-cost gate.
8. (Universality) A library / SPA / agent-plugin repo with no tests still seeds a public-surface doc and at least one hotspot spec via the fallback tier — not an empty Tier-2 pool.

## Dependencies

- `magic-first-day-init.adr` — this plan implements it.
- `bootstrap-scale-modes.plan` — extended; reuses scale detection and the existing `detect-*` catalogs.
- `skills/_shared/spec-contract.md`, `precision-rules.md` — synthesis contracts; **new `rule-contract.md`** added here.
- Existing detectors: scale, stack, domains, modules, entry-points, **surface**, hotspots, cross-cutting, agent-files, extract-routing.
- ai-kit:prompt-engineer pre-implementation review (token budget + prompt-quality findings).

## Risks

- **Synthesis quality** → single-confirm preview + `edit`/deselect; weak specs dropped before their source is read.
- **Token cost ~2× on medium repos** → stub/body split keeps peak spend behind the gate; Tier-2 capped per mode; per-item cost shown in preview.
- **Detector false positives** (data-model / integrations) → allowlist-driven and conservative; unknown deps omitted, not guessed.
- **Config value leakage into git** → bolded `detect-config` security rule + dedicated security test (M6).
- **Lazy-reading regression** → enforced by the detect/compose sub-phase boundary; composition contracts not loaded during detection.
- **Preview overwhelm on large repos** → group by tier, cap hotspot stubs, summarize import yield.
- **Universality over-fit / false surface** → `detect-surface` is role-based and skips when no named surface exists or entry points already cover it; the hotspot fallback keeps the tests-aware primary tier unchanged; the instruction-modules exception is positive-evidence-gated on all three signals.

## Relations

- `implements` → `magic-first-day-init.adr`
- `extends` → `bootstrap-scale-modes.plan`
- `related` → `zero-content-onboarding-implementation.plan`
- `related` → `precision-over-coverage.adr`
