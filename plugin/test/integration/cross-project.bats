#!/usr/bin/env bats
# Real CLI end to end for explicit-target-project-per-call.adr: a session in
# project A reaches project B through the project argument of the stdio MCP
# server, and the shipped launchers guard, check, and recap per project.
#
# Needs an Archcore CLI from 0.15.0: set ARCHCORE_BIN to a build of the
# current tree when the installed CLI is older.

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  common_setup
  mcp_start
  # bin/cli-gte and the launchers call `archcore` by name; CI builds `archcore-source`.
  mkdir -p "$BATS_TEST_TMPDIR/.bin"
  ln -sf "$MCP_BINARY" "$BATS_TEST_TMPDIR/.bin/archcore"
  export PATH="$BATS_TEST_TMPDIR/.bin:$PATH"
  if [ "$("$PLUGIN_ROOT/bin/cli-gte" 0.15.0)" != "yes" ]; then
    skip "needs archcore CLI >= 0.15.0 (set ARCHCORE_BIN)"
  fi
  # Once a release outruns the source build (0.15.0-dev < v0.15.0), the
  # launcher's update advisory is due in a fresh data dir and lands as plain
  # text after the JSON recap, which the recap assertions parse with jq.
  # Stamp it as shown, the way test/unit/session-start.bats does.
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  date +%s > "$CLAUDE_PLUGIN_DATA/archcore/last-update-advisory"
  WORKSPACE=$(cd "$BATS_TEST_TMPDIR" && pwd -P)
  A=$(cd "$MCP_ROOT" && pwd -P)
  B="$WORKSPACE/b"
  mkdir -p "$B/src"
}

teardown() {
  mcp_stop
}

@test "MCP: init_project and create_document reach project B from a session in A" {
  mcp_tool init_project "$(jq -cn --arg p "$B" '{project:$p,sync_mode:"none"}')"
  mcp_assert --arg p "$B" '.project == $p and .already_initialized == false'
  [ -d "$B/.archcore" ]

  mcp_tool create_document "$(jq -cn --arg p "$B" '{project:$p,type:"adr",filename:"in-b",status:"draft"}')"
  mcp_assert --arg p "$B" '.project == $p and .path == ".archcore/in-b.adr.md"'
  [ -f "$B/.archcore/in-b.adr.md" ]
  [ ! -f "$A/.archcore/in-b.adr.md" ]

  # A call without project answers as before: the session root as the server spells it.
  mcp_tool list_documents '{}'
  mcp_assert --arg a "$MCP_ROOT" '.project == $a and (.documents | length) == 0'

  # project naming the session directory through another spelling is one root.
  mcp_tool list_documents "$(jq -cn --arg p "$A" '{project:$p}')"
  mcp_assert --arg a "$MCP_ROOT" '.project == $a and (.documents | length) == 0'
}

@test "MCP: a project without .archcore/ is refused and left untouched" {
  mcp_tool create_document "$(jq -cn --arg p "$B" '{project:$p,type:"adr",filename:"x"}')" true
  case "$MCP_TEXT" in
    *init_project*) ;;
    *) fail "the refusal does not name init_project: $MCP_TEXT" ;;
  esac
  case "$MCP_TEXT" in
    *"$WORKSPACE"*) fail "the refusal embeds an absolute path: $MCP_TEXT" ;;
  esac
  [ ! -d "$B/.archcore" ]
}

@test "launcher: a direct write into B's .archcore/ from a session in A is denied and names project B" {
  mkdir -p "$B/.archcore"
  payload=$(jq -cn --arg cwd "$A" --arg f "$B/.archcore/x.adr.md" '{tool_name:"Write",cwd:$cwd,tool_input:{file_path:$f}}')
  run sh -c "printf '%s' '$payload' | '$PLUGIN_ROOT/bin/pre-tool-use'"
  [ "$status" -eq 2 ] || fail "exit $status, want the deny; output: $output"
  assert_output --partial "update_document"
  assert_output --partial "project=\"$B\""
}

@test "launcher: a source edit in B is allowed and names project B once per session" {
  mkdir -p "$B/.archcore"
  payload=$(jq -cn --arg cwd "$A" --arg f "$B/src/x.go" --arg s "$BATS_TEST_NAME" '{tool_name:"Write",cwd:$cwd,session_id:$s,tool_input:{file_path:$f}}')
  run sh -c "printf '%s' '$payload' | '$PLUGIN_ROOT/bin/pre-tool-use'"
  assert_success
  assert_output --partial "[Archcore Project]"
  run sh -c "printf '%s' '$payload' | '$PLUGIN_ROOT/bin/pre-tool-use'"
  assert_success
  refute_output --partial "[Archcore Project]"
}

@test "launcher: post-tool-use validates the document in the project the call named" {
  mcp_tool init_project "$(jq -cn --arg p "$B" '{project:$p,sync_mode:"none"}')"
  mcp_tool create_document "$(jq -cn --arg p "$B" '{project:$p,type:"adr",filename:"thin",status:"draft"}')"
  mcp_tool update_document "$(jq -cn --arg p "$B" '{project:$p,path:".archcore/thin.adr.md",content:"## Context\n\nShort.\n"}')"
  payload=$(jq -cn --arg cwd "$A" --arg p "$B" '{tool_name:"mcp__plugin_archcore_archcore__update_document",cwd:$cwd,tool_input:{project:$p,path:".archcore/thin.adr.md"}}')
  run sh -c "printf '%s' '$payload' | '$PLUGIN_ROOT/bin/post-tool-use'"
  assert_success
  assert_output --partial "thin.adr.md"
}

@test "launcher: session-start in the workspace parent lists A and B and advises no init" {
  mkdir -p "$B/.archcore"
  cd "$WORKSPACE"
  run sh -c "printf '%s' '{\"session_id\":\"$BATS_TEST_NAME\",\"source\":\"startup\"}' | '$PLUGIN_ROOT/bin/session-start'"
  assert_success
  recap=$(printf '%s' "$output" | jq -r '.hookSpecificOutput.additionalContext')
  case "$recap" in
    *"PROJECTS BELOW: "*"$B"*) ;;
    *) fail "the recap does not list B: $recap" ;;
  esac
  case "$recap" in
    *"PROJECTS BELOW: "*"$A"*) ;;
    *) fail "the recap does not list A: $recap" ;;
  esac
  case "$recap" in
    *init_project*) fail "the recap advises init_project: $recap" ;;
  esac
}
