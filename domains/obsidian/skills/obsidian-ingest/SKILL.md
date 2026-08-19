---
name: obsidian-ingest
description: >
  Processes an arbitrary markdown file (article, web clip, transcript, notes, draft
  note) and turns it into structured knowledge for an Obsidian base using the PARA
  methodology. The skill extracts atomic entities (concepts, insights, facts), invents
  logical names for notes, creates a source literature note, and builds wikilinks and frontmatter.
  Use this skill whenever the user wants to "add this to the knowledge base", "rework this
  article", "process these notes", "extract the knowledge from this text", "ingest this", "put this in
  the vault", "put this in the base", or brings a markdown file to save into Obsidian.
---

# obsidian-ingest — Knowledge ingest into the Obsidian vault

This skill is the ingest pipeline coordinator. Your job: take a single markdown file, understand
what knowledge is hidden in it, decide how best to represent it in the base, and create the necessary files.
You do the formatting, frontmatter, and link enrichment yourself — that's an integral part of ingest.

Before starting, read the vault rules:
- `AGENTS.md` — general principles for working with the base
- `.claude/rules/note-types-frontmatter.md` — frontmatter templates and fields
- `.claude/rules/knowledge-structures.md` — atomic notes, MOCs, syntheses
- `.claude/rules/content-style.md` — language, style, wikilinks
- `.claude/rules/tags.md` — tag taxonomy

---

## Step 1. Read and understand the source

Read the input file in full. Determine:

- **Content type**: article / web clip, transcript, meeting notes, draft note, arbitrary markdown
- **Main topic**: one phrase describing what this text is about
- **Domain**: which tags it belongs to (sre, kubernetes, observability, career, concept, etc.)
- **Confidence level**: `medium` if there's a single source, `high` if you see the idea corroborated multiple times

---

## Step 2. Extract entities (knowledge extraction)

Types:

- **Concepts and models** — abstract ideas, frameworks, thinking patterns
- **Insights** — non-trivial conclusions not obvious from the title
- **Facts** — specific statements, data, figures
- **Practices** — "how-to"s, processes, techniques

Atomicity criteria — `knowledge-structures.md` (the "When to split a section into its own note" section). Typical result: 3–8 entities from a single source.

---

## Step 3. Come up with note names

Style (claim-based, not topic-based) and forbidden characters — `file-naming.md`.

Before naming — run `rg -l "<keyword>"` across the vault. If a similar note already exists, the plan includes updating the existing one instead of creating a new one.

---

## Step 4. Draft the ingest plan

Plan → confirmation → action protocol — `workflows.md`. Plan format:

```
📄 Source takeaway note: "Source title" → 03. Ресурсы/<Тема>/<Источник>/
   Topic: ...

📝 New atomic notes:
   1. "Note A title" → 03. Ресурсы/<Тема>/База знаний/
      Gist: one sentence
   2. "Note B title" → 03. Ресурсы/<Тема>/База знаний/
      Gist: one sentence

🔄 Update existing notes:
   - [[Existing note]] — link to the source note via up/other, clarify ...
   - [[Another note]] — add a wikilink to the source note

🗺 MOC (if needed):
   - [[Topic MOC]] — add the new notes to the list
```

If you want to update an existing note — read its content before editing it and describe in the plan exactly what you're changing. Don't overwrite it silently.

---

## Step 5. Create the files

After the plan is confirmed, create files in this order:

### 5a. Source takeaway note

Create it in `03. Ресурсы/<Тема>/<Источник>/` (its own subfolder — `00. Оглавление.md` for the overview, plus one file per chapter/section if the source is long) from the `11. шаблон тезисов по видео-книге-статье.md` template — see `template-usage.md`:

```markdown
---
aliases: []
tags:
  - book        # or article / video — whichever fits
source: "Source type or where it came from (e.g. Бумажная книга)"
author: "Author name"
up: []
---

### Основные идеи:
- [[Atomic note A]] — one sentence on what it's about
- [[Atomic note B]] — one sentence on what it's about

### Цитаты:
- "An important quote or excerpt worth keeping verbatim"

### Мои мысли:
- Your own take, not a restatement of the source

### Применение:
- Where this connects to existing projects/topics
```

### 5b. Atomic notes

For each entity from the plan, create a note in `03. Ресурсы/<Тема>/База знаний/`:

```markdown
---
aliases: []
tags:
  - thought
up:
  - "[[Source takeaway note]]"
links: []
confidence: medium
other: []
---

## Gist

1–3 paragraphs in your own words. Don't copy verbatim from the source — rephrase it.
Write so it's understandable a year from now without opening the source.

## Context

Where this idea came from, in what context it arose.

## Related ideas

- [[Another note]] — why it's related
```

### 5c. Update existing notes

For each note in the plan:
- Add a wikilink to the new atomic note or source takeaway note (`up` or `other`, as fits)
- If the information expands or clarifies — add a paragraph with a reference to the source
- If it contradicts — add a callout:
  ```
  > [!warning] Contradiction: this idea conflicts with [[New note]]
  ```

### 5d. Update wiki-log.md

Add an entry to `_Система/wiki-log.md`:
```
- [[YYYY-MM-DD]] Ingest: "Source title" → N notes created, M updated
```

---

## Step 6. Final report

After creating the files, give a short report:

```
✅ Created:
- [[Source takeaway note]]
- [[Note A]]
- [[Note B]]

🔄 Updated:
- [[Existing note]] — link to the source added

⚠️ Decisions worth double-checking:
- "Note C" — I couldn't find anything similar in the base, so I created a new one. You might
  already have something similar under a different name.
- Set confidence to medium — the topic has a single source
```

---

## Rules that must not be broken

- **Don't delete the source file** — the user decides what to do with it
- **Don't rewrite existing notes from scratch** — only add to them
- **Don't create "dead" wikilinks** — links only to notes that actually exist (or were
  just created)
- **Don't multiply tags** — use the existing taxonomy from `.claude/rules/tags.md`
- **Don't add frontmatter fields that aren't in the templates** — maintain consistency
- **Preserve the author's voice**: if the source has a lively, conversational text — don't turn
  the notes into dry documentation

---

## Reference: where to place notes in PARA

| Type | Folder |
|-----|-------|
| Source takeaway note (article, book, video) | `03. Ресурсы/<Тема>/<Источник>/` |
| Atomic thought / concept / insight | `03. Ресурсы/<Тема>/База знаний/` |
| MOC / map / synthesis | `02. Сферы/07. Карты/` |
| If the topic is clearly tied to a project | `01. Проекты/<Проект>/` |
