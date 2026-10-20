---
id: template-usage
aliases: []
tags: []
---
# Rules for choosing and using templates

Templates live in `_Система/templates/`. Always create a note from a template, not "from scratch" — this ensures consistent frontmatter.

The vault currently has 12 templates — fewer and more general-purpose than this doc used to assume. Several situations that used to get their own dedicated template now share one; see "Distinguishing similar templates" below for how to pick between them.

---

## How to choose a template

### What exactly are you trying to capture?

| Situation | Template |
|----------|--------|
| Read a book / watched a video / read an article, post, RFC | `11. шаблон тезисов по видео-книге-статье.md` — set `tags` to `book`/`video`/`article` |
| Attended a talk live / watched a recording | `06. шаблон тезисов по докладу.md` |
| Attended a lecture (course, coursework) | `04. конспект по лекции.md` |
| A thought, a raw idea, or a term/model/concept | `03. шаблон идея.md` — the tag you set marks which (see below) |
| A knowledge map for a topic (5+ notes) | `08. шаблон карты.md` |
| A profile of a person / contact | `10. шаблон человека.md` |
| A draft post for a blog/channel | `02. черновик поста.md` |
| Notes from a meeting or call | `07. созвон или встреча.md` |
| A conference (multiple talks) | `05. обзор конференции.md` as the umbrella note, plus one `06. шаблон тезисов по докладу.md` per talk |
| A standalone SRE incident/lesson (not tied to one project) | `09. шаблон инцидента.md` |
| A project postmortem | `12. шаблон постмортемы.md` |
| A daily entry | `01. ежедневная заметка.md` |

No current template for: a project index note, a standalone quote (quotes now live inline in the merged source-takeaways note — see below), a progress review against a goal, an essay, a weekly/monthly/yearly journal summary, an ADR, or a micro-progress log. If one of these comes up, ask the user how to proceed rather than improvising frontmatter from scratch.

---

## Distinguishing similar templates

### Idea vs. thought vs. concept

Same template file (`03. шаблон идея.md`) for all three — the old separate templates are gone. What used to be a template choice is now a tag choice made by hand after creating the note. Full breakdown in `note-types-frontmatter.md`; short version:

| | Thought | Idea | Concept |
|--|---------|------|---------|
| **Tag** | `thought` | `fleeting` + `inbox/review` | `concept` |
| **Nature** | Already-formed personal insight | Raw, not yet thought through | A model / term / framework, backed by sources |

**Rule:** when in doubt, tag `fleeting` + `inbox/review` and sort it out later during inbox processing.

### Source takeaways: book vs. video vs. article

One merged template (`11. шаблон тезисов по видео-книге-статье.md`) covers all three — set `tags` to whichever fits. There's no separate per-quote literature note anymore: quotes go straight into that note's `Цитаты` section, alongside `Основные идеи` / `Мои мысли` / `Применение`.

### Talk vs. Conference

- `06. шаблон тезисов по докладу.md` — one talk, one speaker.
- `05. обзор конференции.md` — the whole conference as an event, linking out to the individual talk notes.

### Incident vs. postmortem

- `09. шаблон инцидента.md` — a standalone SRE lesson/incident (`severity`, `date`), not necessarily tied to one project.
- `12. шаблон постмортемы.md` — a project-specific retrospective (`status`, `up`), filed under that project's own `Постмортемы/` subfolder in `01. Проекты/<Проект>/`.

(This split is inferred from how the two templates and real postmortem notes are structured, not from an explicit rule — flag it to the user if a case doesn't cleanly fit either.)

### Map — no separate "topic" staging template anymore

The old "topic that hasn't become a MOC yet" template doesn't exist — there's just `08. шаблон карты.md`. If a topic hasn't reached 5+ related notes, hold off on creating a map at all rather than reaching for a placeholder.

---

## Rules for filling in templates

### Required when creating any note

1. Fill in `aliases` — at least synonyms or abbreviations of the title
2. Specify `up` — what the note relates to (a MOC, project, area)
3. Set the correct tag from the taxonomy (see `tags.md`)

None of these are pre-filled by the raw template file — add them by hand (or via `obsidian-enrich-note`) right after creating the note.

### The `confidence` field

Required for reference/concept notes under `03. Ресурсы/<Тема>/База знаний/` and source takeaway notes (see `note-types-frontmatter.md`).

- `low` — a single source or a personal opinion without verification
- `medium` — there's a source, but few examples
- `high` — confirmed by two independent sources or personal experience

Don't set `high` without grounds — better `medium` with a note than overstated confidence.

### The `status` field (projects, postmortems, drafts)

| Value | When |
|----------|------|
| `Todo` | Just created, not started yet |
| `WIP` | In progress |
| `Done` | Completed |

---

## Journal notes: when to create them

Only `01. ежедневная заметка.md` currently exists — daily entries, created automatically via Templater. There's no weekly/monthly/yearly summary template right now; if the user wants one, ask before building ad hoc frontmatter for it rather than assuming the old cadence still applies.

Don't create daily entries retroactively if there's nothing left to record — better to skip.

---

## Incident / postmortem

See "Incident vs. postmortem" above for which of the two templates fits. Fill it in **while things are fresh** (within 24–48 hours), while details are still vivid.

- `09. шаблон инцидента.md` — fill `severity` and `date` (the date of the incident, not the note).
- `12. шаблон постмортемы.md` — fill `status`, `up` (the project's index note). `status: Done` only once conclusions and action items are written up.

Link to the project or service via `up`. Move action items to the task tracker, or add them as a checklist in the corresponding project.

---

## What not to create via a template

- Sections within other notes (`##` blocks)
- Placeholder links (`[[Nonexistent note]]`)
- Temporary drafts in `00. Входящие/` — these are created freeform and processed later
