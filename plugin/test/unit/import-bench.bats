#!/usr/bin/env bats
# Harness integrity uses a fake model process; live model quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export IMPORT_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export IMPORT_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset IMPORT_BENCH_LIMIT IMPORT_BENCH_MODEL BENCH_REPLY_MODE
  printf '1\t-\tpackage.json, CLAUDE.md 4 KB\tplain\tS\tCLAUDE.md\tmine\n2\timport\tseeded repo, open plan\tresume\t-\t-\t-\n' > "$IMPORT_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
prompt=$(cat)
case "${BENCH_REPLY_MODE:-}" in
  error) echo 'host unavailable' >&2; exit 7 ;;
  wrongtier) reply='init: plain L mine' ;;
  *) case "$prompt" in *'open plan'*) reply='init: resume none none' ;; *) reply='init: plain S mine' ;; esac ;;
esac
if [ "${BENCH_REPLY_MODE:-}" = multiline ]; then reply=$(printf 'Because\n%s' "$reply"); fi
jq -cn --arg result "$reply" '{is_error:false,result:$result}'
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/import-bench.sh"
}

@test "import bench passes a matching route, tier, and verdict, and skips a check on -" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" "$(grep -Fxc -- '--strict-mcp-config' "$BENCH_ARGS")"
  grep -Fq 'Size tiers and waves' "$IMPORT_BENCH_OUTPUT_DIR/1.prompt" || { fail "prompt lacks the size-tier excerpt of the import track"; return 1; }
  grep -Fq 'Triage verdicts' "$IMPORT_BENCH_OUTPUT_DIR/1.prompt" || { fail "prompt lacks the source catalog"; return 1; }
}

@test "import bench reads - as empty arguments" {
  run sh "$BENCH"
  assert_success
  # The arguments line of fixture 1 is empty: the header is followed by the next header.
  awk '/^--- ARGUMENTS/ { getline; print; exit }' "$IMPORT_BENCH_OUTPUT_DIR/1.prompt" | grep -qx '' \
    || { fail "fixture 1 arguments were not emptied"; return 1; }
}

@test "import bench fails a wrong tier" {
  export BENCH_REPLY_MODE=wrongtier
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "import bench distinguishes a CLI failure and rejects multiline replies" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  export BENCH_REPLY_MODE=multiline
  run sh "$BENCH"
  assert_equal "$status" 1
}

@test "import bench rejects a fixture with an unknown route, tier, or verdict" {
  printf '1\t-\tx\tseed\tS\t-\t-\n' > "$IMPORT_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  printf '1\t-\tx\tplain\tXL\t-\t-\n' > "$IMPORT_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "an invalid corpus invoked the model"; return 1; }
}

@test "the shipped import fixtures are well-formed" {
  unset IMPORT_BENCH_FIXTURES
  export IMPORT_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
}
