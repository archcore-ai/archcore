---
title: "SessionStart Globals Disclosure"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "globals"
  - "integrations"
---

## Purpose & Scope

This specification defines the `GLOBALS` block: a bounded, zero-read summary of every declared global source in the SessionStart context. It closes the disclosure half of the discovery gap that @.archcore/mcp/global-recall-guarantees.rfc.md records.

It is implemented in `buildSessionContext` (@cli/cmd/hooks_common.go) and `InspectGlobals` (@cli/internal/docs/inspect.go).

Out of scope: read-tool paging and ranking (@.archcore/mcp/global-recall-guarantees.rfc.md), the MCP server instructions paragraph, and any content-derived summary (global tags or titles — rejected on cost).

## Surface

Block format:

```
GLOBALS (read with this project; write a source's documents through its project):
  - archcore — 42 docs (knowledge 40, vision 1, experience 1) · product/ 14, concepts/ 14 · project="/home/dev/archcore"
  ⚠ global source "company" not found at "../company/.archcore" — skipped until it is cloned; local documents are unaffected
  Local documents take precedence over same-topic globals; a matching global is still part of the answer.
```

## Normative Behavior

1. WHEN `settings.json` declares at least one global source, the builder MUST render a `GLOBALS` block after the `BRANCH` line, or after `CORPUS` when `BRANCH` is absent.
2. WHEN no global source is declared, the builder MUST NOT render the block or its heading.
3. The builder MUST render one line per declared source, in declaration order.
4. WHEN a source is `GlobalOK`, the builder MUST render its id, its document count, and its per-category counts on the line.
5. The builder MUST list a source's top-level directories with document counts, ordered by count descending, ties alphabetical.
6. The builder MUST derive every count from filenames and directory names alone.
7. The builder MUST NOT read the content of any global document.
8. The builder MUST render at most 6 directories per source, and at most 8 source lines.
9. WHEN directories or sources are dropped, the builder MUST name the dropped count.
10. WHEN a source is in a fatal, missing, or empty state, the builder MUST render its inspection message inside the block, prefixed with "⚠".
11. WHEN at least one source line renders, the builder MUST append the sentence "Local documents take precedence over same-topic globals; a matching global is still part of the answer."
12. WHEN the block renders, the builder MUST label the `CORPUS` count "local documents".
13. WHEN the block holds at least one `GlobalOK` source, the builder MUST append the total global document count to the connected banner.
14. The builder MUST reuse the `InspectGlobals` walk for every count and MUST NOT add a corpus scan to the session start.
15. IF `config.LoadGlobals` fails, THEN the builder MUST render only the invalid-settings warning.
16. WHEN a source is missing, the builder MUST end its line with "skipped until it is cloned; local documents are unaffected".
17. WHEN a `GlobalOK` source resolves to a directory named `.archcore`, the builder MUST end its line with ` · project="<root>"`, where `<root>` is the parent of that directory — the `project` argument that writes the source (@.archcore/globals/source-mounts.spec.md).

## Constraints & Invariants

- Output ceiling: 8 source lines, their warnings, the heading, and the precedence sentence. Block size is a function of the ceilings, never of corpus size.
- I/O ceiling: directory walks only, identical to the cost `InspectGlobals` pays (@cli/internal/docs/inspect.go `countGlobalDocs`).
- No global document path, title, or tag appears in the block; global content surfaces through the MCP read tools only.
- No absolute filesystem path appears in any line except the `project` root of clause 17; a warning line carries the declared id and the declared path only (@.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md).

## Failure Behavior

| Condition | Response |
| --- | --- |
| Invalid `settings.json` | invalid-settings warning; no `GLOBALS` block |
| Fatal source (not a directory, unreadable, self-overlap, duplicate) | "⚠" line ending "fix .archcore/settings.json"; remaining sources still render |
| Missing source (not cloned yet) | "⚠" line ending "skipped until it is cloned; local documents are unaffected"; remaining sources still render |
| Empty source | "⚠" contains-no-documents line |
| Unreadable subdirectory during the count | source classified unreadable; "⚠" line |

## Conformance

An implementation conforms when it satisfies clauses 1–17 and the failure rows, and `TestBuildSessionContext_ScansTheCorpusOnce` (@cli/cmd/hook_scan_budget_test.go) still passes. The block tests sit in @cli/cmd/hooks_globals_block_test.go.
