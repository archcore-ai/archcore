#!/usr/bin/env bats
# Structure tests: validate agent files

setup() {
  load '../helpers/common'
  common_setup
}

@test "archcore-assistant.md exists" {
  [ -f "$PLUGIN_ROOT/agents/archcore-assistant.md" ]
}

@test "archcore-auditor.md exists" {
  [ -f "$PLUGIN_ROOT/agents/archcore-auditor.md" ]
}

@test "assistant has required frontmatter fields" {
  local file="$PLUGIN_ROOT/agents/archcore-assistant.md"
  head -20 "$file" | grep -q '^name:'
  head -20 "$file" | grep -q '^description:'
  head -20 "$file" | grep -q '^model:'
  head -20 "$file" | grep -q '^maxTurns:'
  head -20 "$file" | grep -q '^tools:'
}

@test "auditor has required frontmatter fields" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.md"
  head -20 "$file" | grep -q '^name:'
  head -20 "$file" | grep -q '^description:'
  head -20 "$file" | grep -q '^model:'
  head -20 "$file" | grep -q '^maxTurns:'
  head -20 "$file" | grep -q '^tools:'
}

@test "auditor is a background agent" {
  head -20 "$PLUGIN_ROOT/agents/archcore-auditor.md" | grep -q '^background: true'
}

@test "auditor has only read-only MCP tools" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.md"
  # Auditor should NOT have create/update/remove/add_relation/remove_relation tools
  if grep -q 'mcp__archcore__create_document\|mcp__archcore__update_document\|mcp__archcore__remove_document' "$file"; then
    fail "Auditor has write MCP tools"
  fi
}

@test "assistant has write MCP tools" {
  local file="$PLUGIN_ROOT/agents/archcore-assistant.md"
  grep -q 'mcp__archcore__create_document' "$file"
  grep -q 'mcp__archcore__update_document' "$file"
}

@test "assistant has knowledge tree bootstrap preamble" {
  local file="$PLUGIN_ROOT/agents/archcore-assistant.md"
  grep -q 'First Step — Bootstrap Knowledge Tree' "$file"
  grep -q 'list_documents' "$file"
  grep -q 'list_relations' "$file"
  grep -q 'subagent-knowledge-tree-bootstrap.adr' "$file"
}

@test "auditor has knowledge tree bootstrap preamble" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.md"
  grep -q 'First Step — Bootstrap Knowledge Tree' "$file"
  grep -q 'list_documents' "$file"
  grep -q 'list_relations' "$file"
  grep -q 'subagent-knowledge-tree-bootstrap.adr' "$file"
}
