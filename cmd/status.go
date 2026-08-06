package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/docs"
	archsync "archcore-cli/internal/sync"
	"archcore-cli/templates"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check .archcore/ structure and document health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			issues := runStatus(cwd)
			if issues > 0 {
				return ErrAlreadyReported
			}
			return nil
		},
	}
}

// runStatus checks .archcore/ structure and documents. Returns issue count.
func runStatus(baseDir string) int {
	issues := runStatusChecks(baseDir)
	if issues > 0 {
		fmt.Println(display.Warn.Render(fmt.Sprintf("  %d issue(s) found", issues)))
	}
	return issues
}

// runStatusChecks performs all read-only validation checks without printing a summary.
// It is used by both the status and doctor commands.
func runStatusChecks(baseDir string) int {
	report := collectStatus(baseDir)
	report.print()
	return report.issues()
}

// collectStatus runs every check and returns the result as data, printing
// nothing. The hook path uses it; runStatusChecks prints what it returns.
func collectStatus(baseDir string) *statusReport {
	r := &statusReport{}

	if !config.DirExists(baseDir) {
		r.fail(".archcore/ directory not found")
		r.hint("Run 'archcore init' to set up")
		return r
	}

	r.ok(".archcore/ exists")

	// One scan serves every document check below. The post-tool-use hook runs
	// this on every document mutation, and the structural checks used to re-read
	// each file with os.ReadFile — bypassing the scan cache and doubling the I/O.
	//
	// Local only: mounted read-only globals belong to their own repo and the
	// consumer cannot fix their naming or tags. Their health is reported by
	// checkGlobalSources, which is also why these checks keep running when a
	// declared global is broken.
	corpus, scanErr := docs.ScanLocal(baseDir, true)

	r.merge(checkFiles(corpus, scanErr))
	r.merge(checkGlobalSources(baseDir))
	r.merge(checkTagHygiene(localDocuments(corpus), scanErr))
	r.merge(checkManifest(baseDir))
	return r
}

// checkGlobalSources reports the health of declared global sources. Fatal states
// (missing, not a directory, unreadable, self-overlap, duplicate path) and a
// present-but-invalid settings.json are counted as issues; an empty source is a
// warning only.
func checkGlobalSources(baseDir string) *statusReport {
	r := &statusReport{}

	inspections, err := docs.InspectGlobals(baseDir)
	if err != nil {
		r.failf("invalid .archcore/settings.json: %v", err)
		return r
	}
	for _, in := range inspections {
		switch {
		case in.State == docs.GlobalEmpty:
			r.warnf("%s", in.Message())
		case in.State.Fatal():
			r.failf("%s", in.Message())
		default: // GlobalOK
			r.okf("global source %q (%d document(s))", in.ID, in.Docs)
		}
	}
	return r
}

func checkFiles(corpus []docs.Document, scanErr error) *statusReport {
	r := &statusReport{}
	if scanErr != nil {
		r.failf("error scanning .archcore/: %v", scanErr)
		return r
	}

	for _, doc := range corpus {
		r.merge(checkNaming(doc.Path, doc.Filename))

		// Type validity only — no category placement check, the directory is
		// free-form.
		if doc.Type != "" && !templates.IsValidType(string(doc.Type)) {
			r.failf("%s: unknown document type %q", doc.Path, doc.Type)
			r.hintf("valid types: %s", strings.Join(templates.ValidTypes(), ", "))
		}

		r.merge(checkFrontmatter(doc.Content, doc.Path))
	}

	return r
}

func checkNaming(relPath, filename string) *statusReport {
	r := &statusReport{}
	name := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(name, ".")

	if len(parts) < 2 {
		r.failf("%s: filename must match <slug>.<type>.md", relPath)
		r.hint("example: oauth-user.adr.md")
		// Without a type segment there is no slug to validate.
		return r
	}

	slug := strings.Join(parts[:len(parts)-1], ".")
	if !templates.SlugRe.MatchString(slug) {
		r.failf("%s: slug must be lowercase alphanumeric with hyphens", relPath)
		r.hint("example: my-feature")
	}

	return r
}

// checkFrontmatter validates one document's structure. It takes the text the
// scan already read; a file the scan could not read arrives as an empty string
// and is reported as missing frontmatter, which is what it looks like from here.
func checkFrontmatter(raw, relPath string) *statusReport {
	r := &statusReport{}

	// Normalize CRLF and strip a UTF-8 BOM before structural checks, matching
	// templates.SplitDocument — a Windows-edited document is not "missing"
	// its frontmatter.
	content := strings.ReplaceAll(raw, "\r\n", "\n")
	content = strings.TrimPrefix(content, "\ufeff")

	// Each structural failure below returns: once the frontmatter block cannot
	// be located or parsed, every later check would report a consequence of the
	// same defect.
	if !strings.HasPrefix(content, "---\n") {
		r.failf("%s: missing YAML frontmatter", relPath)
		r.hint("file must start with --- delimiters")
		return r
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		r.failf("%s: missing closing --- delimiter", relPath)
		return r
	}

	fmContent := content[4 : 4+endIdx]

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		r.failf("%s: invalid YAML in frontmatter", relPath)
		r.hintf("%s", err.Error())
		return r
	}

	// Check required fields.
	for _, field := range []string{"title", "status"} {
		val, ok := fm[field]
		if !ok {
			r.failf("%s: missing required field %q", relPath, field)
		} else if str, isStr := val.(string); isStr && str == "" {
			r.failf("%s: missing required field %q", relPath, field)
		}
	}

	return r
}

func checkTagHygiene(corpus []docs.Document, scanErr error) *statusReport {
	r := &statusReport{}

	if scanErr != nil {
		r.failf("error scanning documents for tag check: %v", scanErr)
		return r
	}

	tagCount := make(map[string]int)
	for _, doc := range corpus {
		for _, tag := range doc.Tags {
			if !templates.TagRe.MatchString(tag) {
				r.failf("%s: invalid tag %q", doc.Path, tag)
				continue
			}
			tagCount[tag]++
		}
	}

	// Warn about singleton tags (possible typos). A warning is not an issue, so
	// a corpus with only singletons still reports hygiene as OK below.
	//
	// Sorted, because this output is user-visible and also reaches an agent
	// through the PostToolUse hook: ranging the map made an unchanged corpus
	// print its warnings in a different order every run.
	for _, tag := range slices.Sorted(maps.Keys(tagCount)) {
		if tagCount[tag] == 1 {
			r.warnf("tag %q is used only once (possible typo)", tag)
		}
	}

	if r.issues() == 0 {
		r.okf("Tag hygiene OK (%d unique tag(s))", len(tagCount))
	}

	return r
}

func checkManifest(baseDir string) *statusReport {
	r := &statusReport{}
	manifestPath := filepath.Join(baseDir, ".archcore", archsync.ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		r.ok("No sync manifest (first sync pending)")
		return r
	}
	if err != nil {
		r.failf("Cannot read sync manifest: %v", err)
		return r
	}

	jsonIssues := archsync.ValidateManifestJSON(data)
	for _, issue := range jsonIssues {
		r.failf("Sync manifest: %s", issue)
	}

	if len(jsonIssues) == 0 {
		var m archsync.Manifest
		if err := json.Unmarshal(data, &m); err == nil {
			if m.Files == nil {
				m.Files = make(map[string]string)
			}
			semIssues := archsync.ValidateManifest(&m)
			for _, issue := range semIssues {
				r.failf("Sync manifest: %s", issue)
			}

			danglingIssues := checkDanglingRelations(baseDir, m.Relations)
			for _, issue := range danglingIssues {
				r.failf("Sync manifest: %s", issue)
			}
			// One hint for the whole batch, not one per orphaned relation.
			if len(danglingIssues) > 0 {
				r.hint("Run 'archcore doctor --fix' to remove orphaned relations")
			}

			if len(semIssues) == 0 && len(danglingIssues) == 0 {
				r.okf("Sync manifest valid (%d file(s) tracked, %d relation(s))", len(m.Files), len(m.Relations))
			}
		} else {
			r.failf("Sync manifest: invalid JSON: %v", err)
		}
	}

	// Likewise: one closing hint per check, however many manifest issues there were.
	if r.issues() > 0 {
		r.hint("Delete .archcore/.sync-state.json and re-sync")
	}

	return r
}

// fixManifest removes orphaned relations from the sync manifest.
// Returns the number of relations removed and any error.
func fixManifest(baseDir string) (int, error) {
	manifestPath := filepath.Join(baseDir, ".archcore", archsync.ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return 0, nil // no manifest — nothing to fix
	}

	var m archsync.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, nil // corrupt manifest — checkManifest will report it
	}

	archcoreDir := filepath.Join(baseDir, ".archcore")
	removed := m.CleanupRelations(archcoreDir)
	if removed == 0 {
		return 0, nil
	}

	if err := archsync.SaveManifest(baseDir, &m); err != nil {
		return 0, fmt.Errorf("failed to save manifest after cleanup: %w", err)
	}
	return removed, nil
}

func checkDanglingRelations(baseDir string, relations []archsync.Relation) []string {
	var issues []string
	for _, rel := range relations {
		srcPath := filepath.Join(baseDir, ".archcore", rel.Source)
		if _, err := os.Stat(srcPath); errors.Is(err, fs.ErrNotExist) {
			issues = append(issues, fmt.Sprintf("relation source %q does not exist on disk", rel.Source))
		}
		tgtPath := filepath.Join(baseDir, ".archcore", rel.Target)
		if _, err := os.Stat(tgtPath); errors.Is(err, fs.ErrNotExist) {
			issues = append(issues, fmt.Sprintf("relation target %q does not exist on disk", rel.Target))
		}
	}
	return issues
}
