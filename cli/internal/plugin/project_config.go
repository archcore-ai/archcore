package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"archcore-cli/internal/git"
)

// maxProjectConfigBytes bounds one project config read. It protects the memory
// of a step that reads a file the user owns and this process does not size; a
// larger file is a config the step could not read.
const maxProjectConfigBytes = 1 << 20

// errProjectConfigUnreadable reports a project config that exists and could
// not be read: oversized, unreadable, or not JSONC.
var errProjectConfigUnreadable = errors.New("project config cannot be read")

// projectEvidence fills in what a project-bound host's config says. OpenCode
// has no listing command, and the `plugin` array it loads plugins from is the
// host's own record, so with the CLI present that array stands as its listing.
func projectEvidence(ctx context.Context, ev Evidence, spec HostSpec, projectRoot string) Evidence {
	ev.ProjectRoot = projectRoot
	// Guard, not advisory: the listing is what lets removal and update read
	// the host as "not installed" and stay silent, so a config that cannot be
	// read leaves it unanswered — the same as a host listing that did not
	// parse (readListing) — fail-open-or-fail-closed-reads.rule §6.
	listed, err := loadProjectConfigListing(ctx, spec, projectRoot)
	ev.ListingOK = ev.CLIPresent && err == nil
	ev.Listed = ev.CLIPresent && listed
	ev.RegistryListed = readProjectConfigListing(ctx, spec, projectRoot)
	return ev
}

// loadProjectConfigListing reports whether a config file of a project-bound
// host lists the plugin package, bare or pinned as `<package>@<version>`. It
// fails closed: when no file lists the package and one that exists cannot be
// read, it returns errProjectConfigUnreadable instead of false.
func loadProjectConfigListing(ctx context.Context, spec HostSpec, projectRoot string) (bool, error) {
	var unreadable error
	for _, dir := range configDirs(ctx, projectRoot) {
		for _, rel := range spec.ProjectConfigPaths {
			listed, err := loadConfigListsPackage(filepath.Join(dir, filepath.FromSlash(rel)), openCodePackage)
			if listed {
				return true, nil
			}
			if err != nil {
				unreadable = err
			}
		}
	}
	return false, unreadable
}

// readProjectConfigListing is the presence answer for the evidence that only
// reports, as registryListsPlugin is for the other hosts. It fails open.
func readProjectConfigListing(ctx context.Context, spec HostSpec, projectRoot string) bool {
	listed, err := loadProjectConfigListing(ctx, spec, projectRoot)
	// Advisory: a config that cannot be read is no evidence the plugin is
	// listed — fail-open-or-fail-closed-reads.rule §7.
	if err != nil {
		return false
	}
	return listed
}

// configDirs lists the directories whose config OpenCode loads for a session
// in projectRoot: projectRoot and each parent up to the root of its git working
// tree, nearest first. That root is also where `opencode plugin` writes its
// entry (packages/opencode/src/plugin/install.ts, patchDir, OpenCode 1.18.34).
// Outside git OpenCode writes into the directory it runs in, so the list is
// projectRoot alone.
func configDirs(ctx context.Context, projectRoot string) []string {
	// Advisory: without git, or with a root git does not place above the
	// project, the project directory is the one place OpenCode writes to —
	// fail-open-or-fail-closed-reads.rule §7.
	top, err := git.Toplevel(ctx, projectRoot)
	if err != nil {
		return []string{projectRoot}
	}
	// git answers with symlinks resolved (/private/var on macOS), so the
	// project is compared in the same form.
	dir, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return []string{projectRoot}
	}
	top = filepath.Clean(top)
	if rel, err := filepath.Rel(top, dir); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return []string{projectRoot}
	}
	dirs := []string{dir}
	for dir != top {
		dir = filepath.Dir(dir)
		dirs = append(dirs, dir)
	}
	return dirs
}

// loadConfigListsPackage reads one JSONC config file without writing it. A
// missing file is a config that lists nothing.
func loadConfigListsPackage(path, pkg string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	// Guard: a file that exists and cannot be read is no answer, never "lists
	// nothing" — fail-open-or-fail-closed-reads.rule §6.
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxProjectConfigBytes {
		return false, errProjectConfigUnreadable
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, errProjectConfigUnreadable
	}
	var config struct {
		Plugin []json.RawMessage `json:"plugin"`
	}
	if json.Unmarshal(dropTrailingCommas(stripComments(bytes.TrimPrefix(data, utf8BOM))), &config) != nil {
		return false, errProjectConfigUnreadable
	}
	return slices.ContainsFunc(config.Plugin, func(raw json.RawMessage) bool {
		spec := pluginSpecifier(raw)
		return spec == pkg || strings.HasPrefix(spec, pkg+"@")
	}), nil
}

// pluginSpecifier is the package an entry of the `plugin` array names: the
// entry itself, or the first element of a `[spec, options]` pair
// (packages/opencode/src/config/plugin.ts, OpenCode 1.18.34).
func pluginSpecifier(raw json.RawMessage) string {
	var spec string
	if json.Unmarshal(raw, &spec) == nil {
		return spec
	}
	var pair []json.RawMessage
	if json.Unmarshal(raw, &pair) != nil || len(pair) == 0 || json.Unmarshal(pair[0], &spec) != nil {
		return ""
	}
	return spec
}

var (
	utf8BOM     = []byte("\xef\xbb\xbf")
	lineComment = []byte("//")
	blockOpen   = []byte("/*")
	blockClose  = []byte("*/")
)

// stripComments removes the comments JSONC allows outside strings. A block
// comment becomes one space, so two tokens it separated stay separate.
func stripComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		switch {
		case data[i] == '"':
			end := stringEnd(data, i)
			out = append(out, data[i:end]...)
			i = end - 1
		case bytes.HasPrefix(data[i:], lineComment):
			end := bytes.IndexByte(data[i:], '\n')
			if end < 0 {
				return out
			}
			i += end - 1
		case bytes.HasPrefix(data[i:], blockOpen):
			end := bytes.Index(data[i+2:], blockClose)
			if end < 0 {
				return out
			}
			i += end + 3
			out = append(out, ' ')
		default:
			out = append(out, data[i])
		}
	}
	return out
}

// dropTrailingCommas removes a comma that JSONC allows before a closing
// bracket and JSON does not.
func dropTrailingCommas(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '"':
			end := stringEnd(data, i)
			out = append(out, data[i:end]...)
			i = end - 1
		case ',':
			rest := bytes.TrimLeft(data[i+1:], " \t\r\n")
			if len(rest) > 0 && (rest[0] == '}' || rest[0] == ']') {
				continue
			}
			out = append(out, ',')
		default:
			out = append(out, data[i])
		}
	}
	return out
}

// stringEnd returns the index just past the JSON string that opens at start.
func stringEnd(data []byte, start int) int {
	for i := start + 1; i < len(data); i++ {
		switch data[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return len(data)
}
