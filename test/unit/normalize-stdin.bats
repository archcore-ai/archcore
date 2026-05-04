#!/usr/bin/env bats
# Tests for bin/lib/normalize-stdin.sh

setup() {
  load '../helpers/common'
  common_setup
}

# --- Host detection ---

@test "detects claude-code host from stdin" {
  run_normalizer '{"tool_name":"Write","tool_input":{"file_path":"src/app.py"}}'
  assert_success
  assert_line "HOST=claude-code"
}

@test "detects cursor host from conversation_id" {
  run_normalizer '{"conversation_id":"abc","hook_event_name":"preToolUse","tool_name":"Write"}'
  assert_success
  assert_line "HOST=cursor"
}

@test "detects copilot host from hookEventName" {
  run_normalizer '{"hookEventName":"PreToolUse","tool_name":"Write"}'
  assert_success
  assert_line "HOST=copilot"
}

@test "detects codex host from turn_id" {
  run_normalizer '{"turn_id":"abc","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"src/app.py"}}'
  assert_success
  assert_line "HOST=codex"
}

@test "cursor wins over codex when both conversation_id and turn_id present" {
  run_normalizer '{"conversation_id":"x","turn_id":"y","hook_event_name":"preToolUse","tool_name":"Write"}'
  assert_success
  assert_line "HOST=cursor"
}

@test "copilot wins over codex when both hookEventName and turn_id present" {
  run_normalizer '{"hookEventName":"PreToolUse","turn_id":"y","tool_name":"Write"}'
  assert_success
  assert_line "HOST=copilot"
}

@test "env ARCHCORE_HOST overrides detection" {
  run_normalizer_with_env '{"tool_name":"Write"}' "cursor"
  assert_success
  assert_line "HOST=cursor"
}

@test "empty stdin defaults to claude-code" {
  run_normalizer ''
  assert_success
  assert_line "HOST=claude-code"
}

@test "malformed stdin defaults to claude-code" {
  run_normalizer 'not json at all'
  assert_success
  assert_line "HOST=claude-code"
}

@test "missing fields defaults to claude-code" {
  run_normalizer '{"some_unknown_field":"value"}'
  assert_success
  assert_line "HOST=claude-code"
}

# --- Claude Code field extraction ---

@test "claude-code: extracts tool_name" {
  run_normalizer '{"tool_name":"mcp__archcore__create_document","tool_input":{}}'
  assert_success
  assert_line "TOOL=mcp__archcore__create_document"
}

@test "claude-code: extracts file_path" {
  run_normalizer '{"tool_name":"Write","tool_input":{"file_path":".archcore/my.adr.md"}}'
  assert_success
  assert_line "FILE=.archcore/my.adr.md"
}

@test "claude-code: extracts doc path" {
  run_normalizer '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"auth/jwt.adr.md"}}'
  assert_success
  assert_line "DOC=auth/jwt.adr.md"
}

@test "claude-code: empty tool_name yields empty TOOL" {
  run_normalizer '{"tool_input":{"file_path":"x.py"}}'
  assert_success
  assert_line "TOOL="
}

# --- Cursor field extraction ---

@test "cursor preToolUse: tool_name unchanged" {
  run_normalizer '{"conversation_id":"x","hook_event_name":"preToolUse","tool_name":"Write"}'
  assert_success
  assert_line "TOOL=Write"
}

@test "cursor afterMCPExecution: bare tool gets mcp__archcore__ prefix" {
  run_normalizer '{"conversation_id":"x","hook_event_name":"afterMCPExecution","tool_name":"create_document"}'
  assert_success
  assert_line "TOOL=mcp__archcore__create_document"
}

@test "cursor beforeMCPExecution: bare tool gets mcp__archcore__ prefix" {
  run_normalizer '{"conversation_id":"x","hook_event_name":"beforeMCPExecution","tool_name":"update_document"}'
  assert_success
  assert_line "TOOL=mcp__archcore__update_document"
}

@test "cursor afterMCPExecution: extracts path from escaped tool_input" {
  run_normalizer '{"conversation_id":"x","hook_event_name":"afterMCPExecution","tool_name":"update_document","tool_input":"{\"path\":\"auth/jwt.adr.md\"}"}'
  assert_success
  assert_line "DOC=auth/jwt.adr.md"
}

@test "cursor: extracts file_path" {
  run_normalizer '{"conversation_id":"x","hook_event_name":"preToolUse","tool_name":"Write","tool_input":{"file_path":".archcore/my.md"}}'
  assert_success
  assert_line "FILE=.archcore/my.md"
}

# --- Copilot field extraction ---

@test "copilot: extracts tool_name" {
  run_normalizer '{"hookEventName":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"x.py"}}'
  assert_success
  assert_line "TOOL=Write"
}

# --- Codex field extraction ---

@test "codex: preserves snake_case mcp tool_name" {
  run_normalizer '{"turn_id":"abc","hook_event_name":"PostToolUse","tool_name":"mcp__archcore__create_document","tool_input":{}}'
  assert_success
  assert_line "TOOL=mcp__archcore__create_document"
}

@test "codex: extracts file_path from tool_input" {
  run_normalizer '{"turn_id":"abc","hook_event_name":"PreToolUse","tool_name":"apply_patch","tool_input":{"file_path":".archcore/test.adr.md"}}'
  assert_success
  assert_line "FILE=.archcore/test.adr.md"
}

@test "codex: extracts doc path from tool_input" {
  run_normalizer '{"turn_id":"abc","hook_event_name":"PostToolUse","tool_name":"mcp__archcore__update_document","tool_input":{"path":"auth/jwt.adr.md"}}'
  assert_success
  assert_line "DOC=auth/jwt.adr.md"
}

# --- archcore_hook_block ---

@test "archcore_hook_block exits with code 2" {
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  "'
  assert_failure 2
}

@test "archcore_hook_block writes reason to stderr" {
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>&1'
  assert_output --partial "blocked reason"
}

# --- archcore_hook_info ---

@test "archcore_hook_info claude-code: outputs hookSpecificOutput JSON" {
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output --partial '"hookSpecificOutput"'
  assert_output --partial '"additionalContext":"test message"'
}

@test "archcore_hook_info cursor: outputs additional_context JSON" {
  run sh -c 'printf "%s" "{\"conversation_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output --partial '"additional_context":"test message"'
}

@test "archcore_hook_info codex: outputs hookSpecificOutput JSON" {
  run sh -c 'printf "%s" "{\"turn_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output --partial '"hookSpecificOutput"'
  assert_output --partial '"additionalContext":"test message"'
}

@test "archcore_hook_info escapes quotes in message" {
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"say \\\"hello\\\"\"
  "'
  assert_success
  assert_output --partial '\"hello\"'
}

# --- archcore_hook_allow ---

@test "archcore_hook_allow exits with code 0" {
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_allow
  "'
  assert_success
  assert_output ""
}
