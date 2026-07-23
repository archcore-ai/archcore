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

@test "detects copilot host from legacy hookEventName payload (fallback heuristic)" {
  run_normalizer '{"hookEventName":"PreToolUse","tool_name":"Write"}'
  assert_success
  assert_line "HOST=copilot"
}

@test "detects copilot host from native camelCase toolName (no hookEventName)" {
  run_normalizer '{"sessionId":"s1","timestamp":1751700000000,"cwd":"/work","toolName":"create","toolArgs":"{\"file_path\":\"x.md\"}"}'
  assert_success
  assert_line "HOST=copilot"
}

@test "copilot camelCase markers do not misroute snake_case hosts" {
  # claude-code / codex / cursor payloads are snake_case — the copilot markers
  # (toolName/toolArgs) must never match them. Guards the deny-semantics
  # asymmetry: a misdetected claude payload would fail open on block.
  run_normalizer '{"tool_name":"Write","tool_input":{"file_path":"src/app.py"}}'
  assert_line "HOST=claude-code"
  run_normalizer '{"turn_id":"abc","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"src/app.py"}}'
  assert_line "HOST=codex"
  run_normalizer '{"conversation_id":"abc","hook_event_name":"preToolUse","tool_name":"Write"}'
  assert_line "HOST=cursor"
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

@test "copilot: extracts toolName from native payload" {
  run_normalizer '{"sessionId":"s1","toolName":"create","toolArgs":"{\"file_path\":\".archcore/my.rule.md\"}"}'
  assert_success
  assert_line "TOOL=create"
}

@test "copilot: extracts file path from escaped toolArgs" {
  run_normalizer '{"sessionId":"s1","toolName":"create","toolArgs":"{\"file_path\":\".archcore/my.rule.md\",\"content\":\"x\"}"}'
  assert_success
  assert_line "FILE=.archcore/my.rule.md"
}

@test "copilot: extracts doc path from escaped toolArgs (MCP)" {
  run_normalizer '{"sessionId":"s1","toolName":"mcp__archcore__update_document","toolArgs":"{\"path\":\"auth/jwt.adr.md\"}"}'
  assert_success
  assert_line "DOC=auth/jwt.adr.md"
}

@test "copilot: legacy hybrid payload still extracts tool_name" {
  run_normalizer '{"hookEventName":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"x.py"}}'
  assert_success
  assert_line "TOOL=Write"
  assert_line "FILE=x.py"
}

# --- OpenCode (env-only host; bridge contract) ---

@test "env ARCHCORE_HOST=opencode is preserved (not clobbered to claude-code)" {
  # Load-bearing: without an explicit opencode extraction case, the * fallback
  # rewrites ARCHCORE_HOST to claude-code and misroutes helper output.
  run_normalizer_with_env '{"tool_name":"write"}' "opencode"
  assert_success
  assert_line "HOST=opencode"
}

@test "opencode: extracts tool_name, file_path, and doc path" {
  run_normalizer_with_env '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"auth/jwt.adr.md","file_path":".archcore/x.md"}}' "opencode"
  assert_success
  assert_line "TOOL=mcp__archcore__update_document"
  assert_line "FILE=.archcore/x.md"
  assert_line "DOC=auth/jwt.adr.md"
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

@test "archcore_hook_block claude-code: emits nothing on stdout" {
  # Block output goes to stderr only. Pins the shipped contract so a future
  # host-specific stdout-JSON deny arm can never leak into the default path.
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>/dev/null'
  assert_failure 2
  assert_output ""
}

@test "archcore_hook_block cursor: exits 2 with reason on stderr" {
  run sh -c 'printf "%s" "{\"conversation_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>&1'
  assert_failure 2
  assert_output --partial "blocked reason"
}

@test "archcore_hook_block codex: exits 2 with reason on stderr" {
  run sh -c 'printf "%s" "{\"turn_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>&1'
  assert_failure 2
  assert_output --partial "blocked reason"
}

@test "archcore_hook_block copilot: emits permissionDecision deny JSON and exits 0" {
  # Copilot deny contract: stdout JSON + exit 0 (exit 2 is only a warning there).
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=copilot sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>/dev/null'
  assert_success
  assert_output '{"permissionDecision":"deny","permissionDecisionReason":"blocked reason\n"}'
}

@test "archcore_hook_block opencode: exits 2 with reason on stderr, nothing on stdout" {
  # Bridge contract: exit 2 + stderr → the TS bridge throws Error(stderr).
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=opencode sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"blocked reason\"
  " 2>/dev/null'
  assert_failure 2
  assert_output ""
}

@test "archcore_hook_block copilot: escapes quotes and newlines in reason" {
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=copilot sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_block \"line one
line \\\"two\\\"\"
  "'
  assert_success
  assert_output --partial 'line one\nline \"two\"'
}

# --- archcore_hook_info ---

@test "archcore_hook_info claude-code: exact hookSpecificOutput JSON" {
  # Exact match (not --partial): any byte change to the shipped arm fails loudly.
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"test message"}}'
}

@test "archcore_hook_info cursor: exact additional_context JSON" {
  run sh -c 'printf "%s" "{\"conversation_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output '{"additional_context":"test message"}'
}

@test "archcore_hook_info codex: exact hookSpecificOutput JSON" {
  run sh -c 'printf "%s" "{\"turn_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"test message"}}'
}

@test "archcore_hook_info copilot: exact top-level additionalContext JSON" {
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=copilot sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output '{"additionalContext":"test message"}'
}

@test "archcore_hook_info opencode: plain text, no JSON wrapper" {
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=opencode sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_info \"test message\"
  "'
  assert_success
  assert_output 'test message'
}

# --- archcore_hook_pretool_info ---

@test "archcore_hook_pretool_info claude-code: exact hookSpecificOutput JSON" {
  # awk in the helper appends a literal \n after each input line.
  run sh -c 'printf "%s" "{}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_pretool_info \"test message\"
  "'
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"test message\n"}}'
}

@test "archcore_hook_pretool_info cursor: exact additional_context JSON" {
  run sh -c 'printf "%s" "{\"conversation_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_pretool_info \"test message\"
  "'
  assert_success
  assert_output '{"additional_context":"test message\n"}'
}

@test "archcore_hook_pretool_info codex: exact hookSpecificOutput JSON" {
  run sh -c 'printf "%s" "{\"turn_id\":\"x\"}" | sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_pretool_info \"test message\"
  "'
  assert_success
  assert_output '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"test message\n"}}'
}

@test "archcore_hook_pretool_info copilot: exact top-level additionalContext JSON" {
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=copilot sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_pretool_info \"test message\"
  "'
  assert_success
  assert_output '{"additionalContext":"test message\n"}'
}

@test "archcore_hook_pretool_info opencode: plain text, no JSON wrapper" {
  run sh -c 'printf "%s" "{}" | ARCHCORE_HOST=opencode sh -c "
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    archcore_hook_pretool_info \"test message\"
  "'
  assert_success
  assert_output 'test message'
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
