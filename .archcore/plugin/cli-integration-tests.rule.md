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

1. WHEN a change modifies code that invokes the `archcore` CLI, the author MUST pin the invoked subcommand in a test.
2. WHEN a change modifies code that invokes the `archcore` CLI, the author MUST pin the passed arguments in a test.
3. The author MUST cover every `bin/*` shell-out to `archcore` with a unit test asserting the subcommand and its arguments.
4. The author MUST restrict every `args` array in `.mcp.json`, `.codex.mcp.json`, and `docs/cursor.mcp.example.json` to canonical subcommands.
5. The author MUST restrict every subcommand named by a script referenced from `hooks/*.json` to canonical subcommands.
6. The author MUST guard every prescriptive `` `archcore <subcmd>` `` reference in `README.md` with `@test/structure/readme-cli-references.bats`.
7. WHEN the author adds a test under items 1–3, that test MUST fail against a subcommand outside the canonical surface.
8. WHEN skill or agent prose names a shell `archcore <subcmd>`, the reviewer MUST check it against the canonical surface.
9. WHEN a skill or an agent needs CLI work, the author SHOULD route the work through an `mcp__archcore__*` tool instead of a shell-out.
10. WHEN the canonical CLI surface changes upstream, the author MUST update `ARCHCORE_SUBCOMMANDS` in `@test/structure/readme-cli-references.bats`.
11. WHEN the canonical CLI surface changes upstream, the author MUST update the equivalent constant in each unit test that asserts on that surface.
12. WHEN a `bin/*` script starts invoking a new subcommand, the author MUST add its invocation assertion before merge.

### Notes (non-normative)

- The canonical surface as of plugin v0.7.4 is `config | doctor | help | hooks | init | mcp | status | update`, hardcoded as `ARCHCORE_SUBCOMMANDS` in `@test/structure/readme-cli-references.bats`.
- Item 7 is the fault-injection obligation: a test that has never been observed failing has an unknown failure mode, and the Copilot release recorded in `copilot-adapter-design.adr` shipped exactly because its assertions matched the defect instead of catching it.
- Two assertion mechanisms satisfy item 3. A launcher that hands its payload to a CLI hook leaf is pinned by capturing the mock CLI's `"$@"` and diffing its stdin — see `@test/unit/hook-launchers.bats`. A script that shells out for its own purposes is pinned through `MOCK_ARCHCORE_LOG`, provided by `mock_archcore_logging` in `@test/helpers/common.bash` — see `@test/unit/session-start.bats`.
- `docs/cursor.mcp.example.json` may carry `--project <path>` after `mcp`. `--project` is a flag, not a subcommand, and `@test/structure/cursor-plugin.bats` locks its position.
- Item 6 excludes `.archcore/` design documents. They hold historical specification text that may name renamed commands.

## Rationale

A shipped bug motivated this rule: `bin/validate-archcore` invoked `archcore validate`, which is not a CLI subcommand. The CLI exited 1 on every PostToolUse mutation, the hook suppressed the failure with `|| true` under `timeout 2`, and production logged nothing while the suite reported green — `mock_archcore` returned canned output for any subcommand. The failure class is structural, not incidental: hook scripts suppress errors by design, a test asserting only `assert_success` passes on a silent failure, and a plausible subcommand string in prose is never rejected by the CLI. The bug surfaced manually in Codex CLI, where hook output is visible; items 1–12 remove the dependence on that visibility.

### What this rule no longer references

The rule was first written around the bundled launcher (`$LAUNCHER` and the `bin/archcore` indirection) and a hardcoded allowlist in `test/structure/cli-contract.bats` pinned to `bin/CLI_VERSION`. Plugin v0.4.0 removed the launcher, `CLI_VERSION`, and that structure test; the plugin now invokes `archcore <subcmd>` directly through PATH (see `remove-bundled-launcher-global-cli.idea`).

Plugin v0.7.0 (`ca6dfb4`) removed the second layer of scaffolding this rule was written against. `cli-owns-layers-4-5.adr` moved the hook policy into the CLI binary, so `bin/validate-archcore` — the script whose bug motivated the rule — no longer exists, and `test/unit/validate-archcore.bats` went with it. What remains is thinner and easier to pin: three launchers, each shelling out to one `archcore hooks <host> <leaf>` invocation. The obligation is unchanged, and the failure class is unchanged with it — a wrong leaf name still exits non-zero, and both launchers still fail open.

## Examples

### Good

```bash
# Unit test (test/unit/hook-launchers.bats): pin the exact CLI leaf and args.
@test "post-tool-use: claude-code payload → 'hooks claude-code post-tool-use'" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks claude-code post-tool-use "
}
```

```bash
# Unit test (test/unit/session-start.bats): pin an own-purpose shell-out.
@test "update advisory: probe is exactly 'update --check', never a bare update" {
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""
  run "$PLUGIN_ROOT/bin/session-start" <<< "$CLAUDE_PAYLOAD"
  assert_success
  grep -qx 'update --check' "$MOCK_ARCHCORE_LOG" \
    || fail "expected 'update --check', got: $(cat "$MOCK_ARCHCORE_LOG")"
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
# Mock that swallows any input — phantom leaf passes silently.
mock_archcore "All checks passed ✓"
run_with_fixture post-tool-use claude-code/mcp-create.json
assert_success   # <-- meaningless; even `archcore unicorn` would pass
```

```bash
# Asserting only on the script's stdout. The CLI returns 1, the
# launcher swallows the error, the script prints nothing. Test passes.
run_with_fixture post-tool-use claude-code/mcp-create.json
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
- `@test/unit/hook-launchers.bats` — pins the CLI leaf for both mutation launchers (`hooks <host> pre-tool-use` and `hooks <host> post-tool-use`), diffs the forwarded stdin byte for byte, asserts the `codex` → `codex-cli` agent-id mapping, and asserts the silent exit 0 when the CLI is absent or older than 0.7.0.
- `@test/unit/session-start.bats` — covers the missing-CLI fallback (the hook exits 0 and emits install guidance instead of blocking the session), asserts `session-start invokes only allowlisted subcommands` and `advisory path still invokes only the 'hooks' subcommand`, pins the update probe as exactly `update --check`, and asserts the plugin-install-dir guard for sibling `.cursor-plugin/`, `.claude-plugin/`, and `.codex-plugin/` manifests.
- Code review rejects a change that does not satisfy items 1–12.
