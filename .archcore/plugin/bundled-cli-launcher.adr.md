---
title: "Bundled CLI Launcher with Auto-Install and Plugin-Owned MCP"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

The plugin previously required users to install the Archcore CLI out-of-band (via `curl | bash`, `go install`, or package managers) and register the MCP server themselves — either per-user (`claude mcp add archcore archcore mcp -s user`) or per-repo (`.mcp.json` at the project root). This was captured in the Multi-Host Plugin Architecture ADR under the "MCP ownership boundary" section, which justified the choice on the grounds of avoiding Claude Code's duplicate-MCP suppression (v2.1.71+).

In practice this produced real friction:

- First-run onboarding required three separate, correctly-sequenced steps (install CLI → register MCP → reload plugin). Users routinely stopped after `/plugin install`.
- Install scripts (`curl | bash`) are a non-starter in many enterprise environments.
- The `claude mcp add ...` step is discoverable only by reading the README — `/plugin install` gives no hint that MCP registration is still required.
- Error messages from `bin/session-start` when MCP was unreachable ("install the CLI and register the MCP server") were ignored or misread as install failures.

The Claude Code plugin runtime now supports `${CLAUDE_PLUGIN_ROOT}` substitution in `.mcp.json` shipped at the plugin root, and treats plugin-provided MCP servers as first-class. Duplicate suppression only kicks in when the `command`/`args` exactly match a user- or project-registered server — and if a user has installed `archcore` globally, the PATH resolution inside the launcher picks it up, so the effective command is identical to the user's global registration and deduping is benign.

### Drivers

- Zero-setup install is the single largest adoption lever for the plugin.
- The Go CLI ships single-file binaries per platform via GitHub Releases, making platform-targeted auto-download tractable.
- Enterprise/offline environments can still pin their own binary via `ARCHCORE_BIN` or `ARCHCORE_SKIP_DOWNLOAD=1`.

## Decision

**The plugin bundles a shell/PowerShell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`) that resolves the Archcore CLI on demand, and ships `.mcp.json` at the plugin root pointing MCP registration at that launcher.**

### Resolution order (both POSIX and Windows launchers)

1. `$ARCHCORE_BIN` — explicit path to a binary (enterprise pin / local development).
2. `archcore` on `PATH` — respects an existing global install.
3. Plugin-managed cache: `<cache>/archcore-v${VERSION}` where `<cache>` is `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli` (Windows: `$env:LOCALAPPDATA\archcore-plugin\cli`).
4. Download from `github.com/archcore-ai/cli/releases/download/v${VERSION}/archcore_<os>_<arch>.{tar.gz,zip}`, verify against `checksums.txt` (SHA-256), atomically install into the cache, then `exec`.

`ARCHCORE_SKIP_DOWNLOAD=1` disables step 4 and exits 1 instead — used by `bin/session-start` to keep SessionStart non-blocking on first run. The first MCP tool call triggers the download instead.

### MCP registration

The plugin root ships `.mcp.json`:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

Claude Code reads this and registers `archcore` as a plugin-provided MCP server. The command points at the launcher — the launcher resolves to the right binary at invocation time.

### Pinned CLI version

`bin/CLI_VERSION` is a single-line file containing the semver of the CLI release the plugin is tested against (currently `0.1.7`). The launcher reads this and uses it for cache keying and the download URL. Bumping the plugin's CLI pin is a one-file change.

### Checksum verification

Downloads are verified against the release's `checksums.txt` using `sha256sum` or `shasum -a 256` (POSIX) / `Get-FileHash -Algorithm SHA256` (Windows). Mismatches abort the install. No fallback: if checksums can't be computed, the launcher refuses to run the binary.

### Windows-specific handling

`bin/archcore.ps1` strips the Mark-of-the-Web ADS via `Unblock-File` after staging, so Windows SmartScreen does not prompt on first execution. Architecture detection uses `RuntimeInformation.OSArchitecture` (not process architecture) so x64 PowerShell running under ARM64 Prism emulation still installs the correct ARM64 binary.

## Alternatives Considered

### 1. Keep external-only CLI install (status quo)

Continue requiring users to install the CLI separately and register MCP themselves.

**Rejected because:** The friction is load-bearing in a way that blocks adoption. Every reported "plugin doesn't work" issue traced back to incomplete CLI/MCP setup. The multi-host ADR's original rationale (avoiding duplicate-suppression) is solved differently — by the launcher deferring to `PATH` when a global install exists.

### 2. Ship the CLI binary directly in the plugin repo

Vendor per-platform binaries under `bin/vendor/` and pick one at hook invocation time.

**Rejected because:**
- Inflates the plugin repo to ~60MB (four platform binaries).
- Marketplace distribution becomes version-coupled to the CLI — every CLI release forces a plugin release.
- License/provenance surface area grows (signed binaries in a plugin repo raise supply-chain review flags).

### 3. Post-install script via the marketplace

Run a `postInstall` hook to download and install the CLI when the plugin is first installed.

**Rejected because:**
- Claude Code and Cursor plugin runtimes differ in lifecycle-hook support (Cursor has none equivalent).
- First-run-at-install is the wrong place to fail; first-run-at-use lets the user see the one-time download as progress feedback.

### 4. Keep MCP out of the plugin, only ship the launcher

Add `bin/archcore` but leave MCP registration to the user.

**Rejected because:** It solves only half the friction. The whole point is eliminating the `claude mcp add` step.

## Consequences

### Positive

- **Zero-setup install.** `/plugin install archcore` is the only required user action. First MCP call triggers a one-time ~5s download.
- **Respects existing installs.** Users who already have `archcore` on `PATH` (via Homebrew, `go install`, enterprise package) hit that binary — no conflict, no duplicate cache, no surprise.
- **Enterprise/offline escape hatches.** `ARCHCORE_BIN` pins an explicit binary. `ARCHCORE_SKIP_DOWNLOAD=1` disables network access at the launcher layer.
- **Survives plugin updates.** The cache lives under `$CLAUDE_PLUGIN_DATA/archcore/cli` (Claude Code's stable data dir), so plugin re-installs don't re-download.
- **Security.** Downloads are checksum-verified before execution. No `curl | sh`.

### Negative

- **Plugin now owns part of the CLI lifecycle.** Cache invalidation (stale cached binaries after CLI bugfix releases) requires bumping `bin/CLI_VERSION` in the plugin and shipping a plugin release. Mitigation: cache is version-keyed by filename (`archcore-v${VERSION}`), so a pin bump always downloads fresh.
- **First-run network dependency.** Air-gapped environments that don't pre-install the CLI fail at the first MCP call with a network error. Mitigation: documented `ARCHCORE_BIN` / `ARCHCORE_SKIP_DOWNLOAD=1` workflow in the README.
- **Multi-host divergence risk.** Cursor does not support `${CLAUDE_PLUGIN_ROOT}` substitution in plugin-provided MCP configs the same way. Current behavior: Cursor users still register MCP externally (via project `mcp.json` or Cursor MCP settings). The launcher still works for them — it just isn't wired in via a plugin-shipped `.mcp.json`. This is a deliberate host-by-host rollout; the ADR does not claim parity across hosts.
- **Inverts the Multi-Host Plugin Architecture ADR's "MCP ownership boundary" section.** That section is now historically accurate (rationale at the time) but no longer describes current behavior. See that ADR for the cross-link; this ADR supersedes the "plugin does not ship an MCP server configuration" claim for Claude Code specifically.
- **Supply-chain surface area.** The launcher executes downloaded binaries. Checksum verification is the only gate. Any compromise of the GitHub Releases signing pipeline compromises the plugin's trust model. Acceptable given the CLI was the trust root already; the launcher doesn't introduce a new trust boundary.
