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

Any change to plugin code that invokes the `archcore` CLI MUST be accompanied by tests that assert the exact subcommand and arguments invoked. A passing test that did not verify *which* CLI subcommand ran does not satisfy this rule.

In particular:

1. Every shell-out from a `bin/*` script to the CLI (`archcore <subcmd> [args...]`) MUST be covered by a unit test that asserts the invoked subcommand via the `MOCK_ARCHCORE_LOG` mechanism (see `mock_archcore_logging` in `test/helpers/common.bash`).
2. Every `args` array in `.mcp.json`, `.codex.mcp.json`, and `cursor.mcp.json`, and every subcommand in any new `hooks/*.json`-referenced script, MUST name only canonical subcommands. The canonical surface as of plugin v0.4.0 is `config | doctor | help | hooks | init | mcp | status | update`.
3. Every prescriptive `` `archcore <subcmd>` `` reference in `README.md` MUST be guarded by `test/structure/readme-cli-references.bats`. Internal `.archcore/` design docs are intentionally excluded from this guard — they hold historical spec text that may legitimately reference renamed commands.
4. Skill or agent prose that instructs the agent to run `archcore <subcmd>` as a shell command MUST be reviewed against the canonical CLI surface; prefer routing CLI work through MCP tools (`mcp__archcore__*`) rather than shell-outs, so the agent stays inside the validated path.

A change is "covered" only when the test would fail if the code regressed to a phantom subcommand.

## Rationale

A real bug shipped because no test caught it: `bin/validate-archcore` invoked `archcore validate`, which is not a CLI subcommand. The CLI returned exit 1 on every PostToolUse mutation, but the hook wraps the call in `|| true` and uses `timeout 2`, so production silently logged nothing while the test suite reported green — `mock_archcore` returned canned output regardless of the subcommand.

This is a structural class of failure, not a one-off:

- Hook scripts run with short timeouts and `|| true` error suppression. A wrong subcommand fails silently in production.
- A test that asserts only `assert_success` is satisfied by the silent failure.
- README and design docs that reference `archcore validate` look correct because the *string* is plausible and the CLI never prevented anyone from typing it.

Locking the contract at the test layer closes this gap so it cannot return through inattention or a CLI version bump.

The why-now: the bug was caught manually in Codex CLI, where hook output is more visible. We do not want to depend on accidental visibility for a contract this important.

### What this rule no longer references

This rule was originally framed around the bundled launcher (`$LAUNCHER` / `bin/archcore` indirection) and a hardcoded allowlist in `test/structure/cli-contract.bats` pinned to `bin/CLI_VERSION`. As of plugin v0.4.0, the launcher and `CLI_VERSION` are gone — the plugin invokes `archcore <subcmd>` directly via PATH (see `remove-bundled-launcher-global-cli.idea`). The rule's spirit is unchanged: pin the exact subcommand invocation at the test layer. The mechanics are simpler now because there is no intermediate launcher to mock.

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
# Structure test (test/structure/readme-cli-references.bats):
ARCHCORE_SUBCOMMANDS="config doctor help hooks init mcp status update"

@test "every \`archcore <subcmd>\` reference in README.md names a real subcommand" {
  local refs
  refs=$(grep -oE '`archcore[[:space:]]+[a-z][a-z0-9-]*' "$PLUGIN_ROOT/README.md" \
    | sed -E 's/^`archcore[[:space:]]+//' \
    | sort -u)
  local sub
  while IFS= read -r sub; do
    case " $ARCHCORE_SUBCOMMANDS " in
      *" $sub "*) ;;
      *) fail "README.md references phantom subcommand '$sub'" ;;
    esac
  done <<< "$refs"
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
# Asserting only on the script's stdout. The CLI returns 1, the
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

The rule is enforced by these tests, which ship in the plugin:

- **`test/structure/readme-cli-references.bats`** — every code-quoted `` `archcore <subcmd>` `` in `README.md` must name a canonical subcommand. The allowlist is hardcoded in the test file and tracks the CLI's `archcore --help` surface.
- **`test/unit/validate-archcore.bats`** — invocation-log assertions using `MOCK_ARCHCORE_LOG` for `validate-archcore`. The two relevant tests are `validate-archcore calls archcore doctor (not validate)` and `validate-archcore invokes only allowlisted subcommands`.
- **`test/unit/session-start.bats`** — covers the missing-CLI fallback (the hook must exit 0 and emit install guidance, not block the session) and verifies `session-start invokes only the 'hooks' subcommand`.

When the canonical CLI surface changes upstream (new subcommand added, renamed, or removed), update `ARCHCORE_SUBCOMMANDS` in `readme-cli-references.bats` and the equivalent constant in any unit test that asserts on it. Any new subcommand the plugin starts invoking from a bin/ script gets its own invocation-log assertion before merge.

A change that does not satisfy this rule is rejected in code review. The rule applies to:

- Any script under `bin/` that calls `archcore`
- `args` arrays in `.mcp.json`, `.codex.mcp.json`, `cursor.mcp.json`
- Any new hook config (`hooks/*.json`) referencing a CLI-invoking script
- README and other user-facing prescriptive docs naming `` `archcore <subcmd>` `` invocations
- Skill or agent prompt text that instructs the agent to run an `archcore` shell command
