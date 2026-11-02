---
title: "Precision over Coverage in Archcore Documentation"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "precision"
---

## Context

Research on documentation points to accuracy, freshness, and relevance, not volume. Aghajani et al. (2020) found code-doc inconsistency (59%) and superfluous content (55%) among the top maintenance pains of 146 practitioners. Mohsenimofidi et al. (arxiv 2510.21413, 2026) measured auto-generated `AGENTS.md`-style context files across 466 open-source projects. The files reduced agent task success in 5 of 8 settings and raised inference cost by 20–23%.

Archcore skills used the pattern `Compose content covering [section list]`, which left content quality to LLM defaults. The result was descriptive prose such as "the system handles requests appropriately". Documents were created but rarely re-read, because they encoded no falsifiable, code-grounded claim.

## Decision

Archcore default skill behavior generates **operational artifacts rather than descriptive prose**, enforced by per-type content contracts shipped under `skills/_shared/` that skills MUST load before composition, and by the precision check that the Archcore engine (the CLI) runs after `create_document` and `update_document` and reports through the PostToolUse hook.

The mechanism has five parts:

- Each major document type — `adr`, `spec`, `rule`, `guide` — has an explicit content contract shipped with the plugin under `skills/_shared/`. `skills/_shared/adr-contract.md` is the first of them.
- A skill authoring one of those types MUST load the matching contract and `skills/_shared/precision-rules.md` before composing.
- The engine's precision check warns when a created or updated document carries a forbidden vague word, lacks a mandatory section, or has incomplete frontmatter. The check is advisory and never blocks a write.
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
- An existing document in any `.archcore/` is not retroactively flagged as invalid: no validation gate rejects one, and the post-write check reports without blocking. `docs-style/controlled-technical-writing.rule` narrows this for the plugin's own corpus.
- The check is not diff-only. The engine's precision check reads the whole body, so an edit to an older document surfaces that document's pre-existing findings too. A report is not a rejection, so the no-rejection guarantee holds.

## Superseded when

- Adoption metrics show precision mode reducing `/archcore:*` invocations by more than 30% over a 60-day window.
- A controlled internal comparison shows contract-driven composition yielding no measurable improvement in document re-use rate — reads after creation, or references from other documents — over 90 days.
- Anthropic, Cursor, or a comparable host vendor publishes a different official position grounded in new benchmarks that contradict the current research base.
