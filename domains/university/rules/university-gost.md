# university-gost

Base requirements for a graduation thesis (ВКР) in the `university` domain. Skills and agents reference this file as `university-gost`. Source: GOST 7.32-2017 (research reports), GOST R 7.0.100-2018 and GOST 7.1-2003 (bibliography). A university's own methodical guidelines override this file where they differ — ask the user for them before the first draft.

## Language

The thesis text is in Russian; technical terms and identifiers stay in English. Formal impersonal register: «в работе рассмотрено», «было выполнено», no first person, no colloquialisms, no marketing tone. Abbreviations are spelled out at first mention.

## Structure and order

1. Титульный лист — counted as page 1, number not shown
2. Реферат (if required by the university)
3. Содержание
4. Термины и определения; Перечень сокращений и обозначений (if there are many)
5. Введение
6. Основная часть — numbered sections
7. Заключение
8. Список использованных источников
9. Приложения

Structural elements (2–5, 7–8) have no number; headings are uppercase, centered, no trailing period. Every structural element and every section of the main part starts on a new page.

### Введение must state

актуальность · цель · задачи (each maps to a section or subsection) · объект · предмет · методы · практическая значимость · структура работы.

### Заключение

Conclusions per task, in the same order as the задачи; no new facts that are absent from the main part.

## Page layout and text

| parameter | value |
|-----------|-------|
| format | A4 |
| margins | left 30 mm, right 15 mm, top 20 mm, bottom 20 mm |
| font | Times New Roman, black, 14 pt (12–14 pt allowed by the standard) |
| line spacing | 1.5 |
| paragraph indent | 1.25 cm |
| alignment | justified |
| page numbers | Arabic, continuous, bottom center, no period |

Tables may use a smaller font (10–12 pt) and single spacing. Listings use a monospace font and normally go to appendices.

## Headings

- Sections numbered `1`, `2`, …; subsections `1.1`, `1.2`; sub-subsections `1.1.1`. No trailing period; no hyphenation; no underline.
- A section with subsections has no text between its heading and the first subsection heading except a short lead-in; a subsection that exists has at least one sibling (no lone `1.1`).
- Heading text does not repeat the thesis topic verbatim.

## Figures

- Caption **below** the figure, centered: `Рисунок 1 – Название`; dash is an en dash with spaces, no trailing period.
- Numbering continuous or within a section (`Рисунок 2.1`); inside an appendix `Рисунок А.1`. One scheme per thesis.
- Every figure is referenced in the text before it appears: «на рисунке 1», «(рисунок 1)».
- Figure is a real image, not a placeholder; labels on it are readable at print size.

## Tables

- Caption **above** the table, left aligned, no indent: `Таблица 1 – Название`; numbering rules as for figures (`Таблица А.1` in appendices).
- Every table is referenced in the text: «в таблице 1».
- Header row on every page when a table breaks; the continuation is marked `Продолжение таблицы 1`.
- Empty cells are filled with a dash; units go to the header.

## Formulas

- Own line, centered; number in parentheses flush right: `(1)` or `(2.1)`, `(А.1)` in appendices.
- Explanation starts with «где» on a new line without indent and without a colon; every symbol is explained once, in order of appearance.
- Reference in text: «по формуле (1)».

## Lists

- First level: dash (`–`) or a lowercase letter with a parenthesis; second level: Arabic digit with a parenthesis.
- Each item starts at the paragraph indent; items end with `;`, the last one with `.`. Lists of whole sentences each start with a capital letter and end with a period.

## Sources

- Section `СПИСОК ИСПОЛЬЗОВАННЫХ ИСТОЧНИКОВ`; entries numbered `1.`, `2.` in order of first citation (or alphabetically if the university requires it — pick one).
- Entry format per GOST R 7.0.100-2018: author, title, publisher/journal, year, pages; electronic resources with URL and access date.
- In-text citation: `[3]`, `[3, с. 45]`, `[3–5]`. Every entry is cited at least once; every citation points at an existing entry.
- Minimum count and share of recent (last 5 years) sources are set by the university; do not invent a number.
- Never fabricate a source. A reference that cannot be verified is marked `[проверить]` for the author, not made up.

## Appendices

- Heading: `ПРИЛОЖЕНИЕ А` on one line, title below it, both centered. Letters: А Б В Г Д Е Ж И К Л М Н П Р С Т У Ф Х Ц Ш Щ Э Ю Я (no Ё, З, Й, О, Ч, Ь, Ы, Ъ).
- Every appendix is referenced in the text: «в приложении А».

## Writing rules

- **Don't invent** results, measurements, statistics, interview data, survey counts, source titles. If the data is missing — leave `[ДАННЫЕ: что нужно]` in the text and tell the user.
- **Don't plagiarize**: a source is paraphrased and cited, a verbatim quote is in quotation marks with a page number. Originality percentage is the university's own system's job, not ours.
- Every claim of fact in the theory part has a citation; the practical part describes what the author did, not what is "known".
- One thought per paragraph; no paragraph shorter than 3 sentences except a lead-in.

## What this rule does not cover

University-specific title page text, signatures sheet, annotation in English, anti-plagiarism certificate, mandatory minimum volume — all come from the university's methodical guidelines. Ask the user; don't assume.
