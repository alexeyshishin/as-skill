---
name: obsidian-untangle-knot
description: "Offload an Obsidian hub note that too many notes link to (an in-degree tangle). The main signal of a tangle is a high in-degree (tens to hundreds of incoming links); out-degree is secondary. The skill creates sub-MOCs under the hub's existing categories and **redirects incoming links from resource notes** to the specific sub-MOC instead of the general one. The original MOC is kept as an entry point — it can still be linked to when the category is unclear. Use this skill whenever the user says: \"untangle this knot\", \"offload MOC X\", \"too many links to X\", \"re-link these notes\", \"X has grown too big\", \"this map has become useless\", \"I don't see structure in this MOC\", \"too many incoming/outgoing links\", \"process this hub note\", \"find my hub notes\", \"link dump\", \"junk-drawer map\". Also apply it when it's clear from context that the user means an overloaded graph node, even if they didn't call it a hub."
---

# obsidian-untangle-knot

## Core idea

In a personal knowledge base, dozens or hundreds of notes end up linking to the same MOCs — that's natural: when you add an article, a book, or a new idea, it's easier to link to the general `[[DevOps]]` than to hunt for the specific sub-map. Over time the general MOC turns from a map into a backlink dump: you open it and drown in noise.

Out-links on a hub grow slowly (it's still a **map**, deliberately maintained). In-links grow spontaneously and fast. So the main signal of a tangle is **high in-degree**, not out-degree.

The skill:

1. Finds/accepts a hub and counts its in/out links
2. Relies on the hub's **already-existing structure** (subheadings, explicit categories) as ready-made landing points — it doesn't invent new categories when none were asked for
3. Creates a sub-MOC for each category
4. **Main action:** goes through incoming links from resource notes and switches them from the general MOC to the appropriate sub-MOC
5. The original MOC **stays** — it remains the right entry point for the "don't know exactly where to link" case

```
before:                       after:
                             [hub] (~ stays, but fewer in-links)
[hub] ←── 200 in-links       ├── [sub-MOC A] ←── 60 in-links (former general ones)
  ├── 22 out-links           ├── [sub-MOC B] ←── 40
                             ├── [sub-MOC C] ←── 30
                             └── [sub-MOC D] ←── 25
                             (general, unclassified ~45 — stay on the hub)
```

Base rules (PARA, tags, templates, style, naming) — in `.agents/rules/`. In particular: `knowledge-structures.md` (MOC), `note-types-frontmatter.md` (frontmatter), `workflows.md` (plan → confirmation → action protocol), `file-naming.md` (names).

**Vault root folder (`<vault>/`):** set at install time (`--vault` or the `BEAR_VAULT` env var). See `.agents/rules/vault-struct.md`.
**Output language:** Russian, technical terms in English.

---

## When to use this and when not to

| Situation | Skill |
|----------|-------|
| **A single note/MOC overloaded with links** | **this skill** |
| The content of a single note needs splitting into atomic notes | `obsidian-split-note` |
| Lecture notes — extract concepts | `obsidian-refactor-lecture` |
| Cross-check a single note for contradictions and gaps | `obsidian-note-critic` |
| Inbox notes — sort and file them | `obsidian-refactor-inbox` |
| Update a single note's frontmatter | `obsidian-enrich-note` |
| Visual callout map of a single note | `obsidian-note-canvas-map` |

Difference from `obsidian-split-note`: that skill splits the **content** of a single note into atoms. Here we split **links** (the graph), not text — the note body is rewritten into a table of contents, and no new atomic notes are created.

---

## Algorithm

### 1. Choosing the hub note

Two modes.

**Direct** — the user already named the note ("offload [[DevOps]]"). Find the file by name:

```bash
fd -t f "DevOps.md" .
```

If there are several matches, ask which one.

**Discovery** — the user wants to find candidates ("find my hub notes", "where am I overloaded"). Scan `03. Ресурсы/07. Карты/` (by default) or another folder on request:

```bash
# Count out-links: number of [[wikilink]] in the body + frontmatter
for f in "03. Ресурсы/07. Карты/"*.md; do
  out=$(rg -o '\[\[[^\]]+\]\]' "$f" | wc -l)
  echo "$out  $f"
done | sort -rn | head -10
```

```bash
# Count in-links: how many times the note is linked to
rg -c -F "[[Note name]]" .
```

Show the top-10 by `in + out` and ask which one to offload. Thresholds — see step 2 below.

### 2. Counting links and diagnosis

**In-links are three distinct streams, not one metric.** This matters a lot: a link from a neighbor's `up:` and a link in an article's body are different phenomena with different handling strategies. Ignoring the distinction means doing the job incompletely.

| Stream | What it is | How to find it | Re-linking |
|-------|---------|------------|--------------|
| **A. Hub's out-neighbors** | Notes that the hub **links to** (from the body or `down`) | Parse the hub's body/frontmatter | Iteration 1 (scaffold) |
| **B. Up-children** | Notes with `up: [[Hub]]` in their frontmatter that are **not mentioned** in the hub's body | Scan the frontmatter of all notes | Iteration 1 (scaffold) — **don't skip this!** |
| **C. In-links in bodies** | Contextual mentions of `[[Hub]]` in the body of other notes | `rg -F "[[Hub]]"` across bodies | Iteration 2 (heavy, optional) |

**Counting:**

```bash
# Stream A: out-degree
# In the hub's body — wikilinks via regex \[\[([^\]\|#]+)
# In the hub's frontmatter — up, down, other, links, sources

# Stream B: up-children — ONLY from the `up:` field, not from other frontmatter fields!
# A crude frontmatter grep also catches unrelated hits: [[Hub]] can appear in other/links/category/sources —
# that's different semantics, and re-linking those is NOT allowed.
# Parse precisely — only the multi-line or single-line up: form
python3 - <<'PY'
import os, re
res = []
for dp, _, fns in os.walk('.'):
    for fn in fns:
        if not fn.endswith('.md'): continue
        p = os.path.join(dp, fn)
        try:
            with open(p, encoding='utf-8') as f: c = f.read(4000)
        except: continue
        m = re.match(r'^---\n(.*?)\n---', c, re.DOTALL)
        if not m: continue
        fm = m.group(1)
        # Multi-line (up:\n  - "[[X]]"\n  - ...)
        block = re.search(r'^up:\s*\n(.*?)(?=^[a-zA-Z_]+:|\Z)', fm, re.M|re.S)
        if block and re.search(r'-\s*"?\[\[Hub\]\]"?', block.group(1)):
            res.append(p); continue
        # Single-line (up: "[[X]]")
        s = re.search(r'^up:\s*(.+)$', fm, re.M)
        if s and '[[Hub]]' in s.group(1):
            res.append(p)
print(len(res))
PY

# Stream C: in-links in bodies (overall count)
rg -l -F "[[Hub]]" --type md . | wc -l
rg -l -F "[[Hub]|" --type md . | wc -l  # with pipe aliases
```

Record:

- **A** = unique out-degree
- **B** = unique up-children
- **C** = unique in-links in bodies
- The overlap A ∩ B (an out-neighbor that **also** has `up: [[Hub]]`) is handled under stream A (it covers both cases)

**Why this matters:** Stream B is the most "forgotten" load. A note has `up: [[Hub]]` but isn't mentioned in the hub's body. If you only offload stream A, these notes are left with a stale categorization.

**Important note on other frontmatter fields:**

`[[Hub]]` can show up in several frontmatter fields — each with its own semantics:

| Field | What it is | Touch it during offload? |
|------|---------|------------------------|
| `up` | Parent in the PARA/MOC hierarchy | **Yes** — this is stream B |
| `other` | Horizontal link, a related topic | **No** — it's a contextual link, not a category |
| `links` | External links or related wikilinks | **No** — navigation, not categorization |
| `category` | User-defined catalog field (books/articles/videos — a "shelf") | **No** — this is categorization by content type, not by topic |
| `sources` | Sources (literature wikilinks or plain text) | **No, never** — a record of sources |
| `down` | Child notes (on the hub) | Part of stream A |

If you mix these up and touch `[[Hub]]` in `other`/`links`/`category`, you'll break the semantics. The skill must **parse the YAML precisely** and only work with the `up` field.

**"Tangle" threshold (by in-degree):**

| in-degree | Diagnosis | Action |
|-----------|-----------|--------|
| ≥ 100 | Strong in-tangle | Untangle it (this skill) |
| 50–100 | Emerging in-tangle | Can be untangled; ask the user |
| 20–50 | Weak tangle | Untangling is possible, but the gain from re-linking is small |
| < 20 | Not a tangle | Probably too early |

**Out-degree** affects whether untangling is feasible at all:

- **N < 5** with high in → nowhere **to** untangle into, there's not enough to build sub-MOCs from. The skill declines and suggests `obsidian-ingest` or `obsidian-note-critic`.
- **N = 5–10** → 2–3 categories can be put together, if they're already outlined by subheadings
- **N > 10** → there's room for proper categorization

High out with low in (out=30, in=5) is an **out-tangle**, which is rare in your base (out grows more slowly). The scenario is different: the hub's body gets rewritten into a table of contents, and sub-MOCs are created for the sake of tidiness. This is the "light branch" of the skill, see the special case below.

### 3. Reading the neighbors (no changes)

For each of the hub's out-neighbors — a quick read and **classification**:

- **Does the file exist?** If the wikilink is dangling, it's not a neighbor but an empty link. Record it in a separate "dangling out-links" list; it does not go into the `up` re-linking.
- **File name and path**
- **Thesis** (one phrase — taken from the first meaningful line of the body or from aliases)
- **Structural tag** and PARA folder
- **`confidence`** (if present)
- **`up`** — the most important field. Its value determines the re-linking branch in step 6.2:
    - `up` is empty → add the sub-MOC
    - `up` contains `[[Hub]]` → replace it with the sub-MOC
    - `up` contains `[[Some – Alternative – sub-MOC]]` that **exists as a file** → **leave it alone**, the note already has a categorization (see below)
    - `up` contains a wikilink that **doesn't exist** (dangling) → pick up the intent: it's an attempted categorization, route it to the new sub-MOC

**Classifying the neighbor:**

| Neighbor type | Action |
|------------|--------|
| Atomic note / concept (`#thought`) | standard re-linking |
| MOC / map (`#moc`, MapOfContent) | standard re-linking; add as `down` on the new sub-MOC |
| **Mention note** (`#person`, a link to a company, a tool) | **not considered a categorizable neighbor** — stays on the hub as a contextual link, doesn't move to the sub-MOC |
| Note from `02. Сферы/05. Медийность/` (Telegram, content) | has its own lifecycle (`status`, `publication_date`, `link`); don't touch `up` |
| Note whose `up` points to an **existing** alternative sub-MOC | already categorized — leave it; add the alternative to the new sub-MOC's `other` as a related map |

Don't read the full body — the thesis and frontmatter are enough for categorization. If the thesis is unclear, look at the first heading.

### 4. Categorization

The main principle: **sub-MOCs are landing points for future in-links.** A category needs to be "semantically stable" enough that it can meaningfully be linked to from a new note.

**If the hub is already structured with subheadings (`##` sections), use them as ready-made categories.** Don't invent axes if the map's author has already done that work. It's enough to turn subsections into sub-MOCs, even if they have just 3–4 notes each. This is gentler and preserves the author's mental model.

**If there are no subheadings**, pick the axis from the table below that best fits the material:

| Axis | When to apply | Example categories |
|-----|----------------|------------------|
| By knowledge domain | When the hub mixes several fields | SRE, Economics, Psychology |
| By level of abstraction | When the hub is a large discipline | Principles, Tools, Experience, Sources |
| By artifact type | When neighbors differ in nature | Concepts, Literature notes, Syntheses, Projects |
| By maturity level | When `confidence` is present | Confirmed, Disputed, In progress |
| By existing tags | When frontmatter already has a stable taxonomy | per `tags.md` |

**Category size:**

- **Minimum 2 notes** if the category is already outlined by a subheading or existing structure — it's "semantically stable", so a sub-MOC is justified.
- **Minimum 3 notes** if the category is invented from scratch — otherwise it's just noise.
- If a category has > 20, it's a tangle in its own right — split it internally (a 2-level structure: `Hub → group → sub-MOC`).

**Category name** — claim-based, per `file-naming.md`. For a family of sub-MOCs from the same parent, use the format `<Parent> – <Category>`: `DevOps – Управление инфраструктурой`, `DevOps – Деплой и релиз`. This preserves kinship in the graph.

**A "Misc" section** in the hub is where the singletons go — the ones that don't fit any category. Don't build a sub-MOC for them.

**Stop signal:** if no axis produces categories, the material is monolithic. There's nothing to untangle. Say so and suggest `obsidian-note-critic`.

### 5. Offload plan → confirmation

**The offload happens in two iterations** — scaffold and in-links. This matters: the actual drop in in-degree only comes in the second one. Don't promise the user miracles after the first.

**Iteration 1 (scaffold, safe):**

- Create the sub-MOCs
- Re-link `up` on the **out-neighbors** (stream A)
- Re-link `up` on the **up-children** (stream B) — can be many times larger than A
- Minimally update the hub's body (pointers, frontmatter `down`)

Both `up` re-linkings are combined — these are frontmatter edits, a symmetric and safe operation. Split into sub-batches only if up-children > 50: batches of 10–20 at a time, by category/folder.

**Iteration 2 (in-links in bodies, optional):**

- Redirect in-links **in bodies** of resource notes (stream C)
- This is heavier: it needs the context of the link to determine the category
- Really applicable only to notes where `[[Hub]]` is the central topic of discussion. For most mentions of `[[Hub]]`, it's a related contextual link, and re-linking it breaks the meaning of the text
- **This iteration may turn out to be barely needed** — if in-degree has already dropped by the expected amount after iteration 1

Plan format:

```
🧶 Hub: [[DevOps]]
   in: 222   out: 22   type: MOC
   dangling out-links: 4 (Deployment Pipeline, Change Management, GitOps, ChatOps)
   neighbors with an alternative up: 2 (Blue-Green, Canary — linked to [[Стратегии развертывания...]])

═══ Iteration 1 (scaffold) ═══

Create sub-MOCs (in 03. Ресурсы/07. Карты/):
  1. [[DevOps – Основы]]                       → 3 out-links
  2. [[DevOps – Управление инфраструктурой]]   → 4
  3. [[DevOps – Деплой и релиз]]               → 1 (Blue-Green/Canary stay on the alt. map)
  4. [[DevOps – Процессы и подходы]]           → 1 (GitOps/ChatOps dangling)

Misc (singletons, stay on the hub): DORA Metrics, Culture, Security, DevSecOps

Re-link `up` on out-neighbors: ~8 notes (after filtering out dangling/alternative/mentions)

Minimally update the [[DevOps]] body:
   — a pointer line `→ Sub-map: [[sub-MOC]]` in each corresponding ## subheading
   — frontmatter `down`: add links to the new sub-MOCs
   — Don't rewrite the whole body: the structure is already there

Expected effect: the original's in-degree will drop only slightly (~5–10),
                  because the links in the bodies remain. The scaffold is in place.

═══ Iteration 2 (in-links, separate run) ═══

Redirect in-links in the bodies of resource notes:
   Total resource in-links: ~160
       — 03. Ресурсы/04. Заметки/        131  ← heavy, split into sub-batches
       — 03. Ресурсы/07. Карты/           17
       — 03. Ресурсы/02. Статьи/           5
       — 03. Ресурсы/01. Книги/            3
       — 03. Ресурсы/03. Литературные/    3
       — 03. Ресурсы/05. Видео/            1
   Not touching (~60): 01. Проекты/, 02. Сферы/, 05. Дневник/, 04. Архив/, 00. Входящие/

Expected effect: the original's in-degree will drop by 50–70% (from the resource mass).
```

Wait for confirmation **on each iteration separately**. After iteration 1, show the report and ask whether to proceed to iteration 2.

Confirmation exceptions — same as in `workflows.md`: "just do it", "no confirmation needed".

### 6. Execution

The order of steps matters, so you don't end up with broken links at intermediate stages:

#### 6.1. Create the sub-MOCs

In `03. Ресурсы/07. Карты/`, from `Шаблон карты.md`:

```yaml
---
aliases: []
tags:
  - moc
up:
  - "[[Original hub]]"
down: []   # will be filled with links to the category's notes
links: []
other: []
---
```

In the body — a short description of the category (1–2 sentences) and a list of links:

```markdown
## Заметки

- [[Note 1]] — what's in it
- [[Note 2]] — what's in it

## Связанные карты

- [[Neighbor MOC]]
- [[Alternative sub-MOC]] — a related map, if a parallel categorization already exists for the topic
```

**If, in step 3, some neighbor was found to already have an existing alternative sub-MOC** (e.g., `Blue-Green` is already linked to `Стратегии развертывания приложений в Kubernetes`), add that sub-MOC right away to the `other` field of our sub-MOC and to the "Связанные карты" section. This respects the author's prior categorization — we don't duplicate it, we reference it.

#### 6.2. Re-link `up` on out-neighbors

For each out-neighbor, determine the branch **based on step 3's result** and act accordingly:

**Branch A — `up` points to the hub itself** (`[[DevOps]]`):

Replace it with `[[sub-MOC]]`. If `up` has several parents, replace **only** the link to the hub, leave the rest.

**Branch B — `up` is empty:**

Add `[[sub-MOC]]` (not the hub) — we set the correct categorization right away.

**Branch C — `up` contains a dangling wikilink** (e.g., `[[DevOps – Практики]]` which doesn't exist as a file):

This is the **author's intent** to categorize — pick it up. Replace the dangling link with the actual sub-MOC (`[[DevOps – Процессы и подходы]]` in our example). This matters: the author already tried to create a category, and we're closing that loop.

**Branch D — `up` contains an existing alternative sub-MOC** (e.g., `[[Стратегии развертывания приложений в Kubernetes]]`, the file exists):

**Don't touch it.** The note is already categorized, it has its "own parent". In the offload plan, this note is listed as **"stays on the alt. map"**, and in our new sub-MOC we reference the related map via `other` and "Связанные карты".

**Branch E — mention neighbors** (`#person`, a link to a company, a tool):

Not considered categorizable. Don't touch `up`, the note stays on the hub as a contextual link.

**Branch F — Telegram/content notes** (`02. Сферы/05. Медийность/Telegram/`):

Don't touch `up` — these notes have their own lifecycle and frontmatter (`status`, `publication_date`, `link`).

**Branch G — the neighbor's `up` already has a more precise parent, and `[[Hub]]` is redundant:**

When `up: [[Other MOC]], [[Hub]]`, and **the other MOC is semantically more precise** for this note (e.g., `Incident Management` has `up: [[Site Reliability Engineering]], [[DevOps]]` — it's about SRE/ITSM, not a DevOps subcategory), the right action is to **remove the link to the hub from up**, not re-link it.

Sign of branch G: none of our sub-MOCs fit **semantically**, and the note already has another parent that's more precise. This is a sign that `[[Hub]]` in `up` was an erroneous double-link — navigational, not categorical.

**Don't confuse this with branch D:** in D, the note is already categorized by an alternative sub-MOC (of the same family as the hub) — there, the alt. parent is a different categorization of the same topic. In G, the note is about **a different topic**, and the hub was simply redundant.

**Bonus fix along the way:** if a note's body has dangling wikilinks left over from an old attempt to categorize the hub (e.g., `[[DevOps – Практики]]` — a nonexistent file), it's worth replacing them with an actual wikilink (`[[DevOps|практиками DevOps]]` via a pipe alias, or a matching sub-MOC). This is incidental graph hygiene, not the skill's main job.

**General nuances:**

- Preserve the list/string format (YAML) as it is in the source note.
- Literature sources stay in `sources` unchanged (the "up + sources in sync" policy from `note-types-frontmatter.md` — sync both sides if you change one).

#### 6.3. Redirect in-links (the main step)

This is the skill's **primary** action. Get the list of all notes that link to the original:

```bash
rg -l -F "[[Hub name]]" --type md . > /tmp/in-links.txt
```

**Folder filter (important):**

Only notes of an **informational/resource nature** are eligible for re-linking:

- ✅ `03. Ресурсы/01. Книги/`
- ✅ `03. Ресурсы/02. Статьи/`
- ✅ `03. Ресурсы/03. Литературные заметки/`
- ✅ `03. Ресурсы/04. Заметки/` (concepts, thoughts)
- ✅ `03. Ресурсы/05. Видео/`
- ✅ `03. Ресурсы/07. Карты/` (other MOCs)

**Don't touch** — links made intentionally as a "general entry point"; re-linking them breaks the historical context:

- ❌ `01. Проекты/` — project notes deliberately link to the general MOC
- ❌ `02. Сферы/03. Работа/` — meetings, 1:1s — contextual links
- ❌ `02. Сферы/06. Конференции/` — talk notes as a point in time
- ❌ `05. Дневник/` — journal, historical context
- ❌ `04. Архив/` — archived material
- ❌ `00. Входящие/` — unprocessed, leave for `obsidian-refactor-inbox`

For each eligible note:

1. Read the context around the link (one or two lines around it)
2. Determine which category the note's material belongs to
3. **If it's determinable** — replace `[[Original]]` with `[[Sub-MOC]]` in the note's body
4. **If it's not determinable, or it's a general context** — leave `[[Original]]` as-is. This is the correct behavior: the general MOC is indeed a "general entry point".
5. If a single note has several links to the original, handle each one separately — they might go to different sub-MOCs.

**Don't touch links inside:**

- `> [!quote]` and `> [!note]` callout blocks — these are historical quotes
- `sources` in frontmatter — a record of sources
- Callout blocks explicitly marked "Review", "Change history", "Context"

**Borderline cases** (category unclear, the note sits on the fence) — leave the original link and list them in the "Under question" section of the report.

#### 6.4. Minimally update the original hub's body

**Don't rewrite the hub's body from scratch.** Structuring by subheadings is the author's mental model, and it's valuable. It's enough to insert links to the sub-MOCs.

**If the hub has subheadings (`## Category`):**

For each subheading we extracted into a sub-MOC, add a compact pointer **as the first line after the heading**:

```markdown
## Управление инфраструктурой

→ Sub-map: [[DevOps – Управление инфраструктурой]]

- [[Infrastructure as Code (IaC)]]
- [[Infrastructure from Code (IfC)]]
    - [[Сравнение и отличия между IaC и IfC]]
- [[Configuration Management]]
```

**Don't touch** the existing out-links under the heading — they stay as convenient navigation. The sub-MOC and the hub show the same information at different levels of detail.

**If the hub has no subheadings** and the body is a flat list:

Group the links by category, add subheadings. This is already a more substantial restructuring of the body, but **preserve the order and wording** where possible.

**In the original's frontmatter:**

- Add wikilinks to the new sub-MOCs in `down` (alongside the existing `down` entries)
- Leave everything else (`up`, `aliases`, `links`, `other`, `tags`, Bases/Dataview) **untouched**

#### 6.5. Fill in `down` on the sub-MOCs

In each sub-MOC's `down`, list all the notes that now have `up: [[Sub-MOC]]`. This can be done manually (as in the plan) — no need to rescan.

### 7. Report

**After iteration 1 (scaffold) — a mandatory caveat:**

```
✅ The offload scaffold for [[DevOps]] is in place

   in (exact [[DevOps]]):  192 → 191  (−1)
   in (via pipe):            30 →  30   (=)

⚠️ Note: the original's in-degree dropped only slightly — that's expected.
   We only re-linked `up` in frontmatter (8 neighbors).
   Links to [[DevOps]] in the bodies of resource notes remain.
   The real effect will come after iteration 2.

Sub-MOCs created: 4
  [[DevOps – Основы]]                       in=3
  [[DevOps – Управление инфраструктурой]]   in=8
  [[DevOps – Деплой и релиз]]               in=5
  [[DevOps – Процессы и подходы]]           in=5

`up` re-linked on out-neighbors: 8 (out of 22 after filtering)

Under question / separate work:
  • Dangling out-links on the hub: [[Deployment Pipeline]], [[Change Management]],
    [[GitOps]], [[ChatOps]] — no notes exist, placeholders remain on the hub.
  • Alternative categorization: [[Blue-Green]], [[Canary Deploy]]
    stayed on [[Стратегии развертывания приложений в Kubernetes]];
    this map was added to the new "Деплой и релиз" sub-MOC's `other`.

Proceed to iteration 2 (redirecting ~160 in-links in bodies)?
```

**After iteration 2 (in-links):**

```
✅ [[DevOps]] offloaded

   in:  191 → 60–80   (real drop)

In-links redirected by sub-batch:
   03. Ресурсы/04. Заметки/    105 of 131
   03. Ресурсы/07. Карты/       13 of 17
   03. Ресурсы/02. Статьи/       3 of 5
   03. Ресурсы/01. Книги/        2 of 3
   03. Ресурсы/03. Лит.заметки/  2 of 3
   03. Ресурсы/05. Видео/        1 of 1

Kept as-is: ~60 (outside the resource area — intentional general links) +
                  ~10 resource notes where the category was unclear (see "Under question")

Under question (10):
  • [[Note X]] — between "Deployment" and "Processes"; left as [[DevOps]].
  ...
```

If anything new turned up along the way (a neighboring note is also a hub, a contradiction between two neighbors, duplicates within a category), mention it in the report, but **don't go there** in the same iteration.

---

## Special cases

### A note isn't a MOC but has accumulated links

An atomic note/concept has spontaneously become a hub (e.g. `Kubernetes` with 30 links). Two options:

1. **Turn it into a MOC** — if the note is essentially already a map. Change `#thought` → `#moc`, update the frontmatter per `Шаблон карты.md`, move it to `03. Ресурсы/07. Карты/`. Backlinks won't break — Obsidian updates them on rename.
2. **Keep it as a concept** — if the content stands on its own and deserves to remain a concept. Offload the links (categorize neighbors into sub-MOCs), but don't rework the note itself.

Ask the user at the moment the hub is chosen.

### No category emerges

If no axis yields ≥3 notes per category, the material isn't a tangle. Say so directly:

> There are a lot of links, but there's no categorical structure among them — the neighbors are all in one semantic field. This isn't a tangle, it's genuinely the center of the topic. There's nothing to untangle.

Suggest `obsidian-note-critic` for a deeper look, or `obsidian-split-note` if the note is large.

### In-links to the hub from project notes

Don't re-link references from `01. Проекты/`, `02. Сферы/`, `05. Дневник/`, `04. Архив/`, `00. Входящие/`. These links were made as a "general entry point"; re-linking them breaks the historical context. Only re-link in-links from resource notes (`03. Ресурсы/`).

### The hub is already partly structured

If the hub has subsections in its body (`## Принципы`, `## Инструменты`), that's the **author's ready-made categorization** — use it, don't invent one from scratch. It's enough to move each subheading out into its own sub-MOC and add a compact pointer at the start of the section.

### Out-tangle (low in, high out)

Rare in this base (out usually grows more slowly), but possible — e.g., a freshly assembled large map that was just put together. Scenario:

- Re-linking in-links isn't needed (there are few of them)
- The main action is rewriting the hub's body into a table of contents by sub-MOC
- Sub-MOCs are created for the sake of graph tidiness, not to redirect a flow of links

If you land on an out-tangle, ask the user what effect they want (navigation vs. graph-view load). The action is chosen based on that answer.

### Concept-hub (an atomic note with a large in-count)

An atomic note/concept (like `Kubernetes`) has accumulated in-links. Two options:

1. **Turn it into a MOC** — if the content already reads more like a map. Change `#thought` → `#moc`, move it to `03. Ресурсы/07. Карты/`. Obsidian will update backlinks on rename.
2. **Keep it as a concept + create a MOC alongside it** — if the content stands on its own and is valuable as a concept. Create `Kubernetes – MOC.md`, and redirect the in-links there; the concept stays a narrow "definition/idea" object.

Ask the user at the moment the hub is chosen.

### Bidirectional links (a note in both `up` and `down` of the original)

A symptom of confusion. Show them explicitly: "in both up and down at once: N notes". Ask which direction to keep. By default: the more general one stays in `up`, the more specific one in `down`. There should be no bidirectional link.

---

## What NOT to do

- **Don't delete the original** — it stays as the root MOC, the top level of the table of contents and the general entry point.
- **Don't rewrite the hub's body from scratch** if it's already structured with subheadings — add a pointer to the sub-MOC, don't break the author's mental model.
- **Don't re-link references outside `03. Ресурсы/`** — projects, journal, conferences, archive, inbox are left untouched; their links to the general MOC are intentional.
- **Don't ignore a neighbor's existing `up`.** If it points to a dangling link, pick up the author's intent. If it points to an existing alternative sub-MOC, respect it and leave it alone — add that map to the new sub-MOC's `other` as a related map.
- **Don't treat mentions of people and companies** (`#person`, `[[Some Company]]`) as categorizable neighbors — they stay on the hub as contextual links.
- **Don't touch Telegram/content notes** in `02. Сферы/05. Медийность/` — they have their own lifecycle (`status`, `publication_date`, `link`).
- **Don't try to re-link a dangling out-link on the hub** — there's no note, the placeholder stays on the hub until it's manually created via `obsidian-ingest`.
- **Don't touch links inside** `> [!quote]`, `> [!note]`, `> [!warning]`, and other callout blocks — that's historical context or quotes.
- **Don't touch `sources`** when re-linking `up` — it's a record of sources, kept in sync with `up` per the policy in `note-types-frontmatter.md`.
- **Don't move neighboring notes between PARA folders** — we're re-linking connections, not migrating structure.
- **Don't do recursive offloading** — one hub per run.
- **Don't merge the scaffold and in-links into one iteration** — after the scaffold, let the user see the result and decide whether to go into the heavy part.
- **Don't clear `links` and `other`** on neighbors — only `up` is subject to replacement.
- **Don't invent categories** when the hub already has ready-made subheadings — use what's there.
- **Don't promise a big drop in in-degree after iteration 1** — that's the scaffold; the real effect comes in the second iteration.

---

## Checklist before finishing

**Sub-MOCs:**

- [ ] Created in `03. Ресурсы/07. Карты/` with `#moc` and frontmatter per `Шаблон карты.md`
- [ ] `up` points to the original hub
- [ ] `down` is filled in (manually as a list, or via Dataview)
- [ ] Names contain no forbidden characters (`file-naming.md`); format `<Parent> – <Category>`

**Out-neighbors (notes the hub links to):**

- [ ] `up` re-linked from the original to the matching sub-MOC (where applicable per the structural tag)
- [ ] Literature sources are in sync in `up` and `sources` (policy B from `note-types-frontmatter.md`)

**In-links (the main part):**

- [ ] Went through the resource in-links (only `03. Ресурсы/`)
- [ ] Where the category was clear — replaced `[[hub]]` with `[[sub-MOC]]`
- [ ] Where the category was unclear — left as-is, listed under "Under question"
- [ ] Didn't touch `01. Проекты/`, `02. Сферы/`, `05. Дневник/`, `04. Архив/`, `00. Входящие/`
- [ ] Didn't touch callout blocks or `sources`

**Original hub:**

- [ ] Body — minimally updated (a pointer at the start of each subheading), not rewritten wholesale
- [ ] `down` extended with the new sub-MOCs
- [ ] Bases/Dataview, `aliases`, `up`, `links`, `other`, `tags` — unchanged
