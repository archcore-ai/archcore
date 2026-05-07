package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type DocumentType string

const (
	TypeADR      DocumentType = "adr"
	TypeRFC      DocumentType = "rfc"
	TypeRule     DocumentType = "rule"
	TypeGuide    DocumentType = "guide"
	TypeDoc      DocumentType = "doc"
	TypeTaskType DocumentType = "task-type"
	TypeCPAT     DocumentType = "cpat"
	TypePRD      DocumentType = "prd"
	TypeIdea     DocumentType = "idea"
	TypePlan     DocumentType = "plan"
	TypeSpec     DocumentType = "spec"
	TypeMRD      DocumentType = "mrd"
	TypeBRD      DocumentType = "brd"
	TypeURD      DocumentType = "urd"
	TypeBRS      DocumentType = "brs"
	TypeStRS     DocumentType = "strs"
	TypeSyRS     DocumentType = "syrs"
	TypeSRS      DocumentType = "srs"
)

const (
	CategoryVision     = "vision"
	CategoryKnowledge  = "knowledge"
	CategoryExperience = "experience"
)

const (
	StatusDraft    = "draft"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)

// TagRe validates a single tag: lowercase alphanumeric with hyphens, underscores, colons, and pipes.
var TagRe = regexp.MustCompile(`^[a-z][a-z0-9_:|-]*$`)

// SlugRe validates a document slug: lowercase alphanumeric segments separated by hyphens.
var SlugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Frontmatter holds the parsed YAML frontmatter fields of a document.
type Frontmatter struct {
	Title  string   `yaml:"title"`
	Status string   `yaml:"status"`
	Tags   []string `yaml:"tags"`
}

// SkipFiles are non-document meta files that live in .archcore/ and should be
// skipped during scanning, validation, and sync operations.
var SkipFiles = map[string]bool{
	"settings.json":    true,
	".sync-state.json": true,
}

// ValidStatuses returns all valid document status strings.
func ValidStatuses() []string {
	return []string{StatusDraft, StatusAccepted, StatusRejected}
}

// IsValidStatus checks whether the given string is a valid document status.
func IsValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusAccepted, StatusRejected:
		return true
	}
	return false
}

var categoryMap = map[DocumentType]string{
	TypePRD:  CategoryVision,
	TypeIdea: CategoryVision,
	TypePlan: CategoryVision,
	TypeMRD:  CategoryVision,
	TypeBRD:  CategoryVision,
	TypeURD:  CategoryVision,
	TypeBRS:  CategoryVision,
	TypeStRS: CategoryVision,
	TypeSyRS: CategoryVision,
	TypeSRS:  CategoryVision,

	TypeADR:     CategoryKnowledge,
	TypeRFC:     CategoryKnowledge,
	TypeRule:    CategoryKnowledge,
	TypeGuide:   CategoryKnowledge,
	TypeDoc:     CategoryKnowledge,
	TypeSpec:    CategoryKnowledge,

	TypeTaskType: CategoryExperience,
	TypeCPAT:     CategoryExperience,
}

// CategoryForType returns the category directory for a document type.
func CategoryForType(docType DocumentType) string {
	if cat, ok := categoryMap[docType]; ok {
		return cat
	}
	return CategoryKnowledge
}

// ValidTypes returns all valid document type strings.
func ValidTypes() []string {
	return []string{
		string(TypeADR),
		string(TypeRFC),
		string(TypeRule),
		string(TypeGuide),
		string(TypeDoc),
		string(TypeSpec),
		string(TypeTaskType),
		string(TypeCPAT),
		string(TypePRD),
		string(TypeIdea),
		string(TypePlan),
		string(TypeMRD),
		string(TypeBRD),
		string(TypeURD),
		string(TypeBRS),
		string(TypeStRS),
		string(TypeSyRS),
		string(TypeSRS),
	}
}

// TypesByCategory returns types grouped by category.
func TypesByCategory() map[string][]string {
	result := map[string][]string{}
	for dt, cat := range categoryMap {
		result[cat] = append(result[cat], string(dt))
	}
	return result
}

// IsValidType checks whether the given string is a valid document type.
func IsValidType(t string) bool {
	_, ok := categoryMap[DocumentType(t)]
	return ok
}

// ExtractDocType extracts the type from a filename like "use-postgres.adr.md".
func ExtractDocType(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// ExtractSlug extracts the slug from a filename like "use-postgres.adr.md".
func ExtractSlug(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], ".")
	}
	return name
}

// SplitDocument splits raw document bytes into frontmatter fields and body.
// It returns a Frontmatter struct and the markdown body after the closing "---".
func SplitDocument(data []byte) (Frontmatter, string) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return Frontmatter{}, s
	}

	end := strings.Index(s[4:], "\n---\n")
	if end == -1 {
		// Handle frontmatter closed by "\n---" at EOF (no trailing newline).
		if idx := strings.Index(s[4:], "\n---"); idx != -1 && idx+4+len("\n---") == len(s) {
			fmContent := s[4 : 4+idx]
			var fm Frontmatter
			_ = yaml.Unmarshal([]byte(fmContent), &fm)
			return fm, ""
		}
		return Frontmatter{}, s
	}
	end += 4 // adjust for the offset

	fmContent := s[4:end]
	var fm Frontmatter
	_ = yaml.Unmarshal([]byte(fmContent), &fm)

	body := s[end+5:] // skip past "\n---\n"
	body = strings.TrimPrefix(body, "\n")

	return fm, body
}

// WalkArchcoreFiles walks archcoreDir recursively, calling fn for each .md
// document file found. It skips hidden directories, non-.md files, and known
// meta files (settings.json, .sync-state.json).
func WalkArchcoreFiles(archcoreDir string, fn func(path string, d fs.DirEntry) error) error {
	return filepath.WalkDir(archcoreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		name := d.Name()

		// Skip hidden directories (but not .archcore itself).
		if d.IsDir() && strings.HasPrefix(name, ".") && path != archcoreDir {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		// Skip non-.md files and known meta files.
		if !strings.HasSuffix(name, ".md") || SkipFiles[name] {
			return nil
		}

		return fn(path, d)
	})
}

func GenerateTemplate(documentType DocumentType) string {
	switch documentType {
	case TypeADR:
		return generateADRTemplate()
	case TypeRFC:
		return generateRFCTemplate()
	case TypeRule:
		return generateRuleTemplate()
	case TypeGuide:
		return generateGuideTemplate()
	case TypeDoc:
		return generateDocTemplate()
	case TypeSpec:
		return generateSpecTemplate()
	case TypeTaskType:
		return generateTaskTypeTemplate()
	case TypeCPAT:
		return generateCPATTemplate()
	case TypePRD:
		return generatePRDTemplate()
	case TypeIdea:
		return generateIdeaTemplate()
	case TypePlan:
		return generatePlanTemplate()
	case TypeMRD:
		return generateMRDTemplate()
	case TypeBRD:
		return generateBRDTemplate()
	case TypeURD:
		return generateURDTemplate()
	case TypeBRS:
		return generateBRSTemplate()
	case TypeStRS:
		return generateStRSTemplate()
	case TypeSyRS:
		return generateSyRSTemplate()
	case TypeSRS:
		return generateSRSTemplate()
	default:
		return generateDocTemplate()
	}
}

func generateADRTemplate() string {
	return `## Context

Describe the context and problem statement that motivates this decision.

### Current State

- What is the current situation?
- What constraints exist?
- What pain points are we experiencing?

### Problem Statement

Clear, concise description of the problem that needs to be solved.

## Decision

State the decision that was made.

### Rationale

Explain why this decision was chosen over alternatives:

- Key factors that influenced the decision
- Trade-offs that were considered
- Assumptions that were made

## Alternatives Considered

### Alternative 1: [Name]

- Description of the alternative
- Why it was not chosen

### Alternative 2: [Name]

- Description of the alternative
- Why it was not chosen

## Consequences

### Positive

- Benefit 1: Description
- Benefit 2: Description
- Benefit 3: Description

### Negative

- Trade-off 1: Description and mitigation
- Trade-off 2: Description and mitigation

### Risks

- Risk 1: Description and mitigation strategy
- Risk 2: Description and mitigation strategy

## Implementation Notes

Key implementation considerations:

- Migration path (if applicable)
- Dependencies affected
- Timeline considerations
`
}

func generateRFCTemplate() string {
	return `## Summary

One-paragraph summary of the proposal.

## Motivation

### Problem Statement

What problem does this proposal solve? Be specific about the pain points.

### Goals

- Goal 1: Description
- Goal 2: Description

### Non-Goals

What is explicitly out of scope:

- Non-goal 1: Why it's excluded
- Non-goal 2: Why it's excluded

## Detailed Design

### Overview

High-level description of the proposed solution.

### Architecture

Describe the architectural changes:

- Components affected
- New components introduced
- Data flow changes

### API Changes

Describe any API changes:

` + "```" + `
// Example API or interface changes
` + "```" + `

### Data Model

Describe data model changes if applicable:

- New entities
- Schema changes
- Migration requirements

### Implementation Details

Step-by-step implementation approach:

1. Phase 1: Description
2. Phase 2: Description
3. Phase 3: Description

## Drawbacks

### Technical Drawbacks

- Drawback 1: Description and severity
- Drawback 2: Description and severity

### Operational Drawbacks

- Drawback 1: Description
- Drawback 2: Description

## Alternatives

### Alternative 1: [Name]

- Description of approach
- Pros and cons
- Why not chosen

### Alternative 2: [Name]

- Description of approach
- Pros and cons
- Why not chosen

## Security Considerations

- Security implication 1
- Security implication 2
- Mitigation strategies

## Testing Strategy

- Unit testing approach
- Integration testing approach
- Performance testing requirements

## Rollout Plan

### Phases

1. Phase 1: Description and success criteria
2. Phase 2: Description and success criteria
3. Phase 3: Description and success criteria

### Rollback Plan

How to rollback if issues are discovered.

## Unresolved Questions

- Question 1: Context and potential answers
- Question 2: Context and potential answers

## Implementation Plan

- [ ] Step 1: Description
- [ ] Step 2: Description
- [ ] Step 3: Description
- [ ] Step 4: Description
- [ ] Step 5: Description
`
}

func generateRuleTemplate() string {
	return `## Description

Brief description of what this rule covers and why it exists.

## Rule

State the rule clearly as imperative statements:

1. [Rule as imperative statement]
2. [Rule as imperative statement]
3. [Rule as imperative statement]

## Rationale

Why this rule exists and what problems it prevents.

## Examples

### Good

` + "```" + `
// Example of correct usage
` + "```" + `

` + "```" + `
// Another example of correct usage
` + "```" + `

### Bad

` + "```" + `
// Example of incorrect usage
` + "```" + `

` + "```" + `
// Another example of incorrect usage
` + "```" + `

## Exceptions

- Exception 1: When this rule does not apply
- Exception 2: When this rule does not apply

## Enforcement

How this rule is enforced:

- Enforcement method 1: Description
- Enforcement method 2: Description
`
}

func generateGuideTemplate() string {
	return `## Overview

Brief overview of what this guide covers and what the reader will accomplish.

### Target Audience

Who should read this guide:

- Role 1: What they'll learn
- Role 2: What they'll learn

### Time Estimate

Approximate time to complete: X minutes

## Prerequisites

### Required Knowledge

- Prerequisite 1: Brief description
- Prerequisite 2: Brief description

### Required Tools

- Tool 1: Version and installation link
- Tool 2: Version and installation link

### Required Access

- Access 1: How to obtain
- Access 2: How to obtain

## Steps

### Step 1: [Title]

Description of what this step accomplishes.

` + "```" + `
# Commands or code for this step
` + "```" + `

**Expected result:** What you should see after this step.

### Step 2: [Title]

Description of what this step accomplishes.

` + "```" + `
# Commands or code for this step
` + "```" + `

**Expected result:** What you should see after this step.

### Step 3: [Title]

Description of what this step accomplishes.

` + "```" + `
# Commands or code for this step
` + "```" + `

**Expected result:** What you should see after this step.

### Step 4: [Title]

Description of what this step accomplishes.

**Expected result:** What you should see after this step.

## Verification

How to verify everything is working correctly:

1. Verification step 1
2. Verification step 2
3. Verification step 3

## Common Issues

### Issue 1: [Error message or symptom]

**Cause:** Why this happens

**Solution:**

` + "```" + `
# Commands to fix
` + "```" + `

### Issue 2: [Error message or symptom]

**Cause:** Why this happens

**Solution:** Steps to resolve

### Issue 3: [Error message or symptom]

**Cause:** Why this happens

**Solution:** Steps to resolve

## Next Steps

What to do after completing this guide:

- Next step 1: Link or description
- Next step 2: Link or description
`
}

func generateDocTemplate() string {
	return `## Overview

Brief description of what this document covers and its purpose.

### Scope

What this document includes:

- Topic 1
- Topic 2
- Topic 3

What this document does not cover:

- Out of scope 1
- Out of scope 2

## Content

### Section 1: [Title]

Main content for this section.

Key points:

- Point 1: Description
- Point 2: Description
- Point 3: Description

### Section 2: [Title]

Main content for this section.

Key points:

- Point 1: Description
- Point 2: Description

### Section 3: [Title]

Main content for this section.

## Examples

### Example 1: [Title]

Context for when to use this example.

` + "```" + `
// Code or configuration example
` + "```" + `

### Example 2: [Title]

Context for when to use this example.

` + "```" + `
// Code or configuration example
` + "```" + `

## Best Practices

- Best practice 1: Description
- Best practice 2: Description
- Best practice 3: Description

## FAQ

### Q: Common question 1?

Answer to the question.

### Q: Common question 2?

Answer to the question.
`
}

func generateTaskTypeTemplate() string {
	return `## What

What this typical task covers and what the end result looks like.

## When to Use

Use when:

- Condition 1
- Condition 2

Do NOT use when:

- Condition: use [alternative] instead

## Steps

1. Step one — what to do and where (@path/to/file)
2. Step two — what to do next
3. Step three — final checks

## Example

` + "```" + `
// Small code snippet or @-reference to a real implementation
` + "```" + `

## Things to Watch Out For

- Pitfall or gotcha 1
- Edge case to keep in mind
- Common mistake to avoid
`
}

func generateCPATTemplate() string {
	return `## What Changed

The pattern, convention, or approach that changed.

## Why

What problem the old way caused and why the change was needed.

## Before

` + "```" + `
// Old pattern
` + "```" + `

## After

` + "```" + `
// New pattern
` + "```" + `

## Scope

Affected files and modules:

- @path/to/affected/module
- @path/to/another/file

## Notes

- Exceptions where the old pattern is still acceptable
- Migration notes or timeline
`
}

func generatePRDTemplate() string {
	return `## Vision

### Product Vision Statement

One-sentence vision of what this product/feature will achieve.

### Strategic Alignment

How this aligns with company/team goals:

- Strategic goal 1: How this product supports it
- Strategic goal 2: How this product supports it

## Problem Statement

### Current State

Describe the current situation and its limitations:

- Pain point 1: Description and impact
- Pain point 2: Description and impact
- Pain point 3: Description and impact

### Target Users

| User Segment | Description | Key Needs |
|--------------|-------------|-----------|
| Segment 1 | Description | Needs |
| Segment 2 | Description | Needs |

### User Stories

As a [user type], I want [goal] so that [benefit].

- User story 1
- User story 2
- User story 3

## Goals and Success Metrics

### Goals

| Goal | Description | Priority |
|------|-------------|----------|
| Goal 1 | Description | P0/P1/P2 |
| Goal 2 | Description | P0/P1/P2 |
| Goal 3 | Description | P0/P1/P2 |

### Success Metrics

| Metric | Current | Target | Timeline |
|--------|---------|--------|----------|
| Metric 1 | Value | Value | Date |
| Metric 2 | Value | Value | Date |

### Non-Goals

Explicitly out of scope for this version:

- Non-goal 1: Reason
- Non-goal 2: Reason

## Requirements

### Functional Requirements

#### P0 (Must Have)

| ID | Requirement | Acceptance Criteria |
|----|-------------|---------------------|
| FR-001 | Description | Criteria |
| FR-002 | Description | Criteria |
| FR-003 | Description | Criteria |

#### P1 (Should Have)

| ID | Requirement | Acceptance Criteria |
|----|-------------|---------------------|
| FR-101 | Description | Criteria |
| FR-102 | Description | Criteria |

#### P2 (Nice to Have)

| ID | Requirement | Acceptance Criteria |
|----|-------------|---------------------|
| FR-201 | Description | Criteria |

### Non-Functional Requirements

| Category | Requirement | Target |
|----------|-------------|--------|
| Performance | Description | Metric |
| Scalability | Description | Metric |
| Security | Description | Metric |
| Reliability | Description | Metric |

## Constraints

### Technical Constraints

- Constraint 1: Description and impact
- Constraint 2: Description and impact

### Business Constraints

- Constraint 1: Description and impact
- Constraint 2: Description and impact

### Dependencies

| Dependency | Type | Owner | Status |
|------------|------|-------|--------|
| Dependency 1 | Internal/External | Team | Status |
| Dependency 2 | Internal/External | Team | Status |

## Solution Overview

### Proposed Approach

High-level description of the proposed solution.

### Key Components

- Component 1: Purpose
- Component 2: Purpose
- Component 3: Purpose

### User Experience

Key UX considerations:

- UX consideration 1
- UX consideration 2

### Technical Considerations

Key technical considerations (defer details to RFC/design doc):

- Consideration 1
- Consideration 2

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Risk 1 | High/Med/Low | High/Med/Low | Strategy |
| Risk 2 | High/Med/Low | High/Med/Low | Strategy |

## Timeline

### Milestones

| Milestone | Target Date | Description |
|-----------|-------------|-------------|
| Milestone 1 | Date | Description |
| Milestone 2 | Date | Description |
| Milestone 3 | Date | Description |

### Phases

- Phase 1: Scope and timeline
- Phase 2: Scope and timeline
- Phase 3: Scope and timeline

## Open Questions

| Question | Context | Decision Owner | Due Date |
|----------|---------|----------------|----------|
| Question 1 | Context | Owner | Date |
| Question 2 | Context | Owner | Date |

## Appendix

### Glossary

| Term | Definition |
|------|------------|
| Term 1 | Definition |
| Term 2 | Definition |

### References

- Reference 1: Link
- Reference 2: Link
`
}

func generateIdeaTemplate() string {
	return `## Idea

Describe the core idea in 2-3 sentences.

### Problem / Opportunity

- What problem does it solve?
- What opportunity does it open?

## Value

### For Users

### For Business

### For Team

## Possible Implementation

### Technical Approach

### Integrations

## Risks and Constraints

### Potential Risks

### Known Constraints

## Next Steps

- [ ] Step 1
- [ ] Step 2
`
}

func generatePlanTemplate() string {
	return `## Goal

Describe the desired outcome in one sentence.

### Context

- What motivated creating this plan?

## Tasks

### Phase 1: [Name]

- [ ] Task 1
- [ ] Task 2

### Phase 2: [Name]

- [ ] Task 1

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Dependencies

| Dependency | Type | Status |
|------------|------|--------|

## Notes
`
}

func generateSpecTemplate() string {
	return `## Purpose

This specification defines the canonical technical contract for [subject].

It is normative for:
- [implementation(s), service(s), component(s)]
- [consumers, producers, integrators]
- [test and validation scope]

## Scope

### Covers

- [What this specification defines]
- [Boundaries of the contract]
- [Actors, systems, or interfaces in scope]

### Does Not Cover

- [Explicitly out of scope]
- [Adjacent concerns covered elsewhere]
- [Operational or implementation details not governed by this spec]

## Authority

This document is the normative specification for [subject].

If implementation, tests, or operational behavior differ from this specification, this specification takes precedence until it is amended.

## Subject

- **Name**: [Canonical name]
- **Kind**: [service | component | interface | schema | protocol]
- **Primary responsibility**: [single sentence]
- **Consumers / dependents**: [who relies on this contract]

## Definitions

Only include terms used normatively in this document.

| Term | Definition |
|------|------------|
| [Term 1] | [Precise definition] |
| [Term 2] | [Precise definition] |

## Contract Surface

Define the externally observable contract using architectural descriptions and identifier-based references. Reference where things are defined rather than copying their full definitions.

Prefer:
- File path references using @-notation: @path/to/types.ts, @internal/config/config.go
- Named identifiers: interface AIChatProps, type MessageRole, func ValidateConfig
- Prose or table descriptions of shape and semantics

Reserve code blocks for wire-level or protocol-level contracts only — where the exact textual format is itself the normative artifact (e.g., HTTP endpoint shape, CLI flag grammar, binary frame format). Do NOT copy full interface, type, or struct definitions from source files — they will become stale when the source changes.

### Interfaces

For each externally observable interface, state:
- The canonical identifier name and where it is defined (@path/to/file)
- What it represents and who consumes it
- The semantically significant fields or parameters (prose or table)

Use a code block ONLY when the exact textual format of the boundary is itself normative (e.g., HTTP endpoint, wire message structure).

### Inputs

| Input | Type | Description | Required |
|-------|------|-------------|----------|
| [input] | [type] | [meaning] | [yes/no] |

### Outputs

| Output | Type | Description |
|--------|------|-------------|
| [output] | [type] | [meaning] |

## Normative Behavior

Use RFC 2119 terms: MUST, MUST NOT, SHOULD, SHOULD NOT, MAY.

1. The system MUST [required behavior].
2. The system MUST [required behavior].
3. The system MUST NOT [forbidden behavior].
4. The system SHOULD [recommended behavior].
5. The system MAY [optional behavior].

### Preconditions

- [Condition that must be true before behavior applies]
- [Required inputs or environmental assumptions]

### Postconditions

- [Condition that must be true after successful processing]
- [Observable outcome guaranteed by the system]

## State Model

Include only if the subject is stateful.

### States

| State | Meaning |
|-------|---------|
| [state] | [definition] |

### State Transitions

| Current State | Event | Next State | Side Effects |
|---------------|-------|------------|--------------|
| [state] | [event] | [state] | [effects] |

## Constraints

| Constraint | Value | Rationale |
|------------|-------|-----------|
| [e.g., Max payload size] | [e.g., 1 MB] | [why] |
| [e.g., Rate limit] | [e.g., 100 req/s] | [why] |
| [e.g., Max processing time] | [e.g., 200 ms p95] | [why] |

## Invariants

These conditions must always hold.

- [Condition that must always hold]
- [Condition that must always hold]
- [Condition that must always hold]

## Error Handling

| Condition | Response | Recovery |
|-----------|----------|----------|
| [error condition] | [error code / message / behavior] | [recovery action] |
| [error condition] | [error code / message / behavior] | [recovery action] |

### Failure Semantics

- [What is retriable vs non-retriable]
- [Whether processing is atomic, partial, idempotent, eventually consistent, etc.]
- [Timeout, cancellation, and duplicate handling semantics]

## Conformance

An implementation conforms to this specification if it satisfies:

- all MUST and MUST NOT requirements
- all stated invariants
- all applicable interface requirements
- all applicable error-handling requirements
- all required state transition rules, if a state model is defined

## Examples

Include only examples that clarify conformance-critical behavior.

### Example: [Scenario Name]

` + "```" + `txt
// Input
[example input]

// Output
[expected output]

// Notes
[why this example matters]
` + "```" + `

## Security Considerations

Include only if relevant.

- [Authentication / authorization expectations]
- [Data handling constraints]
- [Trust boundaries]
- [Abuse / misuse considerations]

## Privacy Considerations

Include only if relevant.

- [Data classification]
- [Retention or minimization constraints]
- [PII / sensitive data handling expectations]

## Compatibility

Include only if relevant.

- [Backward compatibility guarantees]
- [Forward compatibility assumptions]
- [Version negotiation behavior]
- [Deprecation policy for consumers]

## Migration Notes

Include only for breaking or behaviorally significant changes.

- [Breaking changes]
- [Required migration steps]
- [Temporary compatibility behavior]
`
}

func generateMRDTemplate() string {
	return `## Market Landscape

### Industry Trends

- Trend 1: Description and relevance
- Trend 2: Description and relevance

### Market Size

- Total market size and growth rate
- Key segments and their dynamics

### Market Dynamics

- Key drivers accelerating the market
- Key inhibitors or headwinds

## TAM / SAM / SOM

| Metric | Value | Methodology |
|--------|-------|-------------|
| TAM (Total Addressable Market) | $ | Top-down / bottom-up |
| SAM (Serviceable Addressable Market) | $ | Segmentation criteria |
| SOM (Serviceable Obtainable Market) | $ | Realistic capture rate |

### Assumptions

- Assumption 1: Basis for estimate
- Assumption 2: Basis for estimate

## Competitive Analysis

### Competitors

| Competitor | Segment | Strengths | Weaknesses |
|------------|---------|-----------|------------|
| Competitor 1 | Segment | Strengths | Weaknesses |
| Competitor 2 | Segment | Strengths | Weaknesses |

### Positioning

Where the opportunity sits relative to competitors.

### Differentiation

Key differentiators that create defensible advantage:

- Differentiator 1: Description
- Differentiator 2: Description

## Market Needs

### Pain Points

| Pain Point | Severity | Affected Segment | Current Workaround |
|------------|----------|------------------|--------------------|
| Pain point 1 | High/Med/Low | Segment | Workaround |
| Pain point 2 | High/Med/Low | Segment | Workaround |

### Unmet Needs

- Unmet need 1: Why it remains unmet
- Unmet need 2: Why it remains unmet

### Opportunities

- Opportunity 1: How it connects to market needs
- Opportunity 2: How it connects to market needs

## Opportunity and Timing

### Market Window

- Why now? What has changed to create this opportunity?
- What is the window of opportunity?

### Urgency

- Competitive pressure or first-mover considerations
- External events creating urgency (regulatory, technology shifts)

### Market Readiness

- Customer readiness to adopt
- Technology maturity level
- Ecosystem readiness (partners, integrations)

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Risk 1 | High/Med/Low | High/Med/Low | Strategy |
| Risk 2 | High/Med/Low | High/Med/Low | Strategy |

## References

- Market research sources
- Industry reports
- Related documents
`
}

func generateBRDTemplate() string {
	return `## Business Objectives

### Goals

| Goal | Description | Strategic Alignment |
|------|-------------|---------------------|
| Goal 1 | Description | How it supports strategy |
| Goal 2 | Description | How it supports strategy |

### Strategic Alignment

How this initiative aligns with company/division strategy:

- Strategic priority 1: Connection
- Strategic priority 2: Connection

## Stakeholders

### Sponsors and Decision-Makers

| Stakeholder | Role | Interest | Influence |
|-------------|------|----------|-----------|
| Stakeholder 1 | Sponsor / Approver / Advisor | Description | High/Med/Low |
| Stakeholder 2 | Sponsor / Approver / Advisor | Description | High/Med/Low |

### Influence Map

Key relationships and decision dynamics:

- Decision-maker 1: What they care about, what would block approval
- Decision-maker 2: What they care about, what would block approval

## Business Rules and Constraints

### Policies

- Policy 1: Description and impact on initiative
- Policy 2: Description and impact on initiative

### Regulations

- Regulation 1: Compliance requirement and scope
- Regulation 2: Compliance requirement and scope

### Budget

| Category | Estimated Cost | Funding Source |
|----------|---------------|----------------|
| Development | $ | Source |
| Operations | $ | Source |
| Total | $ | |

## Success Metrics and ROI

### KPIs

| KPI | Current Baseline | Target | Timeline |
|-----|-----------------|--------|----------|
| KPI 1 | Value | Value | Date |
| KPI 2 | Value | Value | Date |

### Expected Returns

| Benefit | Quantified Value | Confidence |
|---------|-----------------|------------|
| Revenue impact | $ | High/Med/Low |
| Cost savings | $ | High/Med/Low |
| Efficiency gain | % | High/Med/Low |

### Payback Period

- Initial investment: $
- Expected payback period: X months/years
- Break-even point: Date

## Dependencies

### Organizational Dependencies

- Dependency 1: Team/department and what is needed
- Dependency 2: Team/department and what is needed

### Technical Dependencies

- Dependency 1: System/platform and what is needed
- Dependency 2: System/platform and what is needed

### External Dependencies

- Dependency 1: Vendor/partner and what is needed
- Dependency 2: Vendor/partner and what is needed

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Risk 1 | High/Med/Low | High/Med/Low | Strategy |
| Risk 2 | High/Med/Low | High/Med/Low | Strategy |

## References

- Related business documents
- Strategic plans
- Related documents
`
}

func generateURDTemplate() string {
	return `## User Personas

### Persona 1: [Name]

| Attribute | Description |
|-----------|-------------|
| Role | Description |
| Goals | What they are trying to achieve |
| Pain Points | Current frustrations |
| Context | Environment, tools, constraints |
| Technical Skill | Novice / Intermediate / Expert |

### Persona 2: [Name]

| Attribute | Description |
|-----------|-------------|
| Role | Description |
| Goals | What they are trying to achieve |
| Pain Points | Current frustrations |
| Context | Environment, tools, constraints |
| Technical Skill | Novice / Intermediate / Expert |

## User Journeys

### Current State

How users accomplish their goals today:

1. Step 1: What happens, pain points
2. Step 2: What happens, pain points
3. Step 3: What happens, pain points

### Desired State

How users should accomplish their goals:

1. Step 1: What happens, improvement over current
2. Step 2: What happens, improvement over current
3. Step 3: What happens, improvement over current

### Touchpoints

| Touchpoint | Channel | User Action | System Response |
|------------|---------|-------------|-----------------|
| Touchpoint 1 | Web/Mobile/API | Action | Response |
| Touchpoint 2 | Web/Mobile/API | Action | Response |

## User Requirements

### Functional Needs Per Persona

#### [Persona 1 Name]

| ID | Requirement | Priority | Rationale |
|----|-------------|----------|-----------|
| UR-001 | Description | P0/P1/P2 | Why this matters to the persona |
| UR-002 | Description | P0/P1/P2 | Why this matters to the persona |

#### [Persona 2 Name]

| ID | Requirement | Priority | Rationale |
|----|-------------|----------|-----------|
| UR-101 | Description | P0/P1/P2 | Why this matters to the persona |
| UR-102 | Description | P0/P1/P2 | Why this matters to the persona |

## Usability Requirements

### Accessibility

- Accessibility standard (e.g., WCAG 2.1 AA)
- Specific accessibility requirements

### Learnability

- Time to first productive use: Target
- Training required: Yes/No, scope
- Documentation requirements

### Efficiency

- Task completion time targets
- Number of steps for common tasks
- Error rate targets

## Acceptance Criteria

User-facing validation conditions:

| ID | Criterion | Persona | Verification Method |
|----|-----------|---------|---------------------|
| AC-001 | Given [context], when [action], then [outcome] | Persona | Test / Demo / Survey |
| AC-002 | Given [context], when [action], then [outcome] | Persona | Test / Demo / Survey |

## References

- User research data
- Analytics and usage data
- Related documents
`
}

func generateBRSTemplate() string {
	return `## Business Purpose and Scope

### Purpose

Why this business initiative exists — the business problem or opportunity being addressed.

### Scope

What is included and excluded from this business requirements specification.

## Business Overview

### Stakeholders

| Stakeholder | Role | Interest | Impact |
|-------------|------|----------|--------|
| Stakeholder 1 | Sponsor / Owner / Advisor | Description | High/Med/Low |
| Stakeholder 2 | Sponsor / Owner / Advisor | Description | High/Med/Low |

### Business Environment

- Market context and industry landscape
- Regulatory and compliance landscape
- Technology constraints and opportunities
- Organizational context (capabilities, culture, structure)

### Information Environment

- Key information systems and data sources in the current business context
- Data flows between business units and external parties
- Information management constraints (retention, privacy, sovereignty)

## Mission, Goals and Objectives

| ID | Goal / Objective | Description | Priority | Success Measure |
|----|------------------|-------------|----------|-----------------|
| BG-001 | Goal 1 | Description | High/Med/Low | Measurable outcome |
| BG-002 | Goal 2 | Description | High/Med/Low | Measurable outcome |
| BO-001 | Objective 1 | Description | High/Med/Low | Measurable outcome |
| BO-002 | Objective 2 | Description | High/Med/Low | Measurable outcome |

## Business Operations

### Business Processes

Key business processes affected or created by this initiative:

1. Process 1: Description, current state, desired state
2. Process 2: Description, current state, desired state

### Business Policies and Rules

| ID | Policy / Rule | Description | Source |
|----|---------------|-------------|--------|
| BP-001 | Policy 1 | Description | Regulatory / Internal / Industry |
| BP-002 | Rule 1 | Description | Regulatory / Internal / Industry |

## Business Constraints

| Constraint | Type | Impact | Mitigation |
|------------|------|--------|------------|
| Constraint 1 | Budget / Timeline / Resource / Legal | Description | Strategy |
| Constraint 2 | Budget / Timeline / Resource / Legal | Description | Strategy |

## High-Level Operational Concept

### Operational Scenarios

Describe how the business will operate once the initiative is realized:

1. Scenario 1: [Actor] does [action] to achieve [outcome]
2. Scenario 2: [Actor] does [action] to achieve [outcome]

### Operational Modes

| Mode | Description | Conditions |
|------|-------------|------------|
| Normal | Standard business operations | Default |
| Degraded | Reduced capability operations | Description of trigger |
| Maintenance | Planned downtime or transition | Description of trigger |

### Business Operational Quality

Quality expectations at the business level (not system-level metrics):

| ID | Quality Attribute | Requirement | Target |
|----|-------------------|-------------|--------|
| BQ-001 | Service level | Description | Value (e.g., 99.9% order fulfillment) |
| BQ-002 | Customer satisfaction | Description | Value (e.g., NPS > 50) |

## Project Constraints

- Schedule: Key milestones and deadlines
- Budget: Funding limits and allocation
- Resources: Team capacity, skills, availability
- Organizational: Approval processes, governance

## Success Criteria

| ID | Criterion | Metric | Target | Timeline |
|----|-----------|--------|--------|----------|
| SC-001 | Criterion 1 | Metric | Target value | Date |
| SC-002 | Criterion 2 | Metric | Target value | Date |

## Assumptions and Dependencies

| ID | Type | Description | Risk if Invalid |
|----|------|-------------|-----------------|
| A-001 | Assumption | Description | Impact |
| A-002 | Assumption | Description | Impact |
| D-001 | Dependency | Description | Impact |
| D-002 | Dependency | Description | Impact |

## Traceability

This BRS formalizes requirements from source documents via ` + "`implements`" + ` relation:

- MRD: [link to market requirements document]
- BRD: [link to business requirements document]

| BRS Requirement | Source Document | Source Requirement |
|-----------------|-----------------|---------------------|
| BG-001 | MRD / BRD | Source ID or section |
| BO-001 | MRD / BRD | Source ID or section |
`
}

func generateStRSTemplate() string {
	return `## Purpose and Scope

What this stakeholder requirements specification covers — the system or initiative boundary from the stakeholder perspective.

## System Overview

High-level description of the proposed system or initiative — what it does, major components, and how stakeholders interact with it.

## Business Context

### Business Environment

- Market and industry context relevant to stakeholders
- Regulatory landscape affecting stakeholder operations
- Organizational context (structure, culture, capabilities)

### Mission, Goals and Objectives

Stakeholder-facing goals derived from the business mission:

| ID | Goal / Objective | Stakeholder Class | Description | Success Measure |
|----|------------------|-------------------|-------------|-----------------|
| SG-001 | Goal 1 | Class | Description | Measurable outcome |
| SG-002 | Goal 2 | Class | Description | Measurable outcome |

## Stakeholder Classes

| Stakeholder Class | Description | Priority | Key Concerns |
|-------------------|-------------|----------|--------------|
| Class 1 | Description | Primary / Secondary | Concerns |
| Class 2 | Description | Primary / Secondary | Concerns |

## Operational Concept (ConOps)

### Current Operations

How stakeholders accomplish their goals today:

1. Step 1: What happens, pain points
2. Step 2: What happens, pain points
3. Step 3: What happens, pain points

### Proposed Operations

How stakeholders should accomplish their goals:

1. Step 1: What happens, improvement over current
2. Step 2: What happens, improvement over current
3. Step 3: What happens, improvement over current

### Operational Scenarios

| ID | Scenario | Stakeholder Class | Trigger | Expected Outcome |
|----|----------|-------------------|---------|------------------|
| OS-001 | Scenario 1 | Class | Event or condition | Outcome |
| OS-002 | Scenario 2 | Class | Event or condition | Outcome |

## Stakeholder Requirements

### User Requirements

| ID | Requirement | Stakeholder Class | Priority | Rationale |
|----|-------------|-------------------|----------|-----------|
| SR-001 | Description | Class | P0/P1/P2 | Why this matters |
| SR-002 | Description | Class | P0/P1/P2 | Why this matters |

### Usability Requirements

- Accessibility standard (e.g., WCAG 2.1 AA)
- Learnability: time to first productive use
- Efficiency: task completion time targets
- Error tolerance: acceptable error rates

### Quality Requirements

| Attribute | Requirement | Metric | Target |
|-----------|-------------|--------|--------|
| Performance | Description | Metric | Value |
| Reliability | Description | Metric | Value |
| Availability | Description | Metric | Value |

## System Processes

Key processes the system must support from the stakeholder perspective:

1. Process 1: Which stakeholder class uses it, expected behavior, frequency
2. Process 2: Which stakeholder class uses it, expected behavior, frequency

## Operational Policies and Rules

| ID | Policy / Rule | Description | Source |
|----|---------------|-------------|--------|
| OP-001 | Policy 1 | Description | Regulatory / Organizational |
| OP-002 | Rule 1 | Description | Regulatory / Organizational |

## Operational Constraints

### Modes and States

| Mode / State | Description | Stakeholder Impact | Transitions |
|--------------|-------------|--------------------|-------------|
| Normal | Standard operations | Full capability | Default |
| Degraded | Reduced service | Limited capability | Trigger condition |
| Maintenance | Planned outage | No service | Scheduled |

## Compliance and Regulatory

| Regulation / Standard | Requirement | Applicable Section | Compliance Approach |
|-----------------------|-------------|---------------------|---------------------|
| Regulation 1 | Description | Section | Approach |
| Standard 1 | Description | Section | Approach |

## Project Constraints

- Schedule: Key milestones and deadlines
- Budget: Funding limits and allocation
- Technology: Platform, language, infrastructure constraints
- Organizational: Team capacity, skills, processes

## Traceability

This StRS formalizes stakeholder requirements from source documents via ` + "`implements`" + ` relation:

- URD: [link to user requirements document]
- BRS: [link to business requirements specification]

| StRS Requirement | Source Document | Source Requirement |
|------------------|-----------------|---------------------|
| SR-001 | URD / BRS | Source ID or section |
| SR-002 | URD / BRS | Source ID or section |
`
}

func generateSyRSTemplate() string {
	return `## System Purpose and Scope

What the system does and why it exists.

### System Boundary

What is inside vs. outside the system:

- Inside: Components, services, and capabilities owned by this system
- Outside: External systems, users, and services the system interacts with

## System Overview

High-level description of the system architecture — major components, data flows, and deployment topology.

## System Requirements

### Functional Requirements

| ID | Requirement | Description | Priority | Verification Method |
|----|-------------|-------------|----------|---------------------|
| SyR-F-001 | Requirement 1 | Description | P0/P1/P2 | Test / Inspection / Analysis / Demo |
| SyR-F-002 | Requirement 2 | Description | P0/P1/P2 | Test / Inspection / Analysis / Demo |

### Usability Requirements

| ID | Requirement | Target | Verification Method |
|----|-------------|--------|---------------------|
| SyR-U-001 | Requirement 1 | Target value | Method |
| SyR-U-002 | Requirement 2 | Target value | Method |

### Performance Requirements

| ID | Requirement | Metric | Target | Verification Method |
|----|-------------|--------|--------|---------------------|
| SyR-P-001 | Throughput | req/s | Value | Load test |
| SyR-P-002 | Latency | ms (P95) | Value | Load test |
| SyR-P-003 | Resource usage | CPU/Memory | Value | Monitoring |

### Security Requirements

| ID | Requirement | Description | Verification Method |
|----|-------------|-------------|---------------------|
| SyR-S-001 | Requirement 1 | Description | Pen test / Audit / Review |
| SyR-S-002 | Requirement 2 | Description | Pen test / Audit / Review |

### Reliability Requirements

| ID | Requirement | Metric | Target | Verification Method |
|----|-------------|--------|--------|---------------------|
| SyR-R-001 | Availability | Uptime % | Value | Monitoring |
| SyR-R-002 | MTTR | Hours | Value | Incident review |
| SyR-R-003 | Data durability | RPO/RTO | Value | DR test |

### Information Management Requirements

| ID | Requirement | Description | Verification Method |
|----|-------------|-------------|---------------------|
| SyR-I-001 | Data retention | How long data must be stored before archival/deletion | Audit |
| SyR-I-002 | Data privacy | PII handling, anonymization, access control | Review / Pen test |
| SyR-I-003 | Data backup | Backup frequency, recovery procedures | DR test |

## System Interfaces

### User Interfaces

| Interface | Description | Format / Protocol |
|-----------|-------------|-------------------|
| UI 1 | Description | Web / Mobile / CLI |
| UI 2 | Description | Web / Mobile / CLI |

### System-to-System Interfaces

| Interface | External System | Direction | Protocol | Data Format |
|-----------|-----------------|-----------|----------|-------------|
| Interface 1 | System name | In / Out / Bidirectional | REST / gRPC / Event | JSON / Protobuf |
| Interface 2 | System name | In / Out / Bidirectional | REST / gRPC / Event | JSON / Protobuf |

### Hardware Interfaces

If applicable — physical devices, sensors, actuators.

## System Operations

### Modes and States

| Mode / State | Description | Transitions | Behavior |
|--------------|-------------|-------------|----------|
| Normal | Full system operation | Default | All features available |
| Degraded | Partial capability | Trigger: failure condition | Graceful degradation strategy |
| Maintenance | Planned downtime | Trigger: operator command | Controlled shutdown/startup |

### Physical and Environmental

- Operating conditions (temperature, network, power)
- Physical deployment constraints
- Geographic distribution requirements

## Policy and Regulation

- Applicable industry standards and regulations
- Certification requirements
- Data residency and sovereignty constraints

## Life Cycle Sustainment

- Maintenance strategy and schedule
- Support tiers and SLA targets
- Evolution and extensibility approach
- End-of-life considerations

## Assumptions and Dependencies

| ID | Type | Description | Risk if Invalid |
|----|------|-------------|-----------------|
| A-001 | Assumption | Description | Impact |
| D-001 | Dependency | Description | Impact |

## Verification Approach

| Requirement ID | Verification Method | Acceptance Criteria |
|----------------|---------------------|---------------------|
| SyR-F-001 | Test / Inspection / Analysis / Demo | Criteria |
| SyR-P-001 | Load test | Criteria |
| SyR-S-001 | Penetration test | Criteria |

## Traceability

This SyRS decomposes requirements from the stakeholder requirements specification via ` + "`implements`" + ` relation:

- StRS: [link to stakeholder requirements specification]

| SyRS Requirement | Source Document | Source Requirement |
|------------------|-----------------|---------------------|
| SyR-F-001 | StRS | Source ID or section |
| SyR-P-001 | StRS | Source ID or section |
`
}

func generateSRSTemplate() string {
	return `## Purpose and Scope

### Component

Which software component, service, or module this SRS covers.

### Boundaries

What is in scope vs. delegated to other components or services.

## Product Perspective

### Functions

High-level summary of the software functions provided by this component.

### User Characteristics

Expected users of this component — their roles, skills, and interaction patterns.

### Limitations

Known constraints and design boundaries for this component.

### Assumptions and Dependencies

| ID | Type | Description | Risk if Invalid |
|----|------|-------------|-----------------|
| A-001 | Assumption | Description | Impact |
| D-001 | Dependency | Description | Impact |

## Software Requirements

### Functional Requirements

| ID | Requirement | Input | Processing | Output | Priority |
|----|-------------|-------|------------|--------|----------|
| SR-F-001 | Requirement 1 | Input description | Processing logic | Output description | P0/P1/P2 |
| SR-F-002 | Requirement 2 | Input description | Processing logic | Output description | P0/P1/P2 |

### Behavioral Requirements

State machines, workflows, or sequences that define component behavior:

1. State/workflow 1: Description of transitions and triggers
2. State/workflow 2: Description of transitions and triggers

### Error Handling

| Error Condition | Detection | Response | Recovery |
|-----------------|-----------|----------|----------|
| Error 1 | How detected | System response | Recovery steps |
| Error 2 | How detected | System response | Recovery steps |

## External Interfaces

### API Endpoints

| Method | Path | Description | Request | Response | Auth |
|--------|------|-------------|---------|----------|------|
| GET | /api/v1/resource | Description | Params | 200: Schema | Bearer / API key |
| POST | /api/v1/resource | Description | Body schema | 201: Schema | Bearer / API key |

### Internal Interfaces

Interfaces to other internal components or services:

| Interface | Target Component | Direction | Contract |
|-----------|-----------------|-----------|----------|
| Interface 1 | Component name | In / Out | Function signature / Event schema |
| Interface 2 | Component name | In / Out | Function signature / Event schema |

## Data Requirements

### Logical Database

Key entities and their relationships:

| Entity | Description | Key Attributes | Relationships |
|--------|-------------|----------------|---------------|
| Entity 1 | Description | Attributes | Relations |
| Entity 2 | Description | Attributes | Relations |

### Data Flows

How data moves through the component — input sources, transformations, output destinations.

## Usability Requirements

- Accessibility standard (e.g., WCAG 2.1 AA)
- Learnability: time for a new developer/user to become productive
- Error messages: clarity, actionability, consistency
- API ergonomics: discoverability, consistency of naming/patterns

## Performance

| Metric | Requirement | Measurement Method | Target |
|--------|-------------|-------------------|--------|
| Response time | P95 latency | Load test | Value |
| Throughput | Requests/sec | Load test | Value |
| Memory | Peak usage | Profiling | Value |

## Design Constraints

### Standards Compliance

- Coding standards and conventions
- Framework and library constraints
- Protocol compliance requirements

## Software Quality Attributes

| Attribute | Requirement | Metric | Target |
|-----------|-------------|--------|--------|
| Maintainability | Description | Metric | Value |
| Testability | Description | Coverage % | Value |
| Portability | Description | Metric | Value |
| Scalability | Description | Metric | Value |

## Verification Matrix

| Requirement ID | Test Type | Test Description | Pass Criteria |
|----------------|-----------|------------------|---------------|
| SR-F-001 | Unit / Integration / E2E | Description | Criteria |
| SR-F-002 | Unit / Integration / E2E | Description | Criteria |

## Traceability

This SRS decomposes requirements from the system requirements specification via ` + "`implements`" + ` relation:

- SyRS: [link to system requirements specification]

| SRS Requirement | Source Document | Source Requirement |
|-----------------|-----------------|---------------------|
| SR-F-001 | SyRS | Source ID or section |
| SR-F-002 | SyRS | Source ID or section |
`
}

