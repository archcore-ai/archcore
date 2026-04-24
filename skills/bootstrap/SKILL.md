---
name: bootstrap
argument-hint: ""
description: "First-time onboarding: seed an empty .archcore/ with a short stack rule, a run-the-app guide, and (optionally) imports from existing agent-instruction files (CLAUDE.md, AGENTS.md, .cursorrules, etc.). Activate when user says 'bootstrap archcore', 'initialize archcore', 'set up archcore', 'seed archcore', 'first-time setup', 'what should I do first', or asks how to start after a fresh install. Do NOT activate for creating individual documents (use /archcore:capture, /archcore:decide, /archcore:standard), for feature planning (use /archcore:plan), for documentation audits (use /archcore:review), or for loading existing context (use /archcore:context)."
---

# /archcore:bootstrap

First-time onboarding. Generates a small, useful starting set of `.archcore/` documents so later push-mode (`check-code-alignment` hook) and pull-mode (`/archcore:context`) have something to inject. Steps 1–2 generate directly; Step 3 is opt-in and previewed. If the generated stack rule or run guide isn't right, edit or delete the file after — they're a few lines each.

_Run this once on a fresh repo. Re-running is safe: each step detects existing artifacts and asks before overwriting._

## When to use

- "Bootstrap archcore"
- "Initialize archcore / set up archcore"
- "Seed archcore / what should I do first"
- Empty `.archcore/` with the SessionStart nudge pointing here

**Not bootstrap:**
- Document creation → `/archcore:capture`, `/archcore:decide`, `/archcore:standard`
- Feature planning → `/archcore:plan`
- Docs audit → `/archcore:review`
- Loading existing context → `/archcore:context`

## Execution

### Step 0: Check state

Call `mcp__archcore__list_documents()`. Derive:

- `has_stack_rule` — any `rule` whose title contains "stack" in a `conventions/` directory
- `has_run_guide` — any `guide` whose title contains "run" or "running" in an `onboarding/` directory
- `has_imports` — any document with tag `imported`

If all three are true, the repo is already bootstrapped. Reply with:

> Bootstrap already ran. You have a stack rule, a run guide, and imported agent-files. Use `/archcore:context` to see what's loaded, or re-run specific steps (e.g. "regenerate the stack rule") on demand.

Otherwise proceed. Mention in the opening line that this is a three-step flow — Steps 1 and 2 generate directly; Step 3 (import) is opt-in.

### Step 1: Stack rule

1. **Check idempotency.** If `has_stack_rule` is true, show the existing rule's title and path. Ask: *"Stack rule exists. Regenerate (overwrite), skip this step, or keep and continue?"* Honor the user's choice. On regenerate, warn that manual edits will be lost.

2. **Detect the stack.** Read `skills/bootstrap/lib/detect-stack.md` for the manifest list, the signal-bearing dependency allowlist, exclusions, and the rule template.

    Read in order, stopping at the first match per language: `package.json`, `pyproject.toml`, `Pipfile`, `requirements.txt`, `Cargo.toml`, `go.mod`, `Gemfile`, `composer.json`, `*.csproj`, `pom.xml`, `build.gradle*`. Polyglot repos are allowed — collect signals from each manifest you find.

    Extract top-level (declared, not transitive) dependencies. Apply the allowlist from `detect-stack.md`. Apply the exclusions (`@types/*`, `eslint-*`, `prettier`, test runners, build tools). Cap the final signal set at **5 items total** across all manifests.

    If no manifest exists, fall back to file-extension detection: scan the top-level source directories (`src/`, `lib/`, `app/`, or the repo root) for the majority language(s), up to 2.

3. **Compose the body.** Use the template in `detect-stack.md`. Drop any template line whose placeholder has no detected signal — never leave placeholders unfilled. Stay imperative. No versions. No library enumerations. The final rule body should be ≤ 6 lines.

4. **Create.** Call `mcp__archcore__create_document` with:
    - `type: 'rule'`
    - `filename: 'project-stack'`
    - `directory: 'conventions'`
    - `title: 'Project stack'`
    - `status: 'accepted'`
    - `content: <the composed body>`
    - `tags: ['stack', 'conventions']`

    Report one line: detected signals + resulting path. Example: *"Stack: TypeScript, React, Node → `.archcore/conventions/project-stack.rule.md`"*. If the user wants to edit, they can open the file or say "regenerate the stack rule".

### Step 2: Run-the-app guide

1. **Check idempotency.** If `has_run_guide` is true, show the existing guide's title and path. Ask: *"Run guide exists. Regenerate (overwrite), skip this step, or keep and continue?"* Honor the user's choice. On regenerate, warn that manual edits will be lost.

2. **Detect shape.** Read `skills/bootstrap/lib/extract-run-instructions.md` for monorepo markers, README-section regex, command-block extraction rules, and the template.

    First, detect monorepo by looking for any of: `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `lerna.json`, OR multiple `package.json` files under `apps/` or `packages/` (at least 2). If detected, the guide gets per-app subsections plus a workspace-level section.

3. **Extract commands.** Two paths, stop at the first that yields usable commands:

    - **README path** — read `README.md` (or `README.{en,ru,*}.md` if the plain one is absent). Find the first section matching the regex in `extract-run-instructions.md`. Pull fenced ```bash / ```sh / ```shell / ```zsh blocks from that section. Keep only commands that plausibly install, run, or test (filter in `extract-run-instructions.md`).
    - **Scripts path** — if README yields nothing, read `scripts:` from `package.json` (or the equivalent section for other languages: `[tool.poetry.scripts]`, `Cargo.toml` `[[bin]]`, `Gemfile` rake tasks via `Rakefile`, `composer.json` `scripts`). Pick `dev`, `start`, `build`, `test`, `lint` if present.
    - **Still nothing** — ask the user one open question: *"I couldn't extract run commands automatically. In one line: how do you run this app locally?"* Use the answer verbatim.

4. **Detect prerequisites.** Look for runtime-version declarations: `engines` in `package.json`, `python` in `pyproject.toml` `[project]`, `rust-version` in `Cargo.toml`, `go` directive in `go.mod`. State them as-is. Do not invent prerequisites the manifests don't declare.

5. **Compose the body.** Use the template in `extract-run-instructions.md`. Single-app vs monorepo has separate skeletons there. The total body must be ≤ 15 lines for single-app; monorepos may go longer but keep each app's subsection ≤ 6 lines.

    Strip marketing prose — never copy flavor text from the README; only the commands and the detected prerequisite lines.

6. **Create.** Call `mcp__archcore__create_document` with:
    - `type: 'guide'`
    - `filename: 'running-the-project'`
    - `directory: 'onboarding'`
    - `title: 'Running the project locally'`
    - `status: 'accepted'`
    - `content: <the composed body>`
    - `tags: ['onboarding']`

    Report one line: command source + resulting path. Example: *"Run commands from `README.md` → `.archcore/onboarding/running-the-project.guide.md`"*. If the user wants to edit, they can open the file or say "regenerate the run guide".

### Step 3: Import agent-instruction files (opt-in)

This step is **opt-in**. It is the slowest and most token-intensive step; always ask before starting.

1. **Detect candidates.** Read `skills/bootstrap/lib/agent-files.md` for the canonical list of probe paths. For each path or glob: check existence, measure byte size. Collect the set of existing files.

    If the set is empty, announce: *"No agent-instruction files found. Nothing to import."* and finish.

2. **Estimate cost.** Sum combined byte size and file count. Estimate document yield as `ceil(combined_bytes / 800)`, capped at 25. Token cost rough estimate: `combined_bytes * 2` for extract mode, `~200 * file_count` for link mode.

3. **Classify cost tier.** Trigger the `HIGH COST` warning if **any** of:
    - combined size > **50 KB**
    - file count > **5**
    - estimated yield > **8 documents**

4. **Prompt the user.** Show the list of detected files with per-file size. Then:

    - **Normal cost** — *"Found N files (X KB). Parsing will create up to ~Y documents. **do** / skip?"*
    - **HIGH cost** — prefix with `⚠️ HIGH COST:` and state that `do` must be typed explicitly (Enter / y / yes alone are not enough).

    If user says skip (or declines the `do` for HIGH COST), exit Step 3 cleanly.

5. **Skip already-imported files.** Call `mcp__archcore__list_documents(tags=['imported'])`. For each detected file, compute its slug (see "Source slugging" in `lib/agent-files.md`). If any existing document carries the tag `source:<slug>`, mark that file as already imported and exclude from this run. Report: *"Skipping N files already imported."*

6. **Per-file mode.** For each remaining file, ask: *"`{path}` ({size}) — link (default), extract, or skip?"*

    - **link** (recommended) — create one `doc` per file. Body is a **single-line pointer** (see below); no content is copied. Fast, zero duplication.
    - **extract** — read the file body, split it into semantic blocks, route each block per `lib/extract-routing.md` into `rule` / `adr` / `doc` documents. Slow, produces more documents, heuristic quality risk.
    - **skip** — do not process this file in this run.

    Accept batch answers ("link all" / "skip all") when the user gives them.

7. **Source-of-truth representation (tag + body convention).** The CLI MCP currently strips unknown frontmatter fields, so sources are encoded in tags + body:

    - **Tags (required on every imported document):**
        - `imported` — literal marker, enables blanket queries.
        - `source:<slug>` — slugified source filename. Slug rules:
            - lowercase, alphanumeric + hyphens only
            - dots → hyphens, slashes → hyphens
            - collapse repeated hyphens
            - preserve the extension segment (prevents `.md`/`.mdc` collisions)
            - leading `.` in a filename is dropped before slugging
        - Examples: `AGENTS.md` → `source:agents-md`; `.cursorrules` → `source:cursorrules`; `.cursor/rules/styling.mdc` → `source:cursor-rules-styling-mdc`.
    - **Body first line (required, exact format):**
        ```
        > Imported from `<exact-relative-path>` on <ISO-8601-date>.
        ```
        Use the relative path from the repo root (e.g., `AGENTS.md`, `.cursor/rules/styling.mdc`). Use the current date in `YYYY-MM-DD` form.

8. **Build the create list.**

    - **Link mode per file** → one `doc` with:
        - `type: 'doc'`
        - `title: 'Imported: <basename of source>'`
        - `directory: 'imports'`
        - `filename: 'imported-<slug>'`
        - `status: 'accepted'`
        - `tags: ['imported', 'source:<slug>']`
        - `content:` — ONLY the body first line (pointer) and nothing else. Target body length < 200 bytes (the empty-state threshold) so the imported stub does not defeat the nudge on re-install into an otherwise-empty repo.

    - **Extract mode per file** → as many documents as semantic blocks, per the routing in `lib/extract-routing.md`. Each created document carries the same `imported` + `source:<slug>` tags and the same body first-line pointer, followed by the extracted content. Add a `related` edge from each extracted document back to a single umbrella `doc` (create the umbrella first, via link-mode rules).

9. **Dry-run preview.** Before any `create_document` calls, show the user a flat list:

    > Will create N documents: X rule(s), Y adr(s), Z doc(s). Confirm? (y/n)

    On `n`, cancel all creates for Step 3 without partial state.

10. **Batch execute.** For each planned document: `mcp__archcore__create_document(...)`. After all creates succeed, add `related` edges between extracted blocks and their umbrella docs via `mcp__archcore__add_relation`. On individual create failure: surface the error, continue with the remaining items (roll-forward; do not delete successful creates).

11. **Report.** One-line summary per file: *"`<path>` → created N documents (link/extract)"* or *"`<path>` → skipped"*.

### Final message

Summarize what was created and what was skipped. Close the loop with:

> Done. Edit a file under a path mentioned in the stack rule and you'll see relevant context injected automatically via `check-code-alignment`. Run `/archcore:context` anytime to surface what applies to a code area.

## Result

Up to several new documents in `.archcore/`: a short imperative stack rule, a run-the-app guide, and optional per-file imports of existing agent-instruction files. Each step is independently skippable; all creates are preview-then-confirm; re-runs are idempotent.
