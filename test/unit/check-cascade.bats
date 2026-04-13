#!/usr/bin/env bats
# Tests for bin/check-cascade

setup() {
  load '../helpers/common'
  common_setup
  # Create a temp working directory with .archcore/
  WORK_DIR="$BATS_TEST_TMPDIR/workdir"
  mkdir -p "$WORK_DIR/.archcore"
}

# Helper: create sync-state.json with given relations
create_sync_state() {
  local relations="$1"
  cat > "$WORK_DIR/.archcore/.sync-state.json" <<EOF
{
  "version": 1,
  "files": {},
  "relations": [${relations}]
}
EOF
}

# Helper: run check-cascade in the work directory
run_cascade() {
  local fixture="$1"
  run sh -c "cd '$WORK_DIR' && cat '${FIXTURES}/stdin/${fixture}' | '${PLUGIN_ROOT}/bin/check-cascade'"
}

run_cascade_stdin() {
  local stdin_data="$1"
  run sh -c "cd '$WORK_DIR' && printf '%s' '${stdin_data}' | '${PLUGIN_ROOT}/bin/check-cascade'"
}

# --- No-op cases ---

@test "no doc_path exits silently" {
  create_sync_state ''
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{}}'
  assert_success
  assert_output ""
}

@test "no sync-state.json exits silently" {
  rm -f "$WORK_DIR/.archcore/.sync-state.json"
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

@test "empty relations array exits silently" {
  create_sync_state ''
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

# --- Cascade detection ---

@test "detects implements relation" {
  create_sync_state '{"source":"impl.plan.md","target":"my.adr.md","type":"implements"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output --partial "impl.plan.md"
  assert_output --partial "implements"
}

@test "detects depends_on relation" {
  create_sync_state '{"source":"child.spec.md","target":"my.prd.md","type":"depends_on"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.prd.md"}}'
  assert_success
  assert_output --partial "child.spec.md"
  assert_output --partial "depends_on"
}

@test "detects extends relation" {
  create_sync_state '{"source":"ext.rfc.md","target":"base.adr.md","type":"extends"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"base.adr.md"}}'
  assert_success
  assert_output --partial "ext.rfc.md"
  assert_output --partial "extends"
}

@test "ignores related relation type" {
  create_sync_state '{"source":"other.adr.md","target":"my.adr.md","type":"related"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

@test "ignores non-matching target" {
  create_sync_state '{"source":"impl.plan.md","target":"other.adr.md","type":"implements"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

@test "multiple affected docs all shown" {
  create_sync_state '{"source":"a.plan.md","target":"my.adr.md","type":"implements"},{"source":"b.spec.md","target":"my.adr.md","type":"depends_on"},{"source":"c.rfc.md","target":"my.adr.md","type":"extends"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output --partial "a.plan.md"
  assert_output --partial "b.spec.md"
  assert_output --partial "c.rfc.md"
}

@test "strips .archcore/ prefix from path" {
  create_sync_state '{"source":"impl.plan.md","target":"my.adr.md","type":"implements"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/my.adr.md"}}'
  assert_success
  assert_output --partial "impl.plan.md"
}

@test "output contains actualize suggestion" {
  create_sync_state '{"source":"impl.plan.md","target":"my.adr.md","type":"implements"}'
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output --partial "/archcore:actualize"
}

# --- Multi-host ---

@test "cursor: detects cascade from afterMCPExecution" {
  create_sync_state '{"source":"impl.plan.md","target":"auth/jwt-strategy.adr.md","type":"implements"}'
  run_cascade cursor/mcp-update.json
  assert_success
  assert_output --partial "impl.plan.md"
}

# --- Pretty-printed sync-state ---

@test "handles pretty-printed sync-state.json" {
  cat > "$WORK_DIR/.archcore/.sync-state.json" <<'EOF'
{
  "version": 1,
  "files": {},
  "relations": [
    {
      "source": "impl.plan.md",
      "target": "my.adr.md",
      "type": "implements"
    }
  ]
}
EOF
  run_cascade_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output --partial "impl.plan.md"
}
