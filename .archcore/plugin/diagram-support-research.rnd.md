---
title: "Diagram Support — Effectiveness Evidence and Token Forecast"
status: draft
tags:
  - "diagrams"
  - "plugin"
  - "precision"
---

## Goal

Decide whether embedding Mermaid diagrams in Archcore documents is net-positive, for which document types, and at what cost in agent-context tokens, before anything is built. The output is a placement map across the plugin layers.

## Questions

1. Is there measured evidence that a diagram helps either audience of an Archcore document — a human reader seeing it rendered, and an AI coding agent reading it as text in context?
2. What does a diagram cost in tokens, and does that cost change the decision?
3. Which plugin layer carries which part of diagram support?

## Approach

Two literature sweeps, one over cognitive science and software-engineering controlled experiments for the human side and one over LLM and agent studies for the agent side, with sources cited inline. Plus direct token measurement on 2026-07-06 of three Archcore-scale Mermaid diagrams — a C4 context, an ERD, and a sequence — against prose conveying the same relations, using a characters-to-token heuristic of roughly 3.7 characters per token for structured text and 4.5 for prose.

## Findings

**Human side — the benefit is real but conditional.** Words plus pictures beat words alone: the Mayer multimedia principle reports a lab median of d≈1.35, and a 2025 field meta-analysis reports g≈0.39 with spatial contiguity at g≈0.74 (https://par.nsf.gov/servlets/purl/10637927). The mechanism is locality, reduced search, and perceptual inference (Larkin & Simon 1987, Cognitive Science 11(1)). A diagram separated from its explaining text imposes a split-attention load that erases the gain (Chandler & Sweller 1992). In software specifically, UML raised change correctness by 54% among 20 professionals while adding roughly 14% time (Dzidek et al. 2008, IEEE TSE 34(3)); diagrams help most with comment-free code and experienced maintainers (Scanniello et al. 2018, EMSE, aggregating 12 experiments); and a DFD with traceability raised security-analysis correctness by 41% (Schneider et al. 2024, arXiv:2401.04446).

**Human side — the counter-evidence, which is load-bearing for scope.** More diagram detail did not improve maintenance (Fernández-Sáez et al. 2016, EMSE, a level-of-detail null result). Analysis-phase models did not help comprehension (Scanniello & Gravino 2013, ACM TOSEM). Thirty-five of 50 engineers did not use UML at all (Petre 2013, ICSE). And an outdated diagram is the dominant reported hurdle, eroding trust (Fernández-Sáez et al. 2018, EMSE). [assumption] No controlled study measures the comprehension benefit of the C4 model; its value is asserted rather than measured.

**Agent side — structure helps, but the relation graph already carries most of it.** A repository-level code graph raised an AI software-engineering agent's resolve rate by 8.56% for Agentless (RepoGraph, arXiv:2410.14684), and graph-structured reasoning raised quality while lowering cost (Graph of Thoughts, +62% and −31%; Besta et al. 2023, arXiv:2308.09687). Archcore already exposes a relation graph, so an embedded diagram is partly redundant for the agent, and its non-redundant contribution is the shapes the graph does not encode: sequence order, cardinality, trust boundaries, and state transitions. Multimodal models reason unreliably over a rendered diagram image, dropping up to 30% and falling back on OCR shortcuts, while text-encoded Mermaid is read directly and generated reliably by Markdown-trained models (MermaidSeqBench, arXiv:2511.14967) — so for the agent the plugin embeds Mermaid source and never an image. Added low-value tokens also degrade retrieval from the middle of a long context (Liu et al. 2023), so a stale diagram loaded on every context entry actively misleads, which is the agent-side echo of the human staleness finding.

**Token cost — measured, and not the deciding variable.** At Archcore scale, meaning few nodes, Mermaid is not more compact than prose: the C4 measured about 128 tokens against about 68 for prose, a factor of 1.9; the ERD about 123 against 90, a factor of 1.4; and the sequence about 102 against 81, a factor of 1.3, all on 2026-07-06. Compaction appears only at high edge density, which a disciplined small diagram avoids. [assumption] The absolute cost is negligible: roughly 10 diagrams in a mature `.archcore/` is about 1.3k tokens stored, a `/archcore:context` load carries 0–2 diagrams for about 0–260 tokens, under 0.2% of a 200k window, and roughly 30 loads per session is about 6k tokens against hundreds of thousands consumed. The governing costs are context dilution and staleness amplification rather than the token bill, and both are handled by the non-derivable-signal test, a provenance link, and freshness — not by a token budget.

## Recommendation

Adopt a narrow, staleness-governed policy on the cheapest layers first. A diagram earns inclusion only when it carries a signal absent from any single source file *and* absent from the relation graph, sits adjacent to its caption and an `@source` provenance reference, and is Mermaid source.

Placement by plugin layer:

- **Shared runtime asset**, as `skills/_shared/diagram-contract.md` — the non-derivable-signal test, the diagram-kind to document-type map, and the Mermaid-only, contiguity, and size rules. One file, all hosts, and the primary home.
- **Skills** — one load line in `capture` for spec and doc, in the `plan` references for prd, spec, and feature, and in `decide` for the architecture cascade. `init` synthesizes a C4 into the architecture-overview doc and an ERD into the data-model doc as `draft`, user-confirmed, per the extractive-facts and confirmed-synthesis discipline.
- **Hooks**, as a non-blocking `bin/check-diagram` — flag a Mermaid block missing an adjacent `@source`, and extend drift detection so a diagram whose `@source` changed is flagged stale. This freshness gate is the precondition the evidence says makes diagrams net-positive.
- **MCP and core**, in the CLI — first-class diagram objects, queryable and relatable to code and requirement documents, only if the shared-asset stage proves demand. It carries its own ADR and encodes only the shapes the relation graph lacks.

Non-goals: no new skill and no new command, no token-budget gating, no rendered images, and C4 stays human-facing inside a `doc`, kept out of agent-injected context.

Diagram kind to document type: C4 and system context map to `doc` as the human-facing architecture overview; DFD to `spec` and `doc`; ERD to `spec` and the data-model `doc`, where [assumption] an annotated ERD or class-style representation is preferable, since De Lucia et al. 2010 found UML class diagrams beat ER diagrams for data-model comprehension; sequence to `spec` and `adr`; state to `spec` and `rule`; and use-case or user-story-map to `prd`, `urd`, `srs`, `idea`, and `plan`.

## Next Action

Draft `skills/_shared/diagram-contract.md` as Stage 0, and inject C4 and ERD synthesis into `init`. Defer `bin/check-diagram` to Stage 1, and defer first-class core objects to a separate ADR gated on the Stage 0 evidence.
