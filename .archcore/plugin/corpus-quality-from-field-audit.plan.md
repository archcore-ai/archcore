---
title: "Corpus Quality From a Field Audit: Language-Neutral Prose, Stable References, Review That Verifies"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "docs-style"
  - "precision"
---

## Goal

Raise the quality of every Archcore corpus, in any document language, by closing the gaps a field audit exposed. The audit read a real 294-document Russian corpus on 2026-10-06 and checked claims against `origin/master`:

- 23 of 53 sampled checkable claims were false or stale — versions, file trees, line anchors, counts. [assumption] The sample was chosen for checkability, so the true rate is likely lower.
- 0 drafts moved to `accepted` after their work merged: all 6 draft `adr` and 27 draft `spec` described shipped code.
- 85% of 766 relations were `related`, and about 47% of sampled `related` edges carried no claim. Meanwhile 8 of 10 sampled body mentions of another document justified an edge that did not exist.
- 7 ways of stating an obligation appeared across 54 `rule` documents, and 56% of measured `rule` clauses carried no BCP 14 keyword.

### Why review did not catch it

Five properties of `/archcore:review` explain the miss. Each is grounded in `@plugin/plugins/archcore/skills/_shared/tracks/actualize.md` and `@plugin/plugins/archcore/skills/review/SKILL.md` as they stood before this plan.

| Gap | Effect in the field corpus |
|---|---|
| Code drift flagged a document only when a cited path appeared in the current branch diff | A path deleted by an earlier merged branch was never flagged again; 8 cited paths and 64 backticked paths did not resolve |
| The default branch produced a counts-only dashboard | The merged corpus — where every stale fact lives — got no check |
| Temporal checks flagged drafts by age (over 30 days) only | Drafts 1–8 weeks old with shipped work were invisible |
| `deep` named "consistency" as tags, naming, and directories | Contradicting facts across documents (two runtime versions, two cookie lifetimes) went unreported |
| No claim was verified unless its path was in the diff | Facts with no path — a framework version, a runtime version — were never compared with their owning file |

## Tasks

### Done in this change

1. [x] Record the decision that code references name a file or a directory — `code-references-name-a-file-or-directory.adr`.
2. [x] Record the decision that structure tokens stay English — `language-neutral-structure-tokens.adr`.
3. [x] Add precision Rules 9–12: references, structure tokens, owned values, hidden markup — @plugin/plugins/archcore/skills/_shared/precision-rules.md.
4. [x] State the inline-mention relation decision in precision Rule 5.
5. [x] Add `WHERE` and the keywordless-clause defect to precision Rule 7.
6. [x] Require an existing verifier in a rule's Enforcement — @plugin/plugins/archcore/skills/_shared/rule-contract.md.
7. [x] Add body-signal candidates and the dense-`related` shape — @plugin/plugins/archcore/skills/_shared/relation-authoring.md.
8. [x] Add reference resolution, line anchors, and shipped drafts to actualize detection.
9. [x] Add deep checks: claim sampling, cross-document conflict, type fitness, relation candidates.
10. [x] Run reference checks on the default branch in the review health dashboard.
11. [x] Mirror the new staleness and consistency dimensions into the auditor agent and its two copies.
12. [x] Report a graded clause with no BCP 14 keyword, naming its native modal — @cli/internal/advisory/precision.go.
13. [x] Add the `NativeModals` and `CodeReferenceExtensions` tables — @cli/templates/precision.go.
14. [x] Report a code reference that carries a line number.
15. [x] Recognize `WHERE` as an EARS opener.
16. [x] Extend the MCP language directive with structure tokens — @cli/internal/mcp/server.go.
17. [x] Replace the line anchors in `architecture/advisory-subsystem.doc`.
18. [x] Move reference resolution and line anchors into `bin/check-references`, with history and rename detection.
19. [x] Give observations no verdict; bound the deep checks to 20 documents and 3 claims each.
20. [x] Exclude branch-created drafts from the shipped-draft check; closeout offers their acceptance.
21. [x] Point the auditor at the track and the helper output instead of restating the checks.

### Open — this repository

22. [ ] Replace line anchors in the 13 local documents the helper reports.
23. [ ] Add a corpus-wide precision report command so `deep` reads engine findings instead of re-deriving them.
24. [ ] Move `bin/check-references` into the engine, so hosts without a shell get the reference checks.
25. [ ] Cap a `spec` by words beside lines; 327 measured spec lines exceed 400 characters.
26. [ ] Replace the `doc` template's `## Content` container with guidance for descriptive `##` headings.
27. [ ] Teach `scripts/prose-conformance.py` the keywordless-clause and line-anchor counts.
28. [x] Re-run the deep probe on 16 field documents: 217k tokens, 39 calls.

### Open — owner: the `archcore` global source

These edits target read-only shared rules; this repository names them and does not write them.

29. [ ] `concepts/document-prose-canon`: set the `spec` body metric to 120 lines, matching the engine and `spec-contract.md`.
30. [ ] `concepts/relation-conventions`: change `adr related rule` to `rule implements adr`, matching the decision track.
31. [ ] `concepts/document-prose-canon`: add the `WHERE` form to F1 and the structure-token rule.
32. [ ] `concepts/document-prose-canon`: label P5 and P7 as engineering rules, not plain-language rules.
33. [ ] `concepts/controlled-technical-writing`: add the file-or-directory reference rule beside rule 12.

## Acceptance Criteria

- `go test ./...` and `golangci-lint run ./...` pass in `cli/`; `make -C plugin test` passes.
- A Russian, German, or Japanese graded clause without a keyword produces one finding that names its native modal — `@cli/internal/advisory/precision_language_test.go`.
- A reference such as `server.go:34-40` or `app.tsx#L12` produces a finding; `cache.example.com:6379` does not.
- On the field corpus, `/archcore:review` on the default branch reports the unresolved paths and the line anchors.
- On the field corpus, `/archcore:review deep` reports a sampled claim accuracy per type and at least the two runtime-version conflicts.
- The global source carries tasks 29–33, and a re-run of the conflict list finds no plugin–canon disagreement.

## Dependencies

- `plugin/code-references-name-a-file-or-directory.adr` and `plugin/language-neutral-structure-tokens.adr`, both accepted on 2026-10-06.
- Tasks 29–33 wait for a change in the `archcore` global source repository.
- [assumption] The engine change reaches users with the next CLI release; plugin prompt changes reach them with the next plugin release.
