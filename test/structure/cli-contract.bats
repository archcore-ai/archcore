#!/usr/bin/env bats
# Structure tests: every place that invokes the bundled archcore CLI must use
# a real subcommand. Catches the class of bug where a script calls a phantom
# subcommand (e.g. `archcore validate` instead of `archcore doctor`) — the
# launcher then exits 1 and the hook silently misbehaves.
#
# Sources of truth:
#   - bin/CLI_VERSION pins the bundled CLI version. When that bumps,
#     re-verify the allowlist below against `archcore --help`.
#   - The "live cross-check" test at the bottom verifies the allowlist matches
#     `./bin/archcore --help` if the launcher is resolvable in the test env.

setup() {
  load '../helpers/common'
  common_setup
}

# Canonical archcore CLI subcommands — keep in sync with bin/CLI_VERSION.
# As of 0.2.2: config, doctor, help, hooks, init, mcp, status, update.
ARCHCORE_SUBCOMMANDS="config doctor help hooks init mcp status update"

# Return 0 if $1 is a whitespace-separated token in $ARCHCORE_SUBCOMMANDS.
_is_allowlisted() {
  case " $ARCHCORE_SUBCOMMANDS " in
    *" $1 "*) return 0 ;;
    *)        return 1 ;;
  esac
}

# Extract every subcommand following `"$LAUNCHER"` in the given file.
# Skips `2>&1`, `2>/dev/null`, and other redirection tokens by requiring a
# leading lowercase letter. Prints one subcommand per line.
_extract_launcher_subcommands() {
  grep -oE '"\$LAUNCHER"[[:space:]]+[a-z][a-z0-9-]*' "$1" \
    | sed -E 's/^"\$LAUNCHER"[[:space:]]+//'
}

# --- Per-script invocation tests ---

@test "bin/validate-archcore invokes only allowlisted subcommands" {
  local file="$PLUGIN_ROOT/bin/validate-archcore"
  local subcommands
  subcommands=$(_extract_launcher_subcommands "$file")
  [ -n "$subcommands" ] || fail "no \$LAUNCHER invocation found in $file"

  while IFS= read -r sub; do
    _is_allowlisted "$sub" || fail "validate-archcore invokes phantom subcommand '$sub'. Allowed: $ARCHCORE_SUBCOMMANDS"
  done <<< "$subcommands"
}

@test "bin/session-start invokes only allowlisted subcommands" {
  local file="$PLUGIN_ROOT/bin/session-start"
  local subcommands
  subcommands=$(_extract_launcher_subcommands "$file")
  [ -n "$subcommands" ] || fail "no \$LAUNCHER invocation found in $file"

  while IFS= read -r sub; do
    _is_allowlisted "$sub" || fail "session-start invokes phantom subcommand '$sub'. Allowed: $ARCHCORE_SUBCOMMANDS"
  done <<< "$subcommands"
}

@test "every bin/ script that invokes \$LAUNCHER uses only allowlisted subcommands" {
  # Future-proof: any new hook script that shells out to the launcher gets
  # checked automatically. Excludes the launcher itself and the lib/ helpers.
  local offenders=""
  for f in "$PLUGIN_ROOT"/bin/*; do
    [ -f "$f" ] || continue
    case "$(basename "$f")" in
      archcore|archcore.cmd|archcore.ps1|CLI_VERSION) continue ;;
    esac
    grep -q '"\$LAUNCHER"' "$f" || continue

    while IFS= read -r sub; do
      [ -z "$sub" ] && continue
      if ! _is_allowlisted "$sub"; then
        offenders="$offenders $(basename "$f"):$sub"
      fi
    done <<< "$(_extract_launcher_subcommands "$f")"
  done
  [ -z "$offenders" ] || fail "Phantom subcommands invoked:$offenders. Allowed: $ARCHCORE_SUBCOMMANDS"
}

# --- MCP launcher configs ---

@test ".mcp.json invokes the launcher with an allowlisted subcommand" {
  local file="$PLUGIN_ROOT/.mcp.json"
  local sub
  sub=$(jq -r '.mcpServers.archcore.args[0]' < "$file")
  _is_allowlisted "$sub" || fail ".mcp.json args[0]='$sub' is not an archcore subcommand. Allowed: $ARCHCORE_SUBCOMMANDS"
}

@test ".codex.mcp.json invokes the launcher with an allowlisted subcommand" {
  local file="$PLUGIN_ROOT/.codex.mcp.json"
  local sub
  sub=$(jq -r '.mcpServers.archcore.args[0]' < "$file")
  _is_allowlisted "$sub" || fail ".codex.mcp.json args[0]='$sub' is not an archcore subcommand. Allowed: $ARCHCORE_SUBCOMMANDS"
}

# --- Sentinel: no executable code references the historical phantoms ---

@test "no executable code references phantom subcommands 'validate' or 'sync'" {
  # These two have shown up in real bugs (validate) or planning docs (sync).
  # Guard against either re-entering executable code.
  local hits=""
  for f in "$PLUGIN_ROOT"/bin/* "$PLUGIN_ROOT"/.mcp.json "$PLUGIN_ROOT"/.codex.mcp.json "$PLUGIN_ROOT"/hooks/*.json; do
    [ -f "$f" ] || continue
    case "$(basename "$f")" in
      archcore|archcore.cmd|archcore.ps1|CLI_VERSION) continue ;;
    esac
    # Match `archcore validate` / `archcore sync` (whitespace-separated).
    # Allow it inside the literal "validate-archcore" name (script reference).
    if grep -nE 'archcore[[:space:]]+(validate|sync)([[:space:]]|$)' "$f" \
       | grep -v 'validate-archcore' >/dev/null 2>&1; then
      hits="$hits $(basename "$f")"
    fi
  done
  [ -z "$hits" ] || fail "Phantom subcommand references found in:$hits"
}

# --- Live cross-check (skipped if launcher unavailable) ---

@test "hardcoded allowlist matches live archcore --help" {
  # Try to enumerate the real CLI's subcommands. If the launcher can't resolve
  # the binary (no cache, no network, restricted CI), skip — we cannot ground
  # the allowlist in this environment, and the static tests above still apply.
  local help_out
  help_out=$(ARCHCORE_SKIP_DOWNLOAD=1 "$PLUGIN_ROOT/bin/archcore" --help 2>/dev/null) \
    || skip "archcore launcher cannot resolve CLI in this environment"

  # Parse the "Available Commands:" block: every non-blank, non-flag line
  # whose first token is a lowercase word.
  local live
  live=$(printf '%s\n' "$help_out" \
    | awk '/^Available Commands:/{flag=1; next} /^Flags:/{flag=0} flag && /^[[:space:]]+[a-z]/ {print $1}' \
    | sort -u \
    | tr '\n' ' ' \
    | sed 's/ $//')

  local expected
  expected=$(printf '%s\n' $ARCHCORE_SUBCOMMANDS | sort -u | tr '\n' ' ' | sed 's/ $//')

  [ "$live" = "$expected" ] || fail "Allowlist drift. Hardcoded: '$expected'. Live: '$live'. Update ARCHCORE_SUBCOMMANDS in $BATS_TEST_FILENAME."
}
