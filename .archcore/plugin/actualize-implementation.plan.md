---
title: "Actualize System Implementation Plan"
status: rejected
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
  - "skills"
---

**Outcome (2026-05-15).** The plan was executed. Layer 1 as `bin/check-staleness` and Layer 2 as `bin/check-cascade` shipped as designed. Layer 3, the deep analysis, shipped as the `--drift` mode of `/archcore:audit` rather than as a standalone intent skill, per `skill-surface-collapse.adr`, with its protocol at `skills/audit/lib/drift-detection.md`. Every acceptance criterion below is met under that naming. `audit` was later absorbed into `/archcore:review` by `four-command-palette.adr`; read `/archcore:review --drift` for the current invocation. The drift, staleness, and cascade detection system itself remains valid.

## Goal

Implement the three-layer freshness detection system specified by `actualize-system.adr` and `actualize-system.spec`, delivering two new bin scripts, an updated session-start script, updated hook configs, the deep-analysis mode inside the inspection skill, and an updated help skill.

## Tasks

### Phase 1 — Layer 1, passive detection at SessionStart

**1.1 Create `bin/check-staleness`**, a POSIX shell script of roughly 50 lines that detects code-document drift through git. It checks that the working directory is a git repository; finds the last `.archcore/` commit and exits 0 when there is none, because the documents were never committed; diffs the non-`.archcore/` paths from that commit to HEAD and exits 0 when nothing changed; counts the changed files; greps each document for a directory reference from the changed set; and emits a formatted warning capped at 2 KB, rate-limited to once per 24 hours.

**1.2 Extend `bin/session-start`** by roughly 5 lines, calling `bin/check-staleness` after the successful `archcore hooks <host> session-start` line, so the staleness output is appended to the session context.

### Phase 2 — Layer 2, reactive cascade detection at PostToolUse

**2.1 Create `bin/check-cascade`**, a POSIX shell script of roughly 60 lines. It reads JSON from stdin; extracts the updated document path and exits 0 when extraction fails; queries the relation graph in `.archcore/.sync-state.json` for relations whose target matches that path and whose type is `implements`, `depends_on`, or `extends`; exits 0 when none match; extracts the document title from the tool result or the path; and emits the host context envelope listing the affected documents.

**2.2 Update the hook configs** by adding one post-mutation entry per host, matching `mcp__archcore__update_document` and invoking `bin/check-cascade` under the plugin-root variable with a 3-second timeout, in each host's hooks config.

### Phase 3 — Layer 3, deep analysis as the drift mode

**3.1 Move the drift protocol into the inspection skill.** Per `skill-surface-collapse.adr`, Layer 3 ships as `--drift`. The phase was originally scoped to a standalone skill; the same content — the routing table, the three-dimension analysis, and the assisted-fix flow — now lives at `skills/audit/lib/drift-detection.md` and loads on the flag.

The `audit` frontmatter carries `name: audit`, an argument hint of `[--deep] [--drift] [category, tag, or scope]`, and a description naming the dashboard, the deep coverage audit, and drift detection. The skill body carries the title and one-liner, a When to Use section with explicit anti-triggers for `capture` and `decide`, a routing table across the three modes, the per-mode execution, and the result. The drift steps load from the lib file and run in order: gather through `list_documents`, `list_relations`, and `git log`; apply the scope filter from the argument; analyze code-to-doc drift; analyze doc-to-doc cascade; analyze temporal staleness; report grouped by severity as critical, cascade, and temporal; and offer an assisted fix through `update_document`, one finding at a time.

### Phase 4 — integration updates

**4.1** Update the help skill to document the three modes of `audit`.

**4.2** Add a sixth audit dimension, code-document correlation, to the auditor agent, so the background auditor checks whether a document references a code path that has changed.

### Phase 5 — validation

**5.1 Structural.** Verify that both new bin scripts are executable, that every host hook config carries the cascade matcher, that the drift lib file exists and is loaded by the audit skill, and that the skill count at plan completion is 7.

**5.2 Content.** Verify that `bin/check-staleness` exits 0 in every case, keeps output under 2 KB, and works without git; that `bin/check-cascade` exits 0 in every case, reads the sync state correctly, and emits valid JSON; that the audit skill carries all five sections and routes its three modes deterministically; and that the help skill lists all 7 commands.

**5.3 Integration.** Verify that `bin/session-start` calls the staleness check after context loading, that each host's cascade matcher fires only on `update_document`, and that no existing hook behavior breaks.

## Acceptance Criteria

- [x] `bin/check-staleness` produces a code-drift warning when `.archcore/` is behind the code changes.
- [x] `bin/check-staleness` exits cleanly with no output when there is no drift or git is unavailable.
- [x] `bin/session-start` includes the staleness output in the session context.
- [x] `bin/check-cascade` produces a cascade warning after `update_document` when a dependent exists.
- [x] `bin/check-cascade` exits cleanly with no output when there is no cascade.
- [x] Every host hook config registers the cascade script on `update_document`.
- [x] `/archcore:review --drift` exists with routing, the three-dimension analysis, and the assisted fix.
- [x] The help skill listed all 7 primary commands, including `audit`, before `help` was removed under v2; command descriptions and CLI help now absorb that role per `four-command-palette.adr`.
- [x] The auditor agent includes the code-document correlation dimension.
- [x] Every bin script is POSIX shell compatible and exits 0.
- [x] Every bin script degrades gracefully when git or the CLI is unavailable.
- [x] The skill directory count at plan completion is 7.

## Dependencies

- `actualize-system.adr` for the architectural decision and `actualize-system.spec` for the detailed contract.
- `hooks-validation-system.spec` for the extended hook contract.
- `plugin-architecture.spec`, `skills-system.spec`, and `commands-system.spec` for the intent skills, the hooks, and the command surface.
- `skill-surface-collapse.adr`, the decision that folded Layer 3 into the audit skill.
