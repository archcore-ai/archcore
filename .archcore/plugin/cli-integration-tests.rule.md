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

1. WHEN a change modifies plugin code that invokes the `archcore` CLI, the author MUST add or update a test that asserts the exact subcommand the code invokes.
2. WHEN a change modifies plugin code that invokes the `archcore` CLI, the author MUST add or update a test that asserts the exact arguments the code passes.
3. The author MUST cover every shell-out from a `bin/*` script to `archcore <subcmd> [args...]` with a unit test that asserts the invoked subcommand through the `MOCK_ARCHCORE_LOG` mechanism provided by `mock_archcore_logging` in `@test/helpers/common.bash`.
4. The author MUST restrict every `args` array in `.mcp.json`, `.codex.mcp.json`, and `docs/cursor.mcp.example.json` to canonical subcommands.
5. The author MUST restrict every subcommand named by a script referenced from `hooks/*.json` to canonical subcommands.
6. The author MUST guard every prescriptive `` `archcore <subcmd>` `` reference in `README.md` with `@test/structure/readme-cli-references.bats`.
7. The author MUST verify that a test added under items 1–3 fails when the code regresses to a subcommand outside the canonical surface.
8. WHEN skill or agent prose instructs the agent to run `archcore <subcmd>` as a shell command, the reviewer MUST check that subcommand against the canonical CLI surface.
9. WHEN a skill or an agent needs CLI work, the author SHOULD route the work through an `mcp__archcore__*` tool instead of a shell-out.
10. WHEN the canonical CLI surface changes upstream, the author MUST update `ARCHCORE_SUBCOMMANDS` in `@test/structure/readme-cli-references.bats`.
11. WHEN the canonical CLI surface changes upstream, the author MUST update the equivalent constant in each unit test that asserts on that surface.
12. WHEN a `bin/*` script begins to invoke a subcommand it did not invoke before, the author MUST add an invocation-log assertion for that subcommand before merge.

### Notes (non-normative)

- The canonical surface as of plugin v0.4.0 is `config | doctor | help | hooks | init | mcp | status | update`.
- `docs/cursor.mcp.example.json` may carry `--project <path>` after `mcp`. `--project` is a flag, not a subcommand, and `@test/structure/cursor-plugin.bats` locks its position.
- Item 6 excludes `.archcore/` design documents. They hold historical specification text that may name renamed commands.

## Rationale

A shipped bug motivated this rule: `bin/validate-archcore` invoked `archcore validate`, which is not a CLI subcommand. The CLI exited 1 on every PostToolUse mutation, the hook suppressed the failure with `|| true` under `timeout 2`, and production logged nothing while the suite reported green — `mock_archcore` returned canned output for any subcommand. The failure class is structural, not incidental: hook scripts suppress errors by design, a test asserting only `assert_success` passes on a silent failure, and a plausible subcommand string in prose is never rejected by the CLI. The bug surfaced manually in Codex CLI, where hook output is visible; items 1–12 remove the dependence on that visibility.

### What this rule no longer references

The rule was first written around the bundled launcher (`$LAUNCHER` and the `bin/archcore` indirection) and a hardcoded allowlist in `test/structure/cli-contract.bats` pinned to `bin/CLI_VERSION`. Plugin v0.4.0 removed the launcher and `CLI_VERSION`; the plugin now invokes `archcore <subcmd>` directly through PATH (see `remove-bundled-launcher-global-cli.idea`). The obligation is unchanged — pin the exact subcommand at the test layer — and the mechanics are shorter because no intermediate launcher needs mocking.

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

- `@test/structure/readme-cli-references.bats` — asserts that every code-quoted `` `archcore <subcmd>` `` in `README.md` names a canonical subcommand. The allowlist is hardcoded in the test file and tracks the `archcore --help` surface.
- `@test/structure/cursor-plugin.bats` — locks `docs/cursor.mcp.example.json`: the command is `archcore`, the `args` array contains `mcp` followed by `--project ${workspaceFolder}`, no `cwd` field is present, and no legacy `cursor.mcp.json` exists at the plugin root.
- `@test/unit/validate-archcore.bats` — asserts invocation logs through `MOCK_ARCHCORE_LOG`. The two relevant tests are `validate-archcore calls archcore doctor (not validate)` and `validate-archcore invokes only allowlisted subcommands`.
- `@test/unit/session-start.bats` — covers the missing-CLI fallback (the hook exits 0 and emits install guidance instead of blocking the session), asserts `session-start invokes only the 'hooks' subcommand`, and asserts the plugin-install-dir guard (silent exit when sibling `.cursor-plugin/`, `.claude-plugin/`, or `.codex-plugin/` manifests are present).
- Code review rejects a change that does not satisfy items 1–12.
