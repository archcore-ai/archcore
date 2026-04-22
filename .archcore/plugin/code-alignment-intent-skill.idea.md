---
title: "Code-Oriented Intent Skill — /archcore:align"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Idea

Add a new Layer 1 intent skill `/archcore:align` that takes a code area as argument (file path, directory, component name, or feature scope) and returns the applicable constraints from the knowledge base: ADRs that govern the area, rules that apply, specs the code must conform to, and cpats (before/after patterns) it should follow.

This is the user-pull counterpart to the automatic push mechanism proposed in `pre-code-context-injection.idea.md`. The hook is invisible and unavoidable; the skill is explicit and scoped. Both serve the same JTBD #1 — ensure code changes respect existing architecture and decisions — but at different moments in the workflow.

### Why a new intent skill, and not an extension of `/archcore:capture` or `/archcore:review`

- `/archcore:capture` creates documents _about_ code. `/archcore:align` reads documents _for_ code. Opposite direction.
- `/archcore:review` audits documentation health. It operates on the doc graph, not on code paths.
- `/archcore:actualize` detects when documentation has fallen behind code changes. It is a freshness-reporter, not a constraint-resolver.

`/archcore:align` fills an empty slot: the code-centric read operation. All 9 existing intents are document-centric.

### Routing table

| Signal                                  | Route                         | Scope                            |
| --------------------------------------- | ----------------------------- | -------------------------------- |
| File path (`src/api/handlers/users.ts`) | → path-match resolver         | Documents referencing the path   |
| Directory (`src/payments/`)             | → prefix-match resolver       | Documents referencing the dir    |
| Component name (`AuthMiddleware`)       | → symbol search + path-match  | Documents referencing the symbol |
| Feature scope (`"auth redesign"`)       | → tag match + content search  | Tagged docs + text match         |
| No arguments                            | → ask one clarifying question | —                                |

Output ranking matches the hook: specificity → type priority (`rule` > `adr` > `spec` > `cpat` > `guide`) → recency.

### Execution

```
1. Parse $ARGUMENTS → determine scope type (path, dir, symbol, feature)
2. Query via MCP tools:
   - list_documents
   - list_relations
   - For matched docs: get_document to extract relevant excerpts
3. Rank and format:
   ## Rules that apply
   - rule:api-handlers-layout — "Handlers live in src/api/handlers/, one per resource"
   - rule:money-arithmetic — "All monetary math uses BigDecimal, never float"

   ## Decisions that govern this area
   - adr:rest-conventions — "We use REST, not RPC-over-HTTP" (accepted 2026-01-15)
   - adr:postgres-for-transactions — "Transactional writes go through Postgres, not Mongo"

   ## Specs the code must conform to
   - spec:payment-flow — "Payments contract, see §3.2 for idempotency"

   ## Patterns to follow
   - cpat:handler-error-wrapping — "Use withErrorBoundary(), not try/catch"

   ## Related in the graph
   - plan:auth-redesign — in progress, touches this area
4. Offer follow-up:
   - "Show full content of any of these?"
   - "Start implementation with these constraints loaded?"
```

### Frontmatter

```yaml
---
name: align
argument-hint: "[file path, directory, component, or feature]"
description: |
  Load the rules, ADRs, specs, and patterns that apply to a code area before changing it.
  Activate when user says "what rules apply to X", "before I refactor Y", "prepare to edit Z",
  "what should I know before touching payments".
  Do NOT activate for creating docs (use /archcore:capture or /archcore:decide).
  Do NOT activate for audits (use /archcore:review or /archcore:actualize).
---
```

No `disable-model-invocation` flag — this is Layer 1 and must be auto-invocable per the Inverted Invocation Policy. Routing-sensitive trigger phrases in the description.

## Value

### Complements the hook without duplicating it

The hook injects context at the moment of `Write|Edit`. The skill loads context deliberately, before the user even asks the agent to code. Two different moments, two different user intents, same underlying index.

### Gives users an explicit pull lever

Some users want to see the constraints before committing to a change: "I'm thinking about refactoring payments — what applies?" The hook answers this only after the agent starts writing, which is too late for humans doing exploratory thinking.

### Makes the knowledge base queryable by code location

Today the graph is queryable by document type, tag, and relation. It is not queryable by code path. This skill fills that missing axis.

### Natural first-60-seconds demo

The hero prompt becomes: "What rules apply to my src/api/handlers/?" → agent returns the list → obvious value. This works today as a read-only operation, without shipping the hook — so the skill can precede the hook in rollout.

## Possible Implementation

### Phase 1 — Path-match resolver (1–2 days)

- New `skills/align/SKILL.md` at 100–200 lines per Layer 1 conventions
- Executes via existing MCP tools: `list_documents`, `get_document`, `list_relations`
- Grep-based path resolution via CLI or via agent's Grep tool — either works for v1
- Output format fixed by the template above

### Phase 2 — Symbol and feature resolvers (2–3 days)

- Symbol search via Grep across codebase for the identifier, then path-match on hit locations
- Feature search via tag match + content search across documents
- Disambiguation question when multiple resolvers apply

### Phase 3 — Index integration (1 day after pre-code hook Phase 2)

- Reuse the path index from `pre-code-context-injection.idea.md` Phase 2 — one shared index, two consumers
- Replaces grep with index lookup for the path and prefix cases

### Spec updates

- `skills-system.spec.md`: intent-skill count becomes 10, update invariants
- `plugin-architecture.spec.md`: update the Layer 1 component inventory
- `inverted-invocation-policy.adr.md`: extend the invocation matrix row for intent to include `align`

## Risks and Constraints

- **Skill count creep.** Layer 1 currently has 9 intents; adding a 10th is fine, but the constraint "maximum 9 intent skills" in `plugin-architecture.spec.md` must be revised explicitly, not silently.
- **Duplicate surface with the hook.** If both ship, users get the same context twice (once from the skill, once from the hook when they actually write). Mitigation: session-level de-duplication in the hook (do not re-inject documents already shown this session) — referenced in the hook idea's Phase 3.
- **Overlap with `/archcore:capture`.** Users might confuse them. The routing description must be explicit: `capture` is for _writing_ docs about code, `align` is for _reading_ docs for code. Trigger-phrase discipline required.
- **Empty-result UX.** If no documents apply to a given code area, the skill must return something useful: "No documented rules or decisions cover this area. Consider `/archcore:capture` to document it." Avoid looking broken on greenfield areas.
- **Prompt routing precision.** "Before I refactor X" is ambiguous — could route to `align` (load constraints) or to a track skill (plan the refactor). Anti-trigger should explicitly say: `align` is read-only; use `/archcore:architecture-track` for a planned flow.
- **Content truncation.** Returning full document bodies explodes output length on well-documented areas. Truncate to one-liner summaries + "/archcore:adr <name> for full content" follow-up. Full content is one get_document away.
- **Layer 1 line limit.** Layer 1 intent skills must not exceed 300 lines. The routing and execution logic for 4 resolvers (path, dir, symbol, feature) fits comfortably.

## Related work in this repo

- `jtbd-alignment-analysis.idea.md` — lists this skill as one of the three mechanisms that close the JTBD #1 gap
- `pre-code-context-injection.idea.md` — push counterpart; this skill is the pull counterpart
- `plugin-architecture.spec.md` — intent-skill count and Layer 1 inventory need update
- `skills-system.spec.md` — Layer 1 definition extends to a 10th skill
- `inverted-invocation-policy.adr.md` — invocation matrix extends
