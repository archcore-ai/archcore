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
  run_normalizer '{"sessionId":"s1","timestamp":1784816794176,"cwd":"/work","toolName":"create","toolArgs":"{\"path\":\"/work/x.md\"}"}'
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

@test "claude-code: duplicated key at two depths — first occurrence wins" {
  # Pins the first-occurrence contract of _archcore_json_val. A PostToolUse
  # blob carries the same key twice at different depths: tool_input.path
  # (what the host sends for the guards) comes first, and the MCP server's
  # tool_response echoes a resolved absolute path under the same key later.
  # Extraction must take the FIRST occurrence — last-occurrence extraction
  # would hand the guards the response echo instead of the tool input.
  run_normalizer '{"session_id":"s1","transcript_path":"/Users/dev/.claude/projects/work/t.jsonl","cwd":"/work","hook_event_name":"PostToolUse","tool_name":"mcp__archcore__update_document","tool_input":{"path":"auth/jwt.adr.md","content":"# updated"},"tool_response":{"structuredContent":{"path":"/work/.archcore/auth/jwt.adr.md","status":"updated"}}}'
  assert_success
  # Expected: DOC=auth/jwt.adr.md (first occurrence, from tool_input).
  # DOC=/work/.archcore/auth/jwt.adr.md means normalize-stdin.sh picked the
  # LAST occurrence (tool_response echo) — first-occurrence contract broken.
  assert_line "DOC=auth/jwt.adr.md"
  refute_line "DOC=/work/.archcore/auth/jwt.adr.md"
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

@test "cursor: a non-registered MCP event leaves the bare tool name alone" {
  # cursor.hooks.json registers afterMCPExecution and nothing else, so an
  # event the config never asks for cannot reach a guard — and the normalizer
  # does not pretend otherwise. If beforeMCPExecution is ever registered, the
  # case arm in normalize-stdin.sh must list it and this test flips.
  run_normalizer '{"conversation_id":"x","hook_event_name":"beforeMCPExecution","tool_name":"update_document"}'
  assert_success
  assert_line "TOOL=update_document"
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
  run_normalizer '{"sessionId":"s1","toolName":"create","toolArgs":"{\"path\":\"/work/.archcore/my.rule.md\"}"}'
  assert_success
  assert_line "TOOL=create"
}

@test "copilot: extracts native create path from escaped toolArgs" {
  run_normalizer '{"sessionId":"s1","toolName":"create","toolArgs":"{\"path\":\"/work/.archcore/my.rule.md\",\"file_text\":\"x\"}"}'
  assert_success
  assert_line "FILE=/work/.archcore/my.rule.md"
}

@test "copilot: extracts native edit path from escaped toolArgs" {
  run_normalizer '{"sessionId":"s1","toolName":"edit","toolArgs":"{\"path\":\"/work/src/app.py\",\"old_str\":\"a\",\"new_str\":\"b\"}"}'
  assert_success
  assert_line "FILE=/work/src/app.py"
}

@test "copilot: normalizes native MCP tool name and extracts doc path" {
  run_normalizer '{"sessionId":"s1","toolName":"archcore-update_document","toolArgs":"{\"path\":\".archcore/copilot-hook-probe.doc.md\"}"}'
  assert_success
  assert_line "TOOL=mcp__archcore__update_document"
  assert_line "DOC=.archcore/copilot-hook-probe.doc.md"
}

# --- Canonical MCP tool naming (all three registrations) ---
#
# The same archcore MCP server reaches the model under three names depending on
# how it was registered. Downstream scripts (validate-archcore, check-precision,
# check-cascade) gate on the project naming alone, so a payload that arrives
# under either other name must be folded here — otherwise those hooks fire and
# silently do nothing, which is invisible until a user's document skips
# validation entirely. Hook MATCHERS carry both namings too, but that is a
# different layer: matchers decide whether a script runs, this decides what it
# sees.

@test "claude-code: plugin-bundled MCP naming folds to the canonical name" {
  run_normalizer '{"tool_name":"mcp__plugin_archcore_archcore__create_document","tool_input":{"path":".archcore/x.adr.md"}}'
  assert_success
  assert_line "TOOL=mcp__archcore__create_document"
  assert_line "DOC=.archcore/x.adr.md"
}

@test "codex: plugin-bundled MCP naming folds to the canonical name" {
  run_normalizer '{"turn_id":"t1","tool_name":"mcp__plugin_archcore_archcore__update_document","tool_input":{"path":".archcore/y.spec.md"}}'
  assert_success
  assert_line "TOOL=mcp__archcore__update_document"
}

@test "opencode: plugin-bundled MCP naming folds to the canonical name" {
  run sh -c "printf '%s' '{\"tool_name\":\"mcp__plugin_archcore_archcore__add_relation\"}' | ARCHCORE_HOST=opencode sh -c '
    . \"$PLUGIN_ROOT/bin/lib/normalize-stdin.sh\"
    echo \"TOOL=\$ARCHCORE_TOOL_NAME\"'"
  assert_success
  assert_line "TOOL=mcp__archcore__add_relation"
}

@test "canonical project naming passes through untouched" {
  run_normalizer '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":".archcore/z.adr.md"}}'
  assert_success
  assert_line "TOOL=mcp__archcore__create_document"
}

@test "a foreign MCP server is never rewritten" {
  # Only the archcore server's own namings fold. A tool from some other plugin
  # that happens to share the prefix shape must reach the scripts verbatim, or
  # the guards would start claiming authority over documents they do not own.
  run_normalizer '{"tool_name":"mcp__plugin_other_other__create_document"}'
  assert_success
  assert_line "TOOL=mcp__plugin_other_other__create_document"
}

@test "copilot: a non-MCP native tool name is not mistaken for a server prefix" {
  # "archcore-" folding must not swallow ordinary Copilot file tools.
  run_normalizer '{"sessionId":"s1","toolName":"create","toolArgs":"{\"path\":\"/work/src/a.ts\"}"}'
  assert_success
  assert_line "TOOL=create"
  assert_line "FILE=/work/src/a.ts"
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
  # Copilot deny contract: stdout JSON + exit 0. Not because exit 2 fails to
  # block — every non-zero exit denies there — but because permissionDecisionReason
  # is the only way the deny carries its reason (copilot-adapter-design.adr).
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
