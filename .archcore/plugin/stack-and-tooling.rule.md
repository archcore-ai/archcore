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

1. **Executable code in the plugin repository MUST be POSIX shell.** All scripts under `bin/` and `bin/lib/` (launcher, hooks, validators, helpers) are `#!/bin/sh` and pass `shellcheck` when available. Do not introduce `#!/usr/bin/env python3`, `#!/usr/bin/env node`, `#!/usr/bin/env bash` shebangs, or other runtimes for executable plugin code.
2. **The archcore CLI is Go**, but it lives in a separate repo (`archcore-ai/cli`) and is consumed as a single binary by the plugin's bundled launcher. The plugin repository itself contains **no Go source**; do not add a `go.mod` or any `.go` files here.
3. **Plugin-facing configuration MUST be declarative JSON/TOML/Markdown.** Host manifests (`.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`), MCP configs (`.mcp.json`, `.codex.mcp.json`), hook configs (`hooks/*.json`), marketplace entries (`.agents/plugins/marketplace.json`), skills (`skills/*/SKILL.md`), and agent definitions (`agents/*.md`, `agents/*.toml`) are the only allowed shapes for declarative state. No YAML at runtime, no programmatic config generators.
4. **Tests MUST be Bats.** Add new tests under `test/structure/`, `test/unit/`, or `test/integration/` as `.bats` files. Use `bats-support` and `bats-assert` helpers already vendored under `test/helpers/`. Do not add a separate test runner (no jest, pytest, go test, hurl, etc.) inside this repo.
5. **Document operations on `.archcore/` MUST go through MCP tools** — see `mcp-only-operations.rule`. The stack rule does not relax that requirement; it strengthens it (no direct file writes from new tooling either).
6. **IMPORTANT — no new languages, runtimes, build tools, package managers, or distribution mechanisms without a written ADR.** Before any change introduces Python, Node.js, Ruby, Rust, an additional Go module, a compiled trampoline binary, a Make/CMake/Bazel layer, a container runtime requirement, a curl-piped install step, or any other novel tooling, an ADR MUST be recorded in `.archcore/plugin/<slug>.adr.md` and accepted by the maintainer. "Just adding one Python script" is exactly the kind of incremental drift this rule blocks. If a problem looks unsolvable within the current stack, the answer is an ADR proposal, not silent expansion.

## Rationale

The plugin's value proposition is **portability**: one repo loads cleanly into Claude Code, Codex, and Cursor, on macOS / Linux / Windows-with-WSL, with no per-host install steps beyond cloning. Every additional runtime is a new failure mode for every user on every platform — `python3 not in PATH`, `node version mismatch`, `Go cross-compile signing on macOS`, "your Codex MCP fails because we shipped a binary that wasn't notarized".

Concrete past lessons:

- **Codex MCP cwd**: the obvious fix was a Python trampoline that reads `$PWD` and `chdir`s. Worked end-to-end, but added a third language to a two-language plugin. We rejected it in favor of an opt-in `ARCHCORE_CWD` env var the existing sh launcher honors — two lines of shell + a user-side wrapper. See `codex-mcp-cwd-rebase-to-user-project.idea` and `codex-path-resolution.adr` for the full debate. The Python implementation is preserved in git history as a reminder of what we did NOT ship.
- **Bundled launcher**: a download-on-first-use sh script that fetches a single Go binary from `archcore-ai/cli` releases. We deliberately did not adopt a package manager (homebrew/apt/winget) or a vendored binary blob in the plugin repo. See `bundled-cli-launcher.adr`.

This rule exists to make those decisions stick. When future work runs into an awkward limitation of POSIX shell, the impulse is "let me just add Python here". This rule says: stop, write an ADR, get a decision, then proceed.

## Examples

### Good

```text
# A new hook validator script
# bin/check-something — POSIX sh, sources lib/normalize-stdin.sh, passes shellcheck
#!/bin/sh
set -eu
. "$(dirname "$0")/lib/normalize-stdin.sh"
# ... shell logic ...

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
```

## Enforcement

- Code review: any PR that adds an executable file with a non-`#!/bin/sh` shebang, a new manifest format, a new test runner, or a new top-level config file MUST link to an accepted ADR under `.archcore/plugin/`. PRs without such a link block merge.
- Structure tests: `test/structure/*.bats` should pin the contract. Add an assertion that `bin/*` files start with `#!/bin/sh` (already present per `test/structure/bin-shape.bats`) and that no `.py`, `.go`, `.js`, `.ts`, `.rb`, etc. files exist under `bin/` or the repo root outside `reference-materials/` and `test_project/`.
- Skill writing: skills MUST NOT instruct the agent to invoke non-shell tooling in plugin scripts. If a skill needs Python/node/etc., the dependency lives in the user's project, not in the plugin.
- New contributors: the `plugin-development.guide` calls out this rule prominently in its onboarding section.
- When a contributor genuinely needs a new tool, the path is: open an issue, draft an ADR following the template (`Context / Decision / Alternatives Considered / Consequences`), get review, get acceptance, only then implement. Default answer to "can we add X" is "write the ADR first".
