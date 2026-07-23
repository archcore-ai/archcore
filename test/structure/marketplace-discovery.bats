#!/usr/bin/env bats
# Cross-host marketplace discovery guard (regression test for issue #2).
#
# Every host marketplace catalog lives at the repo root and points its plugin
# `source`/`path` at a dedicated subdirectory that holds the per-host manifest.
#
# Issue #2: the Codex catalog pointed `source.path` at the marketplace ROOT
# (`"./"`). Codex requires a dedicated subdirectory and silently fails to
# discover a plugin whose manifest sits at the marketplace root — even though
# `.codex-plugin/plugin.json` physically existed there. So "the manifest
# exists at the resolved path" is NOT a sufficient check (it passed under the
# bug). The load-bearing assertion is: the source must resolve to a
# subdirectory that is NOT the marketplace root.
#
# These tests pin all three catalogs so a regression to `"./"` / `"."` (or a
# move of the plugin without updating a catalog) fails loudly.

setup() {
  load '../helpers/common'
  common_setup
}

# Resolve a catalog source string (relative to the marketplace root = REPO_ROOT)
# to an absolute directory, asserting it is a real subdirectory below the root.
# Args: <source-string> <host-label> <expected-manifest-relpath>
assert_subdir_source() {
  local src="$1" host="$2" manifest="$3"

  [ -n "$src" ] || fail "$host: marketplace source is empty"

  local resolved
  resolved=$(cd "$REPO_ROOT" && cd "$src" 2>/dev/null && pwd) \
    || fail "$host: source '$src' does not resolve to an existing directory"

  [ "$resolved" != "$REPO_ROOT" ] \
    || fail "$host: source '$src' points at the marketplace ROOT — Codex (and the canonical Claude/Cursor layout) cannot discover a plugin there. Use a subdirectory like ./plugins/archcore (issue #2)."

  case "$resolved/" in
    "$REPO_ROOT"/*) ;;
    *) fail "$host: source '$src' resolves outside the marketplace root ($resolved)";;
  esac

  [ -f "$resolved/$manifest" ] \
    || fail "$host: no manifest at resolved source — expected '$manifest' under '$src'"
}

@test "Codex catalog source.path resolves to a manifest subdirectory (not root)" {
  local cat="$REPO_ROOT/.agents/plugins/marketplace.json"
  local src
  src=$(jq -r '.plugins[0].source.path' < "$cat")
  assert_subdir_source "$src" "Codex" ".codex-plugin/plugin.json"
}

@test "Claude catalog source resolves to a manifest subdirectory (not root)" {
  local cat="$REPO_ROOT/.claude-plugin/marketplace.json"
  local src
  src=$(jq -r '.plugins[0].source' < "$cat")
  assert_subdir_source "$src" "Claude" ".claude-plugin/plugin.json"
}

@test "Cursor catalog source resolves to a manifest subdirectory (not root)" {
  local cat="$REPO_ROOT/.cursor-plugin/marketplace.json"
  local src
  src=$(jq -r '.plugins[0].source' < "$cat")
  assert_subdir_source "$src" "Cursor" ".cursor-plugin/plugin.json"
}

@test "all three catalogs point at the same plugin directory" {
  local codex claude cursor
  codex=$(jq -r '.plugins[0].source.path' < "$REPO_ROOT/.agents/plugins/marketplace.json")
  claude=$(jq -r '.plugins[0].source' < "$REPO_ROOT/.claude-plugin/marketplace.json")
  cursor=$(jq -r '.plugins[0].source' < "$REPO_ROOT/.cursor-plugin/marketplace.json")
  [ "$codex" = "$claude" ] && [ "$claude" = "$cursor" ] \
    || fail "catalog source drift: codex='$codex' claude='$claude' cursor='$cursor' (single source of truth expected)"
}
