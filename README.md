# as-skill

Репозиторий, в котором находятся шаблоны агентов, скиллов и правил. За основу взята мультидоменная архитектура:
- **Obsidian**
- **Git**
- **Content**
- **Code**
- **DevOps**

## Оглавление

- [Домены](#домены)
- [Структура репозитория](#структура-репозитория)
- [Установка (CLI `as-skill`)](#установка-cli-as-skill)
- [Разработка самого харнесса](#разработка-самого-харнесса)
- [Контрибьютинг и безопасность](#контрибьютинг-и-безопасность)
- [Лицензия](#лицензия)

## Домены

| Домен | Требования | Описание | Что внутри |
|-------|---------|-----------|-----------|
| **[`obsidian`](domains/obsidian/)** | env `OBSIDIAN_VAULT` | PKB на Obsidian по PARA — приём источников в базу, разбор inbox, обогащение фронтматтера, критика заметок, разбиение больших заметок и лекций, распутывание хабов-MOC. | **8 правил, 8 скиллов, 4 агента.** |
| **[`git`](domains/git/)** | — | Git workflow — Conventional Commits, описания PR. | **1 правило, 2 скилла, 1 агент.** |
| **[`content`](domains/content/)** | — | Контент — Telegram-посты, статьи, техническая документация, контент-завод, очеловечивание русскоязычного текста. | **2 правила, 5 скиллов, 1 агент.** |
| **[`code`](domains/code/)** | — | Цикл разработки: план → билд → ревью → дебаг. Портативный мини-харнесс — ставится в любой другой проект через `as-skill install domains code core --copy`. | **4 скилла, 4 агента, 1 hook** (test-gate). |
| **[`core`](domains/core/)** | — | Скиллы, не завязанные на конкретную область: ультра-сжатый режим общения, project memory bank, дефрагментация memory bank. | **4 скилла.** |
| **[`devops`](domains/devops/)** | — | Kubernetes, Helm, GitLab CI, ArgoCD — Consilium-агенты (архитектура, security, SRE, диагностика; read-only, советуют) и Executing-агенты (CI, Helm, K8s; пишут пайплайны, чарты, манифесты). | **7 агентов**, пока без правил и скиллов. |

Требования проверяются на установке: нет `OBSIDIAN_VAULT` — пропускается домен `obsidian`, остальные домены `requires_env` не имеют и ставятся всегда.

### Как задать `OBSIDIAN_VAULT`

Обычная переменная окружения — путь до корня Obsidian-хранилища:

```bash
export OBSIDIAN_VAULT="$HOME/path/to/vault"
```

Чтобы не выставлять её в каждой новой сессии, добавьте эту строку в
`~/.zshrc` (или `~/.bashrc`) и перезапустите шелл / выполните `source
~/.zshrc`. Проверить, что переменная видна:

```bash
echo "$OBSIDIAN_VAULT"
```

Пусто — `as-skill install domain obsidian` / `install all` пропустят домен
`obsidian` (для `all` — с предупреждением, для явного запроса домена —
жёсткая ошибка), см. [`tools/README.md`](tools/README.md).

## Структура репозитория

```
as-skill/
├── domains/                 ← Single Source of Truth
│   ├── obsidian/
│   │   ├── manifest.yaml    ← targets, requires_env
│   │   ├── rules/
│   │   ├── skills/<имя>/{SKILL.md, README.md}
│   │   └── agents/
│   ├── git/     {manifest, rules, skills, agents}
│   ├── content/ {manifest, rules, skills, agents}
│   ├── code/    {manifest, skills, agents, hooks/}
│   ├── core/    {manifest, skills}    # caveman, memory-bank, memory-bank-defrag, swarm-report
│   └── devops/  {manifest, agents}
│
├── tools/
│   ├── main.go             # Инсталлятор — CLI as-skill, см. tools/README.md
│   ├── internal/ 
│   └── README.md
│
├── install.sh              # Собирает as-skill и кладёт в PATH текущей оболочки (source install.sh)
├── AGENTS.md               # Точка входа для AI-агента
├── CONTRIBUTING.md         # Как править репозиторий: добавление доменов/скиллов
├── SECURITY.md             # Модель угроз, что делает инсталлятор с данными
├── LICENCE                
└── README.md
```

Правится напрямую: `domains/` и `AGENTS.md`.

## Установка (CLI `as-skill`)

Домены и скиллы ставятся в `.claude/` целевого проекта тулом `tools/` (бинарь `as-skill`).

По умолчанию `install` ставит **symlink** — правки в `domains/` видны в проекте сразу, без переустановки (подробнее — в разделе [«Разработка самого харнесса»](#разработка-самого-харнесса)).

`install --copy` вместо этого делает статический снэпшот — для шаринга или для проектов, которые не должны зависеть от жизненного цикла этой копии репозитория. Так, например, ставится домен `code` в чужой проект: `as-skill install domains code core --project <path> --copy`.

### Быстрый старт — из корня репозитория

```bash
source install.sh
```

Соберёт `as-skill`, сразу добавит его в `PATH` текущей оболочки (sh/bash/zsh) и
сохранит PATH для новых терминалов: путь пишется в `~/.config/as-skill/env`, а в
rc-файл (`~/.zshrc` для zsh; `~/.bash_profile` и `~/.bashrc` для bash; `~/.profile`
иначе) дописывается одна guard-строка без machine-specific пути — безопасно для
dotfiles-репозиториев. Уже открытые терминалы подхватят PATH ещё одним
`source install.sh` (или откройте новый — там сработает само).

Если запустить не через
`source` (`./install.sh` или `bash install.sh`) — бинарь всё равно соберётся,
но в PATH родительской оболочки попасть не сможет (так уже работают дочерние
процессы); скрипт сам это увидит и подскажет запустить `source install.sh`.
Для новых терминалов PATH при этом всё равно сохранится.
Нужен установленный `go` (на macOS: `brew install go`).

Ручная сборка без `install.sh`, тоже из корня репозитория:

```bash
go build -o as-skill ./tools
```

### Команды и флаги

Самое частое:

```bash
as-skill install all                        # всё сразу: домены, symlink
as-skill install domains code core --copy   # два домена снэпшотом — так code ставится в чужой проект
as-skill status                             # что уже стоит и в каком состоянии (OK/MISSING/BROKEN)
```

Полный список команд (включая `uninstall`, `doctor`, `check`, `list`) и флагов
(`--project`, `--harness-root`, `--copy`, `--dry-run`,
`--force`) с описанием каждого — в [`tools/README.md`](tools/README.md).

### Поддерживаемые платформы

| Платформа | CLI `as-skill` | `install.sh` | `install` (symlink, по умолчанию) |
|-----------|----------------|---------------|-------------------------------------|
| macOS | ✅ | ✅ | ✅ |
| Linux | ✅ | ✅ | ✅ |
| Windows | ✅ | ⚠️ нужен Git Bash или WSL | ⚠️ см. ниже |

`as-skill` сам по себе — обычный Go-бинарь без platform-specific кода, кросс-платформенный из коробки.

Ограничения на Windows:

- **`install.sh`** — `sh`-скрипт (нужен для `source install.sh` и автодобавления в `PATH`). На Windows запускайте его через Git Bash или WSL, либо соберите бинарь напрямую в PowerShell/cmd:
  ```powershell
  go build -o as-skill.exe ./tools
  ```
- **Symlink-режим по умолчанию** — `install` без `--copy` вызывает `os.Symlink`, а создание symlink на Windows требует привилегию `SeCreateSymbolicLinkPrivilege`, которой у обычного пользователя нет. Сработает только если:
  - включён **Developer Mode** (Windows 10 1703+ / Windows 11 — даёт эту привилегию обычным пользователям), или
  - терминал запущен **от имени администратора**.

  Без этого каждый файл/папка при установке будет падать с ошибкой нехватки прав. Если Developer Mode не включён и повышать права не хочется — ставьте с `--copy`:
  ```powershell
  as-skill install all --copy
  ```
  Копия работает без ограничений, но не подхватывает live-правки в `domains/` — под неё нужно переустанавливать заново после изменений в харнессе.

- **fish** — автосохранение PATH из `install.sh` рассчитано на POSIX rc-файлы (`~/.zshrc`, `~/.bash_profile`/`~/.bashrc`, `~/.profile`) и не трогает `config.fish` (другой синтаксис). Скрипт это определяет по `$SHELL`, пропускает правку rc-файла и печатает подсказку — добавьте PATH вручную: `fish_add_path <путь к чекауту as-skill>`.

## Разработка самого харнесса

Корневой `.claude/` этого репозитория (гитигнорится) — не ручная копия `domains/code/*` + части `domains/git/*`, а результат symlink-установки: правки под `domains/` видны в нём сразу, без переустановки. Пересоздать с нуля:

```bash
rm -rf .claude
./as-skill install domains code git core --project .
```

## Контрибьютинг и безопасность

- Как править репозиторий, добавлять домены и скиллы — [`CONTRIBUTING.md`](CONTRIBUTING.md).
- Модель угроз, что делает инсталлятор с данными, как сообщить об уязвимости — [`SECURITY.md`](SECURITY.md).

## Лицензия

MIT — см. [`LICENCE`](LICENCE).
