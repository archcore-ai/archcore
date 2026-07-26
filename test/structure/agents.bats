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

@test "auditor.toml denies EVERY mutating MCP tool under both namings" {
  # The deny-list is the read-only auditor's ONLY enforcement surface for MCP
  # tools: sandbox_mode=read-only constrains this agent's shell, not the MCP
  # server process that executes the tool, and the list fails OPEN — any
  # mutating tool absent here is callable. So it MUST be exhaustive over the
  # server's whole write surface, not just the historical document/relation
  # ops (init_project + install_host_config write outside .archcore/).
  local file="$PLUGIN_ROOT/agents/archcore-auditor.toml"
  local tool
  for tool in create_document update_document remove_document \
              add_relation remove_relation \
              init_project install_host_config; do
    grep -qF "\"mcp__archcore__${tool}\"" "$file" \
      || fail "auditor.toml missing deny for mcp__archcore__${tool}"
    grep -qF "\"mcp__plugin_archcore_archcore__${tool}\"" "$file" \
      || fail "auditor.toml missing deny for mcp__plugin_archcore_archcore__${tool}"
  done
}

@test "agent tool lists cover BOTH project and plugin MCP tool naming" {
  # Same premise as the hooks matcher test (host-wiring-parity.adr.md): a
  # project .mcp.json yields mcp__archcore__*, a plugin-bundled server yields
  # mcp__plugin_archcore_archcore__*. Allow-lists (md tools:) missing a twin
  # fail closed but lose capability; deny-lists (toml disabled_tools) missing
  # a twin fail OPEN — the read-only auditor could mutate. Every archcore
  # tool mentioned in an agent tool list must appear under both namings.
  local file tool suffix
  for file in "$PLUGIN_ROOT/agents/archcore-assistant.md" \
              "$PLUGIN_ROOT/agents/archcore-auditor.md" \
              "$PLUGIN_ROOT/agents/archcore-auditor.toml"; do
    while IFS= read -r tool; do
      case "$tool" in
        mcp__plugin_archcore_archcore__*)
          suffix="${tool#mcp__plugin_archcore_archcore__}"
          grep -q "mcp__archcore__${suffix}\b" "$file" \
            || fail "$(basename "$file"): lists $tool but not its project-naming twin"
          ;;
        mcp__archcore__*)
          suffix="${tool#mcp__archcore__}"
          grep -q "mcp__plugin_archcore_archcore__${suffix}\b" "$file" \
            || fail "$(basename "$file"): lists $tool but not its plugin-naming twin"
          ;;
      esac
    done < <(grep -oE 'mcp__(plugin_archcore_)?archcore__[a-z_]+' "$file" | sort -u)
  done
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

@test "all agent files document the archcore init CLI recovery path" {
  # When MCP tools are unavailable (CLI not installed or unresolvable on PATH),
  # mcp__archcore__init_project cannot fire. Agents must fall back to telling
  # the user how to recover: install the CLI and run `archcore init` in the
  # terminal. This must be present in every agent surface across hosts —
  # .md (Claude/Cursor) and .toml (Codex) — or the fallback drifts silently.
  for f in archcore-assistant.md archcore-auditor.md archcore-assistant.toml archcore-auditor.toml; do
    local file="$PLUGIN_ROOT/agents/$f"
    grep -q 'archcore init' "$file" \
      || fail "$f: missing 'archcore init' recovery instruction for MCP-unavailable case"
  done
}
