---
title: "CLI Integration Changes Require Strict Tests"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "rule"
  - "testing"
  - "validation"
---

## Rule

Any change to plugin code that invokes the bundled `archcore` CLI — directly or through `bin/archcore` — MUST be accompanied by tests that assert the exact subcommand and arguments invoked. A passing test that did not verify *which* CLI subcommand ran does not satisfy this rule.

In particular:

1. Every shell-out from a `bin/*` script to the launcher (`"$LAUNCHER"` / `"$SCRIPT_DIR/archcore"`) MUST be covered by a unit test that asserts the invoked subcommand via the `MOCK_ARCHCORE_LOG` mechanism (see `mock_archcore_logging` in `test/helpers/common.bash`).
2. Every `args` array in `.mcp.json` and `.codex.mcp.json`, and every subcommand in any new `hooks/*.json`-referenced script, MUST be covered by the allowlist guard in `test/structure/cli-contract.bats`.
3. Every prescriptive `` `archcore <subcmd>` `` reference in `README.md` MUST be guarded by `test/structure/readme-cli-references.bats`.
4. Skill or agent prose that instructs the agent to run `archcore <subcmd>` as a shell command MUST be reviewed against the canonical CLI surface and either pinned by an additional structure test or rewritten to delegate through the launcher (preferred).

A change is "covered" only when the test would fail if the code regressed to a phantom subcommand.

## Rationale

A real bug shipped because no test caught it: `bin/validate-archcore` invoked `archcore validate`, which is not a CLI subcommand (the canonical surface is `config | doctor | help | hooks | init | mcp | status | update`). The launcher returned exit 1 on every PostToolUse mutation, but the hook wraps the call in `|| true` and uses `timeout 2`, so production silently logged nothing while the test suite reported green — `mock_archcore` returned canned output regardless of the subcommand.

This is a structural class of failure, not a one-off:

- Hook scripts run with short timeouts and `|| true` error suppression. A wrong subcommand fails silently in production.
- A test that asserts only `assert_success` is satisfied by the silent failure.
- README and design docs that reference `archcore validate` look correct because the *string* is plausible and the CLI never prevented anyone from typing it.

Locking the contract at the test layer closes this gap so it cannot return through inattention or a CLI version bump.

The why-now: the bug was caught manually in Codex CLI, where hook output is more visible. We do not want to depend on accidental visibility for a contract this important.

## Examples

### Good

```bash
# Unit test (test/unit/validate-archcore.bats):
@test "validate-archcore calls archcore doctor (not validate)" {
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging "All checks passed ✓"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
  grep -qx 'doctor' "$MOCK_ARCHCORE_LOG" \
    || fail "expected 'doctor', got: $(cat "$MOCK_ARCHCORE_LOG")"
  ! grep -qx 'validate' "$MOCK_ARCHCORE_LOG" \
    || fail "phantom subcommand 'validate' was invoked"
}
```

```bash
# Structure test (test/structure/cli-contract.bats):
ARCHCORE_SUBCOMMANDS="config doctor help hooks init mcp status update"

@test "bin/validate-archcore invokes only allowlisted subcommands" {
  local sub
  for sub in $(grep -oE '"\$LAUNCHER"[[:space:]]+[a-z][a-z0-9-]*' \
                 "$PLUGIN_ROOT/bin/validate-archcore" \
                 | sed -E 's/^"\$LAUNCHER"[[:space:]]+//'); do
    case " $ARCHCORE_SUBCOMMANDS " in
      *" $sub "*) ;;
      *) fail "phantom subcommand '$sub'" ;;
    esac
  done
}
```

### Bad

```bash
# Mock that swallows any input — phantom subcommand passes silently.
mock_archcore "All checks passed ✓"
run_with_fixture validate-archcore claude-code/mcp-create.json
assert_success   # <-- meaningless; even `archcore unicorn` would pass
```

```bash
# Asserting only on the script's stdout. The launcher returns 1, the
# hook swallows the error, the script prints nothing. Test passes.
run_with_fixture validate-archcore claude-code/mcp-create.json
assert_success
assert_output ""
```

```markdown
<!-- README.md prose without a guarding test -->
- **Validation** — runs `archcore some-future-name` after every document mutation
```

## Enforcement

The rule is enforced by these tests, which already ship in the plugin:

- **`test/structure/cli-contract.bats`** — the allowlist guard. Scans `bin/*` scripts, `.mcp.json`, `.codex.mcp.json`, and `hooks/*.json`-referenced scripts; fails if any subcommand passed to the launcher is not in the canonical surface. Also ships a sentinel that fails on any executable reference to the historical phantoms `archcore validate` and `archcore sync`.
- **`test/structure/readme-cli-references.bats`** — every code-quoted `` `archcore <subcmd>` `` in `README.md` must be allowlisted.
- **`test/unit/validate-archcore.bats`** and **`test/unit/session-start.bats`** — invocation-log assertions using `MOCK_ARCHCORE_LOG`. Pattern is documented in `plugin-testing.guide.md` step 7.
- **Live cross-check** — `cli-contract.bats` parses `archcore --help` when the launcher resolves the binary and fails when the hardcoded allowlist drifts from the live surface.

When `bin/CLI_VERSION` bumps, the live cross-check signals which subcommands changed; the hardcoded allowlist must be updated in lockstep, and any new subcommand the plugin starts invoking gets its own invocation-log assertion before merge.

A change that does not satisfy this rule is rejected in code review. The rule applies to:

- Any script under `bin/` that calls `"$LAUNCHER"` or `archcore`
- `.mcp.json` and `.codex.mcp.json` `args` arrays
- Any new hook config (`hooks/*.json`) referencing a CLI-invoking script
- README and other user-facing prescriptive docs naming `` `archcore <subcmd>` `` invocations
- Skill or agent prompt text that instructs the agent to run an `archcore` shell command
