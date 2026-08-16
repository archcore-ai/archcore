package advisory

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"archcore-cli/templates"
)

// Cap and ordering behavior of the precision findings.
//
// Each finding lists at most a handful of examples. Which examples it picks has
// to be a function of the document's content, not of where in the document the
// offenders happen to sit — otherwise reformatting a file silently changes the
// advisory and the reader learns to distrust it.

const specPreamble = "## Purpose\n" +
	"A body long enough to clear the placeholder floor, repeated for length. " +
	"A body long enough to clear the placeholder floor, repeated for length. " +
	"A body long enough to clear the placeholder floor, repeated for length.\n" +
	"## Surface\ns\n## Normative Behavior\n"

// findingContaining returns the one finding holding needle.
func findingContaining(t *testing.T, findings []string, needle string) string {
	t.Helper()
	for _, f := range findings {
		if strings.Contains(f, needle) {
			return f
		}
	}
	t.Fatalf("no finding contains %q; got:\n%s", needle, strings.Join(findings, "\n"))
	return ""
}

// examplesIn pulls the comma-separated list out of "prefix (a, b, c) — advice".
func examplesIn(t *testing.T, finding string) []string {
	t.Helper()
	open := strings.Index(finding, "(")
	close := strings.Index(finding, ")")
	if open < 0 || close < open {
		t.Fatalf("finding carries no example list: %s", finding)
	}
	return strings.Split(finding[open+1:close], ", ")
}

// TestPrecision_VaguenessCapIsOrderIndependent: with more offenders than the
// cap, the reported five must be the alphabetically first five. Capping in
// encounter order made the answer depend on document order.
func TestPrecision_VaguenessCapIsOrderIndependent(t *testing.T) {
	t.Parallel()
	// Eight offenders, deliberately in reverse alphabetical order.
	body := "## Context\nThe streamlined, seamless, robust, optimal, modern, flexible, " +
		"efficient and convenient design. " + strings.Repeat("Padding for the body floor. ", 8) +
		"\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n"

	got := PrecisionFindings(templates.TypeADR,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	examples := examplesIn(t, findingContaining(t, got, "vague wording"))
	want := []string{"convenient", "efficient", "flexible", "modern", "optimal"}
	if !slices.Equal(examples, want) {
		t.Errorf("vagueness examples = %v, want the five alphabetically first %v", examples, want)
	}
}

// TestPrecision_CrossDocCapIsOrderIndependent: same contract for the
// cross-document link finding.
func TestPrecision_CrossDocCapIsOrderIndependent(t *testing.T) {
	t.Parallel()
	body := "## Context\nSee .archcore/e.adr.md, .archcore/d.adr.md, .archcore/c.adr.md, " +
		".archcore/b.adr.md and .archcore/a.adr.md. " + strings.Repeat("Padding for the body floor. ", 8) +
		"\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n"

	got := PrecisionFindings(templates.TypeADR,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	examples := examplesIn(t, findingContaining(t, got, "body links other"))
	want := []string{".archcore/a.adr.md", ".archcore/b.adr.md", ".archcore/c.adr.md"}
	if !slices.Equal(examples, want) {
		t.Errorf("cross-document examples = %v, want the three alphabetically first %v", examples, want)
	}
}

// TestPrecision_PassiveCapHoldsAcrossClauses: the cap used to break out of the
// per-line loop only, so once the count passed three the equality test never
// matched again and the list grew without bound.
func TestPrecision_PassiveCapHoldsAcrossClauses(t *testing.T) {
	t.Parallel()
	body := specPreamble +
		"1. The key MUST be rotated.\n" +
		"2. The token SHOULD be refreshed.\n" +
		"3. The cache MUST be purged.\n" +
		"4. The log MUST be archived.\n" +
		"5. The session SHOULD be renewed.\n" +
		"## Conformance\nc\n"

	got := PrecisionFindings(templates.TypeSpec,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	examples := examplesIn(t, findingContaining(t, got, "subjectless passive"))
	// The literal list, not maxPassiveHits: comparing the count against the
	// constant follows it wherever it moves. Passives are kept in clause order.
	want := []string{"MUST be rotated", "SHOULD be refreshed", "MUST be purged"}
	if !slices.Equal(examples, want) {
		t.Errorf("passive examples = %v, want the first three in clause order %v", examples, want)
	}
}

// TestPrecision_PassiveCapDropsLaterClauses: the cap must bind on the number of
// distinct passives, not on how many clauses carry them.
func TestPrecision_PassiveCapDropsLaterClauses(t *testing.T) {
	t.Parallel()
	body := specPreamble +
		"1. The key MUST be rotated.\n" +
		"2. The token SHOULD be refreshed.\n" +
		"3. The cache MUST be purged.\n" +
		"4. The log MUST be archived.\n" +
		"## Conformance\nc\n"

	got := PrecisionFindings(templates.TypeSpec,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	finding := findingContaining(t, got, "subjectless passive")
	if strings.Contains(finding, "MUST be archived") {
		t.Errorf("the fourth passive survived a three-example budget:\n%s", finding)
	}
}

// TestPrecision_SpecBodyLineCap covers the 80-line boundary in both directions.
func TestPrecision_SpecBodyLineCap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		lines   int
		wantHit bool
	}{
		{name: "at the cap", lines: templates.MaxSpecBodyLines},
		{name: "one over the cap", lines: templates.MaxSpecBodyLines + 1, wantHit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lines := []string{"## Purpose", "## Surface", "## Normative Behavior",
				"1. The server MUST answer.", "## Conformance"}
			for len(lines) < tt.lines {
				lines = append(lines, fmt.Sprintf("filler line %d", len(lines)))
			}
			body := strings.Join(lines, "\n")

			got := PrecisionFindings(templates.TypeSpec,
				templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

			hit := strings.Contains(strings.Join(got, " | "), "spec body is")
			if hit != tt.wantHit {
				t.Errorf("body-length finding = %v, want %v (%d lines)", hit, tt.wantHit, tt.lines)
			}
		})
	}
}

// TestPrecision_RussianSubjectlessPassive covers the Russian half of passiveRe,
// which no case exercised.
func TestPrecision_RussianSubjectlessPassive(t *testing.T) {
	t.Parallel()
	body := specPreamble + "1. Ключ MUST ротироваться каждые 90 дней.\n## Conformance\nc\n"

	got := PrecisionFindings(templates.TypeSpec,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	if !strings.Contains(strings.Join(got, " | "), "subjectless passive") {
		t.Errorf("a Russian subjectless passive was not flagged:\n%s", strings.Join(got, "\n"))
	}
}

// TestPrecision_PassiveOutsideANumberedClauseIsIgnored: only numbered clauses
// are graded requirements, so prose using the same shape is not a finding.
func TestPrecision_PassiveOutsideANumberedClauseIsIgnored(t *testing.T) {
	t.Parallel()
	body := specPreamble + "The key MUST be rotated, as discussed.\n1. The server MUST answer.\n## Conformance\nc\n"

	got := PrecisionFindings(templates.TypeSpec,
		templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

	if strings.Contains(strings.Join(got, " | "), "subjectless passive") {
		t.Errorf("a passive in prose was graded as a requirement:\n%s", strings.Join(got, "\n"))
	}
}

// noisySpecBody trips more distinct checks than maxReportFindings admits, so the
// cap test exercises the cut rather than the happy path. Written as concatenated
// lines because the body carries a fenced code block.
var noisySpecBody = "---\n---\n" +
	"## Normative Behavior\n\n" +
	"1. The gateway SHALL retry the appropriate request and MUST NOT exceed the configured ceiling, etc. when the upstream is slow and the operator has enabled the flag.\n" +
	"2. Requests are validated before they reach the handler.\n" +
	"3. Compare with .archcore/plugin/one.adr.md and .archcore/plugin/two.spec.md for the rest of the contract.\n\n" +
	"## Notes\n\n" +
	"Pasted implementation detail rather than an @path reference:\n\n" +
	"```go\n" +
	"func handle() {\n" +
	"\tvalidate()\n" +
	"\tforward()\n" +
	"\tlog()\n" +
	"\treturn\n" +
	"}\n" +
	"```\n" +
	// Padding past MaxSpecBodyLines, so the body-length check fires too.
	strings.Repeat("Filler prose line carrying no obligation.\n", 80)

// TestPrecision_ReportFindingsCap: the report leaving the hook is bounded by
// maxReportFindings and states how much it dropped. The findings arrive in
// fixed check order, so the same document always keeps the same head.
func TestPrecision_ReportFindingsCap(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	// A spec tripping more distinct checks than the report may show: empty
	// frontmatter (title, status), every required section missing but Normative
	// Behavior, a foreign section, vague wording, cross-document links, a pasted
	// code block, SHALL in place of a BCP 14 modal, a compound requirement, a
	// subjectless passive, a trailing condition, an open-ended list, and clauses
	// past the word cap.
	writeArchcoreDoc(t, base, "noisy.spec.md", noisySpecBody)

	got := Precision(base, "mcp__archcore__update_document", "noisy.spec.md")

	lines := strings.Split(got, "\n")
	// The header, maxReportFindings findings, and the line counting the rest.
	if len(lines) != maxReportFindings+2 {
		t.Fatalf("report carries %d lines, want %d (fixture must trip more than the cap — strengthen noisySpecBody if the cap grew):\n%s",
			len(lines), maxReportFindings+2, got)
	}
	if last := lines[len(lines)-1]; !strings.Contains(last, "not shown") {
		t.Errorf("report does not say what it dropped: %q", last)
	}
}

// TestPrecision_ReportUnderCapCarriesNoCutLine: a report that fits is not
// suffixed with a count of nothing.
func TestPrecision_ReportUnderCapCarriesNoCutLine(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	writeArchcoreDoc(t, base, "small.rule.md",
		"---\ntitle: T\nstatus: draft\n---\n## Rule\n1. The developer MUST validate input in `cmd/`.\n## Rationale\nr\n## Enforcement\n`ci`\n")

	got := Precision(base, "mcp__archcore__update_document", "small.rule.md")

	if got == "" {
		t.Fatal("expected the body-floor finding for a short document")
	}
	if strings.Contains(got, "not shown") {
		t.Errorf("an under-cap report claims a cut:\n%s", got)
	}
}

// TestPrecision_SectionHeadingMatching pins the heading rule the section check
// implements: the name, then whitespace or end of line.
func TestPrecision_SectionHeadingMatching(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		heading string
		wantHit bool // a missing-section finding is expected
	}{
		{name: "exact heading", heading: "## Purpose"},
		{name: "heading with a qualifier", heading: "## Purpose & Scope"},
		{name: "heading with trailing space", heading: "## Purpose "},
		{name: "extra leading space is still a heading", heading: "##   Purpose"},
		{name: "a longer word is not the section", heading: "## Purposeful", wantHit: true},
		{name: "level-3 heading does not count", heading: "### Purpose", wantHit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := tt.heading + "\n## Surface\ns\n## Normative Behavior\n1. The server MUST answer.\n## Conformance\nc\n"

			got := PrecisionFindings(templates.TypeSpec,
				templates.Frontmatter{Title: "T", Status: templates.StatusDraft}, body)

			hit := strings.Contains(strings.Join(got, " | "), "missing section: ## Purpose")
			if hit != tt.wantHit {
				t.Errorf("missing-Purpose finding = %v, want %v for %q", hit, tt.wantHit, tt.heading)
			}
		})
	}
}
