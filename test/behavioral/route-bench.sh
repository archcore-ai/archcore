#!/bin/sh
# Behavioral conductor bench. Runs model calls only when explicitly requested.
# ROUTE_BENCH_MODEL selects the model; ROUTE_BENCH_LIMIT limits fixture count.
# ROUTE_BENCH_FIXTURES overrides input; ROUTE_BENCH_OUTPUT_DIR retains raw replies.
# Exit 1: routing/format mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${ROUTE_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/routing-bench.tsv}
CONTRACT="$REPO_ROOT/plugins/archcore/skills/_shared/delta-routing.md"
GRANULARITY="$REPO_ROOT/plugins/archcore/skills/_shared/capability-granularity.md"
TAB=$(printf '\t')

[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI not found on PATH" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${ROUTE_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "ROUTE_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 4 || $1 == "" || $2 == "" || $3 == "" || $4 !~ /^(null|amendment|capability|decision|umbrella)$/ || seen[$1]++ {bad=1}
  {n++}
  END {exit (bad || !n)}
' "$FIXTURES"; then
  echo "fixtures must contain unique IDs and four nonempty fields with a valid route" >&2
  exit 2
fi

contract_text=$(cat "$CONTRACT")
granularity_text=$(cat "$GRANULARITY")
if [ -n "${ROUTE_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$ROUTE_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-route-bench.XXXXXX") || exit 2
  trap 'rm -rf "$results_dir"' EXIT
fi
results_dir=$(CDPATH='' cd -- "$results_dir" && pwd)
: > "$results_dir/pass"
: > "$results_dir/fail"
: > "$results_dir/error"

n=0
printf 'id\texpected\tgot\tverdict\tannouncement\n'
while IFS="$TAB" read -r id task grounding expected || [ -n "$id" ]; do
  case "$id" in ''|\#*) continue ;; esac
  case "$id$task$grounding$expected" in *[![:space:]]*) ;; *) continue ;; esac
  n=$((n + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  printf '%s\n' \
    "You are the route conductor defined by the two contracts below. Apply them literally." \
    "Output EXACTLY one announcement line in the contract's Route announcement format and nothing else." \
    "" \
    "--- CONTRACT: skills/_shared/delta-routing.md ---" "$contract_text" \
    "--- CONTRACT: skills/_shared/capability-granularity.md ---" "$granularity_text" \
    "--- TASK ---" "$task" \
    "--- GROUNDING RESULT (already established; do not re-derive) ---" "$grounding" \
    > "$results_dir/$n.prompt"
  set -- -p --tools '' --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
    --setting-sources user --no-session-persistence --output-format json
  if [ -n "${ROUTE_BENCH_MODEL:-}" ]; then set -- "$@" --model "$ROUTE_BENCH_MODEL"; fi
  cli_status=0
  (cd "$results_dir" && claude "$@" < "$results_dir/$n.prompt") \
    > "$results_dir/$n.json" 2> "$results_dir/$n.stderr" || cli_status=$?
  if [ "$cli_status" -ne 0 ] || ! jq -e '.is_error == false and (.result | type == "string")' "$results_dir/$n.json" >/dev/null 2>&1; then
    printf '%s\t%s\tnone\tERROR\tCLI failed; see %s/%s.stderr and .json\n' "$id" "$expected" "$results_dir" "$n"
    echo "$id" >> "$results_dir/error"
    continue
  fi
  out=$(jq -r '.result' "$results_dir/$n.json")
  # Inline code is the same rendered announcement, not a different route.
  case "$out" in '`route:'*'`') out=${out#'`'}; out=${out%'`'} ;; esac
  got=$(printf '%s\n' "$out" | sed -n 's/^route: \([a-z][a-z]*\).*/\1/p')
  if [ "$got" = "$expected" ] && [ "$(printf '%s\n' "$out" | wc -l | tr -d ' ')" -eq 1 ]; then
    verdict=pass; echo "$id" >> "$results_dir/pass"
  else
    verdict=FAIL; echo "$id" >> "$results_dir/fail"
  fi
  announcement=$(printf '%s' "$out" | tr '\t\n' '  ')
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$expected" "${got:-none}" "$verdict" "$announcement"
done < "$FIXTURES"

passed=$(wc -l < "$results_dir/pass" | tr -d ' ')
failed=$(wc -l < "$results_dir/fail" | tr -d ' ')
errors=$(wc -l < "$results_dir/error" | tr -d ' ')
echo "# route-bench: $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
