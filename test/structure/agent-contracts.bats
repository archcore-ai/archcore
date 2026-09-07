#!/usr/bin/env bats
# Agent capability and instruction contracts, including host-format parity.

setup() {
  load '../helpers/common'
  common_setup
}

md_body() {
  awk 'BEGIN { fm=0; started=0 } /^---$/ && fm<2 { fm++; next }
    fm==2 { if (!started && $0=="") next; started=1; print }' "$1"
}

@test "both agents allow the complete read tool set under all host namings" {
  local agent tool prefix frontmatter
  for agent in archcore-assistant archcore-auditor; do
    frontmatter=$(awk 'NR==1 {next} /^---$/ {exit} {print}' "$PLUGIN_ROOT/agents/$agent.md")
    for tool in list_documents search_documents get_document list_relations; do
      for prefix in mcp__archcore__ mcp__plugin_archcore_archcore__ archcore-; do
        grep -Fqx "  - $prefix$tool" <<< "$frontmatter" \
          || { fail "$agent lacks $prefix$tool in its allowlist"; return 1; }
      done
    done
  done
}

@test "Codex agent instructions and descriptions match the Markdown agents" {
  local agent md toml expected actual description
  for agent in archcore-assistant archcore-auditor; do
    md="$PLUGIN_ROOT/agents/$agent.md"
    toml="$PLUGIN_ROOT/agents/$agent.toml"
    expected=$(md_body "$md")
    # The repository stores instructions as a TOML multiline literal string.
    actual=$(awk -v delimiter="'''" '
      $0 == "developer_instructions = " delimiter {body=1; next}
      body && $0 == delimiter {exit}
      body {print}
    ' "$toml")
    assert_equal "$actual" "$expected" || return 1
    description=$(awk '/^description: >$/ {body=1; next} body && /^  / {sub(/^  /, ""); result=result (result=="" ? "" : " ") $0; next} body {exit} END {print result}' "$md")
    [ -n "$description" ] || { fail "$md has no folded description block"; return 1; }
    actual=$(sed -n 's/^description = "\(.*\)"$/\1/p' "$toml")
    assert_equal "$actual" "$description" || return 1
  done
}

@test "inventory consumers require pagination and reject a non-progressing page" {
  local file
  for file in "$PLUGIN_ROOT/agents/archcore-assistant.md" "$PLUGIN_ROOT/agents/archcore-auditor.md" "$PLUGIN_ROOT/skills/review/SKILL.md"; do
    grep -Fq 'truncated: false' "$file" || { fail "$file never finishes pagination"; return 1; }
    grep -Fq '`offset`' "$file" || { fail "$file has no page offset"; return 1; }
    grep -Fq '`returned`' "$file" || { fail "$file does not advance by returned rows"; return 1; }
    grep -Fq 'truncated page returns zero documents' "$file" || { fail "$file can loop on an empty page"; return 1; }
  done
}

@test "research delegation passes the current probe and never invents an old CLI" {
  local file
  for file in "$PLUGIN_ROOT/skills/plan/SKILL.md" "$PLUGIN_ROOT/skills/document/SKILL.md" "$PLUGIN_ROOT/agents/archcore-assistant.md"; do
    grep -Fq 'needs-vocabulary-probe' "$file" || { fail "$file has no missing-handoff recovery"; return 1; }
    grep -Fq 'absolute plugin root' "$file" || { fail "$file omits the plugin root"; return 1; }
  done
  grep -Fq 'Missing handoff is not evidence of an old CLI.' "$PLUGIN_ROOT/agents/archcore-assistant.md" \
    || { fail "missing handoff can trigger a false legacy fallback"; return 1; }
}

@test "auditor uses caller git evidence and respects evidence status conventions" {
  local file="$PLUGIN_ROOT/agents/archcore-auditor.md"
  grep -Fq 'git history supplied by the caller' "$file" || { fail "auditor lacks a git evidence source"; return 1; }
  grep -Fq 'as unverified' "$file" || { fail "missing history can appear verified"; return 1; }
  if grep -Fq '`Bash`' "$file"; then fail "auditor prescribes a tool outside its allowlist"; return 1; fi
  grep -Fq 'complete evidence draft can await a second reader' "$file" || { fail "evidence draft can be flagged as stale status"; return 1; }
  tr '\n' ' ' < "$file" | grep -Fq 'placeholders alone are not a defect' \
    || { fail "evidence provenance placeholders can be flagged as defects"; return 1; }
}

@test "plan route names retain Derivation; research entries sit on plan and document, evidence only on document" {
  local file hint
  grep -Fq 'Fix that route; run Derivation to compute its package' "$PLUGIN_ROOT/skills/plan/SKILL.md" \
    || { fail "named routes can skip Derivation"; return 1; }
  for file in "$PLUGIN_ROOT/skills/plan/SKILL.md" "$PLUGIN_ROOT/commands/plan.md"; do
    hint=$(sed -n '/^argument-hint:/p' "$file")
    [[ "$hint" == *research* ]] || { fail "$file hides research in its hint"; return 1; }
    [[ "$hint" != *rnd* ]] || { fail "$file advertises rnd as a plan entry"; return 1; }
    [[ "$hint" != *evidence* ]] || { fail "$file advertises evidence as a plan entry"; return 1; }
  done
  hint=$(sed -n '/^argument-hint:/p' "$PLUGIN_ROOT/commands/document.md")
  [[ "$hint" == *research* && "$hint" == *evidence* ]] || { fail "document hides research or evidence in its hint"; return 1; }
  [[ "$hint" != *'|rnd|'* ]] || { fail "document advertises an unsupported rnd expert entry"; return 1; }
  if grep -Fq '/archcore:plan --track' "$PLUGIN_ROOT/agents/archcore-assistant.md"; then
    fail "assistant prescribes retired --track syntax"; return 1
  fi
}

@test "command hints match the supported arguments of their skills" {
  local skill command_hint skill_hint
  for skill in init plan document review; do
    command_hint=$(sed -n '/^argument-hint:/p' "$PLUGIN_ROOT/commands/$skill.md")
    skill_hint=$(sed -n '/^argument-hint:/p' "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    assert_equal "$command_hint" "$skill_hint" || return 1
  done
}

@test "a single-document assistant read still resolves the path with list_documents" {
  local exception
  exception=$(sed -n '/^\*\*Narrow exception\.\*\*/p' "$PLUGIN_ROOT/agents/archcore-assistant.md")
  [[ "$exception" == *'resolve the path with `list_documents`, then call `get_document`'* ]] \
    || { fail "the single-read exception bypasses path discovery"; return 1; }
}
