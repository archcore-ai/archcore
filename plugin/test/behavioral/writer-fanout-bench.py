#!/usr/bin/env python3
"""Writer fan-out bench: does splitting document writing across agents or models save time?

Runs real model calls and costs money; nothing runs it automatically (make test-writer-fanout-bench).

  writer-fanout-bench.py HOST VARIANT [--strong M] [--fast M]   one run in a fresh git worktree
  writer-fanout-bench.py report                                 table over all runs

HOST     claude | codex
VARIANT  A  strong model writes the whole package sequentially in one session
         B  strong model writes briefs -> fast writers in parallel -> fast model adds relations
         C  same as B, but the writers and the relation pass use the strong model
         D  same as C, but each writer gets its contract and precision-rules inline and reads nothing

The fixture's facts hold at its "ref" commit, so a run checks out that ref (WRITER_BENCH_REF overrides
it). The package documents update_document's `edits`; a ref without that code needs WRITER_BENCH_PATCH,
a diff applied to every worktree. Each run builds the worktree's CLI and puts it first on PATH, so the
agents' MCP server and hooks come from the tree under test. WRITER_BENCH_OUTPUT_DIR keeps runs and
worktrees (default: $TMPDIR/writer-fanout-bench).
"""
import argparse, concurrent.futures as cf, json, os, re, shutil, sys, time

import benchlib as b

PKG = json.load(open(os.path.join(b.HERE, "fixtures", "writer-fanout-package.json")))
OUT = b.out_dir("WRITER_BENCH_OUTPUT_DIR", "writer-fanout-bench")
REF = os.environ.get("WRITER_BENCH_REF") or PKG["ref"]
PATCH = os.environ.get("WRITER_BENCH_PATCH")
DEFAULTS = {"claude": ("claude-opus-5-5", "claude-sonnet-5"), "codex": ("gpt-6-astra", "gpt-5.6-luna")}
TRUTH = ("cli/internal/mcp/tools/update_document.go", "cli/internal/mcp/tools/update_document_test.go")

RULES = """You are running a non-interactive benchmark. Do not ask questions; nobody will answer.
Change .archcore/ only through the Archcore MCP tools; never write .archcore/ files directly.
Before you write a document of a given type, read plugin/plugins/archcore/skills/_shared/<type>-contract.md when that file exists, and read plugin/plugins/archcore/skills/_shared/precision-rules.md. Follow the writing policy in AGENTS.md.
Create every document with status "draft".
If a tool result or hook message reports [Archcore Precision] or [Archcore Validation] findings for a document you wrote, fix them with update_document; do at most 2 fix rounds per document.
Do not create, update, or relate any document outside the package."""


def doc_path(d):
    return f".archcore/{d['directory']}/{d['filename']}.{d['type']}.md"


def doc_line(d):
    return f"- type={d['type']} directory={d['directory']} filename={d['filename']} title=\"{d['title']}\" — {d['intent']}"


def rel_lines():
    return "\n".join(f"- {r['source']} --{r['type']}--> {r['target']}" for r in PKG["relations"])


def package_block():
    ctx = "\n".join("- " + c for c in PKG["context"])
    docs = "\n".join(doc_line(d) for d in PKG["docs"])
    return f"TASK: {PKG['task']}\n\nCONTEXT FROM THE CONVERSATION (verified facts you may use):\n{ctx}\n\nTHE USER CONFIRMED THIS PACKAGE:\n{docs}"


def prompt_all():
    return f"{RULES}\n\n{package_block()}\n\nCreate every document of the package, one after another. Then add these relations with add_relation:\n{rel_lines()}\nWhen done, reply DONE and the list of paths."


def prompt_briefs():
    return f"""You are running a non-interactive benchmark. Do not ask questions; nobody will answer.

{package_block()}

Your job: prepare one writing brief per document for separate writer agents. The writers will NOT see this conversation, and they re-read code only to confirm a name. Read the code you need now and verify every fact you put in a brief. Do NOT call create_document, update_document, or add_relation.
Reply with only a JSON array in a ```json fence, one object per document in package order:
{{"filename": "...", "type": "...", "brief": "<self-contained brief: purpose, verified facts with @paths and identifiers, key points per section, claims the document must not make>"}}
Keep each brief at most 300 words."""


def prompt_writer(d, brief):
    return f"""{RULES}

Write exactly one document:
{doc_line(d)}

BRIEF FROM THE LEAD AGENT (your source of facts):
{brief}

Read code only to confirm an identifier or path the brief names. Do not add relations; the lead agent adds them. When done, reply DONE and the path."""


SHARED = os.path.join(b.REPO, "plugin/plugins/archcore/skills/_shared")


def prompt_writer_inline(d, brief):
    contract = os.path.join(SHARED, f"{d['type']}-contract.md")
    parts = [open(contract).read()] if os.path.exists(contract) else []
    parts.append(open(os.path.join(SHARED, "precision-rules.md")).read())
    refs = "\n\n".join(f"<reference>\n{p}\n</reference>" for p in parts)
    return f"""You are running a non-interactive benchmark. Do not ask questions; nobody will answer.
Change .archcore/ only through the Archcore MCP tools. Create the document with status "draft".
Everything you need is in this prompt: do not read files, run commands, or search documents.
Call create_document once. If the result or a hook message reports [Archcore Precision] or [Archcore Validation] findings, fix them with update_document; do at most 2 fix rounds.

Write exactly one document:
{doc_line(d)}

BRIEF FROM THE LEAD AGENT (your source of facts):
{brief}

RULES FOR THIS DOCUMENT TYPE (the content contract, then the precision rules):
{refs}

Do not add relations; the lead agent adds them. When done, reply DONE and the path."""


def prompt_relations():
    return f"You are running a non-interactive benchmark. Do not ask questions.\nAdd these relations with the Archcore add_relation tool and do nothing else:\n{rel_lines()}\nReply DONE."


def parse_briefs(text):
    block = re.search(r"```json\s*(.*?)```", text, re.S)
    briefs = json.loads(block.group(1) if block else text)
    return {x["filename"] + "." + x["type"]: x["brief"] for x in briefs}


def preflight(wt):
    """Refuse a tree the fixture does not describe, before any model call is paid for."""
    present = [doc_path(d) for d in PKG["docs"] if os.path.exists(os.path.join(wt, doc_path(d)))]
    if present:
        sys.exit(f"{REF}: package documents already exist ({', '.join(present)}); pick a ref before they were recorded")
    if "func applyEdits" not in open(os.path.join(wt, TRUTH[0])).read():
        sys.exit(f"{REF}: update_document has no edits; set WRITER_BENCH_PATCH to the change the package documents")


def evaluate(wt, run_dir, env):
    os.makedirs(os.path.join(run_dir, "docs"), exist_ok=True)
    docs = []
    for d in PKG["docs"]:
        p = doc_path(d)
        full = os.path.join(wt, p)
        row = dict(path=p, exists=os.path.exists(full), chars=0, findings=None)
        if row["exists"]:
            row["chars"] = len(open(full).read())
            row["findings"], row["findings_text"] = b.precision_findings(env, wt, p)
            shutil.copy(full, os.path.join(run_dir, "docs", os.path.basename(p)))
        docs.append(row)
    manifest = json.load(open(os.path.join(wt, ".archcore", ".sync-state.json")))
    have = {(r["source"], r["target"], r["type"]) for r in manifest.get("relations", [])}
    strip = lambda s: s.removeprefix(".archcore/")
    rels = sum((strip(r["source"]), strip(r["target"]), r["type"]) in have for r in PKG["relations"])
    return docs, rels


def run(host, variant, strong, fast):
    run_id = f"{host}-{variant}-{time.strftime('%m%d-%H%M%S')}"
    run_dir = os.path.join(OUT, "runs", run_id)
    os.makedirs(os.path.join(OUT, "wt"), exist_ok=True)
    os.makedirs(os.path.join(run_dir, "truth"))
    log = lambda name: os.path.join(run_dir, name + ".jsonl")
    phases, agents = {}, []
    with b.worktree(os.path.join(OUT, "wt", run_id), REF, PATCH) as wt:
        preflight(wt)
        for f in TRUTH:
            shutil.copy(os.path.join(wt, f), os.path.join(run_dir, "truth", os.path.basename(f)))
        env = b.build_cli(wt, os.path.join(run_dir, "bin"))
        t0 = time.time()
        if variant == "A":
            agents.append(("all", b.agent(host, strong, prompt_all(), wt, log("all"), env)))
            phases["all"] = agents[-1][1]["wall"]
        else:
            writer = fast if variant == "B" else strong
            write_prompt = prompt_writer_inline if variant == "D" else prompt_writer
            write_tools = "" if variant == "D" else "Read,Grep,Glob"
            lead = b.agent(host, strong, prompt_briefs(), wt, log("briefs"), env)
            agents.append(("briefs", lead))
            phases["briefs"] = lead["wall"]
            briefs = parse_briefs(lead["text"])
            json.dump(briefs, open(os.path.join(run_dir, "briefs.json"), "w"), indent=2, ensure_ascii=False)
            t1 = time.time()
            with cf.ThreadPoolExecutor(len(PKG["docs"])) as pool:
                futs = {pool.submit(b.agent, host, writer, write_prompt(d, briefs.get(f"{d['filename']}.{d['type']}", "")),
                                    wt, log("write-" + d["filename"] + "." + d["type"]), env, write_tools): d for d in PKG["docs"]}
                for fut in cf.as_completed(futs):
                    agents.append(("write:" + futs[fut]["type"], fut.result()))
            phases["writers"] = round(time.time() - t1, 1)
            agents.append(("relations", b.agent(host, writer, prompt_relations(), wt, log("relations"), env, "")))
            phases["relations"] = agents[-1][1]["wall"]
        total = round(time.time() - t0, 1)
        docs, rels = evaluate(wt, run_dir, env)
    result = dict(run=run_id, host=host, variant=variant, strong=strong, fast=fast if variant == "B" else None,
                  ref=REF, patch=PATCH, total=total, phases=phases, docs=docs, relations=rels,
                  agents=[dict(role=r, **{k: v for k, v in a.items() if k != "text"}) for r, a in agents])
    json.dump(result, open(os.path.join(run_dir, "result.json"), "w"), indent=2, ensure_ascii=False)
    print(json.dumps({k: result[k] for k in ("run", "total", "phases", "relations")}))


def load_runs():
    rows = []
    for d in sorted(os.listdir(os.path.join(OUT, "runs"))):
        f = os.path.join(OUT, "runs", d, "result.json")
        if os.path.exists(f):
            rows.append(json.load(open(f)))
    return rows


def report(rows):
    print(f"{'run':24}{'total s':>8}  {'phases':34}{'docs':>5}{'chars':>7}{'find':>5}{'rels':>5}{'upd':>4}{'out tok':>8}{'in tok':>9}{'$':>7}")
    for r in rows:
        ag = r["agents"]
        upd = sum(1 for a in ag for c in a["calls"] if c[1] == "update_document")
        ph = " ".join(f"{k}={v:.0f}" for k, v in r["phases"].items())
        docs = [d for d in r["docs"] if d["exists"]]
        print(f"{r['run']:24}{r['total']:8.0f}  {ph:34}{len(docs):5}{sum(d['chars'] for d in docs):7}"
              f"{sum(d['findings'] or 0 for d in docs):5}{r['relations']:5}{upd:4}"
              f"{sum(a['out_tok'] for a in ag):8}{sum(a['in_tok'] for a in ag):9}{b.cost_cell(ag):>7}")

    med = lambda xs: sorted(xs)[len(xs) // 2] if xs else 0
    groups = {}
    for r in rows:
        groups.setdefault((r["host"], r["variant"]), []).append(r)
    print(f"\n{'host/variant':14}{'n':>3}{'median s':>10}{'min':>6}{'max':>6}{'writers s':>11}{'1st create s':>14}{'in tok':>10}{'$':>7}")
    for (h, v), rs in sorted(groups.items()):
        totals = [r["total"] for r in rs]
        writers = [r["phases"].get("writers", r["phases"].get("all")) for r in rs]
        firsts = [next(c[0] for c in a["calls"] if c[1] == "create_document")
                  for r in rs for a in r["agents"]
                  if a["role"].startswith(("write:", "all")) and any(c[1] == "create_document" for c in a["calls"])]
        costs = [float(c) for c in (b.cost_cell(r["agents"]) for r in rs) if c != "n/a"]
        cost = f"{med(costs):.2f}" if costs else "n/a"
        print(f"{h + ' ' + v:14}{len(rs):3}{med(totals):10.0f}{min(totals):6.0f}{max(totals):6.0f}{med(writers):11.0f}"
              f"{med(firsts):14.0f}{med([sum(a['in_tok'] for a in r['agents']) for r in rs]):10}{cost:>7}")


if __name__ == "__main__":
    if sys.argv[1:] == ["report"]:
        report(load_runs())
        sys.exit()
    ap = argparse.ArgumentParser()
    ap.add_argument("host", choices=["claude", "codex"])
    ap.add_argument("variant", choices=["A", "B", "C", "D"])
    ap.add_argument("--strong")
    ap.add_argument("--fast")
    a = ap.parse_args()
    s, f = DEFAULTS[a.host]
    run(a.host, a.variant, a.strong or s, a.fast or f)
