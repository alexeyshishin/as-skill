---
id: template-usage
aliases: []
tags: []
---
# Rules for choosing and using templates

Templates live in `_Система/1. Шаблоны/`. Always create a note from a template, not "from scratch" — this ensures consistent frontmatter.

---

## How to choose a template

### What exactly are you trying to capture?

| Situation | Template |
|----------|--------|
| Read a book | `Шаблон тезисов по книге.md` |
| Watched a video / talk from YouTube | `Шаблон тезисов по видео.md` |
| Read an article / post / RFC | `Шаблон тезисов по статье.md` |
| Attended a talk live / watched a recording | `Шаблон тезисов по докладу.md` |
| Attended a lecture (course, coursework) | `Конспект по лекции.md` |
| A specific quote from a source | `Шаблон литературной цитаты.md` |
| A thought / insight (personal, from your own head) | `Шаблон мысли.md` |
| A raw idea, not yet fully formed | `Шаблон идея.md` |
| A term, model, framework, concept | `Шаблон концепта.md` |
| A project (work, personal, education) | `Шаблон проекта.md` |
| A knowledge map for a topic (5+ notes) | `Шаблон карты.md` |
| A profile of a person / contact | `Шаблон человека.md` |
| A draft post for the Telegram channel | `Черновик поста.md` |
| An essay / in-depth treatment of a topic | `Шаблон эссе.md` |
| Notes from a meeting or call | `Созвон или встреча.md` |
| A conference (multiple talks) | `Обзор конференции.md` |
| A progress review against a goal | `Шаблон селф-ревью по цели.md` |
| An incident / postmortem | `Шаблон инцидента.md` |
| An architectural decision (ADR) | `Шаблон ADR.md` |
| A daily entry | `Ежедневная заметка.md` |
| A weekly summary | `Еженедельная заметка.md` |
| A monthly summary | `Ежемесячная заметка.md` |
| A yearly summary | `Ежегодная заметка.md` |
| A quick entry (habit, metric) | `Шаблон микропрогресса.md` |

---

## Distinguishing similar templates

### Thought vs. Idea vs. Concept

| | `Шаблон мысли.md` | `Шаблон идея.md` | `Шаблон концепта.md` |
|--|-------------------|-----------------|----------------------|
| **Tag** | `thought` | `fleeting` + `inbox/review` | `concept` |
| **Nature** | A personal insight, already formed | Raw, not yet thought through | A model / term / framework |
| **Source** | From your own head | From your own head, "don't lose this" | From a source or practice |
| **What's next** | Link to a MOC or project | Process → turn into a thought or concept | Grow it via `sources` |
| **confidence** | `low` (personal opinion) | not specified | `medium` / `high` |

**Rule:** when in doubt, create a `Шаблон идея.md` tagged `inbox/review`, and sort it out later during inbox processing.

### Source takeaways vs. Literature note

- `Шаблон тезисов по X.md` — a top-level note covering the whole source (book, article, video). One per source.
- `Литературная.md` — a specific quote or idea from the source. There can be several per source.

Connection: from the takeaways note, link to literature notes via `down`; from literature notes, `up` points to the takeaways note.

### Talk vs. Conference

- `Шаблон тезисов по докладу.md` — one talk, one speaker.
- `Обзор конференции.md` — the whole conference as an event. Contains links to individual talk notes via `## Посетил следующие доклады`.

### Map vs. Topic

- `Шаблон карты.md` (`MapOfContent`) — a navigational entry point for a topic, containing **links** to notes.
- `Шаблон топика.md` — a topic being studied that hasn't become a MOC yet. Use it at the start, when there are fewer than five notes.

Transition: once a topic has accumulated 5+ related notes — turn it into a map.

---

## Rules for filling in templates

### Required when creating any note

1. Fill in `aliases` — at least synonyms or abbreviations of the title
2. Specify `up` — what the note relates to (a MOC, project, area)
3. Set the correct tag from the taxonomy (see `tags.md`)

### The `confidence` field

Required for: `Шаблон мысли.md`, `Шаблон концепта.md`, `Литературная.md`, `Шаблон тезисов по книге.md`, and similar templates.

- `low` — a single source or a personal opinion without verification
- `medium` — there's a source, but few examples
- `high` — confirmed by two independent sources or personal experience

Don't set `high` without grounds — better `medium` with a note than overstated confidence.

### The `sources` field

Never clear it. Only add new sources. Format — wikilinks to takeaway notes or external links.

### The `status` field (projects, takeaways, drafts)

| Value | When |
|----------|------|
| `Todo` | Just created, not started yet |
| `WIP` | In progress |
| `Done` | Completed |

---

## Journal notes: when to create them

| Template | When to create |
|--------|----------------|
| `Ежедневная заметка.md` | Every day, automatically via Templater |
| `Еженедельная заметка.md` | At the start of the week or Sunday evening |
| `Ежемесячная заметка.md` | At the start of the month |
| `Ежегодная заметка.md` | At the start of the year, or on December 31 |

Don't create these retroactively if there's nothing left to record — better to skip.

---

## Incident / postmortem

`Шаблон инцидента.md` — for recording incident retrospectives. Fill it in **while things are fresh** (within 24–48 hours of the incident), while details are still vivid.

Required fields:

- `severity` — P1/P2/P3 or Critical/Major/Minor
- `date` — the date of the incident (not the date the note was written)
- `status: Done` — only after the conclusions and action items are written up

Link to the project or service via `up`. Move action items to the task tracker, or add them as a checklist in the corresponding project.

---

## What not to create via a template

- Sections within other notes (`##` blocks)
- Placeholder links (`[[Nonexistent note]]`)
- Temporary drafts in `00. Входящие/` — these are created freeform and processed later
