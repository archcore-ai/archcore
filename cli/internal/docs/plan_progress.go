package docs

import (
	"regexp"
	"strings"
)

// PlanProgress is the state of a proof-tracked plan as its Tasks section records it.
type PlanProgress struct {
	// Phase is the heading text of the frontier phase, or empty when every task is proven.
	Phase string
	// Proven counts the frontier tasks that are checked and carry a proof line in the pinned form.
	Proven int
	// Total counts the numbered tasks of the frontier phase.
	Total int
	// Claimed counts checked tasks in any phase whose proof line is missing or not in the pinned form.
	Claimed int
	// Next is the line of the first unchecked frontier task, from its T<n> on.
	Next string
	// Waiting counts the bullets of the Waiting on section.
	Waiting int
}

var (
	planTaskRe    = regexp.MustCompile(`^\s*[0-9]+\.\s+\[([ xX])\]\s+(T[0-9]+\b.*)$`)
	planProofRe   = regexp.MustCompile(`^\s*-\s*proof:\s*(.*)$`)
	planPhaseRe   = regexp.MustCompile(`^###\s+(.+?)\s*$`)
	planSectionRe = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	planBulletRe  = regexp.MustCompile(`^\s*[-*]\s+\S`)
	// planProvenRe is the pinned proof form, `<check> @ <commit sha>`, with nothing after the sha.
	planProvenRe = regexp.MustCompile(`^[^\s—-].*\s@\s*[0-9a-fA-F]{7,40}$`)
)

type planTask struct {
	checked bool
	proven  bool
	text    string
}

type planPhase struct {
	name  string
	tasks []planTask
}

// ParsePlanProgress reads the Tasks and Waiting on sections of a plan body. It
// reports false for a legacy plan — one with no `[ ] T<n>` task in Tasks — so a
// caller shows nothing for it (session-start-context.spec#21-22).
func ParsePlanProgress(content string) (PlanProgress, bool) {
	var phases []planPhase
	var progress PlanProgress
	section := ""
	inFence := false
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if fence := strings.TrimSpace(line); strings.HasPrefix(fence, "```") || strings.HasPrefix(fence, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := planSectionRe.FindStringSubmatch(line); m != nil {
			section = m[1]
			continue
		}
		switch section {
		case "Tasks":
			phases = readTaskLine(phases, line)
		case "Waiting on":
			if planBulletRe.MatchString(line) {
				progress.Waiting++
			}
		}
	}
	tracked := false
	for _, phase := range phases {
		for _, task := range phase.tasks {
			tracked = true
			if task.checked && !task.proven {
				progress.Claimed++
			}
		}
	}
	if !tracked {
		return PlanProgress{}, false
	}
	for _, phase := range phases {
		open := ""
		proven := 0
		for _, task := range phase.tasks {
			if task.checked && task.proven {
				proven++
			}
			if !task.checked && open == "" {
				open = task.text
			}
		}
		if proven < len(phase.tasks) || len(phase.tasks) == 0 {
			progress.Phase, progress.Proven, progress.Total, progress.Next = phase.name, proven, len(phase.tasks), open
			break
		}
	}
	return progress, true
}

func readTaskLine(phases []planPhase, line string) []planPhase {
	if m := planPhaseRe.FindStringSubmatch(line); m != nil {
		return append(phases, planPhase{name: m[1]})
	}
	if m := planTaskRe.FindStringSubmatch(line); m != nil {
		if len(phases) == 0 {
			phases = append(phases, planPhase{name: "Tasks"})
		}
		last := &phases[len(phases)-1]
		last.tasks = append(last.tasks, planTask{checked: m[1] != " ", text: m[2]})
		return phases
	}
	if m := planProofRe.FindStringSubmatch(line); m != nil && len(phases) > 0 {
		last := &phases[len(phases)-1]
		if n := len(last.tasks); n > 0 {
			last.tasks[n-1].proven = planProvenRe.MatchString(strings.TrimSpace(m[1]))
		}
	}
	return phases
}
