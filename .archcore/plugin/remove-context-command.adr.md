---
title: "Remove /archcore:context — CLI Hooks and Command Grounding Absorb the Pull Moment"
status: accepted
tags:
  - "commands"
  - "hooks"
  - "plugin"
---

## Context

`/archcore:context` is a retrieval pipeline, not a conversation: its four modes (path, topic, git-changes, pickup) map one-to-one onto retrieval mechanisms that already exist or ship in CLI release A, and the maintainer observes the command is rarely invoked (2026-08-04). Meanwhile the SessionStart hook injects a full-corpus listing that grows linearly — a 300-document repo injects roughly 300 lines every session (@cli `cmd/hooks_common.go` buildSessionContext) — so the passive channel needs the ranking work regardless of the skill's fate.

## Decision

Remove `/archcore:context` one release after CLI release A ships its replacements — ranked SessionStart recap (pickup mode), Go `pre-tool-use` injection (path mode), the existing `search_documents` MCP tool plus the plan/review grounding phases (topic mode), and direct git commands in review grounding (git-changes mode) — keeping the skill alive during the overlap release as a strangler.

## Alternatives Considered

1. Keep `context` as an expert command beside the hooks — rejected because two owners of one retrieval pipeline drift; the recorded rejection of a second pull skill already established one index, one owner.
2. Delete `context` in the same release as the palette swap — ruled out because think-time pull would be orphaned before the hook replacements prove out; injection hit-rate on Claude Code and Cursor is measured first.
3. Improve the skill instead of the hooks — rejected because a skill covers only hosts with skill support, while CLI hooks reach all 8 registry hosts.
4. Ship dedicated CLI retrieval commands (`archcore context <query>`, `archcore scope`) as the replacements — rejected because `search_documents` already serves topic pull and the branch boundary is plain git the agent runs directly; a second public surface for either duplicates an existing mechanism (maintainer decision, 2026-08-04).

## Consequences

- All 8 registry hosts receive passive context via CLI hooks; today 4 hosts receive SessionStart only.
- Topic pull stays on the existing `search_documents` MCP tool; the CLI public command surface does not grow.
- Prompt-only hosts retain only the instructions-block nudge for think-time pull — declared the accepted floor.
- The command spec's context section, the context filtering-pipeline doc, and the context implementation plan require rewrite or archival in the palette-swap release.

## Superseded when

- Injection hit-rate on Claude Code and Cursor after one release falls below [METRIC REQUIRED].
- A registry host ships a think-time hook event that the CLI handler cannot serve.