---
title: "Testing Two Registries Against Each Other, and Naming a Test Seam"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "testing"
---

## Overview

Two patterns that keep showing up in this repository's tests and are easy to apply wrongly. The
first guards a closed set declared in more than one place. The second lets a test observe or replace
something the production call chain does not expose.

### Target Audience

Anyone adding a registry, a matcher over one, or a test that needs to reach inside production code.

## Prerequisites

- Read `unit-testing-patterns.guide.md` for table-driven structure, `t.TempDir()` use, and the
  stdout-capture rules.
- Know which package owns the set you are about to duplicate.

## Steps

### 1. Find the second declaration

A closed set is often declared twice: the registry, and something that filters or dispatches on it.
Here the MCP tool names live in the server's registration, in `cmd.archcoreMCPTools`, and in the
host-side matcher `wiring.mcpDocumentTools`. Each pair that must agree needs a test.

### 2. Read one side, do not retype it

The load-bearing rule. Build the real thing and ask it what it holds.

```go
srv := mcpserver.NewServer(base, "test", mcpserver.WithHostWiring(hostWiringExecutor(base)))
registered := srv.ListTools()
```

A second hand-written list is not the registration — it is a copy of it, and one copy agrees with
another no matter what the code does. An earlier version of this test listed the constructors
instead; adding a tool to the server left it green.

### 3. Assert both directions

A set in the registry and not in the consumer is one bug; the reverse is another. Check each way,
with a message naming the consequence rather than the mismatch.

### 4. Refuse to pass on an empty read

The read side can come back empty — `ListTools` returns `nil` for a server holding nothing — and
every assertion then passes vacuously.

```go
if len(registered) == 0 {
    t.Fatal("the server registered no tools, so this test proves nothing")
}
```

### 5. Build the set from the tree, not from a list

When the subject is a command tree or a registry that can grow, walk it. A test that enumerates
members cannot fail when a member is added, which is the case it exists for. Where an exemption is
genuinely needed, put it in a map whose value is the reason, and assert that the exemption still
applies.

### 6. Name a seam and say production does not use it

A package-level variable or counter that exists only so a test can reach behavior the parameters do
not expose. Declare it with a comment saying so:

```go
// hookExit is os.Exit, indirected so a test can observe the code without the
// process leaving. Production never reassigns it.
var hookExit = os.Exit
```

Without the note the next reader cannot tell a seam from a mutable global, and the one after that
writes to it.

### 7. Prefer a parameter to a seam

A seam is the fallback. If the behavior can be reached by passing a value — a base directory, a
clock, an interface — pass it. Add a seam only when the alternative is exporting API that exists for
the test alone.

## Verification

- [ ] The agreement test reads one side from a constructed object
- [ ] Both directions are asserted
- [ ] An empty read fails rather than passes
- [ ] A tree-shaped subject is walked, not listed
- [ ] Every exemption carries its reason in the data, not in a comment beside it
- [ ] Every seam says that production does not use it
- [ ] The test fails when the behavior is removed — verified by removing it, not assumed

## Common Issues

### The test passes and the registries disagree

Both sides are hand-written lists. Rebuild one from the running system. If nothing constructs the set
at runtime, that is the finding: the set has no owner.

### The exemption list rots

An entry stays after the reason expires, and the test now excuses a real gap. Assert the exemption in
both directions — a listed member that has since gained the property should fail too.

### A seam is read in production

Grep for the variable. If a non-test file reads it, it is not a seam; it is configuration, and it
belongs in a parameter or a settings field.
