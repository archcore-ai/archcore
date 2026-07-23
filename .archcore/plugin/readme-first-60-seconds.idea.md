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

Add a tightly-scoped "First 60 seconds" section at the very top of `README.md` that lets a new visitor understand the value of the Archcore Plugin and see it working without scrolling past the fold. Three components:

1. **Hero visual (15–20 s)** — asciinema cast or animated GIF: the user types one natural-language prompt ("record the decision to use PostgreSQL"), and the terminal shows `decide` routing → ADR appears in `.archcore/` → relation suggestion. No voiceover, no slide transitions — raw terminal.
2. **Before / after diagram** — a two-column Mermaid flowchart immediately under the title: on the left, a flat pile of ad-hoc Markdown files; on the right, a typed, related graph with ADR → rule → guide, PRD → plan.
3. **Outcome bullets** — three lines that name the visible deltas the user will experience in the first session (e.g., "typed doc graph", "skill routing from natural language", "MCP-enforced writes") — each line ≤ 12 words.

Current state: the README starts with a long "What it does" + "Without vs. with Archcore" narrative that assumes the reader is already bought in. A visitor who doesn't already know what an ADR is closes the tab before reaching the value.

## Value

- **Adoption funnel top-of-funnel lift.** Claude Code marketplace traffic is high-intent but low-patience; the first visible screen decides whether the visitor installs or bounces. Comparable plugins with demo assets (Ars Contexta blog post, claude-code-spec-workflow screenshots) consistently surface in community "best plugins of 2026" roundups.
- **Zero-to-installed without jargon.** The visual shows what the plugin does before the reader needs to learn the vocabulary (ADR, MCP, intent routing, tracks, relations). The narrative section below can keep using domain terms.
- **Reduces support load.** Users who bounce after install often come back with "how do I use this?" — the hero reel answers that question before install, self-filtering confused users and attracting the right ones.
- **Cheap to produce.** One asciinema recording ≈ 10 min of work; exporting to SVG or GIF is one command.

## Possible Implementation

1. **Record the hero cast.**
   - Use `asciinema rec readme-hero.cast --idle-time-limit 1.5`.
   - Start from `claude` prompt in a fresh repo with `.archcore/` already initialized (or record the init as the first step).
   - Prompt: "record the decision to use PostgreSQL as our primary database — we considered DynamoDB and MongoDB".
   - Show Claude auto-invoking `/archcore:decide`, creating the ADR, and offering `rule + guide` follow-up.
   - Total ≤ 20 s, ≤ 12 visible lines in the final frame.
2. **Host the cast.**
   - Publish to asciinema.org (free, embeds via SVG).
   - Embed via `[![asciicast](https://asciinema.org/a/<id>.svg)](https://asciinema.org/a/<id>)` as the first element after the H1.
   - Alt: export via `agg` to GIF, commit to `.github/assets/readme-hero.gif`, reference via `![](./.github/assets/readme-hero.gif)`.
3. **Before / after Mermaid diagram.**
   - Place immediately under the asciinema. Render inline with triple-backtick `mermaid` block.
   - Left side: three orphan boxes (`notes.md`, `architecture.md`, `decisions.md`) with no edges.
   - Right side: typed nodes grouped by category (vision / knowledge / experience) with `implements`, `related`, `depends_on` edges.
4. **Outcome bullets.**
   - 3 lines, markdown list, no sub-bullets.
   - Draft:
     - "Typed documents (ADR, PRD, spec, rule …) in `.archcore/`, tracked by Git."
     - "Auto-routing from natural language — `decide`, `plan`, `capture` pick the right type."
     - "MCP-enforced writes: direct edits to `.archcore/` are blocked, validation runs after every change."
5. **Re-order the rest of the README.**
   - Keep the existing "Quick Start" and install sections.
   - Move "What it does" / "Without vs. with Archcore" below the hero (they remain for deeper context, just no longer compete with the hero for first-screen space).
6. **CI for visual assets.**
   - If committing a GIF, add a size check to CI: fail if `.github/assets/readme-hero.gif` > 2 MB.
   - If embedding asciinema, no CI check needed — external asset.

## Risks and Constraints

- **Asset staleness.** The hero cast will go out of sync whenever intent-routing behavior changes (e.g., Phase 1 description rewrite). Mitigation: re-record on every release that touches intent skills; track recording source (exact prompt + seed) in a comment near the embed so re-recording is cheap.
- **GitHub README rendering limits.** Large GIFs (> 10 MB) are rejected or rendered slowly on mobile. Asciinema SVG is preferred — tiny payload, renders inline, clickable for full playback.
- **Accessibility.** A GIF/SVG is opaque to screen readers. Always include a descriptive `alt` text and a plain-text "what you'll see" paragraph next to the visual.
- **Ars Contexta parallel.** Their blog post includes a walkthrough but no hero reel at the top of the repo — the README hero is differentiation, not mimicry. We should not mirror their "20-minute interview" framing; our value prop is the opposite (zero-config install).
- **License/branding constraints.** Asciinema recordings shouldn't include personal data (absolute paths, hostnames in prompt, API keys in env echoes). Record in a throwaway repo with a clean env.
- **Scope discipline.** The "First 60 seconds" section MUST stay under 20 lines of Markdown (excluding the embedded asset). If it grows, it stops being a fast-scan hero and turns into narrative — defeating the purpose.

## Related work in this repo

- The Inverted Invocation Policy ADR made auto-routing actually work; the hero reel is the single best way to show that routing in action.
- The Plugin Development Roadmap and Claude Plugin PRD track marketplace submission as open work — the hero asset is a prerequisite for competitive listings.
