---
title: "OpenCode 2.x Update: Authorize With opencode plugin list, Print a Hint When the Service Disagrees With the Config"
status: draft
tags:
  - "component:cli"
  - "integrations"
  - "opencode"
  - "update"
---

## Goal

On OpenCode 2.x, `archcore update` runs `opencode plugin update @archcore/opencode-plugin` only when the host's own listing, `opencode plugin list`, shows the plugin. When the global config lists the plugin and the running OpenCode service does not, the step prints one hint line and runs nothing.

Incident, observed 2026-11-02 on opencode 2.1.4:

- `~/.config/opencode/opencode.json` holds `"plugins": ["@archcore/opencode-plugin"]`; `loadProjectConfigListing` in @cli/internal/plugin/project_config.go reads it as listed.
- `opencode plugin update <pkg>` calls `cli.plugin.inspect`, which asks the running `opencode serve --service` through `plugin.list({location:{directory:cwd}})` and fails with `Plugin is not configured: <pkg>` when the answer is empty (strings of the 2.1.4 binary).
- The service, PID 5611, ran with `XDG_CONFIG_HOME=/private/tmp/archcore-live.PUUoVS/oc-config`, a directory that no longer existed. `opencode plugin list` printed `No plugins found` from every directory tried.
- The step printed `✗ OpenCode: this command failed`, with no reason the user could act on.

Root cause in archcore: the 2.x row authorizes the update from the config file, not from the host's answer. `updating-the-plugin.spec` Surface states "the host's own answer first", and its OpenCode row states "The CLI finds the plugin in the global config without a project root". The second statement is wrong for 2.1.4: the CLI finds the plugin through the service.

Decisions taken by the maintainer on 2026-11-02:

- Authorization on 2.x: `opencode plugin list`, not the config file.
- Config lists the plugin, service does not: one hint line, no command.
- After review (2026-11-02): only the update reads the service listing, and only when the config lists the plugin. Probed: with no service running, `opencode plugin list` starts `opencode serve --service` with the caller's environment.

## Tasks

### Phase 1 — Evidence

1. Add a `Listing` of `opencode plugin list` to the `Since2` row. @cli/internal/plugin/hosts.go
2. Read the `Since2` listing in `ReadServiceListings`, for the update only, on 2.x with the config listing the plugin. @cli/internal/plugin/project_config.go @cli/cmd/plugin_run.go
3. Add a parser for the 2.x table: header `ID VERSION SOURCE`, one row per plugin. @cli/internal/plugin/evidence.go
4. Match the row by exact `@archcore/opencode-plugin` or `@archcore/opencode-plugin@<version>`, not by `namesPlugin`. @cli/internal/plugin/evidence.go
5. Read `No plugins found` as a parsed, empty listing; read any other unparsed output as no answer. @cli/internal/plugin/evidence.go
6. Store the service answer in new `Evidence` fields, `ServiceListingOK` and `ServiceListed`. @cli/internal/plugin/plugin.go
7. Keep `Listed` from the config files, so install, remove, and status keep today's behavior. @cli/internal/plugin/project_config.go
8. Run the listing in the update command's directory; both already inherit it. @cli/internal/plugin/exec.go

### Phase 2 — Planning

9. In `planUpdate`, on OpenCode 2.x, run the update only when `ServiceListingOK && ServiceListed`. @cli/internal/plugin/plan.go
10. When config lists the plugin and the parsed service listing does not, return a note action. @cli/internal/plugin/plan.go
11. Word the note as one line naming the service and the update command. @cli/internal/plugin/hosts.go
12. When the service listing failed or did not parse, skip silently, per Failure Behavior 1. @cli/internal/plugin/plan.go

Note text as shipped: `The OpenCode config lists @archcore/opencode-plugin, but the running OpenCode service does not see it. Run: opencode service stop, then: opencode plugin update @archcore/opencode-plugin`. Stop, not restart: the next OpenCode command starts a fresh service with the caller's environment; the environment a restart uses is unverified. The evidence does not keep which config file listed the plugin, so the line names no path.

### Phase 3 — Tests

13. Extend the 2.x update test: listing shows the plugin → one update command. @cli/internal/plugin/plan_test.go
14. Add a test: config lists, service listing empty → one note, no command. @cli/internal/plugin/plan_test.go
15. Add a test: service listing failed → no command, no line. @cli/internal/plugin/plan_test.go
16. Add parser tests from captured 2.1.4 output: table row, `No plugins found`, garbage. @cli/internal/plugin/evidence_test.go
17. Add a test: 1.x runs no listing command and keeps requirements 27 and 28. @cli/internal/plugin/evidence_test.go

### Phase 4 — Live probe

18. Run `archcore update` with a healthy service; record that the update runs. No target file found.
19. Start a service with a temp `XDG_CONFIG_HOME`; record that the hint prints. No target file found.
20. Stop the probe service after the probe, so no stale service stays. No target file found.

## Acceptance Criteria

- With OpenCode 2.1.4, a healthy service, and the plugin in the global config, `archcore update` runs `opencode plugin update @archcore/opencode-plugin` and reports success.
- With the plugin in the global config and a service that does not see it, `archcore update` runs no OpenCode command and prints exactly one hint line.
- With no plugin in any config and no plugin in the service listing, the step prints nothing for OpenCode.
- With a failing `opencode plugin list`, the step prints nothing for OpenCode and runs no mutating command.
- On OpenCode 1.x, behavior is unchanged: no listing command, no update command, no line.
- `go test ./internal/plugin/...` passes.

## Dependencies

- `updating-the-plugin.spec` — requirements 5, 7, 29 and the OpenCode Surface row change; see Declared Delta.
- `plugin-delivery.spec` — owns the shared engine and the host table.
- The live probe follows `host-probe-protocol.spec`.

Open points [assumption]:

- The 2.1.4 table format has no `--json` form; a later OpenCode release may change it. A parse failure then gives silence, not a wrong update.
- The listing asks the service for the working directory's location. A project-level config in that directory can add entries; the parser still matches only the package name.

## Declared Delta

- modifies: `updating-the-plugin.spec` → OpenCode Surface row: replace "The CLI finds the plugin in the global config without a project root" with "`opencode plugin update` finds the plugin through the running OpenCode service (`plugin.list`), so its listing is `opencode plugin list` (2.1.4)".
- modifies: `updating-the-plugin.spec` → requirement 29: "WHEN `opencode plugin list` on OpenCode 2.x shows the plugin, the plugin-update step MUST run `opencode plugin update @archcore/opencode-plugin`."
- adds: `updating-the-plugin.spec` → requirements 30 and 31: the hint line, and no command, when config and service listing disagree.
- adds: `updating-the-plugin.spec` → requirement 32: no `opencode plugin list` without a config entry.
- modifies: `plugin-delivery.spec` → OpenCode evidence and `supported-ai-agents.doc`: the 2.x update-only service listing.
- modifies: `updating-the-plugin.spec` → Surface, Evidence order: add `opencode plugin list` (2.x) to the listing commands.
- unchanged: install, remove, and status keep reading the config files as their listing and run no `opencode plugin list`.
