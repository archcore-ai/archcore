#!/usr/bin/env bats
# Real CLI, stdio MCP, and temporary storage. This does not simulate LLM gates.

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  common_setup
  mcp_start
}

teardown() {
  mcp_stop
}

@test "MCP exposes research vocabulary and the seven directed relation types" {
  mcp_call tools/list '{}'
  mcp_assert '[.tools[] | select(.name == "create_document") | .inputSchema.properties.type.enum[]] | contains(["research","evidence","rnd"])'
  mcp_assert '[.tools[] | select(.name == "add_relation") | .inputSchema.properties.type.enum[]] | sort == (["related","implements","extends","depends_on","supports","contradicts","supersedes"] | sort)'
}

@test "MCP research template has coverage sections and belongs to vision" {
  mcp_create research territory
  mcp_assert '.category == "vision"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Goal","Scope","Coverage","Sources","Findings","Synthesis","Open Gaps"])'
}

@test "MCP evidence template records provenance and permits a standalone material" {
  mcp_create evidence material
  mcp_assert '.category == "knowledge"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Locator","Extract","Notes"])'
  local key
  for key in 'Address:' 'Access date:' 'Publication date:' 'Publisher:'; do
    mcp_assert '.content | contains($key)' --arg key "$key"
  done
  mcp_tool list_relations '{}'
  mcp_assert '.relations == []'
}

@test "MCP preserves the legacy rnd recommendation template" {
  mcp_create rnd decision-probe
  mcp_assert '.category == "vision"'
  mcp_get "$MCP_PATH"
  mcp_assert '[.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Research Goal","Context & Trigger","Questions / Hypotheses","Approach","Findings","Implications","Recommendation","Next Action","Risks & Unknowns","Related Materials"]'
}

@test "MCP failed evidence edge leaves one material available for retry" {
  mcp_create evidence material
  local material="$MCP_PATH"
  mcp_edge "$material" .archcore/missing.research.md supports true
  [[ "$MCP_TEXT" == *'not found'* || "$MCP_TEXT" == *'does not exist'* ]] \
    || { fail "missing endpoint was not rejected: $MCP_TEXT"; return 1; }
  mcp_tool list_relations '{}'
  mcp_assert '.relations == []'
  mcp_tool list_documents '{"types":["evidence"]}'
  mcp_assert '[.documents[].path] == [$path]' --arg path "$material"
  mcp_create research territory
  mcp_edge "$material" "$MCP_PATH" supports
  mcp_tool list_relations '{}'
  mcp_assert '.relations | length == 1'
  mcp_get "$material"
  mcp_assert '.type == "evidence"'
}

@test "MCP reads back directed edges and deduplicates the first evidence edge" {
  mcp_create research territory
  local research="$MCP_PATH"
  mcp_create evidence material
  local material="$MCP_PATH"
  mcp_create evidence new-material
  local newer="$MCP_PATH"
  mcp_create rnd decision-probe
  local rnd="$MCP_PATH"
  mcp_edge "$material" "$research" supports
  mcp_edge "$material" "$research" contradicts
  mcp_edge "$newer" "$material" supersedes
  mcp_edge "$rnd" "$research" depends_on
  mcp_edge "$material" "$research" supports
  mcp_tool list_relations '{}'
  mcp_assert '[.relations[] | [.source,.target,.type]] | sort == ([
    ["material.evidence.md","territory.research.md","supports"],
    ["material.evidence.md","territory.research.md","contradicts"],
    ["new-material.evidence.md","material.evidence.md","supersedes"],
    ["decision-probe.rnd.md","territory.research.md","depends_on"]] | sort)'
  mcp_get "$research"
  mcp_assert '[.incoming_relations[].type] | sort == (["supports","contradicts","depends_on"] | sort)'
  mcp_tool list_documents '{"types":["evidence"]}'
  mcp_assert '.documents | length == 2'
  mcp_tool list_documents '{"category":"vision"}'
  mcp_assert '[.documents[].type] | sort == ["research","rnd"]'
  mcp_tool list_documents '{"category":"knowledge"}'
  mcp_assert '[.documents[].type] == ["evidence","evidence"]'
}

@test "MCP update preserves pending track state and the legacy rnd type" {
  mcp_create rnd decision-probe
  local path="$MCP_PATH" state content args
  mcp_get "$path"
  state=$'\n<!-- archcore:track\ntrack: research\ngate: research.gather\ndeferred: pending edge\nartifact_type: rnd\n-->\n'
  content=$(jq -r '.content' <<< "$MCP_RESULT")
  args=$(jq -cn --arg path "$path" --arg content "$content$state" '{path:$path,content:$content}')
  mcp_tool update_document "$args"
  mcp_get "$path"
  mcp_assert '.type == "rnd" and (.content | contains($state))' --arg state "$state"
}
