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

`03. Ресурсы` is organized by subject/domain, not by note type (see `vault-struct.md`) — so `#book`/`#article`/`#video`/`#thought` don't each get their own dedicated folder anymore. Which subject folder a note lands in is a judgment call: pick the existing `<Тема>` it belongs to, or ask the user if none fits.

| Tag | Folder |
|-----|-------|
| `#project` | `01. Проекты/<Проект>/` |
| `#meeting` | `02. Сферы/02. Работа/` (call, 1:1, sync) |
| `#lecture` | `02. Сферы/03. Образование/02. Конспекты/` |
| `#book` | `03. Ресурсы/Книги/<Источник>/` — one subfolder per book: an `00. Оглавление` overview note plus per-chapter notes |
| `#article` / `#video` | `03. Ресурсы/<Тема>/` — same one-subfolder-per-source pattern as `#book`, filed under whichever subject domain the source belongs to; use `Книги` if nothing narrower fits |
| `#thought` | `03. Ресурсы/<Тема>/База знаний/<Подтема>/` — atomic concept/reference note, filed by subject rather than by note type |
| `#moc` | `02. Сферы/07. Карты/` — vault-wide maps hub (not under `03. Ресурсы` anymore) |

No current folder for `#person` or `#conference` — the vault doesn't have `Люди`/`Конференции` folders right now, even though the `#person` tag and its template still exist. If a note like that comes up, ask the user where it should live instead of assuming a path.

`#literature-note` is retired — there's no more standalone per-quote literature note. Quotes now live inline, in the `Цитаты` section of the merged source note (`#book`/`#article`/`#video`) — see `template-usage.md`.

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
| `#telegram` | Material for the personal Telegram channel |
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
