---
title: "Plugin Testing Guide"
status: accepted
tags:
  - "development"
  - "plugin"
  - "testing"
---

## Prerequisites

- [bats-core](https://github.com/bats-core/bats-core) — test runner for shell scripts
  - macOS: `brew install bats-core`
  - Linux: `apt install bats`
- [jq](https://jqlang.github.io/jq/) — JSON validation in structure tests
  - macOS: `brew install jq`
  - Linux: `apt install jq`
- [ShellCheck](https://www.shellcheck.net/) (optional) — static analysis for shell scripts
  - macOS: `brew install shellcheck`
  - Linux: `apt install shellcheck`
- Git submodules initialized: `git submodule update --init`
  - Pulls `bats-support` and `bats-assert` into `test/helpers/`

## Steps

### 1. Run the full verification

```bash
make verify
```

Runs all checks in order: JSON validation → permission check → ShellCheck → bats tests. This is the single command to use before committing changes.

### 2. Run only the test suite

```bash
make test
```

Runs both unit and structure tests via bats-core (136 tests total as of CLI v0.1.7).

To run a subset:

```bash
make test-unit       # 81 unit tests (incl. launcher.bats)
make test-structure  # 55 structure tests
```

To run a single test file:

```bash
PLUGIN_ROOT=$(pwd) bats test/unit/normalize-stdin.bats
PLUGIN_ROOT=$(pwd) bats test/unit/launcher.bats      # CLI launcher tests
```

### 3. Run ShellCheck lint

```bash
make lint
```

Runs `shellcheck -s sh -x` on all bin/ scripts. The `-x` flag follows `source` directives so the normalizer library is checked in context.

### 4. Run quick structural checks (no bats needed)

```bash
make check-json    # validates all JSON configs via jq
make check-perms   # verifies bin/ scripts are executable
```

### 5. Run the AI-assisted verification skill

Inside Claude Code or Cursor:

```
/archcore:verify
```

This skill runs automated tests, then performs manual cross-reference checks that only an AI agent can do (README accuracy, archcore docs consistency, live MCP tool smoke tests).

### 6. Write a new test

**Unit test** — for bin/ script logic (stdin parsing, exit codes, output):

1. Create `test/unit/<script-name>.bats`
2. Use the standard setup:
   ```bash
   setup() {
     load '../helpers/common'
     common_setup
   }
   ```
3. Use helpers from `test/helpers/common.bash`:
   - `run_with_fixture <script> <fixture-path>` — run script with fixture file as stdin
   - `run_with_stdin <script> <inline-json>` — run script with inline stdin
   - `mock_archcore <output> [exit-code]` — create a mock `archcore` CLI on `PATH` for the test (picked up by the launcher's PATH resolution step)
   - `run_normalizer <json>` — source normalize-stdin.sh and print exported vars
4. Use bats-assert for assertions: `assert_success`, `assert_failure <code>`, `assert_output --partial <text>`

**Testing the CLI launcher** — `test/unit/launcher.bats` exercises `bin/archcore` directly:

- `ARCHCORE_BIN` override wins over all other resolution
- `archcore` on `PATH` is preferred over the cache
- Cache hit: `<cache>/archcore-v${VERSION}` is used when present
- Cache miss + `ARCHCORE_SKIP_DOWNLOAD=1` → exit 1 with diagnostic
- Checksum mismatch → abort, no install
- Loop guard: `PATH` containing the launcher itself is skipped

When writing launcher tests, override the cache directory via `CLAUDE_PLUGIN_DATA=$BATS_TEST_TMPDIR` so the test doesn't touch the user's real cache.

**Structure test** — for config/file validation:

1. Create `test/structure/<topic>.bats`
2. Same setup as unit tests
3. Use `$PLUGIN_ROOT` to reference project files
4. Use `jq` for JSON validation, `grep` for frontmatter checks
5. For scripts that invoke the CLI through the launcher, the `scripts.bats` structure tests verify the `"$SCRIPT_DIR/archcore"` invocation pattern

**Fixture** — mock stdin JSON for hook scripts:

1. Create `test/fixtures/stdin/<host>/<name>.json`
2. Hosts: `claude-code/`, `cursor/`, `copilot/`, `malformed/`
3. Match the exact JSON structure the hook receives from that host

## Verification

- `make verify` exits 0 with "All checks passed"
- All 136 tests show `ok` in the TAP output
- ShellCheck reports "all clean"
- No `not ok` lines in test output
- After breaking something intentionally (e.g., remove execute permission from a bin script, or clear `bin/CLI_VERSION`), the relevant test fails

## Common Issues

### bats-core not found

```
bats-core not found. Install: brew install bats-core
```

Install bats-core for your platform (see Prerequisites).

### bats-support/bats-assert not found

```
Could not find '.../bats-support/load'
```

Git submodules not initialized. Run:

```bash
git submodule update --init
```

### timeout command not found (macOS)

The test suite provides a `timeout` shim automatically for macOS. If you see timeout-related failures outside of tests, install GNU coreutils: `brew install coreutils`.

### Tests pass locally but fail in CI

- Check that `submodules: true` is set in the checkout step of the GitHub Actions workflow
- Ensure the CI runner has `jq` installed (it's not always pre-installed)
- On Linux, `/bin/sh` is `dash` (strict POSIX). On macOS, `/bin/sh` is bash in POSIX mode. If a test reveals a bashism in a bin script, fix the script — the bin scripts must be POSIX-compatible.

### Launcher tests try to download during CI

Launcher tests must pass `ARCHCORE_SKIP_DOWNLOAD=1` or pre-populate a mock binary in the overridden cache. If you see network timeouts in CI, a test is letting the download path execute. Audit the test for missing `ARCHCORE_SKIP_DOWNLOAD=1` or `ARCHCORE_BIN=<mock>`.

### ShellCheck SC2034 in normalize-stdin.sh

This is suppressed by a directive at the top of the file. The variables (ARCHCORE_HOST, ARCHCORE_TOOL_NAME, etc.) are exported for use by sourcing scripts.

### Adding a new bin script

When adding a new bin/ script:
1. Add `#!/bin/sh` shebang (POSIX launcher) or appropriate PowerShell header for Windows-only scripts
2. Make it executable: `chmod +x bin/<name>`
3. If it reads hook stdin, source the normalizer: `. "$SCRIPT_DIR/lib/normalize-stdin.sh"`
4. Add `# shellcheck source=lib/normalize-stdin.sh` before the source line
5. If the script invokes the Archcore CLI, call it as `"$SCRIPT_DIR/archcore"` (the launcher) so resolution order (ARCHCORE_BIN → PATH → cache → download) applies uniformly
6. Write tests in `test/unit/<name>.bats`
7. The structure tests will automatically verify permissions and shebang
