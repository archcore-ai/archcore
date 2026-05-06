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

  # Must match cli-contract.bats allowlist.
  local allowed=" config doctor help hooks init mcp status update "
  while IFS= read -r sub; do
    [ -z "$sub" ] && continue
    case "$allowed" in
      *" $sub "*) ;;
      *) fail "validate-archcore invoked non-allowlisted subcommand '$sub'" ;;
    esac
  done < "$MOCK_ARCHCORE_LOG"
}
