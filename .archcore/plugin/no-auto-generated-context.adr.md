---
title: "No Auto-Generated Context Blobs"
status: rejected
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

**Superseded on 2026-06-26 by `magic-first-day-init.adr`.** The absolute ban recorded here is replaced by a bounded model: extractive facts plus human-confirmed synthesis, gated behind a single preview and confirm step. The guardrails below are retained in the successor — no unconfirmed prose blobs, no document over 200 lines, precision over coverage, and an overview that indexes rather than summarizes. Only the empty-state minimalism is reversed, because it produced the "init does nothing" feedback.

## Context

Research on AI agent context engineering — arxiv 2510.21413, "Context Engineering for AI Agents in Open-Source Software", Mohsenimofidi et al., 2026 — measured `AGENTS.md`-style auto-generated context files across 466 open-source projects and found that LLM-generated context files reduced agent task success in 5 of 8 settings, increased inference cost by 20–23%, and added 2.45–3.92 steps per task, while human-curated files yielded only about +4 percentage points. Cloudflare's production AI code review system independently identified files longer than 200 lines, and tool names without runnable commands, as performance penalizers. The naive product instinct for a documentation-oriented plugin is to scan the repository and produce summary documents; this decision closed that path before it was taken.

## Decision

Archcore produces no auto-generated repository summary, no `AGENTS.md`-style blob, and no document generated wholesale by an LLM scan of the codebase; every `.archcore/` document is created intentionally in response to a specific authoring intent, through a skill that elicits context from the user or harvests evidence for that one document.

## Alternatives Considered

1. **Generate an `AGENTS.md` per project at bootstrap** — rejected because arxiv 2510.21413 quantifies the harm at scale, at −5 of 8 settings and +20–23% cost, and the harm is structural rather than implementation-specific.
2. **Generate per-module summary documents to seed initial context** — rejected because it reproduces the same failure mode at finer granularity and creates documents the user never consciously authored, which undermines the trust signal of `.archcore/`.
3. **Allow an opt-in auto-summary skill behind a flag** — rejected because even opt-in availability normalizes a known anti-pattern, and over time users mistake auto-output for canonical documents, which degrades the corpus.

## Consequences

- Every document in `.archcore/` carries intentional authorship, so a human or agent reader can treat it as a deliberate artifact rather than machine fill.
- A skill that needs codebase context falls back to grep and read at composition time rather than to a pre-generated artifact, which costs a few seconds per invocation and keeps the context fresh.
- Tradeoff: init output stays minimal, so a user running `/archcore:init` on an existing repository gets no knowledge tree for free, and the first useful state of `.archcore/` requires deliberate authoring. This tradeoff is what the successor reverses.

## Superseded when

- A benchmark at comparable scale, above 400 open-source repositories, shows auto-generated context files improving agent task success by 5 percentage points or more, net of cost.
- Anthropic, Cursor, or a comparable host vendor documents a different official position based on newer evidence and revises its own context-engineering guidance.

Neither condition was met. The supersession was a product decision: first-day onboarding impact was judged to outweigh strict minimalism, with the research's actual failure mode — unconfirmed wholesale prose blobs and files over 200 lines — fenced off by the successor's preview and confirm gate and its retained guardrails.
