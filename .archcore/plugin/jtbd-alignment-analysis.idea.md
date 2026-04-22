---
title: "JTBD-Implementation Alignment Analysis — Repo-Alignment Gap"
status: draft
tags:
  - "architecture"
  - "marketing"
  - "plugin"
  - "vision"
---

## Idea

Analyze the Archcore Plugin against four Jobs-To-Be-Done, map each to the mechanisms that actually exist in the plugin today, and identify where the README promise outruns the engineered reality. Use this analysis to decide between two strategic paths: soften positioning to match implementation (Path A), or build the missing mechanisms so the primary JTBD becomes an engineered guarantee rather than a marketing claim (Path B).

### The four JTBDs

1. **Make a feature without breaking the repo's logic.** Agent places code where the architecture expects, follows rules, respects prior ADRs. Repo-alignment layer at coding time.
2. **Continue work without re-explaining the project.** Agent picks up prior decisions, patterns, and focus across sessions, hosts, and subagents.
3. **Record a new decision so it actually affects the next code.** Decision gets captured, AND influences future agent behavior. Document creation is a means; behavior change is the end.
4. **Walk me through a complex change-flow spanning multiple artifacts.** Multi-document cascades (PRD → plan, ADR → rule → guide) orchestrated end-to-end.

### What the plugin actually delivers per JTBD

#### JTBD #1 — Repo-alignment at coding time

Mechanisms present:

- `SessionStart` hook loads **document index** (paths, titles, types, tags, relation count) into main-session context — not content
- `PreToolUse Write|Edit` blocks writes to `.archcore/*.md` — **not** to `src/**`
- `PostToolUse` validates and detects cascade on MCP mutations — **only** on document mutations
- Intent skills (`capture`, `plan`, `decide`) activate on documentation requests, **not** on code requests
- `archcore-auditor` can cross-reference code and docs, but runs only on explicit audit request

Missing mechanism — the core gap:

- No hook fires before source-code changes
- No mechanism forces relevant ADRs/rules/specs/cpats into context when the agent is about to edit `src/api/handlers/`
- The agent must _choose_ to `get_document` on its own — and often does not
- Subagents do not inherit `SessionStart` context (see `subagent-knowledge-tree-preload.idea.md`)

Verdict: implemented as _passive nudge_ ("context is available"), not as _active guardrail_ ("context is applied"). This is the weakest-implemented JTBD.

#### JTBD #2 — Session continuity

Mechanisms present:

- `SessionStart` index load (strong for metadata-level pickup)
- Tag and relation summary (strong for "what exists")
- `check-staleness` once per 24h (moderate — warns of drift, does not fix)
- `check-cascade` on update_document (strong within current session)

Limitations worth naming:

- Index loads metadata, not content. "What was decided" requires follow-up `get_document` calls
- Subagents start blind
- Cross-host continuity is delivered by git, not by the plugin — any host with the plugin reads the same `.archcore/` from the same repo

Verdict: the strongest-implemented JTBD. This is what continuity actually buys today.

#### JTBD #3 — Decision → future code

Mechanisms present:

- `/archcore:decide` creates ADR and offers rule + guide follow-up
- `/archcore:standard-track` creates the full ADR → rule → guide chain
- PostToolUse validates and writes to git

Gap between "decision captured" and "future code respects decision":

- `check-cascade` only inspects doc→doc relations, not doc→code dependencies
- The follow-up rule + guide offer is a _suggestion_, not a _guardrail_
- Once created, the decision influences the next session only insofar as the agent reads it — same gap as JTBD #1
- `/archcore:actualize` detects "docs lag code", but not "code lags docs"

Honest reformulation: Archcore today fulfills "record a decision so the agent _sees_ it next session", not "so the agent _applies_ it". Reaching "applies" requires the same pre-code mechanisms missing in JTBD #1.

Positioning corollary: if JTBD #3 is primary in marketing, the lead entry-point should be `/archcore:standard-track`, not `/archcore:decide`. A rule is the form in which a decision becomes applicable as a constraint.

#### JTBD #4 — Multi-step cascade

Mechanisms present:

- 6 tracks (`product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`)
- Sequential creation with auto-relations
- Auto-invocable per Inverted Invocation Policy
- PostToolUse validates each step

Verdict: the most strongly engineered JTBD. Correctly positioned as "Advanced" — it is the deepest capability but the wrong primary frame (competes head-on with Spec Kit, and is document-centric where the main user job is code-centric).

### Promise-vs-reality matrix

| JTBD                        | Positioning rank (promise) | Implementation rank (reality) | Delta                                    |
| --------------------------- | -------------------------- | ----------------------------- | ---------------------------------------- |
| #1 Repo-alignment at coding | 1 (primary)                | 3 (weak — passive context)    | **Large gap**                            |
| #2 Session continuity       | 2 (secondary)              | 1 (strongest)                 | Aligned                                  |
| #3 Decision → future code   | 3 (supporting)             | 3 (half of the loop missing)  | Medium gap                               |
| #4 Multi-step cascades      | 4 (advanced)               | 2 (very strong)               | Inverse — implementation exceeds promise |

## Value

- Surfaces the gap between README claims and engineered guarantees before a visible installation bounce rate makes it obvious
- Gives a concrete action list: three mechanisms (pre-code context injection, code-alignment intent skill, subagent knowledge preload) close most of the JTBD #1 gap
- Frames positioning trade-off as explicit paths, not drift
- Honest reframing of JTBD #3 ("sees" vs "applies") prevents a second round of positioning debt

## Possible Implementation

Two strategic paths.

### Path A — Align positioning to current reality

- Demote JTBD #1 in the README. Replace "the agent places code where your system expects it" with "the agent _sees_ your architecture, rules, and decisions — so it can respect them"
- Promote JTBD #2 to primary promise. The current `SessionStart` context load is the honest differentiator vs memory tools (claude-mem, Memory Bank) because it's typed, relation-aware, and Git-backed — not just recall
- Re-frame JTBD #3 around `/archcore:standard-track` as the entry point, since that's the only path that produces a rule applicable as a constraint
- Cost: a README rewrite. Zero engineering.

### Path B — Engineer JTBD #1 into a guarantee

Three concrete additions, all with precedent in existing specs or idea documents:

1. **Sub-agent knowledge preload** — implement Option A of `subagent-knowledge-tree-preload.idea.md` (prompt preamble mandating `list_documents` + `list_relations` on agent start). Short diff, host-portable, elevates from `draft` because it now sits on the critical path of JTBD #1 and JTBD #2.
2. **Pre-code context injection** — new `PreToolUse Write|Edit` hook that runs on paths outside `.archcore/`, resolves documents referencing the target path, and injects a compact "relevant rules/ADRs/specs/cpats" summary as `additionalContext`. See `pre-code-context-injection.idea.md`.
3. **Code-oriented intent skill** — new Layer 1 intent `/archcore:align` that takes a code area as argument and returns applicable constraints from the knowledge base. See `code-alignment-intent-skill.idea.md`.

After these three, JTBD #1 shifts from "agent can see context" to "agent must see context before coding" — that is the difference between a prompt-library and a guardrail, and that is the real differentiator vs Spec Kit, claude-mem, and Memory Bank.

### Recommended interim

1. Immediately: soften the README promise on JTBD #1 to prevent first-session disappointment
2. Short-term: ship sub-agent preload (Option A) — cheapest win, unblocks JTBD #1/#2 for delegated work
3. Medium-term: pre-code context injection — the single highest-impact addition to the plugin
4. After that: re-promote JTBD #1 to primary with a hero reel that actually demonstrates the mechanism

## Risks and Constraints

- **Positioning churn risk.** Rewriting the README before Path B ships means rewriting again after. Mitigation: write the honest Path A copy to stand on its own, then extend (not replace) after Path B.
- **Pre-code hook performance.** `PreToolUse` has a 1-second budget. Path-matching across the whole document corpus on every `Write|Edit` must be pre-indexed or cached. Addressed in the concrete idea doc.
- **Hook fatigue.** If the pre-code hook injects on every edit, users will filter it out mentally. The trigger must be selective: only when the file path is referenced in at least one document, and only the top 3 most relevant docs.
- **Subagent preamble drift.** The `remove-skill-verify-mcp-preamble.cpat` explicitly removed a similar preamble pattern from SKILL.md. The subagent case is different (subagents do not run inside the main session and do not receive SessionStart) — the rationale must be spelled out explicitly in the preamble to prevent future cleanup by analogy.
- **JTBD #3 reframing affects `/archcore:decide`.** If `standard-track` becomes the primary "decision → future code" entry point, `/archcore:decide` should explicitly offer `standard-track` as its follow-up path, not just an optional rule + guide suggestion.
- **Scope discipline.** This analysis identifies three mechanisms. Shipping all three at once is tempting; doing so creates a coordination dependency where the hero reel, README rewrite, and three engineering tasks all block each other. Prefer sequential delivery.

## Related work in this repo

- `claude-plugin.prd.md` — primary promise aligned with JTBD #1 in spirit, but no FR in the PRD expresses a pre-code guardrail. Trace gap: requirements did not compile into architecture for this JTBD.
- `plugin-architecture.spec.md` — all invariants concern document operations; none concern code operations. This is the structural fingerprint of the gap identified here.
- `inverted-invocation-policy.adr.md` — made routing through Layer 1 actually happen. The three proposals here extend the same logic from document-centric routing to code-centric routing.
- `subagent-knowledge-tree-preload.idea.md` — now sits on the critical path of this analysis; status should be revisited.
- `readme-first-60-seconds.idea.md` — hero prompt choice is constrained by this analysis: until Path B ships, the prompt should demonstrate JTBD #2 (what works), not JTBD #1 (what does not yet).
