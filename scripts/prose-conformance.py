#!/usr/bin/env python3
"""Measure a document corpus against the prose canon.

Reports the checks the shared rule `concepts/document-prose-canon` states but
that the engine does not yet measure, so the effect of adding a check is a
number rather than an opinion. Read-only: it never edits a document.

Usage:
    scripts/prose-conformance.py [repo-root ...]

With no argument it measures the three ecosystem repositories relative to this
one. A "logical item" is a numbered clause with its wrapped continuation lines
joined, so a clause that spans two source lines counts once.

Every predicate below mirrors `internal/advisory/precision.go` deliberately. A
number produced here justifies a check that ships there, so a divergence makes
the justification measure something the engine will never report. Where the two
languages spell a guard differently the difference is noted at the pattern.
"""
import collections
import os
import re
import sys

MODAL = re.compile(r"\b(MUST|SHOULD|SHALL|MAY)\b")
NOT_MODAL = re.compile(r"\b(MUST|SHOULD|SHALL|MAY) NOT\b")
# the trailing (\s|$) keeps a version out: "1.25 or newer" is not clause 1
NUMBERED = re.compile(r"^\s*[0-9]+\.(\s|$)")
LEADING_NUMBER = re.compile(r"^\s*[0-9]+\.\s*")
# the emphasis run is optional: the corpus writes "1. **WHEN** ..."
EARS_OPENER = re.compile(r"^\s*[0-9]+\.\s*[*_]*(WHEN|WHILE|IF)\b")
INLINE_CODE = re.compile(r"`[^`]*`")
# A condition word after the modal: the obligation came before its trigger.
# The modal stays case-sensitive, as in the engine — a graded requirement is
# uppercase by contract, and "must" in prose is not one. "once" is absent for
# the same reason it is absent there: in requirement prose it is an adverb.
# [\W\d_] is Python's spelling of the engine's [^\p{L}] — "not a letter".
TRAILING_COND = re.compile(
    r"\b(MUST|SHOULD|MAY)( NOT)?\b.*?"
    r"(?:(?i:\b(when|if|while|unless|whenever)\b)"
    r"|(?:^|[\W\d_])(если|когда)(?:[\W\d_]|$))"
)

# S8 and S9 of the prose canon, kept as literal lists so they can be compared
# line by line against templates.OpenListMarkers / templates.AmbiguousAlternatives.
OPEN_LIST_MARKERS = ["etc.", "and so on", "и т.д.", "и т. д.", "и т.п.", "и т. п."]
AMBIGUOUS_ALTERNATIVES = ["and/or", "и/или"]


def marker_re(markers):
    """Left-guarded, right-unguarded — the engine's buildMarkerRe.

    An entry may end in punctuation ("etc."), where a trailing boundary would
    demand a word character after the dot. Unguarded on the left, "and so on"
    matched inside "a brand so on the shelf".
    """
    alternation = "|".join(re.escape(m) for m in markers)
    return re.compile(rf"(?:^|[\W\d_])({alternation})", re.I)


OPEN_LIST = marker_re(OPEN_LIST_MARKERS)
AND_OR = marker_re(AMBIGUOUS_ALTERNATIVES)

# S2 of the prose canon
MAX_STEP_WORDS = 20
MAX_CLAUSE_WORDS = 25

# section heading → which normative body the type keeps its clauses in.
# Mirrors templates.ClauseSections, names and aliases in full. The four
# ISO 29148 types are absent there and here: they carry requirements as
# identified table rows, not as clauses.
NORMATIVE_SECTIONS = {
    "spec": ("Normative Behavior", "Failure Behavior", "Error Handling"),
    "rule": ("Rule",),
}
# mirrors templates.StepSections
STEP_SECTIONS = {
    "guide": ("Steps", "Procedure"),
    "task-type": ("Steps",),
    "plan": ("Tasks",),
}


def heading_matches(head, names):
    """Mirror the engine's sectionNameMatches: the name, then a word boundary.

    Case-sensitive full names, as in Go — a loose lowercase prefix let "Rules"
    count as the Rule section here while the engine rejected it.
    """
    return any(
        head == n or head.startswith((n + " ", n + "\t")) for n in names
    )
# types the canon puts on the ISO profile: a modal in a numbered clause is a
# defect. A step is graded as a step, so it is not counted here twice.
ISO_PROFILE = {"adr", "rfc", "doc", "prd", "plan", "idea", "rnd", "cpat",
               "mrd", "brd", "urd"}


def strip_code(lines):
    """Drop fenced code blocks: an example is not the document's own prose.

    An unpaired run of fences is read as no fence at all, as in the engine:
    swallowing the rest of the document would make the count go quiet exactly
    where a stray marker sits.
    """
    if sum(1 for line in lines if line.lstrip().startswith("```")) % 2:
        return list(lines)
    out, in_fence = [], False
    for line in lines:
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
            continue
        if not in_fence:
            out.append(line)
    return out


def sections(lines):
    """Yield (heading, body). The text before the first heading is yielded under
    the empty heading, because the engine's whole-body scan sees it too."""
    head, buf = "", []
    for line in lines:
        if line.startswith("## "):
            yield head, buf
            head, buf = line[3:].strip(), []
        else:
            buf.append(line)
    yield head, buf


def logical_items(buf):
    """Join wrapped continuation lines so one clause counts once.

    A heading line is neither an item nor part of one: folded in, "### Notes"
    and the prose under it were charged to the last clause's word budget.
    """
    items, cur = [], None
    for line in buf:
        if line.lstrip().startswith("#"):
            if cur:
                items.append(cur)
            cur = None
        elif NUMBERED.match(line):
            if cur:
                items.append(cur)
            cur = line.strip()
        elif cur is not None:
            if line.strip() == "":
                items.append(cur)
                cur = None
            else:
                cur += " " + line.strip()
    if cur:
        items.append(cur)
    return items


def word_count(item):
    """Words without the item's own numbering: "1." is the document's structure,
    not the author's word budget. The engine's wordCount does the same."""
    return len(LEADING_NUMBER.sub("", item).split())


def measure(roots):
    per_type = collections.defaultdict(collections.Counter)
    for root in roots:
        base = os.path.join(root, ".archcore")
        if not os.path.isdir(base):
            print(f"skip: no .archcore under {root}", file=sys.stderr)
            continue
        for dirpath, _, files in os.walk(base):
            for name in files:
                match = re.match(r".*\.([a-z-]+)\.md$", name)
                if not match:
                    continue
                dtype = match.group(1)
                counter = per_type[dtype]
                counter["docs"] += 1
                with open(os.path.join(dirpath, name), encoding="utf-8") as fh:
                    lines = strip_code(fh.read().split("\n"))

                open_list_doc = and_or_doc = False
                for head, buf in sections(lines):
                    normative = heading_matches(head, NORMATIVE_SECTIONS.get(dtype, ()))
                    step = heading_matches(head, STEP_SECTIONS.get(dtype, ()))
                    for item in logical_items(buf):
                        words = word_count(item)
                        if normative:
                            counter["clauses"] += 1
                            if not MODAL.search(item):
                                counter["clause_no_modal"] += 1
                            elif not EARS_OPENER.match(item):
                                if TRAILING_COND.search(INLINE_CODE.sub(" ", item)):
                                    counter["clause_trailing_cond"] += 1
                            if EARS_OPENER.match(item):
                                counter["clause_ears"] += 1
                            if words > MAX_CLAUSE_WORDS:
                                counter["clause_over"] += 1
                            collapsed = NOT_MODAL.sub(r"\1", item)
                            if len(MODAL.findall(collapsed)) >= 2:
                                counter["clause_two_modals"] += 1
                        if step:
                            counter["steps"] += 1
                            if words > MAX_STEP_WORDS:
                                counter["step_over"] += 1
                            if MODAL.search(item):
                                counter["step_modal_items"] += 1
                        # S8 and S9 reach clauses and steps only, which is the
                        # engine's scope: markerHits(openListRe, clauses, steps).
                        # Inline code comes out first, as in the engine: a
                        # backticked marker is named, not used.
                        if normative or step:
                            plain = INLINE_CODE.sub(" ", item)
                            open_list_doc |= bool(OPEN_LIST.search(plain))
                            and_or_doc |= bool(AND_OR.search(plain))
                        if dtype in ISO_PROFILE and not step and MODAL.search(item):
                            counter["iso_modal_items"] += 1
                counter["open_list_docs"] += int(open_list_doc)
                counter["and_or_docs"] += int(and_or_doc)
    return per_type


def pct(part, whole):
    return f"{100 * part / whole:.0f}%" if whole else "—"


def main():
    roots = sys.argv[1:]
    if not roots:
        here = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        parent = os.path.dirname(here)
        roots = [os.path.join(parent, r) for r in ("cli", "plugin", "global")]

    per_type = measure(roots)
    total_docs = sum(c["docs"] for c in per_type.values())

    print(f"corpus: {total_docs} documents in {len(roots)} repositor"
          f"{'y' if len(roots) == 1 else 'ies'}\n")

    print("Graded clauses (spec, rule) — canon S2: clause <= 25 words")
    print(f"{'type':<8}{'clauses':>9}{'no modal':>10}{'2 modals':>10}"
          f"{'EARS open':>11}{'cond after':>12}{'over 25w':>10}")
    for dtype in sorted(NORMATIVE_SECTIONS):
        c = per_type.get(dtype)
        if not c or not c["clauses"]:
            continue
        n = c["clauses"]
        print(f"{dtype:<8}{n:>9}{pct(c['clause_no_modal'], n):>10}"
              f"{pct(c['clause_two_modals'], n):>10}{pct(c['clause_ears'], n):>11}"
              f"{pct(c['clause_trailing_cond'], n):>12}{pct(c['clause_over'], n):>10}")

    print("\nProcedure steps (guide, task-type, plan) — canon S2: step <= 20 words")
    for dtype in sorted(STEP_SECTIONS):
        c = per_type.get(dtype)
        if not c or not c["steps"]:
            continue
        print(f"{dtype:<12}{c['steps']:>5} steps   over 20w: "
              f"{pct(c['step_over'], c['steps'])}   with a modal: "
              f"{pct(c['step_modal_items'], c['steps'])}")

    print("\nISO-profile types — canon rule 5: no BCP 14 modal in a numbered clause")
    for dtype in sorted(ISO_PROFILE):
        c = per_type.get(dtype)
        if not c or not c["iso_modal_items"]:
            continue
        print(f"{dtype:<12}{c['iso_modal_items']:>4} numbered clause(s) carry a modal"
              f"   ({c['docs']} docs)")

    open_docs = sum(c["open_list_docs"] for c in per_type.values())
    andor_docs = sum(c["and_or_docs"] for c in per_type.values())
    print(f"\nS8 open-ended list marker in a clause or step: {open_docs} document(s)")
    print(f"S9 ambiguous alternative in a clause or step: {andor_docs} document(s)")


if __name__ == "__main__":
    main()
