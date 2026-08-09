# Personal knowledge base structure

## Vault overview

- **Type**: Personal knowledge base (PKB) built on Obsidian
- **Content language**: Mostly **Russian** (technical terms in English, formulas in LaTeX)
- **Methodology**: **PARA** (Projects, Areas, Resources, Archives)
- **Vault root:** set at install time via `--vault <path>` or the `BEAR_VAULT` env var. Referred to below as `<vault>/`.

## Vault structure

```
<vault>/
├── _Система                        # everything supporting my Obsidian setup
│   ├── 1. Шаблоны                  # Templates
│   ├── 2. Статика                  # Images
│   ├── 3. Холсты                   # Canvas and Excalidraw
│   ├── 4. Copilot                  # For the Obsidian Copilot plugin
│   ├── 5. Bases                    # Bases queries and files
│   ├── 6. Clippings                # Saved via Obsidian Web Clipper
│   ├── 7. Zotero                   # Import from Zotero
│   └── 8. iBooks                   # Import from Apple Books
├── 00. Входящие                    # PARA - Main inbox
├── 01. Проекты                     # PARA - Projects
├── 02. Сферы                       # PARA - Life areas
│   ├── 01. Люди                    # Notes about people
│   ├── 02. Личное развитие
│   ├── 03. Работа
│   ├── 04. Образование
│   ├── 05. Медийность              # My posts to blog, channel, and other media
│   ├── 06. Конференции             # Conference notes and program-committee activity
│   ├── 07. Здоровье                # My health
│   └── 08. Личный бренд            # Personal brand development
├── 03. Ресурсы                     # PARA - Resources
│   ├── 01. Книги
│   ├── 02. Статьи
│   ├── 03. Литературные заметки
│   ├── 04. Заметки
│   ├── 05. Видео
│   ├── 06. Промты                  # Useful prompts I've collected or written for LLM/GPT
│   └── 07. Карты                   # Maps, MOCs
├── 04. Архив                       # PARA - Archive
└── 05. Дневник                     # Daily entries, personal journal
```
