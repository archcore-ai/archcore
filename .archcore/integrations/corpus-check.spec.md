---
title: "Corpus Check — The Document Claims a Lookup Settles"
status: draft
tags:
  - "component:cli"
  - "precision"
  - "review"
---

## Purpose & Scope

This spec defines `archcore check`: one deterministic pass over the local documents that reports every claim a lookup settles, so `/archcore:review` spends model turns only on claims that need judgment. Normative for @cli/internal/advisory/corpus_check.go, @cli/internal/advisory/code_scan.go, and @cli/cmd/check.go. Dependents: the actualize track's check 4 (@plugin/plugins/archcore/skills/_shared/tracks/actualize.md) and the `archcore-auditor` agent.

Out of scope: verdicts; semantic claims; global sources.

## Surface

- Invocation: `archcore check [scope...]`, where a scope is a document or a directory under `.archcore/`; `--project` names the root.
- Output: one JSON object on stdout with `documents`, `totals` per kind, `findings`, `omitted` per kind, and `skipped` per kind.
- Finding fields: `kind`, `doc`, `other`, `subject`, `evidence`, `count`, `historical`, `status`, `normative`, `line`.
- Kinds: `unresolved_path`, `line_anchor`, `missing_identifier`, `missing_string`, `draft_cited_by_code`, `dangling_code_citation`, `duplicate_text`, `restated_code`.
- Claim types: `rule`, `cpat`, `adr`, `spec`, `doc`, `guide`, `scenario` — the types whose statements claim present state.
- Git-dependent kinds: `unresolved_path`, `missing_identifier`, `missing_string`, `draft_cited_by_code`, `dangling_code_citation`, `restated_code` — the kinds that need the tracked-file list.
- Prohibition line: a line with a negated BCP 14 keyword, the keyword followed by `NOT`.
- Cited path: a backticked or `@`-prefixed token that holds a `/`. A segment may hold parentheses, as in the route group `app/(shop)/page.tsx`.
- Scanned code: tracked files outside `.archcore/` with an extension from `templates.CodeReferenceExtensions` or a localization format, at most 1 MiB each. Markdown outside `.archcore/` is scanned for document citations only.

## Normative Behavior

1. The check MUST skip every document with status `rejected`.
2. WHEN a cited path resolves under no project or package root, the check MUST report `unresolved_path` with the last commit that touched it, or `never-tracked`.
3. WHEN that commit moved the path, the check MUST name the new path in `evidence`.
4. The check MUST NOT report a cited path whose first segment exists under no root.
5. WHEN a cited slash glob matches no tracked file and no directory of one under any root, the check MUST report `unresolved_path` with evidence `glob-matches-nothing`.
6. WHEN a document cites a line anchor, the check MUST report one `line_anchor` with the count and the first anchor.
7. WHEN a claim-type document cites a code identifier no scanned file holds as a whole token, the check MUST report `missing_identifier`.
8. The check MUST NOT report an identifier that a prohibition line of the same document names.
9. The check MUST NOT report an identifier that contains `xxx` in any letter case.
10. WHEN a `spec` or `scenario` quotes «interface text» no scanned file holds, the check MUST report `missing_string`.
11. WHEN every cited line of a subject in a document narrates a removal, the check MUST set `historical: true`.
12. WHEN a cited line of a subject carries a BCP 14 keyword, the check MUST set `normative: true`.
13. The check MUST order a non-historical finding before a historical one.
14. Among findings of equal history, the check MUST order an `accepted` document's finding before any other status.
15. Among findings of equal history and status, the check MUST order a normative finding before a prose one.
16. WHEN scanned code cites `<slug>.<type>` of a local `draft`, the check MUST report `draft_cited_by_code`.
17. WHEN scanned code cites `<slug> <type>` and that local document exists as a `draft`, the check MUST report `draft_cited_by_code`.
18. WHEN scanned code cites `<slug>.<type>` of no local document, the check MUST report `dangling_code_citation`.
19. WHEN two documents share three or more eight-word runs, the check MUST report `duplicate_text` with the shared count.
20. The check MUST NOT count a run that four or more documents share.
21. WHEN scanned code holds at least three lines and half the non-empty lines of a fenced code block, the check MUST report `restated_code`.
22. WHEN the first line that names an unresolved path narrates a removal, the check MUST set `historical: true` on its `unresolved_path` finding.

## Constraints & Invariants

- Constraint: each kind MUST NOT exceed 40 findings; `omitted` counts the rest.
- Constraint: the pass MUST read each scanned file at most once.
- Constraint: an `unresolved_path`, `missing_identifier`, or `missing_string` finding MUST carry in `line` the first live cited line, trimmed and cut to 200 runes, so the review decides it without a second lookup.
- Invariant: a finding is a settled lookup, never a verdict; the review decides each one.
- Invariant: within one priority, findings keep the order each check produced them in.
- Invariant: a `<slug>.<type>` code citation needs a hyphen in its slug; `data.plan` in code is a field access.
- Invariant: a `<slug> <type>` citation counts only for a document that exists; "the plan" in a comment is a phrase.
- Invariant: a `<slug>.<type>` followed by `.` and a file-name character is a file name (`checkout-flow.spec.ts`), not a citation; a sentence-final `.` or `.md` ends a citation.
- Invariant: on 2026-11-18 the pass took 9.5 s over the 265 documents and 15 341 tracked files of a field corpus. A `git grep -F -f` over the 3 953 cited identifiers did not finish in 300 s.

## Failure Behavior

1. IF the tracked-file list cannot be read, THEN the check MUST report each git-dependent kind under `skipped` with the git error as the reason.
2. IF the tracked-file list cannot be read, THEN the check MUST still run `line_anchor` and `duplicate_text`.
3. IF a scanned file cannot be read, THEN the check MUST skip it and keep the other findings.

## Conformance

An implementation is conformant when it satisfies behaviors 1–22, holds every constraint and invariant, and degrades per the failure rules. @cli/internal/advisory/corpus_check_test.go seeds one finding and one look-alike per kind, pins the priority order, the moved-to evidence, and the skip outside git.