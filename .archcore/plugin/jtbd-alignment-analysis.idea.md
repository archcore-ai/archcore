---
title: "JTBD-Implementation Alignment Analysis — Repo-Alignment Gap"
status: accepted
tags:
  - "architecture"
  - "marketing"
  - "plugin"
  - "vision"
---

**Status — gap closed.** As of plugin 0.3.0 the central finding — that JTBD #1 was engineered as a passive nudge rather than an active guardrail — is addressed on both axes. Pull mode shipped as the `/archcore:context` skill in commit `3dccbd5`, taking a code area, a topic, or an empty argument and returning grouped rules, ADRs, specs, and cpats with guide routing and top-5 truncation. Push mode shipped as `bin/check-code-alignment` in commit `87d384c`, running on every source-file edit outside `.archcore/` and injecting the top 3 applicable documents by specificity and then type priority. The third Path B proposal, a dedicated `/archcore:align` intent, is superseded rather than shipped, because `/archcore:context` already provides the pull surface. The first, sub-agent knowledge preload, is tracked separately and its first implementation is the sub-agent preamble mandate.

Positioning followed: the README hero line reads "auto-injected before source edits, surfaced on demand", which matches the engineered reality, so JTBD #1 is no longer an overclaim; and the JTBD #3 reframing landed, with the third demo prompt switching from ADR-only to a decision-with-standard-cascade prompt, so the narrative connects the cascade's rule output to push injection — decision to enforced constraint, rather than decision to relation-graph inspection.

**Naming note (2026-05-15).** The original Path B referenced `/archcore:standard-track` as the marketing entry for JTBD #3. After `skill-surface-collapse.adr` the standard cascade lives inside `/archcore:decide`. The narrative is unchanged — decisions become rules become applied constraints — but the entry point is `/archcore:decide`. Read any stale `/archcore:standard-track` or `/archcore:standard` below as `/archcore:decide`, and `/archcore:actualize` as `/archcore:audit --drift`.

**Naming note (v2).** After `four-command-palette.adr` and `remove-context-command.adr` the seven-command surface narrated below is historical. Read `/archcore:context` throughout as the removed pull command — CLI hooks and command grounding absorb it. Read `/archcore:decide` throughout as `/archcore:document`. Read `/archcore:audit --drift` throughout as `/archcore:review --drift`. This covers every stale instance below: "Pull mode shipped as the `/archcore:context` skill", "the `/archcore:context` pull skill", "`/archcore:decide` creates the ADR or RFC", "`/archcore:audit --drift` for deep on-demand analysis", "reachable through `/archcore:decide`", and "the standard cascade lives inside `/archcore:decide`".

## Idea

Analyze the plugin against four jobs-to-be-done, map each to the mechanisms that actually exist, and identify where the README promise outruns the engineered reality.

The four jobs are: **#1**, make a feature without breaking the repository's logic, so the agent places code where the architecture expects, follows the rules, and respects prior ADRs — repo alignment at coding time; **#2**, continue work without re-explaining the project, so the agent picks up prior decisions, patterns, and focus across sessions, hosts, and sub-agents; **#3**, record a new decision so it actually affects the next code, meaning the decision is both captured and influential; and **#4**, walk through a complex change flow spanning multiple artifacts, with multi-document cascades orchestrated end to end.

**JTBD #1 — repo alignment at coding time.** The mechanisms are the SessionStart index load, the pre-mutation guard blocking `.archcore/` writes, the injection guard supplying the top 3 applicable documents for a source edit, the `/archcore:context` pull skill, PostToolUse validation and cascade detection on MCP mutations, and `archcore-auditor` cross-referencing code against documents on request. With both push and pull present, the active-guardrail status is achieved.

**JTBD #2 — session continuity.** The mechanisms are the SessionStart index load, the tag and relation summary, `bin/check-staleness` once per 24 hours, `bin/check-cascade` on `update_document`, and `/archcore:audit --drift` for deep on-demand analysis. This remains the most strongly implemented job.

**JTBD #3 — decision to future code.** `/archcore:decide` creates the ADR or RFC and offers the full cascade through optional CPAT, rule, and guide; PostToolUse validates and the result lands in git; and push injection means that once a rule exists, every source-file edit sees it. The honest reformulation is that Archcore fulfills "record a decision so the next edit respects it" when the decision runs through the full cascade, ending in a rule the injection guard can carry.

**JTBD #4 — multi-step cascade.** Four flows are reachable through `/archcore:plan` with their references, the continuation cascade is reachable through `/archcore:decide`, creation is sequential with automatic relations, every flow auto-invokes from natural language, and PostToolUse validates each step. Strongly engineered, and now reachable without the user picking a flow name.

**Promise against reality, as measured on 2026-04-22, before the fix:**

| JTBD | Positioning rank (promise) | Implementation rank (reality) | Delta |
| --- | --- | --- | --- |
| #1 Repo-alignment at coding | 1 (primary) | 3 (weak — passive context) | **Large gap** |
| #2 Session continuity | 2 (secondary) | 1 (strongest) | Aligned |
| #3 Decision → future code | 3 (supporting) | 3 (half of the loop missing) | Medium gap |
| #4 Multi-step cascades | 4 (advanced) | 2 (very strong) | Inverse — implementation exceeds promise |

After the fix, JTBD #1's implementation rank moves to the top tier on both push and pull, matching its primary positioning, and the JTBD #3 demo prompt closes the "sees versus applies" copy gap.

## Value

- It surfaces the gap between README claims and engineered guarantees before a visible installation bounce rate makes it obvious.
- It yields a concrete action list: three mechanisms close most of the JTBD #1 gap.
- It frames the positioning trade-off as explicit paths rather than drift.
- The honest reframing of JTBD #3 prevents a second round of positioning debt.

## Possible Implementation

**Path A — align positioning to the current reality.** Demote JTBD #1 in the README, promote JTBD #2 to the primary promise, and reframe JTBD #3 around the standard cascade as the entry point, since that is the only path producing a rule usable as a constraint. The first two were not pursued; the third was, and the entry point is now `/archcore:decide` with its continuations.

**Path B — engineer JTBD #1 into a guarantee**, through three additions: sub-agent knowledge preload, implemented as the preamble mandate; pre-code context injection, shipped as `bin/check-code-alignment`; and a code-oriented intent skill, shipped as `/archcore:context`. After these, JTBD #1 shifts from "the agent can see context" to "the agent must see context before coding". Path B shipped.

## Risks

- **Positioning churn.** Rewriting the README before Path B shipped would have meant rewriting it again afterwards. Mitigated by shipping Path B first.
- **Pre-code hook performance.** The pre-mutation budget is 1 second, so path matching must be pre-indexed or cached. Addressed in the implementation plan.
- **Hook fatigue.** The trigger is selective: it fires only when the file path is referenced by at least one document, and it caps at three.
- **Sub-agent preamble drift.** `remove-skill-verify-mcp-preamble.cpat` removed a similar-looking preamble from `SKILL.md` files. The sub-agent case differs, and the rationale is spelled out inside the preamble itself.
- **Scope discipline.** Sequential delivery was preferred, and followed.
