# code hooks

Shell-хуки Claude Code для домена `code`. Хуки Memory Bank живут в `core` (`domains/core/hooks/`). Ставятся installer-ом в `<project>/.claude/hooks/`, регистрируются в `<project>/.claude/settings.json` скиллом `/code-init`.

Все хуки fail-open: любая ошибка (`trap 'exit 0' ERR`) — хук молча пропускается, сессию не ломает. Без `jq` все они no-op.

## Хуки

| Хук | Событие | Что делает |
|-----|---------|------------|
| `test-gate.sh` | `Stop` | Блокирует «готово», если правились кодовые файлы (`ts tsx js jsx py dart swift kt go rb java tf`), а тестовая команда за сессию не запускалась. Просит прогнать тесты и показать реальный вывод. |
| `prettier-format.sh` | `PostToolUse` (`Edit\|Write\|MultiEdit\|Bash`) | Форматирует prettier-ом, но только когда условие test-gate выполнено. `Edit`/`Write`/`MultiEdit` — один файл; `Bash` с тестовой командой — все файлы, правленные за сессию. |
| `test-gate-lib.sh` | — | Не хук. Общая библиотека для `test-gate` и `prettier-format`: единственное определение «test-gate выполнен». |

## test-gate

- Тестовая команда определяется по проекту: `pubspec.yaml` → `flutter test`, `pyproject.toml` → `pytest`, `go.mod` → `go test`, `build.gradle[.kts]` → `./gradlew test`, `package.json` со `scripts.test` → `npm test`, `Makefile` с целью `test:` → `make test`. Не найдена — гейт ничего не требует.
- Защита от зацикливания: не более 2 блокировок на один `git HEAD` (счётчик в `.claude/sessions/<session_id>.testcount`); внутри Stop-continuation не блокирует повторно.
- Если тест реально нельзя запустить, агент должен объяснить почему.

## prettier-format

- No-op без prettier (локальный `node_modules/.bin/prettier` или глобальный) и без prettier-конфига проекта (`.prettierrc*`, `prettier.config.*` или ключ `prettier` в `package.json`) — чужие дефолты проекту не навязываются.
- Форматирует только файлы внутри проекта: `ts tsx js jsx mjs cjs json css scss html vue`.
- Stop срабатывает после всех `PostToolUse`, поэтому ждать `test-gate` хук не может — он перепроверяет то же условие через `test-gate-lib.sh`.
- Если что-то переформатировано, сообщает агенту список файлов — перечитать перед следующей правкой.

## Регистрация

Хук сам объявляет событие и matcher строкой в шапке:

```
# register: <Event> [<matcher>]
```

`skills/code-init/scripts/register-hooks.sh [--project DIR] [--dry-run]` читает эти строки и дописывает недостающие записи в `<project>/.claude/settings.json`. Идемпотентен, чужие ключи не трогает, глобальный `~/.claude/settings.json` не пишет. Новый хук = новый файл с `# register:`, правки скрипта не нужны.

## Установка

```
as-skill install domains code core --project <target> --copy
```

Затем `/code-init` в целевом проекте — зарегистрирует хуки (с согласия пользователя).

## Удаление

```
as-skill uninstall domain code --project <target>
```

Снимает файлы хуков и заодно их записи из `<project>/.claude/settings.json`: только команды, указывающие на снятые файлы (`"$CLAUDE_PROJECT_DIR"/.claude/hooks/<name>` и эквиваленты). Остальные ключи и чужие хуки остаются, порядок ключей сохраняется (файл перезаписывается с отступом 2 пробела), `settings.local.json` не трогается. Хук, который `as-skill` не отслеживает, не снимается и остаётся зарегистрированным. `--dry-run` печатает `would unregister`.
