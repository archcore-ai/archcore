---
title: "README \"First 60 Seconds\" Onboarding Section"
status: draft
tags:
  - "marketing"
  - "onboarding"
  - "plugin"
  - "readme"
---

## Idea

Add a tightly scoped "First 60 seconds" section at the very top of `README.md`, so a new visitor understands the value of the plugin and sees it working without scrolling past the fold. It has three components.

1. **A hero visual of 15–20 seconds** — an asciinema cast or animated GIF in which the user types one natural-language prompt, "record the decision to use PostgreSQL", and the terminal shows `decide` routing, the ADR appearing in `.archcore/`, and the relation suggestion. Raw terminal, with no voiceover and no slide transitions.
2. **A before-and-after diagram** — a two-column Mermaid flowchart directly under the title: a flat pile of ad-hoc Markdown files on the left, and a typed, related graph on the right with ADR → rule → guide and PRD → plan.
3. **Three outcome bullets** naming the visible deltas of the first session — a typed document graph, skill routing from natural language, MCP-enforced writes — each at 12 words or fewer.

The README currently opens with a long "What it does" and "Without vs. with Archcore" narrative that assumes the reader is already bought in, so a visitor who does not already know what an ADR is closes the tab before reaching the value.

## Value

- **Top-of-funnel lift.** Marketplace traffic is high-intent and low-patience, so the first visible screen decides whether the visitor installs or bounces. [assumption] Comparable plugins with demo assets surface consistently in community roundups; no conversion measurement exists for this repository.
- **Zero to installed without jargon.** The visual shows what the plugin does before the reader must learn the vocabulary — ADR, MCP, intent routing, relations — leaving the narrative section below free to keep using domain terms.
- **Lower support load.** A user who bounces after install often returns asking how to use it; the hero answers that before install, self-filtering confused users and attracting the right ones.
- **Low production cost.** One asciinema recording is about 10 minutes of work, and exporting to SVG or GIF is one command.

## Possible Implementation

1. **Record the hero cast** with `asciinema rec readme-hero.cast --idle-time-limit 1.5`, starting from a fresh repository with `.archcore/` initialized, or recording the init as the first step. Use the prompt "record the decision to use PostgreSQL as our primary database — we considered DynamoDB and MongoDB", show the auto-invocation of `/archcore:document` (decision track creates the ADR and offers the rule/guide continuation per `track-layer.spec`), and keep it to 20 seconds or fewer and 12 visible lines in the final frame.
2. **Host the cast** on asciinema.org, which is free and embeds as SVG, referenced as the first element after the H1. The alternative is exporting to GIF with `agg`, committing it under `.github/assets/`, and referencing it directly.
3. **Add the before-and-after Mermaid diagram** immediately under the cast, rendered inline: three orphan boxes with no edges on the left, and typed nodes grouped by category with `implements`, `related`, and `depends_on` edges on the right.
4. **Write the three outcome bullets** as a flat markdown list with no sub-bullets — typed documents tracked by Git, auto-routing from natural language, and MCP-enforced writes with validation after every change.
5. **Reorder the rest of the README**, keeping the existing quick-start and install sections and moving the narrative below the hero, where it remains for deeper context without competing for first-screen space.
6. **Add CI for the visual asset** if a GIF is committed, failing when it exceeds 2 MB. An embedded asciinema SVG is an external asset and needs no check.

## Risks

- **Asset staleness.** The cast goes out of sync whenever intent-routing behavior changes. Mitigated by re-recording on any release that touches an intent skill, and by keeping the recording source — the exact prompt and setup — in a comment beside the embed so re-recording is cheap.
- **Rendering limits.** A GIF above 10 MB is rejected or renders slowly on mobile. An asciinema SVG is preferred: tiny payload, inline rendering, clickable for full playback.
- **Accessibility.** A GIF or SVG is opaque to a screen reader, so the section must carry descriptive alt text and a plain-text "what you'll see" paragraph beside the visual.
- **Positioning.** The comparable project's blog post includes a walkthrough but no hero at the top of its repository, so the hero is differentiation rather than mimicry. Its "20-minute interview" framing should not be mirrored, because the value proposition here is the opposite.
- **Privacy and branding.** A recording must not include absolute paths, hostnames in the prompt, or API keys echoed from the environment. Record in a throwaway repository with a clean environment.
- **Scope discipline.** The section MUST stay under 20 lines of Markdown, excluding the embedded asset. Beyond that it stops being a fast-scan hero and becomes narrative, which defeats its purpose.
