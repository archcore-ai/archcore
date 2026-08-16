#!/usr/bin/env bats
# Codex stdout contract for bin/session-start (codex-adapter.spec items 4–7).
#
# Codex tests only whether the TRIMMED stdout starts with '{' or '[' —
# looks_like_json, codex-rs/hooks/src/engine/output_parser.rs. If it does and
# the document does not parse against the event shape, the run is marked Failed
# with "hook returned invalid session start JSON output" and the whole payload
# is discarded. Before 0.7.3 the script printed the CLI hook's JSON and then
# appended plain-text advisories, so any session with an empty .archcore/ or a
# pending CLI update lost its entire Archcore context on this host.
#
# Every test below asserts the invariant through assert_codex_single_document:
# stdout is either exactly one JSON document, or text that does not start with
# '{' or '['. Never a mix. The claude-code / cursor / copilot shapes are pinned
# elsewhere (session-start-goldens.bats, session-start-emit-matrix.bats) and
# must stay byte-identical.

setup() {
  load '../helpers/common'
  common_setup
  command -v jq >/dev/null 2>&1 || skip "jq not installed"

  CLI_DOC='{"hookSpecificOutput":{"additionalContext":"CORPUS: 3 documents","hookEventName":"SessionStart"},"systemMessage":"Archcore v0.7.2 · MCP connected · 3 docs"}'
  export CLI_DOC
}

# The host predicate, mirrored. A test that only greps for its advisory would
# have passed on the defect this file exists to prevent.
assert_codex_single_document() {
  local out="$1"
  local first
  first=$(printf '%s' "$out" | sed -e 's/^[[:space:]]*//' | cut -c1)
  case "$first" in
    '{' | '[')
      printf '%s' "$out" | jq -s -e 'length == 1' > /dev/null \
        || fail "stdout starts like JSON but is not exactly one document: '$out'"
      ;;
    *)
      printf '%s' "$out" | grep -q '^[[:space:]]*[{[]' \
        && fail "plain-text arm must not start with a JSON bracket: '$out'"
      ;;
  esac
}

# A project whose .archcore/ carries one substantial document — no empty-state
# nudge, so only the arm under test fires.
_seeded_project() {
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf -- '---\ntitle: seeded\nstatus: accepted\n---\n\n' > "$workdir/.archcore/seeded.md"
  yes 'lorem ipsum dolor sit amet. ' 2>/dev/null | head -c 300 >> "$workdir/.archcore/seeded.md"
  printf '%s' "$workdir"
}

# .archcore/ exists but is functionally empty (no .md over 200 bytes).
_empty_project() {
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf 'stub\n' > "$workdir/.archcore/stub.md"
  printf '%s' "$workdir"
}

# Mock CLI answering `hooks` with MOCK_HOOKS_OUTPUT and, when
# MOCK_UPDATE_AVAILABLE is set, `update --check` with a pending version.
_mock_hooks_and_update() {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
case "$1" in
  hooks)  cat > /dev/null; printf '%s\n' "$MOCK_HOOKS_OUTPUT"; exit 0 ;;
  update) [ "$2" = "--check" ] && [ -n "$MOCK_UPDATE_AVAILABLE" ] \
            && echo "update available: $MOCK_UPDATE_AVAILABLE"; exit 0 ;;
  --version) echo "v0.7.2"; exit 0 ;;
  *) exit 0 ;;
esac
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

_run_codex_session_start() {
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=codex '${PLUGIN_ROOT}/bin/session-start' 2>'$BATS_TEST_TMPDIR/stderr'"
}

@test "codex: seeded project with no advisory passes the CLI document through byte for byte" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT="$CLI_DOC"
  cd "$(_seeded_project)"

  # Compared through a file, not through "$output": bats strips trailing
  # newlines, so an assertion on $output alone cannot see a stray one — a
  # mutation that appended "\n" to the pass-through arm passed every check.
  local raw="$BATS_TEST_TMPDIR/passthrough.out"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=codex '${PLUGIN_ROOT}/bin/session-start' > '$raw' 2>'$BATS_TEST_TMPDIR/stderr'"
  assert_success
  assert_codex_single_document "$(cat "$raw")"
  printf '%s' "$CLI_DOC" > "$BATS_TEST_TMPDIR/expected.out"
  cmp -s "$raw" "$BATS_TEST_TMPDIR/expected.out" \
    || fail "pass-through is not byte-identical: $(od -c "$raw" | tail -3)"
}

@test "codex: empty .archcore/ splices the nudge into additionalContext, one document" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT="$CLI_DOC"
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.hookEventName == "SessionStart"' > /dev/null \
    || fail "spliced document lost its hookEventName: '$output'"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains(".archcore/ is empty")' > /dev/null \
    || fail "empty-state nudge is not inside additionalContext: '$output'"
}

@test "codex: end-anchor CLI document (additionalContext last) splices the nudge, one document" {
  # Exercises the _ac_cx_end arm of _archcore_flush_codex: the CLI document
  # ends with additionalContext — the shape session-start's own emitter
  # produces. Before this test that splice arm was unreachable by any mock,
  # so a fault there shipped corrupt JSON to Codex with the suite green.
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT='{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"CORPUS: 3 documents"}}'
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.hookEventName == "SessionStart"' > /dev/null \
    || fail "end-anchor splice lost hookEventName: '$output'"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains(".archcore/ is empty")' > /dev/null \
    || fail "empty-state nudge is not inside additionalContext: '$output'"
}

@test "codex: pending CLI update splices the advisory into additionalContext, one document" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT="$CLI_DOC"
  export MOCK_UPDATE_AVAILABLE="v9.9.9"
  cd "$(_seeded_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains("CLI update available")' > /dev/null \
    || fail "update advisory is not inside additionalContext: '$output'"
}

@test "codex: two advisories in one run still leave exactly one document" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT="$CLI_DOC"
  export MOCK_UPDATE_AVAILABLE="v9.9.9"
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains(".archcore/ is empty") and contains("CLI update available")' > /dev/null \
    || fail "expected both advisories in one context value: '$output'"
}

@test "codex: escaped quotes inside additionalContext survive the splice" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT='{"hookSpecificOutput":{"additionalContext":"doc: \"First 60 Seconds\" and a path C:\\\\tmp","hookEventName":"SessionStart"},"systemMessage":"ok"}'
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains("\"First 60 Seconds\"")' > /dev/null \
    || fail "quoted title did not survive the splice: '$output'"
}

@test "codex: systemMessage survives the splice" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT="$CLI_DOC"
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  printf '%s' "$output" | jq -e '.systemMessage | test("MCP connected")' > /dev/null \
    || fail "systemMessage was dropped by the splice: '$output'"
}

@test "codex: unrecognized CLI payload leaves stdout unchanged and routes the advisory to stderr" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT='{"unexpected":1}'
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  [ "$output" = '{"unexpected":1}' ] \
    || fail "unrecognized payload must pass through unchanged, got: '$output'"
  grep -q ".archcore/ is empty" "$BATS_TEST_TMPDIR/stderr" \
    || fail "advisory must fall back to stderr, stderr was: '$(cat "$BATS_TEST_TMPDIR/stderr")'"
}

@test "codex: empty CLI payload wraps the advisories in a document of their own" {
  _mock_hooks_and_update
  export MOCK_HOOKS_OUTPUT=""
  cd "$(_empty_project)"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains(".archcore/ is empty")' > /dev/null \
    || fail "expected the advisory inside a SessionStart document, got: '$output'"
}

# The trap this test guards: every message the script writes opens with
# "[Archcore]", and looks_like_json fires on '[' as well as '{'. Emitted as
# plain text they parse as a malformed JSON array and fail the hook run.
@test "codex: missing .archcore/ emits a JSON document, never a bare [Archcore] line" {
  _mock_hooks_and_update
  cd "$BATS_TEST_TMPDIR"

  _run_codex_session_start
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.hookEventName == "SessionStart"' > /dev/null \
    || fail "expected a SessionStart document, got: '$output'"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains("no .archcore/ directory")' > /dev/null \
    || fail "expected the missing-.archcore nudge inside additionalContext, got: '$output'"
}

@test "codex: CLI missing from PATH still emits a JSON document" {
  rm -f "$MOCK_BIN/archcore"
  cd "$BATS_TEST_TMPDIR"

  run sh -c "printf '%s' '{}' | PATH='/usr/bin:/bin' ARCHCORE_HOST=codex '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_codex_single_document "$output"
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext | contains("CLI not found on PATH")' > /dev/null \
    || fail "expected the install advisory inside additionalContext, got: '$output'"
}
