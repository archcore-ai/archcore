---
title: "Context-Driven Routing Replaces the Mode Words and the ΔΠMR Route Table"
status: accepted
tags:
  - "architecture"
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

The conductor in @plugin/plugins/archcore/skills/_shared/delta-routing.md selects the document package from a capability count (`creates = 1` gives one `spec` and one `plan`, `creates ≥ 2` gives a `prd`, specs, and a `plan`, `security-compliance` engages iso links) and raises a size label S/M/L/XL that no gate reads: over `plugin/plugins/archcore/skills/` the label appears only in its declaration, the `archcore:track` state block, and `plan/SKILL.md`. The 2026-10-15 baseline in the delta-routing bench document shows the route-name score saturated at 44 of 44 on `claude-fable-5-1` and `claude-haiku-5-5`, while the held-out traces show the package mismatch the companion RFC names: a library upgrade with a fixed rollout order routes to `decision` with an `adr` and a `plan`, a privilege-escalation fix with no traceability obligation routes to `amendment (size L)` with iso links `requirements-cascade.brs → srs`, and a partially delivered plan leaves its delivered public endpoint uncaptured (`created=none; plan=retain`). The same day's runs show the mode words carry no information the wording lacks: the 14 mode-less rows of document-bench classify without a mode word (27 of 27 on `claude-fable-5-1`), and hosts passed no document mode word in 10 of 10 skill-bench calls. Neither stated supersede trigger of `command-entry-grammar.adr` is met (document-bench is not below 18 of 20; no fifth command exists), so this record states its own reason: the mode list is a second grammar the user has to learn, and the context layer makes it redundant.

## Decision

Adopted context-driven routing as @.archcore/plugin/context-driven-skill-routing-and-review-learning.rfc.md specifies: the hints of `/archcore:plan`, `/archcore:document`, and `/archcore:review` become `[topic]`, `[subject]`, and `[path, tag, or scope]`, no first word is reserved as a mode and no alias of a former mode word exists, every invocation runs one shared read-only context resolution (request and supplied material, open `archcore:track` drafts, accepted local and global canon, branch state, code and tests) and selects artifacts by claim ownership and a named future reader task instead of the ΔΠMR package table, its route names, and its size escalators, the announcement reports only the decisions that change an instrument, a check, or a write, the leading-slug rule of `explicit-type-request-wins.adr` stays and a leading `research` is a request for the type `research` while comparison or choice wording selects `rnd`, and a ceiling of 5 questions per invocation, with the remaining material questions kept in `deferred` for the next invocation, replaces the expert-invocation budget.

## Alternatives Considered

1. Keep the mode words and the hints and change only the mode-less path (the 2026-10-15 compatibility draft) — rejected because it keeps the second grammar the RFC exists to remove.
2. Keep the mode words but hide them from the hints — rejected because the visible hint becomes incomplete while the words still work, the reason `command-entry-grammar.adr` already recorded.
3. Retire the mode words through a deprecation window of two minor releases with a retired notice — rejected because it keeps two entry grammars for two releases; a former mode word stays readable as wording, and the release notes carry the change as they did on 2026-09-16.
4. Keep Δ, Π, M, and R as mandatory diagnostics and replace only the package table (RFC alternative 6) — rejected because every run still pays four classifications (0.85k output tokens per announcement on `claude-fable-5-1` at the baseline) and the size label still changes no gate check.
5. Replace plan deletion with a `completed` status so a partially delivered plan needs no capture rule (RFC alternative 5) — deferred because the kernel knows only `draft`, `accepted`, and `rejected` (@cli/templates/templates.go).
6. Treat a leading `research` as topic text, an exception to the leading-slug rule — rejected because one slug with two readings makes the leading-slug rule unpredictable; the user chose the type reading on 2026-10-15.
7. Keep per-gate question maxima without a mode word and raise them by request complexity — rejected because no recorded signal decides complexity without a new classifier, and `deferred` with resume already carries a long interview across invocations; the user chose the fixed ceiling on 2026-10-15.

## Consequences

Positive:

- One entry grammar: three hints of one optional free-text argument each and no closed word list to learn; a leading type slug and a no-argument invocation keep their result class. [expected]
- A security repair without a traceability obligation engages no iso links, a sequenced non-capability migration receives a `plan` without a manufactured `adr`, and the delivered part of an open plan is captured; measured on the held-out cases H1, H2, and the partial plan against the 2026-10-15 baseline in the artifact-level bench. [expected]
- Fewer classification tokens per announcement than the baseline 0.85k output tokens per route call on `claude-fable-5-1`. [expected]
- The question budget is the same on every command and every path: at most 5 questions per invocation, and no answered question is asked again on resume.

Negative:

- Second grammar change in one month after `command-entry-grammar.adr` (2026-09-16): `/archcore:plan sources …` and `/archcore:review drift` become topic text on the day of the release, an ambiguous former mode word (`sources`) can cost one question, and host wrappers that embed a mode word change with the release.
- `/archcore:plan research <candidates>` and `/archcore:document research <report that ends in a recommendation>` produce `research`, not `rnd`; a user who wants `rnd` names it or words the request as a choice. [expected]
- A large initiative asks at most 5 questions per invocation, where the `sdd` per-gate maxima allowed up to 14 across its six gates (@plugin/plugins/archcore/skills/_shared/tracks/sdd.md); the remaining questions wait in `deferred` for the next invocation.
- @plugin/test/structure/command-grammar.bats, the eleven mode rows of @plugin/test/behavioral/fixtures/skill-bench.tsv, @plugin/test/structure/delta-routing.bats, @plugin/test/structure/loop-integrity.bats, `command-surface-v2.spec`, `track-layer.spec`, `delta-routing-conductor.spec` (24 behaviors), and `delta-routing-instruments.spec` (37 behaviors) change; `delta-routing-model.idea` is superseded; alternative 7 of `durable-context-selection-by-reader-task.adr` reverses; the test changes land in the same commit as the routing contract, or `make test` fails on `dev`.
- Context resolution adds MCP reads and tokens to every invocation; the amount is unmeasured before the rollout plan's Phase 2 [assumption]; the review bench already costs 28.1k input tokens per call at the baseline.
- Claim-level selection is less mechanically predictable than a capability-count table, and the route-name bench no longer exists as a guard; the artifact-level bench of the rollout plan's Phase 5 is the only guard.

## Superseded when

- The artifact-level bench records more false extra documents or more missed contracts than the 2026-10-15 baseline on the same model (route-bench 44 of 44; review-bench 12 of 12 on `claude-fable-5-1`, 9 of 12 on `claude-haiku-5-5`).
- Plain-language requests for the former mode paths (market research, ISO cascade, staleness audit, closeout) select the wrong instrument in more than 1 of 10 skill-bench rows on the baseline model.
- Mode-less document entry drops below the same-model baseline of document-bench (27 of 27 on `claude-fable-5-1`; 22 of 27 on `claude-haiku-5-5`).
- The kernel adds a document status or a routing facility that makes the plugin-side selection redundant.
