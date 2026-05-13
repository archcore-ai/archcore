---
title: "Plugin Stack and Tooling — No New Languages Without ADR"
status: accepted
tags:
  - "architecture"
  - "development"
  - "plugin"
  - "rule"
---

## Rule

1. **Executable code in the plugin repository MUST be POSIX shell.** All scripts under `bin/` and `bin/lib/` (hooks, validators, helpers) are `#!/bin/sh` and pass `shellcheck` when available. Do not introduce `#!/usr/bin/env python3`, `#!/usr/bin/env node`, `#!/usr/bin/env bash` shebangs, or other runtimes for executable plugin code.
2. **The archcore CLI is Go**, but it lives in a separate repo (`archcore-ai/cli`) and is consumed as a globally-installed binary on PATH. Users install it via the official installer at https://docs.archcore.ai/cli/install/ (`curl -fsSL https://archcore.ai/install.sh | bash` on POSIX, `irm https://archcore.ai/install.ps1 | iex` on Windows). The plugin repository contains **no Go source**, no bundled binary, no launcher wrapper, and no auto-download path; do not add a `go.mod`, any `.go` files, or any code that fetches/caches the CLI on first use.
3. **Plugin-facing configuration MUST be declarative JSON/TOML/Markdown.** Host manifests (`.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`), MCP configs (`.mcp.json`, `.codex.mcp.json`, and the user-facing template `docs/cursor.mcp.example.json`), hook configs (`hooks/*.json`), marketplace entries (`.agents/plugins/marketplace.json`), skills (`skills/*/SKILL.md`), and agent definitions (`agents/*.md`, `agents/*.toml`) are the only allowed shapes for declarative state. No YAML at runtime, no programmatic config generators.
4. **Tests MUST be Bats.** Add new tests under `test/structure/`, `test/unit/`, or `test/integration/` as `.bats` files. Use `bats-support` and `bats-assert` helpers already vendored under `test/helpers/`. Do not add a separate test runner (no jest, pytest, go test, hurl, etc.) inside this repo.
5. **Document operations on `.archcore/` MUST go through MCP tools** — see `mcp-only-operations.rule`. The stack rule does not relax that requirement; it strengthens it (no direct file writes from new tooling either).
6. **IMPORTANT — no new languages, runtimes, build tools, package managers, or distribution mechanisms without a written ADR.** Before any change introduces Python, Node.js, Ruby, Rust, an additional Go module, a compiled trampoline binary, a Make/CMake/Bazel layer, a container runtime requirement, a plugin-side download-on-first-use mechanism, or any other novel tooling, an ADR MUST be recorded in `.archcore/plugin/<slug>.adr.md` and accepted by the maintainer. "Just adding one Python script" is exactly the kind of incremental drift this rule blocks. If a problem looks unsolvable within the current stack, the answer is an ADR proposal, not silent expansion.

## Rationale

The plugin's value proposition is **portability**: one repo loads cleanly into Claude Code, Codex, and Cursor, on macOS / Linux / Windows-with-WSL, with no per-host install steps beyond cloning. Every additional runtime is a new failure mode for every user on every platform — `python3 not in PATH`, `node version mismatch`, `Go cross-compile signing on macOS`, "your Codex MCP fails because we shipped a binary that wasn't notarized".

Concrete past lessons:

- **Codex MCP cwd**: the obvious fix was a Python trampoline that reads `$PWD` and `chdir`s. Worked end-to-end, but added a third language to a two-language plugin. We rejected it in favor of an opt-in `ARCHCORE_CWD` env var that the bundled launcher honored — two lines of shell + a user-side wrapper. See `codex-mcp-cwd-rebase-to-user-project.idea` (rejected) and `codex-path-resolution.adr` (rejected) for the full historical debate. The Python implementation is preserved in git history as a reminder of what we did NOT ship. Both decisions were then superseded entirely when the launcher was removed (see next bullet).
- **Bundled launcher (rejected and removed)**: a download-on-first-use sh script that fetched a single Go binary from `archcore-ai/cli` releases. We shipped it, then removed it as of plugin v0.4.0 — it caused eight categories of bugs (offline failures, version coupling, cache pollution, security patch lag, etc.) for a one-time install savings that the official installer (`curl | bash`) handles cleanly without coupling CLI lifecycle to plugin releases. See `bundled-cli-launcher.adr` (rejected/superseded) and `remove-bundled-launcher-global-cli.idea` (accepted) for the decision and rollback rationale. The plugin now assumes `archcore` is on PATH; if not, `bin/session-start` prints the install command and exits.

This rule exists to make those decisions stick. When future work runs into an awkward limitation of POSIX shell, the impulse is "let me just add Python here". This rule says: stop, write an ADR, get a decision, then proceed.

## Examples

### Good

```text
# A new hook validator script
# bin/check-something — POSIX sh, sources lib/normalize-stdin.sh, passes shellcheck
#!/bin/sh
set -eu
. "$(dirname "$0")/lib/normalize-stdin.sh"
# ... shell logic; if it calls the CLI, it calls `archcore <subcmd>` directly via PATH ...

# A new test
# test/unit/check-something.bats
@test "check-something rejects malformed input" {
  run check_something < /dev/null
  assert_failure
}

# Stack change: discussed first, then recorded
# .archcore/plugin/use-cobra-v2.adr.md  — accepted before any code change
```

### Bad

```text
# bin/check-something.py    ← Python script in bin/, no ADR
#!/usr/bin/env python3
import json, sys
...

# bin/launcher.go            ← Go source in plugin repo
package main
func main() { ... }

# test/check-something.test.ts  ← TypeScript test inside the plugin
import { describe, it } from "vitest"
...

# .codex.mcp.json adds a non-shell entry point silently
{
  "command": "./bin/archcore-mcp",   ← invokes Python without an ADR
  "args": []
}

# bin/auto-install-cli       ← reintroduces a plugin-side CLI fetcher, no ADR
curl -fsSL https://... -o /tmp/archcore && /tmp/archcore "$@"
```

## Enforcement

- Code review: any PR that adds an executable file with a non-`#!/bin/sh` shebang, a new manifest format, a new test runner, or a new top-level config file MUST link to an accepted ADR under `.archcore/plugin/`. PRs without such a link block merge.
- Structure tests: `test/structure/*.bats` should pin the contract. Add an assertion that `bin/*` files start with `#!/bin/sh` (already present per `test/structure/scripts.bats`) and that no `.py`, `.go`, `.js`, `.ts`, `.rb`, etc. files exist under `bin/` or the repo root outside `reference-materials/` and `test_project/`. The Makefile's `BIN_SCRIPTS` glob is the authoritative list of executable files in `bin/`; do not extend it to cover a binary.
- Skill writing: skills MUST NOT instruct the agent to invoke non-shell tooling in plugin scripts. If a skill needs Python/node/etc., the dependency lives in the user's project, not in the plugin.
- New contributors: the `plugin-development.guide` calls out this rule prominently in its onboarding section.
- When a contributor genuinely needs a new tool, the path is: open an issue, draft an ADR following the template (`Context / Decision / Alternatives Considered / Consequences`), get review, get acceptance, only then implement. Default answer to "can we add X" is "write the ADR first".
