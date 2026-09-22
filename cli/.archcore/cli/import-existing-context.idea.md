---
title: "Import Existing Context from CLAUDE.md, .cursorrules, and Other Agent Configs"
status: draft
tags:
  - "cli"
---

## Idea

An `archcore import` command that takes existing context files (CLAUDE.md, .cursorrules, .github/copilot-instructions.md, README) and converts them into structured `.archcore/` documents. Lowers the entry barrier — users don't start from scratch, they bring along work they've already invested in.

### Problem / Opportunity

- After `archcore init`, a new user sees an empty directory — no sense of immediate value
- Many teams have already invested in CLAUDE.md, .cursorrules, copilot-instructions — this context can be reused
- Competitive advantage: "you've already written everything — we'll just bring it along"

## Value

### For Users

- Instant start: `archcore import --from CLAUDE.md` — and `.archcore/` is already populated
- No need to rewrite existing documentation by hand
- The agent gets context from the very first session

### For Business

- Lowers the switching barrier from competitors (cursor rules → archcore)
- Demonstrates archcore's value on already familiar content
- Increases retention — users see results immediately

### For Team

- Simple feature for demos and marketing
- Good entry point for new users in documentation

## Possible Implementation

### Technical Approach

1. **Detect source format** by filename or flag:
   - `CLAUDE.md` → markdown with sections (build commands, architecture, etc.)
   - `.cursorrules` → plain text with instructions
   - `.github/copilot-instructions.md` → markdown
   - `README.md` → standard readme

2. **Parse and map**: split the file into logical blocks, each block → a separate archcore document:
   - Build/test instructions → `guide`
   - Rules and conventions → `rule`
   - Architecture/overview → `doc`
   - If meaningful splitting isn't possible → a single `doc` with the imported content

3. **CLI interface**:
   ```
   archcore import --from CLAUDE.md
   archcore import --from .cursorrules
   archcore import --auto  # auto-detect known files in repo
   ```

4. **Auto-detect on init**: optionally suggest import when known files are found

### Integrations

- Can work as pure Go parsing (no LLM) for simple cases
- For smarter document splitting — optionally via LLM (V2)

## Risks and Constraints

### Potential Risks

- Parsing quality: CLAUDE.md / .cursorrules have no standard structure, format varies from project to project
- Users may expect "smart" conversion but get a rough import
- Duplicates: if the user imports twice

### Known Constraints

- Without LLM, splitting into document types will be approximate
- Need to decide: one large document vs attempting to split into several

## Next Steps

- [ ] Define MVP scope: support CLAUDE.md + .cursorrules (most popular)
- [ ] Decide parsing strategy: single doc vs attempt to split
- [ ] Prototype on CLAUDE.md from this repository
- [ ] Integration with `archcore init` (suggest import when files are found)

## Related Materials

- Current `archcore init` flow: `cmd/init.go`
- Document templates: `templates/templates.go`
