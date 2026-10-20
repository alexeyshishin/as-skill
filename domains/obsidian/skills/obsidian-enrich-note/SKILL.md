---
name: obsidian-enrich-note
description: Enrich Obsidian note frontmatter — fill in the aliases, up, down, other fields without moving the note. Use this skill when the user asks to update/fill in a note's frontmatter, add aliases or links (up/down/other), or enrich a note with links — especially for notes in the `00. Входящие` folder. The note stays in place; only the frontmatter fields aliases, up, down, other change. Do not use for a full refactor with relocation — that's what obsidian-refactor-inbox is for.
---

# obsidian-enrich-note

A targeted frontmatter update **without moving the file and without changing tags**. Only these change: `aliases`, `up`, `down`, `other`. The note body, tags, `links`, `sources` are left untouched.

Semantics of the `up`/`down`/`other`/`links` fields — in `.claude/rules/content-style.md`. Vault structure — in `.claude/rules/vault-struct.md`.

## When NOT to use this

| Situation | Use instead |
|----------|-----------------|
| Need to change tags and move the note | `obsidian-refactor-inbox` |
| First need to understand what's in the inbox | `obsidian-inbox-review` |
| The note needs to be split | `obsidian-split-note` |

---

## Algorithm

### 1. Read the note

Note down the main topic and the key concepts from the title and body.

### 2. Compose aliases

Think like the user's search query:

- Synonyms of the concept
- English equivalent (if the term is well-established)
- Abbreviations
- Alternative spellings

```yaml
aliases:
  - Закон Гудхарта
  - Goodhart's Law
  - Goodhart Law
```

### 3. Find links

```bash
rg -l "KEYWORD" "03. Ресурсы/" "02. Сферы/" | head -20
```

Where to look:
- `02. Сферы/07. Карты/` — candidates for `up` (MOC)
- `03. Ресурсы/<Тема>/База знаний/` — atomic notes on adjacent topics (`other`)
- If the topic relates to a person, there's no dedicated folder yet — search the whole vault for an existing note about them before adding an `other` link

### 4. Fill in the fields

Rules (from `content-style.md`):

- `up` — the parent topic / MOC this note derives from (from specific to general)
- `down` — what this note gives rise to (from general to specific)
- `other` — horizontal links (adjacent topics, people, MOCs that don't fit under up/down)

An empty field is fine if there's no link. Don't make things up.

### 5. Update frontmatter via Edit

If a field is already filled — **add to it**, don't overwrite it. Wikilinks — in quotes if they contain a colon or start with `[[`:

```yaml
aliases:
  - Синоним 1
  - English Name
up:
  - "[[Родительская тема]]"
down:
  - "[[Дочерняя тема]]"
other:
  - "[[Смежная тема]]"
  - "[[Имя Человека]]"
```

---

## Checklist

- [ ] `aliases` non-empty (at least one)
- [ ] `up` — wikilinks or justifiably empty
- [ ] `down` — wikilinks or justifiably empty
- [ ] `other` — wikilinks (this often has the most links)
- [ ] All wikilinks point to existing files
- [ ] The file stayed in its original folder
- [ ] Tags, body, `links`, `sources` unchanged
