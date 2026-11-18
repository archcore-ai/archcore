#!/usr/bin/env python3
"""Score review-defect-bench runs: recall over labeled defects, precision over every reported finding.

Runs real model calls and costs money; nothing runs it automatically.

  review-defect-judge.py score RUN_DIR... [--judge-model M] [--jobs N]   writes RUN_DIR/score.json
  review-defect-judge.py report [RUN_DIR...]                             tables over scored runs

Recall: a judge with no tools reads the run's final report and the labeled defects of that run's
repo and mode, and says per defect id whether the report states it and which side it gives.
Precision: a second judge call splits the report into distinct findings and maps each to a labeled
defect of the repo (any mode) when it is the same defect. A matched finding counts as true; every
other finding is checked blind by Codex (`codex exec`, read-only sandbox, cwd = the run's fixture copy).
"""
import argparse, collections, concurrent.futures as cf, glob, json, os, re, subprocess, sys

import benchlib as b

# The labeled set cites private repositories, so it lives outside this one.
TRUTH_DIR = os.environ.get("REVIEW_DEFECT_TRUTH") or os.path.expanduser("~/.local/share/archcore-review-bench")
TRUTH = json.load(open(os.path.join(TRUTH_DIR, "review-defects.json")))
OUT = b.out_dir("REVIEW_DEFECT_OUTPUT_DIR", "review-bench")
NO_TOOLS = ["--tools", "", "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}', "--setting-sources", "project",
            "--no-session-persistence", "--output-format", "json"]


def parse_json(text):
    m = re.search(r"```(?:json)?\s*(.*?)```", text, re.S)
    body = m.group(1) if m else text[min([i for i in (text.find("["), text.find("{")) if i >= 0] or [0]):]
    return json.JSONDecoder().raw_decode(body.strip())[0]


def ask_claude(model, prompt, cwd):
    """One tool-less call; returns (parsed JSON, USD)."""
    env = {**os.environ, "ENABLE_CLAUDEAI_MCP_SERVERS": "false"}
    for _ in range(2):
        p = subprocess.run(["claude", "-p", "--model", model, *NO_TOOLS], input=prompt, cwd=cwd, env=env,
                           capture_output=True, text=True)
        try:
            e = json.loads(p.stdout)
            return parse_json(e["result"]), e.get("total_cost_usd") or 0
        except (ValueError, KeyError, TypeError):
            continue
    raise RuntimeError(f"judge failed: {p.stderr[-500:]}")


def defect_lines(ds):
    return "\n".join(f"- id={d['id']} class={d['class']} doc={d['doc']} expected_side={d['expected_side']}\n  defect: {d['statement']}\n  evidence: {d['evidence']}" for d in ds)


def recall_prompt(report, ds):
    return f"""You grade a documentation review report against a list of known defects. Use only the report text; do not guess.
A defect is FOUND when the report states that specific problem (same document or code location and the same wrong fact), even in other words or another language. A vague remark ("some docs may be stale") or a different problem in the same file is NOT found.
For a found defect, give the side the report assigns: "spec-wrong" (the document is stale or wrong), "code-wrong" (the code violates a standing document), or "none" (no side stated, or the report only lists it as a question/ambiguity).

Reply with only a JSON array, one object per defect id in list order:
[{{"id": "...", "found": true|false, "side_reported": "spec-wrong"|"code-wrong"|"none", "quote": "<the report's words, at most 25 words, or empty>"}}]

KNOWN DEFECTS:
{defect_lines(ds)}

REPORT:
<<<
{report}
>>>"""


def extract_prompt(report, ds):
    return f"""Split a documentation review report into its distinct findings. A finding is one claim that something in the documents or the code is wrong, stale, missing, contradictory, unresolved, or should be removed. Skip "ok" verdicts, summaries, counts, next-step advice, and statements about what the review did or could not check. Merge repeats of the same claim into one finding. Write each finding as one self-contained sentence that names the document and the code path or identifier, so someone with only the repository can check it.
Map a finding to a known defect id only when it is the same problem as that defect; otherwise null.

Reply with only a JSON array:
[{{"n": 1, "finding": "...", "side": "spec-wrong"|"code-wrong"|"none", "defect": "<id or null>"}}]

KNOWN DEFECTS:
{defect_lines(ds)}

REPORT:
<<<
{report}
>>>"""


CODEX_PROMPT = """You verify one finding from a review of this repository's documentation (the documents live in .archcore/) against its code. Read the documents and code you need; do not modify anything.
Finding: {finding}
Is this finding true of the code and docs at the current checkout? Answer with only JSON: {{"true": true|false, "evidence": "<file:line and one sentence>"}}"""


def codex_verify(repo, finding, log):
    p = subprocess.run(["codex", "exec", "-s", "read-only", "--skip-git-repo-check", "--ephemeral", "--json", "-o", log + ".last",
                        "-C", repo, CODEX_PROMPT.format(finding=finding)], capture_output=True, text=True, stdin=subprocess.DEVNULL)
    open(log, "w").write(p.stdout)
    tok = sum((json.loads(l).get("usage") or {}).get("input_tokens", 0) for l in p.stdout.splitlines() if '"turn.completed"' in l)
    try:
        v = parse_json(open(log + ".last").read())
        return bool(v.get("true")), v.get("evidence", ""), tok
    except (OSError, ValueError, AttributeError):
        return None, "codex gave no verdict", tok


def claude_verify(repo, finding, log, model="claude-sonnet-5-5"):
    """Fallback blind check by a second Claude model with read-only tools, for a Codex outage.
    Project settings are not loaded, so the repository's own hooks do not fire."""
    env = {**os.environ, "ENABLE_CLAUDEAI_MCP_SERVERS": "false"}
    p = subprocess.run(["claude", "-p", "--model", model, "--tools", "Read,Grep,Glob", "--strict-mcp-config", "--mcp-config",
                        '{"mcpServers":{}}', "--setting-sources", "user", "--disable-slash-commands", "--no-session-persistence",
                        "--output-format", "json"], input=CODEX_PROMPT.format(finding=finding), cwd=repo, env=env,
                       capture_output=True, text=True)
    open(log, "w").write(p.stdout)
    try:
        e = json.loads(p.stdout)
        v = parse_json(e["result"])
        return bool(v.get("true")), "claude: " + v.get("evidence", ""), (e.get("usage") or {}).get("input_tokens", 0)
    except (ValueError, KeyError, TypeError, AttributeError):
        return None, "claude gave no verdict", 0


def verify(repo, finding, log):
    """Codex first; WHEN Codex gives no verdict (an outage or a usage limit), the Claude fallback."""
    if os.environ.get("REVIEW_VERIFIER") != "claude":
        got = codex_verify(repo, finding, log)
        if got[0] is not None:
            return got
    return claude_verify(repo, finding, log + ".claude")


def score(run_dir, model, jobs):
    r = json.load(open(os.path.join(run_dir, "result.json")))
    report = r["text"]
    labeled = [d for d in TRUTH["defects"] if d["repo"] == r["repo"] and d["mode"] == r["mode"]]
    repo_all = [d for d in TRUTH["defects"] if d["repo"] == r["repo"]]
    cwd = os.path.join(OUT, "judge-cwd")
    os.makedirs(cwd, exist_ok=True)
    usd = 0.0
    if report:
        rec, c1 = ask_claude(model, recall_prompt(report, labeled), cwd)
        found, c2 = ask_claude(model, extract_prompt(report, repo_all), cwd)
        usd += c1 + c2
    else:
        rec, found = [], []
    by_id = {x["id"]: x for x in rec}
    per = []
    for d in labeled:
        x = by_id.get(d["id"], {})
        f = bool(x.get("found"))
        per.append(dict(id=d["id"], **{"class": d["class"]}, found=f, side_reported=x.get("side_reported") if f else None,
                        side_match=(x.get("side_reported") == d["expected_side"]) if f else None, quote=x.get("quote", "")))
    known = {d["id"] for d in repo_all}
    vdir = os.path.join(run_dir, "verify")
    os.makedirs(vdir, exist_ok=True)
    todo = [f for f in found if f.get("defect") not in known]
    codex_tok = 0
    with cf.ThreadPoolExecutor(jobs) as pool:
        futs = {pool.submit(verify, os.path.join(run_dir, "repo"), f["finding"], os.path.join(vdir, f"f{f['n']}.jsonl")): f for f in todo}
        for fut in cf.as_completed(futs):
            f = futs[fut]
            f["true"], f["evidence"], t = fut.result()
            codex_tok += t
    for f in found:
        if f.get("defect") in known:
            f["true"], f["evidence"] = True, "matches labeled defect"
    res = dict(run=r["run"], repo=r["repo"], mode=r["mode"], labeled=len(labeled), found=sum(p["found"] for p in per),
               side_match=sum(bool(p["side_match"]) for p in per), defects=per, findings=len(found),
               true=sum(f.get("true") is True for f in found), false=sum(f.get("true") is False for f in found),
               unverified=sum(f.get("true") is None for f in found), items=found, judge_usd=round(usd, 4), codex_input_tokens=codex_tok)
    json.dump(res, open(os.path.join(run_dir, "score.json"), "w"), indent=2, ensure_ascii=False)
    print(f"{r['run']}: recall {res['found']}/{res['labeled']} side {res['side_match']}/{res['found']} "
          f"precision {res['true']}/{res['findings']} (unverified {res['unverified']}) judge ${usd:.2f}")


def report(dirs):
    dirs = dirs or sorted(os.path.dirname(f) for f in glob.glob(os.path.join(OUT, "runs", "*", "score.json")))
    rows = [(json.load(open(os.path.join(d, "result.json"))), json.load(open(os.path.join(d, "score.json")))) for d in dirs]
    print(f"{'run':36}{'min':>6}{'turns':>6}{'calls':>6}{'subs':>5}{'sub calls':>10}{'in Mtok':>8}{'out k':>7}{'$':>7}{'recall':>8}{'side':>6}{'prec':>8}")
    for r, s in rows:
        t = r["tokens"]
        sub_calls = sum(sum(x["calls"].values()) for x in r["subagents"])
        print(f"{r['run']:36}{r['wall'] / 60:6.1f}{r['api_turns'] + r['api_turns_sub']:6}{r['tool_calls_total']:6}{len(r['subagents']):5}"
              f"{sub_calls:10}{(t['input'] + t['cache_read'] + t['cache_write']) / 1e6:8.1f}{t['output'] / 1e3:7.0f}{b.cost_cell([r]):>7}"
              f"{s['found']:>4}/{s['labeled']:<3}{s['side_match']:>3}/{s['found']:<2}{s['true']:>4}/{s['findings']:<3}")
    cls = collections.defaultdict(lambda: [0, 0])
    for _, s in rows:
        for p in s["defects"]:
            cls[(s["mode"], p["class"])][0] += p["found"]
            cls[(s["mode"], p["class"])][1] += 1
    print("\nrecall by class (defect-runs found / defect-runs)")
    for (mode, c), (f, n) in sorted(cls.items()):
        print(f"  {mode:7}{c:26}{f}/{n}")
    print("\nmissed")
    for r, s in rows:
        for p in s["defects"]:
            if not p["found"]:
                print(f"  {r['run']}: {p['id']} ({p['class']})")
    print("\nfalse or unverified findings")
    for r, s in rows:
        for f in s["items"]:
            if f.get("true") is not True:
                print(f"  {r['run']} #{f['n']} [{f.get('true')}]: {f['finding']}\n      codex: {f.get('evidence')}")
    runs_usd = sum(r["cost"] or 0 for r, _ in rows)
    judge_usd = sum(s["judge_usd"] for _, s in rows)
    print(f"\ncost: runs ${runs_usd:.2f}, judge ${judge_usd:.2f}, codex input tokens {sum(s['codex_input_tokens'] for _, s in rows)}")


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=["score", "report"])
    ap.add_argument("dirs", nargs="*")
    ap.add_argument("--judge-model", default="claude-sonnet-5-5")
    ap.add_argument("--jobs", type=int, default=6)
    a = ap.parse_args()
    if a.cmd == "report":
        report(a.dirs)
    else:
        for d in a.dirs:
            score(d, a.judge_model, a.jobs)
