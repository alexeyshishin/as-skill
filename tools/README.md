# as-skill

CLI, который устанавливает домены и скиллы `as-skill` в `.claude/` целевого проекта.

## Сборка

Из корня репозитория (`go.mod` лежит там же, чтобы `domains/` был
достижим):

```bash
go build -o as-skill ./tools
```

или запуск прямо из исходников: `go run ./tools <команда>`.

### `install.sh`

`source install.sh` из корня репозитория собирает `as-skill`, сразу добавляет его в `PATH` текущей оболочки (sh/bash/zsh) и сохраняет PATH для новых терминалов: путь пишется в `~/.config/as-skill/env`, а в rc-файл (`~/.zshrc` для zsh; `~/.bash_profile` и `~/.bashrc` для bash; `~/.profile` иначе) дописывается одна guard-строка без machine-specific пути — безопасно для dotfiles-репозиториев. Уже открытые терминалы подхватят PATH ещё одним `source install.sh`.

Без `source` (`./install.sh`, `bash install.sh`) бинарь соберётся, но в PATH родительской оболочки не попадёт (дочерний процесс так не умеет) — скрипт это видит и подсказывает запустить через `source`. Для новых терминалов PATH сохранится в любом случае.

### Платформы

`as-skill` — обычный Go-бинарь без platform-specific кода: macOS, Linux, Windows.

- **Windows, `install.sh`** — это `sh`-скрипт: запускайте через Git Bash или WSL, либо соберите напрямую: `go build -o as-skill.exe ./tools`.
- **Windows, symlink-режим** — `os.Symlink` требует привилегию `SeCreateSymbolicLinkPrivilege`: нужен включённый **Developer Mode** (Windows 10 1703+ / 11) или терминал **от администратора**. Иначе установка падает с ошибкой прав — ставьте с `--copy` (live-правки из `domains/` тогда не подхватываются, после изменений нужна переустановка).
- **fish** — `install.sh` правит только POSIX rc-файлы и `config.fish` не трогает: определяет это по `$SHELL` и печатает подсказку. Добавьте PATH вручную: `fish_add_path <путь к чекауту as-skill>`.

## Использование

| Команда | Что делает |
|---------|-----------|
| `as-skill install domain <имя>` | symlink один домен (режим по умолчанию) |
| `as-skill install domains <имя> [имя...]` | symlink несколько доменов |
| `as-skill install all` | symlink все домены |
| `as-skill install skill <имя>` | symlink один скилл |
| `as-skill install ... --copy` | те же формы аргументов, но копия-снэпшот вместо symlink |
| `as-skill uninstall domain\|domains\|all\|skill ...` | удалить то, что поставил `install` (symlink для link-записей, реальные файлы/директории для copy-записей; чужого/неотслеживаемого не трогает). Для хуков домена заодно убирает их регистрации из `.claude/settings.json` проекта — только записи, указывающие на снятые файлы хуков; остальные ключи и чужие хуки остаются, `settings.local.json` не трогается, `--dry-run` показывает `would unregister` |
| `as-skill status [--project PATH]` | список записей lockfile: домен/вид/имя, режим, путь, здоровье (OK/MISSING/BROKEN) |
| `as-skill doctor [--project PATH] [--harness-root PATH]` | то же, что `status`, плюс проверки: источник symlink'а пропал, у copy-записи разошёлся хэш (DRIFTED), на диске есть неотслеживаемое (UNTRACKED); печатает только проблемные записи и итог (`N checks, M/N ok, K failed`); ненулевой exit при любой находке — годится как CI gate |
| `as-skill check` | валидирует `domains/*/manifest.yaml` и `skills/*/SKILL.md` в этой копии репозитория; без `--project` |
| `as-skill list [domains\|skills]` | показать, что можно установить |

`as-skill` должен запускаться изнутри (или из поддиректории) этой копии
репозитория, либо с явным `--harness-root` — он читает
`domains/*/manifest.yaml` оттуда. `--project` — независимый
путь: проект, который вы оснащаете.

## Флаги

Общие для `install`/`uninstall`, если не указано иное:

| Флаг | Что делает | По умолчанию |
|------|-----------|--------------|
| `--project PATH` | корень целевого проекта | `.` |
| `--harness-root PATH` | эта копия репозитория | автоопределяется вверх от `.` |
| `--copy` | только `install`: копия-снэпшот вместо symlink | выключено (по умолчанию — symlink) |
| `--dry-run` | ничего не пишет, только печатает план | выключено |
| `--force` | только `install`: перезаписать существующее в целевом пути, даже если as-skill его не отслеживает | выключено (по умолчанию — отказ) |

Если `manifest.yaml` домена в `requires_env` называет неустановленную
переменную (сегодня это только `obsidian` → `$OBSIDIAN_VAULT`) — жёсткая
ошибка для явного `domain`/`domains`/`skill` и предупреждение + пропуск для
`all`. Есть и зеркальный `requires_bin` (проверяется через `$PATH`), но
сегодня его не объявляет ни один домен.
