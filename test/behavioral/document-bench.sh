#!/bin/sh
# Behavioral document-entry bench. Runs model calls only when explicitly requested.
# Measures how /archcore:document classifies a request into a mode and a type
# under the command entry grammar, including requests that carry no mode word.
# DOCUMENT_BENCH_MODEL selects the model; DOCUMENT_BENCH_LIMIT limits fixture count.
# DOCUMENT_BENCH_FIXTURES overrides input; DOCUMENT_BENCH_OUTPUT_DIR retains raw replies.
# Exit 1: classification/format mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${DOCUMENT_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/document-bench.tsv}
SKILL="$REPO_ROOT/plugins/archcore/skills/document/SKILL.md"
GATES="$REPO_ROOT/plugins/archcore/skills/_shared/gate-contract.md"
TAB=$(printf '\t')

[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI not found on PATH" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${DOCUMENT_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "DOCUMENT_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 5 || $1 == "" || $2 == "" || $3 == "" || seen[$1]++ {bad=1}
  $4 !~ /^(decision|code|research|unclear|plan)$/ {bad=1}
  $5 !~ /^(-|none|adr|rfc|rule|spec|doc|guide|scenario|research|rnd|evidence)$/ {bad=1}
  {n++}
  END {exit (bad || !n)}
' "$FIXTURES"; then
  echo "fixtures must contain unique IDs and five fields with a valid mode and type" >&2
  exit 2
fi

skill_text=$(cat "$SKILL")
gates_text=$(sed -n '/^## Entry terms/,/^## Gate record template/p' "$GATES")
if [ -n "${DOCUMENT_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$DOCUMENT_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-document-bench.XXXXXX") || exit 2
  trap 'rm -rf "$results_dir"' EXIT
fi
results_dir=$(CDPATH='' cd -- "$results_dir" && pwd)
: > "$results_dir/pass"
: > "$results_dir/fail"
: > "$results_dir/error"

n=0
printf 'id\texpected\tgot\tverdict\treply\n'
while IFS="$TAB" read -r id request grounding mode type || [ -n "$id" ]; do
  case "$id" in ''|\#*) continue ;; esac
  case "$id$request$grounding$mode$type" in *[![:space:]]*) ;; *) continue ;; esac
  n=$((n + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  printf '%s\n' \
    "You are the /archcore:document skill defined below. Apply it literally, but do not execute any gate." \
    "Decide only where the request enters and which document type the entry gate would select from the request text alone." \
    "Output EXACTLY one line and nothing else: entry: <mode> <type>" \
    "<mode> is one of: decision, code, research — the track this skill enters; unclear — the skill would ask its classifying question; plan — the request belongs to /archcore:plan." \
    "<type> is the document type the request settles (adr, rfc, rule, spec, doc, guide, scenario, research, rnd, evidence), or none when a later gate needs code evidence or an answer to choose it." \
    "" \
    "--- SKILL: skills/document/SKILL.md ---" "$skill_text" \
    "--- CONTRACT EXCERPT: skills/_shared/gate-contract.md ---" "$gates_text" \
    "--- REQUEST (the text after /archcore:document) ---" "$request" \
    "--- GROUNDING RESULT (already established; do not re-derive) ---" "$grounding" \
    > "$results_dir/$n.prompt"
  set -- -p --tools '' --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
    --setting-sources user --no-session-persistence --output-format json
  if [ -n "${DOCUMENT_BENCH_MODEL:-}" ]; then set -- "$@" --model "$DOCUMENT_BENCH_MODEL"; fi
  cli_status=0
  (cd "$results_dir" && claude "$@" < "$results_dir/$n.prompt") \
    > "$results_dir/$n.json" 2> "$results_dir/$n.stderr" || cli_status=$?
  expected="$mode $type"
  if [ "$cli_status" -ne 0 ] || ! jq -e '.is_error == false and (.result | type == "string")' "$results_dir/$n.json" >/dev/null 2>&1; then
    printf '%s\t%s\tnone\tERROR\tCLI failed; see %s/%s.stderr and .json\n' "$id" "$expected" "$results_dir" "$n"
    echo "$id" >> "$results_dir/error"
    continue
  fi
  out=$(jq -r '.result' "$results_dir/$n.json")
  case "$out" in '`entry:'*'`') out=${out#'`'}; out=${out%'`'} ;; esac
  got_mode=$(printf '%s\n' "$out" | sed -n 's/^entry: \([a-z][a-z]*\) \([a-z-][a-z-]*\)$/\1/p')
  got_type=$(printf '%s\n' "$out" | sed -n 's/^entry: \([a-z][a-z]*\) \([a-z-][a-z-]*\)$/\2/p')
  verdict=FAIL
  if [ "$(printf '%s\n' "$out" | wc -l | tr -d ' ')" -eq 1 ] && [ "$got_mode" = "$mode" ]; then
    if [ "$type" = "-" ] || [ "$got_type" = "$type" ]; then verdict=pass; fi
  fi
  if [ "$verdict" = pass ]; then echo "$id" >> "$results_dir/pass"; else echo "$id" >> "$results_dir/fail"; fi
  reply=$(printf '%s' "$out" | tr '\t\n' '  ')
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$expected" "${got_mode:-none} ${got_type:-none}" "$verdict" "$reply"
done < "$FIXTURES"

passed=$(wc -l < "$results_dir/pass" | tr -d ' ')
failed=$(wc -l < "$results_dir/fail" | tr -d ' ')
errors=$(wc -l < "$results_dir/error" | tr -d ' ')
echo "# document-bench: $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
