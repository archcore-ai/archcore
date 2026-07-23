---
name: verify-plugin-integrity
description: Validate plugin format conformance for Claude Code, Cursor, and Codex CLI — statically audits the plugin manifests under plugins/archcore/ (.claude-plugin/, .cursor-plugin/, .codex-plugin/), the repo-root marketplace catalogs (incl. .agents/plugins/marketplace.json), SKILL.md frontmatter, MD + TOML agent files, Codex commands/*.md wrappers, hooks JSON for all three hosts, plugin-shipped MCP wiring (.mcp.json + .codex.mcp.json), Cursor rules, and bin/ scripts against the official Claude Code plugin spec, Cursor plugin spec, OpenAI Codex CLI plugin docs, and Agent Skills specification. No test execution. Use after structural changes to manifests, skills, agents, hooks, or rules; before opening a PR touching plugin structure; or when multi-host (Claude + Cursor + Codex) consistency is in doubt. For the bats test suite, use /archcore:verify instead.
disable-model-invocation: true
---

# /verify-plugin-integrity

Static format-conformance audit for the archcore multi-host plugin. Validates that every plugin artifact matches the official Claude Code, Cursor, OpenAI Codex CLI, and Agent Skills specifications, plus the normative Archcore specs in this repo.

**Does not** execute tests, lint, scripts, or hooks — for that, use `/archcore:verify` (which runs the bats test suite).

---

## Layout (post-relocation — read this first)

The **plugin root is the `plugins/archcore/` subdirectory**, not the repo root. All host-runtime-loaded content lives there: `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `.codex-plugin/plugin.json`, `skills/`, `agents/`, `commands/`, `hooks/`, `bin/`, `rules/`, `assets/`, `.mcp.json`, `.codex.mcp.json`.

The three marketplace catalogs stay at the **repo root** (`.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`, `.agents/plugins/marketplace.json`), each pointing `source`/`path` at `./plugins/archcore`. Rationale: Codex never scans a marketplace root for a plugin — a catalog `source.path` of `./` is silently undiscoverable (issue #2). See `.archcore/plugin/subdirectory-plugin-layout.adr.md`.

**All file paths in the sections below are plugin-root-relative (i.e. under `plugins/archcore/`) unless explicitly marked "repo root".**

---

## Authoritative sources (pin these; do not guess)

### Claude Code (code.claude.com)
- Plugins Reference — https://code.claude.com/docs/en/plugins-reference.md
- Plugins Guide — https://code.claude.com/docs/en/plugins.md
- Agent Skills Guide — https://code.claude.com/docs/en/skills.md
- Hooks Reference — https://code.claude.com/docs/en/hooks.md

### Agent Skills (open standard)
- Specification — https://agentskills.io/specification

### Cursor (cursor.com)
- Plugins Overview — https://cursor.com/docs/plugins
- Plugins Reference (manifest fields) — https://cursor.com/docs/plugins/building
- Plugin spec repo — https://github.com/cursor/plugins
- Rules — https://cursor.com/docs/context/rules
- MCP — https://cursor.com/docs/context/mcp
- Hooks — https://cursor.com/docs/hooks

### OpenAI Codex CLI
- Codex plugins overview — https://developers.openai.com/codex/plugins
- Build plugins guide — https://developers.openai.com/codex/plugins/build
- Codex CLI repo — https://github.com/openai/codex
- When upstream docs are unreachable or in flux, the internal Codex ground truth is `.archcore/plugin/codex-local-plugin-testing.guide.md` (Step 3 invariants), enforced by `test/structure/codex-plugin.bats`.

### Archcore internal conformance (normative for this repo)
- `.archcore/plugin/plugin-architecture.spec.md` — seven-skill intent surface, invocation flags
- `.archcore/plugin/subdirectory-plugin-layout.adr.md` — `plugins/archcore/` layout, catalog `source` rules (issue #2)
- `.archcore/plugin/hooks-validation-system.spec.md` — hook entries, events, anti-regression
- `.archcore/plugin/codex-host-support.prd.md` — Codex packaging functional requirements (F1–F10)
- `.archcore/plugin/codex-local-plugin-testing.guide.md` — live Codex packaging contract (manifest pointers, `${PLUGIN_ROOT}` hook commands, MCP shape, marketplace discovery)
- `.archcore/plugin/agent-system.spec.md` — agent frontmatter + bootstrap preamble
- `.archcore/plugin/skill-file-structure.rule.md` — skill file binding format
- `.archcore/plugin/component-registry.doc.md` — authoritative component counts and per-host config table
- `test/structure/*.bats` — the enforcement layer; where prose docs lag, the bats assertions are the operative pin

**Do NOT cite `.archcore/plugin/multi-host-compatibility-layer.spec.md` as ground truth** — it is rejected/tombstoned (superseded by `remove-bundled-launcher-global-cli.idea`); its body now only points at the live documents above.

If a local spec conflicts with an external official doc, the external doc is ground truth for format; the Archcore docs + bats suite are ground truth for this plugin's *choices* within that format (counts, invocation flags, Codex-specific manifest fields, etc.). For Codex specifically, `codex-local-plugin-testing.guide.md` + `test/structure/codex-plugin.bats` are authoritative whenever the upstream Codex docs are silent or in flux.

---

## When to use

- After editing any file under `plugins/archcore/` (`.claude-plugin/`, `.cursor-plugin/`, `.codex-plugin/`, `skills/`, `agents/`, `commands/`, `hooks/`, `rules/`, `bin/`)
- After changing the repo-root marketplace catalogs (`.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`, `.agents/plugins/marketplace.json`)
- After changing `.mcp.json` / `.codex.mcp.json` at the plugin root, or `docs/cursor.mcp.example.json` at the repo root
- Before opening a PR that touches plugin structure
- When cross-host (Claude + Cursor + Codex) manifest consistency is in doubt
- **Not** for test runs — use `/archcore:verify`
- **Not** for Archcore document freshness vs code — use `/archcore:audit --drift`

---

## Execution

Work entirely with `Read`, `Grep`, and `Bash` (e.g. `jq`, `ls`, `wc -l`). Do **not** call MCP tools — this skill creates no Archcore documents.

Walk through every section below. Record each check as PASS / FAIL / WARN with a one-line reason. Collate into the final report (see Output Format).

### Section 1 — Claude Code plugin manifest

File: `.claude-plugin/plugin.json` (plugin root, i.e. `plugins/archcore/.claude-plugin/plugin.json`)

Per https://code.claude.com/docs/en/plugins-reference.md:

- JSON parses
- `name` present, kebab-case, alphanumeric + hyphens
- `description`, `version` (semver), `author` (object with `name`), `license`, `repository` present (all optional per spec, but the repo convention requires them)
- Any directory override fields (`skills`, `agents`, `commands`, `hooks`, `mcpServers`) start with `./` and resolve to existing paths
- **Forbidden here**: `mcpServers` field inside the manifest itself — MCP lives in `.mcp.json` at the plugin root, not in the plugin manifest
- **Forbidden here**: `rules` field (Cursor-only; not auto-discovered by Claude Code)

### Section 2 — Cursor plugin manifest

File: `.cursor-plugin/plugin.json` (plugin root)

Per https://cursor.com/docs/plugins/building:

- JSON parses
- `name` kebab-case
- `description`, `version`, `author`, `license`, `repository`, `keywords` present
- `hooks` field points to `hooks/cursor.hooks.json` (Cursor-specific file; overrides default `hooks/hooks.json`)
- `skills`, `agents`, `rules` fields (if set) resolve to existing directories
- **Forbidden**: `mcpServers` field inside the manifest (lives in `.cursor/mcp.json` or `~/.cursor/mcp.json` per Cursor docs; not inside plugin manifest)

### Section 3 — Codex CLI plugin manifest

File: `.codex-plugin/plugin.json` (plugin root)

Per `.archcore/plugin/codex-host-support.prd.md` F1 and `.archcore/plugin/codex-local-plugin-testing.guide.md` Step 3, enforced by `test/structure/codex-plugin.bats`:

- JSON parses
- `name` present, kebab-case
- `description`, `version` (semver) present
- Component pointers `skills`, `hooks`, `mcpServers` are present and prefixed with `./` (Codex requires plugin-root-relative paths, i.e. relative to `plugins/archcore/`)
  - `hooks` MUST equal `./hooks/codex.hooks.json`
  - `mcpServers` MUST be a string pointer equal to `./.codex.mcp.json` (Codex resolves this file at plugin load — NOT an inline object)
  - `skills` MUST resolve to `./skills/`
- `interface{}` block present with at minimum `displayName`, `shortDescription`, `longDescription`, `developerName`, `category`, `capabilities`
  - `capabilities` contains `Read` and `Write` — and MUST NOT contain `Interactive` (Codex documents only `Read`/`Write` capability values; `codex-plugin.bats` asserts `.interface.capabilities | index("Interactive")` is null)
- **Forbidden**: legacy top-level UI fields. Any of `displayName`, `category`, or `tags` at the top level (outside `interface{}`) is a hard FAIL per `test/structure/codex-plugin.bats`
- **Forbidden**: an inline `mcpServers` object inside the manifest. Codex requires the pointer-to-file shape; an inline object is non-conformant
- Sibling files: `.codex-plugin/` MUST contain ONLY `plugin.json`. Any additional file (notably a stray `marketplace.json`) is a hard FAIL — Codex marketplace metadata lives at the repo root (see Section 5)

### Section 4 — Cross-host consistency

Compare `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, and `.codex-plugin/plugin.json` (all at the plugin root). The following fields MUST be byte-identical across all three:

- `name`
- `description`
- `version`

Flag any drift — this is the most common regression when bumping versions on one host but forgetting another. Enforced by the cross-host parity assertions in `test/structure/codex-plugin.bats`.

### Section 5 — Marketplace catalogs (repo root)

Files (host-specific locations, all at the **repo root**):

- Claude Code: `.claude-plugin/marketplace.json`
- Cursor: `.cursor-plugin/marketplace.json`
- Codex CLI: `.agents/plugins/marketplace.json` (NOT under `.codex-plugin/`)

Per Claude Code, Cursor, and OpenAI Codex plugin docs:

**Claude Code & Cursor** — same shape:
- JSON parses
- `name` present, kebab-case
- `owner.name` present
- `plugins` is a non-empty array; each entry has `name` and `source`
- `source` equals `./plugins/archcore`

**Codex CLI** — distinct schema (per `.archcore/plugin/codex-host-support.prd.md` F2 and `test/structure/codex-plugin.bats`):
- File path is `.agents/plugins/marketplace.json` (repo root)
- `name` equals `"archcore-plugins"` (note: marketplace name, NOT the plugin name)
- `interface.displayName` present
- `plugins` array; each entry has:
  - `name` (string, the plugin identifier — `"archcore"`)
  - `source` is an **object** with `source: "local"` and `path: "./plugins/archcore"` (Codex marketplace `source` is structured, not a string path like Claude/Cursor)
  - `policy.installation` equals `"INSTALLED_BY_DEFAULT"`
  - `policy.authentication` equals `"ON_INSTALL"`
  - `category` present and byte-equal to the manifest's `interface.category` (currently `"Productivity"`; `codex-plugin.bats` pins the equality)
- `.codex-plugin/marketplace.json` MUST NOT exist at either root (legacy path; FAIL if present)

**Subdirectory-source regression guard** (per `subdirectory-plugin-layout.adr.md` issue #2, enforced by `test/structure/marketplace-discovery.bats`):
- Every catalog `source`/`source.path` must resolve (relative to the repo root) to an existing directory that is **NOT** the marketplace root itself, stays inside the repo, and contains the matching per-host manifest (`.claude-plugin/plugin.json` / `.cursor-plugin/plugin.json` / `.codex-plugin/plugin.json`). Manifest-presence alone is NOT sufficient — a root `source` passed under the bug.
- All three catalog source strings must be identical (`./plugins/archcore` — single source of truth).

### Section 6 — Skills frontmatter audit

Iterate `skills/*/SKILL.md` (plugin root). For each, per https://agentskills.io/specification and https://code.claude.com/docs/en/skills.md:

- YAML frontmatter parses
- `name` is required, ≤ 64 chars, kebab-case (lowercase + hyphens), and **equals the parent directory name**
- `description` is required, ≤ 1024 chars, non-empty
- No unknown top-level fields (allowed: `name`, `description`, `license`, `metadata`, `compatibility`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`, `arguments`, `model`, `effort`, `context`, `agent`, `paths`, `shell`, `hooks`)

Then enforce this repo's surface rules (`.archcore/plugin/plugin-architecture.spec.md` — Seven-Skill Intent Surface; `skill-surface-collapse.adr.md`):

- Exactly **7 skills** on disk (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`), all auto-invocable intent skills
- No skill may carry `disable-model-invocation` or `user-invocable: false`
- No `description` may start with `"Advanced — "` (the track-skill tier was collapsed; its reappearance is a regression)
- `skills/_shared/` holds runtime assets, has no SKILL.md, and is excluded from the count

Per `.archcore/plugin/codex-host-support.prd.md` F7, all SKILL.md files MUST work unchanged in Codex; non-standard frontmatter fields (e.g., `argument-hint`) are tolerated by Codex's loader. No Codex-specific skill validation is required.

Total count sanity check: `ls -d plugins/archcore/skills/*/SKILL.md | wc -l` should match the registry total in `.archcore/plugin/component-registry.doc.md`.

### Section 7 — Markdown agents (Claude Code + Cursor)

Iterate `agents/*.md` (plugin root). Per https://code.claude.com/docs/en/plugins-reference.md (agents section):

- YAML frontmatter parses
- Required: `name`, `description`
- Optional (allowed): `model`, `effort`, `maxTurns`, `tools`, `disallowedTools`, `skills`, `memory`, `background`, `isolation` (only valid value: `"worktree"`), `color`
- **Forbidden in plugin agents**: `hooks`, `mcpServers`, `permissionMode` (Claude Code security restriction — plugin agents cannot configure these)

Archcore-specific (`.archcore/plugin/agent-system.spec.md`):
- Body contains the literal heading `# First Step — Bootstrap Knowledge Tree`
- Body grep-matches the anchor phrase `recent accepted decisions`

### Section 8 — TOML agents (Codex CLI)

Iterate `agents/*.toml` (plugin root). Per `.archcore/plugin/codex-host-support.prd.md` F8 and `.archcore/plugin/agent-system.spec.md`, enforced by `test/structure/agents.bats`:

- TOML parses
- Required keys: `name`, `description`, `developer_instructions`, `sandbox_mode`
- `name` matches the file basename (e.g., `archcore-auditor.toml` → `name = "archcore-auditor"`)
- For each MD agent in `agents/`, a TOML counterpart with the same basename MUST exist (currently: `archcore-assistant`, `archcore-auditor`)
- Additional Codex-specific keys (e.g. `model_reasoning_effort`) are tolerated

Per-agent rules:

- `archcore-auditor.toml` MUST have `sandbox_mode = "read-only"` AND `disabled_tools` containing all five mutating MCP tools: `mcp__archcore__create_document`, `mcp__archcore__update_document`, `mcp__archcore__remove_document`, `mcp__archcore__add_relation`, `mcp__archcore__remove_relation`
- `archcore-assistant.toml` MUST have `sandbox_mode = "workspace-write"` and SHOULD NOT declare `disabled_tools` (assistant has full MCP access by design)

Bootstrap preamble (parity with MD originals — `.archcore/plugin/agent-system.spec.md`):
- `developer_instructions` contains the literal heading `# First Step — Bootstrap Knowledge Tree`
- `developer_instructions` grep-matches the anchor phrase `recent accepted decisions`

### Section 9 — Claude Code hooks

File: `hooks/hooks.json` (plugin root)

Per https://code.claude.com/docs/en/hooks.md:

- JSON parses, top-level `hooks` object present
- Event keys use PascalCase (e.g. `SessionStart`, `PreToolUse`, `PostToolUse`) — Cursor's camelCase is invalid here
- Each entry has `matcher` (string) and `hooks` array; each inner hook has `type: "command"` and `command`
- All `command` values reference `${CLAUDE_PLUGIN_ROOT}/bin/*` and resolve to executable files

**Anti-regression invariant** (`.archcore/plugin/hooks-validation-system.spec.md` + `.archcore/plugin/component-registry.doc.md`):

> PostToolUse must **never** have a `Write|Edit` matcher. It may only match MCP tool names (`mcp__archcore__*`).

Grep `hooks/hooks.json` — if `PostToolUse` block contains `Write` or `Edit` in any matcher, FAIL loudly.

Expected shape (`.archcore/plugin/hooks-validation-system.spec.md` Conformance):
- SessionStart: 1 entry → `bin/session-start`
- PreToolUse: 1 entry, matcher `Write|Edit`, two commands (`check-archcore-write`, `check-code-alignment`)
- PostToolUse: entries matching `mcp__archcore__*` tool names only

### Section 10 — Cursor hooks

File: `hooks/cursor.hooks.json` (plugin root)

Per https://cursor.com/docs/hooks, enforced by `test/structure/hooks.bats`:

- JSON parses, `version: 1` at top level, `hooks` object present
- Event keys use camelCase: `sessionStart`, `preToolUse`, `afterMCPExecution` only (no `postToolUse` — Cursor fires `afterMCPExecution` for MCP work)
- `preToolUse` matcher is **exactly `"Write"`** — Cursor exposes no Edit tool; `Edit`/`apply_patch` don't exist there (`hooks.bats` pins this)
- Commands reference `${CURSOR_PLUGIN_ROOT}/bin/*`
- **Same anti-regression**: no `afterFileEdit` with Write|Edit matcher wiring archcore-sync scripts

### Section 11 — Codex CLI hooks

File: `hooks/codex.hooks.json` (plugin root)

Per `.archcore/plugin/codex-local-plugin-testing.guide.md` Step 3 and `.archcore/plugin/codex-host-support.prd.md` F3, enforced by `test/structure/codex-plugin.bats`:

- JSON parses, top-level `hooks` object present
- Event keys use **PascalCase** (same as Claude Code): `SessionStart`, `PreToolUse`, `PostToolUse` — NOT Cursor's camelCase
- Each entry has `matcher` (string) and `hooks` array; each inner hook has `type: "command"` and `command`
- All `command` values use **`${PLUGIN_ROOT}/bin/...`** — Codex's hooks engine injects `PLUGIN_ROOT` as the canonical, host-neutral plugin-root variable (its discovery engine also injects `CLAUDE_PLUGIN_ROOT`, but only as a backward-compat alias for ported Claude plugins — do not borrow it in a Codex-native config)
- The file MUST NOT contain `${CLAUDE_PLUGIN_ROOT}`, `${CODEX_PLUGIN_ROOT}`, or `${CURSOR_PLUGIN_ROOT}` anywhere (grep for all three — any of these tokens is a hard FAIL)
- The file MUST NOT use `./bin/...` relative commands — they resolve against the user's project CWD at hook spawn, not the plugin install dir

**PreToolUse matcher** MUST include `Write`, `Edit`, AND `apply_patch`:
- `apply_patch` is Codex's native edit primitive; omitting it leaves Codex source edits unguarded
- Two commands on the same matcher: `${PLUGIN_ROOT}/bin/check-archcore-write` (timeout 1) AND `${PLUGIN_ROOT}/bin/check-code-alignment` (timeout 1)

**PostToolUse anti-regression invariant** (same as Claude Code, enforced by `test/structure/codex-plugin.bats`):

> No PostToolUse entry may have a `Write` or `Edit` matcher. Only `mcp__archcore__*` matchers allowed.

Expected PostToolUse shape:
- One entry matching the five mutating MCP tools → `${PLUGIN_ROOT}/bin/validate-archcore` (timeout 3)
- One entry matching `mcp__archcore__update_document` → `${PLUGIN_ROOT}/bin/check-cascade` (timeout 3)
- One entry matching `mcp__archcore__create_document|mcp__archcore__update_document` → `${PLUGIN_ROOT}/bin/check-precision` (timeout 3)

**Runtime caveat** (informational, not a FAIL): live plugin-hook execution is gated behind a Codex feature flag (`codex features enable plugin_hooks`; `under development, false` by default in Codex 0.130.0-era builds) and user trust. Upstream hook docs are in flux — see the guide's "Hook guardrails do not fire" entry. This skill validates static packaging only; live execution is verified by the bats integration suite.

### Section 12 — MCP wiring

Two plugin-shipped MCP configs at the plugin root (`plugins/archcore/`). Both must point at `archcore` resolved via PATH — the plugin does not bundle a launcher (see `remove-bundled-launcher-global-cli.idea.md`). Users install the CLI globally per https://docs.archcore.ai/cli/install/.

**Claude Code — `.mcp.json`** (plugin root):
- JSON parses
- `mcpServers.archcore.command` equals `archcore`
- `mcpServers.archcore.args` equals `["mcp"]`
- File MUST NOT contain `${CLAUDE_PLUGIN_ROOT}` or any `bin/archcore` reference (grep — either is a hard FAIL: the launcher was removed)

**Codex CLI — `.codex.mcp.json`** (plugin root):
- File exists at the plugin root (NOT inside `.codex-plugin/`)
- JSON parses
- Uses the Codex-documented direct server map shape, not a `mcpServers`/`mcp_servers` wrapper
- `archcore.command` equals `archcore`
- `archcore.args` equals `["mcp"]`
- File MUST NOT contain `${CLAUDE_PLUGIN_ROOT}`, `${CODEX_PLUGIN_ROOT}`, `./bin/archcore`, `cwd: "."`, or `env_vars` — all are remnants of the deleted launcher architecture and are hard FAILs

**Cursor — `docs/cursor.mcp.example.json`** (repo root; reference template users copy into `~/.cursor/mcp.json` or `.cursor/mcp.json` — we deliberately do NOT ship a plugin-root `mcp.json` to avoid Cursor's plugin-MCP spawn-from-install-dir bug):
- File exists at `docs/cursor.mcp.example.json` (repo root, NOT at the plugin root)
- `mcpServers.archcore.command` equals `archcore`
- `mcpServers.archcore.args[0]` equals `"mcp"`
- `mcpServers.archcore.args` contains `"--project"` followed by `"${workspaceFolder}"` (this passes the workspace path explicitly because Cursor's MCP stdio schema has no `cwd` field)
- `mcpServers.archcore.cwd` MUST be absent (Cursor ignores it; `cwd` here is a hard FAIL — a leftover from the pre-`--project` design)
- File MUST NOT exist at the plugin root as `cursor.mcp.json` (legacy path) — a `cursor.mcp.json` reference anywhere in the runtime tree (`plugins/archcore/`, `README.md`) outside `docs/` and `test/` is a hard FAIL

### Section 13 — Rules (Cursor-only)

Iterate `rules/*.mdc` (plugin root). Per https://cursor.com/docs/context/rules:

- Frontmatter has `description` (string) and `alwaysApply` (bool); optional `globs`
- No unknown fields
- Body is non-empty Markdown

Note: neither Claude Code nor Codex CLI auto-discover `rules/` — this directory is exclusively consumed by Cursor.

### Section 14 — Bin scripts

Per `.archcore/plugin/remove-bundled-launcher-global-cli.idea.md`, the plugin no longer ships a launcher binary, `bin/archcore*` wrappers, or `bin/CLI_VERSION`. Hard FAIL conditions if any of those files exist (they indicate a partial rollback or a regression).

Required `bin/` shape (plugin root):

- `bin/lib/normalize-stdin.sh` exists (sourced by all hook scripts except `check-staleness`)
- `bin/lib/empty-state.sh` exists (sourced by `session-start` for the empty-state nudge)
- All hook scripts referenced by any of the three hooks configs exist under `bin/` and are executable: `session-start`, `check-archcore-write`, `check-code-alignment`, `validate-archcore`, `check-cascade`, `check-precision`, `check-staleness`
- `bin/git-scope` exists and is executable — it is a **skill helper** for `/archcore:context --git-changes` (invoked by the context skill via Bash), NOT referenced by any hooks config; do not expect it in the hooks JSON
- All scripts start with `#!/bin/sh` (POSIX) and pass `shellcheck -s sh -x` when available (`make lint` is the canonical invocation)
- `bin/session-start` falls back to an install-instructions message pointing at https://docs.archcore.ai/cli/install/ when `archcore` is not on PATH
- The Makefile's `BIN_SCRIPTS` set (`$(wildcard plugins/archcore/bin/check-* plugins/archcore/bin/validate-* plugins/archcore/bin/session-start) plugins/archcore/bin/git-scope` via `PLUGIN_REL`) covers every executable bin script and does NOT reference `bin/archcore`

### Section 15 — Archcore registry spot-check

Light staleness check against `.archcore/plugin/component-registry.doc.md`:

- `ls -d plugins/archcore/skills/*/SKILL.md | wc -l` matches the skill total in the registry (currently 7)
- `ls plugins/archcore/agents/*.md | wc -l` matches the MD agent total in the registry (currently 2)
- `ls plugins/archcore/agents/*.toml | wc -l` matches the TOML agent total (Codex parity — currently 2)
- `ls plugins/archcore/commands/*.md | wc -l` equals 7 — one Codex slash-command wrapper per skill; each carries `description:` frontmatter and references the matching `skills/<name>/SKILL.md` (parity per the registry's Codex Slash Command Wrappers table, enforced by `codex-plugin.bats`)
- Scripts in `bin/` match the registry's Bin Scripts table (hook scripts, `bin/lib/` libraries, and the `git-scope` skill helper)
- Per-host config table all exists: at the plugin root — `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.mcp.json`, `.codex.mcp.json`, `hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`; at the repo root — `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`, `.agents/plugins/marketplace.json`, `docs/cursor.mcp.example.json`

This is a spot-check, not a full audit — for full staleness detection use `/archcore:audit --drift`.

---

## Output Format

```
## Plugin Integrity Report

| #  | Section                          | Status   | Details                                  |
|----|----------------------------------|----------|------------------------------------------|
| 1  | Claude manifest                  | ✓ / ✗    | brief                                    |
| 2  | Cursor manifest                  | ✓ / ✗    | brief                                    |
| 3  | Codex manifest                   | ✓ / ✗    | interface{} block + no legacy top-level  |
| 4  | Cross-host consistency           | ✓ / ✗    | name/description/version match (3 hosts) |
| 5  | Marketplace catalogs             | ✓ / ✗    | subdirectory source guard (3 catalogs)   |
| 6  | Skills frontmatter (N)           | ✓ / ✗    | 7-intent surface + count                 |
| 7  | MD agents (N)                    | ✓ / ✗    | bootstrap preamble + forbidden fields    |
| 8  | TOML agents (N)                  | ✓ / ✗    | sandbox_mode + disabled_tools parity     |
| 9  | Hooks (Claude)                   | ✓ / ✗    | PascalCase + anti-regression invariant   |
| 10 | Hooks (Cursor)                   | ✓ / ✗    | camelCase events, exact Write matcher    |
| 11 | Hooks (Codex)                    | ✓ / ✗    | PascalCase + apply_patch + ${PLUGIN_ROOT}|
| 12 | MCP wiring (.mcp + .codex.mcp)   | ✓ / ✗    | PATH commands, no launcher remnants      |
| 13 | Rules                            | ✓ / ✗    | mdc frontmatter                          |
| 14 | Bin scripts                      | ✓ / ✗    | hook scripts + libs + no launcher        |
| 15 | Registry spot-check              | ✓ / ✗    | counts + wrappers match                  |

Result: X / 15 sections passed.
```

For every FAIL, print one line below the table in the form:

```
- Section N — <what failed>. Fix: <specific action>. Spec: <URL or .archcore file>
```

Cite the specific spec URL (Claude Code / Cursor / OpenAI Codex / agentskills.io) or `.archcore/*.md` line that was violated. Do not paraphrase — quote the rule.

If everything passes, print one line: `All 15 sections passed. Plugin format is conformant.`
