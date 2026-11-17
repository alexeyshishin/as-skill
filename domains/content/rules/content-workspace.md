# content-workspace

Layout of a content workspace: where a post lives at each stage, how files are named, what frontmatter they carry. Skills in the `content` domain reference this as `content-workspace`. `content-init` creates the layout; the other skills read and write by it.

## Finding the workspace

The workspace root is the directory that holds `CONTENT.md`. Look in the current directory, then in `content/`. Not found — there is no workspace: skills work as they would without this rule (ask where to save) and may suggest `content-init`.

## Layout

```
<root>/
├── CONTENT.md           channel passport
├── voice/               the author's reference texts
├── ideas/               one idea per file
├── drafts/              work in progress
├── ready/               edited and approved, waiting to be published
├── published/<YYYY>/    published, by year
├── archive/             dropped or postponed, kept
└── images/<slug>/       the post's images, emoji.txt, variants/
```

## Status is the folder

A post's status is the folder it sits in; there is no `status:` field. A status change is a move (`git mv` when the file is tracked, else `mv`), never a copy.

| Move | When | Who |
|------|------|-----|
| `ideas/` → `drafts/` | a draft is written from the idea | the writing skill (`content-tg-post`, `content-article-draft`) |
| `drafts/` → `ready/` | the author approves the edited text | `content-editor` |
| `ready/` → `published/<YYYY>/` | the author says it's published | `content-editor`, fills `published` and `url` |
| any → `archive/` | the author drops or postpones it | on the author's request |

Move only on the author's word ("approved", "published", "drop it"), never because a text looks finished.

Drafting from an idea moves `ideas/<slug>.md` to `drafts/<slug>.md` and replaces its body with the draft. If the idea held more than the post used (links, other angles), say so first and offer to keep the rest as a new idea.

## Naming

- `slug` — kebab-case, Latin, up to 50 characters, from the topic (`k8s-oom-postmortem`). It's the post's id for its whole life: the same in `ideas/`, `drafts/`, `ready/`, `images/`.
- `ideas/`, `drafts/`, `ready/`, `archive/`: `<slug>.md`
- `published/`: `published/<YYYY>/<YYYY-MM-DD>-<slug>.md`, the date is the publish date
- images: `images/<slug>/cover.png` for a title card, `images/<slug>/01.png`, `02.png`… for text pages
- `images/<slug>/variants/` — title card variants the author hasn't picked from yet, `cover-1.png`…`cover-3.png`
- `images/<slug>/emoji.txt` — emoji on the post's images, one per line; `content-image` reads it so no emoji repeats across posts
- a slug already taken in any folder — add a suffix (`-2`), don't overwrite

## Frontmatter

Post files in `drafts/`, `ready/`, `published/`, `archive/`:

```yaml
---
title: <the post's title or its first line>
format: tg-post | article
platform: <a platform from CONTENT.md: telegram, habr, medium, blog…>
created: YYYY-MM-DD
published:
url:
images: []
tags: []
---
```

- `published`, `url` — empty until the move to `published/`
- `images` — paths from the root (`images/<slug>/cover.png`), added by `content-image`
- the body is the post exactly as it will be published, nothing else

Ideas carry only `created: YYYY-MM-DD`; the body is free-form: the thought, links, raw notes.

## CONTENT.md

The channel passport: topic, audience, platforms and their addresses, the handle for images, signature, CTA. Skills read it before asking and don't ask again for what's there. A field set to `—` is unknown: ask the author, don't guess. Change the passport only with the author's consent.

## voice/

The author's own published texts, verbatim, one per file (`<slug>.md`, no frontmatter). 3–5 are enough. They calibrate the voice for `content-humanizer` and for any draft written in the author's name. Never edit or humanize them — they are the reference.
