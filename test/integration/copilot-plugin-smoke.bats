#!/usr/bin/env bats
# Integration smoke checks for GitHub Copilot CLI plugin packaging + discovery.
#
# Copilot has no marketplace catalog — installation is `copilot plugin install`
# against a directory or `<repo>:<subdir>` spec — so the discovery surface this
# file exercises is the install itself plus what lands on disk afterwards.
#
# The assertions are deliberately filesystem-shaped rather than output-shaped.
# Copilot's CLI output wording is not a contract we control, and a smoke test
# that greps for a phrase breaks on a release note. What the plugin actually
# promises is that every component the manifest points at survives packaging
# with its permissions intact — that is checkable exactly, and it is what broke
# for Codex in issue #2.
#
# Like the Codex smoke test, this ships and skips where the host is absent, so
# a contributor with Copilot installed runs it for free and CI stays quiet.

setup() {
  load '../helpers/common'
  common_setup

  command -v copilot >/dev/null 2>&1 || skip "copilot CLI not installed"

  # COPILOT_HOME replaces the whole ~/.copilot path, so this isolates the
  # install from the developer's own plugins, settings and auth state.
  COPILOT_HOME="$BATS_TEST_TMPDIR/copilot-home"
  mkdir -p "$COPILOT_HOME"
  export COPILOT_HOME
}

# Absolute path of the installed plugin root, located by its manifest so the
# layout under installed-plugins/ stays Copilot's business, not ours.
installed_root() {
  local manifest
  manifest=$(find "$COPILOT_HOME/installed-plugins" -type f -path '*/.plugin/plugin.json' 2>/dev/null | head -1)
  [ -n "$manifest" ] || return 1
  dirname "$(dirname "$manifest")"
}

install_plugin() {
  run env COPILOT_HOME="$COPILOT_HOME" copilot plugin install "$PLUGIN_ROOT"
  assert_success
}

@test "copilot plugin install accepts the plugin directory" {
  install_plugin

  run env COPILOT_HOME="$COPILOT_HOME" copilot plugin list
  assert_success
  assert_output --partial 'archcore'
}

@test "every component the manifest points at survives the install" {
  install_plugin

  local root rel missing=""
  root=$(installed_root) || fail "no .plugin/plugin.json under $COPILOT_HOME/installed-plugins"

  while IFS= read -r rel; do
    [ -z "$rel" ] && continue
    [ -e "$root/${rel#./}" ] || missing="$missing $rel"
  done < <(jq -r '.. | strings | select(startswith("./"))' < "$root/.plugin/plugin.json")
  [ -z "$missing" ] || fail "manifest points at paths absent from the install:$missing"

  # Named explicitly because these two are the ones a packaging change would
  # drop silently: agents in their own directory with the Copilot extension,
  # and the command wrappers that have no default path.
  [ -n "$(find "$root/copilot-agents" -name '*.agent.md' 2>/dev/null)" ] \
    || fail "no *.agent.md files in the installed copilot-agents/"
  [ -n "$(find "$root/commands" -name '*.md' 2>/dev/null)" ] \
    || fail "no command wrappers in the installed commands/"
}

@test "installed hook scripts exist and kept their executable bit" {
  install_plugin

  local root script broken=""
  root=$(installed_root) || fail "plugin not found under $COPILOT_HOME/installed-plugins"

  while IFS= read -r script; do
    [ -z "$script" ] && continue
    if [ ! -f "$script" ]; then
      broken="$broken missing:$script"
    elif [ ! -x "$script" ]; then
      broken="$broken not-executable:$script"
    fi
  done < <(jq -r '.. | .bash? // empty' "$root/hooks/copilot.hooks.json" \
             | sed "s|\"||g; s|\${COPILOT_PLUGIN_ROOT}|$root|g")
  [ -z "$broken" ] || fail "hook scripts unusable after install:$broken"
}

@test "the install registers no plugin-shipped MCP server" {
  # github/copilot-cli#4234: a plugin MCP child is launched in the plugin
  # install directory with no project path, so anything it writes lands in
  # ~/.copilot/installed-plugins/ instead of the user's repo. Archcore's MCP
  # comes from the project's own .mcp.json instead — see
  # copilot-mcp-architecture.adr.
  install_plugin

  local root
  root=$(installed_root) || fail "plugin not found under $COPILOT_HOME/installed-plugins"

  jq -e 'has("mcpServers") | not' "$root/.plugin/plugin.json" > /dev/null \
    || fail "the installed manifest declares mcpServers — see github/copilot-cli#4234"

  # Whether a plugin-root .mcp.json is ALSO auto-discovered is unresolved from
  # documentation (the plugin reference gives mcpServers no default path; the
  # concepts page describes plugin-root .mcp.json as a discovery location).
  # This is the check that settles it on a real host. It is advisory: if the
  # subcommand does not exist on this Copilot version, there is nothing to
  # learn here, and the manifest assertion above still stands.
  run env COPILOT_HOME="$COPILOT_HOME" copilot mcp list
  if [ "$status" -eq 0 ]; then
    refute_output --partial 'archcore'
  fi
}
