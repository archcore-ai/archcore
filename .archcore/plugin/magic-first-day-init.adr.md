---
title: "Magic First-Day Init — Extractive Facts and Confirmed Synthesis"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "onboarding"
  - "plugin"
  - "precision"
  - "skills"
---

## Context

An earlier decision closed the door on any LLM-scan-generated document, citing arxiv 2510.21413, where auto-generated `AGENTS.md`-style context files reduced agent task success in 5 of 8 settings and raised inference cost by 20–23%. Its consequence was an `/archcore:init` that seeded 0 to 4 tiny documents and left everything else as a non-binding to-do list. User feedback was consistent: init "shows poorly" and "creates very little content". On install into an existing repository the first session produced almost nothing an agent could not read from `package.json` in 30 seconds, and the relation graph stayed empty. The funnel from install to first useful state leaked where retention is decided.

The cited research penalized unconfirmed wholesale LLM prose and oversized files. It did not penalize extracting facts or human-confirmed artifacts, and human-curated context was net positive in the same study. The failure mode is the blob, not the act of seeding. A prompt-engineering review confirmed the design. It put the cost of a medium-repository run at roughly double, from about 21.5k to about 46.6k tokens, dominated by reading hotspot source files for spec synthesis.

## Decision

`/archcore:init` populates `.archcore/` in a single pass: **detect → compose → one preview → one confirm → create and wire relations**. Detection and composition are ordered sub-phases that load only the catalogs each needs. Three tiers bound what may be composed. The default depth is `standard`, because init is fully gated: nothing is written before `confirm`, and each depth's cost is shown before the user commits. `light` is the explicit cost-conscious opt-down.

**Tier 1 — extractive facts, auto-composed in full.** These are derived mechanically from manifests, schemas, routing, and config, not synthesized as prose: the stack rule, the run guide, the entry-point inventory, the top-level map, the data-model doc, the integrations doc, the config and env doc, and the public-surface doc. The config doc carries variable names and purpose but never values, which is a security boundary. The public-surface doc captures the role-based outward shape that entry points miss, so library, SPA, plugin, and markdown-tooling repositories still get a substantive structural fact. Each carries its own line cap. A cap that trims a detected list states its remainder as `+N more`, per `decompose-over-truncate.adr`.

**Tier 2 — confirmed synthesis, stubs shown and bodies composed only after confirm.** This tier holds `spec` documents for ranked hotspot modules and cross-cutting `rule` documents for every candidate clearing the recurrence threshold. Cross-cutting synthesis runs at every depth, because it is the highest value-per-token artifact and was being dropped exactly when it mattered most. Each document is composed under the `skills/_shared/` precision contracts. Every init-synthesized spec is created with `status: draft`. The preview shows only spec stubs (title plus the LOC and test ratio that qualified the module) and a coverage line, so sparseness and volume are both visible before `confirm`. The agent reads hotspot source and composes full bodies only after `confirm`, skipping anything the user deselected. Tier 2 artifacts are the only LLM-synthesized documents, and none is created unseen. A size- and churn-gated flagship hotspot composes at a raised body cap or decomposes into at most 3 sub-specs.

The spec budget is a coverage rate over the ranked candidate pool, with no absolute maximum. The earlier clamp on selected-domain count was operative at its minimum on every large repo, because the day-one dialog accepts only 1 to 3 domains. A 773-module monolith and a 45-module service both received 10 specs at `standard`. The rates, floors, and ranking tiers live in `skills/_shared/grounding/detect-hotspots.md`. The domain dialog allocates the budget rather than sizing it: each selected domain keeps a floor of at least 1 spec, and the remainder fills by repo-wide rank. Detect collects signal data for the whole eligible pool, so a `depth:` toggle in the preview re-slices without re-reading. Execution is one preview, one confirm, no staging, no resume. Above 25 specs the preview carries a high-volume notice naming the spec count, the estimated tokens, and the cheaper depth.

**Tier 3 — capstone index.** One `architecture-overview` doc maps the seed as an index keyed by document type and topic, plus structural facts. Per precision Rule 5 it does not enumerate `.archcore/` paths in its body, because cross-document links live in the relation graph. It is capped under 150 lines.

**Imports.** `init-import-mode.adr` governs authored sources (agent-instruction files, decision records, contributor docs, git history). The import track converts them into native documents, staged by an import plan with one confirm per wave. The code seed keeps one preview and one confirm. Modules that authored sources cover leave the hotspot pool before the budget is computed.

**Relation wiring.** Init auto-adds `related`, `implements`, and `depends_on` edges among the seeded documents, so command grounding and the graph are useful immediately. CLI hooks and command grounding replace the former pull command (`four-command-palette.adr`).

The single confirm gate is the curation step. The preview lists every planned document by tier with a token-cost estimate for the expensive items. The user confirms once, deselects items, or aborts. One human confirmation reclassifies the seed from auto-generated to human-curated, the category the research showed to be net positive.

Four guardrails carry over from `precision-over-coverage.adr`:

- No `create_document` or `add_relation` fires before `confirm`. `init_project()` and read-only MCP calls precede the gate, so `cancel` leaves `.archcore/` content-empty.
- No single document exceeds 200 lines, and the capstone stays under 150.
- No vague descriptive prose. Tier 2 synthesis is contract-bound and checked by the engine's precision check.
- No body section enumerates other `.archcore/` documents.

## Alternatives Considered

1. **Keep the empty-state minimalism** — rejected because it is the direct cause of the "init does nothing" feedback and the install-to-retention leak.
2. **Silent auto-creation with no confirm** — rejected because it reproduces the unconfirmed-blob failure mode the research measured.
3. **Per-item confirmation** — rejected because death by a thousand prompts kills day-one impact.
4. **Generate prose summaries per module** — rejected because that is the penalized pattern. Tier 2 emits typed, contract-bound specs for high-signal hotspots only.
5. **Compose full Tier 2 spec bodies before the preview** — rejected because it spends the peak read cost, about 9.9k tokens on a medium repository, on work the user may cancel.
6. **Stage a large budget into waves with a resumable `--continue`** — rejected because staging adds a resume protocol and a partial-seed state machine to a skill whose write boundary is one confirm. The authored-import track is the exception, because its sources are separable.

## Consequences

- Init output rises from 0–4 documents to roughly 6–12 or more on a real repository, with relations. Seed size tracks repository size, not a constant.
- `precision-over-coverage.adr` stays binding, which keeps init inside the research's safe zone.
- Tradeoff: init token cost roughly doubles on a medium repository. The preview shows the cost per item, and the stub-body split confines the spend to confirmed items.
- Tradeoff: [expected] a large repository at `deep` can budget over 100 spec bodies in one pass that is neither staged nor resumable. An interrupted run leaves the documents already created and needs `--refresh` to top up. The high-volume notice is the mitigation.
- Tradeoff: the detector catalogs, the `compose-overview` composer, and `rule-contract.md` in `skills/_shared/` are first-class plugin assets that need maintenance.
- Tradeoff: synthesis quality is a risk surface. The single-confirm preview with deselection is the mitigation, since a weak spec is dropped before its source is read.

## Superseded when

- A controlled measurement shows the confirmed-synthesis seed reducing agent task success, or users routinely cancelling or undoing the seed within the first session.
- Users routinely toggle `depth:light` or deselect more than half the spec stubs on large repositories, which shows the coverage rates set too high.
- A host vendor publishes contrary context-engineering guidance grounded in newer evidence.
