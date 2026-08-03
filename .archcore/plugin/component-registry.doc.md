---
title: "Plugin Component Registry"
status: accepted
tags:
  - "plugin"
  - "reference"
---

## Overview

A reference listing of every component of the Archcore plugin across its four hosts: Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI.

Claude Code, Cursor, and Copilot CLI surface user-invoked workflows directly from skills at `skills/<name>/SKILL.md`. Codex CLI discovers slash commands from `commands/*.md` wrappers — thin host-adapter shims that delegate to the matching skill — and Copilot loads the same wrappers as a secondary surface, because a skill outranks a command of the same name there. The skill is the single behavioral source of truth on all four hosts; no workflow logic lives in `commands/`.

Per `skill-surface-collapse.adr`, the visible `/` palette is exactly **7 auto-invocable intent skills**. There are no per-document-type skills, no track skills, and no utility skills. Track flows live as references under `skills/plan/references/`, and drift detection lives at `skills/audit/lib/drift-detection.md`.

**Layout.** Every component path here is plugin-root-relative, and the plugin root is the `plugins/archcore/` subdirectory — `skills/audit/` means `plugins/archcore/skills/audit/`, and `bin/session-start` means `plugins/archcore/bin/session-start`. The three marketplace catalogs are the exception: they live at the repo root and point their `source` and `path` at `./plugins/archcore`. That root-catalog and subdirectory-manifest split is required for Codex marketplace discovery, because a catalog `source.path` of `./` is never scanned — see `subdirectory-plugin-layout.adr` and issue #2. There are three catalogs and four hosts because Copilot CLI has no marketplace and installs by subdirectory spec. The reference template `docs/cursor.mcp.example.json` also stays at the repo root.

## Content

### Skills (7, auto-invocable by model and user)

No skill carries a `disable-model-invocation` flag. Routing is governed by per-skill `Activate when X. Do NOT activate for Y.` descriptions.

| Skill | Directory | User Intent |
| --- | --- | --- |
| init | `skills/init/` | First-time onboarding — seed `.archcore/` with a stack rule, a run guide, the extras for the detected scale, plus host wiring (project MCP config, SessionStart hook, usage hint — the same files `archcore init` writes; see `host-wiring-parity.adr`) |
| capture | `skills/capture/` | Document a module or component — routes to adr / spec / doc / guide |
| decide | `skills/decide/` | Record a decision (adr) or draft a proposal (rfc); optional standard cascade (cpat → rule → guide) |
| plan | `skills/plan/` | Plan a feature or initiative — routes to a single plan or one of four flows (product / sources / iso / feature) |
| audit | `skills/audit/` | Documentation health — dashboard (default), `--deep` audit, or `--drift` detection |
| context | `skills/context/` | Surface rules and decisions for a code area or pickup |
| help | `skills/help/` | Navigate the system — command catalogue, onboarding |

### Per-flow references

Flows that once lived as standalone track skills now live as references that `plan` loads on demand:

| Reference | Path | Flow |
| --- | --- | --- |
| product-flow | `skills/plan/references/product-flow.md` | idea → prd → plan |
| sources-flow | `skills/plan/references/sources-flow.md` | mrd → brd → urd |
| iso-flow | `skills/plan/references/iso-flow.md` | brs → strs → syrs → srs |
| feature-flow | `skills/plan/references/feature-flow.md` | prd → spec → plan → task-type |

`decide` carries its continuation logic at `skills/decide/references/continuations.md`, covering the ADR → CPAT → rule → guide cascade. `audit` carries its drift-mode protocol at `skills/audit/lib/drift-detection.md`, covering code-drift, cascade, and temporal staleness checks.

### Shared runtime assets (`skills/_shared/`)

Plain-markdown assets loaded at runtime before a skill composes a document. They ship with the plugin, and skill instructions reference plugin-internal paths only, never the consumer's `.archcore/`.

| Asset | Path | Loaded by | Purpose |
| --- | --- | --- | --- |
| `precision-rules.md` | `skills/_shared/precision-rules.md` | `decide`, `capture`, `init`, `plan` | Forbidden vagueness lexicon, imperative phrasing, no-cross-document-section rule, `[assumption]` marker conventions |
| `adr-contract.md` | `skills/_shared/adr-contract.md` | `decide`, `capture` (ADR) | Mandatory sections plus good and bad examples for ADR content, per MADR 4.0 |
| `spec-contract.md` | `skills/_shared/spec-contract.md` | `capture` (spec), `init` (hotspot specs) | Mandatory sections and the "when NOT to write a spec" gate |
| `rule-contract.md` | `skills/_shared/rule-contract.md` | `decide` (rule), `init` (cross-cutting rules) | Mandatory rule body: RFC 2119 statement, applies-to scope, rationale, Good/Bad examples, enforcement |

`precision-over-coverage.adr` records the design rationale for these assets.

### Slash-command wrappers (`commands/`)

Codex CLI requires a thin wrapper per user-facing skill so that `/archcore:<name>` appears in its `/` menu. Each wrapper is a host-adapter shim: `description:` frontmatter plus a one-line delegate instruction pointing at `skills/<name>/SKILL.md`. No workflow logic lives here.

| Wrapper set | Count | Purpose |
| --- | --- | --- |
| `commands/<name>.md` | 7 | One per skill (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`) — surfaces `/archcore:<name>` on Codex CLI, and on Copilot CLI as a fallback behind the skill of the same name |

`@test/structure/codex-plugin.bats` enforces that every entry exists, carries `description:`, and references the matching `skills/<name>/SKILL.md`. `@test/structure/copilot-plugin.bats` enforces that the wrapper set matches the skill set and that the manifest points at `commands/` — Copilot gives that field no default path, so without the pointer the entire `/archcore:*` surface is missing there. Claude Code and Cursor need no wrappers, because they surface skills directly.

### Document-type coverage and the visible `/` surface

Every Archcore document type is reachable through an intent skill or directly through `mcp__archcore__create_document(type=<any>)`; `skills-system.spec` holds the full mapping under "Document-type coverage". All 7 skills on disk are visible, with no hidden or flagged-out skill. Codex CLI exposes the same 7 entries through the `commands/*.md` wrappers, and Copilot exposes them from the skills with the wrappers as backup.

### Agents (2)

| Agent | File | Role | Model | Tools |
| --- | --- | --- | --- | --- |
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent | sonnet | All MCP + Read/Grep/Glob |
| `archcore-auditor` | `agents/archcore-auditor.md` | Read-only documentation auditor | sonnet | Read MCP + Read/Grep/Glob |

`archcore-assistant` handles complex multi-document tasks — creation, requirements engineering, relation management. Foreground, blue, 20 turns maximum. `archcore-auditor` handles documentation health checks — coverage gaps, orphaned documents, stale statuses, and code-document correlation, which cross-references document path mentions against git history to flag drift. Background, yellow, 15 turns maximum.

Both ship in three formats, one per loader:

| Format | Location | Read by |
| --- | --- | --- |
| `<name>.md` | `agents/` | Claude Code, Cursor |
| `<name>.toml` | `agents/` | Codex CLI (adds `sandbox_mode`, `disabled_tools[]`) |
| `<name>.agent.md` | `copilot-agents/` | GitHub Copilot CLI |

The Copilot copies live in a separate directory rather than beside the originals: Copilot loads plugin agents only from the `*.agent.md` extension, and `.agent.md` still matches the `*.md` glob Claude Code and Cursor use, so a sibling copy would hand both hosts two files declaring the same frontmatter `name:`. The TOML variants never had this problem, because their extension is foreign to every md-globbing host. `@test/structure/agents.bats` holds the copies byte-identical with `cmp` and fails when an agent in `agents/` has no counterpart.

Agent tool lists — the `.md` allow-lists and the auditor's TOML deny-list — carry all three MCP tool namings: `mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, and Copilot's flat `archcore-<tool>`. The same test guards the twin pairing, which matters most on the deny-list, because a deny-list fails open under naming drift.

### Hooks (6 entries across 3 events)

| # | Event | Matcher | Handler | Timeout |
| --- | --- | --- | --- | --- |
| 1 | SessionStart | (all) | `bin/session-start` | — |
| 2 | PreToolUse | `Write\|Edit` | `bin/check-archcore-write` | 1s |
| 3 | PreToolUse | `Write\|Edit` | `bin/check-code-alignment` | 1s |
| 4 | PostToolUse | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` (+ `mcp__plugin_archcore_archcore__*` twins) | `bin/validate-archcore` | 3s |
| 5 | PostToolUse | `mcp__archcore__update_document` (+ `mcp__plugin_archcore_archcore__*` twin) | `bin/check-cascade` | 3s |
| 6 | PostToolUse | `mcp__archcore__create_document\|mcp__archcore__update_document` (+ `mcp__plugin_archcore_archcore__*` twins) | `bin/check-precision` | 3s |

Hook configs: `hooks/hooks.json` for Claude Code with PascalCase events; `hooks/cursor.hooks.json` for Cursor with camelCase events plus `afterMCPExecution`, whose `preToolUse` matcher is `Write` only because Cursor exposes no Edit tool; `hooks/codex.hooks.json` for Codex CLI with PascalCase events, an `apply_patch` matcher addition for Codex's native edit primitive, and commands using Codex's canonical `${PLUGIN_ROOT}` substitution; and `hooks/copilot.hooks.json` for Copilot CLI with native camelCase events under a `bash` key rather than `command`, `timeoutSec` rather than `timeout`, a deterministic `ARCHCORE_HOST=copilot`, and `cwd: "."` so hooks run from the user's project. Copilot's `postToolUse` entries carry no matcher; the scripts self-filter there.

Copilot is also the one host whose commands are not a single substitution. Each probes `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds the script, and otherwise warns on stderr and exits 0. `COPILOT_PLUGIN_ROOT` — the sole variable until 2026-07-27 — appears in no GitHub documentation, and `${PLUGIN_ROOT}` is the only documented spelling. Unset, the old form left the literal path `/bin/<script>`, and because every non-zero exit denies on this host, a guard that could not start denied every edit instead of merely failing to run. See `copilot-adapter-design.adr`; `@test/structure/copilot-plugin.bats` executes these commands rather than string-matching them.

Every PostToolUse matcher lists each tool under both namings — a project-level `.mcp.json` yields `mcp__archcore__*`, a plugin-bundled server yields `mcp__plugin_archcore_archcore__*`, and Claude Code matchers without regex metacharacters are exact matches. `@test/structure/hooks.bats` guards this; `host-wiring-parity.adr` holds the rationale. Independently of matchers, `bin/lib/normalize-stdin.sh` folds all three namings to the canonical `mcp__archcore__*` before any guard inspects a tool name, so the scripts themselves match exactly one string.

Hooks 2 and 3 share the `Write|Edit` matcher. Hook 2 blocks direct writes to `.archcore/` markdown; hook 3 injects relevant `.archcore/` context for source-file edits. They act on disjoint path sets by construction, so they never conflict.

Hook 6 is the Phase 1 implementation of the Precision Initiative, per `precision-over-coverage.adr`. It emits soft warnings through `additionalContext` for forbidden vague words, missing mandatory sections, frontmatter gaps, and stub-length bodies, and never blocks.

Both PreToolUse guards run on a 1-second budget on every host, and on Copilot a `preToolUse` timeout fails **open**. `@test/unit/hook-latency.bats` keeps them far inside that budget; it exists because the injection guard once cost roughly 6 ms per matching document, so a knowledge base where about 170 documents mentioned a common source root blew the budget outright — and because injection is additive, the only symptom was context silently ceasing to arrive on exactly the repositories with the most of it.

A prior revision had a `PostToolUse` entry with matcher `Write|Edit` invoking `validate-archcore`. It was removed because PreToolUse already blocks every Write and Edit to `.archcore/` markdown and PostToolUse fires only on success, so the matcher was dead weight forking a shell on every write anywhere in the repository. Structure tests guard against its reintroduction.

`bin/session-start` carries a plugin-install-dir guard, per `cursor-mcp-architecture.adr` extended by `host-wiring-parity.adr`. It exits silently — before the CLI availability check, so a misrouted working directory never even sees the install nudge — when `$PWD` contains an install-cache fragment (`.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `.copilot/installed-plugins/`, `plugins/cache/`), or when a bounded depth-12 upward walk finds a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest, which catches a working directory misrouted into a subdirectory of an install. It also emits a 24-hour rate-limited outdated-CLI advisory backed by `archcore update --check`.

### Bin scripts

The `bin/` tree holds hook scripts, shared shell libraries under `bin/lib/`, and the skill helpers. The plugin bundles no Archcore CLI binary and no launcher wrapper; it invokes `archcore` directly from PATH, and users install the CLI globally from https://docs.archcore.ai/cli/install/. `remove-bundled-launcher-global-cli.idea` holds the rationale.

| Script | Hook event | Purpose |
| --- | --- | --- |
| `bin/lib/normalize-stdin.sh` | (library) | Multi-host stdin normalization. Detects the host (Claude Code, Cursor, Copilot, Codex, OpenCode), extracts fields (`tool_name`, `file_path`, `path`; Copilot's native payload carries the path inside an escaped JSON string under `toolArgs`), folds all three MCP tool namings to the canonical one, and provides the output helpers `archcore_hook_block`, `archcore_hook_info`, `archcore_hook_pretool_info`, and `archcore_hook_allow`. Sourced by every hook script except `check-staleness`. |
| `bin/lib/empty-state.sh` | (library) | Defines `archcore_is_functionally_empty [dir]`: `.archcore/` counts as functionally empty when it holds no `.md` file larger than 200 bytes, which filters stubs, `.gitkeep` placeholders, and scaffolds. Pure POSIX, no jq or awk. Sourced by `bin/session-start` to decide the empty-state nudge. |
| `bin/session-start` | SessionStart | Sources the normalizer, then refuses first to run from inside a plugin install — cache-path fragments plus a bounded depth-12 upward manifest walk, exiting silently before any other output. If `archcore` is off PATH, prints an install message pointing at https://docs.archcore.ai/cli/install/, adding the mandatory wiring step `archcore init --agent copilot --project "$PWD"` on Copilot, and exits 0. Otherwise it detects a missing `.archcore/` and emits init guidance instructing the agent to call `mcp__archcore__init_project`, forked on Copilot to the CLI wiring command because no plugin MCP exists there; or invokes `archcore hooks <host> session-start`, then emits the Copilot-only wiring advisory (per-project `cksum` stamp, `ARCHCORE_HIDE_WIRING_NUDGE=1` opt-out), calls `bin/check-staleness`, and emits the rate-limited outdated-CLI advisory. Always exits 0. |
| `bin/check-archcore-write` | PreToolUse | Blocks direct Write and Edit to `.archcore/**/*.md` — exit 2 plus stderr on Claude Code, Codex, and Cursor, and stdout `{"permissionDecision":"deny",…}` with exit 0 on Copilot, which is how a deny carries its reason there, since every non-zero exit denies on that host and exit 2 would block without explaining why. Allows `.archcore/settings.json` and `.archcore/.sync-state.json`, and every path outside `.archcore/`. |
| `bin/check-code-alignment` | PreToolUse | Injects relevant `.archcore/` context for a source-file Write or Edit. Greps `.archcore/**/*.md` for documents referencing the edit path by directory prefix, longest first, ranks by specificity and then type priority (`rule > cpat > adr > spec > guide`), and emits the top 3 through the host's envelope. Cost is independent of how many documents match. Never blocks, always exits 0, and honors both the `ARCHCORE_DISABLE_INJECTION=1` escape hatch and the `.archcore/settings.json` → `codeAlignment.sourceRoots` override. |
| `bin/validate-archcore` | PostToolUse | Runs `archcore doctor` directly under a 2-second timeout after an MCP document operation, selected by tool-name prefix post-normalization so all three namings reach it. The legacy Write/Edit branch is retained as defensive code and is never reached from the current hooks config. Outputs the host envelope when issues are found and nothing when clean. Exits 0 silently when the CLI is unavailable, and always exits 0. |
| `bin/check-staleness` | SessionStart, called by `bin/session-start` | Detects code-document drift through git: finds source files changed since the last `.archcore/` commit and cross-references them with documents mentioning the affected directories. Rate-limited to once per 24 hours through a timestamp file. Emits only when matching documents exist, as plain text capped at 2 KB. Always exits 0. |
| `bin/check-cascade` | PostToolUse | After `update_document`, queries the `.sync-state.json` relation graph for documents connected to the updated one through `implements`, `depends_on`, or `extends`, and outputs the host envelope listing potentially stale dependents, or nothing when no cascade exists. Always exits 0. |
| `bin/check-precision` | PostToolUse | Phase 1 of the Precision Initiative. After `create_document` and `update_document`, reads the resulting file from disk and runs the forbidden-lexicon, mandatory-section, frontmatter, and body-length checks, emitting soft warnings through `additionalContext`. Always exits 0. |
| `bin/git-scope` | (skill helper, wired to no hooks config) | Resolves a capped, ranked directory set from uncommitted working-tree changes — tracked diff against HEAD plus untracked files, with `.archcore/` excluded — for `/archcore:context --git-changes`. The context skill invokes it through Bash. Emits at most 20 directories ranked by changed-file count plus a `__TOTAL__ <raw-dir-count>` trailer, or a single sentinel (`__USAGE__`, `__NO_GIT__`, `__NOT_REPO__`, `__CLEAN__`). Always exits 0. Covered by the Makefile `BIN_SCRIPTS` lint and permissions set. |
| `bin/detect-host` | (skill helper, wired to no hooks config) | Resolves the current AI host from the environment only (`CLAUDECODE` or `CLAUDE_SKILL_DIR` → claude-code, `CURSOR_TRACE_ID` → cursor, `CODEX_HOME` → codex-cli, with precedence claude > cursor > codex), printing exactly one token or `__UNKNOWN__`, and never reading stdin or the working directory. There is deliberately no Copilot branch: every documented `COPILOT_*` variable is read *from* the user rather than set by the host, and the plugin-root variables exist only inside hook processes, where on Copilot which one arrives is itself undocumented (`copilot-adapter-design.adr`). A Copilot session therefore resolves to `__UNKNOWN__` and reaches wiring through the init skill's ask path, which offers all four hosts. Always exits 0. Covered by the Makefile `BIN_SCRIPTS` set. |
| `bin/cli-gte` | (skill helper, wired to no hooks config) | A deterministic semver gate: `cli-gte <min-version>` prints exactly one token — `yes` when the installed `archcore --version` is at least the minimum, `no`, or `__NO_CLI__` when the CLI is missing or unparsable — and always exits 0. It compares numeric fields one by one, so `0.10.0 ≥ 0.6.0` resolves correctly; the init skill's pre-flight host-wiring gate calls it instead of comparing versions in prose (`host-wiring-parity.adr`). Contract in `@test/unit/cli-gte.bats`. Covered by the Makefile `BIN_SCRIPTS` set. |

### Test suite

| Component | Location | Description |
| --- | --- | --- |
| Unit tests | `test/unit/` | Cover each bin script: stdin parsing, host detection, exit codes, output format, edge cases. `session-start-goldens.bats` pins byte-identical non-Copilot outputs per arm and host, which is the isolation gate for Copilot-only additions; `session-start-copilot-wiring.bats` covers the wiring advisory's corner cases (monorepo git-root walk, per-project stamps, kill switch, leak checks); `session-start-emit-matrix.bats` is `_archcore_emit_info`'s five-host shape matrix plus the Copilot single-JSON-document channel pins. `hook-latency.bats` pins both PreToolUse guards inside the 1-second budget and asserts their cost does not scale with match count. `probe-wrapper.bats` proves the probe harness wrapper is transparent across every stdin fixture. |
| Structure tests | `test/structure/` | Validate JSON configs, skill frontmatter, agent frontmatter, hook references, script permissions, and rules. `hooks.bats` resolves every host's hook scripts from one table by script basename — Copilot's live under `bash` rather than `command` and name three candidate roots per command, so both a per-host copy and a single-variable substitution would misread them silently — and carries the anti-regression invariants plus the per-host allowed-variable check. `json-configs.bats` compares name, description, and version across all four manifests. `cursor-plugin.bats` locks the `docs/cursor.mcp.example.json` shape. `codex-plugin.bats` enforces the Codex manifest, marketplace schema, hooks shape, MCP wiring, TOML agents, and wrapper parity. `copilot-plugin.bats` enforces the Copilot manifest pointers, the absence of `mcpServers`, `./`-path resolution, hooks shape, project-root execution, metadata parity, and — by executing the hook commands under `env -u` — the behavior of the plugin-root candidate chain. `probe-hygiene.bats` and `probe-records.bats` guard the probe protocol. `marketplace-discovery.bats` pins all three catalogs' `source` and `path` to the `plugins/archcore` subdirectory, which is the issue #2 regression. `manifest-version-parity.bats` compares the version across all four host manifests; previously Cursor was compared against nothing, so a bump missing it shipped green. `release-blocklist.bats` pins the `dev → main` strip list against `docs/release.md`. |
| Integration tests | `test/integration/` | Host-install smoke tests, `codex-plugin-smoke.bats` and `copilot-plugin-smoke.bats`. Each skips when its host binary is absent, so they ship and stay quiet in CI while running for free on a contributor's machine. |
| Probe harness | `test/probe/` | `mkprobe` builds a disposable wrapped copy of the plugin for live-session verification. See `host-probe-protocol.spec`. |
| Fixtures | `test/fixtures/stdin/` | Mock stdin JSON for Claude Code, Cursor, Copilot, Codex CLI, OpenCode, and malformed inputs. |
| Helpers | `test/helpers/` | `common.bash` (setup, mocks, timeout shim, exports `REPO_ROOT` and `PLUGIN_ROOT`), plus `bats-support` and `bats-assert` as git submodules. |
| Makefile | `Makefile` | Targets `test`, `test-unit`, `test-structure`, `test-codex-smoke`, `test-copilot-smoke`, `lint`, `check-json`, `check-perms`, `verify`. Dev-only; stripped from the `main` distribution. |
| CI | `.github/workflows/test.yml` | GitHub Actions on push and pull request to `dev`: a macOS and Linux matrix running bats and shellcheck. |
| Release | `.github/workflows/release.yml` | GitHub Actions on tag push: strips the dev-only artifacts and force-pushes the clean tree to `main`. `docs/release.md` holds the blocklist. |

Run `make verify` for the full check and `make test` for the tests alone; `plugin-testing.guide` holds the details.

### MCP server

The plugin ships MCP registration for **Claude Code only**, through `@plugins/archcore/.mcp.json`, which registers one server named `archcore` with `command: "archcore"` and `args: ["mcp"]`. The command resolves through PATH, so the user must have the Archcore CLI installed globally. When the CLI is missing at session start, the MCP server fails to register and `bin/session-start` prints the install instructions, with the wiring step appended on Copilot.

Codex CLI uses `.codex-plugin/plugin.json` to point at the plugin-root `.codex.mcp.json`, which uses Codex's direct server map shape with the same command and args.

Cursor and GitHub Copilot CLI are the two exceptions, for the same reason: each launches a plugin's MCP child outside the user's project.

Cursor spawns plugin MCPs from the plugin install directory, and its stdio schema has no `cwd` field, so the plugin ships no `mcp.json` at the plugin root. Cursor users get `.cursor/mcp.json` written by `/archcore:init` or `archcore init --agent cursor`, carrying `--project ${workspaceFolder}`; copying `docs/cursor.mcp.example.json` by hand remains the fallback before wiring has run. See `cursor-mcp-architecture.adr`.

Copilot launches a plugin's MCP child with the working directory set to the plugin install root and passes it no project path at all, not even the `COPILOT_PROJECT_DIR` its hooks receive (github/copilot-cli#4234). Documents would be written into `~/.copilot/installed-plugins/` while every tool reported success, so `.plugin/plugin.json` declares no `mcpServers`, and Copilot users get a workspace-root `.mcp.json` from `archcore init --agent copilot` on CLI v0.6.4 or later. The Cursor remedy of moving the file out of the plugin root was unavailable, because `.mcp.json` must stay where Claude Code finds it. See `copilot-mcp-architecture.adr`.

`remove-bundled-launcher-global-cli.idea` holds the rationale for the no-bundled-CLI stance. The previous bundled launcher — download-on-first-use, checksum-verified, cached per host — is removed, so the CLI lifecycle decouples cleanly from plugin releases.

### Plugin configs

Component manifests, hooks, and MCP configs are plugin-root-relative under `plugins/archcore/`. The marketplace catalogs and the Cursor reference template live at the repo root.

| File | Host | Purpose |
| --- | --- | --- |
| `.claude-plugin/plugin.json` | Claude Code | Plugin manifest (plugin root) |
| `.cursor-plugin/plugin.json` | Cursor | Plugin manifest (plugin root) with explicit component paths and **no `mcpServers` field**, disabled deliberately per `cursor-mcp-architecture.adr` |
| `.codex-plugin/plugin.json` | Codex CLI | Plugin manifest (plugin root) with `skills`, `hooks`, and `mcpServers` pointers |
| `.plugin/plugin.json` | GitHub Copilot CLI | Plugin manifest (plugin root) with explicit `skills`, `agents` (→ `copilot-agents/`), `commands`, and `hooks` pointers, and **no `mcpServers` field**, disabled deliberately per `copilot-mcp-architecture.adr` |
| `.claude-plugin/marketplace.json` | Claude Code | Marketplace catalog — repo root, `source: ./plugins/archcore` |
| `.cursor-plugin/marketplace.json` | Cursor | Marketplace catalog — repo root, `source: ./plugins/archcore` |
| `.agents/plugins/marketplace.json` | Codex CLI | Marketplace catalog and default-install policy — repo root, `source.path: ./plugins/archcore` |
| (none) | GitHub Copilot CLI | No catalog by design — installs by subdirectory spec `archcore-ai/plugin:plugins/archcore` |
| `.mcp.json` | Claude Code | Plugin-provided MCP registration (plugin root; `command: "archcore"` on PATH) |
| `.codex.mcp.json` | Codex CLI | Plugin-provided MCP registration (plugin root; `command: "archcore"` on PATH) |
| `docs/cursor.mcp.example.json` | Cursor | Reference MCP config for users to copy into `~/.cursor/mcp.json` or `.cursor/mcp.json` — repo root |
| `hooks/hooks.json` | Claude Code | Hook event config (PascalCase) |
| `hooks/cursor.hooks.json` | Cursor | Hook event config (camelCase plus `afterMCPExecution`) |
| `hooks/codex.hooks.json` | Codex CLI | Hook event config (PascalCase plus `apply_patch` matcher) |
| `hooks/copilot.hooks.json` | GitHub Copilot CLI | Hook event config (camelCase, native mutation matchers, project-root cwd, plugin-root candidate chain) |
| `commands/*.md` | Codex CLI, GitHub Copilot CLI | Slash-command wrappers (7) — thin shims delegating to `skills/<name>/SKILL.md` |
| `agents/archcore-assistant.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "workspace-write"`) |
| `agents/archcore-auditor.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "read-only"` plus `disabled_tools[]` under all MCP namings) |
| `copilot-agents/*.agent.md` | GitHub Copilot CLI | Byte-identical copies of the MD agents under the extension Copilot's loader requires |
| `rules/archcore-context.mdc` | Cursor | Always-apply context rule |
| `rules/archcore-files.mdc` | Cursor | `.archcore/` glob-triggered MCP-only rule |

## Examples

The visible `/` surface, as a non-normative illustration of the skills table above:

- `/archcore:init` — seed an empty `.archcore/` on first install
- `/archcore:capture` — document a module or component
- `/archcore:decide` — record a decision (ADR) or draft a proposal (RFC)
- `/archcore:plan` — plan a feature end to end, as a single plan or a full flow
- `/archcore:audit` — dashboard by default, `--deep` audit, or `--drift` detection
- `/archcore:context` — rules and decisions for a code area or pickup
- `/archcore:help` — system guide

Total visible in the `/` menu: **7 commands**. Every Archcore document type is reachable through these skills or directly through `mcp__archcore__create_document(type=<any>)`. Codex CLI surfaces the same 7 entries through the `commands/*.md` wrappers, and Copilot CLI surfaces them from the skills with the wrappers loaded behind them.
