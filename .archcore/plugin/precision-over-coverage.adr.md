---
title: "Precision over Coverage in Archcore Documentation"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Context

Industrial research consistently shows that documentation usefulness is determined by accuracy, freshness, and relevance — not volume. Aghajani et al. (2020) ranked erroneous examples (59%), code-doc inconsistency (59%), outdated examples (51%), and superfluous content (55%) as the top maintenance pains across 146 practitioners. Recent context-engineering research (arxiv 2510.21413, Mohsenimofidi et al., 2026) measured AGENTS.md-style auto-generated context files across 466 OSS projects: such files reduced agent task success in 5 of 8 settings and increased inference cost 20–23%.

Until now, Archcore skills used the pattern `Compose content covering [section list]` — leaving content quality to LLM defaults. This produced descriptive prose ("the system handles requests appropriately") rather than operational artifacts. Documents were created but rarely re-read because they failed to encode falsifiable, code-grounded claims.

## Decision

Archcore default skill behavior generates **operational artifacts**, not descriptive prose. Concretely:

- Each major document type (`adr`, `spec`, `rule`, `guide`) has an explicit content contract shipped with the plugin under `skills/_shared/`. Phase 1 covers `skills/_shared/adr-contract.md`; later phases add the rest.
- Skills authoring documents of those types MUST load the corresponding contract and `skills/_shared/precision-rules.md` before composition.
- A PostToolUse hook (`bin/check-precision`) emits warnings via `additionalContext` when a created or updated document contains forbidden vague words, lacks mandatory sections, has incomplete frontmatter, or is below 200 characters.
- Phase 1 of rollout keeps the hook in soft mode (always exit 0); blocking semantics are deferred to later phases pending observability data.
- Existing documents in any `.archcore/` are not retroactively flagged as invalid; validation on `update_document` is diff-only in later phases (cannot introduce new violations, but does not retroactively flag pre-existing structure).
- Plugin-internal runtime assets (`skills/_shared/`) are the canonical source. Skill instructions must not reference the consumer's project environment (`.archcore/`, source paths, etc.) for loading rules — only plugin-shipped paths.

## Alternatives Considered

1. **Status quo** (keep `Compose content covering [sections]`) — rejected because the pattern produces descriptive constatation rather than operational artifacts; users do not return to such documents and the corpus accumulates passive prose.
2. **Author-only enforcement via lint scripts** — rejected because it shifts cost to authors after writing rather than guiding the LLM during composition; corrects symptoms after the document is committed instead of shaping the draft.
3. **Critic-loop subagent on every document creation** — rejected because it doubles token cost and latency for routine work; subagent is reserved for opt-in `--deep` audit mode in Phase 3.
4. **Place runtime contracts inside `.archcore/plugin/`** — rejected because that directory is project-local; consumer projects do not receive plugin-development artifacts, and skill instructions referencing those paths break for end users. Runtime assets live in `skills/_shared/` instead.

## Consequences

- Document creation latency in full mode increases an estimated 20–40% due to contract loading and (in Phase 2) evidence harvest.
- "Usefulness ratio" (fraction of documents re-read or referenced after creation) is expected to rise; this is the primary success metric.
- Authors must accept stricter skill prompts; draft mode (Phase 4) preserves a fast path for rapid capture.
- Contract files (~3 of them in Phase 1) become first-class plugin assets in `skills/_shared/` and require maintenance as document standards evolve.
- New documents become consistently structured; pre-existing documents remain readable but are signaled as inferior in `/archcore:status` and `/archcore:verify` reports.

## Superseded when

- Adoption metrics show precision mode reduces `/archcore:*` invocations by more than 30% over a 60-day window after Phase 1 rollout.
- A controlled internal comparison shows contract-driven composition yields no measurable improvement in document re-use rate (e.g., reads-after-creation, references-from-other-docs) over 90 days.
- Anthropic, Cursor, or comparable host vendors publish a different official position grounded in new benchmarks that contradict the current research base.
