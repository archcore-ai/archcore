# Archcore examples

A gallery of `.archcore/` context — what it looks like and how you work with it.
Browse them here on GitHub, or open any folder with your AI agent and ask
questions: every example is already wired up for **Claude Code, Cursor, and
Codex CLI**.

## A single project

| Example | What it shows |
| ------- | ------------- |
| [01-minimal](01-minimal/) | the smallest `.archcore` — a few documents |
| [02-fullstack-app](02-fullstack-app/) | a full project: docs organized by area, many document types |
| [03-product-planning](03-product-planning/) | planning a feature: idea → PRD → plan → spec |
| [04-experience-playbook](04-experience-playbook/) | reusable how-tos and team playbooks |

## Sharing standards across projects

| Example | What it shows |
| ------- | ------------- |
| [05-global-single-source](05-global-single-source/) | reuse one shared source |
| [06-global-multiple-sources](06-global-multiple-sources/) | reuse several shared sources at once |
| [07-local-overrides-global](07-local-overrides-global/) | keep the company standards, override one locally |
| [08-global-in-archcore](08-global-in-archcore/) | keep the shared standards inside your own `.archcore/` |

## In a monorepo

| Example | What it shows |
| ------- | ------------- |
| [09-monorepo-shared-global](09-monorepo-shared-global/) | apps share standards kept in a package |
| [10-monorepo-root-global](10-monorepo-root-global/) | apps share standards kept at the repo root |

The [`_global_/`](_global_/) folder holds the shared standards that examples 05–07 reuse.

## Try them

Reading on GitHub needs nothing. To run one, install the CLI so `archcore` is on
your `PATH`:

```bash
curl -fsSL https://archcore.ai/install.sh | sh
```

Then open any example folder with your agent — it's already set up to read that
folder's `.archcore/`, so just ask it about the project. Each example's README
suggests questions to start with.
