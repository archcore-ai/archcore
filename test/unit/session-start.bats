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
