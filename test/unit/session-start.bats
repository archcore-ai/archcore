#!/usr/bin/env bats
# Tests for bin/session-start

setup() {
  load '../helpers/common'
  common_setup
}

@test "reports missing archcore CLI" {
  # Override PATH to exclude real archcore but keep system tools
  run sh -c "PATH='/usr/bin:/bin' && printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "CLI not installed"
  assert_output --partial "archcore.ai/install.sh"
  assert_output --partial "hookSpecificOutput"
}

@test "reports missing .archcore/ directory" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  assert_output --partial "archcore init"
  assert_output --partial "hookSpecificOutput"
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
