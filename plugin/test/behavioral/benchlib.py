"""Shared plumbing for the Python benches in this directory.

Runs headless Claude Code and Codex agents against a throwaway git worktree whose
own CLI build serves the Archcore MCP tools and hooks. Nothing here spends model
tokens by itself; the bench scripts decide what to run.
"""
import contextlib, json, os, re, subprocess, tempfile, time

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(os.path.dirname(HERE)))
MCP_PREFIX = "mcp__plugin_archcore_archcore__"
MCP_TOOLS = [MCP_PREFIX + t for t in ("list_documents", "get_document", "search_documents", "create_document",
                                      "update_document", "add_relation", "list_relations")]
# A dev build reports "dev", which the plugin's cli-gte gate reads as "no CLI" and then skips every hook.
BENCH_VERSION = "0.10.99-bench"


def out_dir(env_var, name):
    return os.environ.get(env_var) or os.path.join(tempfile.gettempdir(), name)


@contextlib.contextmanager
def worktree(path, ref, patch=None):
    subprocess.run(["git", "-C", REPO, "worktree", "add", "--detach", path, ref], check=True, capture_output=True)
    try:
        if patch:
            subprocess.run(["git", "-C", path, "apply", patch], check=True)
        yield path
    finally:
        subprocess.run(["git", "-C", REPO, "worktree", "remove", "--force", path], capture_output=True)


def build_cli(src_root, bin_dir):
    """Build the worktree's CLI into bin_dir and return an env that puts it first on PATH."""
    os.makedirs(bin_dir, exist_ok=True)
    subprocess.run(["go", "build", "-ldflags", f"-X main.version={BENCH_VERSION}", "-o", os.path.join(bin_dir, "archcore"), "."],
                   cwd=os.path.join(src_root, "cli"), check=True)
    return {**os.environ, "PATH": bin_dir + os.pathsep + os.environ["PATH"]}


def agent(host, model, prompt, cwd, log, env, builtins="Read,Grep,Glob"):
    """Run one headless agent; stamp every stdout event with its arrival time.

    builtins limits Claude's built-in tools ("" disables them); the Archcore MCP
    tools stay available. Codex has no per-tool switch, so it relies on the prompt
    and its workspace-write sandbox.
    """
    last = log + ".last"
    if host == "claude":
        cmd = ["claude", "-p", "--model", model, "--output-format", "stream-json", "--verbose",
               "--no-session-persistence", "--tools", builtins, "--allowedTools", *MCP_TOOLS]
        stdin = prompt
    else:
        cmd = ["codex", "exec", "-m", model, "-s", "workspace-write", "--json", "-o", last, prompt]
        stdin = None
    t0 = time.time()
    p = subprocess.Popen(cmd, cwd=cwd, env=env, stdin=subprocess.PIPE if stdin else subprocess.DEVNULL,
                         stdout=subprocess.PIPE, stderr=open(log + ".err", "w"), text=True)
    if stdin:
        p.stdin.write(stdin)
        p.stdin.close()
    m = dict(model=model, calls=[], in_tok=0, out_tok=0, cost=None, text="")
    with open(log, "w") as f:
        for line in p.stdout:
            t = time.time() - t0
            try:
                e = json.loads(line)
            except ValueError:
                continue
            f.write(json.dumps({"t": round(t, 2), "e": e}) + "\n")
            if host == "claude":
                if e.get("type") == "assistant":
                    for b in e["message"].get("content", []):
                        if b.get("type") == "tool_use":
                            m["calls"].append(call_row(t, b["name"], b.get("input") or {}))
                if e.get("type") == "result":
                    u = e.get("usage") or {}
                    m["in_tok"] = u.get("input_tokens", 0) + u.get("cache_read_input_tokens", 0) + u.get("cache_creation_input_tokens", 0)
                    m["out_tok"] = u.get("output_tokens", 0)
                    m["cost"] = e.get("total_cost_usd")
                    m["text"] = e.get("result") or ""
            else:
                it = e.get("item") or {}
                if e.get("type") == "item.completed" and it.get("type") in ("mcp_tool_call", "command_execution", "file_change"):
                    m["calls"].append(call_row(t, it.get("tool") or it.get("type"), it.get("arguments") or {}))
                if e.get("type") == "turn.completed":
                    u = e.get("usage") or {}
                    m["in_tok"] += u.get("input_tokens", 0)
                    m["out_tok"] += u.get("output_tokens", 0)
    p.wait()
    m["wall"] = round(time.time() - t0, 1)
    m["exit"] = p.returncode
    if host == "codex" and os.path.exists(last):
        m["text"] = open(last).read()
    return m


def call_row(t, name, args):
    """[seconds, tool, argument characters, argument keys] for one tool call."""
    args = args if isinstance(args, dict) else {}
    return [round(t, 1), name.split("__")[-1], len(json.dumps(args, ensure_ascii=False)), sorted(args)]


def count_findings(advisory):
    """Count the findings of the [Archcore Precision] section of a post-tool-use advisory."""
    if "[Archcore Precision]" not in advisory:
        return 0, ""
    section = advisory.split("[Archcore Precision]", 1)[1].split("\n\n", 1)[0]
    items = [l for l in section.splitlines() if l.startswith("  - ")]
    more = [int(x) for x in re.findall(r"^  - \+(\d+) more finding", section, re.M)]
    return len(items) - len(more) + sum(more), section.strip()


def precision_findings(env, wt, path):
    payload = json.dumps({"hook_event_name": "PostToolUse", "tool_name": MCP_PREFIX + "update_document",
                          "tool_input": {"path": path}, "cwd": wt})
    out = subprocess.run(["archcore", "hooks", "claude-code", "post-tool-use"], input=payload, cwd=wt, env=env,
                         capture_output=True, text=True).stdout
    return count_findings(out)


def cost_cell(agents):
    costs = [a["cost"] for a in agents if a.get("cost") is not None]
    return f"{sum(costs):.2f}" if costs else "n/a"
