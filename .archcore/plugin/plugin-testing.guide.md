---
title: "Plugin Testing Guide"
status: accepted
tags:
  - "development"
  - "plugin"
  - "testing"
---

## Purpose

Run, extend, and debug the Archcore plugin's test suite: the full verification gate, the unit and structure suites, the host install smoke tests, and the live-session probes — plus the conventions a new test, a new bin script, or a new host must follow.

## Prerequisites

- [bats-core](https://github.com/bats-core/bats-core) — the test runner for shell scripts. macOS: `brew install bats-core`. Linux: `apt install bats`.
- `jq`, ShellCheck, and initialized git submodules. `plugin-development.guide` owns the environment setup; `make verify` fails with a named tool when something is missing, and Common Issues below covers the failures that are specific to the suite.

## Steps

### 1. Run the full verification

```bash
make verify
```

Expected result: exit 0 with `All checks passed`. The target runs JSON validation, the permission check, ShellCheck, and the bats tests, in that order. Use this single command before committing.

### 2. Run only the test suite

```bash
make test
```

Runs both unit and structure tests through bats-core. The suite grows with the plugin, so compare against the count `make test` prints rather than against a number written here.

To run a subset:

```bash
make test-unit       # unit tests for bin/ script logic
make test-structure  # structure tests for configs and frontmatter
```

To run a single test file:

```bash
PLUGIN_ROOT=$(pwd)/plugins/archcore REPO_ROOT=$(pwd) bats test/unit/normalize-stdin.bats
PLUGIN_ROOT=$(pwd)/plugins/archcore REPO_ROOT=$(pwd) bats test/unit/hook-launchers.bats
```

`PLUGIN_ROOT` is worth knowing as more than boilerplate: pointing it at a modified copy of `plugins/archcore` is how fault injection works without touching the tree.

### 3. Run the ShellCheck lint

```bash
make lint
```

Expected result: `ShellCheck: all clean`. The target runs `shellcheck -s sh -x` on every `bin/` script; `-x` follows `source` directives so the normalizer library is checked in context.

### 4. Run the quick structural checks

```bash
make check-json    # validates all JSON configs via jq
make check-perms   # verifies bin/ scripts are executable
```

Neither needs bats.

### 5. Run the host install smoke tests

```bash
make test-codex-smoke     # requires the codex CLI on PATH
make test-copilot-smoke   # requires the copilot CLI on PATH
```

Neither runs as part of `make test`, and both **skip** rather than fail when their host binary is absent — so they ship, stay quiet in CI, and run for free on a contributor's machine that has the host installed. They cover what static tests cannot: that a real install produces a tree the host can load. That is the exact gap issue #2 fell through for Codex, where every structural test was green while marketplace discovery silently found nothing.

The Copilot smoke test asserts filesystem facts rather than CLI output: every path the manifest names survives the install, the hook scripts keep their executable bit, and no plugin MCP server is registered (github/copilot-cli#4234). Host output wording is not a contract the plugin controls; what the plugin promises is what lands on disk.

### 6. Run the plugin integrity check

`make verify` is the canonical integrity check. The former `/archcore:verify` skill was retired by `skill-surface-collapse.adr`; use the Makefile target instead. Inside a host session, ask the model to run `make verify` and report the results if you want AI-assisted verification.

Two structure files carry the per-host regression guards: `test/structure/host-coverage-matrix.bats` holds one enrolled row per hooks config, with an enrollment guard so a new host cannot ship unchecked, and `test/structure/copilot-plugin.bats`, `codex-plugin.bats`, and `cursor-plugin.bats` cover the per-host manifests.

### 7. Write a new unit test

Unit tests cover `bin/` script logic — stdin parsing, exit codes, output.

1. Create `test/unit/<script-name>.bats`.
2. Add the standard setup:
   ```bash
   setup() {
     load '../helpers/common'
     common_setup
   }
   ```
3. Load the helpers from `test/helpers/common.bash`; the table below lists them.
4. Assert with bats-assert: `assert_success`, `assert_failure <code>`, `assert_output --partial <text>`.

| Helper | Use |
|---|---|
| `run_with_fixture <script> <fixture-path>` | run the script with a fixture file as stdin |
| `run_with_stdin <script> <inline-json>` | run the script with inline stdin |
| `run_with_fixture_env <script> <fixture> <host>` | the same, with `ARCHCORE_HOST` forced — env-only hosts such as opencode need it |
| `mock_archcore <output> [exit-code]` | a mock `archcore` on `PATH` answering *any* subcommand; use only when the test does not care which one ran |
| `mock_archcore_logging <output> [exit-code]` | the same, appending `$1` to `$MOCK_ARCHCORE_LOG` when set; use whenever the test asserts which subcommand ran |
| `mock_archcore_multi` | multi-subcommand mock: `$MOCK_DOCTOR_OUTPUT` for `doctor`, `$MOCK_HOOKS_OUTPUT` for `hooks`, logging to `$MOCK_ARCHCORE_LOG` |
| `run_normalizer <json>` | source `normalize-stdin.sh` and print the exported variables |

For CLI fallback paths, `test/unit/session-start.bats` is the model: under a restricted PATH where `archcore` does not resolve, `bin/session-start` still exits 0 and emits the install message pointing at https://docs.archcore.ai/cli/install/.

One caveat when you compare two runs of `session-start`: `archcore hooks <host> session-start` emits its context only the first time it sees a given project, so two invocations in the same directory are not comparable and the second is legitimately empty. Give each invocation its own project directory — see `test/unit/probe-wrapper.bats`.

### 8. Write a new structure test

Structure tests validate configs and files.

1. Create `test/structure/<topic>.bats`.
2. Use the same setup as a unit test.
3. Reference plugin files through `$PLUGIN_ROOT` and repo-root files through `$REPO_ROOT`.
4. Validate JSON with `jq` and frontmatter with `grep`.
5. Follow `scripts.bats` for the direct `archcore` invocation pattern.
6. Guard README references with `readme-cli-references.bats` against the canonical allowlist.

Prefer a table over a copy when a test is per-host. Four hosts means four near-identical tests, and the fourth is the one nobody writes. Worse, a copied test can pass on an empty set: `jq '.. | .command?'` returns nothing for `hooks/copilot.hooks.json`, whose entries use `bash`. `test/structure/hooks.bats` shows the shape — one table of `host|config|plugin-root-variable`, a union accessor, an assertion that the extraction was not empty, and an enrollment guard so a fifth config cannot slip past the table.

### 9. Add a stdin fixture

1. Create `test/fixtures/stdin/<host>/<name>.json`.
2. Use one of the host directories: `claude-code/`, `cursor/`, `codex/`, `copilot/`, `opencode/`, `malformed/`.
3. Match the exact JSON structure the hook receives from that host.

Copilot's native payloads are the least guessable: the edited path lives inside an escaped JSON **string** under `toolArgs`, not in a nested object.

### 10. Assert CLI subcommand invocations

`cli-integration-tests.rule` requires that any change touching a script that invokes `archcore` be covered by a test pinning the exact subcommand. The contract holds at two layers.

At the structure layer, `test/structure/readme-cli-references.bats` extracts every backtick-quoted `archcore <subcmd>` reference in `README.md` and asserts membership in the canonical allowlist `config doctor help hooks init mcp status update`, hardcoded in that test file. Internal `.archcore/` design documents are excluded deliberately: they hold historical specification text that may name renamed commands.

At the unit layer, write a test using `mock_archcore_logging` with `MOCK_ARCHCORE_LOG`:

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

A test asserting only `assert_success` after a CLI invocation is insufficient: hooks swallow non-zero exits, so a phantom subcommand fails silently while the test passes. Always assert what was invoked. `test/unit/hook-launchers.bats` is the worked example — it pins `hooks <host> <leaf>` for both `bin/pre-tool-use` and `bin/post-tool-use`.

When the canonical CLI surface changes upstream, update `ARCHCORE_SUBCOMMANDS` in `readme-cli-references.bats` and add an invocation-log assertion for any new subcommand the plugin starts using.

### 11. Add a new bin script

`plugin-development.guide` owns the authoring conventions — shebang, executable bit, normalizer sourcing, shellcheck directive. What the suite then requires of that script:

1. Invoke `archcore <subcmd>` directly through PATH; the plugin bundles no launcher.
2. Wrap the call in `timeout` and `|| true` when the hook must stay non-blocking.
3. Write tests in `test/unit/<name>.bats` with the invocation assertion from step 10.
4. Add a case to `hook-latency.bats` when the script runs under a hook timeout.

Expected result: the structure tests verify permissions and shebang automatically, and `readme-cli-references.bats` guards any new subcommand named in the README.

### 12. Add a new host

1. Add the manifest, the hooks config, and the `normalize-stdin.sh` case (`host-adapter-contract.spec`).
2. Enroll the host in `host-coverage-matrix.bats`.
3. Enroll it in the `hooks.bats` resolution table.
4. Enroll it in the `json-configs.bats` metadata table.
5. Add stdin fixtures under `test/fixtures/stdin/<host>/`.
6. Add a smoke test under `test/integration/`, guarded by `command -v <host> || skip`.
7. Add its row to the probe records table.

Each of the three enrollments carries its own guard and fails until you do it. The probe row becomes mandatory automatically, because `probe-records.bats` derives the host list from the coverage matrix.

### 13. Guard a timing budget

`test/unit/hook-latency.bats` holds the hook timeout budgets — 2 seconds for PreToolUse, 4 for PostToolUse. It measures the plugin's own share (normalize-stdin, the plugin-cache guard, the `cli-gte` probe) against a mocked CLI, so the numbers stay about this repository rather than about the binary, and adds one end-to-end case that skips when no current CLI is installed.

Two conventions there are worth copying into any new timing test. Assert on **marginal** cost, not on a ratio: the launcher carries a fixed cost of roughly 50 ms, and that floor sits in the denominator — an injected per-document scan measured 83 ms at 25 documents and 309 ms at 200, a 226 ms regression that reads as only 3.7× and slips under a 4× bar. And use best-of-N with wide absolute bars, because CI runners are slower and noisier than a laptop while a real regression is a factor of ten.

### 14. Run the live-session probes

Some questions no bats test can answer: whether a host loads the hooks config at all, whether its matcher fires on the tool name the model actually chose, and whether a deny is honored or merely displayed. Those belong to `host-probe-protocol.spec`, and `test/probe/mkprobe` builds the disposable tree they run against. Everything mechanically checkable belongs in bats instead — `test/unit/hook-launchers.bats` is the model, turning "does the launcher reach the right CLI leaf with the payload intact" from a live question into a CI assertion.

## Verification

`plugin-development.guide` owns the `make verify` gate itself. What follows is how to read the suite's own output.

- Every line of the TAP output reads `ok`. Bats prints the total; there is no fixed number to match.
- No `not ok` line appears in the test output.
- ShellCheck reports `all clean`.
- After you break something on purpose — remove the execute bit from a bin script, or rename a bin script the Makefile references — the relevant test fails.

That last item is a habit rather than a formality, and `cli-integration-tests.rule` item 7 makes it an obligation for CLI-invocation tests. A test never observed failing is a test whose failure mode is unknown, which matters most for timing tests, table-driven tests, and anything iterating a set that could be empty. Point `PLUGIN_ROOT` at a modified copy to inject the defect without touching the tree.

## Common Issues

### bats-core not found

```
bats-core not found. Install: brew install bats-core
```

Install bats-core for your platform, as listed under Prerequisites.

### bats-support or bats-assert not found

```
Could not find '.../bats-support/load'
```

The git submodules are not initialized. Run:

```bash
git submodule update --init
```

### `timeout` command not found on macOS

The test suite provides a `timeout` shim automatically. If timeout-related failures appear outside the tests, install GNU coreutils with `brew install coreutils`.

### Tests pass locally but fail in CI

- Confirm that `submodules: true` is set in the checkout step of the GitHub Actions workflow.
- Confirm the CI runner has `jq` installed; it is not always pre-installed.
- On Linux `/bin/sh` is `dash`, which is strict POSIX; on macOS `/bin/sh` is bash in POSIX mode. If a test reveals a bashism in a bin script, fix the script — the bin scripts must be POSIX-compatible.
- `hook-latency.bats` is the one timing-sensitive file. Suspect an algorithmic regression before suspecting the runner: its bars sit several times above the measured cost, and the regressions they guard are factors of ten. It skips itself when `perl` is unavailable, and its end-to-end case skips without a current CLI.

### ShellCheck SC2034 in normalize-stdin.sh

A directive at the top of the file suppresses it. The variables — `ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, and the rest — are exported for the sourcing scripts.

### The README CLI-reference test fails after a subcommand rename

When the upstream CLI adds, removes, or renames a subcommand, `test/structure/readme-cli-references.bats` may flag an `archcore <subcmd>` reference in the README as phantom. Update `ARCHCORE_SUBCOMMANDS` in that file to match the new surface, then rerun the suite. Per `cli-integration-tests.rule`, any new subcommand the plugin starts invoking also needs a unit-level invocation-log assertion in the corresponding hook test.
