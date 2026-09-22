package templates

import "strings"

// sourceExtensions is the curated set of file extensions considered "source
// files" for the purpose of path-reference extraction in search_documents.
// Matching is case-insensitive; keys are stored lowercase with a leading dot.
var sourceExtensions = map[string]bool{
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".go":   true,
	".py":   true,
	".rb":   true,
	".rs":   true,
	".java": true,
	".c":    true,
	".cpp":  true,
	".h":    true,
	".hpp":  true,
	".cs":   true,
	".php":  true,
	".sh":   true,
	".sql":  true,
	// A Gherkin feature file is the executable half of a scenario, cited from a
	// spec's Conformance or a scenario's Anchors line — scenario-and-journey-types.spec §14.
	".feature": true,
	".md":      true,
	".yaml":    true,
	".yml":     true,
	".json":    true,
}

// IsSourceExtension reports whether ext is in the curated source-file
// extension list. Comparison is case-insensitive; ext may be given with or
// without a leading dot.
func IsSourceExtension(ext string) bool {
	if ext == "" {
		return false
	}
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return sourceExtensions[ext]
}
