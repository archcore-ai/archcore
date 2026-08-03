---
title: "Controlled Technical Writing Policy for Repository Documentation"
status: accepted
tags:
  - "docs-style"
---

## Rule

`@AGENTS.md` is the canonical writing policy for this repository. This rule states the obligations that apply to `.archcore/` documents and points to that file; it does not restate the policy.

1. WHEN an author creates or updates a document under `.archcore/`, the author MUST apply the writing policy in `@AGENTS.md`.
2. The author MUST apply the same policy to `README.md`, `docs/**/*.md`, `templates/**/*.md`, Markdown embedded in Go source, CLI help text, CLI error messages, MCP tool descriptions, MCP prompts, and agent instruction files.
3. The author MUST NOT state or imply that this repository complies with ASD-STE100, ISO 24495-1, or any other external standard. The policy is an internal writing profile.
4. WHEN instructions conflict, the author MUST follow this precedence: explicit user requirements, accepted `.archcore/` rules and decisions, document-type contracts and templates, the repository writing policy, general stylistic preferences.
5. The author MUST preserve Go identifiers, package names, commands, flags, paths, configuration keys, MCP tool names, JSON fields, document type names, and literal values exactly.
6. IF a technical claim has no repository evidence, THEN the author MUST mark it with `[assumption]` or replace it with a visible placeholder such as `[EVIDENCE REQUIRED]`.
7. WHEN an author writes a normative document (`rule`, `spec`, `brs`, `strs`, `syrs`, `srs`), the author MUST put one obligation, one uppercase BCP 14 modal, and one explicit obligated actor in each numbered requirement.
8. WHEN an author writes a descriptive document (`adr`, `rfc`, `doc`, `guide`, `prd`, `idea`, `plan`, `rnd`), the author MUST NOT force normative modals into content that records context, rationale, or exploration.
9. WHEN an author writes a procedure, the author MUST state prerequisites and inputs before the steps, and MUST put one primary action in each numbered step.
10. WHEN a condition controls an action, the author MUST state the condition before the action.
11. The author MUST reference implementation files with `@path/to/file` instead of reproducing implementation bodies.
12. The author MUST state whether described behavior is current, deprecated, planned, or unsupported.
13. The author MUST write repository documentation in English unless the user or the existing document requires another language.
14. The author MUST NOT edit a mounted global source and MUST NOT create a relation to one.

## Rationale

Two readers consume these documents: engineers and AI coding agents. Both fail the same way on ambiguous text — they guess the actor, the trigger, or whether a sentence records a fact, a decision, or a wish. A single writing profile, referenced from one file, keeps the wording of a `rule` verifiable and keeps an `adr` readable as a record of reasoning.

Keeping the full policy in `@AGENTS.md` and only the obligations here prevents two copies of the same contract from drifting apart.

## Examples

Non-normative examples.

### Good

```markdown
WHEN `sync` receives a non-2xx response, the CLI MUST leave the manifest unchanged.
```

```markdown
`init` writes the instruction file for each detected agent. Current behavior; see `@internal/agents/instructions.go`.
```

### Bad

```markdown
The manifest should probably not be updated if something goes wrong, and the CLI must also
log the error and skip the remaining documents.
```

```markdown
The sync engine is fully standards-compliant and very fast.
```

## Enforcement

- `@CLAUDE.md` routes agents to `@AGENTS.md` before any documentation edit.
- `@AGENTS.md` carries the review checklist that an author verifies before returning text.
- Code review rejects documents that introduce unsupported behavior, compatibility statements, or guarantees.

## References

- `@AGENTS.md` — the canonical writing policy
- `@CLAUDE.md` — agent routing and Archcore operation rules
