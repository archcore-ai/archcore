---
title: "search_documents Is a Matching Primitive, Not a Presentation Tool"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Context

When adding the `search_documents` MCP tool, the question arose: should it also take opinions about presentation — group results into categories (rules / decisions / specs / patterns / in-progress), apply top-N per category, classify the caller's input mode (path vs. topic vs. pickup), emit pre-formatted user-facing output?

Two qualitatively different capabilities are mixed in any "find applicable context" feature:

1. **Matching.** Deterministic, exact, testable. "Which documents contain this path reference or this word?" Owned by code.
2. **Presentation.** Opinionated layout decisions — section grouping, top-N cutoffs, category-to-document-type mapping, empty-state copy, routing of auxiliary document types. Evolves with product usage.

Fusing both into one Go tool would make every UX tweak a CLI release, carry hardcoded opinions that projects may want to override, and leave no raw match list for a hook, a CI check, or another skill.

## Decision

Keep `search_documents` as a matching primitive. Its contract is: given filters, return ranked matches with evidence and manifest relations. Nothing more.

The tool accepts filters and search controls (`sort`, `limit`, `match`, `source`, `mode`). It does NOT classify the caller's input mode, group results into user-facing categories, apply top-N-per-category cutoffs, render markdown or empty-state copy, or route secondary document types into parent categories.

Ranking IS done in the primitive, so callers do not re-sort. Ranking is deterministic and testable; layering it outside Go would push nondeterminism back into the hot path. Callers (agents, hooks, skills, CI scripts) compose the primitive into whatever presentation they need.

### What stays on the matching side

Later additions each land on the matching side of this line. The current shape is owned by @.archcore/mcp/search-documents.spec.md.

- `mode=full` attaches the body with its frontmatter stripped; only the byte budget below can shorten it. It overlaps `get_document` on purpose: one round trip replaces N+1. Its small limit bounds keep the tool from becoming a bulk-export surface. Opinionated formatting in full mode (sectioning, rendering, summarization) WOULD cross this line and needs a new decision.
- `coverage`, `hits`, `index`, `match`, `source`, and per-source representation are matching metadata and search controls. They key on `source_id` and `path`, a matching-layer fact, not on a product taxonomy (@.archcore/mcp/global-recall-guarantees.rfc.md).
- The byte budget shortens a `body` to a raw prefix. The tool adds no marker text, no ellipsis, no summary; `body_truncated` and `body_bytes` sit beside the body so the cut stays machine-checkable. The caps on `matches` and relation arrays are cut concerns, like `limit`. The key order of the envelope is a wire property: one host showed the caller only the first 2 KB of a stored result.
- `near_misses` is matching data. It applies one deterministic rule, at least half of the distinct query words, to documents that pass the filters, and reports the words each row lacks. It carries no prose, retry instruction, grouping, or excerpt, and it never enters `results`. The threshold and the cap are contract values, not presentation settings (@.archcore/mcp/empty-all-words-search-partial-matches.rfc.md).

## Rationale

- **Deterministic where correctness matters.** Regexes, specificity math, UTF-8 safety, and filter composition are code with unit tests. Clever prose in documents cannot trick the tool into false positives.
- **Small, focused Go surface.** A presentation-aware variant would add category routing, relation traversal for guides, empty-state branching, and per-caller shims.
- **Reusability.** A PreToolUse hook can stream the raw list, a CI script can audit accepted rules, and a skill can preview by content. None needs a second tool or a CLI release.
- **No opinion lock-in.** Category layout, type-priority weights, and guide routing may vary per project or change over time. Outside the tool they can live in a per-project skill, a team convention, or a product release.
- **Honest limitations.** The tool description states the matching semantics and that ambient rules without `@path` are not reachable via path mode. Callers surface the limit instead of papering over it.

## Alternatives Considered

### Monolithic `resolve_context` tool

One tool that classifies the scope, searches, groups by type into named sections, handles pickup mode, and routes guides via relations. Not chosen: the type-to-section mapping is the parameter projects most want to customize; every UX change needs a CLI release; a hook wanting a raw list must re-implement it or accept the opinionated shape.

### Primitive with a `category` output field

Tag each result with a category derived from its type. Not chosen: it bakes one product taxonomy into the tool, a CI audit would filter the labels out, the mapping is a trivial lookup that callers can own, and a new type would pick its category for every consumer forever.

### Two tools: `search_documents` and `group_documents`

Not chosen: callers that need grouping call both and reassemble, and grouping is trivial map-lookup code that needs no MCP tool.

## Consequences

### Positive

- Matching and ranking are code with regression tests. Presentation stays with callers.
- Hooks, CI linters, skill variants, and external MCP consumers all call the one primitive.

### Negative

- Callers must understand the contract. The spec is the source of truth; casual consumers may expect prettier output.
- Different consumers may render the same results differently. Consistency, if wanted later, is enforced in downstream tooling, not in the CLI.

### Risks

- Ambient rules without `@path` are not reachable via path mode. This is intentional (determinism over recall), but users may read it as broken. The tool description and spec say so.
- Content matching never folds morphology: `rounding` does not find `Arithmetic`, though the default all-words rule finds `money arithmetic`.

## References

- Tool contract: @.archcore/mcp/search-documents.spec.md
- Implementation: @cli/internal/mcp/tools/search_documents.go
- Related MCP decisions: @.archcore/mcp/no-list-tags-tool.adr.md, @.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md
