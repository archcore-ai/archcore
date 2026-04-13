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
