---
name: obsidian-refactor-lecture
description: "Refactor lecture notes in Obsidian — extract concepts, definitions, and theorems from lecture notes into separate notes, turn the lecture into a \"table of contents\" with links, update links to the discipline MOC. Use this skill when the user asks to process, split, or refactor a lecture's notes or several lectures. Triggers: \"process this lecture\", \"split these notes\", \"refactor the lecture\", \"pull the concepts out of this lecture\", \"process these notes\", \"split up the lecture\", \"the lecture is too big\". The skill handles up to 3 sets of notes per run."
---

## Overview

The skill turns a large set of lecture notes into a structured knowledge graph:

- the **original lecture** becomes a "table of contents" — sections are replaced with a short summary + `[[wikilink]]`;
- **extracted concepts** are created as separate notes named `<Discipline> – <Concept>.md`;
- if a discipline MOC exists, it's updated with links to the new notes (no such MOC is guaranteed to exist right now — see "Lecture notes format" below).

**Limit:** no more than **3 sets of notes** per run. If the user names more, ask which three to process now — the rest go in a follow-up run.

---

## Base rules

Vault structure, tag taxonomy, frontmatter, file names — in `.claude/rules/`:
`vault-struct.md`, `tags.md` (the "Structural tags" section), `note-types-frontmatter.md`, `file-naming.md`.

---

## Lecture notes format

A lecture file follows the pattern `<NN>. <Discipline> лекция <YYYY-MM-DD>.md`, filed under `02. Сферы/03. Образование/02. Конспекты/`, and contains:

```yaml
---
tags:
  - university   # real lecture notes use this, not #lecture — check what's already on sibling notes before picking
up: "[[02. Сферы]]"   # observed real notes point straight at 02. Сферы, not at a per-discipline MOC
aliases: []
down: []
links: []
other: []
---
```

There's no confirmed per-discipline MOC or `discipline:` field in the current vault — only one real lecture note was available to check, and it links `up` straight to `02. Сферы`. If a `## Discipline` MOC turns out to exist (check `02. Сферы/07. Карты/` and `02. Сферы/03. Образование/`), link to it instead and update this file; otherwise ask the user before inventing one.

---

## Processing algorithm

### Step 1. Read and identify candidates

Criteria for "extract / keep" — `knowledge-structures.md` (the "When to split a section into its own note" section). One thing specific to lectures: leave **Q&A blocks** in the lecture as-is; extract only if the answer is a full-blown concept ≥10 lines long.

### Step 2. Draft a plan and show it to the user

Plan → confirmation → action protocol — `workflows.md`. Format:

```
Lecture: [[01. ВышМат лекция 2025-11-20]]
Discipline: ВышМат

Extract into 03. Ресурсы/<Тема>/База знаний/ (atomic facts):
  1. [[ВышМат – Транспонированная матрица]] → #thought + #lecture
  2. [[ВышМат – Определитель матрицы]] → #thought + #lecture

Extract into 00. Входящие/ (need further processing):
  3. [[ВышМат – Операции над матрицами]] → summary concept note
  4. [[ВышМат – Обратная матрица]] → needs links to other topics

Leave in the lecture (not atomic / too contextual):
  - Introductory paragraph on matrix applications
```

Ask for confirmation **unless the user said "just do it"** or similar.

### Step 3. Logic for choosing the folder for an extracted note

| Where | Tag | When |
|------|-----|------|
| `03. Ресурсы/<Тема>/База знаний/` | `#thought` + `#lecture` | Self-contained fact: definition, theorem, formula with proof. The note is already "finished". |
| `00. Входящие/` | `#inbox/review` + `#lecture` | Conceptual topic that needs links. Anything that raises doubts. |

**Default to inbox** if it's not obvious.

### Step 4. Create the extracted notes

For each extracted section:

1. **File name:** `<Discipline> – <Concept>.md`
   - The discipline comes from the lecture's title
   - Example: `ВышМат – Транспонированная матрица.md`, `ПиАС ИБ – Многофакторная аутентификация.md`

2. **Check whether such a note already exists** (grep the vault). If it does, don't create a duplicate — only update the links.

3. **Frontmatter of the new note:**

```yaml
---
aliases:
  - Транспонированная матрица      # Concept name
  - Transposed Matrix               # English equivalent (if applicable)
tags:
  - thought                         # for 03. Ресурсы/<Тема>/База знаний/
  # for 00. Входящие/ — add #inbox/review instead
up:
  - "[[01. ВышМат лекция 2025-11-20]]"  # the parent lecture
down: []
links: []
other: []
---
```

4. **Note body:** use the heading `## Определение` / `## Суть` / `## Теорема` depending on the type of content. Transfer the section's content **verbatim**, including LaTeX formulas.

### Step 5. Update the original lecture

1. Each extracted section is **replaced** with a short (1-2 sentence) summary + link:

```markdown
### Транспонированная матрица

Матрица $A^T$, полученная заменой строк на столбцы. Свойства и примеры: [[ВышМат – Транспонированная матрица]]
```

2. In the lecture's frontmatter, update `down` — add wikilinks to all the new notes.

3. If `down` already contains links to nonexistent files (placeholders), replace them with links to the files just created.

### Step 6. Update a discipline MOC, if one exists

Check `02. Сферы/07. Карты/` and `02. Сферы/03. Образование/` for a MOC covering this discipline. If one exists, add links to the new notes there. If not, don't invent one — that's a medium-refactoring call (`workflows.md`), propose it to the user instead of creating it unprompted.

**How to tell whether links need to be added manually:** read the MOC — if it has a Dataview query picking up notes by tag/folder, don't add anything manually. If it's a manual list, add them.

### Step 7. Verify connectivity

- [ ] All new files are created in the correct folders.
- [ ] Every new note's frontmatter is filled in (tags, aliases, up, down).
- [ ] The original is updated: each extracted section → short summary + wikilink.
- [ ] The original's `down` contains wikilinks to all the new notes.
- [ ] No duplicate notes (grep before creating).
- [ ] LaTeX formulas preserved verbatim (not simplified or rewritten).
- [ ] Checked whether a discipline MOC exists and updated it if so.

---

## What NOT to do

- **Don't delete** the original lecture or move it to the archive.
- **Don't rewrite** content during extraction — only transfer it verbatim.
- **Don't change** LaTeX formulas.
- **Don't split too finely** — a 2-3 line section doesn't deserve its own file.
- **Don't process more than 3** sets of notes per run.
- **Don't create duplicates** — always grep-check before creating.

---

## Transformation examples

**Before (in the lecture):**

```markdown
### Транспонированная матрица

Транспонированная матрица — это матрица, полученная из исходной путём замены строк на столбцы...
[100 lines of text + LaTeX]
```

**After (in the lecture):**

```markdown
### Транспонированная матрица

Матрица $A^T$, где строки и столбцы поменяны местами; $(A^T)_{ij} = A_{ji}$. Подробнее: [[ВышМат – Транспонированная матрица]]
```

**New file** `ВышМат – Транспонированная матрица.md` in `03. Ресурсы/<Тема>/База знаний/`:

```markdown
---
aliases:
  - Транспонированная матрица
  - Transposed Matrix
tags:
  - thought
  - lecture
up:
  - "[[01. ВышМат лекция 2025-11-20]]"
down: []
links: []
other: []
---

## Определение

Транспонированная матрица — это матрица, полученная из исходной путём замены строк на столбцы...
[full content from the lecture]
```
