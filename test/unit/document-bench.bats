#!/usr/bin/env bats
# Harness integrity uses a fake model process; live model quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export DOCUMENT_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export DOCUMENT_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset DOCUMENT_BENCH_LIMIT DOCUMENT_BENCH_MODEL BENCH_REPLY_MODE
  printf '1\twe decided to use Postgres\tno adr\tdecision\tadr\n2\tcapture how payments work\tcode found\tcode\t-\n' > "$DOCUMENT_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
prompt=$(cat)
case "${BENCH_REPLY_MODE:-}" in
  error) echo 'host unavailable' >&2; exit 7 ;;
  wrongtype) reply='entry: decision rfc' ;;
  *) case "$prompt" in *'capture how payments work'*) reply='entry: code spec' ;; *) reply='entry: decision adr' ;; esac ;;
esac
if [ "${BENCH_REPLY_MODE:-}" = multiline ]; then reply=$(printf 'Because\n%s' "$reply"); fi
jq -cn --arg result "$reply" '{is_error:false,result:$result}'
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/document-bench.sh"
}

@test "document bench passes a matching mode and type, and skips the type check on -" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" "$(grep -Fxc -- '--strict-mcp-config' "$BENCH_ARGS")"
  grep -Fq 'Entry terms' "$DOCUMENT_BENCH_OUTPUT_DIR/1.prompt" || { fail "prompt lacks the gate-contract entry terms"; return 1; }
}

@test "document bench fails a wrong type" {
  export BENCH_REPLY_MODE=wrongtype
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "document bench distinguishes a CLI failure and rejects multiline replies" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  export BENCH_REPLY_MODE=multiline
  run sh "$BENCH"
  assert_equal "$status" 1
}

@test "document bench rejects a fixture with an unknown mode or type" {
  printf '1\tx\ty\tdescribe\tadr\n' > "$DOCUMENT_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  printf '1\tx\ty\tdecision\tjourney\n' > "$DOCUMENT_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "an invalid corpus invoked the model"; return 1; }
}
