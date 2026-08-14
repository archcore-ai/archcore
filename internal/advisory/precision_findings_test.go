package advisory

import (
	"strings"
	"testing"
	"unicode/utf8"

	"archcore-cli/templates"
)

func TestPrecisionFindings(t *testing.T) {
	t.Parallel()
	longBody := strings.Repeat("This body is long enough to clear the placeholder floor. ", 6)

	tests := []struct {
		name    string
		docType templates.DocumentType
		fm      templates.Frontmatter
		body    string
		wantHit string
		// wantNoHit names a finding this case exists to prove absent. Without
		// it a case can only assert that the whole document is clean, which
		// says nothing about the one check being exercised.
		wantNoHit string
	}{
		{
			name: "clean ADR produces no findings", docType: templates.TypeADR,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
		},
		{
			name: "forbidden lexicon hit", docType: templates.TypeADR,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Context\nWe picked a robust and scalable approach. " + longBody + "\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantHit: "vague wording",
		},
		{
			name: "russian lexicon hit", docType: templates.TypeADR,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Context\nВыбран оптимальный вариант. " + longBody + "\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantHit: "vague wording",
		},
		{
			name: "ADR missing a mandatory section", docType: templates.TypeADR,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Context\n" + longBody + "\n## Decision\nd\n## Consequences\nc\n",
			wantHit: "missing section: ## Alternatives Considered",
		},
		{
			name: "spec legacy Contract Surface heading is accepted", docType: templates.TypeSpec,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose\n" + longBody + "\n## Contract Surface\ns\n## Normative Behavior\n1. The server MUST answer.\n## Conformance\nc\n",
		},
		{
			name: "spec containing SHALL", docType: templates.TypeSpec,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. The server SHALL answer.\n## Conformance\nc\n",
			wantHit: "SHALL found",
		},
		{
			name: "lowercase shall does not trigger", docType: templates.TypeSpec,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose\nWe shall see. " + longBody + "\n## Surface\ns\n## Normative Behavior\n1. The server MUST answer.\n## Conformance\nc\n",
		},
		{
			name: "spec compound requirement", docType: templates.TypeSpec,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. The server MUST answer and MUST log it.\n## Conformance\nc\n",
			wantHit: "compound requirement",
		},
		{
			name: "single MUST NOT clause is not compound", docType: templates.TypeSpec,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. The server MUST NOT answer twice.\n## Conformance\nc\n",
		},
		{
			name: "spec subjectless passive in English", docType: templates.TypeSpec,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. Credentials MUST be rotated monthly.\n## Conformance\nc\n",
			wantHit: "subjectless passive",
		},
		{
			name: "active voice does not trigger the passive finding", docType: templates.TypeSpec,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. The scheduler MUST rotate credentials monthly.\n## Conformance\nc\n",
		},
		{
			name: "frontmatter missing title", docType: templates.TypeADR,
			fm:      templates.Frontmatter{Status: templates.StatusDraft},
			body:    "## Context\n" + longBody + "\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantHit: "missing or empty title",
		},
		{
			name: "body under the placeholder floor", docType: templates.TypePlan,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "Too short.",
			wantHit: "likely a placeholder",
		},
		{
			name: "cross-document body link", docType: templates.TypePlan,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "See .archcore/knowledge/other.adr.md for details. " + longBody,
			wantHit: "move these to the relation graph",
		},
		{
			name: "long code block in an ADR", docType: templates.TypeADR,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Context\n" + longBody + "\n```go\na\nb\nc\nd\ne\n```\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantHit: "prefer an @path/to/file reference",
		},
		{
			name: "clean PRD produces no findings", docType: templates.TypePRD,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\n1. The user starts an export and keeps working.\n",
		},
		{
			name: "BCP 14 modal in a PRD", docType: templates.TypePRD,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\n1. The service MUST return a job ID within 200 ms.\n",
			wantHit: "in a numbered prd clause",
		},
		{
			name: "EARS clause in a PRD requirement", docType: templates.TypePRD,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\n1. WHEN the user requests an export, the job starts.\n",
			wantHit: "EARS clause in a prd requirement",
		},
		{
			name: "PRD carrying a section a plan owns", docType: templates.TypePRD,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\nr\n## Timeline\nPhase 1\n",
			wantHit: "section ## Timeline in a prd",
		},
		{
			name: "PRD carrying a section a spec owns", docType: templates.TypePRD,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\nr\n## Normative Behavior\n1. x\n",
			wantHit: "section ## Normative Behavior in a prd",
		},
		{
			name: "a spec keeps its own notation", docType: templates.TypeSpec,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose & Scope\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n1. WHEN the user acts, the service MUST respond.\n## Conformance\nc\n",
		},
		{
			name: "long code block in a rule does not trigger", docType: templates.TypeRule,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n" + longBody + "\n```go\na\nb\nc\nd\ne\n```\n## Rationale\nr\n## Enforcement\n`manual review`\n",
		},
		{
			name: "short code block in an ADR does not trigger", docType: templates.TypeADR,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n```go\na\nb\nc\nd\n```\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
		},
		{
			name: "requirement past the word cap", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST keep every exported function in `internal/api` documented with a sentence that names its caller, its failure mode, its retry policy and its budget.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "requirement over 25 words",
		},
		{
			name: "requirement inside the word cap does not trigger", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST document every exported function in `internal/api` with a sentence naming its caller and its failure mode.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
		},
		{
			name: "step past the word cap", docType: templates.TypeGuide,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Prerequisites\np\n## Steps\n1. Run the command and then wait for the server to report a healthy status before you continue with the next migration.\n\n" +
				longBody + "\n## Verification\nv\n",
			wantHit: "step over 20 words",
		},
		{
			name: "compound requirement in a rule", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST log the error in `cmd/` and MUST NOT retry the request.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "compound requirement",
		},
		{
			name: "condition stated after the obligation", docType: templates.TypeSpec,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose & Scope\n" + longBody +
				"\n## Surface\ns\n## Normative Behavior\n1. The resolver MUST return the cleaned path when the input is absolute.\n## Conformance\nc\n",
			wantHit: "condition after the obligation",
		},
		{
			name: "clause opening with an EARS trigger keeps its later condition", docType: templates.TypeSpec,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose & Scope\n" + longBody +
				"\n## Surface\ns\n## Normative Behavior\n1. WHEN the input is absolute, the resolver MUST return the cleaned path if the cache is warm.\n## Conformance\nc\n",
		},
		{
			name: "open-ended list in a requirement", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST validate every input in `cmd/`, etc.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "open-ended list",
		},
		{
			name: "ambiguous alternative in a requirement", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST log and/or retry the failed request in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "ambiguous alternative",
		},
		{
			name: "modal in a procedure step", docType: templates.TypeGuide,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Prerequisites\np\n## Steps\n1. The operator MUST restart the daemon.\n\n" +
				longBody + "\n## Verification\nv\n",
			wantHit: "BCP 14 modal in a step",
		},
		{
			name: "modal in an idea clause", docType: templates.TypeIdea,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Idea\n" + longBody +
				"\n## Value\nv\n## Risks and Constraints\n1. The importer MUST reject a malformed archive.\n",
			wantHit: "in a numbered idea clause",
		},
		{
			name: "bullets in an ADR context", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n\n- the current situation\n\n## Decision\nd\n" +
				"## Alternatives Considered\n1. Redis — rejected because the round trip costs 18ms.\n2. Memcached — ruled out because eviction drops sessions.\n## Consequences\nc\n",
			wantHit: "bullets in Context",
		},
		{
			name: "alternative dropped without a reason", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n## Decision\nd\n" +
				"## Alternatives Considered\n1. Redis was also considered.\n2. Memcached — ruled out because eviction drops sessions.\n## Consequences\nc\n",
			wantHit: "alternative with no stated reason",
		},
		{
			name: "single alternative recorded", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n## Decision\nd\n" +
				"## Alternatives Considered\n1. Redis — rejected because the round trip costs 18ms.\n## Consequences\nc\n",
			wantHit: "one alternative recorded",
		},
		{
			name: "cpat describing the old form instead of showing it", docType: templates.TypeCPAT,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Why\n" + longBody + "\n## Before\nWe used a loop.\n## After\n```go\nx\n```\n## Scope\n`internal/**`\n",
			wantHit: "## Before holds no code block",
		},
		{
			name: "rule enforcement naming no verifier", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST validate every input in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\nBe careful.\n",
			wantHit: "Enforcement names no verifier",
		},
		{
			name: "rule naming no file target", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The author MUST state the reason for the change.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`manual review`\n",
			wantHit: "no path or glob in the Rule section",
		},
		{
			// The Good and Bad blocks a rule owes its reader are examples, not
			// that rule's own requirements: measuring them reported every
			// well-written rule as a violation of itself.
			name: "clauses inside a fenced example are not measured", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST validate every input in `cmd/`.\n\n```markdown\n1. The developer MUST do one thing and MUST NOT do the other thing named here in this example clause that runs past every cap\n```\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
		},
		{
			name: "russian rule produces no findings", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. Разработчик MUST проверять ввод в `cmd/`.\n\n" +
				strings.Repeat("Это тело документа достаточно длинное, чтобы пройти порог заглушки. ", 4) +
				"\n## Rationale\nr\n## Enforcement\n`ci`\n",
		},
		{
			// The corpus writes "1. **WHEN** ...". Read as a plain clause, the
			// opener went unrecognized and the trigger it states up front was
			// reported as a trigger stated late.
			name: "bold EARS opener is still an opener", docType: templates.TypeSpec,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Purpose & Scope\n" + longBody +
				"\n## Surface\ns\n## Normative Behavior\n1. **WHEN** the input is absolute, the resolver MUST return the cleaned path if the cache is warm.\n## Conformance\nc\n",
			wantNoHit: "condition after the obligation",
		},
		{
			name: "bold EARS opener in a PRD is still reported", docType: templates.TypePRD,
			fm:      templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:    "## Vision\n" + longBody + "\n## Problem Statement\np\n## Goals and Success Metrics\ng\n## Requirements\n1. **WHEN** the user requests an export, the job starts.\n",
			wantHit: "EARS clause in a prd requirement",
		},
		{
			// "MUST set the `when` field" names a thing; it states no trigger.
			name: "condition word inside an identifier is not a condition", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST set the `when` field in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "condition after the obligation",
		},
		{
			name: "adverbial once is not a condition", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST run the migration once in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "condition after the obligation",
		},
		{
			// "the br-and so on-call" carries the marker as a substring only.
			name: "open-list marker inside a word does not count", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The operator MUST relabel the brand so on-call staff find it in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "open-ended list",
		},
		{
			name: "a version at the start of a line is not a numbered clause", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n1.25 or newer MUST be installed.\n" +
				"## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantNoHit: "in a numbered adr clause",
		},
		{
			// A stray fence used to hide every item after it. The check is
			// advisory and over-eager by design, so unpaired fences are ignored
			// rather than allowed to silence the rest of the document.
			name: "an unclosed fence does not silence the items after it", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST validate input in `cmd/`.\n\n```go\nexample\n\n" +
				"2. The developer MUST keep every exported function in `internal/api` documented with a sentence that names its caller, its failure mode, its retry policy and its budget.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "requirement over 25 words",
		},
		{
			name: "a subsection heading is not folded into the clause above it", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The developer MUST validate input in `cmd/`.\n### Notes\n" +
				"This note runs long enough that folding it into the clause above would carry that clause well past the word cap.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "requirement over 25 words",
		},
		{
			name: "an identifier in backticks is not a file target", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The author MUST write `TODO` in the summary.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantHit: "no path or glob in the Rule section",
		},
		{
			// A plan records claims and still holds a task list. The task is
			// graded as a step, which is the advice its author can act on.
			name: "modal in a plan task reports the step message", docType: templates.TypePlan,
			fm:        templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body:      "## Goal\n" + longBody + "\n## Tasks\n1. The developer MUST migrate the schema.\n## Acceptance Criteria\na\n",
			wantHit:   "BCP 14 modal in a step",
			wantNoHit: "records a claim",
		},
		{
			name: "a fenced sample in Context is not a bullet list", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n\n```yaml\nlist:\n  - a\n```\n\n" +
				"## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
			wantNoHit: "bullets in Context",
		},
		{
			// The clause that forbids "such as `etc.`" names the marker; it does
			// not use one. Matching it reported the prohibition as the defect.
			name: "a backticked marker is named, not used", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. The author MUST NOT use a marker such as `etc.` in `cmd/`.\n\n" +
				longBody + "\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "open-ended list",
		},
		{
			name: "russian rejection reason is a stated reason", docType: templates.TypeADR,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n## Decision\nd\n" +
				"## Alternatives Considered\n1. Redis — отклонён: круговой путь стоит 18 мс.\n2. Memcached — исключён, вытеснение сбрасывает сессии.\n## Consequences\nc\n",
			wantNoHit: "alternative with no stated reason",
		},
		{
			name: "russian scope phrase is a scope", docType: templates.TypeRule,
			fm: templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Rule\n1. Разработчик MUST проверять ввод. Применяется к командам инициализации.\n\n" +
				strings.Repeat("Это тело документа достаточно длинное, чтобы пройти порог заглушки. ", 4) +
				"\n## Rationale\nr\n## Enforcement\n`ci`\n",
			wantNoHit: "no path or glob in the Rule section",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PrecisionFindings(tt.docType, tt.fm, tt.body)
			joined := strings.Join(got, " | ")
			if tt.wantNoHit != "" && strings.Contains(joined, tt.wantNoHit) {
				t.Errorf("findings carry %q, which this case exists to rule out: %s", tt.wantNoHit, joined)
			}
			if tt.wantHit == "" && tt.wantNoHit == "" {
				if len(got) > 0 {
					t.Errorf("expected no findings, got: %s", joined)
				}
				return
			}
			if tt.wantHit != "" && !strings.Contains(joined, tt.wantHit) {
				t.Errorf("findings missing %q, got: %s", tt.wantHit, joined)
			}
		})
	}
}

// TestPrecisionFindings_GeneratedTemplatesAreClean pins the invariant the prd
// template broke: the skeleton create_document hands an author must not come
// back from the post-write hook with findings against its own text.
//
// The prd template lists "MUST / SHOULD / MAY" in the prose that tells the
// author to keep those modals in a linked spec, and the modal check read the
// whole body — so every freshly created prd was reported for the instruction it
// had just been given. The check now reads numbered clauses only, and this test
// fails if any template drifts back across its own rule.
func TestPrecisionFindings_GeneratedTemplatesAreClean(t *testing.T) {
	t.Parallel()
	fm := templates.Frontmatter{Title: "T", Status: templates.StatusDraft}

	for _, typ := range templates.ValidTypes() {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			docType := templates.DocumentType(typ)

			var findings []string
			for _, f := range PrecisionFindings(docType, fm, templates.GenerateTemplate(docType)) {
				// A skeleton is a placeholder by definition, so the body-length
				// floor is the one finding a template may trip.
				if strings.Contains(f, "likely a placeholder") {
					continue
				}
				findings = append(findings, f)
			}
			if len(findings) > 0 {
				t.Errorf("generated %s template produces findings:\n  %s", typ, strings.Join(findings, "\n  "))
			}
		})
	}
}

// TestSectionTables_MatchTemplates is the regression lock for the largest
// silent failure this engine can have: a section rule that names a real heading
// whose body holds no numbered item. Nothing fails when that happens — the check
// simply measures an empty list and reports nothing, so the table claims a
// coverage the engine does not have.
//
// It cost two of the three StepSections entries: the guide template wrote its
// steps as "### Step 1:" subsections and the plan template as "- [ ]"
// checkboxes, so the word cap and the modal check were unreachable for both.
func TestSectionTables_MatchTemplates(t *testing.T) {
	t.Parallel()

	tables := map[string]map[templates.DocumentType][]templates.SectionRule{
		"ClauseSections": templates.ClauseSections,
		"StepSections":   templates.StepSections,
	}
	for label, table := range tables {
		for docType, rules := range table {
			t.Run(label+"/"+string(docType), func(t *testing.T) {
				t.Parallel()
				lines := strings.Split(templates.GenerateTemplate(docType), "\n")
				if items := sectionItems(lines, rules); len(items) == 0 {
					t.Errorf("%s[%q] yields no item from the %s template — the table names a form the template does not emit",
						label, docType, docType)
				}
			})
		}
	}
}

// TestPrecisionFindings_BodyLengthCountsCharacters: the placeholder check
// measured len(body), which is bytes.
//
// Every non-ASCII character costs two or more of them, so a 120-character
// Russian document read as 240+ and skipped the check entirely — under-firing by
// roughly half for exactly the documents the Russian half of the vagueness
// lexicon exists to serve. The number printed alongside claimed to be characters
// either way.
func TestPrecisionFindings_BodyLengthCountsCharacters(t *testing.T) {
	t.Parallel()
	const under = "короткая заготовка. " // 20 runes, 37 bytes

	tests := []struct {
		name    string
		body    string
		wantHit bool
	}{
		// 120 runes / 222 bytes: under the 200-character floor, over it in bytes.
		{name: "a short Cyrillic body is flagged", body: strings.Repeat(under, 6), wantHit: true},
		// 240 runes: over the floor by either measure.
		{name: "a long Cyrillic body is not flagged", body: strings.Repeat(under, 12)},
		{name: "a short ASCII body is flagged", body: strings.Repeat("filler word ", 6), wantHit: true},
		{name: "a long ASCII body is not flagged", body: strings.Repeat("filler word ", 30)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fm := templates.Frontmatter{Title: "T", Status: templates.StatusDraft}
			joined := strings.Join(PrecisionFindings(templates.TypeDoc, fm, tt.body), " | ")

			if got := strings.Contains(joined, "likely a placeholder"); got != tt.wantHit {
				t.Errorf("placeholder finding = %v, want %v (%d runes, %d bytes):\n%s",
					got, tt.wantHit, utf8.RuneCountInString(tt.body), len(tt.body), joined)
			}
		})
	}
}
