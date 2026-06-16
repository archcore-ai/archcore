---
title: "Plan: Forward-Compatible Config Parsing"
status: accepted
tags:
  - "cli"
  - "config"
  - "mcp"
---

## Goal

Сделать разбор `.archcore/settings.json` **forward-compatible**: более старый `archcore`
не должен крашиться и не должен терять функционал, встретив в конфиге поле, которого он не
знает (добавленное более новым CLI). Свойство **field-agnostic** — применяется к любому
неизвестному ключу единообразно, без привязки к конкретной фиче.

> **Статус: реализовано** (core + tests). Остаётся только релиз-процесс (сиквенсинг) и
> опциональный `ARCHCORE_AUTO_UPDATE`. Реализация: `internal/config/config.go`
> (`Settings.Extra`, `knownFields`, `UnmarshalJSON`, `MarshalJSON`, `UnknownFieldNames`),
> warning на entry-points (`cmd/config_warn.go` + `cmd/mcp.go`/`config.go`/`doctor.go`/`sync.go`).

### Проблема

`Settings.UnmarshalJSON` (`internal/config/config.go`) строго отвергает любое неизвестное
поле (`field %q is not allowed for sync type …`). Поэтому **любое** новое поле конфига
роняет на старом CLI каждый путь, грузящий конфиг (`mcp`/`hooks`/`doctor`/`status`). Уже
выпущенные строгие бинари переучить нельзя — лечится только терпимым парсером, выпущенным
**впредь** (и как можно раньше).

## Решение

### A. Soft-ignore неизвестных полей — три свойства (унифицированно, без условий)

1. **Read-tolerant.** Неизвестный ключ → захватывается в `Settings.Extra`, не ошибка. Цикл
   проверки полей различает три случая: allowed-for-mode → декод; известное-но-не-для-этого-
   режима → прежняя жёсткая ошибка; неизвестное этому бинарю → в `Extra`.
2. **Keep-serving.** После игнора конфиг догружается, известные поля заполняются как обычно.
3. **Write-preserving.** При записи `settings.json` (`config set`) неизвестные ключи
   **сохраняются** (merge-tail в `MarshalJSON`: пустой `Extra` → байт-в-байт как раньше;
   непустой → объединение с raw-map), иначе терпимый-но-старый CLI молча сотрёт поле.

Предупреждение печатается **в stderr только на user-facing командах** (`mcp` startup,
`config`, `doctor`, `sync`) — `UnmarshalJSON`/`Load` молчат (горячие пути). **Валидация
значений известных полей остаётся строгой**. Размен: теряется защита от опечатки в *имени*
поля — осознанно, ради forward-compat (разворот строгости из `backup-invalid-configs.adr`).

### B. Сиквенсинг релизов

Терпимый парсер (A) должен выйти в релизе **не позже** того, что вводит новое поле конфига,
и желательно раньше. Он **не чинит задним числом** уже выпущенные строгие бинари — те
по-прежнему отвергают новый конфиг; для них смягчение — на стороне потребителя (нудж).
Поэтому терпимость выкатывать как можно раньше, отдельным шагом.

### C. (опц.) ARCHCORE_AUTO_UPDATE — только opt-in

`archcore update` уже умеет self-replace. Авто-запуск из любого хук-пути — НЕ по умолчанию:
подмена глобального бинаря без спроса, не рестартит уже запущенный MCP (лечит лишь next
session), падает в CI/офлайн/locked, ломает запиненные версии. Допустимо лишь под явным
`ARCHCORE_AUTO_UPDATE=1`.

## Tasks

- [x] `config.UnmarshalJSON`: неизвестное поле → захват в `Extra` (не ошибка); known-wrong-mode
  по-прежнему ошибка; строгая валидация значений известных полей сохранена
- [x] `Settings.MarshalJSON`: round-trip-сохранение неизвестных полей (merge-tail, byte-identical при пустом `Extra`)
- [x] Warning на entry-points в stderr (`cmd/config_warn.go`; `mcp`/`config`/`doctor`/`sync`)
- [x] Тест: парсер на конфиге с неизвестным полем → не ошибка, известные поля заполнены, обслуживание проходит
- [x] Тест: `config set <известное-поле>` не теряет нераспознанное поле (round-trip, e2e)
- [x] Тест (regression): known-wrong-mode и битые значения известных полей по-прежнему ошибка
- [x] Тест: MCP startup (`checkGlobals`) и in-process server терпят неизвестное поле
- [ ] Релиз-процесс: выпускать терпимый парсер не позже добавления нового поля конфига
  (зафиксировать в `how-to-release.guide`)
- [ ] (опц.) `ARCHCORE_AUTO_UPDATE=1` opt-in self-heal, по умолчанию off

## Acceptance Criteria

- Терпимый CLI на конфиге с неизвестными полями: ноль ошибок, известные поля работают, все
  config-загружающие команды (`mcp`/`hooks`/`doctor`/`status`) выполняются. ✅
- `config set` любым полем не удаляет нераспознанные поля из `settings.json`. ✅
- Известные поля по-прежнему строго валидируются (битый `project_id` → ошибка). ✅

## Dependencies

- **Сиквенсинг**: терпимый-парсер-релиз — не позже релиза с новым полем конфига.
- Затрагивает `internal/config/config.go` (`UnmarshalJSON`, `MarshalJSON`, `allowedFields`,
  `knownFields`) и релиз-процесс (`how-to-release.guide`).