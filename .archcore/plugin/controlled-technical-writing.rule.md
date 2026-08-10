---
title: "Controlled Technical Writing — Repository Scope and Enforcement"
status: accepted
tags:
  - "plugin"
  - "precision"
  - "rule"
---

## Rule

The writing profile itself is the shared rule `concepts/controlled-technical-writing` in the mounted `archcore` global source. It carries the sentence contract for normative documents, the procedure obligations, the evidence obligations, and the precedence order. This rule states only what is specific to this repository.

1. WHEN an agent creates or updates a Markdown file under `.archcore/**`, `plugins/archcore/skills/**`, `plugins/archcore/agents/**`, `plugins/archcore/copilot-agents/**`, `plugins/archcore/rules/**`, `docs/**`, or `README.md`, the agent MUST apply the shared writing profile. `@AGENTS.md` carries the profile text for hosts that read an instruction file directly rather than the document graph.
2. WHEN the shared profile's precedence order reaches the document-type contract level, the agent MUST resolve that level in this repository to the per-type contracts in `@plugins/archcore/skills/_shared/`, then to `@plugins/archcore/skills/_shared/precision-rules.md`.
3. WHEN the shared profile requires a visible placeholder for information the repository does not support, the author MUST use one of `[ACTOR REQUIRED]`, `[CONDITION REQUIRED]`, `[METRIC REQUIRED]`, `[LIMIT REQUIRED]`, or `[EVIDENCE REQUIRED]`.
4. The author MUST NOT restate in this file an obligation the shared profile already carries.

## Rationale

Requirement 4 is the point of this file. This repository and the CLI repository each held a full copy of the profile under the same filename, and the copies had already diverged on the precedence order, on which document types the sentence contract binds, and on whether a repository may claim conformance to an external writing standard. One profile with two owners produces two profiles.

Requirement 2 exists because `@plugins/archcore/skills/_shared/spec-contract.md` and `@plugins/archcore/skills/_shared/rule-contract.md` define mandatory sections the general profile does not describe, so the general profile MUST NOT displace them.

Requirement 1 keeps `@AGENTS.md` in the loop deliberately. A host that reads an instruction file but not the document graph still needs the profile, so the text is mirrored there for delivery, not for authority.

## Examples

### Good

A `spec` in this repository omits a section the general profile never mentions, because `spec-contract.md` mandates it and requirement 2 puts the type contract above the profile.

### Bad

```markdown
## Rule
3. In the normative section of a `rule`, the author MUST state one obligation
   in each numbered item.
```

The shared profile already carries that obligation. Restating it here creates the second copy requirement 4 forbids, and the copy is what drifts.

## Enforcement

- The precision check runs inside the CLI binary since v0.7.0 (`cli-owns-layers-4-5.adr`). This repository ships no `check-precision` script; `@plugins/archcore/bin/post-tool-use` is host glue that delegates the raw payload to `archcore hooks <host> post-tool-use` and fails open below CLI 0.7.0.
- The check covers the forbidden lexicon, mandatory sections by type, frontmatter, body length, and — for `spec` bodies — BCP 14 modals, one modal per numbered line, and an active-voice obligated subject. Everything else in the shared profile rests on review.
- The authoring agent applies the review checklist in `@AGENTS.md` before returning a document.
