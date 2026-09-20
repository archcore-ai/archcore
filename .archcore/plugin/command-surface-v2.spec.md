---
title: "Command Surface v2 — init / plan / document / review"
status: accepted
tags:
  - "architecture"
  - "commands"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the plugin's layer-1 command surface after the 7-to-4 redesign: the command set, the entry grammar, routing modes, and the category write-affinity model. Normative for the skill set under @plugins/archcore/skills/ and for host command wrappers. Out of scope: gate internals, interview mechanics, and the `archcore` CLI command surface.

## Surface

- Palette: `/archcore:init` (unchanged), `/archcore:plan`, `/archcore:document`, `/archcore:review`.
- Removed commands and their absorbing homes: `context` → CLI hook injection; `capture` and `decide` → `document`; `audit` → `review` gate; `help` → command descriptions, the init closing summary, and CLI `archcore help`.
- Write affinity: `plan` → vision types; `document` → knowledge types, with explicit `research` filing in vision; `review` → experience types.
- Read scope: all three categories for every command — vision supplies intent and resumption targets, knowledge supplies constraints, experience supplies precedent.
- Entry grammar: `/archcore:<command> [mode] [subject]`. The mode is the first word, a noun from the closed list the argument hint shows; a gate inside the track selects the document type.
- Modes per command: `init` → `import`, `refresh`; `plan` → `sdd`, `sources`, `iso`, `research`; `document` → `decision`, `code`, `research`; `review` → `drift`, `deep`, `closeout`, `experience`.
- Settings: no argument hint carries a `--flag`. `init` offers `depth:light|standard|deep` and `scale:small|medium|large` as toggles inside its preview, per `init-import-mode.adr`. A setting changes how an entry runs, never which entry runs.
- Document mode map: `decision` → `decision.classify` (`adr`, `rfc`, `rule`); `code` → `describe.read` (`spec`, `doc`, `guide`, `scenario`); `research` → `research.frame` (`research`, `rnd`, `evidence`).

## Normative Behavior

1. Before asking a question, the plan skill MUST ground the request in `.archcore/` search, git state, and code.
2. WHEN `plan` produces documents, the plan skill MUST produce vision types as the primary output.
3. WHEN a plan gate surfaces a decision, the plan skill MUST record the `adr` through the decision track.
4. WHEN the user invokes `document` without a mode, the document skill MUST classify the target as decision, code-doc, research material, or unclear before composing.
5. WHEN the classification is unclear, the document skill MUST inspect git state and the working tree before asking one classifying question.
6. WHEN the user invokes `review` without arguments, the review skill MUST resolve the merge base with the default branch and review the changes since divergence.
7. WHEN `review` finds code and documents in conflict, the review skill MUST label the finding `spec-wrong`, `code-wrong`, or `ok`.
8. WHEN reviewed changes repeat an undocumented pattern, the review skill SHOULD offer a `cpat` or `task-type` capture.
9. WHEN a skill gathers context, the skill MUST search all three categories.
10. A skill MUST NOT exclude a category from document reads.
11. WHEN a skill gathers context, the skill SHOULD pass a type filter matched to the command's moment instead of relying on the global type ranking.
12. WHEN a found document has `implements` or `related` relations, the skill SHOULD pull the linked documents one hop across categories.
13. WHEN the first word of the arguments is a mode that the command's argument hint lists, the skill MUST execute the mapped path without routing.
14. WHEN a command reports its result, the skill MUST list produced documents grouped by category.
15. WHEN the user invokes `plan research`, the plan skill MUST enter research frame; the research instrument selects the type per behaviors 22 and 23.
16. The plan skill MUST NOT accept a document type name or a route name as an entry.
17. WHEN the user invokes `document research` with a supplied report, the document skill MUST use the report as research frame inputs.
18. WHEN the user invokes `document research` with one external material and no investigation, the research instrument MUST enter research gather and produce `evidence`.
19. WHEN the package contains no plan, the plan skill MUST bypass the Implement fork.
20. WHEN the engine lacks research vocabulary, the executing skill MUST apply the shared research compatibility contract.
21. WHEN the first word falls outside the command's argument hint, the skill MUST treat it as topic text, not as an entry.
22. WHEN a research request names a pending decision or a set of candidates to choose between, the research instrument MUST produce `rnd`.
23. WHEN a research request names no pending decision and no candidate set, the research instrument MUST produce `research`.
24. WHEN a report supplied to `document research` ends in a recommendation, the research instrument MUST produce `rnd`.
25. WHEN the user invokes `document decision`, the document skill MUST enter `decision.classify`.
26. WHEN the user invokes `document code`, the document skill MUST enter `describe.read`.
27. WHEN the engine lacks the actor-subject vocabulary, the executing skill MUST apply the shared actor-subject compatibility contract.
28. WHEN the subject text names a document type that the mode's track produces, the selecting gate MUST treat that type as settled.
29. The document skill MUST NOT produce a `journey`.
30. WHEN the user invokes `review drift`, the review skill MUST run the actualize track.
31. WHEN the user invokes `review deep`, the review skill MUST run the actualize track over all documents with coverage and relation findings.
32. WHEN the user invokes `review closeout`, the review skill MUST run the closeout track.
33. WHEN the user invokes `review experience`, the review skill MUST run the experience track.
34. WHEN the user invokes `init refresh`, the init skill MUST bypass the already-seeded early exit and compose only missing documents.
35. WHEN the user invokes `init refresh` with a detected domain slug as the subject, the init skill MUST scope the seed to that domain's tree.
36. WHEN a skill reports to the user, the skill MUST NOT print a gate address of the form `<track>.<stage>`.
37. WHEN the user invokes `init import`, the init skill MUST run the import track in @plugins/archcore/skills/_shared/tracks/import.md.

## Constraints & Invariants

- Constraint: the visible palette is exactly `init`, `plan`, `document`, `review`; a palette change requires a superseding ADR.
- Constraint: total questions per invocation MUST NOT exceed the shared elicitation budget.
- Constraint: the argument hint of a command and the argument hint of its skill are identical; together they are the command's complete expert surface.
- Constraint: the description of a command and the description of its skill each name every mode of the argument hint.
- Constraint: no argument hint carries a `--flag`; a setting is a preview toggle and never selects an entry.
- Constraint: `rnd` is produced only by the research instrument's closing test, the spike, or the compatibility fallback.
- Constraint: a standalone material is filed only through `document research`.
- Constraint: a `journey` is produced only at `sdd.require` on `plan`.
- Constraint: the actor-subject types bind only when the engine gate in @plugins/archcore/skills/_shared/actor-subject-compatibility.md returns `yes`.
- Invariant: every one of the 23 document types is producible through at least one command path when the engine supports the vocabulary.
- Invariant: category is computed from the document type; no command asks the user to select a category.
- Invariant: skill content is byte-identical across hosts.

## Failure Behavior

1. IF `.archcore/` exists but contains no documents, THEN the invoked skill MUST proceed on outer-context grounding and report that zero documents were found.
2. IF `.archcore/` does not exist, THEN the invoked skill MUST announce initialization in one line and call `init_project` without asking a question.
3. IF `review` runs on the default branch or the diff is empty, THEN the review skill MUST report project health instead of a branch review.
4. IF git state is unavailable, THEN the review skill MUST request an explicit path or topic.

## Conformance

The skill set is conformant when it satisfies behaviors 1–37, holds all invariants, and degrades per the failure rules. Regression coverage: @test/structure/command-grammar.bats pins the hints, the mode maps, the description parity, and the gate-address rule; @test/behavioral/document-bench.sh measures classification of `document` requests on a live model; @test/behavioral/import-bench.sh measures the route, the size tier, and the triage verdict of `init` on a live model.
