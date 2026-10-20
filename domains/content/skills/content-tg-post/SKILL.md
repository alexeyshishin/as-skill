---
name: content-tg-post
description: >
  Turns an idea, a note, or raw text into a ready-to-post Telegram channel
  post — tight, thesis-first, in the author's voice, within 500-1500
  characters. Use when the user wants "write a TG post," "shape this thought
  for the channel," "short post," "tg post," "shape an idea for subscribers."
---

# content-tg-post — Telegram channel post

Goal: take an idea or raw text and turn it into **one** Telegram post that people want to read to the end.

Before starting, read:
- `~/.claude/rules/content-voice.md` — authorial voice (informal "you," not Wikipedia)
- `~/.claude/rules/content-formatting.md` — the Telegram section

## Step 1. Understand the material

Ask (or pull from context):
- **the source**: an idea in one phrase, a note from Obsidian, a draft, bullet points
- **the channel's theme** (if not already known): technical, personal, mixed
- **the post format**: a thesis / a breakdown / a personal story / an incident post-mortem / a brief how-to

If there isn't enough material for a post — **stop and ask for specifics**: "there isn't enough for a post, I need a real-world example or some numbers."

## Step 2. Find the thesis

One post = one thesis. Ask yourself:
- what do I want the reader to walk away with?
- if they only read the first sentence, will they get the main point?

Formulate the thesis in one phrase. This is the post's first line.

## Step 3. Build the structure

Standard structure for a 500-1500 character post:

```
<Hook / thesis — 1 sentence>

<Context or expansion of the thesis — 2-4 sentences>

<Specifics: an example, a number, code, a story — mandatory>

<Implication / conclusion / question — 1-2 sentences, or none at all>
```

Each block is a separate paragraph, separated by a blank line.

## Step 4. Write it

Applying `content-voice.md`:
- no "dear readers" and no "represents"
- zero AI clichés ("let's dive in," "in this article")
- specifics > generalizations
- short sentences, short paragraphs

Applying `content-formatting.md`:
- technical terms in `backticks`
- code blocks with a language tag
- no `#` headings — Telegram doesn't render them
- MarkdownV2 escape characters (or use HTML formatting if the channel supports it)

## Step 5. Self-check

Before showing it to the user:
- **length** — within the 500-1500 range? If longer — cut it. If shorter — maybe it's a tweet, not a post.
- **thesis in the first line** — if you remove the first 1-3 sentences, is meaning lost? If not — cut the intro.
- **specifics** — is there at least one example / number / command / link?
- **personality** — could anyone have written this? If so — add a personal observation.
- **ending** — does it have one at all? Or does the text just cut off? Either is fine, as long as it's not "thus."

## Step 6. Show it and ask

Show the post in a block (how it will look in Telegram, without metadata). Below it:
- length: X characters
- checks: ✓ thesis first / ✓ specifics / ✓ voice ok

Ask:
1. Accept
2. Trim it (if close to 1500)
3. Strengthen the thesis
4. Add more specifics
5. Produce variants of the title/hook

## What not to do

- don't write posts longer than 1500 characters "because there's a lot to say" — split into a series
- don't use emoji as bullet markers
- don't invent numbers and examples — if the user doesn't have them, ask
- don't glue 2 theses into one post — that's always worse than 2 separate posts
- don't write "as I wrote before" — Telegram is non-linear, the reader may not have seen the previous post
