---
title: "Controlled Technical Writing Profile for Repository Documentation"
status: accepted
tags:
  - "plugin"
  - "precision"
  - "rule"
---

## Context

Before `@AGENTS.md` existed, the repository's only writing constraints were plugin runtime assets: `@plugins/archcore/skills/_shared/precision-rules.md` (forbidden lexicon, imperative modals, `[assumption]` marking, no cross-document body sections, architect voice) and the per-type contracts beside it. Those assets bind an agent composing a `.archcore/` document, and `@plugins/archcore/bin/check-precision` applies its strict-EARS checks — one modal per numbered line (7b) and an active-voice obligated subject (7c) — to `spec` bodies only. Nothing constrained `README.md`, `docs/**`, the skill and agent instruction files under `plugins/archcore/`, or the sentence-level structure of `rule`, `guide`, and decision documents. The result was a corpus of 81 `.archcore/` documents (9258 lines) in which requirement phrasing, actor visibility, and step granularity varied by author and by session.

## Decision

Adopted the internal controlled technical writing profile defined in `@AGENTS.md` — ASD-STE100-inspired for procedures, requirements, rules, specifications, and agent instructions; ISO 24495-1-inspired for document organization and explanatory content — binding on `.archcore/**`, `plugins/archcore/skills/**`, `plugins/archcore/agents/**`, `plugins/archcore/copilot-agents/**`, `plugins/archcore/rules/**`, `docs/**`, and `README.md`, with the five-level precedence order stated in `@AGENTS.md` and a one-time normalization pass over all 81 existing `.archcore/` documents.

## Alternatives Considered

1. **Extend `@plugins/archcore/skills/_shared/precision-rules.md` instead of adding a repository-level policy** — rejected because that file is a plugin runtime asset shipped to consumer projects; repository-specific obligations covering `README.md` and `docs/**` would then bind every downstream plugin user, which `precision-over-coverage.adr.md` already ruled out for project-local content.
2. **Declare ASD-STE100 and ISO 24495-1 compliance** — ruled out because neither standard was applied through its certification process; `@AGENTS.md` states explicitly that the profile is an internal writing profile and not a claim of compliance, certification, or approval.
3. **Generalize the `check-precision` strict-EARS checks (7b, 7c) from `spec` to every normative type and write no policy** — deferred because those checks operate on numbered lines only; they cannot evaluate section ordering, actor visibility in prose, or the placement of a warning before the action it guards.
4. **Leave the existing corpus untouched and apply the profile only to new documents** — rejected by the user on 2026-08-03 in favor of normalizing the whole corpus in one pass, accepting the diff cost recorded below.

## Consequences

- The precedence order gives an agent a single deterministic resolution path when `@AGENTS.md` and a type contract disagree: the type contract wins, so `@plugins/archcore/skills/_shared/spec-contract.md` mandatory sections are not displaced by the general policy.
- [expected] Requirement lines across `rule` and `spec` documents become uniformly checkable by review — one obligation, one modal, one named actor — matching the contract that `check-precision` already enforces for `spec` bodies.
- Tradeoff: all 81 documents are rewritten in one commit, so `git log -p .archcore/plugin/` per-document archaeology crosses a single large diff at that commit.
- Tradeoff: `precision-over-coverage.adr.md` records that existing documents are not retroactively flagged as invalid. That stance is narrowed, not reversed — no validation gate rejects a pre-existing document; the corpus is normalized once by an explicit authoring pass.
- Items beyond the forbidden lexicon and the `spec`-only EARS checks have no automated verifier; they rest on the review checklist in `@AGENTS.md`, applied by the authoring agent.

## Superseded when

- More than 3 of the numbered obligations in `controlled-technical-writing.rule.md` become hook-enforced in `@plugins/archcore/bin/check-precision`, which makes that rule's Enforcement section misstate the verifier.
- The repository publishes documentation in a second natural language, which requires the Russian-language clauses of `@AGENTS.md` to become a separate contract rather than a subsection.
- A consumer project reports that documents authored under the plugin's shipped contracts diverge structurally from documents authored in this repository, indicating the repository policy leaked into the shipped runtime assets.
