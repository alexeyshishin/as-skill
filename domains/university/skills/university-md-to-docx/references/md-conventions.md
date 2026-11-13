# Markdown conventions for md2docx

## Frontmatter (title page)

```yaml
---
ministry: Министерство науки и высшего образования Российской Федерации
university: Федеральное государственное ... университет «...»
faculty: Факультет ...
department: Кафедра ...
work_type: ВЫПУСКНАЯ КВАЛИФИКАЦИОННАЯ РАБОТА
degree: бакалавр
direction: 09.03.04 Программная инженерия
profile: ...
title: Тема работы
author: И. И. Иванов
group: ПИ-41
supervisor: П. П. Петров
supervisor_position: к.т.н., доцент
advisor: ...
head: С. С. Сидоров
city: Омск
year: 2026
numbering: continuous
title_page: true
---
```

Every key is optional; an absent key drops its line. `work_type` defaults to `ВЫПУСКНАЯ КВАЛИФИКАЦИОННАЯ РАБОТА`. `numbering` is `continuous` (`Рисунок 1`) or `section` (`Рисунок 2.1`). `title_page: false` skips the title page and the page-number suppression.

## Headings

| Markdown | Result |
|----------|--------|
| `# Введение`, `# Заключение`, `# Реферат`, `# Список использованных источников`, `# Термины и определения`, `# Перечень сокращений и обозначений` | unnumbered, uppercase, centered, new page |
| `# Название` | section `1 Название`, new page |
| `## Название` | `1.1 Название` |
| `### Название` | `1.1.1 Название` |
| `# Приложение Название` | `ПРИЛОЖЕНИЕ А` + title; letters assigned in order; inside it `##` → `А.1` |

Don't type numbers by hand — they're assigned automatically (a leading `1.2 ` is stripped). `# Содержание` is optional: without it the table of contents is inserted before the first section after the реферат.

## Figures

```markdown
![Архитектура системы](img/arch.png){#fig:arch}
```

Caption goes below: `Рисунок N – Архитектура системы`. Wider than the text block (16.5 cm) shrinks proportionally. A missing file leaves a visible placeholder and a warning.

## Tables

```markdown
Таблица: Сравнение систем {#tbl:cmp}

| Система | Цена | Скорость |
|:--|:-:|--:|
| A | 10 | 5 |
```

The `Таблица:` line goes right above the table. Alignment markers in the separator row are honored. Empty cells become `–`. The header row repeats on page breaks.

## Formulas

```markdown
$$ E = mc^2 $$ {#eq:energy}
```

Own line, centered, number `(N)` flush right. Plain text only — no LaTeX rendering.

## Cross-references

`@fig:id`, `@tbl:id`, `@eq:id` become the number: «на рисунке @fig:arch» → «на рисунке 1». Write the noun and case yourself.

## Lists

`- item` → `– item`. `1. item` → `1) item`; in the sources section `1. item` → `1. item`. Lines indented under an item continue it.

## Other

- `**bold**`, `*italic*`, `` `code` `` inline
- fenced code blocks → monospace, single spacing, left aligned (appendices)
- `\newpage` on its own line — page break
- citations `[1]`, `[1, с. 45]` are plain text; the checker validates them against the sources list
