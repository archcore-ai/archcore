---
title: "Multi-Host Plugin Architecture — Single Repo for Multiple AI Coding Tools"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

The Archcore plugin must work across multiple AI coding hosts that have converged on common open standards (April 2026 research):

- **Agent Skills standard** (agentskills.io) — adopted by Cursor, GitHub Copilot, Codex CLI, Roo Code, Cline, Gemini CLI, Windsurf, JetBrains Junie, OpenHands
- **MCP (Model Context Protocol)** — adopted by all of the above plus Amazon Q, Continue.dev, Zed AI
- **Markdown agent definitions** — adopted by Cursor, GitHub Copilot, Codex CLI, Gemini CLI

Analysis showed that ~95% of plugin code is host-agnostic: skills, agents, hook scripts, and the stdin-normalization library. Only manifest files and hooks configs are host-specific.

### Drivers

- Users of Cursor, Copilot, and Codex CLI need Archcore integration — Codex CLI v0.117.0+ (March 2026) added a plugin system with near-1:1 surface to Claude Code, validating the multi-host bet; GitHub Copilot CLI followed as the fourth host.
- Industry convergence on Agent Skills + MCP makes cross-host support low-effort.
- Maintaining separate repos per host would mean duplicating every skill, agent, and hook script.

## Decision

**Support multiple AI coding hosts from a single repository** with a shared core and thin per-host adapter layer.

The plugin ships:

- **Shared core** (host-agnostic): `skills/`, `agents/` (both `.md` and `.toml` variants), `copilot-agents/` (`*.agent.md` copies, byte-identical), `bin/` (hook scripts + stdin normalizer), `commands/` (slash command wrappers).
- **Per-host adapter files** (configuration only, no logic): plugin manifest, hooks config, and — where the host can safely run one — an MCP config.
- **No bundled CLI**: the Archcore CLI is installed globally by the user per https://docs.archcore.ai/cli/install/. Every MCP config names `archcore` as the command, resolved via PATH. The plugin does not bundle, download, cache, or pin a CLI binary.

The plugin itself lives in a dedicated `plugins/archcore/` subdirectory; the marketplace catalogs stay at the repo root and point each host's plugin `source`/`path` at that subdirectory.

```
repo-root/                               # marketplace CATALOGS + dev tooling
├── .claude-plugin/marketplace.json      # Claude catalog  → source: ./plugins/archcore
├── .cursor-plugin/marketplace.json      # Cursor catalog  → source: ./plugins/archcore
├── .agents/plugins/marketplace.json     # Codex catalog   → path:   ./plugins/archcore
│                                        # (no Copilot catalog — it installs by subdir spec)
├── docs/cursor.mcp.example.json         # Reference template users copy into ~/.cursor/mcp.json
│
└── plugins/archcore/                    # ← the plugin (single source of truth; what each host installs)
    ├── commands/                        # Slash command wrappers (7, host-adapter shims)
    ├── skills/                          # Shared — Agent Skills standard (7 skills)
    ├── agents/                          # Shared — markdown agent definitions + Codex TOML variants
    │   ├── archcore-assistant.md        # Claude Code / Cursor
    │   ├── archcore-assistant.toml      # Codex CLI (sandbox_mode = "workspace-write")
    │   ├── archcore-auditor.md          # Claude Code / Cursor
    │   └── archcore-auditor.toml        # Codex CLI (sandbox_mode = "read-only" + disabled_tools)
    ├── copilot-agents/                  # GitHub Copilot CLI — *.agent.md copies (cmp-tested)
    │   ├── archcore-assistant.agent.md
    │   └── archcore-auditor.agent.md
    ├── bin/                             # Shared — hook scripts + stdin normalizer (no CLI binary)
    │   ├── lib/normalize-stdin.sh
    │   ├── session-start
    │   ├── check-archcore-write
    │   ├── check-code-alignment
    │   ├── validate-archcore
    │   ├── check-cascade
    │   ├── check-precision
    │   └── check-staleness
    │
    ├── .claude-plugin/plugin.json       # Claude Code manifest — mcpServers: "./.claude.mcp.json"
    ├── .cursor-plugin/plugin.json       # Cursor manifest (no `mcpServers` field — deliberate)
    ├── .codex-plugin/plugin.json        # Codex CLI manifest (single file)
    ├── .plugin/plugin.json              # GitHub Copilot CLI manifest — mcpServers: {} (deliberate; see below)
    │
    ├── hooks/
    │   ├── hooks.json                   # Claude Code (PascalCase events)
    │   ├── cursor.hooks.json            # Cursor (camelCase events)
    │   ├── codex.hooks.json             # Codex CLI (PascalCase events + apply_patch matcher)
    │   └── copilot.hooks.json           # GitHub Copilot CLI (camelCase events, "bash" + timeoutSec)
    │
    ├── .claude.mcp.json                 # Claude Code — reached ONLY via the manifest key, never by discovery
    ├── .codex.mcp.json                  # Codex CLI — direct server map
    └── rules/                           # Cursor-only context rules (.mdc)
```

**Catalog vs. plugin location.** The three marketplace catalogs stay at the repo root; each points its plugin `source`/`path` at the `plugins/archcore/` subdirectory, which holds the per-host manifests and all shared content. This subdirectory layout is *required* for Codex marketplace discovery — a catalog `source.path` of `./` (the marketplace root) is not scanned, so the plugin is never discovered there — and it is the canonical layout for Claude Code and Cursor as well. Copilot CLI has no marketplace concept at all and installs by subdir spec (`archcore-ai/plugin:plugins/archcore`), so it needs no fourth catalog. The reporter that surfaced the Codex case is issue #2; the full rationale, the cross-host docs matrix, and the rejected alternatives (generated copy, Windows-breaking symlinks) live in `subdirectory-plugin-layout.adr`, which extends this ADR.

### Shared core principle

Skills, agents, and hook scripts are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### MCP wiring

MCP is wired via plugin-shipped configs on the two hosts where that is safe: Claude Code (`.claude.mcp.json`, reached through the `mcpServers` key in `.claude-plugin/plugin.json`) and Codex CLI (`.codex.mcp.json`, reached through the same key in `.codex-plugin/plugin.json`). Both name `archcore` directly; Codex uses a direct server map, Claude Code uses its `mcpServers` wrapper. The host runtime resolves `archcore` from PATH. Both hosts launch the MCP with cwd inherited from the user's project process, which is the correct workspace.

**No MCP config sits on a filename any host auto-discovers.** Until 0.6.2 the Claude config was `.mcp.json` at the plugin root, discovered by convention with no manifest key — and discovered, it turned out, by two other hosts as well. Both files are now off every discovery list and reachable only through an explicit manifest key, which is what makes "the plugin ships MCP to exactly two hosts" an enforced property rather than an intention. `test/structure/plugin-mcp-isolation.bats` holds the contract.

**Cursor and GitHub Copilot CLI are the exceptions, for the same underlying reason: the host launches a plugin's MCP child somewhere other than the user's project.**

Cursor 2.5+ auto-detects plugin-shipped MCP configs (per the [official plugins reference](https://cursor.com/docs/reference/plugins.md), an `mcp.json` at the plugin root registers under "Plugin MCP Servers"), but it spawns the plugin-MCP from the plugin install directory rather than the workspace, and its MCP stdio schema has no `cwd` field ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861), [forum #99215](https://forum.cursor.com/t/how-get-the-correct-current-work-directory-in-mcp-server/99215)). We therefore ship no plugin-MCP for Cursor: no `mcpServers` field in `.cursor-plugin/plugin.json` and no discoverable MCP filename at the plugin root, with the reference template under `docs/`. Note that its discovery list is `[".mcp.json", "mcp.json"]` — **both** spellings — so before 0.6.2 the Claude config was being registered here too; keeping the plugin root free of "Cursor's filename" was never sufficient. See `cursor-mcp-architecture.adr`.

Copilot CLI launches a plugin's MCP child with cwd set to the plugin install root and passes it no project path — not even the `COPILOT_PROJECT_DIR` its plugin *hooks* receive ([github/copilot-cli#4234](https://github.com/github/copilot-cli/issues/4234)). Documents would be written into `~/.copilot/installed-plugins/` while every tool reported success. Worse, the plugin server and the project server that `archcore init --agent copilot` registers share the key `archcore`, and Copilot merges MCP sources last-wins — so the plugin entry silently *replaced* the one the user was told to create. `.plugin/plugin.json` therefore declares `mcpServers: {}`: an empty declaration rather than an absent key, because an absent key sends Copilot to `.claude-plugin/plugin.json` for a fallback and it adopts the Claude server from there. Copilot users register the server per project via `archcore init --agent copilot`. The Cursor remedy of moving the file off the discovered name **was** available here, contrary to what this ADR said until 2026-08-03 — Claude Code accepts an explicit manifest path, so nothing forced the file to stay at `.mcp.json`. See `copilot-mcp-architecture.adr`.

The plugin does not bundle the CLI, does not download it, and does not cache it; users install it once via the official installer at https://docs.archcore.ai/cli/install/.

### Stdin normalization

Hook scripts source a shared `bin/lib/normalize-stdin.sh` that detects the host from stdin JSON structure and exposes a canonical schema (`ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, etc.). Detection uses each host's distinct stdin fields (Claude Code → `tool_name`; Cursor → `conversation_id`; Copilot → `hookEventName` on legacy payloads, `toolName`/`toolArgs` on native ones; Codex → `turn_id`). Codex shares Claude Code's snake_case schema, so the field-extraction logic for `codex` mirrors `claude-code`. The normalizer also folds all three MCP tool namings — `mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, and Copilot's flat `archcore-<tool>` — to one canonical name, so guard scripts match exactly one string instead of three.

## Alternatives Considered

### 1. Separate repository per host

One repo per host, each containing full copies of skills, agents, and bin scripts.

**Rejected because:** duplication scales with host count; any skill update must be synced across all repos; only ~5% of code is actually host-specific.

### 2. Build system that generates per-host packages

A mono-repo with a build step (e.g., Node.js script) that reads a canonical source and generates separate plugin directories per host.

**Rejected because:** introduces build tooling to a project that is currently pure Markdown + Shell; complexity not warranted — the per-host differences are purely configuration (JSON files); Agent Skills standard already ensures skills work across hosts without transformation.

### 3. Symlinks from host-specific directories to shared source

**Rejected because:** symlinks don't work reliably on Windows; plugin marketplace systems distribute files, not symlinks; fragile when cloned or copied.

### 4. Bundle the CLI inside the plugin (download-on-first-use launcher)

Ship a `bin/archcore` launcher that resolves the Archcore CLI on demand from `$ARCHCORE_BIN`, PATH, a plugin-managed cache, or a GitHub Releases download.

**Tried and reverted.** Shipped briefly under `bundled-cli-launcher.adr` (now rejected), then removed in plugin v0.4.0 (2026-05-12) per `remove-bundled-launcher-global-cli.idea`. Eight bug classes — offline CI failures, security patch lag, uneven host support (Cursor still required manual setup), cache pollution, first-run latency, enterprise friction, version coupling to plugin releases, and 2000+ lines of launcher/test code — made the "zero-setup install" framing a net loss. The official installer at https://docs.archcore.ai/cli/install/ is the supported path; one-time user install replaces the bundled-launcher complexity.

### 5. Ship `mcp.json` at the plugin root with `--project ${workspaceFolder}` in args

This is the canonical Cursor 2.5+ way to register a plugin MCP. Rejected because Cursor's `${workspaceFolder}` interpolation inside plugin-MCP `args` is undocumented and the open feature request ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)) implies plugin-MCPs do not get the interpolation that user-config MCPs do. The same shape was reconsidered for Copilot and rejected harder: #4234 reports that the child gets no project path at all, so there is nothing to interpolate.

### 6. Keep the Claude MCP config on the auto-discovered filename

Held implicitly until 2026-08-03, on the belief that Claude Code could reach a plugin's MCP config only by convention. Measurement retired it: the manifest accepts a path string, an array, or an inline object, and a real marketplace install with the key and no `.mcp.json` reports the server connected. Keeping the discovered name cost one host outright and left a latent plugin-MCP on a second. The transferable lesson is narrow and worth stating: *a constraint attributed to the primary host was never measured on the primary host.*

## Consequences

### Positive

- **Zero skill/agent duplication**: skills and agents maintained in one place; per-host format variants are copies held byte-identical by tests, not forks.
- **Low per-host cost**: adding a new host requires only a manifest (~10-15 lines) and a hooks config (~30-40 lines), plus an MCP config where the host can run one. Codex was the first real test (~1 dev-day for scaffolding plus tests); Copilot confirmed the estimate.
- **Decoupled CLI lifecycle**: the Archcore CLI ships and patches on its own cadence; plugin releases never gate CLI security fixes (and vice versa). `archcore update` is the user-facing upgrade path.
- **Standard compliance**: uses Agent Skills, MCP, and markdown agents — all open standards.
- **Single source of truth**: bug fixes in skills/agents/bin propagate to all hosts automatically; the plugin lives in exactly one place (`plugins/archcore/`), with no per-host copy or symlink to keep in sync.
- **MCP reach is explicit per host**: no host can pick up a config we did not point at it, because no config sits on a name any host looks for.

### Negative

- **CLI install is the user's responsibility**: an unsupported user expectation (e.g., "the plugin should just work") surfaces when `archcore` is missing from PATH. Mitigation: `bin/session-start` prints the install command and a docs link on every fresh session where the CLI is absent; `plugin-development.guide` documents the MCP session-start lifecycle gotcha (installing the CLI mid-session does not reconnect a Claude Code MCP that failed to register at session start — restart required).
- **Claude Code's MCP now rests on a single manifest key**: the conventional filename used to be an independent second route and is gone. Delete the key and the primary host loses every document tool, silently. Pinned by `test/structure/plugin-mcp-isolation.bats` with mutation coverage.
- **Stdin normalization complexity**: hook scripts must handle multiple JSON formats. Mitigated by the centralized normalizer.
- **Hook event mapping is imperfect**: not all hosts have equivalent hook events or equivalent semantics. Cursor has no direct `SessionStart` equivalent and its PreToolUse matcher is `Write` only, not `Write|Edit`; Copilot's `postToolUse` carries no matcher at all (the scripts self-filter there), its `preToolUse` timeout fails **open**, and its `preToolUse` accepts no context field at all — so the context-injection guard is not registered there. Mitigation: use the closest available event per host, declare the gaps by name in the coverage tests, and keep the PreToolUse guards far inside their budget (`test/unit/hook-latency.bats`).
- **On two hosts the plugin is not self-contained**: Cursor and Copilot users get MCP only after host wiring runs (`archcore init --agent <host>`). Deliberate — see the two MCP ADRs — and the reason `host-wiring-parity.adr` treats wiring as part of init rather than an extra.
- **Subagent format divergence**: Claude Code and Cursor read MD agents with YAML frontmatter; Codex requires TOML; Copilot requires the `*.agent.md` extension and therefore its own directory, since `.agent.md` would otherwise be picked up as a duplicate by the MD-globbing hosts. Mitigated by shipping the variants side by side, with `test/structure/agents.bats` enforcing parity between MD and TOML bodies and byte-identity between MD and `*.agent.md`.
