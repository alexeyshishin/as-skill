# as-skill

CLI, который устанавливает домены и скиллы `as-skill` в `.claude/` целевого проекта.

## Сборка

Из корня репозитория (`go.mod` лежит там же, чтобы `domains/` и `core/` были
достижимы):

```bash
go build -o as-skill ./tools
```

или запуск прямо из исходников: `go run ./tools <команда>`.

## Использование

| Команда | Что делает |
|---------|-----------|
| `as-skill install domain <имя>` | symlink один домен (режим по умолчанию) |
| `as-skill install domains <имя> [имя...]` | symlink несколько доменов |
| `as-skill install all` | symlink все домены + core-скиллы |
| `as-skill install skill <имя>` | symlink один скилл (доменный или core) |
| `as-skill install ... --copy` | те же формы аргументов, но копия-снэпшот вместо symlink |
| `as-skill uninstall domain\|domains\|all\|skill ...` | удалить то, что поставил `install` (symlink для link-записей, реальные файлы/директории для copy-записей; чужого/неотслеживаемого не трогает) |
| `as-skill status [--project PATH]` | список записей lockfile: домен/вид/имя, режим, путь, здоровье (OK/MISSING/BROKEN) |
| `as-skill doctor [--project PATH] [--harness-root PATH]` | то же, что `status`, плюс проверки: источник symlink'а пропал, у copy-записи разошёлся хэш (DRIFTED), на диске есть неотслеживаемое (UNTRACKED); ненулевой exit при любой находке — годится как CI gate |
| `as-skill check` | валидирует `domains/*/manifest.yaml` и `skills/*/SKILL.md` в этой копии репозитория; без `--project` |
| `as-skill list [domains\|skills]` | показать, что можно установить |

`as-skill` должен запускаться изнутри (или из поддиректории) этой копии
репозитория, либо с явным `--harness-root` — он читает
`domains/*/manifest.yaml` и `core/skills/*` оттуда. `--project` — независимый
путь: проект, который вы оснащаете.

## Флаги

Общие для `install`/`uninstall`, если не указано иное:

| Флаг | Что делает | По умолчанию |
|------|-----------|--------------|
| `--project PATH` | корень целевого проекта | `.` |
| `--harness-root PATH` | эта копия репозитория | автоопределяется вверх от `.` |
| `--with-core` | доустановить/удалить `core/skills/*` вместе с `domain`/`domains`/`skill` | выключено (у `all` — всегда включено) |
| `--copy` | только `install`: копия-снэпшот вместо symlink | выключено (по умолчанию — symlink) |
| `--dry-run` | ничего не пишет, только печатает план | выключено |
| `--force` | только `install`: перезаписать существующее в целевом пути, даже если as-skill его не отслеживает | выключено (по умолчанию — отказ) |

Если `manifest.yaml` домена в `requires_env` называет неустановленную
переменную (сегодня это только `obsidian` → `$OBSIDIAN_VAULT`) — жёсткая
ошибка для явного `domain`/`domains`/`skill` и предупреждение + пропуск для
`all`. Есть и зеркальный `requires_bin` (проверяется через `$PATH`), но
сегодня его не объявляет ни один домен.
