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

Remove the bundled shell/PowerShell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`) entirely. Require users to install the Archcore CLI globally (via `brew install archcore-ai/cli`, `go install github.com/archcore-ai/cli@latest`, or equivalent package managers). Have plugins simply `exec archcore` from PATH.

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

**Cost to fix:** One-time 30-second user install (`brew install archcore-ai/cli`) per developer or CI base image.

**Benefits:** All eight bug classes solved. Plugin shrinks by 90%. CLI updates decouple from plugin releases.

## Possible Implementation

1. Delete `bin/archcore`, `bin/archcore.{cmd,ps1}`, `bin/CLI_VERSION`.
2. Update `.mcp.json` → `"command": "archcore"` (resolve via PATH instead of launcher).
3. Update `.codex.mcp.json` → `"command": "archcore"` (remove cwd rebase + env_vars).
4. Update `cursor.mcp.json` → remove `env.ARCHCORE_CWD` (keep `cwd: "${workspaceFolder}"` for good practice).
5. Simplify `bin/session-start` and `bin/validate-archcore` → exec `archcore` directly (no launcher wrapper).
6. Remove launcher-related tests: `test/unit/launcher.bats`, `test/structure/cli-contract.bats`, `test/structure/cli-allowlist-consistency.bats`.
7. Update `test/structure/scripts.bats` and `test/integration/codex-plugin-smoke.bats` to remove launcher assertions.
8. Add CLI availability check to bootstrap skill — attempt `brew install` or `go install` if archcore not found.
9. Update README: add Prerequisites section (install CLI first), remove "Offline / BYO CLI" section + launcher cache descriptions.

## Risks & Constraints

- **Risk 1:** Users forget to install CLI → MCP calls fail with "command not found." Mitigation: clear error messages, SessionStart nudge if CLI missing, bootstrap skill attempts auto-install.
- **Risk 2:** User has old CLI version on PATH → API mismatches. Mitigation: fail-fast version check in `session-start`, document upgrade instructions.
- **Risk 3:** Enterprise can't control CLI version. Mitigation: document standard patterns (Docker base image, Artifactory mirror, package manager tap).
- **Constraint:** CWD guards (Step 0b/0c) were launcher-specific. Without launcher, simpler CWD model: rely on host's `cwd` field + optional `$ARCHCORE_CWD` env (no longer needed without launcher complexity).
