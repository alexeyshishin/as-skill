# Workflows

## The "plan → confirmation → action" protocol

Applies to all skills that create/move/delete files in batches: `obsidian-ingest`, `obsidian-split-note`, `obsidian-refactor-lecture`, `obsidian-refactor-inbox`, `obsidian-note-critic` (when it creates a review note).

1. **Plan.** Before any batch of changes, show a structured list: which files to create, which to update, where to move them, what to delete. Keep it brief (1 line per item), with target paths and tags.
2. **Confirmation.** Wait for the user's confirmation. Exception: the user explicitly said "just do it," "no confirmation needed," "go ahead," or the skill is processing a single file whose outcome is obvious.
3. **Action.** Execute the plan in order: first create new files, then update existing ones, then move, then delete/archive.
4. **Report.** A brief summary: what was created, what was updated, what's still an open question.

If the plan changes mid-execution (an existing note was found instead of needing a new one, a case doesn't fit the template) — stop, update the plan, ask.



## Note refactoring

### Light refactoring (no explicit request needed)

- Fixing typos and grammar
- Adding missing headings for structure
- Regrouping paragraphs by meaning
- Converting a "wall of text" into lists, where appropriate
- Adding missing frontmatter per the note type's template

### Medium refactoring (propose it, execute upon confirmation)

- Splitting a large note into several atomic ones
- Creating a MOC for a topic that's grown
- Moving a note between PARA folders
- Adding Bases or Dataview queries
- Renaming a note (requires updating backlinks)

### Heavy refactoring (only on explicit request)

- Bulk tag renaming
- Changing the frontmatter structure across the whole vault
- Deleting notes or large blocks of text
- Changing Bases or Dataview queries

### Structuring a "messy" note

1. **Organize** — add headings, pull out lists
2. **Summarize** — add a `## Сводка` or `## Итоги` section at the top
3. Pull out **questions** into `## Вопросы`
4. Pull out **action items** into `## Действия`
5. Suggest **splitting out** atomic notes for reusable ideas
6. **Link** it to projects and MOCs via wikilinks
7. Identify **contradictions** with existing knowledge and note them explicitly at the end, linking via wikilinks

---

## Processing inbox notes

1. Notes in `00. Входящие/` — unprocessed inbox
2. Determine the note type (thought, project, resource, task)
3. Add frontmatter per the corresponding template
4. Move it to the correct PARA folder
5. Link it to existing notes and MOCs
6. Remove the `#inbox/review` tag once processed

---

## Ingesting an external source

When the user asks to process an external source (article, book, video, transcript):

1. **Read the source** in full
2. **Create a literature note** in `03. Ресурсы/03. Литературные заметки/` using the `Шаблон литературной цитаты.md` template
3. **Fill in the `sources` field** — the source's title and link
4. **Extract concepts** — for each idea, check whether a note already exists. If it does — update it. If not — create an atomic note in `03. Ресурсы/04. Заметки/`
5. **Update existing concepts** — search `03. Ресурсы/04. Заметки/` and `03. Ресурсы/07. Карты/` for notes touching the same topic. Add the new source to `sources`, refine the wording, raise `confidence` if the idea is confirmed, add a `> [!warning]` if it contradicts. Goal: touch 5–10 existing notes
6. **Add links** in both directions: from the literature note → to the concepts, from the concepts → `up` to the literature note
7. **Set `confidence`** — `medium` if there's a single source, `high` if the idea is confirmed by other notes
8. **Update the MOC** — if the concept belongs to an existing map, add a link there
9. **Record contradictions** — if the new information contradicts existing notes: `> [!warning] Противоречие: эта идея расходится с [[Другая заметка]]`

---

## Creating or updating a MOC

1. Gather all notes on the topic (search by tags, links, folders)
2. Create the MOC using `Шаблон карты.md`
3. Add brief annotations to each link
4. Add a Bases query for automatic updates
5. Link the MOC to its parent and child topics

---

## Structuring a project or life area

1. Make sure there's a central note with goals, status, and links
2. Group supporting notes together (resources, meetings, ideas)
3. Link everything through the central note
4. Use the `up`/`down`/`other` frontmatter fields for navigation

---

## Saving a response to the vault (Query → File-back)

If a conversation produced an in-depth analysis, comparison, or insight — suggest saving it as a note:

- **Atomic thought** → `03. Ресурсы/04. Заметки/` (a concept or a single insight)
- **Synthesis** → `03. Ресурсы/07. Карты/` (a comparison, a conclusion drawn from several sources)

Marker: if the user copied the response into `00. Входящие/` — process it via the standard ingest flow.
After saving — add an entry to `_Система/wiki-log.md`.

---

## Knowledge base check (Lint / Health check)

Runs on explicit request: "check the vault," "lint," "health check."

1. **Orphan notes** — look for notes with no incoming links (backlinks = 0). Suggest adding them to a suitable MOC
2. **Stale notes** — notes with status `WIP` and a `deadline` in the past. Suggest updating the status
3. **Unfinished sections** — notes with a `## TODO` or empty sections
4. **Low confidence** — notes with `confidence: low` that could be strengthened with more links
5. **Duplicates** — notes with similar titles or content. Suggest merging them
6. **Contradictions** — concepts that assert opposite things across different notes. Suggest creating a synthesis note
7. **Report** — briefly: what was found, what was fixed automatically, what needs the user's decision
