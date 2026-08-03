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
  # install from the developer's own plugins, settings and auth state. The
  # directory is deliberately named `.copilot` (not `copilot-home`): the
  # CLI's plugin-cache guard keys on the `.copilot/installed-plugins/` path
  # fragment, and an isolated home that renames the directory would silently
  # opt the whole suite out of the guard's jurisdiction — the exact custom-
  # COPILOT_HOME limitation recorded in copilot-mcp-architecture.adr.
  COPILOT_HOME="$BATS_TEST_TMPDIR/home/.copilot"
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
             | grep -o 'bin/[a-z0-9_-]*' | sort -u | sed "s|^|$root/|")
  [ -z "$broken" ] || fail "hook scripts unusable after install:$broken"
}

@test "an installed plugin does not shadow the project's archcore server" {
  # The end-to-end form of test/structure/plugin-mcp-isolation.bats: that file
  # pins the layout in the repo, this one proves the layout still produces the
  # right merge after Copilot has packaged and installed it.
  #
  # Copilot merges MCP sources user -> workspace -> plugins with last-wins, so
  # a plugin server keyed "archcore" REPLACES the project server that
  # `archcore init --agent copilot` writes under the same key. The replacement
  # runs with cwd=${PLUGIN_ROOT} and archcore >= v0.6.7 refuses to serve from
  # there (next test) — the user is left with no document tools while the
  # wiring advisory still reports the project as wired.
  install_plugin

  local root
  root=$(installed_root) || fail "plugin not found under $COPILOT_HOME/installed-plugins"

  # Half one: no filename Copilot auto-discovers survived the install.
  [ ! -e "$root/.mcp.json" ] \
    || fail "the install carries a plugin-root .mcp.json — Copilot auto-discovers it whatever the manifest says"
  [ ! -e "$root/.github/mcp.json" ] \
    || fail "the install carries .github/mcp.json — Copilot's fallback discovery name"

  # Half two: the Copilot manifest declares its own (empty) mcpServers, so
  # Copilot does not fall back to reading .claude-plugin/plugin.json and pick
  # the Claude-only server back up.
  [ "$(jq -r '.mcpServers | type' "$root/.plugin/plugin.json")" = "object" ] \
    || fail "the installed Copilot manifest must declare an mcpServers object"
  [ "$(jq -r '.mcpServers | length' "$root/.plugin/plugin.json")" = "0" ] \
    || fail "the installed Copilot manifest contributes MCP servers — it must contribute none"

  # And the behaviour those two halves exist for: a project that wires
  # archcore keeps the server it wired. The sentinel command is what makes
  # this an identity check rather than a name check — the failing state also
  # has an entry called "archcore", just not the project's one.
  local proj="$BATS_TEST_TMPDIR/proj"
  mkdir -p "$proj"
  printf '%s' '{"mcpServers":{"archcore":{"command":"WORKSPACE_SENTINEL","args":["mcp"]}}}' > "$proj/.mcp.json"

  run env COPILOT_HOME="$COPILOT_HOME" copilot -C "$proj" mcp list --json
  [ "$status" -eq 0 ] || skip "copilot mcp list --json unavailable on this CLI version"

  local cmd
  cmd=$(printf '%s' "$output" | jq -r '(.mcpServers // .).archcore.command // empty' 2>/dev/null)
  [ "$cmd" = "WORKSPACE_SENTINEL" ] \
    || fail "the project's archcore server was replaced (command=${cmd:-<absent>}) — the plugin is shadowing it again"
}

@test "a server started from the install cache refuses to serve (defense in depth)" {
  # Executable regression of the silent-cache-write incident: a real ADR
  # (test-solution.adr.md) once landed in ~/.copilot/installed-plugins/
  # because `archcore mcp` accepted the install dir as a project root. With
  # CLI >= v0.6.7 the resolver refuses the cache loudly at startup.
  #
  # The test above removes the route that got a server started there in the
  # first place; this one stays because that route is not the only one. A
  # custom COPILOT_HOME, a hand-written user-level config, or any future host
  # that spawns from an install directory all land here, and the guard is the
  # last thing between them and a document written into a plugin cache.
  command -v archcore >/dev/null 2>&1 || skip "archcore CLI not installed"
  [ "$("$PLUGIN_ROOT/bin/cli-gte" 0.6.7)" = "yes" ] \
    || skip "archcore CLI < v0.6.7 (no copilot cache guard yet)"

  install_plugin

  local root
  root=$(installed_root) || fail "plugin not found under $COPILOT_HOME/installed-plugins"

  run sh -c "cd '$root' && archcore mcp < /dev/null"
  [ "$status" -ne 0 ] || fail "archcore mcp served from the install cache instead of refusing"
  assert_output --partial 'plugin install cache'
  [ ! -d "$root/.archcore" ] || fail "the probe left a .archcore/ inside the install cache"
}
