#!/usr/bin/env bats
# Tests for bin/session-start

setup() {
  load '../helpers/common'
  common_setup
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

@test "unresolvable shim is non-fatal (no archcore in PATH, no vendor, no network)" {
  # Override PATH to exclude archcore AND force shim fail-fast so it doesn't
  # attempt a network download during tests.
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
}

# --- MCP auto-register ---

@test "auto-registers archcore in local scope when not registered" {
  # Mock claude CLI. `mcp list` returns no archcore; `mcp add` logs invocation.
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
if [ "$1" = "mcp" ] && [ "$2" = "list" ]; then
  printf 'other-server: /bin/foo\n'
  exit 0
fi
if [ "$1" = "mcp" ] && [ "$2" = "add" ]; then
  printf 'MCP_ADD_ARGS: %s\n' "$*" >> "$MOCK_CLAUDE_LOG"
  exit 0
fi
exit 0
MOCK
  chmod +x "$MOCK_BIN/claude"
  export MOCK_CLAUDE_LOG="$BATS_TEST_TMPDIR/claude.log"
  : > "$MOCK_CLAUDE_LOG"

  # Mock archcore so shim resolution succeeds via PATH (no download)
  mock_archcore ""

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "MOCK_CLAUDE_LOG='$MOCK_CLAUDE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success

  [ -f "$MOCK_CLAUDE_LOG" ]
  grep -q "MCP_ADD_ARGS: mcp add archcore" "$MOCK_CLAUDE_LOG" || fail "expected 'mcp add archcore' invocation in claude log: $(cat "$MOCK_CLAUDE_LOG")"
  grep -q -- "-s local" "$MOCK_CLAUDE_LOG" || fail "expected '-s local' scope flag"
}

@test "skips auto-register when archcore is already in mcp list" {
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
if [ "$1" = "mcp" ] && [ "$2" = "list" ]; then
  printf 'archcore: /path/to/existing/archcore mcp\n'
  exit 0
fi
if [ "$1" = "mcp" ] && [ "$2" = "add" ]; then
  printf 'MCP_ADD_CALLED\n' >> "$MOCK_CLAUDE_LOG"
  exit 0
fi
exit 0
MOCK
  chmod +x "$MOCK_BIN/claude"
  export MOCK_CLAUDE_LOG="$BATS_TEST_TMPDIR/claude.log"
  : > "$MOCK_CLAUDE_LOG"

  mock_archcore ""

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "MOCK_CLAUDE_LOG='$MOCK_CLAUDE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  ! grep -q "MCP_ADD_CALLED" "$MOCK_CLAUDE_LOG" || fail "mcp add should not have been called (archcore already registered)"
}

@test "skips auto-register when claude CLI is unavailable" {
  # No mock claude on PATH (and no system claude given restricted PATH)
  mock_archcore ""

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  # Must succeed even without claude CLI (Cursor host etc.)
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
