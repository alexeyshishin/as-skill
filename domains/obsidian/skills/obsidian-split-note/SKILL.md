---
name: obsidian-split-note
description: Refactor a large Obsidian note — split it into atomic/thematic notes while preserving wikilinks in the original. Use this skill when the user asks to split, divide, or refactor a specific note. The skill creates child notes, inserts [[links]] into the original, and follows the PARA vault structure.
---

# obsidian-split-note

Splits one large note into several smaller ones (atomic or thematic) while preserving connectivity:

- the original stays, its sections are replaced with a short description + `[[wikilink]]`;
- each new note is created in the correct PARA folder;
- the frontmatter of the original and the new notes is kept consistent (`up`/`down` are set).

Base rules (PARA, tags, frontmatter, style, file names) — in `.agents/rules/`. In particular: `tags.md` (the "Structural tags" section), `note-types-frontmatter.md`, `knowledge-structures.md` (atomic notes), `file-naming.md`.

---

## Algorithm

### 1. Read and identify candidates

Criteria for "extract / keep" — `knowledge-structures.md` (the "When to split a section into its own note" section). Determine the original's overall topic and the list of candidates.

### 2. Show the plan and wait for confirmation

Plan → confirmation → action protocol — `workflows.md`. Plan format:

```
Original: [[Title]] (topic: …)
Extract:
  1. [[New note 1 name]] → 03. Ресурсы/04. Заметки/ (#thought)
  2. [[New note 2 name]] → 03. Ресурсы/07. Карты/ (#moc)
Leave in the original: intro paragraph, short conclusion
```

### 3. Create the child notes

For each one:

1. **File name** — see `file-naming.md`. For a series from one source: `<Parent> – <Concept>.md`. For a standalone one: a substantive claim-based name.
2. **Structural tag + folder** — per `tags.md` (usually `#thought` → `03. Ресурсы/04. Заметки/`).
3. **Frontmatter** — from the matching template in `_Система/1. Шаблоны/` (see `template-usage.md`). `up` gets a wikilink to the original.
4. **Body** — transfer the section's content under the heading `## Суть` (or `## Идея` / `## Определение` depending on type).
5. **Before creating** — run `rg -l "Concept name" .` to avoid duplicates.

### 4. Update the original

Each extracted section is replaced with a short description + link:

```markdown
### Виртуальная память

ОС создаёт иллюзию единого адресного пространства. Подробнее: [[Виртуальная память]]
```

Or inline within the flow of text: `… использует [[Виртуальная память|виртуальную память]] для …`.

In the original's frontmatter, add wikilinks to the new notes in `down`.

### 5. Verify connectivity

- All new notes have `up` pointing to the original (or the correct parent)
- The original's `down` lists all the new notes
- No "dangling" wikilinks (links to nonexistent files)
- No duplicates (grep before creating)

---

## What NOT to do

- Don't delete the original — only modify it.
- Don't split too finely (a 2–3 line section doesn't deserve its own file).
- Don't duplicate existing notes — grep first.
- Don't change the folder structure — only create files in existing folders.
- Don't remove wikilinks from the original — the section's replacement must contain a link.

---

## Checklist

- [ ] All new files are created in the correct folders
- [ ] Every new file's frontmatter is filled in per the template
- [ ] The new file's structural tag matches its folder (`tags.md`)
- [ ] The original is updated: expanded sections are replaced with wikilinks
- [ ] The original's `down` lists all the new notes
- [ ] No duplicates (grep before creating)
- [ ] File names contain no forbidden characters (`file-naming.md`)
