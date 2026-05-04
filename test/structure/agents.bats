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
  grep -q 'recent accepted decisions' "$file"
}

@test "auditor has knowledge tree bootstrap preamble" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.md"
  grep -q 'First Step — Bootstrap Knowledge Tree' "$file"
  grep -q 'list_documents' "$file"
  grep -q 'list_relations' "$file"
  grep -q 'subagent-knowledge-tree-bootstrap.adr' "$file"
  grep -q 'recent accepted decisions' "$file"
}

# --- Codex TOML subagents ---

@test "archcore-assistant.toml exists (Codex format)" {
  [ -f "$PLUGIN_ROOT/agents/archcore-assistant.toml" ]
}

@test "archcore-auditor.toml exists (Codex format)" {
  [ -f "$PLUGIN_ROOT/agents/archcore-auditor.toml" ]
}

@test "auditor.toml has read-only sandbox" {
  grep -qE '^sandbox_mode = "read-only"' "$PLUGIN_ROOT/agents/archcore-auditor.toml"
}

@test "auditor.toml disables mutating MCP tools" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.toml"
  grep -q 'mcp__archcore__create_document' "$file"
  grep -q 'mcp__archcore__update_document' "$file"
  grep -q 'mcp__archcore__remove_document' "$file"
  grep -q 'mcp__archcore__add_relation' "$file"
  grep -q 'mcp__archcore__remove_relation' "$file"
}

@test "assistant.toml uses workspace-write sandbox" {
  grep -qE '^sandbox_mode = "workspace-write"' "$PLUGIN_ROOT/agents/archcore-assistant.toml"
}

@test "TOML agents have required top-level fields" {
  for f in archcore-assistant.toml archcore-auditor.toml; do
    local file="$PLUGIN_ROOT/agents/$f"
    grep -qE '^name = ' "$file" || fail "$f: missing name"
    grep -qE '^description = ' "$file" || fail "$f: missing description"
    grep -qE '^developer_instructions = ' "$file" || fail "$f: missing developer_instructions"
    grep -qE '^sandbox_mode = ' "$file" || fail "$f: missing sandbox_mode"
  done
}

@test "TOML agents preserve knowledge tree bootstrap preamble" {
  for f in archcore-assistant.toml archcore-auditor.toml; do
    local file="$PLUGIN_ROOT/agents/$f"
    grep -q 'First Step — Bootstrap Knowledge Tree' "$file" || fail "$f: missing bootstrap preamble"
    grep -q 'subagent-knowledge-tree-bootstrap.adr' "$file" || fail "$f: missing ADR reference"
  done
}
