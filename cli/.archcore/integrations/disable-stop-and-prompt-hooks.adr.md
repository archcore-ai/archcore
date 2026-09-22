---
title: "Remove Stop and UserPromptSubmit Hooks Due to False Positives"
status: accepted
tags:
  - "integrations"
---

## Context

The hooks system had three event types: SessionStart (inject context), Stop (scan assistant messages for keywords), and UserPromptSubmit (scan user prompts for keywords).

Stop and UserPromptSubmit relied on keyword matching against natural language. Common phrases like "step 1:", "always", "root cause", "debug" triggered false positives in normal conversation, causing agents to be blocked or receive unnecessary instructions.

## Decision

Remove Stop and UserPromptSubmit hooks entirely. Keep only SessionStart.

Removed across all agents (Claude Code, Cursor, Gemini CLI):

- Stop/UserPromptSubmit hook commands and event registrations
- `stopKeywords`, `promptKeywords` tables and all matching functions
- `handleStop()`, `handleUserPromptSubmit()` handlers
- `newStopHookCmd()`, `newPromptHookCmd()` factories
- Instruction constants (`adrInstruction`, `planInstruction`, `cpatInstruction`)
- Unused `hookInput` fields (`StopHookActive`, `LastAssistantMessage`, `Prompt`) and `hookOutput` fields (`Decision`, `Reason`)

## Alternatives Considered

1. **Tune keyword lists** — Natural language matching is inherently noisy; more specific phrases miss legitimate cases.
2. **LLM-based classification** — Too slow for hook execution, adds latency to every interaction.

## Consequences

- SessionStart already lists all document types and MCP tools — agents have enough context to suggest documentation without proactive nudges.
- Users with existing hooks configs will have stale Stop/UserPromptSubmit entries referencing removed commands. Re-running `hooks install` won't add them back but won't clean them up either.
