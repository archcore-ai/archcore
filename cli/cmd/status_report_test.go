package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusReport_IssuesCountsOnlyFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(r *statusReport)
		want  int
	}{
		{name: "empty report", build: func(r *statusReport) {}, want: 0},
		{name: "ok lines are not issues", build: func(r *statusReport) { r.ok("fine"); r.ok("also fine") }, want: 0},
		{name: "warnings are not issues", build: func(r *statusReport) { r.warn("empty global"); r.warn("lonely tag") }, want: 0},
		{name: "hints are not issues", build: func(r *statusReport) { r.hint("try this") }, want: 0},
		{name: "failures count", build: func(r *statusReport) { r.fail("broken"); r.fail("also broken") }, want: 2},
		{
			name:  "mixed report counts only failures",
			build: func(r *statusReport) { r.ok("a"); r.warn("b"); r.fail("c"); r.hint("d") },
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &statusReport{}
			tt.build(r)
			if got := r.issues(); got != tt.want {
				t.Errorf("issues() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStatusReport_WriteToRendersEveryLevel(t *testing.T) {
	t.Parallel()
	r := &statusReport{}
	r.ok("all good")
	r.warn("be careful")
	r.fail("it broke")
	r.hint("do this instead")

	var buf bytes.Buffer
	r.writeTo(&buf)
	out := buf.String()

	for _, want := range []string{"all good", "be careful", "it broke", "do this instead"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if lines := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; lines != 4 {
		t.Errorf("rendered %d lines, want 4:\n%s", lines, out)
	}
}

func TestStatusReport_FailuresRespectsLimit(t *testing.T) {
	t.Parallel()
	r := &statusReport{}
	r.ok("noise")
	for _, s := range []string{"one", "two", "three", "four", "five", "six"} {
		r.failf("%s", s)
	}
	r.warn("more noise")

	if got := r.failures(5); len(got) != 5 || got[0] != "one" || got[4] != "five" {
		t.Errorf("failures(5) = %v, want the first five failures", got)
	}
	if got := r.failures(0); len(got) != 6 {
		t.Errorf("failures(0) returned %d, want all 6", len(got))
	}
}

// TestCollectStatus_PrintsNothing is the reason this refactor exists. The
// post-tool-use hook runs these checks while stdout carries the host's
// protocol JSON, so a single stray line corrupts the hook's response.
func TestCollectStatus_PrintsNothing(t *testing.T) {
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nBody.\n")

	var report *statusReport
	out := captureStdout(t, func() { report = collectStatus(base) })

	if out != "" {
		t.Errorf("collectStatus wrote %d bytes to stdout, want none:\n%s", len(out), out)
	}
	if len(report.lines) == 0 {
		t.Error("collectStatus returned an empty report — it should still collect lines")
	}
}

// TestCheckManifest_GroupHintPrintedOnce pins the distinction between a hint
// attached to one failure and a hint that summarizes a whole check. Two
// orphaned relations produce two failures but one remedy.
func TestCheckManifest_GroupHintPrintedOnce(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	manifest := `{"version":1,"files":{},"relations":[` +
		`{"source":"gone-a.adr.md","target":"gone-b.prd.md","type":"implements"},` +
		`{"source":"gone-c.adr.md","target":"gone-d.prd.md","type":"implements"}]}`
	writeArchcoreDoc(t, base, ".sync-state.json", manifest)

	report := checkManifest(base)

	var buf bytes.Buffer
	report.writeTo(&buf)
	out := buf.String()

	if got := strings.Count(out, "archcore doctor --fix"); got != 1 {
		t.Errorf("dangling-relation hint appeared %d times, want 1:\n%s", got, out)
	}
	if got := strings.Count(out, "Delete .archcore/.sync-state.json"); got != 1 {
		t.Errorf("closing manifest hint appeared %d times, want 1:\n%s", got, out)
	}
	if report.issues() != 4 {
		t.Errorf("issues() = %d, want 4 (two relations, each with a missing source and target)", report.issues())
	}
}
