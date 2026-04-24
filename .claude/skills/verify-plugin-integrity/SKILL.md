---
name: verify-plugin-integrity
description: Validate plugin format conformance for Claude Code and Cursor — statically audits .claude-plugin/plugin.json, .cursor-plugin/plugin.json, marketplace manifests, SKILL.md frontmatter, agent files, hooks JSON, rules, and bin/ launcher against the official Claude Code plugin spec, Cursor plugin spec, and Agent Skills specification. No test execution. Use after structural changes to manifests, skills, agents, hooks, or rules; before opening a PR touching plugin structure; or when multi-host (Claude + Cursor) consistency is in doubt. For the bats test suite, use /archcore:verify instead.
disable-model-invocation: true
---

# /verify-plugin-integrity

Static format-conformance audit for the archcore multi-host plugin. Validates that every plugin artifact matches the official Claude Code, Cursor, and Agent Skills specifications, plus the normative Archcore specs in this repo.

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

### Archcore internal conformance (normative for this repo)
- `.archcore/plugin-architecture.spec.md` — layer counts, tier prefixes, invocation flags
- `.archcore/hooks-validation-system.spec.md` — hook entries, events, anti-regression
- `.archcore/multi-host-compatibility-layer.spec.md` — cross-host manifest + hooks rules
- `.archcore/agent-system.spec.md` — agent frontmatter + bootstrap preamble
- `.archcore/skill-file-structure.rule.md` — skill file binding format
- `.archcore/component-registry.doc.md` — authoritative component counts

If a local spec conflicts with an external official doc, the external doc is ground truth for format; the Archcore spec is ground truth for this plugin's *choices* within that format (counts, tier prefixes, etc.).

---

## When to use

- After editing any file under `.claude-plugin/`, `.cursor-plugin/`, `skills/`, `agents/`, `hooks/`, `rules/`, or `bin/`
- Before opening a PR that touches plugin structure
- When cross-host (Claude + Cursor) manifest consistency is in doubt
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
- **Forbidden here**: `mcpServers` field inside the manifest itself (per `.archcore/multi-host-compatibility-layer.spec.md` §7 — MCP lives in `.mcp.json` at repo root, not in the plugin manifest)
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

### Section 3 — Cross-host consistency

Compare `.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json`. The following fields MUST be byte-identical:

- `name`
- `description`
- `version`

Flag any drift — this is the most common regression when bumping versions on one host but forgetting the other.

### Section 4 — Marketplace manifests

Files: `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json`

Per Claude Code and Cursor plugins docs:

- JSON parses
- `name` present, kebab-case
- `owner.name` present
- `plugins` is a non-empty array; each entry has `name` and `source`
- Every `source` path resolves to a real directory (relative to the marketplace file)

### Section 5 — Skills frontmatter audit

Iterate `skills/*/SKILL.md`. For each, per https://agentskills.io/specification and https://code.claude.com/docs/en/skills.md:

- YAML frontmatter parses
- `name` is required, ≤ 64 chars, kebab-case (lowercase + hyphens), and **equals the parent directory name**
- `description` is required, ≤ 1024 chars, non-empty
- No unknown top-level fields (allowed: `name`, `description`, `license`, `metadata`, `compatibility`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`, `arguments`, `model`, `effort`, `context`, `agent`, `paths`, `shell`, `hooks`)

Then enforce this repo's tier rules (`.archcore/plugin-architecture.spec.md` Conformance §1–§10):

- Layer 1 intent skills (10): no `disable-model-invocation`, no `user-invocable: false`
- Layer 2 track skills (6): `description` starts with `"Advanced — "`
- Layer 3 mainstream type skills (10): `disable-model-invocation: true`; non-high-frequency types start with `"Expert — "`
- Layer 3 niche type skills (7): `user-invocable: false`
- Utility skills: `disable-model-invocation: true`

Total count sanity check: `ls -d skills/*/SKILL.md | wc -l` should match the registry total in `.archcore/component-registry.doc.md` (currently 34).

### Section 6 — Agents audit

Iterate `agents/*.md`. Per https://code.claude.com/docs/en/plugins-reference.md (agents section):

- YAML frontmatter parses
- Required: `name`, `description`
- Optional (allowed): `model`, `effort`, `maxTurns`, `tools`, `disallowedTools`, `skills`, `memory`, `background`, `isolation` (only valid value: `"worktree"`), `color`
- **Forbidden in plugin agents**: `hooks`, `mcpServers`, `permissionMode` (Claude Code security restriction — plugin agents cannot configure these)

Archcore-specific (`.archcore/agent-system.spec.md`):
- Body contains the literal heading `# First Step — Bootstrap Knowledge Tree`
- Body grep-matches the anchor phrase `recent accepted decisions`

### Section 7 — Claude Code hooks

File: `hooks/hooks.json`

Per https://code.claude.com/docs/en/hooks.md:

- JSON parses, top-level `hooks` object present
- Event keys use PascalCase (e.g. `SessionStart`, `PreToolUse`, `PostToolUse`) — Cursor's camelCase is invalid here
- Each entry has `matcher` (string) and `hooks` array; each inner hook has `type: "command"` and `command`
- All `command` values reference `${CLAUDE_PLUGIN_ROOT}/bin/*` and resolve to executable files

**Anti-regression invariant** (`.archcore/hooks-validation-system.spec.md` + `.archcore/component-registry.doc.md`):

> PostToolUse must **never** have a `Write|Edit` matcher. It may only match MCP tool names (`mcp__archcore__*`).

Grep `hooks/hooks.json` — if `PostToolUse` block contains `Write` or `Edit` in any matcher, FAIL loudly.

Expected shape (`.archcore/hooks-validation-system.spec.md` Conformance §1):
- SessionStart: 1 entry → `bin/session-start`
- PreToolUse: 1 entry, matcher `Write|Edit`, two commands (`check-archcore-write`, `check-code-alignment`)
- PostToolUse: 2 entries matching `mcp__archcore__*` tool names

### Section 8 — Cursor hooks

File: `hooks/cursor.hooks.json`

Per https://cursor.com/docs/hooks and `.archcore/multi-host-compatibility-layer.spec.md`:

- JSON parses, `version: 1` at top level, `hooks` object present
- Event keys use camelCase: `sessionStart`, `preToolUse`, `afterMCPExecution` only (no `postToolUse` — Cursor fires `afterMCPExecution` for MCP work)
- Commands reference `${CURSOR_PLUGIN_ROOT}/bin/*`
- **Same anti-regression**: no `afterFileEdit` with Write|Edit matcher wiring archcore-sync scripts

### Section 9 — Rules (Cursor-only)

Iterate `rules/*.mdc`. Per https://cursor.com/docs/context/rules:

- Frontmatter has `description` (string) and `alwaysApply` (bool); optional `globs`
- No unknown fields
- Body is non-empty Markdown

Note: Claude Code does **not** auto-discover `rules/` — this directory is exclusively consumed by Cursor.

### Section 10 — Bin launcher

Per `.archcore/bundled-cli-launcher.adr.md`:

- `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1` all exist
- `bin/CLI_VERSION` exists and contains exactly one semver line
- `bin/archcore` is executable (`[ -x ]`)
- All hook scripts referenced by either hooks.json (`session-start`, `check-archcore-write`, `check-code-alignment`, `validate-archcore`, `check-cascade`) exist under `bin/` and are executable

### Section 11 — Archcore registry spot-check

Light staleness check against `.archcore/component-registry.doc.md`:

- `ls -d skills/*/SKILL.md | wc -l` matches the skill total in the registry
- `ls agents/*.md | wc -l` matches the agent total in the registry
- Hook scripts in `bin/` match the registry's Hooks Scripts table

This is a spot-check, not a full audit — for full staleness detection use `/archcore:actualize`.

---

## Output Format

```
## Plugin Integrity Report

| # | Section                         | Status   | Details                                  |
|---|---------------------------------|----------|------------------------------------------|
| 1 | Claude manifest                 | ✓ / ✗    | brief                                    |
| 2 | Cursor manifest                 | ✓ / ✗    | brief                                    |
| 3 | Cross-host consistency          | ✓ / ✗    | name/description/version match           |
| 4 | Marketplace manifests           | ✓ / ✗    | brief                                    |
| 5 | Skills frontmatter (N)          | ✓ / ✗    | tier compliance + count                  |
| 6 | Agents (N)                      | ✓ / ✗    | bootstrap preamble + forbidden fields    |
| 7 | Hooks (Claude)                  | ✓ / ✗    | anti-regression invariant                |
| 8 | Hooks (Cursor)                  | ✓ / ✗    | camelCase events, no postToolUse         |
| 9 | Rules                           | ✓ / ✗    | mdc frontmatter                          |
|10 | Bin launcher                    | ✓ / ✗    | launchers + CLI_VERSION + executable     |
|11 | Registry spot-check             | ✓ / ✗    | counts match                             |

Result: X / 11 sections passed.
```

For every FAIL, print one line below the table in the form:

```
- Section N — <what failed>. Fix: <specific action>. Spec: <URL or .archcore file>
```

Cite the specific spec URL (Claude Code / Cursor / agentskills.io) or `.archcore/*.md` line that was violated. Do not paraphrase — quote the rule.

If everything passes, print one line: `All 11 sections passed. Plugin format is conformant.`
