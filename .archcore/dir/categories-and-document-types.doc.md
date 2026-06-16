---
title: "Categories and Document Types"
status: accepted
tags:
  - "directory-structure"
---

## Overview

The high-level conceptual model — the three virtual categories (**vision**, **knowledge**, **experience**), the `slug.type.md` naming, and the document tracks — lives in the `archcore` global source (`concepts/core-concepts`, `concepts/document-tracks`). This document is the CLI's **detailed type reference and selection guide**: the per-type tables, the requirements layers, and the "choosing the right type" matrix the engine's templates depend on. The category is derived from the document type in the filename, not from the directory path.

## Vision

Documents that describe the future: what we want to build and why.

### Product Track (Simple)

| Type   | Purpose                                                    |
| ------ | ---------------------------------------------------------- |
| `prd`  | Product requirements — goals, scope, acceptance criteria   |
| `idea` | A concept worth exploring — problem, value, rough approach |
| `plan` | A concrete implementation plan with phased tasks           |

### Sources Track (Discovery)

| Type  | Purpose                                                                      |
| ----- | ---------------------------------------------------------------------------- |
| `mrd` | Market analysis — TAM/SAM/SOM, competitive landscape, market needs, timing   |
| `brd` | Business justification — objectives, ROI, stakeholders, budget, constraints  |
| `urd` | User needs — personas, journeys, usability requirements, acceptance criteria |

The sources track captures **where** requirements come from (market → business → users). Documents flow naturally: MRD (market landscape) → BRD (business justification) → URD (user needs).

### ISO Track (Decomposition)

| Type   | ISO Reference | Purpose                                                                                      |
| ------ | ------------- | -------------------------------------------------------------------------------------------- |
| `brs`  | ISO §9.3      | Business requirements specification — mission, goals, operational concept, success criteria   |
| `strs` | ISO §9.4      | Stakeholder requirements specification — per-class requirements with ConOps, compliance       |
| `syrs` | ISO §9.5      | System requirements specification — system boundary, interfaces, modes, verification approach |
| `srs`  | ISO §9.6      | Software requirements specification — per-function/per-endpoint specs, verification matrix    |

The ISO track decomposes requirements through progressively detailed levels: BRS (why the business needs it) → StRS (what stakeholders need) → SyRS (how the system behaves) → SRS (how the software works).

### Requirements Layers — Sources vs Specifications

Sources and Specifications are **separate layers**:

- **Layer A (Sources):** mrd, brd, urd, prd — capture raw requirements from market, business, and user perspectives
- **Layer B (Specifications):** brs, strs, syrs, srs — formalize requirements into ISO-structured specifications

Specifications formalize sources via `implements` relation (spec is the source, source doc is the target):

| Specification | Formalizes | Relation Example |
|---------------|------------|------------------|
| BRS | MRD (market needs), BRD (business objectives) | `brs implements mrd`, `brs implements brd` |
| StRS | URD (user needs), BRS (ISO cascade) | `strs implements urd`, `strs implements brs` |
| SyRS | StRS (ISO cascade) | `syrs implements strs` |
| SRS | SyRS (ISO cascade) | `srs implements syrs` |
| PRD | ≈ all four ISO types (use `related`) | PRD is a pragmatic hybrid covering all levels |

Do NOT confuse source documents (mrd/brd/urd) with specification documents (brs/strs/syrs/srs). Sources are informal, discovery-oriented. Specifications are formal, ISO-structured.

## Knowledge

Documents that capture what we know: decisions, standards, contracts, and reference material.

| Type      | Purpose                                                                                                     |
| --------- | ----------------------------------------------------------------------------------------------------------- |
| `adr`     | A technical decision that has been made, with context and alternatives                                      |
| `rfc`     | A proposal open for review before a decision is made                                                        |
| `rule`    | A mandatory standard — imperative statements with good/bad examples                                         |
| `guide`   | Step-by-step instructions for completing a task                                                             |
| `spec`    | Canonical normative contract — behavior, constraints, invariants, conformance for a specific technical boundary |
| `doc`     | Non-behavioral reference — tables, registries, glossaries, component lists                                  |

## Experience

Documents that encode proven patterns and lessons from practice.

| Type        | Purpose                                                                             |
| ----------- | ----------------------------------------------------------------------------------- |
| `task-type` | A proven workflow for a recurring implementation task — steps, examples, pitfalls   |
| `cpat`      | A code pattern change — how and why a convention or approach changed (was → became) |

## Choosing the Right Type

- **rule vs doc** — rule prescribes behavior ("Always do X") with enforcement. doc describes what exists (tables, registries). Descriptive, non-behavioral content → doc.
- **adr vs rfc** — adr = decision already final. rfc = proposal open for feedback.
- **guide vs doc** — guide has sequential steps to follow. doc is non-sequential reference to look up.
- **spec vs doc** — spec defines a canonical normative contract for a concrete technical boundary (behavior, constraints, invariants, conformance). doc describes what exists without normative requirements. Normative contract → spec; structural reference → doc.
- **spec vs rule** — spec is a technical contract for one component. rule is a cross-cutting team standard. Scoped to a named artifact → spec; applied team-wide → rule.
- **spec vs adr** — spec is the living canonical truth (present-tense: "it works this way"). adr is the decision record (past-tense: "we chose this because"). Both may exist for the same component.
- **task-type vs guide** — task-type is a reusable pattern for a class of tasks (e.g., "how to create a UI-kit component"). guide is instructions for a specific one-time procedure.
- **cpat vs adr** — cpat focuses on a code pattern change with before/after examples. adr records a broader architectural decision with alternatives and consequences.
- **mrd vs prd** — MRD analyzes the MARKET (TAM/SAM/SOM, competitors, timing) without proposing a solution. PRD proposes a PRODUCT with requirements and solution overview.
- **brd vs prd** — BRD focuses on BUSINESS JUSTIFICATION (ROI, budget, organizational impact). PRD focuses on PRODUCT DEFINITION (features, user stories, solution).
- **urd vs prd** — URD captures user needs via PERSONAS and JOURNEYS (discovery-oriented). PRD defines product requirements with acceptance criteria (specification-oriented).
- **mrd vs brd** — MRD is MARKET ANALYSIS (external-facing — industry, competitors, TAM). BRD is BUSINESS JUSTIFICATION (internal-facing — ROI, stakeholders, budget).
- **brd vs urd** — BRD captures ORGANIZATIONAL needs (goals, budget, regulations). URD captures END-USER needs (personas, journeys, usability).
- **brs vs prd** — BRS has ONLY business objectives with ISO structure (mission, operational concept, success criteria), no user stories. PRD has user stories, requirements by priority, solution overview.
- **strs vs prd** — StRS groups requirements PER STAKEHOLDER CLASS with ConOps. PRD lists by priority (P0/P1/P2).
- **syrs vs adr** — SyRS defines WHOLE SYSTEM BOUNDARY with interface contracts and verification. ADR records a single decision.
- **srs vs prd** — SRS has PER-ENDPOINT/PER-FUNCTION requirements with verification matrix. PRD has product-level requirements.
- **brs vs strs** — BRS = WHY (business outcomes, technology-agnostic). StRS = WHAT stakeholders need (operational scenarios, solution-aware).
- **syrs vs srs** — SyRS = WHOLE SYSTEM boundary. SRS = SINGLE COMPONENT's detailed behavior.
- **brs vs brd** — BRS is ISO SPECIFICATION (formalized structure). BRD is INFORMAL SOURCE (business justification, ROI). BRS formalizes what BRD captures informally.
- **strs vs urd** — StRS is ISO SPECIFICATION (per-class requirements with ConOps). URD is INFORMAL SOURCE (personas, journeys). StRS formalizes what URD captures informally.

## Choosing the Right Requirements Track

Three approaches to requirements engineering — choose based on project complexity:

| Track | Documents | Best For |
|-------|-----------|----------|
| Product (simple) | `prd` | Individual features, small teams, rapid prototyping, internal tools |
| Sources (discovery) | `mrd` → `brd` → `urd` | Product teams doing discovery, stakeholder alignment, business analysis |
| ISO (decomposition) | `brs` → `strs` → `syrs` → `srs` | Regulated systems, multi-team projects, complex distributed systems |

All tracks can coexist — use what fits the project. Start simple (PRD), add sources when you need stakeholder alignment, add ISO when you need formal traceability.
