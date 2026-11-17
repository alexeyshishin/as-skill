---
name: content-image
description: >
  Renders post images (1920x1920 PNG) in the author's visual style: a title
  card (big English heading + emoji or logo, optional subtitle) or a text page
  (paragraphs with bold spans, inline emoji, the author's handle). Fills an
  HTML template and screenshots it with headless Chrome — no Figma at run
  time. Use when the user wants "make an image for the post", "a cover for the
  post", "a card with text", "picture for the channel", "post image".
---

# content-image — post images from the design system

Goal: turn a heading or a few paragraphs into a finished PNG that looks like the author's Figma posts, without opening Figma.

The design system (colors, fonts, sizes, offsets) lives in `references/tokens.css`, the layouts in `references/templates/`. Both were measured from the author's Figma file; renders match it to within 1-2 px. Roboto Light/Medium is bundled in `references/fonts/` (Apache 2.0, `LICENSE-Roboto`).

The renderer is `scripts/render.py` (Python 3 stdlib + Chrome/Chromium; set `CHROME_BIN` if Chrome is not found). Emoji come from the system emoji font — on macOS that is Apple Color Emoji, the same set as in the Figma file.

## Step 1. Pick the template

| The author wants | Template |
|------------------|----------|
| a cover, a section title, a series/digest card: a short heading, optionally an emoji or a logo | `title` |
| a quote, a thesis, a short text slide: 1-3 short paragraphs | `text` |

If the material fits neither (a diagram, a chart, a photo collage) — say so; this skill only renders the two templates. Don't improvise a new layout inside a render call.

If it's unclear which one — ask: "a cover with a heading or a card with text?"

## Step 2. Get the content

Take the author's wording as is. Don't rewrite it and don't invent text. If the author asks to write the text itself — do that first with `content-tg-post` or ask for the text. The one exception is a title card heading and subtitle, see below.

**title** — heading and subtitle are always in English, whatever language the post is in
- heading: English, lowercase like every frame in the Figma file (`introduction`, `obsidian`, `digest`), one line, short — roughly up to 14 characters at the default size (`introduction` fits). Longer — the renderer refuses; shorten it or ask for a shorter variant
- subtitle (optional): one short English word under the heading (`daily`, `soon`)
- if the author gives a Russian heading or subtitle, translate it into the shortest natural English word yourself (`лонгриды` → `longreads`, `скоро` → `soon`) and name the translation in your report. The renderer rejects any Cyrillic in the heading and subtitle
- mark (optional): one emoji (`--emoji`) or a logo file (`--icon path.png|svg`), not both

**text** — the body keeps the author's language (Russian as is)
- paragraphs are separated by a blank line, a single newline is a line break
- `**bold**` for emphasis, nothing else is parsed
- emoji go inline; a space before it is not needed (the renderer sets the gap)
- capacity: about 10-12 lines of ~25-30 characters; longer text is refused — split it into several images
- handle: `--handle @name`, or the `CONTENT_IMAGE_HANDLE` environment variable; `--handle ""` turns it off. If neither is set, ask the author once and suggest putting the variable into their shell profile

## Step 3. Render

```bash
python3 scripts/render.py title --heading "digest" --emoji "🧊" --emoji-size 150 -o out/february.png
python3 scripts/render.py title --heading "obsidian" --subtitle "daily" --icon assets/logo.svg -o out/obsidian-daily.png
python3 scripts/render.py text --file post.txt --handle "@name" -o out/post.png
```

`scripts/` is relative to this skill's directory. Emoji size: `125` by default (headings), `150` for the digest-style cards; body emoji are always `72`.

Output path: ask where to put it; if the author has no preference, use `post-images/<slug>.png` in the current directory. Create the folder, don't overwrite an existing file without saying so.

On failure the script exits non-zero with a one-line reason (`content does not fit the canvas…`, `Chrome/Chromium not found…`). Fix the cause; don't retry with the same input.

## Step 4. Check the result

Open the PNG (Read tool) and look at it before showing it:
- the heading and the subtitle are English and lowercase
- the heading and the emoji sit on one line, centered on the canvas
- nothing is clipped, the text does not touch the handle
- emoji rendered as pictures, not as empty boxes
- the handle is correct

Report the file path and what was rendered. Ask whether to adjust the text or make the next image in the series.

## Changing the style

Edit `references/tokens.css` — sizes, colors, offsets live there as variables, the templates only use them. A new layout = a new file in `references/templates/` plus a small builder in `scripts/render.py`. To re-sync with the Figma file, re-read the frames with the Figma MCP and compare the numbers against the tokens; don't guess values.

## What not to do

- don't put Russian, or any Cyrillic, into a title card heading or subtitle; Russian belongs only in the body of a text page
- don't uppercase a heading — the Figma frames are all lowercase
- don't render without looking at the result
- don't shrink font sizes to squeeze more text in — split the text or shorten it
- don't add colors, gradients, shadows or other fonts that are not in the Figma design system
- don't put a handle on a card the author didn't ask a handle for; title cards have none by design
- don't publish or send the images anywhere — the author does that
