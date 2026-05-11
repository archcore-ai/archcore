---
title: "Bootstrap Scale Modes — Small / Medium / Large with Tracked-Context Targets"
status: accepted
tags:
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

`/archcore:bootstrap` today produces two artifacts (stack rule + run guide) derived from manifests alone. On anything beyond a narrow single-domain library this is insufficient: the output doesn't say anything the agent couldn't read in 30 seconds from `package.json` / `go.mod` / `pyproject.toml`. On a multi-domain application it collapses into noise — no single bootstrap output can usefully cover a 50-module monolith.

Fix: **branch bootstrap by repo scale** (small / medium / large), seed scale-appropriate artifacts, and — crucially for this plan — document per-scale **context targets** so bootstrap's one-time seeding is understood as the first step of a longer capture trajectory, not the endpoint.

Scope:

- Language-agnostic core signals (no per-language parser; per-language data tables under `skills/bootstrap/lib/`).
- Extends `zero-content-onboarding-implementation.plan` (Phase B) — does not replace it.
- Defines: scale detection, per-scale bootstrap flow, per-scale context target list (the "plan of what to trace").

Out of scope:

- Automatic capture of everything in the target lists. Bootstrap only seeds bootstrap-designated artifacts; the rest are tracked as a todo for `/archcore:capture`, `/archcore:decide`, `/archcore:standard` to fill on demand.
- Cross-project syncing of target lists.
- Import-graph / call-graph analysis (deferred; requires per-language parsers).

## Scale detection

Three cheap, language-agnostic signals computed in one pass:

- **`domain_count`** — top-level subdirectories under conventional roots (`src/`, `app/`, `apps/`, `packages/`, `services/`, `internal/`, `modules/`, `domains/`, `pkg/`, `lib/`). Exclude conventional-utility names (`utils/`, `helpers/`, `common/`, `shared/`, `types/`, `constants/`, `test/`, `tests/`, `__tests__/`). A subdir counts as a domain only if it contains ≥ 2 source files > 50 LOC.
- **`module_count`** — total source files > 100 LOC (excluding test files, vendored deps, generated code, lockfiles). Language-independent counter using file-extension allowlist.
- **`entry_point_count`** — files matching entry-point patterns: `main.{go,py,rs,kt,java}`, `cmd/*/main.go`, `bin/*`, files with HTTP-route decorators (`@app.route`, `app.get/post`, `router.get/post`, `FastAPI()`, etc.). Heuristic, undercount tolerated.

**Classification thresholds:**

| Mode | Rule |
|---|---|
| **Small** (library, single-domain) | `domain_count ≤ 1` AND `module_count ≤ 15` |
| **Medium** (focused service, framework kit, mid-sized SDK) | `domain_count ≤ 2` AND `15 < module_count ≤ 40` |
| **Large** (multi-domain app, monorepo, modular monolith) | `domain_count ≥ 3` OR `module_count > 40` |

Thresholds live in `skills/bootstrap/lib/detect-scale.md` as editable data. Bootstrap prints the detected mode in its opening line with an override hint (`/archcore:bootstrap --mode=medium`).

---

## Target context per mode

The lists below enumerate what each kind of repo **should eventually contain** in `.archcore/`. Each item is marked:

- **[seed]** — bootstrap writes it directly.
- **[propose]** — bootstrap detects candidates and surfaces them as a to-capture list; user triggers `/archcore:capture`, `/archcore:decide`, or `/archcore:standard` per candidate.
- **[organic]** — no bootstrap involvement; added on demand as the work touches that area.

### Small mode — library / single-domain repo

Typical shape: one cohesive public surface (SDK, CLI, utility lib). Example scale: example-id-sdk (~20 modules, 1 domain, 1 entry as the library consumer).

Target context:

1. **Stack rule** — manifest-derived imperative stack. `[seed]`
2. **Run guide** — install/dev/test/build commands. `[seed]`
3. **Hotspot capture candidates** (3 recommended) — top 3 source modules by `LOC + test_LOC`, presented as "would you like to capture a spec for this?" list. `[propose]`
4. **ADR per deliberate non-trivial dependency** (typical: 2–4) — e.g. "use `openid-client` not `passport`", "use `tsup` not `esbuild` directly". Detected candidates = direct deps in stack-signal allowlist categories where ≥ 2 alternatives exist. `[organic]` — but bootstrap can surface "did you want to record a decision for X?" as the 4th entry of the propose list.
5. **Hotspot spec per critical module** (typical: 1–3) — contract + invariants for modules > 150 LOC with ≥ 1.5× test LOC ratio. For example-id-sdk example: `token-mutex`, `token-rotation`, `createAuthClient` contract. `[organic]` following propose list.
6. **Task-type per repeating extension pattern** (typical: 1–2) — detected when ≥ 3 sibling files share a common shape (`*-middleware.ts`, `*_adapter.go`). Presented as "you have N files of this shape; want a task-type doc for adding new ones?". `[propose]`
7. **Rule per cross-cutting convention** (typical: 1–2) — log prefix conventions, error-wrapping shape, test naming. `[organic]`

Total expected steady-state: 5–10 documents.

### Medium mode — focused service / framework kit / mid-sized SDK

Typical shape: one domain but non-trivial internal surface; has entry points (HTTP / CLI / workers) but few of them; has deliberate config choices worth capturing. Examples: a microservice (HTTP API + DB + one worker), a CLI tool with several commands, a framework starter kit.

Target context:

1. **Stack rule** — as in small. `[seed]`
2. **Run guide** — as in small. `[seed]`
3. **Entry-point inventory** (one `doc`) — lists detected entry files with one line per entry (what it exposes, what it depends on). Auto-generated; edited post-hoc. `[seed]`
4. **Hotspot capture candidates** (3–5 recommended). `[propose]`
5. **Cross-cutting rule candidate** (1 proposed at bootstrap) — e.g. detected "every handler wraps in `withAuth`"; ask "codify as rule?". `[propose]`
6. **ADR per architectural decision** (typical: 3–6) — persistence, auth, observability, serialization format, HTTP framework choice. `[organic]`
7. **Spec per hotspot module** (typical: 3–6) — same hotspot heuristic as small mode, broader net. `[organic]`
8. **Rule per cross-cutting concern** (typical: 2–4) — logging, error-handling, config loading, request-context propagation. `[organic]`
9. **Task-type for top 2–3 change patterns** — "add a new HTTP handler", "add a new CLI subcommand", "add a new worker job". `[organic]`

Total expected steady-state: 12–20 documents.

### Large mode — multi-domain application / monorepo / modular monolith

Typical shape: 3+ cohesive domains (billing, auth, catalog, notifications, …), many entry points across domains, possibly multiple services or apps. Examples: a large product backend, a pnpm monorepo with `apps/*` + `packages/*`.

Bootstrap cannot meaningfully seed per-domain artifacts in one pass — too much noise. Strategy: seed **a map of the shape**, then run narrow per-domain passes interactively.

Target context:

1. **Stack rule** (workspace-level) — top-level languages, framework mix, monorepo tooling. `[seed]`
2. **Run guide** (monorepo-aware) — per-app sections + workspace-level install/build/test. `[seed]`
3. **Top-level map** (one `doc`) — list of detected domains with: name, path, approximate module count, one-line auto-summary. Seeded; user corrects summaries post-hoc. `[seed]`
4. **Entry-point inventory** (one `doc`) — all detected entry points grouped by domain. `[seed]`
5. **Domain selection dialog** — "Detected N domains. Which are you working on now? (pick 1–3)" → tags selected domains on the top-level map (e.g. `domain:billing`). `[interactive during bootstrap]`
6. **Per selected domain**: run the small-mode hotspot capture-candidate proposal inside that domain's tree (3–5 modules). `[propose]` — deferred to `/archcore:capture` chain.
7. **ADR per domain-level architectural decision** (typical: 2–5 per domain) — each domain has its own decisions. `[organic]`
8. **Spec per hotspot module** (typical: 2–5 per selected domain) — hotspot heuristic applied within domain. `[organic]`
9. **Spec per cohesive domain boundary** (typical: 1 per neighbouring-domain pair that actually talks) — what's public between domains, what's private within. `[organic]`
10. **Cross-cutting rules** (global, repo-wide — typical: 4–8) — logging, error-handling, auth-check, transaction boundaries, telemetry, metrics, request ID propagation, feature flags. Detected as repeated patterns across ≥ 3 domains. `[organic]`, bootstrap flags candidates.
11. **Task-type for top 3–5 change patterns** (repo-wide or per-domain) — "add a new domain event", "add a new API endpoint", "add a new background job". `[organic]`

Total expected steady-state: 20–40+ documents depending on domain count. Acceptable — the map + per-domain `domain:*` tag lets `/archcore:context` scope queries.

---

## Per-mode bootstrap flow summary

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

Language-agnostic signal modules under `skills/bootstrap/lib/` — each a markdown data file editable without changing skill body:

- **`detect-scale.md`** — thresholds table, override semantics, exclude-lists for utility directories.
- **`detect-domains.md`** — conventional roots (`src/`, `app/`, …), utility-folder exclusion list, domain-cohesion rule (≥ 2 files > 50 LOC).
- **`detect-modules.md`** — source-extension allowlist per language; test-file patterns to exclude (`*_test.go`, `*.test.ts`, `test_*.py`, `*Test.java`, `*Spec.kt`); generated-code patterns (`*.pb.go`, `generated/`, `__generated__/`, `dist/`, `build/`).
- **`detect-entry-points.md`** — language-independent patterns (`main.*`, `cmd/*/main.go`) + per-language additions (HTTP-decorator regex, CLI-binary locations).
- **`detect-hotspots.md`** — LOC + test_LOC ranking formula; optional git-activity weighting via `git log --since=3.months --name-only --pretty=format: | sort | uniq -c` (falls back gracefully if git unavailable).
- **`detect-cross-cutting.md`** — repeated-pattern heuristics: N sibling files matching a glob with shared import, N functions wrapped in same middleware, N files starting with same log prefix.

---

## MVP slice status (2026-04-24)

First implementation pass delivered the lib files and a rewritten `SKILL.md` with full mode-branching. Scope decisions for this slice:

- **Delivered**: M1–M6 + M8 partial (SKILL.md rewrite only). Six `detect-*.md` data files written with language coverage for TS/JS, Python, Go (polyglot), and stub entries for Rust / Java / Kotlin / Ruby / PHP / C# / Swift / Scala / Elixir.
- **Validation approach**: manual on three real repos (small: example-id-sdk, medium: this plugin repo, large: any available monorepo). Synthetic fixtures and `bats` assertions (M7) deferred until after manual validation exposes rough edges.
- **Not yet touched**: `commands-system.spec`, `skills-system.spec`, `claude-plugin.prd` (M8 doc updates) — defer until behaviour stabilizes.
- **Open risks for validation**: (1) thresholds may misclassify borderline repos; (2) utility-exclusion list may still leak `src/shared` / `packages/core` style names in real monorepos; (3) hotspot ranking may tilt wrong when test files are distant from source.

---

## Tasks

### M1. Mode detection (blocking)

- [x] Implement signal counters (`domain_count`, `module_count`, `entry_point_count`) as skill-internal logic reading `detect-*.md` data.
- [x] Implement threshold classification and override flag parsing (`--mode=small|medium|large` in SKILL.md frontmatter).
- [x] Print detected mode + override hint in bootstrap opening line (Step 0.5).
- [ ] Unit fixture: library (small), single-app service (medium), pnpm monorepo with 3+ apps (large). Assert correct classification. — **Deferred (M7)**; manual validation first.

### M2. Medium-mode additions

- [x] Entry-point inventory doc generation (Step 4 in SKILL.md, template in `detect-entry-points.md`).
- [x] Cross-cutting rule candidate detection (Step 7 in SKILL.md, uses `detect-cross-cutting.md`); single candidate surfaced, not auto-created.

### M3. Large-mode additions

- [x] Top-level map doc generation (Step 3 in SKILL.md) — auto-summary uses domain README, manifest-per-domain, or shape fallback.
- [x] Entry-point inventory (monorepo-aware — groups by domain). Step 4 handles large-mode grouping.
- [x] Domain selection dialog (Step 5 in SKILL.md) — presents ranked domains (top 5 by activity + size), accepts 1–3 picks.
- [x] Per-domain hotspot capture-candidate pass (Step 6 scoped to selected domain subtrees).

### M4. Hotspot capture-candidate proposal (shared across modes)

- [x] Ranking module (`detect-hotspots.md`).
- [x] Presentation step (Step 6 in SKILL.md): list with one-line rationale per candidate, including suggested doc type (`spec`, `adr`, `rule`, `task-type`).
- [x] No auto-invocation of `/archcore:capture` — output is a todo list in the closing message so user walks through on their pace.

### M5. Language-agnostic signal libraries

- [x] Write the six `detect-*.md` files (`detect-scale`, `detect-domains`, `detect-modules`, `detect-entry-points`, `detect-hotspots`, `detect-cross-cutting`). Coverage: TS/JS + Python + Go polyglot; stubs for Rust, Java, Kotlin, Ruby, PHP, C#, Swift, Scala, Elixir.

### M6. Closing message with tracked-context todos

- [x] Per mode, closing message lists `[organic]` target items not yet seeded (Step "Closing message: outlook" in SKILL.md).
- [x] Per-mode templates (Small / Medium / Large) with specific calls to action (`/archcore:decide`, `/archcore:capture`, `/archcore:standard`, `/archcore:plan`).

### M7. Fixtures and tests (deferred)

- [ ] Small fixture — existing `example-id-sdk`-shaped repo (TS SDK).
- [ ] Medium fixture — synthetic Express API with ~20 modules + 1 HTTP entry + 1 worker (TS).
- [ ] Large fixture — synthetic pnpm monorepo with 4 apps under `apps/` + 3 packages under `packages/` + cross-cutting logger/auth.
- [ ] Small+Python fixture — small Python library with `pytest` tests.
- [ ] Small+Go fixture — small Go library using `go test`.
- [ ] `test/structure/bootstrap-modes.bats` assertions on output counts per mode.

Validation sequence before M7: run `/archcore:bootstrap` (dry-run where possible) on one real repo per tier, record unexpected outputs / misclassifications, then iterate on `detect-*.md` data. M7 fixtures lock in the behaviour we arrived at manually.

### M8. Documentation (partial)

- [x] Update `skills/bootstrap/SKILL.md` with the mode-branch flow.
- [ ] Extend `zero-content-onboarding-implementation.plan` with a cross-reference to this plan (parent plan stays authoritative for Phase A / agent-file import). — **Deferred**.
- [ ] Add a short "scale modes" subsection to the `/archcore:bootstrap` entry in `commands-system.spec`. — **Deferred**.

---

## Acceptance Criteria

1. On the small fixture (TS SDK shape), bootstrap produces: stack rule + run guide + 3-candidate capture list. Output ≤ 20 lines. Runs under 15s.
2. On the medium fixture (service), bootstrap produces: stack rule + run guide + entry-point inventory + 3–5 candidate capture list + 1 cross-cutting rule candidate.
3. On the large fixture (monorepo), bootstrap produces: workspace stack rule + monorepo run guide + top-level map + entry-point inventory, then asks domain selection, then produces per-domain capture candidates for selected domains.
4. Mode detection correctly classifies all fixtures. Override flag (`--mode=X`) forces the chosen mode regardless of signals.
5. Closing message in each mode cites the target-context outlook (seeded vs organic items).
6. All seeded artifacts are idempotent on re-run.
7. Language coverage validated on TS, Python, Go small fixtures.
8. No regression on the existing Phase A (SessionStart nudge) or Phase B3 (agent-file import) flows.

## Dependencies

- **Parent plan**: `zero-content-onboarding-implementation.plan` — this plan assumes Phase B1 (stack rule) and B2 (run guide) are already implemented and extends the flow around them.
- **Idea source**: `zero-content-onboarding.idea` — scale-mode branching is a refinement of the original bootstrap intent.
- **No CLI release dependency** — all reachable via existing MCP surface (`create_document`, `list_documents`, `add_relation`) and file-system reads.
- **Existing type-skills** for `rule`, `guide`, `doc`, `adr`, `spec`, `task-type` — bootstrap composes their content directly; if their templates change, bootstrap output may need adjustment.
- **Optional**: `git` CLI for hotspot weighting. Falls back gracefully when absent.

## Risks

- **Mode misclassification.** Thresholds are heuristic. Mitigation: always print detected mode + override flag. Keep thresholds in `detect-scale.md` as editable data. Over time, collect misclassification reports and tune.
- **Domain detection noise.** `src/utils/`, `src/lib/`, `src/common/` may register as "domains". Mitigation: exclude-list in `detect-domains.md`. Cohesion rule (≥ 2 files > 50 LOC) filters thin directories.
- **Entry-point pattern per-language bias.** Easy to over-tune for TS; under-detects for Java/Kotlin/Rust. Mitigation: data-driven lookup table per language; undercount is acceptable — agent can add missed entries post-hoc.
- **Large-mode domain-dialog fatigue.** User may bounce when asked about N domains. Mitigation: cap presented domains at 5 ranked by activity + size. Rest listed in closing message with "bootstrap others later with `/archcore:bootstrap --domain=X`".
- **Capture-candidate hand-off cost.** If user picks 5 candidates, that's 5 `/archcore:capture` invocations. Mitigation: present as todo list in closing message, not auto-invoked. User walks through on their own pace.
- **Target-context list staleness.** The per-mode target lists in this plan will drift as we learn what's actually useful. Mitigation: treat this plan as living; revisit after first 3 real-repo runs.
- **Over-seeding**. Large-mode seeding (stack rule + run guide + map + entry-points = 4 docs) may itself feel too much in borderline cases. Mitigation: threshold tuning + override flag + idempotent skip of any seed whose value is trivial (e.g. monorepo with one real app → degrade to medium-mode layout).

## Relations

- `extends` → `zero-content-onboarding-implementation.plan` (adds scale-branching on top of Phase B1/B2 + reuses B3)
- `implements` → `zero-content-onboarding.idea` (refines the bootstrap half of the idea)
- `related` → `claude-plugin.prd` (extends FR-8 bootstrap scope)
- `related` → `skills-system.spec` (bootstrap skill gains a mode-branch structure)
- `related` → `commands-system.spec` (override flag `--mode=X` documented here)
- `related` → `code-alignment-intent-skill.idea` (downstream consumer — richer seeds mean richer injection)
