# 04 · experience-playbook

> Team know-how written down: repeatable how-tos and the patterns the team
> follows.

Context isn't only decisions and rules — it's also "how we do recurring work
here," so an agent follows the house playbook instead of improvising.

**What's in `.archcore/` here**

- `add-rest-endpoint.task-type.md` — the steps for a new endpoint
- `add-feature-flag.task-type.md` — how the team ships behind flags
- `migrate-to-react-query.cpat.md` — a before/after code-change pattern

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 04-experience-playbook && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 04-experience-playbook && codex`.

## Ask your agent

- "Walk me through adding a feature flag the way this team does it."
- "How do we migrate a useEffect fetch to React Query here?"
- "What's the checklist for adding a REST endpoint?"
- "When is it safe to remove a feature flag?"
