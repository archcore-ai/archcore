---
title: "UserPromptSubmit Topic Injection via the Internal Hook Runner"
status: draft
tags:
  - "hooks"
  - "plugin"
---

## Idea

On hosts that support a UserPromptSubmit hook event, inject documents matching the topic of the user's prompt — the same ranked retrieval the code-alignment hook applies to file paths, applied to prompt text. Deferred from the CLI v2 boundary work.

## Value

Closes the one pull moment the current hooks miss: context arrives when the agent edits a file (PreToolUse) and when the session starts (SessionStart), but not when the user states intent in prose. Prompt-time injection would front-load the relevant decisions before the agent plans, not after it starts editing.

## Possible Implementation

A hidden `user-prompt-submit` leaf beside the existing CLI hook events, reusing the search ranking; the plugin launchers gain one sibling. Backend is the internal hook runner — not a public command, per the no-new-commands constraint.

## Risks

Prompt text is a noisier retrieval key than a file path — precision-over-coverage demands a high relevance bar or the injection becomes noise on every message. Host support for the event is uneven; ship per host only after a live probe.