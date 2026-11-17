---
name: content-init
description: >
  Sets up a content workspace for posts and articles: CONTENT.md (channel
  passport — topic, audience, platforms, image handle, signature, CTA),
  voice/ with the author's reference texts, status folders ideas/ → drafts/
  → ready/ → published/<year>/ → archive/, and images/. Idempotent: adds only
  what's missing, never overwrites, moves existing drafts or images into the
  layout only on the author's say-so. Use when the user wants "content-init",
  "set up a folder for posts", "structure for drafts and published posts",
  "init the content workspace", "where do I keep my posts".
---

# content-init — content workspace

Goal: one place where every post lives from idea to publication, laid out so the other content skills know where to read and save without asking.

Before starting, read:
- `references/rules/content-workspace.md` — the layout this skill creates; it's the contract the other skills rely on

## Step 1. Find or choose the root

Look for `CONTENT.md` in the current directory, then in `content/`.

- **Found** — the workspace exists, run in repair mode: compare it with the layout in the rule, list the missing folders, and the passport fields still set to `—`. Offer to create the folders and fill those fields. Go to Step 3.
- **Not found** — ask where to create it:
  - the current directory, if it's dedicated to content (empty, or holds only posts and notes)
  - `content/`, if the directory holds anything else (code, a vault, a repo of something else)

  Suggest the fitting one as the default.

## Step 2. Look at what's already there

In the chosen root, and in the current directory if it differs, look for content that predates the workspace:
- folders: `drafts/`, `posts/`, `published/`, `post-images/`, `images/`
- loose `*.md` files that look like posts or drafts

Show what was found and propose where each group goes: `post-images/` → `images/`, finished posts → `published/<YYYY>/`, the rest → `drafts/`. Name the target for every group; the author picks per group: move, or leave as is. Don't rename files or add frontmatter to them unless the author asks — a move is enough.

Nothing found — skip this step silently.

## Step 3. Passport

Ask in one message (in Russian):
- what the channel is about, in 1–2 phrases
- who it's for
- platforms and addresses (`@channel`, a Habr profile, a blog URL) and what goes where (posts, articles)
- the handle for images — suggest the Telegram handle; if `CONTENT_IMAGE_HANDLE` is set, suggest its value
- a signature under posts and a CTA, if there are any

Fill only what the author said; everything else stays `—`. Fields that are in the template but the author didn't mention are not questions to repeat — one round is enough.

## Step 4. Voice samples

Ask for 3–5 of the author's own published texts: paste them, or give file paths or a folder. Save each verbatim to `voice/<slug>.md`, no frontmatter, no edits.

No samples — leave `voice/` empty and say that `content-humanizer` and the writing skills will keep asking for samples until it's filled.

## Step 5. Show the plan, then create

Show the target tree with a mark per entry: `+` new, `=` exists, `→` moved from where. Under it — the `CONTENT.md` that will be written. Wait for the author's OK.

Then:
1. Create the folders from the rule. `published/` gets no year subfolder yet — it's created on the first publish.
2. Put a `.gitkeep` into every folder that stays empty, so git keeps it.
3. Write `CONTENT.md` from `references/templates/CONTENT.md` with the passport answers filled in. In repair mode, edit only the fields the author answered.
4. Write the voice samples.
5. Do the approved moves: `git mv` for tracked files, `mv` otherwise.

## Step 6. Report

Show the resulting tree, what was created and moved, and which passport fields are still `—`. Then one line on how it works from here: `content-tg-post` and `content-article-draft` save drafts to `drafts/`, `content-image` saves to `images/<slug>/`, `content-editor` moves an approved post to `ready/` and a published one to `published/<YYYY>/`.

## What not to do

- don't overwrite an existing file — `CONTENT.md` included; repair mode only adds
- don't delete anything, empty legacy folders included, unless the author asks
- don't invent passport fields — audience, handle, platforms come from the author or stay `—`
- don't edit voice samples, not even typos
- don't add folders outside the rule — the other skills don't know about them
- don't commit — the author decides when
