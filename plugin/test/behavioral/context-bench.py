#!/usr/bin/env python3
"""Context bench: does the agent read the rule that applies before its first write?

Runs real model calls and costs money; nothing runs it automatically.

  context-bench.py run --label L [--model M] [--reps N] [--plugin-dir D] [--cli-bin D]
  context-bench.py report                    table over every recorded run
  context-bench.py rescore RUN_DIR...         rescore runs from their transcripts

Each run copies fixtures/context-bench/ — a small TypeScript project whose .archcore/ holds
two pathless rules and one doc on a root config file, plus unrelated noise — and gives one
headless Claude Code agent one task. The instruction files carry only the managed block that
`archcore init` writes; no hand-written section points at .archcore/. The installed plugin
serves the MCP tools and hooks unless --plugin-dir names another one; --cli-bin puts a CLI
build first on PATH. Scores, from the transcript and the written files:

  saw      a tool result before the first Write/Edit holds the applicable document's marker
  mcp      an Archcore MCP call happened before the first Write/Edit
  comply   the written files follow the applicable document
  done     the task itself was carried out

CONTEXT_BENCH_OUTPUT_DIR keeps the runs (default: $TMPDIR/context-bench).
"""
import argparse, concurrent.futures as cf, glob, json, os, re, shutil, subprocess, sys, time

import benchlib as b

FIXTURE = os.path.join(b.HERE, "fixtures", "context-bench")
OUT = b.out_dir("CONTEXT_BENCH_OUTPUT_DIR", "context-bench")
WRITES = ("Write", "Edit", "MultiEdit", "NotebookEdit")


def read(proj, rel):
    p = os.path.join(proj, rel)
    return open(p).read() if os.path.exists(p) else ""


def check_tests(proj):
    good = os.path.join(proj, "src/utils/__checks__/formatPrice.check.ts")
    wrong = [p for p in glob.glob(os.path.join(proj, "src/**/*.*.ts"), recursive=True)
             if re.search(r"\.(test|spec)\.ts$", p)]
    files = ([good] if os.path.exists(good) else []) + wrong
    text = "".join(open(p).read() for p in files)
    titles = re.findall(r"\b(?:it|test)\(\s*['\"`]([^'\"`]*)", text)
    explicit = re.search(r"import\s*\{[^}]*\bdescribe\b[^}]*\}\s*from\s*['\"]vitest['\"]", text)
    comply = bool(os.path.exists(good) and not wrong and titles
                  and all(t.startswith("проверяет") for t in titles) and explicit)
    return dict(done=bool(files), comply=comply)


def check_guard(proj):
    src = read(proj, "src/billing/invoice.ts")
    m = re.search(r"export function applyDiscount\b[^{]*\{\s*(.*)", src, re.S)
    comply = bool(m and m.group(1).startswith("assertArgs(") and re.search(r"import\s*\{[^}]*assertArgs", src))
    return dict(done=bool(m), comply=comply)


def check_config(proj):
    src = read(proj, "vitest.config.ts")
    done = bool(re.search(r"testTimeout:\s*10[_]?000\b", src))
    return dict(done=done, comply=done and "// reviewed-by: platform" in src)


TASKS = {
    "tests": dict(prompt="Write unit tests for src/utils/formatPrice.ts. Do not run them.",
                  marker="CTXB-TEST-LAYOUT", check=check_tests),
    "guard": dict(prompt="Add an exported function applyDiscount(total: number, percent: number): number to "
                         "src/billing/invoice.ts. It returns the total reduced by the given percent.",
                  marker="CTXB-GUARDS", check=check_guard),
    "config": dict(prompt="Raise the test timeout in vitest.config.ts to 10 seconds.",
                   marker="CTXB-VITEST-CONFIG", check=check_config),
}


def before_first_write(log, marker):
    """(saw, mcp) over the events that precede the first write tool call."""
    saw = mcp = False
    for line in open(log):
        msg = json.loads(line)["e"].get("message")
        for blk in (msg.get("content") or []) if isinstance(msg, dict) else []:
            if not isinstance(blk, dict):
                continue
            if blk.get("type") == "tool_use":
                if blk["name"] in WRITES:
                    return saw, mcp
                mcp = mcp or "archcore" in blk["name"]
            if blk.get("type") == "tool_result" and marker in json.dumps(blk.get("content"), ensure_ascii=False):
                saw = True
    return saw, mcp


def one(task, rep, run_dir, model, env, extra):
    proj = os.path.join(run_dir, f"{task}-r{rep}")
    shutil.copytree(FIXTURE, proj)
    # The rules are old news: push accepted documents out of the recap's 30-day window, as in a real
    # corpus where the session-start recap lists recent drafts and not the rules that bind the edit.
    old = time.time() - 90 * 86400
    for doc in glob.glob(os.path.join(proj, ".archcore", "**", "*.md"), recursive=True):
        if "status: accepted" in open(doc).read():
            os.utime(doc, (old, old))
    subprocess.run(["git", "init", "-q"], cwd=proj, check=True)
    subprocess.run(["git", "add", "-A"], cwd=proj, check=True)
    subprocess.run(["git", "-c", "user.name=bench", "-c", "user.email=bench@example.invalid", "commit", "-qm", "fixture"],
                   cwd=proj, check=True)
    log = proj + ".jsonl"
    b.agent("claude", model, TASKS[task]["prompt"], proj, log, env, "Read,Grep,Glob,Write,Edit,Bash", extra)
    return score(task, rep, proj)


def score(task, rep, proj):
    """Score one finished run from its transcript and its project copy."""
    log, t = proj + ".jsonl", TASKS[task]
    saw, mcp = before_first_write(log, t["marker"])
    events = [json.loads(l) for l in open(log)]
    res = next((x["e"] for x in events if x["e"].get("type") == "result"), {})
    calls = [blk["name"].split("__")[-1] for x in events if isinstance(x["e"].get("message"), dict)
             for blk in x["e"]["message"].get("content") or [] if isinstance(blk, dict) and blk.get("type") == "tool_use"]
    return dict(task=task, rep=rep, saw=saw, mcp=mcp, **t["check"](proj), cost=res.get("total_cost_usd"),
                wall=events[-1]["t"] if events else 0, calls=calls)


def run(label, model, reps, plugin_dir, cli_bin):
    env = dict(os.environ)
    if cli_bin:
        env["PATH"] = os.path.abspath(cli_bin) + os.pathsep + env["PATH"]
    extra = ["--permission-mode", "acceptEdits"] + (["--plugin-dir", os.path.abspath(plugin_dir)] if plugin_dir else [])
    version = subprocess.run(["archcore", "--version"], env=env, capture_output=True, text=True).stdout.strip()
    run_dir = os.path.join(OUT, "runs", f"{label}-{time.strftime('%m%d-%H%M%S')}")
    os.makedirs(run_dir)
    jobs = [(task, rep) for rep in range(reps) for task in TASKS]
    meta = dict(label=label, model=model, cli=version, plugin_dir=plugin_dir or "installed", reps=reps)
    json.dump(meta, open(os.path.join(run_dir, "meta.json"), "w"), indent=2)
    with cf.ThreadPoolExecutor(len(TASKS)) as pool:
        list(pool.map(lambda j: one(*j, run_dir, model, env, extra), jobs))
    table([rescore(run_dir)])


def rescore(run_dir):
    """Rebuild result.json of a run directory from its transcripts and project copies."""
    meta = json.load(open(os.path.join(run_dir, "meta.json")))
    rows = []
    for log in sorted(glob.glob(os.path.join(run_dir, "*-r*.jsonl"))):
        task, rep = os.path.basename(log)[:-6].rsplit("-r", 1)
        rows.append(score(task, int(rep), log[:-6]))
    res = dict(meta=meta, rows=rows)
    json.dump(res, open(os.path.join(run_dir, "result.json"), "w"), indent=2, ensure_ascii=False)
    return res


def table(results):
    print(f"{'label':22}{'task':8}{'n':>3}{'saw':>7}{'mcp':>7}{'comply':>8}{'done':>7}{'cost $':>8}")
    for res in results:
        rows = res["rows"]
        for task in list(TASKS) + ["ALL"]:
            rs = [r for r in rows if task in ("ALL", r["task"])]
            n = len(rs)
            cost = sum(r["cost"] or 0 for r in rs)
            frac = lambda k: f"{sum(r[k] for r in rs)}/{n}"
            print(f"{res['meta']['label']:22}{task:8}{n:3}{frac('saw'):>7}{frac('mcp'):>7}{frac('comply'):>8}"
                  f"{frac('done'):>7}{cost:8.2f}")


def report():
    runs = sorted(glob.glob(os.path.join(OUT, "runs", "*", "result.json")))
    table([json.load(open(f)) for f in runs])


if __name__ == "__main__":
    if sys.argv[1:] == ["report"]:
        report()
        sys.exit()
    if sys.argv[1:2] == ["rescore"]:
        table([rescore(d) for d in sys.argv[2:]])
        sys.exit()
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=["run"])
    ap.add_argument("--label", required=True)
    ap.add_argument("--model", default="claude-sonnet-5-5")
    ap.add_argument("--reps", type=int, default=3)
    ap.add_argument("--plugin-dir")
    ap.add_argument("--cli-bin")
    a = ap.parse_args()
    run(a.label, a.model, a.reps, a.plugin_dir, a.cli_bin)
