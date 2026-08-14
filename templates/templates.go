package templates

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
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
	TypeRnD      DocumentType = "rnd"
	TypeSpec     DocumentType = "spec"
	TypeMRD      DocumentType = "mrd"
	TypeBRD      DocumentType = "brd"
	TypeURD      DocumentType = "urd"
	TypeBRS      DocumentType = "brs"
	TypeStRS     DocumentType = "strs"
	TypeSyRS     DocumentType = "syrs"
	TypeSRS      DocumentType = "srs"
)

type Category string

const (
	CategoryVision     Category = "vision"
	CategoryKnowledge  Category = "knowledge"
	CategoryExperience Category = "experience"
)

type DocStatus string

const (
	StatusDraft    DocStatus = "draft"
	StatusAccepted DocStatus = "accepted"
	StatusRejected DocStatus = "rejected"
)

// TagRe validates a single tag: lowercase alphanumeric with hyphens, underscores, colons, and pipes.
var TagRe = regexp.MustCompile(`^[a-z][a-z0-9_:|-]*$`)

// SlugRe validates a document slug: lowercase alphanumeric segments separated by hyphens.
var SlugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Frontmatter holds the parsed YAML frontmatter fields of a document.
// JSON tags are present so the same struct can serve as the wire shape for
// sync payloads, avoiding a parallel type.
type Frontmatter struct {
	Title  string    `yaml:"title" json:"title"`
	Status DocStatus `yaml:"status" json:"status,omitempty"`
	Tags   []string  `yaml:"tags" json:"tags,omitempty"`
}

// SkipFiles are non-document meta files that live in .archcore/ and should be
// skipped during scanning, validation, and sync operations.
var SkipFiles = map[string]bool{
	"settings.json":    true,
	".sync-state.json": true,
}

// ValidStatusStrings returns all valid document statuses as plain strings.
// Useful for assembling error messages with strings.Join.
func ValidStatusStrings() []string {
	return []string{string(StatusDraft), string(StatusAccepted), string(StatusRejected)}
}

// IsValidStatus checks whether the given value is a valid document status.
func IsValidStatus(s DocStatus) bool {
	switch s {
	case StatusDraft, StatusAccepted, StatusRejected:
		return true
	}
	return false
}

// ValidCategoryStrings returns all valid document categories as plain strings.
// Useful for assembling MCP enum schemas and error messages with strings.Join.
func ValidCategoryStrings() []string {
	return []string{string(CategoryVision), string(CategoryKnowledge), string(CategoryExperience)}
}

// IsValidCategory checks whether the given value is a valid document category.
func IsValidCategory(c Category) bool {
	switch c {
	case CategoryVision, CategoryKnowledge, CategoryExperience:
		return true
	}
	return false
}

var categoryMap = map[DocumentType]Category{
	TypePRD:  CategoryVision,
	TypeIdea: CategoryVision,
	TypePlan: CategoryVision,
	TypeRnD:  CategoryVision,
	TypeMRD:  CategoryVision,
	TypeBRD:  CategoryVision,
	TypeURD:  CategoryVision,
	TypeBRS:  CategoryVision,
	TypeStRS: CategoryVision,
	TypeSyRS: CategoryVision,
	TypeSRS:  CategoryVision,

	TypeADR:   CategoryKnowledge,
	TypeRFC:   CategoryKnowledge,
	TypeRule:  CategoryKnowledge,
	TypeGuide: CategoryKnowledge,
	TypeDoc:   CategoryKnowledge,
	TypeSpec:  CategoryKnowledge,

	TypeTaskType: CategoryExperience,
	TypeCPAT:     CategoryExperience,
}

// CategoryForType returns the category for a document type.
func CategoryForType(docType DocumentType) Category {
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
		string(TypeRnD),
		string(TypeMRD),
		string(TypeBRD),
		string(TypeURD),
		string(TypeBRS),
		string(TypeStRS),
		string(TypeSyRS),
		string(TypeSRS),
	}
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
// The error is non-nil only when a delimited frontmatter block is present but
// its YAML fails to parse; the body is still returned correctly in that case,
// so callers that tolerate broken metadata can ignore the error. A missing or
// unterminated frontmatter block is not an error — the whole input is returned
// as body with empty frontmatter.
func SplitDocument(data []byte) (Frontmatter, string, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	// Strip a UTF-8 BOM so BOM'd files keep their frontmatter.
	s = strings.TrimPrefix(s, "\ufeff")
	if !strings.HasPrefix(s, "---\n") {
		return Frontmatter{}, s, nil
	}

	end := strings.Index(s[4:], "\n---\n")
	if end == -1 {
		// Handle frontmatter closed by "\n---" at EOF (no trailing newline).
		if idx := strings.Index(s[4:], "\n---"); idx != -1 && idx+4+len("\n---") == len(s) {
			var fm Frontmatter
			err := parseFrontmatterYAML(s[4:4+idx], &fm)
			return fm, "", err
		}
		return Frontmatter{}, s, nil
	}
	end += 4 // adjust for the offset

	var fm Frontmatter
	err := parseFrontmatterYAML(s[4:end], &fm)

	body := s[end+5:] // skip past "\n---\n"
	body = strings.TrimPrefix(body, "\n")

	return fm, body, err
}

func parseFrontmatterYAML(content string, fm *Frontmatter) error {
	if err := yaml.Unmarshal([]byte(content), fm); err != nil {
		return fmt.Errorf("invalid frontmatter YAML: %w", err)
	}
	return nil
}

// WalkArchcoreFiles walks archcoreDir recursively, calling fn for each .md
// document file found. It skips hidden directories, non-.md files, and known
// meta files (settings.json, .sync-state.json).
func WalkArchcoreFiles(archcoreDir string, fn func(path string, d fs.DirEntry) error) error {
	return WalkArchcoreFilesSkipping(archcoreDir, nil, fn)
}

// WalkArchcoreFilesSkipping is like WalkArchcoreFiles but also skips any
// subdirectory whose name matches an entry in skipDirs.
func WalkArchcoreFilesSkipping(archcoreDir string, skipDirs []string, fn func(path string, d fs.DirEntry) error) error {
	return filepath.WalkDir(archcoreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		name := d.Name()

		// Never follow symlinks. A symlink inside .archcore/ (or a mounted
		// global) could point at a file outside the tree; reading, hashing, or
		// listing its target would breach the "nothing outside .archcore/ is
		// ever read" invariant (sync-path-security.rule). d.Type() reflects the
		// entry itself (lstat semantics), so this catches symlinks to both files
		// and directories without ever resolving the target.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			// Skip hidden directories (but not .archcore itself).
			if strings.HasPrefix(name, ".") && path != archcoreDir {
				return filepath.SkipDir
			}
			// Skip explicitly excluded directory names.
			if slices.Contains(skipDirs, name) {
				return filepath.SkipDir
			}
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
	case TypeRnD:
		return generateRnDTemplate()
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

State the trigger in 2 to 4 sentences: the situation now, the constraint or
measurement that turns it into a problem, and what breaks if nothing changes.
Cite the evidence — @path/to/file, a metric, a commit, an external authority —
or mark the claim [assumption] when the decision runs ahead of the code.

Prose only in this section. A bullet list drops the connective tissue between
those sentences, and that tissue is what makes the decision follow from them.

## Decision

State the decision that was made.

### Rationale

Explain why this decision was chosen over alternatives:

- Key factors that influenced the decision
- Trade-offs that were considered
- Assumptions that were made

## Alternatives Considered

One numbered item per alternative, each naming what ruled it out. The reason is
the part a later reader cannot reconstruct, and an alternative recorded without
one reads as a preference rather than a choice.

1. [Alternative] — rejected because [the measurement or constraint that ruled it out].
2. [Alternative] — rejected because [the measurement or constraint that ruled it out].

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

State each rule as one numbered clause carrying one BCP 14 modal (MUST /
MUST NOT / SHOULD / MAY, uppercase), one named actor, and the scope it governs —
a path, a glob, or a named situation. Keep a clause at or under 25 words: past
that it usually carries a second obligation. Put the trigger before the response.

1. The [actor] MUST [response]. Applies to ` + "`path/glob`" + `.
2. WHEN [trigger], the [actor] MUST [response]. Applies to ` + "`path/glob`" + `.
3. IF [undesired condition], THEN the [actor] MUST [response]. Applies to [named situation].

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

Name the check that catches a violation, or state that none exists:

- ` + "`lint-rule-or-ci-step`" + ` — what it catches and where it runs.
- ` + "`manual review`" + ` — for the part no automated check covers.
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

One numbered step per action, each at or under 20 words. A step tells the
reader what to do now, so it opens with a verb and carries no BCP 14 modal.

1. [First action — what to do and where] (@path/to/file)

   ` + "```" + `
   # Commands or code for this step
   ` + "```" + `

   **Expected result:** What you should see after this step.

2. [Second action]

   ` + "```" + `
   # Commands or code for this step
   ` + "```" + `

   **Expected result:** What you should see after this step.

3. [Third action]

   ` + "```" + `
   # Commands or code for this step
   ` + "```" + `

   **Expected result:** What you should see after this step.

4. [Final action]

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

One to three sentences: the product or feature, and the result it produces.

## Problem Statement

The situation today, what it costs, and who carries that cost. Name the
beneficiary — a role, a team, or a segment, not "users" in general.

- [What happens today]: [its cost, with a number, or marked [assumption]]
- [What happens today]: [its cost, with a number, or marked [assumption]]

## Goals and Success Metrics

Each metric carries units and a value; where no measurement exists yet, mark
the figure [assumption].

| Goal | Metric | Today | Target |
|------|--------|-------|--------|
| [outcome] | [what is measured, with units] | [value] | [value] |
| [outcome] | [what is measured, with units] | [value] | [value] |

## Requirements

Numbered outcomes this product owes: what it must achieve, for whom, at what
threshold. One outcome per line. State the result, not the mechanism.

1. [Who] [reaches what result].
2. [Who] [reaches what result].
3. [Who] [reaches what result].

Keep out of a prd — each belongs to a linked document:

- EARS clauses and BCP 14 modals (MUST / SHOULD / MAY) -> spec, Normative Behavior
- Interfaces, signatures, states, field-driven rules -> spec, Surface
- Error, edge, and degradation handling -> spec, Failure Behavior
- Phases, tasks, milestones, delivery dates -> plan, Tasks
- Rejected alternatives and why -> adr, Alternatives Considered

Optional sections, when they carry content: ` + "`## Out of Scope`" + `,
` + "`## Dependencies`" + `, ` + "`## Clarifications`" + `.

Scope rule: one prd covers one unit of product decision — a whole product or a
single feature. A feature-scoped prd uses these same four sections at a target
of 40 lines or fewer. A product-level prd links each feature-scoped prd it
covers through an ` + "`implements`" + ` relation.
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

One numbered task per action, each at or under 20 words. Numbering inside the
phase, so a task can be cited as "Phase 2, task 1".

### Phase 1: [Name]

1. [First task — one action, one place]
2. [Second task]

### Phase 2: [Name]

1. [First task of this phase]

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Dependencies

| Dependency | Type | Status |
|------------|------|--------|

## Notes
`
}

func generateRnDTemplate() string {
	return `## Research Goal

State the single question this investigation must answer, in one sentence.

### Scope

- In scope: what this research will and will not cover
- Time box: how long this investigation should take

## Context & Trigger

- What prompted this research now?
- What decision or work is blocked until it concludes?

## Questions / Hypotheses

- Question 1: what we need to know
- Hypothesis 1: what we expect to be true, and how we'd disprove it

## Approach

### Inputs

- Source 1: docs, data, prior art, or people consulted

### Options

- Option 1: brief description
- Option 2: brief description

### Experiments

- Experiment 1: what to try and what outcome would confirm or refute a hypothesis

## Findings

What the investigation actually showed. Stick to evidence, not opinion.

- Finding 1: observation and supporting evidence
- Finding 2: observation and supporting evidence

## Implications

What the findings mean for the decision or work that triggered this research.

- Implication 1: consequence of the findings
- Implication 2: consequence of the findings

## Recommendation

State exactly one and justify it in 1-2 sentences:

- **proceed** — adopt the option / move forward as investigated
- **refine** — promising, but needs another iteration before deciding
- **defer** — park this; revisit when a stated condition changes
- **stop** — do not pursue; the question is answered negatively

## Next Action

The single concrete step that follows from the recommendation.

- [ ] Owner does [action] by [when]

## Risks & Unknowns

- Risk 1: what could invalidate this recommendation
- Unknown 1: what remains unanswered and why it's acceptable for now

## Related Materials

- External references, data sets, or prior art (links or @-paths)
`
}

func generateSpecTemplate() string {
	return `## Purpose & Scope

This specification is normative for [subject] — [one-line statement of what it is].

Depended on by: [external code, teams, UI surfaces, or sibling modules].

Out of scope: [adjacent concerns covered elsewhere, with a pointer].

## Surface

What dependents see of the subject. Reference source definitions with @-notation
(@path/to/file) — do NOT copy interface, type, or struct bodies; copies go stale.

- [Interface / entry point]: @path/to/file — [what it represents, who consumes it]
- [Part / state / field-driver — for feature subjects]: [field] drives [behavior]
- States: [state1 → state2 → ...] (include only if the subject is stateful)

Use a code block ONLY where the exact textual format is itself normative
(HTTP endpoint shape, CLI flag grammar, wire format).

## Normative Behavior

Numbered EARS-shaped requirements with BCP 14 keywords (MUST / SHOULD / MAY,
uppercase only — RFC 2119 + RFC 8174). Keep MUST sparing: interoperation or
harm prevention only. Strict EARS:

- Active voice with an obligated subject — never a subjectless passive
  ("tokens MUST be rotated" obligates no component).
- One numbered line = one requirement = one modal keyword (MUST NOT counts
  as one); split "MUST X and MUST NOT Y" into two numbered lines.
- Event responses (command invoked, request received, state change) open
  with "WHEN [trigger]," — the event is the trigger, never the subject.

1. The [subject] MUST [response].
2. WHEN [trigger], the [subject] MUST [response].
3. WHILE [state], the [subject] SHOULD [response].

## Constraints & Invariants

Plain BCP 14 statements (no EARS clauses needed here).

- Constraint: [hard limit] — [rationale]
- Invariant: [condition that MUST always hold]

## Failure Behavior

Error and edge conditions with the observable outcome of each: response and
recovery semantics (retriable? idempotent? timeout?) and degradation on bad,
empty, or missing input or on dependency failure. Same notation and rules.

1. IF [undesired condition], THEN the [subject] MUST [observable outcome].
2. WHEN [dependency] fails, the [subject] MUST [degradation behavior].

## Conformance

An implementation conforms when it satisfies all MUST requirements, all
invariants, and all failure rules above.

Optionally close with ONE non-normative example (<= 5 lines, Given/When/Then)
anchoring the most load-bearing behavior:

` + "```" + `txt
Given [initial state]
When [action]
Then [expected outcome]
` + "```" + `
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
