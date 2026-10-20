# Personal knowledge base structure

## Vault overview

- **Type**: Personal knowledge base (PKB) built on Obsidian
- **Content language**: Mostly **Russian** (technical terms in English, formulas in LaTeX)
- **Methodology**: **PARA** (Projects, Areas, Resources, Archives)
- **Vault root:** set at install time via `--vault <path>` or the `OBSIDIAN_VAULT` env var. Referred to below as `<vault>/`.

## Vault structure

```
<vault>/
├── _Система                        # everything supporting the Obsidian setup
│   ├── _Система.md                 # index note for this section
│   ├── templates/                  # Note templates (Templater) — see template-usage.md
│   └── cache/                      # Pasted images and other attachments
├── 00. Входящие                    # PARA - Main inbox (flat, unprocessed notes)
├── 01. Проекты                     # PARA - Projects
│   └── <NN. Проект>/                # One numbered subfolder per active project. Internal
│                                    # structure is project-specific — commonly some subset
│                                    # of: 00. Входящие, Отчёты, Постмортемы, Задачи, Планы,
│                                    # Презентации, Архив
├── 02. Сферы                       # PARA - Life areas
│   ├── 01. Личное развитие
│   ├── 02. Работа
│   ├── 03. Образование
│   ├── 04. Личный блог             # Posts and articles for my own blog/channel
│   ├── 05. Личный бренд            # Personal brand development
│   ├── 06. Здоровье                # Health
│   └── 07. Карты                   # MOCs/maps — vault-wide navigation hub: one index note
│                                    # per top-level PARA section plus domain-specific
│                                    # navigation notes for 03. Ресурсы
├── 03. Ресурсы                     # PARA - Resources, grouped by subject/domain rather than
│   │                                # by note type (no vault-wide Статьи/Заметки/Видео split)
│   └── <NN. Тема>/                  # A subject area or "Книги"; internal structure varies —
│                                    # a topic folder commonly splits into Кейсы/Курсы/База
│                                    # знаний (each further split by sub-topic); a book folder
│                                    # holds one "00. Оглавление" note plus per-chapter notes
├── 04. Архив                       # PARA - Archive (flat)
└── 05. Дневник                     # Daily entries, personal journal (flat, `DD-MM-YYYY.md`)
```

## Notes for maintainers

- `01. Проекты` and the subject folders under `03. Ресурсы` are intentionally shown with placeholder names (`<NN. Проект>`, `<NN. Тема>`) — their actual contents are personal/employer-specific and churn as projects start and finish. Don't hardcode real project/company names here; route by the numbered-folder pattern instead.
- `#person`/`02. Сферы/.../Люди` and `#conference`/`.../Конференции` were dropped from `tags.md`'s structural-tags table — the vault has no folder for either right now (though the `#person` tag and its template still exist). If one of those notes comes up, ask the user where it should live.
- The `3. Холсты`, `4. Copilot`, `5. Bases`, `6. Clippings`, `7. Zotero`, `8. iBooks` subfolders under `_Система` aren't present in the current vault; `cache/` is the only attachment-style folder in use today.
- `sources:` (the frontmatter list field from the old "option B" up+sources policy) doesn't appear in a single note in the current vault — treated as retired in `note-types-frontmatter.md`. If this was accidental drift rather than an intentional change, flag it — reviving it would mean re-auditing everything ingested since the policy stopped being applied.

The rest of the domain (`tags.md`, `knowledge-structures.md`, `workflows.md`, `template-usage.md`, `note-types-frontmatter.md`, `file-naming.md`, and the skills/agents that hardcode these paths) was synced to this structure on 2026-08-15 — re-run this same read-the-backup-and-diff exercise periodically, since folder numbering (projects especially) has already shifted once.
