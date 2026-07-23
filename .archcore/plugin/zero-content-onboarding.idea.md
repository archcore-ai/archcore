---
title: "Zero-Content Onboarding — From Empty .archcore to First Useful State"
status: accepted
tags:
  - "onboarding"
  - "plugin"
  - "vision"
---

> **Outcome (2026-05-15):** Idea executed. The skill shipped as `skills/init/` and the command is `/archcore:init` per `skill-surface-collapse.adr.md` (originally drafted as `skills/bootstrap/`). Variants A and B shipped; Variant C remains deferred. Read `/archcore:bootstrap` and `skills/bootstrap/` below as `/archcore:init` and `skills/init/`.

## Idea

A new user installs the plugin on an existing repo. `.archcore/` is empty. SessionStart loads zero documents. They open a file in `src/api/` — `check-code-alignment` greps an empty knowledge base and injects nothing. `/archcore:context src/api/` returns "no documents reference this path." `/archcore:audit` reports "0 documents."

The plugin gives no signal that anything has changed. The user does not know what to do next, sees no value, and uninstalls or forgets. **All the machinery that delivers JTBD #1 / #2 / #3 silently no-ops on a fresh repo until the user manually populates `.archcore/` — but nothing in the plugin guides them through that first step.**

This is a structural blind spot: every push and pull mechanism is content-conditional, and the plugin currently treats the empty state as a normal state rather than as a special case requiring its own product surface.

The proposal: add an **activation layer** that turns the empty state into a guided first step. Three variants exist along a cost/scope spectrum, listed cheap-to-ambitious:

### Variant A — Empty-state nudge in SessionStart (cheapest)

When `bin/session-start` (or its CLI counterpart) detects an empty or missing `.archcore/`, it emits a single advisory line into the SessionStart payload:

> `.archcore/ is empty. Try /archcore:capture to document an existing convention, or /archcore:decide to record a team decision. Use /archcore:help for the full surface.`

That's it. No new skill, no scanning, no UI — one branch in the existing hook. Pure copy. Cost: ~10 lines of shell. Idempotent — silent on populated repos.

This already removes the "I installed it and nothing happens" failure mode. It does not produce the first document, but it tells the user how to.

### Variant B — `/archcore:init` intent skill (middle ground)

A new intent skill that runs an explicit first-time onboarding flow:

1. **Detect state.** If `.archcore/` already has ≥3 documents, inform the user it's not their first session and route them to `/archcore:context` or `/archcore:audit`. Otherwise proceed.
2. **One question.** "What's the most important convention or decision in this repo that you'd want a future agent to know about?" — single open question, one round trip.
3. **Route to existing skill.** Based on the answer's shape:
    - "we always do X" → route to `/archcore:decide` (ADR → optional rule → optional guide cascade)
    - "we decided to use X over Y because Z" → route to `/archcore:decide`
    - "X works like this …" → route to `/archcore:capture` (auto-routes to spec / doc / guide)
    - Ambiguous → propose `/archcore:capture` as default
4. **Close the loop.** After the first document is created, the skill prints one paragraph: "you'll see this document automatically appear in `additionalContext` next time you edit a file under <path>. Check via /archcore:context <path> to confirm." This makes the value-loop visible immediately.

This is a thin orchestrator over already-shipped skills. Net new code: one SKILL.md with routing logic. Reuses `/archcore:decide`, `/archcore:capture` underneath.

### Variant C — `/archcore:init --scan` with repo introspection (ambitious)

Same skill as B, but adds an opt-in scan mode that reads the repo to suggest concrete document candidates:

- Read `README.md`, top-level directory names, `package.json` / `pyproject.toml` / `Cargo.toml` for stack signals.
- Propose 3–5 candidate documents with concrete titles: "ADR: [stack choice] for primary persistence based on package.json", "rule: API handlers live in src/api/", "doc: monorepo layout".
- User picks 0..N to seed; skill creates the chosen documents using the existing creation skills.

This is the "magic first day" path — but it's also the one that fails loudly when the scan suggests garbage. False-positive scans create the opposite problem: the user dismisses the plugin as "it just generates noise."

## Value

- **Closes the silent-noop failure mode.** Today's worst outcome on install: nothing happens, user uninstalls. Variant A alone removes this.
- **Bridges install → first useful state.** Plugin marketplace adoption is funnel-shaped: install is cheap, retention requires a "wow" moment within the first session.
- **Activates JTBD #1/#2/#3 mechanisms.** Push (`check-code-alignment`), pull (`/archcore:context`), and influence loops (`/archcore:decide` → injection) all need at least one document to demonstrate.
- **Aligns README promise with first-session reality.**
- **Distinguishes from competitors.** Memory tools (claude-mem, Mem0) accumulate state automatically. Spec Kit assumes the user is already building from a spec. Archcore's typed-document model puts the user in the position of having to author the first artifact — onboarding makes that authoring trivial.

## Possible Implementation

Recommended sequence: **A first, B next, C deferred.**

### Step 1 — Variant A (cheapest, immediate)

- Modify `bin/session-start` (and the CLI's session-start handler if it owns the empty-state check). Detect missing or empty `.archcore/` and append a single advisory paragraph.
- Add a `ARCHCORE_HIDE_EMPTY_NUDGE=1` env var for users with intentionally empty test repos.
- Test: structural assertion in the existing session-start bats suite.

### Step 2 — Variant B (`/archcore:init` skill)

- New `skills/init/SKILL.md` following the standard intent-skill structure.
- Triggers: explicit `/archcore:init` invocation. Mentioned in Variant A's nudge as the recommended first step.
- Routing logic: detect state, branch into "already populated" vs "first time", ask the single question, route into existing skills.
- Reuse `/archcore:capture`, `/archcore:decide` — init is glue, not new functionality.
- Test: structural test that the SKILL.md exists with the standard frontmatter.
- Update `commands-system.spec.md` and `skills-system.spec.md` to register the new intent skill.

### Step 3 — Variant C (deferred until B has user feedback)

- Decide go/no-go after observing 2–4 weeks of B in the wild.
- If B users consistently abandon at the "what's most important" question, scan-based suggestions (C) are likely the next step.

### Cross-cutting changes

- README — add a single paragraph after "Try these 3 prompts first": "Empty repo? Run `/archcore:init` first."
- `claude-plugin.prd.md` — add a goal/FR for "first-session activation on empty `.archcore/`".
- `development-roadmap.plan.md` — slot Variant A under near-term, Variant B in the next release window.

## Risks and Constraints

- **Nudge fatigue (Variant A).** Mitigation: env-var opt-out plus a one-time "got it, don't show again" flag stored in plugin state once any document is created.
- **Init quality bar (Variant B).** A bad first-document experience is worse than no first-document experience. Mitigation: keep the init question open enough to absorb unclear answers.
- **Scope creep into Variant C.** Resist — scan false-positives are the high-cost failure mode.
- **Conflicts with `readme-first-60-seconds`.** Both are needed; they should not be merged.
- **Empty-state detection edge cases.** Threshold: count documents with `status` in `{accepted, draft}` AND `body length ≥ 200 chars`.
- **Init skill discoverability paradox.** If users don't know `/archcore:init` exists, they won't run it. Variant A's nudge solves this.
- **CLI vs plugin ownership of the empty-state check.** SessionStart payload is composed by the CLI's MCP handler; the plugin's `bin/session-start` wraps it. For Variant A it can ship plugin-side first and migrate to CLI later.

## Related work in this repo

- `jtbd-alignment-analysis.idea` (accepted) — analyzes JTBD #1/#2/#3 mechanisms but assumes content already exists.
- `readme-first-60-seconds.idea` (draft) — adjacent funnel stage.
- `context-skill-implementation.plan` and `pre-code-hook-implementation.plan` — both shipped mechanisms that visibly no-op on empty `.archcore/`.
- `claude-plugin.prd.md` — Phase B adds FR-7 and FR-8.
- `skill-surface-collapse.adr` — establishes the 7-skill surface that `/archcore:init` is part of.
- `commands-system.spec` and `skills-system.spec` — register the new intent skill.
