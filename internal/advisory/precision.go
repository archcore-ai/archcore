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
	// maxReportFindings caps the findings one advisory report injects into the
	// model's context. Findings arrive in fixed check order, so the cut is
	// stable and the count of what it dropped is stated in the report itself.
	//
	// 12 rather than the original 5: each check emits at most ONE finding (its
	// own examples are already capped by maxPrecisionHits above), so the count
	// here is a count of distinct problem KINDS, not of occurrences. A
	// backlogged document trips a dozen kinds, and cutting at 5 hid whole
	// categories on exactly the documents that needed the report most — the cut
	// line stated a number, never which kinds went missing. 12 covers the
	// realistic spread per type while still bounding what one write injects.
	maxReportFindings = 12
)

var (
	// enVaguenessRe is built from the canon with word boundaries: English words
	// do not inflect enough to need substring matching, and "variously" is not
	// the defect "various" is.
	enVaguenessRe = buildLexiconRe(templates.VaguenessLexiconEN)
	// phraseVaguenessRe carries the same word boundaries. A raw substring match
	// reported "why the change was needed" as the phrase "as needed", which is
	// how the cpat template came back flagged for its own skeleton.
	phraseVaguenessRe = buildLexiconRe(templates.VaguenessPhrases)
	// crossDocRefRe finds links to other .archcore/ documents in the body.
	crossDocRefRe = regexp.MustCompile(`\.archcore/[A-Za-z0-9_./-]+\.md`)
	// numberedLineRe finds the numbered clauses of a spec's normative sections.
	// The number is followed by whitespace or by nothing: without that guard
	// "1.25 or newer is required" reads as clause 1 and gets graded as one.
	numberedLineRe = regexp.MustCompile(`^\s*[0-9]+\.(?:\s|$)`)
	// modalRe finds BCP 14 modals. Case-sensitive: a graded requirement is
	// uppercase by contract, and "must" in prose is not one.
	modalRe = regexp.MustCompile(`\b(MUST|SHOULD|SHALL|MAY)\b`)
	// notModalRe collapses the negative forms so "MUST NOT" counts once.
	notModalRe = regexp.MustCompile(`\b(MUST|SHOULD|SHALL|MAY) NOT\b`)
	// shallRe finds the notation a graded spec must not use.
	shallRe = regexp.MustCompile(`\bSHALL\b`)
	// earsOpenerRe finds a numbered clause opening with an EARS trigger. A prd
	// requirement states an outcome; the trigger/response form is spec notation.
	//
	// The emphasis run is optional because the corpus writes "1. **WHEN** ...".
	// Without it the opener went unrecognized, so a bolded EARS clause both
	// escaped the prd check and was reported for stating its condition late.
	earsOpenerRe = regexp.MustCompile(`^\s*[0-9]+\.\s*[*_]*(WHEN|WHILE|IF)\b`)
	// passiveRe finds an obligation with no obligated subject, in English
	// ("MUST be rotated") and Russian ("MUST ротироваться").
	//
	// The Russian alternative carries no closing \b: Go's is an ASCII word
	// boundary, and a Cyrillic letter is not an ASCII word character, so the
	// boundary never holds and the whole alternative could never match. The
	// shell original relied on grep's Unicode-aware \b.
	passiveRe = regexp.MustCompile(`\b(MUST|SHOULD)( NOT)? be [a-z]+(ed|en)\b|\b(MUST|SHOULD)( NOT)? \S*(ться|тся)`)
	// leadingNumberRe strips a clause's own numbering before the words are
	// counted, so "1." is not charged to the author's word budget.
	leadingNumberRe = regexp.MustCompile(`^\s*[0-9]+\.\s*`)
	// trailingCondRe finds a condition placed after the obligation it controls.
	// The modal stays case-sensitive (a graded requirement is uppercase by
	// contract) while the condition word is not.
	//
	// "once" is absent from the list on purpose: in requirement prose it is an
	// adverb ("MUST run the migration once") far more often than a conjunction,
	// and it is the one entry whose hits were all of that kind.
	//
	// The two Russian conditions are delimited by "not a letter" rather than by
	// \b: Go's boundary is ASCII-only, and an unguarded "пока" would match
	// inside "показать".
	trailingCondRe = regexp.MustCompile(
		`\b(MUST|SHOULD|MAY)( NOT)?\b.*?(?:(?i:\b(when|if|while|unless|whenever)\b)|(?:^|[^\p{L}])(если|когда)(?:[^\p{L}]|$))`)
	// inlineCodeRe spans a backticked identifier. A condition word inside one
	// names a thing rather than stating a trigger: "MUST set the `when` field"
	// carries no condition, and matching it reported the name as the defect.
	inlineCodeRe = regexp.MustCompile("`[^`]*`")
	// openListRe and ambiguousAltRe carry no word boundary on the right: "etc."
	// ends in punctuation, where a trailing \b would demand a word character
	// after the dot and never hold.
	openListRe     = buildMarkerRe(templates.OpenListMarkers)
	ambiguousAltRe = buildMarkerRe(templates.AmbiguousAlternatives)
	// scopeRe finds the anchor that makes a rule injectable: a backticked path
	// or glob, an @path reference, or the phrase naming the situation. A rule
	// with none of them matches no file, so the pre-write hook can never surface
	// it and the rule fires for nobody.
	//
	// The backtick alternative demands a "/" or a "." inside it. Any inline code
	// used to satisfy it, so a rule whose only backticks held `TODO` passed a
	// check whose message claims a file target was found. The one (?i) covers
	// both phrases after it: in RE2 a flag group runs to the end of the
	// enclosing group.
	scopeRe = regexp.MustCompile("`[^`]*[/.][^`]*`|@[A-Za-z0-9_./-]+|(?i)applies to|применяется к")
	// verifierRe finds the named check an Enforcement section owes its reader.
	// One (?i) covers every alternative after it: in RE2 a flag group runs to
	// the end of the enclosing group.
	verifierRe = regexp.MustCompile("`[^`]+`|@[A-Za-z0-9_./-]+|(?i)manual review|code review")
	// bulletRe finds a list item, which an adr's Context section must not use.
	bulletRe = regexp.MustCompile(`(?m)^\s*[-*+]\s`)
	// rejectionRe finds the active verb that says why an alternative lost. The
	// Russian entries are stems, left-guarded by "not a letter" for the same
	// reason the other Cyrillic patterns are: Go's \b is ASCII-only, and the
	// unguarded right side lets the inflected endings match.
	rejectionRe = regexp.MustCompile(`(?i)\b(rejected|ruled out|deferred|discarded|dropped|declined|superseded|not chosen)\b` +
		`|(?i:(?:^|[^\p{L}])(отклон|отверг|отброш|отложен|исключ|не выбран))`)
)

// buildMarkerRe compiles a case-insensitive alternation guarded on the left by
// "not a letter" and unguarded on the right. Both lists defeat Go's \b: an entry
// may end in punctuation ("etc."), where a trailing boundary would demand a word
// character after the dot, and the Russian entries sit outside ASCII \b
// entirely — so the left guard is spelled as a character class instead.
//
// Unguarded on the left, "and so on" matched inside "a brand so on the shelf".
// The marker itself is group 1, because the guard consumes a character that the
// finding must not echo.
func buildMarkerRe(markers []string) *regexp.Regexp {
	quoted := make([]string, len(markers))
	for i, m := range markers {
		quoted[i] = regexp.QuoteMeta(m)
	}
	return regexp.MustCompile(`(?i)(?:^|[^\p{L}])(` + strings.Join(quoted, "|") + `)`)
}

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
	// Kept out of PrecisionFindings: that function judges one document from its
	// own text, and this one needs the relation graph and the store.
	findings = append(findings, Restatement(baseDir, rel, body)...)
	if len(findings) == 0 {
		return ""
	}

	// The findings arrive in the fixed order PrecisionFindings emits — shared
	// checks before per-type ones — so the cut keeps the same head for the same
	// document and the report says how much it dropped.
	dropped := 0
	if len(findings) > maxReportFindings {
		dropped = len(findings) - maxReportFindings
		findings = findings[:maxReportFindings]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Precision] %s (advisory):\n", rel)
	for _, f := range findings {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	if dropped > 0 {
		fmt.Fprintf(&b, "  - +%d more finding(s) not shown (report cap %d)\n", dropped, maxReportFindings)
	}
	return strings.TrimRight(b.String(), "\n")
}

// PrecisionFindings runs every check. Order is fixed so the output reads the
// same way each time.
func PrecisionFindings(docType templates.DocumentType, fm templates.Frontmatter, body string) []string {
	var out []string
	lines := strings.Split(body, "\n")

	if hits := findVaguenessHits(lines); len(hits) > 0 {
		out = append(out, fmt.Sprintf("vague wording (%s) — replace with a concrete fact, version, threshold, or measurement",
			strings.Join(hits, ", ")))
	}

	for _, section := range templates.RequiredSections[docType] {
		if !hasSection(lines, section) {
			out = append(out, fmt.Sprintf("missing section: ## %s", section.Name))
		}
	}

	for _, foreign := range templates.ForeignSections[docType] {
		if hasSection(lines, foreign.Section) {
			out = append(out, fmt.Sprintf("section ## %s in a %s — a %s owns that content; link the two documents instead",
				foreign.Section.Name, docType, foreign.Owner))
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

	clauses := sectionItems(lines, templates.ClauseSections[docType])
	steps := sectionItems(lines, templates.StepSections[docType])
	out = append(out, profileFindings(docType, lines, clauses, steps)...)

	//exhaustive:ignore // Only these types own a contract past the shared and
	// profile-driven checks above; every other type is judged entirely by those.
	switch docType {
	case templates.TypeSpec:
		out = append(out, specFindings(body, clauses)...)
	case templates.TypePRD:
		out = append(out, prdFindings(lines)...)
	case templates.TypeRule:
		out = append(out, ruleFindings(lines, clauses)...)
	case templates.TypeADR:
		out = append(out, adrFindings(lines)...)
	case templates.TypeCPAT:
		out = append(out, cpatFindings(lines)...)
	}
	return out
}

// profileFindings are the checks the prose canon drives from its own tables:
// which headings hold graded requirements, which hold procedure steps, and which
// half of the writing profile binds the type. No branch here names a document
// type — adding a type to those tables is what gives it these checks.
func profileFindings(docType templates.DocumentType, lines, clauses, steps []string) []string {
	var out []string

	if hits := overLengthHits(clauses, templates.MaxClauseWords); len(hits) > 0 {
		out = append(out, fmt.Sprintf("requirement over %d words (%s) — split the clause, or move the qualifier into a requirement of its own",
			templates.MaxClauseWords, strings.Join(hits, "; ")))
	}
	if hits := overLengthHits(steps, templates.MaxStepWords); len(hits) > 0 {
		out = append(out, fmt.Sprintf("step over %d words (%s) — one step carries one action",
			templates.MaxStepWords, strings.Join(hits, "; ")))
	}

	for _, clause := range clauses {
		collapsed := notModalRe.ReplaceAllString(clause, "$1")
		if len(modalRe.FindAllString(collapsed, -1)) >= 2 {
			out = append(out, `compound requirement — a numbered clause carries two modals; split "MUST X and MUST NOT Y" into two clauses`)
			break
		}
	}

	if hits := conditionAfterObligation(clauses); len(hits) > 0 {
		out = append(out, fmt.Sprintf("condition after the obligation (%s) — open with WHEN, WHILE, or IF so the trigger is read before the response",
			strings.Join(hits, "; ")))
	}

	if hits := markerHits(openListRe, clauses, steps); len(hits) > 0 {
		out = append(out, fmt.Sprintf("open-ended list (%s) — state the last member, or name the property that decides membership",
			strings.Join(hits, ", ")))
	}
	if hits := markerHits(ambiguousAltRe, clauses, steps); len(hits) > 0 {
		out = append(out, fmt.Sprintf("ambiguous alternative (%s) — state which branch binds",
			strings.Join(hits, ", ")))
	}

	// Each modal check follows the table that owns the item, so neither has to
	// name a type. A step is a step because StepSections says so, whatever
	// profile the type carries — a plan records claims and still holds a task
	// list, and reading its tasks as claims produced the wrong advice.
	if hits := modalHits(steps); len(hits) > 0 {
		out = append(out, fmt.Sprintf("BCP 14 modal in a step (%s) — a step states an action to take, not an obligation that holds",
			strings.Join(hits, ", ")))
	}
	if templates.ProseProfiles[docType] == templates.ProfileISO {
		if hits := modalHits(claimItems(lines, steps)); len(hits) > 0 {
			out = append(out, fmt.Sprintf("BCP 14 modal (%s) in a numbered %s clause — this type records a claim, not a graded obligation; put the graded behavior in a linked spec or rule",
				strings.Join(hits, ", "), docType))
		}
	}

	return out
}

// claimItems returns the numbered items that are not procedure steps. A step
// already carries its own finding, and reporting one item twice under two names
// says nothing the first message did not.
func claimItems(lines, steps []string) []string {
	items := numberedItems(lines)
	if len(steps) == 0 {
		return items
	}
	return slices.DeleteFunc(items, func(item string) bool {
		return slices.Contains(steps, item)
	})
}

// prdFindings keeps a prd on its own side of the boundary: it states an
// outcome, and the graded behavior that satisfies the outcome belongs to a
// linked spec. The modal half of that boundary is now the ISO-profile check in
// profileFindings, which every claim-recording type shares; what stays here is
// the notation a prd alone must not borrow.
func prdFindings(lines []string) []string {
	var out []string

	for _, line := range lines {
		if earsOpenerRe.MatchString(line) {
			out = append(out, fmt.Sprintf("EARS clause in a prd requirement (%q) — a prd requirement states an outcome, not a trigger and a response; that form belongs in a spec",
				quote(strings.TrimSpace(line))))
			break
		}
	}

	return out
}

// ruleFindings check what makes a rule usable rather than merely true: an agent
// editing a file has to be able to tell whether the rule reaches that file, and
// a reviewer has to be able to tell who checks it.
func ruleFindings(lines, clauses []string) []string {
	var out []string

	// Document-level, not clause-level. The canon accepts "a path, a glob, or a
	// named situation", and a named situation is prose that no pattern finds —
	// asking each clause for a machine-readable anchor reported 35 of the
	// corpus's 39 rules, nearly all of which state their scope in a sentence.
	// What is decidable is whether the rule names a file target anywhere at all,
	// because a rule that names none cannot be matched to a changed path by the
	// code-alignment hook. Document-level, that is 6 of the 39.
	if len(clauses) > 0 && !slices.ContainsFunc(clauses, scopeRe.MatchString) {
		out = append(out, "no path or glob in the Rule section — push-mode injection matches a rule to an edited file, so a rule that names no file target never fires")
	}

	if body := sectionBody(lines, templates.SectionEnforcement); body != "" && !verifierRe.MatchString(body) {
		out = append(out, `Enforcement names no verifier — name the hook, lint rule, CI step, or test, or write "manual review"`)
	}

	return out
}

// adrFindings check the three places an adr stops being a decision record: a
// context with no evidence, a list of alternatives too short to have compared
// anything, and an alternative dropped without a stated reason.
func adrFindings(lines []string) []string {
	var out []string

	// Fenced blocks come out first: a "- item" inside a pasted YAML sample is
	// not the bullet list the adr contract forbids, and the contract is about
	// the author's own prose.
	context := outsideFences(sectionBody(lines, templates.SectionContext))
	if strings.TrimSpace(context) != "" && bulletRe.MatchString(context) {
		out = append(out, "bullets in Context — an adr states its trigger as 2 to 4 sentences; a bullet list drops the connective tissue that makes the decision follow")
	}

	alternatives := sectionItems(lines, []templates.SectionRule{templates.SectionAlternatives})
	if len(alternatives) == 1 {
		out = append(out, "one alternative recorded — a decision with nothing to compare against records a preference, not a choice")
	}
	var unexplained []string
	for _, alt := range alternatives {
		if !rejectionRe.MatchString(alt) {
			unexplained = append(unexplained, quote(alt))
		}
	}
	if hits := distinctCapped(unexplained, maxPrecisionHits); len(hits) > 0 {
		out = append(out, fmt.Sprintf("alternative with no stated reason (%s) — say what ruled it out; the reason is the part a later reader cannot reconstruct",
			strings.Join(hits, "; ")))
	}

	return out
}

// cpatFindings check the one thing a cpat exists to carry. The exact text of the
// old form and the new form is the artifact, so a cpat that describes them in
// prose has kept the summary and dropped the pattern.
func cpatFindings(lines []string) []string {
	var out []string
	for _, rule := range []templates.SectionRule{templates.SectionBefore, templates.SectionAfter} {
		if body := sectionBody(lines, rule); body != "" && !strings.Contains(body, "```") {
			out = append(out, fmt.Sprintf("## %s holds no code block — a cpat shows the form that changed, it does not describe it", rule.Name))
		}
	}
	return out
}

// distinctCapped keeps the first n distinct entries in sorted order, so the
// reported set does not depend on where the offenders appear.
//
// Sorting before the cap, not after: capping in encounter order makes the
// reported set a function of where the offenders sit, so reformatting a
// document silently changes which ones it names.
func distinctCapped(in []string, n int) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	out = slices.Compact(out)
	return out[:min(len(out), n)]
}

// specFindings are the checks that apply only to a normative contract. The
// compound-requirement check moved to profileFindings, where every type with
// graded clauses gets it.
func specFindings(body string, clauses []string) []string {
	var out []string

	if shallRe.MatchString(body) {
		out = append(out, "spec notation: SHALL found — grade with BCP 14 modals instead (MUST / SHOULD / MAY)")
	}

	if n := strings.Count(strings.TrimSpace(body), "\n") + 1; n > templates.MaxSpecBodyLines {
		out = append(out, fmt.Sprintf("spec body is %d lines (cap %d) — a spec that long is describing, not specifying",
			n, templates.MaxSpecBodyLines))
	}

	if passives := findPassiveHits(clauses); len(passives) > 0 {
		out = append(out, fmt.Sprintf("subjectless passive (%s) — name the obligated component as the subject, or use WHEN <trigger> for an event",
			strings.Join(passives, ", ")))
	}

	return out
}

// findPassiveHits collects distinct subjectless passives from graded clauses,
// capped at maxPassiveHits.
func findPassiveHits(clauses []string) []string {
	var out []string
	for _, line := range clauses {
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

// findVaguenessHits collects the offenders from all three lexicons, lowercased
// so the matches collapse with the canon's own spelling.
//
// Headings are excluded. The lexicon exists to catch a claim that hides a fact,
// and a heading is a label, not a claim: the doc template owes its reader a
// "## Best Practices" section, which is not the defect "we follow best
// practices" is.
func findVaguenessHits(lines []string) []string {
	var prose strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		prose.WriteString(line)
		prose.WriteByte('\n')
	}
	body := prose.String()

	var hits []string
	for _, m := range enVaguenessRe.FindAllString(body, -1) {
		hits = append(hits, strings.ToLower(m))
	}
	for _, m := range phraseVaguenessRe.FindAllString(body, -1) {
		hits = append(hits, strings.ToLower(m))
	}
	// Russian stays a substring match: the words inflect, and "неоптимальный"
	// carries "оптимальн" and exactly as little falsifiable content.
	lower := strings.ToLower(body)
	for _, stem := range templates.VaguenessLexiconRU {
		if strings.Contains(lower, stem) {
			hits = append(hits, stem)
		}
	}
	return distinctCapped(hits, maxPrecisionHits)
}

// findCrossDocHits collects the document references the body links directly.
func findCrossDocHits(body string) []string {
	return distinctCapped(crossDocRefRe.FindAllString(body, -1), maxCrossDocHits)
}

// hasSection reports whether the body carries a level-2 heading matching the
// rule, under its canonical name or any historical alias.
func hasSection(lines []string, rule templates.SectionRule) bool {
	for _, line := range lines {
		heading, ok := headingText(line)
		if ok && ruleMatches(heading, rule) {
			return true
		}
	}
	return false
}

// headingText returns the text of a level-2 heading. A deeper heading ("### x")
// is not one: its "#" sits where the required space would be, so a subsection
// never closes the section that contains it.
func headingText(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "##")
	if !ok || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return "", false
	}
	return strings.TrimLeft(rest, " \t"), true
}

// ruleMatches reports whether a heading satisfies a section rule under its
// canonical name or any historical alias.
func ruleMatches(heading string, rule templates.SectionRule) bool {
	if sectionNameMatches(heading, rule.Name) {
		return true
	}
	for _, alias := range rule.Aliases {
		if sectionNameMatches(heading, alias) {
			return true
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

// collectItems returns the body's numbered items with their wrapped
// continuation lines joined, so a clause that spans two source lines is measured
// once and its second half is not read as prose.
//
// When match is non-nil only the sections whose heading satisfies it contribute.
// A fenced block never contributes: the Good and Bad examples a rule owes its
// reader are not that rule's own requirements, and counting them reported every
// well-written rule as a violation of itself.
//
// An unpaired run of fences is read as no fence at all. The alternative is to
// swallow every item after the stray marker, and this check is advisory and
// deliberately over-eager: it reports too much rather than going quiet.
func collectItems(lines []string, match func(heading string) bool) []string {
	var out []string
	var parts []string
	inScope := match == nil
	inFence := false
	paired := fenceCount(lines)%2 == 0

	flush := func() {
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, " "))
			parts = parts[:0]
		}
	}

	for _, line := range lines {
		if paired && isFence(line) {
			flush()
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if heading, ok := headingText(line); ok {
			flush()
			if match != nil {
				inScope = match(heading)
			}
			continue
		}
		// A deeper heading does not close the section it sits in, but it is not
		// part of the item above it either: folded in, "### Error handling" and
		// the prose under it were charged to the last clause's word budget.
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			flush()
			continue
		}
		if !inScope {
			continue
		}
		switch {
		case numberedLineRe.MatchString(line):
			flush()
			parts = append(parts, strings.TrimSpace(line))
		case strings.TrimSpace(line) == "":
			flush()
		case len(parts) > 0:
			parts = append(parts, strings.TrimSpace(line))
		}
	}
	flush()
	return out
}

// isFence reports whether the line opens or closes a fenced block.
func isFence(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t\r\f\v"), "```")
}

// fenceCount counts the fence markers, so an unpaired run can be recognized
// before it decides what the parser sees.
func fenceCount(lines []string) int {
	n := 0
	for _, line := range lines {
		if isFence(line) {
			n++
		}
	}
	return n
}

// outsideFences returns the text with its fenced blocks removed. A pasted
// sample is not the section's own prose, and an unpaired run leaves the text
// alone for the same reason collectItems does.
func outsideFences(body string) string {
	lines := strings.Split(body, "\n")
	if fenceCount(lines)%2 != 0 {
		return body
	}
	var b strings.Builder
	inFence := false
	for _, line := range lines {
		if isFence(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// sectionItems returns the numbered items inside the sections named by rules.
func sectionItems(lines []string, rules []templates.SectionRule) []string {
	if len(rules) == 0 {
		return nil
	}
	return collectItems(lines, func(heading string) bool {
		for _, rule := range rules {
			if ruleMatches(heading, rule) {
				return true
			}
		}
		return false
	})
}

// numberedItems returns every numbered item of the body, wherever it sits.
func numberedItems(lines []string) []string {
	return collectItems(lines, nil)
}

// sectionBody returns the text under the first heading matching the rule, up to
// the next level-2 heading. An absent section returns the empty string, which
// callers read as "nothing to judge" — a missing section is already reported by
// the RequiredSections check, and reporting it twice says nothing new.
func sectionBody(lines []string, rule templates.SectionRule) string {
	var b strings.Builder
	inSection := false
	for _, line := range lines {
		if heading, ok := headingText(line); ok {
			if inSection {
				break
			}
			inSection = ruleMatches(heading, rule)
			continue
		}
		if inSection {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

// wordCount counts an item's words without its own numbering: "1." is the
// document's structure, not the author's word budget.
func wordCount(item string) int {
	return len(strings.Fields(leadingNumberRe.ReplaceAllString(item, "")))
}

// overLengthHits quotes each item past the cap, sorted and capped so the
// reported set does not depend on where in the document the offenders sit.
func overLengthHits(items []string, limit int) []string {
	var hits []string
	for _, item := range items {
		if wordCount(item) > limit {
			hits = append(hits, quote(item))
		}
	}
	return distinctCapped(hits, maxPrecisionHits)
}

// conditionAfterObligation quotes the clauses that state their trigger after the
// response it triggers. A clause already opening with an EARS keyword is exempt:
// its condition is where it belongs, and a second condition later in the same
// clause is a qualifier rather than the trigger.
func conditionAfterObligation(clauses []string) []string {
	var hits []string
	for _, clause := range clauses {
		if earsOpenerRe.MatchString(clause) {
			continue
		}
		if trailingCondRe.MatchString(inlineCodeRe.ReplaceAllString(clause, " ")) {
			hits = append(hits, quote(clause))
		}
	}
	return distinctCapped(hits, maxPrecisionHits)
}

// markerHits collects the distinct marker occurrences across every item group.
// Group 1 is the marker: buildMarkerRe's left guard consumes the character
// before it, which the finding must not echo back.
//
// Inline code comes out first, for the same reason conditionAfterObligation
// strips it: a backticked marker is named rather than used. The clause that
// forbids "such as `etc.`" is not itself an open-ended list.
func markerHits(re *regexp.Regexp, groups ...[]string) []string {
	var hits []string
	for _, group := range groups {
		for _, item := range group {
			stripped := inlineCodeRe.ReplaceAllString(item, " ")
			for _, m := range re.FindAllStringSubmatch(stripped, -1) {
				hits = append(hits, strings.ToLower(m[1]))
			}
		}
	}
	return distinctCapped(hits, maxPrecisionHits)
}

// modalHits collects the distinct BCP 14 keywords the items carry.
func modalHits(items []string) []string {
	var hits []string
	for _, item := range items {
		hits = append(hits, modalRe.FindAllString(item, -1)...)
	}
	return distinctCapped(hits, maxPrecisionHits)
}

// hasLongCodeBlock reports whether any fenced block holds MaxCodeBlockLines or
// more content lines. The fences themselves are not counted, and an unclosed
// block never triggers.
func hasLongCodeBlock(lines []string) bool {
	inBlock, count := false, 0
	for _, line := range lines {
		if isFence(line) {
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
