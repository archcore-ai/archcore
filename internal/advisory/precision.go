package advisory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"archcore-cli/internal/docs"
	"archcore-cli/templates"
)

// Precision check.
//
// Advisory only, and deliberately over-eager: a false "look at this" costs a
// glance, a missed vague requirement costs an argument three months later. The
// canon it measures against lives in templates/precision.go; this file is only
// the engine, so either can change without touching the other.

const (
	// maxPrecisionHits caps how many examples one finding lists.
	maxPrecisionHits = 5
	// maxCrossDocHits caps the cross-document reference finding.
	maxCrossDocHits = 3
	// maxPassiveHits caps the subjectless-passive finding.
	maxPassiveHits = 3
)

var (
	// enVaguenessRe is built from the canon with word boundaries: English words
	// do not inflect enough to need substring matching, and "variously" is not
	// the defect "various" is.
	enVaguenessRe = buildLexiconRe(templates.VaguenessLexiconEN)
	// crossDocRefRe finds links to other .archcore/ documents in the body.
	crossDocRefRe = regexp.MustCompile(`\.archcore/[A-Za-z0-9_./-]+\.md`)
	// numberedLineRe finds the numbered clauses of a spec's normative sections.
	numberedLineRe = regexp.MustCompile(`^\s*[0-9]+\.`)
	// modalRe finds BCP 14 modals. Case-sensitive: a graded requirement is
	// uppercase by contract, and "must" in prose is not one.
	modalRe = regexp.MustCompile(`\b(MUST|SHOULD|SHALL|MAY)\b`)
	// notModalRe collapses the negative forms so "MUST NOT" counts once.
	notModalRe = regexp.MustCompile(`\b(MUST|SHOULD|SHALL|MAY) NOT\b`)
	// shallRe finds the notation a graded spec must not use.
	shallRe = regexp.MustCompile(`\bSHALL\b`)
	// passiveRe finds an obligation with no obligated subject, in English
	// ("MUST be rotated") and Russian ("MUST ротироваться").
	//
	// The Russian alternative carries no closing \b: Go's is an ASCII word
	// boundary, and a Cyrillic letter is not an ASCII word character, so the
	// boundary never holds and the whole alternative could never match. The
	// shell original relied on grep's Unicode-aware \b.
	passiveRe = regexp.MustCompile(`\b(MUST|SHOULD)( NOT)? be [a-z]+(ed|en)\b|\b(MUST|SHOULD)( NOT)? \S*(ться|тся)`)
)

// buildLexiconRe compiles a case-insensitive word-boundary alternation over a
// word list. The entries are quoted because the lexicon is an exported var:
// without it, one entry carrying a metacharacter panics at package init and
// takes down every archcore command, not just the hooks.
func buildLexiconRe(words []string) *regexp.Regexp {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
}

// Precision checks a freshly written document and returns the findings.
func Precision(baseDir, tool, docPath string) string {
	if !strings.HasSuffix(tool, "create_document") && !strings.HasSuffix(tool, "update_document") {
		return ""
	}
	if docPath == "" {
		return ""
	}

	// docPath comes from hook stdin. Validate it before it reaches the
	// filesystem: an unchecked "../" reads a file outside the project and puts
	// what it finds in front of the model.
	//
	// ValidateReadPath rather than ValidateArchcorePath: the lexical check alone
	// leaves a symlinked ancestor (.archcore/escape -> /elsewhere) resolving out
	// of the store, and the findings below echo the file's own wording and its
	// document links. Nil globals keeps the external-global branch closed — this
	// advisory only ever fires after a write, and a global is never written.
	if filepath.IsAbs(docPath) {
		return "" // hosts address documents relative to the store
	}
	rel := filepath.ToSlash(docPath)
	if !strings.HasPrefix(rel, ".archcore/") {
		rel = ".archcore/" + rel
	}
	rel, err := docs.ValidateReadPath(baseDir, rel, nil)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(rel)))
	if err != nil {
		return "" // the write may have been a removal, or the path is not ours
	}

	docType := templates.DocumentType(templates.ExtractDocType(filepath.Base(rel)))
	fm, body, _ := templates.SplitDocument(data)
	findings := PrecisionFindings(docType, fm, body)
	if len(findings) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Precision] %s (advisory):\n", rel)
	for _, f := range findings {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	return strings.TrimRight(b.String(), "\n")
}

// PrecisionFindings runs every check. Order is fixed so the output reads the
// same way each time.
func PrecisionFindings(docType templates.DocumentType, fm templates.Frontmatter, body string) []string {
	var out []string
	lines := strings.Split(body, "\n")

	if hits := findVaguenessHits(body); len(hits) > 0 {
		out = append(out, fmt.Sprintf("vague wording (%s) — replace with a concrete fact, version, threshold, or measurement",
			strings.Join(hits, ", ")))
	}

	for _, section := range templates.RequiredSections[docType] {
		if !hasSection(lines, section) {
			out = append(out, fmt.Sprintf("missing section: ## %s", section.Name))
		}
	}

	if strings.TrimSpace(fm.Title) == "" {
		out = append(out, "frontmatter: missing or empty title")
	}
	if !templates.IsValidStatus(fm.Status) {
		out = append(out, "frontmatter: missing or invalid status")
	}

	// Counted in runes: MinBodyChars is a character floor, and len() would make
	// a 120-character Russian document read as 240 and skip the check entirely —
	// while the number printed alongside it claims to be characters.
	if n := utf8.RuneCountInString(strings.TrimSpace(body)); n < templates.MinBodyChars {
		out = append(out, fmt.Sprintf("body is %d characters (under %d — likely a placeholder)", n, templates.MinBodyChars))
	}

	if hits := findCrossDocHits(body); len(hits) > 0 {
		out = append(out, fmt.Sprintf("body links other .archcore/ documents (%s) — move these to the relation graph with add_relation",
			strings.Join(hits, ", ")))
	}

	if templates.ArchitectVoiceTypes[docType] && hasLongCodeBlock(lines) {
		out = append(out, fmt.Sprintf("code block of %d+ lines in %s — prefer an @path/to/file reference over pasted implementation detail",
			templates.MaxCodeBlockLines, docType))
	}

	if docType == templates.TypeSpec {
		out = append(out, specFindings(body, lines)...)
	}
	return out
}

// specFindings are the checks that apply only to a normative contract.
func specFindings(body string, lines []string) []string {
	var out []string

	if shallRe.MatchString(body) {
		out = append(out, "spec notation: SHALL found — grade with BCP 14 modals instead (MUST / SHOULD / MAY)")
	}

	if n := strings.Count(strings.TrimSpace(body), "\n") + 1; n > templates.MaxSpecBodyLines {
		out = append(out, fmt.Sprintf("spec body is %d lines (cap %d) — a spec that long is describing, not specifying",
			n, templates.MaxSpecBodyLines))
	}

	// Only numbered clauses are graded requirements; prose and constraint lists
	// are not, and checking them would make the finding meaningless.
	for _, line := range lines {
		if !numberedLineRe.MatchString(line) {
			continue
		}
		collapsed := notModalRe.ReplaceAllString(line, "$1")
		if len(modalRe.FindAllString(collapsed, -1)) >= 2 {
			out = append(out, `compound requirement — a numbered clause carries two modals; split "MUST X and MUST NOT Y" into two clauses`)
			break
		}
	}

	if passives := findPassiveHits(lines); len(passives) > 0 {
		out = append(out, fmt.Sprintf("subjectless passive (%s) — name the obligated component as the subject, or use WHEN <trigger> for an event",
			strings.Join(passives, ", ")))
	}

	return out
}

// findPassiveHits collects distinct subjectless passives from numbered clauses,
// capped at maxPassiveHits.
func findPassiveHits(lines []string) []string {
	var out []string
	for _, line := range lines {
		if !numberedLineRe.MatchString(line) {
			continue
		}
		for _, hit := range passiveRe.FindAllString(line, -1) {
			if slices.Contains(out, hit) {
				continue
			}
			out = append(out, hit)
			if len(out) == maxPassiveHits {
				return out
			}
		}
	}
	return out
}

// findVaguenessHits collects the distinct offenders, then keeps the first
// maxPrecisionHits in sorted order.
//
// Sorting before the cap, not after: capping in encounter order makes the
// reported set a function of where the words appear, so reformatting a document
// silently changes which offenders it names.
func findVaguenessHits(body string) []string {
	var hits []string
	add := func(s string) {
		s = strings.ToLower(s)
		if !slices.Contains(hits, s) {
			hits = append(hits, s)
		}
	}

	for _, m := range enVaguenessRe.FindAllString(body, -1) {
		add(m)
	}
	lower := strings.ToLower(body)
	for _, stem := range templates.VaguenessLexiconRU {
		if strings.Contains(lower, stem) {
			add(stem)
		}
	}
	for _, phrase := range templates.VaguenessPhrases {
		if strings.Contains(lower, phrase) {
			add(phrase)
		}
	}
	slices.Sort(hits)
	return hits[:min(len(hits), maxPrecisionHits)]
}

// findCrossDocHits collects the distinct document references, then keeps the
// first maxCrossDocHits in sorted order (see findVaguenessHits).
func findCrossDocHits(body string) []string {
	var hits []string
	for _, m := range crossDocRefRe.FindAllString(body, -1) {
		if !slices.Contains(hits, m) {
			hits = append(hits, m)
		}
	}
	slices.Sort(hits)
	return hits[:min(len(hits), maxCrossDocHits)]
}

// hasSection reports whether the body carries a level-2 heading matching the
// rule, under its canonical name or any historical alias.
func hasSection(lines []string, rule templates.SectionRule) bool {
	for _, line := range lines {
		rest, ok := strings.CutPrefix(line, "##")
		if !ok || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		rest = strings.TrimLeft(rest, " \t")
		if sectionNameMatches(rest, rule.Name) {
			return true
		}
		for _, alias := range rule.Aliases {
			if sectionNameMatches(rest, alias) {
				return true
			}
		}
	}
	return false
}

// sectionNameMatches reports whether a heading's text starts with name followed
// by whitespace or the end of the line — so "## Purpose & Scope" satisfies
// "Purpose" but "## Purposeful" does not.
func sectionNameMatches(heading, name string) bool {
	rest, ok := strings.CutPrefix(heading, name)
	if !ok {
		return false
	}
	rest = strings.TrimRight(rest, "\r")
	return rest == "" || rest[0] == ' ' || rest[0] == '\t'
}

// hasLongCodeBlock reports whether any fenced block holds MaxCodeBlockLines or
// more content lines. The fences themselves are not counted, and an unclosed
// block never triggers.
func hasLongCodeBlock(lines []string) bool {
	inBlock, count := false, 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t\r\f\v"), "```") {
			if inBlock && count >= templates.MaxCodeBlockLines {
				return true
			}
			inBlock = !inBlock
			count = 0
			continue
		}
		if inBlock {
			count++
		}
	}
	return false
}
