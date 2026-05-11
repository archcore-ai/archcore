#!/usr/bin/env bats
# Tests for bin/archcore (plugin-bundled CLI launcher)

setup() {
  load '../helpers/common'
  common_setup
}

# Run the launcher with a restricted PATH so no real archcore is resolvable.
# MOCK_BIN is retained (mocks placed there are picked up); $PLUGIN_ROOT/bin
# is excluded (we're testing the launcher itself, not re-entering via PATH).
# Cache directory is redirected to a scratch dir so tests never touch the
# user's real cache.
run_launcher() {
  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 CLAUDE_PLUGIN_DATA='$BATS_TEST_TMPDIR/pdata' '$PLUGIN_ROOT/bin/archcore' $*"
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

@test "ignores ARCHCORE_BIN when path not executable and falls through to cache" {
  run sh -c "ARCHCORE_BIN='/does/not/exist' PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 CLAUDE_PLUGIN_DATA='$BATS_TEST_TMPDIR/pdata' '$PLUGIN_ROOT/bin/archcore' --version"
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

# --- Cache ---

@test "uses cached binary when PATH lacks archcore and cache file exists" {
  local cache_dir="$BATS_TEST_TMPDIR/pdata/archcore/cli"
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  local cached_bin="$cache_dir/archcore-v$version"

  mkdir -p "$cache_dir"
  cat > "$cached_bin" <<'CACHE'
#!/bin/sh
printf 'from-cache\n'
CACHE
  chmod +x "$cached_bin"

  run sh -c "PATH='/usr/bin:/bin' CLAUDE_PLUGIN_DATA='$BATS_TEST_TMPDIR/pdata' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-cache"
}

@test "cache directory falls back to XDG_DATA_HOME when CLAUDE_PLUGIN_DATA unset" {
  local xdg="$BATS_TEST_TMPDIR/xdg"
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  local cached_bin="$xdg/archcore-plugin/cli/archcore-v$version"

  mkdir -p "$(dirname "$cached_bin")"
  cat > "$cached_bin" <<'CACHE'
#!/bin/sh
printf 'from-xdg-cache\n'
CACHE
  chmod +x "$cached_bin"

  run sh -c "PATH='/usr/bin:/bin' XDG_DATA_HOME='$xdg' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-xdg-cache"
}

@test "cache resolves to CODEX_PLUGIN_DATA when set" {
  local cache_dir="$BATS_TEST_TMPDIR/codex-pdata/archcore/cli"
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  local cached_bin="$cache_dir/archcore-v$version"

  mkdir -p "$cache_dir"
  cat > "$cached_bin" <<'CACHE'
#!/bin/sh
printf 'from-codex-cache\n'
CACHE
  chmod +x "$cached_bin"

  run sh -c "PATH='/usr/bin:/bin' CODEX_PLUGIN_DATA='$BATS_TEST_TMPDIR/codex-pdata' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-codex-cache"
}

@test "CODEX_PLUGIN_DATA takes precedence over CLAUDE_PLUGIN_DATA" {
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")

  local codex_cache="$BATS_TEST_TMPDIR/codex-pdata/archcore/cli"
  mkdir -p "$codex_cache"
  cat > "$codex_cache/archcore-v$version" <<'CACHE'
#!/bin/sh
printf 'from-codex\n'
CACHE
  chmod +x "$codex_cache/archcore-v$version"

  local claude_cache="$BATS_TEST_TMPDIR/claude-pdata/archcore/cli"
  mkdir -p "$claude_cache"
  cat > "$claude_cache/archcore-v$version" <<'CACHE'
#!/bin/sh
printf 'from-claude\n'
CACHE
  chmod +x "$claude_cache/archcore-v$version"

  run sh -c "PATH='/usr/bin:/bin' CODEX_PLUGIN_DATA='$BATS_TEST_TMPDIR/codex-pdata' CLAUDE_PLUGIN_DATA='$BATS_TEST_TMPDIR/claude-pdata' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-codex"
}

@test "falls back to CLAUDE_PLUGIN_DATA when CODEX_PLUGIN_DATA unset" {
  local cache_dir="$BATS_TEST_TMPDIR/pdata/archcore/cli"
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  local cached_bin="$cache_dir/archcore-v$version"

  mkdir -p "$cache_dir"
  cat > "$cached_bin" <<'CACHE'
#!/bin/sh
printf 'from-claude-cache\n'
CACHE
  chmod +x "$cached_bin"

  run sh -c "PATH='/usr/bin:/bin' CLAUDE_PLUGIN_DATA='$BATS_TEST_TMPDIR/pdata' '$PLUGIN_ROOT/bin/archcore' --version"
  assert_success
  assert_output "from-claude-cache"
}

# --- Download skip ---

@test "fails with clear message when SKIP_DOWNLOAD=1 and no cache" {
  run_launcher --version
  assert_failure
  assert_output --partial "not cached"
  assert_output --partial "ARCHCORE_SKIP_DOWNLOAD=1"
  assert_output --partial "ARCHCORE_BIN"
}

@test "missing CLI_VERSION file is a clear error" {
  # Use a scratch copy of the launcher without CLI_VERSION next to it.
  local scratch="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$scratch"
  cp "$PLUGIN_ROOT/bin/archcore" "$scratch/archcore"

  run sh -c "PATH='/usr/bin:/bin' '$scratch/archcore' --version"
  assert_failure
  assert_output --partial "CLI_VERSION"
}

# --- Argument pass-through ---

@test "passes arguments to resolved binary verbatim" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'args: %s\n' "$*"
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp --foo bar"
  assert_success
  assert_output "args: mcp --foo bar"
}

# --- Exit code propagation ---

@test "propagates resolved binary exit code" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
exit 42
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' anything"
  [ "$status" -eq 42 ] || fail "expected exit 42, got $status"
}

# --- $ARCHCORE_CWD opt-in chdir ---
#
# Custom-name env var that POSIX sh does NOT resync on startup (unlike $PWD).
# Codex's .env_clear() passes it through when listed in env_vars. The launcher
# `cd`s to it before exec'ing the real CLI so the MCP server operates in the
# user's project, not the plugin cache.
# See: .archcore/plugin/codex-mcp-cwd-rebase-to-user-project.idea.md

@test "launcher cd's to \$ARCHCORE_CWD when set to a real directory" {
  local target
  mkdir -p "$BATS_TEST_TMPDIR/user-project"
  target=$(cd "$BATS_TEST_TMPDIR/user-project" && pwd -P)

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
pwd
MOCK
  chmod +x "$MOCK_BIN/archcore"

  # Run from a different dir to prove the launcher chdir's based on env, not inherited cwd.
  run sh -c "cd '$BATS_TEST_TMPDIR' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_CWD='$target' '$PLUGIN_ROOT/bin/archcore' anything"
  assert_success
  assert_output "$target"
}

@test "launcher ignores \$ARCHCORE_CWD when the directory does not exist" {
  local start
  mkdir -p "$BATS_TEST_TMPDIR/start"
  start=$(cd "$BATS_TEST_TMPDIR/start" && pwd -P)

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
pwd
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$start' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_CWD='$BATS_TEST_TMPDIR/does-not-exist' '$PLUGIN_ROOT/bin/archcore' anything"
  assert_success
  assert_output "$start"
}

@test "launcher is a no-op when \$ARCHCORE_CWD is unset" {
  local start
  mkdir -p "$BATS_TEST_TMPDIR/start"
  start=$(cd "$BATS_TEST_TMPDIR/start" && pwd -P)

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
pwd
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$start' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' anything"
  assert_success
  assert_output "$start"
}

# --- Plugin-install-dir MCP guard (0b) ---
#
# Helper: build a fake plugin install dir at the given path with the three
# marker files the guard checks for. Copies the real launcher + CLI_VERSION.
_fake_plugin_install() {
  local dir="$1"
  mkdir -p "$dir/bin" "$dir/.codex-plugin"
  printf '{}\n' > "$dir/.codex-plugin/plugin.json"
  printf '{}\n' > "$dir/.codex.mcp.json"
  cp "$PLUGIN_ROOT/bin/archcore" "$dir/bin/archcore"
  cp "$PLUGIN_ROOT/bin/CLI_VERSION" "$dir/bin/CLI_VERSION"
}

@test "guard: refuses to start MCP from a plugin-cache cwd without ARCHCORE_CWD" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"

  run sh -c "cd '$cache_dir' && PATH='/usr/bin:/bin' '$cache_dir/bin/archcore' mcp"
  assert_failure
  assert_output --partial "Refusing to start MCP from the plugin install dir"
  assert_output --partial "ARCHCORE_CWD"
  assert_output --partial "shell wrapper"
}

@test "guard: does NOT refuse from a plugin dev checkout (non-cache path)" {
  local dev_dir="$BATS_TEST_TMPDIR/my-plugin-checkout"
  _fake_plugin_install "$dev_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'dev-checkout-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$dev_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' '$dev_dir/bin/archcore' mcp"
  assert_success
  assert_output "dev-checkout-ok"
}

@test "guard: does NOT refuse from cache path when ARCHCORE_CWD redirects to user project" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"
  local user_proj
  mkdir -p "$BATS_TEST_TMPDIR/user-proj"
  user_proj=$(cd "$BATS_TEST_TMPDIR/user-proj" && pwd -P)

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
pwd
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_CWD='$user_proj' '$cache_dir/bin/archcore' mcp"
  assert_success
  assert_output "$user_proj"
}

@test "guard: ARCHCORE_ALLOW_PLUGIN_CWD=1 escape hatch bypasses refusal" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'plugin-cwd-allowed\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_ALLOW_PLUGIN_CWD=1 '$cache_dir/bin/archcore' mcp"
  assert_success
  assert_output "plugin-cwd-allowed"
}

@test "guard: does NOT refuse non-mcp subcommands from cache path" {
  # User-invoked CLI calls (doctor, status, init…) show their cwd to the
  # user — only the silent MCP child needs the guard.
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'sub=%s\n' "$1"
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' '$cache_dir/bin/archcore' doctor"
  assert_success
  assert_output "sub=doctor"
}

@test "guard: does NOT refuse cache-path cwd when only some plugin markers present" {
  # Real user project that happens to live under */plugins/cache/* (unlikely
  # but possible) but lacks our specific marker files — must NOT be blocked.
  local cache_like="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  mkdir -p "$cache_like"
  # Only ONE of the three markers — guard requires all three.
  printf '{}\n' > "$cache_like/.codex.mcp.json"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'no-block\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_like' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp"
  assert_success
  assert_output "no-block"
}
