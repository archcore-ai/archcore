---
name: verify-plugin-integrity
description: Validate plugin format conformance for Claude Code, Cursor, and Codex CLI — statically audits .claude-plugin/plugin.json, .cursor-plugin/plugin.json, .codex-plugin/plugin.json, marketplace manifests (incl. .agents/plugins/marketplace.json), SKILL.md frontmatter, MD + TOML agent files, hooks JSON for all three hosts, plugin-shipped MCP wiring (.mcp.json + .codex.mcp.json), Cursor rules, and bin/ launcher against the official Claude Code plugin spec, Cursor plugin spec, OpenAI Codex CLI plugin docs, and Agent Skills specification. No test execution. Use after structural changes to manifests, skills, agents, hooks, or rules; before opening a PR touching plugin structure; or when multi-host (Claude + Cursor + Codex) consistency is in doubt. For the bats test suite, use /archcore:verify instead.
disable-model-invocation: true
---

# /verify-plugin-integrity

Static format-conformance audit for the archcore multi-host plugin. Validates that every plugin artifact matches the official Claude Code, Cursor, OpenAI Codex CLI, and Agent Skills specifications, plus the normative Archcore specs in this repo.

**Does not** execute tests, lint, scripts, or hooks — for that, use `/archcore:verify` (which runs the bats test suite).

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

### OpenAI Codex CLI (openai/codex)
- Codex CLI repo — https://github.com/openai/codex
- Codex plugins overview — https://github.com/openai/codex/blob/main/docs/plugins.md (when not reachable, treat the internal Archcore spec as ground truth)
- Codex hooks docs — https://github.com/openai/codex/blob/main/docs/hooks.md (gated by `[features].codex_hooks = true` in `~/.codex/config.toml`)

### Archcore internal conformance (normative for this repo)
- `.archcore/plugin/plugin-architecture.spec.md` — layer counts, tier prefixes, invocation flags
- `.archcore/plugin/hooks-validation-system.spec.md` — hook entries, events, anti-regression
- `.archcore/plugin/multi-host-compatibility-layer.spec.md` — cross-host manifest + hooks + MCP rules (sections 4–7 are the Codex ground truth for this repo)
- `.archcore/plugin/codex-host-support.prd.md` — Codex packaging functional requirements (F1–F10)
- `.archcore/plugin/agent-system.spec.md` — agent frontmatter + bootstrap preamble
- `.archcore/plugin/skill-file-structure.rule.md` — skill file binding format
- `.archcore/plugin/component-registry.doc.md` — authoritative component counts and per-host config table

If a local spec conflicts with an external official doc, the external doc is ground truth for format; the Archcore spec is ground truth for this plugin's *choices* within that format (counts, tier prefixes, Codex-specific manifest fields, etc.). For Codex specifically, the Compatibility Layer spec is authoritative whenever the upstream Codex docs are silent or in flux.

---

## When to use

- After editing any file under `.claude-plugin/`, `.cursor-plugin/`, `.codex-plugin/`, `.agents/plugins/`, `skills/`, `agents/`, `hooks/`, `rules/`, or `bin/`
- After changing `.mcp.json` or `.codex.mcp.json` at the plugin root
- Before opening a PR that touches plugin structure
- When cross-host (Claude + Cursor + Codex) manifest consistency is in doubt
- **Not** for test runs — use `/archcore:verify`
- **Not** for Archcore document freshness vs code — use `/archcore:actualize`

---

## Execution

Work entirely with `Read`, `Grep`, and `Bash` (e.g. `jq`, `ls`, `wc -l`). Do **not** call MCP tools — this skill creates no Archcore documents.

Walk through every section below. Record each check as PASS / FAIL / WARN with a one-line reason. Collate into the final report (see Output Format).

### Section 1 — Claude Code plugin manifest

File: `.claude-plugin/plugin.json`

Per https://code.claude.com/docs/en/plugins-reference.md:

- JSON parses
- `name` present, kebab-case, alphanumeric + hyphens
- `description`, `version` (semver), `author` (object with `name`), `license`, `repository` present (all optional per spec, but the repo convention requires them)
- Any directory override fields (`skills`, `agents`, `commands`, `hooks`, `mcpServers`) start with `./` and resolve to existing paths
- **Forbidden here**: `mcpServers` field inside the manifest itself (per `.archcore/plugin/multi-host-compatibility-layer.spec.md` §6 — MCP lives in `.mcp.json` at repo root, not in the plugin manifest)
- **Forbidden here**: `rules` field (Cursor-only; not auto-discovered by Claude Code)

### Section 2 — Cursor plugin manifest

File: `.cursor-plugin/plugin.json`

Per https://cursor.com/docs/plugins/building:

- JSON parses
- `name` kebab-case
- `description`, `version`, `author`, `license`, `repository`, `keywords` present
- `hooks` field points to `hooks/cursor.hooks.json` (Cursor-specific file; overrides default `hooks/hooks.json`)
- `skills`, `agents`, `rules` fields (if set) resolve to existing directories
- **Forbidden**: `mcpServers` field inside the manifest (lives in `.cursor/mcp.json` or `~/.cursor/mcp.json` per Cursor docs; not inside plugin manifest)

### Section 3 — Codex CLI plugin manifest

File: `.codex-plugin/plugin.json`

Per `.archcore/plugin/multi-host-compatibility-layer.spec.md` §5 (Codex CLI subsection) and `.archcore/plugin/codex-host-support.prd.md` F1, with cross-reference to https://github.com/openai/codex:

- JSON parses
- `name` present, kebab-case
- `description`, `version` (semver) present
- Component pointers `skills`, `hooks`, `mcpServers` are present and prefixed with `./` (Codex requires plugin-relative paths)
  - `hooks` MUST equal `./hooks/codex.hooks.json`
  - `mcpServers` MUST be a string pointer equal to `./.codex.mcp.json` (Codex resolves this file at plugin load — NOT an inline object)
  - `skills` MUST resolve to `./skills/`
- `interface{}` block present with at minimum `displayName`, `shortDescription`, `longDescription`, `developerName`, `category`, `capabilities` — and `capabilities` includes `Interactive`, `Read`, `Write`
- **Forbidden**: legacy top-level UI fields. Any of `displayName`, `category`, or `tags` at the top level (outside `interface{}`) is a hard FAIL per `test/structure/codex-plugin.bats`
- **Forbidden**: an inline `mcpServers` object inside the manifest. Codex requires the pointer-to-file shape; an inline object is non-conformant
- Sibling files: `.codex-plugin/` MUST contain ONLY `plugin.json`. Any additional file (notably a stray `marketplace.json`) is a hard FAIL — Codex marketplace metadata lives elsewhere (see Section 5)

### Section 4 — Cross-host consistency

Compare `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, and `.codex-plugin/plugin.json`. The following fields MUST be byte-identical across all three:

- `name`
- `description`
- `version`

Flag any drift — this is the most common regression when bumping versions on one host but forgetting another. Per `multi-host-compatibility-layer.spec.md` §5 normative bullet: *"Plugin manifests MUST use identical name, description, and version across all hosts."*

### Section 5 — Marketplace manifests

Files (host-specific locations):

- Claude Code: `.claude-plugin/marketplace.json`
- Cursor: `.cursor-plugin/marketplace.json`
- Codex CLI: `.agents/plugins/marketplace.json` (NOT under `.codex-plugin/`)

Per Claude Code, Cursor, and OpenAI Codex plugin docs:

**Claude Code & Cursor** — same shape:
- JSON parses
- `name` present, kebab-case
- `owner.name` present
- `plugins` is a non-empty array; each entry has `name` and `source`
- Every `source` path resolves to a real directory (relative to the marketplace file)

**Codex CLI** — distinct schema (per `.archcore/plugin/codex-host-support.prd.md` F2 and `test/structure/codex-plugin.bats`):
- File path is `.agents/plugins/marketplace.json`
- `name` equals `"archcore-plugins"` (note: marketplace name, NOT the plugin name)
- `interface.displayName` present
- `plugins` array; each entry has:
  - `name` (string, the plugin identifier — `"archcore"`)
  - `source` is an **object** with `source: "local"` and `path: "./"` (Codex marketplace `source` is structured, not a string path like Claude/Cursor)
  - `policy.installation` equals `"INSTALLED_BY_DEFAULT"`
  - `policy.authentication` equals `"ON_INSTALL"`
  - `category` present (e.g., `"Coding"`)
- `.codex-plugin/marketplace.json` MUST NOT exist (legacy path; FAIL if present)

### Section 6 — Skills frontmatter audit

Iterate `skills/*/SKILL.md`. For each, per https://agentskills.io/specification and https://code.claude.com/docs/en/skills.md:

- YAML frontmatter parses
- `name` is required, ≤ 64 chars, kebab-case (lowercase + hyphens), and **equals the parent directory name**
- `description` is required, ≤ 1024 chars, non-empty
- No unknown top-level fields (allowed: `name`, `description`, `license`, `metadata`, `compatibility`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`, `arguments`, `model`, `effort`, `context`, `agent`, `paths`, `shell`, `hooks`)

Then enforce this repo's tier rules (`.archcore/plugin/plugin-architecture.spec.md` Conformance):

- Layer 1 intent skills: no `disable-model-invocation`, no `user-invocable: false`
- Layer 2 track skills: `description` starts with `"Advanced — "`
- Utility skills: `disable-model-invocation: true`

Per `.archcore/plugin/codex-host-support.prd.md` F7, all SKILL.md files MUST work unchanged in Codex; non-standard frontmatter fields (e.g., `argument-hint`) are tolerated by Codex's loader. No Codex-specific skill validation is required.

Total count sanity check: `ls -d skills/*/SKILL.md | wc -l` should match the registry total in `.archcore/plugin/component-registry.doc.md`.

### Section 7 — Markdown agents (Claude Code + Cursor)

Iterate `agents/*.md`. Per https://code.claude.com/docs/en/plugins-reference.md (agents section):

- YAML frontmatter parses
- Required: `name`, `description`
- Optional (allowed): `model`, `effort`, `maxTurns`, `tools`, `disallowedTools`, `skills`, `memory`, `background`, `isolation` (only valid value: `"worktree"`), `color`
- **Forbidden in plugin agents**: `hooks`, `mcpServers`, `permissionMode` (Claude Code security restriction — plugin agents cannot configure these)

Archcore-specific (`.archcore/plugin/agent-system.spec.md`):
- Body contains the literal heading `# First Step — Bootstrap Knowledge Tree`
- Body grep-matches the anchor phrase `recent accepted decisions`

### Section 8 — TOML agents (Codex CLI)

Iterate `agents/*.toml`. Per `.archcore/plugin/multi-host-compatibility-layer.spec.md` §7 and `.archcore/plugin/codex-host-support.prd.md` F8:

- TOML parses
- Required keys: `name`, `description`, `developer_instructions`, `sandbox_mode`
- `name` matches the file basename (e.g., `archcore-auditor.toml` → `name = "archcore-auditor"`)
- For each MD agent in `agents/`, a TOML counterpart with the same basename MUST exist (currently: `archcore-assistant`, `archcore-auditor`)

Per-agent rules:

- `archcore-auditor.toml` MUST have `sandbox_mode = "read-only"` AND `disabled_tools` containing all five mutating MCP tools: `mcp__archcore__create_document`, `mcp__archcore__update_document`, `mcp__archcore__remove_document`, `mcp__archcore__add_relation`, `mcp__archcore__remove_relation`
- `archcore-assistant.toml` MUST have `sandbox_mode = "workspace-write"` and SHOULD NOT declare `disabled_tools` (assistant has full MCP access by design)

Bootstrap preamble (parity with MD originals — `.archcore/plugin/agent-system.spec.md`):
- `developer_instructions` contains the literal heading `# First Step — Bootstrap Knowledge Tree`
- `developer_instructions` grep-matches the anchor phrase `recent accepted decisions`

### Section 9 — Claude Code hooks

File: `hooks/hooks.json`

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

File: `hooks/cursor.hooks.json`

Per https://cursor.com/docs/hooks and `.archcore/plugin/multi-host-compatibility-layer.spec.md` §4:

- JSON parses, `version: 1` at top level, `hooks` object present
- Event keys use camelCase: `sessionStart`, `preToolUse`, `afterMCPExecution` only (no `postToolUse` — Cursor fires `afterMCPExecution` for MCP work)
- Commands reference `${CURSOR_PLUGIN_ROOT}/bin/*`
- **Same anti-regression**: no `afterFileEdit` with Write|Edit matcher wiring archcore-sync scripts

### Section 11 — Codex CLI hooks

File: `hooks/codex.hooks.json`

Per `.archcore/plugin/multi-host-compatibility-layer.spec.md` §4 (Codex subsection) and `.archcore/plugin/codex-host-support.prd.md` F3, with cross-reference to https://github.com/openai/codex:

- JSON parses, top-level `hooks` object present
- Event keys use **PascalCase** (same as Claude Code): `SessionStart`, `PreToolUse`, `PostToolUse` — NOT Cursor's camelCase
- Each entry has `matcher` (string) and `hooks` array; each inner hook has `type: "command"` and `command`
- All `command` values are **plugin-relative** — start with `./` (e.g. `./bin/session-start`). Codex does NOT expose a documented `${CODEX_PLUGIN_ROOT}` substitution
- The file MUST NOT contain `${CLAUDE_PLUGIN_ROOT}` or `${CODEX_PLUGIN_ROOT}` anywhere (grep for both — either token is a hard FAIL)

**PreToolUse matcher** MUST include `Write`, `Edit`, AND `apply_patch`:
- `apply_patch` is Codex's native edit primitive; omitting it leaves Codex source edits unguarded
- Two commands on the same matcher: `./bin/check-archcore-write` (timeout 1) AND `./bin/check-code-alignment` (timeout 1)

**PostToolUse anti-regression invariant** (same as Claude Code, enforced by `test/structure/codex-plugin.bats`):

> No PostToolUse entry may have a `Write` or `Edit` matcher. Only `mcp__archcore__*` matchers allowed.

Expected PostToolUse shape:
- One entry matching the five mutating MCP tools → `./bin/validate-archcore` (timeout 3)
- One entry matching `mcp__archcore__update_document` → `./bin/check-cascade` (timeout 3)
- One entry matching `mcp__archcore__create_document|mcp__archcore__update_document` → `./bin/check-precision` (timeout 3)

**Runtime caveat** (informational, not a FAIL): Codex hook execution requires `[features].codex_hooks = true` in `~/.codex/config.toml`. This skill validates static packaging only — live execution is verified by the bats integration suite.

### Section 12 — MCP wiring

Two plugin-shipped MCP configs at the plugin root.

**Claude Code — `.mcp.json`** (per `.archcore/plugin/multi-host-compatibility-layer.spec.md` §6):
- JSON parses
- `mcpServers.archcore.command` equals `${CLAUDE_PLUGIN_ROOT}/bin/archcore`
- `mcpServers.archcore.args` equals `["mcp"]`

**Codex CLI — `.codex.mcp.json`** (per `multi-host-compatibility-layer.spec.md` §6 Codex subsection and `test/structure/codex-plugin.bats`):
- File exists at the plugin root (NOT inside `.codex-plugin/`)
- JSON parses
- `mcpServers.archcore.command` equals `./bin/archcore` (plugin-relative — NOT `${CLAUDE_PLUGIN_ROOT}` or `${CODEX_PLUGIN_ROOT}`)
- `mcpServers.archcore.args` equals `["mcp"]`
- File MUST NOT contain `${CLAUDE_PLUGIN_ROOT}` or `${CODEX_PLUGIN_ROOT}` (grep — either token is a hard FAIL)

**Cursor**: no plugin-shipped MCP file. Cursor users register MCP externally via Cursor's settings UI or a project `mcp.json`. PASS if no Cursor-specific MCP file is present at the plugin root; the launcher itself remains the same binary.

### Section 13 — Rules (Cursor-only)

Iterate `rules/*.mdc`. Per https://cursor.com/docs/context/rules:

- Frontmatter has `description` (string) and `alwaysApply` (bool); optional `globs`
- No unknown fields
- Body is non-empty Markdown

Note: neither Claude Code nor Codex CLI auto-discover `rules/` — this directory is exclusively consumed by Cursor.

### Section 14 — Bin launcher

Per `.archcore/plugin/bundled-cli-launcher.adr.md`:

- `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1` all exist
- `bin/CLI_VERSION` exists and contains exactly one semver line
- `bin/archcore` is executable (`[ -x ]`)
- `bin/lib/normalize-stdin.sh` exists (sourced by all hook scripts)
- All hook scripts referenced by any of the three hooks configs (`session-start`, `check-archcore-write`, `check-code-alignment`, `validate-archcore`, `check-cascade`, `check-precision`) exist under `bin/` and are executable

### Section 15 — Archcore registry spot-check

Light staleness check against `.archcore/plugin/component-registry.doc.md`:

- `ls -d skills/*/SKILL.md | wc -l` matches the skill total in the registry
- `ls agents/*.md | wc -l` matches the MD agent total in the registry
- `ls agents/*.toml | wc -l` matches the TOML agent total (Codex parity — currently 2)
- Hook scripts in `bin/` match the registry's Hook Scripts table
- Per-host config table (`.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.mcp.json`, `.codex.mcp.json`, `hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, `.agents/plugins/marketplace.json`) all exist

This is a spot-check, not a full audit — for full staleness detection use `/archcore:actualize`.

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
| 5  | Marketplace manifests            | ✓ / ✗    | Claude + Cursor + Codex (.agents/...)    |
| 6  | Skills frontmatter (N)           | ✓ / ✗    | tier compliance + count                  |
| 7  | MD agents (N)                    | ✓ / ✗    | bootstrap preamble + forbidden fields    |
| 8  | TOML agents (N)                  | ✓ / ✗    | sandbox_mode + disabled_tools parity     |
| 9  | Hooks (Claude)                   | ✓ / ✗    | PascalCase + anti-regression invariant   |
| 10 | Hooks (Cursor)                   | ✓ / ✗    | camelCase events, no postToolUse         |
| 11 | Hooks (Codex)                    | ✓ / ✗    | PascalCase + apply_patch + relative cmds |
| 12 | MCP wiring (.mcp + .codex.mcp)   | ✓ / ✗    | launcher commands, no host-root tokens   |
| 13 | Rules                            | ✓ / ✗    | mdc frontmatter                          |
| 14 | Bin launcher                     | ✓ / ✗    | launchers + CLI_VERSION + executable     |
| 15 | Registry spot-check              | ✓ / ✗    | counts match                             |

Result: X / 15 sections passed.
```

For every FAIL, print one line below the table in the form:

```
- Section N — <what failed>. Fix: <specific action>. Spec: <URL or .archcore file>
```

Cite the specific spec URL (Claude Code / Cursor / OpenAI Codex / agentskills.io) or `.archcore/*.md` line that was violated. Do not paraphrase — quote the rule.

If everything passes, print one line: `All 15 sections passed. Plugin format is conformant.`
