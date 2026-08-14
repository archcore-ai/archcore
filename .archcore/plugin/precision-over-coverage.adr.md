---
title: "Precision over Coverage in Archcore Documentation"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Context

Industrial research consistently shows that documentation usefulness is determined by accuracy, freshness, and relevance rather than volume. Aghajani et al. (2020) ranked erroneous examples at 59%, code-doc inconsistency at 59%, outdated examples at 51%, and superfluous content at 55% as the top maintenance pains across 146 practitioners. Context-engineering research — arxiv 2510.21413, Mohsenimofidi et al., 2026 — measured `AGENTS.md`-style auto-generated context files across 466 open-source projects and found they reduced agent task success in 5 of 8 settings and increased inference cost by 20–23%. Against that evidence, Archcore skills used the pattern `Compose content covering [section list]`, leaving content quality to LLM defaults, which produced descriptive prose such as "the system handles requests appropriately" rather than operational artifacts; documents were created but rarely re-read, because they encoded no falsifiable, code-grounded claim.

## Decision

Archcore default skill behavior generates **operational artifacts rather than descriptive prose**, enforced by per-type content contracts shipped under `skills/_shared/` that skills MUST load before composition, and by a `bin/check-precision` PostToolUse hook that emits warnings through `additionalContext`.

The mechanism has five parts:

- Each major document type — `adr`, `spec`, `rule`, `guide` — has an explicit content contract shipped with the plugin under `skills/_shared/`. Phase 1 covered `skills/_shared/adr-contract.md`; later phases added the rest.
- A skill authoring one of those types MUST load the matching contract and `skills/_shared/precision-rules.md` before composing.
- `bin/check-precision` warns when a created or updated document carries a forbidden vague word, lacks a mandatory section, has incomplete frontmatter, or falls below 200 characters.
- Phase 1 kept the hook in soft mode, always exiting 0; blocking semantics were deferred pending observability data.
- Plugin-internal runtime assets under `skills/_shared/` are the canonical source. A skill instruction MUST NOT reference the consumer's project environment for loading rules — only plugin-shipped paths.

## Alternatives Considered

1. **Keep the status quo of `Compose content covering [sections]`** — rejected because the pattern produces descriptive statement rather than an operational artifact; users do not return to such a document, and the corpus accumulates passive prose.
2. **Author-only enforcement through lint scripts** — rejected because it shifts cost to the author after writing rather than guiding the LLM during composition, correcting symptoms after the document is committed instead of shaping the draft.
3. **A critic-loop subagent on every document creation** — rejected because it doubles token cost and latency for routine work; the subagent is reserved for the opt-in `--deep` audit mode.
4. **Placing runtime contracts inside `.archcore/plugin/`** — rejected because that directory is project-local: a consumer project never receives plugin-development artifacts, so a skill instruction referencing those paths breaks for end users. Runtime assets live in `skills/_shared/` instead.

## Consequences

- New documents become consistently structured, and a pre-existing document remains readable while being signaled as inferior in `/archcore:review --deep` reports and in `make verify` output.
- [expected] The usefulness ratio — the fraction of documents re-read or referenced after creation — rises. This is the primary success metric.
- Tradeoff: document creation latency in full mode increases an estimated 20–40%, from contract loading and evidence harvest.
- Tradeoff: authors accept stricter skill prompts; draft mode preserves a fast path for rapid capture.
- Tradeoff: the contract files become first-class plugin assets in `skills/_shared/` and require maintenance as document standards evolve.
- An existing document in any `.archcore/` is not retroactively flagged as invalid: no validation gate rejects one, and the post-write check reports without blocking. `controlled-technical-writing-profile.adr` narrows this on 2026-08-03 by normalizing the plugin's own corpus in one authoring pass.
- Correction. The diff-only property this record claimed for `update_document` never shipped. Neither the deleted `check-precision` script nor the engine's `Precision` scopes a check to the changed lines — both read the whole body — so an edit to an older document surfaces that document's pre-existing findings as well. The no-rejection guarantee above is unaffected: a report is not a rejection.

## Superseded when

- Adoption metrics show precision mode reducing `/archcore:*` invocations by more than 30% over a 60-day window.
- A controlled internal comparison shows contract-driven composition yielding no measurable improvement in document re-use rate — reads after creation, or references from other documents — over 90 days.
- Anthropic, Cursor, or a comparable host vendor publishes a different official position grounded in new benchmarks that contradict the current research base.
