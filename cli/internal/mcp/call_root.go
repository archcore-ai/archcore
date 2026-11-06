package mcp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/projectroot"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type callRootKey struct{}

// callRoot is the tools.RootProvider every handler receives. It returns the
// root the call-root resolver settled before the handler ran, so the resolver,
// the handler, and the response field agree on one root per call
// (call-root-resolution.spec §8).
type callRoot struct{ session tools.RootProvider }

func (c callRoot) Root(ctx context.Context) string {
	if root, ok := ctx.Value(callRootKey{}).(string); ok {
		return root
	}
	return c.session.Root(ctx)
}

// callRootResolver settles the root of one tool call: the project argument,
// then the project that holds an absolute for_path, then the session root
// (call-root-resolution.spec). It keeps nothing between calls (§17).
type callRootResolver struct{ session tools.RootProvider }

// wrap resolves the call root, refuses the call when the root cannot be
// served, runs handle against that root, and names the root in the response.
func (r callRootResolver) wrap(toolName string, handle server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		root, refusal := r.resolve(ctx, toolName, request)
		if refusal != "" {
			return mcp.NewToolResultError(refusal), nil
		}
		ctx = tools.WithSessionRoot(context.WithValue(ctx, callRootKey{}, root), r.settledSessionRoot(ctx))
		result, err := handle(ctx, request)
		if err != nil {
			return result, err
		}
		source := tools.SourceOf(ctx, root)
		if relationEndpointTools[toolName] {
			// "source" already names the relation endpoint in these payloads.
			source = ""
		}
		return tools.WithProjectField(result, root, source), nil
	}
}

// relationEndpointTools are the tools whose payload carries a top-level "source"
// key of its own; the response names their project only.
var relationEndpointTools = map[string]bool{"add_relation": true, "remove_relation": true}

func (r callRootResolver) resolve(ctx context.Context, toolName string, request mcp.CallToolRequest) (root, refusal string) {
	if project := strings.TrimSpace(request.GetString(tools.ProjectArgument, "")); project != "" {
		root, refusal = resolveProjectArgument(project, toolName == "init_project")
		// The session root is not served here, only compared: a project that is
		// the session's directory keeps the session's spelling, so one directory
		// is one root on the wire and in every cache key. The comparison reads the
		// settled root and never waits on roots/list (call-root-resolution.spec §2).
		if session := r.settledSessionRoot(ctx); refusal == "" && projectroot.SameDir(root, session) {
			root = session
		}
	} else {
		root = r.session.Root(ctx)
		if toolName == "search_documents" {
			root, refusal = resolveForPathRoot(root, strings.TrimSpace(request.GetString("for_path", "")))
		}
	}
	if refusal != "" {
		return "", refusal
	}
	if toolName == "add_relation" || toolName == "remove_relation" {
		refusal = relationRootRefusal(root, request)
	}
	return root, refusal
}

// settledRooter is a root provider that can report its root without deciding
// it again.
type settledRooter interface{ settledRoot() string }

func (r callRootResolver) settledSessionRoot(ctx context.Context) string {
	if s, ok := r.session.(settledRooter); ok {
		return s.settledRoot()
	}
	return r.session.Root(ctx)
}

const projectFormRefusal = "project must be an absolute path to a project root, the directory that holds .archcore/"

// resolveProjectArgument returns the root a project argument names. init_project
// alone accepts a directory without .archcore/, because creating it is that
// tool's purpose (call-root-resolution.spec §12).
func resolveProjectArgument(project string, initializing bool) (root, refusal string) {
	if !filepath.IsAbs(project) {
		return "", projectFormRefusal
	}
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil {
		// A guard: a path the resolver cannot evaluate is a root it must not serve.
		return "", "project is not an existing directory"
	}
	if filepath.Base(resolved) == ".archcore" {
		return "", projectFormRefusal + ", not .archcore/ itself"
	}
	if err := checkRoot(resolved, !initializing); err != nil {
		return "", projectRefusal(err)
	}
	return resolved, ""
}

// resolveForPathRoot returns the root that serves a for_path. A relative or
// empty for_path keeps the session root, exactly as before this contract. An
// absolute for_path selects the nearest project above it
// (call-root-resolution.spec §5–§6).
func resolveForPathRoot(session, forPath string) (root, refusal string) {
	if !filepath.IsAbs(forPath) {
		return session, ""
	}
	nearest, ok := projectroot.Nearest(forPath)
	switch {
	case ok && projectroot.SameDir(nearest, session):
		return session, ""
	case ok:
		if err := checkRoot(nearest, true); err != nil {
			return "", rootRefusal("the project that holds for_path", err)
		}
		return projectroot.Canonical(nearest), ""
	case isWithin(session, forPath):
		// An uninitialized session root answers as it does today
		// (call-root-resolution.spec, Failure Behavior 9).
		return session, ""
	default:
		return "", "for_path names a file outside every Archcore project; pass project to name the project to search"
	}
}

const crossProjectRelationRefusal = "source and target must belong to one project: a relation stays inside one project; pass project to choose it"

// relationRootRefusal refuses a relation whose source or target is an
// absolute path outside root (call-root-resolution.spec, Failure Behavior 7).
// A relative path always resolves under root, so only an absolute one can
// name a second project.
func relationRootRefusal(root string, request mcp.CallToolRequest) string {
	for _, key := range []string{"source", "target"} {
		path := strings.TrimSpace(request.GetString(key, ""))
		if !filepath.IsAbs(path) {
			continue
		}
		owner, ok := projectroot.Nearest(path)
		if !ok || !projectroot.SameDir(owner, root) {
			return crossProjectRelationRefusal
		}
	}
	return ""
}

// projectRefusal renders a checkRoot verdict on a project argument: the form
// the argument must take, or how to create the store it lacks.
func projectRefusal(err error) string {
	switch {
	case errors.Is(err, errRootNotAbsolute):
		return projectFormRefusal
	case errors.Is(err, errRootNoArchcore):
		return rootRefusal("project", err) + "; to create it, call init_project with the same project argument after the user asks for it"
	default:
		return rootRefusal("project", err)
	}
}

// rootRefusal renders a checkRoot verdict about subject, for a tool error and
// for the root provider's warning stream alike. No message carries the path
// the caller sent (no-absolute-paths-in-mcp-errors.rule §5).
func rootRefusal(subject string, err error) string {
	fatal, isFatal := errors.AsType[fatalGlobalError](err)
	switch {
	case errors.Is(err, errRootNotAbsolute):
		return subject + " is not absolute"
	case errors.Is(err, errRootNotDirectory):
		return subject + " is not an existing directory"
	case errors.Is(err, errRootPluginCache):
		return subject + " is inside an AI-host plugin install cache, not a user project"
	case errors.Is(err, errRootNoArchcore):
		return subject + " holds no .archcore/ directory"
	case errors.Is(err, errRootSettingsUnreadable):
		return subject + " has an unreadable .archcore/settings.json"
	case isFatal:
		return fmt.Sprintf("%s declares a global source %q that is misconfigured", subject, fatal.sourceID)
	default:
		return subject + " cannot be verified as a project root"
	}
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
