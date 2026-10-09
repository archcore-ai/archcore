---
title: "The Managed Block Carries a Format Version, and Archcore Reports a Stale Block Instead of Rewriting It"
status: draft
tags:
  - "component:cli"
  - "integrations"
  - "multi-host"
---

## Context

The managed block in `CLAUDE.md`, `AGENTS.md`, and `GEMINI.md` changes only when someone runs `archcore instructions install`, `archcore doctor --fix`, or `archcore init` (@cli/cmd/instructions.go, @cli/cmd/doctor.go). Nothing compares a committed block with the text the installed CLI writes, so a project keeps the old block after every CLI or plugin update. The block is committed to git, so one teammate with a newer CLI can commit a block that names `search_documents(for_path=…)`, which an older CLI rejects with `specify at least one filter`. A block written at the end of a large `AGENTS.md` stays there, past the 32 KiB that Codex reads (`project_doc_max_bytes`).

## Decision

Adopted an integer block-format version in the block's header line (`managed by archcore init, block v2 — edit outside these markers`, with a header without a version read as v1) and a read-only check in the session-start recap and in `archcore doctor` that compares each block with the one the installed CLI writes and names the command that fixes the difference — refresh the block, update the CLI, or move the block to the top — at most once per project in 24 hours on session start (user choice, 2026-10-09).

## Alternatives Considered

1. Rewrite a stale block automatically on session start or on `archcore mcp` start — rejected because it puts unrequested diffs into user-curated, committed files, which `instruction-nudge-on-init.adr` allows only with consent.
2. Detect a stale block by comparing text only, without a version — rejected because a mismatch cannot say whether the block or the CLI is the older side, and the fix differs.
3. Stamp the CLI release version into the header — rejected because every release would mark every block stale even when the block text did not change.
4. Report through `archcore doctor` only — rejected because nobody runs `doctor` unprompted; the session start reaches every session.

## Consequences

Positive:

- [expected] A project learns that its block is stale in the first session after a teammate installs a newer CLI.
- A teammate whose CLI is older than the committed block gets "update the CLI", not a failed `for_path` call.
- A block past 32 KiB in `AGENTS.md` is reported with the fix, which the placement decision alone could not reach.

Negative:

- Every change to the block body needs a version bump; a test pins the body to its version, so a forgotten bump fails the build.
- The header line of every block changes once, so every project shows one block diff at its next refresh.
- [expected] The advisory adds one line to the session-start recap, at most once a day per project.

## Superseded when

- A host loads instruction files from a source Archcore can version without editing the file (for example an `@import` of a file Archcore owns).
- The block body changes more than twice a quarter, so version bumps become routine noise.
