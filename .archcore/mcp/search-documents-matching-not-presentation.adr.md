---
title: "search_documents Is a Matching Primitive, Not a Presentation Tool"
status: accepted
tags:
  - "mcp"
---

## Context

When adding the `search_documents` MCP tool, the question arose: should it also take opinions about presentation — group results into categories (rules / decisions / specs / patterns / in-progress), apply top-N per category, classify the caller's input mode (path vs. topic vs. pickup), emit pre-formatted user-facing output?

### Current State

- `list_documents` filters on frontmatter metadata only. Deterministic, testable, narrow.
- `get_document` fetches one known document. No discovery.
- Agents working on a task run ad-hoc loops of `list_documents` + multiple `get_document` + `grep` to build context. Slow, non-deterministic, token-heavy.

### Problem Statement

Two qualitatively different capabilities are mixed in any "find applicable context" feature:

1. **Matching.** Deterministic, exact, testable. "Which documents contain this path reference or this substring?" Owned by code.
2. **Presentation.** Opinionated layout decisions — section grouping, top-N cutoffs, category-to-document-type mapping, empty-state copy, routing of auxiliary doc types (e.g., guides linked to rules via relations). Evolves with product usage.

Fusing both into a single Go tool would mean every UX tweak (rename a section, add a new document type, change a header) becomes a CLI release. The CLI would also carry hardcoded opinions that projects may want to override. And the search capability would be unreusable by anyone who wants a raw match list for a different purpose (a hook, a CI check, a different skill).

## Decision

Keep `search_documents` as a matching primitive. Its contract surface is: "given filters, return ranked matches with evidence and manifest relations". Nothing more.

Specifically, the tool:

- Accepts filters (`path_ref`, `content`, `types`, `status`, `mtime_after`) and search controls (`sort`, `limit`).
- Returns a ranked, flat JSON array of `searchResult` objects with per-match evidence and enriched manifest relations.
- Does NOT classify the caller's input mode (empty / path / topic).
- Does NOT group results by type into user-facing categories.
- Does NOT apply top-N-per-category cutoffs.
- Does NOT render markdown, headings, or empty-state copy.
- Does NOT route guides or any other secondary doc type into parent categories.

Ranking IS done in the primitive via the `sort` parameter, so callers do not need to re-sort. Ranking is a deterministic, testable concern; layering it outside Go would push nondeterminism back into the hot path.

Callers (agents, hooks, skills, CI scripts) compose `search_documents` into whatever presentation they need. Multiple callers can compose it differently without fighting each other or requiring CLI changes.

### Rationale

- **Deterministic where correctness matters.** Regexes, specificity math, UTF-8 safety, filter composition — all code with unit tests. The tool cannot be "tricked" into false positives by clever prose in documents.
- **Small, focused Go surface.** The primitive is ~300 LOC including helpers. A presentation-aware variant would easily double this with category routing, relation traversal for guides, empty-state branching, and per-caller customization shims.
- **Reusability.** The primitive is general-purpose. A future PreToolUse hook can call `search_documents(path_ref="...", sort="mtime")` and stream the raw list. A CI script can call `search_documents(types=["rule"], status="accepted")` to audit rules. A future skill can call `search_documents(content="...", limit=20)` for semantic preview. None of these require a second Go tool or CLI release.
- **No opinion lock-in.** Category layout, type-priority weights, guide routing via relations — these are legitimate decisions that may vary per project or evolve over time. Keeping them out of the tool means they can live wherever they make sense (per-project skill, team convention, product release).
- **Honest limitations surfaced.** The tool description documents that content search is strict substring and that ambient rules without `@path` are not reachable via path mode. Callers do not silently paper over these limits with fabricated fallbacks — they either surface them to users or extend their own handling.

## Alternatives Considered

### Alternative 1: Monolithic `resolve_context` tool

A single tool that took a scope argument, classified it, ran the search, grouped by type into categories (rules, decisions, specs, patterns, in_progress), handled pickup mode, routed guides via relations, and returned an already-structured JSON with named sections.

**Why not chosen:**

- Type-to-section mapping is exactly the parameter projects are most likely to want to customize or extend.
- Every UX change requires a CLI release.
- No reuse: a hook wanting a raw search list has to either re-implement or accept the opinionated shape.
- Go LOC roughly doubles for opinionated logic.
- The weak axis (section grouping, top-N cutoffs, empty-state copy) lives where it's hardest to iterate.

### Alternative 2: Primitive with "category" output field

Keep the primitive, but have it tag each result with a category string (rules/decisions/specs/...) derived from its type, so callers don't need a mapping table.

**Why not chosen:**

- Bakes in a specific taxonomy at the tool level. "Rules" and "Decisions" and "Patterns" reflect one product framing; a different consumer (CI audit, for instance) has no use for these labels and would filter them out.
- The mapping is trivial (one lookup per type) — duplicating it in callers is cheaper than an inflexible contract.
- Makes future type additions a protocol change: adding a new type means picking its category, and that picks the category for every consumer forever.

### Alternative 3: Two Go tools — `search_documents` + `group_documents`

Split at an even finer grain: the first tool does search, the second tool takes a result set and buckets it.

**Why not chosen:**

- Solves a problem that didn't exist. Callers that need grouping would invoke both tools and reassemble; callers that don't need grouping save one call. The combined cost is the same as the current design; the complexity is higher.
- "Grouping" is trivial map-lookup code. It doesn't need to be an MCP tool.

## Consequences

### Positive

- **Small, testable Go surface.** ~300 LOC, ~22 table-driven test cases. Matching and ranking regression-guarded.
- **Reusable primitive.** Future hooks, CI linters, multiple skill variants, external MCP consumers — all can call `search_documents` directly without a second tool.
- **Determinism where it matters.** Matching and ranking are code and fully tested. Presentation concerns stay with callers.
- **No opinion lock-in at the tool level.** Section layout, type priorities, empty-state copy, guide routing — all caller-owned.
- **Honest limitations documented at the tool.** Callers know strict-substring semantics and `@path`-reachability semantics before writing client code.

### Negative

- **Callers must understand the contract.** The `search_documents.spec.md` is the source of truth. Casual consumers may expect prettier output; they have to read the spec. Acceptable — spec is short and complete.
- **Downstream callers own presentation choices.** Different consumers may diverge in how they render the same search results. If consistency is later desired, it will need to be enforced in downstream tooling, not in the CLI.

### Risks

- **Ambient rules without `@path` are not reachable via path mode.** Intentional (determinism > recall), but users may misread as "broken". Mitigated by explicit mention in the tool description and spec. May later be reinforced by an `archcore doctor` hygiene warning.
- **Content matching is strict substring.** `money rounding` will not find `Money Arithmetic`. Mitigated by explicit mention in the tool description; callers are expected to be honest about empty results.

## Implementation Notes

- Delivered in the initial `search_documents` implementation sprint.
- Ranking is computed in Go via the `sort="relevance"` / `sort="mtime"` parameter. Callers MUST NOT re-sort; the tool's order is authoritative.
- Relations enrichment (`incoming_relations` / `outgoing_relations`) is in-tool so callers avoid N extra `list_relations` calls when traversing relation graphs.
- UTF-8 safety and lazy body loading were added post-review based on code-review feedback; both have regression tests.

### Addendum (2026-06): `mode=full` does not cross the matching/presentation line

A later change added a `mode` parameter (`snippets` default / `full`). In `full` mode each result inlines the matched document's body (frontmatter stripped) so a caller can read the doc without a follow-up `get_document`; full mode carries smaller limit bounds (default 3, max 20) to keep one response token-bounded. This decision is unchanged by that addition, and the boundary is worth restating explicitly:

- **`mode=full` returns raw data, not opinionated layout.** It does none of the things this ADR forbids — no category grouping, no top-N-per-category, no type-to-section mapping, no markdown rendering, no empty-state copy, no guide routing. It simply attaches the unmodified body. "Presentation" here means *opinionated, product-evolving layout decisions*; inlining a raw body is neither opinionated nor likely to churn.
- **It does overlap `get_document` — deliberately.** The "Current State" framing above treated matching (`search_documents`) and single-doc fetch (`get_document`) as cleanly separate. `mode=full` softens that line: discover-and-read can now happen in one call, and the tool description steers callers toward `search_documents(mode=full)` over `search + get_document` when the goal is to read the matches. This was accepted as a token-efficiency convenience (one round-trip instead of N+1), not a re-architecture — `get_document` remains the right tool when the caller already knows the exact path and needs the relation graph for a single doc.
- **Bounds protect the primitive's character.** The small full-mode limits keep the tool from becoming a bulk-export surface; it stays a *matching* tool that can optionally hand back the matched bodies, not a documents dump.

If full mode ever grows opinionated formatting (sectioning, rendering, summarization), that WOULD cross this ADR's line and should be revisited as a new decision.

## References

- Tool contract: `.archcore/mcp/search-documents.spec.md`
- Implementation: @internal/mcp/tools/search_documents.go
- Related MCP decisions: `.archcore/mcp/no-list-tags-tool.adr.md`, `.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md`
