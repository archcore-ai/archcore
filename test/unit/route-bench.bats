#!/usr/bin/env bats
# Harness integrity uses a fake model process; live model quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export ROUTE_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export ROUTE_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset ROUTE_BENCH_LIMIT ROUTE_BENCH_MODEL BENCH_REPLY_MODE
  printf '1\tFix typo\tno delta\tnull\n2\tAdd feature\tcreates one capability\tcapability\n' > "$ROUTE_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
prompt=$(cat)
case "${BENCH_REPLY_MODE:-}" in
  error) echo 'host unavailable' >&2; exit 7 ;;
  wrong) route=umbrella ;;
  *) case "$prompt" in *'Add feature'*) route=capability ;; *) route=null ;; esac ;;
esac
reply="route: $route (size S) — Δ: none"
if [ "${BENCH_REPLY_MODE:-}" = multiline ]; then reply=$(printf 'Explanation\n%s' "$reply"); fi
if [ "${BENCH_REPLY_MODE:-}" = inline ]; then reply='`'"$reply"'`'; fi
jq -cn --arg result "$reply" '{is_error:false,result:$result}'
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/route-bench.sh"
}

@test "route bench evaluates every fixture and retains raw replies" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
  [ -s "$ROUTE_BENCH_OUTPUT_DIR/1.json" ] || { fail "first model response was discarded"; return 1; }
  [ -s "$ROUTE_BENCH_OUTPUT_DIR/2.json" ] || { fail "second model response was discarded"; return 1; }
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" "$(grep -Fxc -- '--strict-mcp-config' "$BENCH_ARGS")" \
    || { fail "a bench call can load project MCP servers"; return 1; }
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" 2
}

@test "route bench rejects a wrong route instead of reporting green" {
  export BENCH_REPLY_MODE=wrong
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "route bench distinguishes a CLI failure from a routing mismatch" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  assert_output --partial '0 pass, 0 fail, 2 errors'
  grep -Fq 'host unavailable' "$ROUTE_BENCH_OUTPUT_DIR/1.stderr" || { fail "CLI error was discarded"; return 1; }
}

@test "route bench rejects multiline answers despite a matching route line" {
  export BENCH_REPLY_MODE=multiline
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "route bench rejects an empty or malformed fixture corpus" {
  printf '# no rows\n' > "$ROUTE_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "empty corpus invoked the model"; return 1; }
  printf '1\tIncomplete row\n' > "$ROUTE_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "route bench limit stops at the requested fixture and rejects invalid limits" {
  export ROUTE_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
  assert_output --partial '1 pass, 0 fail, 0 errors'
  [ ! -e "$ROUTE_BENCH_OUTPUT_DIR/2.json" ] || { fail "limit did not stop the second call"; return 1; }
  export ROUTE_BENCH_LIMIT=invalid
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "route bench accepts a single inline-code announcement and ignores blank rows" {
  export BENCH_REPLY_MODE=inline
  printf '   \n\t\n' >> "$ROUTE_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
}
