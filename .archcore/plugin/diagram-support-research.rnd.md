---
title: "Diagram Support — Effectiveness Evidence and Token Forecast"
status: draft
tags:
  - "diagrams"
  - "plugin"
  - "precision"
---

## Goal

Decide whether embedding diagrams (Mermaid) in Archcore documents is net-positive, for which document types, and what it costs in agent-context tokens — before building anything. Output: a placement map across plugin layers.

## Questions

1. Is there measured evidence that diagrams help the two audiences of an Archcore document — human readers (rendered on GitHub) and AI coding agents (read as text in context)?
2. What does a diagram cost in tokens, and does that cost change the decision?
3. Which plugin layer carries which part of diagram support?

## Approach

- Two literature sweeps: cognitive science + software-engineering controlled experiments (human side); LLM/agent studies (agent side). Sources cited inline.
- Direct token measurement (2026-07-06): three Archcore-scale Mermaid diagrams (C4 context, ERD, sequence) versus prose conveying the same relations, using a chars→token heuristic (structured text ~3.7 chars/token, prose ~4.5 chars/token).

## Findings

**Human side — benefit is real but conditional.** Words+pictures beat words alone (Mayer multimedia principle: lab median d≈1.35; 2025 field meta-analysis g≈0.39, spatial contiguity g≈0.74 — https://par.nsf.gov/servlets/purl/10637927), via locality, reduced search, and perceptual inference (Larkin & Simon 1987, Cognitive Science 11(1) — https://onlinelibrary.wiley.com/doi/10.1111/j.1551-6708.1987.tb00863.x). A diagram separated from its explaining text imposes split-attention load that erases the gain (Chandler & Sweller 1992). In software: UML raised change correctness +54% among 20 professionals but added ~14% time (Dzidek et al. 2008, IEEE TSE 34(3)); diagrams help comment-free code and experienced maintainers most (Scanniello et al. 2018, EMSE, 12-experiment aggregation); DFD + traceability raised security-analysis correctness +41% (Schneider et al. 2024, arXiv:2401.04446).

**Human side — counter-evidence (load-bearing for scope).** More diagram detail did not improve maintenance (Fernández-Sáez et al. 2016, EMSE — level-of-detail null); analysis-phase models did not help comprehension (Scanniello & Gravino 2013, ACM TOSEM); 35/50 engineers did not use UML at all (Petre 2013, ICSE); outdated diagrams are the dominant reported hurdle and erode trust (Fernández-Sáez et al. 2018, EMSE). [assumption] No controlled study measures C4-model comprehension benefit — its value is asserted, not measured.

**Agent side — structure helps, but the relation graph already carries most of it.** A repository-level code graph raised an AI SWE agent's resolve rate (+8.56% for Agentless; RepoGraph, arXiv:2410.14684); graph-structured reasoning raised quality and lowered cost (Graph of Thoughts +62% / −31%; Besta et al. 2023, arXiv:2308.09687). Archcore already exposes a relation graph, so an embedded diagram is partly redundant for the agent; its non-redundant contribution is the shapes the graph does not encode — sequence order, cardinality, trust boundaries, state transitions. Multimodal models reason unreliably over rendered diagram images (up to 30% drop, OCR-shortcut reliance; ChartQA line), while text-encoded Mermaid is read directly and generated reliably by Markdown-trained models (MermaidSeqBench, arXiv:2511.14967) — so for the agent, embed Mermaid source, never an image. Added low-value tokens degrade retrieval from the middle of long context (Lost in the Middle, Liu et al. 2023): a stale diagram loaded on every context entry actively misleads — the agent-side echo of the human staleness finding.

**Token cost — measured, and not the deciding variable.** At Archcore scale (few nodes) Mermaid is not more compact than prose: C4 ~128 vs ~68 tokens (×1.9), ERD ~123 vs ~90 (×1.4), sequence ~102 vs ~81 (×1.3), measured 2026-07-06. Compaction appears only at high edge density, which disciplined small diagrams avoid. [assumption] Absolute cost is negligible: ~10 diagrams in a mature `.archcore` ≈ 1.3k tokens stored; a `/archcore:context` load carries 0–2 diagrams ≈ +0–260 tokens (under 0.2% of a 200k window); ~30 loads per session ≈ 6k tokens against hundreds of thousands consumed. The governing costs are context dilution and staleness amplification, not the token bill — both handled by the non-derivable-signal test + provenance link + freshness, not by a token budget.

## Recommendation

Adopt a narrow, staleness-governed policy on the cheapest layers first. A diagram earns inclusion only when it carries a signal absent from any single source file AND absent from the relation graph, sits adjacent to its caption and an `@source` provenance reference, and is Mermaid source.

Placement by plugin layer:

- **Shared runtime asset** (`skills/_shared/diagram-contract.md`) — non-derivable-signal test, diagram-kind→document-type map, Mermaid-only + contiguity + size rules. One file, all hosts. Primary home.
- **Skills** — one load line in `capture` (spec/doc), `plan` references (prd/spec/feature), `decide` (architecture cascade); `init` synthesizes a C4 into the architecture-overview doc and an ERD into the data-model doc as `draft`, user-confirmed (per the extractive-facts + confirmed-synthesis discipline).
- **Hooks** (`bin/check-diagram`, non-blocking) — flag a Mermaid block missing an adjacent `@source`; extend drift detection so a diagram whose `@source` changed is flagged stale. This freshness gate is the precondition the evidence says makes diagrams net-positive.
- **MCP/Core** (CLI) — first-class diagram objects (queryable, relatable to `@code` and requirement documents) only if the shared-asset stage proves demand; carries its own ADR; encodes only shapes the relation graph lacks.

Non-goals: no new skill or `/` command; no token-budget gating; no rendered images; C4 stays human-facing in a `doc`, kept out of agent-injected context.

Diagram kind → document type: C4 / system context → `doc` (architecture overview, human-facing); DFD → `spec`, `doc`; ERD → `spec`, data-model `doc` ([assumption] prefer an annotated ERD or class-style representation — De Lucia et al. 2010 found UML class diagrams beat ER diagrams for data-model comprehension); sequence → `spec`, `adr`; state → `spec`, `rule`; use-case / user-story-map → `prd`, `urd`, `srs`, `idea`, `plan`.

## Next Action

Draft `skills/_shared/diagram-contract.md` (Stage 0) and inject C4/ERD synthesis into `init`. Defer `bin/check-diagram` to Stage 1 and first-class core objects to a separate ADR gated on Stage 0 evidence.