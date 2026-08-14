package cmd

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"archcore-cli/internal/advisory"
	"archcore-cli/internal/docs"
	"archcore-cli/internal/git"
	"archcore-cli/internal/stamp"
	archsync "archcore-cli/internal/sync"
	"archcore-cli/templates"
)

// maxSessionTags caps the number of tags emitted in SessionStart context.
// Capped at 20 (top-N by frequency) to limit static token overhead per session
// while preserving enough coverage for projects with rich tag namespaces.
const maxSessionTags = 20

// Session recap budget. Output size is a function of these caps, not of corpus
// size (session-start-context.spec).
const (
	// maxRecapDocs bounds the document lines across both blocks, so output size
	// is a function of the budget rather than of corpus size.
	maxRecapDocs = 24
	// acceptedFloor reserves room for recent decisions even when many drafts are
	// open; without it a repository mid-refactor would show only its own mess.
	acceptedFloor = 6
	// recentWindow bounds "recently accepted". Older decisions are still findable
	// through the MCP tools; they are just not news.
	recentWindow = 30 * 24 * time.Hour
)

// sessionDocCounts carries the banner counts out of buildSessionContext: the
// local corpus size and the mounted global document total.
type sessionDocCounts struct {
	local  int
	global int
}

// buildSessionContext generates the session-start context string
// that is injected into agents at session start.
//
// Callers embed the returned string in a per-host JSON envelope. This function
// owns the text inside that envelope and never the envelope itself — the plugin
// splices its own advisories into the same document on Copilot, and a change to
// the wrapper would break that splice silently.
func buildSessionContext(ctx context.Context, baseDir string) (string, sessionDocCounts) {
	// Local only, content included. Global content stays behind the MCP read
	// tools; scanning it here would read and parse every global document only
	// to discard the bodies. The GLOBALS block below needs counts, not content,
	// and takes them from the InspectGlobals walk. The local content is free —
	// the frontmatter parse already reads each file — and the staleness
	// correlation below needs it, which is what keeps this to one scan per
	// session start.
	corpus, _ := docs.ScanLocal(baseDir, true)

	// Global health and counts come from InspectGlobals rather than from a scan
	// error: it classifies every failure mode the scan reports plus two the scan
	// cannot see — an empty source scans cleanly, and an invalid settings.json
	// makes the read path silently observe no globals at all.
	inspections, iErr := docs.InspectGlobals(baseDir)

	var b strings.Builder
	b.WriteString("[Archcore — Git-native context for AI coding agents]\n")
	b.WriteString("You have MCP tools available: list_documents, get_document, search_documents, create_document, update_document, remove_document, add_relation, remove_relation, list_relations.\n")
	if iErr != nil {
		// Fail closed to a warning: no GLOBALS block renders on an unverifiable
		// declaration (session-globals-disclosure.spec clause 18).
		fmt.Fprintf(&b, "\n⚠ invalid .archcore/settings.json: %v — global sources not loaded; context limited to local documents\n", iErr)
	}

	writeCorpusLine(&b, corpus, len(inspections) > 0)
	writeBranchLine(ctx, &b, baseDir)
	globalDocs := writeGlobalsBlock(&b, inspections)
	writeRecap(&b, corpus)

	if advisory := advisory.Staleness(ctx, baseDir, stamp.DirFor("staleness-stamps"), corpus); advisory != "" {
		fmt.Fprintf(&b, "\n%s", advisory)
	}

	// Tag frequencies cover every local document, rejected included: a tag used
	// only by a rejected document is still part of the project's vocabulary and
	// the agent should reuse it rather than invent a synonym.
	tagFreq := make(map[string]int)
	for _, doc := range corpus {
		for _, tag := range doc.Tags {
			tagFreq[tag]++
		}
	}
	if len(tagFreq) > 0 {
		type tagCount struct {
			tag   string
			count int
		}
		sorted := make([]tagCount, 0, len(tagFreq))
		for tag, count := range tagFreq {
			sorted = append(sorted, tagCount{tag, count})
		}
		slices.SortFunc(sorted, func(a, b tagCount) int {
			if c := cmp.Compare(b.count, a.count); c != 0 {
				return c
			}
			return cmp.Compare(a.tag, b.tag)
		})
		limit := min(maxSessionTags, len(sorted))
		tags := make([]string, limit)
		for i := range limit {
			tags[i] = sorted[i].tag
		}
		fmt.Fprintf(&b, "\nEXISTING TAGS: %s\n", strings.Join(tags, ", "))
	}

	// Document relations summary.
	if m, mErr := archsync.LoadManifest(baseDir); mErr == nil && len(m.Relations) > 0 {
		fmt.Fprintf(&b, "\nDOCUMENT RELATIONS: %d relation(s) stored.\n", len(m.Relations))
		b.WriteString("  Use list_relations, add_relation, remove_relation MCP tools to manage.\n")
	}

	b.WriteString("\nRefer to MCP server instructions for document types, workflow rules, and usage guidance.\n")

	// The local count is every local document, including rejected ones: it
	// feeds the "N docs" banner, which reports what the project holds, not what
	// this recap chose to show.
	return b.String(), sessionDocCounts{local: len(corpus), global: globalDocs}
}

// Ceilings for the GLOBALS block. Output size is a function of these and never
// of corpus size (session-globals-disclosure.spec clauses 8-11).
const (
	maxGlobalsSources = 8
	maxGlobalsDirs    = 6
)

// globalsPrecedenceLine is the exact sentence clause 13 of
// session-globals-disclosure.spec pins.
const globalsPrecedenceLine = "Local documents take precedence over same-topic globals."

// writeGlobalsBlock renders the GLOBALS block from the inspections and returns
// the mounted document total across healthy sources, for the banner.
//
// Every number is filename- or dirname-derived during the InspectGlobals walk —
// no global document content is read here (session-globals-disclosure.spec).
func writeGlobalsBlock(b *strings.Builder, inspections []docs.GlobalInspection) int {
	if len(inspections) == 0 {
		return 0
	}
	// The banner total sums every healthy source independently of the render
	// ceiling below: a source the block truncates away is still mounted, and
	// clause 15 of session-globals-disclosure.spec asks for the total.
	globalDocs := 0
	for _, in := range inspections {
		if in.State == docs.GlobalOK {
			globalDocs += in.Docs
		}
	}
	b.WriteString("\nGLOBALS (read-only, query via MCP read tools):\n")
	okLines := 0
	for i, in := range inspections {
		if i == maxGlobalsSources {
			fmt.Fprintf(b, "  … and %d more sources\n", len(inspections)-i)
			break
		}
		if in.State.Fatal() {
			fmt.Fprintf(b, "  ⚠ %s — clone it or fix .archcore/settings.json\n", in.Message())
			continue
		}
		if in.State == docs.GlobalEmpty {
			fmt.Fprintf(b, "  ⚠ %s\n", in.Message())
			continue
		}
		okLines++
		fmt.Fprintf(b, "  - %s — %d docs (%s)%s\n", in.ID, in.Docs, categorySummary(in.DocsByCategory), dirSummary(in.TopDirs))
	}
	if okLines > 0 {
		b.WriteString("  " + globalsPrecedenceLine + "\n")
	}
	return globalDocs
}

// categorySummary renders per-category counts in the fixed category order the
// CORPUS line uses, skipping empty categories.
func categorySummary(byCategory map[templates.Category]int) string {
	var parts []string
	for _, cat := range []templates.Category{templates.CategoryKnowledge, templates.CategoryVision, templates.CategoryExperience} {
		if n := byCategory[cat]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", cat, n))
		}
	}
	return strings.Join(parts, ", ")
}

// dirSummary renders the top-level directory counts — the topical map that lets
// an agent phrase a query against a corpus it has never read. Ordered by count
// descending, ties alphabetical, cut at maxGlobalsDirs with the drop named.
func dirSummary(topDirs map[string]int) string {
	if len(topDirs) == 0 {
		return ""
	}
	type dirCount struct {
		dir   string
		count int
	}
	sorted := make([]dirCount, 0, len(topDirs))
	for dir, count := range topDirs {
		sorted = append(sorted, dirCount{dir, count})
	}
	slices.SortFunc(sorted, func(a, c dirCount) int {
		if d := cmp.Compare(c.count, a.count); d != 0 {
			return d
		}
		return cmp.Compare(a.dir, c.dir)
	})
	limit := min(maxGlobalsDirs, len(sorted))
	parts := make([]string, limit)
	for i := range limit {
		parts[i] = fmt.Sprintf("%s/ %d", sorted[i].dir, sorted[i].count)
	}
	out := " · " + strings.Join(parts, ", ")
	if rest := len(sorted) - limit; rest > 0 {
		out += fmt.Sprintf(" … and %d more dirs", rest)
	}
	return out
}

// writeCorpusLine summarizes the whole corpus in one line: what exists, by
// category and by status. The rejected count appears here and nowhere else —
// enough to tell the agent refused work exists without pushing it as guidance.
//
// hasGlobals switches the label to "local documents" so the CORPUS count and
// the GLOBALS totals cannot read as a contradiction
// (session-globals-disclosure.spec clause 14).
func writeCorpusLine(b *strings.Builder, corpus []docs.Document, hasGlobals bool) {
	label := "documents"
	if hasGlobals {
		label = "local documents"
	}
	if len(corpus) == 0 {
		fmt.Fprintf(b, "\nCORPUS: no %s yet — use create_document to record the first one.\n", label)
		return
	}

	byCategory := make(map[templates.Category]int, 3)
	byStatus := make(map[templates.DocStatus]int, 3)
	for _, d := range corpus {
		byCategory[d.Category]++
		byStatus[effectiveStatus(d)]++
	}

	var cats []string
	for _, cat := range []templates.Category{templates.CategoryKnowledge, templates.CategoryVision, templates.CategoryExperience} {
		if n := byCategory[cat]; n > 0 {
			cats = append(cats, fmt.Sprintf("%s %d", cat, n))
		}
	}

	var statuses []string
	for _, st := range []templates.DocStatus{templates.StatusDraft, templates.StatusAccepted, templates.StatusRejected} {
		if n := byStatus[st]; n > 0 {
			statuses = append(statuses, fmt.Sprintf("%s %d", st, n))
		}
	}

	fmt.Fprintf(b, "\nCORPUS: %d %s — %s · %s\n",
		len(corpus), label, strings.Join(cats, ", "), strings.Join(statuses, ", "))
}

// writeBranchLine names the checked-out branch. It is omitted entirely outside a
// git working tree, on a detached HEAD, or when git is unavailable — an absent
// line is better than one that says "unknown".
func writeBranchLine(ctx context.Context, b *strings.Builder, baseDir string) {
	branch, err := git.CurrentBranch(ctx, baseDir)
	if err != nil || branch == "" {
		return
	}
	fmt.Fprintf(b, "BRANCH: %s\n", branch)
}

// writeRecap emits the two blocks that carry the moving parts of the project:
// what is open, and what was decided recently. Rejected documents appear in
// neither — pushing a refused decision reads as guidance and invites the agent
// to re-walk a dead end. The read tools still return them on request.
func writeRecap(b *strings.Builder, corpus []docs.Document) {
	var drafts, accepted []docs.Document
	cutoff := time.Now().Add(-recentWindow)
	for _, d := range corpus {
		if !d.InAgentContext() {
			continue
		}
		//exhaustive:ignore // A rejected document is excluded from the recap upstream; naming it here would imply it can reach this switch.
		switch effectiveStatus(d) {
		case templates.StatusDraft:
			drafts = append(drafts, d)
		case templates.StatusAccepted:
			if d.ModTime.After(cutoff) {
				accepted = append(accepted, d)
			}
		}
	}
	byNewest := func(a, c docs.Document) int { return c.ModTime.Compare(a.ModTime) }
	slices.SortStableFunc(drafts, byNewest)
	slices.SortStableFunc(accepted, byNewest)

	// Drafts claim the budget first — open work is what a session resumes — but
	// recent decisions keep a floor so a busy repository still shows them.
	nDraft := min(len(drafts), maxRecapDocs-min(len(accepted), acceptedFloor))
	nAccepted := min(len(accepted), maxRecapDocs-nDraft)

	writeRecapBlock(b, "IN PROGRESS (draft, newest first)", drafts, nDraft, "status=\"draft\"")
	writeRecapBlock(b, "RECENTLY ACCEPTED (last 30 days)", accepted, nAccepted, "status=\"accepted\"")
}

// writeRecapBlock renders one block, truncating from the end and naming what was
// dropped so a truncated list never reads as a complete one.
func writeRecapBlock(b *strings.Builder, heading string, corpus []docs.Document, limit int, filterHint string) {
	if len(corpus) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s:\n", heading)
	for _, d := range corpus[:limit] {
		// The full path, not the filename: it is what get_document takes.
		if d.Title != "" {
			fmt.Fprintf(b, "  - %s — %q\n", d.Path, d.Title)
			continue
		}
		fmt.Fprintf(b, "  - %s\n", d.Path)
	}
	if rest := len(corpus) - limit; rest > 0 {
		fmt.Fprintf(b, "  … and %d more — list_documents(%s)\n", rest, filterHint)
	}
}

// effectiveStatus resolves a document's status, treating an absent or unknown
// value as draft — the documented default for a new document.
func effectiveStatus(d docs.Document) templates.DocStatus {
	if templates.IsValidStatus(d.Status) {
		return d.Status
	}
	return templates.StatusDraft
}

// localDocuments returns only documents owned by the primary project, dropping
// mounted read-only global sources. Globals are surfaced exclusively through the
// MCP read tools (list/get/search), never in CLI status or session-start context.
func localDocuments(corpus []docs.Document) []docs.Document {
	out := make([]docs.Document, 0, len(corpus))
	for _, d := range corpus {
		if d.IsLocal() {
			out = append(out, d)
		}
	}
	return out
}
