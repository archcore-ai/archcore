---
title: "Path Index in .sync-state.json — Constant-Time Code-Alignment Lookup"
status: draft
tags:
  - "architecture"
  - "hooks"
  - "plugin"
---

## Summary

Extend the CLI to maintain a reverse index from source-path tokens to the `.archcore/` documents referencing them, persisted as an additive `path_index` object in `.sync-state.json`. The code-alignment hook then replaces its per-token corpus scan with a map lookup, targeting under 200 ms at 100 documents and essentially flat scaling beyond. Deferred: the promotion trigger is the first real repository whose injection breaches the CLI's 1-second internal budget, or a deliberate performance-hardening release.

## Motivation

Code-alignment injection fires on every source-file edit. Scanning cost tracks the number of matching documents, and the failure mode is silent: on timeout the edit proceeds and no context arrives — push-mode stops working on exactly the repositories with the most context to give. What each scan answers — "which documents mention this directory prefix" — is a reverse index, answerable from a precomputed map in microseconds. Since CLI v0.7.0 both the producer (`archcore sync`) and the consumer (the CLI's pre-tool-use handler) live in the same binary, which removes the original cross-repo coordination cost.

## Detailed Design

- **Schema.** `path_index: { schema: "v1", built_at, built_by, roots: [...], tokens: { "<prefix>/": ["<doc path>", ...] }, docs: { "<doc path>": { type, title, tokens } } }` — `tokens` is the forward lookup, `docs` a metadata sidecar (type for ranking, title for rendering, no per-candidate frontmatter parse), `roots` records what the index was built against, `schema`/`built_by` carry forward compatibility.
- **Token extraction.** Scan document bodies (code fences included) and advisory frontmatter path arrays for references starting with a configured source root; normalize (strip quotes, trailing punctuation; keep the trailing slash as a directory marker; reject spaces and control characters); expand each reference into its prefix set capped at 5 levels; deduplicate per document.
- **Corpus.** Only the five injected types contribute: `rule`, `cpat`, `adr`, `spec`, `guide`; only `accepted` and `draft` statuses. Vision and requirements types are excluded so aspirational content never rides a code edit.
- **Integration.** A pure `internal/pathindex` package (`Extract`, `Build`); one extra call on the existing `archcore sync` write path — same cadence, same atomic write, no new command. `--no-path-index` for emergency disablement; `doctor` accepts but never requires the field.
- **Consumer fallback.** A missing index, a schema mismatch, or a corrupt object falls back to the live scan silently; both paths must emit identical ranking on the same fixture.
- **Budgets.** Index size ≤ 5% of summed body bytes of included documents; build ≤ 50 ms per 100 documents; lookup wall-clock ≤ 200 ms on a 100-document fixture.

## Drawbacks

- Staleness between sync runs — mitigated by falling back to the live scan for tokens the index lacks, so there is no silent wrong answer.
- Code-fence false positives overestimate candidates — acceptable, ranking compensates; exclude fences in a schema bump if noise becomes user-visible.
- Single-file growth: ~100 KB at 500 documents is fine for a tracked file; revisit a sibling artifact near 4000.

## Alternatives

- **Query at read time with no index** — rejected because the hook shares a 1-second budget and the scan cost tracks matches, which is the observed failure mode.
- **A full-text inverted content index** — rejected because `search_documents` already owns topic retrieval; this index answers only the path-prefix question.
- **A per-document mtime index** — rejected because mtime is already read from the filesystem during ranking.