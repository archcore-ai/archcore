package docs

import (
	"strings"
	"testing"
)

func TestParsePlanProgress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		tracked bool
		want    PlanProgress
	}{
		{
			name:    "legacy numbered tasks",
			content: "## Tasks\n\n### Phase 1: Core\n\n1. Add the model.\n2. Add the API.\n\n## Acceptance Criteria\n\n- [ ] Criterion 1\n",
			tracked: false,
		},
		{
			name:    "acceptance checkboxes alone stay legacy",
			content: "## Goal\n\nShip it.\n\n## Acceptance Criteria\n\n- [x] Criterion 1\n1. [ ] Criterion 2\n",
			tracked: false,
		},
		{
			name: "frontier with one proven task",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - files: @src/cache.js\n   - proof: cache test @ e67a927\n\n" +
				"2. [ ] T2 Add the cap\n\n   - proof: —\n\n### Phase 2: Index\n\nCovers: index#1; expand when Phase 1 is proven.\n\n" +
				"## Waiting on\n\n- Approval of the schema (owner: maintainer)\n- A host run (owner: author)\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Proven: 1, Total: 2, Next: "T2 Add the cap", Waiting: 2},
		},
		{
			name:    "checked task without proof is a claim",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - proof: —\n\n2. [ ] T2 Add the cap\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Proven: 0, Total: 2, Claimed: 1, Next: "T2 Add the cap"},
		},
		{
			name:    "a proof outside the pinned form is a claim",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - proof: e67a927 — 6 tests pass\n\n2. [x] T2 Add the cap\n\n   - proof: cap test @ e67a927; failed before\n\n3. [ ] T3 Add the index\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Proven: 0, Total: 3, Claimed: 2, Next: "T3 Add the index"},
		},
		{
			name:    "proven phase points at the unexpanded one",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - proof: cache test @ e67a927\n\n### Phase 2: Index\n\nCovers: index#1.\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 2: Index", Total: 0},
		},
		{
			name:    "every phase proven",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [X] T1 Add the cache\n\n   - proof: cache test @ e67a927\n",
			tracked: true,
			want:    PlanProgress{},
		},
		{
			name:    "legacy checkbox tasks without T<n> stay legacy",
			content: "## Tasks\n\n1. [x] Record the decision in an ADR\n2. [ ] Update the guide\n",
			tracked: false,
		},
		{
			name:    "tasks before any phase heading form the Tasks phase",
			content: "## Tasks\n\n1. [ ] T1 Add the cache\n",
			tracked: true,
			want:    PlanProgress{Phase: "Tasks", Total: 1, Next: "T1 Add the cache"},
		},
		{
			name:    "a phase checked only by claims stays the frontier",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 Add the cache\n\n   - proof: —\n\n### Phase 2: Index\n\nCovers: index#1.\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Total: 1, Claimed: 1},
		},
		{
			name: "proof form boundaries",
			content: "## Tasks\n\n### Phase 1: Cache\n\n1. [x] T1 a\n\n   - proof: — @ e67a927\n\n2. [x] T2 b\n\n   - proof: t @ e67a92\n\n" +
				"3. [x] T3 c\n\n   - proof: t @ " + strings.Repeat("a", 41) + "\n\n4. [x] T4 d\n\n   - proof: t @ E67A927\n\n" +
				"5. [x] T5 e\n\n   - proof: t @ " + strings.Repeat("a", 40) + "\n\n6. [ ] T6 f\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Proven: 2, Total: 6, Claimed: 3, Next: "T6 f"},
		},
		{
			name:    "CRLF line endings and tilde fences",
			content: "## Tasks\r\n\r\n~~~markdown\r\n1. [x] T9 Example\r\n~~~\r\n\r\n### Phase 1: Cache\r\n\r\n1. [x] T1 Add the cache\r\n\r\n   - proof: cache test @ e67a927\r\n\r\n2. [ ] T2 Add the cap\r\n",
			tracked: true,
			want:    PlanProgress{Phase: "Phase 1: Cache", Proven: 1, Total: 2, Next: "T2 Add the cap"},
		},
		{
			name:    "tasks inside a fence do not count",
			content: "## Tasks\n\n```markdown\n1. [ ] T1 Example\n```\n\n1. Add the model.\n",
			tracked: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, tracked := ParsePlanProgress(tt.content)
			if tracked != tt.tracked {
				t.Fatalf("tracked = %v, want %v", tracked, tt.tracked)
			}
			if got != tt.want {
				t.Errorf("progress = %+v, want %+v", got, tt.want)
			}
		})
	}
}
