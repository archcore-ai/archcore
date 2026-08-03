---
title: "Zero-Content Onboarding — From Empty .archcore to First Useful State"
status: accepted
tags:
  - "onboarding"
  - "plugin"
  - "vision"
---

**Outcome (2026-05-15).** The idea was executed. The skill shipped as `skills/init/` and the command is `/archcore:init`, per `skill-surface-collapse.adr`; it was originally drafted as `skills/bootstrap/`. Variants A and B shipped, and Variant C remains deferred. Read every `/archcore:bootstrap` below as `/archcore:init`.

## Idea

A new user installs the plugin on an existing repository. `.archcore/` is empty, SessionStart loads zero documents, `check-code-alignment` greps an empty knowledge base and injects nothing, `/archcore:context src/api/` reports that no document references the path, and `/archcore:audit` reports zero documents. The plugin gives no signal that anything changed, so the user does not know what to do next, sees no value, and uninstalls or forgets.

Every push and pull mechanism is content-conditional, so all the machinery that delivers the three jobs-to-be-done silently no-ops on a fresh repository, and nothing guides the user through the first step. The plugin treats the empty state as a normal state rather than as a special case needing its own product surface.

The proposal adds an **activation layer** that turns the empty state into a guided first step, in three variants along a cost spectrum.

**Variant A — an empty-state nudge in SessionStart, the cheapest.** When `bin/session-start`, or its CLI counterpart, detects a missing or empty `.archcore/`, it emits one advisory line into the SessionStart payload naming `/archcore:capture`, `/archcore:decide`, and `/archcore:help`. No new skill, no scanning, no UI — one branch in an existing hook, about 10 lines of shell, idempotent and silent on a populated repository. It does not produce the first document, but it tells the user how to.

**Variant B — an `/archcore:init` intent skill, the middle ground.** It detects state, routing a repository that already holds three or more documents to `/archcore:context` or `/archcore:audit` and otherwise proceeding; asks one open question — what is the most important convention or decision in this repository that a future agent should know; routes by the shape of the answer, sending "we always do X" and "we decided X over Y because Z" to `/archcore:decide`, "X works like this" to `/archcore:capture`, and anything ambiguous to `/archcore:capture` as the default; and then closes the loop with one paragraph explaining that the document will appear automatically in the context of the next edit under that path, which makes the value loop visible immediately. It is a thin orchestrator over already-shipped skills.

**Variant C — `/archcore:init --scan` with repository introspection, the ambitious one.** The same skill plus an opt-in scan that reads `README.md`, the top-level directory names, and the manifest for stack signals, proposes 3–5 candidate documents with concrete titles, and seeds whichever the user picks. This is the magic-first-day path, and also the one that fails loudly when the scan suggests garbage, because a false-positive scan creates the opposite problem — the user dismisses the plugin as noise.

## Value

- **It closes the silent no-op failure mode.** The worst outcome on install today is that nothing happens and the user uninstalls; Variant A alone removes it.
- **It bridges install to first useful state.** Marketplace adoption is funnel-shaped: install is cheap, and retention needs a visible payoff inside the first session.
- **It activates the three JTBD mechanisms.** Push through `check-code-alignment`, pull through `/archcore:context`, and the influence loop from `/archcore:decide` all need at least one document to demonstrate anything.
- **It aligns the README promise with first-session reality.**
- **It differentiates from the alternatives.** Memory tools accumulate state automatically and Spec Kit assumes the user already builds from a spec, whereas Archcore's typed-document model puts the user in the position of authoring the first artifact — onboarding makes that authoring trivial.

## Possible Implementation

The recommended sequence is A first, B next, C deferred.

**Step 1, Variant A.** Modify `bin/session-start`, and the CLI's session-start handler if it owns the empty-state check, to detect a missing or empty `.archcore/` and append one advisory paragraph. Add `ARCHCORE_HIDE_EMPTY_NUDGE=1` for users with intentionally empty test repositories. Cover it with a structural assertion in the existing session-start bats suite.

**Step 2, Variant B.** Add `skills/init/SKILL.md` in the standard intent-skill structure, triggered by explicit invocation and named in Variant A's nudge as the recommended first step. Its routing detects state, branches between already-populated and first-time, asks the single question, and routes into the existing skills, so init is glue rather than new functionality. Cover it with a structural test asserting the file and its frontmatter, and register the skill in the commands and skills specifications.

**Step 3, Variant C.** Decide go or no-go after observing 2–4 weeks of Variant B in use. If users consistently abandon at the open question, scan-based suggestions are the likely next step.

Cross-cutting: add one README paragraph telling a user with an empty repository to run `/archcore:init` first; add a goal and functional requirement for first-session activation to the product requirements; and slot Variant A into the near-term roadmap with Variant B in the following release window.

## Risks

- **Nudge fatigue in Variant A.** Mitigated by the environment-variable opt-out plus a one-time acknowledgement flag stored in plugin state once any document exists.
- **The init quality bar in Variant B.** A bad first-document experience is worse than none, so the init question stays open enough to absorb an unclear answer.
- **Scope creep into Variant C.** Resisted, because a scan false positive is the high-cost failure mode.
- **Empty-state detection edge cases.** The threshold counts documents whose `status` is `accepted` or `draft` and whose body is at least 200 characters.
- **The init discoverability paradox.** A user who does not know `/archcore:init` exists will not run it, which is exactly what Variant A's nudge solves.
- **CLI versus plugin ownership of the empty-state check.** The SessionStart payload is composed by the CLI's MCP handler and wrapped by the plugin's `bin/session-start`, so Variant A can ship plugin-side first and migrate to the CLI later.
