# Note types, templates, and frontmatter

Templates live in `_Система/templates/`.

## Templates by note type

The vault currently has 12 templates — leaner than what used to be documented here. Several old per-type templates were merged or dropped; there's no dedicated template right now for a project index note, an essay, a topic-to-MOC staging note, an ADR, a micro-progress log, or a weekly/monthly/yearly journal entry. If one of those is needed, ask the user rather than inventing frontmatter, or build off the closest existing template by hand (e.g. `03. шаблон идея.md` plus manually added `sources`/`confidence` fields for an ad hoc concept note).

| Template | Purpose |
|--------|-----------|
| `01. ежедневная заметка.md` | Daily journal entry |
| `02. черновик поста.md` | Draft for a blog/channel post |
| `03. шаблон идея.md` | Idea, thought, or concept — one shared template; the tag you set marks maturity (see "Idea vs. thought vs. concept" below) |
| `04. конспект по лекции.md` | Lecture notes from a course |
| `05. обзор конференции.md` | Overview of an attended conference |
| `06. шаблон тезисов по докладу.md` | Takeaways from a single talk |
| `07. созвон или встреча.md` | Meeting/call notes |
| `08. шаблон карты.md` | MOC (Map of Content) |
| `09. шаблон инцидента.md` | Standalone SRE incident/lesson note |
| `10. шаблон человека.md` | Contact profile |
| `11. шаблон тезисов по видео-книге-статье.md` | Takeaways from a book, video, or article — one merged template; set `tags` to whichever fits (`book`/`video`/`article`) |
| `12. шаблон постмортемы.md` | Project postmortem — filed under the project's own `Постмортемы/` subfolder, `up` points to the project's index note |

The template's leading number is just Templater menu ordering and can shift — treat this table, not a hardcoded filename elsewhere, as the source of truth for what a template is currently called.

### Idea vs. thought vs. concept

All three now come from the same `03. шаблон идея.md` file — the old separate "мысль"/"идея"/"концепт" templates are gone. What used to be a template choice is now a tag choice, set by hand after creating the note:

| | Tag | Nature |
|--|-----|--------|
| **Thought** | `thought` | An insight, already formed |
| **Idea** | `fleeting` + `inbox/review` | Raw, not yet thought through — process later |
| **Concept** | `concept` | A model/framework backed by sources — add `confidence` and (if relevant) `up`/`other` links to the source |

**Rule:** when in doubt, tag `fleeting` + `inbox/review` and sort it out later during inbox processing.

## Standard frontmatter fields

| Key | Type | Description | When to use |
|------|-----|----------|--------------------|
| `aliases` | list | Alternative names | Widely used (~70% of notes) |
| `tags` | list | Note tags | Always |
| `status` | string | `Todo`, `WIP`, `Done` | Projects, postmortems |
| `up` | list | Parent notes | Widely used |
| `down` | list | Child notes | Widely used |
| `links` | list | Web links | Always |
| `other` | list | Horizontal connections | Always |
| `confidence` | string | `high`, `medium`, `low` | Widely used on reference/concept notes under `03. Ресурсы/<Тема>/База знаний/` |
| `date of update` | string | Last-edit date, `DD.MM.YYYY` | Widely used on reference/concept notes |
| `source` | string | Where a book/video/article came from (e.g. `Бумажная книга`) | Book/video/article takeaway notes |
| `author` | string | The source's author | Book/video/article takeaway notes |
| `deadline` | string | Deadline as a wikilink `"[[2026-03-05]]"` | Documented, but not currently used anywhere in the vault |

### Frontmatter rules

- Keep all existing keys — don't delete anything already there
- Don't invent new keys without reason
- Don't change the format of existing values (e.g. don't turn a list into a string)
- Only add new fields if they're consistent with other notes of the same type

### Provenance for book/video/article notes

A source's provenance (`source`, `author`) lives directly on its own takeaway note — there's no separate `sources` field duplicated onto every concept note it feeds. On a concept/reference note that draws on a source, link back with `up` (or `other`, if the source isn't the primary parent) to the source's takeaway note; that link is enough on its own.

**History:** an earlier "option B" policy required duplicating a literature source into both `up` and a `sources` list field on every note it touched (see `00. Входящие/Литературные источники в up – четыре варианта политики хранилища`). As of this doc's last sync with the vault (2026-08-15), `sources:` doesn't appear in a single note — the policy isn't in current practice. Don't reintroduce it without checking with the user first.

### Frontmatter examples

**Project:**

```yaml
---
tags:
  - project
status: WIP
up: "[[01. Проекты]]"
aliases: []
down: []
links: []
other: []
---
```

**Daily note:**

```yaml
---
tags:
  - journal/daily
date: "{{date:DD-MM-YYYY}}"
---
### Задачи на день:
- [ ]
### Дополнительно:
-
### Привычки
-
### Оценка дня:
-
```

**Concept/reference note:**

```yaml
---
date of update: 27.12.2025
tags:
  - devops
confidence: medium
up: "[[Родительская тема]]"
aliases: []
down: []
links: []
other: []
---
```

## Confidence levels

Used on reference/concept notes and syntheses.

| Level | Meaning |
|---------|----------|
| `high` | Well-studied idea, confirmed by several sources, validated in practice |
| `medium` | Backed by sources, but few examples or a single source |
| `low` | A single mention, anecdotal or speculative — needs verification |

### Confidence rules

- Required for note types: concept/reference notes (tag `concept` or a domain tag under `03. Ресурсы`), literature/source takeaway notes, syntheses
- When created from a single source — default to `medium`
- If the idea is theoretical or you're unsure — set `low` and note it in the text
- When updated with new sources — raise the level
- Don't set `high` without at least two independent sources or personal experience
