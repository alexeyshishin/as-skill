# as-skill

CLI, который ставит домены и скиллы в каталоги платформы целевого проекта: `--platform claude|codex|cursor|opencode`, по умолчанию `claude` → `.claude/`. Матрица платформ — в [корневом README](../README.md#платформы).

## Сборка

Из корня репозитория:

```bash
go build -o as-skill ./tools     # или: go run ./tools <команда>
source install.sh                # то же + добавляет as-skill в PATH
```

`source install.sh` пишет путь в `~/.config/as-skill/env` и одну guard-строку в rc-файл (zsh, bash; fish — вручную: `fish_add_path <путь>`). Без `source` бинарь соберётся, но в PATH текущей оболочки не попадёт.

Windows: `install.sh` запускать через Git Bash/WSL. Symlink-режим требует Developer Mode или терминал от администратора, иначе — `--copy`.

## Команды

| Команда | Что делает |
|---------|-----------|
| `install domain\|domains\|all\|skill <имя...>` | поставить (symlink по умолчанию, `--copy` — снэпшот) |
| `uninstall domain\|domains\|all\|skill <имя...>` | снять только то, что поставил `install`; чужое не трогает, регистрации хуков снимает |
| `hooks enable\|disable <хук\|all> --platform P` | зарегистрировать/снять хуки в `.claude/settings.json`, `.codex/hooks.json`, `.cursor/hooks.json`; идемпотентно, чужие записи и порядок ключей сохраняет, невалидный JSON отклоняет |
| `status [--platform P]` | записи lockfile: режим, платформы, путь, здоровье |
| `doctor` | `status` + проверки; печатает только проблемы; ненулевой exit при находке (годится для CI) |
| `check` | валидирует манифесты, `SKILL.md`, имена агентов и ссылки в `references/` этой копии репозитория |
| `list [domains\|skills\|platforms]` | что можно поставить; `platforms` — матрица возможностей и дата проверки |

`as-skill` запускается внутри этой копии репозитория или с `--harness-root`. `--project` — проект, который вы оснащаете.

`doctor` находит: BROKEN/MISSING, UNTRACKED, DRIFTED, STALE (источник отрендеренного агента изменился). Предупреждения без влияния на exit code: DUPLICATE-SKILL, UNVERIFIED, HOOKS-TRUST (Codex).

## Флаги

| Флаг | Что делает | По умолчанию |
|------|-----------|--------------|
| `--project PATH` | корень целевого проекта | `.` |
| `--harness-root PATH` | эта копия репозитория | авто, вверх от `.` |
| `--platform P` | `claude\|codex\|cursor\|opencode\|all`, список через запятую или повтором; также для `status`/`doctor`/`hooks` | `claude` |
| `--copy` | `install`: копия вместо symlink | symlink |
| `--dry-run` | только печатает план | выкл. |
| `--force` | `install`: перезаписать неотслеживаемое и обойти gitignore-проверку | отказ |

В link-режиме для каталогов `.agents`, `.codex`, `.cursor`, `.opencode` внутри git-репозитория проверяется `git check-ignore`: не в `.gitignore` — установка отклоняется (добавьте каталог или ставьте `--copy`).

Домен с `requires_env` (сегодня только `obsidian` → `$OBSIDIAN_VAULT`): явный `install` падает с ошибкой, `install all` пропускает с предупреждением.
