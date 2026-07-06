---
title: "Skill-Level Elicitation Pass — Portable Clarify Interview via SKILL.md, No New Tool"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "precision"
  - "skills"
---

## Idea

Add a bounded **clarify pass** to the skill layer: intent-heavy authoring skills instruct the host agent to run a short structured interview **before** composing a document, guided by a shared runtime contract. The questioning lives entirely in portable skill markdown ("ask the user …") — **not** in MCP Elicitation and **not** in the host-native `AskUserQuestion` tool.

This is the one elicitation variant of three that fits Archcore. The other two were rejected:

- **MCP Elicitation** (`elicitation/create`) is the wrong layer. Archcore's MCP server is thin document CRUD; the SKILL.md is the orchestrator. Elicitation is for servers that drive interactive workflows — ours does not. It is also capability-gated, primitives-only (no nested/array schema), and unsupported on OpenCode (one of the five host targets), forcing a conversational fallback regardless.
- **Host `AskUserQuestion`** is Claude-Code-only, model-owned (a plugin cannot call it or define its questions), and unavailable in subagents — so `archcore-assistant`, the multi-document workhorse where clarification pays most, could never use it.

Skill-text elicitation avoids both traps. It is byte-identical markdown that any host agent can execute in prose, satisfying the host-adapter contract's "content identical across hosts" invariant. Where a host has a native question widget (Claude Code renders skill questions through `AskUserQuestion`; Cursor/Windsurf have their own), the agent uses it for free; elsewhere it degrades to clean prose questions. No host dependency, one code path, all five hosts, and subagents included.

The pass is the elicitation counterpart to init's evidence extraction. Split document types by where the bottleneck lives: extraction serves documents whose answer is **in the code** (spec, doc, most capture, init facts); the clarify pass serves documents whose answer is **only in the user's head** (prd, idea, the sources and ISO requirement cascades, and ADRs about genuine trade-offs).

## Value

- Attacks the exact failure the precision philosophy exists to fight — plausible-but-wrong LLM synthesis — on documents where no code evidence exists to ground it. The pass replaces a guess with the user's actual answer.
- Closes the loop on `[assumption]` markers the precision rules already emit: today nothing resolves them; the pass surfaces the top-ranked assumptions as questions and writes the answers back.
- The written-back answers form a `## Clarifications` section — itself an operational, falsifiable, traceable artifact, consistent with precision-over-coverage rather than at odds with it.
- Makes the sources (mrd/brd/urd) and ISO (brs/strs/syrs/srs) cascades credible: those are by definition stakeholder-interview artifacts, and generating them with zero elicitation is the weakest part of the plan flows.
- Differentiator: lets Archcore *produce* a well-formed spec/PRD from a rough intent, instead of assuming the user already arrived with one.

## Possible Implementation

1. Add `skills/_shared/elicitation-contract.md`, structured like the existing precision/adr/spec/rule contracts. It defines a bounded protocol lifted from Spec Kit `/clarify` and Kiro Analyze-Requirements:
   - Rank candidate gaps by **Impact × Uncertainty**; ask only where there is **no sensible default**.
   - Hard cap **3–5 questions**; stop early on a user "done" signal.
   - **Recommendation-led options** (recommended choice first, mirroring the existing "(Recommended)" convention) plus a free-text escape hatch.
   - **Write back** into a `## Clarifications` section and/or resolve `[assumption]` markers in place.
   - Host-agnostic phrasing only — "ask the user", never a tool name.
2. Wire the contract into intent-heavy skills: `plan` (product + feature + sources + iso flows — highest value), `decide` (only for ADRs with real trade-offs, gated on Decision Drivers), and `capture` (only when code evidence is thin/ambiguous — the Spec Kit "max 3 markers" discipline).
3. Leave `init` unchanged. Its preview→confirm is already the right gate; do not bolt an interview onto first-run.
4. Raise the per-step question cap **for intent composition only** in the skills and commands specs, citing the contract; the "at most one scope question" default stays for everything else.

## Risks

- **Question fatigue vs the minimalist ethos.** Mitigation: hard cap, recommendation-led, only-when-no-sensible-default, and only on intent-heavy types — never on capture-a-known-module or init.
- **Portability regressions.** Mitigation: skill layer only; forbid any hard dependency on `AskUserQuestion` or MCP elicitation; structure tests already pin "no host-conditional text in skills".
- **Determinism/testing.** An interactive step is non-deterministic; keep it a specified contract (like adr-contract) so it stays reviewable, and test the contract's presence/loading rather than the dialogue.
- **Palette pressure.** The 7-command surface is ADR-gated; this must ship as a mode/pass inside existing skills, not `/archcore:clarify`.

## Open decisions

- **Batch vs one-at-a-time.** Batched (≤4 questions in one turn, matching the Claude widget's 4×2–4 shape) minimizes round-trips for a terminal-first tool; one-at-a-time (Spec Kit) reads better on non-widget hosts.
- **Opt-in vs automatic.** Start with an explicit `--clarify` flag on `plan`/`decide`, promote to automatic-for-`sources`/`iso` once proven — conservative given the question-minimal default.
