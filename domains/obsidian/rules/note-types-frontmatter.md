# Note types, templates, and frontmatter

Templates live in `_Система/1. Шаблоны/`.

## Templates by note type

| Template | Purpose |
|--------|-----------|
| `Шаблон проекта.md` | Project structure (goals, status, deadline, progress) |
| `Ежедневная заметка.md` | Journal entry with navigation |
| `Еженедельная заметка.md` | Weekly review |
| `Ежегодная заметка.md` | Year-end summary |
| `Шаблон идея.md` | Main template for recording atomic notes and ideas |
| `Шаблон человека.md` | Contact profile |
| `Шаблон карты.md` | MOC (Map of Content) |
| `Шаблон топика.md` | A topic to study |
| `Шаблон эссе.md` | A full-length essay |
| `Шаблон литературной цитаты.md` | Notes on books/articles |
| `Созвон или встреча.md` | Meeting notes |
| `Черновик поста.md` | Post drafts |
| `Обзор конференции.md` | An overview note for an attended conference (overall impression, talks attended, new contacts) |
| `Шаблон тезисов по докладу.md` | A collection of takeaways from a talk |
| `Шаблон тезисов по видео.md` | A collection of takeaways from a watched video |
| `Шаблон тезисов по книге.md` | A collection of takeaways from a book |
| `Шаблон тезисов по статье.md` | A collection of takeaways from an article |
| `Конспект по лекции.md` | Lecture notes from a course |

## Standard frontmatter fields

| Key | Type | Description | When to use |
|------|-----|----------|--------------------|
| `aliases` | list | Alternative names | Always |
| `tags` | list | Note tags | Always |
| `status` | string | `Todo`, `WIP`, `Done` | Projects/Posts |
| `deadline` | string | Deadline as a wikilink `"[[2026-03-05]]"` | Projects |
| `up` | list | Parent notes | Always |
| `down` | list | Child notes | Always |
| `links` | list | Web links | Always |
| `other` | list | Horizontal connections | Always |
| `confidence` | string | `high`, `medium`, `low` | Concepts, literature notes, syntheses |
| `sources` | list | List of source notes/documents | Literature notes, syntheses |

### Frontmatter rules

- Keep all existing keys — don't delete anything already there
- Don't invent new keys without reason
- Don't change the format of existing values (e.g. don't turn a list into a string)
- Only add new fields if they're consistent with other notes of the same type
- **The `sources` field** — never clear it. Only add new sources

### Policy: literature sources in up + sources (option B)

A literature note, book, article, video — is **simultaneously a source and a topical parent** in this vault. The corresponding link must be:

- in `up` — as the parent of the topic (for graph navigation)
- in `sources` — as the source (for "where did I learn this" queries)

For example, for an atomic note from the book "The Phoenix Project":

```yaml
up:
  - "[[Книга – Проект Феникс]]"
  - "[[DevOps]]"
sources:
  - "[[Книга – Проект Феникс]]"
```

**Rules:**

- If a source is in `up` — it must also be duplicated in `sources`. They stay in sync.
- Exception: **navigational MOC indexes** (`Книжная полка`, `Статьи`, `Видео`, `Конференция`, and similar ones tagged `MapOfContent` in the sources folders) — these are parents, **not sources**. Don't duplicate them into `sources`.
- When updating either side (up or sources) — keep them in sync.

**History:** before 2026-05-12, the `sources` field wasn't populated during ingest — the source lived only in `up`. After option B was adopted, a one-time migration was run, and the format was locked in as a rule. See [[00. Входящие/Литературные источники в up – четыре варианта политики хранилища]].

### Frontmatter examples

**Project:**

```yaml
---
aliases: []
tags:
  - project
status: Todo
deadline: "[[2026-04-29]]"
up: []
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
aliases: []
---

**< [[Вчера]] | [[Неделя]] | [[Месяц]] | [[Завтра]] >**
```

**Concept/resource:**

```yaml
---
tags:
  - concept
aliases: []
up: []
links: []
sources: []
confidence: medium
---
```

## Confidence levels

Used in concepts, literature notes, and syntheses.

| Level | Meaning |
|---------|----------|
| `high` | Well-studied idea, confirmed by several sources, validated in practice |
| `medium` | Backed by sources, but few examples or a single source |
| `low` | A single mention, anecdotal or speculative — needs verification |

### Confidence rules

- Required for note types: Concept, Literature note, Synthesis
- When created from a single source — default to `medium`
- If the idea is theoretical or you're unsure — set `low` and note it in the text
- When updated with new sources — raise the level and update `sources`
- Don't set `high` without at least two independent sources or personal experience
