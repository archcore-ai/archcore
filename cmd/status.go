package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/sync"
	"archcore-cli/templates"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check .archcore/ structure and document health",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			issues := runStatus(cwd)
			if issues > 0 {
				return fmt.Errorf("%d issue(s) found", issues)
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
	if !config.DirExists(baseDir) {
		fmt.Println(display.FailLine(".archcore/ directory not found"))
		fmt.Println(display.HintLine("Run 'archcore init' to set up"))
		return 1
	}

	issues := 0
	fmt.Println(display.CheckLine(".archcore/ exists"))
	issues += checkFiles(baseDir)
	docs, scanErr := tools.ScanDocuments(baseDir)
	issues += checkTagHygiene(docs, scanErr)
	issues += checkManifest(baseDir)
	return issues
}

func checkFiles(baseDir string) int {
	issues := 0
	archcoreDir := filepath.Join(baseDir, ".archcore")

	walkErr := templates.WalkArchcoreFiles(archcoreDir, func(path string, d fs.DirEntry) error {
		name := d.Name()

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		issues += checkNaming(relPath, name)

		// Check type validity (but no category placement check — directory is free-form).
		docType := templates.ExtractDocType(name)
		if docType != "" && !templates.IsValidType(docType) {
			issues++
			fmt.Println(display.FailLine(fmt.Sprintf("%s: unknown document type %q", relPath, docType)))
			fmt.Println(display.HintLine(fmt.Sprintf("valid types: %s", strings.Join(templates.ValidTypes(), ", "))))
		}

		issues += checkFrontmatter(path, relPath)
		return nil
	})
	if walkErr != nil {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("error scanning .archcore/: %v", walkErr)))
	}

	return issues
}

func checkNaming(relPath, filename string) int {
	issues := 0
	name := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(name, ".")

	if len(parts) < 2 {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: filename must match <slug>.<type>.md", relPath)))
		fmt.Println(display.HintLine("example: oauth-user.adr.md"))
		return issues
	}

	slug := strings.Join(parts[:len(parts)-1], ".")
	if !templates.SlugRe.MatchString(slug) {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: slug must be lowercase alphanumeric with hyphens", relPath)))
		fmt.Println(display.HintLine("example: my-feature"))
	}

	return issues
}

func checkFrontmatter(absPath, relPath string) int {
	issues := 0

	data, err := os.ReadFile(absPath)
	if err != nil {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: cannot read file: %v", relPath, err)))
		return issues
	}
	content := string(data)

	// Check for frontmatter delimiters.
	if !strings.HasPrefix(content, "---\n") {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: missing YAML frontmatter", relPath)))
		fmt.Println(display.HintLine("file must start with --- delimiters"))
		return issues
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: missing closing --- delimiter", relPath)))
		return issues
	}

	fmContent := content[4 : 4+endIdx]

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("%s: invalid YAML in frontmatter", relPath)))
		fmt.Println(display.HintLine(err.Error()))
		return issues
	}

	// Check required fields.
	for _, field := range []string{"title", "status"} {
		val, ok := fm[field]
		if !ok {
			issues++
			fmt.Println(display.FailLine(fmt.Sprintf("%s: missing required field %q", relPath, field)))
		} else if str, isStr := val.(string); isStr && str == "" {
			issues++
			fmt.Println(display.FailLine(fmt.Sprintf("%s: missing required field %q", relPath, field)))
		}
	}

	return issues
}

func checkTagHygiene(docs []tools.LocalDocument, scanErr error) int {
	issues := 0
	if scanErr != nil {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("error scanning documents for tag check: %v", scanErr)))
		return issues
	}

	tagCount := make(map[string]int)
	for _, doc := range docs {
		for _, tag := range doc.Tags {
			if !templates.TagRe.MatchString(tag) {
				issues++
				fmt.Println(display.FailLine(fmt.Sprintf("%s: invalid tag %q", doc.Path, tag)))
				continue
			}
			tagCount[tag]++
		}
	}

	// Warn about singleton tags (possible typos).
	for tag, count := range tagCount {
		if count == 1 {
			fmt.Println(display.WarnLine(fmt.Sprintf("tag %q is used only once (possible typo)", tag)))
		}
	}

	if issues == 0 {
		fmt.Println(display.CheckLine(fmt.Sprintf("Tag hygiene OK (%d unique tag(s))", len(tagCount))))
	}

	return issues
}

func checkManifest(baseDir string) int {
	issues := 0
	manifestPath := filepath.Join(baseDir, ".archcore", sync.ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println(display.CheckLine("No sync manifest (first sync pending)"))
		return 0
	}
	if err != nil {
		fmt.Println(display.FailLine(fmt.Sprintf("Cannot read sync manifest: %v", err)))
		return 1
	}

	jsonIssues := sync.ValidateManifestJSON(data)
	for _, issue := range jsonIssues {
		issues++
		fmt.Println(display.FailLine(fmt.Sprintf("Sync manifest: %s", issue)))
	}

	if len(jsonIssues) == 0 {
		var m sync.Manifest
		if err := json.Unmarshal(data, &m); err == nil {
			if m.Files == nil {
				m.Files = make(map[string]string)
			}
			semIssues := sync.ValidateManifest(&m)
			for _, issue := range semIssues {
				issues++
				fmt.Println(display.FailLine(fmt.Sprintf("Sync manifest: %s", issue)))
			}

			danglingIssues := checkDanglingRelations(baseDir, m.Relations)
			for _, issue := range danglingIssues {
				issues++
				fmt.Println(display.FailLine(fmt.Sprintf("Sync manifest: %s", issue)))
			}
			if len(danglingIssues) > 0 {
				fmt.Println(display.HintLine("Run 'archcore doctor --fix' to remove orphaned relations"))
			}

			if len(semIssues) == 0 && len(danglingIssues) == 0 {
				fmt.Println(display.CheckLine(fmt.Sprintf("Sync manifest valid (%d file(s) tracked, %d relation(s))", len(m.Files), len(m.Relations))))
			}
		} else {
			issues++
			fmt.Println(display.FailLine(fmt.Sprintf("Sync manifest: invalid JSON: %v", err)))
		}
	}

	if issues > 0 {
		fmt.Println(display.HintLine("Delete .archcore/.sync-state.json and re-sync"))
	}

	return issues
}

// fixManifest removes orphaned relations from the sync manifest.
// Returns the number of relations removed and any error.
func fixManifest(baseDir string) (int, error) {
	manifestPath := filepath.Join(baseDir, ".archcore", sync.ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return 0, nil // no manifest — nothing to fix
	}

	var m sync.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, nil // corrupt manifest — checkManifest will report it
	}

	archcoreDir := filepath.Join(baseDir, ".archcore")
	removed := m.CleanupRelations(archcoreDir)
	if removed == 0 {
		return 0, nil
	}

	if err := sync.SaveManifest(baseDir, &m); err != nil {
		return 0, fmt.Errorf("failed to save manifest after cleanup: %w", err)
	}
	return removed, nil
}

func checkDanglingRelations(baseDir string, relations []sync.Relation) []string {
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
