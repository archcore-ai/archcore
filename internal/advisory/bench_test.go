package advisory

import (
	"testing"

	"archcore-cli/internal/docs"
	"archcore-cli/internal/testsupport"
)

// Benchmarks for the two advisory paths that run inside a host budget.
//
// Each iteration resets the scan cache, because a hook is a short-lived process
// that always starts cold. Measuring the warm path would report a number the
// real invocation never sees.
//
// These are measurements, not assertions: a wall-clock threshold in a test is
// flaky on shared CI and would be the first thing disabled. The budget claim is
// asserted deterministically by the scan-count tests in cmd.

func benchCorpus(tb testing.TB, n int) string {
	tb.Helper()
	return testsupport.BuildCorpus(tb, tb.TempDir(), n)
}

// BenchmarkCodeAlignment runs the pre-write path. It is the one that blocks the
// user: the installed host timeout is one second and the budget is p95 ≤ 150 ms.
func BenchmarkCodeAlignment(b *testing.B) {
	for _, n := range []int{300, 3000} {
		b.Run(sizeName(n), func(b *testing.B) {
			base := benchCorpus(b, n)
			b.ReportAllocs()
			for b.Loop() {
				docs.ResetCache()
				CodeAlignment(base, "src/api/handler.go")
			}
		})
	}
}

// BenchmarkPrecision runs one post-write document check. Post-write only
// reports, so its budget is the looser one (3 s).
func BenchmarkPrecision(b *testing.B) {
	for _, n := range []int{300, 3000} {
		b.Run(sizeName(n), func(b *testing.B) {
			base := benchCorpus(b, n)
			b.ReportAllocs()
			for b.Loop() {
				docs.ResetCache()
				Precision(base, "mcp__archcore__update_document", "domain00/doc-00000.rule.md")
			}
		})
	}
}

func sizeName(n int) string {
	if n >= 1000 {
		return string(rune('0'+n/1000)) + "000docs"
	}
	return string(rune('0'+n/100)) + "00docs"
}
