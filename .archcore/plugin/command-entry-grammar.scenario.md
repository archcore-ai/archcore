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

The argument surface of `/archcore:init`, `/archcore:plan`, `/archcore:document`, and `/archcore:review`; illustrates clauses 13, 16, 17, 18, 21, 25, 26, 28, 29, 30, 34, and 36 of the command-surface spec. The routing fixtures, the document bench, and the skill-selection bench depend on these examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Expert | a user who knows the Archcore document types | the exact type with no question |
| Newcomer | a user who knows the task, not the types | the right document from plain words |

## Flows

### Expert

Anchors: @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/commands/document.md, @plugin/plugins/archcore/skills/_shared/tracks/decision.md, @plugin/test/fixtures/routing/fixtures.tsv

1. Expert types `/archcore:document`; the host shows `[decision|code|research] [subject]`.
2. Expert types `decision rfc for gRPC`; the skill opens the RFC branch without a question.
3. Expert types `/archcore:review drift`; the skill runs the drift check on the branch.
4. Expert types `/archcore:plan sdd csv export`; the skill runs the full package.

Extensions:

- 2a. Expert types `rfc for gRPC` with no mode; the skill reads the words as a topic and classifies them.
- 3a. Expert types `--drift`; the skill reads the flag as a topic, and the release notes name the new form.

### Newcomer

Anchors: @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/describe.md, @plugin/plugins/archcore/skills/_shared/tracks/research.md, @plugin/test/behavioral/fixtures/skill-bench.tsv

1. Newcomer writes "document the auth module" with no command; the host starts the document skill.
2. Newcomer types `/archcore:document code the auth module`; the skill reads the code and drafts a spec.
3. Newcomer types `/archcore:document research` with a vendor benchmark; the skill files one evidence record.
4. Newcomer reads the result; the report names documents and modes, and no gate address.

Extensions:

- 2a. The code shows no contract; the skill asks one type question.
- 3a. Newcomer supplies a report ending in a recommendation; the skill files an rnd.

## Examples

Background: the plugin carries the command entry grammar, and CLI 0.8.4 is on PATH.

### Naming the type inside a decision

Illustrates: clause 28.
Given the repository holds no RFC on gRPC.
When Expert types `/archcore:document decision rfc for switching to gRPC`.
Then Expert sees an RFC draft, with no question asked.

### Filing one external material

Illustrates: clause 18.
Given no investigation on the vendor benchmark exists.
When Newcomer types `/archcore:document research` with the benchmark attached.
Then Newcomer sees one evidence draft and no research draft.

### Asking for a journey through document

Illustrates: clause 29.
Given no journey on onboarding exists.
When Newcomer types `/archcore:document code the onboarding path of a trial user`.
Then Newcomer sees no journey draft and a pointer to `/archcore:plan`.

### Mode words across the palette

Illustrates: clauses 13, 16, 21, 25, 26, 30, 34.

| Input | Entry | notes |
|---|---|---|
| `document decision we chose Postgres` | decision track, classify | adr by default |
| `document code the payment module` | describe track, read | spec, doc, guide, or scenario |
| `document research <report>` | research track, frame | research or rnd by closing test |
| `document adr we chose Postgres` | topic text, classification | `adr` is not a mode |
| `plan research compare queues` | research track, frame | rnd: a candidate set |
| `plan capability csv export` | topic text, route computed | route names are not modes |
| `review drift` | actualize track | former `--drift` |
| `review closeout` | closeout track | also by completion phrasing |
| `init refresh` | init without early exit | former `--refresh` |

### Reading a result

Illustrates: clause 36.
Given the decision track closed on an ADR.
When Expert reads the closing report.
Then Expert sees the ADR path and its category, and no `decision.adr` string.

## Open Questions

- Does selection accuracy hold on Cursor, Codex, and Copilot, where no skill-selection bench runs?
- Does a host other than Claude Code show a long argument hint in full, or does it cut the mode list?
