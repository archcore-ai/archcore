package templates

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {
	tests := []struct {
		name         string
		documentType DocumentType
		wantContains []string
	}{
		{
			name:         "ADR template",
			documentType: TypeADR,
			wantContains: []string{"## Context", "## Decision", "## Consequences", "### Positive", "### Negative", "## Alternatives Considered"},
		},
		{
			name:         "RFC template",
			documentType: TypeRFC,
			wantContains: []string{"## Summary", "## Motivation", "## Detailed Design", "## Drawbacks", "## Alternatives", "## Unresolved Questions", "## Implementation Plan", "## Security Considerations"},
		},
		{
			name:         "Rule template",
			documentType: TypeRule,
			wantContains: []string{"## Description", "## Rule", "## Examples", "### Good", "### Bad", "## Exceptions", "## Enforcement"},
		},
		{
			name:         "Guide template",
			documentType: TypeGuide,
			wantContains: []string{"## Overview", "## Prerequisites", "## Steps", "## Common Issues", "## Verification"},
		},
		{
			name:         "Doc template",
			documentType: TypeDoc,
			wantContains: []string{"## Overview", "## Content", "## Examples", "## Best Practices", "## FAQ"},
		},
		{
			name:         "Task-Type template",
			documentType: TypeTaskType,
			wantContains: []string{"## What", "## When to Use", "## Steps", "## Example", "## Things to Watch Out For"},
		},
		{
			name:         "CPAT template",
			documentType: TypeCPAT,
			wantContains: []string{"## What Changed", "## Why", "## Before", "## After", "## Scope"},
		},
		{
			name:         "PRD template",
			documentType: TypePRD,
			wantContains: []string{"## Vision", "## Problem Statement", "## Goals and Success Metrics", "## Requirements"},
		},
		{
			name:         "Idea template",
			documentType: TypeIdea,
			wantContains: []string{"## Idea", "## Value", "## Possible Implementation", "## Risks and Constraints", "## Next Steps"},
		},
		{
			name:         "Plan template",
			documentType: TypePlan,
			wantContains: []string{"## Goal", "## Tasks", "## Acceptance Criteria", "## Dependencies", "## Notes"},
		},
		{
			name:         "RnD template",
			documentType: TypeRnD,
			wantContains: []string{"## Research Goal", "## Context & Trigger", "## Questions / Hypotheses", "## Approach", "## Findings", "## Implications", "## Recommendation", "## Next Action", "## Risks & Unknowns", "## Related Materials"},
		},
		{
			name:         "Spec template",
			documentType: TypeSpec,
			wantContains: []string{"## Purpose & Scope", "## Surface", "## Normative Behavior", "## Constraints & Invariants", "## Failure Behavior", "## Conformance"},
		},
		{
			name:         "MRD template",
			documentType: TypeMRD,
			wantContains: []string{"## Market Landscape", "## TAM / SAM / SOM", "## Competitive Analysis", "## Market Needs", "## Opportunity and Timing"},
		},
		{
			name:         "BRD template",
			documentType: TypeBRD,
			wantContains: []string{"## Business Objectives", "## Stakeholders", "## Business Rules and Constraints", "## Success Metrics and ROI", "## Dependencies"},
		},
		{
			name:         "URD template",
			documentType: TypeURD,
			wantContains: []string{"## User Personas", "## User Journeys", "## User Requirements", "## Usability Requirements", "## Acceptance Criteria"},
		},
		{
			name:         "BRS template",
			documentType: TypeBRS,
			wantContains: []string{"## Business Purpose and Scope", "## Business Overview", "### Information Environment", "## Mission, Goals and Objectives", "## Business Operations", "## Business Constraints", "## High-Level Operational Concept", "### Business Operational Quality", "## Project Constraints", "## Success Criteria", "## Assumptions and Dependencies", "## Traceability"},
		},
		{
			name:         "StRS template",
			documentType: TypeStRS,
			wantContains: []string{"## Purpose and Scope", "## System Overview", "## Business Context", "## Stakeholder Classes", "## Operational Concept", "## Stakeholder Requirements", "## System Processes", "## Operational Policies and Rules", "## Operational Constraints", "## Compliance and Regulatory", "## Project Constraints", "## Traceability"},
		},
		{
			name:         "SyRS template",
			documentType: TypeSyRS,
			wantContains: []string{"## System Purpose and Scope", "## System Overview", "## System Requirements", "### Information Management Requirements", "## System Interfaces", "## System Operations", "## Policy and Regulation", "## Life Cycle Sustainment", "## Assumptions and Dependencies", "## Verification Approach", "## Traceability"},
		},
		{
			name:         "SRS template",
			documentType: TypeSRS,
			wantContains: []string{"## Purpose and Scope", "## Product Perspective", "### Assumptions and Dependencies", "## Software Requirements", "## External Interfaces", "## Data Requirements", "## Usability Requirements", "## Performance", "## Design Constraints", "## Software Quality Attributes", "## Verification Matrix", "## Traceability"},
		},
		{
			name:         "Unknown type falls back to doc template",
			documentType: DocumentType("unknown"),
			wantContains: []string{"## Overview", "## Content", "## Examples"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GenerateTemplate(tt.documentType)

			if got == "" {
				t.Errorf("GenerateTemplate(%q) returned empty string", tt.documentType)
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateTemplate(%q) missing expected section: %q", tt.documentType, want)
				}
			}
		})
	}
}

func TestGenerateADRTemplate(t *testing.T) {
	template := generateADRTemplate()

	requiredSections := []string{
		"## Context",
		"## Decision",
		"## Consequences",
		"### Positive",
		"### Negative",
		"## Alternatives Considered",
		"### Rationale",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("ADR template missing section: %q", section)
		}
	}

	forbiddenSections := []string{
		"## References",
		"## Related Documents",
	}
	for _, section := range forbiddenSections {
		if strings.Contains(template, section) {
			t.Errorf("ADR template must not contain cross-document section: %q", section)
		}
	}

	// Canonical tokens rather than a prose word: the template owes its author the
	// evidence obligation, and "[assumption]" is the marker the canon names for
	// the case where no evidence exists yet.
	for _, token := range []string{"[assumption]", "@path/to/file"} {
		if !strings.Contains(template, token) {
			t.Errorf("ADR template should teach the evidence obligation: missing %q", token)
		}
	}
}

func TestGenerateRFCTemplate(t *testing.T) {
	template := generateRFCTemplate()

	requiredSections := []string{
		"## Summary",
		"## Motivation",
		"## Detailed Design",
		"## Drawbacks",
		"## Alternatives",
		"## Unresolved Questions",
		"## Implementation Plan",
		"## Security Considerations",
		"## Testing Strategy",
		"## Rollout Plan",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("RFC template missing section: %q", section)
		}
	}

	if !strings.Contains(template, "- [ ]") {
		t.Error("RFC template should include checkboxes in Implementation Plan")
	}
}

func TestGenerateRuleTemplate(t *testing.T) {
	template := generateRuleTemplate()

	requiredSections := []string{
		"## Description",
		"## Rule",
		"## Examples",
		"### Good",
		"### Bad",
		"## Exceptions",
		"## Enforcement",
		"## Rationale",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Rule template missing section: %q", section)
		}
	}

	if strings.Contains(template, "## References") {
		t.Error("Rule template must not contain a References section listing other archcore documents")
	}

	if !strings.Contains(template, "```") {
		t.Error("Rule template should include code block markers")
	}
}

func TestGenerateGuideTemplate(t *testing.T) {
	template := generateGuideTemplate()

	requiredSections := []string{
		"## Overview",
		"## Prerequisites",
		"## Steps",
		"## Common Issues",
		"## Verification",
		"## Next Steps",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Guide template missing section: %q", section)
		}
	}

	if strings.Contains(template, "## Related Resources") {
		t.Error("Guide template must not contain a Related Resources section listing other archcore documents")
	}

	// Numbered steps, not "### Step N:" subsections. StepSections names ## Steps
	// as the guide's step body and the engine reads a step as a numbered item,
	// so a subsection-shaped template made every step check unreachable.
	for _, step := range []string{"\n1. ", "\n2. ", "\n3. ", "\n4. "} {
		if !strings.Contains(template, step) {
			t.Errorf("Guide template missing numbered step: %q", step)
		}
	}
}

func TestGenerateDocTemplate(t *testing.T) {
	template := generateDocTemplate()

	requiredSections := []string{
		"## Overview",
		"## Content",
		"## Examples",
		"## Best Practices",
		"## FAQ",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Doc template missing section: %q", section)
		}
	}

	if strings.Contains(template, "## Related Resources") {
		t.Error("Doc template must not contain a Related Resources section listing other archcore documents")
	}
}

func TestGenerateSpecTemplate(t *testing.T) {
	template := generateSpecTemplate()

	requiredSections := []string{
		"## Purpose & Scope",
		"## Surface",
		"## Normative Behavior",
		"## Constraints & Invariants",
		"## Failure Behavior",
		"## Conformance",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Spec template missing section: %q", section)
		}
	}

	// Legacy sections must not resurface — the lean six-section canon replaced them.
	for _, legacy := range []string{"## Authority", "## Subject", "## Definitions", "## Contract Surface", "## Error Handling", "## State Model"} {
		if strings.Contains(template, legacy) {
			t.Errorf("Spec template contains legacy section: %q", legacy)
		}
	}

	// Behavior lines follow EARS clause structure.
	for _, clause := range []string{"WHEN", "WHILE", "IF", "THEN"} {
		if !strings.Contains(template, clause) {
			t.Errorf("Spec template missing EARS clause keyword: %q", clause)
		}
	}

	// BCP 14 keywords with the standards cited.
	for _, keyword := range []string{"MUST", "SHOULD", "MAY", "RFC 2119", "RFC 8174"} {
		if !strings.Contains(template, keyword) {
			t.Errorf("Spec template missing normative keyword or citation: %q", keyword)
		}
	}

	// Strict-EARS rules: active voice with an obligated subject, one requirement
	// (one modal) per numbered line, and explicit WHEN for event responses —
	// codified so the guidance cannot silently regress.
	for _, rule := range []string{
		"never a subjectless passive",
		"one requirement = one modal",
		"MUST X and MUST NOT Y",
		"the event is the trigger, never the subject",
		"Same notation and rules",
	} {
		if !strings.Contains(template, rule) {
			t.Errorf("Spec template missing strict-EARS rule phrase: %q", rule)
		}
	}

	// The template must stay lean — the spec body cap is 80 lines.
	if lines := strings.Count(template, "\n"); lines > 80 {
		t.Errorf("Spec template is %d lines, must stay within the 80-line spec body cap", lines)
	}

	// Code block markers stay balanced (one Given/When/Then example block).
	codeBlockCount := strings.Count(template, "```")
	if codeBlockCount%2 != 0 {
		t.Errorf("code block markers = %d, should be even", codeBlockCount)
	}

	// Cross-document linking belongs in the relation graph, not in the body.
	if strings.Contains(template, "Related Artifacts") {
		t.Error("Spec template must not contain a Related Artifacts section listing other archcore documents")
	}
}

func TestGenerateTaskTypeTemplate(t *testing.T) {
	template := generateTaskTypeTemplate()

	requiredSections := []string{
		"## What",
		"## When to Use",
		"## Steps",
		"## Example",
		"## Things to Watch Out For",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("TaskType template missing section: %q", section)
		}
	}
}

func TestGenerateCPATTemplate(t *testing.T) {
	template := generateCPATTemplate()

	requiredSections := []string{
		"## What Changed",
		"## Why",
		"## Before",
		"## After",
		"## Scope",
		"## Notes",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("CPAT template missing section: %q", section)
		}
	}

	if !strings.Contains(template, "```") {
		t.Error("CPAT template should include code blocks")
	}
}

func TestGeneratePRDTemplate(t *testing.T) {
	template := generatePRDTemplate()

	requiredSections := []string{
		"## Vision",
		"## Problem Statement",
		"## Goals and Success Metrics",
		"## Requirements",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("PRD template missing section: %q", section)
		}
	}

	// The four sections above are the whole canon. A heading beyond them
	// invites content another type owns, which is how a prd and its spec end
	// up restating each other.
	//
	// Counted line by line rather than by substring: the template names
	// optional sections inline as `## Out of Scope`, and a substring count
	// would take those for headings.
	headings := 0
	for line := range strings.SplitSeq(template, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings++
		}
	}
	if headings != len(requiredSections) {
		t.Errorf("PRD template top-level headings = %d, want %d", headings, len(requiredSections))
	}

	// Sections owned by spec, plan, and adr must not reappear as prd headings.
	for _, foreign := range []string{
		"## Solution Overview",
		"## Timeline",
		"## Constraints",
		"## Risks and Mitigations",
		"### Functional Requirements",
		"### Non-Functional Requirements",
	} {
		if strings.Contains(template, foreign) {
			t.Errorf("PRD template carries a section owned by another type: %q", foreign)
		}
	}

	if !strings.Contains(template, "|") {
		t.Error("PRD template should include the success-metrics table")
	}
}

func TestTemplateStructure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		docType  DocumentType
		minBytes int
	}{
		{TypeADR, 510},
		{TypeRFC, 940},
		{TypeRule, 320},
		{TypeGuide, 830},
		{TypeDoc, 440},
		{TypeSpec, 1900},
		{TypeTaskType, 252},
		{TypeCPAT, 198},
		{TypePRD, 1700},
		{TypeIdea, 175},
		{TypePlan, 175},
		{TypeRnD, 1300},
		{TypeMRD, 1151},
		{TypeBRD, 1279},
		{TypeURD, 1289},
		{TypeBRS, 2115},
		{TypeStRS, 2247},
		{TypeSyRS, 2431},
		{TypeSRS, 2168},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			t.Parallel()
			got := GenerateTemplate(tt.docType)
			if n := len(got); n < tt.minBytes {
				t.Errorf("GenerateTemplate(%q) = %d bytes, want >= %d", tt.docType, n, tt.minBytes)
			}
		})
	}
}

func TestTemplateMarkdownFormatting(t *testing.T) {
	for _, typ := range ValidTypes() {
		t.Run(typ, func(t *testing.T) {
			template := GenerateTemplate(DocumentType(typ))

			hasH2 := false
			for line := range strings.SplitSeq(template, "\n") {
				if strings.HasPrefix(line, "## ") {
					hasH2 = true
					break
				}
			}
			if !hasH2 {
				t.Error("template should contain at least one H2 header line starting with '## '")
			}

			if !strings.HasSuffix(template, "\n") {
				t.Error("template should end with newline")
			}

			if strings.Contains(template, "\n\n\n\n") {
				t.Error("template has excessive empty lines")
			}
		})
	}
}

func TestRuleTemplate_CodeBlocks(t *testing.T) {
	template := generateRuleTemplate()

	codeBlockCount := strings.Count(template, "```")

	if codeBlockCount%2 != 0 {
		t.Errorf("code block markers = %d, should be even", codeBlockCount)
	}

	if codeBlockCount < 8 {
		t.Errorf("code block markers = %d, should have at least 8 (4 blocks)", codeBlockCount)
	}
}

func TestRFCTemplate_Checkboxes(t *testing.T) {
	template := generateRFCTemplate()

	checkboxCount := strings.Count(template, "- [ ]")

	if checkboxCount < 5 {
		t.Errorf("checkbox count = %d, want at least 5", checkboxCount)
	}

	if !strings.Contains(template, "## Implementation Plan") {
		t.Error("RFC should have Implementation Plan section")
	}
}

func TestGuideTemplate_CodeBlocks(t *testing.T) {
	template := generateGuideTemplate()

	codeBlockCount := strings.Count(template, "```")

	if codeBlockCount < 6 {
		t.Errorf("Guide template code blocks = %d, should have at least 6", codeBlockCount)
	}
}

func TestPRDTemplate_Tables(t *testing.T) {
	template := generatePRDTemplate()

	tableCount := strings.Count(template, "|")

	// One table: the Goal / Metric / Today / Target grid under Goals and
	// Success Metrics. The old template carried six, and five of them held
	// spec, plan, and adr content.
	if tableCount < 15 {
		t.Errorf("PRD table elements = %d, should have at least 15", tableCount)
	}
}

func TestGenerateIdeaTemplate(t *testing.T) {
	template := generateIdeaTemplate()

	requiredSections := []string{
		"## Idea",
		"### Problem / Opportunity",
		"## Value",
		"### For Users",
		"### For Business",
		"### For Team",
		"## Possible Implementation",
		"### Technical Approach",
		"### Integrations",
		"## Risks and Constraints",
		"### Potential Risks",
		"### Known Constraints",
		"## Next Steps",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Idea template missing section: %q", section)
		}
	}

	if strings.Contains(template, "## Related Materials") {
		t.Error("Idea template must not contain a Related Materials section listing other archcore documents")
	}

	// Verify checklist format
	if !strings.Contains(template, "- [ ]") {
		t.Error("Idea template should include task checkboxes")
	}

	// Verify it has guidance text
	if !strings.Contains(template, "Describe") || !strings.Contains(template, "What") {
		t.Error("Idea template should include guidance questions")
	}
}

func TestGeneratePlanTemplate(t *testing.T) {
	template := generatePlanTemplate()

	requiredSections := []string{
		"## Goal",
		"### Context",
		"## Tasks",
		"### Phase 1:",
		"### Phase 2:",
		"## Acceptance Criteria",
		"## Dependencies",
		"## Notes",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Plan template missing section: %q", section)
		}
	}

	// Numbered tasks under the phases, checkboxes only in Acceptance Criteria.
	// StepSections names ## Tasks as the plan's step body, and the engine reads
	// a step as a numbered item — a checkbox list made the cap unreachable.
	for _, task := range []string{"\n1. ", "\n2. "} {
		if !strings.Contains(template, task) {
			t.Errorf("Plan template missing numbered task: %q", task)
		}
	}
	checkboxCount := strings.Count(template, "- [ ]")
	if checkboxCount < 2 {
		t.Errorf("Plan template should keep its acceptance checkboxes, got %d", checkboxCount)
	}

	// Verify table format for dependencies
	if !strings.Contains(template, "|") {
		t.Error("Plan template should include table for dependencies")
	}

	// Verify phase structure
	phaseCount := strings.Count(template, "### Phase")
	if phaseCount < 2 {
		t.Errorf("Plan template should have at least 2 phases, got %d", phaseCount)
	}
}

func TestSplitDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		wantFM   Frontmatter
		wantBody string
		wantErr  bool
	}{
		{
			name:  "standard frontmatter",
			input: "---\ntitle: My Doc\nstatus: draft\n---\n\n## Body",
			wantFM: Frontmatter{
				Title:  "My Doc",
				Status: "draft",
			},
			wantBody: "## Body",
		},
		{
			name:  "with tags",
			input: "---\ntitle: Tagged Doc\nstatus: accepted\ntags:\n  - frontend\n  - auth\n---\n\nBody text",
			wantFM: Frontmatter{
				Title:  "Tagged Doc",
				Status: "accepted",
				Tags:   []string{"frontend", "auth"},
			},
			wantBody: "Body text",
		},
		{
			name:  "single tag",
			input: "---\ntitle: Single Tag\nstatus: draft\ntags:\n  - backend\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Single Tag",
				Status: "draft",
				Tags:   []string{"backend"},
			},
			wantBody: "Body",
		},
		{
			name:  "flow style tags",
			input: "---\ntitle: Flow Tags\nstatus: draft\ntags: [api, security]\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Flow Tags",
				Status: "draft",
				Tags:   []string{"api", "security"},
			},
			wantBody: "Body",
		},
		{
			name:     "no frontmatter",
			input:    "## Just Markdown\nSome content.",
			wantFM:   Frontmatter{},
			wantBody: "## Just Markdown\nSome content.",
		},
		{
			name:     "unclosed frontmatter",
			input:    "---\ntitle: Broken\nstatus: draft\n",
			wantFM:   Frontmatter{},
			wantBody: "---\ntitle: Broken\nstatus: draft\n",
		},
		{
			name:  "windows line endings",
			input: "---\r\ntitle: Win Doc\r\nstatus: draft\r\ntags:\r\n  - win\r\n---\r\n\r\nBody",
			wantFM: Frontmatter{
				Title:  "Win Doc",
				Status: "draft",
				Tags:   []string{"win"},
			},
			wantBody: "Body",
		},
		{
			name:     "empty",
			input:    "",
			wantFM:   Frontmatter{},
			wantBody: "",
		},
		{
			name:  "quoted title",
			input: "---\ntitle: \"Quoted Title\"\nstatus: draft\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Quoted Title",
				Status: "draft",
			},
			wantBody: "Body",
		},
		{
			name:  "escaped quotes in title",
			input: "---\ntitle: \"Title with \\\"quotes\\\"\"\nstatus: draft\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Title with \"quotes\"",
				Status: "draft",
			},
			wantBody: "Body",
		},
		{
			name:  "unknown fields ignored",
			input: "---\ntitle: Extra Fields\nstatus: draft\ncustom: value\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Extra Fields",
				Status: "draft",
			},
			wantBody: "Body",
		},
		{
			name:  "YAML special values as tags",
			input: "---\ntitle: Special\nstatus: draft\ntags:\n  - \"true\"\n  - \"null\"\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "Special",
				Status: "draft",
				Tags:   []string{"true", "null"},
			},
			wantBody: "Body",
		},
		{
			name:  "frontmatter closed at EOF no trailing newline",
			input: "---\ntitle: No Newline\nstatus: draft\n---",
			wantFM: Frontmatter{
				Title:  "No Newline",
				Status: "draft",
			},
			wantBody: "",
		},
		{
			name:  "no blank line after closing delimiter",
			input: "---\ntitle: No Gap\nstatus: draft\n---\nBody immediately",
			wantFM: Frontmatter{
				Title:  "No Gap",
				Status: "draft",
			},
			wantBody: "Body immediately",
		},
		{
			name:  "triple-dash in body preserved",
			input: "---\ntitle: Dashes\nstatus: draft\n---\n\nBody with\n---\nmore content",
			wantFM: Frontmatter{
				Title:  "Dashes",
				Status: "draft",
			},
			wantBody: "Body with\n---\nmore content",
		},
		{
			name:  "utf-8 bom before frontmatter",
			input: "\ufeff---\ntitle: BOM Doc\nstatus: draft\n---\n\nBody",
			wantFM: Frontmatter{
				Title:  "BOM Doc",
				Status: "draft",
			},
			wantBody: "Body",
		},
		{
			name:     "utf-8 bom without frontmatter",
			input:    "\ufeff## Just Markdown",
			wantFM:   Frontmatter{},
			wantBody: "## Just Markdown",
		},
		{
			name:     "invalid yaml preserves body",
			input:    "---\ntitle: [broken\nstatus: draft\n---\n\nBody survives",
			wantFM:   Frontmatter{},
			wantBody: "Body survives",
			wantErr:  true,
		},
		{
			name:     "invalid yaml closed at EOF",
			input:    "---\ntitle: [broken\n---",
			wantFM:   Frontmatter{},
			wantBody: "",
			wantErr:  true,
		},
		{
			name:     "tab-indented yaml is an error",
			input:    "---\n\ttitle: Tab Doc\n---\n\nBody",
			wantFM:   Frontmatter{},
			wantBody: "Body",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fm, body, err := SplitDocument([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if fm.Title != tt.wantFM.Title {
				t.Errorf("title = %q, want %q", fm.Title, tt.wantFM.Title)
			}
			if fm.Status != tt.wantFM.Status {
				t.Errorf("status = %q, want %q", fm.Status, tt.wantFM.Status)
			}
			if !reflect.DeepEqual(fm.Tags, tt.wantFM.Tags) {
				t.Errorf("tags = %v, want %v", fm.Tags, tt.wantFM.Tags)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestExtractDocType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "normal case", filename: "use-postgres.adr.md", want: "adr"},
		{name: "multi-dot slug", filename: "my-feature.v2.adr.md", want: "adr"},
		{name: "no type segment", filename: "readme.md", want: ""},
		{name: "no .md suffix", filename: "readme", want: ""},
		{name: "degenerate", filename: ".md", want: ""},
		{name: "empty string", filename: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractDocType(tt.filename)
			if got != tt.want {
				t.Errorf("ExtractDocType(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "normal", filename: "use-postgres.adr.md", want: "use-postgres"},
		{name: "multi-dot slug preserves dots", filename: "my-feature.v2.adr.md", want: "my-feature.v2"},
		{name: "no type segment", filename: "readme.md", want: "readme"},
		{name: "no .md", filename: "readme", want: "readme"},
		{name: "degenerate", filename: ".md", want: ""},
		{name: "empty", filename: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractSlug(tt.filename)
			if got != tt.want {
				t.Errorf("ExtractSlug(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsValidType(t *testing.T) {
	t.Parallel()
	for _, typ := range ValidTypes() {
		t.Run("valid/"+typ, func(t *testing.T) {
			t.Parallel()
			if !IsValidType(typ) {
				t.Errorf("IsValidType(%q) = false, want true", typ)
			}
		})
	}
	invalid := []string{"", "unknown", "ADR", "Adr", "task_type", "task type"}
	for _, v := range invalid {
		t.Run("invalid/"+v, func(t *testing.T) {
			t.Parallel()
			if IsValidType(v) {
				t.Errorf("IsValidType(%q) = true, want false", v)
			}
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	t.Parallel()
	for _, s := range []DocStatus{StatusDraft, StatusAccepted, StatusRejected} {
		t.Run("valid/"+string(s), func(t *testing.T) {
			t.Parallel()
			if !IsValidStatus(s) {
				t.Errorf("IsValidStatus(%q) = false, want true", s)
			}
		})
	}
	invalid := []DocStatus{"", "Draft", "ACCEPTED", "pending", "active"}
	for _, v := range invalid {
		t.Run("invalid/"+string(v), func(t *testing.T) {
			t.Parallel()
			if IsValidStatus(v) {
				t.Errorf("IsValidStatus(%q) = true, want false", v)
			}
		})
	}
}

func TestCategoryForType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		docType DocumentType
		wantCat Category
	}{
		{TypePRD, CategoryVision},
		{TypeIdea, CategoryVision},
		{TypePlan, CategoryVision},
		{TypeRnD, CategoryVision},
		{TypeMRD, CategoryVision},
		{TypeBRD, CategoryVision},
		{TypeURD, CategoryVision},
		{TypeBRS, CategoryVision},
		{TypeStRS, CategoryVision},
		{TypeSyRS, CategoryVision},
		{TypeSRS, CategoryVision},
		{TypeADR, CategoryKnowledge},
		{TypeRFC, CategoryKnowledge},
		{TypeRule, CategoryKnowledge},
		{TypeGuide, CategoryKnowledge},
		{TypeDoc, CategoryKnowledge},
		{TypeSpec, CategoryKnowledge},
		{TypeTaskType, CategoryExperience},
		{TypeCPAT, CategoryExperience},
		{DocumentType("unknown"), CategoryKnowledge},
	}
	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			t.Parallel()
			got := CategoryForType(tt.docType)
			if got != tt.wantCat {
				t.Errorf("CategoryForType(%q) = %q, want %q", tt.docType, got, tt.wantCat)
			}
		})
	}
}

func TestValidTypes_Completeness(t *testing.T) {
	t.Parallel()
	types := ValidTypes()
	seen := map[string]bool{}
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate type %q in ValidTypes()", typ)
		}
		seen[typ] = true
		if !IsValidType(typ) {
			t.Errorf("IsValidType(%q) = false for entry from ValidTypes()", typ)
		}
	}
	if len(types) != len(categoryMap) {
		t.Errorf("ValidTypes() count = %d, categoryMap size = %d; they must match", len(types), len(categoryMap))
	}
}

// TestProseProfiles_Completeness pins what the ProseProfiles doc comment claims:
// every type carries a profile. A missing entry reads as the zero profile, which
// matches neither half, so the type silently falls out of every profile-driven
// check instead of failing anywhere.
func TestProseProfiles_Completeness(t *testing.T) {
	t.Parallel()

	for _, typ := range ValidTypes() {
		profile, ok := ProseProfiles[DocumentType(typ)]
		if !ok {
			t.Errorf("ProseProfiles has no entry for %q", typ)
			continue
		}
		if profile != ProfileSTE && profile != ProfileISO {
			t.Errorf("ProseProfiles[%q] = %q, want %q or %q", typ, profile, ProfileSTE, ProfileISO)
		}
	}
	if len(ProseProfiles) != len(ValidTypes()) {
		t.Errorf("ProseProfiles size = %d, ValidTypes() count = %d; they must match",
			len(ProseProfiles), len(ValidTypes()))
	}
}

// TestClauseAndStepSections_NameRealHeadings pins the cheaper half of the
// table/template agreement: a section rule must name a heading the type's own
// template emits. The engine-side test (advisory.TestSectionTables_MatchTemplates)
// pins the other half — that the heading holds numbered items.
func TestClauseAndStepSections_NameRealHeadings(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, label string, table map[DocumentType][]SectionRule) {
		t.Helper()
		for docType, rules := range table {
			body := GenerateTemplate(docType)
			for _, rule := range rules {
				if !strings.Contains(body, "## "+rule.Name) {
					t.Errorf("%s[%q] names %q, which the %s template does not emit",
						label, docType, rule.Name, docType)
				}
			}
		}
	}
	check(t, "ClauseSections", ClauseSections)
	check(t, "StepSections", StepSections)
}

func TestWalkArchcoreFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files := []struct {
		relPath string
		dir     bool
	}{
		{"use-postgres.adr.md", false},
		{"sub", true},
		{"sub/another.rfc.md", false},
		{".hidden", true},
		{".hidden/secret.md", false},
		{"settings.json", false},
		{".sync-state.json", false},
		{"README.txt", false},
	}
	for _, f := range files {
		full := filepath.Join(dir, f.relPath)
		if f.dir {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatalf("MkdirAll %s: %v", full, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatalf("MkdirAll parent of %s: %v", full, err)
			}
			if err := os.WriteFile(full, []byte("content"), 0o644); err != nil {
				t.Fatalf("WriteFile %s: %v", full, err)
			}
		}
	}
	var collected []string
	err := WalkArchcoreFiles(dir, func(path string, d fs.DirEntry) error {
		collected = append(collected, d.Name())
		return nil
	})
	if err != nil {
		t.Fatalf("WalkArchcoreFiles returned error: %v", err)
	}
	sort.Strings(collected)
	want := []string{"another.rfc.md", "use-postgres.adr.md"}
	if !reflect.DeepEqual(collected, want) {
		t.Errorf("collected = %v, want %v", collected, want)
	}
}

func TestWalkArchcoreFiles_NonExistentDir(t *testing.T) {
	t.Parallel()
	err := WalkArchcoreFiles("/nonexistent/path/that/does/not/exist", func(path string, d fs.DirEntry) error {
		return nil
	})
	if err != nil {
		t.Errorf("WalkArchcoreFiles on non-existent dir returned error: %v", err)
	}
}

func TestWalkArchcoreFiles_CallbackError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	docPath := filepath.Join(dir, "my-doc.adr.md")
	if err := os.WriteFile(docPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	sentinel := errors.New("sentinel callback error")
	err := WalkArchcoreFiles(dir, func(path string, d fs.DirEntry) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

// TestWalkArchcoreFilesSkipping_SkipsSymlinks guards the single walk choke point
// used by list_documents, search_documents, status, and sync: a symlinked .md
// entry (to a file or a directory) that points outside the tree must never be
// visited, so nothing outside .archcore/ is ever read, hashed, or listed.
func TestWalkArchcoreFilesSkipping_SkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not portable to Windows")
	}
	dir := t.TempDir()
	real := filepath.Join(dir, "real.adr.md")
	if err := os.WriteFile(real, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// A symlinked .md file pointing at a secret outside the tree.
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.adr.md")
	if err := os.WriteFile(secret, []byte("TOP-SECRET"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "leak.adr.md")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	// A symlinked directory that would otherwise be descended into.
	if err := os.Symlink(outside, filepath.Join(dir, "linkdir")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	var visited []string
	err := WalkArchcoreFilesSkipping(dir, nil, func(path string, d fs.DirEntry) error {
		visited = append(visited, filepath.Base(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(visited) != 1 || visited[0] != "real.adr.md" {
		t.Fatalf("visited = %v, want only [real.adr.md] (symlinks must be skipped)", visited)
	}
}
