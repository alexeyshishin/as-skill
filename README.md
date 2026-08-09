# alexey-shishin-skills

Репозиторий, в котором находятся шаблоны агентов, скиллов и правил. За основу взята мультидоменная архитектура:
- **Git**
- **Obsidian**
- **Code**
- **Content**
- **DevOps**

## Домены

| Домен | требует | Что внутри |
|-------|---------|-----------|
| **[`obsidian`](domains/obsidian/)** | env `OBSIDIAN_VAULT` | PKB на Obsidian по PARA — приём источников в базу, разбор inbox, обогащение фронтматтера, критика заметок, разбиение больших заметок и лекций, распутывание хабов-MOC. **8 правил, 8 скиллов, 4 агента.** |
| **[`git`](domains/git/)** | — | Git workflow — Conventional Commits, описания PR. **1 правило, 2 скилла, 1 агент.** |
| **[`content`](domains/content/)** | — | Контент — Telegram-посты, статьи, техническая документация, контент-завод, очеловечивание русскоязычного текста. **2 правила, 5 скиллов, 1 агент.** |
| **[`code`](domains/code/)** | — | Цикл разработки: план → билд → ревью → дебаг, плюс установка мини-харнесса в целевой проект. **5 скиллов, 4 агента, 1 hook** (test-gate). |
| **[`devops`](domains/devops/)** | — | Kubernetes, Helm, GitLab CI, ArgoCD — Consilium-агенты (архитектура, security, SRE, диагностика; read-only, советуют) и Executing-агенты (CI, Helm, K8s; пишут пайплайны, чарты, манифесты). **7 агентов**, пока без правил и скиллов. |

Требования проверяются на установке: нет `OBSIDIAN_VAULT` — пропускается домен `obsidian`, остальные домены `requires_env` не имеют и ставятся всегда.

## Core-скиллы

Вне доменов, в `core/skills/` — не завязаны на конкретную область, ставятся всегда, без `manifest.yaml`:

| Скилл | Что делает |
|-------|-----------|
| **[`caveman`](core/skills/caveman/)** | Режим ультра-сжатого общения — режет объём ответа, сохраняя техническую точность. Always-on по умолчанию, см. `AGENTS.md`. |
| **[`memory-bank`](core/skills/memory-bank/)** | Ведёт лёгкую «энциклопедию проекта» в `.memory-bank/`: архитектура, стек, соглашения, активные задачи. |
| **[`memory-bank-defrag`](core/skills/memory-bank-defrag/)** | Дефрагментирует и актуализирует memory bank — сворачивает накопившиеся патчи в чистое текущее состояние. |
| **[`swarm-report`](core/skills/swarm-report/)** | Заготовка — содержимое ещё не написано. |

## Структура репозитория

```
claude-harness/
├── domains/                 # ← ЕДИНСТВЕННЫЙ источник правды по доменам
│   ├── obsidian/
│   │   ├── manifest.yaml    ← targets, requires_env
│   │   ├── rules/
│   │   ├── skills/<имя>/{SKILL.md, README.md}
│   │   └── agents/
│   ├── git/     {manifest, rules, skills, agents}
│   ├── content/ {manifest, rules, skills, agents}
│   ├── code/    {manifest, skills, agents, hooks/}
│   └── devops/  {manifest, agents}   ← rules/ и skills/ ещё не заведены
│
├── core/
│   └── skills/               # Скиллы без домена — ставятся всегда
│       ├── caveman/
│       ├── memory-bank/
│       ├── memory-bank-defrag/
│       └── swarm-report/
│
├── tools/
│   ├── cli.go              # Инсталлятор — CLI as-skill, см. tools/README.md
│   └── README.md
│
├── install.sh              # Собирает as-skill и кладёт в PATH текущей оболочки (source install.sh)
├── AGENTS.md               # Точка входа для AI-агента — домены, deploy-таргеты, контракт между доменами
├── LICENCE                 # MIT
└── README.md
```

Правится напрямую: `domains/`, `core/skills/` (для скиллов без домена) и `AGENTS.md`.


## Установка (CLI `as-skill`)

Домены и скиллы ставятся в `.claude/` целевого проекта тулом `tools/cli.go` (бинарь `as-skill`). По умолчанию `install` ставит **symlink** — правки в `domains/`/`core/skills/` видны в проекте сразу, без переустановки (для совместной разработки, см. раздел ниже про сам харнесс). `install --copy` вместо этого делает статический снэпшот — для шаринга или проектов, которые не должны зависеть от жизненного цикла этой копии репозитория (так, например, работает `/code-setup`, который всегда ставит себя в чужой проект через `--copy`).

Быстрый старт — из корня репозитория:

```bash
source install.sh
```

Соберёт `as-skill` и сразу добавит его в `PATH` текущей оболочки (sh/bash/zsh) —
командой можно пользоваться сразу же, без `./`. Если запустить не через
`source` (`./install.sh` или `bash install.sh`) — бинарь всё равно соберётся,
но в PATH родительской оболочки попасть не сможет (так уже работают дочерние
процессы); скрипт сам это увидит и подскажет запустить `source install.sh`.
Нужен установленный `go` (на macOS: `brew install go`).

Ручная сборка без `install.sh`, тоже из корня репозитория:

```bash
go build -o as-skill ./tools
```

Режимы установки:

```
as-skill install domain  <имя>               symlink один домен (по умолчанию)
as-skill install domains <имя> [имя...]      symlink несколько доменов
as-skill install all                          symlink все домены (у кого выполнен requires_env) + core-скиллы
as-skill install skill   <имя>               symlink один скилл, доменный или core
as-skill install ... --copy                   любая из форм выше — копией-снэпшотом вместо symlink
as-skill uninstall domain|domains|all|skill ... снести то, что поставил install
as-skill status / doctor / check              здоровье установки / self-check репозитория
as-skill list [domains|skills]                что вообще можно установить
```

Флаги `--project` (куда ставить, по умолчанию `.`), `--harness-root` (эта копия репозитория, автоопределяется), `--with-core`, `--copy`, `--dry-run`, `--force`, поведение при незаполненном `requires_env`/`requires_bin` (сейчас `requires_env` есть только у `obsidian` → `$OBSIDIAN_VAULT`) — всё в [`tools/README.md`](tools/README.md).

## Разработка самого харнесса

Корневой `.claude/` этого репозитория (гитигнорится) — не ручная копия `domains/code/*` + части `domains/git/*`, а результат symlink-установки: правки под `domains/` видны в нём сразу, без переустановки. Пересоздать с нуля:

```bash
rm -rf .claude
./as-skill install domains code git --project .
```

(`--project .` — установка в корень самого харнесса; без `--copy`, потому что для совместной разработки нужны именно живые symlink'и на `domains/`, а не отдельный снэпшот).

## Лицензия

MIT — см. [`LICENCE`](LICENCE).
