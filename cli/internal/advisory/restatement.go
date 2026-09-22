package advisory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"archcore-cli/internal/docs"
	archsync "archcore-cli/internal/sync"
	"archcore-cli/templates"
)

// Restatement check.
//
// Ownership rule 1 of the content-kind ownership table
// (prd-spec-plan-content-ownership.adr): a statement written into one document
// is not written into a second. A track produces several documents on one
// topic and links them with `implements`, so this check reads the documents the
// written one builds on and reports a statement that survived the move nearly
// word for word.
//
// Near-verbatim only, by design. Two documents on one topic share vocabulary,
// and a paraphrase scores far under the threshold — a prd requirement and the
// spec behavior that grades it are meant to differ. What this catches is the
// copy: the measured failure was one sentence appearing in an idea, a prd, and
// a plan unchanged.

const (
	// minRestatementTokens is the shortest statement worth comparing. Below it
	// a full overlap means two short lines share a few words, not that one was
	// copied from the other.
	minRestatementTokens = 6
	// restatementThreshold is the overlap at which two statements are the same
	// statement. Measured against this corpus: a copied line scores 1.0, and
	// the closest genuine prd/spec pair scores 0.67.
	restatementThreshold = 0.85
	// maxRestatementTargets caps the upstream documents read per write, so a
	// densely linked document cannot stall the hook.
	maxRestatementTargets = 5
	// maxRestatementHits caps the finding.
	maxRestatementHits = 3
	// maxQuotedLineRunes is how much of an offending line a finding echoes.
	// Shared with the prd notation findings in precision.go, which quote a line
	// back to the author for the same reason.
	maxQuotedLineRunes = 60
)

// contentRelations are the relation types that make one document's content flow
// into another's.
//
// "related" and "depends_on" are both excluded: an association implies no
// content flow at all, and a dependency orders two documents without moving
// text between them. An overlap across either is a shared topic, not a copy.
var contentRelations = map[archsync.RelationType]bool{
	archsync.RelImplements: true,
	archsync.RelExtends:    true,
}

// listMarkerRe strips the bullet or number that opens a statement line.
var listMarkerRe = regexp.MustCompile(`^\s*(?:[-*+]|[0-9]+\.)\s+`)

// restatementStopwords are the tokens that carry no content for this
// comparison. The notation words are here for the same reason: a prd line and
// the spec line that grades it differ by exactly "MUST" and "WHEN", so counting
// those would report a difference where the substance is identical.
var restatementStopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "of": true,
	"to": true, "in": true, "on": true, "for": true, "is": true, "are": true,
	"be": true, "that": true, "this": true, "it": true, "as": true, "at": true,
	"by": true, "with": true, "from": true, "its": true, "one": true,
	"must": true, "should": true, "may": true, "shall": true, "not": true,
	"when": true, "while": true, "if": true, "then": true,
}

// Restatement reports statements in body that a document it builds on already
// carries. rel is the written document's path, with or without the store
// prefix.
func Restatement(baseDir, rel, body string) []string {
	statements := statementLines(body)
	if len(statements) == 0 {
		return nil
	}

	// Advisory read, fail-open: a manifest that will not load means the graph is
	// unavailable, not that the document is clean. A missed finding is the right
	// cost here; a blocked write is not.
	manifest, err := archsync.LoadManifest(baseDir)
	if err != nil {
		return nil
	}

	var findings []string
	reported := make([]bool, len(statements))
	// Both directions: a duplicate is a duplicate whichever side was written
	// last, and the spec that implements a prd is the outgoing edge of the spec
	// but the incoming edge of the prd.
	for _, neighbour := range contentNeighbours(manifest, rel, maxRestatementTargets) {
		other, ok := readStatements(baseDir, neighbour)
		if !ok {
			continue
		}

		for i, stmt := range statements {
			// One finding per statement, whatever the neighbour count: a line
			// copied into three linked documents is one defect, and reporting
			// it once per neighbour would spend the whole cap echoing one line.
			if reported[i] || !restates(stmt.tokens, other) {
				continue
			}
			reported[i] = true
			findings = append(findings, fmt.Sprintf(
				"%q restates .archcore/%s — one statement has one owning document; state what this document owns and link the two",
				quote(stmt.text), neighbour))
			if len(findings) == maxRestatementHits {
				return findings
			}
		}
	}
	return findings
}

// contentNeighbours lists the documents whose content flows into rel or out of
// it, without repeats, capped at n.
//
// Sorted before the cut, not after: the manifest keeps relations in insertion
// order, so cutting that order would let an unrelated relation added later
// decide which neighbours this check reads — and therefore which findings it
// reports.
func contentNeighbours(manifest *archsync.Manifest, rel string, n int) []string {
	self := docs.NormalizeRelPath(rel)
	outgoing, incoming := manifest.RelationsFor(self)

	var out []string
	add := func(path string) {
		if path != self {
			out = append(out, path)
		}
	}
	for _, r := range outgoing {
		if contentRelations[r.Type] {
			add(r.Target)
		}
	}
	for _, r := range incoming {
		if contentRelations[r.Type] {
			add(r.Source)
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	return out[:min(len(out), n)]
}

// statement is one comparable line of a document body. tokens and set hold the
// same words: every statement of the written document is compared against every
// statement of every neighbour, so the lookup side is a set rather than a scan.
type statement struct {
	text   string
	tokens []string
	set    map[string]bool
}

// quote trims a line to an echo short enough to read inside a finding, counted
// in runes so a Russian line is cut where the reader would cut it. Findings
// quote rather than number: the line count here is relative to the body, and a
// reader counting lines in the file would land somewhere else.
func quote(s string) string {
	r := []rune(s)
	if len(r) > maxQuotedLineRunes {
		return string(r[:maxQuotedLineRunes]) + "…"
	}
	return s
}

// statementLines collects the list items of a body — the form every content
// contract uses for a requirement, a behavior, a task, and a criterion.
// Prose paragraphs are skipped: they restate by summarizing, which this
// comparison would report at random.
func statementLines(body string) []statement {
	var out []statement
	inFence := false
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence || !listMarkerRe.MatchString(line) {
			continue
		}
		text := strings.TrimSpace(listMarkerRe.ReplaceAllString(line, ""))
		tokens := contentTokens(text)
		if len(tokens) < minRestatementTokens {
			continue
		}
		set := make(map[string]bool, len(tokens))
		for _, t := range tokens {
			set[t] = true
		}
		out = append(out, statement{text: text, tokens: tokens, set: set})
	}
	return out
}

// readStatements loads one upstream document and returns its comparable lines.
// The path comes from the manifest, and the manifest is project data, so it
// passes the same read guard as any other hook input.
func readStatements(baseDir, target string) ([]statement, bool) {
	// Both reads are advisory and fail open: a neighbour that will not validate
	// or will not open is a neighbour this check cannot judge. Skipping it costs
	// one missed comparison, where refusing the write would cost the author the
	// write itself.
	rel, err := docs.ValidateReadPath(baseDir, ".archcore/"+target, nil)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(rel)))
	if err != nil {
		return nil, false
	}
	_, body, _ := templates.SplitDocument(data)
	lines := statementLines(body)
	return lines, len(lines) > 0
}

// restates reports whether tokens reproduce any upstream statement.
func restates(tokens []string, upstream []statement) bool {
	for _, u := range upstream {
		if overlap(tokens, u.set) >= restatementThreshold {
			return true
		}
	}
	return false
}

// overlap is the shared fraction of the shorter statement. Dividing by the
// shorter side is what lets an upstream line survive into a longer downstream
// one — the observed shape, where a copied sentence picks up a trailing clause.
//
// Both sides are already deduplicated by contentTokens, so len(b) is its token
// count and every hit counts once.
func overlap(a []string, b map[string]bool) float64 {
	shorter := min(len(a), len(b))
	if shorter == 0 {
		return 0
	}
	shared := 0
	for _, t := range a {
		if b[t] {
			shared++
		}
	}
	return float64(shared) / float64(shorter)
}

// contentTokens lowercases a line and keeps the words that carry its content,
// deduplicated so a repeated word cannot inflate an overlap past the threshold
// on its own. Splitting on non-letter/non-digit runes rather than on ASCII
// keeps a Russian document comparable.
func contentTokens(line string) []string {
	lowered := strings.ToLower(line)
	fields := strings.FieldsFunc(lowered, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var out []string
	for _, f := range fields {
		if restatementStopwords[f] {
			continue
		}
		out = append(out, f)
	}
	slices.Sort(out)
	return slices.Compact(out)
}
