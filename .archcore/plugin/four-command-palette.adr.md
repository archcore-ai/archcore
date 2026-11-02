---
title: "Collapse Skill Surface to Four Commands with a Gated Track Layer"
status: accepted
tags:
  - "architecture"
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

The plugin ships 7 user skills (@plugin/plugins/archcore/skills/), four of which — `capture`, `decide`, `context`, `audit` — are system verbs that the target developer audience does not map to their jobs, while job verbs (`plan`, `review`) are self-explanatory; the maintainer reported user confusion with the current set ("не всегда понятно что вообще делать", 2026-08). A comparison of 23 AI-SDLC tools (session research, 2026-08-04) showed the surfaces that win adoption expose job-moment verbs, and the repo's own collapse history (27 → 18 → 7 skills) was driven by routing overlap between narrow entries.

## Decision

Collapse the visible palette to four auto-invocable commands — `init`, `plan`, `document`, `review` — with a non-palette track layer (six gated flows under `skills/_shared/tracks/`) selected by routing signals, absorbing `capture` and `decide` into `document`, `audit` into `review`, and removing `context` and `help`.

**Why `context` is removed.** `/archcore:context` is a retrieval pipeline, not a conversation, and it is rarely invoked (maintainer, 2026-08-04). Its modes map onto mechanisms the CLI already provides: a ranked SessionStart recap, `pre-tool-use` path injection, `search_documents` for topic pull, and plain git for the branch boundary. CLI hooks reach all 8 registry hosts, while a skill reaches only hosts with skill support. Keeping `context` beside the hooks gives one retrieval pipeline two owners that drift. Dedicated CLI retrieval commands (`archcore context <query>`, `archcore scope`) were rejected because they duplicate `search_documents` and git. Prompt-only hosts keep only the instruction-block nudge, declared the accepted floor.

**Why no invocation flags.** Neither `disable-model-invocation` nor `user-invocable` is in the agentskills.io standard. Cursor hid a plugin skill carrying `disable-model-invocation: true` from `/` entirely (acknowledged 2026-03-24, closed without a fix 2026-04-17) and does not document `user-invocable`. Codex supports neither field in `SKILL.md` and keeps invocation policy in `agents/openai.yaml`. A tiered invocation policy therefore works only on Claude Code. Every palette skill is auto-invocable with no invocation flag, which keeps the four commands identical on all hosts.

## Alternatives Considered

1. Keep the 7-command surface and add track flags — rejected because the four system-verb names remain the discovery bottleneck regardless of flags; the naming, not the count, blocks adoption. [assumption]
2. Standalone track skills (one palette entry per flow) — ruled out because the repo already ran this shape twice and collapsed it both times for measured routing overlap (27 → 7 history).
3. Five commands retaining `help` — rejected because a static catalog skill duplicates content already carried by command descriptions, the init closing summary, and CLI `archcore help`, and every palette entry costs trigger-routing precision.
4. Track selection by user menu — ruled out because a visible track menu reintroduces the taxonomy into the user surface; routing signals (explicit expert invocation, graph state, branch state, wording) select tracks deterministically.

## Consequences

- Reduces the visible palette from 7 to 4 commands; every remaining trigger is a job verb.
- New flows land as one track file plus one routing-table row, with zero palette growth.
- [expected] Auto-invocation routing precision holds or improves versus the 7-skill baseline; the trigger-phrase regression suite measures this before release.
- `document` reverses the recorded `capture`-over-`document` naming choice; the regression suite MUST pass on `document` triggers before the swap ships.
- One release touches 4 host manifests, hooks configs, and mirrored agent files; per-host probe records gate the release.

## Superseded when

- The trigger regression suite shows `document` auto-invocation precision below the current `capture` baseline.
- Tracks leak back into the palette (any track becomes user-invocable as a separate command).