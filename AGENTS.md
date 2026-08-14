# Repository Agent Instructions

## Purpose

This repository develops the Archcore agent plugin and its technical documentation system.

Write technical documentation so that human readers and AI agents can identify the subject, understand the constraints, and apply the information without guessing.

Two profiles govern every document: an ASD-STE100-inspired profile constrains the sentence, and an ISO 24495-1-inspired profile constrains the structure. Both apply everywhere; the document type decides which half binds and which half advises.

This policy is an internal writing profile. It is not a claim of compliance, certification, or approval by ASD, ISO, or any standards organization.

Authority is split across two shared Archcore rules, both mounted read-only from the `archcore` global source. `concepts/controlled-technical-writing` carries the profile — how an author writes. `concepts/document-prose-canon` carries the assignment — which profile, line format, and metric each document type takes. The text below mirrors the first so that a host reading this file directly still receives the profile; it does not mirror the assignment. When this file and a shared rule differ, the shared rule wins and this file is corrected. Repository-specific scope, precedence resolution, and enforcement live in `.archcore/plugin/controlled-technical-writing.rule.md`.

## Scope

Apply this policy when creating or updating:

- `.archcore/**/*.md`;
- `plugins/archcore/skills/**/*.md`;
- `plugins/archcore/agents/**/*.md`;
- `plugins/archcore/copilot-agents/**/*.md`;
- `plugins/archcore/rules/**/*.md`;
- `docs/**/*.md`;
- `README.md`;
- other user-facing technical documentation in this repository.

Do not rewrite:

- source code;
- code identifiers;
- commands;
- paths;
- configuration keys;
- API names;
- literal values;
- generated output;
- exact quotations from external sources.

## Precedence

Apply instructions in this order:

1. Explicit user requirements.
2. The content contract for the Archcore document type.
3. `plugins/archcore/skills/_shared/precision-rules.md`.
4. This repository writing policy.
5. General stylistic preferences.

When a type-specific contract conflicts with a general rule in this file, follow the type-specific contract.

## General writing rules

1. Identify the intended reader and the task that the document supports.
2. State the purpose or principal outcome near the beginning.
3. Organize sections in the order in which the reader needs the information.
4. Express one primary idea in each sentence.
5. Express one topic in each paragraph or list item.
6. Use an explicit actor when responsibility or behavior matters.
7. Prefer active voice when it identifies the responsible component.
8. Put a condition before the action, requirement, or result that depends on it.
9. Use one preferred term for one concept.
10. Do not replace an established term with a synonym for stylistic variety.
11. Preserve code identifiers, paths, commands, API names, flags, field names, and configuration keys exactly.
12. Replace qualitative claims with facts, thresholds, versions, units, or observable outcomes.
13. Mark unsupported technical claims with `[assumption]`.
14. Distinguish current behavior, proposed behavior, and future possibilities.
15. Distinguish facts, decisions, requirements, examples, and assumptions.
16. Separate instructions, notes, warnings, examples, and explanations.
17. Do not hide a required action inside a note or explanatory paragraph.
18. Remove filler that adds no architectural, operational, or normative information.
19. Do not invent missing constraints, measurements, behavior, rationale, or guarantees.
20. Use visible placeholders when required information is unavailable.

Use placeholders such as:

- `[ACTOR REQUIRED]`
- `[CONDITION REQUIRED]`
- `[METRIC REQUIRED]`
- `[LIMIT REQUIRED]`
- `[EVIDENCE REQUIRED]`

## Terminology

Maintain stable terminology within a document and across related Archcore documents.

- Use the canonical repository term for each concept.
- Define an unfamiliar term before relying on it.
- Do not use the same term for different concepts.
- Do not translate or paraphrase code identifiers.
- Avoid ambiguous pronouns such as `it`, `this`, or `they` when more than one referent is possible.
- When two existing terms might represent the same concept, surface the conflict instead of silently normalizing it.

## Procedures and guides

Apply these rules to guides, runbooks, migrations, recovery procedures, installation instructions, and numbered agent workflows.

1. State prerequisites before the procedure.
2. State required inputs before the procedure.
3. Use the imperative form for procedural actions.
4. Put one primary action in each numbered step.
5. Put the condition before the step that it controls.
6. Put a warning before the hazardous or destructive action.
7. Do not put mandatory actions in notes.
8. Separate alternative flows into branches or subsections.
9. Preserve commands exactly as the reader must enter them.
10. State the expected result after important verification points.
11. Include rollback or recovery instructions only when they are supported by repository evidence.
12. Keep background explanation outside the numbered steps.

Prefer this structure when applicable:

1. Purpose
2. Prerequisites
3. Inputs
4. Procedure
5. Verification
6. Rollback
7. Troubleshooting

Prefer this step form:

`If <condition>, <imperative action>.`

Add an observable result when needed:

`Expected result: <observable outcome>.`

## Rules, specifications, and requirements

Apply these rules to `rule`, `spec`, `brs`, `strs`, `syrs`, `srs`, contracts, and acceptance criteria.

1. Put one requirement in each numbered item.
2. Use one normative modal in each requirement.
3. State the obligated actor explicitly.
4. Put the trigger or condition before the obligation.
5. Use `MUST`, `MUST NOT`, `SHOULD`, or `MAY` in uppercase for normative meaning.
6. Use `MUST` only for behavior required for correctness, interoperability, safety, or an established repository constraint.
7. Make each requirement objectively verifiable.
8. Give requirements stable identifiers when traceability is needed.
9. State measurable limits when repository evidence provides them.
10. Do not use open-ended lists such as `etc.` in normative statements.
11. Do not hide requirements in rationale, examples, headings, or notes.
12. Do not combine independent obligations with `and` or `or`. Split them into separate requirements.
13. Preserve the notation and mandatory sections defined by the relevant Archcore content contract.

Prefer these forms:

- `The <actor> MUST <response>.`
- `WHEN <trigger>, the <actor> MUST <response>.`
- `WHILE <state>, the <actor> MUST <response>.`
- `IF <undesired condition>, THEN the <actor> MUST <response>.`

## Decisions and explanatory documents

Apply these rules to ADRs, RFCs, architecture documents, PRDs, plans, reference documents, and explanatory README sections.

1. Do not force imperative or normative phrasing into descriptive content.
2. State the purpose, conclusion, or decision before supporting detail.
3. Separate context from the decision or proposal.
4. Separate mechanism from rationale.
5. Separate benefits from verified outcomes.
6. Identify trade-offs, limitations, and consequences.
7. Label examples as non-normative examples.
8. Use headings that describe the reader's question or task.
9. Keep paragraphs short enough to expose the logical structure.
10. Reference source files with `@path/to/file` instead of reproducing implementation bodies.

## Agent instructions and skills

Apply the procedural rules to agent-facing instructions under `plugins/archcore/`.

- Write instructions as direct actions.
- Put routing conditions before the routed action.
- Keep one required action in each numbered instruction.
- Name the tool, file, document type, or state explicitly.
- Separate mandatory behavior from rationale.
- State exceptions immediately after the rule they modify.
- Do not rely on an instruction implied only by an example.
- Use the same name for a workflow, document type, tool, and state throughout the file.
- Do not introduce a second term for an existing Archcore concept.

## Language

English is the default language for repository documentation unless the user or existing document requires another language.

For Russian documentation:

- apply the same structural, terminology, evidence, and condition-first rules;
- use explicit actors in requirements and procedures;
- avoid impersonal normative expressions when they hide responsibility;
- do not imitate ASD-STE100 English vocabulary or English grammar;
- preserve BCP 14 keywords when the Archcore document contract requires them.

## Review checklist

Before finalizing technical documentation, silently verify:

- The intended reader and task are clear.
- The purpose appears near the beginning.
- Terminology is consistent.
- Code identifiers and literal values are unchanged.
- Every normative statement has an explicit actor.
- Every requirement contains one obligation and one modal.
- Every procedural step contains one primary action.
- Conditions appear before dependent actions.
- Required actions are not hidden in notes.
- Qualitative claims have evidence, a measurement, or `[assumption]`.
- Facts, proposals, assumptions, and examples are distinguishable.
- No unsupported behavior or guarantee was introduced.
- The document follows its Archcore type contract.

Revise known violations before returning the document. Do not include the checklist or a writing-quality score in the generated document unless the user asks for a review report.

<!-- archcore:start --> managed by `archcore init` — edit outside these markers
## Archcore — project context for this repo

This repo's architecture, decisions, rules, specs and patterns live in `.archcore/`,
reachable through the Archcore MCP tools. Consult them even on code you think you
know — a decision or rule may already constrain it.

- Touching this repo's real code or behavior → search first; read only what matches.
- A decision was made ("we'll use X", "from now on Y") → record it.
- A module / API / system has no doc — or a search comes back empty → capture it.
- Planning a feature or refactor → scope it against what's already decided.

A `.archcore/` may also mount read-only **global sources** — shared, org-wide
context not shown in the session-start list. `list_documents` / `search_documents`
surface them alongside local docs, tagged `source_kind: "global"`. When present,
treat them as defaults a local doc can override — never edit or relate to one.

The search is cheap — lean on it. Skip it only for turns this repo would have no
opinion on: syntax trivia, throwaway snippets, pure mechanics.
<!-- archcore:end -->
