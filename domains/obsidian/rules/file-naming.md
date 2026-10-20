# Note file naming

## Forbidden characters

These characters break file systems (Windows NTFS, some Linux configurations) and/or wikilinks in Obsidian — never use them in file names:

```
: / \ * ? " < > |
```

If a colon or vertical bar seems natural in a title — rephrase it or replace it with a dash.

| Bad | Good |
|-------|--------|
| `SLI: метрика надёжности.md` | `SLI как метрика надёжности.md` |
| `Вопрос: как выбрать SLO?.md` | `Как выбрать SLO.md` |
| `SLO vs SLA \| сравнение.md` | `SLO и SLA – внутреннее против внешнего.md` |

Cyrillic, spaces, dashes, parentheses — allowed.

## Style: claim-based, not topic-based

A note's name should express a **claim/insight**, not a topic. It's the note's key in the graph and often becomes the heading of a future atomic note.

| Bad (topic) | Good (claim) |
|---------------|----------------|
| `Exponential backoff` | `Exponential backoff защищает от каскадных сбоев` |
| `О мониторинге` | `Мониторинг min/max ловит сбои, которые среднее скрывает` |
| `Error budget` | `Error budget как разрешение на риск` |
| `Ретраи` | `Ретраи без backoff превращают деградацию в катастрофу` |

Signs of a good name:

- **Specific** — not "О работе с командой" ("On working with the team"), but "Психологическая безопасность как основа командного доверия" ("Psychological safety as the basis of team trust")
- **Self-contained** — understandable without the context of the source
- **Concise** — 4–8 words
- **In the vault's language** — Russian, with technical terms left in English (`Error budget`, `Cardinality Problem`)

Applied in `obsidian-ingest` (names of atomic notes), `book-highlights-processor` (callout headings → future file names), `obsidian-split-note`, `obsidian-refactor-lecture`.

## Naming patterns

### A series of notes from one source / discipline

`<Parent> – <Concept>.md` (with an en dash `–`, not a hyphen `-`):

```
ВышМат – Транспонированная матрица.md
Теория ОС – Виртуальная память.md
ACID – Изолированность (isolation).md
```

### Standalone atomic note

An autonomous, claim-based title (expresses an idea, not a topic):

```
Error budget как разрешение на риск.md
Ретраи без backoff превращают деградацию в катастрофу.md
```

### Lectures

`Lecture – <Discipline> – <YYYY-MM-DD> – <Topic>.md`

### Literature notes

Source name + author: `LN – <Title> – <Author>.md`, or simply a meaningful title for the quote.

## Checks before creating

1. No forbidden characters
2. No duplicate in the vault — `grep -r` for the intended name and its aliases
3. The name is self-contained (understandable outside the context of the source)
4. Technical terms — English, everything else — Russian (see `content-style.md`)
