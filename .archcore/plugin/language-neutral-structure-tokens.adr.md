---
title: "Structure Tokens Stay English in Documents of Any Language"
status: accepted
tags:
  - "component:cli"
  - "component:plugin"
  - "docs-style"
  - "precision"
---

## Context

The MCP language directive in `@cli/internal/mcp/server.go` keeps frontmatter keys and status values in English and says nothing about section headings or requirement keywords. A field audit of a Russian corpus (294 documents, 2026-10-06) found 7 ways of stating an obligation across 54 `rule` documents — BCP 14 keywords in 10, `ДОЛЖЕН` in 12, descriptive present tense in 11, the rest lowercase or imperative — and 19 `rule` documents headed `## Правило`, which the English-only section match in `@cli/templates/precision.go` does not see. The engine has no finding for a graded clause without a modal, so 56% of measured `rule` clauses carried none and nothing reported it.

## Decision

In a document of any language, the `##` section headings of the type template, the BCP 14 keywords (`MUST`, `MUST NOT`, `SHOULD`, `SHOULD NOT`, `MAY`), and the EARS keywords (`WHEN`, `WHILE`, `IF`, `THEN`, `WHERE`) stay as English tokens, and the rest of each line, including the actor, is written in the document language without English articles.

## Alternatives Considered

1. Native keyword sets per language mapped to BCP 14, such as `ДОЛЖЕН` / `СЛЕДУЕТ` / `МОЖЕТ` — rejected because each language needs its inflected forms and casing rules, scripts without letter case (Chinese, Japanese) cannot mark a keyword by uppercase, and every agent and the engine would then parse one vocabulary per language. [assumption] ISO/IEC Directives Part 2 pair the English verbal forms with French ones, which shows the approach works for a closed set of languages, not for an open one.
2. Localized heading aliases per language in `RequiredSections` — rejected because the alias set never closes: one corpus already used `Правило`, `Обоснование`, `Зачем`, `Соблюдение`, and `Шаги` for three English headings, and each new language multiplies the table.
3. English for the whole document regardless of the `language` setting — rejected because `language` is an accepted setting (`cli-ui/language-setting.idea`) and a team writes its rationale and context better in its own language.

## Consequences

- One token set parses in every script; an English uppercase keyword stays visible inside Cyrillic, CJK, or Arabic text.
- The engine can add a finding for a graded clause with no BCP 14 keyword and, from a per-language data table, name the native modal it found (`должен`, `muss`, `doit`, `必须`) with its replacement. A new language is one table row, not a code change.
- The section-heading checks work unchanged in every language.
- Cost: a non-English graded line mixes two languages by design, for example `WHEN заказ оплачен, сервис оплаты MUST отправить чек.`
- Cost: existing non-English documents with translated `##` headings report missing sections until the headings are restored.

## Superseded when

- A host or the engine adds per-language keyword parsing with a closed, tested vocabulary for at least the languages that `language` settings in the field use.
- Authors in a non-English corpus keep translating headings after the directive change: more than 10% of new documents in 30 days, measured by the missing-section finding.
