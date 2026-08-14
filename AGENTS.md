# Repository Agent Instructions

## Purpose

This repository develops the Archcore CLI, MCP server, host integrations, and document templates.

Write technical documentation so that human readers and AI agents can identify the subject, understand the constraints, and apply the information without guessing.

Two profiles govern every document: an ASD-STE100-inspired profile constrains the sentence, and an ISO 24495-1-inspired profile constrains the structure. Both apply everywhere; the document type decides which half binds and which half advises.

The assignment — profile, line format, and metric per document type — lives in the shared Archcore rule `concepts/document-prose-canon`, mounted read-only from the `archcore` global source. Do not restate that assignment here, and do not classify a document into a profile by hand when the rule already assigns one to its type.

This policy is an internal writing profile. It is not a claim of compliance, certification, or approval by ASD, ISO, or any standards organization.

## Scope

Apply this policy when creating or updating:

- `.archcore/**/*.md`;
- `README.md`;
- `docs/**/*.md`;
- `templates/**/*.md`;
- Markdown embedded in Go source;
- CLI help text and error messages;
- MCP tool descriptions and prompts;
- agent instruction files;
- user-facing installation, migration, recovery, and troubleshooting instructions.

Do not rewrite or translate:

- Go identifiers;
- package names;
- commands;
- flags;
- paths;
- configuration keys;
- MCP tool names;
- JSON fields;
- document type names;
- literal values;
- generated output;
- exact quotations from external sources.

## Precedence

Apply instructions in this order:

1. Explicit user requirements.
2. Accepted rules and decisions in `.archcore/`.
3. Type-specific document contracts and templates.
4. This repository writing policy.
5. General stylistic preferences.

When instructions conflict, follow the higher-priority instruction.

## Archcore-first context

Before researching project behavior, architecture, conventions, or implementation approaches:

1. Search `.archcore/` documents first.
2. Read only the documents relevant to the current task.
3. Search the codebase when `.archcore/` does not provide enough evidence.
4. Use external sources only when repository context and source code are insufficient.

Treat mounted global Archcore documents as read-only defaults. Do not edit them or create relations to them.

## General writing rules

1. Identify the intended reader and the task that the text supports.
2. State the purpose, result, or principal conclusion near the beginning.
3. Organize information in the order in which the reader needs it.
4. Express one primary idea in each sentence.
5. Express one topic in each paragraph or list item.
6. Use an explicit actor when responsibility or behavior matters.
7. Prefer active voice when it identifies the responsible component.
8. Put a condition before the action, requirement, or result that depends on it.
9. Use one preferred term for one concept.
10. Do not replace an established term with a synonym for stylistic variety.
11. Preserve identifiers, commands, flags, paths, API names, configuration keys, document types, and literal values exactly.
12. Replace qualitative claims with facts, versions, thresholds, units, or observable outcomes.
13. Mark unsupported technical claims with `[assumption]`.
14. Distinguish current behavior, proposed behavior, planned behavior, and possible future behavior.
15. Distinguish facts, decisions, requirements, examples, and assumptions.
16. Separate instructions, notes, warnings, examples, and explanations.
17. Do not hide required actions inside notes or explanatory paragraphs.
18. Remove filler that adds no architectural, operational, or normative information.
19. Do not invent missing constraints, measurements, behavior, rationale, compatibility, or guarantees.
20. Use visible placeholders when required information is unavailable.

Use placeholders such as:

- `[ACTOR REQUIRED]`
- `[CONDITION REQUIRED]`
- `[METRIC REQUIRED]`
- `[LIMIT REQUIRED]`
- `[EVIDENCE REQUIRED]`

## Terminology

Maintain stable terminology within a file and across related Archcore documents.

- Use the canonical repository term for each concept.
- Define an unfamiliar term before relying on it.
- Do not use the same term for different concepts.
- Do not translate or paraphrase code identifiers.
- Avoid ambiguous pronouns when more than one referent is possible.
- When two existing terms might represent the same concept, surface the conflict instead of silently normalizing it.

Use these repository terms consistently:

- Archcore CLI
- Archcore MCP server
- MCP tool
- document type
- relation type
- host wiring
- SessionStart hook
- managed block
- local document
- global source
- project context
- `.archcore/`

## Procedures and guides

Apply these rules to installation instructions, migrations, runbooks, recovery procedures, troubleshooting, and numbered workflows.

1. State prerequisites before the procedure.
2. State required inputs before the procedure.
3. Use the imperative form for actions.
4. Put one primary action in each numbered step.
5. Put the condition before the step that it controls.
6. Put a warning before a hazardous, destructive, or irreversible action.
7. Do not put mandatory actions in notes.
8. Separate alternative flows into branches or subsections.
9. Preserve commands exactly as the reader must enter them.
10. State the expected result after important verification points.
11. Include rollback or recovery instructions only when repository evidence supports them.
12. Keep background explanation outside the numbered steps.
13. Name the supported platform or host before platform-specific instructions.
14. Do not present unsupported installation methods as alternatives.

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

## Requirements, rules, and specifications

Apply these rules to `rule`, `spec`, `brs`, `strs`, `syrs`, `srs`, contracts, invariants, acceptance criteria, MCP contracts, and CLI behavior contracts.

1. Put one requirement in each numbered item.
2. Use one normative modal in each requirement.
3. State the obligated actor explicitly.
4. Put the trigger or condition before the obligation.
5. Use `MUST`, `MUST NOT`, `SHOULD`, or `MAY` in uppercase for normative meaning.
6. Use `MUST` only for behavior required for correctness, interoperability, safety, or an accepted repository constraint.
7. Make each requirement objectively verifiable.
8. Give requirements stable identifiers when traceability is needed.
9. State measurable limits when repository evidence provides them.
10. Do not use open-ended lists such as `etc.` in normative statements.
11. Do not hide requirements in rationale, examples, headings, or notes.
12. Do not combine independent obligations with `and` or `or`. Split them into separate requirements.
13. Preserve the notation and mandatory sections defined by the relevant Archcore document type.
14. State error behavior and observable recovery behavior when the subject can fail.
15. State path-safety, filesystem-safety, and information-disclosure constraints explicitly when they apply.

Prefer these forms:

- `The <actor> MUST <response>.`
- `WHEN <trigger>, the <actor> MUST <response>.`
- `WHILE <state>, the <actor> MUST <response>.`
- `IF <undesired condition>, THEN the <actor> MUST <response>.`

## Architecture, decisions, and explanatory documents

Apply these rules to ADRs, RFCs, architecture documents, PRDs, plans, research documents, reference documents, and README sections.

1. Do not force imperative or normative phrasing into descriptive content.
2. State the purpose, conclusion, or decision before supporting detail.
3. Separate context from the decision or proposal.
4. Separate mechanism from rationale.
5. Separate benefits from verified outcomes.
6. Identify trade-offs, limitations, compatibility effects, and operational consequences.
7. Label examples as non-normative examples.
8. Use headings that describe the reader's question or task.
9. Keep paragraphs short enough to expose the logical structure.
10. Reference source files with `@path/to/file` instead of reproducing implementation bodies.
11. Distinguish the CLI surface, MCP surface, hook surface, and internal package behavior.
12. State whether behavior is current, deprecated, planned, or unsupported.

## CLI help and errors

For CLI help text:

- Begin with the action that the command performs.
- State required arguments before optional arguments.
- Use the exact command and flag names.
- Put platform-specific differences in separate sections.
- Include one canonical example when it removes ambiguity.
- Do not list unsupported or historical installation methods as current options.

For error messages:

- State what failed.
- State the affected object or input.
- State the next supported action when one exists.
- Do not expose absolute filesystem paths, secrets, tokens, or internal implementation details.
- Do not use blame-oriented language.
- Do not claim that an operation succeeded when only part of it succeeded.

## MCP tools and prompts

When writing MCP tool descriptions, schemas, prompts, and server instructions:

- Name the tool action in the first sentence.
- State preconditions before the action.
- State write boundaries explicitly.
- State path and input validation constraints explicitly.
- State whether an operation is read-only or mutating.
- State the observable result.
- State partial-failure behavior when multiple operations can succeed independently.
- Use the same tool name and argument name everywhere.
- Do not rely on examples to define required behavior.
- Keep one required action in each numbered instruction.

## Agent instructions

When writing instructions for Claude Code, Codex CLI, Cursor, Copilot, Gemini CLI, or another agent:

- Write instructions as direct actions.
- Put routing conditions before the routed action.
- Keep one required action in each numbered instruction.
- Name the tool, file, document type, host, or state explicitly.
- Separate mandatory behavior from rationale.
- State exceptions immediately after the rule they modify.
- Do not rely on an instruction implied only by an example.
- Preserve Archcore workflow and document-type terminology.
- Do not duplicate detailed contracts that can be referenced from one canonical file.

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
- The purpose or result appears near the beginning.
- Terminology is consistent.
- Identifiers and literal values are unchanged.
- Every normative statement has an explicit actor.
- Every requirement contains one obligation and one modal.
- Every procedural step contains one primary action.
- Conditions appear before dependent actions.
- Required actions are not hidden in notes.
- Qualitative claims have evidence, a measurement, or `[assumption]`.
- Facts, proposals, assumptions, and examples are distinguishable.
- No unsupported behavior, compatibility statement, or guarantee was introduced.
- The text follows the relevant Archcore document type and accepted repository rules.

Revise known violations before returning the text. Do not include the checklist or a writing-quality score unless the user asks for a review report.

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
