---
title: "Controlled Technical Writing — Binding Constraints and Precedence"
status: accepted
tags:
  - "plugin"
  - "precision"
  - "rule"
---

## Rule

1. WHEN an agent creates or updates a Markdown file under `.archcore/**`, `plugins/archcore/skills/**`, `plugins/archcore/agents/**`, `plugins/archcore/copilot-agents/**`, `plugins/archcore/rules/**`, `docs/**`, or `README.md`, the agent MUST apply the controlled technical writing policy in `@AGENTS.md`.
2. IF two instruction sources conflict for a file in the scope of item 1, THEN the agent MUST resolve the conflict in this order: explicit user requirement, the Archcore type contract in `@plugins/archcore/skills/_shared/`, `@plugins/archcore/skills/_shared/precision-rules.md`, `@AGENTS.md`, general stylistic preference.
3. In the normative section of a `rule`, `spec`, `brs`, `strs`, `syrs`, or `srs` document, the author MUST state one obligation in each numbered item.
4. In the normative section of a `rule`, `spec`, `brs`, `strs`, `syrs`, or `srs` document, the author MUST use one uppercase modal — `MUST`, `MUST NOT`, `SHOULD`, or `MAY` — in each numbered item.
5. In the normative section of a `rule`, `spec`, `brs`, `strs`, `syrs`, or `srs` document, the author MUST name the obligated actor in each numbered item.
6. WHEN a numbered requirement depends on a trigger or a state, the author MUST place the trigger or the state before the obligation.
7. In a numbered requirement, the author MUST NOT use an open-ended list marker such as `etc.`.
8. The author MUST NOT place a required action only in a note, a rationale paragraph, a heading, or an example.
9. In a `guide` document, the author MUST state prerequisites and required inputs before the first numbered step.
10. In a `guide` document, the author MUST put one primary action in each numbered step.
11. In a `guide` document, the author MUST place a warning before the hazardous or destructive action that the warning guards.
12. WHEN an agent applies this policy to an existing file, the agent MUST preserve code identifiers, paths, commands, configuration keys, API names, flags, and literal values unchanged.
13. IF required information is unavailable, THEN the author MUST insert a visible placeholder — `[ACTOR REQUIRED]`, `[CONDITION REQUIRED]`, `[METRIC REQUIRED]`, `[LIMIT REQUIRED]`, or `[EVIDENCE REQUIRED]` — in place of the missing constraint, measurement, behavior, rationale, or guarantee.
14. The author MUST NOT invent a constraint, a measurement, a behavior, a rationale, or a guarantee that repository evidence does not support.
15. The author MUST NOT include the `@AGENTS.md` review checklist in a generated document.
16. IF the user does not request a review report, THEN the author MUST NOT include a writing-quality score in a generated document.

## Rationale

`@AGENTS.md` defines the profile; this rule states which parts bind inside `.archcore/` and how conflicts resolve. `@plugins/archcore/bin/check-precision` already applies the one-modal check (7b) and the active-voice-subject check (7c) to numbered lines in `spec` bodies; items 3–8 extend the same sentence contract to every normative type by review. Item 2 exists because `@plugins/archcore/skills/_shared/spec-contract.md` and `@plugins/archcore/skills/_shared/rule-contract.md` define mandatory sections that `@AGENTS.md` does not describe, so the general policy MUST NOT displace them.

## Examples

### Good

```markdown
## Rule
3. WHEN `archcore init` finds an existing `.mcp.json`, the CLI MUST merge the
   `archcore` server key without rewriting the other keys.
4. IF the merge fails, THEN the CLI MUST exit non-zero.
5. IF the merge fails, THEN the CLI MUST leave `.mcp.json` unchanged.
```

```markdown
## Prerequisites
- Go 1.24 or later on `PATH`.
- A clean worktree — `git status --porcelain` returns no output.

## Procedure
1. Run `make build`.
2. Run `make verify`.
   Expected result: the command exits 0 and prints no `ERROR` line.

Warning: step 3 deletes `.archcore/.sync-state.json` and cannot be undone.
3. Run `rm .archcore/.sync-state.json`.
```

### Bad

```markdown
## Rule
3. The MCP config should be merged carefully and the CLI must not break other
   keys, fail silently, etc.
```

Item 3 above names no actor for the first clause, carries two obligations in one
item, mixes `should` and `must` in one statement, ends with an open-ended `etc.`,
and leaves the failure path unstated.

```markdown
## Procedure
1. Run `make build` and then `make verify`, but note that you need Go 1.24 and a
   clean worktree first.
```

Step 1 above carries two actions, hides both prerequisites inside a note, and
places the prerequisites after the step that depends on them.

## Enforcement

- `@plugins/archcore/bin/check-precision` (PostToolUse, soft mode, always exits 0) checks the forbidden lexicon, mandatory sections by type, frontmatter title and status, body length, cross-document body references, multi-line code blocks in architect-voice types, and — for `spec` bodies only — BCP 14 modals, the 80-line body cap, one modal per numbered line, and an active-voice obligated subject.
- Items 3–16 outside `spec` bodies: manual review. No hook checks them.
- The authoring agent applies the review checklist in `@AGENTS.md` before returning a document.
