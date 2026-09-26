#!/usr/bin/env python3
"""Update bench: what does update_document's `edits` save on one review round?

Runs real model calls and costs money; nothing runs it automatically (make test-edits-bench).

  edits-bench.py HOST [--model M] [--reps N]   N rounds of both variants on every task (default 3)
  edits-bench.py report                        table over all rounds

HOST     claude | codex
VARIANT  before  the CLI at the fixture ref: update_document takes only a full body
         after   the same ref plus EDITS_BENCH_PATCH: update_document also takes `edits`

Each task gives one agent a document and a reviewer's comments. Both variants get the same
prompt, which does not mention `edits`: the bench measures whether an agent finds it and what it
saves. A worktree of the ref holds the corpus, and each variant builds its own CLI, which serves
the MCP tools and the hooks from PATH. Checks are deterministic: the requested text is present,
the replaced text is absent, and the frontmatter and the untouched sections are byte-identical.
EDITS_BENCH_OUTPUT_DIR keeps the rounds (default: $TMPDIR/edits-bench).
"""
import argparse, concurrent.futures as cf, difflib, json, os, subprocess, sys, time

import benchlib as b

FIX = json.load(open(os.path.join(b.HERE, "fixtures", "edits-bench.json")))
OUT = b.out_dir("EDITS_BENCH_OUTPUT_DIR", "edits-bench")
REF = os.environ.get("EDITS_BENCH_REF") or FIX["ref"]
PATCH = os.environ.get("EDITS_BENCH_PATCH")
DEFAULTS = {"claude": "claude-opus-5-5", "codex": "gpt-6-astra"}


def prompt(task):
    comments = "\n".join(f"{i + 1}. {c}" for i, c in enumerate(task["comments"]))
    return f"""You are running a non-interactive benchmark. Do not ask questions; nobody will answer.
Change .archcore/ only through the Archcore MCP tools; never write .archcore/ files directly.
A reviewer left these comments on {task['path']}. Read the document with get_document, apply every comment, and change nothing else. Keep the title, status, and tags.

{comments}

When done, reply DONE."""


def section(text, heading):
    start = text.find("\n" + heading + "\n")
    if start < 0:
        return None
    end = text.find("\n## ", start + len(heading) + 2)
    return text[start:end if end >= 0 else len(text)]


def frontmatter(text):
    return text.split("\n---\n", 1)[0] if text.startswith("---\n") else ""


def check(task, before, after):
    return dict(
        contains=sum(s in after for s in task["contains"]), contains_of=len(task["contains"]),
        absent=sum(s not in after for s in task["absent"]), absent_of=len(task["absent"]),
        keep=all(section(before, h) is not None and section(before, h) == section(after, h) for h in task["keep"]),
        frontmatter=frontmatter(before) == frontmatter(after),
        similarity=round(difflib.SequenceMatcher(None, before, after, autojunk=False).ratio(), 3),
    )


def has_edits(wt):
    return "func applyEdits" in open(os.path.join(wt, "cli/internal/mcp/tools/update_document.go")).read()


def round_(host, model, variant, round_dir):
    """One variant on every task in parallel, in its own worktree and CLI build."""
    os.makedirs(round_dir)
    patch = PATCH if variant == "after" else None
    with b.worktree(os.path.join(OUT, "wt", os.path.basename(round_dir)), REF, patch) as wt:
        if has_edits(wt) != (variant == "after"):
            sys.exit(f"{REF}: the '{variant}' tree must {'have' if variant == 'after' else 'lack'} edits; "
                     f"use a ref without edits and set EDITS_BENCH_PATCH")
        env = b.build_cli(wt, os.path.join(round_dir, "bin"))
        originals = {t["id"]: open(os.path.join(wt, t["path"])).read() for t in FIX["tasks"]}
        with cf.ThreadPoolExecutor(len(FIX["tasks"])) as pool:
            futs = {pool.submit(b.agent, host, model, prompt(t), wt, os.path.join(round_dir, t["id"] + ".jsonl"), env, ""): t
                    for t in FIX["tasks"]}
            rows = []
            for fut in cf.as_completed(futs):
                t, a = futs[fut], fut.result()
                after = open(os.path.join(wt, t["path"])).read()
                updates = [c for c in a["calls"] if c[1] == "update_document"]
                rows.append(dict(task=t["id"], variant=variant, host=host, model=model, wall=a["wall"], out_tok=a["out_tok"],
                                 in_tok=a["in_tok"], cost=a["cost"], updates=len(updates),
                                 update_chars=sum(c[2] for c in updates), used_edits=any("edits" in c[3] for c in updates),
                                 calls=[c[:3] for c in a["calls"]], **check(t, originals[t["id"]], after)))
                open(os.path.join(round_dir, t["id"] + ".after.md"), "w").write(after)
    json.dump(rows, open(os.path.join(round_dir, "result.json"), "w"), indent=2)
    return rows


def run(host, model, reps):
    stamp = time.strftime("%m%d-%H%M%S")
    for rep in range(reps):
        for variant in (("before", "after") if rep % 2 == 0 else ("after", "before")):
            rows = round_(host, model, variant, os.path.join(OUT, "runs", f"{host}-{stamp}-r{rep}-{variant}"))
            print(json.dumps({"host": host, "rep": rep, "variant": variant,
                              "wall": max(r["wall"] for r in rows), "ok": sum(ok(r) for r in rows), "of": len(rows)}))


def ok(r):
    return r["contains"] == r["contains_of"] and r["absent"] == r["absent_of"] and r["keep"] and r["frontmatter"]


def report():
    rows = []
    for d in sorted(os.listdir(os.path.join(OUT, "runs"))):
        f = os.path.join(OUT, "runs", d, "result.json")
        if os.path.exists(f):
            rows += json.load(open(f))
    med = lambda xs: sorted(xs)[len(xs) // 2] if xs else 0
    print(f"{'host':7}{'task':13}{'variant':8}{'n':>3}{'wall s':>8}{'out tok':>9}{'upd chars':>11}{'edits used':>12}{'correct':>9}{'cost $':>8}")
    groups = {}
    for r in rows:
        groups.setdefault((r["host"], r["task"], r["variant"]), []).append(r)
        groups.setdefault((r["host"], "ALL", r["variant"]), []).append(r)
    for (h, t, v), rs in sorted(groups.items(), key=lambda kv: (kv[0][0], kv[0][1] == "ALL", kv[0][1], kv[0][2] != "before")):
        costs = [r["cost"] for r in rs if r["cost"] is not None]
        cost = f"{med(costs):.2f}" if costs else "n/a"
        print(f"{h:7}{t:13}{v:8}{len(rs):3}{med([r['wall'] for r in rs]):8.0f}{med([r['out_tok'] for r in rs]):9}"
              f"{med([r['update_chars'] for r in rs]):11}{sum(r['used_edits'] for r in rs):>7}/{len(rs):<4}"
              f"{sum(ok(r) for r in rs):>5}/{len(rs):<3}{cost:>8}")

if __name__ == "__main__":
    if sys.argv[1:] == ["report"]:
        report()
        sys.exit()
    ap = argparse.ArgumentParser()
    ap.add_argument("host", choices=["claude", "codex"])
    ap.add_argument("--model")
    ap.add_argument("--reps", type=int, default=3)
    a = ap.parse_args()
    run(a.host, a.model or DEFAULTS[a.host], a.reps)
