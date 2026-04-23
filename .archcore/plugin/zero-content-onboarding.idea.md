---
title: "Zero-Content Onboarding — From Empty .archcore to First Useful State"
status: draft
tags:
  - "onboarding"
  - "plugin"
  - "vision"
---

## Idea

A new user installs the plugin on an existing repo. `.archcore/` is empty. SessionStart loads zero documents. They open a file in `src/api/` — `check-code-alignment` greps an empty knowledge base and injects nothing. `/archcore:context src/api/` returns "no documents reference this path." `/archcore:status` reports "0 documents."

The plugin gives no signal that anything has changed. The user does not know what to do next, sees no value, and uninstalls or forgets. **All the machinery that delivers JTBD #1 / #2 / #3 silently no-ops on a fresh repo until the user manually populates `.archcore/` — but nothing in the plugin guides them through that first step.**

This is a structural blind spot: every push and pull mechanism is content-conditional, and the plugin currently treats the empty state as a normal state rather than as a special case requiring its own product surface.

The proposal: add an **activation layer** that turns the empty state into a guided first step. Three variants exist along a cost/scope spectrum, listed cheap-to-ambitious:

### Variant A — Empty-state nudge in SessionStart (cheapest)

When `bin/session-start` (or its CLI counterpart) detects an empty or missing `.archcore/`, it emits a single advisory line into the SessionStart payload:

> `.archcore/ is empty. Try /archcore:capture to document an existing convention, or /archcore:standard to codify a team rule. Use /archcore:help for the full surface.`

That's it. No new skill, no scanning, no UI — one branch in the existing hook. Pure copy. Cost: ~10 lines of shell. Idempotent — silent on populated repos.

This already removes the "I installed it and nothing happens" failure mode. It does not produce the first document, but it tells the user how to.

### Variant B — `/archcore:bootstrap` intent skill (middle ground)

A new Layer 1 intent skill that runs an explicit first-time onboarding flow:

1. **Detect state.** If `.archcore/` already has ≥3 documents, inform the user it's not their first session and route them to `/archcore:context` or `/archcore:status`. Otherwise proceed.
2. **One question.** "What's the most important convention or decision in this repo that you'd want a future agent to know about?" — single open question, one round trip.
3. **Route to existing skill.** Based on the answer's shape:
    - "we always do X" → route to `/archcore:standard` (ADR → rule → guide chain)
    - "we decided to use X over Y because Z" → route to `/archcore:decide`
    - "X works like this …" → route to `/archcore:capture` (auto-routes to spec / doc / guide)
    - Ambiguous → propose `/archcore:capture` as default
4. **Close the loop.** After the first document is created, the skill prints one paragraph: "you'll see this document automatically appear in `additionalContext` next time you edit a file under <path>. Check via /archcore:context <path> to confirm." This makes the value-loop visible immediately.

This is a thin orchestrator over already-shipped skills. Net new code: one SKILL.md with routing logic. Reuses `/archcore:standard`, `/archcore:decide`, `/archcore:capture` underneath.

### Variant C — `/archcore:bootstrap --scan` with repo introspection (ambitious)

Same skill as B, but adds an opt-in scan mode that reads the repo to suggest concrete document candidates:

- Read `README.md`, top-level directory names, `package.json` / `pyproject.toml` / `Cargo.toml` for stack signals.
- Propose 3–5 candidate documents with concrete titles: "ADR: [stack choice] for primary persistence based on package.json", "rule: API handlers live in src/api/ (detected from existing structure)", "doc: monorepo layout (detected 3 top-level packages)".
- User picks 0..N to seed; skill creates the chosen documents using the existing type skills.

This is the "magic first day" path — but it's also the one that fails loudly when the scan suggests garbage. False-positive scans create the opposite problem: the user dismisses the plugin as "it just generates noise."

## Value

- **Closes the silent-noop failure mode.** Today's worst outcome on install: nothing happens, user uninstalls. Variant A alone removes this. The cost is 10 lines.
- **Bridges install → first useful state.** Plugin marketplace adoption is funnel-shaped: install is cheap, retention requires a "wow" moment within the first session. Without zero-content onboarding, the wow moment is delayed until the user manually creates their first rule — which most users won't do unprompted.
- **Activates JTBD #1/#2/#3 mechanisms.** Push (`check-code-alignment`), pull (`/archcore:context`), and influence loops (`/archcore:decide` → injection) all need at least one document to demonstrate. Onboarding is the gating step before any of those three jobs visibly works.
- **Aligns README promise with first-session reality.** README's three try-me prompts tacitly assume the user already has documents to work with. Adding Variant A or B closes the gap between "what the README implies happens on install" and "what the user actually sees."
- **Distinguishes from competitors.** Memory tools (claude-mem, Mem0) accumulate state automatically over time and do not need bootstrap. Spec Kit assumes the user is already building from a spec. Archcore's typed-document model puts the user in the position of having to author the first artifact — onboarding makes that authoring trivial instead of intimidating.

## Possible Implementation

Recommended sequence: **A first, B next, C deferred.**

### Step 1 — Variant A (cheapest, immediate)

- Modify `bin/session-start` (and the CLI's session-start handler if it owns the empty-state check). Detect missing or empty `.archcore/` and append a single advisory paragraph to the existing payload.
- Add a `ARCHCORE_HIDE_EMPTY_NUDGE=1` env var for users with intentionally empty test repos.
- Test: structural assertion in the existing session-start bats suite that the nudge appears on an empty fixture and disappears once the fixture has any `.md` file.
- One-line README changelog.

### Step 2 — Variant B (`/archcore:bootstrap` skill)

- New `skills/bootstrap/SKILL.md` following the standard intent-skill structure.
- Triggers: explicit `/archcore:bootstrap` invocation. Mentioned in Variant A's nudge as the recommended first step (replacing the `/archcore:capture` and `/archcore:standard` references).
- Routing logic: detect state (count documents via `list_documents`), branch into "already populated" vs "first time", ask the single question, route into existing skills.
- Reuse `/archcore:capture`, `/archcore:standard`, `/archcore:decide` — bootstrap is glue, not new functionality.
- Close the loop with the visibility line ("you'll see this in `additionalContext` next time …") to prove the value chain works.
- Test: structural test that the SKILL.md exists with the standard frontmatter.
- Update `commands-system.spec.md` and `skills-system.spec.md` to register the new intent skill.

### Step 3 — Variant C (deferred until B has user feedback)

- Decide go/no-go after observing 2–4 weeks of B in the wild.
- If B users consistently abandon at the "what's most important" question, scan-based suggestions (C) are likely the next step. If B users pick a direction confidently from the open prompt, C is unnecessary noise.
- The decision should be data-informed; do not pre-build C.

### Cross-cutting changes

- README — add a single paragraph after "Try these 3 prompts first": "Empty repo? Run `/archcore:bootstrap` first."
- `claude-plugin.prd.md` — add a goal/FR for "first-session activation on empty `.archcore/`".
- `development-roadmap.plan.md` — slot Variant A under near-term, Variant B in the next release window.

## Risks and Constraints

- **Nudge fatigue (Variant A).** Users with intentionally empty `.archcore/` (e.g., running plugin tests in a throwaway repo) will see the message every session. Mitigation: env-var opt-out plus a one-time "got it, don't show again" flag stored in `.archcore/.local/state.json` (or its CLI equivalent) once any document is created.
- **Bootstrap quality bar (Variant B).** A bad first-document experience is worse than no first-document experience — if the routing question produces a confused output, users learn to distrust the plugin. Mitigation: keep the bootstrap question open enough to absorb unclear answers (route to `/archcore:capture` as default, since capture handles ambiguity well).
- **Scope creep into Variant C.** Tempting to scan and pre-fill on day one. Resist — scan false-positives are the high-cost failure mode and we have no signal yet that the open prompt isn't enough.
- **Conflicts with `readme-first-60-seconds`.** That idea targets the pre-install funnel (visitor sees value before clicking install). This idea targets post-install activation (installed user sees value before disengaging). Both are needed; they should not be merged. Variant A's nudge text and the README hero should reference each other but stay separate documents/surfaces.
- **Empty-state detection edge cases.** A repo with only `.archcore/.gitkeep` or with a partially-populated `.archcore/` (1–2 stub documents from a prior session that didn't take) is functionally empty for the user but technically populated. Threshold: count documents with `status` in `{accepted, draft}` AND `body length ≥ 200 chars`. Stub or scaffold-only documents don't count.
- **Bootstrap skill discoverability paradox.** If users don't know `/archcore:bootstrap` exists, they won't run it. Variant A's nudge solves this — the empty-state hook teaches the user about the skill at exactly the moment they need it. Variants B and C are valuable only if A ships first to surface them.
- **CLI vs plugin ownership of the empty-state check.** SessionStart payload is composed by the CLI's MCP handler; the plugin's `bin/session-start` wraps it. Either layer can do the check, but only one should — duplicate nudges would be a regression. Likely the CLI is the right layer (consistent across hosts), but for Variant A it can ship plugin-side first and migrate to CLI later.

## Related work in this repo

- `jtbd-alignment-analysis.idea` (accepted) — analyzes JTBD #1/#2/#3 mechanisms but assumes content already exists. This idea fills the missing pre-condition: how does content get there in the first place.
- `readme-first-60-seconds.idea` (draft) — adjacent funnel stage (pre-install visitor → installer). This idea picks up where that one ends (installer → first useful state). Both should ship; neither replaces the other.
- `context-skill-implementation.plan` and `pre-code-hook-implementation.plan` — both shipped mechanisms that visibly no-op on empty `.archcore/`. Onboarding is what makes their first invocation produce visible output.
- `claude-plugin.prd.md` — currently has no FR or goal that addresses the empty-state user journey. Adding one as part of Step 1 closes a documented PRD gap.
- `inverted-invocation-policy.adr` — established the routing layer (Layer 1 intents). `/archcore:bootstrap` is a new Layer 1 intent and should follow the same auto-invocation semantics.
- `commands-system.spec` and `skills-system.spec` — would need to register the new intent skill if Variant B ships.
