#!/usr/bin/env python3
"""Review defect bench: which labeled doc/code defects does /archcore:review report?

Runs real model calls and costs money; nothing runs it automatically.

  review-defect-bench.py run --repo R --mode audit|branch --rep N [--plugin-dir D] [--cli BIN] [--label L]
  review-defect-bench.py report                      table over every recorded run (scores from review-defect-judge.py)

The labeled set is private and stays outside this repository: review-defects.json and the
review-seed-<repo>.patch files live under REVIEW_DEFECT_TRUTH (default
~/.local/share/archcore-review-bench). Fixtures are git clones pinned at the commits in
review-defects.json, kept under REVIEW_DEFECT_FIXTURES (default /tmp/review-bench/fixtures/<repo>).
Each clone has its default branch with an offline origin/<default> ref at that commit and the
seeded branch made from its patch (the run recreates the branch from the patch when it is missing).
A run copies the fixture (APFS clone when available), checks out the seeded branch in branch mode,
puts the given CLI first on PATH as `archcore`, and runs one headless Claude Code session with the
given plugin tree. Write tools are disallowed; the prompt also asks for a report only. A run stops
at 40 minutes or 60M input tokens and is recorded as a failure.
REVIEW_DEFECT_OUTPUT_DIR keeps runs (default /tmp/review-bench); runs land in <out>/runs/<id>/.
"""
import argparse, collections, glob, json, os, shutil, subprocess, sys, time

import benchlib as b

# The labeled set cites private repositories, so it lives outside this one.
TRUTH_DIR = os.environ.get("REVIEW_DEFECT_TRUTH") or os.path.expanduser("~/.local/share/archcore-review-bench")
TRUTH = json.load(open(os.path.join(TRUTH_DIR, "review-defects.json")))
OUT = b.out_dir("REVIEW_DEFECT_OUTPUT_DIR", "review-bench")
FIXTURES = os.environ.get("REVIEW_DEFECT_FIXTURES") or os.path.join(OUT, "fixtures")
TIMEOUT_S, MAX_INPUT = 40 * 60, 60_000_000
SUFFIX = "Report the findings only. Do not write, update, or remove any document or file, and do not ask questions."
PROMPTS = {"branch": "/archcore:review", "audit": "/archcore:review audit the knowledge base — are the docs out of date?"}
READ_MCP = [f"mcp__{s}__{t}" for s in ("archcore", "plugin_archcore_archcore")
            for t in ("list_documents", "get_document", "search_documents", "list_relations")]
WRITE_TOOLS = ["Write", "Edit", "MultiEdit", "NotebookEdit"] + [
    f"mcp__{s}__{t}" for s in ("archcore", "plugin_archcore_archcore")
    for t in ("create_document", "update_document", "remove_document", "add_relation", "remove_relation",
              "init_project", "install_host_config")]


def git(repo, *args, **kw):
    return subprocess.run(["git", "-C", repo, *args], check=True, capture_output=True, text=True, **kw).stdout.strip()


def prepare(repo_name, mode, dest):
    fx = TRUTH["fixtures"][repo_name]
    src = os.path.join(FIXTURES, repo_name)
    if subprocess.run(["cp", "-cR", src, dest], capture_output=True).returncode:  # clonefile: instant on APFS
        shutil.copytree(src, dest, symlinks=True)
    git(dest, "checkout", "-q", fx["default_branch"])
    if git(dest, "rev-parse", "HEAD") != fx["commit"]:
        sys.exit(f"{src}: {fx['default_branch']} is not at {fx['commit']}")
    if mode == "branch":
        if subprocess.run(["git", "-C", dest, "rev-parse", "--verify", "-q", fx["seed_branch"]], capture_output=True).returncode:
            git(dest, "checkout", "-q", "-b", fx["seed_branch"])
            git(dest, "-c", "user.name=bench", "-c", "user.email=bench@example.invalid", "am", "-q",
                "--committer-date-is-author-date", os.path.join(TRUTH_DIR, fx["seed_patch"]))
        else:
            git(dest, "checkout", "-q", fx["seed_branch"])
    return git(dest, "rev-parse", "HEAD")


def run(args):
    run_id = f"{args.label}-{args.repo}-{args.mode}-r{args.rep}"
    run_dir = os.path.join(OUT, "runs", run_id)
    if os.path.exists(run_dir):
        sys.exit(f"{run_dir} exists")
    os.makedirs(os.path.join(run_dir, "bin"))
    repo = os.path.join(run_dir, "repo")
    head = prepare(args.repo, args.mode, repo)
    os.symlink(os.path.abspath(args.cli), os.path.join(run_dir, "bin", "archcore"))
    env = {**os.environ, "PATH": os.path.join(run_dir, "bin") + os.pathsep + os.environ["PATH"],
           "ENABLE_CLAUDEAI_MCP_SERVERS": "false"}
    prompt = f"{PROMPTS[args.mode]}\n\n{SUFFIX}"
    cmd = ["claude", "-p", "--model", args.model, "--output-format", "stream-json", "--verbose",
           "--no-session-persistence", "--setting-sources", "project,local",
           "--plugin-dir", os.path.abspath(args.plugin_dir),
           "--allowedTools", "Bash", "Read", "Grep", "Glob", "Task", "Agent", "Skill", *READ_MCP,
           "--disallowedTools", *WRITE_TOOLS, *(["Agent", "Task"] if args.no_agents else [])]
    meta = dict(run=run_id, repo=args.repo, mode=args.mode, rep=args.rep, label=args.label, model=args.model, no_agents=args.no_agents,
                plugin_dir=os.path.abspath(args.plugin_dir), cli=os.path.realpath(args.cli), head=head, prompt=prompt,
                cli_version=subprocess.run(["archcore", "--version"], env=env, capture_output=True, text=True).stdout.strip())
    json.dump(meta, open(os.path.join(run_dir, "meta.json"), "w"), indent=2, ensure_ascii=False)
    log = os.path.join(run_dir, "stream.jsonl")
    t0 = time.time()
    p = subprocess.Popen(cmd, cwd=repo, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                         stderr=open(log + ".err", "w"), text=True)
    p.stdin.write(prompt)
    p.stdin.close()
    usage, failure = {}, None  # message id -> input tokens (stream-json repeats a message once per content block)
    with open(log, "w") as f:
        for line in p.stdout:
            t = time.time() - t0
            try:
                e = json.loads(line)
            except ValueError:
                continue
            f.write(json.dumps({"t": round(t, 2), "e": e}, ensure_ascii=False) + "\n")
            if e.get("type") == "assistant":
                m = e["message"]
                u = m.get("usage") or {}
                usage[m.get("id")] = u.get("input_tokens", 0) + u.get("cache_read_input_tokens", 0) + u.get("cache_creation_input_tokens", 0)
            if failure is None and (t > TIMEOUT_S or sum(usage.values()) > MAX_INPUT):
                failure = "timeout" if t > TIMEOUT_S else "input-token-cap"
                p.kill()
    p.wait()
    res = summarize(log)
    res.update(meta, wall=round(time.time() - t0, 1), exit=p.returncode, failure=failure or (None if res["text"] else "no-result"),
               dirty=git(repo, "status", "--porcelain"))
    json.dump(res, open(os.path.join(run_dir, "result.json"), "w"), indent=2, ensure_ascii=False)
    open(os.path.join(run_dir, "report.md"), "w").write(res["text"])
    print(json.dumps({k: res[k] for k in ("run", "wall", "failure", "api_turns", "cost")}))


def summarize(log):
    """Tool calls, subagents, turns, and tokens of one stream-json transcript; the main thread has parent None."""
    events = [json.loads(l)["e"] for l in open(log)]
    msgs, calls, sub_of = {}, [], {}
    for e in events:
        if e.get("type") != "assistant":
            continue
        m, parent = e["message"], e.get("parent_tool_use_id")
        msgs[m.get("id")] = (parent, m.get("usage") or {})
        for blk in m.get("content") or []:
            if blk.get("type") == "tool_use":
                calls.append((parent, blk["name"].split("__")[-1], blk.get("id"), blk.get("input") or {}))
    for parent, name, cid, inp in calls:
        if name in ("Task", "Agent"):
            sub_of[cid] = inp.get("subagent_type") or "general-purpose"
    main = collections.Counter(n for p, n, _, _ in calls if p is None)
    subs = {cid: dict(type=t, calls=collections.Counter(n for p, n, _, _ in calls if p == cid)) for cid, t in sub_of.items()}
    res = next((e for e in reversed(events) if e.get("type") == "result"), {})
    mu = res.get("modelUsage") or {}
    tok = lambda k: sum(v.get(k, 0) for v in mu.values())
    if mu:
        tokens = dict(input=tok("inputTokens"), cache_read=tok("cacheReadInputTokens"),
                      cache_write=tok("cacheCreationInputTokens"), output=tok("outputTokens"))
    else:  # killed run: no result event, sum the per-message usage instead
        us = [u for _, u in msgs.values()]
        tokens = dict(input=sum(u.get("input_tokens", 0) for u in us), cache_read=sum(u.get("cache_read_input_tokens", 0) for u in us),
                      cache_write=sum(u.get("cache_creation_input_tokens", 0) for u in us), output=sum(u.get("output_tokens", 0) for u in us))
    return dict(text=res.get("result") or "", cost=res.get("total_cost_usd"), num_turns=res.get("num_turns"),
                api_turns=sum(1 for p, _ in msgs.values() if p is None), api_turns_sub=sum(1 for p, _ in msgs.values() if p),
                tool_calls=dict(main), tool_calls_total=len(calls), subagents=list(subs.values()), tokens=tokens,
                model_usage=mu, subagent_stats=res.get("subagent_stats"))


def report():
    rows = [json.load(open(f)) for f in sorted(glob.glob(os.path.join(OUT, "runs", "*", "result.json")))]
    print(f"{'run':40}{'fail':>8}{'min':>6}{'turns':>7}{'calls':>7}{'subs':>5}{'in Mtok':>9}{'out ktok':>9}{'$':>8}{'recall':>9}{'prec':>9}")
    for r in rows:
        sc_f = os.path.join(OUT, "runs", r["run"], "score.json")
        sc = json.load(open(sc_f)) if os.path.exists(sc_f) else {}
        t = r["tokens"]
        rec = f"{sc['found']}/{sc['labeled']}" if sc else "-"
        prec = f"{sc['true']}/{sc['findings']}" if sc else "-"
        print(f"{r['run']:40}{str(r['failure'] or '-'):>8}{r['wall'] / 60:6.1f}{r['api_turns'] + r['api_turns_sub']:7}"
              f"{r['tool_calls_total']:7}{len(r['subagents']):5}{(t['input'] + t['cache_read'] + t['cache_write']) / 1e6:9.2f}"
              f"{t['output'] / 1e3:9.1f}{b.cost_cell([r]):>8}{rec:>9}{prec:>9}")


if __name__ == "__main__":
    if sys.argv[1:] == ["report"]:
        report()
        sys.exit()
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=["run"])
    ap.add_argument("--repo", required=True, choices=sorted(TRUTH["fixtures"]))
    ap.add_argument("--mode", required=True, choices=["audit", "branch"])
    ap.add_argument("--rep", type=int, default=1)
    ap.add_argument("--plugin-dir", default=os.path.join(b.REPO, "plugin", "plugins", "archcore"))
    ap.add_argument("--cli", default=shutil.which("archcore"))
    ap.add_argument("--model", default="claude-opus-5-5")
    ap.add_argument("--label", default="base")
    ap.add_argument("--no-agents", action="store_true", help="disallow subagents (Agent, Task) for the A/B")
    run(ap.parse_args())
