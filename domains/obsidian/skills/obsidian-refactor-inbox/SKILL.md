---
name: obsidian-refactor-inbox
description: Process inbox notes in Obsidian. Use this skill when the user asks to process notes from the `00. Входящие` folder — updating tags, adding aliases, creating links to existing notes (people, MOCs, knowledge), moving them to the right folder per the PARA methodology. The skill determines the note type (atomic thought, resource, person, project, conference notes), adds the corresponding tags and links, and never deletes notes — only archives them with the #archive tag.
---

# obsidian-refactor-inbox

The full processing cycle for a note from `00. Входящие/`: typing → frontmatter → links → moving into PARA.

Base rules (structure, tags, frontmatter, style) — in `.claude/rules/`. This covers only what's specific to inbox processing.

## Related skills

| When | What to use |
|-------|------------------|
| Just want to inspect the inbox first | `obsidian-inbox-review` |
| Enrich frontmatter without moving | `obsidian-enrich-note` |
| Split a large note into parts | `obsidian-split-note` |

---

## Algorithm

### 1. Read the note and determine its type

Structural tag + folder — from the `tags.md` table (the "Structural tags" section). Summary:

| Content type | Tag | Folder |
|-----------------|-----|-------|
| Atomic thought / concept | `#thought` | `03. Ресурсы/04. Заметки/` |
| MOC / overview map | `#moc` | `03. Ресурсы/07. Карты/` |
| Person / contact | `#person` | `02. Сферы/01. Люди/` |
| Literature note | `#literature-note` | `03. Ресурсы/03. Литературные заметки/` |
| Book | `#book` | `03. Ресурсы/01. Книги/` |
| Article | `#article` | `03. Ресурсы/02. Статьи/` |
| Video | `#video` | `03. Ресурсы/05. Видео/` |
| Project | `#project` | `01. Проекты/` |
| Meeting / 1:1 | `#meeting` | `02. Сферы/03. Работа/` |
| Conference / talk notes | `#conference` | `02. Сферы/06. Конференции/` |
| Journal entry | `#journal/daily` | `05. Дневник/<year>/<month>/` |
| No longer relevant | `#archive` | `04. Архив/` |

### 2. Update frontmatter

Templates and fields — `.claude/rules/note-types-frontmatter.md`. The note must have at least:

- `aliases` — synonyms, English variant, abbreviations (see `enrich-note` below)
- `tags` — structural tag + domain (see `tags.md`)
- `up`, `down`, `other`, `links` — links (rules in `content-style.md`)

### 3. Enrich aliases and links

This is a sub-task — proceed as `obsidian-enrich-note` would:

- Find related notes in `03. Ресурсы/07. Карты/` and `03. Ресурсы/04. Заметки/`
- For people — check `02. Сферы/01. Люди/`
- `up` — the parent topic / MOC, `down` — what this note gives rise to, `other` — horizontal links
- `links` — external URLs only

```bash
# Find notes on a topic
rg -l "KEYWORD" "03. Ресурсы/" | head -20
```

### 4. Move into PARA

```bash
mv "00. Входящие/Note.md" "03. Ресурсы/04. Заметки/Note.md"
```

### 5. Remove `#inbox/review`

Remove it from `tags` (if present).

### 6. Archiving

**Never delete notes.** If a note is outdated:

1. Move it to `04. Архив/`
2. Add the `#archive` tag
3. Preserve all links and connections

---

## Result checklist

- [ ] The `#inbox/review` tag is removed
- [ ] A structural tag is added (see `tags.md`)
- [ ] `aliases` are filled in (synonyms / EN version)
- [ ] `up` — wikilinks or justifiably empty
- [ ] `down` — wikilinks or justifiably empty
- [ ] `other` — wikilinks to adjacent topics / people / MOCs
- [ ] `links` contains only external URLs (not wikilinks)
- [ ] The file is moved to the correct PARA folder
- [ ] All wikilinks point to existing files
