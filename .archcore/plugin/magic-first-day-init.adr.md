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

## Context

`no-auto-generated-context.adr` (now superseded) closed the door on any LLM-scan-generated documents, citing arxiv 2510.21413: auto-generated, AGENTS.md-style context files reduced agent task success in 5 of 8 settings and raised inference cost 20–23%. The designed-in consequence was an `/archcore:init` that seeds **0–4 tiny documents** (stack rule, run guide, and — in larger modes — an entry-point inventory and a top-level map) and surfaces everything else as a non-binding "propose" to-do list it explicitly refuses to auto-execute.

User feedback is consistent: init "shows poorly" and "creates very little content." On install into an existing repo with no `.archcore/`, the first session produces almost nothing the agent could not read from `package.json` in 30 seconds — a critique already recorded verbatim in `bootstrap-scale-modes.plan`. Push (`check-code-alignment`) and pull (`/archcore:context`) both visibly no-op, and the relation graph is empty. The funnel from install → first useful state leaks at exactly the moment retention is decided.

Product decision: **first-day impact outweighs strict empty-state minimalism.** A user landing in a context-free project must get a populated, navigable `.archcore/` from the first `/archcore:init`.

The reconciliation that keeps us out of the anti-pattern: the cited research penalized *unconfirmed, wholesale LLM prose* and oversized files — not the extraction of facts, and not human-confirmed artifacts (human-curated context was net-positive in the same study). The measured failure mode is the *blob*, not the *act of seeding*. Day-one magic is therefore achievable on the safe side of that line.

A pre-implementation prompt-engineering review (ai-kit:prompt-engineer) confirmed the design is sound but raised four structural fixes — folded into the Decision below and detailed in `magic-first-day-init.plan`. The review also quantified cost: a medium-repo run roughly doubles (~21.5k → ~46.6k tokens, ≈ $0.11 → $0.26 at Sonnet pricing). The dominant driver is reading hotspot source files for spec synthesis; the Decision defers that read behind the confirm gate so the peak cost is only spent on what the user keeps.

## Decision

`/archcore:init` populates `.archcore/` in a single pass: **detect → compose → one preview → one confirm → create + wire-relations**. Detection and composition are ordered sub-phases that load only the catalogs each needs, preserving the established lazy-reading discipline. What may be composed is bounded by tier:

- **Tier 1 — Extractive facts (auto-composed in full).** Derived mechanically from manifests, schemas, routing, and config — not synthesized prose: stack rule, run guide, entry-point inventory, top-level map, **data-model doc** (ORM schemas / migrations / `*.proto`), **integrations doc** (third-party SDK dependencies → external services), **config/env doc** (variable names + purpose only — **never values**, a security boundary), **public-surface doc** (the role-based outward shape entry points don't cover — web routes, a library's exported API, a multi-command CLI's commands, an agent-plugin's skills/commands, mobile screens — so library / SPA / plugin / markdown-tooling repos still get a substantive structural fact). Available in all modes; breadth scales with repo size. Each has its own line cap in its catalog.
- **Tier 2 — Confirmed synthesis (stubs shown, bodies composed only after confirm).** `spec` documents for the per-mode top-N hotspot modules (small 3 / medium 5 / large 3 per selected domain) and 0–2 cross-cutting `rule`s, each composed under the `skills/_shared/` precision contracts — including a new `rule-contract.md` added for the rule bodies. To keep the highest-cost operation behind the curation gate, the preview shows only spec **stubs** (title + the LOC / test-ratio that qualified the module). The agent reads hotspot source files and composes full spec bodies **after** `confirm`, skipping any the user deselected. Tier-2 artifacts are the only LLM-synthesized documents, and they are never created unseen. Hotspot ranking uses a tests-aware **primary** tier with a **test-independent fallback** (fan-in / public surface / size / churn) so test-less repos — scripts, SPAs, ML, CLIs, agent-plugin tooling — still surface real specs instead of an empty pool.
- **Tier 3 — Capstone index.** One `architecture-overview` `doc` that maps the seed: an index keyed by document type and topic, plus structural facts (module / domain counts, language, framework, ORM). Per `precision-rules.md` Rule 5 it does **not** enumerate `.archcore/` paths in its body — cross-document links live exclusively in the relation graph. Capped under 150 lines. Not a prose summary of the codebase.
- **Imports.** Existing `CLAUDE.md` / `AGENTS.md` / `.cursorrules` are parsed into typed documents in the main flow. The current HIGH-cost gate (combined > 50 KB OR > 5 files OR yield > 8 documents) is preserved and shown as a line item in the preview.
- **Relation wiring.** Init auto-adds `related` / `implements` / `depends_on` edges among the seeded documents so `/archcore:context` and the graph are useful immediately.

The **single confirm gate is the curation step**, not a formality. The preview lists every planned document by tier, with a per-item token-cost estimate for the expensive Tier-2 stubs and the import; the user confirms once (`confirm`), deselects items (`edit`), or aborts (`cancel`). One human confirmation reclassifies the whole seed from "auto-generated" to "human-curated" — the category the research showed to be net-positive.

Retained guardrails (carried forward from the superseded ADR and from `precision-over-coverage.adr`, which stays in force):

- No `create_document` or `add_relation` fires before `confirm`. `init_project()` and read-only MCP calls (`list_documents`, `get_document`) are infrastructure that precedes the gate — so `cancel` leaves `.archcore/` content-empty while idempotency is preserved.
- No single document exceeds 200 lines (the capstone, ≤ 150).
- No vague descriptive prose; Tier 2 synthesis is contract-bound and checked by `bin/check-precision`.
- No body section enumerates other `.archcore/` documents (Rule 5); the overview is an index of types/topics, not a directory of paths, and never an AGENTS.md-style monolithic blob.

## Alternatives Considered

1. **Keep `no-auto-generated-context` as-is** — rejected; its empty-state minimalism is the direct cause of the "init does nothing" feedback and the install→retention leak.
2. **Silent auto-creation with no confirm** — rejected; this reproduces the exact unconfirmed-blob failure mode the research measured and removes the curation gate that makes seeding safe.
3. **Per-item confirmation (today's "propose" model, made mandatory)** — rejected; death by a thousand prompts kills the day-one "wow" and is essentially the status quo users already reject.
4. **Generate prose summaries per module** — rejected; that is the penalized pattern. Tier 2 emits typed, contract-bound `spec`s for a handful of high-signal hotspots only, behind the confirm gate.
5. **Compose full Tier-2 spec bodies before the preview** — rejected; it spends the peak read cost (~9.9k tokens for a medium repo) on work the user may `cancel`. Stubs-before / bodies-after-confirm keeps that cost behind the gate.

## Consequences

- Init output rises from 0–4 documents to roughly 6–12+ on a real repo, with relations, so push/pull mechanisms and the graph demonstrate value in the first session.
- `precision-over-coverage.adr` remains binding; the line caps and "no unconfirmed blob / no path enumeration" rules carry forward, keeping init inside the research's safe zone.
- Init token cost roughly doubles on a medium repo (Tier-2 synthesis + import extraction); the cost is surfaced per-item in the preview so the user opts in with eyes open, and the stub/body split confines the spend to confirmed items.
- New first-class plugin assets requiring maintenance: detector catalogs `detect-data-model`, `detect-integrations`, `detect-config`, `detect-surface`; the `compose-overview` composer; and a `rule-contract.md` in `skills/_shared/`. A later universality pass added the public-surface fact, the hotspot test-independent fallback tier, and a narrow instruction-modules exception in `detect-modules` (instruction/prompt files count as modules only for agent-plugin / markdown-tooling repos), so init seeds substantively for library / SPA / CLI / plugin repos, not only server apps.
- Synthesis quality becomes a risk surface; the single-confirm preview with `edit`/deselect is the mitigation — a weak spec is dropped with one word before its source is ever read.

## Superseded when

- A controlled measurement shows the confirmed-synthesis seed reduces agent task success, or users routinely `cancel`/undo the seed within the first session.
- Anthropic, Cursor, or a comparable host vendor publishes contrary context-engineering guidance grounded in newer evidence.
