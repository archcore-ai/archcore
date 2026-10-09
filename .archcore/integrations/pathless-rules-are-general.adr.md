---
title: "A Rule or Code Pattern That Names No Code Path Applies to Every File"
status: draft
tags:
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
---

## Context

The pre-edit hint in @cli/internal/advisory/code_alignment.go matches a document only when its body contains a directory of the edited file, so a rule that names no path never reaches an edit. In the Litres monorepo 15 of 54 rules name no `apps/` or `packages/` path (count of 2026-10-09), among them `guard-clauses-first.rule.md`, the rule the author of MR !7667 missed. The CLI's own `CLAUDE.md` works around the same gap by telling agents to load the `code-quality` tag by hand before Go work.

## Decision

Adopted the rule that an accepted `rule` or `cpat` whose body carries no code reference recognised by the `path_ref` matcher is general and enters every file-context result as a title row, under a row cap with the remainder stated, with no new frontmatter field (user choice, 2026-10-09).

## Alternatives Considered

1. An `applies_to` frontmatter field with glob masks such as `**/*.test.*` — deferred because a rule reaches an edit only after someone tags it, and the CLI must first validate the field; it stays the refinement path for rules that apply to one kind of file.
2. `applies_to` and general rules in one change — deferred because it doubles the scope of this initiative without changing what reaches the agent first.
3. Keep directory matching only and document a per-project tag load, as `CLAUDE.md` does for `code-quality` — rejected because every project would need a hand-written instruction, which is the gap the bench measures.

## Consequences

Positive:

- The 15 pathless rules of the monorepo reach every file-context result without editing a single document.
- A project gets the behavior with no migration, because the test reads the body the corpus already has.

Negative:

- A general rule unrelated to the file still takes a row, for example `oauth-security.rule.md` beside `Button.tsx`.
- [expected] A title row costs about 100 characters, so the cap on general rows has to hold inside the 2 048-rune hook budget beside the matched rows.
- A rule that mentions one example path stops being general, even when its obligation covers every file.

## Superseded when

- A measured corpus holds more general rules than the general-row cap shows, so the remainder line replaces most of them.
- `applies_to` lands and at least half of the rules in a measured corpus carry it.
