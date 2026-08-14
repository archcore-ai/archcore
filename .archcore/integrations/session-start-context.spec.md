---
title: "SessionStart Context Output Contract"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "performance"
---

## Purpose &amp; Scope

This spec defines what `archcore hooks <host> session-start` emits: the payload
text and the invariants that keep it bounded and host-safe. Normative for
`buildSessionContext` (@cmd/hooks_common.go) and every per-host writer that
wraps its result.

Out of scope: the per-host JSON envelope shapes, the dedup stamp, the
document model, and the content of the `GLOBALS` block, which
@.archcore/globals/session-globals-disclosure.spec.md owns.

## Surface

One context string, embedded by the caller in a host-specific JSON envelope.

Sections, in order: the header and MCP tool line; the invalid-settings warning
when present; `CORPUS`; `BRANCH` when resolvable; `GLOBALS` when sources are
declared; `IN PROGRESS`; `RECENTLY ACCEPTED`; the staleness advisory when due;
`EXISTING TAGS`; `DOCUMENT RELATIONS`; the closing pointer to the server
instructions.

## Normative Behavior

1. The context string MUST hold at most 24 document lines across `IN PROGRESS`
   and `RECENTLY ACCEPTED` combined.
2. WHEN a block is truncated, the builder MUST state how many documents were
   omitted and name the `list_documents` filter that returns them.
3. The builder MUST render each document line with the full `.archcore/`-prefixed
   path, so the reader can pass it to `get_document` unchanged.
4. The builder MUST exclude a document with status `rejected` from both blocks.
5. WHEN the corpus holds at least one `rejected` document, the builder MUST
   report their count in `CORPUS`.
6. The builder MUST include tags from every local document, `rejected` included.
7. The builder MUST return a document count covering every local document,
   `rejected` included.
8. The builder MUST treat an absent or unrecognized status as `draft`.
9. The builder MUST order both blocks by modification time, newest first.
10. The builder MUST limit `RECENTLY ACCEPTED` to documents modified within the
    last 30 days.
11. The builder MUST cap `EXISTING TAGS` at 20 tags, ranked by frequency.
12. The builder MUST NOT emit any JSON. The caller owns the envelope.
13. WHEN the staleness advisory has fired for a project within 24 hours, the
    builder MUST omit it.
14. The staleness advisory MUST NOT name a command the CLI does not own.
15. The builder MUST reserve up to 6 budget lines for `RECENTLY ACCEPTED`.
16. The builder MUST give `IN PROGRESS` every line that reserve leaves.
17. WHEN `settings.json` declares at least one global source, the builder MUST
    render the `GLOBALS` block per
    @.archcore/globals/session-globals-disclosure.spec.md.
18. WHEN the `GLOBALS` block renders, the builder MUST label the `CORPUS` count
    "local documents".

## Constraints &amp; Invariants

- Invariant: output length is a function of the line budget, not of corpus size.
  Between a 300-document and a 3000-document corpus the length MAY differ only
  by the width of the `CORPUS` counters. The `GLOBALS` block carries its own
  ceilings for the same reason.
- Invariant: this function owns the text inside the envelope and never the
  envelope itself. The plugin splices its own advisories into the same JSON
  document on Copilot, where a failed parse discards the entire payload — a
  change to the wrapper would break that splice with no error anywhere.
- Constraint: `CORPUS` omits any status counted zero, not only `rejected`.
- Constraint: every dedup scope keeps its own stamp directory. A sweep expires
  everything older than its own window, so a shared directory would let the
  10-minute session scope erase the 24-hour staleness budget.
- Constraint: no global document line appears in this context. The `GLOBALS`
  block reports per-source metadata only; global content stays behind the MCP
  read tools (@.archcore/globals/session-globals-disclosure.spec.md).

## Failure Behavior

1. IF a declared global source is unreadable, THEN the builder MUST degrade to a
   local-only scan, render the source's warning inside the `GLOBALS` block, and
   keep every local document.
2. IF `settings.json` is present but invalid, THEN the builder MUST emit a
   warning naming the failure, MUST NOT render a `GLOBALS` block, and MUST
   continue.
3. IF the directory is not a git working tree, or git is absent, or HEAD is
   detached, THEN the builder MUST omit the `BRANCH` line rather than render a
   placeholder.
4. IF the corpus is empty, THEN the builder MUST say so in `CORPUS` and MUST
   omit both recap blocks.
5. IF the staleness correlation finds no affected document, THEN the builder
   MUST omit the advisory and MUST NOT consume the 24-hour budget.

## Conformance

The builder is conformant when it satisfies behaviors 1–18, holds every
invariant, and degrades per the failure rules. The budget invariant is verified
against synthetic corpora of 300 and 3000 documents; the `GLOBALS` behaviors
are verified by @cmd/hooks_globals_block_test.go.