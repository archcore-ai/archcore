#!/usr/bin/env bats
# The plugin launcher carries a copy of the CLI's context address, because it
# speaks when the CLI is absent. This test holds the copy equal to
# agents.ContextAddress (context-address-delivery.spec, invariant 1).

setup() {
  load '../helpers/common'
  common_setup
}

@test "the launcher's context address equals the CLI constant" {
  local source="$WORKSPACE_ROOT/cli/internal/agents/instructions.go"
  [ -f "$source" ] || skip "the CLI source is not in this checkout"

  local cli_address launcher_address
  cli_address=$(sed -n 's/^const ContextAddress = "\(.*\)"$/\1/p' "$source")
  launcher_address=$(sed -n "s/^  _ac_address='\(.*\)'\$/\1/p" "$PLUGIN_ROOT/bin/session-start")

  [ -n "$cli_address" ] || fail "ContextAddress not found in $source"
  [ -n "$launcher_address" ] || fail "_ac_address not found in bin/session-start"
  [ "$cli_address" = "$launcher_address" ] || {
    printf 'cli:      %s\nlauncher: %s\n' "$cli_address" "$launcher_address" >&2
    fail "the launcher's address drifted from the CLI constant"
  }
}
