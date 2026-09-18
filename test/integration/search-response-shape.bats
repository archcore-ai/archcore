#!/usr/bin/env bats
# Real CLI, stdio MCP, and temporary storage. This does not simulate LLM gates.
#
# skills/_shared/globals.md and the assistant's "Large or partial results"
# paragraph tell the model how to read a large search response: `hits` and
# `index` ahead of `results`, `truncated` under the response byte budget, and
# `body_truncated` on a shortened body. These tests pin the CLI behavior that
# text relies on, against a project that mounts one global source.
#
# The fields arrived in CLI 0.8.5. An older CLI sends none of them and the
# guidance is data-gated for it, so an older binary skips instead of failing.

LARGE_CORPUS=150

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  common_setup
  require_cli 0.8.5
  seed_global_source
  mcp_start
  mcp_tool create_document '{"type":"adr","filename":"use-postgres","title":"Use Postgres","status":"draft","content":"## Decision\nUse Postgres for billing storage.\n"}'
}

teardown() {
  mcp_stop
}

require_cli() {
  local binary answer
  binary=$(command -v "${ARCHCORE_BIN:-archcore}") || skip "Archcore CLI not installed (set ARCHCORE_BIN)"
  # cli-gte resolves `archcore` through PATH, so put the binary under test first.
  mkdir -p "$BATS_TEST_TMPDIR/cli-under-test"
  ln -sf "$binary" "$BATS_TEST_TMPDIR/cli-under-test/archcore"
  answer=$(PATH="$BATS_TEST_TMPDIR/cli-under-test:$PATH" "$PLUGIN_ROOT/bin/cli-gte" "$1")
  [ "$answer" = "yes" ] || skip "CLI older than $1 sends no hits, index, or body_truncated"
}

# settings.json must exist before the server starts; init_project keeps it.
seed_global_source() {
  local root="$BATS_TEST_TMPDIR/project/.archcore" global long i
  global="$root/global/company"
  mkdir -p "$global"
  printf '%s' '{"sync":"none","globals":[{"id":"company","path":".archcore/global/company"}]}' > "$root/settings.json"
  long=$(printf 'Postgres is the org default database. %.0s' $(seq 1 3000))
  printf -- '---\ntitle: "Org Database Default"\nstatus: accepted\n---\n\n## Context\n%s\n' "$long" \
    > "$global/org-database.adr.md"
  case "$BATS_TEST_DESCRIPTION" in
    *"byte budget"*|*"advancing offset"*)
      for i in $(seq 1 "$LARGE_CORPUS"); do
        printf -- '---\ntitle: "Org Topic %s with a descriptive title that pads the row"\nstatus: accepted\n---\n\n## Context\nPostgres is the org default database for topic %s.\n' "$i" "$i" \
          > "$global/org-topic-$i.adr.md"
      done
      ;;
  esac
}

@test "search puts hits and index before results" {
  mcp_tool search_documents '{"content":"postgres"}'
  mcp_assert 'keys_unsorted as $k | ($k | index("results")) as $r
    | ($k | index("hits")) < $r and ($k | index("index")) < $r and ($k | index("coverage")) < $r'
  mcp_assert '.index | length == 2 and all(has("path") and has("title") and has("source_id"))'
}

@test "hits counts every source before the limit cut" {
  mcp_tool search_documents '{"content":"postgres","limit":1}'
  mcp_assert '.hits == {"company":1,"local":1}'
  mcp_assert '(.index | length) == 1 and (.results | length) == 1'
  # The global match did not reach the page; only hits still names it.
  mcp_assert '[.index[].source_id] == ["local"]'
}

@test "a global match carries its source fields" {
  mcp_tool search_documents '{"content":"postgres","source":"global"}'
  mcp_assert '.hits == {"company":1}'
  mcp_assert '.results[0] | .source_kind == "global" and .source_id == "company" and .read_only == true'
}

@test "full mode shortens a long body and get_document returns the rest" {
  local shortened
  mcp_tool search_documents '{"content":"postgres","mode":"full"}'
  mcp_assert '[.results[] | select(.source_id == "local")][0] | (has("body_truncated") | not) and (.body | contains("billing storage"))'
  mcp_assert '[.results[] | select(.source_id == "company")][0] | .body_truncated == true and (.body | length) > 0'
  shortened=$(jq '[.results[] | select(.source_id == "company")][0].body | length' <<< "$MCP_RESULT")
  mcp_get ".archcore/global/company/org-database.adr.md"
  mcp_assert --argjson shortened "$shortened" '(.content | length) > $shortened'
}

@test "the byte budget sets truncated and keeps fewer results than index" {
  mcp_tool search_documents '{"content":"postgres","limit":200}'
  mcp_assert '.truncated == true'
  mcp_assert '(.results | length) < (.index | length)'
  mcp_assert --argjson corpus "$LARGE_CORPUS" '.hits.company == ($corpus + 1) and .hits.local == 1'
}

@test "list_documents reaches every document by advancing offset by returned" {
  local offset=0 seen=0 pages=0 total
  while :; do
    mcp_tool list_documents "$(jq -cn --argjson offset "$offset" '{limit:500,offset:$offset}')"
    mcp_assert '.by_source.local == 1'
    total=$(jq '.total' <<< "$MCP_RESULT")
    seen=$((seen + $(jq '.returned' <<< "$MCP_RESULT")))
    pages=$((pages + 1))
    [ "$(jq '.truncated' <<< "$MCP_RESULT")" = "true" ] || break
    [ "$(jq '.returned' <<< "$MCP_RESULT")" -gt 0 ] || fail "truncated page returned zero documents"
    offset=$seen
    [ "$pages" -lt 20 ] || fail "pagination did not finish in 20 pages"
  done
  [ "$total" -eq $((LARGE_CORPUS + 2)) ] || fail "expected $((LARGE_CORPUS + 2)) documents, total is $total"
  [ "$seen" -eq "$total" ] || fail "paging by returned reached $seen of $total documents"
  [ "$pages" -gt 1 ] || fail "corpus fit one page; the byte budget was not exercised"
}
