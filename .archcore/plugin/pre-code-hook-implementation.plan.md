---
title: "Pre-Code Context Injection Hook Implementation Plan"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
---

**Status — realized, Phase 1 MVP.** Shipped in commit `87d384c` at plugin version 0.3.0. Delivered: `bin/check-code-alignment` as a POSIX-shell pre-mutation hook; the `archcore_hook_pretool_info` helper in `bin/lib/normalize-stdin.sh`; registration in the Claude Code and Cursor hooks configs; 13 bats cases in `@test/unit/check-code-alignment.bats` with the full suite green at 152 of 152; and updates to `hooks-validation-system.spec`, `component-registry.doc`, `multi-host-compatibility-layer.spec`, the source idea now marked accepted, and the README hero copy.

Deferred to later phases: a persistent path index in the sync manifest replacing grep with a constant-time lookup; session-level deduplication so the same document is not re-injected on consecutive edits; a CLI `archcore align <path>` subcommand as a programmatic pull wrapper; violation detection through an `archcore check` subcommand; telemetry and metrics; and greenfield proactive bootstrap.

## Goal

Ship the push-mode counterpart to `/archcore:context`: a pre-mutation hook that injects the applicable rules, ADRs, specs, and cpats as additional context when an agent is about to edit source code outside `.archcore/`. Together with the already-shipped pull skill this closes JTBD #1 into an engineered guarantee and retires the largest README-versus-reality gap identified in `jtbd-alignment-analysis.idea`.

## Scope of this iteration

The MVP is grep-based and shell-only, with no CLI subcommand and no persistent path index. It runs on every mutation outside `.archcore/`, emits the top 3 matches ranked by specificity and then type priority, and holds parity across Claude Code and Cursor.

Explicitly out of scope for the iteration: a CLI `archcore search` subcommand, since the MCP tool exists and a CLI wrapper is separate work; the persistent path index; session-level deduplication; telemetry and metrics; and greenfield proactive bootstrap.

## Architecture

Two hooks coexist on the same pre-mutation matcher. `bin/check-archcore-write` blocks a `.archcore/` markdown write and short-circuits on nothing else. `bin/check-code-alignment` short-circuits silently on everything inside `.archcore/` and does its work only on a source path. Their active path sets are disjoint by construction.

**The `check-code-alignment` algorithm** runs in eight steps. It bails silently when there is no file path, no `.archcore/` directory, a path inside `.archcore/`, or `ARCHCORE_DISABLE_INJECTION=1`. It normalizes an absolute path under the working directory to a relative one. It requires the path to start with a configured source root, defaulting to `src lib app pkg cmd internal apps packages modules components` and overridable through `.archcore/settings.json` at `codeAlignment.sourceRoots`. It generates directory-prefix tokens longest first, capped at 5, so `src/api/handlers/users.ts` yields three. It scans with `grep -rlF` per token, skipping a document already matched by a longer token, and classifies by filename extension so that only `.rule.md`, `.cpat.md`, `.adr.md`, `.spec.md`, and `.guide.md` are eligible. It ranks by specificity length times ten plus a type priority of `rule=5, cpat=4, adr=3, spec=2, guide=1`, and takes the top 3. It renders the `[Archcore Context]` block capped at 2 KB. And it emits through the host's envelope — the wrapped form on Claude Code and Copilot, and the flat form on Cursor, where the pre-mutation event may ignore it, which is a documented limitation for this iteration.

**Hook registration** adds `check-code-alignment` as the second entry in the existing pre-mutation array of `hooks/hooks.json`, after the block guard and with the same 1-second timeout, plus a parallel addition to the Cursor config under its `Write` matcher.

**The shared helper** `archcore_hook_pretool_info` mirrors `archcore_hook_info` but emits the pre-mutation event name. No other hook in this iteration uses it; it exists to keep the JSON-shape logic centralized.

## Acceptance Criteria

1. `bin/check-code-alignment` exists, is executable, and is POSIX shell only.
2. On a source-file mutation with matching documents, it emits valid JSON whose additional context carries the top 3 documents ranked by specificity and then type, and nothing else.
3. On any short-circuit condition it exits 0 with no output.
4. It never returns a non-zero exit, because injection is strictly additive and a failure must not block an edit.
5. Both the Claude Code and Cursor hooks configs list the new script.
6. The existing hook structure tests still pass their event-set invariants.
7. The new unit tests cover the source-root filter, the `.archcore/` skip, token specificity ranking, type priority ranking, top-3 truncation, the empty-`.archcore/` silent pass, the non-source-path silent pass, and the escape-hatch variable.
8. `hooks-validation-system.spec` documents the hook set, which was five at the time of this plan.
9. `component-registry.doc` lists the new script.
10. The plugin version is bumped from 0.2.3 to 0.3.0, as a new capability.
11. The README prompt section is adjusted so the hero claim no longer over-promises, because context injection now happens automatically.

## Manual smoke tests

Run these against this repository after merge. A write to a `SKILL.md` should not inject, because that file is inside the plugin but outside a source root. A write to a hypothetical `src/hooks/foo.sh` should inject the hook-related documents. A write to a `.archcore/` markdown file should be blocked by the guard first, with no injection output on the block path. And with `ARCHCORE_DISABLE_INJECTION=1` set, no path should inject.

## Dependencies

- The existing `bin/lib/normalize-stdin.sh`, which supplies the normalized file path, the host, and the output helpers.
- No new CLI version. The `search_documents` MCP tool shipped in CLI 0.1.7 is orthogonal, used by the pull skill rather than by this hook.

## Risks

- **Performance.** Grep per token across N documents scales as tokens times documents, so 40 documents and 5 tokens is about 200 grep invocations at roughly 2–5 ms each, or 400 ms to 1 s — near the 1-second timeout, acceptable for the MVP and requiring monitoring. The path index eliminates it.
- **False positives.** A document mentioning `src/` generically matches every source edit. Ranking by the longest prefix penalizes that without eliminating it, and user feedback will show whether the top-3 signal-to-noise is adequate.
- **Cursor pre-mutation context.** Whether Cursor respects the context field on that event is uncertain; if it does not, the output is ignored silently. Cursor parity survives through SessionStart and the post-MCP event, so this is not a regression.
- **Repetition fatigue.** With no session dedup, the same rule may be injected on 20 consecutive edits in one directory. Monitor user feedback before adding dedup, which brings state and its own failure modes.
