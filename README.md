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
| **[`caveman`](core/skills/caveman/)** | Режим ультра-сжатого общения — режет объём ответа, сохраняя техническую точность. |
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
│   └── cli.go              # Инсталлятор — заготовка, ещё не реализован
│
├── install.sh              # Заготовка
├── AGENTS.md               # Точка входа для AI-агента — заготовка
├── LICENCE                 # MIT
└── README.md
```

Правится только `domains/` (и `core/skills/` для скиллов без домена) — остальное производное или заготовки под будущую автоматизацию установки.

## Лицензия

MIT — см. [`LICENCE`](LICENCE).
