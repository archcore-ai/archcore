---
title: "Command Entry Grammar — How Users Enter the Four Commands"
status: draft
tags:
  - "actor:plugin-user"
  - "commands"
  - "component:plugin"
  - "component:skills"
  - "plugin"
---

## Subject

The argument surface of `/archcore:init`, `/archcore:plan`, `/archcore:document`, and `/archcore:review` under the free-text grammar of 2026-10-16; illustrates clauses 13, 15, 16, 18, 21, 22, 23, 26, 28, 29, 30, 34, 36, and 45–47 of the command-surface spec. The routing fixtures, the document bench, and the skill-selection bench depend on these examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Expert | a user who knows the Archcore document types | the exact type with no question |
| Newcomer | a user who knows the task, not the types | the right document from plain words |

## Flows

### Expert

Anchors: @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/commands/document.md, @plugin/plugins/archcore/skills/_shared/tracks/decision.md, @plugin/test/fixtures/routing/fixtures.tsv

1. Expert types `/archcore:document`; the host shows `[subject]`.
2. Expert types `rfc for gRPC`; the leading slug opens the RFC branch without a question.
3. Expert types `/archcore:review are the docs stale`; the skill runs the staleness check on the branch.
4. Expert types `/archcore:plan sdd csv export`; the skill reads `sdd` as topic text, prints no notice, and plans CSV export.

Extensions:

- 2a. Expert types `` a proposal, `rfc`, for gRPC ``; the slug in backticks takes the named-type path.
- 3a. Expert types `drift`; the skill reads the word as staleness wording and prints no notice.

### Newcomer

Anchors: @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/describe.md, @plugin/plugins/archcore/skills/_shared/tracks/research.md, @plugin/test/behavioral/fixtures/skill-bench.tsv

1. Newcomer writes "document the auth module" with no command; the host starts the document skill.
2. Newcomer types `/archcore:document the auth module`; the skill reads the code and drafts a spec.
3. Newcomer types `/archcore:document` with a vendor benchmark attached; the skill files one evidence record.
4. Newcomer reads the result; the report names documents, and no gate address.

Extensions:

- 2a. The code shows no contract; the skill asks one type question.
- 3a. Newcomer supplies a report ending in a recommendation; the skill files an rnd.
- 4a. Newcomer types `/archcore:plan` with no argument and one open draft; the skill resumes that draft.

## Examples

Background: the plugin carries the free-text grammar, and CLI 0.8.4 is on PATH.

### Naming the type inside a decision

Illustrates: clauses 28, 29.
Given the repository holds no RFC on gRPC.
When Expert types `/archcore:document rfc for switching to gRPC`.
Then Expert sees an RFC draft, with no question asked.

### Filing one external material

Illustrates: clause 18.
Given no investigation on the vendor benchmark exists.
When Newcomer types `/archcore:document` with the benchmark attached.
Then Newcomer sees one evidence draft and no research draft.

### Describing an intended path without naming a journey

Illustrates: clause 26.
Given no journey on onboarding exists.
When Newcomer types `/archcore:document the onboarding path of a trial user`.
Then Newcomer sees no journey draft and a pointer to `/archcore:plan`.

### Naming a journey

Illustrates: clause 29.
Given no journey, spec, or prd on onboarding exists.
When Expert types `/archcore:document journey for the onboarding path of a trial user`.
Then Expert sees one journey draft and no prd.

### First words across the palette

Illustrates: clauses 13, 15, 16, 21, 22, 23, 30, 34.

| Input | Entry | notes |
|---|---|---|
| `document we chose Postgres` | decision track, classify | adr by default |
| `document the payment module` | describe track, read | spec, doc, guide, task-type, or scenario |
| `document adr we chose Postgres` | named type, decision track | a leading slug is a named type |
| `plan research licence facts for every dataset` | research track, frame | `research`: the wording asks no choice |
| `plan research compare queues and pick one` | research track, frame | `research`: a leading slug is a type request; `compare queues and pick one` without the slug gives `rnd` |
| `plan capability csv export` | topic text, selection computed | route names are not entries |
| `plan sources billing product` | topic text, no notice | retired mode word |
| `review drift` | actualize track, no notice | retired mode word read as staleness wording |
| `review close out the feature` | closeout track | completion phrasing |
| `init refresh` | init without early exit | `init` keeps its modes |

### Reading a result

Illustrates: clause 36.
Given the decision track closed on an ADR.
When Expert reads the closing report.
Then Expert sees the ADR path and its category, and no `decision.adr` string.

## Open Questions

- Does selection accuracy hold on Cursor, Codex, and Copilot, where no skill-selection bench runs?
- Does a former mode word such as `sources` cost a question more often than once in 10 skill-bench rows?
