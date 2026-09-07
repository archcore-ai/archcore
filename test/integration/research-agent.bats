#!/usr/bin/env bats
# Live model + real MCP tests. Explicit target only; never part of default CI.

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  bats_require_minimum_version 1.5.0
  common_setup
  command -v claude >/dev/null 2>&1 || skip "Claude Code CLI not installed"
  command -v "${ARCHCORE_BIN:-archcore}" >/dev/null 2>&1 || skip "Archcore CLI not installed (set ARCHCORE_BIN)"
  mcp_start
  MCP_CONFIG="$BATS_TEST_TMPDIR/mcp.json"
  jq -n --arg binary "$MCP_BINARY" --arg root "$MCP_ROOT" \
    '{mcpServers:{archcore:{command:$binary,args:["mcp","--project",$root]}}}' > "$MCP_CONFIG"
  RESULTS="${RESEARCH_AGENT_OUTPUT_DIR:-$BATS_TEST_TMPDIR/results}/$BATS_TEST_NUMBER"
  mkdir -p "$RESULTS"
}

teardown() {
  mcp_stop
}

invoke_assistant() {
  local prompt="$1" agent="${2:-archcore-assistant}"
  local schema_args=()
  if [ "$agent" = archcore-auditor ]; then
    schema_args=(--json-schema '{"type":"object","properties":{"documents":{"type":"integer"},"orphans":{"type":"integer"},"orphan_paths":{"type":"array","items":{"type":"string"}}},"required":["documents","orphans","orphan_paths"],"additionalProperties":false}')
  fi
  printf '%s\n' "$prompt" > "$RESULTS/prompt.txt"
  local status=0
  (
    cd "$MCP_ROOT" || exit 1
    claude -p --plugin-dir "$PLUGIN_ROOT" --agent "archcore:$agent" \
      --tools Read Grep Glob --allowedTools 'mcp__archcore__*' Read Grep Glob \
      --permission-mode dontAsk --strict-mcp-config --mcp-config "$MCP_CONFIG" \
      --setting-sources user --no-session-persistence --output-format json \
      "${schema_args[@]}" \
      < "$RESULTS/prompt.txt"
  ) > "$RESULTS/response.json" 2> "$RESULTS/stderr.txt" || status=$?
  assert_equal "$status" 0 || return 1
  jq -e '.is_error == false' "$RESULTS/response.json" >/dev/null \
    || { fail "agent call failed; inspect $RESULTS/response.json"; return 1; }
}

@test "live assistant requests a missing probe without creating fallback documents" {
  invoke_assistant "Parent task: record a standalone evidence material. Absolute plugin root: $PLUGIN_ROOT. No vocabulary probe result is available in this delegation. Supplied material: test://measurement/missing-probe, accessed 2026-09-07, unknown publisher and publication date, extract: 37 retries completed. No local consumer is identified."
  run jq -r .result "$RESULTS/response.json"
  assert_output --partial 'needs-vocabulary-probe'
  mcp_tool list_documents '{}'
  mcp_assert '.documents == []'
  mcp_tool list_relations '{}'
  mcp_assert '.relations == []'
}

@test "live assistant records one standalone evidence draft without a consumer edge" {
  invoke_assistant "Parent task: execute document evidence for exactly one supplied measurement. Current invocation vocabulary probe: yes (cli-gte 0.8.3). Absolute plugin root: $PLUGIN_ROOT. User authorizes one draft. Supplied material: Address=test://measurement/retry-run-1; Access date=2026-09-07; Publication date=unknown; Publisher=unknown; Extract=The supplied run completed 37 retries with zero duplicate evidence records.; Notes=This is a user-supplied test measurement; no independent verification was performed. No local consumer exists. Follow the standalone evidence path in skills/_shared/tracks/research.md. Do not invent provenance, create an investigation, or add a relation."
  mcp_tool list_documents '{}'
  mcp_assert '.documents | length == 1'
  mcp_assert '.documents[0].type == "evidence" and .documents[0].status == "draft" and .documents[0].category == "knowledge"'
  local path
  path=$(jq -r '.documents[0].path' <<< "$MCP_RESULT")
  mcp_get "$path"
  mcp_assert '.content | contains("## Locator") and contains("## Extract") and contains("## Notes") and contains("test://measurement/retry-run-1") and contains("37 retries") and (contains("<!-- archcore:track") | not)'
  mcp_tool list_relations '{}'
  mcp_assert '.relations == []'
}

@test "live auditor includes an orphan beyond the default inventory page" {
  local i previous="" path
  for i in $(seq 1 100); do
    mcp_create doc "corpus-$(printf '%03d' "$i")"
    path="$MCP_PATH"
    if [ -n "$previous" ]; then mcp_edge "$path" "$previous" related; fi
    previous="$path"
  done
  mcp_create doc zzz-last-page-orphan
  local orphan="$MCP_PATH"
  mcp_tool list_documents '{"limit":100}'
  mcp_assert '.truncated == true and .returned == 100 and all(.documents[]; .path != $path)' --arg path "$orphan"
  invoke_assistant "Audit only local inventory and relation completeness. Parent git evidence: no code changes in this temporary project; no drift analysis is requested. Absolute plugin root: $PLUGIN_ROOT. Return one JSON object with keys documents (the exact document count), orphans (the exact orphan count), and orphan_paths (every orphan's full .archcore/ path). Do not read document bodies or modify anything." archcore-auditor
  local report
  # This test checks inventory completeness, not surrounding report prose.
  # Parse exactly one JSON object; concatenated objects remain invalid JSON.
  report=$(jq -ce '.structured_output // (.result | capture("(?s)(?<report>\\{.*\\})").report | fromjson)' "$RESULTS/response.json") \
    || { fail "auditor returned no valid inventory object"; return 1; }
  jq -e --arg path "$orphan" '.documents == 101 and .orphans == 1 and .orphan_paths == [$path]' >/dev/null <<< "$report" \
    || { fail "auditor did not cover the full inventory: $report"; return 1; }
}
