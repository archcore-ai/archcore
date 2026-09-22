---
title: "Actor-Subject Vocabulary Compatibility — Engine Gate on CLI 0.8.4 and Command Paths to scenario and journey"
status: draft
tags:
  - "component:plugin"
  - "document-types"
  - "multi-host"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the engine gate for the two actor-subject type names and the command paths that reach them: the compatibility file on the `research-compatibility.md` pattern, its probe trigger and fallback, the `/archcore:document code` path to `scenario`, and the grounding filters of the three command skills. Normative for the `plan`, `document`, and `review` skills, the `archcore-assistant` agent on every host, and host adapters that carry the argument hint. External contract: the Archcore CLI release that ships the vocabulary. Out of scope: the content of either type and the gates that produce them.

## Surface

- Compatibility file: `@plugin/plugins/archcore/skills/_shared/actor-subject-compatibility.md`, on the structure of `@plugin/plugins/archcore/skills/_shared/research-compatibility.md`: version probe, result table, fallback, shared repositories.
- Probe helper: `@plugin/plugins/archcore/bin/cli-gte`, invoked as `cli-gte 0.8.4`; results `yes`, `no`, `__NO_CLI__`. "The probe fails" below means a result other than `yes`.
- Minimum engine: CLI 0.8.4 — tag `v0.8.4` on commit `2a8f6e4` in the `cli` repository, the release that registers 23 types.
- "Either type" below means `scenario` or `journey`; "a request names a type" covers a type name inside the subject text.
- Argument hint of `@plugin/plugins/archcore/skills/document/SKILL.md`: `[decision|code|research] [subject]`; neither type is a mode on any command.
- Grounding filters: the planning-moment filter in `@plugin/plugins/archcore/skills/plan/SKILL.md`, the document-moment filter in the document skill, and the review-moment filter in `@plugin/plugins/archcore/skills/review/SKILL.md`.
- Agent instruction files under `@plugin/plugins/archcore/agents/` and `@plugin/plugins/archcore/copilot-agents/`.

## Normative Behavior

1. The compatibility file MUST set the minimum engine version to `0.8.4`.
2. WHEN a request or a grounding result names either type, the executing skill MUST run the probe before the first MCP call naming it.
3. WHEN the route engages the illustrate instrument, the executing skill MUST run the probe before the first MCP call naming either type.
4. WHEN none of the conditions of behaviors 2 and 3 holds, the executing skill MUST skip the probe.
5. WHEN the probe returns `yes`, the executing skill MUST add both names to its type filters.
6. WHEN the probe fails, the executing skill MUST keep the legacy type filters.
7. IF the probe fails and the subject text names `scenario`, THEN the document skill MUST report the required version and exit without a write.
8. IF the probe fails and the route engages the illustrate instrument, THEN the conductor MUST drop the instrument from the package.
9. WHEN the conductor drops the instrument under behavior 8, the plan skill MUST report the required version once.
10. The executing skill MUST NOT convert or rewrite an existing artifact of either type on an older engine.
11. WHEN the subject text of `document code` names `scenario`, `describe.draft` MUST treat the type as settled.
12. The document skill MUST NOT produce a `journey`.
13. The document argument hint MUST NOT list either type.
14. The plan skill MUST NOT accept either type as an entry.
15. WHEN the executor has no shell tool and no supplied probe result, the executing skill MUST return `needs-vocabulary-probe` to the caller.
16. BEFORE delegating work on either type, the calling skill MUST supply the current probe result and the absolute plugin root.
17. WHEN the server rejects either type after a `yes` probe, the executing skill MUST report the mismatch and stop the operation.
18. The agent instructions on every host MUST name both types and the compatibility file.
19. The plugin's pinned integration CLI MUST be 0.8.4 or later.

## Constraints & Invariants

- Constraint: the probe compares versions in the helper, never in prose; an unparsable banner yields `__NO_CLI__`.
- Constraint: two compatibility files coexist — research at 0.8.3, actor-subject at 0.8.4 — per the separate-file decision recorded on 2026-09-16; a merged vocabulary file is deferred to a third vocabulary release.
- Constraint: the command entry grammar of 2026-09-16 retired the named entries `document scenario` and `document journey`.
- Constraint: the local probe cannot verify a teammate's engine; the shared-repositories section of the compatibility file states this, as the research file does.
- Invariant: relation values are unchanged by this vocabulary, so the shared-manifest hazard of the research release does not recur.
- Invariant: the argument hint of the command and the argument hint of its skill stay identical on every host.
- Invariant: the PATH probe identifies the binary a new server would run; a server started before an upgrade keeps the old engine until restarted.

## Failure Behavior

1. IF `archcore` is absent from PATH, THEN the helper MUST return `__NO_CLI__`.
2. IF the helper returns `__NO_CLI__`, THEN the executing skill MUST keep the legacy vocabulary.
3. IF the probe fallback runs, THEN the executing skill MUST report once: "Scenario and journey require Archcore CLI 0.8.4; skipping actor-subject documents."
4. IF the MCP server is unavailable, THEN the executing skill MUST follow the calling skill's existing recovery path.
5. IF an older engine scans a corpus holding either type, THEN that engine skips those files and reports an invalid type in `status`.

## Conformance

An implementation is conformant when the compatibility file exists with the surfaces above, behaviors 1–19 hold, and the failure rules produce the stated outcomes. Regression coverage: `@plugin/test/unit/cli-gte.bats` (helper contract), `@plugin/test/structure/cli-compat-invariant.bats` (file references resolve), `@plugin/test/structure/actor-subject-compat.bats` (hint parity, agent references), and `@plugin/test/integration/actor-subject-vocabulary.bats` against a real CLI 0.8.4 stdio MCP: 23 types in the `create_document` enum, `scenario` in knowledge, `journey` in vision, template sections as the CLI contract lists them.

Given CLI 0.8.3 on PATH, When the user invokes `document code scenario for checkout`, Then the skill writes nothing and reports the required version once.
