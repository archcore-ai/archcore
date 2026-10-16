#!/usr/bin/env python3
"""Matrix bench: what plan, document, and review create, ask, and keep for one situation.

Runs real model calls and costs money; nothing runs it automatically.

  matrix-bench.py run [--set cases|holdout] [--reps N] [--jobs N] [--only ID,ID] [--model M]
  matrix-bench.py report RUN_DIR...

Each case gives one headless model the current contracts of one command, the command line, and a
situation that stands in for grounding, and scores its one-line `matrix-verdict:` reply against the
expected fields. fixtures/matrix-bench.tsv holds the 59 own cases; fixtures/matrix-holdout.tsv holds
18 cases written by an author who did not write the contracts under test. Keep the two sets apart.
The 2026-10-16 contract fixes answered held-out cases, so that set is tuned; fixtures/matrix-holdout-fresh.tsv
(--set fresh) holds 18 cases a second author wrote from the RFC, the ADR, and the specs, without the contracts.
MATRIX_BENCH_MODEL (default claude-haiku-5-5), MATRIX_BENCH_REPS (default 3), and
MATRIX_BENCH_OUTPUT_DIR (default $TMPDIR/matrix-bench) set the run. MATRIX_BENCH_PLUGIN_ROOT points the
contracts at another plugin checkout, for a baseline on an earlier commit.
fixtures/matrix-experience.tsv holds 23 review cases for experience capture. Set
MATRIX_BENCH_NO_EXPERIENCE_FIELD=1 to drop the experience= field from the reply format and its
expectations: the field's definition names the capture criterion and hides contract changes.
A run exits 1 when a call fails, 2 when a call errors or no case is selected. The report prints
the mean input and output tokens and the mean latency per call.
"""
import argparse, collections, concurrent.futures as cf, glob, json, os, re, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.environ.get("MATRIX_BENCH_PLUGIN_ROOT") or os.path.abspath(os.path.join(HERE, "..", ".."))
SK = "plugins/archcore/skills"
FIXTURES = {"cases": "matrix-bench.tsv", "holdout": "matrix-holdout.tsv", "fresh": "matrix-holdout-fresh.tsv",
            "experience": "matrix-experience.tsv"}

# The contract files each command loads. Update this map when a command's contract files change;
# python-benches.bats checks that each list carries the two contracts every skill loads before a gate.
CONTRACTS = {
    "plan": ["plan/SKILL.md", "_shared/gate-contract.md", "_shared/delta-routing.md",
             "_shared/capability-granularity.md", "_shared/elicitation-contract.md", "_shared/tracks/sdd.md",
             "_shared/tracks/decision.md", "_shared/tracks/research.md", "_shared/tracks/requirements-cascade.md"],
    "document": ["document/SKILL.md", "_shared/gate-contract.md", "_shared/elicitation-contract.md",
                 "_shared/tracks/decision.md", "_shared/tracks/describe.md", "_shared/tracks/research.md"],
    "review": ["review/SKILL.md", "_shared/gate-contract.md", "_shared/durable-context-selection.md",
               "_shared/tracks/closeout.md", "_shared/tracks/experience.md", "_shared/elicitation-contract.md",
               "_shared/verdict-contract.md"],
}

NO_EXPERIENCE_FIELD = bool(os.environ.get("MATRIX_BENCH_NO_EXPERIENCE_FIELD"))
RETRY_DELAY = float(os.environ.get("MATRIX_BENCH_RETRY_DELAY") or 5)

OUTPUT_SPEC = """Output EXACTLY one line and nothing else:
matrix-verdict: entry=<resume|menu|ask|direct|health>; created=<types or none>; updated=<integer>; questions=<integer>; plan=<offer|retain|none>; reads=<integer>; anchors=<integer>; flagged=<integer>; experience=<captured|none>
entry: how the invocation starts. resume = continue an existing draft's track; menu = ask the user to pick among two or more concrete candidates found in grounding, including several resumable drafts; ask = ask an open question about the outcome, subject, or scope before routing; direct = route and work without a scope question; health = report project health.
created: the document types this invocation creates by its end, comma-separated in alphabetical order without spaces, a type repeated once per document, or none.
updated: how many existing documents it updates.
questions: how many questions it asks the user in total, confirmations included.
plan: offer when a matched plan reaches its removal confirmation, retain when a matched plan stays, none when no plan matched or the command is not review.
reads: how many documents and code files it opens in full before its first write or answer.
anchors: how many created documents name at least one code path or file kind they govern.
flagged: how many existing drafts the report flags as closable or stale.
experience: captured when the invocation creates or updates a cpat, task-type, rule, or guide that records a lesson transferable beyond this change; none otherwise."""

KEYS = ["entry", "created", "updated", "questions", "plan", "reads", "anchors", "flagged", "experience"]
EXPECTED_PAIR = re.compile(r"(?:entry|plan|experience)=[a-z]+|(?:created|has|not|any)=[a-z-]+(?:,[a-z-]+)*"
                           r"|(?:updated|questions|reads|anchors|flagged)=(?:>0|\d+(?:-\d+)?)")
if NO_EXPERIENCE_FIELD:
    OUTPUT_SPEC = re.sub(r"; experience=<captured\|none>|\nexperience: captured.*", "", OUTPUT_SPEC)
    KEYS.remove("experience")


def load_cases(path):
    cases = []
    for line in open(path, encoding="utf-8"):
        line = line.rstrip("\n")
        if not line.strip() or line.startswith("#"):
            continue
        f = line.split("\t")
        if len(f) != 6:
            sys.exit(f"bad row ({len(f)} fields): {line[:80]}")
        cid, cmd, args, crit, sit, exp = f
        bad = [p for p in exp.split(";") if not EXPECTED_PAIR.fullmatch(p)]
        if cmd not in CONTRACTS or bad:
            sys.exit(f"bad row {cid}: command {cmd!r}, malformed expectations {bad}")
        cases.append(dict(id=cid, cmd=cmd, args=args, criteria=crit.split(","), situation=sit, expected=exp))
    return cases


def contract_text(cmd):
    return "\n".join(f"--- CONTRACT: skills/{c} ---\n" + open(os.path.join(ROOT, SK, c), encoding="utf-8").read()
                     for c in CONTRACTS[cmd])


def prompt_for(case):
    args = "" if case["args"] == "(no arguments)" else " " + case["args"]
    return "\n".join([
        "You are the Archcore runtime executing one command under the contracts below. Apply them literally.",
        "You cannot call tools. The SITUATION states what grounding (the .archcore/ search, git state, and code reads) established; treat it as complete and true.",
        'Assume the user answers every question you ask with "you decide", and finish the whole invocation.',
        "",
        contract_text(case["cmd"]),
        "--- COMMAND ---",
        f"/archcore:{case['cmd']}{args}" + ("   (invoked with no arguments)" if not args else ""),
        "--- SITUATION ---",
        case["situation"],
        "--- OUTPUT ---",
        OUTPUT_SPEC,
    ])


def parse(out):
    """Return (fields, line count, missing keys); fields is None without a matrix-verdict line."""
    out = out.strip().strip("`").strip()
    lines = [l for l in out.splitlines() if l.strip()]
    line = next((l for l in lines if l.strip().strip("`").startswith("matrix-verdict:")), None)
    if not line:
        return None, len(lines), []
    got = {}
    for k in KEYS:
        m = re.search(rf"[:;]\s*{k}=([^;]+)", line)
        if m:
            got[k] = m.group(1).strip().strip("`")
    return got, len(lines), [k for k in KEYS if k not in got]


def in_range(want, got):
    try:
        g = int(got)
    except (TypeError, ValueError):
        return False
    if want == ">0":
        return g > 0
    if "-" in want:
        a, b = want.split("-", 1)
        return int(a) <= g <= int(b)
    return g == int(want)


def types_of(v):
    if not v or v == "none":
        return []
    return sorted(t.strip() for t in v.split(",") if t.strip())


def score(expected, got):
    """Return (pass, list of failure tags)."""
    if NO_EXPERIENCE_FIELD:
        expected = ";".join(p for p in expected.split(";") if not p.startswith("experience="))
    fails = []
    created = types_of(got.get("created"))
    for pair in expected.split(";"):
        k, want = pair.split("=", 1)
        if k == "created":
            if created != types_of(want):
                exp = types_of(want)
                extra = [t for t in created if created.count(t) > exp.count(t)]
                miss = [t for t in exp if exp.count(t) > created.count(t)]
                if extra: fails.append("excess:" + ",".join(sorted(set(extra))))
                if miss: fails.append("miss:" + ",".join(sorted(set(miss))))
        elif k == "has":
            miss = [t for t in want.split(",") if t not in created]
            if miss: fails.append("miss:" + ",".join(miss))
        elif k == "not":
            bad = [t for t in want.split(",") if t in created]
            if bad: fails.append("excess:" + ",".join(bad))
        elif k == "any":
            if not any(t in created for t in want.split(",")):
                fails.append("miss:" + "|".join(want.split(",")))
        elif k in ("updated", "questions", "reads", "anchors", "flagged"):
            if not in_range(want, got.get(k)):
                fails.append(f"{k}:{got.get(k)}!={want}")
        elif k in ("entry", "plan", "experience"):
            if got.get(k) != want:
                fails.append(f"{k}:{got.get(k)}!={want}")
    return (not fails), fails


def run_one(out_dir, model, case, rep):
    base = os.path.join(out_dir, f"{case['id']}.{rep}")
    if os.path.exists(base + ".rec.json"):
        return json.load(open(base + ".rec.json"))
    open(base + ".prompt", "w").write(prompt_for(case))
    cmd = ["claude", "-p", "--tools", "", "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}',
           "--setting-sources", "user", "--no-session-persistence", "--output-format", "json", "--model", model]
    rec = dict(id=case["id"], rep=rep, cmd=case["cmd"], criteria=case["criteria"], expected=case["expected"])
    j, p = None, None
    for attempt in range(3):
        with open(base + ".prompt") as fin:
            p = subprocess.run(cmd, stdin=fin, capture_output=True, text=True, cwd=out_dir, timeout=600)
        try:
            j = json.loads(p.stdout)
        except json.JSONDecodeError:
            j = None
        if j and not j.get("is_error") and isinstance(j.get("result"), str):
            break
        if attempt < 2:
            time.sleep(RETRY_DELAY * (attempt + 1))
    if not j or j.get("is_error") or not isinstance(j.get("result"), str):
        rec.update(status="error", stderr=p.stderr[-400:])
    else:
        usage = j.get("usage") or {}
        rec["cost_usd"] = j.get("total_cost_usd", 0)
        rec["input_tokens"] = sum(usage.get(k, 0) for k in
                                  ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"))
        rec["output_tokens"] = usage.get("output_tokens", 0)
        rec["duration_ms"] = j.get("duration_ms", 0)
        rec["reply"] = j["result"]
        got, nlines, missing = parse(j["result"])
        if got is None:
            rec.update(status="fail", fails=["format"], got={})
        else:
            ok, fails = score(case["expected"], got)
            if nlines != 1:
                ok, fails = False, fails + ["format:lines"]
            if missing:
                ok, fails = False, fails + ["format:missing:" + ",".join(missing)]
            rec.update(status="pass" if ok else "fail", fails=fails, got=got)
    json.dump(rec, open(base + ".rec.json", "w"), indent=1)
    return rec


def report(dirs):
    for d in dirs:
        recs = [json.load(open(f)) for f in glob.glob(os.path.join(d, "*.rec.json"))]
        by_case = collections.defaultdict(list)
        for r in recs:
            by_case[r["id"]].append(r["status"])
        print(f"{os.path.basename(d.rstrip('/'))}: {sum(r['status'] == 'pass' for r in recs)}/{len(recs)} calls pass, "
              f"{sum(1 for v in by_case.values() if v.count('pass') * 2 > len(v))}/{len(by_case)} cases by majority, "
              f"cost {sum(r.get('cost_usd', 0) for r in recs):.2f} USD")
        for cmd in ("plan", "document", "review"):
            cases = {k: v for k, v in by_case.items() if any(r["id"] == k and r["cmd"] == cmd for r in recs)}
            if cases:
                print(f"  {cmd}: {sum(1 for v in cases.values() if v.count('pass') * 2 > len(v))}/{len(cases)} by majority")
        unstable = sorted(k for k, v in by_case.items() if len(set(v)) > 1)
        print(f"  unstable: {len(unstable)} {' '.join(unstable)}")
        answered = [r for r in recs if r["status"] != "error"]
        if answered:
            n = len(answered)
            print(f"  per call: {sum(r.get('input_tokens', 0) for r in answered) / n:.0f} input tokens, "
                  f"{sum(r.get('output_tokens', 0) for r in answered) / n:.0f} output tokens, "
                  f"{sum(r.get('duration_ms', 0) for r in answered) / n / 1000:.1f} s")
        errors = sum(r["status"] == "error" for r in recs)
        if errors:
            print(f"  errors: {errors}")


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="action", required=True)
    run = sub.add_parser("run")
    run.add_argument("--set", choices=sorted(FIXTURES), default="cases")
    run.add_argument("--reps", type=int, default=int(os.environ.get("MATRIX_BENCH_REPS") or 3))
    run.add_argument("--jobs", type=int, default=8)
    run.add_argument("--only", default="")
    run.add_argument("--model", default=os.environ.get("MATRIX_BENCH_MODEL") or "claude-haiku-5-5")
    rep = sub.add_parser("report")
    rep.add_argument("dirs", nargs="+")
    a = ap.parse_args()
    if a.action == "report":
        return report(a.dirs)
    out = os.path.join(os.environ.get("MATRIX_BENCH_OUTPUT_DIR") or os.path.join(os.environ.get("TMPDIR", "/tmp"), "matrix-bench"),
                       f"{a.set}-{a.model}-{time.strftime('%m%d-%H%M%S')}")
    os.makedirs(out, exist_ok=True)
    cases = load_cases(os.path.join(HERE, "fixtures", FIXTURES[a.set]))
    if a.only:
        keep = set(a.only.split(","))
        cases = [c for c in cases if c["id"] in keep]
    if not cases or a.reps < 1:
        sys.stderr.write("no case selected; check the fixture file, --only, and --reps\n")
        return 2
    jobs = [(c, r) for c in cases for r in range(1, a.reps + 1)]
    print(f"# {len(cases)} cases x {a.reps} reps = {len(jobs)} calls on {a.model} -> {out}", flush=True)
    statuses = []
    with cf.ThreadPoolExecutor(a.jobs) as ex:
        for i, f in enumerate(cf.as_completed([ex.submit(run_one, out, a.model, c, r) for c, r in jobs]), 1):
            rec = f.result()
            statuses.append(rec["status"])
            print(f"{i}/{len(jobs)}\t{rec['id']}\t{rec['rep']}\t{rec['status']}\t{';'.join(rec.get('fails', []))}", flush=True)
    report([out])
    return 2 if "error" in statuses else 1 if "fail" in statuses else 0


if __name__ == "__main__":
    sys.exit(main())
