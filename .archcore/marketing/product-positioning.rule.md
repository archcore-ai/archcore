---
title: "Archcore Product Positioning and Messaging"
status: accepted
tags:
  - "marketing"
---

## Rule

When writing or editing any outward-facing content about Archcore (README, docs, website, social profiles, articles), follow these positioning guidelines.

### Primary positioning

1. The primary phrase is **"Git-native context for AI coding agents"**. Use it as the default tagline, title suffix, and first descriptor.
2. The secondary phrase is **"Context engineering for repositories"**. Use it as a supporting descriptor, never as the primary.
3. The expanded explanation is: **"Archcore helps teams turn scattered repo knowledge into structured context that AI coding agents can find, reuse, and follow."**

### Mechanism phrasing (delivery)

Archcore CLI ships as a local stdio MCP server. The mechanism phrase **"CLI and local MCP server"** (or **"CLI and local stdio MCP server"** when technical precision matters) may be used as a supporting clause after the primary phrase. It is allowed in:

- Repo descriptions (GitHub, package registries, MCP server directories)
- README sublines and subheaders
- Channel-specific bios where space allows

Do NOT lead with the mechanism phrase, replace the primary phrase with it, or use it as the H1. The hierarchy stays: benefit first, mechanism second.

### Messaging hierarchy

| Order | Role | Phrase |
|-------|------|--------|
| First | Benefit | Git-native context for AI coding agents |
| Second | Mechanism (delivery) | CLI and local MCP server |
| Third | Mechanism (concept) | Context engineering for repositories |
| Fourth | Value explanation | Archcore helps teams turn scattered repo knowledge into structured context that AI coding agents can find, reuse, and follow |

### Channel-specific framing

| Channel | Lead with |
|---------|-----------|
| Website | Benefit first — primary phrase as H1 |
| README | Product + mechanism — "Archcore is a git-native context layer for AI coding agents." Add a subline explaining the CLI + local MCP server delivery. |
| Repo description (GitHub, registries) | Primary phrase + delivery clause — "Git-native context for AI coding agents — CLI and local MCP server" |
| Docs | Clarity + ease — "Archcore is a git-native way to structure project context for AI coding agents." |
| Social | Shortest clear phrase — primary phrase |
| Author bio | What you are building — "Building Archcore — git-native context for AI coding agents." |
| Articles | Vocabulary: context engineering, repo context, context quality |

### Vocabulary

The ecosystem-wide preferred vocabulary and the avoid-list (e.g. "shared architectural memory", "system context platform", "context engineering platform") live in the `archcore` global source (`product/messaging-and-voice`); they are not restated here. This document covers only the **CLI-surface** specifics: the benefit-first hierarchy, the mechanism phrasing ("CLI and local MCP server"), channel framing, and the standard copy blocks. CLI-surface additions to the preferred set: "local MCP server", "local stdio MCP server", "MCP-compatible agent".

### Standard copy blocks

**Repo description (1-line with mechanism):** Git-native context for AI coding agents — CLI and local MCP server.

**1-line:** Archcore is a git-native context layer for AI coding agents.

**2-line:** Archcore is a git-native context layer for AI coding agents. It ships as a CLI and a local stdio MCP server, so any MCP-compatible agent can read and write your repo context through standard tools.

**3-line:** Archcore is a git-native context layer for AI coding agents. It helps teams structure decisions, rules, plans, and guides inside the repository. The result is stronger project context across sessions, tools, and workflows.

## Rationale

Consistent positioning prevents messaging drift across channels and contributors. The shift from "architectural memory" to "git-native context" reflects a clearer, more accurate description of what Archcore does. "Memory" implies persistence semantics that do not match the product; "context" accurately describes the value — structured knowledge that agents consume.

The mechanism clause "CLI and local MCP server" is included as an explicit secondary because it explains how Archcore integrates across many agents without a hosted service or bespoke plugins, and prepares the project for submissions to MCP server directories (mcpservers.org, Glama, Smithery). It must remain secondary to keep the leading message product-focused, not protocol-focused.

## Examples

### Good

- "Archcore is a git-native context layer for AI coding agents."
- "Git-native context for AI coding agents — CLI and local MCP server."
- "Archcore ships as a CLI and a local stdio MCP server — any MCP-compatible coding agent can read and write your repo context through standard tools."
- "Structure decisions, rules, plans, and guides in your repo so agents work with stronger project context."
- "Context engineering for repositories: a practical way to make project knowledge easier for agents to discover, reuse, and follow."

### Bad

- "Archcore is a shared architectural memory for AI coding agents." — uses avoided primary framing
- "Archcore is a context engineering platform." — uses avoided framing
- "Archcore is a system context platform for AI agents." — uses avoided framing
- "Archcore gives your repo a durable architectural memory layer." — uses old positioning
- "Archcore is an MCP server for repo context." — leads with mechanism instead of benefit
- "A local stdio MCP server for AI coding agents." — mechanism-first, drops the primary benefit phrase

## Enforcement

Content review. When editing README, docs, website copy, or any public-facing text, verify the primary description aligns with the positioning hierarchy above.