---
title: "Track Machinery Enforcement Audit — Machine Perimeter Versus Prompt Interior"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "validation"
---

## Goal

Determine which parts of the track machinery are machine-enforced versus prompt-enforced, compare the split against the surveyed field, and order hardening work relative to the delta-routing redesign.

## Questions

1. What does the plugin machine-enforce today, and what holds only as prompt contract?
2. How do the field's enforcement classes fail in practice?
3. Which machine mechanisms transfer at low cost?
4. Does hardening precede or follow the routing redesign?

## Approach

Internal reading on 2026-08-15 of @plugins/archcore/skills/_shared/gate-contract.md, @plugins/archcore/skills/_shared/elicitation-contract.md, the host hook configs, and the corpus specs for the hook layer and the track layer. Field side: the seven research passes recorded in the routing-bases research; URLs fetched 2026-08-15.

## Findings

1. The machine perimeter is in place and tested: the PreToolUse write guard denies direct writes to `.archcore/` documents on every host with pre-mutation hooks, fails open without the CLI, and is pinned byte-level by the bats goldens.
2. The gate interior holds only as prompt contract: skip_when evaluation, blocking and advisory exit checks, question budgets, the `archcore:track` state block, resume rules, and the status ceremony carry no machine check.
3. PostToolUse validation — doctor findings, cascade notice, precision scan — exits 0 in every outcome; the prior flow-patterns research recorded non-blocking validation degrading to decoration across the field, and the plugin's own validation layer sits in that class.
4. The status ceremony (track-layer.spec rules 14–15) is not observable at the MCP boundary: a user-confirmed `update_document` status change and an autonomous one carry the same call shape, so the no-autonomous-acceptance invariant binds only through model compliance.
5. Question budgets are arithmetic maintained by the model inside the state block; no validator reads the `budget` or `asked` fields.
6. The state block is model-authored text without schema validation — the class where every surveyed tool shows drift: invented status values (gotalab/cc-sdd#162), corrupted spec structure (codervisor/leanspec#197), state split-brain (Priivacy-ai/spec-kitty#3323).
7. The relations manifest is one shared file per project; plain-git shared state at multi-agent scale failed for Beads and forced a storage migration (https://steve-yegge.medium.com/beads-blows-up-a0a61bb889b4).
8. Enforcement rigor and adoption are uncorrelated in the field: the most-adopted skills framework runs on prompt discipline plus one SessionStart hook (https://github.com/obra/superpowers), the most machine-gated hobby system carries exit-2 hooks and 19 stars (https://github.com/tiago-peixoto/claude-shapeup), and the most machine-enforced tool overall also carries the largest recorded bug surface (Priivacy-ai/spec-kitty, 644 open issues at fetch). BMAD added its machine readiness gate only after consolidating the flow in v6 (https://github.com/bmad-code-org/BMAD-METHOD).
9. Transferable mechanisms, ordered by cost: a server-side status-transition guard modeled on receipts (AI-DLC v2 refuses authority-bearing state changes outside owning tools, https://github.com/awslabs/aidlc-workflows); a state-block shape check inside the existing doctor pass; an exit-1 requirement-coverage gate (Spec Kitty's finalize traceability gate; MoAI-ADK's CoverageIncomplete lint running `--strict` in CI, https://github.com/modu-ai/moai-adk); a token-budget linter (LeanSpec enforces warn 3,500 / error 5,000 tokens via tiktoken, https://github.com/codervisor/leanspec); line-wise merge drivers for shared state files (Spec Kitty ships them for its event log); behavioral scenario tests with an ablation control (the claude-shapeup test harness).
10. The bats suite pins the machine layer byte-level, and no behavioral test exercises track compliance — whether a session honors gates, budgets, or the ceremony is unverified.

## Recommendation

Harden after the delta-routing redesign stabilizes: machine checks and byte-pinned goldens raise the cost of changing gate names and shapes, and the redesign changes both. One item moves ahead of the redesign — the MCP status-transition guard — because drift detection, closeout, and global-source inheritance all read statuses, and the guard is independent of gate shape. Convert the delta-routing bench into behavioral routing tests at phase 1; the remaining mechanisms land with phase 3.

## Next Action

The status-transition guard goes to `/archcore:document` as a decision spanning the plugin and the CLI repository; the coverage gate, the state-block check, and the token linter follow the phase-3 entry of the delta-routing idea.