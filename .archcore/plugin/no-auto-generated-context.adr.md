---
title: "No Auto-Generated Context Blobs"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Context

Recent research on AI agent context engineering (arxiv 2510.21413, "Context Engineering for AI Agents in Open-Source Software", Mohsenimofidi et al., 2026) measured the effect of `AGENTS.md`-style auto-generated context files across 466 open-source projects. LLM-generated context files reduced agent task success in 5 of 8 settings, increased inference cost 20–23%, and added 2.45–3.92 steps per task. Human-curated context files yielded only ~+4 percentage points of improvement, indicating that the bulk of measurable gain comes from intent and curation rather than coverage. Cloudflare's production AI code review system independently identified files >200 lines and tool names without runnable commands as performance penalizers.

A naive product instinct for a documentation-oriented plugin is to "scan the repository and produce summary documents". This ADR closes that path before it is taken.

## Decision

Archcore does not produce auto-generated repository summaries, AGENTS.md-style blobs, or any document generated wholesale by an LLM scan of the codebase. Every `.archcore/` document is created intentionally, in response to a specific authoring intent (decide, capture, plan, standard, etc.), through a skill that elicits context from the user or harvests evidence for that one document.

## Alternatives Considered

1. **Generate AGENTS.md per project on bootstrap** — rejected; arxiv 2510.21413 quantifies the harm at scale (−5/8 success, +20–23% cost) and the harm is structural, not implementation-specific.
2. **Generate per-module summary docs to seed initial context** — rejected; same failure mode at finer granularity, and creates documents users have not consciously authored, undermining the trust signal of `.archcore/`.
3. **Allow opt-in auto-summary skill behind a flag** — rejected; even opt-in availability normalizes a known anti-pattern, and over time users mistake auto-output for canonical docs, degrading the corpus.

## Consequences

- Bootstrap output remains minimal; users do not get a "free" knowledge tree from running `/archcore:bootstrap` on an existing repo. The first useful state of `.archcore/` requires deliberate authoring.
- All documents in `.archcore/` carry intentional authorship; readers (human and agent) can trust them as deliberate artifacts rather than machine fill.
- Skills that need codebase context fall back to grep/read at composition time, not pre-generated artifacts. This costs a few seconds per skill invocation but keeps context fresh.

## Superseded when

- A future benchmark on comparable scale (>400 OSS repos) demonstrates that auto-generated context files improve agent task success rate by ≥5 percentage points net of cost.
- Anthropic, Cursor, or comparable host vendors document a different official position based on newer evidence and revise their own context-engineering guidance.
