---
title: "Zero-Content Onboarding Implementation — SessionStart Nudge + /archcore:init"
status: rejected
tags:
  - "hooks"
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

**Outcome (2026-05-15).** The plan was executed. The skill shipped as `skills/init/` and the command is `/archcore:init` per `skill-surface-collapse.adr`, having been drafted as `bootstrap`; read every reference to bootstrap below as init. The SessionStart nudge of Phase A, the stack-rule generation of B1, the run-guide generation of B2, and the agent-instruction import of B3 all shipped as designed.

## Goal

Implement variants A and B from `zero-content-onboarding.idea`, so a fresh-install user goes from an empty `.archcore/` to a useful seeded state in one short session, through two coupled deliverables.

**Phase A — the SessionStart nudge.** When `.archcore/` is missing or empty, the hook adds one advisory line pointing at `/archcore:init`. Pure copy, roughly 10 lines of shell.

**Phase B — the `/archcore:init` intent skill**, in three sequential steps. B1 and B2 generate their artifacts directly, with no accept-edit-skip prompt, because each output is a short file that is trivially edited, deleted, or regenerated. B3 is opt-in behind a cost warning and a dry-run preview, because it can create many documents at once. B1 generates a terse imperative stack rule from manifest detection, carrying no library inventory and no versions. B2 generates a short run-the-app guide from the README and the scripts, with monorepo awareness. B3 parses the existing agent-instruction files — `CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.cursor/rules/*.mdc`, `.github/copilot-instructions.md`, `.windsurfrules`, `.junie/guidelines.md`, and `CONVENTIONS.md` — with a cost warning on a large input. Its default per-file mode is link by reference, creating a `doc` whose body holds a one-line pointer and whose tags carry the source identifier, duplicating no content; an optional extract mode routes content into typed rules, ADRs, and docs.

**Deferred deliberately.** Variant C, the full repository introspection beyond manifests, awaits observed usage of B. An active guardrail or lint for the generated stack rule is out of scope, because Phase B produces context rather than enforcement. Auto-refresh of an imported document when its source changes stays manual, covered by `/archcore:review --drift`. And CLI-side ownership of empty-state detection is a separate decision once the experience is validated; Phase A ships plugin-side first.

## Tasks

### Phase A — the SessionStart empty-state nudge

**A1 — detection and advisory output.**

| File | Change |
|------|--------|
| `bin/session-start` | Add the empty-state check after the existing CLI invocation. IF `.archcore/` is missing, or contains no document whose body is at least 200 characters and whose status is `accepted` or `draft`, THEN append an advisory paragraph to the payload. |
| `bin/lib/empty-state.sh` | New. A helper returning success when archcore is functionally empty, using `find` and `wc -c`, with no MCP call and no jq dependency. |
| `test/unit/session-start-empty.bats` | New. Three cases: no `.archcore/` directory yields the nudge; a directory holding only a `.gitkeep` and sub-200-character stubs yields the nudge; and a directory holding at least one substantial document yields no nudge. |
| `test/unit/session-start.bats` | Existing cases unchanged. |

The nudge text reads exactly:

```
.archcore/ is empty. Run /archcore:init to seed a stack rule, a run-the-app
guide, and (optionally) imports from existing agent-instruction files like
CLAUDE.md or AGENTS.md. Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.
```

`ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the line entirely. No persistent flag is needed for suppression after init, because once any step completes the empty-state check returns false on its own.

### Phase B — the `/archcore:init` intent skill

**B1 — stack rule generation.** The skill detects manifests, reading in order and stopping at the first found per language across `package.json`, `pnpm-workspace.yaml`, `pyproject.toml`, `Pipfile`, `requirements.txt`, `Cargo.toml`, `go.mod`, `Gemfile`, `composer.json`, `*.csproj`, `pom.xml`, and the Gradle build files, allowing several to coexist. It pulls the top-level declared dependencies from each, filters them to a curated allowlist of stack signals, ignores versions, and caps at 5 signals in total. Where no manifest exists it scans the top-level file extensions and identifies at most 2 majority languages. It then composes an imperative draft from a fixed template whose lines name the language, the build framework with an ADR requirement before introducing an alternative, the persistence library and primary store, the styling library, and the state library — dropping any line whose signal was not detected rather than leaving a placeholder. It creates the document directly as a `rule` named `project-stack` under `conventions/` with status `accepted`, and reports one line naming the signals and the path. Before running it checks for an existing stack rule in that directory and asks whether to regenerate or skip.

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | New. The full intent-skill structure per `skill-file-structure.rule`, with a description triggering on phrases such as "init", "initialize archcore", and "first-time setup". |
| `skills/_shared/grounding/detect-stack.md` | New. The manifest-to-signals lookup tables. |
| `test/structure/skills.bats` | Extended to assert the presence of the skill and its required sections. |

**B2 — run-the-app guide generation.** The skill detects a monorepo through `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `lerna.json`, or several `package.json` files under `apps/` or `packages/`. It reads `README.md` for the first section matching a getting-started, quick-start, installation, development, setup, or local heading, and falls back to the manifest scripts when the README yields nothing usable. It composes from the single-app or monorepo template and creates the document directly as a `guide` named `running-the-project` under `onboarding/` with status `accepted`, skipping when such a guide already exists.

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | Extended with the B2 step. |
| `skills/_shared/grounding/extract-run-instructions.md` | New. The heuristics for README section selection. |

**B3 — the opt-in parse of agent-instruction files.** The skill probes the documented file list held as data in its lib, estimates cost by summing byte size and showing a summary, and applies the high-cost gate: IF the combined size exceeds 50 KB, or the file count exceeds 5, or the estimated yield exceeds 8 documents, THEN it prefixes the warning and requires explicit confirmation. Each detected file gets a mode — link by default, extract, or skip. Every imported document carries the `imported` and `source:<slug>` tags plus a body first line naming the source path and the import date. A dry-run preview shows the full list of intended writes before any creation call. Creation runs per item, with the `related` edges added afterwards. A document carrying a matching `source:<slug>` tag counts as already imported.

| File | Change |
|------|--------|
| `skills/init/SKILL.md` | Extended with the B3 step. |
| `skills/init/lib/agent-files.md` | New. The detection list. |
| `skills/_shared/grounding/extract-routing.md` | New. The imperative, decision, and reference heuristics. |
| `commands-system.spec` | Register the command in the surface. |
| `skills-system.spec` | Add the skill to the skill section. |

**B4 — metadata and routing.** The skill description triggers on the natural-language phrases listed above plus "seed archcore" and "what should I do first". The skill is auto-invocable per `skill-surface-collapse.adr`, and the commands specification documents the trigger phrases.

### Phase C — documentation

| File | Change |
|------|--------|
| `README.md` | Add one line under the try-these section telling a user with an empty repository to run the command first. |
| `claude-plugin.prd` | Add the functional requirements for first-session activation on an empty `.archcore/` and for the init skill. |
| `multi-host-compatibility-layer.spec` | Document the suppression environment variable. |
| `development-roadmap.plan` | Mark zero-content onboarding per phase. |

### Phase D — release

Bump the version in the plugin manifests per the coordinated release plan, and add the README changelog entry.

## Acceptance Criteria

1. The empty-state nudge fires correctly, verified by its bats file.
2. `ARCHCORE_HIDE_EMPTY_NUDGE=1` suppresses the nudge unconditionally.
3. The command is discoverable and auto-invokes on the phrases listed in B4.
4. B1 produces a stack rule of 6 lines or fewer, with no version number and at most 5 stack signals, written directly with no confirm prompt.
5. B2 produces a run guide of 15 lines or fewer, branching on monorepo detection, written directly with no confirm prompt.
6. B3 detects every file in the documented list, reports cost accurately, and gates the high-cost case behind explicit confirmation.
7. B3 link mode creates `doc` documents carrying the canonical tag and body-pointer convention.
8. B3 extract mode routes an imperative to a rule, a decision to an ADR, and reference material to a doc.
9. Every init step is idempotent, so a re-run skips an already-created artifact with a clear message.
10. The product requirements document carries the two new functional requirements.
11. The test suite is green.

## Dependencies

- No CLI release dependency for any phase.
- The existing intent-skill infrastructure, with `skill-surface-collapse.adr` providing the auto-invocation contract.
- The existing per-document-type schemas for rule, guide, doc, and adr.

## Risks

- **B1 detection false positives.** Mitigated by a signal allowlist that excludes type packages, linters, formatters, test runners, and build tools.
- **B2 README extraction quality.** A marketing-heavy README may yield no usable command block, which the scripts fallback covers.
- **B3 extract-mode quality.** Mitigated by the mandatory dry-run preview and the link default.
- **B3 cost-estimate accuracy.** The heuristic of one document per 800 bytes is rough and can be off by half on an extreme input. Accepted.
- **Tag-spec compatibility.** Mitigated by a targeted unit test creating a document with a source tag.
- **Slug collisions.** Mitigated by including the file extension in the slug.
- **Stale source files.** Mitigated by `/archcore:review --drift`, which covers code-document drift.
- **Idempotency edge cases.** Mitigated by a regenerate prompt that warns explicitly about overwriting an edit.
- **Skill discovery depends on Phase A.** Both phases must ship together.
