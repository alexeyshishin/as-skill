<p align="center">
  <img src="assets/banner.svg" width="100%" alt="as-skill — Skills, agents &amp; hooks for AI coding agents" />
</p>

# as-skill

Репозиторий шаблонов агентов, скиллов, правил и хуков для AI-агентов для кода — Claude Code, Codex, Cursor, OpenCode, — разложенных по доменам. Платформа выбирается флагом `-a` (по умолчанию автоопределение, иначе `claude`), см. [Платформы](#платформы).

## Оглавление

- [Домены](#домены)
- [Структура репозитория](#структура-репозитория)
- [Хуки](#хуки)
- [Установка (CLI `as-skill`)](#установка-cli-as-skill)
- [Платформы](#платформы)
- [Контрибьютинг и безопасность](#контрибьютинг-и-безопасность)
- [Лицензия](#лицензия)

## Домены

| Домен | Требования | Описание |
|-------|---------|-----------|
| **[`obsidian`](domains/obsidian/)** | env `OBSIDIAN_VAULT` | PKB на Obsidian по PARA — приём источников в базу, разбор inbox, обогащение фронтматтера, критика заметок, разбиение больших заметок и лекций, распутывание хабов-MOC. |
| **[`git`](domains/git/)** | — | Git workflow — Conventional Commits (только subject, без AI-атрибуции), описания PR, semver-релизы с тегом и GitHub release. |
| **[`content`](domains/content/)** | — | Контент — Telegram-посты, статьи, техническая документация, контент-завод, очеловечивание русскоязычного текста. |
| **[`code`](domains/code/)** | — | Цикл разработки: план → билд → ревью → дебаг. Портативный мини-харнесс — ставится в любой другой проект. |
| **[`core`](domains/core/)** | — | Скиллы, не завязанные на конкретную область: ультра-сжатый режим общения (`caveman`), project memory bank, дефрагментация memory bank, фактчекинг-поиск в вебе (`advanced-search`). |
| **[`devops`](domains/devops/)** | — | Kubernetes, Helm, GitLab CI, ArgoCD — Consilium-агенты (архитектура, security, SRE, диагностика; read-only, советуют) и Executing-агенты (CI, Helm, K8s; пишут пайплайны, чарты, манифесты). |

`obsidian` ставится только при заданном `OBSIDIAN_VAULT` — путь до корня хранилища (`export OBSIDIAN_VAULT="$HOME/path/to/vault"`, например в `~/.zshrc`). Без неё `install all` пропускает домен с предупреждением, явный `install domain obsidian` падает с ошибкой. Остальные домены ставятся всегда.

## Структура репозитория

```
as-skill/
├── domains/                 # Single Source of Truth
│   ├── obsidian/
│   │   ├── manifest.yaml    # targets, requires_env
│   │   ├── rules/
│   │   ├── skills/<имя>/{SKILL.md, README.md}
│   │   └── agents/
│   ├── git/     {manifest, rules, skills, agents}
│   ├── content/ {manifest, rules, skills, agents}
│   ├── code/    {manifest, skills, agents, hooks}
│   ├── core/    {manifest, skills, hooks}
│   └── devops/  {manifest, agents}
│
├── tools/
│   ├── main.go             # Точка входа CLI as-skill, см. tools/README.md
│   ├── internal/           # Диспетчер команд и логика инсталлятора
│   └── README.md
│
├── install.sh              # Собирает as-skill и кладёт в PATH текущей оболочки (source install.sh)
├── CLAUDE.md               # Точка входа для AI-агента
├── CONTRIBUTING.md         # Как править репозиторий: добавление доменов/скиллов
├── SECURITY.md             # Модель угроз, что делает инсталлятор с данными
├── LICENSE
└── README.md
```

## Хуки

Хуки лежат в `domains/<домен>/hooks/` и копируются инсталлятором в `.claude/hooks/` проекта (для `-a codex|cursor` — в `.codex/hooks/` и `.cursor/hooks/`, регистрация через `as-skill hooks enable`; для opencode хуков нет); в `.claude/settings.json` их регистрирует скилл `/code-init` (каждый хук сам объявляет событие и matcher в заголовке `# register:`). При `as-skill remove` регистрации снимаются автоматически.

| Хук | Домен | Событие | Что делает |
|-----|-------|---------|-----------|
| `test-gate` | `code` | Stop | Не даёт завершить задачу, если правились файлы кода, а тесты за сессию не запускались (fail-open, максимум 2 блокировки на коммит). |
| `prettier-format` | `code` | PostToolUse | Запускает prettier, когда условие test-gate выполнено; без конфига prettier — no-op. |
| `memory-bank-load` | `core` | SessionStart | Подгружает `.memory-bank/index.md` (до 16 КБ) и список остальных файлов Memory Bank в контекст. |
| `memory-bank-logger` | `core` | SessionEnd | Фоновый `claude -p` читает транскрипт сессии и обновляет `.memory-bank/` через скилл `memory-bank` (короткие сессии пропускает). |

Подробнее — `domains/code/hooks/README.md` и `domains/core/hooks/README.md`.

## Установка (CLI `as-skill`)

Из корня репозитория (нужен Go ≥ 1.25, на macOS — `brew install go`; первая сборка один раз качает зависимости из Go module proxy, нужна сеть):

```bash
source install.sh   # соберёт as-skill и добавит в PATH (текущий и новые терминалы)
```

Без скрипта: `go build -o as-skill ./tools`.

Два режима установки в каталоги выбранной платформы целевого проекта (по умолчанию `.claude/`, остальные — в разделе [Платформы](#платформы)):

- **symlink** (по умолчанию) — правки в `domains/` видны в проекте сразу, без переустановки. Так живёт корневой `.claude/` этого репозитория — см. [`CONTRIBUTING.md`](CONTRIBUTING.md).
- **`--copy`** — статический снэпшот, не зависит от этой копии репозитория. Так ставится `code` в чужой проект.

```bash
as-skill add all -y                                       # все домены, symlink
as-skill add code core -p <path> --copy -y                # code в чужой проект
as-skill add git core -p <path> -a codex,cursor -y        # другие платформы
as-skill list                                             # что можно поставить; установленное помечено
as-skill doctor -p <path>                                 # что стоит и в каком состоянии
as-skill remove code -p <path> -y                         # снять
```

В терминале `as-skill add` без аргументов открывает диалог: проект → домены → платформы → режим → сводка и подтверждение (`remove` без аргументов — выбор из установленного):

Подсказки диалога — на английском. Иллюстрация (не дословный вывод):

```text
$ as-skill add
Project directory          (по умолчанию — абсолютный cwd)
Domains to add             (ничего не предвыбрано, нужен хотя бы один)
  [x] code — … [installed]
  [ ] obsidian — … (needs $OBSIDIAN_VAULT)
Platforms                  (claude, codex, cursor, opencode)
  [x] claude (detected)  [ ] codex  [ ] cursor  [ ] opencode
Install mode               (symlink — … | copy — …)
harness: …  project: …  domains: …  platforms: … (detected from …)  mode: symlink
Install? (Yes)
```

`(detected)` стоит только у найденных платформ; если ничего не найдено, `claude` предвыбран как значение по умолчанию, без пометки. Установленные домены помечены `[installed]`. Сводка: `harness:`, `warning:` (если checkout в cwd ≠ checkout бинаря), `project:`, `domains:`/`skills:`, `platforms: … (detected from X | default | -a)`, предупреждения о пробелах возможностей платформ и дублях скиллов, `mode:`. Подтверждение по умолчанию — Yes; отказ или Ctrl-C ничего не пишет, печатает `as-skill: error: aborted` и завершает с кодом 1.

Имена и флаги пропускают свои вопросы (имена — домены, `-a` — платформы, `-p` — проект, `--copy` — режим), `-y` берёт всё по умолчанию без вопросов. Диалог открывается, если stdin и stderr — терминал (`CI` не задан, `TERM` не `dumb`): `as-skill add git | cat` (в pipe только stdout) его всё равно откроет, а `</dev/null`, `2>файл`, CI, `TERM=dumb` — нет, тогда нужны имена и `-y` (или `--dry-run`).

Старые `install|uninstall|status` и `--platform` продолжают работать (скрыты из help).

Все команды и флаги, детали `install.sh` и ограничения Windows/fish (symlink требует Developer Mode или админа, иначе `--copy`) — в [`tools/README.md`](tools/README.md).

## Платформы

`-a claude|codex|cursor|opencode[,…]|all` (по умолчанию автоопределение по существующим каталогам платформ, иначе `claude`). Матрица — вывод `as-skill list platforms`:

| Платформа | skills | agents | rules | hooks | Статус |
|-----------|--------|--------|-------|-------|--------|
| claude | `.claude/skills` (link) | `.claude/agents` (link) | `.claude/rules` (link) | `.claude/hooks` -> `.claude/settings.json` | проверено 2026-10-27 |
| codex | `.agents/skills` (link) | `.codex/agents` (render) | `.agents/rules` + блок в `AGENTS.md` | `.codex/hooks` -> `.codex/hooks.json` | не проверено — по документации вендоров на 2026-10-27, реальный CLI не запускался |
| cursor | `.agents/skills` (link) | `.cursor/agents` (render) | `.cursor/rules` (render, `.mdc`) | `.cursor/hooks` -> `.cursor/hooks.json` | не проверено — по документации вендоров на 2026-10-27, реальный CLI не запускался |
| opencode | `.agents/skills` (link) | `.opencode/agents` (render) | `.agents/rules` + блок в `AGENTS.md` | не поддерживается | не проверено — по документации вендоров на 2026-10-27, реальный CLI не запускался |

- **link vs render.** Скиллы и правила claude — symlink (или копия при `--copy`). Агенты на не-claude платформах генерируются в нативном формате: правки тела подхватываются сразу, правки frontmatter требуют переустановки (`as-skill doctor` покажет STALE).
- **Дубли скиллов.** claude+codex вместе: скиллы видны из двух каталогов. Cursor и opencode читают и `.claude/skills`, и `.agents/skills` — видят скиллы дважды; `doctor` предупреждает DUPLICATE-SKILL.
- **Вызов:** `/code-plan` — Claude и Cursor, `$code-plan` — Codex. В OpenCode скилл подхватывается агентом по описанию; `/`-вызов в его документации не описан.
- **Хуки.** Codex: после `hooks enable` подтвердить хуки через `/hooks` (trust). Cursor: `stop` не умеет блокировать — используется `followup_message`, максимум 2 раза. opencode: хуки не поддерживаются. Каталоги сессий (`.claude/sessions` и аналоги) стоит добавить в `.gitignore`.
- **Гард для link-установки.** Symlink-записи указывают на эту копию репозитория: добавьте в `.gitignore` целевого проекта `/.agents /.codex /.cursor /.opencode` (по строке) или ставьте с `--copy`.
- **Известная проблема:** `doctor` показывает DRIFTED после `--copy` двух и более доменов.

## Контрибьютинг и безопасность

- Как править репозиторий, добавлять домены и скиллы — [`CONTRIBUTING.md`](CONTRIBUTING.md).
- Модель угроз, что делает инсталлятор с данными, как сообщить об уязвимости — [`SECURITY.md`](SECURITY.md).

## Лицензия

MIT — см. [`LICENSE`](LICENSE).
