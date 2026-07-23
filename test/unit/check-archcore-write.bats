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

# Copilot deny contract (copilot-adapter-design.adr): stdout permissionDecision
# JSON + exit 0 — exit 2 is only a warning on this host and does NOT block.
@test "copilot: denies write to .archcore/*.md via permissionDecision JSON" {
  run_with_fixture check-archcore-write copilot/pretooluse-create-archcore.json
  assert_success
  assert_output --partial '"permissionDecision":"deny"'
  assert_output --partial 'create_document'
}

@test "copilot: legacy hybrid payload is also denied via permissionDecision JSON" {
  run_with_fixture check-archcore-write copilot/legacy-hybrid-write.json
  assert_success
  assert_output --partial '"permissionDecision":"deny"'
}

@test "copilot: allows regular file silently" {
  run_with_fixture check-archcore-write copilot/pretooluse-edit-regular.json
  assert_success
  assert_output ""
}

@test "cursor: allows regular file" {
  run_with_fixture check-archcore-write cursor/preToolUse-write.json
  assert_success
}

@test "codex: blocks write to .archcore/*.md" {
  run_with_fixture check-archcore-write codex/write-archcore.json
  assert_failure 2
}

@test "codex: blocks apply_patch to .archcore/*.md" {
  run_with_fixture check-archcore-write codex/apply-patch-archcore.json
  assert_failure 2
}

@test "codex: block message mentions MCP tools" {
  run sh -c "cat '${FIXTURES}/stdin/codex/write-archcore.json' | '${PLUGIN_ROOT}/bin/check-archcore-write' 2>&1"
  assert_output --partial "create_document"
}

@test "codex: allows regular file" {
  run_with_fixture check-archcore-write codex/write-regular.json
  assert_success
}

@test "opencode: blocks write to .archcore/*.md (exit 2 + stderr)" {
  run_with_fixture_env check-archcore-write opencode/write-archcore.json opencode
  assert_failure 2
}

@test "opencode: block reason lands on stderr, stdout stays empty" {
  run sh -c "cat '${FIXTURES}/stdin/opencode/write-archcore.json' | ARCHCORE_HOST=opencode '${PLUGIN_ROOT}/bin/check-archcore-write' 2>/dev/null"
  assert_failure 2
  assert_output ""
}

@test "opencode: allows regular file" {
  run_with_fixture_env check-archcore-write opencode/write-regular.json opencode
  assert_success
}
