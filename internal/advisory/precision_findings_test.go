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
			wantHit: "BCP 14 modal in a prd",
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
			body: "## Rule\n" + longBody + "\n```go\na\nb\nc\nd\ne\n```\n## Rationale\nr\n## Enforcement\ne\n",
		},
		{
			name: "short code block in an ADR does not trigger", docType: templates.TypeADR,
			fm:   templates.Frontmatter{Title: "T", Status: templates.StatusDraft},
			body: "## Context\n" + longBody + "\n```go\na\nb\nc\nd\n```\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PrecisionFindings(tt.docType, tt.fm, tt.body)
			joined := strings.Join(got, " | ")
			if tt.wantHit == "" {
				if len(got) > 0 {
					t.Errorf("expected no findings, got: %s", joined)
				}
				return
			}
			if !strings.Contains(joined, tt.wantHit) {
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
