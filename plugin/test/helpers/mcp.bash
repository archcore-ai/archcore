# Real stdio MCP client for Bats. Descriptors 8/9 avoid Bats' TAP descriptor 3.
mcp_start() {
  MCP_BINARY=$(command -v "${ARCHCORE_BIN:-archcore}") || { fail "install Archcore CLI >= 0.8.3 or set ARCHCORE_BIN"; return 1; }
  MCP_ROOT="$BATS_TEST_TMPDIR/project"
  mkdir -p "$MCP_ROOT"
  mkfifo "$BATS_TEST_TMPDIR/mcp-in" "$BATS_TEST_TMPDIR/mcp-out"
  exec 8<>"$BATS_TEST_TMPDIR/mcp-in"
  exec 9<>"$BATS_TEST_TMPDIR/mcp-out"
  mkdir -p "$BATS_TEST_TMPDIR/home"
  (
    cd "$MCP_ROOT" || exit 1
    export HOME="$BATS_TEST_TMPDIR/home"
    export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/config"
    export XDG_DATA_HOME="$BATS_TEST_TMPDIR/data"
    export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache"
    export DO_NOT_TRACK=1 ARCHCORE_TELEMETRY_OPTOUT=1
    exec "$MCP_BINARY" mcp --project "$MCP_ROOT"
  ) <"$BATS_TEST_TMPDIR/mcp-in" >"$BATS_TEST_TMPDIR/mcp-out" 2>"$BATS_TEST_TMPDIR/mcp-stderr" 3>&- 8>&- 9>&- &
  MCP_PID=$!
  MCP_SEQUENCE=0
  mcp_call initialize '{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"plugin-bats","version":"1"}}' || return 1
  printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}' >&8
  mcp_tool init_project '{"sync_mode":"none"}'
}

mcp_stop() {
  if [ -n "${MCP_PID:-}" ]; then
    kill "$MCP_PID" 2>/dev/null || true
    wait "$MCP_PID" 2>/dev/null || true
  fi
  exec 8>&- 9>&-
}

mcp_call() {
  local method="$1" params="$2" message
  MCP_SEQUENCE=$((MCP_SEQUENCE + 1))
  jq -cn --arg method "$method" --argjson params "$params" --argjson id "$MCP_SEQUENCE" \
    '{jsonrpc:"2.0",id:$id,method:$method,params:$params}' >&8 || return 1
  while IFS= read -r -t 15 message <&9; do
    if ! jq -e . >/dev/null 2>&1 <<< "$message"; then
      fail "invalid MCP JSON: $message"; return 1
    fi
    if jq -e --argjson id "$MCP_SEQUENCE" '.id == $id' >/dev/null <<< "$message"; then
      jq -e 'has("result") and (has("error") | not)' >/dev/null <<< "$message" \
        || { fail "MCP protocol error: $message"; return 1; }
      MCP_RESULT=$(jq -c '.result' <<< "$message")
      return 0
    fi
  done
  fail "MCP response timed out: $method; stderr: $(cat "$BATS_TEST_TMPDIR/mcp-stderr")"
  return 1
}

mcp_tool() {
  local name="$1" arguments="$2" expected_error="${3:-false}"
  local params
  params=$(jq -cn --arg name "$name" --argjson arguments "$arguments" '{name:$name,arguments:$arguments}') || return 1
  mcp_call tools/call "$params" || return 1
  jq -e --argjson expected "$expected_error" '(.isError // false) == $expected' >/dev/null <<< "$MCP_RESULT" \
    || { fail "unexpected MCP tool result: $MCP_RESULT"; return 1; }
  MCP_TEXT=$(jq -r '[.content[] | select(.type == "text") | .text] | join("\n")' <<< "$MCP_RESULT")
  if [ "$expected_error" = false ]; then
    MCP_RESULT=$(jq -c . <<< "$MCP_TEXT") || { fail "invalid tool JSON: $MCP_TEXT"; return 1; }
  fi
}

mcp_assert() {
  jq -e "$@" >/dev/null <<< "$MCP_RESULT" || { fail "MCP assertion failed ($*): $MCP_RESULT"; return 1; }
}

mcp_create() {
  local args
  args=$(jq -cn --arg type "$1" --arg filename "$2" '{type:$type,filename:$filename,status:"draft"}')
  mcp_tool create_document "$args" || return 1
  # shellcheck disable=SC2034 # consumed by the calling Bats test
  MCP_PATH=$(jq -r '.path' <<< "$MCP_RESULT")
}

mcp_get() {
  mcp_tool get_document "$(jq -cn --arg path "$1" '{path:$path}')"
}

mcp_edge() {
  mcp_tool add_relation "$(jq -cn --arg source "$1" --arg target "$2" --arg type "$3" '{source:$source,target:$target,type:$type}')" "${4:-false}"
}
