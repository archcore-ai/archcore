---
title: "Command Surface v2 — init / plan / document / review"
status: accepted
tags:
  - "architecture"
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the plugin's layer-1 command surface after the 7-to-4 redesign: the command set, the free-text entry grammar, the retired mode words, and the category write-affinity model. Normative for the skill set under @plugin/plugins/archcore/skills/ and for host command wrappers. Out of scope: gate internals, interview mechanics, and the `archcore` CLI command surface.

## Surface

- Palette: `/archcore:init` (unchanged), `/archcore:plan`, `/archcore:document`, `/archcore:review`.
- Removed commands and their absorbing homes: `context` → CLI hook injection; `capture` and `decide` → `document`; `audit` → `review` gate; `help` → command descriptions, the init closing summary, and CLI `archcore help`.
- Write affinity: `plan` → vision types; `document` → knowledge types, with explicit `research` filing in vision; `review` → experience types.
- Read scope: all three categories for every command — vision supplies intent and resumption targets, knowledge supplies constraints, experience supplies precedent.
- Entry grammar: `/archcore:plan [topic]`, `/archcore:document [subject]`, `/archcore:review [free text: a path, a topic, 'close out the feature', 'are docs stale', 'audit the docs']` — one optional free-text argument; shared context resolution (@plugin/plugins/archcore/skills/_shared/gate-contract.md) selects the track, and a gate inside the track selects the document type. `/archcore:init [mode] [subject]` keeps the modes `import` and `refresh`.
- Retired mode words: `sdd`, `sources`, `iso`, `decision`, `code`, `drift`, `deep`, `closeout`, `experience`. At the head of a `plan`, `document`, or `review` request each is topic text with no handling of its own; the `research` slug at the head is a type request. The list exists for the release notes and the retired-form guard, not for runtime routing.
- Settings: no argument hint carries a `--flag`. `init` offers `depth:light|standard|deep` and `scale:small|medium|large` as toggles inside its preview, per `init-import-mode.adr`. A setting changes how an entry runs, never which entry runs.
- Document track map: a settled or proposed choice → `decision.classify` (`adr`, `rfc`, `rule`); a description of existing code → `describe.read` (`spec`, `doc`, `guide`, `task-type`, `scenario`); a finished report or one external material → `research.frame` (`research`, `rnd`, `evidence`).

## Normative Behavior

1. Before asking a question, the plan skill MUST ground the request in `.archcore/` search, git state, and code.
2. WHEN `plan` produces documents, the plan skill MUST produce vision types as the primary output.
3. WHEN a plan gate surfaces a decision, the plan skill MUST record the `adr` through the decision track.
4. WHEN the user invokes `document`, the document skill MUST classify the target as decision, code-doc, research material, or unclear before composing.
5. WHEN the classification is unclear, the document skill MUST inspect git state and the working tree before asking one classifying question.
6. WHEN the user invokes `review` without arguments on a feature branch, the review skill MUST resolve the merge base with the default branch and review the changes since divergence.
6a. WHEN the user invokes `review` without arguments on the default branch, the review skill MUST review the commits the upstream does not hold plus the uncommitted changes; without an upstream, the uncommitted changes alone.
7. WHEN `review` finds code and documents in conflict, the review skill MUST label the finding `spec-wrong`, `code-wrong`, or `ok`.
8. WHEN reviewed changes carry a repeated edit shape or a practice change that passes the transfer test, the durable-context selection SHOULD list a `cpat`, `task-type`, or procedural `guide` in its preview.
9. WHEN a skill gathers context, the skill MUST search all three categories.
10. A skill MUST NOT exclude a category from document reads.
11. WHEN a skill gathers context, the skill SHOULD pass a type filter matched to the command's moment instead of relying on the global type ranking.
12. WHEN a found document has `implements`, `depends_on`, or `related` relations, the skill SHOULD pull the linked documents one hop across categories.
13. The `plan`, `document`, and `review` skills MUST NOT print a notice for a retired mode word.
14. WHEN a command reports its result, the skill MUST list produced documents grouped by category.
15. WHEN the `research` slug leads a `plan` request, the plan skill MUST enter research frame and apply behaviors 22 and 23.
16. The plan skill MUST NOT treat a route name or a type slug other than `research` as an entry.
17. WHEN the user supplies a finished report to `document`, the document skill MUST use the report as research frame inputs.
18. WHEN the user supplies one external material and no investigation to `document`, the research instrument MUST enter research gather and produce `evidence`.
19. WHEN the selection contains no plan, the plan skill MUST bypass the Implement fork.
20. WHEN the engine lacks research vocabulary, the executing skill MUST apply the shared research compatibility contract.
21. The skill MUST treat every first word other than an explicit type request as topic text.
22. WHEN a research request asks to choose between candidates or settle a pending decision, the research instrument MUST produce `rnd`.
23. WHEN a research request asks no choice, the research instrument MUST produce `research`.
24. WHEN a report supplied to `document` ends in a recommendation, the research instrument MUST produce `rnd`.
25. WHEN a `document` request records a settled or proposed choice, the document skill MUST enter `decision.classify`.
26. WHEN a `document` request describes existing code, the document skill MUST enter `describe.read`.
27. WHEN the engine lacks the actor-subject vocabulary, the executing skill MUST apply the shared actor-subject compatibility contract.
28. WHEN the subject text names a document type that the selected track produces, the selecting gate MUST treat that type as settled.
29. WHEN a document type slug leads the subject, the document skill MUST produce that type before any track selection.
30. WHEN a `review` request asks whether documents are stale against code, the review skill MUST run the actualize track.
31. WHEN a `review` request asks for a full documentation audit, the review skill MUST run the actualize track over all documents.
32. WHEN a `review` request asks to close out finished work, the review skill MUST run the closeout track.
33. WHEN a `review` request asks to capture a repeated pattern, the review skill MUST run the durable-context selection over the named or branch scope without plan matching.
34. WHEN the user invokes `init refresh`, the init skill MUST bypass the already-seeded early exit and compose only missing documents.
35. WHEN the user invokes `init refresh` with a detected domain slug as the subject, the init skill MUST scope the seed to that domain's tree.
36. WHEN a skill reports to the user, the skill MUST NOT print a gate address of the form `<track>.<stage>`.
37. WHEN the user invokes `init import`, the init skill MUST run the import track in @plugin/plugins/archcore/skills/_shared/tracks/import.md.
38. WHEN a plain branch review matches a local `plan`, the review skill MUST run its closeout after the bidirectional check and actualize fixes.
39. WHEN a plain branch review matches no local `plan` to the branch work, the review skill MUST run the durable-context selection.
40. WHEN a matched plan has an unfulfilled task or criterion, the review skill MUST capture its delivered units.
41. WHEN a closeout request meets an empty diff without a named scope, the review skill MUST ask for that scope.
42. WHEN a capture request meets an empty diff without a named scope, the review skill MUST ask for that scope.
43. WHEN a branch review finds a draft with an open state block, the review skill MUST report its gate and resume command.
44. WHEN a review reports a finding, the review skill MUST name the next command for that finding kind.
45. WHEN `plan` or `document` runs without arguments and one matching draft carries a state block, the skill MUST resume that draft.
46. WHEN several candidates qualify for a no-argument `plan` or `document` invocation, the skill MUST show an evidenced menu of at most four items.
47. WHEN a no-argument invocation finds no reliable signal, the skill MUST ask one scope question before any write.
48. WHILE a matched plan has an unfulfilled task or criterion, the review skill MUST NOT offer its removal.

## Constraints & Invariants

- Constraint: the visible palette is exactly `init`, `plan`, `document`, `review`; a palette change requires a superseding ADR.
- Constraint: total questions per invocation MUST NOT exceed the shared elicitation budget.
- Constraint: the argument hint of a command and the argument hint of its skill are identical.
- Constraint: the descriptions of `plan`, `document`, and `review` name no retired mode word; the description of `init` names `import` and `refresh`.
- Constraint: no argument hint carries a `--flag`; a setting is a preview toggle and never selects an entry.
- Constraint: `rnd` is produced only by the research instrument's closing test, the spike, the compatibility fallback, or a request that names `rnd` under behavior 29.
- Constraint: a standalone material is filed only through `document`.
- Constraint: a slug leads the subject when it is the subject's first word or written in backticks; a slug elsewhere in topic text leads nothing. Behavior 29 takes precedence over behavior 21.
- Constraint: behavior 29 takes precedence over behavior 18: `document research <one external material>` produces `research`; a request that asks to file the material as evidence produces `evidence`.
- Constraint: a no-argument `review` reviews the current branch boundary and never reports a project health dashboard; it resumes no draft, because the review-side tracks keep no state block.
- Constraint: a `journey` is produced at `sdd.require` on `plan`, or on the document skill's named-type path when the request names `journey`.
- Constraint: the actor-subject types bind only when the engine gate in @plugin/plugins/archcore/skills/_shared/actor-subject-compatibility.md returns `yes`.
- Invariant: every one of the 23 document types is producible through at least one command path when the engine supports the vocabulary.
- Invariant: category is computed from the document type; no command asks the user to select a category.
- Invariant: skill content is byte-identical across hosts.

## Failure Behavior

1. IF `.archcore/` exists but contains no documents, THEN the invoked skill MUST proceed on outer-context grounding and report that zero documents were found.
2. IF `.archcore/` does not exist, THEN the invoked skill MUST announce initialization in one line and call `init_project` without asking a question.
3. IF the review diff is empty, THEN the review skill MUST request an explicit path, topic, or ref, except under staleness or audit wording, which widens the scope to all documents.
4. IF git state is unavailable, THEN the review skill MUST request an explicit path or topic.

## Conformance

The skill set is conformant when it satisfies behaviors 1–48, holds all invariants, and degrades per the failure rules. Regression coverage: @plugin/test/structure/command-grammar.bats pins the hints, the absence of a retired-word notice, the retired-form guard, the description parity, and the gate-address rule; @plugin/test/structure/loop-integrity.bats pins the exits from stopped, open, and unfinished states across plan, document, and review; @plugin/test/behavioral/document-bench.sh measures classification of `document` requests on a live model; @plugin/test/behavioral/import-bench.sh measures the route, the size tier, and the triage verdict of `init` on a live model.
