#!/usr/bin/env bats
# Integration smoke checks for Codex plugin packaging + discovery.
#
# Every loading check installs through the real marketplace path. Hand-written
# cache layouts bypass the host's installed-plugin registration and can pass
# against a layout that users never load.

setup() {
  load '../helpers/common'
  bats_require_minimum_version 1.5.0
  common_setup

  command -v codex >/dev/null 2>&1 || skip "codex CLI not installed"

  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  export TEST_HOME
  # A project MCP registration must not hide a broken plugin MCP registration.
  cd "$TEST_HOME"
}

@test "codex marketplace add accepts the repo-root marketplace" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success
  assert_output --partial 'Added marketplace `archcore-plugins`'

  [ -f "$TEST_HOME/.codex/config.toml" ]
  grep -q '^\[marketplaces.archcore-plugins\]' "$TEST_HOME/.codex/config.toml"
}

@test "codex plugin list discovers archcore from the subdirectory (issue #2 regression)" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success

  run env HOME="$TEST_HOME" codex plugin list
  assert_success
  # Pre-fix: archcore was absent from `plugin list` because source.path was
  # the marketplace root ("./"), which Codex does not scan for plugins.
  assert_output --partial 'archcore@archcore-plugins'
  # And it must resolve to the dedicated subdirectory, never the repo root.
  assert_output --partial 'plugins/archcore'
}

@test "codex plugin add archcore succeeds (issue #2 regression)" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success

  run env HOME="$TEST_HOME" codex plugin add archcore@archcore-plugins
  # Pre-fix this failed with: plugin `archcore` was not found in marketplace.
  assert_success
  refute_output --partial 'was not found'
  assert_output --partial 'Added plugin `archcore`'
}

install_archcore() {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success || return 1
  run env HOME="$TEST_HOME" codex plugin add archcore@archcore-plugins
  assert_success || return 1
}

@test "codex debug prompt-input loads Archcore skills when plugin is enabled" {
  install_archcore

  run --separate-stderr env HOME="$TEST_HOME" codex debug prompt-input "use archcore review"
  assert_success
  local skills
  skills=$(jq -r '.[] | select(.role == "developer") | .content[] | .text? // empty | select(contains("<skills_instructions>"))' <<< "$output")
  grep -q '^- archcore:review:' <<< "$skills" || { fail "installed review skill is absent from the host catalog"; return 1; }
  # Codex may abbreviate installed skill paths through its rN root aliases.
  grep -Eq '\(file: (r[0-9]+/review/SKILL.md|/[^)]*/skills/review/SKILL.md)\)' <<< "$skills" \
    || { fail "review skill has no resolvable file reference"; return 1; }
}

@test "codex mcp list includes plugin-managed Archcore MCP when plugin is enabled" {
  install_archcore

  run env HOME="$TEST_HOME" codex mcp list --json
  assert_success
  assert_output --partial '"archcore"'
  assert_output --partial '"command": "archcore"'
}
