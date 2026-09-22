#!/usr/bin/env bats
# Real CLI end to end: a Cursor afterMCPExecution payload through the shipped
# post-tool-use launcher must reach `archcore hooks cursor post-tool-use` under
# a tool name the CLI recognizes, so an incomplete document earns its Precision
# advisory. The fixture document is created through the same CLI's stdio MCP
# (helpers/mcp.bash); the launcher then resolves `archcore` through PATH, which
# this file pins to the directory of that same binary so no second install can
# answer.
#
# hooks-validation-system.spec, Normative Behavior 5 and 22–27.

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  common_setup
  mcp_start
  export PATH="$(dirname "$MCP_BINARY"):$PATH"
  CLI_VERSION=$("$MCP_BINARY" --version 2>/dev/null)
  # An ADR with one section and a short body: the CLI's precision scan reports
  # the missing sections. A template-complete document would report nothing and
  # the positive case could not distinguish "gate passed" from "gate skipped".
  mcp_create adr jwt-strategy
  INCOMPLETE_BODY=$(printf '## Context\n\nWe need auth.\n')
  local args
  args=$(jq -cn --arg path "$MCP_PATH" --arg content "$INCOMPLETE_BODY" '{path:$path,content:$content}')
  mcp_tool update_document "$args"
  DOC_REL=${MCP_PATH#.archcore/}
}

teardown() {
  mcp_stop
}

# cursor_event <tool_name> <mcp_server_name or ""> → payload on stdout
cursor_event() {
  local tool="$1" server="$2"
  if [ -n "$server" ]; then
    jq -cn --arg tool "$tool" --arg server "$server" --arg path "$DOC_REL" --arg content "$INCOMPLETE_BODY" \
      '{conversation_id:"conv_1",generation_id:"gen_1",hook_event_name:"afterMCPExecution",tool_name:$tool,tool_input:({path:$path,content:$content}|tojson),mcp_server_name:$server,result_json:"{}",duration:12}'
  else
    jq -cn --arg tool "$tool" --arg path "$DOC_REL" --arg content "$INCOMPLETE_BODY" \
      '{conversation_id:"conv_1",generation_id:"gen_1",hook_event_name:"afterMCPExecution",tool_name:$tool,tool_input:({path:$path,content:$content}|tojson),result_json:"{}",duration:12}'
  fi
}

run_launcher() {
  cd "$MCP_ROOT"
  HOME="$BATS_TEST_TMPDIR/home" XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/config" XDG_DATA_HOME="$BATS_TEST_TMPDIR/data" XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache" \
    run "$PLUGIN_ROOT/bin/post-tool-use"
}

@test "control: the CLI itself reports the incomplete ADR only under a qualified tool name" {
  # The dispatch gap this file guards against, reproduced without the launcher:
  # same CLI, same document, same payload but for the tool-name spelling.
  cd "$MCP_ROOT"
  cursor_event update_document archcore > "$BATS_TEST_TMPDIR/bare.json"
  cursor_event mcp__archcore__update_document archcore > "$BATS_TEST_TMPDIR/qualified.json"
  run "$MCP_BINARY" hooks cursor post-tool-use < "$BATS_TEST_TMPDIR/bare.json"
  assert_success
  assert_output ""
  run "$MCP_BINARY" hooks cursor post-tool-use < "$BATS_TEST_TMPDIR/qualified.json"
  assert_success
  [[ "$output" == *"missing section: ## Decision"* ]] || fail "CLI $CLI_VERSION gave no advisory under the qualified name: $output"
}

@test "positive: bare update_document from server archcore → launcher yields the Precision advisory" {
  cursor_event update_document archcore > "$BATS_TEST_TMPDIR/payload.json"
  run_launcher < "$BATS_TEST_TMPDIR/payload.json"
  assert_success
  [ -n "$output" ] || fail "CLI $CLI_VERSION: empty stdout — the launcher passed a bare tool name the CLI does not gate on"
  echo "$output" | jq -e . >/dev/null || fail "CLI $CLI_VERSION: output is not one JSON document: $output"
  echo "$output" | jq -e '.additional_context | contains("[Archcore Precision]") and contains("missing section: ## Decision")' >/dev/null \
    || fail "CLI $CLI_VERSION: advisory missing from additional_context: $output"
}

@test "negative: the same event without mcp_server_name → no advisory (ownership unproven, payload untouched)" {
  cursor_event update_document "" > "$BATS_TEST_TMPDIR/payload.json"
  run_launcher < "$BATS_TEST_TMPDIR/payload.json"
  assert_success
  assert_output ""
}

@test "negative: the same event from a foreign server → no advisory" {
  cursor_event update_document notion > "$BATS_TEST_TMPDIR/payload.json"
  run_launcher < "$BATS_TEST_TMPDIR/payload.json"
  assert_success
  assert_output ""
}

@test "negative: a non-mutating archcore tool from server archcore → qualified but no advisory (CLI gate, not launcher)" {
  cursor_event get_document archcore > "$BATS_TEST_TMPDIR/payload.json"
  run_launcher < "$BATS_TEST_TMPDIR/payload.json"
  assert_success
  assert_output ""
}
