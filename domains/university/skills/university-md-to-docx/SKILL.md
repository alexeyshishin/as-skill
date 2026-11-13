---
name: university-md-to-docx
description: >
  Converts a Markdown graduation thesis (ВКР) into a .docx formatted to
  GOST 7.32: title page, headings, table of contents, page numbers, figure and
  table captions, formulas, lists, sources, appendices. Then checks the result
  mechanically. Use when the user wants to "convert the thesis to Word",
  "format ВКР per GOST", "сделай docx по ГОСТ", "оформи диплом", "md в docx
  для ВКР", "проверь оформление docx".
---

# university-md-to-docx — Markdown → GOST 7.32 .docx

Goal: one command turns a thesis written in Markdown into a Word file that already satisfies the formatting requirements, and a second one proves it.

Before starting, read `references/rules/university-gost.md` — the requirements the script implements. Markup conventions the script expects are in [`references/md-conventions.md`](references/md-conventions.md); read it when the source `.md` was not written by `university-writer`.

Two scripts ship with this skill, both need `python3` and `python-docx` (`pip install python-docx`):
- [`scripts/md2docx.py`](scripts/md2docx.py) — the converter
- [`scripts/check-docx.py`](scripts/check-docx.py) — mechanical conformance check of a finished `.docx`

## Step 1. Check the input

- find the `.md` (ask the user if there are several); the images it references must exist relative to the file
- the frontmatter carries title-page fields; list the missing ones (`title`, `author`, `supervisor`, `university`, `department`, `city`, `year` at minimum) and ask the user — never invent names, group numbers, or a department
- if the university has its own template or methodical guidelines that differ from `university-gost` — say so now: the script follows GOST 7.32 defaults

## Step 2. Convert

```bash
python3 scripts/md2docx.py thesis.md -o thesis.docx
```

Warnings go to stderr (missing image, table without caption, unresolved `@fig:id`). Show them to the user. `--strict` makes any warning exit 2.

## Step 3. Check

```bash
python3 scripts/check-docx.py thesis.docx
```

Exit 0 — clean. Exit 2 — errors: fix the **Markdown source** (or the frontmatter) and re-convert; don't patch the `.docx` by hand, the next conversion would erase it. A finding that comes from a deliberate university-specific deviation — report it to the user and don't "fix" it.

The checker verifies form, not meaning: it can't tell whether the text is good, the figures are legible, or the sources are real. That is `university-reviewer`'s job.

## Step 4. Tell the user what's left manually

- open the file in Word and accept the prompt to update fields (or press `Ctrl+A`, `F9`): the table of contents and page numbers are fields, they fill in on open
- `Продолжение таблицы N` for tables that break across pages is not generated
- anything the university requires beyond GOST (signatures sheet, annotation in English, anti-plagiarism certificate) is not generated

## What not to do

- don't edit the `.docx` by hand-written XML to "fix" a finding — fix the source
- don't invent frontmatter values (names, group, supervisor, year)
- don't claim the file conforms to the university's requirements — only to GOST 7.32 as encoded in `university-gost`
- don't overwrite an existing `.docx` the user may have edited without asking
