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

Add a bounded **clarify pass** at the skill layer: an intent-heavy authoring skill instructs the host agent to run a short structured interview **before** composing a document, guided by a shared runtime contract. The questioning lives entirely in portable skill markdown — "ask the user …" — rather than in MCP Elicitation or the host-native `AskUserQuestion` tool.

This is the one elicitation variant of three that fits Archcore. **MCP Elicitation** (`elicitation/create`) is the wrong layer: the Archcore MCP server is thin document CRUD and the `SKILL.md` is the orchestrator, while elicitation is meant for servers that drive interactive workflows. It is also capability-gated, primitives-only with no nested or array schema, and unsupported on OpenCode — one of the five host targets — which forces a conversational fallback anyway. **Host `AskUserQuestion`** is Claude-Code-only, model-owned so a plugin can neither call it nor define its questions, and unavailable in subagents, so `archcore-assistant`, the multi-document workhorse where clarification pays most, could never use it.

Skill-text elicitation avoids both traps: it is byte-identical markdown that any host agent can execute in prose, satisfying the host-adapter contract's content-identical invariant. Where a host has a native question widget the agent uses it for free, and elsewhere the pass degrades to clean prose questions — one code path, all five hosts, subagents included.

The pass is the elicitation counterpart to init's evidence extraction, splitting document types by where the bottleneck sits: extraction serves documents whose answer is **in the code** — spec, doc, most capture work, and the init facts — while the clarify pass serves documents whose answer is **only in the user's head**: `prd`, `idea`, the sources and ISO requirement cascades, and ADRs about genuine trade-offs.

## Value

- It attacks the exact failure the precision philosophy exists to fight — plausible-but-wrong LLM synthesis — on documents where no code evidence exists to ground it, replacing a guess with the user's actual answer.
- It closes the loop on the `[assumption]` markers the precision rules already emit. Nothing resolves them today; the pass surfaces the top-ranked assumptions as questions and writes the answers back.
- The written-back answers form a `## Clarifications` section, which is itself an operational, falsifiable, traceable artifact, consistent with `precision-over-coverage.adr` rather than at odds with it.
- It makes the sources and ISO cascades credible, since those are by definition stakeholder-interview artifacts and generating them with zero elicitation is the weakest part of the plan flows.
- It differentiates the product: Archcore could *produce* a well-formed spec or PRD from a rough intent, instead of assuming the user arrived with one.

## Possible Implementation

1. Add `skills/_shared/elicitation-contract.md`, structured like the existing precision, ADR, spec, and rule contracts, defining a bounded protocol lifted from Spec Kit `/clarify` and Kiro Analyze-Requirements: rank candidate gaps by impact times uncertainty and ask only where no sensible default exists; cap at 3–5 questions and stop early on a user "done" signal; lead each option set with the recommended choice and offer a free-text escape hatch; write answers back into a `## Clarifications` section or resolve `[assumption]` markers in place; and use host-agnostic phrasing only, never a tool name.
2. Wire the contract into the intent-heavy skills: `plan` across its product, feature, sources, and ISO flows, which is the highest value; `decide` only for ADRs with real trade-offs, gated on decision drivers; and `capture` only where code evidence is thin or ambiguous.
3. Leave `init` unchanged, because its preview and confirm is already the right gate and an interview does not belong on first run.
4. Raise the per-step question cap for intent composition only, citing the contract in the skills and commands specs, while the at-most-one-scope-question default stays for everything else.

## Risks

- **Question fatigue against the minimalist ethos.** Mitigated by the hard cap, the recommendation-led options, the only-when-no-sensible-default rule, and the restriction to intent-heavy types — never on capturing a known module and never on init.
- **Portability regression.** Mitigated by keeping the pass in the skill layer and forbidding any hard dependency on `AskUserQuestion` or MCP elicitation; the structure tests already pin the absence of host-conditional text in skills.
- **Determinism and testing.** An interactive step is non-deterministic, so it stays a specified contract like `adr-contract.md` and remains reviewable, with tests covering the contract's presence and loading rather than the dialogue.
- **Palette pressure.** The four-command palette is ADR-gated (`four-command-palette.adr`), so this must ship as a mode inside an existing command (plan or document track) rather than as a new command.

## Open decisions

- **Batched versus one at a time.** Batching up to 4 questions in one turn minimizes round-trips for a terminal-first tool and matches the Claude widget's shape; asking one at a time, as Spec Kit does, reads better on a host with no widget.
- **Opt-in versus automatic.** Start with an explicit `--clarify` flag on `plan` and `decide`, and promote to automatic for the sources and ISO flows once proven, which is the conservative path given the question-minimal default.
