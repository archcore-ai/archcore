#!/usr/bin/env bats
# Harness integrity for the Python benches uses fake model processes; live runs are separate targets.

setup() {
  load '../helpers/common'
  common_setup
  command -v python3 >/dev/null 2>&1 || skip "python3 not found"
  export PYTHONPATH="$REPO_ROOT/test/behavioral"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  export WRITER_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/out"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" > "$BENCH_ARGS"
cat >/dev/null
case " $* " in
  *" stream-json "*)
    printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__plugin_archcore_archcore__update_document","input":{"path":"p","edits":[{"old_string":"a","new_string":"b"}]}}]}}'
    printf '%s\n' '{"type":"result","result":"DONE","total_cost_usd":0.5,"usage":{"input_tokens":1,"cache_read_input_tokens":2,"cache_creation_input_tokens":3,"output_tokens":4}}' ;;
  *) printf '%s\n' '{"result":"```json\n[{\"label\":\"D1\",\"overall\":7}]\n```","num_turns":1}' ;;
esac
MOCK
  chmod +x "$MOCK_BIN/claude"
}

py() {
  run python3 -c "import importlib.util as u
def load(name):
    s = u.spec_from_file_location(name.replace('-', '_'), '$REPO_ROOT/test/behavioral/' + name + '.py')
    m = u.module_from_spec(s); s.loader.exec_module(m); return m
$1"
}

@test "count_findings adds the findings a capped report leaves out" {
  py "import benchlib as b
text = '[Archcore Precision] .archcore/x.adr.md (advisory):\n  - vague wording\n  - missing section: ## Context\n  - +3 more finding(s) not shown (report cap 5)\n\n[Archcore Cascade] x changed.\n  - y (implements this document)'
print(b.count_findings(text)[0], b.count_findings('[Archcore Cascade] only')[0])"
  assert_success
  assert_output '5 0'
}

@test "a Claude agent gets no built-in tools beyond the ones named, and its calls are parsed" {
  py "import benchlib as b, os
m = b.agent('claude', 'model', 'prompt', os.environ['BATS_TEST_TMPDIR'], os.environ['BATS_TEST_TMPDIR'] + '/a.jsonl', dict(os.environ), '')
print(m['calls'][0][1], m['calls'][0][3], m['out_tok'], m['in_tok'], m['cost'])"
  assert_success
  assert_output "update_document ['edits', 'path'] 4 6 0.5"
  run grep -A1 -Fx -- '--tools' "$BENCH_ARGS"
  assert_output "$(printf -- '--tools\n')"
  grep -Fxq -- 'mcp__plugin_archcore_archcore__create_document' "$BENCH_ARGS" || { fail "MCP tools are not pre-approved"; return 1; }
}

@test "the fan-out bench reads briefs with or without a json fence" {
  py "b = load('writer-fanout-bench')
fenced = 'Here:\n\`\`\`json\n[{\"filename\": \"x\", \"type\": \"adr\", \"brief\": \"B\"}]\n\`\`\`'
print(b.parse_briefs(fenced), b.parse_briefs('[{\"filename\": \"y\", \"type\": \"spec\", \"brief\": \"C\"}]'))"
  assert_success
  assert_output "{'x.adr': 'B'} {'y.spec': 'C'}"
}

@test "the update bench flags a changed kept section and a changed frontmatter" {
  py "b = load('edits-bench')
task = {'contains': ['new'], 'absent': ['old'], 'keep': ['## Keep']}
before = '---\ntitle: T\n---\n\n## Keep\nstay\n\n## Edit\nold\n'
good = before.replace('old', 'new')
print(b.ok(b.check(task, before, good)), b.ok(b.check(task, before, good.replace('stay', 'moved'))), b.ok(b.check(task, before, good.replace('title: T', 'title: U'))))"
  assert_success
  assert_output 'True False False'
}

@test "the judge runs Claude without tools or MCP servers and reports its tool use" {
  mkdir -p "$WRITER_BENCH_OUTPUT_DIR"
  py "j = load('writer-fanout-judge')
print(j.ask('opus', 'text', '$BATS_TEST_TMPDIR'))"
  assert_success
  assert_output "([{'label': 'D1', 'overall': 7}], 0)"
  grep -Fxq -- '--strict-mcp-config' "$BENCH_ARGS" || { fail "judge keeps MCP servers"; return 1; }
  run grep -A1 -Fx -- '--tools' "$BENCH_ARGS"
  assert_output "$(printf -- '--tools\n')"
}

@test "a run without a cost reports n/a, not zero" {
  py "import benchlib as b
print(b.cost_cell([{'cost': None}]), b.cost_cell([{'cost': 0.25}, {'cost': None}]))"
  assert_success
  assert_output 'n/a 0.25'
}
