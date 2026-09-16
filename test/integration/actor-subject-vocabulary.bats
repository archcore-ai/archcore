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

@test "MCP exposes the actor-subject vocabulary among 23 types and keeps seven relation types" {
  mcp_call tools/list '{}'
  mcp_assert '[.tools[] | select(.name == "create_document") | .inputSchema.properties.type.enum[]] | contains(["scenario","journey"])'
  mcp_assert '[.tools[] | select(.name == "create_document") | .inputSchema.properties.type.enum[]] | length == 23'
  mcp_assert '[.tools[] | select(.name == "add_relation") | .inputSchema.properties.type.enum[]] | length == 7'
}

@test "MCP scenario template has the five sections and belongs to knowledge" {
  mcp_create scenario refund-approval
  mcp_assert '.category == "knowledge"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Subject","Actors","Flows","Examples","Open Questions"])'
  mcp_assert '.content | contains("Anchors:") and contains("Illustrates:")'
}

@test "MCP journey template has the four sections and belongs to vision" {
  mcp_create journey beginner-path
  mcp_assert '.category == "vision"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Intent","Actors","Journeys","Open Questions"])'
  mcp_assert '.content | contains("In order to")'
}

@test "list_documents filters by scenario and journey" {
  mcp_create scenario refund-approval
  mcp_create journey beginner-path
  mcp_tool list_documents '{"types":["scenario","journey"]}'
  mcp_assert '[.documents[].type] | sort == ["journey","scenario"]'
}
