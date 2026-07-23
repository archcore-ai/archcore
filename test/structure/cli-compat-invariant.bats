#!/usr/bin/env bats
# Structure test: lock the CLI-agnosticism invariant for the globals rollout.
#
# The plugin must keep working against ANY archcore CLI version (old or new).
# The original rollout encoded this as "zero mention of globals in skills/" — a
# crude proxy. It is now expressed as the actual safety properties, so the plugin
# MAY understand globals (read the optional MCP source fields, surface them, honor
# read-only) while still degrading to identical behavior on a CLI that omits them:
#
#   1. bin/ (executable code) MUST NOT branch on the optional MCP fields — an old
#      CLI does not emit them, so real logic that depends on them would break.
#   2. The plugin MUST NEVER write the `globals` config key to settings.json — a
#      pre-globals CLI has a strict parser that crashes on the unknown field.
#   3. A skill MAY read the optional fields, but every clause that does MUST be
#      data-gated: absent fields ⇒ behavior identical to the no-globals path.
#   4. The shared globals convention doc must exist wherever a skill loads it.
#
# Cross-repo contract: ../cli/.archcore/features/globals-plugin-compat.plan.md
# (rule 3 revised from "no mention" to "data-gated"). These tests fail if a future
# change couples the plugin to the new CLI contract in a way an old CLI can't honor.

setup() {
  load '../helpers/common'
  common_setup
}

@test "bin/ never branches on optional MCP output fields" {
  # source_kind / read_only / source_id are optional CLI outputs. Branching on
  # them in executable code would break against an old CLI that omits them.
  # Hard zero for bin/. Skills are data-gated prose, guarded separately below.
  local hits
  hits=$(grep -rniE 'source_kind|read_only|source_id' \
    "$PLUGIN_ROOT/bin" 2>/dev/null || true)
  [ -z "$hits" ] || fail "bin/ branches on optional MCP output fields:
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

@test "no skill writes the 'globals' config key to settings.json" {
  # Invariant #2: the plugin never writes globals (incl. /archcore:init). A
  # pre-globals CLI's strict parser crashes on the unknown field. Reading or
  # explaining globals is fine; emitting the JSON key "globals" is not — the
  # quoted key form is what a write into settings.json would produce.
  local hits
  hits=$(grep -rniE '"globals"' "$PLUGIN_ROOT/skills" 2>/dev/null || true)
  [ -z "$hits" ] || fail "skill emits the 'globals' settings.json key (must never write it):
$hits"
}

@test "every skill reading optional MCP fields is data-gated (absent ⇒ unchanged)" {
  # Invariant #3: a skill MAY react to global source fields, but it MUST also
  # state the no-op fallback when they are absent (old CLI / no globals). Without
  # the guard, the clause would silently change behavior on a CLI that omits the
  # fields. This is a stronger old-CLI-safety guarantee than "no mention".
  local f unguarded=""
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    grep -qiE 'no result is global|no global match|absent any global|proceed as usual|proceed unchanged|no change|behave exactly as you would|render exactly as|no badge|only when' \
      "$f" || unguarded="$unguarded
$f"
  done < <(grep -rliE 'source_kind|read_only|[^a-z]global[: ]' "$PLUGIN_ROOT/skills" 2>/dev/null || true)
  [ -z "$unguarded" ] || fail "skill reads global source fields without an absent-default guard:$unguarded"
}

@test "every skills/_shared/globals.md reference resolves" {
  # Invariant #4: clauses load the shared convention doc; it must exist so the
  # data-gating instructions can't dangle.
  local refs missing="" f
  refs=$(grep -rlF 'skills/_shared/globals.md' "$PLUGIN_ROOT/skills" 2>/dev/null || true)
  if [ -n "$refs" ]; then
    [ -f "$PLUGIN_ROOT/skills/_shared/globals.md" ] \
      || missing="skills/_shared/globals.md is referenced but does not exist"
  fi
  [ -z "$missing" ] || fail "$missing"
}
