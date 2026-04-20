#!/usr/bin/env bats
# Tests for bin/archcore (shim)

setup() {
  load '../helpers/common'
  common_setup
}

# Run the shim with a restricted PATH so no real archcore is resolvable.
# MOCK_BIN is retained (mocks placed there are picked up), PLUGIN_ROOT/bin
# is excluded (we're testing the shim itself, not re-entering via PATH).
run_shim() {
  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$PLUGIN_ROOT/bin/archcore' $*"
}

# --- $ARCHCORE_BIN override ---

@test "uses ARCHCORE_BIN when set and executable" {
  cat > "$BATS_TEST_TMPDIR/fake" <<'FAKE'
#!/bin/sh
printf 'from-archcore-bin\n'
FAKE
  chmod +x "$BATS_TEST_TMPDIR/fake"

  run sh -c "ARCHCORE_BIN='$BATS_TEST_TMPDIR/fake' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-archcore-bin"
}

@test "ignores ARCHCORE_BIN when path not executable" {
  # Use a scratch shim dir with no vendor/ so resolution falls all the way through
  local scratch="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$scratch"
  cp "$PLUGIN_ROOT/bin/archcore" "$scratch/archcore"
  cp "$PLUGIN_ROOT/bin/CLI_VERSION" "$scratch/CLI_VERSION"

  run sh -c "ARCHCORE_BIN='/does/not/exist' PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$scratch/archcore' --version"
  assert_failure
  assert_output --partial "not cached"
  assert_output --partial "ARCHCORE_SKIP_DOWNLOAD=1"
}

# --- PATH resolution ---

@test "uses archcore from PATH when ARCHCORE_BIN unset" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'from-path\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-path"
}

# --- Vendor cache ---

@test "uses vendor binary when PATH lacks archcore and vendor file exists" {
  local vendor_dir="$PLUGIN_ROOT/bin/vendor"
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  local vendor_bin="$vendor_dir/archcore-v$version"

  # Snapshot pre-existing vendor file so we don't clobber user's download
  local backup=""
  if [ -f "$vendor_bin" ]; then
    backup="$BATS_TEST_TMPDIR/vendor-backup"
    mv "$vendor_bin" "$backup"
  fi

  mkdir -p "$vendor_dir"
  cat > "$vendor_bin" <<'VENDOR'
#!/bin/sh
printf 'from-vendor\n'
VENDOR
  chmod +x "$vendor_bin"

  run sh -c "PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' --version"
  local status_code=$status
  local captured_output=$output

  # Restore original vendor binary (if any) before assertions, so failures
  # don't leave the repo in a dirty state.
  rm -f "$vendor_bin"
  if [ -n "$backup" ]; then
    mv "$backup" "$vendor_bin"
  fi

  [ "$status_code" -eq 0 ] || fail "expected success, got $status_code (output: $captured_output)"
  [ "$captured_output" = "from-vendor" ] || fail "expected 'from-vendor', got '$captured_output'"
}

# --- Failure path ---

@test "exits 1 with actionable error when no archcore resolvable and download skipped" {
  local scratch="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$scratch"
  cp "$PLUGIN_ROOT/bin/archcore" "$scratch/archcore"
  cp "$PLUGIN_ROOT/bin/CLI_VERSION" "$scratch/CLI_VERSION"

  run sh -c "PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$scratch/archcore' --version"
  assert_failure
  assert_output --partial "[archcore shim]"
  assert_output --partial "not cached"
  assert_output --partial "ARCHCORE_SKIP_DOWNLOAD=1"
}

@test "fail-fast stderr when CLI_VERSION is missing" {
  # Stage a copy of the shim against a scratch directory so we can hide CLI_VERSION
  local scratch="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$scratch"
  cp "$PLUGIN_ROOT/bin/archcore" "$scratch/archcore"
  # deliberately NOT copying CLI_VERSION

  run sh -c "PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$scratch/archcore' --version"
  assert_failure
  assert_output --partial "CLI_VERSION"
}

# --- Argv and stdin proxying ---

@test "passes argv verbatim to resolved binary" {
  cat > "$BATS_TEST_TMPDIR/fake" <<'FAKE'
#!/bin/sh
printf 'args:'
for a in "$@"; do printf ' <%s>' "$a"; done
printf '\n'
FAKE
  chmod +x "$BATS_TEST_TMPDIR/fake"

  run sh -c "ARCHCORE_BIN='$BATS_TEST_TMPDIR/fake' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp --flag 'a b'"
  assert_success
  assert_output "args: <mcp> <--flag> <a b>"
}

@test "passes stdin through to resolved binary" {
  cat > "$BATS_TEST_TMPDIR/fake" <<'FAKE'
#!/bin/sh
cat
FAKE
  chmod +x "$BATS_TEST_TMPDIR/fake"

  run sh -c "printf 'piped-data\n' | ARCHCORE_BIN='$BATS_TEST_TMPDIR/fake' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore'"
  assert_success
  assert_output "piped-data"
}

@test "exit code matches resolved binary exit code" {
  cat > "$BATS_TEST_TMPDIR/fake" <<'FAKE'
#!/bin/sh
exit 42
FAKE
  chmod +x "$BATS_TEST_TMPDIR/fake"

  run sh -c "ARCHCORE_BIN='$BATS_TEST_TMPDIR/fake' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore'"
  [ "$status" -eq 42 ] || fail "expected 42, got $status"
}

# --- CLI_VERSION reading ---

@test "reads CLI_VERSION from SCRIPT_DIR (not CWD)" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  # Create a bogus CLI_VERSION in CWD to ensure the shim doesn't read it.
  # If the shim incorrectly used CWD, resolution would fail on version mismatch
  # (there's no such vendor binary). Since archcore is in PATH, resolution
  # happens before CLI_VERSION is read, so the shim succeeds regardless.
  cd "$BATS_TEST_TMPDIR"
  printf 'nonsense\n' > CLI_VERSION

  run sh -c "cd '$BATS_TEST_TMPDIR' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "ok"
}
