---
title: "SessionStart Globals Disclosure"
status: accepted
tags:
  - "cli"
  - "globals"
  - "integrations"
---

## Purpose &amp; Scope

This specification defines the `GLOBALS` block: a bounded, zero-read summary of every declared global source in the SessionStart context. It closes the disclosure half of the discovery gap recorded in @.archcore/globals/global-discovery-gap.idea.md.

The behavior below is current, implemented in `buildSessionContext` (@cmd/hooks_common.go) and `InspectGlobals` (@internal/docs/inspect.go).

Out of scope: read-tool paging and ranking (@.archcore/mcp/global-recall-guarantees.rfc.md), the MCP server instructions paragraph, and any content-derived summary (global tags or titles — rejected by the cost analysis in the idea).

## Surface

`GlobalInspection` carries two fields derived during the walk `countGlobalDocs` already performs: `DocsByCategory` (per virtual category counts, from the filename type suffix) and `TopDirs` (top-level directory names with document counts).

Block format, with this repository's measured data:

```
GLOBALS (read-only, query via MCP read tools):
  - archcore — 42 docs (knowledge 40, vision 1, experience 1) · product/ 14, concepts/ 14, architecture/ 7, market/ 4, web/ 3
  ⚠ global source "company" not found at "../company/.archcore" — clone it or fix .archcore/settings.json
  Local documents take precedence over same-topic globals.
```

## Normative Behavior

1. WHEN `settings.json` declares at least one global source, the builder MUST render a `GLOBALS` block after the `BRANCH` line, or after `CORPUS` when `BRANCH` is absent.
2. WHEN no global source is declared, the builder MUST NOT render the block or its heading.
3. The builder MUST render one line per declared source, in declaration order.
4. WHEN a source is `GlobalOK`, the builder MUST render its id, its document count, and its per-category counts on the line.
5. The builder MUST list a source's top-level directories with document counts, ordered by count descending, ties alphabetical.
6. The builder MUST derive every count from filenames and directory names alone.
7. The builder MUST NOT read the content of any global document.
8. The builder MUST render at most 6 directories per source.
9. WHEN directories are dropped, the builder MUST name the dropped count on the line.
10. The builder MUST render at most 8 source lines.
11. WHEN sources are dropped, the builder MUST name the dropped count below the last line.
12. WHEN a source is in a fatal or empty state, the builder MUST render its inspection message inside the block, prefixed with "⚠".
13. WHEN at least one source line renders, the builder MUST append the sentence "Local documents take precedence over same-topic globals."
14. WHEN the block renders, the builder MUST label the `CORPUS` count "local documents".
15. WHEN the block holds at least one `GlobalOK` source, the builder MUST append the total global document count to the connected banner.
16. The builder MUST reuse the `InspectGlobals` walk for every count.
17. The builder MUST NOT add a corpus scan to the session start.
18. IF `config.LoadGlobals` fails, THEN the builder MUST render only the invalid-settings warning.

## Constraints &amp; Invariants

- Output ceiling: 8 source lines, their warnings, the heading, and the precedence sentence. Block size is a function of the ceilings, never of corpus size.
- I/O ceiling: directory walks only, identical to the cost `InspectGlobals` pays today (@internal/docs/inspect.go `countGlobalDocs`).
- No global document path, title, or tag appears in the block; global content surfaces through the MCP read tools only (@.archcore/globals/global-sources.spec.md).
- No absolute filesystem path appears in any line; a line carries the declared id and the declared path only (@.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md).
- Category derivation matches the scan: filename type suffix to virtual category (@templates/templates.go `CategoryForType`).
- The governing documents agree with this contract: the invariant and warning placement in @.archcore/globals/global-sources.spec.md, the section order in @.archcore/integrations/session-start-context.spec.md, and clauses 6–7 of @.archcore/globals/globals-are-read-only-everywhere.rule.md were amended together with this specification's acceptance.

## Failure Behavior

| Condition | Response |
| --- | --- |
| Invalid `settings.json` | invalid-settings warning; no `GLOBALS` block |
| Fatal source (missing, not a directory, unreadable, self-overlap, duplicate) | "⚠" line inside the block; remaining sources still render |
| Empty source | "⚠" contains-no-documents line inside the block |
| Unreadable subdirectory during the count | source classified unreadable; "⚠" line (existing `InspectGlobals` behavior) |
| No declared globals | no block, no heading, `CORPUS` label unchanged |

## Conformance

An implementation conforms when it satisfies clauses 1–18 and the failure rows, and `TestBuildSessionContext_ScansTheCorpusOnce` (@cmd/hook_scan_budget_test.go) still passes.

The conformance tests are @cmd/hooks_globals_block_test.go: block presence and absence, both ceilings with named drops, the warning merge, the precedence sentence, the `CORPUS` label, and the banner suffix.