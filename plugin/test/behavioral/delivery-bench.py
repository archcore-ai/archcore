#!/usr/bin/env python3
"""Delivery bench: which documents reach the file context of an edit.

No model calls: it only sends read-only search_documents for_path calls to one `archcore mcp`.

  delivery-bench.py [--repo DIR] [--cli-bin PATH] [--out FILE]

Files: every file an accepted rule, cpat, adr, spec, scenario, guide, or doc names by path, plus
200 tracked code files chosen with seed 7. Reported per run:
- accepted documents per type: delivered, displaced by the cap of 5, not matched, no anchor;
- files at the matched cap, and the mean and maximum candidate count;
- the general part: local and global rows shown, rows omitted;
- bare-name rows: `file` rows whose document names only the base name, per source kind, split by
  whether another tracked file carries that name (bare-file-name-matches-only-when-unique.adr).
"""
import argparse, collections, glob, json, os, random, re, subprocess, sys

ap = argparse.ArgumentParser()
ap.add_argument("--repo", default=os.environ.get("REPO_ROOT") or os.getcwd())
ap.add_argument("--cli-bin", default=os.environ.get("ARCHCORE_BIN") or "archcore")
ap.add_argument("--out")
args = ap.parse_args()
os.chdir(args.repo)


class MCP:
    def __init__(self, binary):
        self.p = subprocess.Popen([binary, "mcp"], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.DEVNULL, text=True, bufsize=1)
        self.n = 0
        self.req("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                                "clientInfo": {"name": "delivery-bench", "version": "0"}})
        self.send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    def send(self, msg):
        self.p.stdin.write(json.dumps(msg) + "\n")
        self.p.stdin.flush()

    def req(self, method, params):
        self.n += 1
        self.send({"jsonrpc": "2.0", "id": self.n, "method": method, "params": params})
        while True:
            line = self.p.stdout.readline()
            if not line:
                raise RuntimeError("archcore mcp closed the stream")
            msg = json.loads(line)
            if msg.get("id") == self.n:
                return msg

    def for_path(self, path):
        msg = self.req("tools/call", {"name": "search_documents", "arguments": {"for_path": path}})
        text = msg.get("result", {}).get("content", [{}])[0].get("text", "{}")
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            return {"_error": text[:200]}

    def close(self):
        self.p.stdin.close()
        self.p.wait(timeout=10)


EDIT_TYPES = {"rule", "cpat", "adr", "spec", "scenario", "guide", "doc"}
REF = re.compile(r"@([A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-*]+)+)|`([A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-]+)+)`")


def norm(p):
    return p[len(".archcore/"):] if p.startswith(".archcore/") else p


docs, bodies = [], {}
for f in glob.glob(".archcore/**/*.md", recursive=True):
    m = re.search(r"\.([a-z-]+)\.md$", f)
    if not m:
        continue
    text = open(f, encoding="utf-8").read()
    status = re.search(r"^status:\s*(\S+)", text, re.M)
    files = {p for p in ((a or b).rstrip(".,;:)") for a, b in REF.findall(text))
             if not p.startswith(".archcore") and "*" not in p and os.path.isfile(p)}
    bodies[norm(f)] = text
    docs.append(dict(path=f, type=m.group(1), status=status.group(1) if status else "?", files=sorted(files)))

accepted = [d for d in docs if d["status"] == "accepted" and d["type"] in EDIT_TYPES]
anchored = sorted({p for d in accepted for p in d["files"]})
tracked = subprocess.run(["git", "ls-files"], capture_output=True, text=True, check=True).stdout.split("\n")
names = collections.Counter(os.path.basename(p) for p in tracked if p)
code = [p for p in tracked if re.search(r"\.(go|ts|tsx|js|sh|py|json|yaml|yml)$|/bin/[^/.]+$|SKILL\.md$|/_shared/.*\.md$", p)
        and not p.startswith(".archcore") and "/testdata/" not in p and os.path.isfile(p)]
random.seed(7)
sample = random.sample(code, min(200, len(code)))

mcp = MCP(args.cli_bin)
per_file = {}
for p in sorted(set(anchored) | set(sample)):
    r = mcp.for_path(p)
    rows = r.get("results", [])
    omitted = r.get("omitted", {})
    per_file[p] = dict(
        matched=[(norm(x["path"]), x["type"], x.get("reason"), x.get("source_kind")) for x in rows if x.get("reason") != "general"],
        general=[(norm(x["path"]), x["type"], x.get("source_kind")) for x in rows if x.get("reason") == "general"],
        omitted_matched=omitted.get("matched", 0), omitted_general=omitted.get("general", 0),
        error=r.get("_error"))
mcp.close()

by_type = collections.defaultdict(collections.Counter)
for d in accepted:
    c = by_type[d["type"]]
    c["accepted"] += 1
    if not d["files"]:
        c["no_anchor"] += 1
        continue
    me = norm(d["path"])
    if any(row[0] == me for p in d["files"] for row in per_file[p]["matched"]):
        c["delivered"] += 1
    elif any(per_file[p]["omitted_matched"] for p in d["files"]):
        c["displaced"] += 1
    else:
        c["not_matched"] += 1

def body_of(path):
    """A global row's path is relative to the project root, outside .archcore/; None when unreadable."""
    if path not in bodies:
        try:
            bodies[path] = open(path, encoding="utf-8").read()
        except OSError:
            bodies[path] = None
    return bodies[path]


bare = collections.defaultdict(collections.Counter)
for p, v in per_file.items():
    for path, _, reason, kind in v["matched"]:
        text = body_of(path)
        if reason == "file" and text is not None and "/" in p and p not in text:
            bare[kind]["shared" if names[os.path.basename(p)] > 1 else "unique"] += 1

candidates = [len(v["matched"]) + v["omitted_matched"] for v in per_file.values()]
general = next(iter(per_file.values()))
result = dict(
    files=len(per_file), anchored_files=len(anchored), sample_files=len(sample),
    by_type={t: dict(c) for t, c in sorted(by_type.items(), key=lambda kv: -kv[1]["accepted"])},
    files_at_matched_cap=sum(1 for v in per_file.values() if v["omitted_matched"]),
    candidates_mean=round(sum(candidates) / len(candidates), 2), candidates_max=max(candidates),
    general_shown=dict(collections.Counter(k for _, _, k in general["general"])),
    general_omitted=general["omitted_general"],
    bare_name_rows_shown={kind: dict(c) for kind, c in sorted(bare.items())},
    errors=[p for p, v in per_file.items() if v["error"]][:10],
)
if args.out:
    json.dump(result, open(args.out, "w"), indent=1)
json.dump(result, sys.stdout, indent=1)
print()
