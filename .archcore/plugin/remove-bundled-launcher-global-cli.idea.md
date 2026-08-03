---
title: "Remove Bundled Launcher — Assume Global CLI on PATH"
status: accepted
tags:
  - "architecture"
  - "cli"
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

1. Delete `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, and `bin/CLI_VERSION`.
2. Set the Claude Code MCP config to `"command": "archcore"`, resolving through PATH.
3. Set `.codex.mcp.json` to `"command": "archcore"`, removing the `cwd` rebase and the `env_vars` allowlist.
4. Remove `env.ARCHCORE_CWD` from the Cursor template. (Recorded at the time as "keep `cwd: "${workspaceFolder}"`". That was wrong: Cursor's MCP stdio schema has no `cwd` field. The shipped template is `docs/cursor.mcp.example.json`, which carries `--project ${workspaceFolder}` in `args` — see `cursor-mcp-architecture.adr`.)
5. Simplify `bin/session-start` and `bin/validate-archcore` to exec `archcore` directly.
6. Remove the launcher-related tests `test/unit/launcher.bats`, `test/structure/cli-contract.bats`, and `test/structure/cli-allowlist-consistency.bats`.
7. Remove the launcher assertions from `test/structure/scripts.bats` and `test/integration/codex-plugin-smoke.bats`.
8. Add a CLI availability check to the onboarding skill that prompts the user once to run the official installer when `archcore` is absent, and recommends no other channel, because a package-manager wrapper risks a version-incompatible binary.
9. Update the README with a prerequisites section pointing at the install docs, removing the offline and BYO-CLI section and the launcher cache descriptions.

## Risks

- **A user forgets to install the CLI**, so MCP calls fail with `command not found`. Mitigated by clear error text, a SessionStart nudge when the CLI is missing, and the onboarding prompt naming the official installer.
- **A user has an old CLI on PATH**, producing API mismatches. Mitigated by a fail-fast version check in `session-start` and by documenting `archcore update` as the upgrade path.
- **An enterprise cannot control the CLI version.** Mitigated by documenting the standard patterns: a Docker base image with the CLI pre-installed, an internal mirror of the install script, or a vendored binary on PATH.
- **The session-start lifecycle gotcha**, where installing the CLI mid-session does not reconnect a Claude Code MCP server that failed to register at session start. Mitigated in the install message and in `plugin-development.guide` troubleshooting.
- **Constraint:** the CWD guards were launcher-specific. Without a launcher the model is simpler — each host's own working directory, plus `archcore mcp --project <path>` where the host cannot guarantee it.

## Outcome (2026-05-12)

Shipped in plugin v0.4.0, commit `2f99997`. All eight bug classes resolved and the plugin source shrank by roughly 2300 lines across the deleted launcher scripts, tests, and version pin. The superseded design records — `bundled-cli-launcher.adr`, `codex-mcp-cwd-rebase-to-user-project.idea`, `codex-path-resolution.adr`, and `cwd-guard-for-cursor-and-claude.idea` — are marked rejected. Commits `2f99997`, `682d079`, and `c0d6019` hold the rollback set.
