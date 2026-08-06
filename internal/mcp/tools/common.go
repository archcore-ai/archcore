package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// loadGlobalsFailClosed loads declared global sources for a write guard. On
// failure it returns a non-nil tool result: a corrupt settings.json must fail
// closed so a mutation can never slip past an unverifiable global list.
func loadGlobalsFailClosed(baseDir string) ([]config.GlobalSource, *mcp.CallToolResult) {
	globals, err := config.LoadGlobals(baseDir)
	if err != nil {
		return nil, errorResult("cannot verify global sources: settings.json is unreadable")
	}
	return globals, nil
}

// validateTags checks that every tag matches the required format.
func validateTags(tags []string) error {
	for _, tag := range tags {
		if !templates.TagRe.MatchString(tag) {
			hint := strings.ToLower(tag)
			if hint != tag && templates.TagRe.MatchString(hint) {
				return fmt.Errorf("invalid tag %q — did you mean %q?", tag, hint)
			}
			return fmt.Errorf("invalid tag %q — must be lowercase alphanumeric with hyphens, underscores, colons, or pipes (e.g. \"frontend\", \"team-platform\", \"team:payments\", \"payment_team\", \"some|flag\")", tag)
		}
	}
	return nil
}

// parseTags validates and normalizes a tag list. Returns nil for empty input.
func parseTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	if err := validateTags(tags); err != nil {
		return nil, err
	}
	return normalizeTags(tags), nil
}

// normalizeTags sorts and deduplicates tags. Returns nil for empty input.
// Always operates on a clone to avoid mutating the caller's slice.
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := slices.Clone(tags)
	slices.Sort(out)
	out = slices.Compact(out)
	return out
}

func errorResult(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(msg)
}

// sanitizeError builds a client-safe error message "<action>: <detail>". When
// err (anywhere in its chain) is a *fs.PathError or *os.LinkError, detail is a
// fixed I/O class instead of err.Error(), which embeds an absolute filesystem
// path (no-absolute-paths-in-mcp-errors.rule). Any other error keeps its own
// text — validation errors are built from relative manifest/settings keys and
// are safe and valuable diagnostics.
func sanitizeError(action string, err error) string {
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	if errors.As(err, &pathErr) || errors.As(err, &linkErr) {
		return action + ": " + describeIOClass(err)
	}
	return action + ": " + err.Error()
}

// SanitizeError exposes sanitizeError to injected executors (the cmd-layer
// host-wiring installer) so per-item error strings embedded in a tool's
// success payload obey the same no-absolute-paths rule as top-level errors.
func SanitizeError(action string, err error) string {
	return sanitizeError(action, err)
}

// describeIOClass maps an OS-level error to a short, path-free description.
func describeIOClass(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "file not found"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.Is(err, fs.ErrExist):
		return "file already exists"
	default:
		return "file system error"
	}
}

// buildDocumentFile reconstructs a full document file from frontmatter fields and body.
func buildDocumentFile(title string, status templates.DocStatus, tags []string, body string) string {
	var buf strings.Builder
	buf.WriteString("---\n")
	fmt.Fprintf(&buf, "title: %q\n", title)
	fmt.Fprintf(&buf, "status: %s\n", status)
	if len(tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range tags {
			fmt.Fprintf(&buf, "  - %q\n", tag)
		}
	}
	buf.WriteString("---\n\n")
	buf.WriteString(body)
	return buf.String()
}

// stripFrontmatter removes YAML frontmatter from content if present.
// This prevents duplicate frontmatter when callers (e.g. AI agents)
// include frontmatter in the content parameter despite the tool
// description specifying body-only content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "---\n") {
		return content
	}
	_, body, _ := templates.SplitDocument([]byte(content))
	return body
}
