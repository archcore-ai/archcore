package templates

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
			wantContains: []string{"## Overview", "## Prerequisites", "## Steps", "### Step 1:", "### Step 2:", "### Step 3:", "## Common Issues", "## Verification"},
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
			wantContains: []string{"## Vision", "## Problem Statement", "## Goals and Success Metrics", "## Requirements", "## Constraints", "## Timeline"},
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
			wantContains: []string{"## Purpose", "## Scope", "## Authority", "## Subject", "## Contract Surface", "## Normative Behavior", "## Constraints", "## Invariants", "## Error Handling", "## Conformance"},
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

	if !strings.Contains(template, "Describe") {
		t.Error("ADR template should include guidance text")
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
		"### Step 1:",
		"### Step 2:",
		"### Step 3:",
		"### Step 4:",
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

	stepCount := strings.Count(template, "### Step")
	if stepCount < 4 {
		t.Errorf("Guide template should have at least 4 steps, got %d", stepCount)
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
		"## Purpose",
		"## Scope",
		"## Authority",
		"## Subject",
		"## Definitions",
		"## Contract Surface",
		"### Interfaces",
		"### Inputs",
		"### Outputs",
		"## Normative Behavior",
		"### Preconditions",
		"### Postconditions",
		"## Constraints",
		"## Invariants",
		"## Error Handling",
		"### Failure Semantics",
		"## Conformance",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Spec template missing section: %q", section)
		}
	}

	// Spec must contain code blocks for conformance-critical examples.
	codeBlockCount := strings.Count(template, "```")
	if codeBlockCount%2 != 0 {
		t.Errorf("code block markers = %d, should be even", codeBlockCount)
	}
	if codeBlockCount < 2 {
		t.Errorf("Spec template should have at least 1 code block pair, got %d markers", codeBlockCount)
	}

	// Spec must contain tables for definitions, inputs, outputs, constraints, error handling.
	pipeCount := strings.Count(template, "|")
	if pipeCount < 30 {
		t.Errorf("Spec template should have substantial table content, got %d pipe characters", pipeCount)
	}

	// Spec must contain RFC 2119 normative keywords.
	for _, keyword := range []string{"MUST", "SHOULD", "MAY"} {
		if !strings.Contains(template, keyword) {
			t.Errorf("Spec template missing RFC 2119 keyword: %q", keyword)
		}
	}

	// Optional sections must be marked with "Include only if" guidance.
	optionalMarkers := strings.Count(template, "Include only if")
	if optionalMarkers < 3 {
		t.Errorf("Spec template should mark optional sections, got %d 'Include only if' markers", optionalMarkers)
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
		"### Product Vision Statement",
		"## Problem Statement",
		"### Target Users",
		"### User Stories",
		"## Goals and Success Metrics",
		"## Requirements",
		"### Functional Requirements",
		"### Non-Functional Requirements",
		"## Constraints",
		"## Solution Overview",
		"## Risks and Mitigations",
		"## Timeline",
		"## Open Questions",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("PRD template missing section: %q", section)
		}
	}

	if !strings.Contains(template, "|") {
		t.Error("PRD template should include tables")
	}

	if !strings.Contains(template, "P0") {
		t.Error("PRD template should include priority levels")
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
		{TypeSpec, 2670},
		{TypeTaskType, 252},
		{TypeCPAT, 198},
		{TypePRD, 1925},
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

	if tableCount < 50 {
		t.Errorf("PRD table elements = %d, should have at least 50", tableCount)
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

	// Verify checklist format
	checkboxCount := strings.Count(template, "- [ ]")
	if checkboxCount < 4 {
		t.Errorf("Plan template should have at least 4 task checkboxes, got %d", checkboxCount)
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
	for _, s := range ValidStatuses() {
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

func TestValidStatuses(t *testing.T) {
	t.Parallel()
	want := []DocStatus{StatusDraft, StatusAccepted, StatusRejected}
	got := ValidStatuses()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ValidStatuses() = %v, want %v", got, want)
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

func TestTypesByCategory(t *testing.T) {
	t.Parallel()
	byCategory := TypesByCategory()
	knownCategories := map[Category]bool{
		CategoryVision:     true,
		CategoryKnowledge:  true,
		CategoryExperience: true,
	}
	for cat := range byCategory {
		if !knownCategories[cat] {
			t.Errorf("unexpected category key %q in TypesByCategory result", cat)
		}
	}
	seen := map[string]Category{}
	for cat, types := range byCategory {
		for _, typ := range types {
			if prev, exists := seen[typ]; exists {
				t.Errorf("type %q appears in both %q and %q", typ, prev, cat)
			}
			seen[typ] = cat
		}
	}
	all := ValidTypes()
	for _, typ := range all {
		if _, found := seen[typ]; !found {
			t.Errorf("type %q from ValidTypes() not found in any TypesByCategory bucket", typ)
		}
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
	byCategory := TypesByCategory()
	total := 0
	for _, bucket := range byCategory {
		total += len(bucket)
	}
	if len(types) != total {
		t.Errorf("ValidTypes() count = %d, TypesByCategory total = %d; they must match", len(types), total)
	}
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
