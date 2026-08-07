---
title: "AI-SDLC Flow Patterns and SDD Gate Research — Evidence Base for the v2 Redesign"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "vision"
---

## Goal

Ground the plugin v2 redesign in market evidence: the flow shapes of AI-SDLC tools, what their gates validate, how their interview protocols work, and why teams abandon them.

## Questions

1. Which flow archetypes exist across AI-SDLC tooling?
2. What does each tool machine-validate at its gates?
3. How do the strongest question-driven elicitation protocols work?
4. Which measured failure modes must the track layer avoid?

## Approach

Web sweeps on 2026-08-04 over primary sources — Spec Kit command templates, OpenSpec docs, Kiro docs, BMAD skills, the grilling skill — plus published critiques carrying measurements; cross-checked against read-only inventories of this repo and the CLI repo. 23 tools compared.

## Findings

1. Eight flow archetypes cover the market: A spec-pipeline (Spec Kit, Kiro, BMAD — per-feature artifacts die after merge); B process wrapper (Superpowers, ECC — validates work, not knowledge); C static preload (AGENTS.md world — manual closure); D personal auto-memory (host memories — app-side, unvalidated); D2 vendor org-brain (Devin Knowledge, Cursor Team Rules); E work ledger (Beads, Taskmaster — task state, not knowledge); F delta-merge into living truth (OpenSpec — strongest closure observed, still partial); G rebuild from scratch (Aider repo-map, Amp).
2. Empty market cell: no surveyed tool combines automatic + team-shared + git-native + machine-validated knowledge closure. The four machine-validated closures found validate tests, task graphs, delta structure, or command mechanics — never knowledge against code.
3. Elicitation protocols: Spec Kit `/clarify` — cap of 5 questions, one at a time, recommended default on each, answer saved into the spec after every reply (https://github.com/github/spec-kit/blob/main/templates/commands/clarify.md); BMAD — numbered critique menus at section checkpoints; grilling — 16–50 questions, depth praised by users, no artifact discipline (https://github.com/mattpocock/skills). Strongest portable combination: grilling depth under `/clarify` persistence, budget scaled by uncertainty.
4. Measured failure modes: doc-to-code line ratio 7–9× and full flow 33.5 min versus 8 min iterative (https://blog.scottlogic.com/2025/11/26/putting-spec-kit-through-its-paces-radical-idea-or-reinvented-waterfall.html); 181,040 versus 91,729 tokens for comparable work with lower success (https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html); non-blocking validation degrades to decoration (Spec Kit `/analyze`, OpenSpec `--strict`); overkill on small changes is the top abandonment cause (https://github.com/github/spec-kit/discussions/1784); no route between a spec and a bug — users cannot tell whether to fix code or spec (https://github.com/github/spec-kit/discussions/1686).
5. Gate lessons adopted into the track-layer design: a mandatory cheap-path skip on every gate; blocking versus advisory declared per check; delta-shaped updates merged into the living canon (archetype F applied at document-type granularity); the draft artifact as the resumable checkpoint (Kiro's approval flag corresponds to `status: accepted`).

## Recommendation

Six tracks mapped to archetypes: sdd ← A; requirements-cascade ← A-formal; decision ← the empty market cell (nobody stores "why"); actualize ← F; describe ← C-to-F promotion; experience ← D. Archetypes C, E, G, D2 deliberately do not become tracks: C is the background hook channel, E is out of product scope, G is the empty-base behavior of every command, D2 is the kernel's globals mechanism.

## Next Action

Execute Stage 0 of the plugin v2 implementation plan: migrate this base itself from the 7-command canon to the v2 canon.