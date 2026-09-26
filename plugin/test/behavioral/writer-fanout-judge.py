#!/usr/bin/env python3
"""Blind quality comparison of the documents produced by writer-fanout-bench.py.

For each (host, variant) the run with the median total time represents the group.
Each judge scores every anonymized document of one type against the code, the
content contract, and the precision rules. writer-fanout-judge.py prints mean scores per group
and per judge; raw verdicts go to $WRITER_BENCH_OUTPUT_DIR/judgements.json.
Runs real model calls and costs money; nothing runs it automatically.

The judges run with no tools, from an empty directory, so they cannot look up
which run wrote a document; every verdict records the judge's tool use to prove it.
The ground truth is the code snapshot the runs were measured against.
"""
import concurrent.futures as cf, json, os, random, re, subprocess, sys, tempfile

import benchlib as b

OUT = b.out_dir("WRITER_BENCH_OUTPUT_DIR", "writer-fanout-bench")
SHARED = os.path.join(b.REPO, "plugin/plugins/archcore/skills/_shared")
PKG = json.load(open(os.path.join(b.HERE, "fixtures", "writer-fanout-package.json")))
JUDGES = {"opus": ("claude", "claude-opus-5-5"), "astra": ("codex", "gpt-6-astra")}


def representative_runs():
    groups = {}
    for d in sorted(os.listdir(os.path.join(OUT, "runs"))):
        f = os.path.join(OUT, "runs", d, "result.json")
        if os.path.exists(f):
            r = json.load(open(f))
            groups.setdefault(f"{r['host']}-{r['variant']}", []).append(r)
    return {g: sorted(rs, key=lambda r: r["total"])[len(rs) // 2]["run"] for g, rs in groups.items()}


def ground_truth(runs):
    snapshots = set()
    for run in runs:
        truth = os.path.join(OUT, "runs", run, "truth")
        if not os.path.isdir(truth):
            sys.exit(f"{run} has no truth/ snapshot; judge only runs recorded by the current bench")
        snapshots.add(tuple(open(os.path.join(truth, f)).read() for f in ("update_document.go", "update_document_test.go")))
    if len(snapshots) != 1:
        sys.exit("the representative runs were measured against different code; judge one ref at a time")
    code, test = snapshots.pop()
    return code, test[test.index("func TestHandleUpdateDocument_Edits"):]


def prompt(doc, entries, code, test):
    contract = os.path.join(SHARED, f"{doc['type']}-contract.md")
    refs = [open(contract).read()] if os.path.exists(contract) else []
    refs.append(open(os.path.join(SHARED, "precision-rules.md")).read())
    ctx = "\n".join("- " + c for c in PKG["context"])
    docs = "\n\n".join(f"<document label=\"{lab}\">\n{text}\n</document>" for lab, text in entries)
    return f"""You are a strict reviewer in a blind comparison. Do not use tools; everything is below. Do not ask questions.

The documents below were written independently for the same request:
type={doc['type']} title="{doc['title']}" — {doc['intent']}

Facts the writers were given:
{ctx}

GROUND TRUTH CODE (cli/internal/mcp/tools/update_document.go):
<code>
{code}
</code>
<test>
{test}
</test>

RULES THE DOCUMENTS MUST FOLLOW (content contract when present, then precision rules):
{chr(10).join(f"<reference>{r}</reference>" for r in refs)}

DOCUMENTS:
{docs}

Score every document from 1 to 10 on:
- accuracy: every behavior claim matches the code; invented behavior, wrong error text, or wrong identifiers cost points;
- contract: required sections, line forms, and type rules are followed;
- precision: one idea per sentence, explicit actors, no vague wording, no filler;
- overall: how much you would trust and keep this document as written.
Reply with only a JSON array in a ```json fence:
[{{"label": "...", "accuracy": n, "contract": n, "precision": n, "overall": n, "note": "<one sentence: the biggest defect>"}}]"""


def ask(judge, text, blank):
    """Return the judge's verdicts and how many tool calls it made (expected: 0)."""
    host, model = JUDGES[judge]
    if host == "claude":
        out = subprocess.run(["claude", "-p", "--model", model, "--output-format", "json", "--no-session-persistence",
                              "--tools", "", "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}'],
                             input=text, cwd=blank, capture_output=True, text=True).stdout
        res = json.loads(out)
        reply, tool_calls = res["result"], res.get("num_turns", 1) - 1
    else:
        last = os.path.join(blank, f".judge-{os.getpid()}-{random.random()}.last")
        events = subprocess.run(["codex", "exec", "-m", model, "-s", "read-only", "--skip-git-repo-check", "--json", "-o", last, text],
                                cwd=blank, stdin=subprocess.DEVNULL, capture_output=True, text=True).stdout
        tool_calls = sum(1 for l in events.splitlines() if '"item.completed"' in l and '"agent_message"' not in l and '"reasoning"' not in l)
        reply = open(last).read()
        os.remove(last)
    block = re.search(r"```json\s*(.*?)```", reply, re.S)
    return json.loads(block.group(1) if block else reply), tool_calls


def main():
    reps = representative_runs()
    code, test = ground_truth(reps.values())
    blank = tempfile.mkdtemp(prefix="judge-")
    jobs = []
    for doc in PKG["docs"]:
        name = f"{doc['filename']}.{doc['type']}.md"
        entries = []
        for group, run in sorted(reps.items()):
            path = os.path.join(OUT, "runs", run, "docs", name)
            if os.path.exists(path):
                entries.append((group, open(path).read()))
        random.Random(doc["type"]).shuffle(entries)
        labels = {f"D{i + 1}": group for i, (group, _) in enumerate(entries)}
        text = prompt(doc, [(lab, entries[i][1]) for i, lab in enumerate(labels)], code, test)
        for judge in JUDGES:
            jobs.append((doc["type"], judge, labels, text))

    verdicts = []
    with cf.ThreadPoolExecutor(len(jobs)) as pool:
        futs = {pool.submit(ask, judge, text, blank): (t, judge, labels) for t, judge, labels, text in jobs}
        for fut in cf.as_completed(futs):
            t, judge, labels = futs[fut]
            try:
                scores, tool_calls = fut.result()
                for v in scores:
                    verdicts.append(dict(type=t, judge=judge, group=labels.get(v["label"], "?"), tool_calls=tool_calls, **v))
            except Exception as e:  # one failed judge call must not lose the others
                print(f"judge {judge} on {t} failed: {e}", file=sys.stderr)
    json.dump(dict(representatives=reps, verdicts=verdicts), open(os.path.join(OUT, "judgements.json"), "w"), indent=2)

    keys = ("accuracy", "contract", "precision", "overall")
    print("representative runs:", json.dumps(reps))
    print("judge tool calls (expected 0):", max((v["tool_calls"] for v in verdicts), default=0))
    print(f"{'group':10}{'judge':7}" + "".join(f"{k:>11}" for k in keys) + "   docs")
    for g in sorted({v["group"] for v in verdicts}):
        for j in JUDGES:
            vs = [v for v in verdicts if v["group"] == g and v["judge"] == j]
            if vs:
                print(f"{g:10}{j:7}" + "".join(f"{sum(v[k] for v in vs) / len(vs):11.1f}" for k in keys) + f"   {len(vs)}")


if __name__ == "__main__":
    main()
