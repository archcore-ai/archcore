#!/bin/sh
# route-bench.sh — behavioral routing bench (LLM-in-the-loop, non-CI).
#
# Replays the routing fixtures through an LLM acting as the conductor defined
# by skills/_shared/delta-routing.md and scores the announced route against the
# expected route. Structural tests cannot hold routing semantics; this harness
# does. It needs the `claude` CLI on PATH and spends model tokens — run it on
# demand (`make test-routing-bench`), never in CI.
#
# Env knobs:
#   ROUTE_BENCH_MODEL  model passed to `claude -p` (default: the CLI default)
#   ROUTE_BENCH_LIMIT  run only the first N fixtures (default: all)
#
# Output: one TSV line per fixture (id, expected, got, verdict, announcement),
# then a summary line. Exit 1 when any fixture mismatches.
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES="$REPO_ROOT/test/behavioral/fixtures/routing-bench.tsv"
CONTRACT="$REPO_ROOT/plugins/archcore/skills/_shared/delta-routing.md"
GRANULARITY="$REPO_ROOT/plugins/archcore/skills/_shared/capability-granularity.md"
TAB=$(printf '\t')

[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI not found on PATH" >&2; exit 2; }

MODEL_ARGS=""
[ -n "${ROUTE_BENCH_MODEL:-}" ] && MODEL_ARGS="--model ${ROUTE_BENCH_MODEL}"
LIMIT="${ROUTE_BENCH_LIMIT:-0}"

contract_text=$(cat "$CONTRACT")
granularity_text=$(cat "$GRANULARITY")

tally_dir="${TMPDIR:-/tmp}/route-bench.$$"
mkdir -p "$tally_dir"
trap 'rm -rf "$tally_dir"' EXIT
: > "$tally_dir/pass"
: > "$tally_dir/fail"

n=0
printf 'id\texpected\tgot\tverdict\tannouncement\n'
grep -v '^#' "$FIXTURES" | while IFS="$TAB" read -r id task grounding expected; do
  [ -n "$id" ] || continue
  n=$((n + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  out=$(printf '%s\n' \
    "You are the route conductor defined by the two contracts below. Apply them literally." \
    "Output EXACTLY one announcement line in the contract's Route announcement format and nothing else." \
    "" \
    "--- CONTRACT: skills/_shared/delta-routing.md ---" \
    "$contract_text" \
    "--- CONTRACT: skills/_shared/capability-granularity.md ---" \
    "$granularity_text" \
    "--- TASK ---" \
    "$task" \
    "--- GROUNDING RESULT (already established; do not re-derive) ---" \
    "$grounding" \
    | claude -p $MODEL_ARGS 2>/dev/null | grep -m1 '^route:' || true)
  got=$(printf '%s' "$out" | sed -n 's/^route: \([a-z][a-z]*\).*/\1/p')
  if [ "$got" = "$expected" ]; then
    verdict="pass"; echo "$id" >> "$tally_dir/pass"
  else
    verdict="FAIL"; echo "$id" >> "$tally_dir/fail"
  fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$expected" "${got:-none}" "$verdict" "$out"
done

passed=$(wc -l < "$tally_dir/pass" | tr -d ' ')
failed=$(wc -l < "$tally_dir/fail" | tr -d ' ')
echo "# route-bench: ${passed} pass, ${failed} fail"
[ "$failed" -eq 0 ]
