package tools

import "context"

// RootProvider answers which project root one tool call operates on. The
// project root is resolved per call rather than captured at construction, so a
// session that moves into a git worktree mid-session moves the server with it
// (project-root-resolution.spec §3). The provider decides; a handler only asks.
//
// Root never fails: a provider that cannot reach the client, or that refuses
// what the client reported, answers with the root it is already serving
// (project-root-resolution.spec §13 and §14, and the Failure table).
type RootProvider interface {
	Root(ctx context.Context) string
}

// StaticRoot is the RootProvider of a server pinned to one project root — the
// --project and ARCHCORE_PROJECT_ROOT case (project-root-resolution.spec §1),
// and the provider every unit test uses.
type StaticRoot string

// Root returns the pinned project root.
func (r StaticRoot) Root(context.Context) string { return string(r) }
