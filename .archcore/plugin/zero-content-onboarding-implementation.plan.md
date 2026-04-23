---
title: "Zero-Content Onboarding Implementation — SessionStart Nudge + /archcore:bootstrap"
status: draft
tags:
  - "hooks"
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Implement **Variants A + B** from `zero-content-onboarding.idea` so a fresh-install user goes from empty `.archcore/` to a useful seeded state in one short session. Two coupled deliverables:

1. **Phase A — SessionStart nudge.** When `.archcore/` is empty/missing, the SessionStart hook adds one advisory line pointing the user at `/archcore:bootstrap`. Pure copy, ~10 lines of shell.
2. **Phase B — `/archcore:bootstrap` intent skill.** Three sequential steps, each independently confirmable:
   - **B1.** Generate a terse **stack rule** (imperative, no library inventory, no versions) from manifest detection.
   - **B2.** Generate a short **run-the-app guide** from README + scripts, monorepo-aware.
   - **B3.** **Opt-in parse** of existing agent-instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.cursor/rules/*.mdc`, `.github/copilot-instructions.md`, `.windsurfrules`, `.junie/guidelines.md`, `CONVENTIONS.md`) with cost warning when input is large. Default mode per file is **link by reference** — a `doc` whose body holds a one-line pointer and whose tags carry the source identifier (no content duplication). Optional **extract** mode routes content into typed rules / ADRs / docs.

### Out of scope (intentionally deferred)

- **Variant C** (`/archcore:bootstrap --scan`) — full repo introspection beyond manifests. Decide go/no-go after observing real B usage.
- **Active guardrail / lint** for the generated stack rule — Phase B produces context, not enforcement.
- **Auto-refresh** of imported docs when source files change. Initially manual (`/archcore:actualize` covers it).
- **CLI-side ownership** of empty-state detection. Phase A ships plugin-side first; migration to CLI is a separate decision once the UX is validated.

## Tasks

### Phase A — SessionStart empty-state nudge

#### A1. Empty-state detection and advisory output

**Files touched.**

| File | Change |
|------|--------|
| `bin/session-start` | Add empty-state check after the existing CLI `session-start` invocation. If `.archcore/` is missing OR contains zero documents with `body length ≥ 200 chars` and `status ∈ {accepted, draft}`, append an advisory paragraph to the SessionStart payload. |
| `bin/lib/empty-state.sh` | **New.** Helper that returns 0 if archcore is "functionally empty" (uses the threshold above), 1 otherwise. Uses `find` + `wc -c`; no MCP calls, no jq dependency. |
| `test/unit/session-start-empty.bats` | **New.** Three cases: (a) no `.archcore/` directory → nudge present; (b) `.archcore/` with only `.gitkeep` and stub files < 200 chars → nudge present; (c) `.archcore/` with at least one substantial document → nudge absent. |
| `test/unit/session-start.bats` | Existing cases unchanged — they run with populated fixtures, so the nudge stays absent. |

**Nudge text (exact).**

```
.archcore/ is empty. Run /archcore:bootstrap to seed a stack rule, a run-the-app
guide, and (optionally) imports from existing agent-instruction files like
CLAUDE.md or AGENTS.md. Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.
```

**Escape hatch.** `ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the line entirely. Documented in `multi-host-compatibility-layer.spec` env-var table.

**One-time suppression after bootstrap.** Once `/archcore:bootstrap` completes any step, the empty-state check naturally returns `false` (documents now exist). No persistent flag needed for the v1.

### Phase B — `/archcore:bootstrap` intent skill

#### B1. Stack rule generation

**Skill behaviour.**

1. **Detect manifests.** Read in order, stop at first found per-language: `package.json`, `pnpm-workspace.yaml` (workspace marker), `pyproject.toml`, `Pipfile`, `requirements.txt`, `Cargo.toml`, `go.mod`, `Gemfile`, `composer.json`, `*.csproj`, `pom.xml`, `build.gradle*`. Multiple co-existing manifests are allowed (polyglot repos).
2. **Extract signal-bearing deps.** For each manifest, pull top-level (declared, not transitive) dependencies. Filter to a curated allowlist of "stack signals" — frameworks (Next.js, Django, Rails, Spring), runtimes (Node, Bun, Deno), primary persistence (Prisma, SQLAlchemy, ActiveRecord, sqlx), primary UI (React, Vue, Svelte), styling (Tailwind, Stitches), state (Redux, Zustand, Reatom, Pinia). Ignore versions. Cap at **5 signals total** across all manifests.
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
5. **Show preview, ask user.** "I detected {stack signals}. Proposing the rule below — accept, edit, or skip?" Three options. On accept or after edit, call `mcp__archcore__create_document(type='rule', filename='project-stack', directory='conventions', status='accepted')`.
6. **Idempotency.** Before running, check via `list_documents` for an existing `rule` with title containing "stack" in `conventions/` directory. If exists, ask: regenerate / skip.

**Files touched.**

| File | Change |
|------|--------|
| `skills/bootstrap/SKILL.md` | **New.** Full intent-skill structure per `skill-file-structure.rule`. Frontmatter `description` triggers on phrases like "bootstrap", "initialize archcore", "first-time setup", "set up archcore". |
| `skills/bootstrap/lib/detect-stack.md` | **New (skill-internal reference).** Manifest-to-signals lookup tables that the skill reads at runtime. Markdown so it stays editable, not code. |
| `test/structure/skills.bats` | Extend to assert presence of `bootstrap` SKILL.md and required sections. |

**Acceptance for B1.** On a fixture `package.json` declaring `next`, `react`, `prisma`, `tailwindcss`, the skill produces a draft rule mentioning those four signals (and only those), without versions, in ≤ 6 lines.

#### B2. Run-the-app guide generation

**Skill behaviour.**

1. **Monorepo detection.** Look for `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `lerna.json`, OR multiple `package.json` files under `apps/` or `packages/`. If detected, the guide gets per-app sections + a workspace-level section.
2. **README extraction.** Read `README.md` (and `README.{en,ru}.md` variants). Look for the first section matching `(?i)getting started|quick start|installation|development|setup|local`. Extract command blocks (fenced ```bash / ```sh / ```shell) within that section.
3. **Scripts fallback.** If README has no usable section, read `scripts:` from `package.json` (or equivalent for other languages: `[tool.poetry.scripts]`, `Cargo.toml [[bin]]`, etc.). Pick `dev`, `start`, `build`, `test`, `lint` if present.
4. **Compose draft.** Template:
   - **Single-app:**
       ```
       ## Prerequisites
       {detected runtime versions, e.g., "Node 20+ (declared in package.json engines)"}

       ## Install
       {detected install command, e.g., "pnpm install"}

       ## Run locally
       {detected dev command, e.g., "pnpm dev"}

       ## Test
       {detected test command, if present}
       ```
   - **Monorepo:** prerequisites + workspace install at top, then per-app subsection with that app's commands.
5. **Preview + confirm + create.** Same UX as B1. Save as `guide` type with filename `running-the-project`, directory `onboarding/`, status `accepted`.
6. **Idempotency.** Skip if a `guide` titled "running" or "run" exists in `onboarding/`.

**Files touched.**

| File | Change |
|------|--------|
| `skills/bootstrap/SKILL.md` | Extend with B2 step. |
| `skills/bootstrap/lib/extract-run-instructions.md` | **New.** Heuristics for README section selection and command-block extraction. |

**Acceptance for B2.** On a fixture repo with README "Quick Start" section containing `pnpm install` + `pnpm dev`, the skill produces a guide with those commands in correct order, ≤ 15 lines, no marketing prose copied.

#### B3. Opt-in parse of agent-instruction files

**Skill behaviour.**

1. **Detect agent files.** Probe (existence + size) the following paths in order:
    - `CLAUDE.md`, `CLAUDE.local.md`
    - `AGENTS.md`
    - `.cursorrules` (legacy)
    - `.cursor/rules/*.mdc`, `.cursor/rules/*.md`
    - `.github/copilot-instructions.md`
    - `.github/instructions/*.md`
    - `.windsurfrules`
    - `.windsurf/rules/*.md`
    - `.junie/guidelines.md`
    - `CONVENTIONS.md` (Aider convention, repo root)

    The list is **stored as data** in `skills/bootstrap/lib/agent-files.md` so additions don't require touching the skill body.

2. **Cost estimation.** Sum byte-size of detected files; count files; estimate doc yield. Heuristic for yield: combined file size / 800 bytes (one document per ~800 chars of source, capped at 25). Show user:
    ```
    Found N files (X KB total). Parsing will create up to ~Y documents and may
    take ~30–90 seconds and consume ~Z tokens.
    ```

3. **Cost warning threshold.** If any of the following triggers, prefix the message with `⚠️ HIGH COST:` and require explicit `do` (not just Enter):
    - combined size > **50 KB**, OR
    - file count > **5**, OR
    - estimated yield > **8 documents**.

4. **Per-file mode selection.** For each detected file, the user picks one of three modes:
    - **link** (default) — create one `doc` with `title` derived from the source path. Body contains only a one-line pointer (see Source-of-truth representation below). No content duplication. Tag `imported` plus a `source:<slug>` tag are added so the document is queryable and re-runs are idempotent.
    - **extract** — read body, split into semantic blocks, route each block:
        - **Imperative** (regex hits like `\b(must|should|always|never|use|do not)\b` plus rule-like sentences) → `rule`, one per coherent block.
        - **Decision with rationale** (heuristic: contains "we chose" / "we decided" / "because" / "rather than") → `adr`.
        - **Reference / overview** (no imperatives, no decisions) → `doc` with the same source pointer convention as link mode.
        Each created document carries the `imported` and `source:<slug>` tags AND the body pointer line, plus a `related` edge to a single umbrella `doc` representing the original file.
    - **skip** — record in skill state (in-memory for the session) so the file is not re-prompted on `/archcore:bootstrap` re-run unless `--reset` is passed.

5. **Source-of-truth representation.** Probed against the current CLI MCP (2026-04-23): unknown frontmatter fields like `source:` are **silently stripped** by `create_document`. Only canonical fields (`title`, `status`, `tags`) survive the round-trip. Therefore B3 uses a **tag + body convention** instead of frontmatter:

    - **Tags** (mandatory on every imported document):
        - `imported` — literal marker; enables blanket queries like "show me everything bootstrap pulled in."
        - `source:<slug>` — slugified source filename (e.g., `source:agents-md`, `source:cursorrules`, `source:cursor-rules-styling-mdc`). Slug rules: lowercase, alphanumeric + hyphens, dots replaced with hyphens, slashes collapsed. The colon is allowed in the tag spec.
    - **Body first line** (mandatory, exact format):
        ```
        > Imported from `<exact-relative-path>` on <ISO-8601-date>.
        ```
        The relative path is what cannot be encoded in a slug (slashes, dots). The user/agent reads it to navigate to the source file.
    - **Idempotency lookup**: `list_documents` filtered by `tag=imported`, then check whether `source:<slug>` is already present for each detected agent file's slug. If yes, skip with a note. If no, propose import.

    Schema migration path (deferred, out of this plan's scope): if the CLI later accepts custom frontmatter, B3 can migrate to a true `source:` field via a one-pass `get_document` → re-create round trip. The `imported` tag stays for backward compatibility.

6. **Dry-run preview.** Before any `create_document` calls, show the user the full list: *"Will create N documents: rule(3), adr(1), doc(2). Confirm?"* Single y/n on the batch.

7. **Batch create.** Use `mcp__archcore__create_document` per item; create all `related` edges via `mcp__archcore__add_relation` after creates succeed. If any create fails, roll forward (do not delete partial set), surface the error, continue with remaining items.

8. **Idempotency.** A doc with the matching `source:<slug>` tag is treated as already imported; skip with a note. Re-running B3 only processes new files.

**Files touched.**

| File | Change |
|------|--------|
| `skills/bootstrap/SKILL.md` | Extend with B3 step. |
| `skills/bootstrap/lib/agent-files.md` | **New.** Detection list (paths + glob patterns), kept editable. |
| `skills/bootstrap/lib/extract-routing.md` | **New.** Imperative/decision/reference heuristics with examples. Skill reads this at runtime. |
| `commands-system.spec.md` | Register `/archcore:bootstrap` in the intent-skill table. |
| `skills-system.spec.md` | Add bootstrap to the intent-skill section. |

**Acceptance for B3.**

- On a fixture repo with `AGENTS.md` (3 KB) and `.cursorrules` (1 KB):
  - Detection lists both files.
  - Cost message reports "2 files, 4 KB, ~5 docs, ~30s, no warning."
  - Default link mode produces 2 `doc` documents with tags `["imported", "source:agents-md"]` and `["imported", "source:cursorrules"]`, body first line in the canonical `> Imported from …` format.
  - Re-run produces "no new files to import."
- On a fixture repo with 8 files totalling 80 KB:
  - Cost message starts with `⚠️ HIGH COST:`.
  - Skill requires explicit `do` confirmation (not Enter).

#### B4. Skill metadata and routing

- `description` in `skills/bootstrap/SKILL.md` frontmatter triggers on natural-language phrases: "bootstrap", "initialize archcore", "first-time setup", "set up archcore", "seed archcore", "what should I do first".
- Skill is auto-invocable per `inverted-invocation-policy.adr` (Layer 1 intent).
- `commands-system.spec` documents the trigger phrases.

### Phase C — Documentation

| File | Change |
|------|--------|
| `README.md` | Add a single line under "Try these 3 prompts first": *"Empty repo? Run `/archcore:bootstrap` first to seed the basics."* |
| `claude-plugin.prd.md` | Add **FR-7** for first-session activation on empty `.archcore/` (Phase A) and **FR-8** for `/archcore:bootstrap` skill (Phase B). |
| `multi-host-compatibility-layer.spec.md` | Document `ARCHCORE_HIDE_EMPTY_NUDGE` env var. |
| `development-roadmap.plan.md` | Mark "zero-content onboarding" as in-progress / shipped per phase. |

### Phase D — Release

- `.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json`: bump `version` per coordinated release plan (target version not fixed in this plan; coordinated separately).
- README changelog entry.

## Acceptance Criteria

1. **Empty-state nudge fires correctly:** `bin/session-start` emits the advisory line on missing or substantively-empty `.archcore/`; absent on populated `.archcore/`. Verified by `session-start-empty.bats`.
2. **`ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the nudge** unconditionally. Verified by a dedicated bats case.
3. **`/archcore:bootstrap` is discoverable and auto-invoked** on phrases listed in B4. Verified by structural assertions in `test/structure/skills.bats`.
4. **B1 produces a stack rule** of ≤ 6 lines, no version numbers, ≤ 5 stack signals, on the fixture repo described.
5. **B2 produces a run guide** of ≤ 15 lines, with monorepo-detection branching working on fixture repos for both single-app and pnpm workspace.
6. **B3 detects all files in the documented list**, reports cost accurately within ±20%, and gates HIGH COST behind explicit `do` confirmation.
7. **B3 link mode creates `doc` documents with the canonical tag + body pointer convention** (`imported` + `source:<slug>` tags; body first line `> Imported from \`<path>\` on <date>.`) and zero content duplication. Verified by checking that `body length < 200 chars` for link-mode imports and that the tag set matches the source filename slug.
8. **B3 extract mode routes imperatives → rule, decisions → adr, reference → doc** on the fixture file with mixed content; all extracted documents carry the `imported` and `source:<slug>` tags plus the body pointer line.
9. **All bootstrap steps are idempotent** — re-run skips already-created artifacts with a clear message. Idempotency check uses tag-based lookup (`source:<slug>`).
10. **PRD updated** with FR-7 and FR-8; spec docs reflect the new skill and env var.
11. **Test suite green:** baseline + new bats cases.

## Dependencies

- **No CLI release dependency** for any phase. All reachable via existing MCP surface (`create_document`, `list_documents`, `add_relation`). Source-of-truth schema decision was probed and resolved: tag + body convention (see B3 step 5).
- **Existing intent-skill infrastructure** — `inverted-invocation-policy.adr` and `intent-based-skill-architecture.adr` provide the auto-invocation contract. No changes there.
- **Existing type-skills** for rule, guide, doc, adr — bootstrap reuses their schemas; if their templates change, bootstrap output may need adjustment.

## Risks

- **B1 detection false positives.** "react" in dev-deps may not be the actual UI lib; "@types/react" alone shouldn't trigger. Mitigation: signal allowlist excludes `@types/*`, `eslint-*`, `prettier`, test runners, build tools. Always show preview before write.
- **B2 README extraction quality.** Marketing-heavy READMEs may yield no usable command blocks. Fallback to `scripts:` extraction is solid; if that also fails, ask user one open question and proceed manually.
- **B3 extract-mode quality.** Heuristic routing produces imperfect rules/ADRs. Mitigation: dry-run preview is mandatory; user sees titles + types before any write. Default mode is link (zero quality risk), extract is opt-in per file.
- **B3 cost estimate accuracy.** Heuristic 1 doc / 800 bytes is rough. Real yield may diverge ±50% on extreme inputs. Acceptable — the warning is directional, not contractual. Refine after first usage data.
- **Tag-spec compatibility.** The tag convention `source:<slug>` relies on the existing tag regex permitting colons. If a future plugin tightens the regex, the tag form breaks. Mitigation: lock the regex behaviour by adding a targeted unit test that creates a doc with `source:agents-md` and asserts persistence; if that ever fails, fall back to `source-agents-md` (hyphen-separated) — semantics preserved, idempotency lookup adapts.
- **Slug collisions.** Two source files slugify to the same `<slug>` (e.g., `.cursor/rules/styling.mdc` and `.cursor/rules/styling.md`). Mitigation: include the file extension in the slug (`source:cursor-rules-styling-mdc` vs `source:cursor-rules-styling-md`).
- **Stale source files.** When `AGENTS.md` is edited after import, the Archcore wrapper does not auto-refresh. Acceptable for v1 — `/archcore:actualize` already covers code-doc drift; extending it to imported sources is a future extension.
- **Idempotency edge cases.** User edits the bootstrap-generated stack rule, then re-runs bootstrap → rule already exists → skill prompts "regenerate?" → user says yes → user's edits lost. Mitigation: regenerate prompt explicitly warns "this will overwrite your edits to project-stack.rule.md" and shows a 1-line diff preview before proceeding.
- **Skill discovery dependency on Variant A.** Phase B is valuable only if users find it. Phase A (the nudge) is the discovery mechanism. They must ship together; do not stage them across releases.

## Relations

- `implements` → `zero-content-onboarding.idea` (this plan executes variants A and B; defers C)
- `extends` → `claude-plugin.prd` (adds FR-7 and FR-8)
- `extends` → `commands-system.spec` (registers new intent skill)
- `extends` → `skills-system.spec` (adds bootstrap to the intent-skill section)
- `extends` → `hooks-validation-system.spec` (Phase A modifies SessionStart behaviour)
- `related` → `inverted-invocation-policy.adr` (auto-invocation rules apply to bootstrap)
- `related` → `intent-based-skill-architecture.adr` (skill follows the standard intent contract)
- `related` → `skill-file-structure.rule` (SKILL.md must conform)
- `related` → `readme-first-60-seconds.idea` (adjacent funnel stage; both ship)
