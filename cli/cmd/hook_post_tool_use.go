package cmd

import (
	"context"
	"fmt"
	"strings"

	"archcore-cli/internal/advisory"
	"archcore-cli/internal/docs"
	archsync "archcore-cli/internal/sync"
)

// Post-write checks.
//
// These run after a document mutation and can only report. The tool has already
// executed, so there is nothing to block — which is why this path always exits
// zero and why a failure in any one check must not silence the others.

// maxValidationIssues caps how many problems are reported at once. A wall of
// findings is read as noise; the first few are read as a to-do list.
const maxValidationIssues = 5

// postToolUseHandler runs the checks a document mutation warrants.
func postToolUseHandler(_ context.Context, r hookRequest) hookDecision {
	baseDir := r.baseDir
	tool := r.payload.toolName()
	// Gate on "did this change anything", not on "is this ours". Cursor installs
	// its post-MCP event with no matcher, so every read would otherwise scan the
	// whole corpus and report on a call that changed nothing.
	if !isMutatingArchcoreTool(tool) {
		return allowHook()
	}

	var sections []string
	if s := validationAdvisory(baseDir); s != "" {
		sections = append(sections, s)
	}
	if s := cascadeAdvisory(baseDir, tool, r.payload.docPath()); s != "" {
		sections = append(sections, s)
	}
	if s := advisory.Precision(baseDir, tool, r.payload.docPath()); s != "" {
		sections = append(sections, s)
	}

	if len(sections) == 0 {
		return allowHook()
	}
	return adviseHook(strings.Join(sections, "\n\n"))
}

// validationAdvisory reports structural problems in the knowledge base.
//
// It runs the checks in-process rather than shelling out to `archcore doctor`:
// the hook's stdout carries the host protocol, so a check that prints cannot be
// used here at all. collectStatus returns the same findings as data.
func validationAdvisory(baseDir string) string {
	report := collectStatus(baseDir)
	failures := report.failures(maxValidationIssues)
	if len(failures) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("[Archcore Validation] Issues found after the write:\n")
	for _, f := range failures {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	if rest := report.issues() - len(failures); rest > 0 {
		fmt.Fprintf(&b, "  … and %d more — run archcore doctor\n", rest)
	}
	return strings.TrimRight(b.String(), "\n")
}

// cascadeRelations are the relation types that make one document depend on
// another's content. "related" is excluded: it is an association, not a
// dependency, so an update on one side implies nothing about the other.
var cascadeRelations = map[archsync.RelationType]bool{
	archsync.RelImplements: true,
	archsync.RelDependsOn:  true,
	archsync.RelExtends:    true,
}

// cascadeAdvisory names the documents that may need review because the document
// they build on just changed.
func cascadeAdvisory(baseDir, tool, docPath string) string {
	if !strings.HasSuffix(tool, "update_document") || docPath == "" {
		return "" // only an update can invalidate a dependent
	}

	manifest, err := archsync.LoadManifest(baseDir)
	if err != nil {
		return ""
	}

	// Manifest paths carry no ".archcore/" prefix; tool paths do.
	_, incoming := manifest.RelationsFor(docs.NormalizeRelPath(docPath))
	var affected []string
	for _, rel := range incoming {
		if cascadeRelations[rel.Type] {
			affected = append(affected, fmt.Sprintf("  - %s (%s this document)", rel.Source, rel.Type))
		}
	}
	if len(affected) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Cascade] %s changed. Documents that may need review:\n", docs.NormalizeRelPath(docPath))
	b.WriteString(strings.Join(affected, "\n"))
	return b.String()
}
