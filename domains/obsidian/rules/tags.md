# Tags

Prefer the existing, stable set of tags. Don't multiply new ones unnecessarily.

## Core set

| Tag | Purpose |
|-----|-----------|
| `#project` | Project |
| `#journal/daily` | Daily entry |
| `#journal/weekly` | Weekly entry |
| `#journal/monthly` | Monthly entry |
| `#thought` | Atomic thought |
| `#book` | Book |
| `#inbox/review` | Needs processing / review |
| `#archive` | Note moved to `04. Архив/` |

## Structural tags (by PARA folder)

Used to route a note to the right folder. Choose from this list — don't invent new ones. One structural tag per note.

| Tag | Folder |
|-----|-------|
| `#project` | `01. Проекты/` |
| `#person` | `02. Сферы/01. Люди/` |
| `#meeting` | `02. Сферы/03. Работа/` (call, 1:1, sync) |
| `#conference` | `02. Сферы/06. Конференции/` (talk, meetup) |
| `#book` | `03. Ресурсы/01. Книги/` |
| `#article` | `03. Ресурсы/02. Статьи/` |
| `#literature-note` | `03. Ресурсы/03. Литературные заметки/` |
| `#thought` | `03. Ресурсы/04. Заметки/` (atomic thought / concept) |
| `#video` | `03. Ресурсы/05. Видео/` |
| `#moc` | `03. Ресурсы/07. Карты/` (Map of Content) |
| `#lecture` | lecture notes (alongside `#resource`) |

## Extended taxonomy by domain

**Tech and work:**

| Tag | Purpose |
|-----|-----------|
| `#sre` | Site Reliability Engineering, reliability, incidents |
| `#reliability` | Reliability as an engineering discipline (HA, fault tolerance) — without an explicit tie to SRE practices |
| `#devops` | DevOps practices, CI/CD, infrastructure |
| `#kubernetes` | Kubernetes, k8s, orchestration |
| `#observability` | Monitoring, logging, tracing, Prometheus |
| `#linux` | Linux, kernel, filesystem, networking, utilities |
| `#macos` | macOS, tools, workstation setup |
| `#security` | Information security (general) |
| `#appsec` | Application Security, Secure SDLC, SAST/DAST/Fuzzing |
| `#architecture` | Software architecture, System Design, distributed systems |
| `#database` | Databases, replication, sharding, ACID |
| `#golang` | Go: language, patterns, libraries |
| `#python` | Python: language, tools |
| `#git` | Git: commands, workflow, hooks |
| `#agile` | Agile, Scrum, Kanban, Lean, team metrics |
| `#career` | Career, growth, mentorship |
| `#management` | Team management, processes |
| `#productivity` | Personal productivity, GTD, planning, habits |

**Knowledge, learning, and writing:**

| Tag | Purpose |
|-----|-----------|
| `#concept` | Concept, model, framework |
| `#synthesis` | Synthesis note (comparison, cross-analysis) |
| `#fleeting` | A fleeting thought, needs processing |
| `#zettelkasten` | Luhmann's method, note-based thinking, permanent/literature notes |
| `#writing` | Writing as a thinking tool, plain-language style, editing, essay-writing |
| `#education` | Education, learning, teaching methods, curricula |
| `#llm` | LLMs, AI assistants, agents, prompt engineering |

**Humanities domains:**

| Tag | Purpose |
|-----|-----------|
| `#economics` | Macro- and microeconomics, economic models, taxes, GDP |
| `#finance` | Personal finance, investing, the stock market, portfolio analysis |
| `#psychology` | Psychology, cognitive biases, emotions, motivation |
| `#health` | Health, sleep, sports, medical topics |
| `#history` | Historical facts and event analysis |
| `#philosophy` | Philosophy, worldview concepts, civilizational approaches |
| `#politics` | Politics, government structure, ideologies |

**Media presence and publicity:**

| Tag | Purpose |
|-----|-----------|
| `#content` | Drafts and ideas for publications |
| `#telegram` | Material for the "Мишка на сервере" Telegram channel |
| `#talk` | Talks, conferences, presentations |

**Knowledge and learning:**

| Tag | Purpose |
|-----|-----------|
| `#concept` | Concept, model, framework |
| `#synthesis` | Synthesis note (comparison, cross-analysis) |
| `#fleeting` | A fleeting thought, needs processing |

**Confidence statuses:**

| Tag | Purpose |
|-----|-----------|
| `#confidence/high` | Well verified |
| `#confidence/low` | Needs verification |

## Rules for working with tags

- Put tags in the frontmatter (`tags` field), not inline in the text
- Don't delete or rename tags in bulk without an explicit request
- If a new tag is needed — use a hierarchical structure (e.g. `#journal/daily`, not `#daily`)
- Check whether a suitable tag already exists before creating a new one
