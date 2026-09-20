# Init Import Mode and Simplified Init Signature — Design

Date: 2026-09-20. Status: design under user review.
Reader: the implementer of the `init` skill rework and the reviewer of that work.

## Purpose

Give `/archcore:init` a mode that converts the authored context of any repository — agent-instruction files, decision records, contributor docs, internal pages of a published docs site, and recoverable git history — into native `.archcore/` documents. The same work brings the `init` skill under `skill-file-structure.rule` and shortens its signature.

After this work:

- `/archcore:init` takes the form `[import|refresh] [path or domain]`.
- A plain init wires the host, runs one assessment gate, creates the extractive facts, and lets the gate divide the synthesis budget between authored sources and code.
- `/archcore:init import` runs the maximal import, staged by a plan document when the volume requires it.
- No created document carries a mark that it came from an import.

## Context

- `plugins/archcore/skills/init/SKILL.md` holds 442 lines. `skill-file-structure.rule` item 15 sets a maximum of 300. The other three skills hold 119–180 lines and keep flow logic in tracks or `lib/`.
- The current import lives inside the plain init (`init/lib/agent-files.md`, `_shared/grounding/extract-routing.md`). It probes 13 agent-file paths. An aggregate file becomes a one-line link stub. A modular rule file becomes a verbatim copy. Every result carries the tags `imported` and `source:<slug>`, a pointer line `> Imported from … on <date>`, the filename prefix `imported-`, and, for non-rules, the directory `imported/`. An extract with several results adds an umbrella `doc`.
- `magic-first-day-init.adr` (accepted) places imports in the main flow as one preview line, and rejects staging with resume (alternative 6) because staging adds a resume protocol to a skill with one confirm. The same ADR guarantees that no document is created unseen.
- `command-entry-grammar.adr` (draft) fixes the form `[mode] [subject]` with a closed list of noun modes, and keeps a flag only for a setting.
- `plan-discharge-by-deletion.adr` (draft) removes a completed plan from the corpus.

## Settled choices

The user settled these on 2026-09-20:

1. Plain init order: host wiring when missing, then the assessment gate, then the fill. The gate measures the authored backlog. A small backlog leads to the current code seed.
2. The extractive facts are created in every route. The gate gives the synthesis budget to the side with more evidence. Authored content wins over synthesized content on the same topic.
3. Source files change only at a separate `retire` gate at the end, with its own confirm, for agent-instruction files only.
4. A published docs tree is classified page by page by audience. Pages for end users are not converted. One extractive fact records the docs site.
5. Signature: `[import|refresh] [path or domain]`. `domain <slug>` becomes the subject of `refresh`. `--depth` and `--scale` leave the signature and stay as preview toggles `depth:` and `scale:`.
6. A new ADR replaces the "Imports" paragraph of `magic-first-day-init.adr`, limits the rejection of staging to the seed, and introduces one confirm per wave for import. `magic-first-day-init.adr` receives a short update note. The seed keeps one preview and one confirm.

## Approach

Approach B, target-driven conversion. Alternative considered:

- A, source-driven (file → blocks → one document per block): rejected because the corpus then copies the structure of the sources, and two sources on one topic give two documents.

In approach B the unit of work is the target document. Knowledge units from all sources are clustered by topic, and each cluster becomes one document composed under its type contract.

Why staging holds for import and not for the seed: an interrupted seed tops up through `refresh`, because a fact dedupes by its tag and a spec dedupes by the module slug. A target document of an import merges spans of several sources and takes its name from the topic. Without `source:` tags a second run cannot tell which spans are converted. The plan document is the condition for idempotency, not a convenience.

## Command surface

| Invocation | Run |
|---|---|
| `/archcore:init` | wiring → `assess` → facts → fill divided by the gate |
| `/archcore:init import [path]` | maximal import, within `path` when given |
| `/archcore:init refresh [domain]` | top-up; with a domain, the former `domain <slug>` pass |

Import has no depth. Import is maximal by definition, and the plan balances the volume. Subject resolution on `refresh`: an existing path wins, then a domain slug, then one question.

## Components

### 1. Assessment gate (`assess`)

- Does: measures the authored backlog without reading file bodies — paths, byte sizes after stripping the archcore managed block, heading counts, and mentions of source paths inside L1–L2 files.
- Output: an estimate of target documents, the size tier, and the set of hotspot modules that authored sources already cover. The coverage set is an estimate from path mentions; the preview labels it as an estimate.
- Used by: plain init and `import`. Plain init reads levels L1–L2. `import` reads all levels.
- Effect on the seed: the budget formula of `magic-first-day-init.adr` is unchanged. Modules in the coverage set leave the eligible pool.

### 2. Source discovery (`init/lib/sources.md`, replaces `agent-files.md`)

- L1, agent instructions: the 13 current paths, nested `CLAUDE.md` and `AGENTS.md` in subdirectories, `.claude/rules/`.
- L2, decision and design records: `docs/adr`, `adr/`, `decisions/`, `rfcs/`, `design/`, `proposals/`, files named `NNNN-*.md`, `ARCHITECTURE.md`, `DESIGN.md`.
- L3, contributor docs: `CONTRIBUTING`, `DEVELOPMENT`, `TESTING`, `STYLEGUIDE`, developer sections of `README`, package READMEs, `runbooks/`, `ops/`, the pull-request template, a `docs/` tree without a publish config.
- L4, published docs: a tree with a publish config (Docusaurus, MkDocs, Starlight, VitePress, Mintlify, Sphinx, mdBook). Pages with internal knowledge are candidates. Tutorials, API reference, marketing, and changelogs are not.
- L5, git history: deleted Markdown files whose decisions the code still confirms; renames; doc freshness against the churn of the code the doc names; long commit and merge messages as evidence for an `adr`. Git history ranks, supports, and recovers. Git history is never the only source of a `rule`. On a shallow clone L5 is off and the report says so.
- Out of scope: wikis, issue trackers, external tools, code comments.
- The path lists are examples, not a closed checklist, as in the detect catalogs.

### 3. Triage

Each source receives one verdict with a reason shown in the preview: `convert`, `mine` (part of the file), `reference` (a fact pointer only), or `skip` (changelog, license, generated, vendored, translation, marketing).

### 4. Conversion (`_shared/grounding/convert-routing.md`, replaces `extract-routing.md`)

- Unit to type: rule → `rule`; decision with rationale → `adr`; open proposal → `rfc`; procedure → `guide`; boundary contract → `spec`; reference → `doc`; intent → `idea` or `plan`.
- The text is rewritten under the type contract and `precision-rules.md`. When a mandatory section has no source content, the unit takes a type whose contract the content satisfies. Nothing is invented.
- Grounding: every `@path` exists; no claim contradicts the code. A contradiction with the code or between two sources goes to a conflicts list for the user. The import does not choose silently.
- No import marks: no `imported` tag, no `source:` tag, no pointer line, no `imported-` prefix, no `imported/` directory, no umbrella `doc`. Directory by domain or topic. Status `draft`. Relations as for native documents.
- Provenance (source, span → target document) lives only in the import plan.

### 5. Staging and the import plan

Size tiers, thresholds to be calibrated by the behavioral bench [assumption]:

| Tier | Target documents | Execution |
|---|---|---|
| S | ≤ 8 | inline: one preview, one confirm, no plan document |
| M | 9–40 | confirm creates the import `plan`; all waves run in one session |
| L | > 40, or a token estimate above `[LIMIT REQUIRED]` | confirm creates the plan and runs wave 1 (L1 rules, L2 decisions); a later `/archcore:init import` finds the plan and continues |

- The table describes `/archcore:init import`. A plain init keeps one preview and one confirm: in tier S the confirm converts inline beside the seed; in tiers M and L the confirm creates the seed and the import plan and runs wave 1, and the closing message names `/archcore:init import` for the remaining waves.
- Priority: rule, decision, contract, procedure, reference; then freshness; then churn of the governed code.
- The plan grows. A source found during conversion (an `@import`, a link to an in-repo doc) is appended with the state `proposed`. A `proposed` row is not converted until the user confirms it at the end-of-wave checkpoint. This keeps the guarantee that no document is created unseen.
- When the last row closes, the plan is removed per `plan-discharge-by-deletion.adr`. The corpus then holds no mention of the import.

### 6. Import track (`_shared/tracks/import.md`, under `gate-contract.md`)

`assess → discover → triage → plan (confirm) → convert (waves, checkpoint confirm when the plan grew) → verify → retire (own confirm) → discharge`.

- `verify`: `bin/check-precision`, line caps, relations, the conflicts list, and a fidelity check — each normative statement of a target `rule` or `adr` is compared with its source span, so that the form changes and the obligation does not.
- `retire`: agent-instruction files only, only on a clean git tree. The gate shows, per file, the sections that the corpus now covers and offers to remove them. The archcore managed block and uncovered content stay. `docs/`, `README`, and published docs are never edited.

### 7. Skill file layout

- `init/SKILL.md` (≤ 300 lines): arguments, routing table, pre-flight, the `assess` gate, result.
- `init/lib/seed-detect.md`, `init/lib/seed-compose.md` (≤ 200 each): the current phases A–E.
- `init/lib/sources.md`, `_shared/grounding/convert-routing.md`, `_shared/tracks/import.md`: as above.
- Removed: `init/lib/agent-files.md`, `_shared/grounding/extract-routing.md`.

## Risks

- Rewriting an authored rule can shift its meaning; one modal changes agent behavior. Mitigation: status `draft`, the fidelity check, the conflicts list. The residual risk is material for `rule`.
- Commit messages as evidence can add noise. They never produce a document alone.
- `retire` on files joined by `@import` chains can remove content another file still expects. `retire` resolves the chain before it proposes a removal; an unresolved chain excludes the file.
- Breaking change for recorded invocations: `init domain <slug>`, `--depth`, `--scale`.

## Records and tests

- After approval (integration rule 7): a draft ADR for the import mode, a draft spec for the import track, an update note on `magic-first-day-init.adr`, edits to `command-entry-grammar.adr` (draft). `command-surface-v2.spec` is accepted; its init clauses change only with the user's acceptance.
- `test/structure/command-grammar.bats` for the new hint; a structure test for the 300- and 200-line caps on `init`.
- A behavioral bench with fixtures: only `CLAUDE.md`; modular rules plus an ADR directory; a Docusaurus site; a deleted ADR in history; a repository above the L threshold; an interrupted L run that resumes without duplicates.
