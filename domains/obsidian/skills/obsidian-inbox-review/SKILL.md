---
name: obsidian-inbox-review
description: Analyze and review notes from the inbox (`00. Входящие`) — inspection and reporting only, no file changes. The output is a list of notes with recommendations for category and action. For actually processing them (tags, links, moving) use obsidian-refactor-inbox.
---

# obsidian-inbox-review

Read-only, report only. Files are not modified. The output is a structured list of recommendations.

Vault structure, tags, and frontmatter — in `.agents/rules/` (`vault-struct.md`, `tags.md`, `note-types-frontmatter.md`).

**Triggers:** "go through the inbox", "inbox review", "what's in my inbox", "help with #inbox/review", "review my notes".

---

## Algorithm

### 1. Scan

```bash
# All notes in the inbox, by modification date (oldest first)
ls -t "00. Входящие/" | tac

# Only notes tagged #inbox/review
rg -l "#inbox/review" "00. Входящие/"
```

### 2. Analyze each note (no changes!)

1. Read the content
2. Determine the main topic (one note = one topic)
3. Check links: are there wikilinks, is it mentioned in any MOC
4. Decide on the category
5. Formulate a recommendation

### 3. Categorization

Pick the structural tag and folder per `.agents/rules/tags.md` (the "Structural tags" section). Summary decision table:

| What's in the note | Tag | Folder |
|---------------|-----|-------|
| Atomic thought / concept | `#thought` | `03. Ресурсы/04. Заметки/` |
| Map / navigation | `#moc` | `03. Ресурсы/07. Карты/` |
| Person | `#person` | `02. Сферы/01. Люди/` |
| Book / article / video | `#book` / `#article` / `#video` | `03. Ресурсы/01–05` |
| Literature note / quote | `#literature-note` | `03. Ресурсы/03. Литературные заметки/` |
| Project | `#project` | `01. Проекты/` |
| Meeting / 1:1 | `#meeting` | `02. Сферы/03. Работа/` |
| Conference notes | `#conference` | `02. Сферы/06. Конференции/` |
| Journal / reflection | `#journal/daily` | `05. Дневник/<year>/<month>/` |
| Content for a channel | + `#telegram` or `#content` | `02. Сферы/05. Медийность/` |
| No longer relevant | `#archive` | `04. Архив/` |
| Duplicate | — | merge or flag |

### 4. Prioritization

Process in this order:
1. Notes with an explicit wikilink to active projects
2. Notes with `TODO` / `#inbox/action`
3. Recent ones (last 7 days)
4. Old ones with no links (often archive candidates)

---

## Report format

```
📥 Inbox: N notes (M tagged #inbox/review)

🎯 High priority (linked to active projects):
  • [[Note A]] → #thought, 03. Ресурсы/04. Заметки/
    Linked to [[Project X]]; add an `up` link to the MOC.

  • [[Note B]] → #meeting, 02. Сферы/03. Работа/

⚖️ Medium priority:
  • [[Note C]] → needs more work on the topic, leave in inbox

❓ Needs a decision:
  • [[Note D]] — looks like a duplicate of [[Existing note]]
  • [[Note E]] — topic unclear, ask the user

🗄 Archive candidates:
  • [[Old note]] — no links, no longer relevant

📊 Stats: X to process, Y to archive, Z unclear.
```

After the report, ask which notes to process via `obsidian-refactor-inbox`.

---

## Constraints

- No file changes at this stage
- Don't invent links — if there are no similar notes, say so
- Work incrementally — 10–15 notes per iteration

---

## When NOT to use this

| Situation | Use instead |
|----------|-----------------|
| Process the note right away (tags, links, move it) | `obsidian-refactor-inbox` |
| Only update frontmatter without moving | `obsidian-enrich-note` |
| The note is large and needs to be split | `obsidian-split-note` |
| Create a map for a topic | `obsidian-note-canvas-map` |
