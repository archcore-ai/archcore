#!/bin/sh
# extract-gates.sh — emit normalized gate records from a track file.
#
# Usage: extract-gates.sh <track-file>
#
# Reads a track file (skills/_shared/tracks/<track>.md) and prints one record
# per '### gate: <name>' section, in file order, with a blank line after each
# record. Each record has exactly these six lines in this fixed order:
#
#   gate: <track>.<stage>       gate name from the '### gate:' heading
#   skip_when: yes|no           do the gate's Entry conditions declare skip_when?
#   budget: <value>             the Elicitation knobs budget value
#   taxonomy: <value>           the taxonomy knob, comma-joined ('none' if none)
#   produces: <type> <status>   Produces type + status, or 'none'
#   next: <targets...>          Next target gate names, and/or 'end' when the
#                               Next line names exit; space-separated
#
# Normalization (keeps the output stable across cosmetic prose edits):
#   - continuation lines of a wrapped bullet are joined; runs of whitespace
#     collapse to one space; leading/trailing whitespace is trimmed
#   - backticks are stripped
#   - rationale prose is cut: a value is truncated at the first '[assumption]'
#     marker or the first ' <em dash> ' separator, whichever comes first; one
#     trailing '.' is then stripped
#   - taxonomy: the boilerplate suffix ' from skills/_shared/coverage-taxonomy.md'
#     is stripped
#   - produces: an inline '- Produces: none ...' value normalizes to 'none';
#     otherwise the record joins the normalized 'type:' and 'status:' subfield
#     values with one space
#   - next: tokens matching the gate-name shape <track>.<stage> (lowercase
#     words joined by one dot, hyphens allowed) are kept in order of first
#     appearance, deduplicated; tokens containing '/' (contract paths) are
#     dropped; the word 'exit' contributes the terminal marker 'end'
#
# A field absent from the gate record prints with an empty value, so drift in
# field presence is visible in a diff. Portable POSIX sh + awk (no GNU-only
# features).
set -eu

if [ $# -ne 1 ]; then
  echo "usage: $0 <track-file>" >&2
  exit 2
fi
if [ ! -f "$1" ]; then
  echo "extract-gates: no such file: $1" >&2
  exit 2
fi

awk '
function trim(s) { sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s); return s }
function squeeze(s) { gsub(/[ \t]+/, " ", s); return trim(s) }
function stripticks(s) { gsub(/`/, "", s); return s }

# Truncate at the earliest of "[assumption]" or " <em dash> " (rationale
# prose), then trim and strip one trailing period. \342\200\224 is the UTF-8
# em dash, written in octal so the script survives encoding mangling.
function cutmeta(s,   i, j, p) {
  p = 0
  i = index(s, "[assumption]")
  if (i > 0) p = i
  j = index(s, " \342\200\224 ")
  if (j > 0 && (p == 0 || j < p)) p = j
  if (p > 0) s = substr(s, 1, p - 1)
  s = trim(s)
  sub(/\.$/, "", s)
  return trim(s)
}

function normval(s) { return cutmeta(squeeze(stripticks(s))) }

function normtax(s,   i) {
  s = normval(s)
  i = index(s, " from skills/_shared/coverage-taxonomy.md")
  if (i > 0) s = substr(s, 1, i - 1)
  sub(/\.$/, "", s)
  return trim(s)
}

function normnext(s,   out, arr, n, i, t, endf, seen) {
  s = stripticks(s)
  gsub("[^A-Za-z0-9._/-]", " ", s)
  n = split(s, arr, " ")
  out = ""
  endf = 0
  for (i = 1; i <= n; i++) {
    t = arr[i]
    sub(/\.+$/, "", t)
    if (t == "") continue
    if (index(t, "/") > 0) continue
    if (t == "exit" || t == "end") { endf = 1; continue }
    if (t ~ /^[a-z][a-z0-9-]*\.[a-z][a-z0-9-]*$/ && !(t in seen)) {
      seen[t] = 1
      out = (out == "" ? t : out " " t)
    }
  }
  if (endf) out = (out == "" ? "end" : out " end")
  return out
}

# Finalize the accumulated logical line (a bullet plus its continuations)
# into the current gate buffers.
function flushline(   line, v, sb) {
  if (cur == "") return
  line = cur
  cur = ""
  if (line ~ /^- /) {
    if (line ~ /^- Purpose:/) topf = "purpose"
    else if (line ~ /^- Entry conditions:/) topf = "entry"
    else if (line ~ /^- Elicitation knobs:/) topf = "knobs"
    else if (line ~ /^- Produces:/) {
      topf = "produces"
      v = line
      sub(/^- Produces:/, "", v)
      if (trim(v) != "") produces_inline = v
    }
    else if (line ~ /^- Exit checks:/) topf = "exitchecks"
    else if (line ~ /^- Next:/) {
      topf = "next"
      v = line
      sub(/^- Next:/, "", v)
      next_raw = v
    }
    else topf = "other"
  } else if (line ~ /^[ \t]+- /) {
    sb = line
    sub(/^[ \t]+- /, "", sb)
    if (topf == "entry" && sb ~ /^skip_when:/) skipw = 1
    else if (topf == "knobs" && sb ~ /^budget:/) { sub(/^budget:/, "", sb); budget_raw = sb }
    else if (topf == "knobs" && sb ~ /^taxonomy:/) { sub(/^taxonomy:/, "", sb); tax_raw = sb }
    else if (topf == "produces" && sb ~ /^type:/) { sub(/^type:/, "", sb); ptype_raw = sb }
    else if (topf == "produces" && sb ~ /^status:/) { sub(/^status:/, "", sb); pstat_raw = sb }
  }
}

function flushgate(   prod) {
  if (gate == "") return
  flushline()
  print "gate: " gate
  print "skip_when: " (skipw ? "yes" : "no")
  print "budget: " normval(budget_raw)
  print "taxonomy: " normtax(tax_raw)
  if (trim(produces_inline) != "") prod = normval(produces_inline)
  else if (trim(ptype_raw) != "") {
    prod = normval(ptype_raw)
    if (trim(pstat_raw) != "") prod = prod " " normval(pstat_raw)
  }
  else prod = ""
  print "produces: " prod
  print "next: " normnext(next_raw)
  print ""
  gate = ""; skipw = 0; topf = ""
  budget_raw = ""; tax_raw = ""; produces_inline = ""
  ptype_raw = ""; pstat_raw = ""; next_raw = ""
}

# Any heading ends the current gate section; a "### gate:" heading opens one.
/^#/ {
  flushgate()
  if ($0 ~ /^### gate:/) {
    gate = $0
    sub(/^### gate:[ \t]*/, "", gate)
    gate = trim(gate)
  }
  next
}

{
  if (gate == "") next
  if ($0 ~ /^[ \t]*- /) { flushline(); cur = $0 }
  else if ($0 ~ /^[ \t]*$/) flushline()
  else if (cur != "") cur = cur " " trim($0)
  next
}

END { flushgate() }
' "$1"
