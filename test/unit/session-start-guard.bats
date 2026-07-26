#!/usr/bin/env bats
# Plugin-install-dir guard for bin/session-start.
#
# Existing coverage (test/unit/session-start.bats): silent exit when cwd holds
# .plugin/.cursor-plugin/.claude-plugin/.codex-plugin manifest siblings — all
# exercised with claude-code-shaped stdin ('{}').
#
# Covered here:
#   - the guard holds for EVERY host's stdin shape (Cursor is the host with
#     the confirmed misrouted-cwd behavior, so a cursor-shaped payload guard
#     test is the regression pin that matters most);
#   - the guard fires BEFORE any archcore invocation and before the CLI
#     availability check (no install nudge from inside a plugin cache);
#   - layer 1: install-cache path fragments in $PWD, manifest-free — the only
#     layer that catches manifest-less cache paths;
#   - layer 2: bounded (depth-12) upward manifest walk, including both sides
#     of the bound so deeply-nested legitimate projects are not silenced.

setup() {
  load '../helpers/common'
  common_setup
}

make_fake_plugin() {
  # $1 = manifest dir name (e.g. .cursor-plugin)
  local root="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$root/$1" "$root/.archcore"
  echo '{"name":"fake"}' > "$root/$1/plugin.json"
  printf '%s' "$root"
}

@test "guard holds for cursor-shaped stdin (conversation_id) in plugin install dir" {
  mock_archcore ""
  cd "$(make_fake_plugin .cursor-plugin)"

  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit for cursor payload, got: '$output'"
}

@test "guard holds for codex-shaped stdin (turn_id) in plugin install dir" {
  mock_archcore ""
  cd "$(make_fake_plugin .codex-plugin)"

  run sh -c "printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit for codex payload, got: '$output'"
}

@test "guard holds when ARCHCORE_HOST forces copilot in plugin install dir" {
  mock_archcore ""
  cd "$(make_fake_plugin .claude-plugin)"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit for copilot host, got: '$output'"
}

@test "guard fires before archcore CLI is ever invoked" {
  # The guard's job is to prevent the plugin's bundled .archcore/ from being
  # read as the user's knowledge base — so the CLI must not run at all.
  # MOCK_ARCHCORE_LOG is exported, so the piped subshell inherits it.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging "should-not-appear"
  cd "$(make_fake_plugin .cursor-plugin)"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ ! -f "$MOCK_ARCHCORE_LOG" ] || [ ! -s "$MOCK_ARCHCORE_LOG" ] \
    || fail "archcore CLI was invoked despite the plugin-dir guard: $(cat "$MOCK_ARCHCORE_LOG")"
}

@test "guard fires before the CLI availability check (no install nudge from a cache dir)" {
  # archcore deliberately NOT on PATH: from a plugin install dir the guard
  # must exit silently BEFORE the CLI check would emit its install message.
  cd "$(make_fake_plugin .cursor-plugin)"

  run sh -c "printf '%s' '{}' | PATH='$MOCK_BIN:/usr/bin:/bin' '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit even without archcore on PATH, got: '$output'"
}

@test "guard covers subdirectories of a plugin install (no manifest at cwd)" {
  mock_archcore ""
  local root
  root="$(make_fake_plugin .cursor-plugin)"
  mkdir -p "$root/skills/init"
  cd "$root/skills/init"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit from install subdir, got: '$output'"
}

@test "layer 1: install-cache path fragments silence the hook without any manifest" {
  # The fragment match is the ONLY layer that catches manifest-less cache
  # paths — e.g. a partially-extracted or foreign plugin's cache dir.
  mock_archcore "SHOULD-NOT-APPEAR"
  local fragment
  for fragment in ".cursor/plugins" ".claude/plugins" ".codex/plugins" "plugins/cache"; do
    local dir="$BATS_TEST_TMPDIR/home/$fragment/some-plugin/abc123/deep/deeper"
    mkdir -p "$dir"
    cd "$dir"
    run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    [ -z "$output" ] \
      || fail "fragment '$fragment': expected silent exit, got: '$output'"
  done
}

@test "layer 2: manifest 11 levels above cwd is still caught by the walk" {
  mock_archcore "SHOULD-NOT-APPEAR"
  local root
  root="$(make_fake_plugin .cursor-plugin)"
  local sub="$root/a/b/c/d/e/f/g/h/i/j/k"   # cwd is 11 dirs below the manifest
  mkdir -p "$sub"
  cd "$sub"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit (manifest at depth 11), got: '$output'"
}

@test "layer 2: manifest beyond the depth-12 bound does NOT silence a legit project" {
  # The bound is a safety valve: a legitimate project nested 13+ dirs under
  # some ancestor that happens to carry a plugin manifest must still get its
  # session context.
  mock_archcore "CONTEXT-EMITTED"
  local root
  root="$(make_fake_plugin .cursor-plugin)"
  local sub="$root/a/b/c/d/e/f/g/h/i/j/k/l/m"   # 13 dirs below the manifest
  mkdir -p "$sub/.archcore"
  echo "x" > "$sub/.archcore/keep"
  cd "$sub"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -n "$output" ] || fail "expected session-start to run normally beyond the walk bound"
  [[ "$output" == *"CONTEXT-EMITTED"* ]] \
    || fail "expected CLI context beyond the walk bound, got: '$output'"
}
