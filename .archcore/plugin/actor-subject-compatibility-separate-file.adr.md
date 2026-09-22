---
title: "Separate actor-subject-compatibility.md at CLI 0.8.4 Beside research-compatibility.md"
status: draft
tags:
  - "component:plugin"
  - "document-types"
  - "multi-host"
  - "plugin"
  - "skills"
---

## Context

The research vocabulary release introduced @plugin/plugins/archcore/skills/_shared/research-compatibility.md: a probe through `bin/cli-gte 0.8.3`, a result table, a fallback, and a shared-repositories note. Nine sources reference that file — three skills, two track files, the conductor contract, and the three agent instruction files — and `@plugin/test/structure/cli-compat-invariant.bats` pins its references. Two accepted specs cite it by path: @.archcore/plugin/agent-system.spec.md and @.archcore/plugin/track-layer.spec.md. The actor-subject vocabulary (`scenario`, `journey`) ships in CLI v0.8.4 (commit `2a8f6e4` in the `cli` repository) and needs its own minimum. The compatibility spec carried the file layout as `[assumption]` until the brainstorming session of 2026-09-16.

## Decision

Ship a second file, `skills/_shared/actor-subject-compatibility.md`, on the structure of the research file, with minimum `0.8.4` and its own trigger: a request or grounding result naming `scenario` or `journey`, or a route engaging the illustrate instrument. The research file is unchanged. Each file owns one vocabulary and one minimum; a skill loads the file whose trigger fires, and both when both fire.

## Alternatives Considered

- **One merged `vocabulary-compatibility.md` with a table of feature → minimum version.** Rejected for this delta: it renames a file that nine sources, one structure test, and two accepted specs reference, and widens the release beyond the vocabulary it delivers. It stays the candidate refactor once a third vocabulary release appears.
- **A second section inside `research-compatibility.md`.** Rejected: the file name would stop matching its content, and the research contract's minimum and fallback wording would have to be re-scoped per section.

## Consequences

### Enabled

- [expected] The research contract, its tests, and its references stay byte-identical.
- [expected] The new file and its structure test follow one precedent, so review compares two files of one shape.
- [expected] Each probe names one minimum; no skill compares versions in prose.

### Costs and limits

- [expected] A request that names both vocabularies runs `cli-gte` twice in one invocation; the helper is a shell script with no network call, so the cost is one process per probe.
- [expected] Two files state the same shared-repositories caveat; a change to that caveat edits both.
- The merge into one file is deferred, not rejected; a third vocabulary release is the trigger to revisit.

## Superseded when

- A third engine-gated vocabulary ships and the three files are merged into one table-driven contract.
- The engine exposes its registered types through a tool, making the version probe unnecessary.
