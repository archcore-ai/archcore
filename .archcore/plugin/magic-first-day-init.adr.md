---
title: "Magic First-Day Init — Extractive Facts and Confirmed Synthesis"
status: accepted
tags:
  - "architecture"
  - "onboarding"
  - "plugin"
  - "precision"
  - "skills"
---

**Update (2026-07-01).** A real-world run on a 773-module, 24-domain Python-FastAPI monolith showed the tiered design under-seeding at scale: `light`, then the default, produced only 16 documents — 3 hotspot specs and zero cross-cutting rules — on a repository with strong, extractable conventions. Three reversals, detailed in `SKILL.md`, `skills/init/lib/detect-hotspots.md`, and `skills/init/lib/detect-cross-cutting.md`, keep the tiering intact and change its numbers. First, cross-cutting synthesis now runs at every depth — `light` at 2 candidates or fewer, `standard` at 3, `deep` at 4 — instead of being skipped at `light`, because it is the highest value-per-token artifact and was being dropped exactly when it mattered most. Second, large mode's flat repo-wide spec cap is replaced by a per-selected-domain floor plus repo-wide-rank fill, scaling with how many domains the day-one dialog focuses on, and data-model breadth now covers every schema-bearing domain rather than only the dialog-selected ones. Third, the default depth changed from `light` to `standard`, because init is fully gated — nothing is written before `confirm`, and every depth's cost is shown before the user commits — so the default should be the good first-day seed rather than the cheapest one, with `light` as the explicit cost-conscious opt-down. A size and churn-gated flagship spec tier was also added for very large or hot hotspots, and init-synthesized hotspot specs now consistently get `status: draft`. The tier boundaries, the single-preview and single-confirm gate, and the line caps are unchanged.

## Context

`no-auto-generated-context.adr`, now superseded, closed the door on any LLM-scan-generated document, citing arxiv 2510.21413, where auto-generated `AGENTS.md`-style context files reduced agent task success in 5 of 8 settings and raised inference cost by 20–23%. Its designed-in consequence was an `/archcore:init` that seeded 0 to 4 tiny documents and surfaced everything else as a non-binding to-do list it refused to auto-execute. User feedback was consistent: init "shows poorly" and "creates very little content" — on install into an existing repository the first session produced almost nothing an agent could not read from `package.json` in 30 seconds, push and pull both visibly no-opped, and the relation graph stayed empty, so the funnel from install to first useful state leaked exactly where retention is decided.

The reconciliation that stays clear of the anti-pattern is that the cited research penalized unconfirmed wholesale LLM prose and oversized files, not the extraction of facts and not human-confirmed artifacts — human-curated context was net positive in the same study. The measured failure mode is the blob rather than the act of seeding. A pre-implementation prompt-engineering review confirmed the design and raised four structural fixes, folded into the decision below, and quantified the cost: a medium-repository run roughly doubles, from about 21.5k to about 46.6k tokens, or about $0.11 to $0.26 at Sonnet pricing, dominated by reading hotspot source files for spec synthesis.

## Decision

`/archcore:init` populates `.archcore/` in a single pass — **detect → compose → one preview → one confirm → create and wire relations** — with detection and composition as ordered sub-phases that load only the catalogs each needs, and with what may be composed bounded by three tiers.

**Tier 1 — extractive facts, auto-composed in full.** Derived mechanically from manifests, schemas, routing, and config rather than synthesized as prose: the stack rule, the run guide, the entry-point inventory, the top-level map, the data-model doc from ORM schemas, migrations, and `*.proto`, the integrations doc mapping third-party SDK dependencies to external services, the config and env doc carrying variable names and purpose but **never values**, which is a security boundary, and the public-surface doc capturing the role-based outward shape that entry points miss — web routes, a library's exported API, a multi-command CLI's commands, an agent plugin's skills and commands, mobile screens — so library, SPA, plugin, and markdown-tooling repositories still receive a substantive structural fact. These are available in every mode, their breadth scales with repository size, and each carries its own line cap in its catalog.

**Tier 2 — confirmed synthesis, stubs shown and bodies composed only after confirm.** `spec` documents for the per-mode, per-depth top-N hotspot modules — a flat scale baseline in small and medium modes, and a per-selected-domain floor plus repo-wide-rank fill in large mode — and cross-cutting `rule` documents up to the active depth's cap, each composed under the `skills/_shared/` precision contracts including `rule-contract.md`. A size and churn-gated flagship hotspot composes at a raised body cap or decomposes into at most 3 sub-specs by separable sub-surface, and every init-synthesized spec is created with `status: draft`. To keep the highest-cost operation behind the curation gate, the preview shows only spec stubs — title plus the LOC and test ratio that qualified the module — together with a coverage line naming specs per load-bearing module, domains seeded, and cross-cutting rules, so sparseness is visible before `confirm`. The agent reads hotspot source files and composes full bodies only after `confirm`, skipping anything the user deselected. Tier 2 artifacts are the only LLM-synthesized documents, and none is created unseen. Hotspot ranking uses a tests-aware primary tier with a test-independent fallback on fan-in, public surface, size, and churn, so test-less repositories still surface real specs instead of an empty pool.

**Tier 3 — capstone index.** One `architecture-overview` doc that maps the seed: an index keyed by document type and topic, plus structural facts such as module and domain counts, language, framework, and ORM. Per precision Rule 5 it does not enumerate `.archcore/` paths in its body, because cross-document links live exclusively in the relation graph. It is capped under 150 lines and is not a prose summary of the codebase.

**Imports.** An existing `CLAUDE.md`, `AGENTS.md`, or `.cursorrules` is parsed into typed documents in the main flow, behind the preserved high-cost gate of combined size above 50 KB, more than 5 files, or a yield above 8 documents, shown as a line item in the preview.

**Relation wiring.** Init auto-adds `related`, `implements`, and `depends_on` edges among the seeded documents, so `/archcore:context` and the graph are useful immediately.

The single confirm gate is the curation step rather than a formality: the preview lists every planned document by tier with a per-item token-cost estimate for the expensive Tier 2 stubs and the import, and the user confirms once, deselects items, or aborts. One human confirmation reclassifies the whole seed from auto-generated to human-curated, which is the category the research showed to be net positive.

Four guardrails carry forward from the superseded decision and from `precision-over-coverage.adr`, which stays in force: no `create_document` or `add_relation` fires before `confirm`, while `init_project()` and the read-only MCP calls are infrastructure that precedes the gate, so `cancel` leaves `.archcore/` content-empty with idempotency preserved; no single document exceeds 200 lines, and the capstone 150; no vague descriptive prose, with Tier 2 synthesis contract-bound and checked by `bin/check-precision`; and no body section enumerates other `.archcore/` documents.

## Alternatives Considered

1. **Keep `no-auto-generated-context` as it stood** — rejected because its empty-state minimalism is the direct cause of the "init does nothing" feedback and the install-to-retention leak.
2. **Silent auto-creation with no confirm** — rejected because it reproduces the unconfirmed-blob failure mode the research measured and removes the curation gate that makes seeding safe.
3. **Per-item confirmation, making today's propose model mandatory** — rejected because death by a thousand prompts kills the day-one impact and is essentially the status quo users already reject.
4. **Generate prose summaries per module** — rejected because that is the penalized pattern; Tier 2 emits typed, contract-bound specs for a handful of high-signal hotspots only, behind the confirm gate.
5. **Compose full Tier 2 spec bodies before the preview** — rejected because it spends the peak read cost, about 9.9k tokens on a medium repository, on work the user may cancel. Stubs before and bodies after `confirm` keeps that cost behind the gate.

## Consequences

- Init output rises from 0–4 documents to roughly 6–12 or more on a real repository, with relations, so the push and pull mechanisms and the graph demonstrate value in the first session.
- `precision-over-coverage.adr` remains binding, and the line caps plus the no-unconfirmed-blob and no-path-enumeration rules carry forward, keeping init inside the research's safe zone.
- Tradeoff: init token cost roughly doubles on a medium repository, from Tier 2 synthesis and import extraction. The cost is surfaced per item in the preview so the user opts in knowingly, and the stub-body split confines the spend to confirmed items.
- Tradeoff: new first-class plugin assets require maintenance — the detector catalogs `detect-data-model`, `detect-integrations`, `detect-config`, and `detect-surface`, the `compose-overview` composer, and `rule-contract.md` in `skills/_shared/`. A later universality pass added the public-surface fact, the hotspot test-independent fallback tier, and a narrow instruction-modules exception in `detect-modules`, so init seeds substantively for library, SPA, CLI, and plugin repositories rather than only server applications.
- Tradeoff: synthesis quality becomes a risk surface. The single-confirm preview with deselection is the mitigation, since a weak spec is dropped with one word before its source is ever read.

## Superseded when

- A controlled measurement shows the confirmed-synthesis seed reducing agent task success, or users routinely cancelling or undoing the seed within the first session.
- Anthropic, Cursor, or a comparable host vendor publishes contrary context-engineering guidance grounded in newer evidence.
