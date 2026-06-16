---
title: "Fix Plan: Mounted Global Sources Prototype"
status: accepted
tags:
  - "config"
  - "globals"
  - "integrations"
  - "mcp"
  - "prototype"
---

## Resolution

**Superseded by the settings.json-only model.** Problems 1–5 are resolved; problem 6
(lifecycle / distribution) and problem 7 (version & plugin compatibility) remain open. Canonical docs:

- Decision: @.archcore/globals/global-sources-via-settings.adr.md
- Contract: @.archcore/globals/global-sources.spec.md
- Standards: @.archcore/globals/declaring-global-sources.rule.md, @.archcore/globals/local-overrides-global.rule.md
- Mandatory sources: @.archcore/globals/globals-are-mandatory.adr.md
- Vendoring how-to: @.archcore/globals/vendoring-a-global.guide.md

What changed since this plan was written: the launch-flag MVP (`--project` + `global: true`
marker) was built and then removed in favor of declaring globals only in the consumer's
`settings.json`. `path` now points directly at the global's `.archcore` directory (problem 5),
`isGlobalPath` computes real path prefixes in absolute space (problem 1), `annotateSource`
shares the same resolver (problem 2), `GlobalSource` is validated (problem 3), and the
per-entry `required` flag was dropped — every declared global is now mandatory, enforced at
scan time and at MCP startup (problem 4, see @.archcore/globals/globals-are-mandatory.adr.md).

---

## Goal

Исправить критические инженерные проблемы прототипа Mounted Global Sources,
выявленные при ревью. Проблемы 1–4 реализуются одним PR; 5–6 — отдельные решения.

### Context

Прототип реализован в `internal/mcp/tools/`, `internal/config/config.go`,
`templates/templates.go`. Fixture — `examples/07-local-overrides-global/` (с общим источником `examples/_global_/company-standards/`).
RFC — см. `Mounted Global Sources.pdf` в корне проекта.

---

## Проблемы и план исправления

### Проблема 1 — `isGlobalPath` работает только для конвенции `.archcore/global/` (HIGH) ✅ РЕШЕНО

`isGlobalPath` проверял наличие строки `.archcore/global/` в пути. Если `gs.Path = "../company-repo"` —
охрана не срабатывала. **Реализовано**: `isGlobalPathAbs(baseDir, relPath, globals)` вычисляет
реальные path-префиксы из `gs.Path` в абсолютном пространстве (`common.go`).

### Проблема 2 — `annotateSource` и `scanDocuments` — расходящаяся логика (MEDIUM) ✅ РЕШЕНО

**Реализовано**: `annotateSource` использует тот же `resolveGlobalPath` resolver — расхождение устранено.

### Проблема 3 — `GlobalSource` не валидируется (MEDIUM) ✅ РЕШЕНО

**Реализовано**: `Settings.Validate()` проверяет пустой/битый/зарезервированный (`local`)/
дублирующийся `id` и пустой `path`. (Запрет `../` снят сознательно — кросс-проектные ссылки штатны.)

### Проблема 4 — `required: true` — мёртвое поле (LOW/MEDIUM) ✅ РЕШЕНО

**Реализовано (позже уточнено)**: поле `required` в итоге удалено — все globals обязательны.
Отсутствие источника — ошибка в `scanDocuments` (scan time) и в `checkGlobals` на старте
MCP-сервера. См. @.archcore/globals/globals-are-mandatory.adr.md.

### Проблема 5 — Double-nesting в `path` (DESIGN) ✅ РЕШЕНО

**Решение принято и реализовано**: `path` указывает прямо на `.archcore`-каталог источника,
без автодобавления сегмента. `path = "../company-standards/.archcore"`.

### Проблема 6 — Нет lifecycle management (OPERATIONAL) ⏳ ОТЛОЖЕНО

Globals нужно вручную клонировать и обновлять. Нет `archcore globals pull`, lockfile,
`archcore status` для mounted sources, `archcore doctor` для broken globals.
**Статус**: отдельный milestone, ещё не начат. Промежуточная ручная дистрибуция описана в
@.archcore/globals/vendoring-a-global.guide.md.

### Проблема 7 — Версионная и плагинная совместимость globals (HIGH) 🔧 В РАБОТЕ

`globals` живёт в `settings.json`, а `Settings.UnmarshalJSON` строго отвергает неизвестные
поля → уже выпущенные «старые» CLI ловят `field "globals" is not allowed` и роняют каждый
путь, грузящий конфиг (`mcp`/`hooks`/`doctor`/`status`). Бьёт по тем, кто получил `globals`
в конфиге, но не обновил CLI.

Асимметрия: **новый CLI + старый плагин — безопасно** (плагин globals-агностичен, получает
фичу прозрачно); **новый плагин + старый CLI** ломается только в клетке «старый CLI ×
`globals` в конфиге» — причина CLI+конфиг, не версия плагина.

**Работа разнесена на два исполнительных плана** (отдельно CLI, отдельно plugin):

- **CLI** → @.archcore/features/globals-cli-forward-compat.plan.md — soft-ignore парсинга
  конфига + сиквенсинг релизов.
- **Plugin** → @.archcore/features/globals-plugin-compat.plan.md — инвариант агностичности +
  compatibility-advisory + grep-hygiene.

---

## Tasks

- [x] Рефактор `isGlobalPath` → принимает `[]GlobalSource`, вычисляет реальные префиксы
- [x] Унифицировать `annotateSource` с тем же resolver (устраняет проблему #2)
- [x] Валидация `GlobalSource` в `Settings.Validate()` (проблема #3)
- [x] Принять решение по `required` полю: поле удалено, все globals обязательны (проблема #4)
- [x] Принять дизайн-решение по проблеме #5 (path указывает на `.archcore`-каталог)
- [ ] Lifecycle management (`archcore globals pull` и др.) — отдельный milestone (проблема #6)
- [ ] Версионная/плагинная совместимость — два отдельных плана (проблема #7): CLI и Plugin (см. ссылки выше)