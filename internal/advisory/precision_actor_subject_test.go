// Actor-subject canon properties pinned here, per
// scenario-and-journey-advisory-canon.spec:
//  1. Every Flows subsection cites its code with an Anchors: line (§7).
//  2. Every numbered step under Flows or Journeys opens with an actor (§8).
//  3. A Given/When/Then line in Examples, Flows, or Journeys is a step, numbered
//     or not: word cap and modal check (§4-6).
//  4. Both types share the spec's 120-line cap through MaxBodyLines (§9-10).
//  5. Foreign headings are reported in both directions (§11-14).
//  6. Without an Actors table the actor check stays silent (Failure §1).
//  7. An Actors row without outer pipes still names its actor (GFM).
//  8. A fenced block under Flows is neither a flow nor an anchor.
package advisory

import (
	"strings"
	"testing"

	"archcore-cli/templates"
)

const scenarioBody = `## Subject

The refund flow of the store, illustrating clauses 2 and 5 of the linked spec.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Anna | a buyer with one order | her money back |
| The support agent | staff on the refund queue | a closed ticket |

## Flows

### Anna

Anchors: @internal/refund/refund.go, @internal/refund/refund_test.go

1. Anna opens the order; the system shows the refund button.
2. Anna requests the refund; the system shows the refund as approved.

## Examples

### A refund inside the window

Illustrates: clause 2

Given Anna bought a book on 1 September
When Anna requests a refund on 10 September
Then Anna sees the refund approved

## Open Questions

- Whether a partial refund exists.
`

const journeyBody = `## Intent

In order to get my money back
As a buyer
I want a refund without a call

We want the buyer to finish a refund from the order page.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| The buyer | a customer with one order | a refund |

## Journeys

### The buyer

1. The buyer opens the order.
2. The buyer asks for the refund and reads the answer.

## Open Questions

- Whether a partial refund exists.
`

func findingsFor(t *testing.T, typ templates.DocumentType, body string) string {
	t.Helper()
	fm := templates.Frontmatter{Title: "T", Status: templates.StatusDraft}
	return strings.Join(PrecisionFindings(typ, fm, body), "\n")
}

func TestPrecision_ActorSubjectCleanBodiesReportNothing(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		typ  templates.DocumentType
		body string
	}{
		{name: "scenario", typ: templates.TypeScenario, body: scenarioBody},
		{name: "journey", typ: templates.TypeJourney, body: journeyBody},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := findingsFor(t, tt.typ, tt.body); got != "" {
				t.Errorf("clean %s reports:\n%s", tt.name, got)
			}
		})
	}
}

func TestPrecision_ActorSubjectFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		typ     templates.DocumentType
		mutate  func(string) string
		want    string
		wantNot string
	}{
		{
			name: "flow without anchors is named",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Anchors: @internal/refund/refund.go, @internal/refund/refund_test.go\n", "", 1)
			},
			want: "flow without an Anchors: line (Anna)",
		},
		{
			name: "anchors line under the section heading covers a flow with no subsection",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "### Anna\n\n", "", 1)
			},
			wantNot: "flow without an Anchors",
		},
		{
			name: "flow with no subsection and no anchors is named after the section",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				b = strings.Replace(b, "### Anna\n\n", "", 1)
				return strings.Replace(b, "Anchors: @internal/refund/refund.go, @internal/refund/refund_test.go\n\n", "", 1)
			},
			want: "flow without an Anchors: line (Flows)",
		},
		{
			name: "step opening with a longer word that starts with the actor is reported",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "1. Anna opens the order;", "1. Annabelle opens the order;", 1)
			},
			want: "step opens with no actor (1. Annabelle opens the order;",
		},
		{
			name: "step opening with the component is reported",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "1. Anna opens the order; the system shows the refund button.", "1. The system shows the refund button when the order opens.", 1)
			},
			want: "step opens with no actor (1. The system shows the refund button when the order opens.)",
		},
		{
			name: "step opening with the second actor passes",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "1. Anna opens the order;", "1. The support agent opens the order;", 1)
			},
			wantNot: "step opens with no actor",
		},
		{
			name: "modal in a flow step is a modal in a step",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "the system shows the refund button.", "the system MUST show the refund button.", 1)
			},
			want: "BCP 14 modal in a step (MUST)",
		},
		{
			name: "modal in a Then line is a modal in a step",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Then Anna sees the refund approved", "Then Anna MUST see the refund approved", 1)
			},
			want: "BCP 14 modal in a step (MUST)",
		},
		{
			name: "long Given line is a step over the cap",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Given Anna bought a book on 1 September", "Given Anna bought a book on 1 September together with two more books and a bookmark and a card and wrapping paper", 1)
			},
			want: "step over 20 words",
		},
		{
			name: "journey step without an actor is reported",
			typ:  templates.TypeJourney,
			mutate: func(b string) string {
				return strings.Replace(b, "1. The buyer opens the order.", "1. Open the order.", 1)
			},
			want: "step opens with no actor (1. Open the order.)",
		},
		{
			name: "journey body over the cap names the actor split",
			typ:  templates.TypeJourney,
			mutate: func(b string) string {
				return b + strings.Repeat("Filler prose line carrying no obligation.\n", templates.MaxSpecBodyLines)
			},
			want: "journey body is 144 lines (cap 120) — if it has taken on rules or data, move them to the spec; otherwise split by actor",
		},
		{
			name: "scenario body over the cap names the actor split",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return b + strings.Repeat("Filler prose line carrying no obligation.\n", templates.MaxSpecBodyLines)
			},
			want: "(cap 120) — split by actor",
		},
		{
			name: "normative heading in a scenario names the spec as owner",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return b + "\n## Normative Behavior\n\n1. The service MUST approve the refund.\n"
			},
			want: "section ## Normative Behavior in a scenario — a spec owns that content",
		},
		{
			name: "requirements heading in a journey names the prd as owner",
			typ:  templates.TypeJourney,
			mutate: func(b string) string {
				return b + "\n## Requirements\n\n1. The buyer gets a refund.\n"
			},
			want: "section ## Requirements in a journey — a prd owns that content",
		},
		{
			name: "modal in a numbered Subject line is a claim defect",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "## Actors", "1. The store MUST refund within 14 days.\n\n## Actors", 1)
			},
			want: "BCP 14 modal (MUST) in a numbered scenario clause",
		},
		{
			name: "long Given line under Flows is a step over the cap",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "1. Anna opens the order;",
					"Given Anna waited one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen days\n1. Anna opens the order;", 1)
			},
			want: "step over 20 words (Given Anna waited",
		},
		{
			name: "modal in a When line under Journeys is a modal in a step",
			typ:  templates.TypeJourney,
			mutate: func(b string) string {
				return strings.Replace(b, "1. The buyer opens the order.",
					"When the buyer SHOULD read the answer\n1. The buyer opens the order.", 1)
			},
			want: "BCP 14 modal in a step (SHOULD)",
		},
		{
			name: "actors table without outer pipes still names the actors",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				b = strings.Replace(b, "| Actor | Who they are | What they want |\n|---|---|---|\n| Anna | a buyer with one order | her money back |\n| The support agent | staff on the refund queue | a closed ticket |",
					"Actor | Who they are | What they want\n---|---|---\nAnna | a buyer with one order | her money back\nThe support agent | staff on the refund queue | a closed ticket", 1)
				return strings.Replace(b, "1. Anna opens the order;", "1. The system opens the order;", 1)
			},
			want:    "step opens with no actor (1. The system opens the order;",
			wantNot: "1. Anna requests the refund",
		},
		{
			name: "heading inside a fenced block under Flows is not a flow",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "## Examples", "```\n### Example\n```\n\n## Examples", 1)
			},
			wantNot: "flow without an Anchors: line",
		},
		{
			name: "anchors line inside a fenced block does not anchor the flow",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Anchors: @internal/refund/refund.go, @internal/refund/refund_test.go\n",
					"```\nAnchors: @internal/refund/refund.go\n```\n", 1)
			},
			want: "flow without an Anchors: line (Anna)",
		},
		{
			name: "row inside a fenced block under Actors names no actor",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				b = strings.Replace(b, "## Flows", "```\n| Bob | a tester | nothing |\n```\n\n## Flows", 1)
				return strings.Replace(b, "1. Anna opens the order;", "1. Bob opens the order;", 1)
			},
			want: "step opens with no actor (1. Bob opens the order;",
		},
		{
			name: "numbered Given step under Flows needs no actor",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "1. Anna opens the order;", "1. Given the order is open;", 1)
			},
			wantNot: "step opens with no actor",
		},
		{
			name: "modal in a But line is a modal in a step",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Then Anna sees the refund approved\n",
					"Then Anna sees the refund approved\nBut Anna MAY not see the total\n", 1)
			},
			want: "BCP 14 modal in a step (MAY)",
		},
		{
			name: "a line that only starts with a keyword's letters is not a step",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				return strings.Replace(b, "Then Anna sees the refund approved\n",
					"Then Anna sees the refund approved\nWhenever Anna pays late, the store MUST decline the refund.\n", 1)
			},
			wantNot: "modal in a step",
		},
		{
			name: "no actors table keeps the actor check silent",
			typ:  templates.TypeScenario,
			mutate: func(b string) string {
				start := strings.Index(b, "## Actors")
				end := strings.Index(b, "## Flows")
				return strings.Replace(b[:start]+b[end:], "1. Anna opens", "1. Open", 1)
			},
			want:    "missing section: ## Actors",
			wantNot: "step opens with no actor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := scenarioBody
			if tt.typ == templates.TypeJourney {
				base = journeyBody
			}
			got := findingsFor(t, tt.typ, tt.mutate(base))
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("findings lack %q:\n%s", tt.want, got)
			}
			if tt.wantNot != "" && strings.Contains(got, tt.wantNot) {
				t.Errorf("findings carry %q:\n%s", tt.wantNot, got)
			}
		})
	}
}

func TestPrecision_ActorSubjectHeadingsAreForeignInSpecAndPRD(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		typ   templates.DocumentType
		title string
		owner string
	}{
		{name: "flows in a spec", typ: templates.TypeSpec, title: "Flows", owner: "scenario"},
		{name: "examples in a spec", typ: templates.TypeSpec, title: "Examples", owner: "scenario"},
		{name: "journeys in a spec", typ: templates.TypeSpec, title: "Journeys", owner: "journey"},
		{name: "flows in a prd", typ: templates.TypePRD, title: "Flows", owner: "scenario"},
		{name: "examples in a prd", typ: templates.TypePRD, title: "Examples", owner: "scenario"},
		{name: "journeys in a prd", typ: templates.TypePRD, title: "Journeys", owner: "journey"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := templates.GenerateTemplate(tt.typ) + "\n## " + tt.title + "\n\nContent.\n"
			want := "section ## " + tt.title + " in a " + string(tt.typ) + " — a " + tt.owner + " owns that content"
			if got := findingsFor(t, tt.typ, body); !strings.Contains(got, want) {
				t.Errorf("findings lack %q:\n%s", want, got)
			}
		})
	}
	for _, typ := range []templates.DocumentType{templates.TypeRule, templates.TypeDoc} {
		t.Run("examples stay native in a "+string(typ), func(t *testing.T) {
			t.Parallel()
			if got := findingsFor(t, typ, templates.GenerateTemplate(typ)); strings.Contains(got, "## Examples in a") {
				t.Errorf("the %s template's own Examples heading is reported as foreign:\n%s", typ, got)
			}
		})
	}
}

func TestPrecision_ActorSubjectSections(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		typ      templates.DocumentType
		sections []string
	}{
		{name: "scenario", typ: templates.TypeScenario, sections: []string{"Subject", "Actors", "Flows", "Examples", "Open Questions"}},
		{name: "journey", typ: templates.TypeJourney, sections: []string{"Intent", "Actors", "Journeys", "Open Questions"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, section := range tt.sections {
				body := strings.Replace(templates.GenerateTemplate(tt.typ), "## "+section+"\n", "## Removed\n", 1)
				if got := findingsFor(t, tt.typ, body); !strings.Contains(got, "missing section: ## "+section) {
					t.Errorf("missing %s not reported: %v", section, got)
				}
			}
		})
	}
}

// TestOverCapRemedies_MatchBodyCaps holds the two declarations of the capped
// set against each other: a type in the cap table with no remedy would end its
// finding in a dangling dash, and a remedy for an uncapped type is dead text.
func TestOverCapRemedies_MatchBodyCaps(t *testing.T) {
	t.Parallel()
	if len(templates.MaxBodyLines) == 0 {
		t.Fatal("MaxBodyLines is empty, so this test proves nothing")
	}
	for typ := range templates.MaxBodyLines {
		if overCapRemedies[typ] == "" {
			t.Errorf("%s carries a body cap and no over-cap remedy", typ)
		}
	}
	for typ := range overCapRemedies {
		if _, capped := templates.MaxBodyLines[typ]; !capped {
			t.Errorf("%s carries an over-cap remedy and no body cap", typ)
		}
	}
}
