---
title: "Plugin Usage Catalog — Lifecycles, Single Commands, and Automatic Behavior"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "plugin"
  - "reference"
---

## Overview

This catalog lists every way a developer or a coding agent can use the Archcore plugin: multi-command lifecycles, single command entries, and behavior that runs with no command. Readers: a user choosing an entry for a task, and a plugin author checking that a change keeps each case reachable.

Each row names the situation, the entry the user types, what runs, what the run produces, and the usual next step. The rows index existing contracts; they add no obligation. Entry grammar: `command-surface-v2.spec` — since 2026-10-16 `plan`, `document`, and `review` take one optional free-text argument and no mode word; `init` keeps `import` and `refresh`. Cross-command edges: the transition register in `track-handoffs.spec`. Sources read on 2026-10-09 and updated 2026-10-16: `@plugin/plugins/archcore/skills/`, `@plugin/plugins/archcore/hooks/`, `@plugin/plugins/archcore/bin/`, `@plugin/plugins/archcore/agents/`.

All produced documents start at `draft`. Only `closeout.accept` and `decision.resolve` change a status, each after the user's confirmation.

## Content

### A. Lifecycles across commands

| # | Lifecycle | Entry sequence | Produces | Ends when |
|---|---|---|---|---|
| A1 | First day | `/archcore:init` → session-start recap → `/archcore:plan` or `/archcore:document` | host wiring, extractive facts, hotspot `spec`s, an architecture overview | the seed preview is confirmed once |
| A2 | Build a feature | `/archcore:plan <feature>` → implementation → `/archcore:review` | the owners the document-need test selects — typically `spec` + `plan`, a `prd` only for an outcome beyond one contract; review updates, accepts, captures, removes the plan | closeout removes the fulfilled plan after confirmation |
| A3 | Amend existing behavior | `/archcore:plan <change>` → amend the covering `spec` → implementation → `/archcore:review` | an amended `spec`; a new `spec` through callable describe when none covers the capability; a `plan` when work spans 2 or more dependent tasks | as A2 |
| A4 | Code-first, no plan | implementation by the user or another agent → `/archcore:review` | durable-context units through one preview: `spec`, `guide`, `doc`, `scenario`, `adr`, `cpat`, `task-type`, or none | the preview is answered |
| A5 | Proposal | `/archcore:document <proposal>` (`rfc`) → team discussion → `/archcore:document <verdict>` resolve | `rfc`, then on acceptance an `adr` trail with an optional `rule` and `guide` cascade | the `rfc` is accepted or rejected |
| A6 | Investigation before planning | `/archcore:plan research <topic>` or "investigate X" → selection resumes with the revised claims | `research` (scope coverage) or `rnd` (wording that asks to choose), optional `evidence` | the investigation closes, then A2 continues |
| A7 | Market and user discovery | `/archcore:plan <topic>` naming a market, business, or user question → `sdd.require` → contract → decompose | `mrd`, `brd`, or `urd` for that question, then `prd`, `spec`, `plan` | as A2 |
| A8 | Regulated work | `/archcore:plan <topic>` naming the traceability obligation | only the `brs`, `strs`, `syrs`, or `srs` levels the obligation requires | the required levels exist, then A2 continues |
| A9 | Migrate existing documentation | `/archcore:init import` → waves under an import plan → `/archcore:review` | native typed documents from CLAUDE.md, AGENTS.md, rule folders, ADR folders, contributor docs, git history | the import plan's waves complete |
| A10 | Keep docs current | session-start staleness warning → `/archcore:review are the docs stale` | confirmed fixes per document; `code-wrong` findings reported, never fixed | every finding has a verdict |

### B. Single commands

`/archcore:init`

| Situation | Entry | Result |
|---|---|---|
| Fresh clone, code present | `/archcore:init` | one preview: host wiring, facts, hotspot `spec`s; one confirm |
| Repository with no code and no authored docs | `/archcore:init` | host wiring only, behind its own confirm |
| Facts appeared since the first init | `/archcore:init refresh` | only missing documents; existing ones are skipped |
| One domain of a large repository needs depth | `/archcore:init refresh <domain>` | the seed scoped to that domain's tree |
| Agent instructions, ADR folders, or docs exist | `/archcore:init import [path]` | conversion to native types; staged by a plan above 8 targets |

`/archcore:plan [topic]`

| Situation | Entry | Result |
|---|---|---|
| A draft stopped mid-selection | `/archcore:plan` (no arguments) | resumes the stopped gate; several drafts → a menu of at most four; otherwise one question on what to plan |
| A small fix with no new canon | `/archcore:plan <fix>` | `selection: none`: no documents, the next action named |
| A non-capability migration with a fixed task order | `/archcore:plan <migration>` | one `plan`, no `spec` or `adr` |
| A choice must be made before building | `/archcore:plan <topic>` with an open choice | decision instrument: `adr` or `rfc` |
| One new capability | `/archcore:plan <feature>` | `spec` + `plan`; a `scenario` (and a `journey` beside a `prd`) under the illustrate condition |
| An outcome beyond one contract | `/archcore:plan <initiative>` | `prd` + one `spec` per capability + one `plan` |
| A feasibility question blocks the selection | a `feasibility` unknown | spike: timeboxed `rnd` with Goal, Questions, Findings; spike code never merges |

`/archcore:document [subject]`

| Situation | Entry | Result |
|---|---|---|
| A decision is already made | `/archcore:document we chose <choice>` | `adr`; optional `rule` + `guide` cascade |
| A proposal needs team acceptance | `/archcore:document a proposal for <change>` | `rfc` |
| A team standard over an existing decision | `/archcore:document make <standard> our standard` | cascade from the existing `adr`: `rule`, `guide` |
| An `rfc` was accepted or rejected | `/archcore:document` resolve wording | status change on confirmation; accepted → cascade |
| A module others rely on | `/archcore:document <module>` | `spec` |
| Reference data: a registry, glossary, lookup | `/archcore:document <subject>` | `doc`, only when a path cannot explain the data |
| A procedure a human runs | `/archcore:document <procedure>` | `guide` |
| A procedure an agent repeats | `/archcore:document <procedure>` | `task-type` |
| A user flow with examples over an existing `spec` | `/archcore:document scenario for <flow>` | `scenario` with `depends_on` → that `spec` |
| A finished report | `/archcore:document <report>` | `research`, or `rnd` when it ends in a recommendation |
| One external material | `/archcore:document <material>` | `evidence` |
| Unclear what to record | `/archcore:document` (no arguments) | resumes a matching draft, or git investigation, then one classifying question |
| Any document type named explicitly, for example an intended path with no `prd` | `/archcore:document journey for <path>` | that type under its contract; a poorly fitting type gets one report line, no question |

`/archcore:review [free text: a path, a topic, 'close out <plan>', 'are docs stale', 'audit']`

| Situation | Entry | Result |
|---|---|---|
| Branch ready for merge, a plan matches | `/archcore:review` | bidirectional check, closeout per plan, durable-context capture, plan removal on confirmation, experience offer |
| Branch ready for merge, a plan partly delivered | `/archcore:review` | the delivered units are captured; the plan stays with its remaining work named; no removal offer |
| Branch ready for merge, no plan | `/archcore:review` | bidirectional check, durable-context selection with one preview, experience offer |
| On the default branch with unpushed or uncommitted work | `/archcore:review` | the same branch review over the commits the upstream lacks plus the uncommitted changes |
| Empty diff | `/archcore:review` | one question for a path, topic, or ref |
| Docs may be stale | `/archcore:review are the docs stale` | actualize verdicts and confirmed fixes |
| Full corpus audit | `/archcore:review audit the whole knowledge base` | staleness plus claim sampling, cross-document conflicts, type fitness, relation candidates, coverage gaps |
| Close a finished feature explicitly | `/archcore:review close out <plan>` | closeout track; asks for a scope on the default branch |
| A repeated change or a reasoned practice change | `/archcore:review capture the pattern in <scope>` | `cpat` or `task-type` offer; `guide` for a human actor |
| A retired mode word leads the request | `/archcore:review drift` | no notice; the word is read as wording |

### C. Behavior with no command

| Trigger | What happens | Owner |
|---|---|---|
| Session start | recap of the corpus, open drafts, staleness; empty-state nudge to `/archcore:init` | `@plugin/plugins/archcore/bin/session-start` |
| An agent edits a code file | code-alignment advisory with the documents that claim the file | `@plugin/plugins/archcore/bin/pre-tool-use` |
| An agent writes `.archcore/` outside MCP | the write guard denies it | `@plugin/plugins/archcore/bin/pre-tool-use` |
| A document is created or updated | precision findings and a cascade notice naming related documents | `@plugin/plugins/archcore/bin/post-tool-use` |
| A Claude Code session runs (no `/archcore:*` command in the request) | next-step hints — at session start, after a settled decision, before a push, and the resume command of a stopped track — plus a line counting the Archcore documents found and read in the turn | `@plugin/plugins/archcore/hooks/next-step.tsx`, `@plugin/plugins/archcore/tests/next-step.test.ts` |
| Multi-document work in a conversation | the `archcore-assistant` agent | `@plugin/plugins/archcore/agents/archcore-assistant.md` |
| A read-only health audit | the `archcore-auditor` agent; never writes, never asks | `@plugin/plugins/archcore/agents/archcore-auditor.md` |
| Any agent question about the project | direct MCP reads: `search_documents`, `get_document`, `list_relations` | MCP server |

### D. Leaving a stopped or open state

| State | Exit |
|---|---|
| A gate stopped on an unanswered user-owned choice | answer, delegate ("you decide"), or proceed anyway; resume with the command the stop names |
| A draft carries an open track state block | the owning command resumes it with no arguments; `/archcore:review` lists it with its resume command |
| Drafts created with no matching plan | `/archcore:review` with a completion request offers their acceptance |
| An open `rfc` | `/archcore:document` with the verdict resolves it |
| A plan with unfinished tasks | the plan stays; review captures its delivered units and names the remaining work |

## Examples

Non-normative examples of one request per lifecycle entry:

- "Plan CSV export for reports" → A2: one `spec`, one `plan`.
- "Review my branch" after another agent added an endpoint → A4: the preview proposes one `spec` and names an internal helper as omitted.
- "We decided to use PostgreSQL 16" → `/archcore:document`: one `adr`, with an offer of a `rule` + `guide` cascade.
- "Import our CLAUDE.md and the docs/adr folder" → A9: `/archcore:init import`.
