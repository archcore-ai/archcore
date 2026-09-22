#!/bin/sh
# Behavioral host skill-selection bench. Runs model calls only when explicitly requested.
# Loads the plugin into a real Claude Code host (`claude -p --plugin-dir`) with only the
# Skill tool, sends a user message that names no command, and records which skill the
# host invokes first. Hooks and project MCP servers stay off; built-in skills compete.
# SKILL_BENCH_MODEL selects the model; SKILL_BENCH_LIMIT limits fixture count.
# SKILL_BENCH_FIXTURES overrides input; SKILL_BENCH_OUTPUT_DIR retains raw transcripts.
# Exit 1: selection mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${SKILL_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/skill-bench.tsv}
PLUGIN_DIR="$REPO_ROOT/plugins/archcore"
TAB=$(printf '\t')

[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI not found on PATH" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${SKILL_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "SKILL_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 4 || $1 == "" || $2 == "" || seen[$1]++ {bad=1}
  $3 !~ /^(init|plan|document|review|none)$/ {bad=1}
  $4 !~ /^(-|none|refresh|domain|sdd|sources|iso|research|decision|code|drift|deep|closeout|experience)$/ {bad=1}
  {n++}
  END {exit (bad || !n)}
' "$FIXTURES"; then
  echo "fixtures must contain unique IDs and four fields with a valid skill and mode" >&2
  exit 2
fi

if [ -n "${SKILL_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$SKILL_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-skill-bench.XXXXXX") || exit 2
  trap 'rm -rf "$results_dir"' EXIT
fi
results_dir=$(CDPATH='' cd -- "$results_dir" && pwd)
workdir="$results_dir/project"
mkdir -p "$workdir"
[ -d "$workdir/.git" ] || git -C "$workdir" init -q 2>/dev/null || true
: > "$results_dir/pass"
: > "$results_dir/fail"
: > "$results_dir/error"

n=0
printf 'id\texpected\tgot\tverdict\targs\n'
while IFS="$TAB" read -r id message skill mode || [ -n "$id" ]; do
  case "$id" in ''|\#*) continue ;; esac
  case "$id$message$skill$mode" in *[![:space:]]*) ;; *) continue ;; esac
  n=$((n + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  printf '%s' "$message" > "$results_dir/$n.message"
  set -- -p --plugin-dir "$PLUGIN_DIR" --tools Skill --setting-sources project \
    --settings '{"disableAllHooks":true}' --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
    --no-session-persistence --output-format stream-json --verbose --max-turns 3
  if [ -n "${SKILL_BENCH_MODEL:-}" ]; then set -- "$@" --model "$SKILL_BENCH_MODEL"; fi
  # A max-turns stop exits non-zero by design; only an empty transcript is a CLI failure.
  (cd "$workdir" && claude "$@" < "$results_dir/$n.message") \
    > "$results_dir/$n.jsonl" 2> "$results_dir/$n.stderr" || true
  expected="$skill $mode"
  if ! jq -e -s 'map(select(.type == "system" and .subtype == "init")) | length > 0' "$results_dir/$n.jsonl" >/dev/null 2>&1; then
    printf '%s\t%s\tnone\tERROR\tCLI failed; see %s/%s.stderr and .jsonl\n' "$id" "$expected" "$results_dir" "$n"
    echo "$id" >> "$results_dir/error"
    continue
  fi
  call=$(jq -c -s '[.[] | select(.type == "assistant") | .message.content[]? | select(.type == "tool_use" and .name == "Skill") | .input] | first // empty' "$results_dir/$n.jsonl")
  if [ -z "$call" ]; then
    got=none; args=""
  else
    got=$(printf '%s' "$call" | jq -r '.skill // ""')
    args=$(printf '%s' "$call" | jq -r '.args // ""' | tr '\t\n' '  ')
    case "$got" in
      archcore:*) got=${got#archcore:} ;;
      *) got="other:$got" ;;
    esac
  fi
  first_word=$(printf '%s' "$args" | sed -E 's/^[[:space:]]*([a-z-]+).*/\1/')
  verdict=FAIL
  # A negative fixture only forbids an archcore skill; a built-in skill may still fire.
  if [ "$skill" = none ]; then
    case "$got" in none|other:*) verdict=pass ;; esac
  elif [ "$got" = "$skill" ]; then
    case "$mode" in
      -) verdict=pass ;;
      none) case "$first_word" in refresh|domain|sdd|sources|iso|research|decision|code|drift|deep|closeout|experience) ;; *) verdict=pass ;; esac ;;
      *) [ "$first_word" = "$mode" ] && verdict=pass ;;
    esac
  fi
  if [ "$verdict" = pass ]; then echo "$id" >> "$results_dir/pass"; else echo "$id" >> "$results_dir/fail"; fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$expected" "$got" "$verdict" "$args"
done < "$FIXTURES"

passed=$(wc -l < "$results_dir/pass" | tr -d ' ')
failed=$(wc -l < "$results_dir/fail" | tr -d ' ')
errors=$(wc -l < "$results_dir/error" | tr -d ' ')
echo "# skill-bench: $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
