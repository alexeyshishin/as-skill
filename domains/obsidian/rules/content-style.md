# Text style, language, and internal links

## Text style and language

### Do

- Write in **Russian** — that's the vault's primary language
- Preserve the author's **personal tone** and perspective, don't turn notes into formal documentation
- Use **brief, concise wording** — this is a personal knowledge base, not a textbook
- Leave technical terms in English: `Dataview`, `MOC`, `SRE`, `DevOps`, `plugin`, `Kubernetes`, `Prometheus`
- Use Markdown formatting: headings, lists, callout blocks
- Preserve the existing heading hierarchy (`##`, `###`, `####`)
- One main topic per note

### Don't

- Don't translate plugin names, commands, paths, or tags
- Don't turn personal notes into Wikipedia articles or official documentation
- Don't add "filler" or introductory paragraphs that aren't in the original
- Don't change the author's style to a bureaucratic or academic one
- Don't use formal address — informal address or impersonal constructions are fine in notes

## Internal links (Wikilinks)

### Do

- Use **wikilinks** (`[[Note name]]`) for all connections between notes
- Link to existing notes wherever their topics are mentioned
- When creating a new note — link to it from the parent note or MOC
- When renaming a note — update the incoming links (backlinks), if the user asks
- Use aliases via `[[Note|display text]]` when it improves readability
- In frontmatter, use the `up: []`, `down: []`, `links: []`, `other: []` fields for explicit connections:
    - `up` — parent notes (specific to general)
    - `down` — child notes (general to specific)
    - `links` — web links only
    - `other` — horizontal connections (similar topics) and other associations

### Don't

- Don't create "dead" links to nonexistent notes without the user's explicit intent
- Don't remove existing links — they're part of the knowledge graph
- Don't link every other word — link only what's genuinely related

## Markdown formatting rules

Configured in `.markdownlint.yaml`:

- Heading style — **ATX** (`## Heading`, not `Heading\n===`)
- List indentation — **4 spaces**
- Hard tabs allowed (`no-hard-tabs: false`)
- Line-length limit disabled (3600 characters — effectively none)
- MD041 disabled — the file's first element doesn't have to be an h1 heading (because of frontmatter)
