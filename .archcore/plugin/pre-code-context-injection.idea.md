---
title: "Pre-Code Context Injection — PreToolUse Hook for Source-File Edits"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "validation"
---

## Idea

Add a `PreToolUse Write|Edit` hook entry that fires on a source-file path outside `.archcore/` and injects a compact list of relevant documents — ADRs, rules, specs, and cpats — into the agent's context before the write executes, as `additionalContext` carrying one-line excerpts rather than full document bodies.

This closes the largest gap named in `jtbd-alignment-analysis.idea`: the absence of any mechanism that activates when the agent is about to modify code rather than documentation. Without the hook, "Archcore makes the agent code with your project's architecture, rules, and decisions" is an aspirational claim; with it, the claim becomes an engineered guarantee the plugin can demonstrate on first install.

## Status — realized, Phase 1

Shipped in plugin 0.3.0 as `@plugins/archcore/bin/check-code-alignment`, registered as a second entry on the pre-mutation matcher in the Claude Code and Cursor hooks configs, coexisting with the blocking guard. `pre-code-hook-implementation.plan` holds the execution plan and acceptance criteria, and `hooks-validation-system.spec` holds the formal contract as Hook 3.

Phase 1 is grep-based with no path index: it scans `.archcore/**/*.md` with one `grep -rlF` per token on each edit, ranks by specificity — the longest matching directory prefix — and then by type priority `rule > cpat > adr > spec > guide`, restricting eligibility to those five types because `prd`, `idea`, `plan`, and `rfc` are not enforceable at a line of code. Output caps at the top 3 matches and 2 KB. The settings shape is `codeAlignment.sourceRoots` inside `.archcore/settings.json`, defaulting to `src lib app pkg cmd internal apps packages modules components` when absent, with `ARCHCORE_DISABLE_INJECTION=1` as the escape hatch. Thirteen cases in `@test/unit/check-code-alignment.bats` cover the silent-pass paths, injection correctness, specificity ranking, top-3 truncation, the type allowlist, the settings override, the Cursor JSON shape, and non-blocking safety.

**The concrete flow.** An agent calls Write or Edit on `src/api/handlers/users.ts`. The blocking guard allows it, because the path is outside `.archcore/`. The injection guard then builds the token set — `src/api/handlers/`, `src/api/`, `src/`, longest first and capped at 5 — scans for documents referencing those tokens, and injects a block headed `[Archcore Context] Before editing src/api/handlers/users.ts:` listing up to three matches as `<type>: <title> [<short-doc-path>]`. The write proceeds with the constraints already in context.

The two entries act on disjoint path sets by construction: the blocking guard handles `.archcore/` markdown and passes everything else through, while the injection guard short-circuits silently inside `.archcore/` and does real work only on a source path.

**The path index is deferred to Phase 2.** The originally proposed pre-built index in the sync manifest remains future work; the current scan is acceptable for a corpus of 50 documents or fewer inside the 1-second hook timeout, and a larger corpus benefits from the index. A CLI subcommand backing `search_documents` is the natural home for it, and would unify the hook with CLI-level search now that `/archcore:context` is removed under v2 (`remove-context-command.adr`).

## Value

**It closes the primary JTBD gap.** Without the hook the agent reads a rule or an ADR only if it spontaneously decides to; with it, every source-file edit carries its applicable constraints. That is the difference between a knowledge base the agent *can* read and one it *must* see.

**It scales with the knowledge base.** The more rules and cpats a team captures, the more the hook delivers, which is exactly the growth dynamic Archcore wants. A team that records one decision and one rule for a module gets automatic enforcement for every future agent edit in that module, across sessions and — combined with the sub-agent bootstrap — across delegated work.

**It differentiates from memory tools.** Those products solve recall of past context; none injects typed, project-specific constraints at the moment of code change. This hook is about constraints at the boundary rather than recall, which is the clearest wedge against generic memory products.

**It is cheap to demonstrate.** A README hero becomes compelling with no narrative: the user asks for a feature, the rule and the ADR appear in the agent's context before it writes, and the code respects both.

## Remaining phases

**Phase 2 — the path index.** Persist a path index in the sync manifest, updated on every `create_document`, `update_document`, and `remove_document`, making lookup constant-time. The performance budget is that the hook completes in under 500 ms on a 500-document repository. This is a prerequisite above roughly 50 documents.

**Phase 3 — ranking and session dedup.** Suppress re-injection of the same document inside one session unless the document changed, which reduces repetition fatigue, and refine specificity scoring so a document mentioning many paths is penalized as generic reference.

**Phase 4 — measurement.** Opt-in telemetry counting injections per session and the most-cited documents, feeding back into `/archcore:review --deep` as the most-applied rules.

## Risks

- **Performance.** The Phase 1 grep-per-token scan costs tokens times documents, acceptable at 50 documents or fewer and degrading beyond that. The Phase 2 index is the fix.
- **False positives.** A document referencing `src/` generically matches every source edit. Specificity ranking mitigates this without eliminating it, so precision needs monitoring through user feedback.
- **Noise against value.** Two pre-mutation entries on the same matcher double the per-edit shell fork. The short-circuit paths keep the overhead small for a non-source file, but cumulative overhead on a hot repository is worth watching.
- **The trigger surface is narrow.** `Write|Edit` catches an inline edit but not code reviewed in a planning tool and pasted later. Accepted for a first version.
- **Coupling to path conventions.** A monorepo with non-standard roots needs `codeAlignment.sourceRoots` configured, and the conservative default set covers the common layouts.
- **Sub-agent compatibility.** Hooks fire for a sub-agent tool call too, so combined with `subagent-knowledge-tree-bootstrap.adr` delegated work is covered.
- **Cursor parity.** Whether Cursor's `preToolUse` respects the context field is host-version-dependent, and the degradation is graceful: if it is ignored, the hook becomes a no-op there while SessionStart context and CLI-hook grounding still carry the pull path (context removed under v2).
- **User control.** The environment variable gives a global off-switch, and per-path muting is not implemented.
