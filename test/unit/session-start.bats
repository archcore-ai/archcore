#!/usr/bin/env bats
# Tests for bin/session-start

setup() {
  load '../helpers/common'
  common_setup
}

@test "reports missing .archcore/ directory and tells agent to call init_project" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  assert_output --partial "mcp__archcore__init_project"
  assert_output --partial "hookSpecificOutput"
}

# --- Per-host emit-shape pins (three shipped hosts) -------------------------
# claude-code gets the SessionStart hookSpecificOutput JSON wrapper; cursor and
# codex fall to the plain-text arm. Pinned so host-expansion edits to the emit
# function provably leave the shipped hosts untouched.

@test "init nudge: claude-code emits SessionStart hookSpecificOutput JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"tool_name\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '"hookEventName":"SessionStart"'
}

@test "init nudge: cursor emits plain text (no JSON wrapper)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "hookSpecificOutput"
}

@test "init nudge: codex emits plain text (no JSON wrapper)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "hookSpecificOutput"
}

@test "CLI-missing notice: claude-code emits SessionStart hookSpecificOutput JSON" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '"hookEventName":"SessionStart"'
  assert_output --partial "install.sh"
}

@test "CLI-missing notice: cursor emits plain text (no JSON wrapper)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  refute_output --partial "hookSpecificOutput"
}

@test "CLI-missing notice: codex emits plain text (no JSON wrapper)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  refute_output --partial "hookSpecificOutput"
}

# --- Copilot emit shape (native top-level additionalContext) ----------------

@test "init nudge: copilot emits top-level additionalContext JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '{"additionalContext":"'
  refute_output --partial "hookSpecificOutput"
}

@test "CLI-missing notice: copilot emits top-level additionalContext JSON" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '{"additionalContext":"'
  assert_output --partial "install.sh"
  refute_output --partial "hookSpecificOutput"
}

@test "copilot: passes host arg to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{\"sessionId\":\"s1\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: copilot"
}

# --- OpenCode emit shape (plain text; bridge reads stdout verbatim) ---------

@test "init nudge: opencode emits plain text (no JSON wrapper)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=opencode '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "hookSpecificOutput"
  refute_output --partial '"additionalContext"'
}

@test "CLI-missing notice: opencode emits plain text (no JSON wrapper)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=opencode; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  refute_output --partial '"additionalContext"'
}

@test "opencode: passes host arg to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "ARCHCORE_HOST=opencode; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: opencode"
}

@test "refuses to run from a plugin install dir (.plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "survives when launcher cannot resolve CLI (no PATH, no cache, no network)" {
  # Initialized project + restricted PATH + ARCHCORE_SKIP_DOWNLOAD=1:
  # launcher exits 1, but session-start wraps with '|| true' and still succeeds.
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
}

@test "runs archcore hooks when both CLI and dir exist" {
  # Create mock archcore that logs the command
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOOKS_CALLED: $*"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  # Create temp dir with .archcore/
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "printf '%s' '{\"test\":true}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOOKS_CALLED: hooks claude-code session-start"
}

@test "passes host from stdin to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: cursor"
}

@test "session-start invokes only the 'hooks' subcommand" {
  # Lock the contract: session-start must call `archcore hooks`, nothing else.
  # Catches accidental regressions where the script swaps to a phantom or
  # different subcommand.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "MOCK_ARCHCORE_LOG='$MOCK_ARCHCORE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"

  local invoked
  invoked=$(sort -u < "$MOCK_ARCHCORE_LOG" | tr '\n' ' ' | sed 's/ $//')
  [ "$invoked" = "hooks" ] \
    || fail "expected only 'hooks' subcommand, got: '$invoked'"
}

@test "refuses to run from a plugin install dir (cursor-plugin sibling)" {
  # If session-start is launched with cwd inside the plugin install cache,
  # it must NOT emit context — otherwise it would surface the plugin's own
  # bundled .archcore/ as the user's knowledge base.
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.cursor-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.cursor-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "refuses to run from a plugin install dir (claude-plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.claude-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.claude-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "refuses to run from a plugin install dir (codex-plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.codex-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.codex-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "staleness check failure is non-fatal" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "context loaded"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
}

# --- Old-CLI compatibility advisory ----------------------------------------
# An old (pre-globals) CLI rejects `globals` in settings.json and exits non-zero
# on every config-loading command. session-start must turn that crash into one
# clear, rate-limited "update CLI" nudge — never a hard block.

# Mock: `archcore hooks` consumes stdin, writes a config-rejection to stderr,
# and exits non-zero. Args after `mock_old_cli` become the stderr message.
mock_old_cli() {
  local stderr_msg="${1:-field \"globals\" is not allowed for sync type \"none\"}"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ -n "\$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "\$1" >> "\$MOCK_ARCHCORE_LOG"
if [ "\$1" = "hooks" ]; then
  cat > /dev/null
  printf '%s\n' '${stderr_msg}' >&2
  exit 1
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "old CLI + globals in config → emits update-CLI advisory, exit 0" {
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"company","path":".archcore/global/company"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"
  assert_output --partial "install.sh"
}

@test "old CLI rejects an unknown field via stderr (no literal globals) → advisory" {
  # Validates the OR branch: settings carries no literal "globals", but stderr
  # shows the parser's rejection signature, so the advisory still fires.
  mock_old_cli 'field "foo" is not allowed for sync type "none"'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","foo":true}' > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"
}

@test "non-config CLI failure → no advisory (silent fall-through)" {
  # The key guard against bare-non-zero nudging: a generic failure with no
  # config-rejection signal and no globals in settings must NOT nudge.
  mock_old_cli 'fatal: not a git repository'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none"}' > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "globals in config + non-parser failure → no advisory (regression lock)" {
  # A current CLI that understands globals can still fail for unrelated reasons.
  # Presence of globals must NOT be sufficient — only a real parser-rejection
  # stderr signature may fire the advisory. (Codex P2: drop the globals
  # short-circuit so non-config failures stay silent even in globals projects.)
  mock_old_cli 'error: operation not allowed'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "CLI succeeds with globals in config → context flows, no advisory" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  cat > /dev/null
  echo "context loaded"
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "context loaded"
  refute_output --partial "too old"
}

@test "advisory is rate-limited to once per 24h" {
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  # First run emits the advisory and writes the stamp.
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"

  # Second run within 24h is suppressed.
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "advisory path still invokes only the 'hooks' subcommand" {
  # Rule-mandated (cli-integration-tests.rule): even on the failure path the
  # script must shell out to nothing but `archcore hooks`.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "MOCK_ARCHCORE_LOG='$MOCK_ARCHCORE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"

  local invoked
  invoked=$(sort -u < "$MOCK_ARCHCORE_LOG" | tr '\n' ' ' | sed 's/ $//')
  [ "$invoked" = "hooks" ] \
    || fail "expected only 'hooks' subcommand, got: '$invoked'"
}
