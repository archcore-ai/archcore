---
title: "Archcore Plugin v2 — Four Commands over Gated Tracks"
status: accepted
tags:
  - "commands"
  - "plugin"
  - "vision"
---

## Vision

One tool, four commands, at the moments a developer already inhabits: connect once (`init`), then `plan`, `document`, `review`. Knowledge lands in `.archcore/` as typed, linked, statused markdown and returns to the agent at the moment it edits code — without being asked. Types, categories, tracks, and gates stay under the hood; the user learns the layer model through command results, never through menus.

## Problem

1. The 7-command surface exposes system verbs (`capture`, `decide`, `context`, `audit`) that target users do not map to their jobs; the maintainer reports users not knowing what to invoke.
2. The one-question cap makes requirement-stage documents non-credible — their content lives in the user's head and no command may interview for it.
3. Accumulated context is under-used: pre-edit injection covers 5 of 18 types, and session context is a full-corpus listing that grows linearly with base size.
4. Market SDD artifacts die after merge; no surveyed tool continuously validates team knowledge against code (see the flow-patterns research).

## Goals & Metrics

- A fully-specified request produces its documents with zero questions; a vague request stays within 5 (measured by the trigger and golden-transcript suites).
- An interrupted multi-stage flow resumes in a new session without repeating one answered question.
- Every one of the 18 document types is producible through a command path, and each ranks first in at least one injection moment. [expected]
- On a fresh repo, `init` to the first accepted document fits in one session.

## Requirements

1. The palette is `init`, `plan`, `document`, `review`; no other visible entries.
2. `plan` grounds in the repo and the base before its first question, produces vision documents, and ends with tasks mapped to files plus an implement-or-handoff fork.
3. `document` is the single entry for decisions and code documentation; it classifies the target without exposing a type menu and offers a rule/spec continuation after a decision.
4. `review` without arguments reviews the branch since the default branch; each finding carries a verdict — code wrong or document stale; fixes apply one document at a time with confirmation; after work it offers pattern capture.
5. Expert forms skip routing: `plan iso`, `document decision`, `review --deep`.
6. The background channel works without commands: session recap, pre-edit injection, direct-write guard, staleness advisories.
7. On an empty base every command works from code and questions alone, and each run leaves the base richer.
8. Every interview question carries a recommended answer; "you decide" records an `[assumption]`; auto mode never exceeds 5 questions per invocation.