package advisory

import (
	"cmp"
	"context"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"archcore-cli/internal/docs"
	"archcore-cli/internal/git"
	"archcore-cli/templates"
)

// The corpus check is the mechanical half of a review (corpus-check.spec): every
// claim a document makes that a lookup can settle, settled once, so the model
// spends its turns on the claims only judgment can settle.

// CheckKind names one class of corpus-check finding. It reaches the CLI output.
type CheckKind string

// Corpus-check finding kinds (corpus-check.spec, Surface).
const (
	CheckUnresolvedPath    CheckKind = "unresolved_path"
	CheckLineAnchor        CheckKind = "line_anchor"
	CheckMissingIdentifier CheckKind = "missing_identifier"
	CheckMissingString     CheckKind = "missing_string"
	CheckDraftCitedByCode  CheckKind = "draft_cited_by_code"
	CheckDanglingCodeCite  CheckKind = "dangling_code_citation"
	CheckDuplicateText     CheckKind = "duplicate_text"
	CheckRestatedCode      CheckKind = "restated_code"
)

// MaxFindingsPerKind bounds the findings of one kind in the output, so a corpus
// with one systemic defect does not crowd the other kinds out of the reviewer's
// context; Omitted reports the rest.
const MaxFindingsPerKind = 40

// maxLineRunes bounds the cited line or sample a finding carries: one sentence
// of context per finding in the reviewer's context, not a paragraph.
const maxLineRunes = 200

// Thresholds that separate a claim from noise; none bounds output.
const (
	minDuplicateShingles   = 3
	shingleWords           = 8
	boilerplateShingleDocs = 4
	minRestatedBlockLines  = 3
	minRestatedLineLength  = 16
	minIdentifierLength    = 5
)

// CheckFinding is one settled claim. Fields a kind does not use stay empty.
type CheckFinding struct {
	Kind CheckKind `json:"kind"`
	Doc  string    `json:"doc,omitempty"`
	// Other is the second document of a duplicate_text pair.
	Other string `json:"other,omitempty"`
	// Subject is the path, identifier, string, or citation the finding is about.
	Subject string `json:"subject,omitempty"`
	// Evidence is a commit, a moved-to path, a code file, or a sample of text.
	Evidence string `json:"evidence,omitempty"`
	Count    int    `json:"count,omitempty"`
	// Historical marks a finding on a line that itself says the thing is gone.
	Historical bool `json:"historical,omitempty"`
	// Status is the status of Doc; an accepted document binds, so it leads.
	Status string `json:"status,omitempty"`
	// Normative marks a subject cited on a line with a BCP 14 keyword.
	Normative bool `json:"normative,omitempty"`
	// Line is the cited line, trimmed and cut to maxLineRunes, so the reviewer
	// decides the claim without opening the document.
	Line string `json:"line,omitempty"`
}

// CorpusCheck is the bounded result: findings per kind, capped, with totals.
type CorpusCheck struct {
	Documents int                  `json:"documents"`
	Totals    map[CheckKind]int    `json:"totals"`
	Findings  []CheckFinding       `json:"findings"`
	Omitted   map[CheckKind]int    `json:"omitted,omitempty"`
	Skipped   map[CheckKind]string `json:"skipped,omitempty"`
}

var (
	// A path segment may hold parentheses: a route group such as app/(shop)/page.tsx.
	globTokenRe  = regexp.MustCompile("`([A-Za-z0-9_.][A-Za-z0-9_./()*-]*\\*[A-Za-z0-9_./()*-]*)`|@([A-Za-z0-9_.][A-Za-z0-9_./()*-]*\\*[A-Za-z0-9_./()*-]*)")
	refTokenRe   = regexp.MustCompile("@[A-Za-z0-9_.][A-Za-z0-9_./()-]*|`[A-Za-z0-9_.][A-Za-z0-9_./()-]*`")
	backtickRe   = regexp.MustCompile("`([^`\n]+)`")
	identifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\(\))?$`)
	guillemetRe  = regexp.MustCompile(`«([^«»\n]{2,60})»`)
	// codeCiteRe needs a hyphen in the slug: "data.plan" in code is a field access.
	codeCiteRe    = `[a-z0-9]+(?:-[a-z0-9]+)+\.(` + strings.Join(docTypeNames(), "|") + `)\b`
	docSuffixRe   = regexp.MustCompile(`\.(` + strings.Join(docTypeNames(), "|") + `)(\.md)?$`)
	normativeRe   = regexp.MustCompile(`\b(MUST|SHALL|SHOULD|MAY)\b`)
	prohibitionRe = regexp.MustCompile(`\b(MUST|SHALL|SHOULD) NOT\b`)
	// placeholderRe matches a template name such as XxxError, not a real one.
	placeholderRe  = regexp.MustCompile(`(?i)xxx`)
	historicalRe   = regexp.MustCompile(`(?i)removed|deleted|no longer|replaced|renamed|was |удал|снят|больше нет|заменён|заменен|переименован|раньше|бывш`)
	wordRe         = regexp.MustCompile(`[\p{L}\p{N}_]+`)
	packageMarkers = []string{"package.json", "go.mod", "Cargo.toml", "pyproject.toml"}
)

func docTypeNames() []string {
	names := templates.ValidTypes()
	slices.SortFunc(names, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return names
}

// claimTypes are the types whose statements claim present state; an idea or a
// plan names things that do not exist yet by design.
var claimTypes = map[templates.DocumentType]bool{
	templates.TypeRule: true, templates.TypeCPAT: true, templates.TypeADR: true, templates.TypeSpec: true,
	templates.TypeDoc: true, templates.TypeGuide: true, templates.TypeScenario: true,
}

// CheckCorpusFor runs every corpus check over the local documents under baseDir
// that scope admits; an empty scope admits every document.
func CheckCorpusFor(ctx context.Context, baseDir string, scope []string) (CorpusCheck, error) {
	corpus, err := docs.ScanLocal(baseDir, true)
	if err != nil {
		return CorpusCheck{}, err
	}
	all := make(map[string]docs.Document)
	status := make(map[string]string)
	var live []docs.Document
	for _, d := range corpus {
		all[d.Slug+"."+string(d.Type)] = d
		status[d.Path] = string(d.Status)
		if d.Status != templates.StatusRejected && inCheckScope(d.Path, scope) {
			d.Content = stripFrontmatter(d.Content)
			live = append(live, d)
		}
	}
	slices.SortFunc(live, func(a, b docs.Document) int { return cmp.Compare(a.Path, b.Path) })

	c := &checkRun{ctx: ctx, base: baseDir, status: status, out: CorpusCheck{Documents: len(live), Totals: map[CheckKind]int{}}}
	// Advisory read: without the tracked-file list, each kind that needs it is
	// reported as skipped with the reason, and the other kinds still run.
	tracked, err := git.TrackedFiles(ctx, baseDir)
	c.references(live, tracked)
	c.duplicates(live)
	if err != nil {
		for _, k := range []CheckKind{CheckUnresolvedPath, CheckMissingIdentifier, CheckMissingString, CheckDraftCitedByCode, CheckDanglingCodeCite, CheckRestatedCode} {
			c.skip(k, err.Error())
		}
		return c.finish(), nil
	}
	ids, texts, blocks := collectIdentifiers(live), collectStrings(live), collectBlocks(live)
	var substrings []string
	for t := range texts {
		substrings = append(substrings, t)
	}
	for _, b := range blocks {
		substrings = append(substrings, b.lines...)
	}
	scan := scanCode(baseDir, tracked, ids, substrings)
	c.reportMissing(CheckMissingIdentifier, ids, scan.identifiers)
	c.reportMissing(CheckMissingString, texts, scan.substrings)
	c.codeCitations(all, scope, scan.citations, scan.looseCitations)
	c.restatedCode(blocks, scan.substrings)
	return c.finish(), nil
}

type checkRun struct {
	ctx    context.Context
	base   string
	status map[string]string
	out    CorpusCheck
}

func (c *checkRun) add(f CheckFinding) {
	f.Status = c.status[f.Doc]
	c.out.Totals[f.Kind]++
	c.out.Findings = append(c.out.Findings, f)
}

func (c *checkRun) skip(kind CheckKind, why string) {
	if c.out.Skipped == nil {
		c.out.Skipped = map[CheckKind]string{}
	}
	c.out.Skipped[kind] = why
}

// priority orders findings for a reader who stops early: a live claim before
// history, an accepted document before a draft, a normative line before prose.
func priority(f CheckFinding) int {
	p := 0
	if f.Historical {
		p += 4
	}
	if f.Status != string(templates.StatusAccepted) {
		p += 2
	}
	if !f.Normative {
		p++
	}
	return p
}

// finish orders the findings by priority, keeping the order of each check
// within one priority, and caps each kind.
func (c *checkRun) finish() CorpusCheck {
	slices.SortStableFunc(c.out.Findings, func(a, b CheckFinding) int { return cmp.Compare(priority(a), priority(b)) })
	kept := make(map[CheckKind]int)
	var out []CheckFinding
	for _, f := range c.out.Findings {
		if kept[f.Kind] < MaxFindingsPerKind {
			kept[f.Kind]++
			out = append(out, f)
		}
	}
	for kind, total := range c.out.Totals {
		if total > kept[kind] {
			if c.out.Omitted == nil {
				c.out.Omitted = map[CheckKind]int{}
			}
			c.out.Omitted[kind] = total - kept[kind]
		}
	}
	c.out.Findings = out
	if c.out.Findings == nil {
		c.out.Findings = []CheckFinding{}
	}
	return c.out
}

func inCheckScope(docPath string, scope []string) bool {
	if len(scope) == 0 {
		return true
	}
	for _, s := range scope {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "./"), ".archcore/")
		if strings.HasPrefix(strings.TrimPrefix(docPath, ".archcore/"), s) {
			return true
		}
	}
	return false
}

func stripFrontmatter(content string) string {
	if rest, ok := strings.CutPrefix(content, "---\n"); ok {
		if _, body, found := strings.Cut(rest, "\n---"); found {
			return body
		}
	}
	return content
}

// references reports cited paths that resolve under no project or package root,
// and documents that cite line anchors (precision Rule 9). A path whose first
// segment exists under no root is a fragment of the prose around it, not a
// project path, and is skipped.
func (c *checkRun) references(live []docs.Document, tracked []string) {
	roots := []string{"."}
	for _, f := range tracked {
		if slices.Contains(packageMarkers, path.Base(f)) && strings.Contains(f, "/") {
			roots = append(roots, path.Dir(f))
		}
	}
	slices.Sort(roots)
	roots = slices.Compact(roots)
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(c.base, filepath.FromSlash(rel)))
		return err == nil
	}
	for _, d := range live {
		if anchors := lineAnchorRe.FindAllString(d.Content, -1); len(anchors) > 0 {
			c.add(CheckFinding{Kind: CheckLineAnchor, Doc: d.Path, Count: len(anchors), Subject: anchors[0]})
		}
		if tracked == nil {
			continue
		}
		c.unmatchedGlobs(d, roots, tracked, exists)
		seen := map[string]bool{}
		for _, tok := range refTokenRe.FindAllString(d.Content, -1) {
			p, ok := citedPath(tok)
			if !ok || seen[p] {
				continue
			}
			seen[p] = true
			first, _, _ := strings.Cut(p, "/")
			anchored, found := false, false
			for _, r := range roots {
				anchored = anchored || exists(path.Join(r, first))
				if exists(path.Join(r, p)) {
					found = true
					break
				}
			}
			if found || !anchored {
				continue
			}
			line := lineNaming(d.Content, p)
			f := CheckFinding{Kind: CheckUnresolvedPath, Doc: d.Path, Subject: p, Evidence: "never-tracked", Normative: normativeMention(d.Content, p), Historical: historicalRe.MatchString(line), Line: clip(line)}
			if sha, err := git.LastCommitTouching(c.ctx, c.base, p); err == nil && sha != "" {
				f.Evidence = sha[:min(len(sha), 10)]
				if moved := git.RenamedTo(c.ctx, c.base, sha, p); moved != "" {
					f.Evidence += " moved to " + moved
				}
			}
			c.add(f)
		}
	}
}

// unmatchedGlobs reports cited slash globs that match no tracked file under any
// root, such as `locales/*.json` over a locales/<lang>/ layout. A glob whose
// first segment exists under no root is prose, as in references.
func (c *checkRun) unmatchedGlobs(d docs.Document, roots, tracked []string, exists func(string) bool) {
	seen := map[string]bool{}
	for _, m := range globTokenRe.FindAllStringSubmatch(d.Content, -1) {
		g := strings.TrimPrefix(m[1]+m[2], "./")
		first, _, _ := strings.Cut(g, "/")
		if seen[g] || !strings.Contains(g, "/") || strings.Contains(first, "*") || strings.HasPrefix(g, ".archcore/") {
			continue
		}
		seen[g] = true
		var res []*regexp.Regexp
		for _, r := range roots {
			if exists(path.Join(r, first)) {
				// A glob may name directories ("packages/*"); git tracks only files.
				res = append(res, globRegexp(path.Join(r, g)), globRegexp(path.Join(r, g)+"/**"))
			}
		}
		if len(res) == 0 || slices.Contains(res, nil) {
			continue
		}
		if !slices.ContainsFunc(tracked, func(f string) bool {
			return slices.ContainsFunc(res, func(re *regexp.Regexp) bool { return re.MatchString(f) })
		}) {
			c.add(CheckFinding{Kind: CheckUnresolvedPath, Doc: d.Path, Subject: g, Evidence: "glob-matches-nothing", Normative: normativeMention(d.Content, g), Line: clip(lineNaming(d.Content, g))})
		}
	}
}

// normativeMention reports whether a line of content that names subject carries
// a BCP 14 keyword.
func normativeMention(content, subject string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		if strings.Contains(line, subject) && normativeRe.MatchString(line) {
			return true
		}
	}
	return false
}

// lineNaming returns the first line of content that names subject.
func lineNaming(content, subject string) string {
	for line := range strings.SplitSeq(content, "\n") {
		if strings.Contains(line, subject) {
			return line
		}
	}
	return ""
}

// clip trims a line and cuts it to maxLineRunes.
func clip(line string) string {
	line = strings.TrimSpace(line)
	if r := []rune(line); len(r) > maxLineRunes {
		return string(r[:maxLineRunes]) + "…"
	}
	return line
}

// citedPath returns the project path a reference token cites, or false for a
// token that is not one: elided, globbed, templated, absolute, a URL,
// document-relative, an Archcore document, or a name with no directory.
func citedPath(tok string) (string, bool) {
	p := strings.Trim(strings.TrimPrefix(tok, "@"), "`")
	if i := strings.IndexAny(p, ":#"); i >= 0 {
		p = p[:i]
	}
	p = strings.TrimRight(p, ".,;:)")
	switch {
	case !strings.Contains(p, "/"), strings.HasPrefix(p, "/"), strings.Contains(p, "..."),
		strings.ContainsAny(p, "*<{$"), strings.HasPrefix(p, "http"),
		strings.HasPrefix(p, "./"), strings.HasPrefix(p, "../"), strings.HasPrefix(p, ".archcore/"):
		return "", false
	}
	if docSuffixRe.MatchString(p) {
		return "", false
	}
	if !strings.HasSuffix(p, "/") && !strings.Contains(path.Base(p), ".") {
		return "", false
	}
	return p, true
}

// codeLike reports whether a backticked token names a code identifier rather
// than a word: two case humps, an underscore, or a call.
func codeLike(tok string) bool {
	if len(tok) < minIdentifierLength || !identifierRe.MatchString(tok) {
		return false
	}
	bare := strings.TrimSuffix(tok, "()")
	if strings.HasSuffix(tok, "()") || strings.Contains(bare, "_") {
		return true
	}
	humps := 0
	for i := 1; i < len(bare); i++ {
		if bare[i] >= 'A' && bare[i] <= 'Z' && bare[i-1] >= 'a' && bare[i-1] <= 'z' {
			humps++
		}
	}
	return humps >= 1
}

type citedToken struct {
	doc, line string
}

func (t citedToken) normative() bool { return normativeRe.MatchString(t.line) }

// collectIdentifiers returns the code identifiers that claim-type documents cite.
func collectIdentifiers(live []docs.Document) map[string][]citedToken {
	cites := map[string][]citedToken{}
	for _, d := range live {
		if !claimTypes[d.Type] {
			continue
		}
		for line := range strings.SplitSeq(d.Content, "\n") {
			for _, m := range backtickRe.FindAllStringSubmatch(line, -1) {
				if codeLike(m[1]) && !placeholderRe.MatchString(m[1]) {
					id := strings.TrimSuffix(m[1], "()")
					cites[id] = append(cites[id], citedToken{doc: d.Path, line: line})
				}
			}
		}
	}
	return cites
}

// collectStrings returns the «quoted» interface text that specs, scenarios, and
// rules state.
func collectStrings(live []docs.Document) map[string][]citedToken {
	cites := map[string][]citedToken{}
	for _, d := range live {
		// A rule quotes phrasing as often as interface text; a spec or a scenario
		// quotes what the screen shows.
		if d.Type != templates.TypeSpec && d.Type != templates.TypeScenario {
			continue
		}
		for line := range strings.SplitSeq(d.Content, "\n") {
			for _, m := range guillemetRe.FindAllStringSubmatch(line, -1) {
				cites[m[1]] = append(cites[m[1]], citedToken{doc: d.Path, line: line})
			}
		}
	}
	return cites
}

func (c *checkRun) reportMissing(kind CheckKind, cites map[string][]citedToken, found map[string]bool) {
	type row struct {
		doc, subject, line    string
		historical, normative bool
	}
	var rows []row
	for subject, at := range cites {
		if found[subject] {
			continue
		}
		perDoc := map[string]int{}
		forbids := map[string]bool{}
		for _, t := range at {
			// A document that forbids an identifier anywhere does not claim it exists;
			// a forbidden action on a named screen still claims the screen.
			forbids[t.doc] = forbids[t.doc] || kind == CheckMissingIdentifier && prohibitionRe.MatchString(t.line)
		}
		for _, t := range at {
			if forbids[t.doc] {
				continue
			}
			if i, ok := perDoc[t.doc]; ok {
				// One live or normative line in the document is enough to make it a claim.
				if rows[i].historical && !historicalRe.MatchString(t.line) {
					rows[i].historical, rows[i].line = false, t.line
				}
				rows[i].normative = rows[i].normative || t.normative()
				continue
			}
			perDoc[t.doc] = len(rows)
			rows = append(rows, row{doc: t.doc, subject: subject, line: t.line, historical: historicalRe.MatchString(t.line), normative: t.normative()})
		}
	}
	// Live claims first: a historical line is history to cut, not drift to fix.
	slices.SortFunc(rows, func(a, b row) int {
		if a.historical != b.historical {
			if a.historical {
				return 1
			}
			return -1
		}
		return cmp.Or(cmp.Compare(a.doc, b.doc), cmp.Compare(a.subject, b.subject))
	})
	for _, r := range rows {
		c.add(CheckFinding{Kind: kind, Doc: r.doc, Subject: r.subject, Historical: r.historical, Normative: r.normative, Line: clip(r.line)})
	}
}

// codeCitations reports draft documents that code cites as its contract, and
// citations of documents that do not exist (cite-the-governing-document-from-code.rule).
// The loose "<slug> <type>" form counts only for a document that exists.
func (c *checkRun) codeCitations(all map[string]docs.Document, scope []string, citedBy, loose map[string][]string) {
	for cite, files := range loose {
		if _, ok := all[cite]; ok {
			citedBy[cite] = append(citedBy[cite], files...)
		}
	}
	for _, cite := range slices.Sorted(maps.Keys(citedBy)) {
		files := slices.Compact(slices.Sorted(slices.Values(citedBy[cite])))
		d, ok := all[cite]
		switch {
		case !ok:
			c.add(CheckFinding{Kind: CheckDanglingCodeCite, Subject: cite, Evidence: files[0], Count: len(files)})
		case d.Status == templates.StatusDraft && inCheckScope(d.Path, scope):
			c.add(CheckFinding{Kind: CheckDraftCitedByCode, Doc: d.Path, Evidence: files[0], Count: len(files)})
		}
	}
}

// duplicates reports document pairs that share runs of eight words. A run that
// four or more documents share is template boilerplate and does not count.
func (c *checkRun) duplicates(live []docs.Document) {
	owners := map[string][]int{}
	for i, d := range live {
		words := wordRe.FindAllString(strings.ToLower(d.Content), -1)
		seen := map[string]bool{}
		for j := 0; j+shingleWords <= len(words); j++ {
			s := strings.Join(words[j:j+shingleWords], " ")
			if !seen[s] {
				seen[s] = true
				owners[s] = append(owners[s], i)
			}
		}
	}
	type pair struct{ a, b int }
	shared := map[pair]int{}
	sample := map[pair]string{}
	for s, docsWith := range owners {
		if len(docsWith) < 2 || len(docsWith) >= boilerplateShingleDocs {
			continue
		}
		for x := 0; x < len(docsWith); x++ {
			for y := x + 1; y < len(docsWith); y++ {
				p := pair{docsWith[x], docsWith[y]}
				shared[p]++
				if sample[p] == "" || s < sample[p] {
					sample[p] = s
				}
			}
		}
	}
	pairs := slices.Collect(maps.Keys(shared))
	slices.SortFunc(pairs, func(p, q pair) int {
		return cmp.Or(cmp.Compare(shared[q], shared[p]), cmp.Compare(p.a, q.a), cmp.Compare(p.b, q.b))
	})
	for _, p := range pairs {
		if shared[p] >= minDuplicateShingles {
			c.add(CheckFinding{Kind: CheckDuplicateText, Doc: live[p.a].Path, Other: live[p.b].Path, Count: shared[p], Evidence: sample[p]})
		}
	}
}

type codeBlock struct {
	doc string
	// lines are the lines long enough to compare; total counts every non-empty line.
	lines []string
	total int
}

// collectBlocks returns the fenced code blocks of the documents, each line trimmed.
func collectBlocks(live []docs.Document) []codeBlock {
	var blocks []codeBlock
	for _, d := range live {
		inFence, total := false, 0
		var cur []string
		for line := range strings.SplitSeq(d.Content, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "```") {
				if inFence && len(cur) >= minRestatedBlockLines {
					blocks = append(blocks, codeBlock{doc: d.Path, lines: cur, total: total})
				}
				inFence, cur, total = !inFence, nil, 0
				continue
			}
			if inFence && t != "" {
				total++
			}
			if inFence && len(t) >= minRestatedLineLength {
				cur = append(cur, t)
			}
		}
	}
	return blocks
}

// restatedCode reports fenced code blocks whose lines a tracked source file
// already holds: the document copies what the code itself states.
func (c *checkRun) restatedCode(blocks []codeBlock, found map[string]bool) {
	for _, b := range blocks {
		n := 0
		for _, l := range b.lines {
			if found[l] {
				n++
			}
		}
		// Most of the block, not a stray line, must already be in the code.
		if n >= minRestatedBlockLines && n*2 >= b.total {
			c.add(CheckFinding{Kind: CheckRestatedCode, Doc: b.doc, Count: n, Evidence: clip(b.lines[0])})
		}
	}
}
