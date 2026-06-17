---
title: "Plan: Plugin Compatibility for the Globals Rollout (Plugin)"
status: accepted
tags:
  - "cli"
  - "globals"
  - "integrations"
  - "mcp"
---

## Goal

Гарантировать, что плагин остаётся совместим с **любой** версией CLI (старой и новой), не
создаёт опасную клетку «старый CLI × globals» и мягко доводит не-обновившихся до апдейта CLI.
Код плагина — в отдельном репозитории `archcore/plugin` (`plugins/archcore`). Контекст —
проблема 7 зонтичного плана (@.archcore/features/globals-prototype-fixes.plan.md).

## Решение

### A. Инвариант агностичности плагина (4 правила)

1. Никогда не писать `globals` в `settings.json` (в т.ч. `/archcore:init`).
2. В исполняемом пути — только стабильные глаголы CLI (`mcp`/`hooks`/`doctor`/`--version`);
   новые команды/флаги — лишь в тексте-подсказке.
3. Новые поля MCP (`source_kind`/`read_only`/`source_id`) — опциональны. **В `bin/`
   (исполняемый код) — не ветвить логику на их наличие** (на старом CLI их нет). **Skills
   МОГУТ их читать и сурфейсить globals — но каждая клауза обязана быть data-gated**:
   отсутствие полей ⇒ поведение идентично пути без globals.
4. Version-mismatch → нудж, никогда не hard-block.

**Ревизия (2026-06-17): плагин понимает globals.** Инвариант больше не «0 упоминаний globals
в `skills/`», а набор свойств выше: `bin/` остаётся агностичным; skills читают опциональные
поля только data-gated (absent ⇒ как без globals); плагин никогда не пишет `globals`. Защита —
property-based чек `test/structure/cli-compat-invariant.bats` в репо `archcore/plugin`. Причина
разворота: поля живые на всех трёх read-инструментах (`global-sources.spec` §4.1), а плагин —
их штатный потребитель; грубый прокси `grep=0` мешал capability, не добавляя реальной защиты.

### B. Compatibility-advisory в bin/session-start

Локальный детект без сети: `grep -q '"globals"' .archcore/settings.json` + факт ошибки вызова
`archcore hooks … session-start` (либо `archcore --version` < min). При мисматче — не прокидывать
криптокраш, выдать «обнови archcore CLI: `archcore update`», сессию продолжить. Разово,
rate-limit через stamp-файл (паттерн `check-staleness`). Версионно-общий, не только под globals.

### C. Grep-hygiene для FS-сканирующих скриптов

`bin/check-staleness` (`grep -rl … .archcore/`) и `bin/check-code-alignment`
(`grep -rlF … .archcore --include='*.md'`) обходят весь `.archcore/`, включая in-tree
`.archcore/global/…`, и шумят нуджами на read-only глобалах. Добавить `--exclude-dir=global`
и/или пропуск объявленных globals. Soft, не блокирует — но шум убрать.

## Tasks

- [x] Зафиксировать инвариант агностичности — property-based чек `cli-compat-invariant.bats`
  в репо `archcore/plugin` (bin/ hard-zero на поля; skills — data-gated; никогда не писать `globals`)
- [ ] `bin/session-start`: compatibility-advisory (локальный детект + rate-limited нудж на апдейт CLI)
- [ ] `bin/check-staleness`: пропускать `.archcore/global` (+ объявленные globals)
- [ ] `bin/check-code-alignment`: то же
- [x] Ревью: ни один skill/bin не пишет `globals`; `bin/` не ветвится на `source_kind`/`read_only`;
  skills читают их только с absent-guard (`skills/_shared/globals.md` + клаузы context/audit/capture/decide/plan)

## Acceptance Criteria

- Новый плагин на старом CLI без globals — работает как раньше.
- На «старый CLI × globals в конфиге» — вместо криптокраша понятный разовый нудж.
- `check-staleness`/`check-code-alignment` не шумят на in-tree globals.
- `bin/` не содержит `source_kind|read_only|source_id` и не пишет `globals`; каждый skill,
  читающий эти поля, имеет absent-default guard (старый CLI без полей ⇒ поведение как раньше).

## Dependencies

- Нудж адресует апдейт CLI → опирается на CLI-план: `globals-cli-forward-compat.plan`.
- Решение по globals: `global-sources-via-settings.adr`.
- Код в отдельном репозитории `archcore/plugin` — этот план координационный.
