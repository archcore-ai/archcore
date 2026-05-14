---
title: "Zero-Content Onboarding Implementation — SessionStart Nudge + /archcore:init"
status: accepted
tags:
  - "hooks"
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

> **Outcome (2026-05-15):** Plan executed. The skill shipped as `skills/init/` and the command is `/archcore:init` per `skill-surface-collapse.adr.md` (originally drafted as `skills/bootstrap/` and `/archcore:bootstrap`). All references below to bootstrap should be read as init. SessionStart nudge (Phase A), stack-rule generation (B1), run-guide generation (B2), and agent-instruction file import (B3) all shipped as designed.

## Goal

Implement **Variants A + B** from `zero-content-onboarding.idea` so a fresh-install user goes from empty `.archcore/` to a useful seeded state in one short session. Two coupled deliverables:

1. **Phase A — SessionStart nudge.** When `.archcore/` is empty/missing, the SessionStart hook adds one advisory line pointing the user at `/archcore:init`. Pure copy, ~10 lines of shell.
2. **Phase B — `/archcore:init` intent skill.** Three sequential steps. B1 and B2 generate artifacts directly (no accept/edit/skip prompt — the output is a short file that is trivially edited, deleted, or regenerated on demand). B3 is opt-in with a cost warning and a dry-run preview because it can create many documents at once.
   - **B1.** Generate a terse **stack rule** (imperative, no library inventory, no versions) from manifest detection.
   - **B2.** Generate a short **run-the-app guide** from README + scripts, monorepo-aware.
   - **B3.** **Opt-in parse** of existing agent-instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.cursor/rules/*.mdc`, `.github/copilot-instructions.md`, `.windsurfrules`, `.junie/guidelines.md`, `CONVENTIONS.md`) with cost warning when input is large. Default mode per file is **link by reference** — a `doc` whose body holds a one-line pointer and whose tags carry the source identifier (no content duplication). Optional **extract** mode routes content into typed rules / ADRs / docs.

### Out of scope (intentionally deferred)

- **Variant C** (`/archcore:init --scan`) — full repo introspection beyond manifests. Decide go/no-go after observing real B usage.
- **Active guardrail / lint** for the generated stack rule — Phase B produces context, not enforcement.
- **Auto-refresh** of imported docs when source files change. Initially manual (`/archcore:audit --drift` covers it).
- **CLI-side ownership** of empty-state detection. Phase A ships plugin-side first; migration to CLI is a separate decision once the UX is validated.

## Tasks

### Phase A — SessionStart empty-state nudge

#### A1. Empty-state detection and advisory output

**Files touched.**

| File | Change |
|------|--------|
| `bin/session-start` | Add empty-state check after the existing CLI `session-start` invocation. If `.archcore/` is missing OR contains zero documents with `body length ≥ 200 chars` and `status ∈ {accepted, draft}`, append an advisory paragraph to the SessionStart payload. |
| `bin/lib/empty-state.sh` | **New.** Helper that returns 0 if archcore is "functionally empty", 1 otherwise. Uses `find` + `wc -c`; no MCP calls, no jq dependency. |
| `test/unit/session-start-empty.bats` | **New.** Three cases: (a) no `.archcore/` directory → nudge present; (b) `.archcore/` with only `.gitkeep` and stub files < 200 chars → nudge present; (c) `.archcore/` with at least one substantial document → nudge absent. |
| `test/unit/session-start.bats` | Existing cases unchanged. |

**Nudge text (exact).**

```
.archcore/ is empty. Run /archcore:init to seed a stack rule, a run-the-app
guide, and (optionally) imports from existing agent-instruction files like
CLAUDE.md or AGENTS.md. Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.
```

**Escape hatch.** `ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the line entirely.

**One-time suppression after init.** Once `/archcore:init` completes any step, the empty-state check naturally returns `false`. No persistent flag needed for v1.

### Phase B — `/archcore:init` intent skill

#### B1. Stack rule generation

**Skill behaviour.**

1. **Detect manifests.** Read in order, stop at first found per-language: `package.json`, `pnpm-workspace.yaml`, `pyproject.toml`, `Pipfile`, `requirements.txt`, `Cargo.toml`, `go.mod`, `Gemfile`, `composer.json`, `*.csproj`, `pom.xml`, `build.gradle*`. Multiple co-existing manifests are allowed.
2. **Extract signal-bearing deps.** For each manifest, pull top-level declared dependencies. Filter to a curated allowlist of "stack signals". Ignore versions. Cap at **5 signals total** across all manifests.
3. **Detect language fallback.** If no manifest exists: scan top-level file extensions, identify majority language(s). At most 2.
4. **Compose imperative draft.** Template:
    ```
    Code in {language(s)}.
    Build with {framework} (do not introduce alternative frameworks without an ADR).
    Persist via {persistence-lib} when adding storage; default to {primary-store}.
    Style with {styling-lib} classes/components; do not introduce alternative styling.
    Manage state with {state-lib}.
    ```
    Lines without detected signals are dropped, not left as placeholders.
5. **Create directly.** Call `mcp__archcore__create_document(type='rule', filename='project-stack', directory='conventions', status='accepted')`. Report one line: *"Stack: {signals} → {path}"*.
6. **Idempotency.** Before running, check via `list_documents` for an existing `rule` with title containing "stack" in `conventions/` directory. If exists, ask: regenerate / skip.

**Files touched.**

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | **New.** Full intent-skill structure per `skill-file-structure.rule`. Frontmatter `description` triggers on phrases like "init", "initialize archcore", "first-time setup", "set up archcore". |
| `skills/init/lib/detect-stack.md` | **New.** Manifest-to-signals lookup tables. |
| `test/structure/skills.bats` | Extend to assert presence of `init` SKILL.md and required sections. |

#### B2. Run-the-app guide generation

**Skill behaviour.**

1. **Monorepo detection.** Look for `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `lerna.json`, OR multiple `package.json` files under `apps/` or `packages/`.
2. **README extraction.** Read `README.md` and look for the first section matching `(?i)getting started|quick start|installation|development|setup|local`.
3. **Scripts fallback.** If README has no usable section, read `scripts:` from `package.json` (or equivalent).
4. **Compose draft.** Single-app or monorepo template (see source for full templates).
5. **Create directly.** Save as `guide` type with filename `running-the-project`, directory `onboarding/`, status `accepted`.
6. **Idempotency.** Skip if a `guide` titled "running" or "run" exists in `onboarding/`.

**Files touched.**

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | Extend with B2 step. |
| `skills/init/lib/extract-run-instructions.md` | **New.** Heuristics for README section selection. |

#### B3. Opt-in parse of agent-instruction files

**Skill behaviour.**

1. **Detect agent files.** Probe the documented list (stored as data in `skills/init/lib/agent-files.md`).
2. **Cost estimation.** Sum byte-size; show user a summary.
3. **Cost warning threshold.** If combined size > 50 KB, or file count > 5, or estimated yield > 8 documents, prefix with `⚠️ HIGH COST:` and require explicit `do` confirmation.
4. **Per-file mode selection.** Each detected file gets a mode: link (default), extract, or skip.
5. **Source-of-truth representation.** Tag + body convention — every imported document carries `imported` and `source:<slug>` tags plus a body first line `> Imported from \`<path>\` on <ISO-date>.`
6. **Dry-run preview.** Show users the full list of intended writes before any `create_document` calls.
7. **Batch create.** `create_document` per item; create all `related` edges after.
8. **Idempotency.** Docs with matching `source:<slug>` tag are treated as already imported.

**Files touched.**

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | Extend with B3 step. |
| `skills/init/lib/agent-files.md` | **New.** Detection list. |
| `skills/init/lib/extract-routing.md` | **New.** Imperative/decision/reference heuristics. |
| `commands-system.spec.md` | Register `/archcore:init` in the command surface. |
| `skills-system.spec.md` | Add init to the skill section. |

#### B4. Skill metadata and routing

- `description` in `skills/init/SKILL.md` frontmatter triggers on natural-language phrases: "init", "initialize archcore", "first-time setup", "set up archcore", "seed archcore", "what should I do first".
- Skill is auto-invocable per `skill-surface-collapse.adr`.
- `commands-system.spec` documents the trigger phrases.

### Phase C — Documentation

| File | Change |
|------|--------|
| `README.md` | Add a single line under "Try these 3 prompts first": *"Empty repo? Run `/archcore:init` first to seed the basics."* |
| `claude-plugin.prd.md` | Add **FR-7** for first-session activation on empty `.archcore/` (Phase A) and **FR-8** for `/archcore:init` skill (Phase B). |
| `multi-host-compatibility-layer.spec.md` | Document `ARCHCORE_HIDE_EMPTY_NUDGE` env var. |
| `development-roadmap.plan.md` | Mark "zero-content onboarding" as in-progress / shipped per phase. |

### Phase D — Release

- Plugin manifests bump `version` per coordinated release plan.
- README changelog entry.

## Acceptance Criteria

1. **Empty-state nudge fires correctly:** verified by `session-start-empty.bats`.
2. **`ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the nudge** unconditionally.
3. **`/archcore:init` is discoverable and auto-invoked** on phrases listed in B4.
4. **B1 produces a stack rule** of ≤ 6 lines, no version numbers, ≤ 5 stack signals, written directly without a confirm prompt.
5. **B2 produces a run guide** of ≤ 15 lines, with monorepo-detection branching, written directly without a confirm prompt.
6. **B3 detects all files** in the documented list, reports cost accurately, and gates HIGH COST behind explicit confirmation.
7. **B3 link mode** creates `doc` documents with the canonical tag + body pointer convention.
8. **B3 extract mode** routes imperatives → rule, decisions → adr, reference → doc.
9. **All init steps are idempotent** — re-run skips already-created artifacts with a clear message.
10. **PRD updated** with FR-7 and FR-8.
11. **Test suite green.**

## Dependencies

- **No CLI release dependency** for any phase.
- **Existing intent-skill infrastructure** — `skill-surface-collapse.adr.md` provides the auto-invocation contract.
- **Existing per-document-type schemas** (rule, guide, doc, adr).

## Risks

- **B1 detection false positives.** Mitigation: signal allowlist excludes `@types/*`, `eslint-*`, `prettier`, test runners, build tools.
- **B2 README extraction quality.** Marketing-heavy READMEs may yield no usable command blocks. Fallback to `scripts:` extraction.
- **B3 extract-mode quality.** Mitigation: dry-run preview is mandatory; default mode is link.
- **B3 cost estimate accuracy.** Heuristic 1 doc / 800 bytes is rough; ±50% on extreme inputs. Acceptable.
- **Tag-spec compatibility.** Mitigation: targeted unit test that creates a doc with `source:agents-md`.
- **Slug collisions.** Mitigation: include the file extension in the slug.
- **Stale source files.** Mitigation: `/archcore:audit --drift` covers code-doc drift.
- **Idempotency edge cases.** Mitigation: regenerate prompt explicitly warns about overwriting edits.
- **Skill discovery dependency on Variant A.** Phase A and Phase B must ship together.

## Relations

- `implements` → `zero-content-onboarding.idea` (this plan executes variants A and B; defers C)
- `extends` → `claude-plugin.prd` (adds FR-7 and FR-8)
- `extends` → `commands-system.spec` (registers new intent skill)
- `extends` → `skills-system.spec` (adds init to the skill section)
- `extends` → `hooks-validation-system.spec` (Phase A modifies SessionStart behaviour)
- `related` → `skill-surface-collapse.adr` (the rename `bootstrap` → `init`)
- `related` → `skill-file-structure.rule` (SKILL.md must conform)
- `related` → `readme-first-60-seconds.idea`
