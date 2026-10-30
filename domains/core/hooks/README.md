# core hooks

Shell-хуки Claude Code для домена `core`. Ставятся installer-ом в `<project>/.claude/hooks/`, регистрируются в `<project>/.claude/settings.json` по явному выбору на шаге `hooks` диалога `as-skill add` (по умолчанию ничего), скиллом `/code-init` или `as-skill hooks enable` (механизм `# register:` и удаление — см. `domains/code/hooks/README.md`).

Хуки fail-open: любая ошибка (`trap 'exit 0' ERR`) — хук молча пропускается. Без `jq` — no-op.

## Хуки

| Хук | Событие | Что делает |
|-----|---------|------------|
| `memory-bank-load.sh` | `SessionStart` (`startup\|clear\|compact`) | Кладёт в контекст `.memory-bank/index.md` (до 16 KB) и список остальных файлов Memory Bank (до 100). |
| `memory-bank-logger.sh` | `SessionEnd` | Фоном запускает `claude -p`: тот читает транскрипт сессии и по скиллу `memory-bank` дополняет `.memory-bank/`. |

## memory-bank-load

- No-op без `.memory-bank/index.md`.
- Индекс усекается на 16 KB с пометкой; остальные файлы агент читает по необходимости, а не платит за все каждую сессию.

## memory-bank-logger

- No-op без `jq`, `claude`, `.memory-bank/index.md`, транскрипта и для сессий короче 3 пользовательских промптов.
- Возвращается сразу: `claude -p` стартует отвязанным процессом (`--permission-mode acceptEdits`, инструменты `Read Glob Grep Edit Write`), лог — `.claude/sessions/memory-bank-logger.log`. Правит только `.memory-bank/`; нечего добавлять — ничего не меняет.
- Защита от рекурсии: дочерний процесс идёт с `MEMORY_BANK_LOGGER=1`, его собственный SessionEnd хук пропускает. Одна сессия логируется один раз (`.claude/sessions/<session_id>.mblogged`), параллельный запуск блокирует lock-каталог (протухает через 10 минут).
- Платный: каждая подходящая сессия = один headless-вызов модели.

## Платформы

| Хук | claude | codex | cursor | opencode |
|-----|--------|-------|--------|----------|
| `memory-bank-load.sh` (`# platforms: claude,codex,cursor`) | supported | unverified | unverified | unsupported |
| `memory-bank-logger.sh` (`# platforms: claude`) | supported | unsupported | unsupported | unsupported |

Для Codex `SessionStart` принят Claude-совместимый `hookSpecificOutput.additionalContext` по документации (2026-10-27): UNVERIFIED, реальный payload не снимался. Codex требует trust хуков (`/hooks`). Для Cursor вывод `{"additional_context": ...}` по документации (UNVERIFIED). Остальные платформы — см. `domains/code/hooks/README.md`.
