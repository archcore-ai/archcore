---
title: "JTBD-Implementation Alignment Analysis — Repo-Alignment Gap"
status: accepted
tags:
  - "architecture"
  - "marketing"
  - "plugin"
  - "vision"
---

## Status — Gap Closed (primary JTBD #1 mechanisms shipped, JTBD #3 reframing landed)

As of plugin 0.3.0, the analysis's central finding — that JTBD #1 was engineered as a passive nudge rather than an active guardrail — has been addressed on both axes.

Shipped mechanisms (both required to move from "sees" to "applies"):

- **Pull mode** — `/archcore:context` skill (commit `3dccbd5`). User-invoked, takes a code area / topic / or empty argument, returns grouped rules / ADRs / specs / cpats with guide routing and top-5 truncation.
- **Push mode** — `bin/check-code-alignment` PreToolUse Write|Edit hook (commit `87d384c`). Runs on every source-file edit outside `.archcore/`, injects top-3 applicable docs by specificity → type priority as `additionalContext`.

Path B proposal #3 — `/archcore:align` code-oriented intent skill — is **superseded**, not shipped. `/archcore:context` already provides the pull surface.

Path B proposal #1 — subagent knowledge preload — is **still open**. Tracked in `subagent-knowledge-tree-preload.idea.md` and `subagent-knowledge-tree-bootstrap.adr.md` (accepted); first implementation lives in the sub-agent preamble mandate.

Positioning status:

- README hero line now reads "auto-injected before source edits, surfaced on demand" — matches the engineered reality.
- JTBD #1 is no longer an overclaim.
- JTBD #3 reframing landed in README copy — the "Try these 3 prompts first" third prompt switched from ADR-only to a decision-with-standard-cascade prompt. Narrative now connects `/archcore:decide`'s standard-cascade output (rule) to `check-code-alignment` push-injection, so decision → enforced constraint, not decision → relation-graph inspection.

> **Update (2026-05-15):** The original "Path B proposal" referenced `/archcore:standard-track` as the marketing entry for JTBD #3. After `skill-surface-collapse.adr.md`, the standard cascade lives inside `/archcore:decide` (ADR → optional CPAT → optional rule → optional guide). The narrative is the same — decisions become rules become applied constraints — but the user-facing entry point is now `/archcore:decide`, not a separate track. Other stale references below to `/archcore:standard-track`, `/archcore:actualize`, and `/archcore:standard` should be read as `/archcore:decide` and `/archcore:audit --drift` respectively.

Kept as historical context below: the original gap analysis and promise-vs-reality matrix as of 2026-04-22.

## Idea

Analyze the Archcore Plugin against four Jobs-To-Be-Done, map each to the mechanisms that actually exist in the plugin today, and identify where the README promise outruns the engineered reality.

### The four JTBDs

1. **Make a feature without breaking the repo's logic.** Agent places code where the architecture expects, follows rules, respects prior ADRs. Repo-alignment layer at coding time.
2. **Continue work without re-explaining the project.** Agent picks up prior decisions, patterns, and focus across sessions, hosts, and subagents.
3. **Record a new decision so it actually affects the next code.** Decision gets captured, AND influences future agent behavior.
4. **Walk me through a complex change-flow spanning multiple artifacts.** Multi-document cascades (PRD → plan, ADR → rule → guide) orchestrated end-to-end.

### What the plugin actually delivers per JTBD

#### JTBD #1 — Repo-alignment at coding time

Mechanisms present (post-fix):

- `SessionStart` hook loads document index
- `PreToolUse Write|Edit` `check-archcore-write` blocks writes to `.archcore/*.md`
- **`PreToolUse Write|Edit` `check-code-alignment` injects top-3 applicable docs for source-file edits** (shipped post-analysis)
- **`/archcore:context` pull skill returns rules/ADRs/specs/cpats for a code area** (shipped post-analysis)
- `PostToolUse` validates and detects cascade on MCP mutations
- `archcore-auditor` cross-references code and docs on explicit audit request

Verdict (updated): JTBD #1 has both push and pull surfaces. Active guardrail status achieved.

#### JTBD #2 — Session continuity

Mechanisms present:

- `SessionStart` index load
- Tag and relation summary
- `bin/check-staleness` once per 24h
- `bin/check-cascade` on `update_document`
- `/archcore:audit --drift` for deep on-demand staleness analysis

Verdict: still the strongest-implemented JTBD.

#### JTBD #3 — Decision → future code

Mechanisms present:

- `/archcore:decide` creates ADR, RFC, and offers the full standard cascade (ADR → optional CPAT → optional rule → optional guide).
- PostToolUse validates and writes to git.
- `check-code-alignment` push injection means once a rule exists, every source-file edit sees it.

Honest reformulation (post-fix): Archcore now fulfills "record a decision so the next edit respects it" when the decision is taken through the full cascade (ending in a rule that `check-code-alignment` can inject). `/archcore:decide` is the entry point.

#### JTBD #4 — Multi-step cascade

Mechanisms present:

- Four flows reachable via `/archcore:plan` (product, sources, iso, feature) with references under `skills/plan/references/`.
- Continuation cascade reachable via `/archcore:decide` (ADR → optional CPAT → optional rule → optional guide).
- Sequential creation with auto-relations.
- All flows auto-invocable from natural language.
- PostToolUse validates each step.

Verdict: still strongly engineered, now reachable from natural language without the user picking a flow name.

### Promise-vs-reality matrix (as of 2026-04-22, pre-fix)

| JTBD                        | Positioning rank (promise) | Implementation rank (reality) | Delta                                    |
| --------------------------- | -------------------------- | ----------------------------- | ---------------------------------------- |
| #1 Repo-alignment at coding | 1 (primary)                | 3 (weak — passive context)    | **Large gap**                            |
| #2 Session continuity       | 2 (secondary)              | 1 (strongest)                 | Aligned                                  |
| #3 Decision → future code   | 3 (supporting)             | 3 (half of the loop missing)  | Medium gap                               |
| #4 Multi-step cascades      | 4 (advanced)               | 2 (very strong)               | Inverse — implementation exceeds promise |

Updated state (post-fix): JTBD #1 implementation rank moves from 3 → 1-tier by both push and pull, matching the primary positioning. JTBD #3 demo prompt in README now uses `/archcore:decide` + push-injection narrative, closing the "sees vs applies" copy gap.

## Value

- Surfaces the gap between README claims and engineered guarantees before a visible installation bounce rate makes it obvious
- Gives a concrete action list: three mechanisms (pre-code context injection, `/archcore:context` skill, subagent knowledge preload) close most of the JTBD #1 gap
- Frames positioning trade-off as explicit paths, not drift
- Honest reframing of JTBD #3 ("sees" vs "applies") prevents a second round of positioning debt

## Possible Implementation

Two strategic paths.

### Path A — Align positioning to current reality

- Demote JTBD #1 in the README. — *Not pursued; Path B shipped instead.*
- Promote JTBD #2 to primary promise. — *Not pursued.*
- Re-frame JTBD #3 around the standard cascade as the entry point, since that's the only path that produces a rule applicable as a constraint. — *Pursued; entry point is now `/archcore:decide` with continuations.*

### Path B — Engineer JTBD #1 into a guarantee

Three concrete additions:

1. **Sub-agent knowledge preload** — implement Option A of `subagent-knowledge-tree-preload.idea.md` (prompt preamble mandating `list_documents` + `list_relations` on agent start). — *Implemented as preamble mandate.*
2. **Pre-code context injection** — new `PreToolUse Write|Edit` hook that runs on paths outside `.archcore/`. — *Shipped as `bin/check-code-alignment`.*
3. **Code-oriented intent skill** — new intent that takes a code area and returns applicable constraints. — *Shipped as `/archcore:context`.*

After these three, JTBD #1 shifts from "agent can see context" to "agent must see context before coding".

## Risks and Constraints

- **Positioning churn risk.** Rewriting the README before Path B ships means rewriting again after. — *Mitigated; Path B shipped first.*
- **Pre-code hook performance.** `PreToolUse` has a 1-second budget. Path-matching must be pre-indexed or cached. — *Addressed in the implementation plan.*
- **Hook fatigue.** Trigger is selective: only when the file path is referenced in at least one document, and only top 3 docs.
- **Subagent preamble drift.** The `remove-skill-verify-mcp-preamble.cpat` removed a similar preamble pattern from SKILL.md. The subagent case is different and the rationale is spelled out explicitly in the preamble.
- **JTBD #3 reframing affects `/archcore:decide`.** — *Resolved by `skill-surface-collapse.adr.md`: `decide` is now the entry point for the standard cascade, with optional rule and guide continuations.*
- **Scope discipline.** Sequential delivery preferred. — *Followed.*

## Related work in this repo

- `claude-plugin.prd.md` — primary promise aligned with JTBD #1 in spirit, FRs added in v0.3.0 to express pre-code guardrail.
- `plugin-architecture.spec.md` — invariants now cover both document and source-code path operations.
- `inverted-invocation-policy.adr.md` — superseded by `skill-surface-collapse.adr.md` for the per-class invocation flags; the auto-invocation principle endures.
- `skill-surface-collapse.adr.md` — final 7-skill surface that hosts JTBD #1's pull (`context`) and JTBD #3's decision cascade (`decide`).
- `subagent-knowledge-tree-preload.idea.md` — now sits on the critical path of this analysis.
- `readme-first-60-seconds.idea.md` — hero prompt choice now demonstrates the active guardrail.
