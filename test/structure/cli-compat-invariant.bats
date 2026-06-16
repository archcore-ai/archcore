#!/usr/bin/env bats
# Structure test: lock the CLI-agnosticism invariant for the globals rollout.
#
# The plugin must keep working against ANY archcore CLI version (old or new).
# That means it must never branch on optional new MCP output fields, never write
# the `globals` config key, and only read the `globals` literal in the single
# place that detects an old-CLI×globals mismatch (bin/session-start). These tests
# fail if a future change quietly couples the plugin to the new CLI contract.

setup() {
  load '../helpers/common'
  common_setup
}

@test "no plugin code branches on optional MCP output fields" {
  # source_kind / read_only / source_id are optional CLI outputs. Branching on
  # them would break against an old CLI that does not emit them. Hard zero.
  local hits
  hits=$(grep -rniE 'source_kind|read_only|source_id' \
    "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/bin" 2>/dev/null || true)
  [ -z "$hits" ] || fail "found references to optional MCP output fields:
$hits"
}

@test "the 'globals' literal appears in bin/ only inside session-start" {
  # Reading settings.json for \"globals\" to explain an old-CLI crash is host-side
  # defense, not consuming a CLI contract — allowed, but ONLY in session-start.
  local offenders
  offenders=$(grep -rliE 'globals' "$PLUGIN_ROOT/bin" 2>/dev/null \
    | grep -v '/session-start$' || true)
  [ -z "$offenders" ] || fail "unexpected 'globals' reference in bin/ outside session-start:
$offenders"
}

@test "no skill references the 'globals' config key" {
  # The plugin stays globals-agnostic: skills must never write or depend on the
  # globals key (invariant #1 — never write globals to settings.json).
  local hits
  hits=$(grep -rniE 'globals' "$PLUGIN_ROOT/skills" 2>/dev/null || true)
  [ -z "$hits" ] || fail "skills must not reference 'globals':
$hits"
}
