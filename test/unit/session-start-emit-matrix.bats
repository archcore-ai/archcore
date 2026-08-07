#!/usr/bin/env bats
# Emit-shape matrix for bin/session-start's own _archcore_emit_info and the
# copilot output-channel pins.
#
# _archcore_emit_info is a LOCAL sibling of lib/normalize-stdin.sh's
# archcore_hook_info with a DIFFERENT shape table (codex/cursor get plain text
# here, not a JSON wrapper), so it cannot enroll in
# output-helpers-matrix.bats' expected_shape_ok — this file is its dedicated
# matrix (driven through the missing-.archcore arm, which routes through the
# function on every host).
#
# The channel pins document the copilot stdout contract: exactly one JSON
# document (the CLI hook's), any advisory/staleness lines trail as plain
# text. Host tolerance of trailing text is a live-probe item
# (host-probe-protocol.spec.md); these tests pin the plugin-side shape so it
# can only change deliberately.

setup() {
  load '../helpers/common'
  common_setup
}

# --- _archcore_emit_info shape matrix (missing-.archcore arm) -----------------

@test "emit matrix: claude-code wraps in hookSpecificOutput/SessionStart JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  printf '%s' "$output" | jq -e '.hookSpecificOutput.hookEventName == "SessionStart"' > /dev/null \
    || fail "claude-code emit is not the SessionStart hookSpecificOutput wrapper: '$output'"
}

@test "emit matrix: copilot emits bare top-level additionalContext JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  printf '%s' "$output" | jq -e '(.additionalContext | type == "string") and (has("hookSpecificOutput") | not)' > /dev/null \
    || fail "copilot emit is not the bare additionalContext document: '$output'"
}

@test "emit matrix: cursor, codex, opencode emit plain text (no JSON document)" {
  mock_archcore ""
  local host stdin
  for host in cursor codex opencode; do
    case "$host" in
      cursor) stdin='{"conversation_id":"x"}' ;;
      codex) stdin='{"turn_id":"x"}' ;;
      *) stdin='{}' ;;
    esac
    cd "$BATS_TEST_TMPDIR"
    if [ "$host" = opencode ]; then
      run sh -c "printf '%s' '$stdin' | ARCHCORE_HOST=opencode '${PLUGIN_ROOT}/bin/session-start'"
    else
      run sh -c "printf '%s' '$stdin' | '${PLUGIN_ROOT}/bin/session-start'"
    fi
    assert_success
    [ -n "$output" ] || fail "host '$host' fell through the emit matrix with empty output"
    case "$output" in
      "{"*) fail "host '$host' emitted a JSON document where plain text is the contract: '$output'" ;;
    esac
  done
}

@test "emit matrix: every host produces non-empty output on the nudge arm" {
  # The silent-fall-through class: a host missing from the emit case must
  # never mean a silent no-op nudge.
  mock_archcore ""
  local host
  for host in claude-code cursor codex copilot opencode; do
    cd "$BATS_TEST_TMPDIR"
    run sh -c "printf '%s' '{}' | ARCHCORE_HOST=$host '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    [ -n "$output" ] || fail "host '$host': empty output on the missing-.archcore arm"
  done
}

# --- printf format-string hygiene ---------------------------------------------

@test "emit matrix: no printf in session-start uses a variable as its format string" {
  # printf "$msg" (or a $var embedded anywhere in a double-quoted format)
  # would turn any % in a message into a format directive and any \n into a
  # literal newline — the injection class the emit helpers avoid by keeping
  # formats single-quoted literals and passing text as arguments. Every
  # legitimate format in this script IS single-quoted, so the invariant is
  # simply: printf's first argument never starts with a double quote or a
  # variable (escaped-quote tricks inside a double-quoted format cannot
  # dodge this, unlike a [^"]*\$ scan).
  run grep -nE 'printf[[:space:]]+("|\$)' "$PLUGIN_ROOT/bin/session-start"
  [ "$status" -ne 0 ] || fail "double-quoted or variable printf format string: $output"
}

# --- copilot single-document contract -------------------------------------------
#
# Copilot's documented stdout rule (docs.github.com/en/copilot/reference/
# hooks-reference): a line that is a single complete JSON object with
# "type":"progress" is consumed as a progress event and REMOVED from the
# output stream. "Every other line — blank lines, plain text, and JSON objects
# that are not progress messages — is preserved verbatim." When the hook exits
# the preserved lines are concatenated, trimmed, and parsed with a SINGLE
# JSON.parse call; if that fails, "the hook is treated as producing no output".
#
# So trailing plain text after the CLI hook's JSON does not degrade to "context
# plus a note" — it discards the whole payload, the ~9 KB of Archcore context
# included. Every advisory on this host therefore has to arrive INSIDE the one
# document, and these tests assert exactly that: strip the progress lines, and
# what remains must parse as one document, with every message present in it.

# Applies the host rule to "$output" and prints the resulting document.
# Fails the test if the remainder is not exactly one JSON document.
copilot_document() {
  local stripped line
  stripped=""
  while IFS= read -r line; do
    if printf '%s' "$line" | jq -e 'type == "object" and .type == "progress"' > /dev/null 2>&1; then
      continue
    fi
    stripped="${stripped}${line}
"
  done <<< "$output"

  # `jq -s` slurps a stream into an array; length 1 is the single-JSON.parse
  # equivalent. A stray plain-text line makes it a parse error, and a second
  # JSON object makes it length 2 — both are the failure this pins.
  printf '%s' "$stripped" | jq -e -s 'length == 1' > /dev/null 2>&1 \
    || fail "stdout is not one JSON document after stripping progress lines:
--- raw ---
$output
--- after stripping progress lines ---
$stripped"
  printf '%s' "$stripped" | jq -s -r '.[0].additionalContext // ""'
}

# The staleness advisory moved into the CLI's session-start recap in v0.7.0,
# so it arrives inside the CLI hook's own document — the buffering path this
# suite guards is exercised by the empty-state-nudge test below.

@test "copilot single doc: the empty-state nudge travels inside the document" {
  export MOCK_HOOKS_OUTPUT='{"additionalContext":"ctx"}'
  mock_archcore_multi
  local d="$BATS_TEST_TMPDIR/empty-channel"
  mkdir -p "$d/.archcore"
  echo "stub" > "$d/.archcore/stub.md"
  printf '%s' '{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}' > "$d/.mcp.json"
  cd "$d"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local ctx
  ctx=$(copilot_document)
  case "$ctx" in
    *ctx*) ;;
    *) fail "the CLI hook's context did not survive into the document: '$ctx'" ;;
  esac
  case "$ctx" in
    *".archcore/ is empty"*) ;;
    *) fail "the empty-state nudge is missing from the document: '$ctx'" ;;
  esac
}

@test "copilot single doc: the wiring advisory travels inside the document" {
  # The advisory that exists precisely because the session has no MCP tools is
  # also the one that would be dropped by a broken channel — it always trails
  # the CLI's JSON.
  export MOCK_HOOKS_OUTPUT='{"additionalContext":"ctx"}'
  mock_archcore_multi
  local d="$BATS_TEST_TMPDIR/wiring-doc"
  mkdir -p "$d/.archcore"
  printf -- '---\ntitle: A\n---\n\n# A\n\nSubstantive body text for the doc.\n' > "$d/.archcore/a.adr.md"
  cd "$d"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local ctx
  ctx=$(copilot_document)
  case "$ctx" in
    *"not wired for Copilot"*) ;;
    *) fail "the wiring advisory is missing from the document: '$ctx'" ;;
  esac
}

@test "copilot single doc: the update advisory travels inside the document" {
  # This arm also covers the empty-CLI-output path: mock_archcore_with_update
  # answers `hooks` with a blank line, so the advisory has to become the
  # document on its own rather than appending to one.
  mock_archcore_with_update
  local d="$BATS_TEST_TMPDIR/update-doc"
  mkdir -p "$d/.archcore"
  printf -- '---\ntitle: A\n---\n\n# A\n\nSubstantive body text for the doc.\n' > "$d/.archcore/a.adr.md"
  printf '%s' '{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}' > "$d/.mcp.json"
  cd "$d"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local ctx
  ctx=$(copilot_document)
  case "$ctx" in
    *"CLI update available"*) ;;
    *) fail "the update advisory is missing from the document: '$ctx'" ;;
  esac
}

@test "copilot single doc: several advisories at once still make one document" {
  # Empty state + wiring + update all fire in the same session. This is the
  # arm that produced three trailing plain-text lines and cost the user the
  # entire payload.
  mock_archcore_with_update
  local d="$BATS_TEST_TMPDIR/all-doc"
  mkdir -p "$d/.archcore"
  echo "stub" > "$d/.archcore/stub.md"
  cd "$d"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local ctx
  ctx=$(copilot_document)
  local missing=""
  case "$ctx" in *".archcore/ is empty"*) ;; *) missing="$missing empty-state" ;; esac
  case "$ctx" in *"not wired for Copilot"*) ;; *) missing="$missing wiring" ;; esac
  case "$ctx" in *"CLI update available"*) ;; *) missing="$missing update" ;; esac
  [ -z "$missing" ] || fail "advisories missing from the single document:$missing
$ctx"
}

@test "copilot single doc: an unrecognised CLI payload is never rewritten" {
  # The splice only fires on the exact {"additionalContext":"..."} shape it
  # knows. If the CLI ever emits something else, corrupting it would be worse
  # than dropping the advisory — so the payload passes through untouched and
  # the advisories go out as progress lines, which the host strips before
  # parsing. Either way the single-document rule holds.
  export MOCK_HOOKS_OUTPUT='{"hookSpecificOutput":{"additionalContext":"ctx"}}'
  mock_archcore_multi
  local d="$BATS_TEST_TMPDIR/foreign-doc"
  mkdir -p "$d/.archcore"
  echo "stub" > "$d/.archcore/stub.md"
  cd "$d"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  copilot_document > /dev/null
  printf '%s\n' "$output" | grep -qF '{"hookSpecificOutput":{"additionalContext":"ctx"}}' \
    || fail "the unrecognised CLI payload was rewritten instead of passed through:
$output"
  printf '%s\n' "$output" | grep -q '"type":"progress"' \
    || fail "advisories were dropped entirely instead of falling back to progress lines:
$output"
}
