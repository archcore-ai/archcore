#!/usr/bin/env bats
# Host-logic locality (host-adapter-contract.spec): host-conditional branching
# may live ONLY in bin/lib/normalize-stdin.sh and bin/session-start. The check
# scripts consume the normalized ARCHCORE_* schema and stay host-agnostic;
# skills/agents/commands/rules must carry no host markers at all.

setup() {
  load '../helpers/common'
}

@test "ARCHCORE_HOST branching appears only in normalize-stdin.sh and session-start" {
  local hits expected
  hits=$(grep -rl 'ARCHCORE_HOST' "$PLUGIN_ROOT/bin" | sort)
  expected=$(printf '%s\n%s\n' \
    "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh" \
    "$PLUGIN_ROOT/bin/session-start" | sort)
  [ "$hits" = "$expected" ] \
    || fail "ARCHCORE_HOST leaked outside the allowed files: $hits"
}

@test "check scripts are host-agnostic (no ARCHCORE_HOST case statements)" {
  local script
  for script in check-archcore-write check-code-alignment validate-archcore \
                check-cascade check-precision check-staleness git-scope; do
    if grep -q 'ARCHCORE_HOST' "$PLUGIN_ROOT/bin/$script"; then
      fail "bin/$script must not branch on ARCHCORE_HOST"
    fi
  done
}

@test "skills/, agents/, commands/, rules/ contain no host-conditional markers" {
  local hits
  hits=$(grep -rlE 'ARCHCORE_HOST|CLAUDE_PLUGIN_ROOT|CURSOR_PLUGIN_ROOT|COPILOT_PLUGIN_ROOT|PLUGIN_ROOT' \
    "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/commands" "$PLUGIN_ROOT/rules" 2>/dev/null || true)
  [ -z "$hits" ] || fail "host markers found in shared content: $hits"
}
