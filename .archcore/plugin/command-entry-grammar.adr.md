---
title: "Command Entry Grammar — Mode Word First, Types Chosen at Gates"
status: draft
tags:
  - "architecture"
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

**Update (2026-09-20).** `init-import-mode.adr` changes the `init` row: the hint is `[import|refresh] [path or domain]`. `domain <slug>` is the subject of `refresh`, and `--depth` and `--scale` are preview toggles (`depth:`, `scale:`), so no argument hint carries a flag. The table and the flag sentence below show the 2026-09-16 state.

## Context

On 2026-09-16 the four argument hints used three grammars: `init` takes only `--key=value` flags, `review` mixes `--deep` and `--drift` with a positional scope, and `plan` and `document` take a positional word (@plugin/plugins/archcore/skills/init/SKILL.md, @plugin/plugins/archcore/skills/review/SKILL.md, @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/document/SKILL.md). The `plan` hint `[topic] [sdd | sources | iso | research]` shows the topic first, but @plugin/plugins/archcore/skills/_shared/delta-routing.md reads the path from the leading word, and `plan/SKILL.md` step 2 accepts a document type name that the same file says is not an entry. The `document` hint lists ten type names under the placeholder `[module, topic, or decision]`, although only `adr`, `rfc`, and `rule` enter the decision track; the user read the command as "record a decision". Hidden entries exist that no hint shows: five route names on `plan`, and `actualize`, `experience`, `closeout`, `cpat`, and `task-type` on `review`.

## Decision

Every command takes the form `/archcore:<command> [mode] [subject]`, where the mode is the first word, a noun from a closed list that the argument hint shows in full, and a gate inside the track selects the document type, as the research instrument already does on `plan`.

| Command | Argument hint |
|---|---|
| `init` | `[refresh\|domain <slug>] [--depth=light\|standard\|deep] [--scale=small\|medium\|large]` |
| `plan` | `[sdd\|sources\|iso\|research] [topic]` |
| `document` | `[decision\|code\|research] [subject]` |
| `review` | `[drift\|deep\|closeout\|experience] [path, tag, or scope]` |

The mapping on `document` is `decision` → `decision.classify` (`adr`, `rfc`, `rule`), `code` → `describe.read` (`spec`, `doc`, `guide`, `scenario`), and `research` → `research.frame` (`research`, `rnd`, `evidence`). A type name inside the subject text is a signal that the selecting gate's `skip_when` reads, so `document decision rfc for gRPC` reaches `decision.rfc` with no question. A `--flag` stays only for a setting that does not change the entry: `--depth` and `--scale` on `init`. The `init` setting formerly named `--mode` is `--scale`, because the word "mode" now names the first word of every command. The five route names stop being `plan` entries. The mode word is user surface; the track identifier and the `<track>.<stage>` gate address stay internal and do not appear in a user-facing report.

## Alternatives Considered

1. **Mode words with the ten type names kept as hidden aliases** — rejected because the hint would stop being the complete expert surface, the defect the 2026-09-07 revision of the research vocabulary removed when it retired `plan rnd` and `plan evidence`.
2. **Two-level form `document [mode] [type] [subject]`** — rejected because the hint grows back to ten type names and `plan` has no second level, so the two commands would differ again.
3. **Verbs as modes (`decide`, `describe`, `research`)** — rejected because "document describe" reads as two verbs, and the `plan` and `review` modes are already nouns.
4. **Track identifiers as modes (`decision`, `describe`, `research`)** — rejected because `describe` is a verb and the user read the dotted gate address `describe.read` as a function name; renaming the track instead touches every golden, fixture, and track-layer clause.
5. **Keep `--drift`, `--deep`, `--refresh`, `--domain` as flags** — rejected because a flag and a mode word would select entries in two syntaxes on one surface.
6. **Keep the `init` setting name `--mode`** — rejected because a prompt review found the executor must separate two meanings of one word inside `init/SKILL.md`.

## Consequences

### Enabled

- One reading rule for all four commands: the first word selects the path, the rest is the subject.
- The `document` hint shrinks from ten names to three, and each name answers "document what?".
- A user who knows the type still gets it with zero questions, through the gate's `skip_when`. On 2026-09-16 @plugin/test/behavioral/document-bench.sh scored 19 of 20 fixtures on the first run, 14 of them with no mode word; the one miss (an accepted RFC resolution read as `rfc`) passed 3 of 3 runs after the mode table named the `adr` that `decision.resolve` records.

### Costs and limits

- Breaking change for recorded invocations: `document adr|rfc|rule|spec|doc|guide|research|evidence|scenario|journey`, `review --drift`, `review --deep`, `init --refresh`, `init --domain=<slug>`, and `init --mode=<scale>` become topic text or an unknown setting; the release notes name each retired form.
- The mode word `code` differs from the track identifier `describe`; the mapping lives in one table in `document/SKILL.md`.
- The document bench runs 20 fixtures on one model; it does not measure host auto-invocation from a natural-language message.
- The global RFCs `concepts/research-and-evidence-types` (accepted) and `concepts/scenario-and-journey-types` (draft) in the `archcore` global source name `document evidence`, `document scenario`, and `document journey` as entries; this local decision overrides them for the plugin, and the global documents stay unchanged.

## Superseded when

- The document bench drops below 18 of 20 fixtures on a supported model.
- A fifth command joins the surface and needs a mode that is not a noun.
