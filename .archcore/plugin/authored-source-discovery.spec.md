---
title: "Authored Source Discovery — Five Levels and Triage Verdicts"
status: draft
tags:
  - "onboarding"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec governs how `/archcore:init` finds authored context in a repository and what verdict each found source receives. Dependents: the assessment gate of a plain init, which reads levels L1–L2, and the import track, which reads all five levels and converts what this spec marks. Out of scope: how a marked source becomes documents (conversion spec) and stores outside the repository — wikis, issue trackers, code comments.

## Surface

- Catalog file: `init/lib/sources.md` — level definitions, example paths, publish-config markers, skip classes. The catalog replaces @plugins/archcore/skills/init/lib/agent-files.md.
- Levels: `L1` agent instructions; `L2` decision and design records; `L3` contributor docs; `L4` published docs tree; `L5` git history.
- Source record fields: `path`, `level`, `bytes`, `headings`, `verdict`, `reason`, `last_change`.
- Verdicts: `convert`, `mine`, `reference`, `skip`.
- Git inputs for L5: deleted Markdown paths, renames, last-change dates, commit and merge message bodies.

## Normative Behavior

1. The discovery step MUST treat every path list in the catalog as examples, not as a closed checklist.
2. The discovery step MUST assign each found source exactly one level.
3. The discovery step MUST find nested `CLAUDE.md` and `AGENTS.md` files below the repository root as L1 sources.
4. WHEN a directory tree holds a publish config, the discovery step MUST mark every page of that tree as L4.
5. WHEN a Markdown tree under `docs/` holds no publish config, the discovery step MUST mark its pages as L3.
6. The triage step MUST give each source exactly one verdict with a one-line reason.
7. WHEN an L4 page addresses end users — tutorial, API reference, marketing, changelog — the triage step MUST give the verdict `skip`.
8. WHEN an L4 page records architecture, decisions, internals, or contributor process, the triage step MUST give the verdict `convert` or `mine`.
9. WHEN an L4 tree exists, the triage step MUST plan one `reference` fact that records the site's path, build tool, and covered topics.
10. WHEN a file mixes authored knowledge with content of a skip class, the triage step MUST give the verdict `mine`.
11. The triage step MUST give the verdict `skip` to licenses, changelogs, generated files, vendored files, translations of a converted page, and marketing pages.
12. WHEN L5 finds a deleted decision record, the discovery step MUST add it as a candidate only after the code confirms its decision.
13. The discovery step MUST use commit and merge messages as evidence for a target `adr`, never as a standalone source.
14. WHEN git history exists, the discovery step MUST record each source's `last_change` from git.
15. WHEN a path subject scopes the import, the discovery step MUST limit every level to that path.
16. The preview MUST show each source with its level, verdict, and reason.
17. During assessment, the discovery step MUST record each in-scope L4 site's publish config as one `reference` source per config root.
18. The discovery step MUST retain the site record regardless of page verdicts or the config's byte size.

## Constraints & Invariants

- Constraint: discovery before the plan confirm reads paths, sizes, headings, and git metadata only; full bodies are read after the confirm, because the read is the main token cost.
- Constraint: L5 MUST NOT be the only source of a target `rule`; a rule without a standing authored source has no owner to confirm it.
- Invariant: the archcore managed block (`<!-- archcore:start -->` … `<!-- archcore:end -->`) is never a source and never counts toward a size.
- Invariant: a file whose only content is the managed block produces no source record.
- Invariant: `.archcore/`, `.git/`, dependency directories, and build outputs are never scanned.

## Failure Behavior

1. IF the repository is a shallow clone, THEN the discovery step MUST disable L5 and state that in the report.
2. IF the repository has no git history, THEN the discovery step MUST run L1–L4 and omit `last_change`.
3. IF paths and headings leave a page's audience undecided, THEN the triage step MUST assign `mine` pending the body read.
4. IF a path subject matches no source, THEN the discovery step MUST report zero sources and create no plan.
5. IF two publish configs cover one tree, THEN the discovery step MUST treat the tree as one L4 site per config root.

## Conformance

An implementation conforms when it satisfies behaviors 1–18, holds the three invariants and both constraints, and follows the five failure rules.

Given a repository with `docs/docusaurus.config.js`, `docs/tutorial/intro.md`, and `docs/internals/locking.md`
When discovery and triage run
Then `intro.md` is `skip`, `locking.md` is `convert`, and one `reference` fact records the docs site.
