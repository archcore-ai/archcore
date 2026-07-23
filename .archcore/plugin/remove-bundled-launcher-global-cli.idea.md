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

Remove the bundled shell/PowerShell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`) entirely. Require users to install the Archcore CLI globally via the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` on macOS/Linux/WSL, or `irm https://archcore.ai/install.ps1 | iex` on Windows PowerShell 5.1+. Have plugins simply `exec archcore` from PATH.

## Value

The bundled launcher causes eight categories of bugs that are expensive to maintain:

1. **Offline failures in CI/CD** — launcher download fails in air-gapped environments. Workaround: `ARCHCORE_SKIP_DOWNLOAD=1` + manual `ARCHCORE_BIN` pin (undocumented, fragile).
2. **Security patch lag** — CLI bugfix requires plugin release cycle (2–4 weeks). Workaround: users stuck until plugin bumps `CLI_VERSION`.
3. **Uneven host support** — Claude Code and Codex get bundled launcher; Cursor users still do manual MCP setup. Inconsistent UX.
4. **Cache pollution** — same binary cached 3+ times (Claude, Cursor, Codex data dirs) at 5 MB each.
5. **First-run latency** — MCP calls block 5–10 sec on download if cache miss.
6. **Enterprise friction** — no documented way to pre-install in Docker/Artifactory. Users hack `ARCHCORE_BIN`.
7. **Version coupling** — plugin pins CLI version; users can't patch without plugin update.
8. **Plugin bloat** — 200+ lines of launcher code per platform + cache logic inflates plugin size and test surface.

**Cost to fix:** One-time ~30-second user install via the official installer per developer or CI base image. The installer is the supported, documented entry point at https://docs.archcore.ai/cli/install/ — it auto-detects platform/arch, downloads the binary from GitHub Releases, verifies the checksum, and places it on PATH. Subsequent updates use `archcore update`.

**Benefits:** All eight bug classes solved. Plugin shrinks by 90%. CLI updates decouple from plugin releases.

## Possible Implementation

1. Delete `bin/archcore`, `bin/archcore.{cmd,ps1}`, `bin/CLI_VERSION`.
2. Update `.mcp.json` → `"command": "archcore"` (resolve via PATH instead of launcher).
3. Update `.codex.mcp.json` → `"command": "archcore"` (remove cwd rebase + env_vars).
4. Update `cursor.mcp.json` → remove `env.ARCHCORE_CWD` (keep `cwd: "${workspaceFolder}"` for good practice).
5. Simplify `bin/session-start` and `bin/validate-archcore` → exec `archcore` directly (no launcher wrapper).
6. Remove launcher-related tests: `test/unit/launcher.bats`, `test/structure/cli-contract.bats`, `test/structure/cli-allowlist-consistency.bats`.
7. Update `test/structure/scripts.bats` and `test/integration/codex-plugin-smoke.bats` to remove launcher assertions.
8. Add CLI availability check to bootstrap skill — prompt the user once to run the official installer (`curl -fsSL https://archcore.ai/install.sh | bash`) if `archcore` is not found. Do NOT recommend other install paths (`brew`, `go install`, package-manager wrappers) — they are not the supported channel and risk version-incompatible binaries.
9. Update README: add Prerequisites section pointing at https://docs.archcore.ai/cli/install/, remove "Offline / BYO CLI" section + launcher cache descriptions.

## Risks & Constraints

- **Risk 1:** Users forget to install CLI → MCP calls fail with "command not found." Mitigation: clear error messages, SessionStart nudge if CLI missing, bootstrap skill prompts the official installer.
- **Risk 2:** User has old CLI version on PATH → API mismatches. Mitigation: fail-fast version check in `session-start`, document `archcore update` as the upgrade path.
- **Risk 3:** Enterprise can't control CLI version. Mitigation: document standard patterns (Docker base image with pre-installed CLI, internal mirror of `archcore.ai/install.sh`, vendored binary on PATH).
- **Risk 4:** MCP session-start lifecycle gotcha — installing the CLI mid-session does not reconnect a Claude Code MCP server that failed to register at session start. Mitigation: documented in `bin/session-start`'s install message and in `plugin-development.guide.md` ("MCP server not connecting" troubleshooting).
- **Constraint:** CWD guards (Step 0b/0c) were launcher-specific. Without launcher, simpler CWD model: rely on host's `cwd` field + optional `$ARCHCORE_CWD` env (no longer needed without launcher complexity).

## Outcome (2026-05-12)

Shipped in plugin v0.4.0 (commit `2f99997`). All eight bug classes resolved. Plugin source shrank by ~2300 lines (deleted launcher scripts + tests + version pin). The three superseded design docs (`bundled-cli-launcher.adr`, `codex-mcp-cwd-rebase-to-user-project.idea`, `codex-path-resolution.adr`, `cwd-guard-for-cursor-and-claude.idea`) are marked rejected. See commits `2f99997`, `682d079`, `c0d6019` for the rollback set.
