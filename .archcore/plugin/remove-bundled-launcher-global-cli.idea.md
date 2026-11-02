---
title: "Remove Bundled Launcher — Assume Global CLI on PATH"
status: accepted
tags:
  - "architecture"
  - "cli"
  - "component:plugin"
  - "multi-host"
  - "plugin"
---

## Idea

Remove the bundled shell and PowerShell launcher — `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, and `bin/CLI_VERSION` — entirely, require the user to install the Archcore CLI globally from https://docs.archcore.ai/cli/install/, and have every plugin config exec `archcore` from PATH.

## Value

The bundled launcher produced eight categories of bug, each expensive to maintain:

1. **Offline failures in CI/CD**, where the launcher download fails in an air-gapped environment, worked around only by an undocumented `ARCHCORE_SKIP_DOWNLOAD=1` plus a manual `ARCHCORE_BIN` pin.
2. **Security-patch lag**, where a CLI bugfix waits on a plugin release cycle of 2–4 weeks, leaving users stuck until the plugin bumps `CLI_VERSION`.
3. **Uneven host support**, where Claude Code and Codex got the launcher while Cursor users still did manual MCP setup.
4. **Cache pollution**, with the same binary cached three or more times across host data directories at about 5 MB each.
5. **First-run latency**, where an MCP call blocks 5–10 seconds on download after a cache miss.
6. **Enterprise friction**, with no documented way to pre-install in Docker or Artifactory, pushing users to hack `ARCHCORE_BIN`.
7. **Version coupling**, where the plugin pins the CLI version so users cannot patch without a plugin update.
8. **Plugin bloat**, with more than 200 lines of launcher code per platform plus cache logic inflating both plugin size and test surface.

The cost of removing them is one roughly 30-second install per developer or CI base image, through the supported installer that auto-detects platform and architecture, downloads from GitHub Releases, verifies the checksum, and places the binary on PATH, with `archcore update` handling later upgrades. All eight bug classes resolve, the plugin shrinks substantially, and CLI updates decouple from plugin releases.

## Possible Implementation

Delete the launcher files and the version pin, and point every MCP config at `"command": "archcore"` on PATH. Drop the `cwd` rebase and `env_vars` allowlist from the Codex config. For Cursor, the shipped template carries `--project ${workspaceFolder}` in `args`, because Cursor's MCP stdio schema has no `cwd` field (`cursor-mcp-architecture.adr`). The onboarding skill prompts once for the official installer when `archcore` is absent and recommends no other channel, because a package-manager wrapper risks a version-incompatible binary.

## Risks

- **A user forgets to install the CLI**, so MCP calls fail with `command not found`. Mitigated by clear error text, a SessionStart nudge when the CLI is missing, and the onboarding prompt naming the official installer.
- **A user has an old CLI on PATH**, producing API mismatches. Mitigated by a fail-fast version check in `session-start` and by documenting `archcore update` as the upgrade path.
- **An enterprise cannot control the CLI version.** Mitigated by documenting the standard patterns: a Docker base image with the CLI pre-installed, an internal mirror of the install script, or a vendored binary on PATH.
- **The session-start lifecycle gotcha**, where installing the CLI mid-session does not reconnect a Claude Code MCP server that failed to register at session start. Mitigated in the install message and in `plugin-development.guide` troubleshooting.
- **Constraint:** the CWD guards were launcher-specific. Without a launcher the model is simpler — each host's own working directory, plus `archcore mcp --project <path>` where the host cannot guarantee it.

## Outcome (2026-05-12)

Shipped in plugin v0.4.0, commit `2f99997`. All eight bug classes resolved and the plugin source shrank by roughly 2300 lines across the deleted launcher scripts, tests, and version pin. The superseded launcher, Codex cwd-rebase, and cwd-guard designs were removed from the corpus; `multi-host-plugin-architecture.adr` holds the surviving host-path decisions. Commits `2f99997`, `682d079`, and `c0d6019` hold the rollback set.
