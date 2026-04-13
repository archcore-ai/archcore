#!/usr/bin/env bats
# Tests for bin/check-archcore-write

setup() {
  load '../helpers/common'
  common_setup
}

# --- Blocking ---

@test "blocks write to .archcore/*.md" {
  run_with_fixture check-archcore-write claude-code/write-archcore.json
  assert_failure 2
}

@test "blocks edit to .archcore/*.md" {
  run_with_fixture check-archcore-write claude-code/edit-archcore.json
  assert_failure 2
}

@test "block message mentions MCP tools" {
  run sh -c "cat '${FIXTURES}/stdin/claude-code/write-archcore.json' | '${PLUGIN_ROOT}/bin/check-archcore-write' 2>&1"
  assert_output --partial "create_document"
  assert_output --partial "update_document"
  assert_output --partial "remove_document"
}

@test "blocks nested .archcore/ path" {
  run_with_stdin check-archcore-write '{"tool_name":"Write","tool_input":{"file_path":"project/.archcore/deep/doc.prd.md"}}'
  assert_failure 2
}

# --- Allowing ---

@test "allows .archcore/settings.json" {
  run_with_fixture check-archcore-write claude-code/write-archcore-settings.json
  assert_success
}

@test "allows .archcore/.sync-state.json" {
  run_with_fixture check-archcore-write claude-code/write-archcore-syncstate.json
  assert_success
}

@test "allows regular file" {
  run_with_fixture check-archcore-write claude-code/write-regular.json
  assert_success
}

@test "allows when no file_path" {
  run_with_stdin check-archcore-write '{"tool_name":"Write","tool_input":{}}'
  assert_success
}

@test "allows empty stdin" {
  run_with_stdin check-archcore-write ''
  assert_success
}

# --- Multi-host ---

@test "cursor: blocks write to .archcore/*.md" {
  run_with_fixture check-archcore-write cursor/write-archcore.json
  assert_failure 2
}

@test "copilot: blocks write to .archcore/*.md" {
  run_with_fixture check-archcore-write copilot/write-archcore.json
  assert_failure 2
}

@test "cursor: allows regular file" {
  run_with_fixture check-archcore-write cursor/preToolUse-write.json
  assert_success
}
