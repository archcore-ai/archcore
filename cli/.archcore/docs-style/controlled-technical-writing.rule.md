---
title: "Controlled Technical Writing — Repository Scope and Enforcement"
status: accepted
tags:
  - "docs-style"
---

## Rule

Two shared rules in the mounted `archcore` global source own the writing policy. `concepts/controlled-technical-writing` carries the profile — the sentence contract for normative documents, the procedure obligations, the evidence obligations, and the precedence order. `concepts/document-prose-canon` carries the assignment — which profile, line format, and metric each document type takes. This rule states only what is specific to this repository.

1. WHEN an agent creates or updates a Markdown file under `.archcore/**`, `README.md`, `docs/**`, `templates/**`, Markdown embedded in Go source, CLI help text, a CLI error message, an MCP tool description, an MCP prompt, or an agent instruction file, the agent MUST apply the shared profile and the prose canon. `@AGENTS.md` carries the profile text for a host that reads an instruction file rather than the document graph.
2. WHEN the shared profile's precedence order reaches the document-type contract level, the agent MUST resolve that level in this repository to the type template in `@templates/templates.go`, then to the section contract in `@templates/precision.go`.
3. WHEN the shared profile requires a visible placeholder for information this repository does not support, the author MUST use one of `[ACTOR REQUIRED]`, `[CONDITION REQUIRED]`, `[METRIC REQUIRED]`, `[LIMIT REQUIRED]`, or `[EVIDENCE REQUIRED]`.
4. WHEN the prose canon changes an assignment, the maintainer MUST carry the change into `@templates/precision.go` and `@templates/templates.go`. This repository is where the canon reaches a project that installs no runtime.
5. The author MUST NOT restate in this file an obligation that the shared profile or the prose canon already carries.

## Rationale

Requirement 5 is the point of this file. This repository and the plugin repository each held a full copy of the profile under the same filename, and the copies had already diverged on the precedence order, on the covered document types, and on whether a repository may claim conformance to an external writing standard. The split between the two profiles was then stated a third time in `@CLAUDE.md`, with a fourth wording in the plugin's instruction file. One policy with four owners produces four policies.

Requirement 4 exists because the engine is the delivery mechanism. The global source is mounted by the two tool repositories only; a consumer project receives the canon through the document templates and the post-write checks this repository ships.

Requirement 2 exists because the templates and the section contract define structure that the general profile does not describe, so the general profile MUST NOT displace them.

## Examples

Non-normative examples.

### Good

```markdown
WHEN `sync` receives a non-2xx response, the CLI MUST leave the manifest unchanged.
```

A `guide` in this repository states a step in the imperative and keeps it under 20 words, because the prose canon assigns `guide` the STE profile and format F3.

### Bad

```markdown
## Rule
3. In the normative section of a `rule`, the author MUST state one obligation
   in each numbered item.
```

The shared profile already carries that obligation. Restating it here creates the second copy requirement 5 forbids, and the copy is what drifts.

## Enforcement

- `@CLAUDE.md` routes agents to `@AGENTS.md` before any documentation edit.
- The post-write precision check in `@internal/advisory/precision.go` measures a document against the canon data in `@templates/precision.go`. The check reports and never blocks. One report carries at most 12 findings and ends with a count of what the cap dropped.
- Automated coverage is partial by design. The forbidden lexicon, the heading set per type, the line format per type, and the body metrics are measurable; actor visibility in prose, evidence, and tense are review-time checks.
- Code review rejects a document that introduces unsupported behavior, a compatibility statement, or a guarantee.
