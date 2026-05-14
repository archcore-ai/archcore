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

Runs both unit and structure tests via bats-core (212 tests total as of the global-CLI cutover).

To run a subset:

```bash
make test-unit       # unit tests for bin/ script logic
make test-structure  # structure tests for configs and frontmatter
```

To run a single test file:

```bash
PLUGIN_ROOT=$(pwd) bats test/unit/normalize-stdin.bats
PLUGIN_ROOT=$(pwd) bats test/unit/validate-archcore.bats
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

### 5. Plugin integrity check

`make verify` is the canonical way to run plugin integrity checks. The previous `/archcore:verify` skill was retired by `skill-surface-collapse.adr.md` — use the Makefile target instead. Inside a host session, ask the model to "run make verify and report the results" if you want AI-assisted verification.

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
   - `mock_archcore <output> [exit-code]` — create a mock `archcore` CLI on `PATH` that returns canned output for *any* subcommand. Use only when the test does not care which subcommand was called.
   - `mock_archcore_logging <output> [exit-code]` — same as `mock_archcore`, but every invocation appends the subcommand argument (`$1`) to `$MOCK_ARCHCORE_LOG` if that env var is set. Use whenever the test needs to assert which subcommand the script invoked (recommended for any hook that shells out to the CLI).
   - `mock_archcore_multi` — multi-subcommand mock (responds with `$MOCK_DOCTOR_OUTPUT` for `doctor`, `$MOCK_HOOKS_OUTPUT` for `hooks`); also logs to `$MOCK_ARCHCORE_LOG` when set.
   - `run_normalizer <json>` — source normalize-stdin.sh and print exported vars
4. Use bats-assert for assertions: `assert_success`, `assert_failure <code>`, `assert_output --partial <text>`

**Testing CLI fallback paths** — `test/unit/session-start.bats` covers the missing-CLI fallback: with a restricted PATH where `archcore` is not resolvable, `bin/session-start` must still exit 0 and emit the install message pointing at https://docs.archcore.ai/cli/install/. Other hook scripts that shell out to `archcore` (e.g., `validate-archcore`) use `mock_archcore_logging` to assert which subcommand was invoked.

**Structure test** — for config/file validation:

1. Create `test/structure/<topic>.bats`
2. Same setup as unit tests
3. Use `$PLUGIN_ROOT` to reference project files
4. Use `jq` for JSON validation, `grep` for frontmatter checks
5. For scripts that invoke the CLI, the `scripts.bats` structure tests verify the direct `archcore` invocation pattern (no launcher indirection). README references to `archcore <subcmd>` are guarded by `readme-cli-references.bats` against the canonical surface allowlist.

**Fixture** — mock stdin JSON for hook scripts:

1. Create `test/fixtures/stdin/<host>/<name>.json`
2. Hosts: `claude-code/`, `cursor/`, `copilot/`, `malformed/`
3. Match the exact JSON structure the hook receives from that host

### 7. Assert CLI subcommand invocations

Per `cli-integration-tests.rule.md`, any change that touches a script invoking `archcore` MUST be covered by tests that pin the exact subcommand. The plugin enforces this contract at two layers:

**Structure layer** — `test/structure/readme-cli-references.bats` extracts every backtick-quoted `archcore <subcmd>` reference in `README.md` and asserts each is a member of the canonical surface allowlist: `config doctor help hooks init mcp status update`. The allowlist is hardcoded in the test file. Internal `.archcore/` design docs are intentionally excluded — they hold historical spec text that may legitimately reference renamed commands.

**Unit layer** — for any hook script that calls `archcore`, write a test that uses `mock_archcore_logging` plus `MOCK_ARCHCORE_LOG`:

```bash
@test "<script> calls only the expected subcommand" {
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""
  run_with_fixture <script> <fixture>
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"
  grep -qx '<expected-subcommand>' "$MOCK_ARCHCORE_LOG" \
    || fail "expected '<expected-subcommand>', got: $(cat "$MOCK_ARCHCORE_LOG")"
}
```

A test that asserts only `assert_success` after a CLI invocation is insufficient — hooks swallow non-zero exits, so a phantom subcommand fails silently and the test still passes. Always assert what was invoked.

When the canonical CLI surface changes upstream (new subcommand added/removed), update the `ARCHCORE_SUBCOMMANDS` constant in `readme-cli-references.bats` and add an invocation-log assertion for any new subcommand the plugin starts using.

## Verification

- `make verify` exits 0 with "All checks passed"
- All 212 tests show `ok` in the TAP output
- ShellCheck reports "all clean"
- No `not ok` lines in test output
- After breaking something intentionally (e.g., remove execute permission from a bin script, or rename a bin script the Makefile references), the relevant test fails

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

### ShellCheck SC2034 in normalize-stdin.sh

This is suppressed by a directive at the top of the file. The variables (ARCHCORE_HOST, ARCHCORE_TOOL_NAME, etc.) are exported for use by sourcing scripts.

### README CLI-reference test fails after a subcommand rename

When the upstream CLI adds, removes, or renames a subcommand, `test/structure/readme-cli-references.bats` may flag a `archcore <subcmd>` reference in the README as phantom. Update `ARCHCORE_SUBCOMMANDS` in that file to match the new surface, then rerun the suite. Per `cli-integration-tests.rule.md`, any new subcommand the plugin starts invoking also needs a unit-level invocation-log assertion in the corresponding hook test.

### Adding a new bin script

When adding a new bin/ script:
1. Add `#!/bin/sh` shebang (POSIX) or appropriate PowerShell header for Windows-only scripts
2. Make it executable: `chmod +x bin/<name>`
3. If it reads hook stdin, source the normalizer: `. "$SCRIPT_DIR/lib/normalize-stdin.sh"`
4. Add `# shellcheck source=lib/normalize-stdin.sh` before the source line
5. If the script invokes the Archcore CLI, call `archcore <subcmd>` directly (resolved via PATH). The plugin no longer bundles a launcher — assume the user has installed the CLI per https://docs.archcore.ai/cli/install/. Wrap the call with `timeout` and `|| true` if the hook must remain non-blocking.
6. Write tests in `test/unit/<name>.bats`. If the script invokes the CLI, include the invocation-log assertion described in step 7 above (mandated by `cli-integration-tests.rule.md`).
7. The structure tests will automatically verify permissions and shebang. README references to new subcommands are guarded by `readme-cli-references.bats`.
