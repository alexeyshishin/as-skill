# Knowledge structures: MOC, synthesis, atomic notes

## MOC (Map of Content)

A MOC is a navigational note that collects links on a given topic.
The main MOC is `README.md` at the root. Topic MOCs live in `03. Ресурсы/07. Карты/`.

### When to create a MOC

- The topic is mentioned in 5+ notes and there's no single entry point
- The user explicitly asks to create a map for a topic
- A new major knowledge area emerges

### MOC structure

````markdown
---
aliases: []
tags: []
up: []
down: []
links: []
other: []
---

## Описание

A brief (2–3 sentence) description of the topic in your own words.

## Ключевые заметки

- [[Note 1]] — brief explanation
- [[Note 2]] — brief explanation

## Связанные темы

- [[Another MOC or topic]]

## Автосбор заметок

```dataview
LIST
FROM [[]] OR #needed-tag
SORT file.mtime DESC
```
````

### MOC rules

- Use `Шаблон карты.md` as the base
- In `up`, put the parent MOC or area (if any)
- In `down`, put the major child subtopics
- Add a Dataview query to auto-collect related notes
- Briefly annotate each link — not just a list, but a list with context

---

## Synthesis notes

A synthesis is a note comparing several concepts/tools/approaches and producing a cross-cutting conclusion.
Stored in `03. Ресурсы/07. Карты/`, tagged `#synthesis`.

### When to create a synthesis

- There are 2+ concepts/tools to compare
- An insight has emerged that spans several notes at once
- The user explicitly asks to "compare," "which is better," "which approach to choose"

### Synthesis note structure

```markdown
---
tags:
  - synthesis
aliases: []
up: []
down: []
links: []
other: []
sources: []
confidence: medium
---

## Сравнение

| Критерий | Вариант А | Вариант Б |
|----------|-----------|-----------|
| ...      | ...       | ...       |

## Анализ

Cross-cutting conclusions — what's shared, where they diverge, and why.

## Рекомендация

When to use which approach, and why.

## Связанные заметки

- [[Note A]] — brief explanation
```

### Synthesis rules

- List all source notes in `sources`
- If the conclusion is based on a single note — it's not a synthesis, it's an atomic thought
- When source notes are updated — check that the synthesis is still accurate
- Every conclusion must reference specific notes

---

## Atomic notes

An atomic note is one key idea, thought, or insight. Use `Шаблон мысли.md`.

### When to split one out

- A long note contains a standalone thought that's useful outside its context
- The same idea repeats across several notes
- A note has grown and contains 3+ unrelated topics

### How to split it out

1. **Identify the atomic idea** — one thought, one insight, one fact
2. **Create a new note** in `00. Входящие/` with a clear title
3. **Frontmatter**: `thought` tag, `up` and `links` fields
4. **Write the essence** — 1–3 paragraphs in your own words (don't copy verbatim from the source note)
5. **Add context** — where the thought came from, what it's connected to
6. **Add links** — in `links`, reference the source note and related ones
7. **In the source note**, replace the expanded block with a link: `[[New atomic note]]`

### Don't

- Don't split out everything — atomicity ≠ splitting for the sake of splitting
- Don't lose context: the new note must be understandable on its own, without opening the source
- Don't delete text from the source note without replacing it with a link
- Don't add links to absolutely every note in frontmatter — it becomes overloaded

---

## When to split a section out into a separate note

Applied in `obsidian-split-note`, `obsidian-refactor-lecture`, `obsidian-ingest` — anywhere atomic notes are extracted from a large note.

### Signs it should be split out

- Describes one concept / theorem / algorithm / insight self-sufficiently
- ≥ 5–7 lines of substantial text (or a formula + explanation)
- Useful in other contexts without the source note
- Not a duplicate of an existing note — `rg -l "<title>"` before creating

### Signs it should stay in the original

- An introductory or connecting paragraph ("this chapter covers…")
- A 2–3 line section with no self-sufficient content
- A list of examples illustrating an adjacent section
- Q&A blocks and answers to lecture questions (unless the answer grew into a full concept ≥ 10 lines)
- If in doubt — leave it in the original; better to under-split than over-split

### Naming and tagging the new file

- Name — per `file-naming.md` (claim-based, no forbidden characters)
- Structural tag and folder — per `tags.md` (the "Structural tags" section)
- In the new note, `up` links to the original; in the original, `down` gets a link to the new note
