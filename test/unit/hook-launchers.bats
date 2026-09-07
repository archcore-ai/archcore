#!/usr/bin/env bats
# Unit tests for bin/pre-tool-use and bin/post-tool-use.
#
# Since v0.7.0 the hook POLICY (write guard, code alignment, validation,
# cascade, precision) lives in the archcore CLI; these launchers carry only the
# host glue. What must hold, per launcher:
#
#   1. fail OPEN silently when the CLI is absent or older than 0.7.0 — a
#      pre-0.7 CLI has no pre/post-tool-use leaves, and its usage error would
#      read as a deny on Copilot (every non-zero exit denies there);
#   2. map the shell host id "codex" to the CLI agent id "codex-cli";
#   3. hand stdin through byte-for-byte and propagate stdout and the exit code
#      unchanged — exit 2 + stderr IS the deny protocol on claude-compat hosts;
#   4. refuse to run from a plugin-cache cwd even when the CLI is present;
#   5. post-tool-use only, Cursor only: qualify the bare archcore tool name of an
#      afterMCPExecution event whose mcp_server_name is "archcore", touching
#      that one value and nothing else; every ambiguous shape passes through
#      unchanged (hooks-validation-system.spec, Normative Behavior 5 and 22–27).

setup() {
  load '../helpers/common'
  common_setup
}

# make_cli <version>: install a mock archcore CLI into MOCK_BIN.
# --version prints the given version; any other invocation records its args and
# stdin, then runs $BATS_TEST_TMPDIR/cli-behavior when present (exit 0 otherwise).
make_cli() {
  local version="$1"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
if [ "\$1" = "--version" ]; then
  echo "archcore v${version}"
  exit 0
fi
printf '%s ' "\$@" > "$BATS_TEST_TMPDIR/cli-args"
cat > "$BATS_TEST_TMPDIR/cli-stdin"
if [ -f "$BATS_TEST_TMPDIR/cli-behavior" ]; then
  . "$BATS_TEST_TMPDIR/cli-behavior"
fi
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

CLAUDE_PAYLOAD='{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"/tmp/f.md"}}'
CODEX_PAYLOAD='{"turn_id":"t1","tool_name":"Write","tool_input":{"file_path":"/tmp/f.md"}}'

# --- fail-open gate ----------------------------------------------------------

@test "pre-tool-use: no CLI on PATH → silent allow (exit 0, no output)" {
  run env PATH="$MOCK_BIN:/usr/bin:/bin" "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  assert_output ""
}

@test "pre-tool-use: CLI older than 0.7.0 → silent allow, hook leaf never invoked" {
  make_cli "0.6.9"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  assert_output ""
  [ ! -f "$BATS_TEST_TMPDIR/cli-args" ]
}

@test "post-tool-use: CLI older than 0.7.0 → silent allow, hook leaf never invoked" {
  make_cli "0.6.9"
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  assert_output ""
  [ ! -f "$BATS_TEST_TMPDIR/cli-args" ]
}

# --- delegation: args, stdin, stdout, exit code ------------------------------

@test "pre-tool-use: claude-code payload → 'hooks claude-code pre-tool-use', stdin passed byte-for-byte" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks claude-code pre-tool-use "
  # normalize-stdin captures stdin via command substitution, which strips the
  # trailing newline <<< appended; the JSON body itself must survive unchanged.
  printf '%s' "$CLAUDE_PAYLOAD" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: claude-code payload → 'hooks claude-code post-tool-use'" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks claude-code post-tool-use "
}

@test "pre-tool-use: CLI stdout reaches the host unchanged" {
  make_cli "0.7.0"
  printf 'printf %s "{\\"hookSpecificOutput\\":{}}"\nexit 0\n' > "$BATS_TEST_TMPDIR/cli-behavior"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  assert_output '{"hookSpecificOutput":{}}'
}

@test "pre-tool-use: CLI deny (exit 2 + stderr) propagates unchanged" {
  make_cli "0.7.0"
  printf 'echo "Direct writes are not allowed" >&2\nexit 2\n' > "$BATS_TEST_TMPDIR/cli-behavior"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
  [ "$status" -eq 2 ]
  [[ "$output" == *"Direct writes are not allowed"* ]]
}

# --- host mapping ------------------------------------------------------------

@test "pre-tool-use: codex payload (turn_id) → CLI agent id codex-cli" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CODEX_PAYLOAD"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks codex-cli pre-tool-use "
}

@test "post-tool-use: ARCHCORE_HOST=copilot env pin → CLI host copilot" {
  make_cli "0.7.0"
  ARCHCORE_HOST=copilot run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks copilot post-tool-use "
}

@test "pre-tool-use: cursor payload (conversation_id) → CLI host cursor" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/pre-tool-use" <<< '{"conversation_id":"c1","hook_event_name":"preToolUse","tool_name":"Write"}'
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks cursor pre-tool-use "
}

# --- plugin-cache cwd guard --------------------------------------------------

@test "pre-tool-use: each plugin-cache fragment triggers the guard on its own" {
  # One directory per fragment, shaped so it matches exactly ONE pattern —
  # a path like .claude/plugins/cache/ matches two fragments at once and
  # would keep passing when either one is deleted.
  make_cli "0.7.0"
  local frag
  for frag in ".cursor/plugins" ".claude/plugins" ".codex/plugins" ".copilot/installed-plugins" "plugins/cache"; do
    rm -f "$BATS_TEST_TMPDIR/cli-args"
    mkdir -p "$BATS_TEST_TMPDIR/frag/$frag/inner"
    cd "$BATS_TEST_TMPDIR/frag/$frag/inner"
    run "$PLUGIN_ROOT/bin/pre-tool-use" <<< "$CLAUDE_PAYLOAD"
    assert_success
    assert_output ""
    [ ! -f "$BATS_TEST_TMPDIR/cli-args" ] || fail "guard missed fragment: $frag"
    cd "$BATS_TEST_TMPDIR"
    rm -rf "$BATS_TEST_TMPDIR/frag"
  done
}

@test "post-tool-use: cwd inside an unpacked plugin install (manifest walk) → silent exit 0" {
  make_cli "0.7.0"
  mkdir -p "$BATS_TEST_TMPDIR/install/.claude-plugin" "$BATS_TEST_TMPDIR/install/skills/init"
  printf '{}' > "$BATS_TEST_TMPDIR/install/.claude-plugin/plugin.json"
  cd "$BATS_TEST_TMPDIR/install/skills/init"
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$CLAUDE_PAYLOAD"
  assert_success
  assert_output ""
  [ ! -f "$BATS_TEST_TMPDIR/cli-args" ]
}

# --- Cursor afterMCPExecution tool-name qualification -------------------------

# cursor_payload <tool_name> <server-json-fragment> [<tool_input-json-string>]
# Builds one afterMCPExecution payload. The fragment is inserted verbatim so a
# test can omit mcp_server_name entirely.
CURSOR_TOOL_INPUT='"{\"path\":\"auth/jwt.adr.md\",\"content\":\"# Updated\"}"'
cursor_payload() {
  local tool="$1" server="$2" input="${3:-$CURSOR_TOOL_INPUT}"
  printf '{"conversation_id":"c1","hook_event_name":"afterMCPExecution","tool_name":"%s","tool_input":%s%s,"result_json":"{}","duration":12}' \
    "$tool" "$input" "$server"
}

@test "post-tool-use: cursor afterMCPExecution from server archcore → bare tool_name qualified, rest byte-identical" {
  make_cli "0.7.0"
  local payload expected
  payload=$(cursor_payload update_document ',"mcp_server_name":"archcore"')
  expected=${payload/\"tool_name\":\"update_document\"/\"tool_name\":\"mcp__archcore__update_document\"}
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks cursor post-tool-use "
  printf '%s' "$expected" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
  # the qualified stdin is still one JSON object with the same fields
  jq -e '.tool_name == "mcp__archcore__update_document" and .mcp_server_name == "archcore" and (.tool_input | fromjson | .path) == "auth/jwt.adr.md"' \
    "$BATS_TEST_TMPDIR/cli-stdin" >/dev/null
}

@test "post-tool-use: cursor fixture afterMCPExecution-update-archcore.json → qualified" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" < "$FIXTURES/stdin/cursor/afterMCPExecution-update-archcore.json"
  assert_success
  jq -e '.tool_name == "mcp__archcore__update_document"' "$BATS_TEST_TMPDIR/cli-stdin" >/dev/null
}

@test "post-tool-use: cursor afterMCPExecution from a foreign server → payload unchanged" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" < "$FIXTURES/stdin/cursor/afterMCPExecution-update-foreign.json"
  assert_success
  diff "$FIXTURES/stdin/cursor/afterMCPExecution-update-foreign.json" "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor afterMCPExecution without mcp_server_name → payload unchanged (ownership unproven)" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" < "$FIXTURES/stdin/cursor/mcp-update.json"
  assert_success
  printf '%s' "$(cat "$FIXTURES/stdin/cursor/mcp-update.json")" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor afterMCPExecution, server archcore, tool not registered by archcore → unchanged" {
  make_cli "0.7.0"
  local payload
  payload=$(cursor_payload deploy_document ',"mcp_server_name":"archcore"')
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor afterMCPExecution with an already qualified tool_name → unchanged" {
  make_cli "0.7.0"
  local payload
  payload=$(cursor_payload mcp__archcore__update_document ',"mcp_server_name":"archcore"')
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor afterMCPExecution, server name that merely contains archcore → unchanged" {
  make_cli "0.7.0"
  local payload
  payload=$(cursor_payload update_document ',"mcp_server_name":"archcore_docs"')
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor qualification never rewrites a tool_name inside nested JSON strings" {
  make_cli "0.7.0"
  # Document content that quotes the key: inside tool_input it is an escaped
  # string, so it must survive byte for byte while the top-level key changes.
  local input='"{\"path\":\"auth/jwt.adr.md\",\"content\":\"{\\\"tool_name\\\":\\\"update_document\\\"} and \\\"tool_name\\\": \\\"update_document\\\"\"}"'
  local payload expected
  payload=$(cursor_payload update_document ',"mcp_server_name":"archcore"' "$input")
  expected=${payload/\"tool_name\":\"update_document\"/\"tool_name\":\"mcp__archcore__update_document\"}
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  printf '%s' "$expected" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
  # exactly one qualified spelling: the top-level key
  [ "$(grep -o 'mcp__archcore__update_document' "$BATS_TEST_TMPDIR/cli-stdin" | wc -l | tr -d ' ')" -eq 1 ]
  # the nested content still holds the bare spelling, untouched
  jq -e '.tool_input | fromjson | .content | contains("\"tool_name\":\"update_document\"")' "$BATS_TEST_TMPDIR/cli-stdin" >/dev/null
}

@test "post-tool-use: cursor afterMCPExecution with two real tool_name keys → unchanged (ambiguous shape)" {
  make_cli "0.7.0"
  # tool_input as an OBJECT carrying its own tool_name key is not the
  # documented event shape; the launcher must not guess which key to rewrite.
  local payload
  payload=$(cursor_payload update_document ',"mcp_server_name":"archcore"' '{"path":"auth/jwt.adr.md","tool_name":"update_document"}')
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: malformed cursor payload → passed through unchanged, no JSON invented" {
  make_cli "0.7.0"
  local payload='{"conversation_id":"c1","hook_event_name":"afterMCPExecution","tool_name":"update_document","mcp_server_name":"archcore"'
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  # Truncated JSON: the launcher must not turn it into something the CLI would
  # act on. What reaches the CLI is either the original bytes or, at most, the
  # same bytes with the one value qualified — never a different document.
  run jq -e . "$BATS_TEST_TMPDIR/cli-stdin"
  assert_failure
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin" \
    || printf '%s' "${payload/\"tool_name\":\"update_document\"/\"tool_name\":\"mcp__archcore__update_document\"}" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: cursor preToolUse payload → unchanged (qualification is post-event only)" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/post-tool-use" < "$FIXTURES/stdin/cursor/write-archcore.json"
  assert_success
  printf '%s' "$(cat "$FIXTURES/stdin/cursor/write-archcore.json")" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "post-tool-use: claude-code payload carrying mcp_server_name → unchanged (cursor-only branch)" {
  make_cli "0.7.0"
  local payload='{"session_id":"s1","hook_event_name":"afterMCPExecution","tool_name":"update_document","tool_input":{"path":"auth/jwt.adr.md"},"mcp_server_name":"archcore"}'
  run "$PLUGIN_ROOT/bin/post-tool-use" <<< "$payload"
  assert_success
  run cat "$BATS_TEST_TMPDIR/cli-args"
  assert_output "hooks claude-code post-tool-use "
  printf '%s' "$payload" | diff - "$BATS_TEST_TMPDIR/cli-stdin"
}

@test "pre-tool-use: cursor afterMCPExecution from server archcore → unchanged (launcher stays transparent)" {
  make_cli "0.7.0"
  run "$PLUGIN_ROOT/bin/pre-tool-use" < "$FIXTURES/stdin/cursor/afterMCPExecution-update-archcore.json"
  assert_success
  diff "$FIXTURES/stdin/cursor/afterMCPExecution-update-archcore.json" "$BATS_TEST_TMPDIR/cli-stdin"
}
