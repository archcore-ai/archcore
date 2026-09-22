package cmd

import (
	"fmt"
	"io"
	"os"

	"archcore-cli/internal/display"
)

// Structured status report.
//
// Checks return data so the hook path can run them without printing: on a hook,
// stdout carries the host's protocol JSON.
//
// The model is deliberately flat — an ordered list of rendered lines. Hints are
// lines too, whether they belong to one failure or summarize a whole check, so
// output order is preserved by construction and a group hint stays a single
// line instead of being duplicated onto every failure it follows.

// checkLevel classifies one report line. Only levelFail counts as an issue:
// a warning is informational (an empty global source, a tag used once) and must
// not turn `archcore status` into a failing command.
type checkLevel int

const (
	levelOK checkLevel = iota
	levelWarn
	levelFail
	levelHint
)

// reportLine is one line of status output.
type reportLine struct {
	level checkLevel
	text  string
}

// statusReport accumulates the lines one or more checks produced.
type statusReport struct {
	lines []reportLine
}

// Plain and formatted forms are separate so `go vet` can check the verbs.
// A single variadic method that skipped Sprintf when no arguments were passed
// made r.fail(text) %-safe and r.fail(text, x) not, and vet's printf-wrapper
// analysis recognizes the f-suffix convention but not the conditional one.

func (r *statusReport) ok(text string)   { r.add(levelOK, text) }
func (r *statusReport) warn(text string) { r.add(levelWarn, text) }
func (r *statusReport) fail(text string) { r.add(levelFail, text) }
func (r *statusReport) hint(text string) { r.add(levelHint, text) }

func (r *statusReport) okf(format string, a ...any)   { r.add(levelOK, fmt.Sprintf(format, a...)) }
func (r *statusReport) warnf(format string, a ...any) { r.add(levelWarn, fmt.Sprintf(format, a...)) }
func (r *statusReport) failf(format string, a ...any) { r.add(levelFail, fmt.Sprintf(format, a...)) }
func (r *statusReport) hintf(format string, a ...any) { r.add(levelHint, fmt.Sprintf(format, a...)) }

func (r *statusReport) add(level checkLevel, text string) {
	r.lines = append(r.lines, reportLine{level: level, text: text})
}

// merge appends another check's lines, preserving order.
func (r *statusReport) merge(other *statusReport) {
	if other == nil {
		return
	}
	r.lines = append(r.lines, other.lines...)
}

// issues counts the failures. Warnings and hints are excluded by design.
func (r *statusReport) issues() int {
	n := 0
	for _, l := range r.lines {
		if l.level == levelFail {
			n++
		}
	}
	return n
}

// failures returns the failure texts, most important first, capped at limit.
// The hook path uses it to surface problems without writing to stdout; a limit
// of zero or less returns them all.
func (r *statusReport) failures(limit int) []string {
	var out []string
	for _, l := range r.lines {
		if l.level != levelFail {
			continue
		}
		out = append(out, l.text)
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
}

// writeTo renders the report. Callers that want the terminal use print().
func (r *statusReport) writeTo(w io.Writer) {
	for _, l := range r.lines {
		switch l.level {
		case levelOK:
			fmt.Fprintln(w, display.CheckLine(l.text))
		case levelWarn:
			fmt.Fprintln(w, display.WarnLine(l.text))
		case levelFail:
			fmt.Fprintln(w, display.FailLine(l.text))
		case levelHint:
			fmt.Fprintln(w, display.HintLine(l.text))
		}
	}
}

// print renders the report to os.Stdout.
//
// os.Stdout, resolved at call time, is load-bearing: the command tests swap
// os.Stdout around Execute() and read the pipe, while the cobra writer they
// also set is never read. Routing this through cmd.OutOrStdout() would make
// every one of those tests observe an empty command.
func (r *statusReport) print() { r.writeTo(os.Stdout) }
