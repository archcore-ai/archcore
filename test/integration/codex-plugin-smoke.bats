#!/usr/bin/env bats
# Integration smoke checks for Codex plugin packaging.

setup() {
  load '../helpers/common'
  common_setup

  command -v codex >/dev/null 2>&1 || skip "codex CLI not installed"

  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  export TEST_HOME
}

@test "codex marketplace add accepts repo root marketplace" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$PLUGIN_ROOT"
  assert_success
  assert_output --partial 'Added marketplace `archcore-plugins`'

  [ -f "$TEST_HOME/.codex/config.toml" ]
  grep -q '^\[marketplaces.archcore-plugins\]' "$TEST_HOME/.codex/config.toml"
  grep -q "source = \"$PLUGIN_ROOT\"" "$TEST_HOME/.codex/config.toml"
}

@test "codex debug prompt-input loads Archcore skills when plugin is enabled" {
  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/skills" "$installed_root/skills"
  ln -s "$PLUGIN_ROOT/agents" "$installed_root/agents"
  ln -s "$PLUGIN_ROOT/hooks" "$installed_root/hooks"
  ln -s "$PLUGIN_ROOT/bin" "$installed_root/bin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"

  mkdir -p "$TEST_HOME/.codex"
  {
    printf '[marketplaces.archcore-plugins]\n'
    printf 'last_updated = "2026-05-04T00:00:00Z"\n'
    printf 'source_type = "local"\n'
    printf 'source = "%s"\n' "$PLUGIN_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex debug prompt-input "use archcore review"
  assert_success
  assert_output --partial 'archcore:review'
  assert_output --partial 'skills/review/SKILL.md'
}

@test "codex mcp list includes plugin-managed Archcore MCP when plugin is enabled" {
  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"
  ln -s "$PLUGIN_ROOT/bin" "$installed_root/bin"

  mkdir -p "$TEST_HOME/.codex"
  {
    printf '[marketplaces.archcore-plugins]\n'
    printf 'last_updated = "2026-05-04T00:00:00Z"\n'
    printf 'source_type = "local"\n'
    printf 'source = "%s"\n' "$PLUGIN_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex mcp list --json
  assert_success
  assert_output --partial '"archcore"'
  assert_output --partial '"command": "./bin/archcore"'
}

@test "installed Codex MCP launcher uses ARCHCORE_CWD and guards plugin-cache cwd" {
  if ! env PATH="/usr/bin:/bin" ARCHCORE_SKIP_DOWNLOAD=1 "$PLUGIN_ROOT/bin/archcore" --version >/dev/null 2>&1; then
    skip "archcore CLI cache is not warm; run ./bin/archcore --help once before this smoke test"
  fi

  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"
  ln -s "$PLUGIN_ROOT/bin" "$installed_root/bin"

  local user_project="$BATS_TEST_TMPDIR/user-project"
  mkdir -p "$user_project/.archcore"
  printf -- '---\ntitle: codex review marker\nstatus: draft\n---\n\nmarker body\n' \
    > "$user_project/.archcore/codex-review-marker.doc.md"

  local rpc="$BATS_TEST_TMPDIR/list-documents.jsonl"
  printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"codex-smoke","version":"0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_documents","arguments":{}}}' \
    > "$rpc"

  run env -i \
    HOME="$HOME" \
    PATH="/usr/bin:/bin" \
    USER="${USER:-}" \
    LANG="${LANG:-C}" \
    TERM="${TERM:-xterm}" \
    TMPDIR="${TMPDIR:-/tmp}" \
    ARCHCORE_SKIP_DOWNLOAD=1 \
    ARCHCORE_CWD="$user_project" \
    sh -c 'cd "$1" && ./bin/archcore mcp < "$2"' sh "$installed_root" "$rpc"
  assert_success
  assert_output --partial "codex-review-marker"

  run env -i \
    HOME="$HOME" \
    PATH="/usr/bin:/bin" \
    USER="${USER:-}" \
    LANG="${LANG:-C}" \
    TERM="${TERM:-xterm}" \
    TMPDIR="${TMPDIR:-/tmp}" \
    ARCHCORE_SKIP_DOWNLOAD=1 \
    sh -c 'cd "$1" && ./bin/archcore mcp' sh "$installed_root"
  assert_failure
  assert_output --partial "Refusing to start MCP from the plugin install dir"
  assert_output --partial "ARCHCORE_CWD"
}
