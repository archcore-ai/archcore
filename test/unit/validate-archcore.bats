#!/usr/bin/env bats
# Tests for bin/validate-archcore

setup() {
  load '../helpers/common'
  common_setup
}

# --- Triggers ---

@test "MCP tool triggers validation" {
  mock_archcore "All checks passed ✓"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
}

@test "Write to .archcore/ triggers validation" {
  mock_archcore "All checks passed ✓"
  run_with_fixture validate-archcore claude-code/write-archcore-settings.json
  assert_success
}

@test "Write to regular file skips validation" {
  # No mock needed — archcore should not be called
  run_with_fixture validate-archcore claude-code/write-regular.json
  assert_success
  assert_output ""
}

@test "empty stdin skips validation" {
  run_with_stdin validate-archcore ''
  assert_success
  assert_output ""
}

# --- Validation results ---

@test "clean validation produces no output" {
  mock_archcore "All checks passed ✓ 0 issues"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
  assert_output ""
}

@test "validation errors produce hook_info output" {
  mock_archcore "✗ orphaned relation: x.md → y.md"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
  assert_output --partial "validation found issues"
  assert_output --partial "orphaned relation"
}

@test "FAIL in validation output triggers info" {
  mock_archcore "FAIL: missing required field"
  run_with_fixture validate-archcore claude-code/mcp-update.json
  assert_success
  assert_output --partial "validation found issues"
}

# --- Graceful degradation ---

@test "missing archcore CLI exits silently" {
  # Override PATH to exclude real archcore but keep system tools
  run sh -c "PATH='/usr/bin:/bin' && cat '${FIXTURES}/stdin/claude-code/mcp-create.json' | '${PLUGIN_ROOT}/bin/validate-archcore'"
  assert_success
}

# --- Multi-host ---

@test "cursor MCP tool triggers validation" {
  mock_archcore "All checks passed ✓"
  run_with_fixture validate-archcore cursor/mcp-create.json
  assert_success
}

@test "cursor validation errors use cursor JSON format" {
  mock_archcore "✗ broken relation"
  run_with_fixture validate-archcore cursor/mcp-create.json
  assert_success
  assert_output --partial "additional_context"
}

@test "copilot native MCP update triggers validation with Copilot JSON" {
  mock_archcore "✗ broken relation"
  run_with_fixture validate-archcore copilot/posttooluse-mcp-update.json
  assert_success
  assert_output --partial '"additionalContext"'
  assert_output --partial "validation found issues"
}

# --- Tool-naming reach (host-wiring-parity.adr) ---
#
# validate-archcore's MCP arm gates on mcp__archcore__* only. The same server
# also reaches the model as mcp__plugin_archcore_archcore__* (plugin-bundled,
# Claude Code) and archcore-* (Copilot). normalize-stdin folds both to the
# canonical name; without that fold this hook runs and falls through to the
# Write/Edit arm, whose path check fails, so post-mutation validation is
# skipped in silence — the worst kind of gap, because the hook LOOKS wired.

@test "plugin-bundled MCP naming triggers validation" {
  mock_archcore "✗ broken relation"
  run sh -c "printf '%s' '{\"tool_name\":\"mcp__plugin_archcore_archcore__create_document\",\"tool_input\":{\"path\":\".archcore/x.adr.md\"}}' | '${PLUGIN_ROOT}/bin/validate-archcore'"
  assert_success
  assert_output --partial "validation found issues"
}

@test "Copilot flat MCP naming triggers validation" {
  mock_archcore "✗ broken relation"
  run sh -c "printf '%s' '{\"sessionId\":\"s1\",\"toolName\":\"archcore-create_document\",\"toolArgs\":\"{\\\"path\\\":\\\".archcore/x.adr.md\\\"}\"}' | '${PLUGIN_ROOT}/bin/validate-archcore'"
  assert_success
  assert_output --partial "validation found issues"
}

# --- Invocation contract: which subcommand actually ran? ---

@test "validate-archcore calls archcore doctor (not validate)" {
  # Guard against the real bug class: silently invoking a phantom subcommand.
  # mock_archcore_logging records every invocation to MOCK_ARCHCORE_LOG so we
  # can assert which subcommand the script chose.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging "All checks passed ✓"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"
  grep -qx 'doctor' "$MOCK_ARCHCORE_LOG" \
    || fail "expected 'doctor', got: $(cat "$MOCK_ARCHCORE_LOG")"
  ! grep -qx 'validate' "$MOCK_ARCHCORE_LOG" \
    || fail "phantom subcommand 'validate' was invoked"
}

@test "validate-archcore invokes only allowlisted subcommands" {
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging "All checks passed ✓"
  run_with_fixture validate-archcore claude-code/mcp-create.json
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"

  # Must match the canonical CLI surface allowlist guarded by readme-cli-references.bats.
  # The log records full invocations ("$*"); the subcommand is the first word.
  local allowed=" config doctor help hooks init mcp status update "
  local line sub
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    sub=${line%% *}
    case "$allowed" in
      *" $sub "*) ;;
      *) fail "validate-archcore invoked non-allowlisted subcommand '$sub' (full: '$line')" ;;
    esac
  done < "$MOCK_ARCHCORE_LOG"
}
