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

  # cwd needs a project marker for the mcp sanity guard (Step 0c).
  # Stderr discarded so the diagnostic log doesn't pollute exact-match assertion.
  mkdir -p "$BATS_TEST_TMPDIR/proj"
  : > "$BATS_TEST_TMPDIR/proj/.git"
  run sh -c "cd '$BATS_TEST_TMPDIR/proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp --foo bar 2>/dev/null"
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
  assert_output --partial "ARCHCORE_CWD is set but is not a directory"
  assert_output --partial "$start"
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
# Also drops a `.git` marker so Step 0c (cross-host sanity guard) sees a
# project root — Step 0b's plugin-cache check stays the gate that matters
# for cache-path scenarios. Real plugin checkouts ARE git repos, so this is
# faithful to production conditions.
_fake_plugin_install() {
  local dir="$1"
  mkdir -p "$dir/bin" "$dir/.codex-plugin"
  printf '{}\n' > "$dir/.codex-plugin/plugin.json"
  printf '{}\n' > "$dir/.codex.mcp.json"
  : > "$dir/.git"
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

  # stderr suppressed to keep the [archcore mcp] cwd= diagnostic out of $output.
  run sh -c "cd '$dev_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' '$dev_dir/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "dev-checkout-ok"
}

@test "guard: does NOT refuse from cache path when ARCHCORE_CWD redirects to user project" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"
  local user_proj
  mkdir -p "$BATS_TEST_TMPDIR/user-proj"
  # `.git` marker so Step 0c (cross-host sanity guard) recognizes user_proj
  # as a real project after Step 0 rebases cwd via $ARCHCORE_CWD.
  : > "$BATS_TEST_TMPDIR/user-proj/.git"
  user_proj=$(cd "$BATS_TEST_TMPDIR/user-proj" && pwd -P)

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
pwd
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_CWD='$user_proj' '$cache_dir/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "$user_proj"
}

@test "guard: ARCHCORE_ALLOW_PLUGIN_CWD=1 escape hatch bypasses refusal (back-compat alias)" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'plugin-cwd-allowed\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_ALLOW_PLUGIN_CWD=1 '$cache_dir/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "plugin-cwd-allowed"
}

@test "guard: ARCHCORE_ALLOW_ANY_CWD=1 escape hatch bypasses refusal (canonical name)" {
  local cache_dir="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  _fake_plugin_install "$cache_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'any-cwd-allowed\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_ALLOW_ANY_CWD=1 '$cache_dir/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "any-cwd-allowed"
}

@test "guard: does NOT refuse non-mcp subcommands from cache path" {
  # User-invoked CLI calls (doctor, status, init…) show their cwd to the
  # user — only the silent MCP child needs the guard. Step 0c also skips
  # non-mcp subcommands, so this passes regardless of project markers.
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

@test "guard: does NOT refuse cache-path cwd when only some plugin markers present (Step 0b)" {
  # Real user project that happens to live under */plugins/cache/* (unlikely
  # but possible) and has a project marker — Step 0b requires all three
  # plugin markers, and Step 0c is satisfied by .git.
  local cache_like="$BATS_TEST_TMPDIR/.codex/plugins/cache/m/archcore/0.0.0"
  mkdir -p "$cache_like"
  # Only ONE of the three plugin markers — Step 0b requires all three.
  printf '{}\n' > "$cache_like/.codex.mcp.json"
  # Project marker so Step 0c (sanity guard) sees a real project.
  : > "$cache_like/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'no-block\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cache_like' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "no-block"
}

# --- Step 0c: cross-host cwd sanity guard ---
#
# Refuse `archcore mcp` from cwds that are obviously not a user project:
# filesystem root, $HOME, plugin install dirs, or directories with no project
# markers (.git, .archcore, package.json, go.mod, pyproject.toml, Cargo.toml,
# pom.xml, build.gradle, build.gradle.kts). Convert silent wrong-cwd bugs
# (Cursor global mcp.json without `cwd: "${workspaceFolder}"`, Claude Code's
# ignored `cwd` field) into a loud refusal with per-host fix instructions.
# See: .archcore/plugin/cwd-guard-for-cursor-and-claude.idea.md

@test "sanity guard: refuses mcp when cwd has no project markers" {
  local bare_dir="$BATS_TEST_TMPDIR/no-markers"
  mkdir -p "$bare_dir"

  run sh -c "cd '$bare_dir' && PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "no project markers"
  assert_output --partial "Cursor"
  assert_output --partial "workspaceFolder"
  assert_output --partial "ARCHCORE_ALLOW_ANY_CWD"
}

@test "sanity guard: refuses mcp when cwd is \$HOME" {
  local fake_home="$BATS_TEST_TMPDIR/fake-home"
  mkdir -p "$fake_home"

  # Even though fake_home is empty (no markers), the explicit HOME-match
  # message takes precedence — verifies branch ordering, not just outcome.
  run sh -c "cd '$fake_home' && HOME='$fake_home' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "\$HOME"
}

@test "sanity guard: refuses mcp when cwd == \$CLAUDE_PLUGIN_ROOT" {
  local plugin_root="$BATS_TEST_TMPDIR/fake-plugin-root"
  # Plugin root has its own markers (a checked-out plugin IS a git repo) —
  # but the explicit CLAUDE_PLUGIN_ROOT-match must still refuse.
  mkdir -p "$plugin_root"
  : > "$plugin_root/.git"

  run sh -c "cd '$plugin_root' && CLAUDE_PLUGIN_ROOT='$plugin_root' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "CLAUDE_PLUGIN_ROOT"
}

@test "sanity guard: refuses mcp when cwd == \$CURSOR_PLUGIN_ROOT" {
  local plugin_root="$BATS_TEST_TMPDIR/fake-cursor-plugin"
  mkdir -p "$plugin_root"
  : > "$plugin_root/.git"

  run sh -c "cd '$plugin_root' && CURSOR_PLUGIN_ROOT='$plugin_root' PATH='/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "CURSOR_PLUGIN_ROOT"
}

@test "sanity guard: passes mcp when cwd has .git marker" {
  local proj="$BATS_TEST_TMPDIR/proj-git"
  mkdir -p "$proj"
  : > "$proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'git-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "git-ok"
}

@test "sanity guard: passes mcp when cwd has .archcore directory" {
  local proj="$BATS_TEST_TMPDIR/proj-archcore"
  mkdir -p "$proj/.archcore"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'archcore-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "archcore-ok"
}

@test "sanity guard: passes mcp when cwd has package.json marker" {
  local proj="$BATS_TEST_TMPDIR/proj-node"
  mkdir -p "$proj"
  printf '{}\n' > "$proj/package.json"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'node-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "node-ok"
}

@test "sanity guard: ARCHCORE_ALLOW_ANY_CWD=1 bypasses refusal in bare directory" {
  local bare_dir="$BATS_TEST_TMPDIR/bare"
  mkdir -p "$bare_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'bypass-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$bare_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' ARCHCORE_ALLOW_ANY_CWD=1 '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "bypass-ok"
}

@test "sanity guard: does NOT trigger for non-mcp subcommands" {
  local bare_dir="$BATS_TEST_TMPDIR/bare-doctor"
  mkdir -p "$bare_dir"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'doctor-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$bare_dir' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' doctor"
  assert_success
  assert_output "doctor-ok"
}

# --- Diagnostic stderr log ---

@test "mcp start logs resolved cwd and .archcore state to stderr" {
  local proj="$BATS_TEST_TMPDIR/proj-with-archcore"
  mkdir -p "$proj/.archcore"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"

  # Capture stderr separately; default `run` collapses it into $output but we
  # want explicit confirmation the log goes to stderr, not stdout.
  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>&1 1>/dev/null"
  assert_success
  assert_output --partial "[archcore mcp] cwd="
  assert_output --partial "$proj"
  assert_output --partial "archcore_dir="
  assert_output --partial "(exists)"
}

@test "mcp start logs (missing) when .archcore directory is absent" {
  local proj="$BATS_TEST_TMPDIR/proj-no-archcore"
  mkdir -p "$proj"
  : > "$proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>&1 1>/dev/null"
  assert_success
  assert_output --partial "(missing)"
}

@test "non-mcp subcommands do NOT emit the cwd diagnostic" {
  local proj="$BATS_TEST_TMPDIR/proj-doctor-log"
  mkdir -p "$proj"
  : > "$proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' doctor 2>&1 1>/dev/null"
  assert_success
  refute_output --partial "[archcore mcp]"
}

# --- Additional Step 0c coverage: filesystem root, all project markers, HOME with markers ---

@test "sanity guard: refuses mcp when cwd is filesystem root (/)" {
  run sh -c "cd / && PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$PLUGIN_ROOT/bin/archcore' mcp 2>&1"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "filesystem root"
}

@test "sanity guard: passes mcp when cwd has go.mod marker" {
  local proj="$BATS_TEST_TMPDIR/proj-go"
  mkdir -p "$proj"
  : > "$proj/go.mod"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'go-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "go-ok"
}

@test "sanity guard: passes mcp when cwd has pyproject.toml marker" {
  local proj="$BATS_TEST_TMPDIR/proj-python"
  mkdir -p "$proj"
  : > "$proj/pyproject.toml"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'python-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "python-ok"
}

@test "sanity guard: passes mcp when cwd has Cargo.toml marker" {
  local proj="$BATS_TEST_TMPDIR/proj-rust"
  mkdir -p "$proj"
  : > "$proj/Cargo.toml"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'rust-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "rust-ok"
}

@test "sanity guard: passes mcp when cwd has pom.xml marker" {
  local proj="$BATS_TEST_TMPDIR/proj-maven"
  mkdir -p "$proj"
  : > "$proj/pom.xml"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'maven-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "maven-ok"
}

@test "sanity guard: passes mcp when cwd has build.gradle marker" {
  local proj="$BATS_TEST_TMPDIR/proj-gradle"
  mkdir -p "$proj"
  : > "$proj/build.gradle"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'gradle-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "gradle-ok"
}

@test "sanity guard: passes mcp when cwd has build.gradle.kts marker" {
  local proj="$BATS_TEST_TMPDIR/proj-gradle-kotlin"
  mkdir -p "$proj"
  : > "$proj/build.gradle.kts"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'gradle-kotlin-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "gradle-kotlin-ok"
}

@test "sanity guard: HOME with .git marker is still refused" {
  local fake_home="$BATS_TEST_TMPDIR/fake-home-with-git"
  mkdir -p "$fake_home"
  : > "$fake_home/.git"

  run sh -c "cd '$fake_home' && HOME='$fake_home' PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$PLUGIN_ROOT/bin/archcore' mcp 2>&1"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "\$HOME"
}

# --- CURSOR_PLUGIN_ROOT interactions ---

@test "sanity guard: refuses mcp when cwd == CURSOR_PLUGIN_ROOT and cwd has .git" {
  local plugin_root="$BATS_TEST_TMPDIR/fake-cursor-plugin"
  mkdir -p "$plugin_root"
  cd "$plugin_root" && git init -q

  run sh -c "cd '$plugin_root' && CURSOR_PLUGIN_ROOT='$plugin_root' PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 '$PLUGIN_ROOT/bin/archcore' mcp 2>&1"
  assert_failure
  assert_output --partial "Refusing to start MCP"
  assert_output --partial "CURSOR_PLUGIN_ROOT"
}

@test "sanity guard: CURSOR_PLUGIN_ROOT set but cwd is different valid project — passes" {
  local other_root="$BATS_TEST_TMPDIR/other-cursor-plugin"
  local proj="$BATS_TEST_TMPDIR/proj-cursor-other"
  mkdir -p "$other_root" "$proj"
  : > "$proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'cursor-other-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && CURSOR_PLUGIN_ROOT='$other_root' PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "cursor-other-ok"
}

@test "ARCHCORE_CWD + CURSOR_PLUGIN_ROOT: redirected cwd passes guard" {
  local cursor_root="$BATS_TEST_TMPDIR/cursor-plugin-with-git"
  local target_proj="$BATS_TEST_TMPDIR/target-proj-cursor"
  mkdir -p "$cursor_root" "$target_proj"
  : > "$cursor_root/.git"
  : > "$target_proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'archcore-cwd-rebase-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$cursor_root' && ARCHCORE_CWD='$target_proj' CURSOR_PLUGIN_ROOT='$cursor_root' PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "archcore-cwd-rebase-ok"
}

# --- ARCHCORE_CWD edge cases ---

@test "ARCHCORE_CWD set to an existing file (not dir) — warns and keeps inherited cwd" {
  local proj="$BATS_TEST_TMPDIR/proj-cwd-file"
  local bad_file="$BATS_TEST_TMPDIR/bad-cwd-file"
  mkdir -p "$proj"
  : > "$proj/.git"
  : > "$bad_file"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && ARCHCORE_CWD='$bad_file' PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' doctor 2>&1 1>/dev/null"
  assert_success
  assert_output --partial "ARCHCORE_CWD is set but is not a directory"
  assert_output --partial "$bad_file"
}

@test "ARCHCORE_CWD set to nonexistent path — silently ignored (no warning)" {
  local proj="$BATS_TEST_TMPDIR/proj-cwd-nonexistent"
  mkdir -p "$proj"
  : > "$proj/.git"

  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf 'no-warn-ok\n'
MOCK
  chmod +x "$MOCK_BIN/archcore"

  run sh -c "cd '$proj' && ARCHCORE_CWD='/does/not/exist/xyz' PATH='$MOCK_BIN:/usr/bin:/bin' '$PLUGIN_ROOT/bin/archcore' mcp 2>/dev/null"
  assert_success
  assert_output "no-warn-ok"
  refute_output --partial "is not a directory"
}
