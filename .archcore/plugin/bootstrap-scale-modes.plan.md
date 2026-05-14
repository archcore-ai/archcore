---
title: "Bootstrap (Now Init) Scale Modes — Small / Medium / Large with Tracked-Context Targets"
status: accepted
tags:
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

> **Outcome (2026-05-15):** Plan executed. Skill shipped as `skills/init/` (renamed from `skills/bootstrap/` per `skill-surface-collapse.adr.md`). Lib files live at `skills/init/lib/detect-*.md`. The command is `/archcore:init`. All references below to `/archcore:bootstrap` and `skills/bootstrap/` should be read as `/archcore:init` and `skills/init/`. The three-mode (small/medium/large) detection logic is preserved as designed.

## Goal

`/archcore:init` today produces two artifacts (stack rule + run guide) derived from manifests alone. On anything beyond a narrow single-domain library this is insufficient: the output doesn't say anything the agent couldn't read in 30 seconds from `package.json` / `go.mod` / `pyproject.toml`. On a multi-domain application it collapses into noise — no single init output can usefully cover a 50-module monolith.

Fix: **branch init by repo scale** (small / medium / large), seed scale-appropriate artifacts, and — crucially for this plan — document per-scale **context targets** so init's one-time seeding is understood as the first step of a longer capture trajectory, not the endpoint.

Scope:

- Language-agnostic core signals (no per-language parser; per-language data tables under `skills/init/lib/`).
- Extends `zero-content-onboarding-implementation.plan` (Phase B) — does not replace it.
- Defines: scale detection, per-scale init flow, per-scale context target list (the "plan of what to trace").

Out of scope:

- Automatic capture of everything in the target lists. Init only seeds init-designated artifacts; the rest are tracked as a todo for `/archcore:capture`, `/archcore:decide`, or `/archcore:plan` to fill on demand.
- Cross-project syncing of target lists.
- Import-graph / call-graph analysis (deferred; requires per-language parsers).

## Scale detection

Three cheap, language-agnostic signals computed in one pass:

- **`domain_count`** — top-level subdirectories under conventional roots (`src/`, `app/`, `apps/`, `packages/`, `services/`, `internal/`, `modules/`, `domains/`, `pkg/`, `lib/`). Exclude conventional-utility names (`utils/`, `helpers/`, `common/`, `shared/`, `types/`, `constants/`, `test/`, `tests/`, `__tests__/`). A subdir counts as a domain only if it contains ≥ 2 source files > 50 LOC.
- **`module_count`** — total source files > 100 LOC (excluding test files, vendored deps, generated code, lockfiles).
- **`entry_point_count`** — files matching entry-point patterns: `main.{go,py,rs,kt,java}`, `cmd/*/main.go`, `bin/*`, files with HTTP-route decorators.

**Classification thresholds:**

| Mode | Rule |
|---|---|
| **Small** (library, single-domain) | `domain_count ≤ 1` AND `module_count ≤ 15` |
| **Medium** (focused service, framework kit, mid-sized SDK) | `domain_count ≤ 2` AND `15 < module_count ≤ 40` |
| **Large** (multi-domain app, monorepo, modular monolith) | `domain_count ≥ 3` OR `module_count > 40` |

Thresholds live in `skills/init/lib/detect-scale.md` as editable data. Init prints the detected mode in its opening line with an override hint (`/archcore:init --mode=medium`).

---

## Target context per mode

The lists below enumerate what each kind of repo **should eventually contain** in `.archcore/`. Each item is marked:

- **[seed]** — init writes it directly.
- **[propose]** — init detects candidates and surfaces them as a to-capture list; user triggers `/archcore:capture`, `/archcore:decide`, or `/archcore:plan` per candidate.
- **[organic]** — no init involvement; added on demand as the work touches that area.

### Small mode — library / single-domain repo

Typical shape: one cohesive public surface (SDK, CLI, utility lib).

Target context:

1. **Stack rule** — manifest-derived imperative stack. `[seed]`
2. **Run guide** — install/dev/test/build commands. `[seed]`
3. **Hotspot capture candidates** (3 recommended) — top 3 source modules by `LOC + test_LOC`. `[propose]`
4. **ADR per deliberate non-trivial dependency** (typical: 2–4). `[organic]` — init can surface "did you want to record a decision for X?" as a propose-list entry.
5. **Hotspot spec per critical module** (typical: 1–3). `[organic]`
6. **Task-type per repeating extension pattern** (typical: 1–2). `[propose]`
7. **Rule per cross-cutting convention** (typical: 1–2). `[organic]`

Total expected steady-state: 5–10 documents.

### Medium mode — focused service / framework kit / mid-sized SDK

Typical shape: one domain but non-trivial internal surface; has entry points (HTTP / CLI / workers) but few of them.

Target context:

1. **Stack rule** — as in small. `[seed]`
2. **Run guide** — as in small. `[seed]`
3. **Entry-point inventory** (one `doc`). `[seed]`
4. **Hotspot capture candidates** (3–5 recommended). `[propose]`
5. **Cross-cutting rule candidate** (1 proposed at init). `[propose]`
6. **ADR per architectural decision** (typical: 3–6). `[organic]`
7. **Spec per hotspot module** (typical: 3–6). `[organic]`
8. **Rule per cross-cutting concern** (typical: 2–4). `[organic]`
9. **Task-type for top 2–3 change patterns** — "add a new HTTP handler", "add a new CLI subcommand", "add a new worker job". `[organic]`

Total expected steady-state: 12–20 documents.

### Large mode — multi-domain application / monorepo / modular monolith

Typical shape: 3+ cohesive domains, many entry points across domains.

Init cannot meaningfully seed per-domain artifacts in one pass — too much noise. Strategy: seed **a map of the shape**, then run narrow per-domain passes interactively.

Target context:

1. **Stack rule** (workspace-level). `[seed]`
2. **Run guide** (monorepo-aware). `[seed]`
3. **Top-level map** (one `doc`). `[seed]`
4. **Entry-point inventory** (one `doc`). `[seed]`
5. **Domain selection dialog** — "Detected N domains. Which are you working on now? (pick 1–3)". `[interactive during init]`
6. **Per selected domain**: run the small-mode hotspot capture-candidate proposal inside that domain's tree. `[propose]` — deferred to `/archcore:capture` chain.
7. **ADR per domain-level architectural decision** (typical: 2–5 per domain). `[organic]`
8. **Spec per hotspot module** (typical: 2–5 per selected domain). `[organic]`
9. **Spec per cohesive domain boundary** (typical: 1 per neighbouring-domain pair). `[organic]`
10. **Cross-cutting rules** (global, repo-wide — typical: 4–8). `[organic]`, init flags candidates.
11. **Task-type for top 3–5 change patterns**. `[organic]`

Total expected steady-state: 20–40+ documents. Acceptable — the map + per-domain `domain:*` tag lets `/archcore:context` scope queries.

---

## Per-mode init flow summary

| Step | Small | Medium | Large |
|---|---|---|---|
| 1. Detect scale | ✓ | ✓ | ✓ |
| 2. Announce mode + override hint | ✓ | ✓ | ✓ |
| 3. Stack rule (seed) | ✓ | ✓ | ✓ (workspace) |
| 4. Run guide (seed) | ✓ | ✓ | ✓ (monorepo) |
| 5. Top-level map (seed) | — | — | ✓ |
| 6. Entry-point inventory (seed) | — | ✓ | ✓ |
| 7. Hotspot capture candidates (propose list) | ✓ (3) | ✓ (3–5) | per selected domain (3–5) |
| 8. Cross-cutting rule candidate (propose) | — | ✓ (1) | — (deferred to per-domain pass) |
| 9. Domain selection dialog | — | — | ✓ |
| 10. Opt-in agent-file import (B3, existing) | ✓ | ✓ | ✓ |
| 11. Closing message with tracked-context todos | ✓ | ✓ | ✓ |

Closing message in every mode lists the "over-time targets" not yet created, so the user knows what the tracked context aims for and can drive captures on their own pace.

---

## Common infrastructure (shared across modes)

Language-agnostic signal modules under `skills/init/lib/`:

- **`detect-scale.md`** — thresholds table, override semantics, exclude-lists for utility directories.
- **`detect-domains.md`** — conventional roots, utility-folder exclusion list, domain-cohesion rule.
- **`detect-modules.md`** — source-extension allowlist per language; test-file patterns to exclude; generated-code patterns.
- **`detect-entry-points.md`** — language-independent patterns + per-language additions.
- **`detect-hotspots.md`** — LOC + test_LOC ranking formula; optional git-activity weighting.
- **`detect-cross-cutting.md`** — repeated-pattern heuristics.

---

## MVP slice status (2026-04-24)

First implementation pass delivered the lib files and a rewritten `SKILL.md` with full mode-branching. Scope decisions for this slice:

- **Delivered**: M1–M6 + M8 partial. Six `detect-*.md` data files written with language coverage for TS/JS, Python, Go (polyglot), and stub entries for Rust / Java / Kotlin / Ruby / PHP / C# / Swift / Scala / Elixir.
- **Validation approach**: manual on three real repos.
- **Not yet touched**: doc spec updates (M8 doc updates) — deferred until behaviour stabilizes.

---

## Tasks

### M1. Mode detection (blocking)

- [x] Implement signal counters (`domain_count`, `module_count`, `entry_point_count`).
- [x] Implement threshold classification and override flag parsing (`--mode=small|medium|large` in SKILL.md frontmatter).
- [x] Print detected mode + override hint in init opening line.
- [ ] Unit fixture: library (small), single-app service (medium), pnpm monorepo with 3+ apps (large). — **Deferred (M7)**.

### M2. Medium-mode additions

- [x] Entry-point inventory doc generation.
- [x] Cross-cutting rule candidate detection.

### M3. Large-mode additions

- [x] Top-level map doc generation.
- [x] Entry-point inventory (monorepo-aware).
- [x] Domain selection dialog.
- [x] Per-domain hotspot capture-candidate pass.

### M4. Hotspot capture-candidate proposal (shared across modes)

- [x] Ranking module (`detect-hotspots.md`).
- [x] Presentation step.
- [x] No auto-invocation of `/archcore:capture`.

### M5. Language-agnostic signal libraries

- [x] Write the six `detect-*.md` files.

### M6. Closing message with tracked-context todos

- [x] Per mode, closing message lists `[organic]` target items not yet seeded.
- [x] Per-mode templates with specific calls to action (`/archcore:decide`, `/archcore:capture`, `/archcore:plan`).

### M7. Fixtures and tests (deferred)

- [ ] Small fixture.
- [ ] Medium fixture — synthetic Express API.
- [ ] Large fixture — synthetic pnpm monorepo.
- [ ] Small+Python fixture.
- [ ] Small+Go fixture.
- [ ] `test/structure/init-modes.bats` assertions.

### M8. Documentation (partial)

- [x] Update `skills/init/SKILL.md` with the mode-branch flow.
- [ ] Extend `zero-content-onboarding-implementation.plan` with a cross-reference to this plan. — **Deferred**.
- [ ] Add a short "scale modes" subsection to the `/archcore:init` entry in `commands-system.spec`. — **Deferred**.

---

## Acceptance Criteria

1. On the small fixture (TS SDK shape), init produces: stack rule + run guide + 3-candidate capture list.
2. On the medium fixture (service), init produces: stack rule + run guide + entry-point inventory + 3–5 candidate capture list + 1 cross-cutting rule candidate.
3. On the large fixture (monorepo), init produces: workspace stack rule + monorepo run guide + top-level map + entry-point inventory, then asks domain selection, then produces per-domain capture candidates.
4. Mode detection correctly classifies all fixtures. Override flag forces the chosen mode.
5. Closing message in each mode cites the target-context outlook.
6. All seeded artifacts are idempotent on re-run.
7. Language coverage validated on TS, Python, Go small fixtures.
8. No regression on the existing Phase A (SessionStart nudge) or Phase B3 (agent-file import) flows.

## Dependencies

- **Parent plan**: `zero-content-onboarding-implementation.plan` — assumes Phase B1 (stack rule) and B2 (run guide) are already implemented.
- **Idea source**: `zero-content-onboarding.idea`.
- **No CLI release dependency** — all reachable via existing MCP surface.
- **Existing intent skills** (`capture`, `decide`, `plan`) — init produces propose lists that route to them.
- **Optional**: `git` CLI for hotspot weighting.

## Risks

- **Mode misclassification.** Thresholds are heuristic. Mitigation: always print detected mode + override flag.
- **Domain detection noise.** Mitigation: exclude-list in `detect-domains.md`.
- **Entry-point pattern per-language bias.** Mitigation: data-driven lookup table per language.
- **Large-mode domain-dialog fatigue.** Mitigation: cap presented domains at 5 ranked by activity + size.
- **Capture-candidate hand-off cost.** Mitigation: present as todo list in closing message, not auto-invoked.
- **Target-context list staleness.** Mitigation: treat this plan as living.
- **Over-seeding.** Mitigation: threshold tuning + override flag + idempotent skip.

## Relations

- `extends` → `zero-content-onboarding-implementation.plan` (adds scale-branching on top of Phase B1/B2 + reuses B3)
- `implements` → `zero-content-onboarding.idea` (refines the init half of the idea)
- `related` → `claude-plugin.prd` (extends FR-8 init scope)
- `related` → `skills-system.spec`
- `related` → `commands-system.spec`
- `related` → `skill-surface-collapse.adr` (the rename `bootstrap` → `init`)
- `related` → `code-alignment-intent-skill.idea`
