---
title: "Codex 0.130.0 Plugin Spawn Semantics — MCP cwd Rebase vs Hooks Variable Substitution"
status: accepted
tags:
  - "architecture"
  - "codex"
  - "multi-host"
  - "plugin"
---

## Context

When adding Codex CLI as a third first-class host, the initial implementation assumed Codex would resolve relative `./bin/...` command paths from the plugin install directory — analogous to Claude Code's `${CLAUDE_PLUGIN_ROOT}` substitution for MCP configs. Earlier documents (`codex-host-support.plan.md`, `multi-host-compatibility-layer.spec.md`, `bundled-cli-launcher.adr.md`, `codex-host-support.idea.md`, `codex-host-support.prd.md`) codified that assumption.

Empirical testing against Codex 0.130.0 and direct inspection of `codex-rs` source revealed the assumption was wrong for MCP and partly wrong for hooks. The two mechanisms are distinct and must not be conflated.

The original symptom: from any user project directory, the plugin-shipped MCP server failed to start with `MCP startup failed: No such file or directory (os error 2)`. The plugin appeared to "work" only when Codex was launched from within the plugin source repo, because that directory accidentally contained `bin/archcore` relative to CWD.

## Decision

Codex 0.130.0 uses **two separate mechanisms** for plugin-shipped component path resolution, and they differ between MCP and hooks. Both must be respected by the plugin's host configs.

### MCP servers — `cwd` rebase

For plugin-shipped MCP servers (`.codex.mcp.json` registered via `.codex-plugin/plugin.json`):

- Codex does **NOT** substitute `${CODEX_PLUGIN_ROOT}`, `${CLAUDE_PLUGIN_ROOT}`, `${PLUGIN_ROOT}`, or any other placeholder in `command`, `args`, or `env`.
- Codex does **NOT** export `CODEX_PLUGIN_ROOT` or any plugin-root env var to the spawned MCP process.
- Codex **DOES** rebase a relative `cwd` field against the plugin install root. Source: `codex-rs/core-plugins/src/loader.rs::normalize_plugin_mcp_server_value`.
- The rebased cwd is passed to the OS as `current_dir` when spawning the MCP process. Source: `codex-rs/rmcp-client/src/stdio_server_launcher.rs::launch_server`.
- Without `cwd` set, Codex spawns the MCP process from the user's project CWD. A relative `command: "./bin/archcore"` then fails with ENOENT (`MCP startup failed: No such file or directory (os error 2)`) from any project that doesn't happen to contain `bin/archcore`.

**Correct form for `.codex.mcp.json`:**

```json
{
  "mcpServers": {
    "archcore": {
      "command": "./bin/archcore",
      "args": ["mcp"],
      "cwd": "."
    }
  }
}
```

Verified empirically: with `cwd: "."` the MCP server starts and answers JSON-RPC from a neutral directory outside the plugin source repo. Without `cwd`, ENOENT.

### Hooks — `${PLUGIN_ROOT}` substitution (canonical, host-neutral)

For plugin-shipped hooks (`hooks/codex.hooks.json`), Codex's hooks engine injects two env vars and applies `${KEY}` token substitution over the command string. Direct quote from `codex-rs/hooks/src/engine/discovery.rs`:

```rust
env.insert("PLUGIN_ROOT".to_string(), plugin_root_value.clone());
// For OOTB compat with existing plugins that use this env var.
env.insert("CLAUDE_PLUGIN_ROOT".to_string(), plugin_root_value);
env.insert("PLUGIN_DATA".to_string(), plugin_data_root_value.clone());
// For OOTB compat with existing plugins that use this env var.
env.insert("CLAUDE_PLUGIN_DATA".to_string(), plugin_data_root_value);
```

```rust
let command = source.env.iter().fold(command, |command, (key, value)| {
    command.replace(&format!("${{{key}}}"), value)
});
```

Two important facts from this source:

1. **`PLUGIN_ROOT` is the canonical, host-neutral name.** It comes first and is the name Codex chose for the plugin runtime.
2. **`CLAUDE_PLUGIN_ROOT` is an "OOTB compat" alias** — the comment in source literally labels it as a backward-compatibility shim for porting old Claude plugins. It is NOT the recommended name for a Codex-native hook config.
3. **`CODEX_PLUGIN_ROOT` does not exist in Codex** — neither as env injection nor as substitution placeholder.

Therefore, the Codex-native, semantically clean form for `hooks/codex.hooks.json` uses `${PLUGIN_ROOT}` — Codex's own canonical name. The plugin file should not borrow Claude Code's name; that would be using a compat shim where a native primitive exists.

**Correct form for `hooks/codex.hooks.json`:**

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "${PLUGIN_ROOT}/bin/session-start" }
        ]
      }
    ]
  }
}
```

Plugin-shipped Codex hooks are gated behind the `plugin_hooks` feature flag, which is `under development, false` in Codex 0.130.0. Users must run `codex features enable plugin_hooks` to opt in.

### Per-host hook variable convention (uniformity)

Each host config uses the host's own canonical name. No host file borrows another host's name:

| Hook config | Canonical env var | Rationale |
|---|---|---|
| `hooks/hooks.json` (Claude Code) | `${CLAUDE_PLUGIN_ROOT}` | Claude Code's native injection |
| `hooks/cursor.hooks.json` (Cursor) | `${CURSOR_PLUGIN_ROOT}` | Cursor's native injection |
| `hooks/codex.hooks.json` (Codex) | `${PLUGIN_ROOT}` | Codex's native canonical (Codex never had a `CODEX_PLUGIN_ROOT`; `PLUGIN_ROOT` is what its source defines as canonical) |

This is uniform: each host config uses its host's canonical name, no cross-host pollution.

## Alternatives Considered

### Use `${CLAUDE_PLUGIN_ROOT}` in `hooks/codex.hooks.json`

Rejected. Codex source explicitly labels `CLAUDE_PLUGIN_ROOT` as an "OOTB compat" shim. Using it in a Codex-native config would borrow Claude's name where Codex has its own canonical. It would also introduce confusing cross-host coupling — readers of `codex.hooks.json` would wonder why a Claude env var appears in a Codex file.

### Use `${CODEX_PLUGIN_ROOT}` in `hooks/codex.hooks.json`

Rejected. Codex does not inject `CODEX_PLUGIN_ROOT` into hook env. The substitution would expand to an empty string, and the resulting command (`/bin/session-start`) would fail.

### Global PATH install of the CLI

Install the Archcore CLI globally on the system PATH rather than relying on the plugin-bundled launcher.

Rejected because: loses the plugin-managed launcher, requires a separate install step, breaks the zero-setup install goal established in `bundled-cli-launcher.adr.md`.

### Upstream PR adding `${PLUGIN_ROOT}` substitution to Codex MCP

Submit a PR to `openai/codex` adding placeholder substitution to the MCP config loader so that `${PLUGIN_ROOT}/bin/archcore` works in MCP entries the same way it works in hooks.

Deferred: filed as <https://github.com/openai/codex/issues/19582> (open). Until this lands, the `cwd` rebase is the correct workaround. The workaround stays compatible with future substitution support — `cwd: "."` remains benign even after `${PLUGIN_ROOT}` substitution becomes available in MCP.

### Shell wrapper that auto-discovers the plugin cache

A shell shim at a stable PATH location that locates the plugin install dir at runtime via `codex plugin locate archcore` or by scanning known cache paths.

Rejected because: requires an extra installed shim outside the plugin bundle, adds complexity, is fragile if Codex cache paths change.

## Consequences

- `.codex.mcp.json` MUST set `cwd: "."`. The relative command `./bin/archcore` is correct and intentional — it is resolved against the rebased cwd, not the user's project dir.
- `hooks/codex.hooks.json` MUST use `${PLUGIN_ROOT}/bin/...` command form. Borrowing `${CLAUDE_PLUGIN_ROOT}` (the compat alias) is forbidden — use Codex's own canonical name.
- Tests in `test/structure/codex-plugin.bats` and `test/structure/hooks.bats` enforce both invariants. They explicitly fail if `codex.hooks.json` references any other host's plugin-root env var.
- Plugin hooks remain gated by Codex's `plugin_hooks` feature flag (currently `false` by default). The plugin ships the hook config unconditionally; runtime activation requires user opt-in.
- Upstream issue <https://github.com/openai/codex/issues/19582> tracks adding `${PLUGIN_ROOT}` substitution to Codex MCP. When that lands, the `cwd` workaround can be replaced with the substitution form — but will remain backward-compatible since `cwd: "."` is benign even with substitution.
- All prior documents that claimed Codex used "plugin-relative `./bin/...` commands" or `${CLAUDE_PLUGIN_ROOT}` for hooks were corrected as part of this finding. This ADR is the canonical reference for the two mechanisms going forward.
- The per-host variable convention (`${CLAUDE_PLUGIN_ROOT}` for Claude, `${CURSOR_PLUGIN_ROOT}` for Cursor, `${PLUGIN_ROOT}` for Codex) is uniform: each host config uses its host's canonical name, never another host's.
