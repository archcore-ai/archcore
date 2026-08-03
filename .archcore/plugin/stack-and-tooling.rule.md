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

1. Every executable script under `plugins/archcore/bin/` and `plugins/archcore/bin/lib/` MUST start with `#!/bin/sh`.
2. WHEN `shellcheck` is available, every executable script under `plugins/archcore/bin/` and `plugins/archcore/bin/lib/` MUST pass it.
3. The author MUST NOT introduce a `#!/usr/bin/env python3`, `#!/usr/bin/env node`, or `#!/usr/bin/env bash` shebang for executable plugin code.
4. Exception to items 1–3 (per `opencode-adapter-packaging.adr`, accepted 2026-07-05): the OpenCode host adapter under `plugins/opencode/` MAY be TypeScript executed by OpenCode's Bun runtime.
5. An OpenCode adapter hook MUST shell out to a `plugins/archcore/bin/` script for every guard decision and every validation decision.
6. An OpenCode adapter hook MUST NOT implement guard or validation decision logic in TypeScript.
7. The author MUST NOT convert a file under `plugins/archcore/` to TypeScript under the item 4 exception.
8. The author MUST NOT convert repo-root tooling to TypeScript under the item 4 exception.
9. The plugin repository MUST NOT contain a `go.mod` file.
10. The plugin repository MUST NOT contain a `.go` file.
11. The plugin repository MUST NOT contain a bundled CLI binary.
12. The plugin repository MUST NOT contain a launcher wrapper for the CLI.
13. The plugin repository MUST NOT contain code that downloads or caches the CLI on first use.
14. The author MUST express declarative plugin state only in these shapes: host manifests (`.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`), MCP configs (`.mcp.json`, `.codex.mcp.json`, `docs/cursor.mcp.example.json`), hook configs (`hooks/*.json`), marketplace entries (`.agents/plugins/marketplace.json`), skills (`skills/*/SKILL.md`), and agent definitions (`agents/*.md`, `agents/*.toml`).
15. The author MUST NOT add a runtime YAML config file.
16. The author MUST NOT add a programmatic config generator.
17. Exception to items 14–16 (same ADR): the OpenCode adapter MAY register its configuration programmatically inside `plugins/opencode/`, because the OpenCode runtime exposes no declarative hook config.
18. The author MUST add every new test as a `.bats` file under `test/structure/`, `test/unit/`, or `test/integration/`.
19. The author MUST use the `bats-support` and `bats-assert` helpers vendored under `test/helpers/`.
20. The author MUST NOT add a second test runner to this repository.
21. Exception to items 18–20 (same ADR): the author MUST test TypeScript sources under `plugins/opencode/` with `bun test`.
22. The author MUST place every `bun test` file inside `plugins/opencode/`.
23. The author MUST perform every `.archcore/` document operation through an MCP tool, as `mcp-only-operations.rule` requires.
24. BEFORE a change introduces a new language, a new runtime, a new build tool, a new package manager, or a new distribution mechanism, the author MUST record an ADR under `.archcore/plugin/<slug>.adr.md`.
25. BEFORE such a change lands, the maintainer MUST accept the ADR recorded under item 24.
26. IF a problem appears unsolvable inside the current stack, THEN the author MUST propose an ADR instead of expanding the stack.
27. WHEN a pull request adds an executable file with a shebang other than `#!/bin/sh` outside `plugins/opencode/`, the pull request MUST link to an accepted ADR under `.archcore/plugin/`.
28. WHEN a pull request adds a new manifest format, a new test runner, or a new top-level config file, the pull request MUST link to an accepted ADR under `.archcore/plugin/`.
29. A skill MUST NOT instruct the agent to invoke non-shell tooling inside a plugin script.
30. A structure test SHOULD assert that TypeScript exists only under `plugins/opencode/`.

### Notes (non-normative)

- Item 24 covers Python, Node.js, Ruby, Rust, an additional Go module, a compiled trampoline binary, a Make layer, a CMake layer, a Bazel layer, a container runtime requirement, and a plugin-side download-on-first-use mechanism.
- The archcore CLI is written in Go. It lives in the separate repository `archcore-ai/cli` and is consumed as a globally installed binary on PATH. Users install it from https://docs.archcore.ai/cli/install/ with `curl -fsSL https://archcore.ai/install.sh | bash` on POSIX or `irm https://archcore.ai/install.ps1 | iex` on Windows. Items 9–13 keep that lifecycle out of this repository.
- The `Makefile` variable `BIN_SCRIPTS` is the authoritative list of executable files in `bin/`. It must not be extended to cover a binary.

## Rationale

The plugin's value is portability: one repository loads into Claude Code, Codex, Cursor, and Copilot CLI, on macOS, Linux, and Windows-with-WSL, with no per-host install step beyond cloning. Every added runtime is a new failure mode for every user on every platform — `python3` missing from PATH, a `node` version mismatch, Go cross-compile signing on macOS, or a Codex MCP failure caused by an unnotarized shipped binary.

Three concrete lessons produced these items:

- **Codex MCP cwd.** The direct fix was a Python trampoline that reads `$PWD` and calls `chdir`. It worked end to end and added a third language to a two-language plugin. It was replaced with an opt-in `ARCHCORE_CWD` environment variable honored by the bundled launcher — two lines of shell plus a user-side wrapper. See `codex-mcp-cwd-rebase-to-user-project.idea` (rejected) and `codex-path-resolution.adr` (rejected). The Python implementation stays in git history as a record of what did not ship. Both decisions were superseded when the launcher was removed.
- **Bundled launcher.** A download-on-first-use shell script fetched a single Go binary from `archcore-ai/cli` releases. It shipped, then was removed in plugin v0.4.0 after producing eight categories of bugs, among them offline failures, version coupling, cache pollution, and security-patch lag. The saving was a one-time install step that the official installer already handles without coupling the CLI lifecycle to plugin releases. See `bundled-cli-launcher.adr` (rejected and superseded) and `remove-bundled-launcher-global-cli.idea` (accepted). The plugin now assumes `archcore` is on PATH; when it is not, `bin/session-start` prints the install command and exits.
- **OpenCode adapter, 2026-07-05.** OpenCode exposes hooks only through a Bun-executed JS/TS plugin API, so no declarative hook config exists to reuse. The exception followed exactly the path items 24 and 25 prescribe: research document, maintainer decision, accepted ADR (`opencode-adapter-packaging.adr`), and a directory-scoped carve-out — rather than TypeScript spreading through the repository.

Items 24 and 25 exist to make those decisions hold. When future work meets a limit of POSIX shell, the reflex is to add Python at that spot; the required response is an ADR and a decision first.

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

# OpenCode adapter hook — thin bridge inside plugins/opencode/, per items 4-6
# plugins/opencode/src/hooks.ts
"tool.execute.before": async (input, output) => {
  const res = await runBinScript("check-archcore-write", payload(input, output))
  if (res.blocked) throw new Error(res.reason)   // translation only — no decision logic here
}

# Stack change: ADR accepted before any code change, per items 24-25
# .archcore/plugin/<slug>.adr.md
```

### Bad

```text
# bin/check-something.py    ← Python script in bin/, no ADR — violates items 3 and 24
#!/usr/bin/env python3
import json, sys
...

# bin/launcher.go            ← Go source in plugin repo — violates item 10
package main
func main() { ... }

# test/check-something.test.ts  ← TypeScript test outside plugins/opencode/ — violates item 22
import { describe, it } from "vitest"
...

# plugins/opencode/src/guard.ts  ← guard logic in TS instead of shelling out — violates items 5 and 6
if (filePath.startsWith(".archcore/") && !isMcpWrite(tool)) { deny() }

# .codex.mcp.json adds a non-shell entry point silently — violates items 3 and 24
{
  "command": "./bin/archcore-mcp",   ← invokes Python without an ADR
  "args": []
}

# bin/auto-install-cli       ← reintroduces a plugin-side CLI fetcher — violates item 13
curl -fsSL https://... -o /tmp/archcore && /tmp/archcore "$@"
```

## Enforcement

- Code review: a pull request that trips item 27 or item 28 and carries no ADR link blocks merge.
- `@test/structure/scripts.bats` asserts that every file in `bin/` starts with `#!/bin/sh`, which verifies item 1.
- Structure tests pin the remaining file-shape contracts: no `.py`, `.go`, `.js`, `.ts`, or `.rb` file exists under `bin/` or at the repo root outside `reference-materials/`, `test_project/`, and `plugins/opencode/`.
- `plugin-development.guide` states items 24 and 25 in its onboarding section for new contributors.
- The path for a genuinely new tool is: open an issue, draft an ADR with the sections `Context / Decision / Alternatives Considered / Consequences`, obtain review, obtain acceptance, then implement.
