# 02 · fullstack-app

> What a real project's `.archcore/` looks like — documents of many types,
> organized by area.

A fullstack app's context, grouped into product, frontend, backend, infra, and
team playbooks. Some documents link to each other, so the agent can follow a
thread — e.g. how to add an endpoint → the API contract → the error rule.

**What's in `.archcore/` here**

- `product/` — onboarding requirements, the Q3 roadmap
- `frontend/` — the state-management decision, the component-structure rule
- `backend/` — the API contract, error handling, the migration how-to
- `infra/` — the deployment decision, the observability reference
- `experience/` — the "add an endpoint" playbook

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 02-fullstack-app && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 02-fullstack-app && codex`.

## Ask your agent

- "What are the rules for frontend state management?"
- "How do I add a new API endpoint here?"
- "What's planned for Q3, and what does it depend on?"
- "Where do database migrations go, and what are the constraints?"
