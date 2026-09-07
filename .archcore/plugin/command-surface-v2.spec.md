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

This spec defines the plugin's layer-1 command surface after the 7-to-4 redesign: the command set, routing modes, and the category write-affinity model. Normative for the skill set under @plugins/archcore/skills/ and for host command wrappers. Out of scope: gate internals, interview mechanics, and the `archcore` CLI command surface.

## Surface

- Palette: `/archcore:init` (unchanged), `/archcore:plan`, `/archcore:document`, `/archcore:review`.
- Removed commands and their absorbing homes: `context` → CLI hook injection; `capture` and `decide` → `document`; `audit` → `review` gate; `help` → command descriptions, the init closing summary, and CLI `archcore help`.
- Write affinity: `plan` → vision types; `document` → knowledge types, with explicit `research` filing in vision; `review` → experience types.
- Read scope: all three categories for every command — vision supplies intent and resumption targets, knowledge supplies constraints, experience supplies precedent.
- Invocation forms: no arguments; vague arguments; specific arguments; expert form `<command> <path|type|mode>`. The accepted expert names are exactly those the command's argument hint lists.
- Expert names per command: `plan` → `sdd`, `sources`, `iso`, `research`, plus the five route names; `document` → `adr`, `rfc`, `spec`, `doc`, `guide`, `rule`, `research`, `evidence`; `review` → its own hint. `rnd` is not an expert name on any command.

## Normative Behavior

1. Before asking a question, the plan skill MUST ground the request in `.archcore/` search, git state, and code.
2. WHEN `plan` produces documents, the plan skill MUST produce vision types as the primary output.
3. WHEN a plan gate surfaces a decision, the plan skill MUST record the `adr` through the decision track.
4. WHEN the user invokes `document`, the document skill MUST classify the target as decision, code-doc, or unclear before composing.
5. WHEN the classification is unclear, the document skill MUST inspect git state and the working tree before asking one classifying question.
6. WHEN the user invokes `review` without arguments, the review skill MUST resolve the merge base with the default branch and review the changes since divergence.
7. WHEN `review` finds code and documents in conflict, the review skill MUST label the finding `spec-wrong`, `code-wrong`, or `ok`.
8. WHEN reviewed changes repeat an undocumented pattern, the review skill SHOULD offer a `cpat` or `task-type` capture.
9. WHEN a skill gathers context, the skill MUST search all three categories.
10. A skill MUST NOT exclude a category from document reads.
11. WHEN a skill gathers context, the skill SHOULD pass a type filter matched to the command's moment instead of relying on the global type ranking.
12. WHEN a found document has `implements` or `related` relations, the skill SHOULD pull the linked documents one hop across categories.
13. WHEN the user names a path, type, or mode that the command's argument hint lists, the skill MUST execute the named path without routing.
14. WHEN a command reports its result, the skill MUST list produced documents grouped by category.

15. WHEN the user invokes `plan research`, the plan skill MUST enter research frame; the research instrument selects the type per behaviors 22 and 23.
16. The plan skill MUST NOT expose `rnd` or `evidence` as an entry.
17. WHEN the user invokes `document research`, the document skill MUST use the supplied report as research frame inputs.
18. WHEN the user invokes `document evidence`, the document skill MUST enter research gather for one material.
19. WHEN the package contains no plan, the plan skill MUST bypass the Implement fork.
20. WHEN the engine lacks research vocabulary, the executing skill MUST apply the shared research compatibility contract.
21. WHEN a leading word falls outside the command's argument hint, the skill MUST treat it as topic text, not as an entry.
22. WHEN a research request names a pending decision or a set of candidates to choose between, the research instrument MUST produce `rnd`.
23. WHEN a research request names no pending decision and no candidate set, the research instrument MUST produce `research`.
24. WHEN a report supplied to `document research` ends in a recommendation, the research instrument MUST produce `rnd`.

## Constraints & Invariants

- Constraint: the visible palette is exactly `init`, `plan`, `document`, `review`; a palette change requires a superseding ADR.
- Constraint: total questions per invocation MUST NOT exceed the shared elicitation budget.
- Constraint: the argument hint of a command and the argument hint of its skill are identical; together they are the command's complete expert surface.
- Constraint: `rnd` is produced only by the research instrument's closing test, the spike, or the compatibility fallback.
- Constraint: a standalone material is filed only through `document evidence`.
- Invariant: every one of the 21 document types is producible through at least one command path when the engine supports the vocabulary.
- Invariant: category is computed from the document type; no command asks the user to select a category.
- Invariant: skill content is byte-identical across hosts.

## Failure Behavior

1. IF `.archcore/` exists but contains no documents, THEN the invoked skill MUST proceed on outer-context grounding and report that zero documents were found.
2. IF `.archcore/` does not exist, THEN the invoked skill MUST announce initialization in one line and call `init_project` without asking a question.
3. IF `review` runs on the default branch or the diff is empty, THEN the review skill MUST report project health instead of a branch review.
4. IF git state is unavailable, THEN the review skill MUST request an explicit path or topic.

## Conformance

The skill set is conformant when it satisfies behaviors 1–24, holds all invariants, and degrades per the failure rules.