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

Per `four-command-palette.adr`, the visible `/` palette is exactly four auto-invocable commands (`init`, `plan`, `document`, `review`), with a non-palette gated track layer per `track-layer.spec`. There are no per-document-type skills, no track skills, and no utility skills. Track flows live as gated track files under `skills/_shared/tracks/`, loaded on demand by the commands that route into them.

**Layout.** Every component path here is plugin-root-relative, and the plugin root is the `plugins/archcore/` subdirectory — `skills/init/` means `plugins/archcore/skills/init/`, and `bin/session-start` means `plugins/archcore/bin/session-start`. The three marketplace catalogs are the exception: they live at the repo root and point their `source` and `path` at `./plugins/archcore`. That root-catalog and subdirectory-manifest split is required for Codex marketplace discovery, because a catalog `source.path` of `./` is never scanned — see `subdirectory-plugin-layout.adr` and issue #2. There are three catalogs and four hosts because Copilot CLI has no marketplace and installs by subdirectory spec. The reference template `docs/cursor.mcp.example.json` also stays at the repo root.

## Content

### Skills (4, auto-invocable by model and user)

No skill carries a `disable-model-invocation` flag. Routing is governed by per-skill `Activate when X. Do NOT activate for Y.` descriptions.

| Skill | Directory | User Intent |
| --- | --- | --- |
| init | `skills/init/` | First-time onboarding — seed `.archcore/` with a stack rule, a run guide, the extras for the detected scale, plus host wiring (project MCP config, hooks, usage hint — the same files `archcore init` writes; see `host-wiring-parity.adr`) |
| plan | `skills/plan/` | Plan a feature or initiative — vision-type write affinity, routing into the `sdd`, `requirements-cascade`, or `research` track per `track-layer.spec` |
| document | `skills/document/` | Record a decision or document code — knowledge-type write affinity; absorbs `capture` and `decide`, classifying the target as decision, code-doc, or unclear before routing into the `decision` or `describe` track |
| review | `skills/review/` | Review changes since the default branch, or project health with no diff — experience-type write affinity; absorbs `audit`, routing into the `actualize`, `experience`, or `closeout` track and labeling findings `spec-wrong` / `code-wrong` / `ok` |

`context` and `help` are removed: `context`'s pull moment moves to CLI hook injection, and `help` is absorbed into command descriptions, the init closing summary, and CLI `archcore help` — see `remove-context-command.adr` and `command-surface-v2.spec`.

### Track files (`skills/_shared/tracks/`)

Gated flows beneath the command surface, per `track-layer.spec`. Each is a sequence of gate records with entry and skip predicates, an elicitation budget, and blocking/advisory exit checks:

| Track | Path | Routed from | Flow |
| --- | --- | --- | --- |
| decision | `skills/_shared/tracks/decision.md` | document | classify → adr \| rfc → cascade; `decision.resolve` entry on an existing rfc draft |
| describe | `skills/_shared/tracks/describe.md` | document | read code → spec/doc/guide draft → evidence-gap clarify |
| sdd | `skills/_shared/tracks/sdd.md` | plan | frame → require → design → decompose |
| requirements-cascade | `skills/_shared/tracks/requirements-cascade.md` | plan | `mode: sources` (mrd → brd → urd) \| `mode: iso` (brs → strs → syrs → srs) |
| research | `skills/_shared/tracks/research.md` | plan | frame questions → gather evidence → conclude with recommendation (produces `rnd`) |
| actualize | `skills/_shared/tracks/actualize.md` | review | drift detection → per-finding verdict → confirmed fixes |
| experience | `skills/_shared/tracks/experience.md` | review | repeated-pattern detection → `cpat` \| `task-type` offer |
| closeout | `skills/_shared/tracks/closeout.md` | review | verify plan vs branch diff → confirmed canon merge → confirmed draft → accepted transitions, per `document-status-transitions.adr` |

### Shared runtime assets (`skills/_shared/`)

Plain-markdown assets loaded at runtime before a skill composes a document. They ship with the plugin, and skill instructions reference plugin-internal paths only, never the consumer's `.archcore/`.

| Asset | Path | Loaded by | Purpose |
| --- | --- | --- | --- |
| `precision-rules.md` | `skills/_shared/precision-rules.md` | `document`, `plan`, `init`, `review` | Forbidden vagueness lexicon, imperative phrasing, no-cross-document-section rule, `[assumption]` marker conventions |
| `adr-contract.md` | `skills/_shared/adr-contract.md` | `document` (ADR) | Mandatory sections plus good and bad examples for ADR content, per MADR 4.0 |
| `prd-contract.md` | `skills/_shared/prd-contract.md` | `plan` (`sdd.require`) | Mandatory sections, the outcome-shaped requirement form, the scope rule and its compression path, and the content-kind ownership table that assigns each statement on a topic to one document, per `prd-spec-plan-content-ownership.adr` |
| `spec-contract.md` | `skills/_shared/spec-contract.md` | `document` (spec), `init` (hotspot specs) | Mandatory sections and the "when NOT to write a spec" gate |
| `rule-contract.md` | `skills/_shared/rule-contract.md` | `document` (rule), `init` (cross-cutting rules) | Mandatory rule body: RFC 2119 statement, applies-to scope, rationale, Good/Bad examples, enforcement |
| `elicitation-contract.md` | `skills/_shared/elicitation-contract.md` | all four commands | Bounded user interview — batching, per-gate budgets, the 5-question auto-mode ceiling, escape hatch |
| `gate-contract.md` | `skills/_shared/gate-contract.md` | track files | Gate record template with the fixed six-field order |
| `branch-state.md` | `skills/_shared/branch-state.md` | `plan`, `review` | Plain-git branch boundary: merge-base against the default branch, changed-file listing, sentinels for no-branch / detached HEAD / on-default |
| `coverage-taxonomy.md` | `skills/_shared/coverage-taxonomy.md` | `plan` tracks | Per-family coverage categories mapped to vision/knowledge/experience, and the per-type destinations of `Completion Signals` |
| `globals.md` | `skills/_shared/globals.md` | all four commands, on global-source results only | Local/global reading convention — never modify or relate to a mounted global document |
| `grounding/*.md` (13) | `skills/_shared/grounding/` | `init` primarily; other commands on demand | Detect/extract catalogs (stack, scale, modules, hotspots, data model, integrations, config, entry points, surface, domains, cross-cutting, routing imports, run instructions) |

`precision-over-coverage.adr` records the design rationale for these assets.

### Slash-command wrappers (`commands/`)

Codex CLI requires a thin wrapper per user-facing skill so that `/archcore:<name>` appears in its `/` menu. Each wrapper is a host-adapter shim: `description:` frontmatter plus a one-line delegate instruction pointing at `skills/<name>/SKILL.md`. No workflow logic lives here.

| Wrapper set | Count | Purpose |
| --- | --- | --- |
| `commands/<name>.md` | 4 | One per skill (`init`, `plan`, `document`, `review`) — surfaces `/archcore:<name>` on Codex CLI, and on Copilot CLI as a fallback behind the skill of the same name |

`@test/structure/codex-plugin.bats` enforces that every entry exists, carries `description:`, and references the matching `skills/<name>/SKILL.md`. `@test/structure/copilot-plugin.bats` enforces that the wrapper set matches the skill set and that the manifest points at `commands/` — Copilot gives that field no default path, so without the pointer the entire `/archcore:*` surface is missing there. Claude Code and Cursor need no wrappers, because they surface skills directly.

### Document-type coverage and the visible `/` surface

Every Archcore document type is reachable through an intent skill or directly through `mcp__archcore__create_document(type=<any>)`; `command-surface-v2.spec` holds the full mapping under "Document-type coverage". All 4 skills on disk are visible, with no hidden or flagged-out skill. Codex CLI exposes the same 4 entries through the `commands/*.md` wrappers, and Copilot exposes them from the skills with the wrappers as backup.

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

### Hooks (3 entries across 3 events)

Since v0.7.0 the hook policy — write guard, code-alignment injection, validation, cascade, precision, staleness, ranked recap — executes inside the archcore CLI; the plugin registers one launcher per event (`cli-owns-layers-4-5.adr`, `hooks-validation-system.spec`).

| # | Event | Matcher | Handler | Timeout |
| --- | --- | --- | --- | --- |
| 1 | SessionStart | (all) | `bin/session-start` | — |
| 2 | PreToolUse | `Write\|Edit` | `bin/pre-tool-use` | 2s |
| 3 | PostToolUse | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` (+ `mcp__plugin_archcore_archcore__*` twins) | `bin/post-tool-use` | 4s |

Hook configs: `hooks/hooks.json` for Claude Code with PascalCase events; `hooks/cursor.hooks.json` for Cursor with camelCase events plus `afterMCPExecution` (no matcher), whose `preToolUse` matcher is `Write` only because Cursor exposes no Edit tool; `hooks/codex.hooks.json` for Codex CLI with PascalCase events, an `apply_patch` matcher addition for Codex's native edit primitive, commands using Codex's canonical `${PLUGIN_ROOT}` substitution, and a deterministic `ARCHCORE_HOST=codex` carried as an assignment prefix, because Codex offers no `env` field on a hook handler; and `hooks/copilot.hooks.json` for Copilot CLI with native camelCase events under a `bash` key rather than `command`, `timeoutSec` rather than `timeout`, a deterministic `ARCHCORE_HOST=copilot`, and `cwd: "."` so hooks run from the user's project. Copilot's `postToolUse` entry carries no matcher; the CLI gates on the tool name there.

The Codex host id is pinned rather than inferred because a Codex `SessionStart` payload carries no `turn_id` — the stdin heuristic's Codex marker, documented for turn-scoped events only — so before v0.7.3 every Codex session start was processed as Claude Code, which called the CLI with the wrong dialect and selected the wrong stdout shape (`codex-adapter-conformance.adr`).

Copilot is also the one host whose commands are not a single substitution. Each probes `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds the script, and otherwise warns on stderr and exits 0. `COPILOT_PLUGIN_ROOT` — the sole variable until 2026-07-27 — appears in no GitHub documentation, and `${PLUGIN_ROOT}` is the only documented spelling. Unset, the old form left the literal path `/bin/<script>`, and because every non-zero exit denies on this host, a guard that could not start denied every edit instead of merely failing to run. See `copilot-adapter-design.adr`; `@test/structure/copilot-plugin.bats` executes these commands rather than string-matching them.

Every PostToolUse matcher lists each tool under both namings — a project-level `.mcp.json` yields `mcp__archcore__*`, a plugin-bundled server yields `mcp__plugin_archcore_archcore__*`, and Claude Code matchers without regex metacharacters are exact matches. `@test/structure/hooks.bats` guards this; `host-wiring-parity.adr` holds the rationale. The CLI recognizes all three namings — including Copilot's flat `archcore-<tool>` — when it gates on the tool name behind a matcherless event. On Codex a matcher always sees the `mcp__<server>__<tool>` form, even where the model-visible namespace drops that prefix, because the host rebuilds the canonical name before matching.

The launchers fail OPEN: absent CLI, CLI older than 0.7.0 (`bin/cli-gte` gate), or a plugin-cache cwd all yield a silent exit 0, so a deny can originate only in the CLI's own verdict. A prior revision had a `PostToolUse` entry with matcher `Write|Edit` invoking validation. It was removed because PreToolUse already blocks every Write and Edit to `.archcore/` markdown and PostToolUse fires only on success, so the matcher was dead weight forking a shell on every write anywhere in the repository. Structure tests guard against its reintroduction.

`bin/session-start` carries a plugin-install-dir guard, factored into `bin/lib/plugin-cache-guard.sh` and shared with the launchers, per `cursor-mcp-architecture.adr` extended by `host-wiring-parity.adr`. It exits silently — before the CLI availability check, so a misrouted working directory never even sees the install nudge — when `$PWD` contains an install-cache fragment (`.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `.copilot/installed-plugins/`, `plugins/cache/`), or when a bounded depth-12 upward walk finds a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest. It also emits a 24-hour rate-limited outdated-CLI advisory backed by `archcore update --check`.

### Bin scripts

The `bin/` tree holds the three hook launchers, shared shell libraries under `bin/lib/`, and the skill helpers. The plugin bundles no Archcore CLI binary, no launcher wrapper, and no policy scripts; it invokes `archcore` directly from PATH, and users install the CLI globally from https://docs.archcore.ai/cli/install/. `remove-bundled-launcher-global-cli.idea` holds the rationale.

| Script | Hook event | Purpose |
| --- | --- | --- |
| `bin/lib/normalize-stdin.sh` | (library) | Multi-host stdin normalization. Captures raw stdin, detects the host (Claude Code, Cursor, Copilot, Codex, OpenCode), extracts fields, folds all three MCP tool namings to the canonical one, and provides the output helpers `bin/session-start` uses. Sourced by every stdin-reading script. |
| `bin/lib/plugin-cache-guard.sh` | (library) | Shared misrouted-cwd guard: install-cache path fragments plus a bounded depth-12 upward manifest walk, both silent-exit. Sourced by the launchers. |
| `bin/lib/empty-state.sh` | (library) | Defines `archcore_is_functionally_empty [dir]`: `.archcore/` counts as functionally empty when it holds no `.md` file larger than 200 bytes, which filters stubs, `.gitkeep` placeholders, and scaffolds. Pure POSIX, no jq or awk. Sourced by `bin/session-start` to decide the empty-state nudge. |
| `bin/session-start` | SessionStart | Sources the normalizer, then refuses to run from inside a plugin install. If `archcore` is off PATH, prints an install message pointing at https://docs.archcore.ai/cli/install/, adding the mandatory wiring step `archcore init --agent copilot --project "$PWD"` on Copilot, and exits 0. Otherwise it detects a missing `.archcore/` and emits init guidance instructing the agent to call `mcp__archcore__init_project`, forked on Copilot to the CLI wiring command because no plugin MCP exists there; or invokes `archcore hooks <cli-host> session-start` (mapping `codex` → `codex-cli`), then emits the Copilot-only wiring advisory (per-project `cksum` stamp, `ARCHCORE_HIDE_WIRING_NUDGE=1` opt-out) and the rate-limited outdated-CLI advisory. On Copilot and Codex — the two hosts that reject stdout mixing one JSON document with trailing text — the CLI payload is captured rather than streamed and every advisory is folded into that single document. Staleness arrives inside the CLI recap. Always exits 0. |
| `bin/pre-tool-use` | PreToolUse | Launcher for `archcore hooks <cli-host> pre-tool-use` — the CLI's write guard plus code-alignment context. Cache-guarded, gated on `cli-gte 0.7.0`, byte-transparent: stdin, stdout, stderr, and the exit code pass through unchanged, so exit 2 + stderr denies exactly as the CLI decided. Fails open silently without a current CLI. |
| `bin/post-tool-use` | PostToolUse | Launcher for `archcore hooks <cli-host> post-tool-use` — the CLI's in-process doctor validation, cascade notice, and precision scan. Same contract as `bin/pre-tool-use`; the CLI always exits 0 on this event. |
| `bin/detect-host` | (skill helper, wired to no hooks config) | Resolves the current AI host from the environment only (`CLAUDECODE` or `CLAUDE_SKILL_DIR` → claude-code, `CURSOR_TRACE_ID` → cursor, `CODEX_THREAD_ID` or `CODEX_HOME` → codex-cli, with precedence claude > cursor > codex), printing exactly one token or `__UNKNOWN__`, and never reading stdin or the working directory. `CODEX_THREAD_ID` is the host-injected marker — Codex inserts it into every model-run shell command after environment filtering — while `CODEX_HOME` is a user-set config-directory override kept only as a fallback; keyed on `CODEX_HOME` alone the branch never fired and every Codex session resolved `__UNKNOWN__` (`codex-adapter-conformance.adr`). Both Codex surfaces, CLI and desktop app, resolve to the same `codex-cli` id. There is deliberately no Copilot branch: every documented `COPILOT_*` variable is read *from* the user rather than set by the host, and the plugin-root variables exist only inside hook processes, where on Copilot which one arrives is itself undocumented (`copilot-adapter-design.adr`). A Copilot session therefore resolves to `__UNKNOWN__` and reaches wiring through the init skill's ask path, which offers all four hosts. Always exits 0. Covered by the Makefile `BIN_SCRIPTS` set. |
| `bin/cli-gte` | (skill helper and launcher gate) | A deterministic semver gate: `cli-gte <min-version>` prints exactly one token — `yes` when the installed `archcore --version` is at least the minimum, `no`, or `__NO_CLI__` when the CLI is missing or unparsable — and always exits 0. It compares numeric fields one by one, so `0.10.0 ≥ 0.6.0` resolves correctly. The init skill's pre-flight host-wiring gate and both launchers call it (pinned minimum 0.7.0) instead of comparing versions in prose (`host-wiring-parity.adr`). Contract in `@test/unit/cli-gte.bats`. Covered by the Makefile `BIN_SCRIPTS` set. |

### Test suite

| Component | Location | Description |
| --- | --- | --- |
| Unit tests | `test/unit/` | Cover each bin script: stdin parsing, host detection, exit codes, output format, edge cases. `hook-launchers.bats` pins the launcher contract: fail-open without a current CLI, host-id mapping, byte-transparent stdin/stdout/exit passthrough, and the plugin-cache guard. `session-start-goldens.bats` pins byte-identical non-Copilot outputs per arm and host, which is the isolation gate for Copilot-only additions; `session-start-copilot-wiring.bats` covers the wiring advisory's corner cases (monorepo git-root walk, per-project stamps, kill switch, leak checks); `session-start-emit-matrix.bats` is `_archcore_emit_info`'s five-host shape matrix plus the Copilot single-JSON-document channel pins; `session-start-codex-stdout.bats` pins the Codex single-document contract — pass-through, advisory splicing, escaped quotes, and the stderr fallback. `probe-wrapper.bats` proves the probe harness wrapper is transparent across every stdin fixture. |
| Structure tests | `test/structure/` | Validate JSON configs, skill frontmatter, agent frontmatter, hook references, script permissions, and rules. `hooks.bats` resolves every host's hook scripts from one table by script basename — Copilot's live under `bash` rather than `command` and name three candidate roots per command, so both a per-host copy and a single-variable substitution would misread them silently — and carries the anti-regression invariants plus the per-host allowed-variable check and the `ARCHCORE_HOST=codex` pin. `v2-purity.bats` asserts zero stale v1 command strings across the shipped tree. `json-configs.bats` compares name, description, and version across all four manifests. `cursor-plugin.bats` locks the `docs/cursor.mcp.example.json` shape. `codex-plugin.bats` enforces the Codex manifest, marketplace schema, hooks shape, MCP wiring, TOML agents, and wrapper parity. `copilot-plugin.bats` enforces the Copilot manifest pointers, the absence of `mcpServers`, `./`-path resolution, hooks shape, project-root execution, metadata parity, and — by executing the hook commands under `env -u` — the behavior of the plugin-root candidate chain. `track-goldens.bats` pins every track's gate records (routing structure, skip paths, budgets, taxonomy, produces) against `test/fixtures/goldens/<track>.golden`. `trigger-routing.bats` pins trigger phrases from `test/fixtures/routing/fixtures.tsv` against skill descriptions and track entry gates. `probe-hygiene.bats` and `probe-records.bats` guard the probe protocol. `marketplace-discovery.bats` pins all three catalogs' `source` and `path` to the `plugins/archcore` subdirectory, which is the issue #2 regression. `manifest-version-parity.bats` compares the version across all four host manifests; previously Cursor was compared against nothing, so a bump missing it shipped green. `release-blocklist.bats` pins the `dev → main` strip list against `docs/release.md`. |
| Integration tests | `test/integration/` | Host-install smoke tests, `codex-plugin-smoke.bats` and `copilot-plugin-smoke.bats`. Each skips when its host binary is absent, so they ship and stay quiet in CI while running for free on a contributor's machine. |
| Probe harness | `test/probe/` | `mkprobe` builds a disposable wrapped copy of the plugin for live-session verification. See `host-probe-protocol.spec`. |
| Fixtures | `test/fixtures/stdin/` | Mock stdin JSON for Claude Code, Cursor, Copilot, Codex CLI, OpenCode, and malformed inputs. The Codex `session-start.json` and `mcp-list-documents.json` payloads are captured from a live session per `host-probe-protocol.spec` item 9. |
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
| `hooks/codex.hooks.json` | Codex CLI | Hook event config (PascalCase plus `apply_patch` matcher, `ARCHCORE_HOST=codex` prefix) |
| `hooks/copilot.hooks.json` | GitHub Copilot CLI | Hook event config (camelCase, native mutation matchers, project-root cwd, plugin-root candidate chain) |
| `commands/*.md` | Codex CLI, GitHub Copilot CLI | Slash-command wrappers (4) — thin shims delegating to `skills/<name>/SKILL.md` |
| `agents/archcore-assistant.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "workspace-write"`) |
| `agents/archcore-auditor.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "read-only"` plus `disabled_tools[]` under all MCP namings) |
| `copilot-agents/*.agent.md` | GitHub Copilot CLI | Byte-identical copies of the MD agents under the extension Copilot's loader requires |
| `rules/archcore-context.mdc` | Cursor | Always-apply context rule |
| `rules/archcore-files.mdc` | Cursor | `.archcore/` glob-triggered MCP-only rule |

## Examples

The visible `/` surface, as a non-normative illustration of the skills table above:

- `/archcore:init` — seed an empty `.archcore/` on first install
- `/archcore:plan` — plan a feature end to end, as a single plan or a full gated track flow
- `/archcore:document` — document a module or record a decision (ADR/RFC)
- `/archcore:review` — dashboard by default, `--deep` audit, or `--drift` detection

Total visible in the `/` menu: **four commands**. Every Archcore document type is reachable through these skills or directly through `mcp__archcore__create_document(type=<any>)`. Codex CLI surfaces the same 4 entries through the `commands/*.md` wrappers, and Copilot CLI surfaces them from the skills with the wrappers loaded behind them.
