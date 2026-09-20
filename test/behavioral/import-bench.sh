#!/bin/sh
# Behavioral init-entry bench. Runs model calls only when explicitly requested.
# Measures three decisions of /archcore:init from a described repository: the
# route the run takes, the size tier of the assessment gate, and the triage
# verdict of one probed source. It does not execute an import: conversion
# quality needs a repository run and is outside this harness.
# IMPORT_BENCH_MODEL selects the model; IMPORT_BENCH_LIMIT limits fixture count.
# IMPORT_BENCH_FIXTURES overrides input; IMPORT_BENCH_OUTPUT_DIR retains raw replies.
# Exit 1: decision/format mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${IMPORT_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/import-bench.tsv}
SKILL="$REPO_ROOT/plugins/archcore/skills/init/SKILL.md"
SOURCES="$REPO_ROOT/plugins/archcore/skills/init/lib/sources.md"
TRACK="$REPO_ROOT/plugins/archcore/skills/_shared/tracks/import.md"
TAB=$(printf '\t')

[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI not found on PATH" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${IMPORT_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "IMPORT_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 7 || $1 == "" || $2 == "" || $3 == "" || $6 == "" || seen[$1]++ {bad=1}
  $4 !~ /^(plain|import|import-only|no-source|empty|refresh|resume|seeded)$/ {bad=1}
  $5 !~ /^(-|none|S|M|L)$/ {bad=1}
  $7 !~ /^(-|convert|mine|reference|skip)$/ {bad=1}
  {n++}
  END {exit (bad || !n)}
' "$FIXTURES"; then
  echo "fixtures must contain unique IDs and seven fields with a valid route, tier, and verdict" >&2
  exit 2
fi

skill_text=$(cat "$SKILL")
sources_text=$(cat "$SOURCES")
tiers_text=$(sed -n '/^## Size tiers and waves/,/^### gate: import.assess/p' "$TRACK")
if [ -n "${IMPORT_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$IMPORT_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-import-bench.XXXXXX") || exit 2
  trap 'rm -rf "$results_dir"' EXIT
fi
results_dir=$(CDPATH='' cd -- "$results_dir" && pwd)
: > "$results_dir/pass"
: > "$results_dir/fail"
: > "$results_dir/error"

n=0
printf 'id\texpected\tgot\tverdict\treply\n'
while IFS="$TAB" read -r id args repo route tier probe verdict_exp || [ -n "$id" ]; do
  case "$id" in ''|\#*) continue ;; esac
  n=$((n + 1))
  [ "$args" = "-" ] && args=""
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  printf '%s\n' \
    "You are the /archcore:init skill defined below. Apply it literally, but execute nothing and call no tool." \
    "From the repository description alone, decide three things." \
    "Output EXACTLY one line and nothing else: init: <route> <tier> <verdict>" \
    "<route> is one of: plain — a plain init that composes the code seed; import — the import mode runs the import track from assessment; import-only — no manifest and no source code, but authored sources exist; no-source — the import mode finds no authored source to convert and creates nothing; empty — host wiring only; refresh — the refresh mode tops up a seeded repo; resume — an open import plan is resumed; seeded — the already-seeded early exit." \
    "<tier> is the size tier of the assessment gate for the levels this run assesses: S, M, or L; none when the gate finds no authored source or does not run." \
    "<verdict> is the triage verdict of the PROBE path: convert, mine, reference, or skip; none when the probe is '-' or this run does not assess the probe's level." \
    "" \
    "--- SKILL: skills/init/SKILL.md ---" "$skill_text" \
    "--- CATALOG: skills/init/lib/sources.md ---" "$sources_text" \
    "--- TRACK EXCERPT: skills/_shared/tracks/import.md ---" "$tiers_text" \
    "--- ARGUMENTS (the text after /archcore:init; may be empty) ---" "$args" \
    "--- REPOSITORY (already established; do not re-derive) ---" "$repo" \
    "--- PROBE ---" "$probe" \
    > "$results_dir/$n.prompt"
  set -- -p --tools '' --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
    --setting-sources user --no-session-persistence --output-format json
  if [ -n "${IMPORT_BENCH_MODEL:-}" ]; then set -- "$@" --model "$IMPORT_BENCH_MODEL"; fi
  cli_status=0
  (cd "$results_dir" && claude "$@" < "$results_dir/$n.prompt") \
    > "$results_dir/$n.json" 2> "$results_dir/$n.stderr" || cli_status=$?
  expected="$route $tier $verdict_exp"
  if [ "$cli_status" -ne 0 ] || ! jq -e '.is_error == false and (.result | type == "string")' "$results_dir/$n.json" >/dev/null 2>&1; then
    printf '%s\t%s\tnone\tERROR\tCLI failed; see %s/%s.stderr and .json\n' "$id" "$expected" "$results_dir" "$n"
    echo "$id" >> "$results_dir/error"
    continue
  fi
  out=$(jq -r '.result' "$results_dir/$n.json")
  case "$out" in '`init:'*'`') out=${out#'`'}; out=${out%'`'} ;; esac
  pat='^init: \([a-z-][a-z-]*\) \([A-Za-z][a-z]*\) \([a-z][a-z]*\)$'
  got_route=$(printf '%s\n' "$out" | sed -n "s/$pat/\1/p")
  got_tier=$(printf '%s\n' "$out" | sed -n "s/$pat/\2/p")
  got_verdict=$(printf '%s\n' "$out" | sed -n "s/$pat/\3/p")
  verdict=FAIL
  if [ "$(printf '%s\n' "$out" | wc -l | tr -d ' ')" -eq 1 ] && [ "$got_route" = "$route" ]; then
    verdict=pass
    if [ "$tier" != "-" ] && [ "$got_tier" != "$tier" ]; then verdict=FAIL; fi
    if [ "$verdict_exp" != "-" ] && [ "$got_verdict" != "$verdict_exp" ]; then verdict=FAIL; fi
  fi
  if [ "$verdict" = pass ]; then echo "$id" >> "$results_dir/pass"; else echo "$id" >> "$results_dir/fail"; fi
  reply=$(printf '%s' "$out" | tr '\t\n' '  ')
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$expected" "${got_route:-none} ${got_tier:-none} ${got_verdict:-none}" "$verdict" "$reply"
done < "$FIXTURES"

passed=$(wc -l < "$results_dir/pass" | tr -d ' ')
failed=$(wc -l < "$results_dir/fail" | tr -d ' ')
errors=$(wc -l < "$results_dir/error" | tr -d ' ')
echo "# import-bench: $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
